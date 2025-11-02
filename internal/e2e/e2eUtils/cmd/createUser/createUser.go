package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	if os.Args[1] == "-" {
		buff, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("Error:", err.Error())
			os.Exit(1)
		}
		for line := range bytes.Lines(buff) {
			args := strings.Fields(string(line))
			if len(args) > 0 {
				err = run(args)
			}
			if err != nil {
				fmt.Println("Error:", err.Error())
				os.Exit(1)
			}
		}
	} else {
		err := run(os.Args[1:])
		if err != nil {
			fmt.Println("ERROR:", err.Error())
			os.Exit(1)
		}
	}
}

func run(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: createUser <command> [args]")
	}

	cmd := args[0]
	switch cmd {
	case "create-admin":
		email := "admin@immich.app"
		password := "admin"
		return createAdmin(email, password, "all")
	case "create-user":
		if len(args) < 3 {
			return fmt.Errorf("missing user name or password")
		}
		email := args[1]
		password := args[2]
		return createUser(email, password, "minimal")
	}
	return fmt.Errorf("unknown command: %s", cmd)
}

func createAdmin(email, password, keyName string) error {
	err := e2eutils.AdminSetup(email, password, email)
	if err != nil {
		return err
	}
	tk, err := e2eutils.UserLogin(email, password)
	if err != nil {
		return err
	}
	err = e2eutils.SetUserOnboarding(tk, true)
	if err != nil {
		return err
	}
	key, err := createAPIKey(tk, keyName)
	if err != nil {
		return err
	}
	fmt.Printf("E2E_%s_PASSWORD=%s\n", email, password)
	fmt.Printf("E2E_%s_APIKEY=%s\n", email, key)
	return err
}

func createUser(email, password, keyName string) error {
	admtk, err := e2eutils.UserLogin("admin@immich.app", "admin")
	if err != nil {
		return err
	}
	err = e2eutils.CreateUser(admtk, email, password, email)
	if err != nil {
		return err
	}
	tk, err := e2eutils.UserLogin(email, password)
	if err != nil {
		return err
	}
	err = e2eutils.SetUserOnboarding(tk, true)
	if err != nil {
		return err
	}
	key, err := createAPIKey(tk, keyName)
	if err != nil {
		return err
	}
	fmt.Printf("E2E_%s_PASSWORD=%s\n", email, password)
	fmt.Printf("E2E_%s_APIKEY=%s\n", email, key)
	return err
}

func createAPIKey(tk e2eutils.Token, keyName string) (string, error) {
	p := permissions[keyName]
	if p == nil {
		return "", fmt.Errorf("unknown key name: %s", keyName)
	}

	key, err := e2eutils.CreateApiKey(tk, keyName, p)
	if err != nil {
		return "", err
	}
	return key, nil
}

var permissions map[string][]string = map[string][]string{
	"minimal": {
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
	},
	"all": {
		"all",
	},
}

/*


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

func (us *e2eUsers) UserCreate(email, password, permission string) (*e2eUser, error) {
	err := e2eutils.CreateUser(us.admTk, email, password,)
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

*/
