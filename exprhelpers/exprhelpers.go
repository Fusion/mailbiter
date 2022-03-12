package exprhelpers

import (
	"strings"
	"time"
)

/* STRINGS */

func Lower(s string) string {
	return strings.ToLower(s)
}

/* DATES */

// Converting to int means we can use standard comparison operators
func Date(s string) int64 {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t.Unix()
}

func Now(s string) int64 {
	return time.Now().Unix()
}

func Duration(s string) int64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return int64(d.Seconds())
}
