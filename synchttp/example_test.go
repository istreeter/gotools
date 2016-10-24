package synchttp_test

import (
  "github.com/istreeter/gotools/synchttp"
  "net/http"
  "net/http/httptest"
  "fmt"
  "errors"
  "time"
)

func ExampleRecoveryHandler() {

  badHandler := func(w http.ResponseWriter, r *http.Request) {
    panic(errors.New("making a panic"))
  }
  goodHandler := func(w http.ResponseWriter, r *http.Request) {
    http.Error(w, "someone must have made a panic", http.StatusInternalServerError)
  }
  
  recoveryHandler := synchttp.RecoveryHandler{
    H: http.HandlerFunc(badHandler),
    RecoverH: http.HandlerFunc(goodHandler),
  }

  req := httptest.NewRequest("GET", "http://example.com/foo", nil)

  w := httptest.NewRecorder()
  recoveryHandler.ServeHTTP(w, req)
  fmt.Printf("%d - %s", w.Code, w.Body.String())

  // Output: 500 - someone must have made a panic
}

func ExampleTimedContextHandler() {

  ctxDoneHandler := func(w http.ResponseWriter, r *http.Request) {
    <-r.Context().Done()
    http.Error(w, "timed out", http.StatusServiceUnavailable)
  }

  slowHandler := func(w http.ResponseWriter, r *http.Request) {
    time.Sleep(200 * time.Millisecond)
    w.Write([]byte("Finished sleeping\n"))
  }

  timedHandler := &synchttp.TimedContextHandler{
    H: http.HandlerFunc(ctxDoneHandler),
    Dt: 100 * time.Millisecond,
  }

  syncHandlers := synchttp.Handlers{http.HandlerFunc(slowHandler), timedHandler}
  req := httptest.NewRequest("GET", "http://example.com/foo", nil)

  w1 := httptest.NewRecorder()
  syncHandlers.ServeHTTP(w1, req)
  fmt.Printf("First response: %d - %s", w1.Code, w1.Body.String())

  timedHandler.Dt = 5 * time.Second

  w2 := httptest.NewRecorder()
  syncHandlers.ServeHTTP(w2, req)
  fmt.Printf("Second response: %d - %s", w2.Code, w2.Body.String())

  // Output: First response: 503 - timed out
  // Second response: 200 - Finished sleeping
}

func Example_HandleWithMsgs() {

  mainHandler := http.HandlerFunc(
    func(w http.ResponseWriter, r *http.Request) {
      w.Write([]byte("Everything is OK\n"))
    },
  )

  recoverHandler := http.HandlerFunc(
    func(w http.ResponseWriter, r *http.Request) {
      http.Error(w, "someone must have made a panic", http.StatusInternalServerError)
    },
  )

  timeoutHandler := &synchttp.CtxDoneHandler{H: http.HandlerFunc(
    func(w http.ResponseWriter, r *http.Request) {
      http.Error(w, "timed out", http.StatusServiceUnavailable)
    },
  )}

  wrappedHandler := synchttp.HandleWithMsgs(mainHandler, recoverHandler, timeoutHandler, 200 * time.Millisecond)

  req := httptest.NewRequest("GET", "http://example.com/foo", nil)

  w := httptest.NewRecorder()
  wrappedHandler.ServeHTTP(w, req)
  fmt.Printf("%d - %s", w.Code, w.Body.String())

  // Output: 200 - Everything is OK
}
