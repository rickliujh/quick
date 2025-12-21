/*
Copyright © 2025 Rick Liu
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cfgFileName = ".quick"
	cfgFileType = "yaml"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "quick",
	Short: "Quick is terminal based local index engine for accessing urls and folders quicky",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/%s.%s)", cfgFileName, cfgFileType))

	rootCmd.Flags().BoolP("toggle", "", false, "Help message for toggle")
}

func initConfig() {
	// Lookup order
	// specified file -> current -> home
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find current directory.
		curr, err := os.Getwd()
		cobra.CheckErr(err)
		viper.AddConfigPath(curr)

		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		// Search config in home directory with name ".kickstart-gogrpc" (without extension).
		viper.AddConfigPath(home)

		viper.SetConfigType(cfgFileType)
		viper.SetConfigName(cfgFileName)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
