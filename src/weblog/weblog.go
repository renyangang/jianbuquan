// weblog project weblog.go
package weblog

import (
	"fmt"
	"log"
	"os"
)

const (
	LOGFILE = "web.log"
)

var logger *log.Logger
var logfile *os.File
var debugflag bool

func loginit() {
	debugflag = true
	var err error
	_, err = os.Stat(LOGFILE)
	if err != nil && !os.IsExist(err) {
		logfile = nil
	}
	if logfile == nil {
		logfile, err = os.OpenFile(LOGFILE, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModeAppend|os.ModePerm)
		if err != nil {
			fmt.Printf("open file %s failed.errinfo:%s", LOGFILE, err.Error())
			return
		}
		logger = log.New(logfile, "", log.LstdFlags)
	}
}

func InfoLog(format string, v ...interface{}) {
	loginit()
	logger.SetPrefix("[INFO]")
	logger.Output(2, fmt.Sprintf(format, v))
}

func DebugLog(format string, v ...interface{}) {
	loginit()
	if !debugflag {
		return
	}
	logger.SetPrefix("[DEBUG]")
	logger.SetFlags(log.LstdFlags | log.Llongfile)
	logger.Output(3, fmt.Sprintf(format, v))
}

func ErrorLog(format string, v ...interface{}) {
	loginit()
	logger.SetPrefix("[ERROR]")
	logger.SetFlags(log.LstdFlags | log.Llongfile)
	logger.Output(2, fmt.Sprintf(format, v))
}
