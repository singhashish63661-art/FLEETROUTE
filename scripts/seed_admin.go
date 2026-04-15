package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := "postgres://gpsgo:gpsgo@localhost:5432/gpsgo?sslmode=disable"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	// In Go, bcrypt hash requires a cost. MinCost is 4, DefaultCost is 10.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Error hashing password: %v\n", err)
	}

	// Create a default tenant
	var tenantID string
	err = conn.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, plan)
		VALUES ('Default Tenant', 'default', 'enterprise')
		ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name
		RETURNING id;
	`).Scan(&tenantID)
	if err != nil {
		log.Fatalf("Error creating tenant: %v\n", err)
	}
	fmt.Printf("Default Tenant ID: %s\n", tenantID)

	// Create admin user
	var userID string
	err = conn.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, name, role)
		VALUES ($1, 'admin@gpsgo.com', $2, 'System Admin', 'super_admin')
		ON CONFLICT (tenant_id, email) DO UPDATE SET password_hash=EXCLUDED.password_hash
		RETURNING id;
	`, tenantID, string(hashedPassword)).Scan(&userID)

	if err != nil {
		log.Fatalf("Error creating user: %v\n", err)
	}

	fmt.Printf("Admin User ID: %s\n", userID)
	fmt.Println("Successfully seeded admin user!")
	fmt.Println("Email: admin@gpsgo.com")
	fmt.Println("Password: admin123")
}
