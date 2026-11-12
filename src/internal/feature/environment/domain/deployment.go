package domain

import (
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Result es el resultado de un despliegue.
type Result string

// Resultados de un despliegue.
const (
	ResultSuccess    Result = "success"
	ResultFailed     Result = "failed"
	ResultRolledBack Result = "rolled_back"
)

// EnvironmentRef es el ambiente de un despliegue.
type EnvironmentRef struct {
	Id   string
	Code string
	Name string
	Kind Kind
}

// BuildRef es un BUILD desplegado. SystemKey solo viene al registrar, para validar el sistema.
type BuildRef struct {
	Id        string
	Code      string
	Version   string
	Checksum  string
	SystemKey string
}

// PackageRef es un PKG desplegado. SystemKey solo viene al registrar, para validar el sistema.
type PackageRef struct {
	Id        string
	Code      string
	Title     string
	SystemKey string
}

// DeploymentSnapshot es el contenido de un DEP. Lleva un BUILD o un PKG, nunca los dos ni ninguno.
type DeploymentSnapshot struct {
	Id          string
	Code        shared.Code
	Environment EnvironmentRef
	Build       *BuildRef
	Package     *PackageRef
	Result      Result
	Notes       string
	ExecutedBy  UserRef
	ExecutedAt  time.Time
	CreatedAt   time.Time
}

// Deployment es un DEP-: el registro de un despliegue. No se corrige: se registra otro.
type Deployment struct {
	s DeploymentSnapshot
}

// NewDeploymentParams son los datos para registrar un DEP.
type NewDeploymentParams struct {
	Id          string
	Code        shared.Code
	Environment Snapshot
	Build       *BuildRef
	Package     *PackageRef
	Result      string
	Notes       string
	ExecutedBy  shared.UserId
	ExecutedAt  time.Time
	Now         time.Time
}

// NewDeployment valida y registra un DEP: el ambiente no puede estar retirado, y lo que se
// despliega (un BUILD o un PKG, exactamente uno) debe ser del sistema del ambiente.
func NewDeployment(p NewDeploymentParams) (Deployment, error) {
	var errs axis.FieldErrors
	env := p.Environment
	s := DeploymentSnapshot{
		Id: p.Id, Code: p.Code,
		Environment: EnvironmentRef{Id: env.Id, Code: env.Code.String(), Name: env.Name, Kind: env.Kind},
		Build:       p.Build, Package: p.Package, Result: Result(p.Result), Notes: strings.TrimSpace(p.Notes),
		ExecutedBy: UserRef{Id: p.ExecutedBy}, ExecutedAt: p.ExecutedAt.UTC(), CreatedAt: p.Now,
	}
	if env.RetiredAt != nil {
		errs.Add(axis.InvalidField("environment", "El ambiente está retirado: no admite despliegues."))
	}
	switch {
	case p.Build == nil && p.Package == nil:
		errs.Add(axis.Required("build", "Indica el BUILD o el PKG que se desplegó."))
	case p.Build != nil && p.Package != nil:
		errs.Add(axis.InvalidField("package", "Un despliegue lleva un BUILD o un PKG, no los dos."))
	case p.Build != nil && p.Build.SystemKey != env.System.Key:
		errs.Add(axis.InvalidField("build", "El BUILD es de otro sistema que el ambiente."))
	case p.Package != nil && p.Package.SystemKey != env.System.Key:
		errs.Add(axis.InvalidField("package", "El PKG es de otro sistema que el ambiente."))
	}
	switch s.Result {
	case ResultSuccess, ResultFailed, ResultRolledBack:
	case "":
		errs.Add(axis.Required("result", "Indica el resultado."))
	default:
		errs.Add(axis.InvalidField("result", "El resultado debe ser exitoso, fallido o revertido."))
	}
	switch {
	case p.ExecutedAt.IsZero():
		errs.Add(axis.Required("executedAt", "Indica cuándo se desplegó."))
	case p.ExecutedAt.After(p.Now):
		errs.Add(axis.InvalidField("executedAt", "La fecha del despliegue no puede ser futura."))
	}
	if err := errs.Err(); err != nil {
		return Deployment{}, err
	}
	return Deployment{s: s}, nil
}

// RehydrateDeployment arma un DEP ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateDeployment(s DeploymentSnapshot) Deployment { return Deployment{s: s} }

// Snapshot devuelve el contenido del DEP.
func (d Deployment) Snapshot() DeploymentSnapshot { return d.s }

// Installed es lo que dejó en un ambiente su último despliegue exitoso.
type Installed struct {
	Deployment string
	Build      *BuildRef
	Package    *PackageRef
	DeployedAt time.Time
}
