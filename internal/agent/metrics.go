package agent

type Metrics struct {
	Model              string
	FileCount          int
	RuleFindingCount   int
	ContextChunks      int
	KeptChunks         int
	SkippedFiles       int
	PromptTokensApprox int
	DurationMillis     int64
}
