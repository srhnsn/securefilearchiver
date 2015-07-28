package models

import (
	"fmt"
	"time"
)

const jsonTimeFormat = "2006-01-02 15:04:05.999999999 -0700"

// JSONTime is time.Time which its own marshaller and unmarshaller.
type JSONTime struct {
	time.Time
}

// MarshalJSON stores time.Time data in the format "2006-01-02 15:04:05.999999999 -0700".
func (t JSONTime) MarshalJSON() ([]byte, error) {
	if t.Year() < 2 {
		return []byte(`null`), nil
	}

	return []byte(`"` + t.Format(jsonTimeFormat) + `"`), nil
}

// UnmarshalJSON parses time data of the format "2006-01-02 15:04:05.999999999 -0700".
func (t *JSONTime) UnmarshalJSON(bytes []byte) error {
	if string(bytes) == "null" {
		return nil
	}

	if len(bytes) < 10 {
		return fmt.Errorf("invalid JSONTime: %s", bytes)
	}

	bytes = bytes[1 : len(bytes)-1]

	mytime, err := time.Parse(jsonTimeFormat, string(bytes))

	if err != nil {
		return fmt.Errorf("error parsing JSONTime: %s", err)
	}

	t.Time = mytime
	return nil
}
