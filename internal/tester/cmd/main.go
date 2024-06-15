package main

import (
	"context"
	"fmt"
	"path"

	"github.com/joho/godotenv"
	"github.com/simulot/immich-go/internal/tester"
)

const EnvFile = "../../../e2e.env"

func main() {
	ctx := context.Background()

	err := run(ctx)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func run(ctx context.Context) error {
	var err error
	myEnv, err := godotenv.Read(EnvFile)
	if err != nil {
		return fmt.Errorf("can't read .env file: %w", err)
	}

	tester := tester.ImmichController{
		Env:            myEnv,
		ComposePath:    path.Join(myEnv["IMMICH_PATH"], "docker-compose.yml"),
		ComposeEnvPath: path.Join(myEnv["IMMICH_PATH"], ".env"),
		AppPath:        myEnv["IMMICH_PATH"],
		AppURL:         myEnv["IMMICH_HOST"],
	}
	err = tester.FactoryReset(ctx)
	if err != nil {
		return err
	}
	myEnv["IMMICH_KEY"] = tester.APIKey
	err = godotenv.Write(myEnv, EnvFile)

	return err
}
