package optshttp_test

import (
  "github.com/istreeter/gotools/optshttp"
  "github.com/gorilla/mux"
  "net/http"
  "net/http/httptest"
  "time"
  "fmt"
)

func ExampleUnmarshalForm() {

  type tParams struct {
    Str   string     `form:"str_a"`
    PStr  *string    `form:"str_b"`
    Embedded struct {
      Int   int      `form:"int_a"`
    }                `form:",inline"`
    PInt  *int       `form:"int_b"`
    Date  time.Time  `form:"date_a"`
    PDate *time.Time `form:"date_b"`
  }

  url := `http://example.com/?str_a=foo&str_b=bar&int_a=100&int_b=200&date_a=1985-10-26T01:21:00Z&date_b=1834-08-01T12:00:00Z`
  req, _ := http.NewRequest("GET", url, nil)

  params := &tParams{}
  if err := optshttp.UnmarshalForm(req, params); err != nil {
    panic(err)
  }

  fmt.Println("Str is:", params.Str)
  fmt.Println("PStr is:", *params.PStr)
  fmt.Println("Int is:", params.Embedded.Int)
  fmt.Println("PInt is:", *params.PInt)
  fmt.Println("Date is:", params.Date)
  fmt.Println("PDate is:", *params.PDate)

  // Output:
  // Str is: foo
  // PStr is: bar
  // Int is: 100
  // PInt is: 200
  // Date is: 1985-10-26 01:21:00 +0000 UTC
  // PDate is: 1834-08-01 12:00:00 +0000 UTC
}

func ExampleUnmarshalPath() {

  type tParams struct {
    Str   string     `path:"str_a"`
    PStr  *string    `path:"str_b"`
    Int   int        `path:"int_a"`
    PInt  *int       `path:"int_b"`
    Date  time.Time  `path:"date_a"`
    PDate *time.Time `path:"date_b"`
  }

  handler := func(w http.ResponseWriter, req *http.Request) {
    params := &tParams{}
    if err := optshttp.UnmarshalPath(req, params); err != nil {
      panic(err)
    }

    w.Write([]byte(fmt.Sprintln("Str is:", params.Str)))
    w.Write([]byte(fmt.Sprintln("PStr is:", *params.PStr)))
    w.Write([]byte(fmt.Sprintln("Int is:", params.Int)))
    w.Write([]byte(fmt.Sprintln("PInt is:", *params.PInt)))
    w.Write([]byte(fmt.Sprintln("Date is:", params.Date)))
    w.Write([]byte(fmt.Sprintln("PDate is:", *params.PDate)))
  }

  r := mux.NewRouter()
  r.HandleFunc("/{str_a}/{str_b}/{int_a}/{int_b}/{date_a}/{date_b}", handler)

  url := `http://example.com/foo/bar/100/200/1985-10-26T01:21:00Z/1834-08-01T12:00:00Z`
  req, _ := http.NewRequest("GET", url, nil)

  w := httptest.NewRecorder()
  r.ServeHTTP(w, req)

  fmt.Printf(w.Body.String())


  // Output:
  // Str is: foo
  // PStr is: bar
  // Int is: 100
  // PInt is: 200
  // Date is: 1985-10-26 01:21:00 +0000 UTC
  // PDate is: 1834-08-01 12:00:00 +0000 UTC
}
