package review

// PullRequest is the normalized domain model used by the review pipeline.
// External providers such as GitHub must map their API responses into this
// structure before analysis starts.
type PullRequest struct {
	Owner   string        `json:"owner"`
	Repo    string        `json:"repo"`
	Number  int           `json:"number"`
	Title   string        `json:"title"`
	Body    string        `json:"body"`
	Author  string        `json:"author"`
	BaseSHA string        `json:"base_sha"`
	HeadSHA string        `json:"head_sha"`
	Files   []ChangedFile `json:"files"`
	Commits []Commit      `json:"commits"`
}

type ChangedFile struct {
	Path      string     `json:"path"`
	Status    string     `json:"status"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Patch     string     `json:"patch"`
	Language  string     `json:"language"`
	FileKind  string     `json:"file_kind"`
	RiskHints []RiskHint `json:"risk_hints"`
}

type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
}

type RiskHint struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

type Finding struct {
	ID              string   `json:"id,omitempty"`
	Severity        Severity `json:"severity"`
	Confidence      float64  `json:"confidence"`
	Category        Category `json:"category"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Title           string   `json:"title"`
	Evidence        string   `json:"evidence"`
	Suggestion      string   `json:"suggestion"`
	NeedsHumanCheck bool     `json:"needs_human_check"`
	FeedbackStatus  string   `json:"feedback_status,omitempty"`
}

type ReviewReport struct {
	Summary        string    `json:"summary"`
	Impact         []string  `json:"impact"`
	Findings       []Finding `json:"findings"`
	TestAssessment string    `json:"test_assessment"`
	SkippedFiles   []string  `json:"skipped_files"`
}

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Category string

const (
	CategoryCorrectness     Category = "correctness"
	CategorySecurity        Category = "security"
	CategoryPerformance     Category = "performance"
	CategoryConcurrency     Category = "concurrency"
	CategoryCompatibility   Category = "compatibility"
	CategoryMaintainability Category = "maintainability"
	CategoryTestRisk        Category = "test_risk"
)
