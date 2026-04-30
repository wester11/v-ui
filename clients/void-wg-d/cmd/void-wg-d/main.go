// void-wg-d — standard WireGuard compatible client with extras.
//
// Commands:
//   up   <config.conf>       bring tunnel up
//   down <name|config.conf>  bring tunnel down
//   status <name>            show stats
//   import <config.conf>     copy config to /etc/wireguard/
//   list                     show running tunnels
//
// Flags:
//   --kill-switch            enable kill switch (drop all non-VPN traffic)
//   --bypass <item,...>      IPs/CIDRs/domains routed via original gateway (split tunneling).
//                            Also exempts them from the kill switch if --kill-switch is set.
//   --conf-dir <dir>         config directory (default /etc/wireguard)

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/voidwg/void-wg-d/internal/config"
	"github.com/voidwg/void-wg-d/internal/killswitch"
	"github.com/voidwg/void-wg-d/internal/tunnel"
)

const version = "0.1.0"

func main() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	fs := flag.NewFlagSet("void-wg-d", flag.ExitOnError)
	ks      := fs.Bool("kill-switch", false, "enable kill switch")
	bypass  := fs.String("bypass", "", "comma-separated CIDRs that bypass kill switch")
	confDir := fs.String("conf-dir", "/etc/wireguard", "directory for WireGuard configs")
	ver     := fs.Bool("version", false, "print version")
	_ = fs.Parse(os.Args[1:])

	if *ver {
		fmt.Println("void-wg-d", version)
		os.Exit(0)
	}

	args := fs.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "up":
		if len(rest) == 0 {
			fatal("up requires <config.conf>")
		}
		confPath := resolveConf(rest[0], *confDir)
		cfg, err := config.ParseFile(confPath)
		dieIf(err, "parse config")

		t := tunnel.New(cfg)
		t.KillSwitch = *ks
		if *bypass != "" {
			nets := strings.Split(*bypass, ",")
			// BypassNets is used by kill switch; SplitBypass drives actual route injection.
			t.BypassNets  = nets
			t.SplitBypass = nets
		}
		dieIf(t.Up(), "up")
		log.Info().Str("interface", cfg.Name).Msg("tunnel up")

	case "down":
		if len(rest) == 0 {
			fatal("down requires <name|config.conf>")
		}
		name := tunnelName(rest[0])
		confPath := filepath.Join(*confDir, name+".conf")
		cfg, err := config.ParseFile(confPath)
		dieIf(err, "parse config")

		t := tunnel.New(cfg)
		dieIf(t.Down(), "down")
		log.Info().Str("interface", name).Msg("tunnel down")

	case "status":
		if len(rest) == 0 {
			fatal("status requires <name>")
		}
		confPath := resolveConf(rest[0], *confDir)
		cfg, err := config.ParseFile(confPath)
		dieIf(err, "parse config")

		t := tunnel.New(cfg)
		st, err := t.Status()
		dieIf(err, "status")
		fmt.Println(st)

	case "import":
		if len(rest) == 0 {
			fatal("import requires <config.conf>")
		}
		src := rest[0]
		cfg, err := config.ParseFile(src)
		dieIf(err, "parse config")

		dst := filepath.Join(*confDir, cfg.Name+".conf")
		dieIf(copyFile(src, dst), "copy config")
		dieIf(os.Chmod(dst, 0600), "chmod")
		fmt.Printf("Imported %s -> %s\n", src, dst)

	case "kill-switch":
		if len(rest) == 0 {
			fmt.Println("kill-switch enabled:", killswitch.IsEnabled())
			return
		}
		switch rest[0] {
		case "on":
			iface := "wg0"
			if len(rest) > 1 {
				iface = rest[1]
			}
			var nets []string
			if *bypass != "" {
				nets = strings.Split(*bypass, ",")
			}
			dieIf(killswitch.Enable(iface, nets), "kill-switch enable")
			fmt.Println("kill switch enabled")
		case "off":
			dieIf(killswitch.Disable(), "kill-switch disable")
			fmt.Println("kill switch disabled")
		default:
			fatal("kill-switch [on|off] [iface]")
		}

	case "version":
		fmt.Println("void-wg-d", version)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func resolveConf(arg, confDir string) string {
	if strings.HasSuffix(arg, ".conf") {
		return arg
	}
	return filepath.Join(confDir, arg+".conf")
}

func tunnelName(arg string) string {
	name := filepath.Base(arg)
	return strings.TrimSuffix(name, ".conf")
}

func dieIf(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s: %v\n", msg, err)
		os.Exit(1)
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}

func usage() {
	fmt.Println(`void-wg-d ` + version + ` — WireGuard-compatible client

Usage:
  void-wg-d [flags] <command> [args]

Commands:
  up    <conf>        Bring tunnel up (reads .conf file)
  down  <name|conf>   Bring tunnel down
  status <name|conf>  Show peer stats and split-bypass routes
  import <conf>       Install config to /etc/wireguard/
  kill-switch on|off  Manage kill switch independently
  version             Print version

Flags:
  --kill-switch                  Enable kill switch when bringing tunnel up
  --bypass <item,item,...>        Split tunneling: IPs, CIDRs, or domain names that are
                                  routed via the original gateway (bypassing the VPN).
                                  Domains are resolved to A-records at connect time.
                                  Also exempts these from the kill switch if enabled.
  --conf-dir <dir>               Config directory (default /etc/wireguard)

Examples:
  # Route only specific IPs/domains outside VPN, block everything else:
  sudo void-wg-d up vpn.conf --kill-switch --bypass 8.8.8.8,example.com,192.168.1.0/24

  # Split tunneling without kill switch:
  sudo void-wg-d up vpn.conf --bypass google.com,1.1.1.1

  sudo void-wg-d status wg0
  sudo void-wg-d down wg0
  void-wg-d import ~/Downloads/myvpn.conf`)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
