package sentrywrap

import (
    "encoding/json"
    "sync"
)

var (
    lastErrMu sync.Mutex
    lastErr   string
)

// RecoverAndReport wraps f with panic recovery and records the error as JSON.
func RecoverAndReport(f func()) {
    defer func() {
        if r := recover(); r != nil {
            SetLastError(errFromRecover(r))
        }
    }()
    f()
}

func SetLastError(err error) {
    lastErrMu.Lock()
    defer lastErrMu.Unlock()
    if err == nil {
        lastErr = ""
        return
    }
    payload := map[string]any{"error": err.Error()}
    b, _ := json.Marshal(payload)
    lastErr = string(b)
}

func LastErrorJSON() string {
    lastErrMu.Lock()
    defer lastErrMu.Unlock()
    if lastErr == "" {
        return "{}"
    }
    return lastErr
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func errFromRecover(r any) error {
    switch x := r.(type) {
    case error:
        return x
    case string:
        return simpleError(x)
    default:
        return simpleError("panic")
    }
}
