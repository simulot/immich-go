package client

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"testing"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

// Configuration from environment variables
var (
	ProjectDir = getEnv("project_dir", getProjectDir())
	ImmichURL  = getEnv("e2e_url", "http://localhost:2283")
	// sshHost    = getEnv("e2e_ssh", "")
)

func debug(t *testing.T) {
	e := os.Environ()
	for _, v := range e {
		if strings.HasPrefix(v, "e2e") {
			t.Logf("Env: %s", v)
		}
	}
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getProjectDir() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	o, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(o))
}

type user struct {
	Email,
	Password, APIKey string
}

var (
	users     = map[string]user{}
	userErr   error
	onceUsers sync.Once
)

func readUsers() (map[string]user, error) {
	onceUsers.Do(func() {
		f, err := os.ReadFile(path.Join(ProjectDir, "internal", "e2e", "testdata", "immich-server", "e2eusers.env"))
		if err != nil {
			userErr = err
			return
		}
		for l := range bytes.Lines(f) {
			parts := strings.SplitN(string(l), "=", 2)
			if len(parts) != 2 {
				userErr = fmt.Errorf("malformed user line : %s", l)
				return
			}
			parts[0] = strings.TrimPrefix(parts[0], "E2E_")

			parts[1] = strings.TrimSuffix(parts[1], "\n")
			parts[1] = strings.TrimSuffix(parts[1], "\r")

			emailAndType := strings.Split(parts[0], "_")
			if len(emailAndType) != 2 {
				userErr = fmt.Errorf("unexpected format for : %s", parts[0])
				return
			}
			u := users[emailAndType[0]]
			switch emailAndType[1] {
			case "PASSWORD":
				u.Password = parts[1]
			case "APIKEY":
				u.APIKey = parts[1]
			default:
				return
			}
			users[emailAndType[0]] = u
		}
	})
	return users, userErr
}

func writeUsers(users map[string]user) error {
	f, err := os.Create(path.Join(ProjectDir, "internal", "e2e", "testdata", "immich-server", "e2eusers.env"))
	if err != nil {
		return err
	}
	for email, u := range users {
		_, err := f.WriteString(fmt.Sprintf("E2E_%s_PASSWORD=%s\n", email, u.Password))
		if err != nil {
			return err
		}
		_, err = f.WriteString(fmt.Sprintf("E2E_%s_APIKEY=%s\n", email, u.APIKey))
		if err != nil {
			return err
		}
	}
	return nil
}

func getUser(email string) (user, error) {
	us, err := readUsers()
	if err != nil {
		return user{}, err
	}
	if u, ok := us[email]; ok {
		return u, nil
	}
	return user{}, errors.New("not found")
}

func createUser(keyName string) (user, error) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		return user{}, err
	}
	admtk, err := e2eutils.UserLogin("admin@immich.app", adm.Password)
	if err != nil {
		return user{}, err
	}
	name := randomString(8)
	email := name + "@immich.app"
	password := name
	u := user{Password: password, Email: email}

	err = e2eutils.CreateUser(admtk, email, password, email)
	if err != nil {
		return u, err
	}
	tk, err := e2eutils.UserLogin(email, password)
	if err != nil {
		return u, err
	}
	err = e2eutils.SetUserOnboarding(tk, true)
	if err != nil {
		return u, err
	}
	key, err := createAPIKey(tk, keyName)
	if err != nil {
		return u, err
	}
	u.APIKey = key
	users[email] = u
	err = writeUsers(users)
	return u, err
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
		`asset.copy`,
		`asset.replace`,
		`asset.delete`,
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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}
