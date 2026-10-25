package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var baseDir string
var listenAddr string
var baseURL string
var keepAction, discardAction string

// Destinations for the Keep and Discard actions; empty means the default
// behavior (Keep advances, Discard deletes).
var keepDir, discardDir string

// parseAction validates an action flag value that is either the default word
// or "move:/path/to/folder" and returns the destination ("" for the default).
func parseAction(flag, v, def string) (string, error) {
	if v == "" || v == def {
		return "", nil
	}
	if dir, ok := strings.CutPrefix(v, "move:"); ok {
		dir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		st, err := os.Stat(dir)
		if err != nil {
			return "", fmt.Errorf("--%s destination: %w", flag, err)
		}
		if !st.IsDir() {
			return "", fmt.Errorf("--%s destination %s is not a directory", flag, dir)
		}
		return dir, nil
	}
	return "", fmt.Errorf("invalid --%s %q: use %q or \"move:/path/to/folder\"", flag, v, def)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "cullen",
		Short: "Media culling tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL = strings.TrimRight(baseURL, "/")
			if baseURL != "" && !strings.HasPrefix(baseURL, "/") {
				baseURL = "/" + baseURL
			}
			st, err := os.Stat(baseDir)
			if err != nil {
				return err
			}
			if !st.IsDir() {
				return fmt.Errorf("%s is not a directory", baseDir)
			}
			var err2 error
			if keepDir, err2 = parseAction("keep", keepAction, "next"); err2 != nil {
				return err2
			}
			if discardDir, err2 = parseAction("discard", discardAction, "delete"); err2 != nil {
				return err2
			}
			return startServer()
		},
	}

	rootCmd.Flags().StringVar(&baseDir, "path", "/var/www/html", "Base directory for media")
	rootCmd.Flags().StringVar(&listenAddr, "listen", ":8080", "Listen address of the server")
	rootCmd.Flags().StringVar(&baseURL, "baseurl", "/cullen", "Base URL prefix for all endpoints")
	rootCmd.Flags().StringVar(&keepAction, "keep", "next", `Action for Keep: "next" or "move:/path/to/folder"`)
	rootCmd.Flags().StringVar(&discardAction, "discard", "delete", `Action for Discard: "delete" or "move:/path/to/folder"`)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
