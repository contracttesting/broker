package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/contracttesting/cli/internal/ui"
)

// renderCheck renders v and returns its lines.
func renderCheck(v ui.CheckView) []string {
	var buf bytes.Buffer
	ui.Check(&buf, v)
	return strings.Split(buf.String(), "\n")
}

func TestCheckHeadline(t *testing.T) {
	lines := renderCheck(ui.CheckView{Participant: "app", Environment: "production"})
	require.Equal(t, `"app" cannot be deployed in "production" environment`, lines[0])
}

func TestCheckGroupedLayout(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Participant: "app",
		Environment: "production",
		Counterparts: []ui.Counterpart{
			{
				Name: "users",
				Resources: []ui.Resource{
					{
						Method: "GET", Path: "/users/{id}", Location: "200 response",
						Groups: []ui.BreakGroup{
							{Label: "absent fields", Breaks: []string{
								"$.firstName: absent in users - required in app",
								"$.lastName: absent in users - required in app",
							}},
							{Label: "type mismatches", Breaks: []string{
								"$.id: integer in users - string in app",
							}},
						},
					},
				},
			},
		},
	})

	require.Equal(t, []string{
		`"app" cannot be deployed in "production" environment`,
		"",
		"app is not compatible with users",
		"  GET /users/{id} (200 response)",
		"    * absent fields",
		"      - $.firstName: absent in users - required in app",
		"      - $.lastName: absent in users - required in app",
		"    * type mismatches",
		"      - $.id: integer in users - string in app",
		"",
	}, lines)
}

func TestCheckUngroupedRawReason(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Participant: "front",
		Environment: "production",
		Counterparts: []ui.Counterpart{
			{Name: "accounts", Resources: []ui.Resource{
				{
					Method: "POST", Path: "/accounts", Location: "request",
					Groups: []ui.BreakGroup{
						{Label: "", Breaks: []string{"provider_resource_not_found"}},
					},
				},
			}},
		},
	})

	require.Equal(t, []string{
		`"front" cannot be deployed in "production" environment`,
		"",
		"front is not compatible with accounts",
		"  POST /accounts (request)",
		"    - provider_resource_not_found",
		"",
	}, lines)
}
