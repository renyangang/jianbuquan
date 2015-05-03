// jianbuquan project main.go
package main

import (
	"net/http"
	"webhandler"
	"weblog"
)

func main() {
	mux := http.NewServeMux()
	webhandler.RegisterHandler(mux)
	err := http.ListenAndServe(":80", mux)
	if err != nil {
		weblog.ErrorLog("ListenAndServe: ", err.Error())
	}
}
