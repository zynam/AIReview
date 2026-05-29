package review

// PullRequest is the normalized domain model used by the review pipeline.
// External providers such as GitHub must map their API responses into this
// structure before analysis starts.
type PullRequest struct {
	Owner   string
	Repo    string
	Number  int
	Title   string
	Body    string
	Author  string
	BaseSHA string
	HeadSHA string
	Files   []ChangedFile
	Commits []Commit
}

type ChangedFile struct {
	Path      string
	Status    string
	Additions int
	Deletions int
	Patch     string
	Language  string
	FileKind  string
	RiskHints []RiskHint
}

type Commit struct {
	SHA     string
	Message string
	Author  string
}

type RiskHint struct {
	Category string
	Message  string
	File     string
	Line     int
}

type Finding struct {
	Severity        Severity
	Confidence      float64
	Category        Category
	File            string
	Line            int
	Title           string
	Evidence        string
	Suggestion      string
	NeedsHumanCheck bool
}

type ReviewReport struct {
	Summary        string
	Impact         []string
	Findings       []Finding
	TestAssessment string
	SkippedFiles   []string
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
