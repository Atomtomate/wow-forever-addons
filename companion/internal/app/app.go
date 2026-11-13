// Package app is the companion's commands (setup, run, status, remove, version) wired from the
// packages beside it. It reaches the operating system only through Env, so its tests run a
// whole setup against a fake hub in a temporary folder.
package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/config"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/hubapi"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/install"
)

// Exit codes. The ones above 1 are for scripts: `run --once` and the non-interactive `setup`.
const (
	ExitOK          = 0
	ExitFailed      = 1
	ExitNotLinked   = 2 // setup: the code expired or was declined
	ExitUnlinked    = 3 // run --once: the hub no longer knows this PC
	ExitUnreachable = 4 // run --once: the hub could not be reached
	ExitRunning     = 5 // run --once: another companion is running
	ExitRefused     = 6 // run --once: the hub refused the body
	ExitThrottled   = 7 // run --once: something in front of the hub asked for less
	ExitInterrupted = 130
)

const (
	programFile = "ForeverFriendsCompanion.exe"
	stopWait    = 45 * time.Second
	logMaxBytes = 256 << 10
)

// Platform is every operating-system call the commands make.
type Platform struct {
	ConfigDir       func() (string, error)
	ProgramDir      func() string
	Executable      func() (string, error)
	GameCandidates  func() []string
	ListImages      func() ([]string, error)
	OpenURL         func(url string) error
	SetAutostart    func(exe string, args ...string) error
	RemoveAutostart func() error
	SpawnDetached   func(exe string, args ...string) error
	PrepareConsole  func()
	CaseFold        bool
}

// Env is everything the commands take from outside.
type Env struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Now      func() time.Time
	Sleep    func(context.Context, time.Duration) error
	HTTP     *http.Client
	Addon    iofs.FS
	P        Platform
	Hostname func() (string, error)
}

// Main runs one command and returns the process exit code.
func Main(ctx context.Context, args []string, env Env) int {
	cmd := "setup"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	// run is the background command and must never show a window.
	if cmd != "run" && env.P.PrepareConsole != nil {
		env.P.PrepareConsole()
	}
	if env.Stdin == nil {
		env.Stdin = os.Stdin
	}
	if env.Stdout == nil {
		env.Stdout = os.Stdout
	}
	if env.Stderr == nil {
		env.Stderr = os.Stderr
	}
	toc, err := install.ReadToc(env.Addon)
	if err != nil {
		fmt.Fprintln(env.Stderr, "This build is broken:", err)
		return ExitFailed
	}
	c := &cmdCtx{env: env, version: toc.Version, iface: toc.Interface, ui: newUI(env)}
	switch cmd {
	case "setup":
		return c.setup(ctx, args)
	case "run":
		return c.run(ctx, args)
	case "status":
		return c.status(args)
	case "remove":
		return c.remove(ctx, args)
	case "version":
		fmt.Fprintln(env.Stdout, c.version)
		return ExitOK
	}
	fmt.Fprintf(env.Stderr, "Unknown command %q. Commands: setup (the default), run, status, remove, version.\n", cmd)
	return ExitFailed
}

type cmdCtx struct {
	env     Env
	version string
	iface   string
	ui      *ui
}

func (c *cmdCtx) userAgent() string { return "ForeverFriendsCompanion/" + c.version }

func (c *cmdCtx) flags(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.env.Stderr)
	return fs, fs.String("config-dir", "", "folder for the key, the link and the log (default: this user's)")
}

func (c *cmdCtx) configDir(flagValue string) (config.Dir, error) {
	d := flagValue
	if d == "" {
		var err error
		if d, err = c.env.P.ConfigDir(); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	return config.Dir(d), nil
}

// client builds the signed client for the linked device in dir. The hub's address is checked
// again here: the config file is a file anyone on the PC can edit.
func (c *cmdCtx) client(dir config.Dir, cfg config.Config, store *config.Store) (*hubapi.Client, error) {
	key, err := dir.LoadKey()
	if err != nil {
		return nil, fmt.Errorf("the key is unreadable: %w", err)
	}
	hub, err := hubapi.ParseOrigin(cfg.Hub)
	if err != nil {
		return nil, fmt.Errorf("the saved site address is unusable: %w", err)
	}
	return &hubapi.Client{Origin: hub, Device: cfg.Device, Key: key, Seq: store, HTTP: c.env.HTTP, UserAgent: c.userAgent()}, nil
}

func (c *cmdCtx) isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func (c *cmdCtx) samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if c.env.P.CaseFold {
		return strings.EqualFold(a, b)
	}
	return a == b
}

type ui struct {
	in  *bufio.Reader
	out io.Writer
	yes bool
}

func newUI(env Env) *ui { return &ui{in: bufio.NewReader(env.Stdin), out: env.Stdout} }

func (u *ui) say(format string, a ...any) { fmt.Fprintf(u.out, format+"\n", a...) }

// ask returns the answer, or def on an empty line, at the end of input, or with --yes.
func (u *ui) ask(prompt, def string) string {
	if u.yes {
		return def
	}
	if def != "" {
		fmt.Fprintf(u.out, "%s [%s]: ", prompt, def)
	} else {
		fmt.Fprintf(u.out, "%s: ", prompt)
	}
	line, err := u.in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" && (err == nil || def != "") {
		return def
	}
	return line
}

// confirm is a yes/no question that --yes answers yes.
func (u *ui) confirm(prompt string, defYes bool) bool {
	if u.yes {
		return true
	}
	return u.confirmRisky(prompt, defYes)
}

// confirmRisky is one --yes never answers: it guards something the member did not ask for.
func (u *ui) confirmRisky(prompt string, defYes bool) bool {
	if u.yes {
		return false
	}
	hint := " [y/N] "
	if defYes {
		hint = " [Y/n] "
	}
	fmt.Fprint(u.out, prompt+hint)
	line, _ := u.in.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return defYes
}

var errCannotStop = errors.New("the running companion did not stop; end ForeverFriendsCompanion in Task Manager and run this again")
