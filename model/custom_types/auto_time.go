package custom_types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// AutoTime custom time format
type AutoTime time.Time

func (mt AutoTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	t := time.Time(mt)
	if t.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t, nil
}

func (mt AutoTime) MarshalJSON() ([]byte, error) {
	//b := make([]byte, 0, len("2006-01-02 15:04:05")+2)
	b := time.Time(mt).AppendFormat([]byte{}, "\"2006-01-02 15:04:05\"")
	return b, nil
}

// UnmarshalJSON mirrors MarshalJSON so model values can safely round-trip
// through JSON-backed caches.
func (mt *AutoTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*mt = AutoTime(time.Time{})
		return nil
	}
	t, err := time.Parse(`"2006-01-02 15:04:05"`, string(data))
	if err != nil {
		return fmt.Errorf("decode AutoTime: %w", err)
	}
	*mt = AutoTime(t)
	return nil
}
