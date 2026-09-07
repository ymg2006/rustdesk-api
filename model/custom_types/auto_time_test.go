package custom_types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAutoTimeJSONRoundTrip(t *testing.T) {
	want := time.Date(2026, 9, 7, 8, 30, 15, 0, time.UTC)
	data, err := json.Marshal(AutoTime(want))
	if err != nil {
		t.Fatal(err)
	}
	var got AutoTime
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !time.Time(got).Equal(want) {
		t.Fatalf("round trip = %s, want %s", time.Time(got), want)
	}
}
