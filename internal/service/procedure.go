package service

import (
	"context"
	"oracle-golang/internal/model/request"
	"oracle-golang/internal/model/response"
)

type Repository interface {
	CallProcedure(ctx context.Context, name string, params []request.ProcedureParam) (map[string]any, error)
	GetProcedureInfo(ctx context.Context, procedureName string) ([]map[string]any, error)
}

type ProcedureService struct {
	repo Repository
}

func NewProcedureService(repo Repository) *ProcedureService {
	return &ProcedureService{repo: repo}
}

func (ps *ProcedureService) CallProcedure(ctx context.Context, r request.CallProcedureRequest) (response.CallProcedureResponse, error) {
	result, err := ps.repo.CallProcedure(ctx, r.Name, r.Params)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (ps *ProcedureService) GetProcedureInfo(ctx context.Context, procedureName string) (response.GetProcedureInfoResponse, error) {
	result, err := ps.repo.GetProcedureInfo(ctx, procedureName)
	if err != nil {
		return nil, err
	}
	return result, nil
}
