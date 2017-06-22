package jsonhttp_test

import (
  "github.com/istreeter/gotools/jsonhttp"
  "net/http"
  "net/http/httptest"
  "fmt"
  "time"
  "errors"
)

func ExampleNewErrorHandler() {

  errorHandler := jsonhttp.NewErrorHandler("You made an error", http.StatusBadRequest)

  req := httptest.NewRequest("GET", "http://example.com/foo", nil)
  w := httptest.NewRecorder()
  errorHandler.ServeHTTP(w, req)
  fmt.Printf("%d - %s - %s", w.Code, w.HeaderMap["Content-Type"], w.Body.String())

  // Output: 400 - [application/json; charset=UTF-8] - {"error":true,"message":"You made an error","name":"Bad Request"}
}

func ExampleError() {

  handler := func(w http.ResponseWriter, req *http.Request) {
    jsonhttp.Error(w, "You made an error", http.StatusBadRequest)
  }

  req := httptest.NewRequest("GET", "http://example.com/foo", nil)
  w := httptest.NewRecorder()
  handler(w, req)
  fmt.Printf("%d - %s - %s", w.Code, w.HeaderMap["Content-Type"], w.Body.String())

  // Output: 400 - [application/json; charset=UTF-8] - {"error":true,"message":"You made an error","name":"Bad Request"}
}

func ExampleOK() {

  handler := func(w http.ResponseWriter, req *http.Request) {
    data := map[string]interface{}{"status": "good", "ok": true, "errors": 0}
    jsonhttp.OK(w, data)
  }

  req := httptest.NewRequest("GET", "http://example.com/foo", nil)
  w := httptest.NewRecorder()
  handler(w, req)
  fmt.Printf("%d - %s - %s", w.Code, w.HeaderMap["Content-Type"], w.Body.String())

  // Output: 200 - [application/json; charset=UTF-8] - {"errors":0,"ok":true,"status":"good"}
}

func ExampleHandleWithMsgs() {

  data := map[string]interface{}{"status": "good", "ok": true, "errors": 0}
  var delay time.Duration

  goodHandler := func(w http.ResponseWriter, req *http.Request) {
    time.Sleep(delay)
    jsonhttp.OK(w, data)
  }

  badHandler := func(w http.ResponseWriter, req *http.Request) {
    panic(errors.New("making a panic"))
  }

  handler := jsonhttp.HandleWithMsgs(http.HandlerFunc(goodHandler), 100 * time.Millisecond)
  req := httptest.NewRequest("GET", "http://example.com/foo", nil)

  w := httptest.NewRecorder()
  delay = 50 * time.Millisecond
  handler.ServeHTTP(w, req)
  fmt.Printf("First response: %d - %s - %s", w.Code, w.HeaderMap["Content-Type"], w.Body.String())

  w = httptest.NewRecorder()
  delay = 200 * time.Millisecond
  handler.ServeHTTP(w, req)
  fmt.Printf("Second response: %d - %s - %s", w.Code, w.HeaderMap["Content-Type"], w.Body.String())

  handler = jsonhttp.HandleWithMsgs(http.HandlerFunc(badHandler), 100 * time.Millisecond)
  w = httptest.NewRecorder()
  handler.ServeHTTP(w, req)
  fmt.Printf("Third response: %d - %s - %s", w.Code, w.HeaderMap["Content-Type"], w.Body.String())

  // Output:
  // First response: 200 - [application/json; charset=UTF-8] - {"errors":0,"ok":true,"status":"good"}
  // Second response: 503 - [application/json; charset=UTF-8] - {"error":true,"message":"Server Timeout","name":"Service Unavailable"}
  // Third response: 500 - [application/json; charset=UTF-8] - {"error":true,"message":"Server Error","name":"Internal Server Error"}
}
