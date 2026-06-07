package ui

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/pterm/pterm"
)

// Kind classifies a property-level contract mismatch — exactly the kinds the
// broker reports.
type Kind int

const (
	Missing Kind = iota
	TypeMismatch
	Optional
)

// IssueKind classifies a resource-level failure — a broker reason that breaks
// a whole integration rather than one property.
type IssueKind int

const (
	NotProvided IssueKind = iota
	NotDeployed
	UnknownReason
)

// Mismatch is one property-level incompatibility. It carries structured
// actors and types, not pre-rendered strings — copy is derived at render
// time so wording stays uniform everywhere.
type Mismatch struct {
	Kind     Kind
	Property string

	Omitter, Requirer string // Missing; Optional reuses Requirer

	Wanter, Wants, Sender, Sends string // TypeMismatch; Optional reuses Sender
}

// Issue is a resource-level failure: no property, one fact line, no hint.
// Its actors derive from context at render time — the integration's
// counterpart, the check's pacticipant and target.
type Issue struct {
	Kind       IssueKind
	DeployedIn string // NotDeployed: "" = not deployed anywhere
	Reason     string // UnknownReason: raw broker reason
}

// Integration is one failing interaction with a counterpart. Location is
// "request" | "response 201" | ... — its prefix decides whether hints say
// "send" (request side) or "return" (response side).
type Integration struct {
	Counterpart, Method, Path, Location string
	Mismatches                          []Mismatch
	Issues                              []Issue
}

// CheckView is the failure view of one can-i-deploy check.
type CheckView struct {
	Pacticipant, Target string
	Integrations        []Integration
}

// ruleWidth is Rule's length.
const ruleWidth = 64

// Sprint shorthands for the check palette — pterm semantic styles only.
var (
	gray    = pterm.FgDarkGray.Sprint
	red     = pterm.FgRed.Sprint
	yellow  = pterm.FgYellow.Sprint
	cyan    = pterm.FgCyan.Sprint
	blue    = pterm.FgBlue.Sprint
	bold    = pterm.Bold.Sprint
	redBold = pterm.NewStyle(pterm.FgRed, pterm.Bold).Sprint
)

// Check writes the failure view of one can-i-deploy check: the red verdict
// header, a stats line derived from the integrations (unique counterparts,
// per-kind mismatch tallies — issues never count), then each integration row
// with its contiguous mismatch blocks and issue lines.
func Check(w io.Writer, c CheckView) {
	fmt.Fprintf(w, "%s  %s  %s\n", red("✗  cannot deploy"), redBold(c.Pacticipant), red("→ "+c.Target))

	counterparts := map[string]bool{}
	missing, types, optionals := 0, 0, 0
	for _, in := range c.Integrations {
		counterparts[in.Counterpart] = true
		for _, m := range in.Mismatches {
			switch m.Kind {
			case Missing:
				missing++
			case TypeMismatch:
				types++
			case Optional:
				optionals++
			}
		}
	}
	fmt.Fprintf(w, "   %s %s %s",
		pluralize(len(counterparts), "integration", "integrations"), gray("·"),
		pluralize(missing+types+optionals, "mismatch", "mismatches"))
	if missing+types+optionals > 0 {
		fmt.Fprintf(w, "  (%d missing %s %d type %s %d optional)", missing, gray("·"), types, gray("·"), optionals)
	}
	fmt.Fprintln(w)

	longest := 0
	for _, in := range c.Integrations {
		if n := utf8.RuneCountInString(in.Counterpart); n > longest {
			longest = n
		}
	}
	for _, in := range c.Integrations {
		pad := strings.Repeat(" ", longest-utf8.RuneCountInString(in.Counterpart))
		fmt.Fprintf(w, "\n  %s%s %s %s  %s\n", bold(in.Counterpart), pad, blue(in.Method), in.Path, gray(in.Location))
		request := strings.HasPrefix(in.Location, "request")
		for _, m := range in.Mismatches {
			mismatchBlock(w, m, request)
		}
		for _, issue := range in.Issues {
			issueLine(w, issue, in.Counterpart, c.Pacticipant, c.Target)
		}
	}
}

// Rule writes the gray separator drawn between checks and before the
// summary, flanked by blank lines.
func Rule(w io.Writer) {
	fmt.Fprintf(w, "\n%s\n\n", gray(strings.Repeat("─", ruleWidth)))
}

// ChecksSummary writes the closing verdict line for failed checks.
func ChecksSummary(w io.Writer, failed int) {
	fmt.Fprintf(w, "%s — fix the mismatches, publish new pacts, re-run.\n",
		red("✗ "+pluralize(failed, "check", "checks")+" failed"))
}

// mismatchBlock writes one mismatch as its 3-line block: kind row (label
// padded to 8 so properties align across kinds), detail, hint. Copy follows
// the side — request locations say "send", response locations "return".
func mismatchBlock(w io.Writer, m Mismatch, request bool) {
	verb, into := "return", "in the response"
	if request {
		verb, into = "send", "in the request body"
	}
	switch m.Kind {
	case Missing:
		fmt.Fprintf(w, "    %s  %s %s\n", red("✗"), yellow("missing")+" ", bold(m.Property))
		fmt.Fprintf(w, "       %s omits it %s %s requires it\n", m.Omitter, gray("·"), m.Requirer)
		fmt.Fprintf(w, "       %s %s %s\n", verb, cyan(m.Property), into)
	case TypeMismatch:
		fmt.Fprintf(w, "    %s  %s %s\n", red("≠"), red("type")+"    ", bold(m.Property))
		fmt.Fprintf(w, "       %s wants %s %s %s sends %s\n", m.Wanter, cyan(m.Wants), gray("·"), m.Sender, cyan(m.Sends))
		fmt.Fprintf(w, "       %s %s as a %s\n", verb, cyan(m.Property), m.Wants)
	case Optional:
		fmt.Fprintf(w, "    %s  %s %s\n", red("?"), yellow("optional"), bold(m.Property))
		fmt.Fprintf(w, "       %s requires it %s %s sends it as optional\n", m.Requirer, gray("·"), m.Sender)
		fmt.Fprintf(w, "       make %s required %s\n", cyan(m.Property), into)
	}
}

// issueLine writes one resource-level issue as its single fact line: icon,
// message, no hint, no fix attribution.
func issueLine(w io.Writer, issue Issue, counterpart, pacticipant, target string) {
	switch issue.Kind {
	case NotProvided:
		fmt.Fprintf(w, "    %s  %s does not provide this resource %s %s consumes it\n", red("✗"), counterpart, gray("·"), pacticipant)
	case NotDeployed:
		if issue.DeployedIn == "" {
			fmt.Fprintf(w, "    %s  %s is not deployed in any environment\n", red("✗"), counterpart)
			return
		}
		fmt.Fprintf(w, "    %s  %s is not deployed in %s %s deployed in %s\n", red("✗"), counterpart, target, gray("·"), issue.DeployedIn)
	case UnknownReason:
		fmt.Fprintf(w, "    %s  %s\n", red("✗"), issue.Reason)
	}
}

// pluralize renders n with the matching word form.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
