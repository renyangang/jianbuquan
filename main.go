// jianbuquan project main.go
package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
)

const (
	ListDir    = 0x0001
	PAGE_DIR   = "./page"
	UPLOAD_DIR = "./"
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
		log.Println("Loading template:", templatePath)
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
	renderHtml(w, "regpage", nil)
}
func dailyHandler(w http.ResponseWriter, r *http.Request) {
	renderHtml(w, "dailyreport", nil)
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
				http.Error(w, e.Error(), http.StatusInternalServerError)
				// 或者输出自定义的50x错误页面
				// w.WriteHeader(http.StatusInternalServerError)
				// renderHtml(w, "error", e)
				// logging
				log.Println("WARN: panic in %v. - %v", fn, e)
				log.Println(string(debug.Stack()))
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
func main() {
	mux := http.NewServeMux()
	staticDirHandler(mux, "/assets/", "./public", 0)
	mux.HandleFunc("/", safeHandler(rootHandler))
	mux.HandleFunc("/register", safeHandler(regHandler))
	mux.HandleFunc("/dailyreport", safeHandler(dailyHandler))
	mux.HandleFunc("/ranking", safeHandler(rankingHandler))
	mux.HandleFunc("/mainpage", safeHandler(mainHandler))
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
