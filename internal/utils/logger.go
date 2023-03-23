package utils

import (
	"io"
	"log"
	"os"
)

var LogToFile bool
var LogFilePath *string

func Log(v any) {

	wrt := io.MultiWriter(os.Stdout)

	if LogToFile {
		filePath := "klikbca.log"

		if LogFilePath != nil {
			filePath = *LogFilePath
		}
		logFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Panic(err)
		}
		defer logFile.Close()

		wrt = io.MultiWriter(os.Stdout, logFile)
	}

	ilog := log.New(wrt, "", log.LstdFlags)

	ilog.SetFlags(log.LstdFlags)

	ilog.Println(v)

}
