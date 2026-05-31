package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/version"
)

var (
	serverAddr  string
	storagePath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "gophkeeper-client",
		Short:         "CLI client for GophKeeper password manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "http://localhost:8080", "GophKeeper server address")
	rootCmd.PersistentFlags().StringVar(&storagePath, "storage", "", "local storage path")

	rootCmd.AddCommand(
		versionCmd(),
		registerCmd(),
		loginCmd(),
		syncCmd(),
		listCmd(),
		addLoginCmd(),
		addTextCmd(),
		addCardCmd(),
		addBinaryCmd(),
		getCmd(),
		updateCmd(),
		deleteCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		printError("%s", humanizeError(err))
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show client version and build date",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}
}
