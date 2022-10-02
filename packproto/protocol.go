package packproto

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"

	"github.com/rwl4/go-git-server/packfile"
)

// Protocol implements the git pack protocol
type Protocol struct {
	w io.Writer
	r io.Reader
}

// NewProtocol instantiates a new protocol with the given reader and writer
func NewProtocol(w io.Writer, r io.Reader) *Protocol {
	return &Protocol{w: w, r: r}
}

// // ListReferences writes the references in the pack protocol given the repository
// // and service type
// func (proto *Protocol) ListReferences(service GitServiceType, refs *repository.RepositoryReferences) {

// 	// Start sending info
// 	enc := pktline.NewEncoder(proto.w)
// 	enc.Encode([]byte(fmt.Sprintf("# service=%s\n", service)))
// 	enc.Encode(nil)

// 	// Repo empty so send zeros
// 	if len(refs.Heads) == 0 && len(refs.Tags) == 0 {
// 		b0 := append([]byte("0000000000000000000000000000000000000000"), 32)
// 		b0 = append(b0, nullCapabilities()...)

// 		enc.Encode(append(b0, 10))
// 		enc.Encode(nil)
// 		return
// 	}

// 	// Send HEAD info
// 	head := refs.Head

// 	lh := append([]byte(fmt.Sprintf("%s HEAD", head.Hash.String())), '\x00')
// 	lh = append(lh, capabilities()...)

// 	if service == GitServiceUploadPack {
// 		lh = append(lh, []byte(" symref=HEAD:refs/"+head.Ref)...)
// 	}

// 	enc.Encode(append(lh, 10))

// 	// Send refs - heads
// 	for href, h := range refs.Heads {
// 		enc.Encode([]byte(fmt.Sprintf("%s refs/heads/%s\n", h.String(), href)))
// 	}

// 	// Send refs - tags
// 	for tref, h := range refs.Tags {
// 		enc.Encode([]byte(fmt.Sprintf("%s refs/tags/%s\n", h.String(), tref)))
// 	}

// 	enc.Encode(nil)
// }

// ListReferences writes the references in the pack protocol given the repository
// and service type
func (proto *Protocol) ListReferences(service string, refs []*plumbing.Reference) {

	// Start sending info
	enc := pktline.NewEncoder(proto.w)
	enc.Encode([]byte(fmt.Sprintf("# service=%s\n", service)))
	enc.Flush()

	// Repo empty so send zeros
	if len(refs) == 0 {
		b0 := append([]byte("0000000000000000000000000000000000000000"), 32)
		b0 = append(b0, nullCapabilities()...)

		enc.Encode(append(b0, 10))
		enc.Flush()
		return
	}

	// Send HEAD info
	lh := append([]byte(fmt.Sprintf("%s %s", refs[0].Hash(), refs[0].Name())), '\x00')
	lh = append(lh, capabilities()...)

	if service == transport.UploadPackServiceName {
		// ! TODO: refs/heads/master should not be hardcoded, but should rather come from the HEAD reference.
		// ! refs[0].Target()
		lh = append(lh, []byte(" symref="+refs[0].Name()+":refs/heads/master")...)
	}

	enc.Encode(append(lh, 10))

	for _, ref := range refs[1:] {
		enc.Encode([]byte(fmt.Sprintf("%s %s\n", ref.Hash(), ref.Name())))
	}
	enc.Flush()
}

// UploadPack implements the git upload pack protocol
func (proto *Protocol) UploadPack(store storer.EncodedObjectStorer) ([]byte, error) {
	wants, haves, err := parseUploadPackWantsAndHaves(proto.r)
	if err != nil {
		return nil, err
	}

	log.Printf("DBG [upload-pack] wants=%d haves=%d", len(wants), len(haves))

	enc := pktline.NewEncoder(proto.w)
	enc.Encode([]byte("NAK\n"))

	packenc := packfile.NewEncoder(proto.w, store)
	return packenc.Encode(wants...)
}

// ReceivePack implements the git receive pack protocol
func (proto *Protocol) ReceivePack(objstore storer.Storer) error {
	enc := pktline.NewEncoder(proto.w)

	txs, err := parseReceivePackClientRefLines(proto.r)
	if err != nil {
		enc.Encode([]byte(fmt.Sprintf("unpack %v", err)))
		return err
	}

	// Decode packfile
	packdec := packfile.NewDecoder(proto.r, objstore)
	if err = packdec.Decode(); err != nil {
		enc.Encode([]byte(fmt.Sprintf("unpack %v", err)))
		return err
	}
	enc.Encode([]byte("unpack ok\n"))

	// Update repo refs
	for _, tx := range txs {
		if er := objstore.CheckAndSetReference(tx.new(), tx.old()); er != nil {
			enc.Encode([]byte(fmt.Sprintf("ng %s %v\n", tx.ref, er)))
		} else {
			enc.Encode([]byte(fmt.Sprintf("ok %s\n", tx.ref)))
		}
	}

	enc.Flush()
	return err
}

func parseReceivePackClientRefLines(r io.Reader) ([]txRef, error) {
	var (
		dec   = pktline.NewScanner(r)
		lines [][]byte
	)

	// Read refs from client
	for dec.Scan() {
		line := dec.Bytes()
		if len(line) == 0 {
			break
		}

		lines = append(lines, line)
	}

	if dec.Err() != nil {
		return nil, dec.Err()
	}

	txs := make([]txRef, len(lines))
	for i, l := range lines {
		log.Printf("DBG [receive-pack] %s", l)

		rt, err := newTxRefFromBytes(l)
		if err != nil {
			return nil, err
		}
		txs[i] = rt
	}

	return txs, nil
}

func parseUploadPackWantsAndHaves(r io.Reader) (wants, haves []plumbing.Hash, err error) {
	dec := pktline.NewScanner(r)

	for dec.Scan() {
		line := dec.Bytes()
		if len(line) == 0 {
			continue
		}

		if string(line) == "done" {
			break
		}

		log.Printf("DBG [upload-pack] %s", line)

		op := strings.Split(string(line), " ")
		switch op[0] {
		case "want":
			wants = append(wants, plumbing.NewHash(op[1]))

		case "have":
			haves = append(haves, plumbing.NewHash(op[1]))
		}
	}

	if dec.Err() != nil {
		return nil, nil, dec.Err()
	}

	return
}

func capabilities() []byte {
	//return []byte("report-status delete-refs ofs-delta multi_ack_detailed")
	return []byte("report-status delete-refs ofs-delta")
}

func nullCapabilities() []byte {
	return append(append([]byte("capabilities^{}"), '\x00'), capabilities()...)
}
