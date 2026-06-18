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
	ErrPasswordPolicy          = errors.New("password length must be between 6 and 72")
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
	if secret == "" { secret = "change-me-in-production" }
	return &AuthService{db: db, jwtSecret: []byte(secret)}
}

func (s *AuthService) getRSAKey() *rsa.PrivateKey {
	if s.rsaKey != nil { return s.rsaKey }
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { panic(err) }
	s.rsaKey = k
	return k
}

// ──────────────────────────────────────────────
// 注册 / 登录 / Token
// ──────────────────────────────────────────────

func (s *AuthService) Register(ctx context.Context, username, email, phone, password string) (*models.User, error) {
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
		`SELECT id,username,email,phone,password_hash,created_at,deleted_at FROM users WHERE lower(username)=$1 OR lower(email)=$2 OR phone=$3 LIMIT 1`,
		strings.ToLower(account), strings.ToLower(account), account).
		Scan(&user.ID, &user.Username, &user.Email, &user.Phone, &passwordHash, &user.CreatedAt, &deletedAt)
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
		`SELECT rt.id,u.id,u.username,u.email,u.phone,u.created_at,u.deleted_at FROM refresh_tokens rt JOIN users u ON u.id=rt.user_id WHERE rt.token_hash=$1 AND rt.revoked_at IS NULL AND rt.expires_at>$2 LIMIT 1`,
		hashToken(refreshToken), time.Now()).
		Scan(&tokenID, &user.ID, &user.Username, &user.Email, &user.Phone, &user.CreatedAt, &deletedAt)
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

func (s *AuthService) CreatePasswordResetToken(ctx context.Context, email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var userID int64
	err := database.QueryRowCtx(ctx, s.db, `SELECT id FROM users WHERE email=$1`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) { t, _ := randomToken(32); return t, nil }
	if err != nil { return "", err }
	token, err := randomToken(32); if err != nil { return "", err }
	database.ExecCtx(ctx, s.db,
		`INSERT INTO password_reset_tokens(user_id,token,expires_at) VALUES($1,$2,$3)`,
		userID, token, time.Now().Add(30*time.Minute))
	return token, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
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
		`SELECT id,username,email,phone,created_at,deleted_at FROM users WHERE id=$1`, id).
		Scan(&user.ID, &user.Username, &user.Email, &user.Phone, &user.CreatedAt, &deletedAt)
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
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil { ciphertext, _ = base64.RawStdEncoding.DecodeString(encrypted) }
	if err != nil { return "", ErrEncryptedPasswordInvalid }
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, s.getRSAKey(), ciphertext, nil)
	if err != nil { return "", ErrEncryptedPasswordInvalid }
	password := string(plain)
	if len(password) < 6 || len(password) > 72 { return "", ErrPasswordPolicy }
	return password, nil
}

func (s *AuthService) ResetUserPassword(ctx context.Context, userID int64, newPassword string) error {
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
