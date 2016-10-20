package main

import (
    "log"
    "net/http"
    "time"
    "errors"
    "github.com/istreeter/gotools/synchttp"
    "github.com/istreeter/gotools/jsonhttp"
)

type myHandler struct{
  Msg string `json:"msg"`
}

func (h *myHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  panic(errors.New("panic"))
  jsonhttp.OK(w, h)
}

func main() {

  handler := &myHandler{Msg: "hello"}
  t := synchttp.CtxDoneHandler{H: jsonhttp.NewErrorHandler("this is a timeout", http.StatusServiceUnavailable)}
  s := jsonhttp.NewErrorHandler("this is a server error", http.StatusInternalServerError)
  wrappedHandler := synchttp.HandleWithMsgs(handler, s, t, 2 * time.Second)

  srv := &http.Server{
    Handler: wrappedHandler,
    Addr: ":8000",
  }

  log.Fatal(srv.ListenAndServe())

}

