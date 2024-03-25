package main

import (
	"fmt"
	"os"

	"github.com/simulot/immich-go/helpers/configuration"
)

func main() {
	c, _ := configuration.Read(configuration.DefaultFile())

	c, err := configuration.UI(c)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = c.Write(configuration.DefaultFile())
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
