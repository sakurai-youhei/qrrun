// Command qrrun tunnels a local script file and displays a QR code so that a
// mobile device can run the script by scanning it.
//
// Usage:
//
//	qrrun your-local-script.py
//	echo "print('hello')" | qrrun -
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakurai-youhei/qrrun/internal/app"
)

var (
	version = "v0.0.0-dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var transportName string
	var runtimeName string
	var keepServing bool
	var exitQuietPeriod time.Duration
	var transportStderr bool
	var transportStdout bool
	var debug bool
	var printURL bool
	var transportOpts string

	cmd := &cobra.Command{
		Use:   "qrrun [flags] <script|-]",
		Short: "tunnel local scripts and run via QR",
		Long: `QRrun serves a local script through a secure tunnel and prints a QR code.

scan the QR code to open the script in the selected runtime.

Prerequisites:
	cloudflared must be installed and available on PATH
	QRrun uses Cloudflare Quick Tunnels (trycloudflare.com)

Examples:
	qrrun hello.py
	echo "print('hello')" | qrrun -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keepServingMode := keepServing
			return app.Run(app.Options{
				TransportName:   transportName,
				RuntimeName:     runtimeName,
				ScriptPath:      args[0],
				KeepServing:     keepServingMode,
				ExitQuietPeriod: exitQuietPeriod,
				Debug:           debug,
				TransportStdout: transportStdout,
				TransportStderr: transportStderr,
				PrintURL:        printURL,
				CloudflaredOpts: strings.Fields(transportOpts),
			})
		},
		SilenceUsage: true,
	}

	cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().BoolVar(&keepServing, "keep-serving", false,
		`keep serving requests until interrupted`)
	cmd.Flags().DurationVar(&exitQuietPeriod, "quiet-period", app.DefaultExitQuietPeriod,
		`quiet period before exit`)
	cmd.Flags().StringVar(&runtimeName, "runtime", "pythonista3",
		`target runtime`)
	cmd.Flags().StringVar(&transportName, "transport", "cloudflared",
		`tunnel transport`)
	cmd.Flags().StringVar(&transportOpts, "transport-opts", "",
		`extra args for the transport command`)
	cmd.Flags().BoolVar(&transportStderr, "transport-stderr", false,
		`show transport stderr on console`)
	cmd.Flags().BoolVar(&transportStdout, "transport-stdout", false,
		`show transport stdout on console`)
	cmd.Flags().BoolVar(&debug, "debug", false,
		`show debug logs`)
	cmd.Flags().BoolVar(&printURL, "print-url", false,
		`print only the runtime URL`)

	return cmd
}
