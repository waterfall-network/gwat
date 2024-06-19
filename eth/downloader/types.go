// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package downloader

import (
	"fmt"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// dataPack is a data message returned by a peer for some query.
type dataPack interface {
	PeerId() string
	Items() int
	Stats() string
}

// headerPack is a batch of block headers returned by a peer.
type headerPack struct {
	peerID  string
	headers []*types.Header
}

func (p *headerPack) PeerId() string { return p.peerID }
func (p *headerPack) Items() int     { return len(p.headers) }
func (p *headerPack) Stats() string  { return fmt.Sprintf("%d", len(p.headers)) }

// bodyPack is a batch of block bodies returned by a peer.
type bodyPack struct {
	peerID       string
	transactions [][]*types.Transaction
}

func (p *bodyPack) PeerId() string { return p.peerID }
func (p *bodyPack) Items() int     { return len(p.transactions) }
func (p *bodyPack) Stats() string  { return fmt.Sprintf("%d", len(p.transactions)) }

// receiptPack is a batch of receipts returned by a peer.
type receiptPack struct {
	peerID   string
	receipts [][]*types.Receipt
}

func (p *receiptPack) PeerId() string { return p.peerID }
func (p *receiptPack) Items() int     { return len(p.receipts) }
func (p *receiptPack) Stats() string  { return fmt.Sprintf("%d", len(p.receipts)) }

// statePack is a batch of states returned by a peer.
type statePack struct {
	peerID string
	states [][]byte
}

func (p *statePack) PeerId() string { return p.peerID }
func (p *statePack) Items() int     { return len(p.states) }
func (p *statePack) Stats() string  { return fmt.Sprintf("%d", len(p.states)) }

// dagPack is a dag chain returned by a peer.
type dagPack struct {
	peerID string
	dag    common.HashArray
}

func (p *dagPack) PeerId() string { return p.peerID }
func (p *dagPack) Items() int     { return len(p.dag) }
func (p *dagPack) Stats() string  { return fmt.Sprintf("%d", len(p.dag)) }

type syncPeerChanType int8

const (
	dagCh syncPeerChanType = iota
	headerCh
	bodyCh
	receiptCh
)

// dagPack is a dag chain returned by a peer.
type syncPeerChans struct {
	dagCh     chan dataPack
	headerCh  chan dataPack
	bodyCh    chan dataPack
	receiptCh chan dataPack
	procCount int
}

func initSyncPeerChans() *syncPeerChans {
	return &syncPeerChans{
		dagCh:     make(chan dataPack, 1),
		headerCh:  make(chan dataPack, 1),
		bodyCh:    make(chan dataPack, 1),
		receiptCh: make(chan dataPack, 1),
	}
}

func (p *syncPeerChans) IncrProcCount() {
	p.procCount++
}
func (p *syncPeerChans) DecrProcCount() {
	p.procCount--
}
func (p *syncPeerChans) ProcCount() int {
	return p.procCount
}

func (p *syncPeerChans) Close() {
	close(p.dagCh)
	close(p.headerCh)
	close(p.bodyCh)
	close(p.receiptCh)
	p.procCount = 0
}

func (p *syncPeerChans) GetDagChan() chan dataPack {
	if p.dagCh == nil {
		p.dagCh = make(chan dataPack)
	}
	return p.dagCh
}
func (p *syncPeerChans) GetHeaderChan() chan dataPack {
	if p.headerCh == nil {
		p.headerCh = make(chan dataPack)
	}
	return p.headerCh
}
func (p *syncPeerChans) GetBodyChan() chan dataPack {
	if p.bodyCh == nil {
		p.bodyCh = make(chan dataPack)
	}
	return p.bodyCh
}
func (p *syncPeerChans) GetReceiptChan() chan dataPack {
	if p.receiptCh == nil {
		p.receiptCh = make(chan dataPack)
	}
	return p.receiptCh
}
