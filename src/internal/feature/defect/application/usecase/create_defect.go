package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateDefect abre un DEF ligado a un REQ y al RUN que lo reprodujo.
type CreateDefect struct {
	d Deps
}

// CreateDefectInput es la entrada de CreateDefect. AssigneeId es opcional.
type CreateDefectInput struct {
	Actor           shared.Actor
	Title           string
	Description     string
	RequirementCode string
	RunCode         string
	Severity        string
	AssigneeId      string
}

// Execute exige el permiso def.create.
func (uc *CreateDefect) Execute(ctx context.Context, in CreateDefectInput) (DefectOutput, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionCreate); err != nil {
		return DefectOutput{}, err
	}
	var errs axis.FieldErrors
	requirement, err := uc.d.Repository.FindRequirement(ctx, in.RequirementCode)
	switch {
	case err == nil:
	case !axis.IsCode(err, axis.NotFoundResource):
		return DefectOutput{}, err
	case in.RequirementCode == "":
		errs.Add(axis.Required("requirement", "Indica el REQ afectado."))
	default:
		errs.Add(axis.InvalidField("requirement", "El REQ no existe."))
	}
	run, err := uc.d.Repository.FindRun(ctx, in.RunCode)
	switch {
	case err == nil:
	case !axis.IsCode(err, axis.NotFoundResource):
		return DefectOutput{}, err
	case in.RunCode == "":
		errs.Add(axis.Required("run", "Indica el RUN que reprodujo el defecto."))
	default:
		errs.Add(axis.InvalidField("run", "El RUN no existe."))
	}
	if err := errs.Err(); err != nil {
		return DefectOutput{}, err
	}
	testCases, err := uc.d.TestDesign.RequirementTestCases(ctx, requirement.Code)
	if err != nil {
		return DefectOutput{}, err
	}
	assignee, err := findAssignee(ctx, uc.d.Repository, in.AssigneeId)
	if err != nil {
		return DefectOutput{}, err
	}
	creator, err := uc.d.Repository.FindUser(ctx, in.Actor.Id)
	if err != nil {
		return DefectOutput{}, err
	}
	state, err := uc.d.Workflow.InitialState(ctx)
	if err != nil {
		return DefectOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return DefectOutput{}, err
	}
	now := uc.d.Clock.Now()
	var defect domain.Defect
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		defect, err = domain.New(domain.NewParams{
			Id: id, Code: code, Title: in.Title, Description: in.Description, Requirement: requirement, Run: run,
			RequirementTestCases: testCases,
			Severity:             in.Severity, State: state, Assignee: assignee, CreatedBy: creator, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, defect); err != nil {
			return err
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, DefectId: id, Code: code, Action: port.AuditCreate,
			Field: "severity", NewValue: string(defect.Snapshot().Severity),
		})
	})
	if err != nil {
		return DefectOutput{}, err
	}
	return DefectOutput{Defect: defect.Snapshot()}, nil
}
