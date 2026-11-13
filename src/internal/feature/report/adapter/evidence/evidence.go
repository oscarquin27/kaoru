// Package evidence implementa el puerto Evidence con los casos de uso de los features que saben del PKG:
// bundle (su contenido), traceability (la cadena REQ → RUN y los DEF), regression y production.
package evidence

import (
	"context"
	"errors"
	"slices"

	bundleuc "kaoru/internal/feature/bundle/application/usecase"
	productionuc "kaoru/internal/feature/production/application/usecase"
	regressionuc "kaoru/internal/feature/regression/application/usecase"
	regressiondomain "kaoru/internal/feature/regression/domain"
	"kaoru/internal/feature/report/application/port"
	"kaoru/internal/feature/report/domain"
	traceabilityuc "kaoru/internal/feature/traceability/application/usecase"
)

// pageSize alcanza para todos los CHK, DIAG o SMOKE de un PKG.
const pageSize = 500

// Sources son los casos de uso de los que sale la evidencia.
type Sources struct {
	Package     *bundleuc.GetPackage
	Trace       *traceabilityuc.GetPackageTrace
	Regression  *regressionuc.GetRegression
	Checklists  *productionuc.ListChecklists
	Diagnostics *productionuc.ListDiagnostics
	SmokeTests  *productionuc.ListSmokeTests
}

// Evidence reúne lo que Kaoru sabe de un PKG.
type Evidence struct {
	s Sources
}

var _ port.Evidence = (*Evidence)(nil)

// New crea el adaptador.
func New(s Sources) *Evidence { return &Evidence{s: s} }

// Gather lee cada parte del informe. Lee dentro de la transacción del context, así que ve lo mismo que el bloqueo protege.
func (e *Evidence) Gather(ctx context.Context, packageCode string) (domain.Content, error) {
	var c domain.Content
	pkg, err := e.s.Package.Execute(ctx, bundleuc.GetPackageInput{Code: packageCode})
	if err != nil {
		return c, err
	}
	p := pkg.Package
	c.Package = domain.PackageInfo{
		Id: p.Id, Code: p.Code.String(), Title: p.Title, SystemName: p.System.Name, StateName: p.State.Name, Route: pkg.Route,
	}
	for _, b := range p.Builds {
		c.Package.Builds = append(c.Package.Builds, domain.Build{Code: b.Code, Version: b.Version, Checksum: b.Checksum})
	}
	for _, r := range p.Requirements {
		c.Package.Requirements = append(c.Package.Requirements, domain.Requirement{Code: r.Code, Title: r.Title, StateName: r.State.Name})
	}

	if err := e.trace(ctx, packageCode, &c); err != nil {
		return c, err
	}
	if err := e.regression(ctx, packageCode, &c); err != nil {
		return c, err
	}
	return c, e.production(ctx, packageCode, &c)
}

func (e *Evidence) trace(ctx context.Context, packageCode string, c *domain.Content) error {
	trace, err := e.s.Trace.Execute(ctx, traceabilityuc.GetPackageTraceInput{PackageCode: packageCode})
	if err != nil {
		return err
	}
	for _, r := range trace.Rows {
		c.Trace = append(c.Trace, domain.TraceRow{
			RequirementCode: r.RequirementCode, ChangeCode: r.ChangeCode, Route: r.Route, RiskCode: r.RiskCode, RiskLevel: r.RiskLevel,
			FunctionalCases: r.FunctionalCases, TestCaseCode: r.TestCaseCode, Critical: r.TestCaseCritical,
			RunCode: r.RunCode, RunResult: r.RunResult, Defects: r.Defects,
		})
	}
	for _, r := range trace.Runs {
		c.Runs = append(c.Runs, domain.Run{
			Code: r.Code, TestCaseCode: r.TestCaseCode, Result: r.Result, BuildCode: r.BuildCode, BuildVersion: r.BuildVersion,
			Environment: r.EnvironmentName, ExecutedAt: r.ExecutedAt, ExecutedBy: r.ExecutedByName,
		})
	}
	for _, d := range trace.Defects {
		c.Defects = append(c.Defects, domain.Defect{
			Code: d.Code, Title: d.Title, Severity: d.Severity, StateName: d.State.Name, RequirementCode: d.RequirementCode,
			DeferredBy: d.DeferredBy, DeferredAt: d.DeferredAt, DeferredReason: d.DeferredReason,
		})
	}
	return nil
}

// regression deja la sección vacía si la regresión del PKG no se propuso.
func (e *Evidence) regression(ctx context.Context, packageCode string, c *domain.Content) error {
	g, err := e.s.Regression.Execute(ctx, regressionuc.GetRegressionInput{PackageCode: packageCode})
	if errors.Is(err, regressiondomain.ErrNotProposed) {
		return nil
	}
	if err != nil {
		return err
	}
	s := g.Summary
	r := &domain.Regression{
		ConfirmedAt: g.ConfirmedAt, Proposed: s.Proposed, Included: s.Included, Added: s.Added, Removed: s.Removed,
		Passed: s.Passed, Failed: s.Failed, Pending: s.Pending,
	}
	if g.ConfirmedBy != nil {
		r.ConfirmedBy = g.ConfirmedBy.Name
	}
	for _, it := range g.Items {
		item := domain.RegressionItem{
			TestCaseCode: it.TestCase.Code, Origin: string(it.Origin), Proposed: it.Proposed, Included: it.Included, Reason: it.Reason,
		}
		if it.LastRun != nil {
			item.LastResult = it.LastRun.Result
		}
		if it.DecidedBy != nil {
			item.DecidedBy = it.DecidedBy.Name
		}
		r.Items = append(r.Items, item)
	}
	c.Regression = r
	return nil
}

// production lee los CHK, DIAG y SMOKE del PKG, del más antiguo al más reciente.
func (e *Evidence) production(ctx context.Context, packageCode string, c *domain.Content) error {
	in := productionuc.ListInput{PackageCode: packageCode, Page: 1, Size: pageSize}
	checklists, err := e.s.Checklists.Execute(ctx, in)
	if err != nil {
		return err
	}
	for _, out := range slices.Backward(checklists.Checklists) {
		k := out.Checklist
		chk := domain.Checklist{
			Code: k.Code.String(), Environment: k.Environment.Name, TemplateVersion: k.TemplateVersion,
			StartedBy: k.StartedBy.Name, CreatedAt: k.CreatedAt,
		}
		for _, step := range k.Steps {
			s := domain.ChecklistStep{Position: step.Position, Title: step.Title}
			if step.Signature != nil {
				s.SignedBy, s.SignedAt, s.Comment = step.Signature.SignedBy.Name, step.Signature.SignedAt, step.Signature.Comment
			}
			chk.Steps = append(chk.Steps, s)
		}
		c.Checklists = append(c.Checklists, chk)
	}

	diagnostics, err := e.s.Diagnostics.Execute(ctx, in)
	if err != nil {
		return err
	}
	for _, d := range slices.Backward(diagnostics.Diagnostics) {
		diag := domain.Diagnostic{
			Code: d.Code.String(), Deployment: d.Deployment.Code, Environment: d.Deployment.Environment.Name,
			SubmittedBy: d.SubmittedBy.Name, CreatedAt: d.CreatedAt, Notes: d.Notes,
		}
		if r := d.Review; r != nil {
			diag.ReviewResult, diag.ReviewedBy, diag.ReviewedAt, diag.ReviewNotes = string(r.Result), r.ReviewedBy.Name, r.ReviewedAt, r.Notes
		}
		c.Diagnostics = append(c.Diagnostics, diag)
	}

	smokeTests, err := e.s.SmokeTests.Execute(ctx, in)
	if err != nil {
		return err
	}
	for _, s := range slices.Backward(smokeTests.SmokeTests) {
		c.SmokeTests = append(c.SmokeTests, domain.SmokeTest{
			Code: s.Code.String(), Deployment: s.Deployment.Code, Environment: s.Deployment.Environment.Name,
			Result: string(s.Result), Notes: s.Notes, ExecutedBy: s.ExecutedBy.Name, CreatedAt: s.CreatedAt,
		})
	}
	return nil
}
