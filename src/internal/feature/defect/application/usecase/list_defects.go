package usecase

import (
	"context"

	"kaoru/internal/feature/defect/application/port"
)

// ListDefects devuelve una página de DEF, del más reciente al más antiguo. Leer no exige permiso.
type ListDefects struct {
	d Deps
}

// ListDefectsInput es la entrada de ListDefects. Un filtro vacío no filtra.
type ListDefectsInput struct {
	StateKey        string
	RequirementCode string
	Severity        string
	AssigneeId      string
	Query           string
	Page            int
	Size            int
}

// ListDefectsOutput es una página de DEF.
type ListDefectsOutput struct {
	Defects []DefectOutput
	Total   int
}

// Execute lista los DEF con su tiempo en estado.
func (uc *ListDefects) Execute(ctx context.Context, in ListDefectsInput) (ListDefectsOutput, error) {
	defects, total, err := uc.d.Repository.List(ctx, port.Filter{
		StateKey: in.StateKey, RequirementCode: in.RequirementCode, Severity: in.Severity,
		AssigneeId: in.AssigneeId, Query: in.Query, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListDefectsOutput{}, err
	}
	outs, err := toOutputs(ctx, uc.d, defects)
	if err != nil {
		return ListDefectsOutput{}, err
	}
	return ListDefectsOutput{Defects: outs, Total: total}, nil
}
