package violation

// The broker never renders a sentence out of a violation — the CLI owns the text.
type Violation struct {
	Code    string            `json:"code"`
	Path    string            `json:"path"`
	Source  string            `json:"source"`
	Details map[string]string `json:"details"`
}
