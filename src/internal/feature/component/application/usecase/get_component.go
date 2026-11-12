package usecase

import "context"

// GetComponent devuelve un COMP por su ID legible. Leer no exige permiso.
type GetComponent struct {
	d Deps
}

// GetComponentInput es la entrada de GetComponent.
type GetComponentInput struct {
	Code string
}

// Execute busca el COMP.
func (uc *GetComponent) Execute(ctx context.Context, in GetComponentInput) (ComponentOutput, error) {
	view, err := findComponent(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return ComponentOutput{}, err
	}
	return toOutput(view), nil
}
