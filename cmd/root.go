package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"

	"io/ioutil"
	"log"

	"strings"

	"github.com/go-playground/validator/v10"
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
	Username string `mapstructure:"username" survey:"username" validate:"required"`
	Password string `mapstructure:"password" survey:"password" validate:"required"`
	Host     string `mapstructure:"host" survey:"host" validate:"required"`
	Port     string `mapstructure:"port" survey:"port" validate:"required"`
	Database string `mapstructure:"database" survey:"database" validate:"required"`
	Project  string `mapstructure:"project" survey:"project" validate:"required"`
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
		if runningCmd == "version" || runningCmd == "help" || runningCmd == "init" {
			return nil
		}

		// Find & load config file
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				return errors.New(fmt.Sprintf("%s\nRun 'cappa init'", err.(viper.ConfigFileNotFoundError)))
			} else {
				return errors.New(fmt.Sprintf("Config file was found but another error was produced : %s", err))
			}
		}

		err = viper.Unmarshal(&config)
		if err != nil {
			log.Printf("error unmarshall %s", err)
		}

		valid, errors := isConfigValid(&config)
		if !valid {
			fmt.Fprintf(cmd.OutOrStdout(), "Some values are missing or are incorrect in your config file (run 'cappa init')\n")
			return errors
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

func isConfigValid(config *Config) (bool, error) {
	validate := validator.New()
	err := validate.Struct(config)
	if err != nil {
		errorsList := []string{}
		for _, err := range err.(validator.ValidationErrors) {
			errorsList = append(errorsList, fmt.Sprintf("\n\"%s\" %s", err.Field(), err.Tag()))
		}
		return false, errors.New(fmt.Sprintf("\n%s", strings.Join(errorsList, "")))
	}
	return true, nil
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
