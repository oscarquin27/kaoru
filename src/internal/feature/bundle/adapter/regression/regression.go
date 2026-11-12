// Package regression implementa el puerto Regression con los casos de uso de regression.
package regression

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	regressionuc "kaoru/internal/feature/regression/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Regression avisa a regression de que la selección confirmada de un PKG ya no corresponde.
type Regression struct {
	reopen *regressionuc.ReopenRegression
}

var _ port.Regression = (*Regression)(nil)

// New crea el adaptador.
func New(reopen *regressionuc.ReopenRegression) *Regression { return &Regression{reopen: reopen} }

// Reopen reutiliza la transacción del context y conserva el código de error de regression.
func (r *Regression) Reopen(ctx context.Context, actor shared.UserId, packageCode shared.Code, reason string) error {
	return r.reopen.Execute(ctx, regressionuc.ReopenRegressionInput{
		ActorId: actor, PackageCode: packageCode.String(), Reason: reason,
	})
}
