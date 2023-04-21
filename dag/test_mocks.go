// Code generated by MockGen. DO NOT EDIT.
// Source: dag/dag.go

// Package mock_dag is a generated GoMock package.
package dag

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	accounts "gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	common "gitlab.waterfall.network/waterfall/protocol/gwat/common"
	core "gitlab.waterfall.network/waterfall/protocol/gwat/core"
	state "gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	types "gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	downloader "gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	ethdb "gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	params "gitlab.waterfall.network/waterfall/protocol/gwat/params"
	era "gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	storage "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

// MockBackend is a mock of Backend interface.
type MockBackend struct {
	ctrl     *gomock.Controller
	recorder *MockBackendMockRecorder
}

// MockBackendMockRecorder is the mock recorder for MockBackend.
type MockBackendMockRecorder struct {
	mock *MockBackend
}

// NewMockBackend creates a new mock instance.
func NewMockBackend(ctrl *gomock.Controller) *MockBackend {
	mock := &MockBackend{ctrl: ctrl}
	mock.recorder = &MockBackendMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackend) EXPECT() *MockBackendMockRecorder {
	return m.recorder
}

// AccountManager mocks base method.
func (m *MockBackend) AccountManager() *accounts.Manager {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccountManager")
	ret0, _ := ret[0].(*accounts.Manager)
	return ret0
}

// AccountManager indicates an expected call of AccountManager.
func (mr *MockBackendMockRecorder) AccountManager() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccountManager", reflect.TypeOf((*MockBackend)(nil).AccountManager))
}

// BlockChain mocks base method.
func (m *MockBackend) BlockChain() *core.BlockChain {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockChain")
	ret0, _ := ret[0].(*core.BlockChain)
	return ret0
}

// BlockChain indicates an expected call of BlockChain.
func (mr *MockBackendMockRecorder) BlockChain() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockChain", reflect.TypeOf((*MockBackend)(nil).BlockChain))
}

// CreatorAuthorize mocks base method.
func (m *MockBackend) CreatorAuthorize(creator common.Address) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatorAuthorize", creator)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreatorAuthorize indicates an expected call of CreatorAuthorize.
func (mr *MockBackendMockRecorder) CreatorAuthorize(creator interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatorAuthorize", reflect.TypeOf((*MockBackend)(nil).CreatorAuthorize), creator)
}

// Downloader mocks base method.
func (m *MockBackend) Downloader() *downloader.Downloader {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Downloader")
	ret0, _ := ret[0].(*downloader.Downloader)
	return ret0
}

// Downloader indicates an expected call of Downloader.
func (mr *MockBackendMockRecorder) Downloader() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Downloader", reflect.TypeOf((*MockBackend)(nil).Downloader))
}

// Etherbase mocks base method.
func (m *MockBackend) Etherbase() (common.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Etherbase")
	ret0, _ := ret[0].(common.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Etherbase indicates an expected call of Etherbase.
func (mr *MockBackendMockRecorder) Etherbase() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Etherbase", reflect.TypeOf((*MockBackend)(nil).Etherbase))
}

// IsDevMode mocks base method.
func (m *MockBackend) IsDevMode() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsDevMode")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsDevMode indicates an expected call of IsDevMode.
func (mr *MockBackendMockRecorder) IsDevMode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsDevMode", reflect.TypeOf((*MockBackend)(nil).IsDevMode))
}

// SetEtherbase mocks base method.
func (m *MockBackend) SetEtherbase(etherbase common.Address) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetEtherbase", etherbase)
}

// SetEtherbase indicates an expected call of SetEtherbase.
func (mr *MockBackendMockRecorder) SetEtherbase(etherbase interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetEtherbase", reflect.TypeOf((*MockBackend)(nil).SetEtherbase), etherbase)
}

// TxPool mocks base method.
func (m *MockBackend) TxPool() *core.TxPool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TxPool")
	ret0, _ := ret[0].(*core.TxPool)
	return ret0
}

// TxPool indicates an expected call of TxPool.
func (mr *MockBackendMockRecorder) TxPool() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TxPool", reflect.TypeOf((*MockBackend)(nil).TxPool))
}

// MockblockChain is a mock of blockChain interface.
type MockblockChain struct {
	ctrl     *gomock.Controller
	recorder *MockblockChainMockRecorder
}

// MockblockChainMockRecorder is the mock recorder for MockblockChain.
type MockblockChainMockRecorder struct {
	mock *MockblockChain
}

// NewMockblockChain creates a new mock instance.
func NewMockblockChain(ctrl *gomock.Controller) *MockblockChain {
	mock := &MockblockChain{ctrl: ctrl}
	mock.recorder = &MockblockChainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockblockChain) EXPECT() *MockblockChainMockRecorder {
	return m.recorder
}

// AppendNotProcessedValidatorSyncData mocks base method.
func (m *MockblockChain) AppendNotProcessedValidatorSyncData(valSyncData []*types.ValidatorSync) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AppendNotProcessedValidatorSyncData", valSyncData)
}

// AppendNotProcessedValidatorSyncData indicates an expected call of AppendNotProcessedValidatorSyncData.
func (mr *MockblockChainMockRecorder) AppendNotProcessedValidatorSyncData(valSyncData interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendNotProcessedValidatorSyncData", reflect.TypeOf((*MockblockChain)(nil).AppendNotProcessedValidatorSyncData), valSyncData)
}

// Config mocks base method.
func (m *MockblockChain) Config() *params.ChainConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Config")
	ret0, _ := ret[0].(*params.ChainConfig)
	return ret0
}

// Config indicates an expected call of Config.
func (mr *MockblockChainMockRecorder) Config() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Config", reflect.TypeOf((*MockblockChain)(nil).Config))
}

// DagMuLock mocks base method.
func (m *MockblockChain) DagMuLock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DagMuLock")
}

// DagMuLock indicates an expected call of DagMuLock.
func (mr *MockblockChainMockRecorder) DagMuLock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DagMuLock", reflect.TypeOf((*MockblockChain)(nil).DagMuLock))
}

// DagMuUnlock mocks base method.
func (m *MockblockChain) DagMuUnlock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DagMuUnlock")
}

// DagMuUnlock indicates an expected call of DagMuUnlock.
func (mr *MockblockChainMockRecorder) DagMuUnlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DagMuUnlock", reflect.TypeOf((*MockblockChain)(nil).DagMuUnlock))
}

// Database mocks base method.
func (m *MockblockChain) Database() ethdb.Database {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Database")
	ret0, _ := ret[0].(ethdb.Database)
	return ret0
}

// Database indicates an expected call of Database.
func (mr *MockblockChainMockRecorder) Database() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Database", reflect.TypeOf((*MockblockChain)(nil).Database))
}

// EnterNextEra mocks base method.
func (m *MockblockChain) EnterNextEra(root common.Hash) *era.Era {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnterNextEra", root)
	ret0, _ := ret[0].(*era.Era)
	return ret0
}

// EnterNextEra indicates an expected call of EnterNextEra.
func (mr *MockblockChainMockRecorder) EnterNextEra(root interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnterNextEra", reflect.TypeOf((*MockblockChain)(nil).EnterNextEra), root)
}

// GetBlock mocks base method.
func (m *MockblockChain) GetBlock(hash common.Hash) *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlock", hash)
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// GetBlock indicates an expected call of GetBlock.
func (mr *MockblockChainMockRecorder) GetBlock(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlock", reflect.TypeOf((*MockblockChain)(nil).GetBlock), hash)
}

// GetEraInfo mocks base method.
func (m *MockblockChain) GetEraInfo() *era.EraInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEraInfo")
	ret0, _ := ret[0].(*era.EraInfo)
	return ret0
}

// GetEraInfo indicates an expected call of GetEraInfo.
func (mr *MockblockChainMockRecorder) GetEraInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEraInfo", reflect.TypeOf((*MockblockChain)(nil).GetEraInfo))
}

// GetHeaderByHash mocks base method.
func (m *MockblockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeaderByHash", hash)
	ret0, _ := ret[0].(*types.Header)
	return ret0
}

// GetHeaderByHash indicates an expected call of GetHeaderByHash.
func (mr *MockblockChainMockRecorder) GetHeaderByHash(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeaderByHash", reflect.TypeOf((*MockblockChain)(nil).GetHeaderByHash), hash)
}

// GetLastCoordinatedCheckpoint mocks base method.
func (m *MockblockChain) GetLastCoordinatedCheckpoint() *types.Checkpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastCoordinatedCheckpoint")
	ret0, _ := ret[0].(*types.Checkpoint)
	return ret0
}

// GetLastCoordinatedCheckpoint indicates an expected call of GetLastCoordinatedCheckpoint.
func (mr *MockblockChainMockRecorder) GetLastCoordinatedCheckpoint() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastCoordinatedCheckpoint", reflect.TypeOf((*MockblockChain)(nil).GetLastCoordinatedCheckpoint))
}

// GetLastFinalizedBlock mocks base method.
func (m *MockblockChain) GetLastFinalizedBlock() *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastFinalizedBlock")
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// GetLastFinalizedBlock indicates an expected call of GetLastFinalizedBlock.
func (mr *MockblockChainMockRecorder) GetLastFinalizedBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastFinalizedBlock", reflect.TypeOf((*MockblockChain)(nil).GetLastFinalizedBlock))
}

// GetLastFinalizedHeader mocks base method.
func (m *MockblockChain) GetLastFinalizedHeader() *types.Header {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastFinalizedHeader")
	ret0, _ := ret[0].(*types.Header)
	return ret0
}

// GetLastFinalizedHeader indicates an expected call of GetLastFinalizedHeader.
func (mr *MockblockChainMockRecorder) GetLastFinalizedHeader() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastFinalizedHeader", reflect.TypeOf((*MockblockChain)(nil).GetLastFinalizedHeader))
}

// GetOptimisticSpinesFromCache mocks base method.
func (m *MockblockChain) GetOptimisticSpinesFromCache(slot uint64) common.HashArray {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOptimisticSpinesFromCache", slot)
	ret0, _ := ret[0].(common.HashArray)
	return ret0
}

// GetOptimisticSpinesFromCache indicates an expected call of GetOptimisticSpinesFromCache.
func (mr *MockblockChainMockRecorder) GetOptimisticSpinesFromCache(slot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOptimisticSpinesFromCache", reflect.TypeOf((*MockblockChain)(nil).GetOptimisticSpinesFromCache), slot)
}

// GetSlotInfo mocks base method.
func (m *MockblockChain) GetSlotInfo() *types.SlotInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSlotInfo")
	ret0, _ := ret[0].(*types.SlotInfo)
	return ret0
}

// GetSlotInfo indicates an expected call of GetSlotInfo.
func (mr *MockblockChainMockRecorder) GetSlotInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSlotInfo", reflect.TypeOf((*MockblockChain)(nil).GetSlotInfo))
}

// GetTips mocks base method.
func (m *MockblockChain) GetTips() types.Tips {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTips")
	ret0, _ := ret[0].(types.Tips)
	return ret0
}

// GetTips indicates an expected call of GetTips.
func (mr *MockblockChainMockRecorder) GetTips() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTips", reflect.TypeOf((*MockblockChain)(nil).GetTips))
}

// SetLastCoordinatedCheckpoint mocks base method.
func (m *MockblockChain) SetLastCoordinatedCheckpoint(cp *types.Checkpoint) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetLastCoordinatedCheckpoint", cp)
}

// SetLastCoordinatedCheckpoint indicates an expected call of SetLastCoordinatedCheckpoint.
func (mr *MockblockChainMockRecorder) SetLastCoordinatedCheckpoint(cp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLastCoordinatedCheckpoint", reflect.TypeOf((*MockblockChain)(nil).SetLastCoordinatedCheckpoint), cp)
}

// SetNewEraInfo mocks base method.
func (m *MockblockChain) SetNewEraInfo(newEra era.Era) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetNewEraInfo", newEra)
}

// SetNewEraInfo indicates an expected call of SetNewEraInfo.
func (mr *MockblockChainMockRecorder) SetNewEraInfo(newEra interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNewEraInfo", reflect.TypeOf((*MockblockChain)(nil).SetNewEraInfo), newEra)
}

// SetOptimisticSpinesToCache mocks base method.
func (m *MockblockChain) SetOptimisticSpinesToCache(slot uint64, spines common.HashArray) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetOptimisticSpinesToCache", slot, spines)
}

// SetOptimisticSpinesToCache indicates an expected call of SetOptimisticSpinesToCache.
func (mr *MockblockChainMockRecorder) SetOptimisticSpinesToCache(slot, spines interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetOptimisticSpinesToCache", reflect.TypeOf((*MockblockChain)(nil).SetOptimisticSpinesToCache), slot, spines)
}

// SetSlotInfo mocks base method.
func (m *MockblockChain) SetSlotInfo(si *types.SlotInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetSlotInfo", si)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetSlotInfo indicates an expected call of SetSlotInfo.
func (mr *MockblockChainMockRecorder) SetSlotInfo(si interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSlotInfo", reflect.TypeOf((*MockblockChain)(nil).SetSlotInfo), si)
}

// StartTransitionPeriod mocks base method.
func (m *MockblockChain) StartTransitionPeriod() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StartTransitionPeriod")
}

// StartTransitionPeriod indicates an expected call of StartTransitionPeriod.
func (mr *MockblockChainMockRecorder) StartTransitionPeriod() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartTransitionPeriod", reflect.TypeOf((*MockblockChain)(nil).StartTransitionPeriod))
}

// StateAt mocks base method.
func (m *MockblockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateAt", root)
	ret0, _ := ret[0].(*state.StateDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StateAt indicates an expected call of StateAt.
func (mr *MockblockChainMockRecorder) StateAt(root interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateAt", reflect.TypeOf((*MockblockChain)(nil).StateAt), root)
}

// SyncEraToSlot mocks base method.
func (m *MockblockChain) SyncEraToSlot(slot uint64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SyncEraToSlot", slot)
}

// SyncEraToSlot indicates an expected call of SyncEraToSlot.
func (mr *MockblockChainMockRecorder) SyncEraToSlot(slot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SyncEraToSlot", reflect.TypeOf((*MockblockChain)(nil).SyncEraToSlot), slot)
}

// ValidatorStorage mocks base method.
func (m *MockblockChain) ValidatorStorage() storage.Storage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidatorStorage")
	ret0, _ := ret[0].(storage.Storage)
	return ret0
}

// ValidatorStorage indicates an expected call of ValidatorStorage.
func (mr *MockblockChainMockRecorder) ValidatorStorage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidatorStorage", reflect.TypeOf((*MockblockChain)(nil).ValidatorStorage))
}

// TODO: rm deprecated
// WriteLastCoordinatedHash mocks base method.
//func (m *MockblockChain) WriteLastCoordinatedHash(hash common.Hash) {
//	m.ctrl.T.Helper()
//	m.ctrl.Call(m, "WriteLastCoordinatedHash", hash)
//}

// TODO: rm deprecated
//// WriteLastCoordinatedHash indicates an expected call of WriteLastCoordinatedHash.
//func (mr *MockblockChainMockRecorder) WriteLastCoordinatedHash(hash interface{}) *gomock.Call {
//	mr.mock.ctrl.T.Helper()
//	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteLastCoordinatedHash", reflect.TypeOf((*MockblockChain)(nil).WriteLastCoordinatedHash), hash)
//}

// MockethDownloader is a mock of ethDownloader interface.
type MockethDownloader struct {
	ctrl     *gomock.Controller
	recorder *MockethDownloaderMockRecorder
}

// MockethDownloaderMockRecorder is the mock recorder for MockethDownloader.
type MockethDownloaderMockRecorder struct {
	mock *MockethDownloader
}

// NewMockethDownloader creates a new mock instance.
func NewMockethDownloader(ctrl *gomock.Controller) *MockethDownloader {
	mock := &MockethDownloader{ctrl: ctrl}
	mock.recorder = &MockethDownloaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockethDownloader) EXPECT() *MockethDownloaderMockRecorder {
	return m.recorder
}

// Synchronising mocks base method.
func (m *MockethDownloader) Synchronising() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Synchronising")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Synchronising indicates an expected call of Synchronising.
func (mr *MockethDownloaderMockRecorder) Synchronising() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Synchronising", reflect.TypeOf((*MockethDownloader)(nil).Synchronising))
}
