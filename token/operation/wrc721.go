package operation

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type mintOperation struct {
	operation
	toOperation
	tokenIdOperation
	TokenMetadata []byte
}

// NewMintOperation creates a mint operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewMintOperation(to common.Address, tokenId *big.Int, metadata []byte) (Mint, error) {
	if to == (common.Address{}) {
		return nil, ErrNoTo
	}
	if tokenId == nil {
		return nil, ErrNoTokenId
	}

	return &mintOperation{
		operation: operation{
			Std: StdWRC721,
		},
		toOperation: toOperation{
			ToAddress: to,
		},
		tokenIdOperation: tokenIdOperation{
			Id: tokenId,
		},
		TokenMetadata: metadata,
	}, nil
}

// Code returns op code of a mint token operation
func (op *mintOperation) OpCode() Code {
	return MintCode
}

// Metadata returns copy of the metadata bytes if the field is set.
// Otherwise it returns nil.
func (op *mintOperation) Metadata() ([]byte, bool) {
	if len(op.TokenMetadata) == 0 {
		return nil, false
	}
	return makeCopy(op.TokenMetadata), true
}

// UnmarshalBinary unmarshals a mint operation from byte encoding
func (op *mintOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a mint operation to byte encoding
func (op *mintOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type burnOperation struct {
	operation
	tokenIdOperation
}

// NewBurnOperation creates a burn operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewBurnOperation(tokenId *big.Int) (Burn, error) {
	if tokenId == nil {
		return nil, ErrNoTokenId
	}

	return &burnOperation{
		operation: operation{
			Std: StdWRC721,
		},
		tokenIdOperation: tokenIdOperation{
			Id: tokenId,
		},
	}, nil
}

// Code returns op code of a burn token operation
func (op *burnOperation) OpCode() Code {
	return BurnCode
}

// UnmarshalBinary unmarshals a burn operation from byte encoding
func (op *burnOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a burn operation to byte encoding
func (op *burnOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type setApprovalForAllOperation struct {
	operation
	operatorOperation
	Approved bool
}

// NewSetApprovalForAllOperation creates a set approval for all operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewSetApprovalForAllOperation(operator common.Address, isApproved bool) (SetApprovalForAll, error) {
	if operator == (common.Address{}) {
		return nil, ErrNoOperator
	}

	return &setApprovalForAllOperation{
		operation: operation{
			Std: StdWRC721,
		},
		operatorOperation: operatorOperation{
			OperatorAddress: operator,
		},
		Approved: isApproved,
	}, nil
}

// Code returns op code of an set approval for all operation
func (op *setApprovalForAllOperation) OpCode() Code {
	return SetApprovalForAllCode
}

// Returns flag whether operations on NFT are approved or not
func (op *setApprovalForAllOperation) IsApproved() bool {
	return op.Approved
}

// UnmarshalBinary unmarshals a set approval for all operation from byte encoding
func (op *setApprovalForAllOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a set approval for all operation to byte encoding
func (op *setApprovalForAllOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type isApprovedForAllOperation struct {
	operation
	addressOperation
	ownerOperation
	operatorOperation
}

// NewIsApprovedForAllOperation creates an approved for all operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewIsApprovedForAllOperation(address common.Address, owner common.Address, operator common.Address) (IsApprovedForAll, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	if operator == (common.Address{}) {
		return nil, ErrNoOperator
	}

	return &isApprovedForAllOperation{
		operation: operation{
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
		operatorOperation: operatorOperation{
			OperatorAddress: operator,
		},
	}, nil
}

// Code returns op code of an opproved for all operation
func (op *isApprovedForAllOperation) OpCode() Code {
	return IsApprovedForAllCode
}

// UnmarshalBinary unmarshals a token allowance operation from byte encoding
func (op *isApprovedForAllOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token allowance operation to byte encoding
func (op *isApprovedForAllOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type safeTransferFromOperation struct {
	transferFromOperation
	OperationData []byte
}

// NewSafeTransferFromOperation creates a safe token transfer operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewSafeTransferFromOperation(from common.Address, to common.Address, value *big.Int, data []byte) (SafeTransferFrom, error) {
	transferOp, err := NewTransferFromOperation(StdWRC721, from, to, value)
	if err != nil {
		return nil, err
	}

	return &safeTransferFromOperation{
		transferFromOperation: *transferOp.(*transferFromOperation),
		OperationData:         data,
	}, nil
}

// Data returns copy of the data bytes if the field is set.
// Otherwise it returns nil.
func (op *safeTransferFromOperation) Data() ([]byte, bool) {
	if len(op.OperationData) == 0 {
		return nil, false
	}
	return makeCopy(op.OperationData), true
}

// Code returns op code of a balance of operation
func (op *safeTransferFromOperation) OpCode() Code {
	return SafeTransferFromCode
}

// UnmarshalBinary unmarshals a token safe transfer from operation from byte encoding
func (op *safeTransferFromOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token safe transfer from operation to byte encoding
func (op *safeTransferFromOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type tokenOfOwnerByIndexOperation struct {
	operation
	addressOperation
	ownerOperation
	TokenIndex *big.Int
}

// NewBurnOperation creates a token of owner by index operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewTokenOfOwnerByIndexOperation(address common.Address, owner common.Address, index *big.Int) (TokenOfOwnerByIndex, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	if index == nil {
		return nil, ErrNoIndex
	}

	return &tokenOfOwnerByIndexOperation{
		operation: operation{
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
		TokenIndex: index,
	}, nil
}

// Code returns op code of a token of owner by index token operation
func (op *tokenOfOwnerByIndexOperation) OpCode() Code {
	return TokenOfOwnerByIndexCode
}

// Index returns copy of the index field
func (op *tokenOfOwnerByIndexOperation) Index() *big.Int {
	return new(big.Int).Set(op.TokenIndex)
}

// UnmarshalBinary unmarshals a token of owner by index operation from byte encoding
func (op *tokenOfOwnerByIndexOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token of owner by index operation to byte encoding
func (op *tokenOfOwnerByIndexOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}
