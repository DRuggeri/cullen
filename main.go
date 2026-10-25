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
var keepAction string

// keepDir is the destination for kept files; empty means Keep just advances.
var keepDir string

func parseKeep(v string) error {
	switch {
	case v == "" || v == "next":
		keepDir = ""
		return nil
	case strings.HasPrefix(v, "move:"):
		dir, err := filepath.Abs(strings.TrimPrefix(v, "move:"))
		if err != nil {
			return err
		}
		st, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("keep destination: %w", err)
		}
		if !st.IsDir() {
			return fmt.Errorf("keep destination %s is not a directory", dir)
		}
		keepDir = dir
		return nil
	}
	return fmt.Errorf("invalid --keep %q: use \"next\" or \"move:/path/to/folder\"", v)
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
			if err := parseKeep(keepAction); err != nil {
				return err
			}
			return startServer()
		},
	}

	rootCmd.Flags().StringVar(&baseDir, "path", "/var/www/html", "Base directory for media")
	rootCmd.Flags().StringVar(&listenAddr, "listen", ":8080", "Listen address of the server")
	rootCmd.Flags().StringVar(&baseURL, "baseurl", "/cullen", "Base URL prefix for all endpoints")
	rootCmd.Flags().StringVar(&keepAction, "keep", "next", `Action for Keep: "next" or "move:/path/to/folder"`)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
