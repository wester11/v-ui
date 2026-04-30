// void-wg — enhanced obfuscated WireGuard client (AmneziaWG compatible).
//
// Commands:
//   up     <config.vwg>          bring tunnel up with obfuscation
//   down   <name|config.vwg>     bring tunnel down
//   status <name>                show stats + obfs params
//   import <config.conf>         import standard WG config
//   upgrade <config.conf>        import .conf and add default obfs params -> .vwg
//   genconf                      print a template .vwg config
//   version                      print version
//
// Flags:
//   --kill-switch                enable kill switch
//   --bypass <CIDR,...>          subnets that bypass kill switch
//   --jc <n>                     override Jc (junk count)
//   --jmin <n>                   override Jmin
//   --jmax <n>                   override Jmax

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/voidwg/void-wg/internal/config"
	"github.com/voidwg/void-wg/internal/obfs"
	"github.com/voidwg/void-wg/internal/tunnel"
)

const version = "0.1.0"

func main() {
	fs := flag.NewFlagSet("void-wg", flag.ExitOnError)
	ks      := fs.Bool("kill-switch", false, "enable kill switch (drop non-VPN traffic)")
	bypass  := fs.String("bypass", "", "comma-separated CIDRs that bypass kill switch")
	confDir := fs.String("conf-dir", "/etc/void-wg", "config directory")
	jc      := fs.Int("jc", 0, "override Jc junk count")
	jmin    := fs.Int("jmin", 0, "override Jmin junk min size")
	jmax    := fs.Int("jmax", 0, "override Jmax junk max size")
	ver     := fs.Bool("version", false, "print version")
	_ = fs.Parse(os.Args[1:])

	if *ver {
		fmt.Println("void-wg", version)
		os.Exit(0)
	}

	args := fs.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	// Ensure conf dir exists
	_ = os.MkdirAll(*confDir, 0700)

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "up":
		if len(rest) == 0 {
			fatal("up requires <config.vwg>")
		}
		confPath := resolveConf(rest[0], *confDir)
		cfg, err := config.ParseFile(confPath)
		dieIf(err, "parse config")

		// Apply flag overrides
		if *jc > 0 {
			cfg.Obfs.Jc = uint8(*jc)
		}
		if *jmin > 0 {
			cfg.Obfs.Jmin = uint16(*jmin)
		}
		if *jmax > 0 {
			cfg.Obfs.Jmax = uint16(*jmax)
		}

		t := tunnel.New(cfg, *confDir)
		t.KillSwitch = *ks
		if *bypass != "" {
			t.BypassNets = strings.Split(*bypass, ",")
		}
		dieIf(t.Up(), "up")

		if cfg.Obfs.Enabled {
			fmt.Printf("[void-wg] tunnel up (obfuscated) Jc=%d Jmin=%d Jmax=%d\n",
				cfg.Obfs.Jc, cfg.Obfs.Jmin, cfg.Obfs.Jmax)
		} else {
			fmt.Println("[void-wg] tunnel up (plain WireGuard)")
		}

	case "down":
		if len(rest) == 0 {
			fatal("down requires <name|config.vwg>")
		}
		name := tunnelName(rest[0])
		confPath := filepath.Join(*confDir, name+".vwg")
		cfg, err := config.ParseFile(confPath)
		dieIf(err, "parse config")
		t := tunnel.New(cfg, *confDir)
		dieIf(t.Down(), "down")
		fmt.Printf("[void-wg] tunnel %s down\n", name)

	case "status":
		if len(rest) == 0 {
			fatal("status requires <name>")
		}
		confPath := resolveConf(rest[0], *confDir)
		cfg, err := config.ParseFile(confPath)
		dieIf(err, "parse config")
		t := tunnel.New(cfg, *confDir)
		st, err := t.Status()
		dieIf(err, "status")
		fmt.Println(st)
		if cfg.Obfs.Enabled {
			fmt.Printf("\nObfuscation: ON\n")
			fmt.Printf("  Jc=%d  Jmin=%d  Jmax=%d\n", cfg.Obfs.Jc, cfg.Obfs.Jmin, cfg.Obfs.Jmax)
			fmt.Printf("  S1=%d  S2=%d\n", cfg.Obfs.S1, cfg.Obfs.S2)
			fmt.Printf("  H1=%d  H2=%d  H3=%d  H4=%d\n", cfg.Obfs.H1, cfg.Obfs.H2, cfg.Obfs.H3, cfg.Obfs.H4)
		}

	case "import":
		if len(rest) == 0 {
			fatal("import requires <config.vwg|config.conf>")
		}
		src := rest[0]
		cfg, err := config.ParseFile(src)
		dieIf(err, "parse config")
		dst := filepath.Join(*confDir, cfg.Name+".vwg")
		dieIf(writeConfig(cfg, dst), "write config")
		fmt.Printf("Imported %s -> %s\n", src, dst)

	case "upgrade":
		if len(rest) == 0 {
			fatal("upgrade requires <config.conf>")
		}
		cfg, err := config.UpgradeFromWG(rest[0])
		dieIf(err, "upgrade")
		dst := filepath.Join(*confDir, cfg.Name+".vwg")
		dieIf(writeConfig(cfg, dst), "write config")
		fmt.Printf("Upgraded %s -> %s (obfuscation added)\n", rest[0], dst)
		fmt.Printf("  Jc=%d  Jmin=%d  Jmax=%d\n", cfg.Obfs.Jc, cfg.Obfs.Jmin, cfg.Obfs.Jmax)

	case "genconf":
		p := config.DefaultObfsParams()
		fmt.Printf(`[Interface]
PrivateKey = <your-private-key>
Address = 10.0.0.2/32
DNS = 1.1.1.1

# VoidWG obfuscation (AmneziaWG compatible)
Obfuscation = on
Jc   = %d
Jmin = %d
Jmax = %d
S1   = 0
S2   = 0
H1   = 1
H2   = 2
H3   = 3
H4   = 4

[Peer]
PublicKey = <server-public-key>
Endpoint = <server-ip>:51821
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`, p.Jc, p.Jmin, p.Jmax)

	case "obfs-test":
		// Quick sanity test of the obfs layer
		if len(rest) < 2 {
			fatal("obfs-test <local-addr> <remote-addr>")
		}
		proxy := &obfs.Proxy{
			LocalAddr:  rest[0],
			RemoteAddr: rest[1],
			Params: obfs.Params{
				Jc: 4, Jmin: 40, Jmax: 70,
				H1: 1, H2: 2, H3: 3, H4: 4,
			},
		}
		dieIf(proxy.Start(), "start proxy")
		fmt.Printf("Obfs proxy %s -> %s  (Ctrl+C to stop)\n", rest[0], rest[1])
		select {}

	case "version":
		fmt.Println("void-wg", version)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func resolveConf(arg, confDir string) string {
	if strings.HasSuffix(arg, ".vwg") || strings.HasSuffix(arg, ".conf") {
		return arg
	}
	// try .vwg first, then .conf
	vwg := filepath.Join(confDir, arg+".vwg")
	if _, err := os.Stat(vwg); err == nil {
		return vwg
	}
	return filepath.Join(confDir, arg+".conf")
}

func tunnelName(arg string) string {
	name := filepath.Base(arg)
	name = strings.TrimSuffix(name, ".vwg")
	name = strings.TrimSuffix(name, ".conf")
	return name
}

func writeConfig(cfg *config.Config, dst string) error {
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".vwg-*.tmp")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(tmp, cfg.Marshal()); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = tmp.Close()
	_ = os.Chmod(tmp.Name(), 0600)
	return os.Rename(tmp.Name(), dst)
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
	fmt.Println(`void-wg ` + version + ` — obfuscated WireGuard client

Usage:
  void-wg [flags] <command> [args]

Commands:
  up      <conf.vwg>        Bring tunnel up with obfuscation
  down    <name|conf.vwg>   Bring tunnel down
  status  <name|conf.vwg>   Show connection stats + obfs params
  import  <conf.vwg|.conf>  Install config to /etc/void-wg/
  upgrade <conf.conf>       Upgrade standard WG config to .vwg with obfuscation
  genconf                   Print a template .vwg config
  obfs-test <local> <remote> Test obfuscation proxy
  version                   Print version

Flags:
  --kill-switch        Enable kill switch
  --bypass <CIDR,...>  CIDRs that bypass kill switch
  --conf-dir <dir>     Config directory (default /etc/void-wg)
  --jc <n>             Override Jc (junk packet count)
  --jmin <n>           Override Jmin (junk min size)
  --jmax <n>           Override Jmax (junk max size)

.vwg format extends standard WireGuard .conf with:
  Obfuscation = on
  Jc / Jmin / Jmax / S1 / S2 / H1 / H2 / H3 / H4

Examples:
  sudo void-wg up myvpn.vwg --kill-switch
  sudo void-wg upgrade ~/Downloads/myvpn.conf
  void-wg status myvpn
  void-wg genconf > template.vwg`)
}
