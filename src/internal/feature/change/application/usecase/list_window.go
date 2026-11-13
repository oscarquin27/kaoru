package usecase

import "context"

// ListWindows devuelve las ventanas de cambio vigentes: las de los CHG que no están cerrados ni rechazados.
type ListWindows struct {
	d Deps
}

// Execute lista los CHG con ventana, por inicio de la ventana. Leer no exige permiso.
func (uc *ListWindows) Execute(ctx context.Context) ([]ChangeOutput, error) {
	views, err := uc.d.Repository.ListWindows(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ChangeOutput, len(views))
	for i, v := range views {
		out[i] = toOutput(v)
	}
	return out, nil
}
