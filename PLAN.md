# Design — Validator com walk em métodos `validate*`

Continuação do seu refactoring, mantendo as direções que você fixou: walk como métodos do `DslValidator`, contexto simples (`Source`, `Segment`, `Violations`, `Depth`, `ContractIndex`), regras com método chamado `Validate`, `[]string` de ponta a ponta. O que se repete aparece uma vez; o resto é indicado.

---

## 1. Despacho: hooks tipados com nome único `Validate` (newtypes desambiguam)

Problema real: `endpoint` e `schemaName` são ambos `string`. Se os hooks fossem `Validate(string, ctx)`, a type assertion não distinguiria — `endpoint.syntax` receberia nomes de schema. Newtypes resolvem mantendo o nome `Validate` que você escolheu:

```go
// rules.go
type Endpoint string   // nó: um endpoint cru, como o usuário escreveu
type SchemaName string // nó: um nome de schema referenciado (request body ou response)

type Rule interface {
	Code() string
}

type StatefulRule interface {
	Rule
	Fresh() StatefulRule
}
```

Não é preciso declarar uma interface por nó: **uma interface genérica cobre todos**, e o despacho vira uma função só:

```go
// dispatch.go
type nodeRule[T any] interface {
	Validate(node T, validationContext *ValidationContext)
}

func dispatch[T any](rules []Rule, node T, validationContext *ValidationContext) {
	for _, rule := range rules {
		if applicable, ok := rule.(nodeRule[T]); ok {
			applicable.Validate(node, validationContext)
		}
	}
}
```

Uso: `dispatch(ctx.Rules, Endpoint("/pets/"), ctx)`, `dispatch(ctx.Rules, rest, ctx)`, `dispatch(ctx.Rules, schema, ctx)`… O tipo do nó seleciona as regras — compile-time safe, zero switch, zero string.

Os nós e quem escuta cada um:

| Tipo do nó | Regras |
|---|---|
| `dsl.Rest` | `endpoint.duplicate` (map inteiro do arquivo) |
| `Endpoint` | `endpoint.syntax` (grafia crua) |
| `dsl.Responses` | `status.out_of_range` (map inteiro) |
| `dsl.ResourcePath` | `resource.duplicate` (folha, chave completa) |
| `SchemaName` | `schema.unresolved_name` |
| `dsl.SchemasMap` | `schema.duplicate` |
| `dsl.Schema` | `unresolved_ref`, `too_deep`, `array_without_items`, `invalid_type` |

## 2. Regras: mesma forma de hoje, só o tipo do nó muda

Stateless (padrão para as 8):

```go
type endpointSyntaxRule struct{}

func (endpointSyntaxRule) Code() string { return "endpoint.syntax" }

func (endpointSyntaxRule) Validate(endpoint Endpoint, validationContext *ValidationContext) {
	if reason := endpointViolation(dsl.NormalizeEndpoint(string(endpoint))); reason != "" {
		validationContext.AddViolation(fmt.Sprintf("invalid endpoint %q: %s (%s)", string(endpoint), reason, validationContext.Source))
	}
}
```

Stateful (padrão para `schema.duplicate` e `resource.duplicate`) — nota: `Fresh()` roda **uma vez por execução**, não por fragment, porque duplicata é cross-file:

```go
type schemaDuplicateRule struct {
	seen map[string]string
}

func (schemaDuplicateRule) Code() string { return "schema.duplicate" }

func (schemaDuplicateRule) Fresh() StatefulRule {
	return &schemaDuplicateRule{seen: map[string]string{}}
}

func (r *schemaDuplicateRule) Validate(schemas dsl.SchemasMap, validationContext *ValidationContext) {
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		if declaredIn, taken := r.seen[name]; taken {
			validationContext.AddViolation(fmt.Sprintf("duplicate schema: %s declared in %s and %s", name, declaredIn, validationContext.Source))
			continue
		}
		r.seen[name] = validationContext.Source
	}
}
```

Ajuste único nas demais: o parâmetro vira o tipo do nó e o contexto vira `*ValidationContext` (ponteiro — senão o `AddViolation` acumula numa cópia e se perde).

## 3. Contexto: compartilhado por ponteiro; posição viaja como argumento

Regra de ouro que substitui os navegadores antigos: **o contexto (ponteiro) só carrega o que é da execução inteira** — `Source`, `ContractIndex`, `Rules`, `Violations`. **O que é do ramo** (breadcrumb, resource path, depth) **viaja como argumento** dos `validate*` — argumento é por-frame, então um ramo nunca vaza posição para o vizinho, sem save/restore. `Segment`/`Depth` no contexto são setados imediatamente antes de cada `dispatch`, a partir do argumento local:

```go
// validation_context.go
type ValidationContext struct {
	Source        string
	ContractIndex ContractIndex
	Rules         []Rule        // conjunto fresco da execução — o walk despacha daqui
	Violations    []string

	// setados pelo walk imediatamente antes de cada dispatch
	Segment string       // breadcrumb ("provides GET /pets 200") ou caminho ("Pet.owner.country")
	Depth   DepthCounter // lido pela too_deep
}

func (vc *ValidationContext) AddViolation(violation string) {
	vc.Violations = append(vc.Violations, violation)
}
```

`DepthCounter` volta a semântica de **valor** (cópia por nível = budget por profundidade; o ponteiro atual soma globalmente e depois de 10 nós tudo vira too_deep):

```go
type DepthCounter struct{ levels int }

func (dc DepthCounter) Deeper() DepthCounter { return DepthCounter{levels: dc.levels + 1} }
func (dc DepthCounter) Exceeded() bool       { return dc.levels >= MAX_DEPTH }
```

## 4. `Validate`: índice e regras uma vez, contexto por fragment, coleta no fim

```go
func (v *DslValidator) Validate(fragments []dsl.Fragment) []string {
	contractIndex := NewContractIndex(fragments) // uma vez — namespace é global, não por fragment
	rules := v.freshRules()                      // uma vez — stateful atravessa fragments
	violations := []string{}

	for _, fragment := range sortedBySource(fragments) {
		validationContext := NewValidationContext(fragment.Source, contractIndex, rules)
		v.validateContract(fragment, validationContext)
		violations = append(violations, validationContext.Violations...)
	}

	return violations
}

func (v *DslValidator) freshRules() []Rule {
	rules := make([]Rule, 0, len(v.ruleEntries))
	for _, entry := range v.ruleEntries {
		rule := entry.rule
		if stateful, ok := rule.(StatefulRule); ok {
			rule = stateful.Fresh()
		}
		rules = append(rules, rule)
	}
	return rules
}
```

Isso mata a race do singleton sem criar tipo novo: o `DslValidator` só carrega o catálogo imutável (o campo `violations` dele sai — morto), e tudo que muta vive no contexto da execução. Os `validate*` continuam métodos do `DslValidator`, como você estruturou.

## 5. Walk: provides e consumes convergem em `validateRest`

```go
func (v *DslValidator) validateContract(fragment dsl.Fragment, ctx *ValidationContext) {
	root := dsl.NewResourcePath("")

	v.validateRest(fragment.Contract.Provides.Rest, "provides", root.Append("provides"), ctx)

	for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		v.validateRest(
			fragment.Contract.ConsumesServices[service].Rest,
			joinWhere("consumes", service),
			root.Append("consumes", service),
			ctx,
		)
	}

	v.validateSchemas(fragment.Contract.Schemas, ctx)
}
```

`validateProvides`/`validateConsumes` deixam de existir como corpos distintos — os dois lados são o mesmo passeio por `Rest`; o que difere (breadcrumb e chave de resource com o serviço) entra por argumento. `Provides.Message`/`Consumes.Message` seguem não visitados.

`validateRest` — o nó com os dois gates estruturais e a ordem determinística (`slices.Sorted` em **todo** range de map do walk; não repito nos demais):

```go
func (v *DslValidator) validateRest(rest dsl.Rest, where string, resourcePath dsl.ResourcePath, ctx *ValidationContext) {
	ctx.Segment = where
	dispatch(ctx.Rules, rest, ctx) // endpoint.duplicate vê o map inteiro, pré-normalização

	visited := map[string]bool{}

	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		ctx.Segment = where
		dispatch(ctx.Rules, Endpoint(endpoint), ctx) // endpoint.syntax, grafia crua

		normalized := dsl.NormalizeEndpoint(endpoint)

		if endpointViolation(normalized) != "" {
			continue // sintaxe inválida: métodos não visitados
		}

		if visited[normalized] {
			continue // segunda grafia do duplicado same-file: primeira venceu, não desce
		}
		visited[normalized] = true

		v.validateMethod(rest[endpoint], joinWhere(where, normalized), resourcePath.Append("rest", normalized), ctx)
	}
}
```

`validateMethod` mantém seu despacho por verbo, agora com posição:

```go
func (v *DslValidator) validateMethod(methods dsl.HttpMethods, where string, resourcePath dsl.ResourcePath, ctx *ValidationContext) {
	v.validateGet(methods.Get, joinWhere(where, "GET"), resourcePath.Append("get"), ctx)
	v.validatePost(methods.Post, joinWhere(where, "POST"), resourcePath.Append("post"), ctx)
	// PUT e DELETE repetem o padrão
}
```

Os verbos convergem num helper de responses; POST/PUT somam o ramo de request body (mostro o POST; GET/DELETE são só o `validateResponses`):

```go
func (v *DslValidator) validatePost(post dsl.PostMethod, where string, resourcePath dsl.ResourcePath, ctx *ValidationContext) {
	if post.HasRequestBody() {
		requestWhere := joinWhere(where, "request")
		requestPath := resourcePath.Append("request")

		ctx.Segment = requestWhere
		dispatch(ctx.Rules, requestPath, ctx)                     // resource.duplicate (folha)
		dispatch(ctx.Rules, SchemaName(post.RequestBody), ctx)    // schema.unresolved_name
	}

	v.validateResponses(post.Responses, where, resourcePath.Append("responses"), ctx)
}

func (v *DslValidator) validateResponses(responses dsl.Responses, where string, resourcePath dsl.ResourcePath, ctx *ValidationContext) {
	ctx.Segment = where
	dispatch(ctx.Rules, responses, ctx) // status.out_of_range (map inteiro; não suprime o name check)

	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		status := strconv.Itoa(statusCode)
		ctx.Segment = joinWhere(where, status)

		dispatch(ctx.Rules, resourcePath.Append(status), ctx)             // resource.duplicate
		dispatch(ctx.Rules, SchemaName(responses[statusCode]), ctx)       // schema.unresolved_name
	}
}
```

## 6. Schemas: descida recursiva com depth por valor

```go
func (v *DslValidator) validateSchemas(schemas dsl.SchemasMap, ctx *ValidationContext) {
	ctx.Segment = ""
	dispatch(ctx.Rules, schemas, ctx) // schema.duplicate

	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		v.validateSchema(schemas[name], name, DepthCounter{}, ctx) // cada schema declarado é raiz do próprio budget
	}
}

func (v *DslValidator) validateSchema(schema dsl.Schema, path string, depth DepthCounter, ctx *ValidationContext) {
	ctx.Segment = path
	ctx.Depth = depth
	dispatch(ctx.Rules, schema, ctx) // unresolved_ref, too_deep, array_without_items, invalid_type

	if depth.Exceeded() {
		return // proteção contra ciclo; a MENSAGEM veio da regra acima
	}

	switch {
	case schema.IsRef():
		target, declared := ctx.ContractIndex.Schema(schema.Ref)
		if !declared {
			return // nada a descer; mensagem da unresolved_ref
		}
		v.validateSchema(target, path, depth.Deeper(), ctx)

	case schema.IsArray():
		if schema.Items == nil {
			return // mensagem da array_without_items
		}
		v.validateSchema(*schema.Items, path+"[]", depth.Deeper(), ctx)

	case schema.IsObject():
		for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
			v.validateSchema(schema.Properties[name], path+"."+name, depth.Deeper(), ctx)
		}
	}
}
```

Gates são do walker (decisão estrutural); mensagens são das regras — mesma divisão do design anterior, agora explícita nos seus métodos.

## 7. Decisões que ficam registradas

- **Wire `errors` → `violations`**: mantida (sua mudança no `publish_contract_wire.go`). Consequência: o CLI conforma depois (json tag no componente de resposta + asserts), e os testes de integração do broker que pinam `errors` mudam junto. Follow-up separado.
- **Composição (`Append`/`Without`) removida**: o campo `invariant` do `ruleEntry` fica sem leitor — ou sai também, ou fica documentado como reserva da versão paga. Sugestão: sair agora (código morto), voltar quando a versão paga existir.
- **Limpeza final**: `fmt.Println` de debug, campo `violations` do `DslValidator`, `NewDepthCounter`/`Deeper` mutáveis.
- **Testes**: `tests/validator/` adapta ao design (os de `Append`/`Without` caem); os pinos de mensagem continuam válidos argumento a argumento; manter os testes de determinismo (`_SameInputTwice_`) e de frescor stateful (dois `Validate` seguidos não vazam estado).
- **Verificação**: `go test ./broker/... ./cli/...` da raiz (Docker) — o do CLI vai quebrar no wire até o follow-up do CLI; decidir se os dois mudam na mesma leva.

## Ordem de execução sugerida

1. `rules.go` + `dispatch.go` (newtypes, interface genérica, `dispatch`).
2. Adaptar as 10 regras (assinatura `Validate(<nó>, *ValidationContext)`; `Fresh() StatefulRule`).
3. `ValidationContext` final + `DepthCounter` por valor.
4. `Validate`/`freshRules` (índice e regras fora do loop, coleta de violations).
5. Walk: `validateContract` → `validateRest` → `validateMethod`/verbos → `validateResponses` → `validateSchemas`/`validateSchema`.
6. Limpeza + testes + suíte completa.
