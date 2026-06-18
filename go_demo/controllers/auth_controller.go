package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go-demo/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// ──────────────────────────────────────────────
// 请求体结构
// ──────────────────────────────────────────────

type registerRequest struct {
	Username          string `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email             string `json:"email" form:"email" binding:"required,email"`
	Phone             string `json:"phone" form:"phone" binding:"required,min=5,max=20"`
	PasswordEncrypted string `json:"password_encrypted" form:"password_encrypted" binding:"required"`
}

type loginRequest struct {
	Account           string `json:"account" form:"account"`
	Email             string `json:"email" form:"email"`
	PasswordEncrypted string `json:"password_encrypted" form:"password_encrypted" binding:"required"`
}

type forgotPasswordRequest struct{ Email string `json:"email" form:"email" binding:"required,email"` }

type resetPasswordRequest struct {
	Token                string `json:"token" form:"token" binding:"required"`
	NewPasswordEncrypted string `json:"new_password_encrypted" form:"new_password_encrypted" binding:"required"`
}

type refreshTokenRequest struct{ RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"` }

// ──────────────────────────────────────────────
// Handler 方法
// ──────────────────────────────────────────────

func (c *AuthController) PasswordPublicKey(g *gin.Context) {
	pk, err := c.authService.PasswordPublicKeyPEM()
	if err != nil { g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get password public key"}); return }
	g.JSON(http.StatusOK, gin.H{"algorithm": "RSA-OAEP-256", "public_key": pk})
}

func (c *AuthController) Register(g *gin.Context) {
	var req registerRequest; if err := bindRequest(g, &req); err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	pw, err := c.authService.DecryptClientPassword(req.PasswordEncrypted)
	if err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	user, err := c.authService.Register(g.Request.Context(), req.Username, req.Email, req.Phone, pw)
	if errors.Is(err, services.ErrUserExists) { g.JSON(http.StatusConflict, gin.H{"error": "username, email or phone already exists"}); return }
	if err != nil { g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"}); return }
	g.JSON(http.StatusCreated, gin.H{"user": user})
}

func (c *AuthController) Login(g *gin.Context) {
	var req loginRequest; if err := bindRequest(g, &req); err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	pw, err := c.authService.DecryptClientPassword(req.PasswordEncrypted)
	if err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	account := strings.TrimSpace(req.Account); if account == "" { account = strings.TrimSpace(req.Email) }
	if account == "" { g.JSON(http.StatusBadRequest, gin.H{"error": "account is required"}); return }
	user, tokens, err := c.authService.Login(g.Request.Context(), account, pw)
	if errors.Is(err, services.ErrInvalidCredentials) { g.JSON(http.StatusUnauthorized, gin.H{"error": "invalid account or password"}); return }
	if errors.Is(err, services.ErrUserDeactivated) { g.JSON(http.StatusForbidden, gin.H{"error": "user is deactivated"}); return }
	if err != nil { g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"}); return }
	g.JSON(http.StatusOK, gin.H{"user": user, "tokens": tokens})
}

func (c *AuthController) RefreshToken(g *gin.Context) {
	var req refreshTokenRequest; if err := bindRequest(g, &req); err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	user, tokens, err := c.authService.RefreshTokens(g.Request.Context(), req.RefreshToken)
	if errors.Is(err, services.ErrRefreshInvalid) { g.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token is invalid or expired"}); return }
	if err != nil { g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token"}); return }
	g.JSON(http.StatusOK, gin.H{"user": user, "tokens": tokens})
}

func (c *AuthController) ForgotPassword(g *gin.Context) {
	var req forgotPasswordRequest; if err := bindRequest(g, &req); err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	token, err := c.authService.CreatePasswordResetToken(g.Request.Context(), req.Email)
	if err != nil { g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create password reset token"}); return }
	g.JSON(http.StatusOK, gin.H{"message": "if the email exists, a password reset token has been created", "reset_token": token})
}

func (c *AuthController) ResetPassword(g *gin.Context) {
	var req resetPasswordRequest; if err := bindRequest(g, &req); err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if strings.TrimSpace(req.Token) == "" { g.JSON(http.StatusBadRequest, gin.H{"error": "token is required"}); return }
	pw, err := c.authService.DecryptClientPassword(req.NewPasswordEncrypted)
	if err != nil { g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := c.authService.ResetPassword(g.Request.Context(), req.Token, pw); errors.Is(err, services.ErrTokenInvalid) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "reset token is invalid or expired"}); return
	} else if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"}); return
	}
	g.JSON(http.StatusOK, gin.H{"message": "password has been reset"})
}

func (c *AuthController) Me(g *gin.Context) {
	userIDValue, exists := g.Get("user_id"); if !exists { g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
	userID, ok := userIDValue.(int64); if !ok { g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
	user, err := c.authService.GetUserByID(g.Request.Context(), userID)
	if err != nil { g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"}); return }
	g.JSON(http.StatusOK, gin.H{"user": user})
}

// ──────────────────────────────────────────────
// 辅助函数
// ──────────────────────────────────────────────

func bindRequest(c *gin.Context, req any) error {
	ct := c.GetHeader("Content-Type")
	if strings.Contains(ct, "application/json") { if err := c.ShouldBindJSON(req); err == nil { return nil } }
	if c.Request.URL.RawQuery != "" { return c.ShouldBindQuery(req) }
	return c.ShouldBind(req)
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := parseInt64Param(c, name); if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return 0, false }
	return id, true
}

func parseInt64Param(c *gin.Context, name string) (int64, error) {
	s := c.Param(name); var v int64; _, err := fmt.Sscanf(s, "%d", &v); return v, err
}
