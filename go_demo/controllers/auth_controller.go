package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-demo/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *services.AuthService
	captcha     *services.CaptchaService
}

func NewAuthController(authService *services.AuthService, captcha *services.CaptchaService) *AuthController {
	return &AuthController{authService: authService, captcha: captcha}
}

// ──────────────────────────────────────────────
// 请求体结构
// ──────────────────────────────────────────────

type RegisterRequest struct {
	Username          string `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email             string `json:"email" form:"email" binding:"required,email"`
	Phone             string `json:"phone" form:"phone" binding:"required,min=5,max=20"`
	PasswordEncrypted string `json:"password_encrypted" form:"password_encrypted" binding:"required" desc:"RSA-OAEP-SHA256 加密后的 base64 密码"`
	Captcha           string `json:"captcha" form:"captcha"`
	CaptchaID         string `json:"captcha_id" form:"captcha_id"`
}

type LoginRequest struct {
	Account           string `json:"account" form:"account" desc:"用户名 / 邮箱 / 手机号，任选其一"`
	Email             string `json:"email" form:"email"`
	PasswordEncrypted string `json:"password_encrypted" form:"password_encrypted" binding:"required"`
	Captcha           string `json:"captcha" form:"captcha" desc:"图形验证码，字符值"`
	CaptchaID         string `json:"captcha_id" form:"captcha_id" desc:"图形验证码 ID，来自 /captcha"`
}

type ForgotPasswordRequest struct {
	Email     string `json:"email" form:"email" binding:"required,email"`
	Captcha   string `json:"captcha" form:"captcha"`
	CaptchaID string `json:"captcha_id" form:"captcha_id"`
}

type ResetPasswordRequest struct {
	Token                string `json:"token" form:"token" binding:"required"`
	NewPasswordEncrypted string `json:"new_password_encrypted" form:"new_password_encrypted" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"`
}

// UpdateProfileRequest 当前用户资料更新
type UpdateProfileRequest struct {
	Username  string `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email     string `json:"email" form:"email" binding:"required,email"`
	Phone     string `json:"phone" form:"phone"`
	AvatarURL string `json:"avatar_url" form:"avatar_url" desc:"头像 URL，可以通过 /me/avatar 上传后填入"`
}

// ChangePasswordRequest 当前用户修改密码
type ChangePasswordRequest struct {
	OldPasswordEncrypted string `json:"old_password_encrypted" form:"old_password_encrypted" binding:"required"`
	NewPasswordEncrypted string `json:"new_password_encrypted" form:"new_password_encrypted" binding:"required"`
}

// ──────────────────────────────────────────────
// Handler：公开
// ──────────────────────────────────────────────

func (c *AuthController) PasswordPublicKey(g *gin.Context) {
	pk, err := c.authService.PasswordPublicKeyPEM()
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"algorithm": "RSA-OAEP-256", "public_key": pk})
}

func (c *AuthController) Captcha(g *gin.Context) {
	if c.captcha == nil {
		respondError(g, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	ch, err := c.captcha.Generate(g.Request.Context())
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, ch)
}

func (c *AuthController) Register(g *gin.Context) {
	var req RegisterRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.verifyCaptcha(g, req.CaptchaID, req.Captcha); err != nil {
		return
	}
	pw, err := c.authService.DecryptClientPassword(req.PasswordEncrypted)
	if err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	user, err := c.authService.Register(g.Request.Context(), req.Username, req.Email, req.Phone, pw)
	if errors.Is(err, services.ErrUserExists) {
		respondError(g, http.StatusConflict, "username, email or phone already exists")
		return
	}
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusCreated, gin.H{"user": user})
}

func (c *AuthController) Login(g *gin.Context) {
	var req LoginRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.verifyCaptcha(g, req.CaptchaID, req.Captcha); err != nil {
		return
	}
	pw, err := c.authService.DecryptClientPassword(req.PasswordEncrypted)
	if err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Email)
	}
	if account == "" {
		respondError(g, http.StatusBadRequest, "account is required")
		return
	}
	user, tokens, err := c.authService.Login(g.Request.Context(), account, pw)
	if errors.Is(err, services.ErrInvalidCredentials) {
		respondError(g, http.StatusUnauthorized, "invalid account or password")
		return
	}
	if errors.Is(err, services.ErrUserDeactivated) {
		respondError(g, http.StatusForbidden, "user is deactivated")
		return
	}
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"user": user, "tokens": tokens})
}

func (c *AuthController) RefreshToken(g *gin.Context) {
	var req RefreshTokenRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	user, tokens, err := c.authService.RefreshTokens(g.Request.Context(), req.RefreshToken)
	if errors.Is(err, services.ErrRefreshInvalid) {
		respondError(g, http.StatusUnauthorized, "refresh token is invalid or expired")
		return
	}
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"user": user, "tokens": tokens})
}

func (c *AuthController) ForgotPassword(g *gin.Context) {
	var req ForgotPasswordRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.verifyCaptcha(g, req.CaptchaID, req.Captcha); err != nil {
		return
	}
	if err := c.authService.CreatePasswordResetToken(g.Request.Context(), req.Email); err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	// 统一响应，避免邮箱枚举攻击：无论邮箱是否存在都返回相同消息
	g.JSON(http.StatusOK, gin.H{"message": "if the email exists, a password reset link has been sent"})
}

func (c *AuthController) ResetPassword(g *gin.Context) {
	var req ResetPasswordRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		respondError(g, http.StatusBadRequest, "token is required")
		return
	}
	pw, err := c.authService.DecryptClientPassword(req.NewPasswordEncrypted)
	if err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	err = c.authService.ResetPassword(g.Request.Context(), req.Token, pw)
	if errors.Is(err, services.ErrTokenInvalid) {
		respondError(g, http.StatusBadRequest, "reset token is invalid or expired")
		return
	}
	if errors.Is(err, services.ErrPasswordPolicy) {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"message": "password has been reset"})
}

// ──────────────────────────────────────────────
// Handler：个人中心（已登录）
// ──────────────────────────────────────────────

func (c *AuthController) Me(g *gin.Context) {
	userID, ok := requireUserID(g)
	if !ok {
		return
	}
	user, err := c.authService.GetUserByID(g.Request.Context(), userID)
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"user": user})
}

func (c *AuthController) UpdateProfile(g *gin.Context) {
	userID, ok := requireUserID(g)
	if !ok {
		return
	}
	var req UpdateProfileRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	user, err := c.authService.UpdateProfile(g.Request.Context(), userID, services.ProfileInput{
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		AvatarURL: req.AvatarURL,
	})
	if errors.Is(err, services.ErrUserExists) {
		respondError(g, http.StatusConflict, "username, email or phone already used")
		return
	}
	if err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"user": user})
}

func (c *AuthController) ChangeMyPassword(g *gin.Context) {
	userID, ok := requireUserID(g)
	if !ok {
		return
	}
	var req ChangePasswordRequest
	if err := bindRequest(g, &req); err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	oldPw, err := c.authService.DecryptClientPassword(req.OldPasswordEncrypted)
	if err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	newPw, err := c.authService.DecryptClientPassword(req.NewPasswordEncrypted)
	if err != nil {
		respondError(g, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.authService.ChangePassword(g.Request.Context(), userID, oldPw, newPw); err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			respondError(g, http.StatusUnauthorized, "old password is incorrect")
			return
		}
		if errors.Is(err, services.ErrPasswordPolicy) {
			respondError(g, http.StatusBadRequest, err.Error())
			return
		}
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"message": "password has been updated"})
}

// UploadAvatar 处理 multipart/form-data 头像上传。返回可访问的 URL。
// 前端拿到 URL 后需再调用 PUT /me 将 avatar_url 写入用户资料。
func (c *AuthController) UploadAvatar(g *gin.Context) {
	userID, ok := requireUserID(g)
	if !ok {
		return
	}
	fh, err := g.FormFile("file")
	if err != nil {
		respondError(g, http.StatusBadRequest, "missing file")
		return
	}
	if fh.Size <= 0 || fh.Size > 5<<20 {
		respondError(g, http.StatusBadRequest, "file size must be 1B ~ 5MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		respondError(g, http.StatusBadRequest, "only jpg/png/webp/gif is allowed")
		return
	}
	name := fmt.Sprintf("%d-%d%s", userID, time.Now().UnixNano(), ext)
	dir := filepath.Join("uploads", "avatar", time.Now().Format("200601"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	path := filepath.Join(dir, name)
	if err := g.SaveUploadedFile(fh, path); err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	url := "/" + filepath.ToSlash(path)
	if err := c.authService.SetAvatar(g.Request.Context(), userID, url); err != nil {
		respondError(g, http.StatusInternalServerError, "internal error")
		return
	}
	g.JSON(http.StatusOK, gin.H{"avatar_url": url})
}

// ──────────────────────────────────────────────
// 辅助函数
// ──────────────────────────────────────────────

// verifyCaptcha 在开启验证码校验时统一校验，出错直接写响应并返回非 nil。
// captcha 服务不可用时不阻断（避免开发/单测被卡住）。
func (c *AuthController) verifyCaptcha(g *gin.Context, id, answer string) error {
	if c.captcha == nil {
		return nil
	}
	if strings.TrimSpace(id) == "" && strings.TrimSpace(answer) == "" {
		respondError(g, http.StatusBadRequest, "captcha is required")
		return errors.New("captcha missing")
	}
	if err := c.captcha.Verify(g.Request.Context(), id, answer); err != nil {
		respondError(g, http.StatusBadRequest, "captcha is invalid or expired")
		return err
	}
	return nil
}

func requireUserID(g *gin.Context) (int64, bool) {
	value, exists := g.Get("user_id")
	if !exists {
		respondError(g, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	userID, ok := value.(int64)
	if !ok || userID <= 0 {
		respondError(g, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	return userID, true
}

func respondError(g *gin.Context, status int, msg string) {
	g.JSON(status, gin.H{"error": msg})
}

func bindRequest(c *gin.Context, req any) error {
	ct := c.GetHeader("Content-Type")
	if strings.Contains(ct, "application/json") {
		// 优先走 ShouldBindBodyWithJSON —— 如果前置中间件（如登录限流）已经读过 body，
		// 直接 ShouldBindJSON 会拿到空流；ShouldBindBodyWithJSON 会读缓存副本。
		if err := c.ShouldBindBodyWithJSON(req); err == nil {
			return nil
		}
	}
	if c.Request.URL.RawQuery != "" {
		return c.ShouldBindQuery(req)
	}
	return c.ShouldBind(req)
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := parseInt64Param(c, name)
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func parseInt64Param(c *gin.Context, name string) (int64, error) {
	s := c.Param(name)
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
