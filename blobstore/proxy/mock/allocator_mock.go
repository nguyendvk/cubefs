// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cubefs/cubefs/blobstore/proxy/allocator (interfaces: VolumeMgr)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	clustermgr "github.com/cubefs/cubefs/blobstore/api/clustermgr"
	proxy "github.com/cubefs/cubefs/blobstore/api/proxy"
	codemode "github.com/cubefs/cubefs/blobstore/common/codemode"
	proto "github.com/cubefs/cubefs/blobstore/common/proto"
	gomock "github.com/golang/mock/gomock"
)

// MockVolumeMgr is a mock of VolumeMgr interface.
type MockVolumeMgr struct {
	ctrl     *gomock.Controller
	recorder *MockVolumeMgrMockRecorder
}

// MockVolumeMgrMockRecorder is the mock recorder for MockVolumeMgr.
type MockVolumeMgrMockRecorder struct {
	mock *MockVolumeMgr
}

// NewMockVolumeMgr creates a new mock instance.
func NewMockVolumeMgr(ctrl *gomock.Controller) *MockVolumeMgr {
	mock := &MockVolumeMgr{ctrl: ctrl}
	mock.recorder = &MockVolumeMgrMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVolumeMgr) EXPECT() *MockVolumeMgrMockRecorder {
	return m.recorder
}

// Alloc mocks base method.
func (m *MockVolumeMgr) Alloc(arg0 context.Context, arg1 *proxy.AllocVolsArgs) ([]proxy.AllocRet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Alloc", arg0, arg1)
	ret0, _ := ret[0].([]proxy.AllocRet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Alloc indicates an expected call of Alloc.
func (mr *MockVolumeMgrMockRecorder) Alloc(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Alloc", reflect.TypeOf((*MockVolumeMgr)(nil).Alloc), arg0, arg1)
}

// Close mocks base method.
func (m *MockVolumeMgr) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockVolumeMgrMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockVolumeMgr)(nil).Close))
}

// Discard mocks base method.
func (m *MockVolumeMgr) Discard(arg0 context.Context, arg1 *proxy.DiscardVolsArgs) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Discard", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Discard indicates an expected call of Discard.
func (mr *MockVolumeMgrMockRecorder) Discard(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Discard", reflect.TypeOf((*MockVolumeMgr)(nil).Discard), arg0, arg1)
}

// List mocks base method.
func (m *MockVolumeMgr) List(arg0 context.Context, arg1 codemode.CodeMode) ([]proto.Vid, []clustermgr.AllocVolumeInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].([]proto.Vid)
	ret1, _ := ret[1].([]clustermgr.AllocVolumeInfo)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// List indicates an expected call of List.
func (mr *MockVolumeMgrMockRecorder) List(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockVolumeMgr)(nil).List), arg0, arg1)
}
