package synchttp

import (
  "net/http"
  "time"
  "sync"
  "context"
)

type Handlers []http.Handler

func (handlers Handlers) ServeHTTP (w http.ResponseWriter, req *http.Request) {
  var syncer *resSyncer
  if rw, ok := w.(*responseWriter); ok {
    syncer = rw.syncer
  } else {
    syncer = &resSyncer{rw: w, done: make(chan struct{})}
    defer close(syncer.done)
  }

  var wg sync.WaitGroup
  wg.Add(len(handlers))

  for _, h := range handlers {
    go func(h http.Handler) {
      rw := &responseWriter{
        syncer:syncer,
        header: make(http.Header),
      }
      defer wg.Done()
      defer rw.done()
      h.ServeHTTP(rw, req)
    }(h)
  }

  wgDone := make(chan struct{})
  go func() {
    defer close(wgDone)
    wg.Wait()
    wgDone <- struct{}{}
  }()

  select {
    case <- wgDone:
    case <- syncer.done:
      go func() {<- wgDone}()
  }
}


type TimeoutHandler string

func (t TimeoutHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  if ch := req.Context().Done(); ch != nil {
    <- ch
    http.Error(w, string(t), http.StatusServiceUnavailable)
  }
}

type TimedContextHandler struct {
  H http.Handler
  Dt time.Duration
}

func (t *TimedContextHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  ctx, cancel := context.WithTimeout(req.Context(), t.Dt)
  defer cancel()
  req = req.WithContext(ctx)
  t.H.ServeHTTP(w, req)
}

type RecoveryHandler struct {
  H http.Handler
  Msg string
}
func (t *RecoveryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  defer func () {
    if r := recover(); r != nil {
      http.Error(w, t.Msg, http.StatusInternalServerError)
    }
  }()
  t.H.ServeHTTP(w, req)
}

func HandleWithMsgs(h http.Handler, errMsg string, timeoutMsg TimeoutHandler, dt time.Duration) http.Handler {
  return &TimedContextHandler{
    H: Handlers{&RecoveryHandler{H: h, Msg: errMsg}, timeoutMsg},
    Dt: dt,
  }
}
