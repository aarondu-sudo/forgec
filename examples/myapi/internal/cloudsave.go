package internal

import "time"

// capi:export
func NewCloudSave(appId int64) error {
    return nil
}

// capi:export
type CloudSave struct {
    DeviceID    string
    Key         string
    Checksum    string
    VectorClock map[string]int64
    Timestamp   time.Time
}
