package handlers

import (
	"net/http"
	"strconv"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// pageOptions reads the shared before, after and limit query parameters.
func pageOptions(c *gin.Context) service.PageOptions {
	opts := service.PageOptions{
		BeforeID: c.Query("before"),
		AfterID:  c.Query("after"),
	}
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			opts.Limit = parsed
		}
	}
	return opts.Normalize()
}

// Users serves the profile endpoints of section 5.
type Users struct {
	users *service.User
}

// NewUsers builds the users handler.
func NewUsers(deps service.Deps) *Users { return &Users{users: service.NewUser(deps)} }

// updateProfileRequest is the body of PATCH /api/users/me.
type updateProfileRequest struct {
	DisplayName  *string `json:"displayName"`
	Bio          *string `json:"bio"`
	AvatarURL    *string `json:"avatarUrl"`
	BannerURL    *string `json:"bannerUrl"`
	Status       *string `json:"status"`
	CustomStatus *string `json:"customStatus"`
}

// Me returns the authenticated profile.
func (h *Users) Me(c *gin.Context) {
	user, err := h.users.Get(middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, user)
}

// UpdateMe applies a partial profile update.
func (h *Users) UpdateMe(c *gin.Context) {
	var req updateProfileRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	user, err := h.users.UpdateProfile(middleware.UserID(c), service.UpdateProfileInput{
		DisplayName:  req.DisplayName,
		Bio:          req.Bio,
		AvatarURL:    req.AvatarURL,
		BannerURL:    req.BannerURL,
		Status:       req.Status,
		CustomStatus: req.CustomStatus,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, user)
}

// Get returns one user profile.
func (h *Users) Get(c *gin.Context) {
	user, err := h.users.Get(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, user)
}

// Search returns a paginated user list for the member picker.
func (h *Users) Search(c *gin.Context) {
	page, err := h.users.Search(c.Query("q"), pageOptions(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, page)
}