// /*
// #########################
// #  SAP Steward-CI       #
// #########################
//
// THIS CODE IS GENERATED! DO NOT TOUCH!
//
// Copyright SAP SE.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
//

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/SAP/stewardci-core/pkg/runctl/run (interfaces: Run,Manager,SecretManager)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	cfg "github.com/SAP/stewardci-core/pkg/runctl/cfg"
	run "github.com/SAP/stewardci-core/pkg/runctl/run"
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	v10 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockRun is a mock of Run interface.
type MockRun struct {
	ctrl     *gomock.Controller
	recorder *MockRunMockRecorder
}

// MockRunMockRecorder is the mock recorder for MockRun.
type MockRunMockRecorder struct {
	mock *MockRun
}

// NewMockRun creates a new mock instance.
func NewMockRun(ctrl *gomock.Controller) *MockRun {
	mock := &MockRun{ctrl: ctrl}
	mock.recorder = &MockRunMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRun) EXPECT() *MockRunMockRecorder {
	return m.recorder
}

// GetCompletionTime mocks base method.
func (m *MockRun) GetCompletionTime() *v10.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCompletionTime")
	ret0, _ := ret[0].(*v10.Time)
	return ret0
}

// GetCompletionTime indicates an expected call of GetCompletionTime.
func (mr *MockRunMockRecorder) GetCompletionTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCompletionTime", reflect.TypeOf((*MockRun)(nil).GetCompletionTime))
}

// GetContainerInfo mocks base method.
func (m *MockRun) GetContainerInfo() *v1.ContainerState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContainerInfo")
	ret0, _ := ret[0].(*v1.ContainerState)
	return ret0
}

// GetContainerInfo indicates an expected call of GetContainerInfo.
func (mr *MockRunMockRecorder) GetContainerInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContainerInfo", reflect.TypeOf((*MockRun)(nil).GetContainerInfo))
}

// GetMessage mocks base method.
func (m *MockRun) GetMessage() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMessage")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetMessage indicates an expected call of GetMessage.
func (mr *MockRunMockRecorder) GetMessage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMessage", reflect.TypeOf((*MockRun)(nil).GetMessage))
}

// GetStartTime mocks base method.
func (m *MockRun) GetStartTime() *v10.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStartTime")
	ret0, _ := ret[0].(*v10.Time)
	return ret0
}

// GetStartTime indicates an expected call of GetStartTime.
func (mr *MockRunMockRecorder) GetStartTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStartTime", reflect.TypeOf((*MockRun)(nil).GetStartTime))
}

// IsFinished mocks base method.
func (m *MockRun) IsFinished() (bool, v1alpha1.Result) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsFinished")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(v1alpha1.Result)
	return ret0, ret1
}

// IsFinished indicates an expected call of IsFinished.
func (mr *MockRunMockRecorder) IsFinished() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsFinished", reflect.TypeOf((*MockRun)(nil).IsFinished))
}

// MockManager is a mock of Manager interface.
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager.
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance.
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// Cleanup mocks base method.
func (m *MockManager) Cleanup(arg0 context.Context, arg1 k8s.PipelineRun) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cleanup", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Cleanup indicates an expected call of Cleanup.
func (mr *MockManagerMockRecorder) Cleanup(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cleanup", reflect.TypeOf((*MockManager)(nil).Cleanup), arg0, arg1)
}

// GetRun mocks base method.
func (m *MockManager) GetRun(arg0 context.Context, arg1 k8s.PipelineRun) (run.Run, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRun", arg0, arg1)
	ret0, _ := ret[0].(run.Run)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRun indicates an expected call of GetRun.
func (mr *MockManagerMockRecorder) GetRun(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRun", reflect.TypeOf((*MockManager)(nil).GetRun), arg0, arg1)
}

// Start mocks base method.
func (m *MockManager) Start(arg0 context.Context, arg1 k8s.PipelineRun, arg2 *cfg.PipelineRunsConfigStruct) (string, string, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Start indicates an expected call of Start.
func (mr *MockManagerMockRecorder) Start(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockManager)(nil).Start), arg0, arg1, arg2)
}

// MockSecretManager is a mock of SecretManager interface.
type MockSecretManager struct {
	ctrl     *gomock.Controller
	recorder *MockSecretManagerMockRecorder
}

// MockSecretManagerMockRecorder is the mock recorder for MockSecretManager.
type MockSecretManagerMockRecorder struct {
	mock *MockSecretManager
}

// NewMockSecretManager creates a new mock instance.
func NewMockSecretManager(ctrl *gomock.Controller) *MockSecretManager {
	mock := &MockSecretManager{ctrl: ctrl}
	mock.recorder = &MockSecretManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSecretManager) EXPECT() *MockSecretManagerMockRecorder {
	return m.recorder
}

// CopyAll mocks base method.
func (m *MockSecretManager) CopyAll(arg0 context.Context, arg1 k8s.PipelineRun) (string, []string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CopyAll", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].([]string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CopyAll indicates an expected call of CopyAll.
func (mr *MockSecretManagerMockRecorder) CopyAll(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CopyAll", reflect.TypeOf((*MockSecretManager)(nil).CopyAll), arg0, arg1)
}
