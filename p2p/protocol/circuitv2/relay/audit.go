package relay

import (
	"errors"
	"io"

	"github.com/libp2p/go-libp2p/core/peer"
)

// RelayAudit is an traffic audit tool for relayed connect.
type RelayAudit interface {
	OnRelay(src peer.ID, dest peer.ID, count int64)
}

type RelayAuditPipe struct {
	S peer.ID
	D peer.ID
	A RelayAudit
}

func (p *RelayAuditPipe) CopyBuffer(dst io.Writer, src io.Reader, buf []byte, ch chan bool) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in CopyBuffer")
	}
	return copyBuffer(p, dst, src, buf, ch)
}

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func copyBuffer(p *RelayAuditPipe, dst io.Writer, src io.Reader, buf []byte, ch chan bool) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
outter:
	for {
		select {
		case <-ch:
			err = errors.New("stopped by user")
			break outter
		default:
			nr, er := src.Read(buf)
			if nr > 0 {
				nw, ew := dst.Write(buf[0:nr])
				if nw < 0 || nr < nw {
					nw = 0
					if ew == nil {
						ew = errors.New("invalid write result")
					}
				}
				written += int64(nw)
				if p.A != nil {
					// nitice this `OnRelay` method may block the loop, so if it will take long time, throw the job into a goroutine
					p.A.OnRelay(p.S, p.D, int64(nw))
				}

				if ew != nil {
					err = ew
					break outter
				}
				if nr != nw {
					err = io.ErrShortWrite
					break outter
				}
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break outter
			}
		}
	}
	return written, err
}
