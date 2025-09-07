package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"oracle-golang/internal/model/request"
	"oracle-golang/internal/model/response"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProcedureService is a mock implementation of the ProcedureService interface
type MockProcedureService struct {
	mock.Mock
}

func (m *MockProcedureService) CallProcedure(ctx context.Context, r request.CallProcedureRequest) (response.CallProcedureResponse, error) {
	args := m.Called(ctx, r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(response.CallProcedureResponse), args.Error(1)
}

func (m *MockProcedureService) GetProcedureInfo(ctx context.Context, procedureName string) (response.GetProcedureInfoResponse, error) {
	args := m.Called(ctx, procedureName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(response.GetProcedureInfoResponse), args.Error(1)
}

func TestNewProcedureHandler(t *testing.T) {
	mockService := &MockProcedureService{}
	handler := NewProcedureHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestProcedureHandler_CallProcedure(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		setupMock          func(*MockProcedureService)
		expectedStatusCode int
		validateResponse   func(*testing.T, map[string]any)
	}{
		{
			name: "successful procedure call",
			requestBody: `{
				"name": "test_procedure",
				"params": [
					{"name": "param1", "value": "value1", "type": "IN", "direction": "IN"},
					{"name": "param2", "value": 123, "type": "IN", "direction": "IN"}
				]
			}`,
			setupMock: func(mockService *MockProcedureService) {
				expectedResponse := response.CallProcedureResponse{
					"result": "success",
					"count":  5,
				}
				mockService.On("CallProcedure",
					mock.Anything,
					mock.MatchedBy(func(req request.CallProcedureRequest) bool {
						return req.Name == "test_procedure" && len(req.Params) == 2
					})).Return(expectedResponse, nil)
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Success", resp["message"])
				// Check if success field exists and is boolean
				if success, exists := resp["success"]; exists {
					assert.True(t, success.(bool))
				}
				assert.NotNil(t, resp["data"])
				data := resp["data"].(map[string]any)
				assert.Equal(t, "success", data["result"])
				// Handle both int, float64 and json.Number types for count
				count := data["count"]
				switch v := count.(type) {
				case int:
					assert.Equal(t, 5, v)
				case float64:
					assert.Equal(t, float64(5), v)
				case json.Number:
					countStr := string(v)
					assert.Equal(t, "5", countStr)
				default:
					t.Fatalf("Unexpected type for count: %T", v)
				}
			},
		},
		{
			name: "procedure call with empty parameters",
			requestBody: `{
				"name": "simple_procedure",
				"params": []
			}`,
			setupMock: func(mockService *MockProcedureService) {
				expectedResponse := response.CallProcedureResponse{
					"status": "completed",
				}
				mockService.On("CallProcedure",
					mock.Anything,
					mock.MatchedBy(func(req request.CallProcedureRequest) bool {
						return req.Name == "simple_procedure" && len(req.Params) == 0
					})).Return(expectedResponse, nil)
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Success", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.True(t, success.(bool))
				}
				assert.NotNil(t, resp["data"])
				data := resp["data"].(map[string]any)
				assert.Equal(t, "completed", data["status"])
			},
		},
		{
			name:        "invalid JSON format",
			requestBody: `{"invalid": json}`,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as JSON parsing will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Invalid JSON format", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name: "validation error - missing procedure name",
			requestBody: `{
				"params": [
					{"name": "param1", "value": "value1", "type": "IN", "direction": "IN"}
				]
			}`,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as validation will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Contains(t, resp["message"], "required")
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name: "validation error - missing parameter direction",
			requestBody: `{
				"name": "test_procedure",
				"params": [
					{"name": "param1", "value": "value1", "type": "IN"}
				]
			}`,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as validation will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Contains(t, resp["message"], "direction is required")
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name: "service returns error",
			requestBody: `{
				"name": "error_procedure",
				"params": []
			}`,
			setupMock: func(mockService *MockProcedureService) {
				mockService.On("CallProcedure",
					mock.Anything,
					mock.MatchedBy(func(req request.CallProcedureRequest) bool {
						return req.Name == "error_procedure"
					})).Return(nil, errors.New("database connection error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "database connection error", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name:        "empty request body",
			requestBody: ``,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as JSON parsing will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Invalid JSON format", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockProcedureService{}
			tt.setupMock(mockService)

			handler := NewProcedureHandler(mockService)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/procedure/call", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			handler.CallProcedure(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Parse and validate response
			var fromResponse map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &fromResponse)
			assert.NoError(t, err)

			// Validate response content
			tt.validateResponse(t, fromResponse)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestProcedureHandler_GetProcedureInfo(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		setupMock          func(*MockProcedureService)
		expectedStatusCode int
		validateResponse   func(*testing.T, map[string]any)
	}{
		{
			name: "successful procedure info retrieval",
			requestBody: `{
				"procedure_name": "test_procedure"
			}`,
			setupMock: func(mockService *MockProcedureService) {
				expectedResponse := response.GetProcedureInfoResponse{
					{
						"argument_name": "param1",
						"data_type":     "VARCHAR2",
						"in_out":        "IN",
						"position":      1,
					},
					{
						"argument_name": "param2",
						"data_type":     "NUMBER",
						"in_out":        "OUT",
						"position":      2,
					},
				}
				mockService.On("GetProcedureInfo",
					mock.Anything,
					"test_procedure").Return(expectedResponse, nil)
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Success", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.True(t, success.(bool))
				}
				assert.NotNil(t, resp["data"])

				data := resp["data"].([]any)
				assert.Len(t, data, 2)

				firstParam := data[0].(map[string]any)
				assert.Equal(t, "param1", firstParam["argument_name"])
				assert.Equal(t, "VARCHAR2", firstParam["data_type"])
			},
		},
		{
			name: "procedure not found",
			requestBody: `{
				"procedure_name": "nonexistent_procedure"
			}`,
			setupMock: func(mockService *MockProcedureService) {
				mockService.On("GetProcedureInfo",
					mock.Anything,
					"nonexistent_procedure").Return(response.GetProcedureInfoResponse{}, nil)
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Success", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.True(t, success.(bool))
				}
				data := resp["data"].([]any)
				assert.Len(t, data, 0)
			},
		},
		{
			name:        "invalid JSON format",
			requestBody: `{"invalid": json}`,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as JSON parsing will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "Invalid JSON format", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name: "missing procedure_name",
			requestBody: `{
				"other_field": "value"
			}`,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as validation will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "procedure_name is required", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name: "empty procedure_name",
			requestBody: `{
				"procedure_name": ""
			}`,
			setupMock: func(mockService *MockProcedureService) {
				// No mock setup needed as validation will fail
			},
			expectedStatusCode: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "procedure_name is required", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
		{
			name: "service returns error",
			requestBody: `{
				"procedure_name": "error_procedure"
			}`,
			setupMock: func(mockService *MockProcedureService) {
				mockService.On("GetProcedureInfo",
					mock.Anything,
					"error_procedure").Return(nil, errors.New("database connection failed"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			validateResponse: func(t *testing.T, resp map[string]any) {
				assert.Equal(t, "database connection failed", resp["message"])
				if success, exists := resp["success"]; exists {
					assert.False(t, success.(bool))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockProcedureService{}
			tt.setupMock(mockService)

			handler := NewProcedureHandler(mockService)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/procedure/info", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			handler.GetProcedureInfo(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Parse and validate response
			var fromResponse map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &fromResponse)
			assert.NoError(t, err)

			// Validate response content
			tt.validateResponse(t, fromResponse)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

// Test large payloads
func TestProcedureHandler_LargePayload(t *testing.T) {
	mockService := &MockProcedureService{}

	// Create a large number of parameters
	var params []request.ProcedureParam
	for i := 0; i < 1000; i++ {
		params = append(params, request.ProcedureParam{
			Name:      fmt.Sprintf("param_%d", i),
			Value:     fmt.Sprintf("value_%d", i),
			Type:      "IN",
			Direction: "IN",
		})
	}

	mockService.On("CallProcedure",
		mock.Anything,
		mock.MatchedBy(func(req request.CallProcedureRequest) bool {
			return req.Name == "large_procedure" && len(req.Params) == 1000
		})).Return(response.CallProcedureResponse{"result": "success"}, nil)

	handler := NewProcedureHandler(mockService)

	// Create a request body
	reqBody, _ := json.Marshal(map[string]any{
		"name":   "large_procedure",
		"params": params,
	})
	req := httptest.NewRequest(http.MethodPost, "/procedure/call", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CallProcedure(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// Test HTTP methods
func TestProcedureHandler_HTTPMethods(t *testing.T) {
	mockService := &MockProcedureService{}
	handler := NewProcedureHandler(mockService)

	tests := []struct {
		name       string
		method     string
		endpoint   string
		handleFunc http.HandlerFunc
	}{
		{
			name:       "CallProcedure supports POST",
			method:     http.MethodPost,
			endpoint:   "/procedure/call",
			handleFunc: handler.CallProcedure,
		},
		{
			name:       "GetProcedureInfo supports POST",
			method:     http.MethodPost,
			endpoint:   "/procedure/info",
			handleFunc: handler.GetProcedureInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			tt.handleFunc(w, req)

			// Should not return method not allowed
			assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

// Benchmark tests
func BenchmarkProcedureHandler_CallProcedure(b *testing.B) {
	mockService := &MockProcedureService{}
	expectedResponse := response.CallProcedureResponse{"result": "success"}

	mockService.On("CallProcedure",
		mock.Anything,
		mock.Anything).Return(expectedResponse, nil).Times(b.N)

	handler := NewProcedureHandler(mockService)
	requestBody := `{"name": "test_procedure", "params": []}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/procedure/call", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CallProcedure(w, req)
	}
}

func BenchmarkProcedureHandler_GetProcedureInfo(b *testing.B) {
	mockService := &MockProcedureService{}
	expectedResponse := response.GetProcedureInfoResponse{{"info": "test"}}

	mockService.On("GetProcedureInfo",
		mock.Anything,
		mock.Anything).Return(expectedResponse, nil).Times(b.N)

	handler := NewProcedureHandler(mockService)
	requestBody := `{"procedure_name": "test_procedure"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/procedure/info", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.GetProcedureInfo(w, req)
	}
}
