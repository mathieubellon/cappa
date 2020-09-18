/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// revertCmd represents the revert command
var revertCmd = &cobra.Command{
	Use:   "revert",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("revert called")
		trackerConn := createConnection(config, cliName)
		defer trackerConn.Close(context.Background())

		list, err := listSnapshots(trackerConn)
		if err != nil {
			log.Fatalf("Error while listing snasphot for removal", err)
		}

		var options []string
		for _, snap := range list {
			options = append(options, snap.Name)
		}

		var snapshotSelected string
		prompt := &survey.Select{
			Message: "Select snapshot to revert to primary database :",
			Options: options,
		}
		err = survey.AskOne(prompt, &snapshotSelected, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("User terminated prompt")
			os.Exit(0)
		} else if err != nil {
			log.Fatal(err)
		}

		for _, snap := range list {
			if snap.Name == snapshotSelected {
				rawConn := createConnection(config, cliName)
				defer rawConn.Close(context.Background())

				fromDatabase := fmt.Sprintf("%s_%s", cliName, snap.Hash)
				toDatabase := config.Database

				TerminateDatabaseConnections(rawConn, fromDatabase)
				TerminateDatabaseConnections(rawConn, toDatabase)

				DropDatabase(rawConn, config.Database)

				copy_database(rawConn, fromDatabase, toDatabase)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(revertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// revertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// revertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
