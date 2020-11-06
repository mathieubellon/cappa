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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

type S3Key struct {
	key     string
	updated time.Time
	size    int64
}

type AwsConfig struct {
	AwsAccessKeyId     string `mapstructure:"aws_access_key_id"`
	AwsSecretAccessKey string `mapstructure:"aws_secret_access_key"`
	Dest               string `mapstructure:"destination"`
	Bucket             string `mapstructure:"bucket"`
	Region             string `mapstructure:"region"`
	Prefix             string `mapstructure:"prefix"`
}

// grabCmd represents the grab command
var grabCmd = &cobra.Command{
	Use:   "grab",
	Short: "Grab backup file (.dump) from s3 bucket",
	Long:  `Grab list all files in bucket and allow you to pick to download.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		awsconfig := AwsConfig{
			AwsSecretAccessKey: viper.GetString("aws_secret_access_key"),
			AwsAccessKeyId:     viper.GetString("aws_access_key_id"),
			Dest:               viper.GetString("dest"),
			Bucket:             viper.GetString("bucket"),
			Region:             viper.GetString("region"),
			Prefix:             viper.GetString("prefix"),
		}

		// Create AWS session
		sess := getAwsSession(&awsconfig)

		if awsconfig.Bucket == "" {
			return fmt.Errorf("You must provide a bucket value")
		}
		if awsconfig.Region == "" {
			return fmt.Errorf("You must provide a region value")
		}
		// Grab a list of filenames from source s3
		backupList, err := readBucket(awsconfig.Bucket, awsconfig.Prefix, sess)
		if err != nil {
			log.Printf("Error listing files in bucket : %s", err)
		}

		// Ask user to select one file in list
		filekey, filename, filesize := selectBackupIn(backupList)

		if filekey != "" {
			// Create backups directory if not exists
			_ = os.Mkdir(awsconfig.Dest, 0700)
			err := Download(awsconfig.Bucket, sess, filekey, filename, filesize, awsconfig.Dest)
			if err != nil {
				log.Fatalf("Could not download file : %s", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(grabCmd)

	grabCmd.PersistentFlags().StringP("aws_access_key_id", "k", "", "Aws access key ID")
	grabCmd.PersistentFlags().StringP("aws_secret_access_key", "s", "", "Aws secret access key")
	grabCmd.PersistentFlags().String("dest", ".cappa", "Local directory where to download files")
	grabCmd.PersistentFlags().String("bucket", "", "Aws s3 bucket")
	grabCmd.PersistentFlags().String("region", "", "Aws s3 region")
	grabCmd.PersistentFlags().String("prefix", "", "Prefix, within bucket, where to look for backup files")

	viper.BindPFlag("aws_access_key_id", grabCmd.PersistentFlags().Lookup("aws_access_key_id"))
	viper.BindPFlag("aws_secret_access_key", grabCmd.PersistentFlags().Lookup("aws_secret_access_key"))
	viper.BindPFlag("dest", grabCmd.PersistentFlags().Lookup("dest"))
	viper.BindPFlag("bucket", grabCmd.PersistentFlags().Lookup("bucket"))
	viper.BindPFlag("region", grabCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("prefix", grabCmd.PersistentFlags().Lookup("prefix"))

}

// Read bucket content an return a list of s3 Keys
func readBucket(bucket string, prefix string, sess *session.Session) ([]S3Key, error) {

	// Create S3 service client
	svc := s3.New(sess)

	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	resp, listError := svc.ListObjects(params)
	if listError != nil {
		return nil, listError
	}

	var keyList []S3Key
	for _, key := range resp.Contents {
		keyList = append(keyList, S3Key{*key.Key, *key.LastModified, *key.Size})
	}

	// Sort slice of keys by last modified date first
	sort.Slice(keyList, func(i, j int) bool { return keyList[i].updated.After(keyList[j].updated) })

	return keyList, nil
}

func getAwsSession(aco *AwsConfig) *session.Session {

	var config *aws.Config

	config = &aws.Config{
		Region:      aws.String(aco.Region),
		Credentials: credentials.NewStaticCredentials(aco.AwsAccessKeyId, aco.AwsSecretAccessKey, ""),
	}

	if aco.AwsAccessKeyId != "" && aco.AwsSecretAccessKey != "" {
		log.Println("Found AWS credentials from env, flags or config file, using them")
		// Initialize a session with credentials from config or ENV (not from ~/.aws/credentials)
		config = &aws.Config{
			Region:      aws.String(aco.Region),
			Credentials: credentials.NewStaticCredentials(aco.AwsAccessKeyId, aco.AwsSecretAccessKey, ""),
		}
	} else {
		log.Println("Using shared AWS credentials like ~/.aws/credentials")
		// Initialize a session with credentials from the shared credentials file ~/.aws/credentials.
		config = &aws.Config{
			Region: aws.String(aco.Region),
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
	err := survey.AskOne(prompt, &backupKey, nil)
	if err == terminal.InterruptErr {
		fmt.Println("User terminated prompt")
		os.Exit(0)
	} else if err != nil {
		panic(err)
	}

	backupFilename := filepath.Base(backupKey)
	var backupSize int64
	for _, v := range backupList {
		if v.key == backupKey {
			backupSize = v.size
		}
	}

	return backupKey, backupFilename, backupSize
}

type progressWriter struct {
	writer io.WriterAt
	pb     *pb.ProgressBar
}

func (pw *progressWriter) WriteAt(p []byte, off int64) (int, error) {
	pw.pb.Add(len(p))
	return pw.writer.WriteAt(p, off)
}

// Download downloads a file to the local filesystem using s3downloader
func Download(bucket string, sess *session.Session, filekey string, filename string, filesize int64, destination string) error {

	temp, err := ioutil.TempFile(destination, "cappa-")
	if err != nil {
		panic(err)
	}

	bar := pb.New64(filesize).SetUnits(pb.U_BYTES)
	bar.Start()

	writer := &progressWriter{writer: temp, pb: bar}

	tempfileName := temp.Name()

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filekey),
	}

	downloader := s3manager.NewDownloader(sess)

	if _, err := downloader.Download(writer, params); err != nil {
		bar.Set64(bar.Total)
		return err
	}

	bar.FinishPrint(fmt.Sprintf("Downloaded %s to %s", filename, destination))

	if err := temp.Close(); err != nil {
		panic(err)
	}

	defer func() {
		err := os.Remove(tempfileName)
		if err != nil {
			log.Printf("Could not remove temp file")
		}
	}()

	if err := os.Rename(temp.Name(), fmt.Sprintf("%s/%s", destination, filename)); err != nil {
		panic(err)
	}

	return nil
}
