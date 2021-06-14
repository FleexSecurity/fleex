package utils

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var Log = logrus.New()

func SetLogLevel(level string) {
	// We are not using logrus' trace and panic levels
	switch strings.ToLower(level) {
	case "debug":
		Log.SetLevel(log.DebugLevel)
	case "info":
		Log.SetLevel(log.InfoLevel)
	case "warning":
		Log.SetLevel(log.WarnLevel)
	case "error":
		Log.SetLevel(log.ErrorLevel)
	case "fatal":
		Log.SetLevel(log.FatalLevel)
	default:
		log.Fatal("Bad error level string")
	}
}

func FileToString(filePath string) string {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		Log.Fatal(err)
	}

	return string(content)
}

func StringToFile(filePath, text string) {
	file, err := os.Create(filePath)
	if err != nil {
		Log.Fatal(err)
	}
	file.WriteString(text)
	file.Close()
}

func LinesCount(s string) int {
	n := strings.Count(s, "\n")
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
