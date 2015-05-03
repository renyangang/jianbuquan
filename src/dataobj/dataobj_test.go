// dataobj_test.go
package dataobj

import (
	"testing"
	"time"
)

func Test_serial(t *testing.T) {
	dr := new(DailyRecord)
	dr.Day = time.Now()
	dr.StepNum = 10000
	dr.Distance = 10
	dr.Img = "xge"

	bs, err := dr.Serialization()
	if (err != nil) || bs == nil {
		t.Error("serialization failed.")
	}
	if !dr.UnSerialization(bs) {
		t.Error("Unserialization failed.")
	}
}
