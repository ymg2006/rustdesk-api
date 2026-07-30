package custom_types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// TimeField Nullable time field, used for nullable timestamps in the form of *TimeField pointers in GORM.
// Internally stored as Unix timestamp in seconds, JSON serialized to ISO8601 format.
type TimeField time.Time

// Scan implements the sql.Scanner interface
func (t *TimeField) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var tt time.Time
	switch v := src.(type) {
	case time.Time:
		tt = v
	case []byte:
		parsed, err := time.Parse("2006-01-02 15:04:05", string(v))
		if err != nil {
			return err
		}
		tt = parsed
	case string:
		parsed, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return err
			}
		}
		tt = parsed
	case int64:
		tt = time.Unix(v, 0)
	default:
		return fmt.Errorf("cannot scan TimeField from %T", src)
	}
	*t = TimeField(tt)
	return nil
}

// Value implements the driver.Valuer interface
func (t TimeField) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// MarshalJSON implements json.Marshaler
func (t TimeField) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte("null"), nil
	}
	return tt.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler
func (t *TimeField) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var tt time.Time
	if err := tt.UnmarshalJSON(data); err != nil {
		return err
	}
	*t = TimeField(tt)
	return nil
}

// Time returns time.Time
func (t TimeField) Time() time.Time {
	return time.Time(t)
}
