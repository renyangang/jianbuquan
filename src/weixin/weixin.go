// weixin project weixin.go
package weixin

import (
	"crypto/sha1"
	"dataobj"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

const (
	TOKEN = "jianbuquan_paas"
)

type Request struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   string `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgId        int    `xml:"MsgId"`
	Event        string `xml:"Event"`
}

type MsgItem struct {
	Title       string `xml:"Articles>item>Title"`
	Description string `xml:"Articles>item>Description"`
	PicUrl      string `xml:"Articles>item>PicUrl"`
	Url         string `xml:"Articles>item>Url"`
}

type Response struct {
	ToUserName   string `xml:"xml>ToUserName"`
	FromUserName string `xml:"xml>FromUserName"`
	CreateTime   string `xml:"xml>CreateTime"`
	MsgType      string `xml:"xml>MsgType"`
	//Content      string    `xml:"xml>Content"`
	ArticleCount string    `xml:"xml>ArticleCount"`
	Items        []MsgItem `xml:"xml>Articles"`
}

func makeitemstr(item *MsgItem) string {
	str := "<item><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description><PicUrl><![CDATA[%s]]></PicUrl><Url><![CDATA[%s]]></Url></item>"
	return fmt.Sprintf(str, item.Title, item.Description, item.PicUrl, item.Url)
}
func makexmlstr(res *Response) string {
	str := "<xml><ToUserName><![CDATA[%s]]></ToUserName><FromUserName><![CDATA[%s]]></FromUserName><CreateTime>%s</CreateTime><MsgType><![CDATA[%s]]></MsgType><ArticleCount>%s</ArticleCount><Articles>%s</Articles></xml>"
	itemstr := ""
	for _, v := range res.Items {
		itemstr += makeitemstr(&v)
	}
	return fmt.Sprintf(str, res.ToUserName, res.FromUserName, res.CreateTime, res.MsgType, res.ArticleCount, itemstr)
}

func str2sha1(data string) string {
	t := sha1.New()
	io.WriteString(t, data)
	return fmt.Sprintf("%x", t.Sum(nil))
}

func CheckSignature(signature string, timestamp string, nonce string) bool {
	strs := append([]string{}, TOKEN)
	strs = append(strs, timestamp)
	strs = append(strs, nonce)
	sort.Strings(strs)
	str := strs[0] + strs[1] + strs[2]
	selfsig := str2sha1(str)
	if selfsig == signature {
		return true
	}
	return false
}

func GetRegRes(host string, req Request, appid string) string {
	res := Response{req.FromUserName, req.ToUserName, strconv.FormatInt(time.Now().Unix(), 10), "news", "1", make([]MsgItem, 1)}
	res.Items[0].Title = "个人信息注册"
	res.Items[0].Description = "注册个人工号/姓名信息，开始步行之旅"
	res.Items[0].PicUrl = "http://" + host + "/assets/img/photo/mainitem.jpg"
	res.Items[0].Url = "http://" + host + "/register?appid=" + appid

	return makexmlstr(&res)
}

func GetNormalRes(host string, req Request, user *dataobj.User) string {
	res := Response{req.FromUserName, req.ToUserName, strconv.FormatInt(time.Now().Unix(), 10), "news", "6", make([]MsgItem, 6)}
	res.Items[0].Title = "今日记录提交"
	res.Items[0].Description = "记录每日步行数据"
	res.Items[0].PicUrl = "http://" + host + "/assets/img/photo/mainitem.jpg"
	res.Items[0].Url = "http://" + host + "/dailyreport?id=" + user.Id

	res.Items[1].Title = "个人记录查看"
	res.Items[1].Description = user.Name + " 点击查看你的记录"
	res.Items[1].PicUrl = "http://" + host + "/assets/img/photo/otheritem.jpg"
	res.Items[1].Url = "http://" + host + "/detail?weeknum=0&id=" + user.Id

	res.Items[2].Title = "本周排行"
	res.Items[2].Description = "查看本周步行达人"
	res.Items[2].PicUrl = "http://" + host + "/assets/img/photo/otheritem.jpg"
	res.Items[2].Url = "http://" + host + "/ranking?type=week&num=0"

	res.Items[3].Title = "上周排行"
	res.Items[3].Description = "查看上周步行达人"
	res.Items[3].PicUrl = "http://" + host + "/assets/img/photo/otheritem.jpg"
	res.Items[3].Url = "http://" + host + "/ranking?type=week&num=1"

	res.Items[4].Title = "本月排行"
	res.Items[4].Description = "查看本月步行达人"
	res.Items[4].PicUrl = "http://" + host + "/assets/img/photo/otheritem.jpg"
	res.Items[4].Url = "http://" + host + "/ranking?type=month&num=0"

	res.Items[5].Title = "上月排行"
	res.Items[5].Description = "查看上月步行达人"
	res.Items[5].PicUrl = "http://" + host + "/assets/img/photo/otheritem.jpg"
	res.Items[5].Url = "http://" + host + "/ranking?type=month&num=1"

	return makexmlstr(&res)
}

func MakeResponse(host string, request []byte) string {
	req := Request{}
	xml.Unmarshal(request, &req)
	appid := req.FromUserName
	user := dataobj.GetUserByAppid(appid)
	if !user.IsLoad {
		//发送注册消息
		return GetRegRes(host, req, appid)
	}
	//发送正常菜单
	return GetNormalRes(host, req, user)
}
