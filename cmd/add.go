/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/rickliujh/quick/pkg"
	"github.com/spf13/cobra"
)

var (
	tags    string
	title   string
	comment string
	labels  []string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [link] [tags]",
	Short: "add your link to quick's index",
	Long:  `tags split by comma`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {

		var url = args[0]

		var store Store
		path := filepath.Join(BaseDir(), "links.json")
		LoadJSON(path, &store)

		link := Link{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			URL:       url,
			Title:     title,
			Comment:   comment,
			Tags:      ParseTags(tags),
			Labels:    ParseLabels(labels),
			OpenCount: 0,
		}

		store.Links = append(store.Links, link)

		if err := os.MkdirAll(BaseDir(), 0755); err != nil {
			panic(err)
		}

		if err := SaveJSON(path, &store); err != nil {
			panic(err)
		}

		fmt.Println("added:", url)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	rootCmd.Flags().
		StringVarP(&tags, "tag", "t", "", "tags attach the link, split by comma")
	rootCmd.Flags().
		StringVarP(&title, "title", "n", "", "title for the link")
	rootCmd.Flags().
		StringVarP(&comment, "comment", "c", "", "comment for the link")
	rootCmd.Flags().
		StringArrayVarP(&labels, "label", "l", []string{}, "lables attach to the link")
}
