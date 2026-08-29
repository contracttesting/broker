package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/stretchr/testify/assert"
)

func TestResourcePath_IsConsumer_DecidesByFirstSegment(t *testing.T) {
	path := dsl.NewResourcePath("consumes;provides;rest;/x;get;responses;200")

	assert.True(t, path.IsConsumer())
	assert.False(t, path.IsProvider())
}

func TestResourcePath_IsProvider_DecidesByFirstSegment(t *testing.T) {
	path := dsl.NewResourcePath("provides;rest;/consumes;get;responses;200")

	assert.True(t, path.IsProvider())
	assert.False(t, path.IsConsumer())
}
