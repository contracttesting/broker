package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/stretchr/testify/assert"
)

func TestPropertyPath_Append_OnEmptyReceiver_ReturnsChunkOnly(t *testing.T) {
	pp := dsl.NewPropertyPath("")

	result := pp.Append("$")

	assert.Equal(t, "$", result.String())
}

func TestPropertyPath_AppendArray_SuffixesBrackets(t *testing.T) {
	pp := dsl.NewPropertyPath("$")

	result := pp.AppendArray()

	assert.Equal(t, "$[]", result.String())
}
