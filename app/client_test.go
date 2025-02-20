package app

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

// Unittesting for the OpenClient function
// Done with Mock testing
// Defining Mock variables

// test case meant to reach the first branch
func TestOpenClientServerName(t *testing.T) {
	ctx := context.Background()
	cmd := &cobra.Command{}
	app := &Application{
		client: Client{
			Server: "http://my-server.com/",
		},
		log: &Log{},
	}

	OpenClient(ctx, cmd, app)

	if app.client.Server != "http://my-server.com" {
		t.Errorf("Trailing / not removed as expected")
	}

}

// test case meant to the reach the second branch
func TestOpenClienTimeZone(t *testing.T) {
	ctx := context.Background()
	cmd := &cobra.Command{}
	app := &Application{
		client: Client{
			Server:   "",
			TimeZone: "America/Los_Angeles",
		},
		log: &Log{},
	}

	OpenClient(ctx, cmd, app)

	if app.client.TZ.String() != "America/Los_Angeles" {
		t.Errorf("Initialized wrong TimeZone")
	}
}
