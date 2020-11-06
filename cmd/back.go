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
	"github.com/spf13/cobra"
	"log"
	"os"
)

// snapbackCmd represents the snapback command
var snapbackCmd = &cobra.Command{
	Use:   "back",
	Short: "Reinstall a snapshot in lead database",
	Run: func(cmd *cobra.Command, args []string) {
		restoreFromSnapshot()
	},
}

func init() {
	rootCmd.AddCommand(snapbackCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// snapbackCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// snapbackCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func restoreFromSnapshot() {
	cliDbConn := createConnection(cliDbUrl)
	defer cliDbConn.Close(context.Background())

	list, err := listSnapshots(cliDbConn)
	if err != nil {
		log.Fatalf("Error while listing snasphot for removal : %s", err)
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
			defaultDbConn := createConnection(defaultDbUrl)
			defer defaultDbConn.Close(context.Background())

			fromDatabase := fmt.Sprintf("%s_%s", cliName, snap.Hash)
			toDatabase := getProjectName()

			TerminateDatabaseConnections(defaultDbConn, fromDatabase)
			TerminateDatabaseConnections(defaultDbConn, toDatabase)

			fmt.Printf("Restoring from snapshot %s, please wait ..\n", snapshotSelected)
			DropDatabase(defaultDbConn, toDatabase)

			copy_database(defaultDbConn, fromDatabase, toDatabase)
			fmt.Printf("Restoring from snapshot %s successfull\n", snapshotSelected)
		}
	}
}
