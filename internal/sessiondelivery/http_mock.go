// Code generated by MockGen. DO NOT EDIT.
// Source: http.go

// Package sessiondelivery is a generated GoMock package.
package sessiondelivery

import (
	context "context"
	reflect "reflect"
	time "time"

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

// RenewAccessToken mocks base method.
func (m *MockService) RenewAccessToken(ctx context.Context, refreshToken string) (string, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RenewAccessToken", ctx, refreshToken)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(time.Time)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// RenewAccessToken indicates an expected call of RenewAccessToken.
func (mr *MockServiceMockRecorder) RenewAccessToken(ctx, refreshToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenewAccessToken", reflect.TypeOf((*MockService)(nil).RenewAccessToken), ctx, refreshToken)
}
