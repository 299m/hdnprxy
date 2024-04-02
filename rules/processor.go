package rules

import "fmt"

type ConnectConfig struct {
	Whitelist []string
}

func (c *ConnectConfig) Expand() {
}

type Processor struct {
	rules []Rule
}

func NewProcessor(conncfg *ConnectConfig) *Processor {
	proc := &Processor{
		rules: []Rule{NewConnectRules(conncfg.Whitelist)},
	}
	for _, rule := range proc.rules {
		fmt.Println("CONNECTION rule added ", rule)
	}
	return proc
}

func (p *Processor) Allow(data []byte) RuleResponse {
	for _, rule := range p.rules {
		resp := rule.Allow(data)
		if resp != UNDEFINED {
			return resp
		}
	}
	return ALLOW
}
