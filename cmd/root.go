package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/BurntSushi/toml"
	"github.com/jackc/pgx/v4"
	"github.com/ttacon/chalk"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const cliName string = "cappa"

var (
	version        = "v0.6-beta1"
	commit         = "none"
	date           = "unknown"
	builtBy        = "unknown"
	configFileName = fmt.Sprintf(".%s.toml", strings.ToLower(cliName))
	config         Config
)

type Config struct {
	Username string `mapstructure:"username" survey:"username"`
	Password string `mapstructure:"password" survey:"password"`
	Host     string `mapstructure:"host" survey:"host"`
	Port     string `mapstructure:"port" survey:"port"`
	Database string `mapstructure:"database" survey:"database"`
	Project  string `mapstructure:"project" survey:"project"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   `cappa`,
	Short: `It is like Git, but for development databases`,
	Long: `Cappa allows you to take fast snapshots / restore of your development database.
Useful when you have git branches containing migrations
Heavily (98%) inspired by fastmonkeys/stellar
`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return err
		}
		if !verbose {
			log.SetOutput(ioutil.Discard)
		}
		// Enable line numbers in logging
		log.SetFlags(log.LstdFlags | log.Lshortfile)

		// Some commands does not need a correct config file to be present, return early if we are running
		// excluded commands
		runningCmd := cmd.Name()
		if runningCmd == "version" || runningCmd == "help" {
			return nil
		}

		// Find & load config file
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				return err.(viper.ConfigFileNotFoundError)
			} else {
				return errors.New(fmt.Sprintf("Config file was found but another error was produced : %s", err))
			}
		}
		emptyConfig := Config{}
		if config == emptyConfig {
			return errors.New(fmt.Sprintf("Config file exists but is empty"))
		}
		// Unmarshal config into Config struct
		err = viper.Unmarshal(&config)
		if err != nil {
			log.Printf("error unmarshall %s", err)
		}
		log.Println("Using config file:", viper.ConfigFileUsed())
		log.Printf("Config values : %#v", config)

		// If cli database does not exists, create
		conn := createConnection(config, "")
		defer func() {
			err := conn.Close(context.Background())
			if err != nil {
				log.Printf("Error while closing db connection : %s", err)
			}
		}()
		createTrackerDb(conn)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {

	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "What's wrong ? Speak to me")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName(".cappa") // name of config file (without extension)
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	viper.SetDefault("from-dir", fmt.Sprintf(".%s", cliName))
}

// create connection with postgres db
func createConnection(config Config, database string) *pgx.Conn {
	//Connect to a specific database for dedicated operations or, by default, to postgres database for create/drop operations
	if database == "" {
		database = "postgres"
	}
	// Open the connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.Host, config.Port, config.Username, config.Password, database)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database %s : %v\n", database, err)
	}
	// check the connection
	err = conn.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	log.Printf("Successfully connected to %s", database)
	return conn
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

// This function create the database for tracking snapshots
func createTrackerDb(conn *pgx.Conn) {
	structureSql := `CREATE TABLE snapshots (id SERIAL PRIMARY KEY, hash TEXT UNIQUE NOT NULL, name TEXT NOT NULL,project TEXT NOT NULL, created_at timestamp not null default CURRENT_TIMESTAMP);`
	if !DatabaseExists(conn, cliName) {
		CreateDatabase(conn, cliName)

		trackerConn := createConnection(config, cliName)
		defer func() {
			err := trackerConn.Close(context.Background())
			if err != nil {
				log.Printf("Error while closing db connection : %s", err)
			}
		}()

		log.Print(structureSql)
		_, err := trackerConn.Exec(context.Background(), structureSql)
		if err != nil {
			log.Fatalf("Failed to created cli database: %v\n", err)
		}
		fmt.Printf("Database %s successfully created", cliName)
	}
}
