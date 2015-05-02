// webhandler project webhandler.go
package webhandler

import (
	"dataobj"
	"errors"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"weblog"
)

const (
	ListDir    = 0x0001
	PAGE_DIR   = "./page"
	UPLOAD_DIR = "./public/dailyimg/"
)

var templates map[string]*template.Template

func ttinit() {
	templates = make(map[string]*template.Template)
	fileInfoArr, err := ioutil.ReadDir(PAGE_DIR)
	check(err)

	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}
		templatePath = PAGE_DIR + "/" + templateName
		weblog.DebugLog("Loading template:", templatePath)
		t := template.Must(template.ParseFiles(templatePath))
		tmpl := strings.Replace(path.Base(templateName), ".html", "", -1)
		templates[tmpl] = t
	}
}
func check(err error) {
	if err != nil {
		panic(err)
	}
}
func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) {
	ttinit()
	w.Header().Add("content-type", "text/html")
	err := templates[tmpl].Execute(w, locals)
	check(err)
}
func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderHtml(w, "upload", nil)
	}
	if r.Method == "POST" {
		f, h, err := r.FormFile("image")
		check(err)
		filename := h.Filename
		defer f.Close()
		t, err := ioutil.TempFile(UPLOAD_DIR, filename)
		check(err)
		defer t.Close()
		_, err = io.Copy(t, f)
		check(err)
		http.Redirect(w, r, "/view?id="+filename, http.StatusFound)
	}
}
func regHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderHtml(w, "regpage", nil)
	} else if r.Method == "POST" {
		user := new(dataobj.User)
		user.Appid = r.FormValue("appid")
		user.Id = r.FormValue("id")
		user.Name = r.FormValue("name")
		if user.IsExist() {
			// id冲突
			panic(errors.New("该用户已经存在，请检查id和姓名"))
			return
		}
		ret := true
		olduser := dataobj.GetUserByAppid(user.Appid)
		if olduser.IsLoad {
			ret = user.UpdateItem(olduser)
		} else {
			ret = user.Save()
		}
		if !ret {
			// 失败页面
			panic(errors.New("保存注册信息失败"))
			return
		}
		// 成功直接跳转到每日数据提交页面
		http.Redirect(w, r, "/dailyreport?id="+user.Id, http.StatusFound)
	}

}
func dailyHandler(w http.ResponseWriter, r *http.Request) {
	user := new(dataobj.User)
	user.Id = r.FormValue("id")
	day := time.Now()
	if !user.IsExist() || !user.Load() || !user.GetDailyRecord(day) {
		// 用户不存在,错误页面
		panic(errors.New("用户" + user.Id + "不存在，请注册后从微信提示信息入口处进入"))
		return
	}
	if r.Method == "GET" {
		locals := make(map[string]interface{})
		locals["user"] = user
		renderHtml(w, "dailyreport", locals)
	} else if r.Method == "POST" {
		user.SelfDaily.StepNum, _ = strconv.Atoi(r.FormValue("steps"))
		user.SelfDaily.Distance, _ = strconv.Atoi(r.FormValue("distance"))

		f, h, err := r.FormFile("img")
		check(err)
		defer f.Close()
		filepath := UPLOAD_DIR + user.Id
		err = os.MkdirAll(filepath, os.ModeDir)
		check(err)
		user.SelfDaily.Img = filepath + user.SelfDaily.GetDateStr() + path.Ext(h.Filename)
		t, err := os.OpenFile(user.SelfDaily.Img, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModeType)
		check(err)
		defer t.Close()
		_, err = io.Copy(t, f)
		check(err)
		user.SelfDaily.Img = strings.Replace(user.SelfDaily.Img, "./public", "/assets", -1)
		ret := user.SelfDaily.Save()
		if !ret {
			// 失败页面
			panic(errors.New("今日记录保存失败。"))
			return
		}
		// 成功页面
		http.Redirect(w, r, "/detail?id="+user.Id+"&weeknum=0", http.StatusFound)
	}
}

func detailHandler(w http.ResponseWriter, r *http.Request) {
	user := new(dataobj.User)
	user.Id = r.FormValue("id")
	weeknum, err := strconv.Atoi(r.FormValue("weeknum"))
	day := time.Now()
	if err != nil || !user.IsExist() || !user.Load() || !user.GetDailyRecord(day) {
		// 用户不存在,错误页面
		panic(errors.New("用户" + user.Id + "不存在，请注册后从微信提示信息入口处进入"))
		return
	}
	ret, hasbefore := user.GetDailyRecords(weeknum)
	if !ret {
		panic(errors.New("获取用户记录失败"))
	}
	before := weeknum + 1
	after := weeknum - 1
	if !hasbefore {
		before = -1
	}
	locals := make(map[string]interface{})
	locals["user"] = user
	locals["before"] = before
	locals["after"] = after
	renderHtml(w, "detail", locals)
}

func rankingHandler(w http.ResponseWriter, r *http.Request) {
	renderHtml(w, "ranking", nil)
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/register", http.StatusFound)
}
func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				//http.Error(w, e.Error(), http.StatusInternalServerError)
				// 或者输出自定义的50x错误页面
				w.WriteHeader(http.StatusInternalServerError)
				locals := make(map[string]interface{})
				locals["error"] = e.Error()
				renderHtml(w, "error", locals)
				// logging
				weblog.ErrorLog("WARN: panic in %v. - %v", fn, e)
				weblog.ErrorLog(string(debug.Stack()))
			}
		}()
		fn(w, r)
	}
}
func staticDirHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix)-1:]
		if (flags & ListDir) == 0 {
			if exists := isExists(file); !exists {
				http.NotFound(w, r)
				return
			}
		}
		http.ServeFile(w, r, file)
	})
}

func RegisterHandler(mux *http.ServeMux) {
	staticDirHandler(mux, "/assets/", "./public", 0)
	mux.HandleFunc("/", safeHandler(rootHandler))
	mux.HandleFunc("/register", safeHandler(regHandler))
	mux.HandleFunc("/dailyreport", safeHandler(dailyHandler))
	mux.HandleFunc("/ranking", safeHandler(rankingHandler))
	mux.HandleFunc("/detail", safeHandler(detailHandler))
	mux.HandleFunc("/mainpage", safeHandler(mainHandler))
}
