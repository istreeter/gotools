// +build go1.7
// +build !appengine

package synchttp
import (
  "context"
  "net/http"
)


func (t CtxDoneHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  if ch := req.Context().Done(); ch != nil {
    <- ch
    t.H.ServeHTTP(w, req)
  }
}

func (t *TimedContextHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  ctx, cancel := context.WithTimeout(req.Context(), t.Dt)
  defer cancel()
  req = req.WithContext(ctx)
  t.H.ServeHTTP(w, req)
}

