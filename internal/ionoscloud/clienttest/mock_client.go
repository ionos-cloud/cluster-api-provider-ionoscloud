/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by mockery v2.36.0. DO NOT EDIT.

package client_test

import (
	context "context"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	mock "github.com/stretchr/testify/mock"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

type MockClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockClient) EXPECT() *MockClient_Expecter {
	return &MockClient_Expecter{mock: &_m.Mock}
}

// CheckRequestStatus provides a mock function with given fields: ctx, requestID
func (_m *MockClient) CheckRequestStatus(ctx context.Context, requestID string) (*ionoscloud.RequestStatus, error) {
	ret := _m.Called(ctx, requestID)

	var r0 *ionoscloud.RequestStatus
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*ionoscloud.RequestStatus, error)); ok {
		return rf(ctx, requestID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *ionoscloud.RequestStatus); ok {
		r0 = rf(ctx, requestID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.RequestStatus)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, requestID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_CheckRequestStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckRequestStatus'
type MockClient_CheckRequestStatus_Call struct {
	*mock.Call
}

// CheckRequestStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - requestID string
func (_e *MockClient_Expecter) CheckRequestStatus(ctx interface{}, requestID interface{}) *MockClient_CheckRequestStatus_Call {
	return &MockClient_CheckRequestStatus_Call{Call: _e.mock.On("CheckRequestStatus", ctx, requestID)}
}

func (_c *MockClient_CheckRequestStatus_Call) Run(run func(ctx context.Context, requestID string)) *MockClient_CheckRequestStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockClient_CheckRequestStatus_Call) Return(_a0 *ionoscloud.RequestStatus, _a1 error) *MockClient_CheckRequestStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_CheckRequestStatus_Call) RunAndReturn(run func(context.Context, string) (*ionoscloud.RequestStatus, error)) *MockClient_CheckRequestStatus_Call {
	_c.Call.Return(run)
	return _c
}

// CreateLAN provides a mock function with given fields: ctx, datacenterID, properties
func (_m *MockClient) CreateLAN(ctx context.Context, datacenterID string, properties ionoscloud.LanPropertiesPost) (string, error) {
	ret := _m.Called(ctx, datacenterID, properties)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, ionoscloud.LanPropertiesPost) (string, error)); ok {
		return rf(ctx, datacenterID, properties)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, ionoscloud.LanPropertiesPost) string); ok {
		r0 = rf(ctx, datacenterID, properties)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, ionoscloud.LanPropertiesPost) error); ok {
		r1 = rf(ctx, datacenterID, properties)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_CreateLAN_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateLAN'
type MockClient_CreateLAN_Call struct {
	*mock.Call
}

// CreateLAN is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - properties ionoscloud.LanPropertiesPost
func (_e *MockClient_Expecter) CreateLAN(ctx interface{}, datacenterID interface{}, properties interface{}) *MockClient_CreateLAN_Call {
	return &MockClient_CreateLAN_Call{Call: _e.mock.On("CreateLAN", ctx, datacenterID, properties)}
}

func (_c *MockClient_CreateLAN_Call) Run(run func(ctx context.Context, datacenterID string, properties ionoscloud.LanPropertiesPost)) *MockClient_CreateLAN_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(ionoscloud.LanPropertiesPost))
	})
	return _c
}

func (_c *MockClient_CreateLAN_Call) Return(_a0 string, _a1 error) *MockClient_CreateLAN_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_CreateLAN_Call) RunAndReturn(run func(context.Context, string, ionoscloud.LanPropertiesPost) (string, error)) *MockClient_CreateLAN_Call {
	_c.Call.Return(run)
	return _c
}

// CreateServer provides a mock function with given fields: ctx, datacenterID, properties, entities
func (_m *MockClient) CreateServer(ctx context.Context, datacenterID string, properties ionoscloud.ServerProperties, entities ionoscloud.ServerEntities) (*ionoscloud.Server, string, error) {
	ret := _m.Called(ctx, datacenterID, properties, entities)

	var r0 *ionoscloud.Server
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, ionoscloud.ServerProperties, ionoscloud.ServerEntities) (*ionoscloud.Server, string, error)); ok {
		return rf(ctx, datacenterID, properties, entities)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, ionoscloud.ServerProperties, ionoscloud.ServerEntities) *ionoscloud.Server); ok {
		r0 = rf(ctx, datacenterID, properties, entities)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.Server)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, ionoscloud.ServerProperties, ionoscloud.ServerEntities) string); ok {
		r1 = rf(ctx, datacenterID, properties, entities)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, ionoscloud.ServerProperties, ionoscloud.ServerEntities) error); ok {
		r2 = rf(ctx, datacenterID, properties, entities)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// MockClient_CreateServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateServer'
type MockClient_CreateServer_Call struct {
	*mock.Call
}

// CreateServer is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - properties ionoscloud.ServerProperties
//   - entities ionoscloud.ServerEntities
func (_e *MockClient_Expecter) CreateServer(ctx interface{}, datacenterID interface{}, properties interface{}, entities interface{}) *MockClient_CreateServer_Call {
	return &MockClient_CreateServer_Call{Call: _e.mock.On("CreateServer", ctx, datacenterID, properties, entities)}
}

func (_c *MockClient_CreateServer_Call) Run(run func(ctx context.Context, datacenterID string, properties ionoscloud.ServerProperties, entities ionoscloud.ServerEntities)) *MockClient_CreateServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(ionoscloud.ServerProperties), args[3].(ionoscloud.ServerEntities))
	})
	return _c
}

func (_c *MockClient_CreateServer_Call) Return(_a0 *ionoscloud.Server, _a1 string, _a2 error) *MockClient_CreateServer_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *MockClient_CreateServer_Call) RunAndReturn(run func(context.Context, string, ionoscloud.ServerProperties, ionoscloud.ServerEntities) (*ionoscloud.Server, string, error)) *MockClient_CreateServer_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteIPBlock provides a mock function with given fields: ctx, ipBlockID
func (_m *MockClient) DeleteIPBlock(ctx context.Context, ipBlockID string) (string, error) {
	ret := _m.Called(ctx, ipBlockID)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (string, error)); ok {
		return rf(ctx, ipBlockID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) string); ok {
		r0 = rf(ctx, ipBlockID)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, ipBlockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_DeleteIPBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteIPBlock'
type MockClient_DeleteIPBlock_Call struct {
	*mock.Call
}

// DeleteIPBlock is a helper method to define mock.On call
//   - ctx context.Context
//   - ipBlockID string
func (_e *MockClient_Expecter) DeleteIPBlock(ctx interface{}, ipBlockID interface{}) *MockClient_DeleteIPBlock_Call {
	return &MockClient_DeleteIPBlock_Call{Call: _e.mock.On("DeleteIPBlock", ctx, ipBlockID)}
}

func (_c *MockClient_DeleteIPBlock_Call) Run(run func(ctx context.Context, ipBlockID string)) *MockClient_DeleteIPBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockClient_DeleteIPBlock_Call) Return(requestPath string, err error) *MockClient_DeleteIPBlock_Call {
	_c.Call.Return(requestPath, err)
	return _c
}

func (_c *MockClient_DeleteIPBlock_Call) RunAndReturn(run func(context.Context, string) (string, error)) *MockClient_DeleteIPBlock_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteLAN provides a mock function with given fields: ctx, datacenterID, lanID
func (_m *MockClient) DeleteLAN(ctx context.Context, datacenterID string, lanID string) (string, error) {
	ret := _m.Called(ctx, datacenterID, lanID)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (string, error)); ok {
		return rf(ctx, datacenterID, lanID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, datacenterID, lanID)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, datacenterID, lanID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_DeleteLAN_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteLAN'
type MockClient_DeleteLAN_Call struct {
	*mock.Call
}

// DeleteLAN is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - lanID string
func (_e *MockClient_Expecter) DeleteLAN(ctx interface{}, datacenterID interface{}, lanID interface{}) *MockClient_DeleteLAN_Call {
	return &MockClient_DeleteLAN_Call{Call: _e.mock.On("DeleteLAN", ctx, datacenterID, lanID)}
}

func (_c *MockClient_DeleteLAN_Call) Run(run func(ctx context.Context, datacenterID string, lanID string)) *MockClient_DeleteLAN_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockClient_DeleteLAN_Call) Return(_a0 string, _a1 error) *MockClient_DeleteLAN_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_DeleteLAN_Call) RunAndReturn(run func(context.Context, string, string) (string, error)) *MockClient_DeleteLAN_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteServer provides a mock function with given fields: ctx, datacenterID, serverID
func (_m *MockClient) DeleteServer(ctx context.Context, datacenterID string, serverID string) (string, error) {
	ret := _m.Called(ctx, datacenterID, serverID)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (string, error)); ok {
		return rf(ctx, datacenterID, serverID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, datacenterID, serverID)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, datacenterID, serverID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_DeleteServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteServer'
type MockClient_DeleteServer_Call struct {
	*mock.Call
}

// DeleteServer is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - serverID string
func (_e *MockClient_Expecter) DeleteServer(ctx interface{}, datacenterID interface{}, serverID interface{}) *MockClient_DeleteServer_Call {
	return &MockClient_DeleteServer_Call{Call: _e.mock.On("DeleteServer", ctx, datacenterID, serverID)}
}

func (_c *MockClient_DeleteServer_Call) Run(run func(ctx context.Context, datacenterID string, serverID string)) *MockClient_DeleteServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockClient_DeleteServer_Call) Return(_a0 string, _a1 error) *MockClient_DeleteServer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_DeleteServer_Call) RunAndReturn(run func(context.Context, string, string) (string, error)) *MockClient_DeleteServer_Call {
	_c.Call.Return(run)
	return _c
}

// GetIPBlock provides a mock function with given fields: ctx, ipBlockID
func (_m *MockClient) GetIPBlock(ctx context.Context, ipBlockID string) (*ionoscloud.IpBlock, error) {
	ret := _m.Called(ctx, ipBlockID)

	var r0 *ionoscloud.IpBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*ionoscloud.IpBlock, error)); ok {
		return rf(ctx, ipBlockID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *ionoscloud.IpBlock); ok {
		r0 = rf(ctx, ipBlockID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.IpBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, ipBlockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_GetIPBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetIPBlock'
type MockClient_GetIPBlock_Call struct {
	*mock.Call
}

// GetIPBlock is a helper method to define mock.On call
//   - ctx context.Context
//   - ipBlockID string
func (_e *MockClient_Expecter) GetIPBlock(ctx interface{}, ipBlockID interface{}) *MockClient_GetIPBlock_Call {
	return &MockClient_GetIPBlock_Call{Call: _e.mock.On("GetIPBlock", ctx, ipBlockID)}
}

func (_c *MockClient_GetIPBlock_Call) Run(run func(ctx context.Context, ipBlockID string)) *MockClient_GetIPBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockClient_GetIPBlock_Call) Return(_a0 *ionoscloud.IpBlock, _a1 error) *MockClient_GetIPBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_GetIPBlock_Call) RunAndReturn(run func(context.Context, string) (*ionoscloud.IpBlock, error)) *MockClient_GetIPBlock_Call {
	_c.Call.Return(run)
	return _c
}

// GetRequests provides a mock function with given fields: ctx, method, path
func (_m *MockClient) GetRequests(ctx context.Context, method string, path string) ([]ionoscloud.Request, error) {
	ret := _m.Called(ctx, method, path)

	var r0 []ionoscloud.Request
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]ionoscloud.Request, error)); ok {
		return rf(ctx, method, path)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []ionoscloud.Request); ok {
		r0 = rf(ctx, method, path)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ionoscloud.Request)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, method, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_GetRequests_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRequests'
type MockClient_GetRequests_Call struct {
	*mock.Call
}

// GetRequests is a helper method to define mock.On call
//   - ctx context.Context
//   - method string
//   - path string
func (_e *MockClient_Expecter) GetRequests(ctx interface{}, method interface{}, path interface{}) *MockClient_GetRequests_Call {
	return &MockClient_GetRequests_Call{Call: _e.mock.On("GetRequests", ctx, method, path)}
}

func (_c *MockClient_GetRequests_Call) Run(run func(ctx context.Context, method string, path string)) *MockClient_GetRequests_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockClient_GetRequests_Call) Return(_a0 []ionoscloud.Request, _a1 error) *MockClient_GetRequests_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_GetRequests_Call) RunAndReturn(run func(context.Context, string, string) ([]ionoscloud.Request, error)) *MockClient_GetRequests_Call {
	_c.Call.Return(run)
	return _c
}

// GetServer provides a mock function with given fields: ctx, datacenterID, serverID
func (_m *MockClient) GetServer(ctx context.Context, datacenterID string, serverID string) (*ionoscloud.Server, error) {
	ret := _m.Called(ctx, datacenterID, serverID)

	var r0 *ionoscloud.Server
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*ionoscloud.Server, error)); ok {
		return rf(ctx, datacenterID, serverID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *ionoscloud.Server); ok {
		r0 = rf(ctx, datacenterID, serverID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.Server)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, datacenterID, serverID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_GetServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetServer'
type MockClient_GetServer_Call struct {
	*mock.Call
}

// GetServer is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - serverID string
func (_e *MockClient_Expecter) GetServer(ctx interface{}, datacenterID interface{}, serverID interface{}) *MockClient_GetServer_Call {
	return &MockClient_GetServer_Call{Call: _e.mock.On("GetServer", ctx, datacenterID, serverID)}
}

func (_c *MockClient_GetServer_Call) Run(run func(ctx context.Context, datacenterID string, serverID string)) *MockClient_GetServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockClient_GetServer_Call) Return(_a0 *ionoscloud.Server, _a1 error) *MockClient_GetServer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_GetServer_Call) RunAndReturn(run func(context.Context, string, string) (*ionoscloud.Server, error)) *MockClient_GetServer_Call {
	_c.Call.Return(run)
	return _c
}

// ListIPBlocks provides a mock function with given fields: ctx
func (_m *MockClient) ListIPBlocks(ctx context.Context) (*ionoscloud.IpBlocks, error) {
	ret := _m.Called(ctx)

	var r0 *ionoscloud.IpBlocks
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*ionoscloud.IpBlocks, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *ionoscloud.IpBlocks); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.IpBlocks)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_ListIPBlocks_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListIPBlocks'
type MockClient_ListIPBlocks_Call struct {
	*mock.Call
}

// ListIPBlocks is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockClient_Expecter) ListIPBlocks(ctx interface{}) *MockClient_ListIPBlocks_Call {
	return &MockClient_ListIPBlocks_Call{Call: _e.mock.On("ListIPBlocks", ctx)}
}

func (_c *MockClient_ListIPBlocks_Call) Run(run func(ctx context.Context)) *MockClient_ListIPBlocks_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockClient_ListIPBlocks_Call) Return(_a0 *ionoscloud.IpBlocks, _a1 error) *MockClient_ListIPBlocks_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_ListIPBlocks_Call) RunAndReturn(run func(context.Context) (*ionoscloud.IpBlocks, error)) *MockClient_ListIPBlocks_Call {
	_c.Call.Return(run)
	return _c
}

// ListLANs provides a mock function with given fields: ctx, datacenterID
func (_m *MockClient) ListLANs(ctx context.Context, datacenterID string) (*ionoscloud.Lans, error) {
	ret := _m.Called(ctx, datacenterID)

	var r0 *ionoscloud.Lans
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*ionoscloud.Lans, error)); ok {
		return rf(ctx, datacenterID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *ionoscloud.Lans); ok {
		r0 = rf(ctx, datacenterID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.Lans)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, datacenterID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_ListLANs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListLANs'
type MockClient_ListLANs_Call struct {
	*mock.Call
}

// ListLANs is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
func (_e *MockClient_Expecter) ListLANs(ctx interface{}, datacenterID interface{}) *MockClient_ListLANs_Call {
	return &MockClient_ListLANs_Call{Call: _e.mock.On("ListLANs", ctx, datacenterID)}
}

func (_c *MockClient_ListLANs_Call) Run(run func(ctx context.Context, datacenterID string)) *MockClient_ListLANs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockClient_ListLANs_Call) Return(_a0 *ionoscloud.Lans, _a1 error) *MockClient_ListLANs_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_ListLANs_Call) RunAndReturn(run func(context.Context, string) (*ionoscloud.Lans, error)) *MockClient_ListLANs_Call {
	_c.Call.Return(run)
	return _c
}

// ListServers provides a mock function with given fields: ctx, datacenterID
func (_m *MockClient) ListServers(ctx context.Context, datacenterID string) (*ionoscloud.Servers, error) {
	ret := _m.Called(ctx, datacenterID)

	var r0 *ionoscloud.Servers
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*ionoscloud.Servers, error)); ok {
		return rf(ctx, datacenterID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *ionoscloud.Servers); ok {
		r0 = rf(ctx, datacenterID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ionoscloud.Servers)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, datacenterID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_ListServers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListServers'
type MockClient_ListServers_Call struct {
	*mock.Call
}

// ListServers is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
func (_e *MockClient_Expecter) ListServers(ctx interface{}, datacenterID interface{}) *MockClient_ListServers_Call {
	return &MockClient_ListServers_Call{Call: _e.mock.On("ListServers", ctx, datacenterID)}
}

func (_c *MockClient_ListServers_Call) Run(run func(ctx context.Context, datacenterID string)) *MockClient_ListServers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockClient_ListServers_Call) Return(_a0 *ionoscloud.Servers, _a1 error) *MockClient_ListServers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_ListServers_Call) RunAndReturn(run func(context.Context, string) (*ionoscloud.Servers, error)) *MockClient_ListServers_Call {
	_c.Call.Return(run)
	return _c
}

// PatchLAN provides a mock function with given fields: ctx, datacenterID, lanID, properties
func (_m *MockClient) PatchLAN(ctx context.Context, datacenterID string, lanID string, properties ionoscloud.LanProperties) (string, error) {
	ret := _m.Called(ctx, datacenterID, lanID, properties)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, ionoscloud.LanProperties) (string, error)); ok {
		return rf(ctx, datacenterID, lanID, properties)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, ionoscloud.LanProperties) string); ok {
		r0 = rf(ctx, datacenterID, lanID, properties)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, ionoscloud.LanProperties) error); ok {
		r1 = rf(ctx, datacenterID, lanID, properties)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_PatchLAN_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PatchLAN'
type MockClient_PatchLAN_Call struct {
	*mock.Call
}

// PatchLAN is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - lanID string
//   - properties ionoscloud.LanProperties
func (_e *MockClient_Expecter) PatchLAN(ctx interface{}, datacenterID interface{}, lanID interface{}, properties interface{}) *MockClient_PatchLAN_Call {
	return &MockClient_PatchLAN_Call{Call: _e.mock.On("PatchLAN", ctx, datacenterID, lanID, properties)}
}

func (_c *MockClient_PatchLAN_Call) Run(run func(ctx context.Context, datacenterID string, lanID string, properties ionoscloud.LanProperties)) *MockClient_PatchLAN_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(ionoscloud.LanProperties))
	})
	return _c
}

func (_c *MockClient_PatchLAN_Call) Return(_a0 string, _a1 error) *MockClient_PatchLAN_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_PatchLAN_Call) RunAndReturn(run func(context.Context, string, string, ionoscloud.LanProperties) (string, error)) *MockClient_PatchLAN_Call {
	_c.Call.Return(run)
	return _c
}

// PatchNIC provides a mock function with given fields: ctx, datacenterID, serverID, nicID, properties
func (_m *MockClient) PatchNIC(ctx context.Context, datacenterID string, serverID string, nicID string, properties ionoscloud.NicProperties) (string, error) {
	ret := _m.Called(ctx, datacenterID, serverID, nicID, properties)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, ionoscloud.NicProperties) (string, error)); ok {
		return rf(ctx, datacenterID, serverID, nicID, properties)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, ionoscloud.NicProperties) string); ok {
		r0 = rf(ctx, datacenterID, serverID, nicID, properties)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, ionoscloud.NicProperties) error); ok {
		r1 = rf(ctx, datacenterID, serverID, nicID, properties)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_PatchNIC_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PatchNIC'
type MockClient_PatchNIC_Call struct {
	*mock.Call
}

// PatchNIC is a helper method to define mock.On call
//   - ctx context.Context
//   - datacenterID string
//   - serverID string
//   - nicID string
//   - properties ionoscloud.NicProperties
func (_e *MockClient_Expecter) PatchNIC(ctx interface{}, datacenterID interface{}, serverID interface{}, nicID interface{}, properties interface{}) *MockClient_PatchNIC_Call {
	return &MockClient_PatchNIC_Call{Call: _e.mock.On("PatchNIC", ctx, datacenterID, serverID, nicID, properties)}
}

func (_c *MockClient_PatchNIC_Call) Run(run func(ctx context.Context, datacenterID string, serverID string, nicID string, properties ionoscloud.NicProperties)) *MockClient_PatchNIC_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string), args[4].(ionoscloud.NicProperties))
	})
	return _c
}

func (_c *MockClient_PatchNIC_Call) Return(_a0 string, _a1 error) *MockClient_PatchNIC_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClient_PatchNIC_Call) RunAndReturn(run func(context.Context, string, string, string, ionoscloud.NicProperties) (string, error)) *MockClient_PatchNIC_Call {
	_c.Call.Return(run)
	return _c
}

// ReserveIPBlock provides a mock function with given fields: ctx, name, location, size
func (_m *MockClient) ReserveIPBlock(ctx context.Context, name string, location string, size int32) (string, error) {
	ret := _m.Called(ctx, name, location, size)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int32) (string, error)); ok {
		return rf(ctx, name, location, size)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int32) string); ok {
		r0 = rf(ctx, name, location, size)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, int32) error); ok {
		r1 = rf(ctx, name, location, size)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient_ReserveIPBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReserveIPBlock'
type MockClient_ReserveIPBlock_Call struct {
	*mock.Call
}

// ReserveIPBlock is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - location string
//   - size int32
func (_e *MockClient_Expecter) ReserveIPBlock(ctx interface{}, name interface{}, location interface{}, size interface{}) *MockClient_ReserveIPBlock_Call {
	return &MockClient_ReserveIPBlock_Call{Call: _e.mock.On("ReserveIPBlock", ctx, name, location, size)}
}

func (_c *MockClient_ReserveIPBlock_Call) Run(run func(ctx context.Context, name string, location string, size int32)) *MockClient_ReserveIPBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(int32))
	})
	return _c
}

func (_c *MockClient_ReserveIPBlock_Call) Return(requestPath string, err error) *MockClient_ReserveIPBlock_Call {
	_c.Call.Return(requestPath, err)
	return _c
}

func (_c *MockClient_ReserveIPBlock_Call) RunAndReturn(run func(context.Context, string, string, int32) (string, error)) *MockClient_ReserveIPBlock_Call {
	_c.Call.Return(run)
	return _c
}

// WaitForRequest provides a mock function with given fields: ctx, requestURL
func (_m *MockClient) WaitForRequest(ctx context.Context, requestURL string) error {
	ret := _m.Called(ctx, requestURL)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, requestURL)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockClient_WaitForRequest_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WaitForRequest'
type MockClient_WaitForRequest_Call struct {
	*mock.Call
}

// WaitForRequest is a helper method to define mock.On call
//   - ctx context.Context
//   - requestURL string
func (_e *MockClient_Expecter) WaitForRequest(ctx interface{}, requestURL interface{}) *MockClient_WaitForRequest_Call {
	return &MockClient_WaitForRequest_Call{Call: _e.mock.On("WaitForRequest", ctx, requestURL)}
}

func (_c *MockClient_WaitForRequest_Call) Run(run func(ctx context.Context, requestURL string)) *MockClient_WaitForRequest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockClient_WaitForRequest_Call) Return(_a0 error) *MockClient_WaitForRequest_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClient_WaitForRequest_Call) RunAndReturn(run func(context.Context, string) error) *MockClient_WaitForRequest_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockClient creates a new instance of MockClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClient {
	mock := &MockClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
