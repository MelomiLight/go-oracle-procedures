package request

import (
	"errors"
	"fmt"
	"strings"
)

type CallProcedureRequest struct {
	Name   string           `json:"name"`
	Params []ProcedureParam `json:"params"`
}

type ProcedureParam struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Value     any    `json:"value"`
	Direction string `json:"direction"`
}

func (r *CallProcedureRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("procedure name is required")
	}

	for i, p := range r.Params {
		if strings.TrimSpace(p.Name) != "" ||
			strings.TrimSpace(p.Type) != "" ||
			strings.TrimSpace(p.Direction) != "" {

			if strings.TrimSpace(p.Name) == "" {
				return fmt.Errorf("param[%d] name is required", i)
			}
			if strings.TrimSpace(p.Type) == "" {
				return fmt.Errorf("param[%d] type is required", i)
			}
			if strings.TrimSpace(p.Direction) == "" {
				return fmt.Errorf("param[%d] direction is required", i)
			}
		}
	}
	return nil
}
