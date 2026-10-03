package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"gorm.io/gorm"
)

// Server to client friend events.
const (
	EventFriendRequest  = "friend.request"
	EventFriendAccepted = "friend.accepted"
)

// FriendPayload is the body of friend.request and friend.accepted.
type FriendPayload struct {
	Friend models.FriendView `json:"friend"`
}

// FriendLists is the response body of GET /api/friends.
type FriendLists struct {
	Friends  []models.FriendView `json:"friends"`
	Incoming []models.FriendView `json:"incoming"`
	Outgoing []models.FriendView `json:"outgoing"`
}

// Friend implements the friend request lifecycle. Rows are directional, from
// requester to addressee, so accepting an incoming request creates the
// mirrored accepted row as well.
type Friend struct {
	deps Deps
}

// NewFriend builds the friend service.
func NewFriend(deps Deps) *Friend { return &Friend{deps: deps} }

// Lists returns accepted, incoming and outgoing friendships for the user.
func (s *Friend) Lists(userID string) (*FriendLists, error) {
	accepted := []models.Friend{}
	if err := s.deps.DB.Where("status = ? AND (user_id = ? OR addressee_id = ?)",
		models.FriendAccepted, userID, userID).
		Order("created_at ASC").Find(&accepted).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("list friends: %w", err))
	}
	incomingRows := []models.Friend{}
	if err := s.deps.DB.Where("status = ? AND addressee_id = ?", models.FriendPending, userID).
		Order("created_at ASC").Find(&incomingRows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("list incoming: %w", err))
	}
	outgoingRows := []models.Friend{}
	if err := s.deps.DB.Where("status = ? AND user_id = ?", models.FriendPending, userID).
		Order("created_at ASC").Find(&outgoingRows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("list outgoing: %w", err))
	}

	out := &FriendLists{
		Friends:  make([]models.FriendView, 0, len(accepted)),
		Incoming: make([]models.FriendView, 0, len(incomingRows)),
		Outgoing: make([]models.FriendView, 0, len(outgoingRows)),
	}
	for i := range accepted {
		view, err := s.view(&accepted[i], userID)
		if err != nil {
			return nil, err
		}
		out.Friends = append(out.Friends, view)
	}
	for i := range incomingRows {
		view, err := s.view(&incomingRows[i], userID)
		if err != nil {
			return nil, err
		}
		out.Incoming = append(out.Incoming, view)
	}
	for i := range outgoingRows {
		view, err := s.view(&outgoingRows[i], userID)
		if err != nil {
			return nil, err
		}
		out.Outgoing = append(out.Outgoing, view)
	}
	return out, nil
}

// Request creates a pending friendship, or returns the existing row so the
// handler can answer 200 instead of 201.
func (s *Friend) Request(userID, targetID string) (*models.FriendView, bool, error) {
	if strings.TrimSpace(targetID) == "" {
		return nil, false, httpx.NewField("userId", "required")
	}
	if targetID == userID {
		return nil, false, httpx.NewValidation("validation failed").
			WithFields(map[string]string{"userId": "you cannot add yourself"})
	}
	if _, err := s.other(targetID); err != nil {
		return nil, false, err
	}

	var existing models.Friend
	err := s.deps.DB.Where("user_id = ? AND addressee_id = ?", userID, targetID).
		Take(&existing).Error
	if err == nil {
		view, viewErr := s.view(&existing, userID)
		return &view, false, viewErr
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, httpx.NewInternalFromError(fmt.Errorf("load friend: %w", err))
	}

	row := &models.Friend{UserID: userID, AddresseeID: targetID, Status: models.FriendPending}
	if err := s.deps.DB.Create(row).Error; err != nil {
		return nil, false, httpx.NewInternalFromError(fmt.Errorf("create friend: %w", err))
	}
	s.deps.Pub.PublishToUser(targetID, EventFriendRequest, FriendPayload{
		Friend: row.View(s.mustUser(targetID)),
	})
	view, err := s.view(row, userID)
	if err != nil {
		return nil, false, err
	}
	return &view, true, nil
}

// Accept marks an incoming request as accepted and mirrors it so both
// directions carry the accepted status.
func (s *Friend) Accept(userID, friendID string) (*models.FriendView, error) {
	var row models.Friend
	err := s.deps.DB.Where("id = ?", friendID).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("friend request not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load friend: %w", err))
	}
	if row.AddresseeID != userID {
		return nil, httpx.NewForbidden("only the recipient can accept this request")
	}
	if row.Status == models.FriendAccepted {
		return s.view(&row, userID)
	}

	err = s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Friend{}).Where("id = ?", row.ID).
			Update("status", models.FriendAccepted).Error; err != nil {
			return fmt.Errorf("accept friend request: %w", err)
		}
		var mirrorCount int64
		if err := tx.Model(&models.Friend{}).
			Where("user_id = ? AND addressee_id = ?", row.AddresseeID, row.UserID).
			Count(&mirrorCount).Error; err != nil {
			return fmt.Errorf("count mirror row: %w", err)
		}
		if mirrorCount > 0 {
			if err := tx.Model(&models.Friend{}).
				Where("user_id = ? AND addressee_id = ?", row.AddresseeID, row.UserID).
				Update("status", models.FriendAccepted).Error; err != nil {
				return fmt.Errorf("accept mirror row: %w", err)
			}
			return nil
		}
		mirror := &models.Friend{UserID: row.AddresseeID, AddresseeID: row.UserID, Status: models.FriendAccepted}
		if err := tx.Create(mirror).Error; err != nil {
			return fmt.Errorf("create mirror row: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}

	row.Status = models.FriendAccepted
	view, err := s.view(&row, userID)
	if err != nil {
		return nil, err
	}
	s.deps.Pub.PublishToUser(row.UserID, EventFriendAccepted, FriendPayload{Friend: view})
	return &view, nil
}

// Remove deletes the directional row addressed by id. Accepting a request
// creates two rows, so both directions are cleared together.
func (s *Friend) Remove(userID, friendID string) error {
	var row models.Friend
	err := s.deps.DB.Where("id = ?", friendID).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.NewNotFound("friend not found")
		}
		return httpx.NewInternalFromError(fmt.Errorf("load friend: %w", err))
	}
	if row.UserID != userID && row.AddresseeID != userID {
		return httpx.NewForbidden("you are not a participant of this relationship")
	}
	err = s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", row.ID).Delete(&models.Friend{}).Error; err != nil {
			return fmt.Errorf("delete friend: %w", err)
		}
		mirror := tx.Where("user_id = ? AND addressee_id = ?", row.AddresseeID, row.UserID).
			Where("status = ?", models.FriendAccepted)
		return mirror.Delete(&models.Friend{}).Error
	})
	if err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("delete mirror friend: %w", err))
	}
	return nil
}

// view projects a row with the other participant resolved from viewerID.
func (s *Friend) view(row *models.Friend, viewerID string) (models.FriendView, error) {
	otherID := row.AddresseeID
	if otherID == viewerID {
		otherID = row.UserID
	}
	user, err := s.other(otherID)
	if err != nil {
		return models.FriendView{}, err
	}
	return row.View(user), nil
}

// other loads a user that must exist for a friendship to be meaningful.
func (s *Friend) other(id string) (*models.User, error) {
	var user models.User
	if err := s.deps.DB.Where("id = ?", id).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("user not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load user: %w", err))
	}
	return &user, nil
}

// mustUser resolves a user for an event payload and falls back to an empty
// placeholder so a deleted counterpart never blocks the notification.
func (s *Friend) mustUser(id string) *models.User {
	user, err := s.other(id)
	if err != nil {
		return &models.User{ID: id}
	}
	return user
}