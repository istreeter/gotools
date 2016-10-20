package main

import (
    "log"
    "net/http"
    "time"
    "errors"
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

  wrappedHandler := jsonhttp.HandleWithMsgs(handler, 2 * time.Second)

  srv := &http.Server{
    Handler: wrappedHandler,
    Addr: ":8000",
  }

  log.Fatal(srv.ListenAndServe())

}

