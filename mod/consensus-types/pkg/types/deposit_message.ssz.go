// Code generated by fastssz. DO NOT EDIT.
// Hash: 0fb5813ab33e29f28ad832a610dfd44e808e1b3a993226ed9e8e287e2fc0807d
// Version: 0.1.3
package types

import (
	"github.com/berachain/beacon-kit/mod/primitives/pkg/math"
	ssz "github.com/ferranbt/fastssz"
)

// MarshalSSZ ssz marshals the DepositMessage object
func (d *DepositMessage) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(d)
}

// MarshalSSZTo ssz marshals the DepositMessage object to a target array
func (d *DepositMessage) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'Pubkey'
	dst = append(dst, d.Pubkey[:]...)

	// Field (1) 'Credentials'
	dst = append(dst, d.Credentials[:]...)

	// Field (2) 'Amount'
	dst = ssz.MarshalUint64(dst, uint64(d.Amount))

	return
}

// UnmarshalSSZ ssz unmarshals the DepositMessage object
func (d *DepositMessage) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 88 {
		return ssz.ErrSize
	}

	// Field (0) 'Pubkey'
	copy(d.Pubkey[:], buf[0:48])

	// Field (1) 'Credentials'
	copy(d.Credentials[:], buf[48:80])

	// Field (2) 'Amount'
	d.Amount = math.Gwei(ssz.UnmarshallUint64(buf[80:88]))

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the DepositMessage object
func (d *DepositMessage) SizeSSZ() (size int) {
	size = 88
	return
}

// HashTreeRoot ssz hashes the DepositMessage object
func (d *DepositMessage) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(d)
}

// HashTreeRootWith ssz hashes the DepositMessage object with a hasher
func (d *DepositMessage) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'Pubkey'
	hh.PutBytes(d.Pubkey[:])

	// Field (1) 'Credentials'
	hh.PutBytes(d.Credentials[:])

	// Field (2) 'Amount'
	hh.PutUint64(uint64(d.Amount))

	hh.Merkleize(indx)
	return
}

// GetTree ssz hashes the DepositMessage object
func (d *DepositMessage) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(d)
}
