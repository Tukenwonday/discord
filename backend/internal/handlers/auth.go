// Package handlers holds the gin transport layer. Handlers decode and validate
// requests, delegate to the service package and render the contract envelope;
// they never touch GORM directly.
package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Auth serves the four authentication endpoints of section 5.
type Auth struct {
	auth *service.Auth
}

// NewAuth builds the auth handler.
func NewAuth(deps service.Deps) *Auth { return &Auth{auth: service.NewAuth(deps)} }

// registerRequest is the body of POST /api/auth/register.
type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// loginRequest is the body of POST /api/auth/login.
type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// refreshRequest is the body of POST /api/auth/refresh and logout.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// Register creates an account and answers 201 with the user and token pair.
func (h *Auth) Register(c *gin.Context) {
	var req registerRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	result, err := h.auth.Register(service.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, result)
}

// Login authenticates an existing account and answers 200.
func (h *Auth) Login(c *gin.Context) {
	var req loginRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	result, err := h.auth.Login(service.LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, result)
}

// Refresh rotates a refresh token and answers 200 with a fresh pair.
func (h *Auth) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	pair, err := h.auth.Refresh(req.RefreshToken)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, pair)
}

// Logout revokes a refresh token and answers 204.
func (h *Auth) Logout(c *gin.Context) {
	var req refreshRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	if err := h.auth.Logout(req.RefreshToken); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}