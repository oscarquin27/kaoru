package usecase

import (
	"context"

	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateDefect edita un DEF y audita cada campo que cambió.
type UpdateDefect struct {
	d Deps
}

// UpdateDefectInput es la entrada de UpdateDefect. Un puntero nil deja el campo como está;
// AssigneeSet dice si se toca la asignación (vacío la quita).
type UpdateDefectInput struct {
	Actor       shared.Actor
	Code        string
	Title       *string
	Description *string
	Severity    *string
	AssigneeSet bool
	AssigneeId  string
}

// Execute exige el permiso def.edit.
func (uc *UpdateDefect) Execute(ctx context.Context, in UpdateDefectInput) (DefectOutput, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionEdit); err != nil {
		return DefectOutput{}, err
	}
	edit := domain.Edit{Title: in.Title, Description: in.Description, Severity: in.Severity}
	if in.AssigneeSet {
		assignee, err := findAssignee(ctx, uc.d.Repository, in.AssigneeId)
		if err != nil {
			return DefectOutput{}, err
		}
		edit.Assignee = domain.AssigneeEdit{Set: true, Value: assignee}
	}
	var defect domain.Defect
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if defect, err = findDefect(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		changes, err := defect.Apply(edit, uc.d.Clock.Now())
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.Update(ctx, defect); err != nil {
			return err
		}
		s := defect.Snapshot()
		for _, c := range changes {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, DefectId: s.Id, Code: s.Code, Action: port.AuditUpdate,
				Field: c.Field, OldValue: c.Old, NewValue: c.New,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return DefectOutput{}, err
	}
	return toOutput(ctx, uc.d, defect)
}
