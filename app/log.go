package app

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type MyFormatter struct {
}

func (m *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	var newLog string
	if len(entry.Data) > 0 {
		newLog = fmt.Sprintf("%s [%s] (%s:%d) - %s, %s\n", timestamp, strings.ToUpper(entry.Level.String()), path.Base(entry.Caller.File), entry.Caller.Line, entry.Message, entry.Data)
	} else {
		newLog = fmt.Sprintf("%s [%s] (%s:%d) - %s\n", timestamp, strings.ToUpper(entry.Level.String()), path.Base(entry.Caller.File), entry.Caller.Line, entry.Message)
	}

	b.WriteString(newLog)
	return b.Bytes(), nil
}

func InitLog(logPath string, logLevel string) {
	if len(logPath) > 0 {
		err := CheckAndMkdir(logPath)
		if err != nil {
			log.Fatalf("Failed to create log dir, msg: %s", err)
		}

		writer, err := os.OpenFile(filepath.Join(logPath, "pixiv.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		logrus.SetOutput(writer)
	}

	logLevelP, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Unknown log level '%s', msg: %s", logLevel, err)
	}
	logrus.SetLevel(logLevelP)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&MyFormatter{})
}
