package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type S3Key struct {
	key     string
	updated time.Time
	size    int64
}

type AwsCredentials struct {
	awsAccessKeyId     string
	awsSecretAccessKey string
	bucket             string
	region             string
	prefix             string
	sharedCreds        bool
}

// Read bucket content an return a list of s3 Keys
func readBucket(i AwsCredentials) []S3Key {

	sess := getAwsSession(i)

	// Create S3 service client
	svc := s3.New(sess)

	params := &s3.ListObjectsInput{
		Bucket: aws.String(i.bucket),
		Prefix: aws.String(i.prefix),
	}

	resp, listError := svc.ListObjects(params)
	if listError != nil {
		fmt.Println("Error while listing files in bucket")
		os.Exit(1)
	}

	var keyList []S3Key
	for _, key := range resp.Contents {
		keyList = append(keyList, S3Key{*key.Key, *key.LastModified, *key.Size})
	}

	// Sort slice of keys by last modified date first
	sort.Slice(keyList, func(i, j int) bool { return keyList[i].updated.After(keyList[j].updated) })

	return keyList
}

func getAwsSession(i AwsCredentials) *session.Session {

	var config *aws.Config

	if i.sharedCreds {
		// Initialize a session with credentials from the shared credentials file ~/.aws/credentials.
		config = &aws.Config{
			Region: aws.String(i.region),
		}
	} else {
		// Initialize a session with credentials from config or ENV (not from ~/.aws/credentials)
		config = &aws.Config{
			Region:      aws.String(i.region),
			Credentials: credentials.NewStaticCredentials(i.awsAccessKeyId, i.awsSecretAccessKey, ""),
		}
	}
	sess, err := session.NewSession(
		config,
	)
	if err != nil {
		panic(fmt.Sprintln("Error while creating session"))
	}
	return sess
}

func selectBackupIn(backupList []S3Key) (key string, filename string, size int64) {
	var Selector []string

	for _, backup := range backupList {
		//fmt.Printf("%s | %d\n", backup.key, backup.size)
		Selector = append(Selector, backup.key)
	}
	backupKey := ""
	prompt := &survey.Select{
		Message: "Select backup file:",
		Options: Selector,
	}
	survey.AskOne(prompt, &backupKey, nil)

	backupFilename := filepath.Base(backupKey)
	var backupSize int64
	for _, v := range backupList {
		if v.key == backupKey {
			backupSize = v.size
		}
	}

	return backupKey, backupFilename, backupSize
}

const helpFindAwsCredKeys = `
If you do not use the --shared_credentials flag you must provide:

Config file :  values for aws_access_key_id and aws_secret_access_key keys
Or
In environment variables: values for CAPPA_AWS_ACCESS_KEY_ID and CAPPA_AWS_SECRET_ACCESS_KEY
`

type progressWriter struct {
	writer io.WriterAt
	pb     *pb.ProgressBar
}

func (pw *progressWriter) WriteAt(p []byte, off int64) (int, error) {
	pw.pb.Add(len(p))
	return pw.writer.WriteAt(p, off)
}

// Download downloads a file to the local filesystem using s3downloader
func Download(i AwsCredentials, filekey string, filename string, filesize int64, destination string) error {

	temp, err := ioutil.TempFile(destination, "s3mini-")
	if err != nil {
		panic(err)
	}

	bar := pb.New64(filesize).SetUnits(pb.U_BYTES)
	bar.Start()

	writer := &progressWriter{writer: temp, pb: bar}

	tempfileName := temp.Name()

	params := &s3.GetObjectInput{
		Bucket: aws.String(i.bucket),
		Key:    aws.String(filekey),
	}

	downloader := s3manager.NewDownloader(getAwsSession(i))

	if _, err := downloader.Download(writer, params); err != nil {
		bar.Set64(bar.Total)
		log.Printf("Download failed! Deleting tempfile: %s", tempfileName)
		os.Remove(tempfileName)
		panic(err)
	}

	bar.FinishPrint(fmt.Sprintf("Downloaded %s to %s", filename, destination))

	if err := temp.Close(); err != nil {
		panic(err)
	}

	if err := os.Rename(temp.Name(), fmt.Sprintf("%s/%s", destination, filename)); err != nil {
		panic(err)
	}

	return nil
}

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:     "download",
	Short:   "Download file from S3",
	Long:    "",
	Aliases: []string{"dl"},
	Run: func(cmd *cobra.Command, args []string) {

		connectInfos := AwsCredentials{
			awsAccessKeyId:     viper.GetString("aws_access_key_id"),
			awsSecretAccessKey: viper.GetString("aws_secret_access_key"),
			prefix:             viper.GetString("prefix"),
			bucket:             viper.GetString("bucket"),
			region:             viper.GetString("region"),
			sharedCreds:        viper.GetBool("shared_credentials"),
		}

		if connectInfos.awsAccessKeyId == "" || connectInfos.awsSecretAccessKey == "" {
			if connectInfos.sharedCreds == false {
				fmt.Println(helpFindAwsCredKeys)
				os.Exit(1)
			}
		}

		// Grab a list of filenames from source s3
		backupList := readBucket(connectInfos)

		// Ask user to select one file in list
		filekey, filename, filesize := selectBackupIn(backupList)

		if filekey != "" {
			// Create backups directory if not exists
			_ = os.Mkdir(config.BackupDir, 0700)
			Download(connectInfos, filekey, filename, filesize, config.BackupDir)
		}

	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	// Here you will define your flags and configuration settings.
	downloadCmd.PersistentFlags().StringP("aws_access_key_id", "k", "", "Aws access key ID")
	downloadCmd.PersistentFlags().StringP("aws_secret_access_key", "s", "", "Aws secret access key")
	downloadCmd.PersistentFlags().String("backup_dir", "", "Local directory where to save backup files")
	downloadCmd.PersistentFlags().StringP("bucket", "b", "", "Aws s3 bucket")
	downloadCmd.PersistentFlags().StringP("region", "r", "", "Aws s3 region")
	downloadCmd.PersistentFlags().StringP("prefix", "p", "", "Prefix, within bucket, where to look for backup files")
	downloadCmd.PersistentFlags().BoolP("shared_credentials", "c", false, "Use your local aws credentials in ~/.aws/credentials (default false)")

	viper.BindPFlag("aws_access_key_id", downloadCmd.PersistentFlags().Lookup("aws_access_key_id"))
	viper.BindPFlag("aws_secret_access_key", downloadCmd.PersistentFlags().Lookup("aws_secret_access_key"))
	viper.BindPFlag("backup_dir", downloadCmd.PersistentFlags().Lookup("backup_dir"))
	viper.BindPFlag("bucket", downloadCmd.PersistentFlags().Lookup("bucket"))
	viper.BindPFlag("region", downloadCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("prefix", downloadCmd.PersistentFlags().Lookup("prefix"))
	viper.BindPFlag("shared_credentials", downloadCmd.PersistentFlags().Lookup("shared_credentials"))
}
