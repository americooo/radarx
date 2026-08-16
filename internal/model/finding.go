package model

import "time"

// Severity ranks how urgently a Finding deserves human attention.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Finding is a signal produced by a detection module — never an exploit
// result, always "this looks worth a human's attention."
type Finding struct {
	Module      string    `json:"module"`
	Severity    Severity  `json:"severity"`
	Asset       Asset     `json:"asset"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Evidence    string    `json:"evidence,omitempty"`
	TakenAt     time.Time `json:"taken_at"`
}
