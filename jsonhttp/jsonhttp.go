package jsonhttp

import (
  "encoding/json"
  "net/http"
  "github.com/istreeter/gotools/synchttp"
  "time"
)

var DefaultCtxDoneHandler = &synchttp.CtxDoneHandler{H: NewErrorHandler("Server Timeout", http.StatusServiceUnavailable)}
var DefaultErrorHandler = NewErrorHandler("Server Error", http.StatusInternalServerError)

type errorHandler struct{
  content []byte
  code int
}
func (h *errorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(h.code)
    w.Write(h.content)
}

func NewErrorHandler(message string, code int) http.Handler {
  jsonContent, err := json.Marshal(&errorResponse{Error: true, Message: message})
  if err != nil {
    panic(err)
  }
  jsonContent = append(jsonContent, "\n"...)
  return &errorHandler{content: jsonContent, code: code}
}

type errorResponse struct{
  Error bool `json:"error"`
  Message string `json:"message"`
}

func Error(w http.ResponseWriter, message string, code int) {
  res := &errorResponse{Error: true, Message: message}
  write(w, res, code)
}

func write(w http.ResponseWriter, content interface{}, code int) {
  w.Header().Set("Content-Type", "application/json; charset=UTF-8")
  w.WriteHeader(code)
  if err := json.NewEncoder(w).Encode(content); err != nil {
    panic(err);
  }
}

func OK(w http.ResponseWriter, content interface{}) {
  write(w, content, http.StatusOK)
}

func HandleWithMsgs(h http.Handler, dt time.Duration) http.Handler {
  return synchttp.HandleWithMsgs(h, DefaultErrorHandler, DefaultCtxDoneHandler, dt)
}
