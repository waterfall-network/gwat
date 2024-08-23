// Code generated by MockGen. DO NOT EDIT.
// Source: validator/processor.go

// Package validator is a generated GoMock package.
package validator

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	common "gitlab.waterfall.network/waterfall/protocol/gwat/common"
	state "gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	types "gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	ethdb "gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	params "gitlab.waterfall.network/waterfall/protocol/gwat/params"
	era "gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	storage "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

// MockRef is a mock of Ref interface.
type MockRef struct {
	ctrl     *gomock.Controller
	recorder *MockRefMockRecorder
}

// MockRefMockRecorder is the mock recorder for MockRef.
type MockRefMockRecorder struct {
	mock *MockRef
}

// NewMockRef creates a new mock instance.
func NewMockRef(ctrl *gomock.Controller) *MockRef {
	mock := &MockRef{ctrl: ctrl}
	mock.recorder = &MockRefMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRef) EXPECT() *MockRefMockRecorder {
	return m.recorder
}

// Address mocks base method.
func (m *MockRef) Address() common.Address {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Address")
	ret0, _ := ret[0].(common.Address)
	return ret0
}

// Address indicates an expected call of Address.
func (mr *MockRefMockRecorder) Address() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Address", reflect.TypeOf((*MockRef)(nil).Address))
}

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

// Config mocks base method.
func (m *Mockblockchain) Config() *params.ChainConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Config")
	ret0, _ := ret[0].(*params.ChainConfig)
	return ret0
}

// Config indicates an expected call of Config.
func (mr *MockblockchainMockRecorder) Config() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Config", reflect.TypeOf((*Mockblockchain)(nil).Config))
}

// Database mocks base method.
func (m *Mockblockchain) Database() ethdb.Database {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Database")
	ret0, _ := ret[0].(ethdb.Database)
	return ret0
}

// Database indicates an expected call of Database.
func (mr *MockblockchainMockRecorder) Database() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Database", reflect.TypeOf((*Mockblockchain)(nil).Database))
}

// EnterNextEra mocks base method.
func (m *Mockblockchain) EnterNextEra(fromEpoch uint64, root, hash common.Hash) (*era.Era, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnterNextEra", fromEpoch, root, hash)
	ret0, _ := ret[0].(*era.Era)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnterNextEra indicates an expected call of EnterNextEra.
func (mr *MockblockchainMockRecorder) EnterNextEra(fromEpoch, root, hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnterNextEra", reflect.TypeOf((*Mockblockchain)(nil).EnterNextEra), fromEpoch, root, hash)
}

// EpochToEra mocks base method.
func (m *Mockblockchain) EpochToEra(epoch uint64) *era.Era {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EpochToEra", epoch)
	ret0, _ := ret[0].(*era.Era)
	return ret0
}

// EpochToEra indicates an expected call of EpochToEra.
func (mr *MockblockchainMockRecorder) EpochToEra(epoch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EpochToEra", reflect.TypeOf((*Mockblockchain)(nil).EpochToEra), epoch)
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

// GetEraInfo mocks base method.
func (m *Mockblockchain) GetEraInfo() *era.EraInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEraInfo")
	ret0, _ := ret[0].(*era.EraInfo)
	return ret0
}

// GetEraInfo indicates an expected call of GetEraInfo.
func (mr *MockblockchainMockRecorder) GetEraInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEraInfo", reflect.TypeOf((*Mockblockchain)(nil).GetEraInfo))
}

// GetHeaderByHash mocks base method.
func (m *Mockblockchain) GetHeaderByHash(arg0 common.Hash) *types.Header {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeaderByHash", arg0)
	ret0, _ := ret[0].(*types.Header)
	return ret0
}

// GetHeaderByHash indicates an expected call of GetHeaderByHash.
func (mr *MockblockchainMockRecorder) GetHeaderByHash(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeaderByHash", reflect.TypeOf((*Mockblockchain)(nil).GetHeaderByHash), arg0)
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

// GetTransaction mocks base method.
func (m *Mockblockchain) GetTransaction(txHash common.Hash) (*types.Transaction, common.Hash, uint64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransaction", txHash)
	ret0, _ := ret[0].(*types.Transaction)
	ret1, _ := ret[1].(common.Hash)
	ret2, _ := ret[2].(uint64)
	return ret0, ret1, ret2
}

// GetTransaction indicates an expected call of GetTransaction.
func (mr *MockblockchainMockRecorder) GetTransaction(txHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransaction", reflect.TypeOf((*Mockblockchain)(nil).GetTransaction), txHash)
}

// GetTransactionReceipt mocks base method.
func (m *Mockblockchain) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, common.Hash, uint64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactionReceipt", txHash)
	ret0, _ := ret[0].(*types.Receipt)
	ret1, _ := ret[1].(common.Hash)
	ret2, _ := ret[2].(uint64)
	return ret0, ret1, ret2
}

// GetTransactionReceipt indicates an expected call of GetTransactionReceipt.
func (mr *MockblockchainMockRecorder) GetTransactionReceipt(txHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactionReceipt", reflect.TypeOf((*Mockblockchain)(nil).GetTransactionReceipt), txHash)
}

// GetValidatorSyncData mocks base method.
func (m *Mockblockchain) GetValidatorSyncData(InitTxHash common.Hash) *types.ValidatorSync {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidatorSyncData", InitTxHash)
	ret0, _ := ret[0].(*types.ValidatorSync)
	return ret0
}

// GetValidatorSyncData indicates an expected call of GetValidatorSyncData.
func (mr *MockblockchainMockRecorder) GetValidatorSyncData(InitTxHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidatorSyncData", reflect.TypeOf((*Mockblockchain)(nil).GetValidatorSyncData), InitTxHash)
}

// StartTransitionPeriod mocks base method.
func (m *Mockblockchain) StartTransitionPeriod(cp *types.Checkpoint, spineRoot, spineHash common.Hash) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartTransitionPeriod", cp, spineRoot, spineHash)
	ret0, _ := ret[0].(error)
	return ret0
}

// StartTransitionPeriod indicates an expected call of StartTransitionPeriod.
func (mr *MockblockchainMockRecorder) StartTransitionPeriod(cp, spineRoot, spineHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartTransitionPeriod", reflect.TypeOf((*Mockblockchain)(nil).StartTransitionPeriod), cp, spineRoot, spineHash)
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

// ValidatorStorage mocks base method.
func (m *Mockblockchain) ValidatorStorage() storage.Storage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidatorStorage")
	ret0, _ := ret[0].(storage.Storage)
	return ret0
}

// ValidatorStorage indicates an expected call of ValidatorStorage.
func (mr *MockblockchainMockRecorder) ValidatorStorage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidatorStorage", reflect.TypeOf((*Mockblockchain)(nil).ValidatorStorage))
}

// Mockmessage is a mock of message interface.
type Mockmessage struct {
	ctrl     *gomock.Controller
	recorder *MockmessageMockRecorder
}

// MockmessageMockRecorder is the mock recorder for Mockmessage.
type MockmessageMockRecorder struct {
	mock *Mockmessage
}

// NewMockmessage creates a new mock instance.
func NewMockmessage(ctrl *gomock.Controller) *Mockmessage {
	mock := &Mockmessage{ctrl: ctrl}
	mock.recorder = &MockmessageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockmessage) EXPECT() *MockmessageMockRecorder {
	return m.recorder
}

// Data mocks base method.
func (m *Mockmessage) Data() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Data")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Data indicates an expected call of Data.
func (mr *MockmessageMockRecorder) Data() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Data", reflect.TypeOf((*Mockmessage)(nil).Data))
}

// TxHash mocks base method.
func (m *Mockmessage) TxHash() common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TxHash")
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// TxHash indicates an expected call of TxHash.
func (mr *MockmessageMockRecorder) TxHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TxHash", reflect.TypeOf((*Mockmessage)(nil).TxHash))
}
