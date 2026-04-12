// Command qrrun tunnels a local script file and displays a QR code so that a
// mobile device can run the script by scanning it.
//
// Usage:
//
//	qrrun your-local-script.py arg1 arg2
//	echo "print('hello')" | qrrun - arg1 arg2
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
	var qrLevel string
	var keepServing bool
	var exitQuietPeriod time.Duration
	var transportStderr bool
	var transportStdout bool
	var debug bool
	var printURL bool
	var transportOpts string

	cmd := &cobra.Command{
		Use:   "qrrun [flags] <script|-> [args...]",
		Short: "tunnel local scripts and run via QR",
		Long: `QRrun serves a local script through a secure tunnel and prints a QR code.

scan the QR code to open the script in the selected runtime.

Prerequisites:
	cloudflared must be installed and available on PATH
	QRrun uses Cloudflare Quick Tunnels (trycloudflare.com)

Examples:
	qrrun hello.py arg1 arg2
	echo "print('hello')" | qrrun - arg1 arg2`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keepServingMode := keepServing
			scriptPath := args[0]
			scriptArgs := append([]string(nil), args[1:]...)
			return app.Run(app.Options{
				TransportName:   transportName,
				RuntimeName:     runtimeName,
				QRErrorLevel:    qrLevel,
				ScriptPath:      scriptPath,
				ScriptArgs:      scriptArgs,
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
	cmd.SetUsageTemplate(`Usage:
  {{.UseLine}}

{{if .HasAvailableFlags}}Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Supported QR error correction levels:
	` + app.SupportedQRErrorLevelsText() + `

Supported runtimes:
	pythonista, pythonista2, pythonista3
{{end}}`)

	cmd.Flags().BoolVar(&keepServing, "keep-serving", false,
		`keep serving requests until interrupted`)
	cmd.Flags().DurationVar(&exitQuietPeriod, "quiet-period", app.DefaultExitQuietPeriod,
		`quiet period before exit`)
	cmd.Flags().StringVar(&runtimeName, "runtime", "pythonista3",
		`target runtime`)
	cmd.Flags().StringVar(&qrLevel, "qr-level", app.DefaultQRErrorLevel,
		`QR error correction level`)
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
