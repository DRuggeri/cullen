package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var baseDir string
var listenAddr string
var baseURL string

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
			return startServer()
		},
	}

	rootCmd.Flags().StringVar(&baseDir, "path", "/var/www/html", "Base directory for media")
	rootCmd.Flags().StringVar(&listenAddr, "listen", ":8080", "Listen address of the server")
	rootCmd.Flags().StringVar(&baseURL, "baseurl", "/cullen", "Base URL prefix for all endpoints")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
