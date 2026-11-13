package usecase

import (
	"context"

	"kaoru/internal/feature/report/application/port"
	"kaoru/internal/feature/report/domain"
)

// ListReports devuelve una página de RPT sin su contenido, del más reciente al más antiguo.
type ListReports struct {
	d Deps
}

// ListReportsInput es la entrada de ListReports. PackageCode vacío no filtra.
type ListReportsInput struct {
	PackageCode string
	Page        int
	Size        int
}

// ListReportsOutput es una página de RPT.
type ListReportsOutput struct {
	Reports []domain.Snapshot
	Total   int
}

// Execute lista los RPT. Leer no exige permiso.
func (uc *ListReports) Execute(ctx context.Context, in ListReportsInput) (ListReportsOutput, error) {
	reports, total, err := uc.d.Repository.List(ctx, port.Filter{PackageCode: in.PackageCode, Page: in.Page, Size: in.Size})
	if err != nil {
		return ListReportsOutput{}, err
	}
	out := ListReportsOutput{Reports: make([]domain.Snapshot, len(reports)), Total: total}
	for i, r := range reports {
		out.Reports[i] = r.Snapshot()
	}
	return out, nil
}
