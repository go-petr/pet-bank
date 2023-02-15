// Code generated by MockGen. DO NOT EDIT.
// Source: http.go

// Package userdelivery is a generated GoMock package.
package userdelivery

import (
	context "context"
	reflect "reflect"
	time "time"

	domain "github.com/go-petr/pet-bank/internal/domain"
	gomock "github.com/golang/mock/gomock"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CheckPassword mocks base method.
func (m *MockService) CheckPassword(ctx context.Context, username, password string) (domain.UserWihtoutPassword, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckPassword", ctx, username, password)
	ret0, _ := ret[0].(domain.UserWihtoutPassword)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckPassword indicates an expected call of CheckPassword.
func (mr *MockServiceMockRecorder) CheckPassword(ctx, username, password interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckPassword", reflect.TypeOf((*MockService)(nil).CheckPassword), ctx, username, password)
}

// Create mocks base method.
func (m *MockService) Create(ctx context.Context, username, password, fullname, email string) (domain.UserWihtoutPassword, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, username, password, fullname, email)
	ret0, _ := ret[0].(domain.UserWihtoutPassword)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockServiceMockRecorder) Create(ctx, username, password, fullname, email interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockService)(nil).Create), ctx, username, password, fullname, email)
}

// MockSessionMaker is a mock of SessionMaker interface.
type MockSessionMaker struct {
	ctrl     *gomock.Controller
	recorder *MockSessionMakerMockRecorder
}

// MockSessionMakerMockRecorder is the mock recorder for MockSessionMaker.
type MockSessionMakerMockRecorder struct {
	mock *MockSessionMaker
}

// NewMockSessionMaker creates a new mock instance.
func NewMockSessionMaker(ctrl *gomock.Controller) *MockSessionMaker {
	mock := &MockSessionMaker{ctrl: ctrl}
	mock.recorder = &MockSessionMakerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSessionMaker) EXPECT() *MockSessionMakerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockSessionMaker) Create(ctx context.Context, arg domain.CreateSessionParams) (string, time.Time, domain.Session, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, arg)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(time.Time)
	ret2, _ := ret[2].(domain.Session)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Create indicates an expected call of Create.
func (mr *MockSessionMakerMockRecorder) Create(ctx, arg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockSessionMaker)(nil).Create), ctx, arg)
}