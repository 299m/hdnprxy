package rules

type Rule interface {
	Allow(data []byte) RuleResponse
}

type RuleResponse string

const (
	ALLOW       RuleResponse = "allow"
	REPSONDFAIL RuleResponse = "fail"
	DROPFLAT    RuleResponse = "drop"
	UNDEFINED   RuleResponse = "undefined" /// if the data is not covered by this rule set
)
