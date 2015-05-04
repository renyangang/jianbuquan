// dataobj project dataobj.go
package dataobj

import (
	"bytes"
	"encoding/binary"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"time"
	"weblog"
)

var dbpool *redis.Pool

func init() {
	dbpool = redis.NewPool(getConn, 5)
}

func getConn() (redis.Conn, error) {
	return redis.Dial("tcp", "127.0.0.1:8432")
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
	weblog.DebugLog("create dailyrec with[%v]", idx, dr.Day)
	if !dr.Load() {
		return nil
	}
	return
}

func (dr *DailyRecord) UnSerialization(bs []byte) bool {
	dr.istodayloaded = false
	dr.version = 0
	dr.Day = time.Now()
	weblog.DebugLog("bs is %v", bs)
	if bs == nil || len(bs) <= 0 {
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
		weblog.ErrorLog("decode for DailyRecord.Day.Unix failed.errinfo:%s", err.Error())
		return false
	}
	dr.Day = time.Unix(t, 0)
	weblog.DebugLog("decode day with[%v]", t, dr.Day)
	var tempint int32
	err = binary.Read(buf, binary.BigEndian, &tempint)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.StepNum failed.errinfo:%s", err.Error())
		return false
	}
	dr.StepNum = int(tempint)
	dr.oldStepNum = dr.StepNum
	err = binary.Read(buf, binary.BigEndian, &tempint)
	if err != nil {
		weblog.ErrorLog("decode for DailyRecord.Distance failed.errinfo:%s", err.Error())
		return false
	}
	dr.Distance = int(tempint)
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
	dbconn := dbpool.Get()
	recordlen, err := redis.Int(dbconn.Do("LLEN", dr.user.Id))
	if err != nil {
		weblog.ErrorLog("get dailyrecord len failed in Load.errinfo: %s", err.Error())
		return false
	}
	if int32(recordlen) <= dr.index {
		return dr.UnSerialization(nil)
	}
	bs, err := redis.Bytes(dbconn.Do("LINDEX", dr.user.Id, dr.index))
	if err != nil {
		weblog.ErrorLog("get first dailyrecord failed.errinfo:%s", err.Error())
		return false
	}
	return dr.UnSerialization(bs)
}

func (dr *DailyRecord) Serialization() (bs []byte, err error) {
	bs = []byte{}
	buf := bytes.NewBuffer(bs)
	err = binary.Write(buf, binary.BigEndian, dr.version)
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.version failed.errinfo:%s", err.Error())
		return
	}
	weblog.DebugLog("day serial %v", dr.Day.Unix(), dr.Day)
	err = binary.Write(buf, binary.BigEndian, dr.Day.Unix())
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.Day.Unix failed.errinfo:%s", err.Error())
		return
	}
	err = binary.Write(buf, binary.BigEndian, int32(dr.StepNum))
	if err != nil {
		weblog.ErrorLog("encode for DailyRecord.StepNum failed.errinfo:%s", err.Error())
		return
	}
	err = binary.Write(buf, binary.BigEndian, int32(dr.Distance))
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
	dr.user.UpdateStepRecord(dr)
	dr.user.UpdateDistanceRecord(dr.oldDistance, dr.Distance)
	return true
}

func (dr *DailyRecord) GetDateStr() string {
	return dr.Day.Format("2006-01-02")
}

type User struct {
	Appid         string
	Id            string
	Name          string
	Img           string
	WeekStepNum   int
	MonthStepNum  int
	StepNums      int
	WeekDistance  int
	MonthDistance int
	IsLoad        bool
	Ranking       int
	SelfDailys    []*DailyRecord
	SelfDaily     *DailyRecord
}

// 以当前时间为基数，weeknum获取一周的数据，0位本周，1为上周，依次类推
func (user *User) GetDailyRecords(weeknum int) (ret bool, hasbefore bool) {
	hasbefore = true
	ret = false
	day := time.Now()
	// api一周从周日开始计，我们这里从周一开始
	weekday := day.Weekday()
	daynum := int(weekday)
	subdaynum := 0
	// 计算出周日
	if weeknum > 0 {
		subdaynum = int(weekday) + (7 * (weeknum - 1))
		day = day.AddDate(0, 0, subdaynum*-1)
		daynum = 7
	} else if daynum == 0 {
		daynum = 7 //今天是周日
	}
	if !user.GetDailyRecord(day) {
		return
	}
	user.SelfDailys = make([]*DailyRecord, daynum)
	dbconn := dbpool.Get()
	recordlen, err1 := redis.Int(dbconn.Do("LLEN", user.Id))
	if err1 != nil {
		weblog.ErrorLog("get dailyrecord len failed in GetDailyRecordS.errinfo: %s", err1.Error())
		return
	}
	if subdaynum+7 >= recordlen {
		hasbefore = false
	}
	bsarray, err := redis.Values(dbconn.Do("LRANGE", user.Id, subdaynum, daynum))
	if err != nil {
		weblog.ErrorLog("get dailyrecords failed in GetDailyRecord.errinfo: %s", err.Error())
		return
	}
	if bsarray == nil || len(bsarray) == 0 {
		ret = true
	}
	idx := 0
	for idx = 0; idx < len(bsarray); idx++ {
		user.SelfDailys[daynum-idx-1] = new(DailyRecord)
		user.SelfDailys[daynum-idx-1].user = user
		user.SelfDailys[daynum-idx-1].UnSerialization(bsarray[idx].([]byte))
	}
	for idx < daynum {
		user.SelfDailys[daynum-idx-1] = new(DailyRecord)
		user.SelfDailys[daynum-idx-1].user = user
		user.SelfDailys[daynum-idx-1].StepNum = 0
		user.SelfDailys[daynum-idx-1].Distance = 0
		user.SelfDailys[daynum-idx-1].Img = ""
		user.SelfDailys[daynum-idx-1].Day = day.AddDate(0, 0, idx*-1)
		idx++
	}
	return true, hasbefore
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
	for user.SelfDaily.Day.Before(day) && (user.SelfDaily.Day.YearDay() != day.YearDay() || user.SelfDaily.Day.Year() != day.Year()) {
		weblog.DebugLog("record day[%v] before than day[%v]", user.SelfDaily.Day, day)
		tt, _ := time.ParseDuration("24h")
		d := user.SelfDaily.Day.Add(tt)
		weblog.DebugLog("insert day[%v]", d)
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
	weblog.ErrorLog("find daliyrecord for %v failed.", day)
	return false
}

func (user *User) UpdateStepRecord(dr *DailyRecord) bool {
	if dr.oldStepNum == dr.StepNum {
		return true
	}
weekkey := GetWeekStepKey(dr.Day, 0)
	monthkey := GetMounthStepKey(dr.Day, 0)

	dbconn := dbpool.Get()
	var err error
	_, err = dbconn.Do("ZINCRBY", weekkey, dr.StepNum-dr.oldStepNum, user.Id)
	if err != nil {
		weblog.ErrorLog("set weeksteps failed in user UpdateStepRecord.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("ZINCRBY", monthkey, dr.StepNum-dr.oldStepNum, user.Id)
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

	_, err = dbconn.Do("HMSET", "id:"+user.Id, "weekdistance", user.WeekDistance, "monthdistance", user.MonthDistance)
	if err != nil {
		weblog.ErrorLog("set userinfo failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}

	_, err = dbconn.Do("ZADD", "weekdistance", user.WeekDistance, user.Id)
	if err != nil {
		weblog.ErrorLog("set weekdistance failed in user UpdateDistanceRecord.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("ZADD", "monthdistance", user.MonthDistance, user.Id)
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
	user.IsLoad = false
	user.Id, err = redis.String(dbconn.Do("HGET", "appid:id", user.Appid))
	if err != nil {
		weblog.ErrorLog("get userid failed with appid.errinfo: %s", err.Error())
		return
	}
	if user.Id == "" {
		// haven't registed
		user.IsLoad = false
		return
	}
	user.IsLoad = true
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
	user.IsLoad = true
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
	_, err := dbconn.Do("HSET", "appid:id", user.Appid, user.Id)
	if err != nil {
		weblog.ErrorLog("set appid failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("HMSET", "id:"+user.Id, "name", user.Name, "img", user.Img, "weeksteps", user.WeekStepNum, "monthsteps", user.MonthStepNum, "weekdistance", user.WeekDistance, "monthdistance", user.MonthDistance)
	if err != nil {
		weblog.ErrorLog("set userinfo failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("ZADD", GetWeekStepKey(time.Now(), 0), user.WeekStepNum, user.Id)
	if err != nil {
		weblog.ErrorLog("set weeksteps failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("ZADD", GetMounthStepKey(time.Now(), 0), user.MonthStepNum, user.Id)
	if err != nil {
		weblog.ErrorLog("set monthsteps failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("ZADD", "weekdistance", user.WeekDistance, user.Id)
	if err != nil {
		weblog.ErrorLog("set weekdistance failed in user save.errinfo: %s", err.Error())
		return false
	}
	_, err = dbconn.Do("ZADD", "monthdistance", user.MonthDistance, user.Id)
	if err != nil {
		weblog.ErrorLog("set monthdistance failed in user save.errinfo: %s", err.Error())
		return false
	}
	return true
}

func (user *User) UpdateItem(olduser *User) (ret bool) {
	dbconn := dbpool.Get()
	var err error
	// 可以修改的只有id、name、img
	if user.Id != olduser.Id {
		_, err = dbconn.Do("HSET", "appid:id", user.Appid, user.Id)
		if err != nil {
			weblog.ErrorLog("set appid:id failed in user update")
			return false
		}
		_, err = dbconn.Do("RENAME", "id:"+olduser.Id, "id:"+user.Id)
		if err != nil {
			weblog.ErrorLog("set user info id failed in user update")
			return false
		}
		_, err = dbconn.Do("RENAME", olduser.Id, user.Id)
		if err != nil {
			weblog.ErrorLog("set user daliyrecord id failed in user update")
			return false
		}
		_, err = dbconn.Do("ZREM", GetWeekStepKey(time.Now(), 0), olduser.Id)
		if err != nil {
			weblog.ErrorLog("remove weeksteps failed in user update")
			return false
		}
		_, err = dbconn.Do("ZREM", GetMounthStepKey(time.Now(), 0), olduser.Id)
		if err != nil {
			weblog.ErrorLog("remove monthsteps failed in user update")
			return false
		}
		_, err = dbconn.Do("ZREM", "weekdistance", olduser.Id)
		if err != nil {
			weblog.ErrorLog("remove weekdistance failed in user update")
			return false
		}
		_, err = dbconn.Do("ZREM", "monthdistance", olduser.Id)
		if err != nil {
			weblog.ErrorLog("remove monthdistance failed in user update")
			return false
		}
		_, err = dbconn.Do("ZADD", GetWeekStepKey(time.Now(), 0), olduser.WeekStepNum, user.Id)
		if err != nil {
			weblog.ErrorLog("set weeksteps failed in user update.errinfo: %s", err.Error())
			return false
		}
		_, err = dbconn.Do("ZADD", GetMounthStepKey(time.Now(), 0), olduser.MonthStepNum, user.Id)
		if err != nil {
			weblog.ErrorLog("set monthsteps failed in user update.errinfo: %s", err.Error())
			return false
		}
		_, err = dbconn.Do("ZADD", "weekdistance", olduser.WeekDistance, user.Id)
		if err != nil {
			weblog.ErrorLog("set weekdistance failed in user update.errinfo: %s", err.Error())
			return false
		}
		_, err = dbconn.Do("ZADD", "monthdistance", olduser.MonthDistance, user.Id)
		if err != nil {
			weblog.ErrorLog("set monthdistance failed in user update.errinfo: %s", err.Error())
			return false
		}
	}
	if user.Name != olduser.Name {
		_, err = dbconn.Do("HSET", "id:"+user.Id, "name", user.Name)
		if err != nil {
			weblog.ErrorLog("set name failed in user update")
			return false
		}
	}
	if user.Img != olduser.Img {
		_, err = dbconn.Do("HSET", "id:"+user.Id, "img", user.Img)
		if err != nil {
			weblog.ErrorLog("set img failed in user update")
			return false
		}
	}
	return true
}

//base分页基数，topnum条数，num当前为基准0本周，1上周...
func GetTopStepUsers(key string, base int, topnum int, num int) (users []*User) {
	dbconn := dbpool.Get()
	ids, err := redis.Strings(dbconn.Do("ZREVRANGE", key, base, topnum-1, "WITHSCORES"))
	if err != nil {
		weblog.ErrorLog("get [%s]steps top %d ids with base,num[%d,%d] failed.errinfo: %s", key, topnum, base, num, err.Error())
		return nil
	}
	users = make([]*User, len(ids)/2)
	for idx, id := range ids {
		weblog.DebugLog("[%v]", idx, id)
		if idx%2 == 0 {
			users[idx/2] = new(User)
			users[idx/2].Id = id
			users[idx/2].Load()
			users[idx/2].Ranking = idx/2 + 1
		} else {
			users[idx/2].StepNums, err = strconv.Atoi(id)
			if (idx/2 > 0) && users[idx/2].StepNums == users[idx/2-1].StepNums {
				users[idx/2].Ranking = users[idx/2-1].Ranking
			}
		}
	}
	return
}
func GetTopWeekStepUsers(base int, topnum int, num int) (users []*User) {
	return GetTopStepUsers(GetWeekStepKey(time.Now(), num), base, topnum, num)
}

func GetTopMonthStepUsers(base int, topnum int, num int) (users []*User) {
	return GetTopStepUsers(GetMounthStepKey(time.Now(), num), base, topnum, num)
}

func GetWeekStepKey(day time.Time, num int) string {
	today := int(day.Weekday())
	if today == 0 {
		today = 7
	}
	distance := (today - 1) + (num * 7)
	monday := day.AddDate(0, 0, distance*-1)
	return "weeksteps:" + monday.Format("2006-01-02")
}

func GetMounthStepKey(day time.Time, num int) string {
	month := day.AddDate(0, num*-1, 0)
	return "monthsteps:" + month.Month().String()
}
