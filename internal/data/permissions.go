package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}

	}

	return false
}

type PermissionStore struct {
	DB *pgxpool.Pool
}

func (s PermissionStore) AddPermissionsForUser(userId int64, codes ...string) error {
	stmt := `
	INSERT INTO users_permissions 
	SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := s.DB.Exec(ctx, stmt, userId, codes)
	return err
}

func (s PermissionStore) GetAllPermissionsForUser(userId int64) (Permissions, error) {
	stmt := `
    SELECT permissions.code
    FROM permissions 
    INNER JOIN users_permissions ON users_permissions.permissions_id = permissions.id
    INNER JOIN users ON users_permissions.user_id = users.id
    WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.DB.Query(ctx, stmt, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var permissions Permissions
	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)

		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil

}
