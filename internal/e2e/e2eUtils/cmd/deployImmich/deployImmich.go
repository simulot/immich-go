package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"go.yaml.in/yaml/v3"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("can't get immich", "error", err.Error())
		os.Exit(1)
	}
	slog.Info("immich deploy: done")
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

func run() error {
	ctx := context.Background()

	if len(os.Args) < 2 {
		return errors.New("missing path to immich installation")
	}

	curWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't get current working directory: %w", err)
	}
	defer func() {
		_ = os.Chdir(curWD)
	}()

	ictl, err := e2eutils.NewImmichController(os.Args[1])
	if err != nil {
		return fmt.Errorf("can't create immich controller: %w", err)
	}

	err = ictl.DeployImmich(ctx)
	if err != nil {
		return err
	}

	users := NewE2eUsers()
	adm, err := users.CreateAdminUser("admin@immich.app", "admin", "admin user")
	if err != nil {
		return err
	}
	err = adm.AddUserKey("e2eAll", []string{"all"})
	if err != nil {
		return err
	}

	u, err := users.UserCreate("user1@immich.app", "user1", "user1")
	if err != nil {
		return err
	}
	err = u.AddUserKey("e2eMinimal", minimalPermissions)
	if err != nil {
		return err
	}

	u, err = users.UserCreate("user2@immich.app", "user2", "user2")
	if err != nil {
		return err
	}
	err = u.AddUserKey("e2eMinimal", minimalPermissions)
	if err != nil {
		return err
	}

	e2eUsersName := path.Join(path.Dir(ictl.GetDockerComposeFile()), "e2eusers.yml")
	f, err := os.Create(e2eUsersName)
	if err != nil {
		return fmt.Errorf("can't store ue2e users: %w", err)
	}
	defer f.Close()
	err = yaml.NewEncoder(f).Encode(users)
	if err != nil {
		return fmt.Errorf("can't store ue2e users: %w", err)
	}
	return err
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
	err := e2eutils.AdminSetup(email, password, name)
	if err != nil {
		return nil, err
	}
	u, err := us.UserLogin(email, password)
	if err != nil {
		return nil, err
	}
	us.admTk = u.Token
	slog.Info("admin user created", "user", email)
	return u, err
}

func (us *e2eUsers) UserCreate(email, password, name string) (*e2eUser, error) {
	err := e2eutils.CreateUser(us.admTk, email, password, name)
	if err != nil {
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

	err = e2eutils.SetUserOnboarding(tk, true)
	if err != nil {
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
