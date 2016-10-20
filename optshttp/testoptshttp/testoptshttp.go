package main

import (
    "log"
    "net/http"
    "github.com/istreeter/gotools/optshttp"
    "fmt"
    "time"
)

type myHandler struct{}

func (h myHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  if err, ok := req.Context().Value(optshttp.ErrorKey).(error); ok {
    w.Write([]byte(err.Error()))
    return
  }
  if myText, ok := req.Context().Value("myText").(string); ok {
    w.Write([]byte(fmt.Sprintf("myText %q", myText)))
    return
  }
  if myInt, ok := req.Context().Value("myInt").(int); ok {
    w.Write([]byte(fmt.Sprintf("myInt * 100 = %d", myInt * 100)))
    return
  }
  if myDate, ok := req.Context().Value("myDate").(time.Time); ok {
    w.Write([]byte(fmt.Sprintf("date %v", myDate)))
    return
  }
  w.Write([]byte("hello"))
}

func main() {

  myTypes := make(map[string]int)
  myTypes["myText"] = optshttp.String
  myTypes["myInt"] = optshttp.Int
  myTypes["myDate"] = optshttp.RFC3339Nano

  handler := myHandler{}
  formHandler := &optshttp.FormHandler{
    H: handler,
    Types: myTypes,
  }

  srv := &http.Server{
    Handler: formHandler,
    Addr: ":8000",
  }

  log.Fatal(srv.ListenAndServe())

}

