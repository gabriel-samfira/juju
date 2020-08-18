// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/controller/migrationmaster (interfaces: Backend,OfferConnection)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	description "github.com/juju/description/v2"
	state "github.com/juju/juju/state"
	names "github.com/juju/names/v4"
	version "github.com/juju/version"
	reflect "reflect"
)

// MockBackend is a mock of Backend interface
type MockBackend struct {
	ctrl     *gomock.Controller
	recorder *MockBackendMockRecorder
}

// MockBackendMockRecorder is the mock recorder for MockBackend
type MockBackendMockRecorder struct {
	mock *MockBackend
}

// NewMockBackend creates a new mock instance
func NewMockBackend(ctrl *gomock.Controller) *MockBackend {
	mock := &MockBackend{ctrl: ctrl}
	mock.recorder = &MockBackendMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBackend) EXPECT() *MockBackendMockRecorder {
	return m.recorder
}

// AgentVersion mocks base method
func (m *MockBackend) AgentVersion() (version.Number, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AgentVersion")
	ret0, _ := ret[0].(version.Number)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AgentVersion indicates an expected call of AgentVersion
func (mr *MockBackendMockRecorder) AgentVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AgentVersion", reflect.TypeOf((*MockBackend)(nil).AgentVersion))
}

// Export mocks base method
func (m *MockBackend) Export() (description.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Export")
	ret0, _ := ret[0].(description.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Export indicates an expected call of Export
func (mr *MockBackendMockRecorder) Export() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Export", reflect.TypeOf((*MockBackend)(nil).Export))
}

// LatestMigration mocks base method
func (m *MockBackend) LatestMigration() (state.ModelMigration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LatestMigration")
	ret0, _ := ret[0].(state.ModelMigration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LatestMigration indicates an expected call of LatestMigration
func (mr *MockBackendMockRecorder) LatestMigration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LatestMigration", reflect.TypeOf((*MockBackend)(nil).LatestMigration))
}

// ModelName mocks base method
func (m *MockBackend) ModelName() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelName")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelName indicates an expected call of ModelName
func (mr *MockBackendMockRecorder) ModelName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelName", reflect.TypeOf((*MockBackend)(nil).ModelName))
}

// ModelOwner mocks base method
func (m *MockBackend) ModelOwner() (names.UserTag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelOwner")
	ret0, _ := ret[0].(names.UserTag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelOwner indicates an expected call of ModelOwner
func (mr *MockBackendMockRecorder) ModelOwner() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelOwner", reflect.TypeOf((*MockBackend)(nil).ModelOwner))
}

// ModelUUID mocks base method
func (m *MockBackend) ModelUUID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelUUID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ModelUUID indicates an expected call of ModelUUID
func (mr *MockBackendMockRecorder) ModelUUID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelUUID", reflect.TypeOf((*MockBackend)(nil).ModelUUID))
}

// RemoveExportingModelDocs mocks base method
func (m *MockBackend) RemoveExportingModelDocs() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveExportingModelDocs")
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveExportingModelDocs indicates an expected call of RemoveExportingModelDocs
func (mr *MockBackendMockRecorder) RemoveExportingModelDocs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveExportingModelDocs", reflect.TypeOf((*MockBackend)(nil).RemoveExportingModelDocs))
}

// WatchForMigration mocks base method
func (m *MockBackend) WatchForMigration() state.NotifyWatcher {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchForMigration")
	ret0, _ := ret[0].(state.NotifyWatcher)
	return ret0
}

// WatchForMigration indicates an expected call of WatchForMigration
func (mr *MockBackendMockRecorder) WatchForMigration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchForMigration", reflect.TypeOf((*MockBackend)(nil).WatchForMigration))
}

// MockOfferConnection is a mock of OfferConnection interface
type MockOfferConnection struct {
	ctrl     *gomock.Controller
	recorder *MockOfferConnectionMockRecorder
}

// MockOfferConnectionMockRecorder is the mock recorder for MockOfferConnection
type MockOfferConnectionMockRecorder struct {
	mock *MockOfferConnection
}

// NewMockOfferConnection creates a new mock instance
func NewMockOfferConnection(ctrl *gomock.Controller) *MockOfferConnection {
	mock := &MockOfferConnection{ctrl: ctrl}
	mock.recorder = &MockOfferConnectionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOfferConnection) EXPECT() *MockOfferConnectionMockRecorder {
	return m.recorder
}

// OfferUUID mocks base method
func (m *MockOfferConnection) OfferUUID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OfferUUID")
	ret0, _ := ret[0].(string)
	return ret0
}

// OfferUUID indicates an expected call of OfferUUID
func (mr *MockOfferConnectionMockRecorder) OfferUUID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OfferUUID", reflect.TypeOf((*MockOfferConnection)(nil).OfferUUID))
}

// RelationId mocks base method
func (m *MockOfferConnection) RelationId() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RelationId")
	ret0, _ := ret[0].(int)
	return ret0
}

// RelationId indicates an expected call of RelationId
func (mr *MockOfferConnectionMockRecorder) RelationId() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RelationId", reflect.TypeOf((*MockOfferConnection)(nil).RelationId))
}

// RelationKey mocks base method
func (m *MockOfferConnection) RelationKey() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RelationKey")
	ret0, _ := ret[0].(string)
	return ret0
}

// RelationKey indicates an expected call of RelationKey
func (mr *MockOfferConnectionMockRecorder) RelationKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RelationKey", reflect.TypeOf((*MockOfferConnection)(nil).RelationKey))
}

// SourceModelUUID mocks base method
func (m *MockOfferConnection) SourceModelUUID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SourceModelUUID")
	ret0, _ := ret[0].(string)
	return ret0
}

// SourceModelUUID indicates an expected call of SourceModelUUID
func (mr *MockOfferConnectionMockRecorder) SourceModelUUID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SourceModelUUID", reflect.TypeOf((*MockOfferConnection)(nil).SourceModelUUID))
}

// UserName mocks base method
func (m *MockOfferConnection) UserName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UserName")
	ret0, _ := ret[0].(string)
	return ret0
}

// UserName indicates an expected call of UserName
func (mr *MockOfferConnectionMockRecorder) UserName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UserName", reflect.TypeOf((*MockOfferConnection)(nil).UserName))
}
