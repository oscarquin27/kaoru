// Package usecase contiene los casos de uso de regression.
package usecase

import (
	"context"
	"errors"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/regression/application/port"
	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionManage es el permiso para proponer, ajustar y confirmar la regresión.
const PermissionManage = "regression.manage"

// Errores de los casos de uso.
var (
	ErrPackageNotFound  = axis.New(axis.NotFoundResource, "El PKG no existe.")
	ErrTestCaseNotFound = axis.New(axis.NotFoundResource, "El TC no existe.")
)

// Campos con los que se audita la regresión de un PKG.
const (
	fieldProposed  = "proposed"
	fieldItem      = "item"
	fieldConfirmed = "confirmed"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	GetRegression        *GetRegression
	ProposeRegression    *ProposeRegression
	AdjustRegressionItem *AdjustRegressionItem
	ConfirmRegression    *ConfirmRegression
	ReopenRegression     *ReopenRegression
	GetRegressionFacts   *GetRegressionFacts
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Permissions == nil || d.AuditTrail == nil || d.UnitOfWork == nil || d.Clock == nil {
		return nil, errors.New("regression: faltan dependencias")
	}
	return &UseCases{
		GetRegression:        &GetRegression{d: d},
		ProposeRegression:    &ProposeRegression{d: d},
		AdjustRegressionItem: &AdjustRegressionItem{d: d},
		ConfirmRegression:    &ConfirmRegression{d: d},
		ReopenRegression:     &ReopenRegression{d: d},
		GetRegressionFacts:   &GetRegressionFacts{d: d},
	}, nil
}

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// ItemOutput es un TC de la selección tal como lo devuelven los casos de uso.
type ItemOutput struct {
	TestCase   domain.TestCaseRef
	Title      string
	IsCritical bool
	Origin     domain.Origin
	Proposed   bool
	Included   bool
	Reason     string
	// DecidedBy y DecidedAt son nil si QA no ajustó el TC.
	DecidedBy  *UserRef
	DecidedAt  *time.Time
	Components []port.ComponentRef
	LastRun    *port.RunRef
}

// RegressionOutput es la selección de regresión de un PKG: propuesta contra final, con su avance.
type RegressionOutput struct {
	PackageCode string
	ProposedBy  UserRef
	ProposedAt  time.Time
	// ConfirmedBy y ConfirmedAt son nil mientras la selección no esté confirmada.
	ConfirmedBy *UserRef
	ConfirmedAt *time.Time
	Summary     domain.Summary
	// Items son todos los TC propuestos o agregados, incluidos los que QA quitó, por código.
	Items []ItemOutput
}

func toOutput(pkg port.PackageRef, v port.SelectionView) RegressionOutput {
	s := v.Selection
	out := RegressionOutput{
		PackageCode: pkg.Code.String(),
		ProposedBy:  UserRef{Id: s.ProposedBy, Name: v.ProposedByName}, ProposedAt: s.ProposedAt,
		Summary: domain.Summarize(progress(v)), Items: make([]ItemOutput, len(v.Items)),
	}
	if s.Confirmed() {
		out.ConfirmedBy, out.ConfirmedAt = &UserRef{Id: s.ConfirmedBy, Name: v.ConfirmedByName}, s.ConfirmedAt
	}
	for i, iv := range v.Items {
		item := ItemOutput{
			TestCase: iv.Item.TestCase, Title: iv.Title, IsCritical: iv.IsCritical, Origin: iv.Item.Origin,
			Proposed: iv.Item.Proposed, Included: iv.Item.Included, Reason: iv.Item.Reason,
			Components: iv.Components, LastRun: iv.LastRun,
		}
		if iv.Item.DecidedAt != nil {
			item.DecidedBy, item.DecidedAt = &UserRef{Id: iv.Item.DecidedBy, Name: iv.DecidedByName}, iv.Item.DecidedAt
		}
		out.Items[i] = item
	}
	return out
}

// progress junta cada TC de la selección con el resultado de su último RUN sobre un BUILD del PKG.
func progress(v port.SelectionView) []domain.Progress {
	out := make([]domain.Progress, len(v.Items))
	for i, iv := range v.Items {
		out[i].Item = iv.Item
		if iv.LastRun != nil {
			out[i].LastResult = iv.LastRun.Result
		}
	}
	return out
}

// items devuelve los TC de la selección como los entiende el dominio.
func items(v port.SelectionView) []domain.Item {
	out := make([]domain.Item, len(v.Items))
	for i, iv := range v.Items {
		out[i] = iv.Item
	}
	return out
}

// findPackage valida el código y lee el PKG; un código mal formado tampoco existe.
func findPackage(ctx context.Context, repo port.Repository, raw string) (port.PackageRef, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.PackageRef{}, ErrPackageNotFound
	}
	pkg, err := repo.FindPackage(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return port.PackageRef{}, ErrPackageNotFound
	}
	return pkg, err
}

// lockPackage bloquea el PKG: se usa dentro de la transacción de todo caso de uso que escribe,
// para que dos cambios simultáneos sobre la misma selección no se pisen.
func lockPackage(ctx context.Context, repo port.Repository, raw string) (port.PackageRef, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.PackageRef{}, ErrPackageNotFound
	}
	pkg, err := repo.LockPackage(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return port.PackageRef{}, ErrPackageNotFound
	}
	return pkg, err
}

// findSelection lee la selección del PKG; si todavía no se propuso, devuelve ErrNotProposed.
func findSelection(ctx context.Context, repo port.Repository, packageId string) (port.SelectionView, error) {
	view, err := repo.FindSelection(ctx, packageId)
	if axis.IsCode(err, axis.NotFoundResource) {
		return port.SelectionView{}, domain.ErrNotProposed
	}
	return view, err
}

// readOutput lee la selección ya guardada para devolverla.
func readOutput(ctx context.Context, repo port.Repository, pkg port.PackageRef) (RegressionOutput, error) {
	view, err := findSelection(ctx, repo, pkg.Id)
	if err != nil {
		return RegressionOutput{}, err
	}
	return toOutput(pkg, view), nil
}
