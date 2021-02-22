package packproto

import (
	"errors"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

// TxRef is a transaction to update a repo reference
type txRef struct {
	oldHash plumbing.Hash
	newHash plumbing.Hash
	ref     string
}

// Parses old hash, new hash, and ref from a line in that order
func newTxRefFromBytes(line []byte) (rt txRef, err error) {
	// ! https://github.com/git/git/blob/v2.30.1/Documentation/technical/http-protocol.txt
	// ! We expect the format to be something like
	// ! "....0a53e9ddeaddad63ad106860237bbf53411d11a7 441b40d833fdfa93eb2908e52742248faf0ee993 refs/heads/master\0 report-status"
	// ! but due to https://github.com/go-git/go-git/blob/v5.2.0/plumbing/protocol/packp/updreq_encode.go#L57
	// ! the encoding is broken and there is no space between the "\x00" byte and the list of capabilities.
	// ! See also https://github.com/go-git/go-git/blob/v5.2.0/plumbing/protocol/packp/updreq_encode_test.go#L112
	tokens := strings.Split(string(line), "\x00")

	arr := strings.Split(strings.TrimSpace(tokens[0]), " ")
	if len(arr) != 3 {
		err = errors.New("invalid line: " + string(line))
		return
	}

	rt = txRef{
		oldHash: plumbing.NewHash(arr[0]),
		newHash: plumbing.NewHash(arr[1]),
		ref:     arr[2],
	}

	return
}

// Old returns the old Reference object
func (tx *txRef) old() *plumbing.Reference {
	return plumbing.NewHashReference(plumbing.ReferenceName(tx.ref), tx.oldHash)
}

// New returns the new Reference object
func (tx *txRef) new() *plumbing.Reference {
	return plumbing.NewHashReference(plumbing.ReferenceName(tx.ref), tx.newHash)
}
