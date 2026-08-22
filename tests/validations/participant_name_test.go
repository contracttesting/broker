package validations_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/validations"
	"github.com/stretchr/testify/assert"
)

func TestParticipantName_AcceptsSnakeCase(t *testing.T) {
	for _, name := range []string{"orders", "orders_api", "orders_api_v2", "v2"} {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, validations.ParticipantName(name))
		})
	}
}

func TestParticipantName_RejectsAnythingButSnakeCase(t *testing.T) {
	for _, name := range []string{"", "Orders", "orders-api", "orders api", "_orders", "orders_", "orders__api"} {
		t.Run(name, func(t *testing.T) {
			assert.EqualError(t, validations.ParticipantName(name), "must be snake_case")
		})
	}
}
