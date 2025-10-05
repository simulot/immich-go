package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	force := flag.Bool("force", false, "force reset")
	flag.Parse()
	if *force {
		if err := ResetImmich(); err != nil {
			os.Exit(1)
		}
	} else {
		fmt.Println("use -force to reset")
		os.Exit(1)
	}
}

func ResetImmich() error {
	// Reset immich's database
	// https://github.com/immich-app/immich/blob/main/e2e/src/utils.ts
	//
	c := exec.Command("docker", "exec", "-i", "immich_postgres", "psql", "--dbname=immich", "--username=postgres", "-c",
		`
        delete from stack CASACDE;
        delete from library CASACDE;
        delete from shared_link CASACDE;
        delete from person CASACDE;
        delete from album CASACDE;
        delete from asset CASACDE;
        delete from asset_face CASACDE;
        delete from activity CASACDE;
        delete from tag CASACDE;
        -- delete from api_key CASACDE;
        -- delete from session CASACDE;
        -- delete from user CASACDE;
        -- delete from system_metadata where "key" NOT IN ('reverse-geocoding-state', 'system-flags');
		`,
	)
	out, err := c.CombinedOutput()
	if out != nil {
		fmt.Println(string(out))
	}
	return err
}
