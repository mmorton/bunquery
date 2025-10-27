package bunquery_test

import (
	"fmt"
	"testing"

	"github.com/mmorton/bunquery"
)

func TestUnderscore(t *testing.T) {
	var tests = []struct {
		a    string
		want string
	}{
		{"One", "one"},
		{"ONE", "one"},
		{"OneTwo", "one_two"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s", tt.a)
		t.Run(testname, func(t *testing.T) {
			ans := bunquery.Underscore(tt.a)
			if ans != tt.want {
				t.Errorf("got %s, want %s", ans, tt.want)
			}
		})
	}
}
