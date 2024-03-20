package relay

import (
	"log"
	"time"
)

type DebugLog struct {
	enabled bool
}

func (p *DebugLog) EnableDebugLogs(enabled bool) {
	p.enabled = enabled
}

func (p *DebugLog) LogDebug(message string, preffix string) {
	if p.enabled {
		log.Println(time.Now(), ">", preffix, ">", message)
	}
}
