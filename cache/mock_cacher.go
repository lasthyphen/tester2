// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ava-labs/avalanchego/cache (interfaces: Cacher)

// Package cache is a generated GoMock package.
package cache

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCacher is a mock of Cacher interface.
type MockCacher struct {
	ctrl     *gomock.Controller
	recorder *MockCacherMockRecorder
}

// MockCacherMockRecorder is the mock recorder for MockCacher.
type MockCacherMockRecorder struct {
	mock *MockCacher
}

// NewMockCacher creates a new mock instance.
func NewMockCacher(ctrl *gomock.Controller) *MockCacher {
	mock := &MockCacher{ctrl: ctrl}
	mock.recorder = &MockCacherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCacher) EXPECT() *MockCacherMockRecorder {
	return m.recorder
}

// Evict mocks base method.
func (m *MockCacher) Evict(arg0 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Evict", arg0)
}

// Evict indicates an expected call of Evict.
func (mr *MockCacherMockRecorder) Evict(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Evict", reflect.TypeOf((*MockCacher)(nil).Evict), arg0)
}

// Flush mocks base method.
func (m *MockCacher) Flush() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Flush")
}

// Flush indicates an expected call of Flush.
func (mr *MockCacherMockRecorder) Flush() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Flush", reflect.TypeOf((*MockCacher)(nil).Flush))
}

// Get mocks base method.
func (m *MockCacher) Get(arg0 interface{}) (interface{}, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockCacherMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCacher)(nil).Get), arg0)
}

// Put mocks base method.
func (m *MockCacher) Put(arg0, arg1 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Put", arg0, arg1)
}

// Put indicates an expected call of Put.
func (mr *MockCacherMockRecorder) Put(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockCacher)(nil).Put), arg0, arg1)
}
