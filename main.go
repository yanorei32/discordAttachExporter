package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/go-playground/validator/v10"
	"github.com/leekchan/timeutil"
	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"
)

type Configure struct {
	ParallelDownload	int			`yaml:"parallelDownload" validate:"min=1"`
	DownloadExtensions	[]string	`yaml:"downloadExtension" validate:"dive,gte=1"`
}

type Attachment struct {
	FileName	string	`json:"fileName" validate:"required"`
	Url			string	`json:"url" validate:"required"`
}

type User struct {
	Id			string	`json:"id" validate:"required"`
	IsBot		bool	`json:"isBot"`
}

type Message struct {
	Timestamp	string			`json:"timestamp" validate:"required"`
	Author		User			`json:"author" validate:"required"`
	Attachments	[]Attachment	`json:"attachments"`
}

type ExportedJSON struct {
	Messages	[]Message	`json:"messages"`
}

type DownloadItem struct {
	FileName	string
	Url			string
}

type FailedDownloadItem struct {
	Error	error
	Item	DownloadItem
}

func readConfigure() Configure {
	exe, err := os.Executable()

	if err != nil {
		logrus.Fatal("Failed to lookup executable: ", err)
	}

	bytes, err := ioutil.ReadFile(
		filepath.Join(filepath.Dir(exe), "configure.yml"),
	)

	if err != nil {
		logrus.Fatal("Failed to read configure file: ", err)
	}

	c := Configure{}

	if err := yaml.UnmarshalStrict(bytes, &c); err != nil {
		logrus.Fatal("Failed to parse configure file: ", err)
	}

	if err := validator.New().Struct(c); err != nil {
		logrus.Fatal(err)
	}

	return c
}

func readJSON(path string) ExportedJSON {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Fatal("Failed to read JSON file: ", err)
	}
	
	ejson := ExportedJSON{}
	if err := json.Unmarshal(bytes, &ejson); err != nil {
		logrus.Fatal("Failed to parse JSON file: ", err)
	}

	return ejson
}

func printUsage() {
	exe, err := os.Executable()

	if err != nil {
		logrus.Fatal("Failed to lookup executable: ", err)
	}

	logrus.Println(
		"Usage: ",
		filepath.Base(exe),
		" [DiscordChatExporter JSON]",
	)
}

func getCurrentLocation() *time.Location {
	return time.Now().Location()
}

func containsIgnoreCase(array []string, value string) bool {
	for _, v := range array {
		if strings.ToLower(v) == strings.ToLower(value) {
			return true
		}
	}

	return false
}

func formatDateTimeVRCStyle(t time.Time) string {
	return timeutil.Strftime(
		&t,
		"%Y-%m-%d_%H-%m-%S.%f",
	)[:len("2020-04-01_22-54-21.971")]
}

func download(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf(
			"HTTP Status Code is not 200 (%v)",
			resp.StatusCode,
		)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)

	return err
}

func main() {
	logrus.SetOutput(colorable.NewColorableStdout())

	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		printUsage()
		logrus.Fatal("Illegal argument(s) count.");
	}

	c := readConfigure()

	logrus.Info("Read JSON...");
	json := readJSON(args[0])
	loc := getCurrentLocation()

	items := []DownloadItem{}

	for i, message := range json.Messages {
		logrus.Debugf(
			"Read (%v/%v)",
			i + 1,
			len(json.Messages),
		)

		t, err := time.Parse(time.RFC3339, message.Timestamp)
		if err != nil {
			logrus.Errorf(
				"Failed to parse Datetime (%v): %v",
				message.Timestamp,
				err,
			)

			continue
		}

		t = t.In(loc)

		for _, attachment := range message.Attachments {
			download := len(c.DownloadExtensions) == 0 || containsIgnoreCase(
				c.DownloadExtensions,
				filepath.Ext(attachment.FileName)[1:],
			)

			if !download {
				continue
			}

			item := DownloadItem{}

			item.FileName = fmt.Sprintf(
				"discord_%v_%v_%v",
				formatDateTimeVRCStyle(t),
				message.Author.Id,
				attachment.FileName,
			)

			item.Url = attachment.Url

			items = append(items, item)
			items = append(items, item)
			items = append(items, item)
		}
	}

	logrus.Infof("Download %v file(s)...", len(items))

	wg := &sync.WaitGroup{}
	semaphore := make(chan int, c.ParallelDownload)

	failes := []FailedDownloadItem{}
	var failesM sync.Mutex

	for i, item := range items {
		wg.Add(1)

		go func(item DownloadItem, i int) {
			semaphore <- 1
			defer wg.Done()

			logrus.Info(item.FileName)

			if err := download(item.FileName, item.Url); err != nil {
				fItem := FailedDownloadItem{}
				fItem.Item = item
				fItem.Error = err

				failesM.Lock()
				failes = append(failes, fItem)
				failesM.Unlock()
			}

			<-semaphore
		}(item, i)
	}

	wg.Wait()

	failureCnt := len(failes)
	succeedCnt := len(items) - failureCnt

	if failureCnt == 0 {
		logrus.Infof("%v failure, %v succeed", failureCnt, succeedCnt)
	} else {
		logrus.Warnf("%v failure, %v succeed", failureCnt, succeedCnt)
	}

	for _, fail := range failes {
		logrus.Errorf(
			"%v\n%v",
			fail.Error,
			fail.Item.Url,
		)
	}
}


