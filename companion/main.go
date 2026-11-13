// Command companion is the Forever Friends companion: it links a member's PC to the group's
// site, installs the ForeverFriends addon it carries, uploads the member's characters and
// tells the site whether the game is running. All of it lives in internal/app.
package main

import (
	"context"
	"os"
	"os/signal"

	foreverfriends "github.com/Atomtomate/wow-forever-addons"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := app.Main(ctx, os.Args[1:], app.DefaultEnv(foreverfriends.Addon()))
	stop()
	os.Exit(code)
}
