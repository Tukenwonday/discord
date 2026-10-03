// Package service holds the business logic. Services never touch
// *gin.Context; handlers and the WebSocket hub both call them and both emit the
// same events through the Publisher interface defined here.
package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/cordis/backend/internal/auth"
	"github.com/cordis/backend/internal/cache"
	"github.com/cordis/backend/internal/config"
	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/permissions"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// Publisher emits WebSocket envelopes. The ws.Hub implements it for real
// deliveries and NoopPublisher implements it for background work.
type Publisher interface {
	// PublishToUser delivers an event to every socket of one user.
	PublishToUser(userID, event string, data any)
	// PublishToUsers delivers an event to every socket of the listed users.
	PublishToUsers(userIDs []string, event string, data any)
	// PublishToChannel delivers an event to every socket joined to a channel or
	// direct message conversation.
	PublishToChannel(channelID, event string, data any)
}

// NoopPublisher discards every event, used when no hub is attached.
type NoopPublisher struct{}

// PublishToUser implements Publisher.
func (NoopPublisher) PublishToUser(string, string, any) {}

// PublishToUsers implements Publisher.
func (NoopPublisher) PublishToUsers([]string, string, any) {}

// PublishToChannel implements Publisher.
func (NoopPublisher) PublishToChannel(string, string, any) {}

// Deps carries the shared dependencies of every service.
type Deps struct {
	DB     *gorm.DB
	Cache  *cache.Client
	Tokens *auth.Manager
	Config *config.Config
	Log    zerolog.Logger
	Pub    Publisher
}

// Paginated is the generic cursor page from section 4 of the contract.
type Paginated[T any] struct {
	Items      []T     `json:"items"`
	HasMore    bool    `json:"hasMore"`
	NextCursor *string `json:"nextCursor"`
}

// PageOptions are the cursor parameters accepted by list operations.
type PageOptions struct {
	BeforeID string
	AfterID  string
	Limit    int
}

// DefaultPageLimit is used when the caller omits the limit.
const DefaultPageLimit = 50

// MaxPageLimit caps the limit for every paginated endpoint.
const MaxPageLimit = 100

// Normalize clamps the limit into the 1..MaxPageLimit range.
func (p PageOptions) Normalize() PageOptions {
	if p.Limit <= 0 {
		p.Limit = DefaultPageLimit
	}
	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}
	return p
}

// cursorRow is the ordering key of a row referenced by a cursor.
type cursorRow struct {
	CreatedAt time.Time
	ID        string
}

// resolveCursor loads the ordering key of the row a cursor points at.
func resolveCursor(tx *gorm.DB, table, id string) (cursorRow, bool, error) {
	if id == "" {
		return cursorRow{}, false, nil
	}
	var row cursorRow
	err := tx.Table(table).Select("id, created_at").Where("id = ?", id).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cursorRow{}, false, nil
		}
		return cursorRow{}, false, fmt.Errorf("resolve cursor: %w", err)
	}
	return row, true, nil
}

// paginate applies the contract cursor semantics and runs the query. Rows are
// returned oldest to newest; when before is set the query runs newest to
// oldest internally and is reversed before returning so the client always
// receives an ascending page.
func paginate[T any](tx *gorm.DB, opts PageOptions, table string, q *gorm.DB) ([]T, bool, *string, error) {
	opts = opts.Normalize()
	descending := opts.BeforeID != ""

	switch {
	case opts.AfterID != "":
		cur, ok, err := resolveCursor(tx, table, opts.AfterID)
		if err != nil {
			return nil, false, nil, err
		}
		if ok {
			q = q.Where("created_at > ? OR (created_at = ? AND id > ?)", cur.CreatedAt, cur.CreatedAt, cur.ID)
		}
	case descending:
		cur, ok, err := resolveCursor(tx, table, opts.BeforeID)
		if err != nil {
			return nil, false, nil, err
		}
		if ok {
			q = q.Where("created_at < ? OR (created_at = ? AND id < ?)", cur.CreatedAt, cur.CreatedAt, cur.ID)
		}
	}

	order := "created_at ASC, id ASC"
	if descending {
		order = "created_at DESC, id DESC"
	}

	var rows []T
	if err := q.Order(order).Limit(opts.Limit + 1).Find(&rows).Error; err != nil {
		return nil, false, nil, fmt.Errorf("query page: %w", err)
	}
	hasMore := len(rows) > opts.Limit
	if hasMore {
		rows = rows[:opts.Limit]
	}
	if descending {
		reverseRows(rows)
	}
	var next *string
	if hasMore && len(rows) > 0 {
		if id := lastRowID(rows); id != "" {
			next = &id
		}
	}
	return rows, hasMore, next, nil
}

func reverseRows[T any](rows []T) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

// lastRowID returns the identifier of the newest row of an ascending page by
// reading the primary key through reflection, which keeps paginate generic
// over every model without adding an interface to each type.
func lastRowID[T any](rows []T) string {
	if len(rows) == 0 {
		return ""
	}
	v := reflect.ValueOf(rows[len(rows)-1])
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	field := v.FieldByName("ID")
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

// requireMember loads the membership row of a user in a server.
func requireMember(tx *gorm.DB, serverID, userID string) (*models.Member, error) {
	var member models.Member
	err := tx.Where("server_id = ? AND user_id = ?", serverID, userID).Take(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewForbidden("you are not a member of this server")
		}
		return nil, fmt.Errorf("load member: %w", err)
	}
	return &member, nil
}

// memberRoleIDs lists the roles assigned to a membership row.
func memberRoleIDs(tx *gorm.DB, memberID string) ([]string, error) {
	var ids []string
	if err := tx.Model(&models.MemberRole{}).Where("member_id = ?", memberID).Pluck("role_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("load member roles: %w", err)
	}
	return ids, nil
}

// MemberPermissions returns the merged permission mask of a member and whether
// the member owns the server.
func MemberPermissions(tx *gorm.DB, serverID, userID string) (int64, bool, error) {
	member, err := requireMember(tx, serverID, userID)
	if err != nil {
		return 0, false, err
	}
	var server models.Server
	if err := tx.Select("id, owner_id").Where("id = ?", serverID).Take(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, httpx.NewNotFound("server not found")
		}
		return 0, false, fmt.Errorf("load server: %w", err)
	}
	roleIDs, err := memberRoleIDs(tx, member.ID)
	if err != nil {
		return 0, false, err
	}
	var masks []int64
	if len(roleIDs) > 0 {
		if err := tx.Model(&models.Role{}).
			Where("id IN ? AND server_id = ?", roleIDs, serverID).
			Pluck("permissions", &masks).Error; err != nil {
			return 0, false, fmt.Errorf("load roles: %w", err)
		}
	}
	if len(masks) == 0 {
		// A member without an explicit @everyone row falls back to the default
		// mask so a server stays usable even before roles are assigned.
		masks = []int64{permissions.Everyone}
	}
	return permissions.Compute(masks), server.OwnerID == userID, nil
}

// RequirePermission enforces a single permission for a user in a server.
func RequirePermission(tx *gorm.DB, serverID, userID string, perm permissions.Permission) error {
	mask, isOwner, err := MemberPermissions(tx, serverID, userID)
	if err != nil {
		return err
	}
	if !permissions.Can(mask, perm, isOwner) {
		return httpx.NewForbidden("you do not have permission to perform this action")
	}
	return nil
}

// MemberUserIDs lists the user ids of every member of a server, used to fan
// out channel events.
func MemberUserIDs(tx *gorm.DB, serverID string) ([]string, error) {
	var ids []string
	if err := tx.Model(&models.Member{}).Where("server_id = ?", serverID).Pluck("user_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("load member ids: %w", err)
	}
	return ids, nil
}

// DMUserIDs lists the participants of a direct message conversation.
func DMUserIDs(tx *gorm.DB, dmID string) ([]string, error) {
	var ids []string
	if err := tx.Model(&models.DMRecipient{}).Where("dm_id = ?", dmID).Pluck("user_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("load dm recipients: %w", err)
	}
	return ids, nil
}

// IsDMRecipient reports whether the user belongs to the conversation.
func IsDMRecipient(tx *gorm.DB, dmID, userID string) (bool, error) {
	var count int64
	if err := tx.Model(&models.DMRecipient{}).Where("dm_id = ? AND user_id = ?", dmID, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check dm recipient: %w", err)
	}
	return count > 0, nil
}

// RequireDMRecipient enforces direct message membership.
func RequireDMRecipient(tx *gorm.DB, dmID, userID string) error {
	ok, err := IsDMRecipient(tx, dmID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.NewForbidden("you are not a participant of this conversation")
	}
	return nil
}

// ChannelAudience returns every user id that must receive an event for the
// given channel: the members of its server, or the participants of the direct
// message conversation.
func ChannelAudience(tx *gorm.DB, channelID string) ([]string, error) {
	var channel models.Channel
	err := tx.Where("id = ?", channelID).Take(&channel).Error
	if err == nil {
		return MemberUserIDs(tx, channel.ServerID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load channel: %w", err)
	}
	return DMUserIDs(tx, channelID)
}

// HasFriend reports whether an accepted friendship exists between two users.
func HasFriend(tx *gorm.DB, a, b string) (bool, error) {
	var count int64
	err := tx.Model(&models.Friend{}).
		Where("status = ? AND ((user_id = ? AND addressee_id = ?) OR (user_id = ? AND addressee_id = ?))",
			models.FriendAccepted, a, b, b, a).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check friendship: %w", err)
	}
	return count > 0, nil
}

// touchDM bumps the conversation timestamp so the DM list stays ordered by
// recent activity.
func touchDM(tx *gorm.DB, dmID string) error {
	if err := tx.Model(&models.DirectMessage{}).Where("id = ?", dmID).
		Update("updated_at", time.Now().UTC()).Error; err != nil {
		return fmt.Errorf("touch dm: %w", err)
	}
	return nil
}
// serverOfDestination returns the server owning a destination, or an empty
// string when the destination is a direct message conversation. A missing row
// yields an empty string so callers can fall back to the DM rules.
func serverOfDestination(tx *gorm.DB, destinationID string) (string, error) {
	var channel models.Channel
	err := tx.Select("id, server_id").Where("id = ?", destinationID).Take(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", httpx.NewInternalFromError(fmt.Errorf("load destination: %w", err))
	}
	return channel.ServerID, nil
}

// requireDestination verifies that a user may reach a destination, which is
// either a server channel they belong to or a direct message they are part of.
func requireDestination(tx *gorm.DB, destinationID, userID string) error {
	serverID, err := serverOfDestination(tx, destinationID)
	if err != nil {
		return err
	}
	if serverID == "" {
		return RequireDMRecipient(tx, destinationID, userID)
	}
	if _, err := requireMember(tx, serverID, userID); err != nil {
		return err
	}
	return RequirePermission(tx, serverID, userID, permissions.ViewChannel)
}

// RequireDestination exports the destination access check so the WebSocket hub
// can verify a socket may join a channel or conversation.
func RequireDestination(tx *gorm.DB, destinationID, userID string) error {
	return requireDestination(tx, destinationID, userID)
}

// DestinationServerID exposes serverOfDestination to the WebSocket layer.
func DestinationServerID(tx *gorm.DB, destinationID string) string {
	serverID, err := serverOfDestination(tx, destinationID)
	if err != nil {
		return ""
	}
	return serverID
}

// isForbidden reports whether an API error is a 403, which callers use to
// distinguish "not a member yet" from a real failure.
func isForbidden(err error) bool {
	var apiErr *httpx.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == httpx.CodeForbidden
	}
	return false
}

// ctxBackground is used by background goroutines that must not inherit a
// cancelled request context.
func ctxBackground() context.Context { return context.Background() }