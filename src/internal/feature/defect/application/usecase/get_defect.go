package usecase

import "context"

// GetDefect devuelve un DEF por su ID legible. Leer no exige permiso.
type GetDefect struct {
	d Deps
}

// GetDefectInput es la entrada de GetDefect.
type GetDefectInput struct {
	Code string
}

// Execute busca el DEF.
func (uc *GetDefect) Execute(ctx context.Context, in GetDefectInput) (DefectOutput, error) {
	defect, err := findDefect(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return DefectOutput{}, err
	}
	return toOutput(ctx, uc.d, defect)
}
