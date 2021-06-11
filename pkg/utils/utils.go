package utils

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func FileToString(filePath string) string {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return string(content)
}

func StringToFile(filePath, text string) {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
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
