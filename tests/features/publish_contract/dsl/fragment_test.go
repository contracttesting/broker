package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/stretchr/testify/assert"
)

func TestFragment_RootDevolveODocumentoQuandoEhMapa(t *testing.T) {
	fragment := dsl.Fragment{Source: "api.yaml", Document: map[string]any{"provides": map[string]any{"rest": map[string]any{}}}}

	assert.Equal(t, dsl.Document{"provides": map[string]any{"rest": map[string]any{}}}, fragment.Root())
	assert.NotNil(t, fragment.Root().Mapping("provides").Mapping("rest"))
}

func TestFragment_RootDeDocumentoNilEhNil(t *testing.T) {
	fragment := dsl.Fragment{Source: "empty.yaml"}

	assert.Nil(t, fragment.Root())
	assert.Nil(t, fragment.Root().Mapping("provides"))
}

func TestFragment_RootDeDocumentoQueNaoEhMapaEhNil(t *testing.T) {
	for name, document := range map[string]any{
		"texto":  "just text",
		"lista":  []any{"a", "b"},
		"numero": 42,
	} {
		t.Run(name, func(t *testing.T) {
			fragment := dsl.Fragment{Source: "odd.yaml", Document: document}

			assert.Nil(t, fragment.Root())
		})
	}
}

func TestSortedBySource_OrdenaPelaOrigem(t *testing.T) {
	fragments := []dsl.Fragment{
		{Source: "c.yaml", Document: "c"},
		{Source: "a.yaml", Document: "a"},
		{Source: "b.yaml", Document: "b"},
	}

	assert.Equal(t, []dsl.Fragment{
		{Source: "a.yaml", Document: "a"},
		{Source: "b.yaml", Document: "b"},
		{Source: "c.yaml", Document: "c"},
	}, dsl.SortedBySource(fragments))
}

func TestSortedBySource_NaoAlteraAEntrada(t *testing.T) {
	fragments := []dsl.Fragment{{Source: "b.yaml"}, {Source: "a.yaml"}}

	dsl.SortedBySource(fragments)

	assert.Equal(t, []dsl.Fragment{{Source: "b.yaml"}, {Source: "a.yaml"}}, fragments)
}

func TestSortedBySource_QualquerOrdemDeEntradaDaOMesmoResultado(t *testing.T) {
	forward := dsl.SortedBySource([]dsl.Fragment{{Source: "a.yaml"}, {Source: "b.yaml"}, {Source: "c.yaml"}})
	backward := dsl.SortedBySource([]dsl.Fragment{{Source: "c.yaml"}, {Source: "b.yaml"}, {Source: "a.yaml"}})

	assert.Equal(t, forward, backward)
}

func TestSortedBySource_EntradaVaziaDaListaVazia(t *testing.T) {
	assert.Empty(t, dsl.SortedBySource(nil))
}
