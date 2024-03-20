package relay

import (
	"encoding/hex"
	"log"
	"time"
)

type DebugLog struct {
	enabled bool
	connid  string
}

func (p *DebugLog) EnableDebugLogs(enabled bool, connid string) {
	p.enabled = enabled
	p.connid = connid
}

func (p *DebugLog) LogDebug(message string, preffix string) {
	if p.enabled {
		log.Println(p.connid, ":>", time.Now(), ">", preffix, ">", message)
	}
}

func (p *DebugLog) LogData(message string, preffix string) {
	if p.enabled {
		log.Println(p.connid, ":>", time.Now(), ">", preffix, ">", hex.EncodeToString([]byte(message)))
	}
}
