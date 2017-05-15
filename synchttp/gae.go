// +build appengine

package synchttp
import (
  "golang.org/x/net/context"
  "google.golang.org/appengine"
  "net/http"
)

func (t CtxDoneHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  if ch := appengine.NewContext(req).Done(); ch != nil {
    <- ch
    t.H.ServeHTTP(w, req)
  }
}

func (t *TimedContextHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  ctx, cancel := context.WithTimeout(appengine.NewContext(req), t.Dt)
  defer cancel()
  appengine.WithContext(ctx, req)
  t.H.ServeHTTP(w, req)
}
