package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/glycerine/rbtree"
)

// prevent replay attacks by detecting jobs with
// timestamps that are too old or Nonce's that are not unique

type Nonce int64

type NonceRegistry struct {
	TSrc            TimeSource
	TimeTree        *rbtree.Tree
	NonceHash       map[Nonce]Ntm
	InvalidAfterDur Ntm // in nanoseconds
}

type RealTimeSource struct{}

func NewRealTimeSource() *RealTimeSource {
	return &RealTimeSource{}
}

func (rts *RealTimeSource) Now() Ntm {
	return Ntm(time.Now().UnixNano())
}

type TimeSource interface {
	Now() Ntm
}

func NewNonceRegistry(tsrc TimeSource) *NonceRegistry {

	return &NonceRegistry{
		TSrc: tsrc,
		TimeTree: rbtree.NewTree(func(a, b rbtree.Item) int {
			return int(b.(*Job).Sendtime - a.(*Job).Sendtime)
		}),
		NonceHash:       make(map[Nonce]Ntm),
		InvalidAfterDur: Ntm(10e9), // 10 seconds (in nanoseconds)
	}

}

func (n *NonceRegistry) IsBadStamp(j *Job) bool {
	n.GCReg()
	if n.tooOld(j) {
		return true
	}
	if _, ok := n.NonceHash[Nonce(j.Sendernonce)]; ok {
		//fmt.Printf("\n detected replay of duplicate nonce: %x from job: %s\n", j.Sendernonce, j)
		return true
	}
	return false
}

func (n *NonceRegistry) AddedOkay(j *Job) bool {
	//fmt.Printf("debug: In AddedOkay(j.Sendernonce=%x): NonceRegistry = %s\n", j.Sendernonce, n)

	if j == nil {
		panic("j cannot be nil")
	}
	n.GCReg()
	if n.IsBadStamp(j) {
		return false
	}
	n.TimeTree.Insert(j)
	n.NonceHash[Nonce(j.Sendernonce)] = Ntm(j.Sendtime)
	return true
}

func (n *NonceRegistry) String() string {
	s := n.TimeTreeAsString()
	s += "\nNonceHash:\n"
	for k, v := range n.NonceHash {
		s += fmt.Sprintf("   Nonce: %x, Ntm: %s (%d)\n", k, time.Unix(int64(v/1e9), int64(v%1e9)), v)
	}
	return s
}

func (n *NonceRegistry) TimeTreeAsString() string {
	r := "\n"
	for it := n.TimeTree.Min(); !it.Limit(); it = it.Next() {
		j := it.Item().(*Job)
		r += fmt.Sprintf(" TimeTree: Ntm:%s (%d)  Job.Nonce: %x\n", time.Unix(j.Sendtime/1e9, j.Sendtime%1e9), j.Sendtime, j.Sendernonce)
	}
	return r
}

func (n *NonceRegistry) tooOld(j *Job) bool {
	now := n.TSrc.Now()

	if j.Sendtime == 0 {
		return true
	}
	if now-Ntm(j.Sendtime) >= n.InvalidAfterDur {
		return true
	}
	return false
}

// GCReg: garbage collect old entries
func (n *NonceRegistry) GCReg() {

	it := n.TimeTree.Min()
	for !it.Limit() {
		j := it.Item().(*Job)
		if n.tooOld(j) {
			//fmt.Printf("CGReg detected stale job in registry, deleting: %s\n", j)
			nonce := Nonce(j.Sendernonce)

			// advance before deleting...
			it = it.Next()
			n.TimeTree.DeleteWithKey(j)

			// bound the size of our NonceHash here.
			// We are limited to just the young jobs' Sendernonce.
			delete(n.NonceHash, nonce)
		} else {
			// no need to go further into younger jobs. Avoid full linear scan.
			break
		}
	}
}

// called from NewJob, can't call in SignJob() because that
// is used for verification too.
func StampJob(j *Job) {
	j.Sendtime = int64(time.Now().UnixNano())
	j.Sendernonce = int64(rand.Int())
}
