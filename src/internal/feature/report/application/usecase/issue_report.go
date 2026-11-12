package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/report/application/port"
	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

// IssueReport emite el informe de conformidad de un PKG con lo que hay en Kaoru en ese momento.
// Si el PKG ya tiene uno, emite la versión siguiente, que referencia a la vigente.
type IssueReport struct {
	d Deps
}

// IssueReportInput es la entrada de IssueReport.
type IssueReportInput struct {
	Actor       shared.Actor
	PackageCode string
}

// Execute exige el permiso rpt.issue. El PKG que falta o no existe es un error del campo package.
func (uc *IssueReport) Execute(ctx context.Context, in IssueReportInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionIssue); err != nil {
		return domain.Snapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.Snapshot{}, err
	}
	now := uc.d.Clock.Now()

	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := uc.d.Repository.FindPackage(ctx, in.PackageCode)
		if axis.IsCode(err, axis.NotFoundResource) {
			field := axis.InvalidField("package", "El PKG no existe.")
			if in.PackageCode == "" {
				field = axis.Required("package", "Indica el PKG.")
			}
			return axis.Invalid(field)
		}
		if err != nil {
			return err
		}
		if err := uc.d.Repository.LockPackage(ctx, pkg.Id); err != nil {
			return err
		}
		var previous *domain.Snapshot
		latest, err := uc.d.Repository.Latest(ctx, pkg.Id)
		switch {
		case err == nil:
			s := latest.Snapshot()
			previous = &s
		case !axis.IsCode(err, axis.NotFoundResource):
			return err
		}
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixReport, now.Year())
		if err != nil {
			return err
		}
		issuer, err := uc.d.Repository.UserName(ctx, in.Actor.Id)
		if err != nil {
			return err
		}
		content, err := uc.d.Evidence.Gather(ctx, pkg.Code)
		if err != nil {
			return err
		}
		content.Code, content.IssuedBy, content.IssuedAt = code.String(), issuer, now
		content.Version = 1
		if previous != nil {
			content.Version, content.Previous = previous.Version+1, previous.Code.String()
		}
		report := domain.New(domain.NewParams{
			Id: id, Code: code, Package: pkg, Previous: previous, Content: domain.Render(content),
			IssuedBy: in.Actor.Id, Now: now,
		})
		if err := uc.d.Repository.Insert(ctx, report); err != nil {
			return err
		}
		s := report.Snapshot()
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityId: id, Code: code.String(), Field: "checksum", NewValue: s.Checksum,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindByCode(ctx, code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}
