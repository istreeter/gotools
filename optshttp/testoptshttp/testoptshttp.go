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
  myForm := form{}
  if err := optshttp.UnmarshalForm(req, &myForm); err != nil {
    w.Write([]byte(err.Error()))
  }
  w.Write([]byte(fmt.Sprintf("Text is: %s\n", myForm.Text)))
  w.Write([]byte(fmt.Sprintf("2* Num is: %d\n", 2*myForm.Num)))
  w.Write([]byte(fmt.Sprintf("Date is: %v\n", myForm.Date)))
  if myForm.PText != nil {
    w.Write([]byte(fmt.Sprintf("PText is %s\n", *myForm.PText)))
  }
  if myForm.PNum != nil {
    w.Write([]byte(fmt.Sprintf("PNum is %d\n", 2* *myForm.PNum)))
  }
  if myForm.PDate != nil {
    w.Write([]byte(fmt.Sprintf("PDate is %v\n", *myForm.PDate)))
  }
}

type form struct{
  Text string `form:"text"`
  Num int `form:"num"`
  Date time.Time `form:"date"`
  PText *string `form:"ptext"`
  PNum *int `form:"pint"`
  PDate *time.Time `form:"pdate"`
}

func main() {

  handler := myHandler{}

  srv := &http.Server{
    Handler: handler,
    Addr: ":8000",
  }

  log.Fatal(srv.ListenAndServe())

}

