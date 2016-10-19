package synchttp

import (
  "net/http"
  "sync"
  "errors"
)
// private types

type resSyncer struct {
  mu sync.Mutex
  claimed bool
  rw http.ResponseWriter
  done chan struct{}
}

type responseWriter struct {
  header http.Header
  syncer *resSyncer
  mine bool
  headerWritten bool
  triedClaim bool
}

var NotMineError = errors.New("Write to a responseWriter that has been claimed by another process")

// public interfaces

func (rw *responseWriter) Header() http.Header {
  return rw.header
}

func (rw *responseWriter) WriteHeader(code int) {
  rw.claimSyncer()
  if rw.headerWritten || !rw.mine {
    return
  }
  rw.writeHeader(code)
}

func (rw *responseWriter) Write(content []byte) (int, error) {
  rw.claimSyncer()
  if !rw.mine {
    return 0, NotMineError
  }
  if (!rw.headerWritten) {
    rw.writeHeader(http.StatusOK)
  }
  return rw.syncer.rw.Write(content)
}

// private

func (rw *responseWriter) claimSyncer() {
  if ! rw.triedClaim {
    rw.triedClaim = true
    rw.syncer.mu.Lock()
    defer rw.syncer.mu.Unlock()
    if !rw.syncer.claimed {
      rw.mine = !rw.syncer.claimed
      rw.syncer.claimed = true
    }
  }
}

func (rw *responseWriter) writeHeader(code int) {
  dst := rw.syncer.rw.Header()
  for k, vv := range rw.header {
    dst[k] = vv
  }
  rw.syncer.rw.WriteHeader(code)
  rw.headerWritten = true
}

func (rw *responseWriter) done() {
  if (rw.headerWritten) {
    rw.syncer.done <- struct{}{}
  }
}
