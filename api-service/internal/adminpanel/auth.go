package adminpanel

import (
	"context"
	"strings"
	"time"

	adminctx "github.com/GoAdminGroup/go-admin/context"
	adminauth "github.com/GoAdminGroup/go-admin/modules/auth"
	"github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// BackendAuthProcessor authenticates go-admin login using backend users table.
func BackendAuthProcessor(pool *pgxpool.Pool, logger *zap.Logger) adminauth.Processor {
	return func(ctx *adminctx.Context) (model models.UserModel, ok bool, msg string) {
		email := strings.TrimSpace(strings.ToLower(ctx.FormValue("username")))
		password := ctx.FormValue("password")
		if email == "" || password == "" {
			return model, false, "wrong password or username"
		}

		var (
			passwordHash string
			name         string
			isActive     bool
		)
		err := pool.QueryRow(context.Background(),
			`SELECT password_hash, name, is_active
			 FROM users
			 WHERE email=$1 AND deleted_at IS NULL`,
			email,
		).Scan(&passwordHash, &name, &isActive)
		if err != nil {
			return model, false, "wrong password or username"
		}
		if !isActive {
			return model, false, "account is inactive"
		}
		if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
			return model, false, "wrong password or username"
		}

		// Mirror authenticated backend admin into goadmin identity and permissions.
		if err = ensureGoAdminPrincipal(pool, email, name, passwordHash); err != nil {
			logger.Error("sync goadmin principal", zap.Error(err), zap.String("email", email))
			return model, false, "admin login setup failed"
		}

		var goAdminUserID int64
		if err = pool.QueryRow(context.Background(),
			`SELECT id FROM goadmin_users WHERE username=$1`,
			email,
		).Scan(&goAdminUserID); err != nil {
			return model, false, "admin account not provisioned"
		}

		return models.UserModel{
			Id:       goAdminUserID,
			UserName: email,
			Name:     name,
		}, true, ""
	}
}

func ensureGoAdminPrincipal(pool *pgxpool.Pool, email, name, passwordHash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statements := []string{
		`INSERT INTO goadmin_users (username, name, password, remember_token)
		 VALUES ($1, $2, $3, '')
		 ON CONFLICT (username)
		 DO UPDATE SET name=EXCLUDED.name, password=EXCLUDED.password, updated_at=now();`,
		`INSERT INTO goadmin_roles (name, slug)
		 VALUES ('Administrator', 'administrator')
		 ON CONFLICT (slug) DO NOTHING;`,
		`INSERT INTO goadmin_permissions (name, slug, http_method, http_path)
		 VALUES ('All permission', '*', '', '*')
		 ON CONFLICT (slug) DO NOTHING;`,
		`INSERT INTO goadmin_role_users (role_id, user_id)
		 SELECT r.id, u.id
		 FROM goadmin_roles r, goadmin_users u
		 WHERE r.slug='administrator' AND u.username=$1
		 ON CONFLICT DO NOTHING;`,
		`INSERT INTO goadmin_role_permissions (role_id, permission_id)
		 SELECT r.id, p.id
		 FROM goadmin_roles r, goadmin_permissions p
		 WHERE r.slug='administrator' AND p.slug='*'
		 ON CONFLICT DO NOTHING;`,
	}

	for idx, stmt := range statements {
		if idx == 0 {
			if _, err := pool.Exec(ctx, stmt, email, name, passwordHash); err != nil {
				return err
			}
			continue
		}
		if idx == 3 {
			if _, err := pool.Exec(ctx, stmt, email); err != nil {
				return err
			}
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
