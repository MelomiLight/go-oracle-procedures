package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"oracle-golang/internal/model/request"
	"oracle-golang/internal/model/response"
	"oracle-golang/pkg/util"
)

type ProcedureService interface {
	CallProcedure(ctx context.Context, r request.CallProcedureRequest) (response.CallProcedureResponse, error)
	GetProcedureInfo(ctx context.Context, procedureName string) (response.GetProcedureInfoResponse, error)
}

type ProcedureHandler struct {
	service ProcedureService
}

func NewProcedureHandler(service ProcedureService) *ProcedureHandler {
	return &ProcedureHandler{
		service: service,
	}
}

func (ph *ProcedureHandler) CallProcedure(w http.ResponseWriter, r *http.Request) {
	var req request.CallProcedureRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logMethod(err.Error())
		response.WriteJSON(w, http.StatusBadRequest, response.ErrorResponse("Invalid JSON format", nil))
		return
	}

	if err := req.Validate(); err != nil {
		logMethod(err.Error())
		response.WriteJSON(w, http.StatusBadRequest, response.ErrorResponse(err.Error(), nil))
		return
	}

	result, err := ph.service.CallProcedure(r.Context(), req)
	if err != nil {
		logMethod(err.Error())
		response.WriteJSON(w, http.StatusInternalServerError, response.ErrorResponse(err.Error(), nil))
		return
	}

	response.WriteJSON(w, http.StatusOK, response.SuccessResponse("Success", result))
	return
}

func (ph *ProcedureHandler) GetProcedureInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProcedureName string `json:"procedure_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logMethod(err.Error())
		response.WriteJSON(w, http.StatusBadRequest, response.ErrorResponse("Invalid JSON format", nil))
		return
	}

	if req.ProcedureName == "" {
		logMethod("procedure_name is required")
		response.WriteJSON(w, http.StatusBadRequest, response.ErrorResponse("procedure_name is required", nil))
		return
	}

	result, err := ph.service.GetProcedureInfo(r.Context(), req.ProcedureName)
	if err != nil {
		logMethod(err.Error())
		response.WriteJSON(w, http.StatusInternalServerError, response.ErrorResponse(err.Error(), nil))
		return
	}

	response.WriteJSON(w, http.StatusOK, response.SuccessResponse("Success", result))
}

func logMethod(message string) {
	log.Printf("[%s] %s", util.CurrentMethod(2), message)
}
