// weblog project weblog.go
package weblog

import (
	"fmt"
)

func InfoLog(format string, v ...interface{}) {

}

func DebugLog(format string, v ...interface{}) {

}

func ErrorLog(format string, v ...interface{}) {
	fmt.Errorf(format, v)
}
