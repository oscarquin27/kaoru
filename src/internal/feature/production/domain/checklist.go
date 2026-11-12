// Package domain modela lo que pasa en producción con un PKG: el checklist con sus firmas (CHK),
// el diagnóstico (DIAG) y la prueba de humo (SMOKE) posteriores al despliegue.
package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const maxStepTitleLength = 200

// KindProduction es el tipo de ambiente sobre el que se ejecuta un checklist.
const KindProduction = "production"

// Errores de dominio.
var (
	ErrStepNotFound      = axis.New(axis.NotFoundResource, "El checklist no tiene ese paso.")
	ErrStepAlreadySigned = axis.New(axis.InvalidAction, "El paso ya está firmado. Una firma no se cambia ni se quita.")
)

// System es el sistema bajo prueba al que pertenece una plantilla.
type System struct {
	Key          string
	Name         string
	DefaultRoute string
}

// UserRef es un usuario con el nombre que se muestra.
type UserRef struct {
	Id   shared.UserId
	Name string
}

// TemplateStep es un paso de la plantilla, en el orden en que se ejecuta.
type TemplateStep struct {
	Position    int
	Title       string
	Description string
}

// TemplateSnapshot es el contenido de una versión de la plantilla.
type TemplateSnapshot struct {
	Id        string
	System    System
	Version   int
	Steps     []TemplateStep
	CreatedBy UserRef
	CreatedAt time.Time
}

// Template es una versión de la plantilla del checklist de un sistema. No se edita: se publica otra.
type Template struct {
	s TemplateSnapshot
}

// StepInput es un paso tal como se escribe al publicar una versión.
type StepInput struct {
	Title       string
	Description string
}

// NewTemplateParams son los datos para publicar una versión de la plantilla.
type NewTemplateParams struct {
	Id        string
	System    System
	Version   int
	Steps     []StepInput
	CreatedBy shared.UserId
	Now       time.Time
}

// NewTemplate valida y publica una versión: al menos un paso, y cada paso con título.
func NewTemplate(p NewTemplateParams) (Template, error) {
	var errs axis.FieldErrors
	if len(p.Steps) == 0 {
		errs.Add(axis.Required("steps", "Agrega al menos un paso."))
	}
	steps := make([]TemplateStep, len(p.Steps))
	for i, in := range p.Steps {
		title := strings.TrimSpace(in.Title)
		field := fmt.Sprintf("steps.%d.title", i)
		switch {
		case title == "":
			errs.Add(axis.Required(field, "Escribe el título del paso."))
		case utf8.RuneCountInString(title) > maxStepTitleLength:
			errs.Add(axis.InvalidField(field, "El título del paso no puede pasar de 200 caracteres."))
		}
		steps[i] = TemplateStep{Position: i + 1, Title: title, Description: strings.TrimSpace(in.Description)}
	}
	if err := errs.Err(); err != nil {
		return Template{}, err
	}
	return Template{s: TemplateSnapshot{
		Id: p.Id, System: p.System, Version: p.Version, Steps: steps, CreatedBy: UserRef{Id: p.CreatedBy}, CreatedAt: p.Now,
	}}, nil
}

// RehydrateTemplate arma una versión ya guardada. Solo lo usa el adaptador de persistencia.
func RehydrateTemplate(s TemplateSnapshot) Template { return Template{s: s} }

// Snapshot devuelve el contenido de la versión.
func (t Template) Snapshot() TemplateSnapshot { return t.s }

// PackageRef es el PKG de un checklist, un DIAG o un SMOKE. SystemKey solo viene al validar.
type PackageRef struct {
	Id        string
	Code      string
	Title     string
	SystemKey string
}

// EnvironmentRef es un ambiente. SystemKey y Retired solo vienen al validar.
type EnvironmentRef struct {
	Id        string
	Code      string
	Name      string
	Kind      string
	SystemKey string
	Retired   bool
}

// Signature es la firma de un paso: el usuario de la sesión, la fecha del servidor y su comentario.
type Signature struct {
	Position int
	Comment  string
	SignedBy UserRef
	SignedAt time.Time
}

// Step es un paso de un checklist; Signature es nil mientras no esté firmado.
type Step struct {
	TemplateStep
	Signature *Signature
}

// ChecklistSnapshot es el contenido de un CHK.
type ChecklistSnapshot struct {
	Id          string
	Code        shared.Code
	Package     PackageRef
	Environment EnvironmentRef
	// TemplateId y TemplateVersion son la versión de la plantilla con la que se abrió.
	TemplateId      string
	TemplateVersion int
	Steps           []Step
	StartedBy       UserRef
	CreatedAt       time.Time
}

// Checklist es un CHK-: la ejecución de una versión de la plantilla para un PKG en producción.
type Checklist struct {
	s ChecklistSnapshot
}

// NewChecklistParams son los datos para abrir un CHK.
type NewChecklistParams struct {
	Id          string
	Code        shared.Code
	Package     PackageRef
	Environment EnvironmentRef
	Template    TemplateSnapshot
	StartedBy   shared.UserId
	Now         time.Time
}

// NewChecklist valida y abre un CHK con los pasos de la plantilla, sin firmar. El ambiente debe
// ser de producción, estar en uso y pertenecer al sistema del PKG.
func NewChecklist(p NewChecklistParams) (Checklist, error) {
	var errs axis.FieldErrors
	switch env := p.Environment; {
	case env.Kind != KindProduction:
		errs.Add(axis.InvalidField("environment", "El checklist se ejecuta en un ambiente de producción."))
	case env.Retired:
		errs.Add(axis.InvalidField("environment", "El ambiente está retirado."))
	case env.SystemKey != p.Package.SystemKey:
		errs.Add(axis.InvalidField("environment", "El ambiente es de otro sistema que el PKG."))
	}
	if err := errs.Err(); err != nil {
		return Checklist{}, err
	}
	steps := make([]Step, len(p.Template.Steps))
	for i, s := range p.Template.Steps {
		steps[i] = Step{TemplateStep: s}
	}
	return Checklist{s: ChecklistSnapshot{
		Id: p.Id, Code: p.Code, Package: p.Package, Environment: p.Environment,
		TemplateId: p.Template.Id, TemplateVersion: p.Template.Version, Steps: steps,
		StartedBy: UserRef{Id: p.StartedBy}, CreatedAt: p.Now,
	}}, nil
}

// RehydrateChecklist arma un CHK ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateChecklist(s ChecklistSnapshot) Checklist { return Checklist{s: s} }

// Snapshot devuelve el contenido del CHK.
func (c Checklist) Snapshot() ChecklistSnapshot { return c.s }

// Sign firma un paso. Cada paso se firma una sola vez; el orden entre pasos es libre.
func (c *Checklist) Sign(position int, by shared.UserId, comment string, now time.Time) (Signature, error) {
	for i, step := range c.s.Steps {
		if step.Position != position {
			continue
		}
		if step.Signature != nil {
			return Signature{}, ErrStepAlreadySigned
		}
		signature := Signature{Position: position, Comment: strings.TrimSpace(comment), SignedBy: UserRef{Id: by}, SignedAt: now}
		c.s.Steps[i].Signature = &signature
		return signature, nil
	}
	return Signature{}, ErrStepNotFound
}

// SignedSteps cuenta los pasos firmados.
func (c Checklist) SignedSteps() int {
	n := 0
	for _, step := range c.s.Steps {
		if step.Signature != nil {
			n++
		}
	}
	return n
}

// Complete dice si todos los pasos están firmados.
func (c Checklist) Complete() bool { return c.SignedSteps() == len(c.s.Steps) }
