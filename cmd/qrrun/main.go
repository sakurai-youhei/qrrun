// Command qrrun tunnels a local script file and displays a QR code so that a
// mobile device can run the script by scanning it.
//
// Usage:
//
//	qrrun --transport cloudflared --runtime pythonista3 your-local-script.py
//	echo "print('hello')" | qrrun --transport cloudflared --runtime pythonista3 -
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sakurai-youhei/qrrun/internal/app"
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

	cmd := &cobra.Command{
		Use:   "qrrun [flags] <script|-]",
		Short: "Tunnel local code. Run via QR.",
		Long: `qrrun serves a local script over a secure tunnel and displays a QR code.

Scan the QR code with your mobile device to open the script in the configured
runtime (e.g. Pythonista 3 on iOS).

Prerequisite:
	cloudflared must be installed and available on PATH.
	qrrun uses Cloudflare Quick Tunnels (trycloudflare.com),
	so account login is not required.

Examples:
	qrrun --transport cloudflared --runtime pythonista3 hello.py
	echo "print('hello')" | qrrun --transport cloudflared --runtime pythonista3 -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(app.Options{
				TransportName: transportName,
				RuntimeName:   runtimeName,
				ScriptPath:    args[0],
			})
		},
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&transportName, "transport", "cloudflared",
		`Tunnel transport to use. Available: cloudflared`)
	cmd.Flags().StringVar(&runtimeName, "runtime", "pythonista3",
		`Mobile runtime to target. Available: pythonista3`)

	return cmd
}
