package optshttp

import (
  "net/http"
  "context"
  "time"
  //"github.com/gorilla/mux"
  "fmt"
  "strconv"
)

const (
  String = 1
  Int = 2
  RFC3339Nano = 3

  ErrorKey = "optshttp-error"
)

var typeOf map[int]string = map[int]string{
  String: "string",
  Int: "integer",
  RFC3339Nano: "date in RFC3339Nano format",
}

type optsError struct{
  optType int
  optKey string
  optVal string
}

func (e *optsError) Error() string {
  return fmt.Sprintf("Invalid %s %s: %s", typeOf[e.optType], e.optKey, e.optVal)
}

type FormHandler struct{
  H http.Handler
  Types map[string]int
}

type GorillaHandler struct{
  H http.Handler
  Types map[string]int
}

func (h *FormHandler) ServeHTTP (w http.ResponseWriter, req *http.Request) {
  ctx := req.Context()
  for key, valType := range h.Types {
    var ok bool
    ctx, ok = parseVar(ctx, key, req.FormValue(key), valType)
    if !ok {
      break
    }
  }
  req = req.WithContext(ctx)
  h.H.ServeHTTP(w, req)
}

//func (h *GorillaHandler) ServeHTTP (w http.ResponseWriter, req *http.Request) {
//  ctx := req.Context()
//  vars := mux.Vars(req)
//  for key, valType := range h.Type {
//    var ok bool
//    ctx, ok = parseVar(ctx, key, vars[key], valType)
//    if !ok {
//      break
//    }
//  }
//  req = req.WithContext(ctx)
//  h.ServeHTTP(w, req)
//}

func parseVar(ctx context.Context, key string, valStr string, valType int) (context.Context, bool) {
    ok := true
    if len(valStr) == 0 {
      return ctx, ok
    }
    switch valType {
      case String:
        ctx = context.WithValue(ctx, key, valStr)
      case Int:
         if val, err := strconv.Atoi(valStr); err != nil{
           ctx = context.WithValue(ctx, ErrorKey, &optsError{Int, key, valStr})
           ok = false
         } else {
           ctx = context.WithValue(ctx, key, val)
         }
      case RFC3339Nano:
         if val, err := time.Parse(time.RFC3339Nano, valStr); err != nil{
           ctx = context.WithValue(ctx, ErrorKey, &optsError{RFC3339Nano, key, valStr})
           ok = false
         } else {
           ctx = context.WithValue(ctx, key, val)
         }
       }
     return ctx, ok
}
