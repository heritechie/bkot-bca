package utils

import (
	"bufio"
	"fmt"
	"os"
)

func GetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		fmt.Printf("%s not set\n", key)
		return ""
	} else {
		return val
	}
}

func GetLinesStr(filePath string) []string {
	readFile, err := os.Open("config.txt")

	defer readFile.Close()

	if err != nil {
		fmt.Println(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	return fileLines

}
