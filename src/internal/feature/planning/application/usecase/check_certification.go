package usecase

import (
	"context"

	"kaoru/internal/feature/planning/domain"
)

// CheckCertification dice cómo está planificada hoy la certificación de un CHG. Lo usa la regla
// certification_planned del REQ al entrar en pruebas.
type CheckCertification struct {
	d Deps
}

// Execute lee las actividades de certificación del CHG. Un CHG sin ninguna da CertificationNone.
func (uc *CheckCertification) Execute(ctx context.Context, changeCode string) (domain.Certification, error) {
	activities, err := uc.d.Repository.Certifications(ctx, changeCode)
	if err != nil {
		return domain.Certification{}, err
	}
	return domain.CheckCertification(activities, uc.d.Clock.Now()), nil
}
