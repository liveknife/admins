package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"go-demo/config"
	"go-demo/database"
	"go-demo/models"
	"go-demo/utils"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ──────────────────────────────────────────────
// 全局邮件服务实例（在 main.go 中通过 InitMailer 初始化）
// ──────────────────────────────────────────────
var globalMailer *Mailer

// InitMailer 初始化全局邮件服务。应在程序启动时调用一次。
func InitMailer() { globalMailer = NewMailer() }

// ──────────────────────────────────────────────
// 错误定义（所有包共用）
// ──────────────────────────────────────────────

var (
	ErrInvalidCredentials      = errors.New("invalid email or password")
	ErrUserExists              = errors.New("username or email already exists")
	ErrUserDeactivated         = errors.New("user is deactivated")
	ErrTokenInvalid            = errors.New("reset token is invalid or expired")
	ErrRefreshInvalid          = errors.New("refresh token is invalid or expired")
	ErrUserNotFound            = errors.New("user not found")
	ErrCannotDeleteSelf        = errors.New("cannot delete current user")
	ErrRoleExists              = errors.New("role already exists")
	ErrRoleNotFound            = errors.New("role not found")
	ErrPermissionNotFound      = errors.New("permission not found")
	ErrInvalidRole             = errors.New("role name is required")
	ErrCannotDeleteRole        = errors.New("system role cannot be deleted")
	ErrEncryptedPasswordInvalid = errors.New("encrypted password is invalid")
	ErrPasswordPolicy          = errors.New("password does not meet complexity requirements: minimum 8 characters, at least one uppercase letter, one lowercase letter, one digit, and one special character")
	ErrPasswordUnavailable     = utils.ErrPasswordUnavailable
)

// ──────────────────────────────────────────────
// AuthService 认证核心服务（含 RBAC + 密码）
// ──────────────────────────────────────────────

type AuthService struct {
	db        *sql.DB
	jwtSecret []byte
	rsaKey    *rsa.PrivateKey
}

func NewAuthService(db *sql.DB) *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Print("[WARN] JWT_SECRET is empty, using fallback (unsafe for production)")
		secret = "change-me-in-production"
	}
	if secret == "change-me-in-production" {
		log.Print("[FATAL] JWT_SECRET must be changed from the default value for security")
		log.Fatal("[FATAL] Set a strong random secret via JWT_SECRET environment variable (min 32 chars recommended)")
	}
	svc := &AuthService{db: db, jwtSecret: []byte(secret)}
	// 初始化时加载 RSA key（持久化到文件，重启不丢失）
	svc.rsaKey = loadOrGenerateRSAKey()
	return svc
}

// RSA_KEY_PATH 环境变量可自定义 key 文件路径，默认 ./rsa_key.pem
const defaultRSAKeyPath = "rsa_key.pem"

func rsaKeyPath() string {
	if p := os.Getenv("RSA_KEY_PATH"); p != "" {
		return p
	}
	return defaultRSAKeyPath
}

func loadOrGenerateRSAKey() *rsa.PrivateKey {
	path := rsaKeyPath()
	if data, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
				if priv, ok := key.(*rsa.PrivateKey); ok {
					return priv
				}
			}
		}
	}
	// 文件不存在或格式不对，生成新 key 并保存
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		panic(err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0600); err != nil {
		// 写文件失败不 panic，只警告（内存中 key 仍可用）
		println("warning: failed to save RSA key to", path, err.Error())
	}
	return k
}

func (s *AuthService) getRSAKey() *rsa.PrivateKey {
	// NewAuthService 已预加载，这里仅做防御性返回
	return s.rsaKey
}

// ──────────────────────────────────────────────
// 注册 / 登录 / Token
// ──────────────────────────────────────────────

func (s *AuthService) Register(ctx context.Context, username, email, phone, password string) (*models.User, error) {
	if err := validatePasswordComplexity(password); err != nil { return nil, err }
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	phone = config.NormalizePhone(phone)
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	secret, _ := utils.EncryptPassword(password)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()

	var userCount int
	database.QueryRowTxCtx(ctx, tx, `SELECT COUNT(*) FROM users`).Scan(&userCount)
	id, err := database.InsertID(tx,
		`INSERT INTO users(username,email,phone,password_hash,password_secret) VALUES($1,$2,$3,$4,$5) RETURNING id`,
		username, email, phone, string(hash), secret)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	roleName := "user"
	if userCount == 0 { roleName = "admin" }
	database.ExecTxCtx(ctx, tx,
		`INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name=$2`,
		id, roleName)
	if err := tx.Commit(); err != nil { return nil, err }
	return s.GetUserByID(ctx, id)
}

func (s *AuthService) Login(ctx context.Context, account, password string) (*models.User, *models.TokenPair, error) {
	account = normalizeLoginAccount(account)
	var user models.User; var passwordHash string; var deletedAt sql.NullTime
	err := database.QueryRowCtx(ctx, s.db,
		`SELECT id,username,email,phone,COALESCE(avatar_url,''),password_hash,created_at,deleted_at FROM users WHERE lower(username)=$1 OR lower(email)=$2 OR phone=$3 LIMIT 1`,
		strings.ToLower(account), strings.ToLower(account), account).
		Scan(&user.ID, &user.Username, &user.Email, &user.Phone, &user.AvatarURL, &passwordHash, &user.CreatedAt, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) { return nil, nil, ErrInvalidCredentials }
	if err != nil { return nil, nil, err }
	if deletedAt.Valid { return nil, nil, ErrUserDeactivated }
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	s.updateStoredPasswordSecret(ctx, user.ID, password)
	s.loadUserAccess(ctx, &user)
	tokens, err := s.issueTokenPair(ctx, &user)
	if err != nil { return nil, nil, err }
	return &user, tokens, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*models.User, *models.TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" { return nil, nil, ErrRefreshInvalid }

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return nil, nil, err }
	defer tx.Rollback()

	var tokenID int64; var user models.User; var deletedAt sql.NullTime
	err = database.QueryRowTxCtx(ctx, tx,
		`SELECT rt.id,u.id,u.username,u.email,u.phone,COALESCE(u.avatar_url,''),u.created_at,u.deleted_at FROM refresh_tokens rt JOIN users u ON u.id=rt.user_id WHERE rt.token_hash=$1 AND rt.revoked_at IS NULL AND rt.expires_at>$2 LIMIT 1`,
		hashToken(refreshToken), time.Now()).
		Scan(&tokenID, &user.ID, &user.Username, &user.Email, &user.Phone, &user.AvatarURL, &user.CreatedAt, &deletedAt)
	if err == nil {
		if deletedAt.Valid { return nil, nil, ErrRefreshInvalid }
		s.loadUserAccess(ctx, &user)
	}
	if errors.Is(err, sql.ErrNoRows) { return nil, nil, ErrRefreshInvalid }
	if err != nil { return nil, nil, err }

	database.ExecTxCtx(ctx, tx, `UPDATE refresh_tokens SET revoked_at=`+database.Now()+` WHERE id=$1`, tokenID)
	tokens, err := s.issueTokenPairTx(ctx, tx, &user)
	if err != nil { return nil, nil, err }
	if err := tx.Commit(); err != nil { return nil, nil, err }
	return &user, tokens, nil
}

func (s *AuthService) ValidateAccessToken(tokenString string) (*models.Claims, error) {
	type customClaims struct {
		UserID  int64  `json:"user_id"`
		Account string `json:"account"`
		Email   string `json:"email"`
		Type    string `json:"type"`
		jwt.RegisteredClaims
	}
	claims := &customClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { return nil, ErrTokenInvalid }
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid || claims.Type != "access" { return nil, ErrTokenInvalid }
	return &models.Claims{
		UserID: claims.UserID, Account: claims.Account, Email: claims.Email, Type: claims.Type,
	}, nil
}

// ──────────────────────────────────────────────
// 密码重置
// ──────────────────────────────────────────────

func (s *AuthService) CreatePasswordResetToken(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	var userID int64
	err := database.QueryRowCtx(ctx, s.db, `SELECT id FROM users WHERE email=$1`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		// 邮箱不存在时返回成功，防止枚举
		return nil
	}
	if err != nil {
		return err
	}
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	database.ExecCtx(ctx, s.db,
		`INSERT INTO password_reset_tokens(user_id,token,expires_at) VALUES($1,$2,$3)`,
		userID, token, time.Now().Add(30*time.Minute))
	// 发送密码重置邮件（未配置 SMTP 时自动降级为 dry-run）
	resetLink := BuildResetLink(os.Getenv("FRONTEND_URL"), token)
	globalMailer.SendPasswordResetEmail(email, resetLink)
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := validatePasswordComplexity(newPassword); err != nil { return err }
	token = strings.TrimSpace(token)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	var resetID, userID int64
	err = database.QueryRowTxCtx(ctx, tx,
		`SELECT id,user_id FROM password_reset_tokens WHERE token=$1 AND used_at IS NULL AND expires_at>$2 ORDER BY id DESC LIMIT 1`,
		token, time.Now()).Scan(&resetID, &userID)
	if errors.Is(err, sql.ErrNoRows) { return ErrTokenInvalid }; if err != nil { return err }
	hash, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	nowFn := database.Now()
	database.ExecTxCtx(ctx, tx, `UPDATE users SET password_hash=$1,updated_at=`+nowFn+` WHERE id=$2`, string(hash), userID)
	database.ExecTxCtx(ctx, tx, `UPDATE password_reset_tokens SET used_at=`+nowFn+` WHERE id=$1`, resetID)
	database.ExecTxCtx(ctx, tx, `UPDATE refresh_tokens SET revoked_at=`+nowFn+` WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return tx.Commit()
}

// ──────────────────────────────────────────────
// 用户查询
// ──────────────────────────────────────────────

func (s *AuthService) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	var user models.User; var deletedAt sql.NullTime
	err := database.QueryRowCtx(ctx, s.db,
		`SELECT id,username,email,phone,COALESCE(avatar_url,''),created_at,deleted_at FROM users WHERE id=$1`, id).
		Scan(&user.ID, &user.Username, &user.Email, &user.Phone, &user.AvatarURL, &user.CreatedAt, &deletedAt)
	if err != nil { return nil, err }
	if deletedAt.Valid { user.DeletedAt = &deletedAt.Time }
	s.loadUserAccess(ctx, &user)
	return &user, nil
}

// ──────────────────────────────────────────────
// RSA 密码加解密
// ──────────────────────────────────────────────

func (s *AuthService) PasswordPublicKeyPEM() (string, error) {
	k := s.getRSAKey()
	der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	if err != nil { return "", err }
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

func (s *AuthService) DecryptClientPassword(encrypted string) (string, error) {
	encrypted = strings.TrimSpace(encrypted)
	if encrypted == "" { return "", ErrEncryptedPasswordInvalid }
	var ciphertext []byte
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		ciphertext, err = base64.RawStdEncoding.DecodeString(encrypted)
	}
	if err != nil { return "", ErrEncryptedPasswordInvalid }
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, s.getRSAKey(), ciphertext, nil)
	if err != nil { return "", ErrEncryptedPasswordInvalid }
	return string(plain), nil
}

// validatePasswordComplexity 校验密码强度：
// - 长度 8~72 字符
// - 至少包含大写字母、小写字母、数字、特殊字符各一个
func validatePasswordComplexity(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrPasswordPolicy
	}
	var (
		hasUpper, hasLower, hasDigit, hasSpecial bool
	)
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;':\",./<>?`~", r):
			hasSpecial = true
		}
	}
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return ErrPasswordPolicy
	}
	return nil
}

func (s *AuthService) ResetUserPassword(ctx context.Context, userID int64, newPassword string) error {
	if err := validatePasswordComplexity(newPassword); err != nil { return err }
	if _, err := s.GetUserByID(ctx, userID); err != nil { return err }
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	s.updatePasswordTx(ctx, tx, userID, newPassword)
	return tx.Commit()
}

func (s *AuthService) GetStoredPassword(ctx context.Context, userID int64) (string, error) {
	var secret string
	err := database.QueryRowCtx(ctx, s.db, `SELECT password_secret FROM users WHERE id=$1`, userID).Scan(&secret)
	if err != nil { return "", err }
	return utils.DecryptPassword(secret)
}

// ──────────────────────────────────────────────
// JWT 签发
// ──────────────────────────────────────────────

func (s *AuthService) issueTokenPair(ctx context.Context, user *models.User) (*models.TokenPair, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()
	tokens, err := s.issueTokenPairTx(ctx, tx, user)
	if err != nil { return nil, err }
	return tokens, tx.Commit()
}

func (s *AuthService) issueTokenPairTx(_ context.Context, tx *sql.Tx, user *models.User) (*models.TokenPair, error) {
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour
	now := time.Now()
	accessToken, _ := s.signAccessToken(user, now, now.Add(accessTTL))
	refreshToken, _ := randomToken(48)
	database.ExecTxCtx(context.Background(), tx,
		`INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`,
		user.ID, hashToken(refreshToken), now.Add(refreshTTL))
	return &models.TokenPair{
		AccessToken: accessToken, RefreshToken: refreshToken,
		AccessTokenExpiresIn: int64(accessTTL.Seconds()),
		RefreshTokenExpiresIn: int64(refreshTTL.Seconds()), TokenType: "Bearer",
	}, nil
}

func (s *AuthService) signAccessToken(user *models.User, issuedAt, expiresAt time.Time) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID, "account": user.Email, "email": user.Email, "type": "access",
		"sub": user.Email, "iat": jwt.NewNumericDate(issuedAt), "exp": jwt.NewNumericDate(expiresAt),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// ──────────────────────────────────────────────
// 内部辅助方法
// ──────────────────────────────────────────────

func (s *AuthService) updatePasswordTx(ctx context.Context, tx *sql.Tx, userID int64, password string) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	secret, _ := utils.EncryptPassword(password)
	nowFn := database.Now()
	database.ExecTxCtx(ctx, tx, `UPDATE users SET password_hash=$1,password_secret=$2,updated_at=`+nowFn+` WHERE id=$3`,
		string(hash), secret, userID)
	database.ExecTxCtx(ctx, tx, `UPDATE refresh_tokens SET revoked_at=`+nowFn+` WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return nil
}

func (s *AuthService) updateStoredPasswordSecret(ctx context.Context, userID int64, password string) error {
	secret, _ := utils.EncryptPassword(password)
	nowFn := database.Now()
	database.ExecCtx(ctx, s.db, `UPDATE users SET password_secret=$1,updated_at=`+nowFn+` WHERE id=$2`, secret, userID)
	return nil
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
func hashToken(t string) string { sum := sha256.Sum256([]byte(t)); return hex.EncodeToString(sum[:]) }
func normalizeLoginAccount(a string) string { return strings.Join(strings.Fields(strings.TrimSpace(a)), "") }

func (s *AuthService) loadUserAccess(ctx context.Context, user *models.User) error {
	roles, err := s.rolesForUserID(ctx, user.ID); if err != nil { return err }
	perms, err := s.permissionsForUserID(ctx, user.ID); if err != nil { return err }
	user.Roles = roles; user.Permissions = perms; return nil
}
func (s *AuthService) rolesForUserID(ctx context.Context, userID int64) ([]string, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT r.name FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 ORDER BY r.id ASC`, userID)
	if err != nil { return nil, err }; defer rows.Close()
	var out []string; for rows.Next() { var r string; if err := rows.Scan(&r); err != nil { return nil, err }; out = append(out, r) }
	return out, rows.Err()
}
func (s *AuthService) permissionsForUserID(ctx context.Context, userID int64) ([]string, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT DISTINCT p.code FROM user_roles ur JOIN roles r ON r.id=ur.role_id JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id WHERE ur.user_id=$1 ORDER BY p.code ASC`, userID)
	if err != nil { return nil, err }; defer rows.Close()
	var out []string; for rows.Next() { var p string; if err := rows.Scan(&p); err != nil { return nil, err }; out = append(out, p) }
	return out, rows.Err()
}

// ──────────────────────────────────────────────
// 个人中心：更新资料 / 修改密码 / 头像
// ──────────────────────────────────────────────

// ProfileInput 表示个人资料的更新字段。
type ProfileInput struct {
	Username  string
	Email     string
	Phone     string
	AvatarURL string
}

// UpdateProfile 更新当前用户的基础资料。avatar_url 为空字符串时不覆盖头像。
func (s *AuthService) UpdateProfile(ctx context.Context, userID int64, input ProfileInput) (*models.User, error) {
	username := strings.TrimSpace(input.Username)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	phone := config.NormalizePhone(input.Phone)
	avatar := strings.TrimSpace(input.AvatarURL)

	nowFn := database.Now()
	var (
		result sql.Result
		err    error
	)
	if avatar == "" {
		result, err = database.ExecCtx(ctx, s.db,
			`UPDATE users SET username=$1,email=$2,phone=$3,updated_at=`+nowFn+` WHERE id=$4 AND deleted_at IS NULL`,
			username, email, phone, userID)
	} else {
		result, err = database.ExecCtx(ctx, s.db,
			`UPDATE users SET username=$1,email=$2,phone=$3,avatar_url=$4,updated_at=`+nowFn+` WHERE id=$5 AND deleted_at IS NULL`,
			username, email, phone, avatar, userID)
	}
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, ErrUserNotFound
	}
	return s.GetUserByID(ctx, userID)
}

// ChangePassword 校验旧密码后写入新密码，同时吊销所有 refresh token。
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	if err := validatePasswordComplexity(newPassword); err != nil { return err }
	var storedHash string
	err := database.QueryRowCtx(ctx, s.db, `SELECT password_hash FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&storedHash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(oldPassword)) != nil {
		return ErrInvalidCredentials
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	s.updatePasswordTx(ctx, tx, userID, newPassword)
	return tx.Commit()
}

// SetAvatar 只更新头像 URL。用户中心上传头像后调用。
func (s *AuthService) SetAvatar(ctx context.Context, userID int64, avatarURL string) error {
	nowFn := database.Now()
	_, err := database.ExecCtx(ctx, s.db,
		`UPDATE users SET avatar_url=$1,updated_at=`+nowFn+` WHERE id=$2 AND deleted_at IS NULL`,
		strings.TrimSpace(avatarURL), userID)
	return err
}
