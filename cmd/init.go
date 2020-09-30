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
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/BurntSushi/toml"
	"github.com/ttacon/chalk"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
		writeConfigFile(configFileName)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


func writeConfigFile(filename string) {

	err := runWizard(&config)
	if err != nil {
		log.Fatalf("Error during wizard : %s", err)
	}

	f, err := os.Create(filename)
	if err != nil {
		// failed to create/open the file
		log.Fatal(err)
	}
	if err := toml.NewEncoder(f).Encode(config); err != nil {
		// failed to encode
		log.Fatal(err)
	}

	fmt.Println(chalk.Green.Color(fmt.Sprintf("Config file (%s) created, good to go", configFileName)))

	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
}

func runWizard(config *Config) error {
	// Get parent dir name as project name
	currWorkDirPath, _ := os.Getwd()
	breakPath := strings.Split(currWorkDirPath, "/")
	maybeDirName := breakPath[len(breakPath)-1]

	// Wizard questions
	var qs = []*survey.Question{
		{
			Name:     "username",
			Prompt:   &survey.Input{Message: "Postgres user ?"},
			Validate: survey.Required,
		},
		{
			Name:     "password",
			Prompt:   &survey.Password{Message: "Postgres password ?"},
			Validate: survey.Required,
		},
		{
			Name:     "host",
			Prompt:   &survey.Input{Message: "Postgres server host ?", Default: "127.0.0.1"},
			Validate: survey.Required,
		},
		{
			Name:     "port",
			Prompt:   &survey.Input{Message: "Postgres server port ?", Default: "5432"},
			Validate: survey.Required,
		},
		{
			Name:     "database",
			Prompt:   &survey.Input{Message: "Your working database name"},
			Validate: survey.Required,
		},
		{
			Name:     "project",
			Prompt:   &survey.Input{Message: "This project name", Default: maybeDirName},
			Validate: survey.Required,
		},
	}

	// perform the questions
	err := survey.Ask(qs, config)
	if err == terminal.InterruptErr {
		fmt.Println("User terminated prompt, no config file created")
		os.Exit(0)
	} else if err != nil {
		return err
	}

	return nil
}