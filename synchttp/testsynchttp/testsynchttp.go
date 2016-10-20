package main

import (
    "log"
    "net/http"
    "time"
    "github.com/istreeter/gotools/synchttp"
)

type myHandler struct{}

func (h myHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  log.Println("myHandler writing")
  w.Write([]byte("hello"))
}

type serverErrorHandler []byte

func (s serverErrorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  http.Error(w, string(s), http.StatusInternalServerError)
}

type ctxDoneHandler []byte
func (h ctxDoneHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  http.Error(w, string(h), http.StatusServiceUnavailable)
}

func main() {

  handler := myHandler{}
  t := synchttp.CtxDoneHandler{H: ctxDoneHandler("this is a timeout")}
  s := serverErrorHandler("this is a server error")
  wrappedHandler := synchttp.HandleWithMsgs(handler, s, t, 2 * time.Second)

  srv := &http.Server{
    Handler: wrappedHandler,
    Addr: ":8000",
  }

  log.Fatal(srv.ListenAndServe())

}

