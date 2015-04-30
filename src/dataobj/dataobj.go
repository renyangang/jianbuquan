// dataobj project dataobj.go
package dataobj

import (
	"bytes"
	"encoding/binary"
	"github.com/garyburd/redigo/redis"
	"time"
	"weblog"
)

var dbpool *redis.Pool

func init() {
	dbpool = redis.NewPool(getConn, 5)
}

func getConn() (redis.Conn, error) {
	return redis.Dial("TCP", "192.168.1.2:4321")
}

type RedisObj interface {
	Save() bool
	Load() bool
	Serialization() ([]byte, error)
	UnSerialization([]byte) bool
}

type DailyRecord struct {
	StepNum       int
	Distance      int
	oldStepNum    int
	oldDistance   int
	Day           time.Time
	Img           string
	istodayloaded bool
	index         int32
	version       int32
	user          *User
}

func NewDailyRecord(usr *User, idx int32) (dr *DailyRecord) {
	dr = new(DailyRecord)
	dr.index = idx
	dr.user = usr
	if !dr.Load() {
		return nil
	}
	return
}

func (dr *DailyRecord) UnSerialization(bs []byte) bool {
	dr.istodayloaded = false
	dr.version = 0
	dr.Day = time.Now()
	if len(bs) <= 0 {
		return true
	}
	if len(bs) < 16 {
		weblog.ErrorLog("bytes len is less than 16")
		return false
	}
	buf := bytes.NewBuffer(bs)
	err := binary.Read(buf, binary.BigEndian, &dr.version)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.version failed.errinfo:%s", err.Error())
		return false
	}
	var t int64
	err = binary.Read(buf, binary.BigEndian, &t)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.Day failed.errinfo:%s", err.Error())
		return false
	}
	dr.Day = time.Unix(t, 0)
	err = binary.Read(buf, binary.BigEndian, &dr.StepNum)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.StepNum failed.errinfo:%s", err.Error())
		return false
	}
	dr.oldStepNum = dr.StepNum
	err = binary.Read(buf, binary.BigEndian, &dr.Distance)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.Distance failed.errinfo:%s", err.Error())
		return false
	}
	dr.oldDistance = dr.Distance
	var strlen int32
	err = binary.Read(buf, binary.BigEndian, &strlen)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.Img length failed.errinfo:%s", err.Error())
		return false
	}
	dr.Img = string(buf.Next(int(strlen)))
	today := time.Now()
	dr.istodayloaded = (dr.Day.Year() == today.Year() && dr.Day.Month() == today.Month() && dr.Day.Day() == today.Day())
	return true
}

func (dr *DailyRecord) Load() (ret bool) {
	bs, err := redis.Bytes(dbpool.Get().Do("LINDEX", dr.user.Id, dr.index))
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.Img length failed.errinfo:%s", err.Error())
		return false
	}
	return dr.UnSerialization(bs)
}

func (dr *DailyRecord) Serialization() (bs []byte, err error) {
	bs = make([]byte, 16)
	buf := bytes.NewBuffer(bs)
	err = binary.Write(buf, binary.BigEndian, dr.version)
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.version failed.errinfo:%s", err.Error())
		return
	}
	err = binary.Write(buf, binary.BigEndian, dr.Day.Unix())
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.Day failed.errinfo:%s", err.Error())
		return
	}
	err = binary.Write(buf, binary.BigEndian, dr.StepNum)
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.StepNum failed.errinfo:%s", err.Error())
		return
	}
	err = binary.Write(buf, binary.BigEndian, dr.Distance)
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.Distance failed.errinfo:%s", err.Error())
		return
	}
	imgbuf := bytes.NewBufferString(dr.Img)
	err = binary.Write(buf, binary.BigEndian, int32(imgbuf.Len()))
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.Img length failed.errinfo:%s", err.Error())
		return
	}
	err = binary.Write(buf, binary.BigEndian, imgbuf.Bytes())
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.Img failed.errinfo:%s", err.Error())
		return
	}
	return buf.Bytes(), nil
}

func (dr *DailyRecord) Save() (ret bool) {
	bs, err := dr.Serialization()
	if err != nil {
		weblog.ErrorLog("Serialization DailyRecord for Save failed.errinfo:%s", err.Error())
		return false
	}
	if dr.istodayloaded {
		_, err = dbpool.Get().Do("LSET", dr.user.Id, dr.index, bs)
	} else {
		_, err = dbpool.Get().Do("LPUSH", dr.user.Id, bs)
	}
	if err != nil {
		weblog.ErrorLog("Save DailyRecord failed.errinfo:%s", err.Error())
		return false
	}
	dr.user.UpdateStepRecord(dr.oldStepNum, dr.StepNum)
	dr.user.UpdateDistanceRecord(dr.oldDistance, dr.Distance)
	return true
}

type User struct {
	Appid         string
	Id            string
	Name          string
	Img           string
	WeekStepNum   int
	MonthStepNum  int
	WeekDistance  int
	MonthDistance int
	IsLoad        bool
	SelfDailys    []*DailyRecord
	SelfDaily     *DailyRecord
}

func (user *User) GetDailyRecords(day int) bool {
	user.SelfDailys = make([]*DailyRecord, day)
	if !user.GetDailyRecord(time.Now()) {
		return false
	}

	user.SelfDailys[0] = user.SelfDaily
	if day > 1 {
		dbconn := dbpool.Get()
		bsarray, err := redis.Values(dbconn.Do("LRANGE", user.Id, 1, day-1))
		if err != nil {
			weblog.ErrorLog("get dailyrecords failed in GetDailyRecord.errinfo: %s", err.Error())
			return false
		}
		if bsarray == nil || len(bsarray) == 0 {
			return true
		}
		for i, bs := range bsarray {
			user.SelfDailys[i+1] = new(DailyRecord)
			user.SelfDailys[i+1].user = user
			user.SelfDailys[i+1].UnSerialization(bs.([]byte))
		}
	}
	return true
}

func (user *User) GetDailyRecord(day time.Time) bool {
	if day.After(time.Now()) {
		weblog.ErrorLog("invalid day in GetDailyRecord")
		return false
	}
	dbconn := dbpool.Get()
	recordlen, err := redis.Int(dbconn.Do("LLEN", user.Id))
	if err != nil {
		weblog.ErrorLog("get dailyrecord len failed in GetDailyRecord.errinfo: %s", err.Error())
		return false
	}
	if recordlen == 0 {
		user.SelfDaily = new(DailyRecord)
		user.SelfDaily.Day = day
		user.SelfDaily.istodayloaded = false
		user.SelfDaily.user = user
		return true
	}
	// 如果最新的记录更老，则补全中间的所有记录
	user.SelfDaily = NewDailyRecord(user, 0)
	if user.SelfDaily == nil {
		weblog.ErrorLog("load daily record index 0 failed.")
		return false
	}
	for user.SelfDaily.Day.Before(day) && user.SelfDaily.Day.YearDay() != day.YearDay() {
		tt, _ := time.ParseDuration("24h")
		d := user.SelfDaily.Day.Add(tt)
		user.SelfDaily = new(DailyRecord)
		user.SelfDaily.Day = d
		user.SelfDaily.istodayloaded = false
		user.SelfDaily.user = user
		user.SelfDaily.Save()
	}
	for i := 0; i < recordlen; i++ {
		user.SelfDaily = NewDailyRecord(user, int32(i))
		if user.SelfDaily == nil {
			weblog.ErrorLog("load daily record index %d failed.", i)
			return false
		}
		if user.SelfDaily.Day.Year() == day.Year() && user.SelfDaily.Day.YearDay() == day.YearDay() {
			return true
		}
	}
	weblog.ErrorLog("find daliyrecore for %v failed.", day)
	return false
}

func (user *User) UpdateStepRecord(oldstep int, newstep int) bool {
	if oldstep == newstep {
		return true
	}
	dbconn := dbpool.Get()
	var err error
	user.WeekStepNum, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "weeksteps"))
	if err != nil {
		weblog.ErrorLog("get user weeksteps failed in user UpdateStepRecord.errinfo: %s", err.Error())
		return false
	}
	user.MonthStepNum, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "monthsteps"))
	if err != nil {
		weblog.ErrorLog("get user monthsteps failed in user UpdateStepRecord.errinfo: %s", err.Error())
		return false
	}
	user.WeekStepNum -= oldstep
	user.WeekStepNum += newstep
	user.MonthStepNum -= oldstep
	user.MonthStepNum += newstep

	_, err = redis.Bool(dbconn.Do("HMSET", "id:"+user.Id, "weeksteps", user.WeekStepNum, "monthsteps", user.MonthStepNum))
	if err != nil {
		weblog.ErrorLog("set userinfo failed in user UpdateStepRecord.errinfo: %s", err.Error())
		return false
	}

	_, err = redis.Bool(dbconn.Do("ZADD", "weeksteps", user.WeekStepNum, user.Id))
	if err != nil {
		weblog.ErrorLog("set weeksteps failed in user UpdateStepRecord.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("ZADD", "monthsteps", user.MonthStepNum, user.Id))
	if err != nil {
		weblog.ErrorLog("set monthsteps failed in user UpdateStepRecord.errinfo: %s", err.Error())
		return false
	}

	return true
}

func (user *User) UpdateDistanceRecord(olddistance int, newdistance int) bool {
	if olddistance == newdistance {
		return true
	}
	dbconn := dbpool.Get()
	var err error
	user.WeekDistance, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "weekdistance"))
	if err != nil {
		weblog.ErrorLog("get user weekdistance failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}
	user.MonthDistance, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "monthdistance"))
	if err != nil {
		weblog.ErrorLog("get user monthdistance failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}
	user.WeekDistance -= olddistance
	user.WeekDistance += newdistance
	user.MonthDistance -= olddistance
	user.MonthDistance += newdistance

	_, err = redis.Bool(dbconn.Do("HMSET", "id:"+user.Id, "weekdistance", user.WeekDistance, "monthdistance", user.MonthDistance))
	if err != nil {
		weblog.ErrorLog("set userinfo failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}

	_, err = redis.Bool(dbconn.Do("ZADD", "weekdistance", user.WeekDistance, user.Id))
	if err != nil {
		weblog.ErrorLog("set weekdistance failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("ZADD", "monthdistance", user.MonthDistance, user.Id))
	if err != nil {
		weblog.ErrorLog("set monthdistance failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}

	return true
}

func GetUserByAppid(appid string) (user *User) {
	dbconn := dbpool.Get()
	var err error
	user = new(User)
	user.Appid = appid
	user.Id, err = redis.String(dbconn.Do("GET", "appid", user.Appid))
	if err != nil {
		weblog.ErrorLog("get userid failed with appid.errinfo: %s", err.Error())
		return
	}
	if user.Id == "" {
		// haven't registed
		user.IsLoad = false
		return
	}
	user.Load()
	return
}

func (user *User) Load() (ret bool) {
	dbconn := dbpool.Get()
	var err error
	user.Name, err = redis.String(dbconn.Do("HGET", "id:"+user.Id, "name"))
	if err != nil {
		weblog.ErrorLog("get user name failed in user Load.errinfo: %s", err.Error())
		return false
	}
	user.Img, err = redis.String(dbconn.Do("HGET", "id:"+user.Id, "img"))
	if err != nil {
		weblog.ErrorLog("get user img failed in user Load.errinfo: %s", err.Error())
		return false
	}
	user.WeekStepNum, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "weeksteps"))
	if err != nil {
		weblog.ErrorLog("get user weeksteps failed in user Load.errinfo: %s", err.Error())
		return false
	}
	user.MonthStepNum, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "monthsteps"))
	if err != nil {
		weblog.ErrorLog("get user monthsteps failed in user Load.errinfo: %s", err.Error())
		return false
	}
	user.WeekDistance, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "weekdistance"))
	if err != nil {
		weblog.ErrorLog("get user weekdistance failed in user Load.errinfo: %s", err.Error())
		return false
	}
	user.MonthDistance, err = redis.Int(dbconn.Do("HGET", "id:"+user.Id, "monthdistance"))
	if err != nil {
		weblog.ErrorLog("get user monthdistance failed in user Load.errinfo: %s", err.Error())
		return false
	}
	return true
}

func (user *User) IsExist() bool {
	exist, err := redis.Bool(dbpool.Get().Do("HEXISTS", "id:"+user.Id, "name"))
	if err != nil {
		weblog.ErrorLog("HEXISTS failed in user save.errinfo: %s", err.Error())
		return false
	}
	return exist
}

func (user *User) Save() (ret bool) {
	dbconn := dbpool.Get()
	_, err := redis.Bool(dbconn.Do("SET", "appid", user.Appid, user.Id))
	if err != nil {
		weblog.ErrorLog("set appid failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("HMSET", "id:"+user.Id, "name", user.Name, "img", user.Img, "weeksteps", user.WeekStepNum, "monthsteps", user.MonthStepNum, "weekdistance", user.WeekDistance, "monthdistance", user.MonthDistance))
	if err != nil {
		weblog.ErrorLog("set userinfo failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("ZADD", "weeksteps", user.WeekStepNum, user.Id))
	if err != nil {
		weblog.ErrorLog("set weeksteps failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("ZADD", "monthsteps", user.MonthStepNum, user.Id))
	if err != nil {
		weblog.ErrorLog("set monthsteps failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("ZADD", "weekdistance", user.WeekDistance, user.Id))
	if err != nil {
		weblog.ErrorLog("set weekdistance failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = redis.Bool(dbconn.Do("ZADD", "monthdistance", user.MonthDistance, user.Id))
	if err != nil {
		weblog.ErrorLog("set monthdistance failed in user save.errinfo: %s", err.Error())
		return false
	}
	return true
}

func GetTopWeekStepUsers(topnum int) (users []*User) {
	users = make([]*User, topnum)
	dbconn := dbpool.Get()
	ids, err := redis.Strings(dbconn.Do("ZREVRANGE", "weeksteps", 0, topnum-1))
	if err != nil {
		weblog.ErrorLog("get weeksteps top %d ids failed.errinfo: %s", topnum, err.Error())
		return nil
	}
	for idx, id := range ids {
		users[idx] = new(User)
		users[idx].Id = id
		users[idx].Load()
	}
	return
}

func GetTopMonthStepUsers(topnum int) (users []*User) {
	users = make([]*User, topnum)
	dbconn := dbpool.Get()
	ids, err := redis.Strings(dbconn.Do("ZREVRANGE", "monthsteps", 0, topnum-1))
	if err != nil {
		weblog.ErrorLog("get monthsteps top %d ids failed.errinfo: %s", topnum, err.Error())
		return nil
	}
	for idx, id := range ids {
		users[idx] = new(User)
		users[idx].Id = id
		users[idx].Load()
	}
	return
}
