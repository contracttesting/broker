package components_test

import (
	"testing"

	"github.com/contracttesting/cli/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestBrokerMessage(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"renders message from error envelope", `{"success":false,"message":"environment not found"}`, "environment not found"},
		{"falls back to raw body when not the envelope", `<html>502 Bad Gateway</html>`, "<html>502 Bad Gateway</html>"},
		{"falls back to raw body when message is empty", `{"success":false}`, `{"success":false}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, components.BrokerMessage([]byte(tc.body)))
		})
	}
}
