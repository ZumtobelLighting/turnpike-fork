package turnpike

import (
	"fmt"
	"sync"
	"time"
)

// Global websocket statistics. Periodically gets reported and cleared.
var wsStats websocketStats

const wsStatsRate = 60 * time.Second

var wsStatsDone chan bool

const (
	WSStatsNewPeer = iota
	WSStatsSend
	WSStatsRecv
	WSStatsClose
	WSStatsErrPeerTimeout
	WSStatsErrPeerClosed
	WSStatsItemCount
)

type websocketStats struct {
	count [WSStatsItemCount]uint64
	lock  sync.RWMutex
}

func (s *websocketStats) bumpCount(item int) {
	if (item < 0) || (item >= WSStatsItemCount) {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.count[item]++
}

func (s *websocketStats) formatAndClear() string {
	s.lock.Lock()
	defer s.lock.Unlock()
	str := fmt.Sprintf(
		"WS New: %d Send: %d Recv: %d Close: %d PeerTO: %d PeerClosed: %d",
		s.count[WSStatsNewPeer],
		s.count[WSStatsSend],
		s.count[WSStatsRecv],
		s.count[WSStatsClose],
		s.count[WSStatsErrPeerTimeout],
		s.count[WSStatsErrPeerClosed],
	)
	for _, i := range s.count {
		s.count[i] = 0
	}
	return str
}

// Periodically log the stats then clear them for next time.
// This gets run from the logger init function.
func InitWSStats() {
	ticker := time.NewTicker(wsStatsRate)
	go func() {
		for {
			select {
			case <-wsStatsDone:
				ticker.Stop()
				return
			case <-ticker.C:
				s := wsStats.formatAndClear()
				log.Info(s)
			}
		}
	}()
}

func DoneWSStats() {
	wsStatsDone <- true
}
