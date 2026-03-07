package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler provides HTTP handlers for authentication endpoints.
type AuthHandler struct {
	service *AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service *AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// RegisterPhysician handles POST /auth/register/physician.
func (h *AuthHandler) RegisterPhysician(c *gin.Context) {
	var req PhysicianRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	resp, err := h.service.RegisterPhysician(c.Request.Context(), req)
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("processing", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// RegisterPatient handles POST /auth/register/patient.
func (h *AuthHandler) RegisterPatient(c *gin.Context) {
	var req PatientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	resp, err := h.service.RegisterPatient(c.Request.Context(), req)
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("processing", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		status := mapErrorToStatus(err)
		if status == http.StatusTooManyRequests {
			c.Header("Retry-After", "900")
		}
		c.JSON(status, operationOutcome("login", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// VerifyTOTP handles POST /auth/login/verify-totp.
func (h *AuthHandler) VerifyTOTP(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	userID := GetActorID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, operationOutcome("login", "Authentication required"))
		return
	}

	oldJTI := c.GetString(RequestJTIKey)
	resp, err := h.service.VerifyTOTP(c.Request.Context(), userID, req.Code, oldJTI, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("login", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	jti := c.GetString(RequestJTIKey)
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.service.Logout(c.Request.Context(), jti, req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, operationOutcome("processing", err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// RefreshToken handles POST /auth/refresh.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	resp, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("login", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetMe handles GET /auth/me.
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := GetActorID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, operationOutcome("login", "Authentication required"))
		return
	}

	profile, err := h.service.GetMe(c.Request.Context(), userID)
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("not-found", err.Error()))
		return
	}

	c.JSON(http.StatusOK, profile)
}

// SetupTOTP handles POST /auth/totp/setup.
func (h *AuthHandler) SetupTOTP(c *gin.Context) {
	userID := GetActorID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, operationOutcome("login", "Authentication required"))
		return
	}

	resp, err := h.service.SetupTOTP(c.Request.Context(), userID, "")
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("processing", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// VerifyTOTPSetup handles POST /auth/totp/verify-setup.
func (h *AuthHandler) VerifyTOTPSetup(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	userID := GetActorID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, operationOutcome("login", "Authentication required"))
		return
	}

	resp, err := h.service.VerifyTOTPSetup(c.Request.Context(), userID, req.Code)
	if err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("processing", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ChangePassword handles POST /auth/password/change.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, operationOutcome("invalid", "Invalid request: "+err.Error()))
		return
	}

	userID := GetActorID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, operationOutcome("login", "Authentication required"))
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(mapErrorToStatus(err), operationOutcome("processing", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func operationOutcome(code, diagnostics string) gin.H {
	return gin.H{
		"resourceType": "OperationOutcome",
		"issue": []gin.H{{
			"severity":    "error",
			"code":        code,
			"diagnostics": diagnostics,
		}},
	}
}

func mapErrorToStatus(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid credentials"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "pending") || strings.Contains(msg, "suspended"):
		return http.StatusForbidden
	case strings.Contains(msg, "too many"):
		return http.StatusTooManyRequests
	case strings.Contains(msg, "duplicate") || strings.Contains(msg, "already exists") || strings.Contains(msg, "already enabled"):
		return http.StatusConflict
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "validation failed") || strings.Contains(msg, "missing required"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
