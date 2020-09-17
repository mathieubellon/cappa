package cmd

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/jackc/pgx/v4"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var config Config

const cliName string = "cappa"

type Config struct {
	Username           string `mapstructure:"username"`
	Password           string `mapstructure:"password"`
	Host               string `mapstructure:"host"`
	Port               string `mapstructure:"port"`
	Database           string `mapstructure:"database"`
	BackupDir          string `mapstructure:"backup_dir"`
	AwsAccessKeyId     string `mapstructure:"aws_access_key_id"`
	AwsSecretAccessKey string `mapstructure:"aws_secret_access_key"`
	Bucket             string `mapstructure:"bucket"`
	Region             string `mapstructure:"region"`
	Prefix             string `mapstructure:"prefix"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cappa",
	Short: "It is like Git, but for development databases",
	Long:  `Heavily inspired by fastmonkeys/stellar`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	configFileName := fmt.Sprintf("%s.toml", strings.ToLower(cliName))

	if !fileExists(configFileName) {
		create := false
		prompt := &survey.Confirm{
			Message: "Config file is missing, create one in local directory  ?",
			Default: true,
		}
		err := survey.AskOne(prompt, &create)
		if err == terminal.InterruptErr {
			os.Exit(0)
		}
		if create {
			writeConfigFile(configFileName)
			fmt.Println("Config file created, you will have to re-run this command")
			os.Exit(0)
		}
	}
	// Log as JSON instead of the default ASCII formatter.
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	//rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $PROJECTDIR/.cappa.old.toml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// if _, err := os.Stat("cappa.old.toml"); os.IsNotExist(err) {
	// 	fmt.Println("cappa.old.toml config does not exists, run cappa init")
	// }

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Lookup order (first one found is used)
	// FLAG > ENV > CONFIG
	viper.SetConfigName("cappa") // name of config file (without extension)
	viper.AddConfigPath(".")     // optionally look for config in the working directory
	viper.SetDefault("backup_dir", fmt.Sprintf(".%s", cliName))
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found")
		} else {
			log.Println("Config file was found but another error was produced")
		}
	}
	viper.SetEnvPrefix(strings.ToUpper(cliName))
	viper.AutomaticEnv()

	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("error unmarshall %s", err)
	}
	log.Printf("Connection struct %#v", config)

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

	rawConfig := `username = ""
password = ""
host = "localhost"
port = "5432"
database = ""
#backup_dir = ".cappa"
#aws_access_key_id = ""
#aws_secret_access_key = ""
#bucket = ""
#region = ""
#prefix = ""`

	f, err := os.Create(filename)
	if err != nil {
		// failed to create/open the file
		log.Fatal(err)
	}
	if _, err := f.WriteString(rawConfig); err != nil {
		// failed to encode
		log.Fatal(err)
	}
	defer f.Close()
}
