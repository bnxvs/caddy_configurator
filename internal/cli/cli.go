// Package cli implements the cobra command surface as thin shells over
// internal/config, internal/project, internal/caddy and internal/server.
package cli

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"caddy-configurator/internal/caddy"
	"caddy-configurator/internal/config"
	"caddy-configurator/internal/constants"
	"caddy-configurator/internal/project"
	"caddy-configurator/internal/render"
	"caddy-configurator/internal/server"
)

var rootCmd = &cobra.Command{
	Use:          "caddy-configurator",
	Short:        "Manage Caddy + compose configs from git-friendly YAML files",
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func projectPath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

func printValidationErrors(err error) {
	if verrs, ok := err.(config.ValidationErrors); ok {
		for _, e := range verrs {
			fmt.Fprintln(os.Stderr, "error: "+e.Error())
		}
		return
	}
	fmt.Fprintln(os.Stderr, "error: "+err.Error())
}

// InitProject creates a minimal valid project skeleton. It fails closed
// when the target already contains a project file.
func InitProject(path string, name string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	marker := filepath.Join(abs, constants.ProjectFileName)
	if _, err := os.Stat(marker); err == nil {
		return fmt.Errorf("project already exists at %s: refusing to overwrite", marker)
	}
	if name == "" {
		name = filepath.Base(abs)
	}
	if err := os.MkdirAll(filepath.Join(abs, constants.SitesDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(abs, constants.GeneratedDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(abs, constants.DataDir), 0o755); err != nil {
		return err
	}
	meta := config.ProjectMeta{Version: constants.SchemaVersion, Name: name}
	return config.SaveProjectMeta(abs, meta)
}

// ValidateProject checks schema validity plus the rendered Caddyfile when
// the caddy binary is available.
func ValidateProject(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	p, err := config.LoadProject(abs)
	if err != nil {
		return err
	}
	sites := "sites"
	if len(p.Sites) == 1 {
		sites = "site"
	}
	fmt.Printf("schema: OK (%d %s", len(p.Sites), sites)
	if len(p.Warnings) > 0 {
		fmt.Printf(", %d warnings", len(p.Warnings))
	}
	fmt.Println(")")
	for _, w := range p.Warnings {
		fmt.Printf("warning: %s: %s: %s\n", w.File, w.Field, w.Reason)
	}
	if !caddy.Found() {
		fmt.Println("caddy binary not found in PATH: Caddy validation skipped")
		return nil
	}
	if len(p.Sites) == 0 {
		fmt.Println("no sites: nothing for caddy to validate")
		return nil
	}
	res := caddy.ValidateCaddyfile(render.RenderCaddyfile(p))
	fmt.Print(res.Output)
	if !res.OK {
		return fmt.Errorf("caddy validation failed")
	}
	fmt.Println("caddy: rendered Caddyfile is valid")
	return nil
}

func init() {
	initCmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Create a new configurator project skeleton",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			path := projectPath(args)
			if err := InitProject(path, name); err != nil {
				printValidationErrors(err)
				return err
			}
			abs, _ := filepath.Abs(path)
			fmt.Printf("initialized project at %s\n", abs)
			return nil
		},
	}
	initCmd.Flags().String("name", "", "project name (default: directory name)")

	generateCmd := &cobra.Command{
		Use:   "generate [path]",
		Short: "Render and write generated/Caddyfile and compose file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := project.GenerateProject(projectPath(args))
			if err != nil {
				printValidationErrors(err)
				return err
			}
			fmt.Printf("wrote %s and %s", res.CaddyfilePath, res.ComposePath)
			if res.Formatted {
				fmt.Print(" (formatted with caddy fmt)")
			}
			fmt.Println()
			return nil
		},
	}

	validateCmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate project sources and rendered Caddyfile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidateProject(projectPath(args)); err != nil {
				printValidationErrors(err)
				return err
			}
			return nil
		},
	}

	serveCmd := &cobra.Command{
		Use:   "serve [path]",
		Short: "Serve the local web UI for one project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			bind, _ := cmd.Flags().GetString("bind")
			devFrontend, _ := cmd.Flags().GetString("dev-frontend")
			return ServeProject(projectPath(args), bind, port, devFrontend)
		},
	}
	serveCmd.Flags().Int("port", constants.DefaultPort, "loopback port to listen on")
	serveCmd.Flags().String("bind", constants.DefaultBind, "address to bind (default loopback)")
	serveCmd.Flags().String("dev-frontend", "", "proxy frontend assets to a vite dev server URL")

	rootCmd.AddCommand(initCmd, generateCmd, validateCmd, serveCmd)
}

// ServeProject starts the web UI for one project root and blocks.
func ServeProject(path, bind string, port int, devFrontend string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	srv, err := server.New(abs, devFrontend)
	if err != nil {
		return err
	}
	addr := server.ResolveAddr(bind, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("serving %s at http://%s\n", abs, ln.Addr())
	return http.Serve(ln, srv.Handler())
}
