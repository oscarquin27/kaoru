package usecase

import (
	"context"
)

// GetChecklist devuelve un checklist con sus pasos y firmas. Leer no exige permiso.
type GetChecklist struct {
	d Deps
}

// GetChecklistInput es la entrada de GetChecklist.
type GetChecklistInput struct {
	Code string
}

// Execute devuelve ErrChecklistNotFound si el CHK no existe.
func (uc *GetChecklist) Execute(ctx context.Context, in GetChecklistInput) (ChecklistOutput, error) {
	checklist, err := findChecklist(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return ChecklistOutput{}, err
	}
	return toChecklistOutput(checklist), nil
}
