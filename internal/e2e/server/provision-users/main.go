//go:build e2e

// provision-users is a tool for creating test users and API keys in an Immich instance.
// It is called by the immich-provision-users.sh script.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <IMMICH_URL> <OUTPUT_FILE>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s http://localhost:2283 ./e2eusers.yml\n", os.Args[0])
		os.Exit(1)
	}

	immichURL := os.Args[1]
	outputFile := os.Args[2]

	// Set the Immich URL as environment variable for the utilities
	os.Setenv("E2E_IMMICH_URL", immichURL)

	if err := provisionUsers(outputFile); err != nil {
		slog.Error("failed to provision users", "error", err)
		os.Exit(1)
	}

	slog.Info("users provisioned successfully", "output", outputFile)
}

var minimalPermissions = []string{
	`asset.read`,
	`asset.statistics`,
	`asset.update`,
	`asset.upload`,
	`asset.replace`,
	`asset.download`,
	`album.create`,
	`album.read`,
	`albumAsset.create`,
	`job.create`,
	`job.read`,
	`server.about`,
	`stack.create`,
	`tag.asset`,
	`tag.create`,
	`user.read`,
}

func provisionUsers(outputFile string) error {
	users := NewE2eUsers()

	// Create admin user
	adm, err := users.CreateAdminUser("admin@immich.app", "admin", "admin user")
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	if err := adm.AddUserKey("e2eAll", []string{"all"}); err != nil {
		return fmt.Errorf("failed to create admin API key: %w", err)
	}

	// Create test user 1
	u1, err := users.UserCreate("user1@immich.app", "user1", "user1")
	if err != nil {
		return fmt.Errorf("failed to create user1: %w", err)
	}
	if err := u1.AddUserKey("e2eMinimal", minimalPermissions); err != nil {
		return fmt.Errorf("failed to create user1 API key: %w", err)
	}

	// Create test user 2
	u2, err := users.UserCreate("user2@immich.app", "user2", "user2")
	if err != nil {
		return fmt.Errorf("failed to create user2: %w", err)
	}
	if err := u2.AddUserKey("e2eMinimal", minimalPermissions); err != nil {
		return fmt.Errorf("failed to create user2 API key: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(path.Dir(outputFile), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write credentials to file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if err := yaml.NewEncoder(f).Encode(users); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

type (
	e2eUsers struct {
		Users map[string]*e2eUser
		admTk e2eutils.Token
	}
	e2eUser struct {
		email    string
		Password string
		Token    e2eutils.Token
		Keys     map[string]string
	}
)

func NewE2eUsers() *e2eUsers {
	return &e2eUsers{
		Users: map[string]*e2eUser{},
	}
}

func (us *e2eUsers) CreateAdminUser(email, password, name string) (*e2eUser, error) {
	if err := e2eutils.AdminSetup(email, password, name); err != nil {
		return nil, err
	}
	u, err := us.UserLogin(email, password)
	if err != nil {
		return nil, err
	}
	us.admTk = u.Token
	slog.Info("admin user created", "user", email)
	return u, nil
}

func (us *e2eUsers) UserCreate(email, password, name string) (*e2eUser, error) {
	if err := e2eutils.CreateUser(us.admTk, email, password, name); err != nil {
		return nil, err
	}
	slog.Info("user created", "user", email)
	return us.UserLogin(email, password)
}

func (us *e2eUsers) UserLogin(email, password string) (*e2eUser, error) {
	tk, err := e2eutils.UserLogin(email, password)
	if err != nil {
		return nil, err
	}
	slog.Info("user logged in", "user", email)
	u := &e2eUser{
		email:    email,
		Password: password,
		Token:    tk,
		Keys:     map[string]string{},
	}
	us.Users[email] = u

	if err := e2eutils.SetUserOnboarding(tk, true); err != nil {
		return nil, err
	}
	slog.Info("user onboarded", "user", email)
	return u, nil
}

func (u *e2eUser) AddUserKey(keyName string, permissions []string) error {
	key, err := e2eutils.CreateApiKey(u.Token, keyName, permissions)
	if err != nil {
		return err
	}
	u.Keys[keyName] = key
	slog.Info("user key added", "user", u.email, "key", keyName)
	return nil
}
