package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"net/url"

	"github.com/jackc/pgx/v4"

	"io/ioutil"
	"log"

	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const cliName string = "cappa"

var trackedDbUrl string
var defaultDbUrl string
var cliDbUrl string

var (
	version        = "v0.6"
	commit         = "none"
	date           = "unknown"
	builtBy        = "unknown"
	configFileName = fmt.Sprintf(".%s.toml", strings.ToLower(cliName))
	config         Config
	cfgFile        string
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
	Use:          `cappa`,
	Short:        `It is like Git, but for development databases`,
	Long:         `Cappa allows you to take fast snapshots / restore of your development database.`,
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
		} else {
			initConfig()
		}

		if viper.GetString("database_url") == "" {
			fmt.Print("Error : 'database_url' not set\nPlease provide a connexion url ('postgres://user:password@localhost:5432/dbname')\nIt can be DATABASE_URL in environment variable or database_url in config file\n")
		}

		// Connection URL to the database we want to track
		trackedDbUrl = viper.GetString("database_url")
		log.Printf("Tracked database connection string : %s", trackedDbUrl)

		// Connection URL to the default database (we assume 'postgres') for DELETE/COPY operations of the tracked database
		// Use database_url to get a connection string to default database
		t, _ := url.Parse(trackedDbUrl)
		d := &url.URL{
			Scheme: t.Scheme,
			User:   t.User,
			Host:   t.Host,
			Path:   "postgres",
		}
		defaultDbUrl = d.String()
		log.Printf("Default database connection string : %s", defaultDbUrl)

		// Connection URL to the cli database
		// Use database_url to get a connection string to cli database
		c := &url.URL{
			Scheme: t.Scheme,
			User:   t.User,
			Host:   t.Host,
			Path:   cliName,
		}
		cliDbUrl = c.String()
		log.Printf("CLI database connection string : %s", cliDbUrl)

		//If cli database does not exists, create
		conn := createConnection(defaultDbUrl)
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
	cobra.OnInitialize()
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cappa.toml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "What's wrong ? Speak to me")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Printf("Unable to locate home directory : %s", err)
		}

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".cappa")
	}

	//viper.BindEnv("database_url")
	viper.AutomaticEnv()

	// Find & load config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("%s\nRun 'cappa init'", err.(viper.ConfigFileNotFoundError))
		} else {
			log.Printf("Config file was found but another error was produced : %s", err)
		}
	}

}

// create connection with postgres db
func createConnection(connUrl string) *pgx.Conn {
	//Connect to a specific database for dedicated operations or, by default, to postgres database for create/drop operations
	//if database == "" {
	//	database = "postgres"
	//}

	// Open the connection
	conn, err := pgx.Connect(context.Background(), connUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database with %v : %v\n", connUrl, err)
	}
	// check the connection
	err = conn.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	log.Printf("Successfully connected to %s", connUrl)
	return conn
}

// This function create the database for tracking snapshots
func createTrackerDb(conn *pgx.Conn) {
	structureSql := `CREATE TABLE snapshots (id SERIAL PRIMARY KEY, hash TEXT UNIQUE NOT NULL, name TEXT NOT NULL,project TEXT NOT NULL, created_at timestamp not null default CURRENT_TIMESTAMP);`
	if !DatabaseExists(conn, cliName) {
		CreateDatabase(conn, cliName)

		trackerConn := createConnection(cliDbUrl)
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
