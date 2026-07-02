package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"go-demo/config"
	"go-demo/database"
	"go-demo/models"
	"go-demo/utils"

	"golang.org/x/crypto/bcrypt"
)

// ──────────────────────────────────────────────
// RBAC 服务 — 用户/角色/权限管理
// ──────────────────────────────────────────────

// ListUsers 列出所有用户
func (s *AuthService) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,username,email,phone,COALESCE(avatar_url,''),created_at,deleted_at FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		var deletedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Phone, &u.AvatarURL, &u.CreatedAt, &deletedAt); err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			u.DeletedAt = &deletedAt.Time
		}
		s.loadUserAccess(ctx, &u)
		users = append(users, u)
	}
	return users, rows.Err()
}

// ListUsersPaged 分页列出用户
func (s *AuthService) ListUsersPaged(ctx context.Context, page, pageSize int) ([]models.User, int64, error) {
	var total int64
	if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := normalizePagination(page, pageSize)
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,username,email,phone,COALESCE(avatar_url,''),created_at,deleted_at FROM users ORDER BY id ASC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		var deletedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Phone, &u.AvatarURL, &u.CreatedAt, &deletedAt); err != nil {
			return nil, 0, err
		}
		if deletedAt.Valid {
			u.DeletedAt = &deletedAt.Time
		}
		s.loadUserAccess(ctx, &u)
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// ListRoles 列出所有角色
func (s *AuthService) ListRoles(ctx context.Context) ([]models.Role, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,name,description,created_at FROM roles ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []models.Role
	for rows.Next() {
		var r models.Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Permissions, _ = s.permissionsForRoleID(ctx, r.ID)
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

// ListRolesPaged 分页列出角色
func (s *AuthService) ListRolesPaged(ctx context.Context, page, pageSize int) ([]models.Role, int64, error) {
	var total int64
	if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM roles`).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := normalizePagination(page, pageSize)
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,name,description,created_at FROM roles ORDER BY id ASC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var roles []models.Role
	for rows.Next() {
		var r models.Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		r.Permissions, _ = s.permissionsForRoleID(ctx, r.ID)
		roles = append(roles, r)
	}
	return roles, total, rows.Err()
}

// GetRoleByID 根据 ID 获取角色
func (s *AuthService) GetRoleByID(ctx context.Context, id int64) (*models.Role, error) {
	var role models.Role
	err := database.QueryRowCtx(ctx, s.db, `SELECT id,name,description,created_at FROM roles WHERE id=$1`, id).Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, err
	}
	role.Permissions, _ = s.permissionsForRoleID(ctx, role.ID)
	return &role, nil
}

// CreateRole 创建角色
func (s *AuthService) CreateRole(ctx context.Context, name, description string, permCodes []string) (*models.Role, error) {
	name = normalizeRoleName(name)
	if name == "" {
		return nil, ErrInvalidRole
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	roleID, err := database.InsertID(tx, `INSERT INTO roles(name,description) VALUES($1,$2) RETURNING id`, name, strings.TrimSpace(description))
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrRoleExists
		}
		return nil, err
	}
	if err := s.replaceRolePermissionsTx(ctx, tx, roleID, permCodes); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetRoleByID(ctx, roleID)
}

// UpdateRole 更新角色
func (s *AuthService) UpdateRole(ctx context.Context, id int64, name, description string, permCodes []string) (*models.Role, error) {
	name = normalizeRoleName(name)
	if name == "" {
		return nil, ErrInvalidRole
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	nowFn := database.Now()
	result, _ := database.ExecTxCtx(ctx, tx, `UPDATE roles SET name=$1,description=$2,updated_at=`+nowFn+` WHERE id=$3`, name, strings.TrimSpace(description), id)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrRoleExists
		}
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, ErrRoleNotFound
	}
	s.replaceRolePermissionsTx(ctx, tx, id, permCodes)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetRoleByID(ctx, id)
}

// DeleteRole 删除角色
func (s *AuthService) DeleteRole(_ context.Context, id int64) error {
	var name string
	err := database.QueryRow(s.db, `SELECT name FROM roles WHERE id=$1`, id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoleNotFound
	}
	if err != nil {
		return err
	}
	if name == "admin" || name == "user" {
		return ErrCannotDeleteRole
	}
	result, _ := database.Exec(s.db, `DELETE FROM roles WHERE id=$1`, id)
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// ListPermissions 列出所有权限
func (s *AuthService) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,code,description,created_at FROM permissions ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// SetUserRoles 设置用户角色
func (s *AuthService) SetUserRoles(ctx context.Context, userID int64, roleNames []string) (*models.User, error) {
	if _, err := s.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	database.ExecTxCtx(ctx, tx, `DELETE FROM user_roles WHERE user_id=$1`, userID)
	seen := make(map[string]struct{})
	for _, rn := range roleNames {
		rn = normalizeRoleName(rn)
		if rn == "" {
			continue
		}
		if _, ok := seen[rn]; ok {
			continue
		}
		seen[rn] = struct{}{}
		result, _ := database.ExecTxCtx(ctx, tx, `INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name=$2`, userID, rn)
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return nil, ErrRoleNotFound
		}
	}
	if len(seen) == 0 {
		database.ExecTxCtx(ctx, tx, `INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name='user'`, userID)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

// DeactivateUser 停用用户
func (s *AuthService) DeactivateUser(_ context.Context, currentUserID, userID int64) error {
	if currentUserID == userID {
		return ErrCannotDeleteSelf
	}
	nowFn := database.Now()
	result, _ := database.Exec(s.db, `UPDATE users SET deleted_at=COALESCE(deleted_at,`+nowFn+`),updated_at=`+nowFn+` WHERE id=$1`, userID)
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrUserNotFound
	}
	database.Exec(s.db, `UPDATE refresh_tokens SET revoked_at=`+nowFn+` WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return nil
}

// CreateUser 管理员创建用户
func (s *AuthService) CreateUser(ctx context.Context, username, email, phone, password string, roleNames []string) (*models.User, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	phone = config.NormalizePhone(phone)
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	secret, _ := utils.EncryptPassword(password)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id, err := database.InsertID(tx, `INSERT INTO users(username,email,phone,password_hash,password_secret) VALUES($1,$2,$3,$4,$5) RETURNING id`,
		username, email, phone, string(hash), secret)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	if len(roleNames) == 0 {
		roleNames = []string{"user"}
	}
	for _, rn := range roleNames {
		rn = normalizeRoleName(rn)
		if rn == "" {
			continue
		}
		database.ExecTxCtx(ctx, tx, `INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name=$2`, id, rn)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, id)
}

// UpdateUser 管理员编辑用户
func (s *AuthService) UpdateUser(ctx context.Context, userID int64, username, email, phone string) (*models.User, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	phone = config.NormalizePhone(phone)
	nowFn := database.Now()
	result, err := database.ExecCtx(ctx, s.db,
		`UPDATE users SET username=$1,email=$2,phone=$3,updated_at=`+nowFn+` WHERE id=$4 AND deleted_at IS NULL`,
		username, email, phone, userID)
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

// UserHasPermission 检查用户是否有指定权限
func (s *AuthService) UserHasPermission(ctx context.Context, userID int64, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	var count int
	database.QueryRowCtx(ctx, s.db,
		`SELECT COUNT(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id JOIN users u ON u.id=ur.user_id WHERE ur.user_id=$1 AND p.code=$2 AND u.deleted_at IS NULL`,
		userID, code).Scan(&count)
	return count > 0, nil
}

// ──────────────────────────────────────────────
// 内部辅助方法
// ──────────────────────────────────────────────

func (s *AuthService) permissionsForRoleID(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT p.code FROM role_permissions rp JOIN permissions p ON p.id=rp.permission_id WHERE rp.role_id=$1 ORDER BY p.code ASC`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *AuthService) replaceRolePermissionsTx(ctx context.Context, tx *sql.Tx, roleID int64, codes []string) error {
	database.ExecTxCtx(ctx, tx, `DELETE FROM role_permissions WHERE role_id=$1`, roleID)
	seen := make(map[string]struct{})
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		database.ExecTxCtx(ctx, tx, `INSERT INTO role_permissions(role_id,permission_id) SELECT $1,id FROM permissions WHERE code=$2`, roleID, code)
	}
	return nil
}

func normalizeRoleName(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return pageSize, (page - 1) * pageSize
}
