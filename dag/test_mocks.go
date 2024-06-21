// Code generated by MockGen. DO NOT EDIT.
// Source: dag/dag.go

// Package dag is a generated GoMock package.
package dag

import (
	context "context"
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

// CollectAncestorsAftCpByParents mocks base method.
func (m *MockblockChain) CollectAncestorsAftCpByParents(arg0 common.HashArray, arg1 common.Hash) (bool, types.HeaderMap, common.HashArray, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectAncestorsAftCpByParents", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(types.HeaderMap)
	ret2, _ := ret[2].(common.HashArray)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// CollectAncestorsAftCpByParents indicates an expected call of CollectAncestorsAftCpByParents.
func (mr *MockblockChainMockRecorder) CollectAncestorsAftCpByParents(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectAncestorsAftCpByParents", reflect.TypeOf((*MockblockChain)(nil).CollectAncestorsAftCpByParents), arg0, arg1)
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
func (m *MockblockChain) EnterNextEra(nextEraEpochFrom uint64, root common.Hash) (*era.Era, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnterNextEra", nextEraEpochFrom, root)
	ret0, _ := ret[0].(*era.Era)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnterNextEra indicates an expected call of EnterNextEra.
func (mr *MockblockChainMockRecorder) EnterNextEra(nextEraEpochFrom, root interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnterNextEra", reflect.TypeOf((*MockblockChain)(nil).EnterNextEra), nextEraEpochFrom, root)
}

// EpochToEra mocks base method.
func (m *MockblockChain) EpochToEra(arg0 uint64) *era.Era {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EpochToEra", arg0)
	ret0, _ := ret[0].(*era.Era)
	return ret0
}

// EpochToEra indicates an expected call of EpochToEra.
func (mr *MockblockChainMockRecorder) EpochToEra(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EpochToEra", reflect.TypeOf((*MockblockChain)(nil).EpochToEra), arg0)
}

// Genesis mocks base method.
func (m *MockblockChain) Genesis() *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Genesis")
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// Genesis indicates an expected call of Genesis.
func (mr *MockblockChainMockRecorder) Genesis() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Genesis", reflect.TypeOf((*MockblockChain)(nil).Genesis))
}

// GetBlock mocks base method.
func (m *MockblockChain) GetBlock(ctx context.Context, hash common.Hash) *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlock", ctx, hash)
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// GetBlock indicates an expected call of GetBlock.
func (mr *MockblockChainMockRecorder) GetBlock(ctx, hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlock", reflect.TypeOf((*MockblockChain)(nil).GetBlock), ctx, hash)
}

// GetBlockByHash mocks base method.
func (m *MockblockChain) GetBlockByHash(hash common.Hash) *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByHash", hash)
	ret0, _ := ret[0].(*types.Block)
	return ret0
}

// GetBlockByHash indicates an expected call of GetBlockByHash.
func (mr *MockblockChainMockRecorder) GetBlockByHash(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByHash", reflect.TypeOf((*MockblockChain)(nil).GetBlockByHash), hash)
}

// GetBlockHashesBySlot mocks base method.
func (m *MockblockChain) GetBlockHashesBySlot(slot uint64) common.HashArray {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockHashesBySlot", slot)
	ret0, _ := ret[0].(common.HashArray)
	return ret0
}

// GetBlockHashesBySlot indicates an expected call of GetBlockHashesBySlot.
func (mr *MockblockChainMockRecorder) GetBlockHashesBySlot(slot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockHashesBySlot", reflect.TypeOf((*MockblockChain)(nil).GetBlockHashesBySlot), slot)
}

// GetBlocksByHashes mocks base method.
func (m *MockblockChain) GetBlocksByHashes(hashes common.HashArray) types.BlockMap {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlocksByHashes", hashes)
	ret0, _ := ret[0].(types.BlockMap)
	return ret0
}

// GetBlocksByHashes indicates an expected call of GetBlocksByHashes.
func (mr *MockblockChainMockRecorder) GetBlocksByHashes(hashes interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlocksByHashes", reflect.TypeOf((*MockblockChain)(nil).GetBlocksByHashes), hashes)
}

// GetCoordinatedCheckpoint mocks base method.
func (m *MockblockChain) GetCoordinatedCheckpoint(cpSpine common.Hash) *types.Checkpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCoordinatedCheckpoint", cpSpine)
	ret0, _ := ret[0].(*types.Checkpoint)
	return ret0
}

// GetCoordinatedCheckpoint indicates an expected call of GetCoordinatedCheckpoint.
func (mr *MockblockChainMockRecorder) GetCoordinatedCheckpoint(cpSpine interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCoordinatedCheckpoint", reflect.TypeOf((*MockblockChain)(nil).GetCoordinatedCheckpoint), cpSpine)
}

// GetEpoch mocks base method.
func (m *MockblockChain) GetEpoch(epoch uint64) common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpoch", epoch)
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// GetEpoch indicates an expected call of GetEpoch.
func (mr *MockblockChainMockRecorder) GetEpoch(epoch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpoch", reflect.TypeOf((*MockblockChain)(nil).GetEpoch), epoch)
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
func (m *MockblockChain) GetHeaderByHash(arg0 common.Hash) *types.Header {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeaderByHash", arg0)
	ret0, _ := ret[0].(*types.Header)
	return ret0
}

// GetHeaderByHash indicates an expected call of GetHeaderByHash.
func (mr *MockblockChainMockRecorder) GetHeaderByHash(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeaderByHash", reflect.TypeOf((*MockblockChain)(nil).GetHeaderByHash), arg0)
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

// GetLastFinalizedNumber mocks base method.
func (m *MockblockChain) GetLastFinalizedNumber() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastFinalizedNumber")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetLastFinalizedNumber indicates an expected call of GetLastFinalizedNumber.
func (mr *MockblockChainMockRecorder) GetLastFinalizedNumber() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastFinalizedNumber", reflect.TypeOf((*MockblockChain)(nil).GetLastFinalizedNumber))
}

// GetOptimisticSpines mocks base method.
func (m *MockblockChain) GetOptimisticSpines(gtSlot uint64) ([]common.HashArray, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOptimisticSpines", gtSlot)
	ret0, _ := ret[0].([]common.HashArray)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOptimisticSpines indicates an expected call of GetOptimisticSpines.
func (mr *MockblockChainMockRecorder) GetOptimisticSpines(gtSlot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOptimisticSpines", reflect.TypeOf((*MockblockChain)(nil).GetOptimisticSpines), gtSlot)
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

// HaveEpochBlocks mocks base method.
func (m *MockblockChain) HaveEpochBlocks(epoch uint64) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HaveEpochBlocks", epoch)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HaveEpochBlocks indicates an expected call of HaveEpochBlocks.
func (mr *MockblockChainMockRecorder) HaveEpochBlocks(epoch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HaveEpochBlocks", reflect.TypeOf((*MockblockChain)(nil).HaveEpochBlocks), epoch)
}

// IsCheckpointOutdated mocks base method.
func (m *MockblockChain) IsCheckpointOutdated(arg0 *types.Checkpoint) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsCheckpointOutdated", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsCheckpointOutdated indicates an expected call of IsCheckpointOutdated.
func (mr *MockblockChainMockRecorder) IsCheckpointOutdated(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsCheckpointOutdated", reflect.TypeOf((*MockblockChain)(nil).IsCheckpointOutdated), arg0)
}

// IsSynced mocks base method.
func (m *MockblockChain) IsSynced() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSynced")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSynced indicates an expected call of IsSynced.
func (mr *MockblockChainMockRecorder) IsSynced() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSynced", reflect.TypeOf((*MockblockChain)(nil).IsSynced))
}

// RemoveTips mocks base method.
func (m *MockblockChain) RemoveTips(hashes common.HashArray) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RemoveTips", hashes)
}

// RemoveTips indicates an expected call of RemoveTips.
func (mr *MockblockChainMockRecorder) RemoveTips(hashes interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveTips", reflect.TypeOf((*MockblockChain)(nil).RemoveTips), hashes)
}

// ResetSyncCheckpointCache mocks base method.
func (m *MockblockChain) ResetSyncCheckpointCache() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ResetSyncCheckpointCache")
}

// ResetSyncCheckpointCache indicates an expected call of ResetSyncCheckpointCache.
func (mr *MockblockChainMockRecorder) ResetSyncCheckpointCache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetSyncCheckpointCache", reflect.TypeOf((*MockblockChain)(nil).ResetSyncCheckpointCache))
}

// SetIsSynced mocks base method.
func (m *MockblockChain) SetIsSynced(synced bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetIsSynced", synced)
}

// SetIsSynced indicates an expected call of SetIsSynced.
func (mr *MockblockChainMockRecorder) SetIsSynced(synced interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetIsSynced", reflect.TypeOf((*MockblockChain)(nil).SetIsSynced), synced)
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

// SetSyncCheckpointCache mocks base method.
func (m *MockblockChain) SetSyncCheckpointCache(cp *types.Checkpoint) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetSyncCheckpointCache", cp)
}

// SetSyncCheckpointCache indicates an expected call of SetSyncCheckpointCache.
func (mr *MockblockChainMockRecorder) SetSyncCheckpointCache(cp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSyncCheckpointCache", reflect.TypeOf((*MockblockChain)(nil).SetSyncCheckpointCache), cp)
}

// StartTransitionPeriod mocks base method.
func (m *MockblockChain) StartTransitionPeriod(cp *types.Checkpoint, spineRoot common.Hash) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartTransitionPeriod", cp, spineRoot)
	ret0, _ := ret[0].(error)
	return ret0
}

// StartTransitionPeriod indicates an expected call of StartTransitionPeriod.
func (mr *MockblockChainMockRecorder) StartTransitionPeriod(cp, spineRoot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartTransitionPeriod", reflect.TypeOf((*MockblockChain)(nil).StartTransitionPeriod), cp, spineRoot)
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

// WriteCurrentTips mocks base method.
func (m *MockblockChain) WriteCurrentTips() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WriteCurrentTips")
}

// WriteCurrentTips indicates an expected call of WriteCurrentTips.
func (mr *MockblockChainMockRecorder) WriteCurrentTips() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteCurrentTips", reflect.TypeOf((*MockblockChain)(nil).WriteCurrentTips))
}

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

// DagSync mocks base method.
func (m *MockethDownloader) DagSync(baseSpine common.Hash, spines common.HashArray) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DagSync", baseSpine, spines)
	ret0, _ := ret[0].(error)
	return ret0
}

// DagSync indicates an expected call of DagSync.
func (mr *MockethDownloaderMockRecorder) DagSync(baseSpine, spines interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DagSync", reflect.TypeOf((*MockethDownloader)(nil).DagSync), baseSpine, spines)
}

// MainSync mocks base method.
func (m *MockethDownloader) MainSync(baseSpine common.Hash, spines common.HashArray) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MainSync", baseSpine, spines)
	ret0, _ := ret[0].(error)
	return ret0
}

// MainSync indicates an expected call of MainSync.
func (mr *MockethDownloaderMockRecorder) MainSync(baseSpine, spines interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MainSync", reflect.TypeOf((*MockethDownloader)(nil).MainSync), baseSpine, spines)
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

// Terminate mocks base method.
func (m *MockethDownloader) Terminate() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Terminate")
}

// Terminate indicates an expected call of Terminate.
func (mr *MockethDownloaderMockRecorder) Terminate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Terminate", reflect.TypeOf((*MockethDownloader)(nil).Terminate))
}
