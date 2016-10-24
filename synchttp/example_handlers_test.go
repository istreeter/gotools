package synchttp_test

import (
  "github.com/istreeter/gotools/synchttp"
  "time"
  "net/http"
  "net/http/httptest"
  "fmt"
)

type slowHandler struct{
  delay time.Duration
  name  string
}

func (h *slowHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  time.Sleep(h.delay)
  w.Write([]byte(fmt.Sprintf("%s won the race", h.name)))
}

func ExampleHandlers() {
  h1 := &slowHandler{ 100*time.Millisecond, "h1"}
  h2 := &slowHandler{ 200*time.Millisecond, "h2"}
  syncHandlers := synchttp.Handlers{h1, h2}

  req := httptest.NewRequest("GET", "http://example.com/foo", nil)

  w1 := httptest.NewRecorder()
  syncHandlers.ServeHTTP(w1, req)
  fmt.Printf("First response: %d - %s\n", w1.Code, w1.Body.String())

  h1.delay = 5 * time.Second

  w2 := httptest.NewRecorder()
  syncHandlers.ServeHTTP(w2, req)
  fmt.Printf("Second response: %d - %s\n", w2.Code, w2.Body.String())

  // Output: First response: 200 - h1 won the race
  // Second response: 200 - h2 won the race
}
