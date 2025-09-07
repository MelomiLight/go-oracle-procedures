package service

import (
	"context"
	"errors"
	"oracle-golang/internal/model/request"
	"oracle-golang/internal/model/response"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CallProcedure(ctx context.Context, name string, params []request.ProcedureParam) (map[string]any, error) {
	args := m.Called(ctx, name, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockRepository) GetProcedureInfo(ctx context.Context, procedureName string) ([]map[string]any, error) {
	args := m.Called(ctx, procedureName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]any), args.Error(1)
}

func TestNewProcedureService(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewProcedureService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

func TestProcedureService_CallProcedure(t *testing.T) {
	tests := []struct {
		name           string
		request        request.CallProcedureRequest
		setupMock      func(*MockRepository)
		expectedResult response.CallProcedureResponse
		expectedError  error
	}{
		{
			name: "successful procedure call",
			request: request.CallProcedureRequest{
				Name: "test_procedure",
				Params: []request.ProcedureParam{
					{Name: "param1", Value: "value1", Type: "IN", Direction: "IN"},
					{Name: "param2", Value: 123, Type: "IN", Direction: "IN"},
				},
			},
			setupMock: func(mockRepo *MockRepository) {
				expectedResult := map[string]any{
					"result_col1": "success",
					"result_col2": 456,
				}
				mockRepo.On("CallProcedure",
					mock.Anything, // Use mock.Anything for context
					"test_procedure",
					[]request.ProcedureParam{
						{Name: "param1", Value: "value1", Type: "IN", Direction: "IN"},
						{Name: "param2", Value: 123, Type: "IN", Direction: "IN"},
					}).Return(expectedResult, nil)
			},
			expectedResult: response.CallProcedureResponse{
				"result_col1": "success",
				"result_col2": 456,
			},
			expectedError: nil,
		},
		{
			name: "procedure call with empty parameters",
			request: request.CallProcedureRequest{
				Name:   "simple_procedure",
				Params: []request.ProcedureParam{},
			},
			setupMock: func(mockRepo *MockRepository) {
				expectedResult := map[string]any{
					"status": "completed",
				}
				mockRepo.On("CallProcedure",
					mock.Anything,
					"simple_procedure",
					[]request.ProcedureParam{}).Return(expectedResult, nil)
			},
			expectedResult: response.CallProcedureResponse{
				"status": "completed",
			},
			expectedError: nil,
		},
		{
			name: "repository returns error",
			request: request.CallProcedureRequest{
				Name: "error_procedure",
				Params: []request.ProcedureParam{
					{Name: "param1", Value: "value1", Type: "IN", Direction: "IN"},
				},
			},
			setupMock: func(mockRepo *MockRepository) {
				mockRepo.On("CallProcedure",
					mock.Anything,
					"error_procedure",
					[]request.ProcedureParam{
						{Name: "param1", Value: "value1", Type: "IN", Direction: "IN"},
					}).Return(nil, errors.New("database connection error"))
			},
			expectedResult: nil,
			expectedError:  errors.New("database connection error"),
		},
		{
			name: "procedure call with mixed parameter types",
			request: request.CallProcedureRequest{
				Name: "complex_procedure",
				Params: []request.ProcedureParam{
					{Name: "string_param", Value: "test", Type: "IN", Direction: "IN"},
					{Name: "int_param", Value: 42, Type: "IN", Direction: "IN"},
					{Name: "float_param", Value: 3.14, Type: "IN", Direction: "IN"},
					{Name: "bool_param", Value: true, Type: "OUT", Direction: "OUT"},
				},
			},
			setupMock: func(mockRepo *MockRepository) {
				expectedResult := map[string]any{
					"processed_count": 5,
					"success":         true,
				}
				mockRepo.On("CallProcedure",
					mock.Anything,
					"complex_procedure",
					[]request.ProcedureParam{
						{Name: "string_param", Value: "test", Type: "IN", Direction: "IN"},
						{Name: "int_param", Value: 42, Type: "IN", Direction: "IN"},
						{Name: "float_param", Value: 3.14, Type: "IN", Direction: "IN"},
						{Name: "bool_param", Value: true, Type: "OUT", Direction: "OUT"},
					}).Return(expectedResult, nil)
			},
			expectedResult: response.CallProcedureResponse{
				"processed_count": 5,
				"success":         true,
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewProcedureService(mockRepo)

			// Execute
			result, err := service.CallProcedure(context.Background(), tt.request)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestProcedureService_GetProcedureInfo(t *testing.T) {
	tests := []struct {
		name           string
		procedureName  string
		setupMock      func(*MockRepository)
		expectedResult response.GetProcedureInfoResponse
		expectedError  error
	}{
		{
			name:          "successful procedure info retrieval",
			procedureName: "test_procedure",
			setupMock: func(mockRepo *MockRepository) {
				expectedResult := []map[string]any{
					{
						"ARGUMENT_NAME": "param1",
						"DATA_TYPE":     "VARCHAR2",
						"IN_OUT":        "IN",
						"POSITION":      1,
					},
					{
						"ARGUMENT_NAME": "param2",
						"DATA_TYPE":     "NUMBER",
						"IN_OUT":        "OUT",
						"POSITION":      2,
					},
				}
				mockRepo.On("GetProcedureInfo",
					mock.Anything,
					"test_procedure").Return(expectedResult, nil)
			},
			expectedResult: response.GetProcedureInfoResponse{
				{
					"ARGUMENT_NAME": "param1",
					"DATA_TYPE":     "VARCHAR2",
					"IN_OUT":        "IN",
					"POSITION":      1,
				},
				{
					"ARGUMENT_NAME": "param2",
					"DATA_TYPE":     "NUMBER",
					"IN_OUT":        "OUT",
					"POSITION":      2,
				},
			},
			expectedError: nil,
		},
		{
			name:          "procedure not found",
			procedureName: "nonexistent_procedure",
			setupMock: func(mockRepo *MockRepository) {
				mockRepo.On("GetProcedureInfo",
					mock.Anything,
					"nonexistent_procedure").Return([]map[string]any{}, nil)
			},
			expectedResult: response.GetProcedureInfoResponse{},
			expectedError:  nil,
		},
		{
			name:          "repository returns error",
			procedureName: "error_procedure",
			setupMock: func(mockRepo *MockRepository) {
				mockRepo.On("GetProcedureInfo",
					mock.Anything,
					"error_procedure").Return(nil, errors.New("database connection failed"))
			},
			expectedResult: nil,
			expectedError:  errors.New("database connection failed"),
		},
		{
			name:          "empty procedure name",
			procedureName: "",
			setupMock: func(mockRepo *MockRepository) {
				mockRepo.On("GetProcedureInfo",
					mock.Anything,
					"").Return([]map[string]any{}, nil)
			},
			expectedResult: response.GetProcedureInfoResponse{},
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewProcedureService(mockRepo)

			// Execute
			result, err := service.GetProcedureInfo(context.Background(), tt.procedureName)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

// Test context cancellation
func TestProcedureService_ContextCancellation(t *testing.T) {
	mockRepo := &MockRepository{}

	// Setup mock to simulate context cancellation
	mockRepo.On("CallProcedure",
		mock.Anything,
		"test_procedure",
		mock.Anything).Return(nil, context.Canceled)

	service := NewProcedureService(mockRepo)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := request.CallProcedureRequest{
		Name:   "test_procedure",
		Params: []request.ProcedureParam{},
	}

	result, err := service.CallProcedure(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Nil(t, result)

	mockRepo.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkProcedureService_CallProcedure(b *testing.B) {
	mockRepo := &MockRepository{}
	expectedResult := map[string]any{"result": "success"}

	mockRepo.On("CallProcedure",
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(expectedResult, nil).Times(b.N)

	service := NewProcedureService(mockRepo)
	req := request.CallProcedureRequest{
		Name:   "test_procedure",
		Params: []request.ProcedureParam{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CallProcedure(context.Background(), req)
	}
}

func BenchmarkProcedureService_GetProcedureInfo(b *testing.B) {
	mockRepo := &MockRepository{}
	expectedResult := []map[string]any{{"info": "test"}}

	mockRepo.On("GetProcedureInfo",
		mock.Anything,
		mock.Anything).Return(expectedResult, nil).Times(b.N)

	service := NewProcedureService(mockRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetProcedureInfo(context.Background(), "test_procedure")
	}
}
