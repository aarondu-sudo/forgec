package internal

import "fmt"

// capi:export
func Add(a int32, b int32) (int32, error) {
    return a + b, nil
}

// capi:export
func Ping(code int32) error {
    if code != 0 {
        return fmt.Errorf("ping failed: %d", code)
    }
    return nil
}

