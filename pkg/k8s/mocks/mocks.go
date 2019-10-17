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
// Source: github.com/SAP/stewardci-core/pkg/k8s (interfaces: ClientFactory,NamespaceManager,PipelineRun,PipelineRunFetcher,SecretProvider,TenantFetcher)

// Package mocks is a generated GoMock package.
package mocks

import (
	v1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	v1alpha10 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	externalversions "github.com/SAP/stewardci-core/pkg/client/informers/externalversions"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	v1alpha11 "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/typed/pipeline/v1alpha1"
	externalversions0 "github.com/SAP/stewardci-core/pkg/tektonclient/informers/externalversions"
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	v10 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	reflect "reflect"
)

// MockClientFactory is a mock of ClientFactory interface
type MockClientFactory struct {
	ctrl     *gomock.Controller
	recorder *MockClientFactoryMockRecorder
}

// MockClientFactoryMockRecorder is the mock recorder for MockClientFactory
type MockClientFactoryMockRecorder struct {
	mock *MockClientFactory
}

// NewMockClientFactory creates a new mock instance
func NewMockClientFactory(ctrl *gomock.Controller) *MockClientFactory {
	mock := &MockClientFactory{ctrl: ctrl}
	mock.recorder = &MockClientFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClientFactory) EXPECT() *MockClientFactoryMockRecorder {
	return m.recorder
}

// CoreV1 mocks base method
func (m *MockClientFactory) CoreV1() v10.CoreV1Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CoreV1")
	ret0, _ := ret[0].(v10.CoreV1Interface)
	return ret0
}

// CoreV1 indicates an expected call of CoreV1
func (mr *MockClientFactoryMockRecorder) CoreV1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CoreV1", reflect.TypeOf((*MockClientFactory)(nil).CoreV1))
}

// RbacV1beta1 mocks base method
func (m *MockClientFactory) RbacV1beta1() v1beta1.RbacV1beta1Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RbacV1beta1")
	ret0, _ := ret[0].(v1beta1.RbacV1beta1Interface)
	return ret0
}

// RbacV1beta1 indicates an expected call of RbacV1beta1
func (mr *MockClientFactoryMockRecorder) RbacV1beta1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RbacV1beta1", reflect.TypeOf((*MockClientFactory)(nil).RbacV1beta1))
}

// StewardInformerFactory mocks base method
func (m *MockClientFactory) StewardInformerFactory() externalversions.SharedInformerFactory {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StewardInformerFactory")
	ret0, _ := ret[0].(externalversions.SharedInformerFactory)
	return ret0
}

// StewardInformerFactory indicates an expected call of StewardInformerFactory
func (mr *MockClientFactoryMockRecorder) StewardInformerFactory() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StewardInformerFactory", reflect.TypeOf((*MockClientFactory)(nil).StewardInformerFactory))
}

// StewardV1alpha1 mocks base method
func (m *MockClientFactory) StewardV1alpha1() v1alpha10.StewardV1alpha1Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StewardV1alpha1")
	ret0, _ := ret[0].(v1alpha10.StewardV1alpha1Interface)
	return ret0
}

// StewardV1alpha1 indicates an expected call of StewardV1alpha1
func (mr *MockClientFactoryMockRecorder) StewardV1alpha1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StewardV1alpha1", reflect.TypeOf((*MockClientFactory)(nil).StewardV1alpha1))
}

// TektonInformerFactory mocks base method
func (m *MockClientFactory) TektonInformerFactory() externalversions0.SharedInformerFactory {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TektonInformerFactory")
	ret0, _ := ret[0].(externalversions0.SharedInformerFactory)
	return ret0
}

// TektonInformerFactory indicates an expected call of TektonInformerFactory
func (mr *MockClientFactoryMockRecorder) TektonInformerFactory() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TektonInformerFactory", reflect.TypeOf((*MockClientFactory)(nil).TektonInformerFactory))
}

// TektonV1alpha1 mocks base method
func (m *MockClientFactory) TektonV1alpha1() v1alpha11.TektonV1alpha1Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TektonV1alpha1")
	ret0, _ := ret[0].(v1alpha11.TektonV1alpha1Interface)
	return ret0
}

// TektonV1alpha1 indicates an expected call of TektonV1alpha1
func (mr *MockClientFactoryMockRecorder) TektonV1alpha1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TektonV1alpha1", reflect.TypeOf((*MockClientFactory)(nil).TektonV1alpha1))
}

// MockNamespaceManager is a mock of NamespaceManager interface
type MockNamespaceManager struct {
	ctrl     *gomock.Controller
	recorder *MockNamespaceManagerMockRecorder
}

// MockNamespaceManagerMockRecorder is the mock recorder for MockNamespaceManager
type MockNamespaceManagerMockRecorder struct {
	mock *MockNamespaceManager
}

// NewMockNamespaceManager creates a new mock instance
func NewMockNamespaceManager(ctrl *gomock.Controller) *MockNamespaceManager {
	mock := &MockNamespaceManager{ctrl: ctrl}
	mock.recorder = &MockNamespaceManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockNamespaceManager) EXPECT() *MockNamespaceManagerMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockNamespaceManager) Create(arg0 string, arg1 map[string]string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockNamespaceManagerMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockNamespaceManager)(nil).Create), arg0, arg1)
}

// Delete mocks base method
func (m *MockNamespaceManager) Delete(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockNamespaceManagerMockRecorder) Delete(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNamespaceManager)(nil).Delete), arg0)
}

// MockPipelineRun is a mock of PipelineRun interface
type MockPipelineRun struct {
	ctrl     *gomock.Controller
	recorder *MockPipelineRunMockRecorder
}

// MockPipelineRunMockRecorder is the mock recorder for MockPipelineRun
type MockPipelineRunMockRecorder struct {
	mock *MockPipelineRun
}

// NewMockPipelineRun creates a new mock instance
func NewMockPipelineRun(ctrl *gomock.Controller) *MockPipelineRun {
	mock := &MockPipelineRun{ctrl: ctrl}
	mock.recorder = &MockPipelineRunMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPipelineRun) EXPECT() *MockPipelineRunMockRecorder {
	return m.recorder
}

// AddFinalizer mocks base method
func (m *MockPipelineRun) AddFinalizer() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddFinalizer")
	ret0, _ := ret[0].(error)
	return ret0
}

// AddFinalizer indicates an expected call of AddFinalizer
func (mr *MockPipelineRunMockRecorder) AddFinalizer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddFinalizer", reflect.TypeOf((*MockPipelineRun)(nil).AddFinalizer))
}

// DeleteFinalizerIfExists mocks base method
func (m *MockPipelineRun) DeleteFinalizerIfExists() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFinalizerIfExists")
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFinalizerIfExists indicates an expected call of DeleteFinalizerIfExists
func (mr *MockPipelineRunMockRecorder) DeleteFinalizerIfExists() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFinalizerIfExists", reflect.TypeOf((*MockPipelineRun)(nil).DeleteFinalizerIfExists))
}

// FinishState mocks base method
func (m *MockPipelineRun) FinishState() (*v1alpha1.StateItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FinishState")
	ret0, _ := ret[0].(*v1alpha1.StateItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FinishState indicates an expected call of FinishState
func (mr *MockPipelineRunMockRecorder) FinishState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FinishState", reflect.TypeOf((*MockPipelineRun)(nil).FinishState))
}

// GetKey mocks base method
func (m *MockPipelineRun) GetKey() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKey")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetKey indicates an expected call of GetKey
func (mr *MockPipelineRunMockRecorder) GetKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKey", reflect.TypeOf((*MockPipelineRun)(nil).GetKey))
}

// GetName mocks base method
func (m *MockPipelineRun) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName
func (mr *MockPipelineRunMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockPipelineRun)(nil).GetName))
}

// GetNamespace mocks base method
func (m *MockPipelineRun) GetNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetNamespace indicates an expected call of GetNamespace
func (mr *MockPipelineRunMockRecorder) GetNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNamespace", reflect.TypeOf((*MockPipelineRun)(nil).GetNamespace))
}

// GetRunNamespace mocks base method
func (m *MockPipelineRun) GetRunNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRunNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRunNamespace indicates an expected call of GetRunNamespace
func (mr *MockPipelineRunMockRecorder) GetRunNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRunNamespace", reflect.TypeOf((*MockPipelineRun)(nil).GetRunNamespace))
}

// GetSpec mocks base method
func (m *MockPipelineRun) GetSpec() *v1alpha1.PipelineSpec {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSpec")
	ret0, _ := ret[0].(*v1alpha1.PipelineSpec)
	return ret0
}

// GetSpec indicates an expected call of GetSpec
func (mr *MockPipelineRunMockRecorder) GetSpec() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSpec", reflect.TypeOf((*MockPipelineRun)(nil).GetSpec))
}

// GetStatus mocks base method
func (m *MockPipelineRun) GetStatus() *v1alpha1.PipelineStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStatus")
	ret0, _ := ret[0].(*v1alpha1.PipelineStatus)
	return ret0
}

// GetStatus indicates an expected call of GetStatus
func (mr *MockPipelineRunMockRecorder) GetStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStatus", reflect.TypeOf((*MockPipelineRun)(nil).GetStatus))
}

// HasDeletionTimestamp mocks base method
func (m *MockPipelineRun) HasDeletionTimestamp() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasDeletionTimestamp")
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasDeletionTimestamp indicates an expected call of HasDeletionTimestamp
func (mr *MockPipelineRunMockRecorder) HasDeletionTimestamp() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasDeletionTimestamp", reflect.TypeOf((*MockPipelineRun)(nil).HasDeletionTimestamp))
}

// StoreErrorAsMessage mocks base method
func (m *MockPipelineRun) StoreErrorAsMessage(arg0 error, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreErrorAsMessage", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreErrorAsMessage indicates an expected call of StoreErrorAsMessage
func (mr *MockPipelineRunMockRecorder) StoreErrorAsMessage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreErrorAsMessage", reflect.TypeOf((*MockPipelineRun)(nil).StoreErrorAsMessage), arg0, arg1)
}

// UpdateContainer mocks base method
func (m *MockPipelineRun) UpdateContainer(arg0 *v1.ContainerState) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateContainer", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateContainer indicates an expected call of UpdateContainer
func (mr *MockPipelineRunMockRecorder) UpdateContainer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateContainer", reflect.TypeOf((*MockPipelineRun)(nil).UpdateContainer), arg0)
}

// UpdateLog mocks base method
func (m *MockPipelineRun) UpdateLog() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateLog")
}

// UpdateLog indicates an expected call of UpdateLog
func (mr *MockPipelineRunMockRecorder) UpdateLog() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateLog", reflect.TypeOf((*MockPipelineRun)(nil).UpdateLog))
}

// UpdateMessage mocks base method
func (m *MockPipelineRun) UpdateMessage(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMessage", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateMessage indicates an expected call of UpdateMessage
func (mr *MockPipelineRunMockRecorder) UpdateMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMessage", reflect.TypeOf((*MockPipelineRun)(nil).UpdateMessage), arg0)
}

// UpdateResult mocks base method
func (m *MockPipelineRun) UpdateResult(arg0 v1alpha1.Result) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateResult", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateResult indicates an expected call of UpdateResult
func (mr *MockPipelineRunMockRecorder) UpdateResult(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateResult", reflect.TypeOf((*MockPipelineRun)(nil).UpdateResult), arg0)
}

// UpdateRunNamespace mocks base method
func (m *MockPipelineRun) UpdateRunNamespace(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRunNamespace", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateRunNamespace indicates an expected call of UpdateRunNamespace
func (mr *MockPipelineRunMockRecorder) UpdateRunNamespace(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRunNamespace", reflect.TypeOf((*MockPipelineRun)(nil).UpdateRunNamespace), arg0)
}

// UpdateState mocks base method
func (m *MockPipelineRun) UpdateState(arg0 v1alpha1.State) (*v1alpha1.StateItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateState", arg0)
	ret0, _ := ret[0].(*v1alpha1.StateItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateState indicates an expected call of UpdateState
func (mr *MockPipelineRunMockRecorder) UpdateState(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateState", reflect.TypeOf((*MockPipelineRun)(nil).UpdateState), arg0)
}

// MockPipelineRunFetcher is a mock of PipelineRunFetcher interface
type MockPipelineRunFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockPipelineRunFetcherMockRecorder
}

// MockPipelineRunFetcherMockRecorder is the mock recorder for MockPipelineRunFetcher
type MockPipelineRunFetcherMockRecorder struct {
	mock *MockPipelineRunFetcher
}

// NewMockPipelineRunFetcher creates a new mock instance
func NewMockPipelineRunFetcher(ctrl *gomock.Controller) *MockPipelineRunFetcher {
	mock := &MockPipelineRunFetcher{ctrl: ctrl}
	mock.recorder = &MockPipelineRunFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPipelineRunFetcher) EXPECT() *MockPipelineRunFetcherMockRecorder {
	return m.recorder
}

// ByKey mocks base method
func (m *MockPipelineRunFetcher) ByKey(arg0 string) (k8s.PipelineRun, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ByKey", arg0)
	ret0, _ := ret[0].(k8s.PipelineRun)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ByKey indicates an expected call of ByKey
func (mr *MockPipelineRunFetcherMockRecorder) ByKey(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ByKey", reflect.TypeOf((*MockPipelineRunFetcher)(nil).ByKey), arg0)
}

// ByName mocks base method
func (m *MockPipelineRunFetcher) ByName(arg0, arg1 string) (k8s.PipelineRun, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ByName", arg0, arg1)
	ret0, _ := ret[0].(k8s.PipelineRun)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ByName indicates an expected call of ByName
func (mr *MockPipelineRunFetcherMockRecorder) ByName(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ByName", reflect.TypeOf((*MockPipelineRunFetcher)(nil).ByName), arg0, arg1)
}

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

// MockTenantFetcher is a mock of TenantFetcher interface
type MockTenantFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockTenantFetcherMockRecorder
}

// MockTenantFetcherMockRecorder is the mock recorder for MockTenantFetcher
type MockTenantFetcherMockRecorder struct {
	mock *MockTenantFetcher
}

// NewMockTenantFetcher creates a new mock instance
func NewMockTenantFetcher(ctrl *gomock.Controller) *MockTenantFetcher {
	mock := &MockTenantFetcher{ctrl: ctrl}
	mock.recorder = &MockTenantFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTenantFetcher) EXPECT() *MockTenantFetcherMockRecorder {
	return m.recorder
}

// ByKey mocks base method
func (m *MockTenantFetcher) ByKey(arg0 string) (*v1alpha1.Tenant, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ByKey", arg0)
	ret0, _ := ret[0].(*v1alpha1.Tenant)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ByKey indicates an expected call of ByKey
func (mr *MockTenantFetcherMockRecorder) ByKey(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ByKey", reflect.TypeOf((*MockTenantFetcher)(nil).ByKey), arg0)
}
