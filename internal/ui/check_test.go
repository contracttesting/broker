package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"

	"github.com/contracttesting/cli/internal/ui"
)

// renderCheck renders c and returns its ANSI-stripped lines.
func renderCheck(c ui.CheckView) []string {
	var buf bytes.Buffer
	ui.Check(&buf, c)
	return strings.Split(pterm.RemoveColorFromString(buf.String()), "\n")
}

func TestCheckVerdictHeaderAndStats(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Pacticipant: "app",
		Target:      "production",
		Integrations: []ui.Integration{
			{Counterpart: "accounts", Mismatches: []ui.Mismatch{{Kind: ui.Missing}, {Kind: ui.Missing}}},
			{Counterpart: "auth", Mismatches: []ui.Mismatch{{Kind: ui.TypeMismatch}}},
			{Counterpart: "transactions", Mismatches: []ui.Mismatch{{Kind: ui.TypeMismatch}}},
			{Counterpart: "users", Mismatches: []ui.Mismatch{{Kind: ui.Missing}, {Kind: ui.TypeMismatch}, {Kind: ui.Missing}}},
			{Counterpart: "users", Mismatches: []ui.Mismatch{{Kind: ui.Missing}, {Kind: ui.TypeMismatch}, {Kind: ui.Missing}}},
		},
	})

	require.Equal(t, "✗  cannot deploy  app  → production", lines[0])
	require.Equal(t, "   4 integrations · 10 mismatches  (6 missing · 4 type · 0 optional)", lines[1])
}

func TestCheckStatsSingularIntegration(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Pacticipant: "accounts",
		Target:      "production",
		Integrations: []ui.Integration{
			{Counterpart: "transactions", Mismatches: []ui.Mismatch{{Kind: ui.Missing}, {Kind: ui.Missing}, {Kind: ui.Missing}}},
		},
	})

	require.Equal(t, "   1 integration · 3 mismatches  (3 missing · 0 type · 0 optional)", lines[1])
}

func TestCheckStatsTallyAllThreeKinds(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Pacticipant: "app",
		Target:      "production",
		Integrations: []ui.Integration{
			{Counterpart: "users", Mismatches: []ui.Mismatch{{Kind: ui.Missing}, {Kind: ui.TypeMismatch}, {Kind: ui.Optional}}},
			{Counterpart: "billing", Issues: []ui.Issue{{Kind: ui.NotDeployed}}},
		},
	})

	require.Equal(t, "   2 integrations · 3 mismatches  (1 missing · 1 type · 1 optional)", lines[1])
}

func TestCheckStatsCategoricalOnlyOmitsParenthetical(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Pacticipant: "app",
		Target:      "production",
		Integrations: []ui.Integration{
			{Counterpart: "billing", Issues: []ui.Issue{{Kind: ui.NotProvided}}},
		},
	})

	require.Equal(t, "   1 integration · 0 mismatches", lines[1])
}

func TestCheckIntegrationRows(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Pacticipant: "app",
		Target:      "production",
		Integrations: []ui.Integration{
			{Counterpart: "accounts", Method: "PUT", Path: "/accounts/{id}", Location: "request"},
			{Counterpart: "transactions", Method: "POST", Path: "/transactions", Location: "response 201"},
		},
	})

	require.Equal(t, "  accounts     PUT /accounts/{id}  request", lines[3])
	require.Equal(t, "  transactions POST /transactions  response 201", lines[5])
}

func TestCheckMismatchBlocks(t *testing.T) {
	cases := []struct {
		name        string
		integration ui.Integration
		want        []string
	}{
		{
			name: "missing in request",
			integration: ui.Integration{
				Counterpart: "accounts", Method: "PUT", Path: "/accounts/{id}", Location: "request",
				Mismatches: []ui.Mismatch{{Kind: ui.Missing, Property: "user", Omitter: "app", Requirer: "accounts"}},
			},
			want: []string{
				"    ✗  missing  user",
				"       app omits it · accounts requires it",
				"       send user in the request body",
			},
		},
		{
			name: "missing in response",
			integration: ui.Integration{
				Counterpart: "users", Method: "GET", Path: "/users/{id}", Location: "response 200",
				Mismatches: []ui.Mismatch{{Kind: ui.Missing, Property: "firstName", Omitter: "users", Requirer: "app"}},
			},
			want: []string{
				"    ✗  missing  firstName",
				"       users omits it · app requires it",
				"       return firstName in the response",
			},
		},
		{
			name: "type mismatch in request",
			integration: ui.Integration{
				Counterpart: "accounts", Method: "PUT", Path: "/accounts/{id}", Location: "request",
				Mismatches:  []ui.Mismatch{{Kind: ui.TypeMismatch, Property: "id", Wanter: "accounts", Wants: "string", Sender: "app", Sends: "integer"}},
			},
			want: []string{
				"    ≠  type     id",
				"       accounts wants string · app sends integer",
				"       send id as a string",
			},
		},
		{
			name: "type mismatch in response",
			integration: ui.Integration{
				Counterpart: "auth", Method: "POST", Path: "/auth/users", Location: "response 201",
				Mismatches:  []ui.Mismatch{{Kind: ui.TypeMismatch, Property: "id", Wanter: "app", Wants: "string", Sender: "auth", Sends: "integer"}},
			},
			want: []string{
				"    ≠  type     id",
				"       app wants string · auth sends integer",
				"       return id as a string",
			},
		},
		{
			name: "optional in request",
			integration: ui.Integration{
				Counterpart: "accounts", Method: "PUT", Path: "/accounts/{id}", Location: "request",
				Mismatches:  []ui.Mismatch{{Kind: ui.Optional, Property: "user", Requirer: "accounts", Sender: "app"}},
			},
			want: []string{
				"    ?  optional user",
				"       accounts requires it · app sends it as optional",
				"       make user required in the request body",
			},
		},
		{
			name: "optional in response",
			integration: ui.Integration{
				Counterpart: "users", Method: "GET", Path: "/users/{id}", Location: "response 200",
				Mismatches:  []ui.Mismatch{{Kind: ui.Optional, Property: "lastName", Requirer: "app", Sender: "users"}},
			},
			want: []string{
				"    ?  optional lastName",
				"       app requires it · users sends it as optional",
				"       make lastName required in the response",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := renderCheck(ui.CheckView{Pacticipant: "app", Target: "production", Integrations: []ui.Integration{tc.integration}})
			require.Equal(t, tc.want, lines[4:7])
		})
	}
}

func TestCheckIssueLines(t *testing.T) {
	cases := []struct {
		name  string
		issue ui.Issue
		want  string
	}{
		{
			name:  "not provided",
			issue: ui.Issue{Kind: ui.NotProvided},
			want:  "    ✗  payments does not provide this resource · app consumes it",
		},
		{
			name:  "not deployed in target",
			issue: ui.Issue{Kind: ui.NotDeployed, DeployedIn: "staging, prod"},
			want:  "    ✗  payments is not deployed in production · deployed in staging, prod",
		},
		{
			name:  "not deployed anywhere",
			issue: ui.Issue{Kind: ui.NotDeployed},
			want:  "    ✗  payments is not deployed in any environment",
		},
		{
			name:  "unknown reason",
			issue: ui.Issue{Kind: ui.UnknownReason, Reason: "exotic_future_reason"},
			want:  "    ✗  exotic_future_reason",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := renderCheck(ui.CheckView{
				Pacticipant: "app",
				Target:      "production",
				Integrations: []ui.Integration{
					{Counterpart: "payments", Method: "GET", Path: "/payments", Location: "request", Issues: []ui.Issue{tc.issue}},
				},
			})
			require.Equal(t, tc.want, lines[4])
		})
	}
}

func TestCheckIssuesRenderAfterMismatches(t *testing.T) {
	lines := renderCheck(ui.CheckView{
		Pacticipant: "app",
		Target:      "production",
		Integrations: []ui.Integration{
			{
				Counterpart: "users", Method: "GET", Path: "/users/{id}", Location: "response 200",
				Mismatches: []ui.Mismatch{{Kind: ui.Missing, Property: "firstName", Omitter: "users", Requirer: "app"}},
				Issues:     []ui.Issue{{Kind: ui.UnknownReason, Reason: "exotic_future_reason"}},
			},
		},
	})

	require.Equal(t, "    ✗  missing  firstName", lines[4])
	require.Equal(t, "    ✗  exotic_future_reason", lines[7])
}

func TestRule(t *testing.T) {
	var buf bytes.Buffer
	ui.Rule(&buf)

	require.Equal(t, "\n"+strings.Repeat("─", 64)+"\n\n", pterm.RemoveColorFromString(buf.String()))
}

func TestChecksSummary(t *testing.T) {
	var buf bytes.Buffer
	ui.ChecksSummary(&buf, 1)
	require.Equal(t, "✗ 1 check failed — fix the mismatches, publish new pacts, re-run.\n", pterm.RemoveColorFromString(buf.String()))

	buf.Reset()
	ui.ChecksSummary(&buf, 2)
	require.Equal(t, "✗ 2 checks failed — fix the mismatches, publish new pacts, re-run.\n", pterm.RemoveColorFromString(buf.String()))
}
