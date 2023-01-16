package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "unknown version"
	GoVersion = "unknown version"
	BuildTime = "unknown time"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of pixiv-dl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:\t", Version)
		fmt.Println("GoVersion:\t", GoVersion)
		fmt.Println("BuildTime:\t", BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
