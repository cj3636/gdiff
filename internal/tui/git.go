package tui

// GitContext carries git-related state for the TUI.
type GitContext struct {
	Enabled       bool
	RepoRoot      string
	FilePath      string
	Ref1          string
	Ref2          string
	Status        []string
	Branches      []string
	CurrentBranch string
	CommitHistory []string
	Blame         map[int]string
	ShowBlame     bool
}
