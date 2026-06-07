// Package main is a throwaway demo of the can-i-deploy failure layout: it
// composes the check components from internal/ui with a hardcoded two-check
// fixture so the rendering can be eyeballed. Not wired into the CLI.
package main

import (
	"fmt"
	"os"

	"github.com/contracttesting/cli/internal/ui"
	"github.com/pterm/pterm"
)

func main() {
	w := os.Stdout

	fmt.Fprintln(w, pterm.FgDarkGray.Sprint("$ can-i-deploy --pacticipant app --to production"))
	fmt.Fprintln(w)
	ui.Check(w, appCheck)
	ui.Rule(w)
	ui.Check(w, accountsCheck)
	ui.Rule(w)
	ui.ChecksSummary(w, 2)
}

// appCheck — 4 integrations (users twice) · 10 mismatches (6 missing · 4 type).
var appCheck = ui.CheckView{
	Pacticipant: "app",
	Target:      "production",
	Integrations: []ui.Integration{
		{
			Counterpart: "accounts", Method: "PUT", Path: "/accounts/{id}", Location: "request",
			Mismatches: []ui.Mismatch{
				{Kind: ui.Missing, Property: "$.user", Omitter: "app", Requirer: "accounts"},
				{Kind: ui.Missing, Property: "$.user.id", Omitter: "app", Requirer: "accounts"},
			},
		},
		{
			Counterpart: "auth", Method: "POST", Path: "/auth/users", Location: "response 201",
			Mismatches: []ui.Mismatch{
				{Kind: ui.TypeMismatch, Property: "$.id", Wanter: "app", Wants: "string", Sender: "auth", Sends: "integer"},
			},
		},
		{
			Counterpart: "transactions", Method: "POST", Path: "/transactions", Location: "response 201",
			Mismatches: []ui.Mismatch{
				{Kind: ui.TypeMismatch, Property: "$.id", Wanter: "app", Wants: "string", Sender: "transactions", Sends: "integer"},
			},
		},
		{
			Counterpart: "users", Method: "GET", Path: "/users/{id}", Location: "response 200",
			Mismatches: []ui.Mismatch{
				{Kind: ui.Missing, Property: "$.firstName", Omitter: "users", Requirer: "app"},
				{Kind: ui.TypeMismatch, Property: "$.id", Wanter: "app", Wants: "string", Sender: "users", Sends: "integer"},
				{Kind: ui.Missing, Property: "$.lastName", Omitter: "users", Requirer: "app"},
			},
		},
		{
			Counterpart: "users", Method: "POST", Path: "/users", Location: "response 201",
			Mismatches: []ui.Mismatch{
				{Kind: ui.Missing, Property: "$.firstName", Omitter: "users", Requirer: "app"},
				{Kind: ui.TypeMismatch, Property: "$.id", Wanter: "app", Wants: "string", Sender: "users", Sends: "integer"},
				{Kind: ui.Missing, Property: "$.lastName", Omitter: "users", Requirer: "app"},
			},
		},
	},
}

// accountsCheck — 1 integration · 3 mismatches (3 missing · 0 type).
var accountsCheck = ui.CheckView{
	Pacticipant: "accounts",
	Target:      "production",
	Integrations: []ui.Integration{
		{
			Counterpart: "transactions", Method: "GET", Path: "/accounts/{id}", Location: "response 200",
			Mismatches: []ui.Mismatch{
				{Kind: ui.Missing, Property: "$.type", Omitter: "accounts", Requirer: "transactions"},
				{Kind: ui.Missing, Property: "$.user", Omitter: "accounts", Requirer: "transactions"},
				{Kind: ui.Missing, Property: "$.user.id", Omitter: "accounts", Requirer: "transactions"},
			},
		},
	},
}
