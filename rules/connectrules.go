package rules

import (
	"fmt"
	"regexp"
	"strings"
)

type ConnectRules struct {
	whitelist    []*regexp.Regexp /// acceptable patterns
	defaultAllow RuleResponse
}

func NewConnectRules(whitelistpattern []string) *ConnectRules {
	crules := &ConnectRules{
		whitelist: make([]*regexp.Regexp, len(whitelistpattern)),
	}

	for i, rulepattern := range whitelistpattern {
		reg := regexp.MustCompile(rulepattern)
		crules.whitelist[i] = reg
	}
	crules.defaultAllow = DROPFLAT
	return crules
}

func (c *ConnectRules) Allow(data []byte) RuleResponse {
	if !strings.HasPrefix(string(data), "CONNECT") {
		return UNDEFINED
	}
	host := strings.Split(string(data), " ")[1]
	for _, rule := range c.whitelist {
		fmt.Println("Try match ", rule, " with ", host)
		if rule.MatchString(host) {
			return ALLOW
		}
	}
	return c.defaultAllow
}
