package domain

import "time"

// Content es lo que dice un informe: lo que había en Kaoru sobre el PKG al emitirlo.
// Los demás features dan cada parte; este dominio solo la escribe.
type Content struct {
	Code     string
	Version  int
	Previous string
	IssuedBy string
	IssuedAt time.Time

	Package     PackageInfo
	Trace       []TraceRow
	Runs        []Run
	Regression  *Regression
	Defects     []Defect
	Checklists  []Checklist
	Diagnostics []Diagnostic
	SmokeTests  []SmokeTest
}

// PackageInfo es el encabezado del PKG con su contenido.
type PackageInfo struct {
	Id           string
	Code         string
	Title        string
	SystemName   string
	StateName    string
	Route        string
	Builds       []Build
	Requirements []Requirement
}

// Build es un BUILD del PKG.
type Build struct {
	Code     string
	Version  string
	Checksum string
}

// Requirement es un REQ del PKG.
type Requirement struct {
	Code      string
	Title     string
	StateName string
}

// TraceRow es una fila de la trazabilidad: un REQ con uno de sus TC y el último RUN de ese TC sobre un BUILD del PKG.
type TraceRow struct {
	RequirementCode string
	ChangeCode      string
	Route           string
	RiskCode        string
	RiskLevel       string
	FunctionalCases []string
	TestCaseCode    string
	Critical        bool
	RunCode         string
	RunResult       string
	Defects         []string
}

// Run es un RUN sobre un BUILD del PKG.
type Run struct {
	Code         string
	TestCaseCode string
	Result       string
	BuildCode    string
	BuildVersion string
	Environment  string
	ExecutedAt   time.Time
	ExecutedBy   string
}

// Regression es la selección de regresión del PKG; nil si no se propuso.
type Regression struct {
	ConfirmedBy string
	// ConfirmedAt es nil mientras la selección no esté confirmada.
	ConfirmedAt *time.Time
	Proposed    int
	Included    int
	Added       int
	Removed     int
	Passed      int
	Failed      int
	Pending     int
	Items       []RegressionItem
}

// RegressionItem es un TC propuesto o agregado, con lo que decidió QA.
type RegressionItem struct {
	TestCaseCode string
	Origin       string
	Proposed     bool
	Included     bool
	LastResult   string
	Reason       string
	DecidedBy    string
}

// Defect es un DEF no cerrado de un REQ del PKG. DeferredBy vacío si no está diferido.
type Defect struct {
	Code            string
	Title           string
	Severity        string
	StateName       string
	RequirementCode string
	DeferredBy      string
	DeferredAt      *time.Time
	DeferredReason  string
}

// Checklist es un checklist de producción del PKG con sus firmas.
type Checklist struct {
	Code            string
	Environment     string
	TemplateVersion int
	StartedBy       string
	CreatedAt       time.Time
	Steps           []ChecklistStep
}

// ChecklistStep es un paso; SignedBy vacío si no está firmado.
type ChecklistStep struct {
	Position int
	Title    string
	SignedBy string
	SignedAt time.Time
	Comment  string
}

// Diagnostic es un DIAG del PKG; ReviewResult vacío si QA no lo revisó.
type Diagnostic struct {
	Code         string
	Deployment   string
	Environment  string
	SubmittedBy  string
	CreatedAt    time.Time
	Notes        string
	ReviewResult string
	ReviewedBy   string
	ReviewedAt   time.Time
	ReviewNotes  string
}

// SmokeTest es un SMOKE del PKG.
type SmokeTest struct {
	Code        string
	Deployment  string
	Environment string
	Result      string
	Notes       string
	ExecutedBy  string
	CreatedAt   time.Time
}
