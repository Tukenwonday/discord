// Package permissions implements the bitmask from section 7 of the contract.
package permissions

// Permission is a single bit of the permission mask.
type Permission int64

// The permission bits, matching the contract table exactly.
const (
	CreateInstantInvite Permission = 1 << 0
	KickMembers         Permission = 1 << 1
	BanMembers          Permission = 1 << 2
	Administrator       Permission = 1 << 3
	ManageChannels      Permission = 1 << 4
	ManageServer        Permission = 1 << 5
	AddReactions        Permission = 1 << 6
	ViewAuditLog        Permission = 1 << 7
	PrioritySpeaker     Permission = 1 << 8
	Stream              Permission = 1 << 9
	ViewChannel         Permission = 1 << 10
	SendMessages        Permission = 1 << 11
	SendTTSMessages     Permission = 1 << 12
	ManageMessages      Permission = 1 << 13
	EmbedLinks          Permission = 1 << 14
	AttachFiles         Permission = 1 << 15
	ReadMessageHistory  Permission = 1 << 16
	MentionEveryone     Permission = 1 << 17
	UseExternalEmojis   Permission = 1 << 18
	ViewServerInsights  Permission = 1 << 19
)

// Everyone is the mask granted to the @everyone role of a new server. It never
// contains Administrator.
const Everyone = int64(ViewChannel | SendMessages | ReadMessageHistory | AddReactions | AttachFiles | EmbedLinks)

// All is the union of every defined bit.
const All = int64(CreateInstantInvite | KickMembers | BanMembers | Administrator |
	ManageChannels | ManageServer | AddReactions | ViewAuditLog | PrioritySpeaker |
	Stream | ViewChannel | SendMessages | SendTTSMessages | ManageMessages | EmbedLinks |
	AttachFiles | ReadMessageHistory | MentionEveryone | UseExternalEmojis | ViewServerInsights)

// Names maps every permission to its contract name, used by logs and tests.
var Names = map[Permission]string{
	CreateInstantInvite: "CreateInstantInvite",
	KickMembers:         "KickMembers",
	BanMembers:          "BanMembers",
	Administrator:       "Administrator",
	ManageChannels:      "ManageChannels",
	ManageServer:        "ManageServer",
	AddReactions:        "AddReactions",
	ViewAuditLog:        "ViewAuditLog",
	PrioritySpeaker:     "PrioritySpeaker",
	Stream:              "Stream",
	ViewChannel:         "ViewChannel",
	SendMessages:        "SendMessages",
	SendTTSMessages:     "SendTTSMessages",
	ManageMessages:      "ManageMessages",
	EmbedLinks:          "EmbedLinks",
	AttachFiles:         "AttachFiles",
	ReadMessageHistory:  "ReadMessageHistory",
	MentionEveryone:     "MentionEveryone",
	UseExternalEmojis:   "UseExternalEmojis",
	ViewServerInsights:  "ViewServerInsights",
}

// Has reports whether the mask contains the permission.
func Has(mask int64, perm Permission) bool { return mask&int64(perm) != 0 }

// HasAll reports whether the mask contains every listed permission.
func HasAll(mask int64, perms ...Permission) bool {
	for _, p := range perms {
		if !Has(mask, p) {
			return false
		}
	}
	return true
}

// HasAny reports whether the mask contains at least one listed permission.
func HasAny(mask int64, perms ...Permission) bool {
	for _, p := range perms {
		if Has(mask, p) {
			return true
		}
	}
	return false
}

// Compute merges the permissions of every role of a member into one mask.
func Compute(roles []int64) int64 {
	var mask int64
	for _, r := range roles {
		mask |= r
	}
	return mask
}

// Can reports whether a member holding mask may exercise perm. Members with
// Administrator implicitly hold every permission, and the server owner is
// handled by the caller through the isOwner flag.
func Can(mask int64, perm Permission, isOwner bool) bool {
	if isOwner {
		return true
	}
	if Has(mask, Administrator) {
		return true
	}
	return Has(mask, perm)
}

// Sanitize strips Administrator from a mask so the @everyone role can never
// grant it, as required by the contract.
func Sanitize(mask int64) int64 { return mask &^ int64(Administrator) }