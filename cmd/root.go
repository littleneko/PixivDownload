/*
Copyright © 2023 litao.little@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pixiv",
	Short: "A tool to download pixiv illust",
	Long:  "",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./pixiv.yaml and $HOME/pixiv.yaml)")

	rootCmd.PersistentFlags().String("log-path", "", "Log file path (default is stdout)")
	rootCmd.PersistentFlags().String("log-level", "INFO", "Log level, choices: [DEBUG, INFO, WARNING, ERROR]")
	rootCmd.PersistentFlags().String("cookie", "", "Your Cookies, only need the key-value 'PHPSESSID=abcxyz'")
	rootCmd.PersistentFlags().String("user-agent", defaultUserAgent, "Http User-Agent header")
	err := viper.BindPFlags(rootCmd.PersistentFlags())
	cobra.CheckErr(err)

	err = viper.BindPFlags(downloadCmd.Flags())
	cobra.CheckErr(err)

	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(infoCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".pixiv" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("pixiv")
	}

	viper.SetEnvPrefix("PIXIV")
	viper.AutomaticEnv() // read in environment variables that match
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Fprintln(os.Stderr, "Config file not found, use default config")
		} else {
			// Config file was found but another error was produced
			fmt.Fprintln(os.Stderr, "Failed to read config file.", err)
			panic(err.Error())
		}
	}
}
