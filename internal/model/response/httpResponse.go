package response

import (
	"encoding/json"
	"log"
	"net/http"
)

type HttpResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
	Data    any    `json:"data"`
}

func SuccessResponse(message string, data any) HttpResponse {
	if data == nil {
		data = map[string]any{}
	}
	return HttpResponse{
		Message: message,
		Status:  true,
		Data:    data,
	}
}

func ErrorResponse(message string, data any) HttpResponse {
	if data == nil {
		data = map[string]any{}
	}
	return HttpResponse{
		Message: message,
		Status:  false,
		Data:    data,
	}
}

func WriteJSON(w http.ResponseWriter, statusCode int, resp HttpResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Println("JSON encoding error", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
