// Code generated by MockGen. DO NOT EDIT.
// Source: ./validator/storage/storage.go

// Package storage is a generated GoMock package.
package storage

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	common "gitlab.waterfall.network/waterfall/protocol/gwat/common"
	state "gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	types "gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	vm "gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	era "gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
)

// Mockblockchain is a mock of blockchain interface.
type Mockblockchain struct {
	ctrl     *gomock.Controller
	recorder *MockblockchainMockRecorder
}

// MockblockchainMockRecorder is the mock recorder for Mockblockchain.
type MockblockchainMockRecorder struct {
	mock *Mockblockchain
}

// NewMockblockchain creates a new mock instance.
func NewMockblockchain(ctrl *gomock.Controller) *Mockblockchain {
	mock := &Mockblockchain{ctrl: ctrl}
	mock.recorder = &MockblockchainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockblockchain) EXPECT() *MockblockchainMockRecorder {
	return m.recorder
}

// EpochToEra mocks base method.
func (m *Mockblockchain) EpochToEra(arg0 uint64) *era.Era {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EpochToEra", arg0)
	ret0, _ := ret[0].(*era.Era)
	return ret0
}

// EpochToEra indicates an expected call of EpochToEra.
func (mr *MockblockchainMockRecorder) EpochToEra(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EpochToEra", reflect.TypeOf((*Mockblockchain)(nil).EpochToEra), arg0)
}

// GetBlock mocks base method.
func (m *Mockblockchain) GetBlock(ctx context.Context, hash common.Hash) *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlock", ctx, hash)
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// GetBlock indicates an expected call of GetBlock.
func (mr *MockblockchainMockRecorder) GetBlock(ctx, hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlock", reflect.TypeOf((*Mockblockchain)(nil).GetBlock), ctx, hash)
}

// GetEpoch mocks base method.
func (m *Mockblockchain) GetEpoch(epoch uint64) common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpoch", epoch)
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// GetEpoch indicates an expected call of GetEpoch.
func (mr *MockblockchainMockRecorder) GetEpoch(epoch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpoch", reflect.TypeOf((*Mockblockchain)(nil).GetEpoch), epoch)
}

// GetLastCoordinatedCheckpoint mocks base method.
func (m *Mockblockchain) GetLastCoordinatedCheckpoint() *types.Checkpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastCoordinatedCheckpoint")
	ret0, _ := ret[0].(*types.Checkpoint)
	return ret0
}

// GetLastCoordinatedCheckpoint indicates an expected call of GetLastCoordinatedCheckpoint.
func (mr *MockblockchainMockRecorder) GetLastCoordinatedCheckpoint() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastCoordinatedCheckpoint", reflect.TypeOf((*Mockblockchain)(nil).GetLastCoordinatedCheckpoint))
}

// GetSlotInfo mocks base method.
func (m *Mockblockchain) GetSlotInfo() *types.SlotInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSlotInfo")
	ret0, _ := ret[0].(*types.SlotInfo)
	return ret0
}

// GetSlotInfo indicates an expected call of GetSlotInfo.
func (mr *MockblockchainMockRecorder) GetSlotInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSlotInfo", reflect.TypeOf((*Mockblockchain)(nil).GetSlotInfo))
}

// StateAt mocks base method.
func (m *Mockblockchain) StateAt(root common.Hash) (*state.StateDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateAt", root)
	ret0, _ := ret[0].(*state.StateDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StateAt indicates an expected call of StateAt.
func (mr *MockblockchainMockRecorder) StateAt(root interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateAt", reflect.TypeOf((*Mockblockchain)(nil).StateAt), root)
}

// MockStorage is a mock of Storage interface.
type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
}

// MockStorageMockRecorder is the mock recorder for MockStorage.
type MockStorageMockRecorder struct {
	mock *MockStorage
}

// NewMockStorage creates a new mock instance.
func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

// AddValidatorToList mocks base method.
func (m *MockStorage) AddValidatorToList(stateDB vm.StateDB, index uint64, validator common.Address) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddValidatorToList", stateDB, index, validator)
}

// AddValidatorToList indicates an expected call of AddValidatorToList.
func (mr *MockStorageMockRecorder) AddValidatorToList(stateDB, index, validator interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddValidatorToList", reflect.TypeOf((*MockStorage)(nil).AddValidatorToList), stateDB, index, validator)
}

// GetCreatorsBySlot mocks base method.
func (m *MockStorage) GetCreatorsBySlot(bc blockchain, filter ...uint64) ([]common.Address, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{bc}
	for _, a := range filter {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetCreatorsBySlot", varargs...)
	ret0, _ := ret[0].([]common.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCreatorsBySlot indicates an expected call of GetCreatorsBySlot.
func (mr *MockStorageMockRecorder) GetCreatorsBySlot(bc interface{}, filter ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{bc}, filter...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCreatorsBySlot", reflect.TypeOf((*MockStorage)(nil).GetCreatorsBySlot), varargs...)
}

// GetDepositCount mocks base method.
func (m *MockStorage) GetDepositCount(stateDb vm.StateDB) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDepositCount", stateDb)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetDepositCount indicates an expected call of GetDepositCount.
func (mr *MockStorageMockRecorder) GetDepositCount(stateDb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDepositCount", reflect.TypeOf((*MockStorage)(nil).GetDepositCount), stateDb)
}

// GetValidator mocks base method.
func (m *MockStorage) GetValidator(stateDb vm.StateDB, address common.Address) (*Validator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidator", stateDb, address)
	ret0, _ := ret[0].(*Validator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetValidator indicates an expected call of GetValidator.
func (mr *MockStorageMockRecorder) GetValidator(stateDb, address interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidator", reflect.TypeOf((*MockStorage)(nil).GetValidator), stateDb, address)
}

// GetValidators mocks base method.
func (m *MockStorage) GetValidators(bc blockchain, slot uint64, activeOnly, needAddresses bool, tmpFromWhere string) ([]Validator, []common.Address) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidators", bc, slot, activeOnly, needAddresses, tmpFromWhere)
	ret0, _ := ret[0].([]Validator)
	ret1, _ := ret[1].([]common.Address)
	return ret0, ret1
}

// GetValidators indicates an expected call of GetValidators.
func (mr *MockStorageMockRecorder) GetValidators(bc, slot, activeOnly, needAddresses, tmpFromWhere interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidators", reflect.TypeOf((*MockStorage)(nil).GetValidators), bc, slot, activeOnly, needAddresses, tmpFromWhere)
}

// GetValidatorsList mocks base method.
func (m *MockStorage) GetValidatorsList(stateDb vm.StateDB) []common.Address {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidatorsList", stateDb)
	ret0, _ := ret[0].([]common.Address)
	return ret0
}

// GetValidatorsList indicates an expected call of GetValidatorsList.
func (mr *MockStorageMockRecorder) GetValidatorsList(stateDb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidatorsList", reflect.TypeOf((*MockStorage)(nil).GetValidatorsList), stateDb)
}

// GetValidatorsStateAddress mocks base method.
func (m *MockStorage) GetValidatorsStateAddress() *common.Address {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidatorsStateAddress")
	ret0, _ := ret[0].(*common.Address)
	return ret0
}

// GetValidatorsStateAddress indicates an expected call of GetValidatorsStateAddress.
func (mr *MockStorageMockRecorder) GetValidatorsStateAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidatorsStateAddress", reflect.TypeOf((*MockStorage)(nil).GetValidatorsStateAddress))
}

// IncrementDepositCount mocks base method.
func (m *MockStorage) IncrementDepositCount(stateDb vm.StateDB) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IncrementDepositCount", stateDb)
}

// IncrementDepositCount indicates an expected call of IncrementDepositCount.
func (mr *MockStorageMockRecorder) IncrementDepositCount(stateDb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IncrementDepositCount", reflect.TypeOf((*MockStorage)(nil).IncrementDepositCount), stateDb)
}

// SetValidator mocks base method.
func (m *MockStorage) SetValidator(stateDb vm.StateDB, val *Validator) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetValidator", stateDb, val)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetValidator indicates an expected call of SetValidator.
func (mr *MockStorageMockRecorder) SetValidator(stateDb, val interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetValidator", reflect.TypeOf((*MockStorage)(nil).SetValidator), stateDb, val)
}

// SetValidatorsList mocks base method.
func (m *MockStorage) SetValidatorsList(stateDb vm.StateDB, list []common.Address) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetValidatorsList", stateDb, list)
}

// SetValidatorsList indicates an expected call of SetValidatorsList.
func (mr *MockStorageMockRecorder) SetValidatorsList(stateDb, list interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetValidatorsList", reflect.TypeOf((*MockStorage)(nil).SetValidatorsList), stateDb, list)
}
