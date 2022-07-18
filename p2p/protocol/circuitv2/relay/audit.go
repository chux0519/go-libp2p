package relay

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

// RelayAudit is an traffic audit tool for relayed connect.
type RelayAudit interface {
	OnRelay(src peer.ID, dest peer.ID, count int64)
}
