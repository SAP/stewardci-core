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
// Source: github.com/SAP/stewardci-core/pkg/k8s/secrets (interfaces: SecretProvider,SecretHelper)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	reflect "reflect"
)

// MockSecretProvider is a mock of SecretProvider interface
type MockSecretProvider struct {
	ctrl     *gomock.Controller
	recorder *MockSecretProviderMockRecorder
}

// MockSecretProviderMockRecorder is the mock recorder for MockSecretProvider
type MockSecretProviderMockRecorder struct {
	mock *MockSecretProvider
}

// NewMockSecretProvider creates a new mock instance
func NewMockSecretProvider(ctrl *gomock.Controller) *MockSecretProvider {
	mock := &MockSecretProvider{ctrl: ctrl}
	mock.recorder = &MockSecretProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSecretProvider) EXPECT() *MockSecretProviderMockRecorder {
	return m.recorder
}

// GetSecret mocks base method
func (m *MockSecretProvider) GetSecret(arg0 string) (*v1.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecret", arg0)
	ret0, _ := ret[0].(*v1.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecret indicates an expected call of GetSecret
func (mr *MockSecretProviderMockRecorder) GetSecret(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecret", reflect.TypeOf((*MockSecretProvider)(nil).GetSecret), arg0)
}

// MockSecretHelper is a mock of SecretHelper interface
type MockSecretHelper struct {
	ctrl     *gomock.Controller
	recorder *MockSecretHelperMockRecorder
}

// MockSecretHelperMockRecorder is the mock recorder for MockSecretHelper
type MockSecretHelperMockRecorder struct {
	mock *MockSecretHelper
}

// NewMockSecretHelper creates a new mock instance
func NewMockSecretHelper(ctrl *gomock.Controller) *MockSecretHelper {
	mock := &MockSecretHelper{ctrl: ctrl}
	mock.recorder = &MockSecretHelperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSecretHelper) EXPECT() *MockSecretHelperMockRecorder {
	return m.recorder
}

// CopySecrets mocks base method
func (m *MockSecretHelper) CopySecrets(arg0 []string, arg1 func(*v1.Secret) bool, arg2 ...func(*v1.Secret) *v1.Secret) ([]string, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CopySecrets", varargs...)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CopySecrets indicates an expected call of CopySecrets
func (mr *MockSecretHelperMockRecorder) CopySecrets(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CopySecrets", reflect.TypeOf((*MockSecretHelper)(nil).CopySecrets), varargs...)
}

// CreateSecret mocks base method
func (m *MockSecretHelper) CreateSecret(arg0 *v1.Secret) (*v1.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSecret", arg0)
	ret0, _ := ret[0].(*v1.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSecret indicates an expected call of CreateSecret
func (mr *MockSecretHelperMockRecorder) CreateSecret(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSecret", reflect.TypeOf((*MockSecretHelper)(nil).CreateSecret), arg0)
}

// IsNotFound mocks base method
func (m *MockSecretHelper) IsNotFound(arg0 error) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsNotFound", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsNotFound indicates an expected call of IsNotFound
func (mr *MockSecretHelperMockRecorder) IsNotFound(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsNotFound", reflect.TypeOf((*MockSecretHelper)(nil).IsNotFound), arg0)
}
