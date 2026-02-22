package analysis

// SecurityAssessment represents the AI analysis result for a package
type SecurityAssessment struct {
	IsMalicious   bool     `json:"is_malicious"`
	Confidence    float64  `json:"confidence"`
	Justification string   `json:"justification"`
	Indicators    []string `json:"indicators,omitempty"`
}
