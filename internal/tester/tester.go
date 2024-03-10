// Package tester implement some test functions in real immich server
//
//

package tester

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

type ImmichController struct {
	Env            map[string]string
	ComposePath    string
	ComposeEnvPath string
	AppPath        string
	AppURL         string
	UploadLocation string
	APIKey         string
	AccessToken    string
}

func (s *ImmichController) Down(ctx context.Context) error {
	return s.DockerCompose(ctx, "down")
}

func (s *ImmichController) Up(ctx context.Context) error {
	return s.DockerCompose(ctx, "up", "-d")
}

// FactoryReset recreates a fresh install of the immich server
// - It removes existing docker volumes
// - It removes existing images in
//		- encoded-video
//		- library
//		- thumbs
//		- upload
// - It creates an API Key

func (s *ImmichController) FactoryReset(ctx context.Context) error {
	_ = s.Down(ctx)

	err := s.RemoveDockerVolumes(ctx)
	if err != nil {
		return err
	}

	err = s.Up(ctx)
	if err != nil {
		return err
	}

	err = s.WaitImmich(ctx, 1*time.Minute)
	if err != nil {
		return err
	}

	err = s.CreateAdminLogin(ctx)
	if err != nil {
		return err
	}

	err = s.CreateAPIKey(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *ImmichController) RemoveDockerVolumes(ctx context.Context) error {
	appName := path.Base(s.AppPath)

	volumes, err := s.DockerVolumes(ctx)
	if err != nil {
		return err
	}
	for _, v := range volumes {
		if strings.HasPrefix(v, appName) {
			fmt.Println("removing volume", v)
			o, err := s.DockerRun(ctx, "volume", "rm", v)
			if err != nil {
				fmt.Println("->", err.Error(), "\n", string(o))
				return err
			}
		}
	}
	o, err := s.DockerRun(ctx, "system", "prune", "-f")
	if err != nil {
		fmt.Println(string(o))
		return fmt.Errorf("can't prune docker: %w", err)
	}
	//	rm := "rm -rf " + strings.Join([]string{"encoded-video", "library", "thumbs", "upload"}, " ")
	o, err = SudoRun(ctx, "rm", "-rf", "encoded-video", "library", "thumbs", "upload")
	if err != nil {
		fmt.Println(string(o))
		return fmt.Errorf("can't remove files: %w", err)
	}

	return nil
}

func (s *ImmichController) WaitImmich(ctx context.Context, wait time.Duration) error {
	stopAt := time.Now().Add(wait)
	for time.Now().Before(stopAt) {
		r, err := http.Get(s.AppURL + "/api/server-info/ping")
		if err == nil {
			r.Body.Close()
			if r.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return errors.New("can't join the immich server in the given time")
}

func (s *ImmichController) CreateAdminLogin(ctx context.Context) error {
	r, err := http.Post(s.AppURL+"/api/auth/admin-sign-up", "application/json", strings.NewReader(`{
		"email": "demo@immich.app",
		"password": "demo",
		"name": "John Doe"
	}`))
	if err == nil {
		r.Body.Close()
		if r.StatusCode == http.StatusCreated {
			return nil
		}
		return fmt.Errorf("can't create admin login: %s", r.Status)
	}
	return fmt.Errorf("can't create admin login: %w", err)
}

func (s *ImmichController) CreateAPIKey(ctx context.Context) error {
	client := http.Client{}

	resp, err := client.Post(s.AppURL+"/api/auth/login", "application/json", strings.NewReader(`{"email": "demo@immich.app", "password": "demo"}`))
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			for _, c := range resp.Cookies() {
				if c.Name == "immich_access_token" {
					s.AccessToken = c.Value
					break
				}
			}
			if s.AccessToken == "" {
				return fmt.Errorf("can't get the accessToken")
			}
		} else {
			return fmt.Errorf("can't get the accessToken: %s", resp.Status)
		}
	}
	if err != nil {
		return fmt.Errorf("can't get the accessToken: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.AppURL+"/api/api-key", strings.NewReader(`{"name": "Test controller"}`))
	if err != nil {
		return fmt.Errorf("can't get the API Key: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+s.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err == nil {
		if resp.StatusCode == http.StatusCreated {
			k := struct {
				Secret string `json:"secret"`
			}{}
			err = json.NewDecoder(resp.Body).Decode(&k)
			s.APIKey = k.Secret
			resp.Body.Close()
		} else {
			return fmt.Errorf("can't get the API Key: %s", resp.Status)
		}
	}
	if err != nil || s.APIKey == "" {
		return fmt.Errorf("can't get the API Key: %w", err)
	}
	fmt.Println("APIKey = ", s.APIKey)
	return nil
}
