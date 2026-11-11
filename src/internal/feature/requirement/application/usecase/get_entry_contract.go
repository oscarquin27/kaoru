package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/domain"
)

// ErrEntryItemNotFound es el error de un ítem que no está en el contrato de entrada.
var ErrEntryItemNotFound = axis.New(axis.NotFoundResource, "El ítem no existe en el contrato de entrada.")

// GetEntryContract devuelve el contrato de entrada dev → QA de un REQ, con qué falta.
type GetEntryContract struct {
	d Deps
}

// GetEntryContractInput es la entrada de GetEntryContract.
type GetEntryContractInput struct {
	Code string
}

// Execute lista los ítems con su estado. Leer no exige permiso.
func (uc *GetEntryContract) Execute(ctx context.Context, in GetEntryContractInput) ([]domain.EntryItem, error) {
	requirement, err := findRequirement(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return nil, err
	}
	return uc.d.Repository.ListEntryContract(ctx, requirement.Snapshot().Id)
}
