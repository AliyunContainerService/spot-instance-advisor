package pkg

import (
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"os"
)

const (
	LOGFILE string = "logfile.log"
)

func ConfigLogger(logLevelStr *string) {
	// Create the log file if doesn't exist. And append to it if it already exists.
	f, err := os.OpenFile(LOGFILE, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
	} else {
		logger.SetOutput(f)
	}
	Formatter := new(logger.TextFormatter)
	// You can change the Timestamp format. But you have to use the same date and time.
	// "2006-02-02 15:04:06" Works. If you change any digit, it won't work
	// ie "Mon Jan 2 15:04:05 MST 2006" is the reference time. You can't change it
	Formatter.TimestampFormat = "2006-02-01 15:04:05"
	Formatter.FullTimestamp = true
	logger.SetFormatter(Formatter)

	loglevel, err := logger.ParseLevel(*logLevelStr)
	if err != nil {
		*logLevelStr = logger.WarnLevel.String()
		logger.SetLevel(logger.WarnLevel)
	} else {
		logger.SetLevel(loglevel)
	}
}
