// Package domain modela el PKG: un conjunto de BUILD y REQ listo para desplegar, su contenido
// y las reglas que deciden si puede avanzar.
package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

const maxTitleLength = 200

// Estados del REQ desde los que puede entrar a un PKG. Si está «Aprobado QA», entrar lo pasa a «En paquete».
const (
	RequirementApprovedQA = "approved_qa"
	RequirementInPackage  = "in_package"
)

// Errores de dominio.
var (
	ErrNotInAssembly          = axis.New(axis.InvalidAction, "El contenido de un PKG solo cambia mientras está en armado.")
	ErrBuildOtherSystem       = axis.New(axis.InvalidAction, "El BUILD es de otro sistema que el PKG.")
	ErrRequirementOtherSystem = axis.New(axis.InvalidAction, "El REQ es de otro sistema que el PKG.")
	ErrRequirementNotApproved = axis.New(axis.InvalidAction, "Solo entra a un PKG un REQ «Aprobado QA» o «En paquete».")
)

// State es el estado del PKG en su flujo configurable.
type State struct {
	Id   string
	Key  string
	Name string
	// IsInitial dice si es el estado en el que nace un PKG: «En armado».
	IsInitial bool
}

// System es el sistema bajo prueba al que pertenece el PKG.
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

// BuildRef es un BUILD del PKG.
type BuildRef struct {
	Id       string
	Code     string
	Version  string
	Checksum string
}

// Build es un BUILD del catálogo que se quiere agregar a un PKG.
type Build struct {
	Ref       BuildRef
	SystemKey string
}

// RequirementState es el estado de un REQ, para mostrarlo.
type RequirementState struct {
	Key  string
	Name string
}

// RequirementRef es un REQ del PKG con su estado actual.
type RequirementRef struct {
	Id    string
	Code  string
	Title string
	State RequirementState
}

// Requirement es un REQ que se quiere agregar a un PKG.
type Requirement struct {
	Ref       RequirementRef
	SystemKey string
}

// Snapshot es el contenido de un PKG. Con él se crea, se rehidrata y se lee.
type Snapshot struct {
	Id          string
	Code        shared.Code
	Title       string
	Description string
	System      System
	State       State
	// Builds y Requirements van por código.
	Builds       []BuildRef
	Requirements []RequirementRef
	CreatedBy    UserRef
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Package es un PKG.
type Package struct {
	s Snapshot
}

// NewParams son los datos para crear un PKG.
type NewParams struct {
	Id          string
	Code        shared.Code
	Title       string
	Description string
	System      System
	State       State
	CreatedBy   UserRef
	Now         time.Time
}

// New valida y crea un PKG vacío en el estado inicial de su flujo.
func New(p NewParams) (Package, error) {
	var errs axis.FieldErrors
	title := validTitle(p.Title, &errs)
	if err := errs.Err(); err != nil {
		return Package{}, err
	}
	return Package{s: Snapshot{
		Id: p.Id, Code: p.Code, Title: title, Description: strings.TrimSpace(p.Description),
		System: p.System, State: p.State, CreatedBy: p.CreatedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}}, nil
}

// Rehydrate arma un PKG ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Package { return Package{s: s} }

// Snapshot devuelve el contenido del PKG.
func (p Package) Snapshot() Snapshot { return p.s }

// Edit son los campos a cambiar; un puntero nil deja el campo como está. El sistema no cambia.
type Edit struct {
	Title       *string
	Description *string
}

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (p *Package) Apply(e Edit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := p.s
	if e.Title != nil {
		next.Title = validTitle(*e.Title, &errs)
	}
	if e.Description != nil {
		next.Description = strings.TrimSpace(*e.Description)
	}
	if err := errs.Err(); err != nil {
		return nil, err
	}
	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("title", p.s.Title, next.Title)
	add("description", p.s.Description, next.Description)
	if len(changes) > 0 {
		next.UpdatedAt = now
		p.s = next
	}
	return changes, nil
}

// MoveTo pasa el PKG a otro estado. Que la transición valga lo deciden el flujo y sus reglas.
func (p *Package) MoveTo(state State, now time.Time) FieldChange {
	change := FieldChange{Field: "state", Old: p.s.State.Key, New: state.Key}
	p.s.State = state
	p.s.UpdatedAt = now
	return change
}

// AddBuild agrega un BUILD del mismo sistema. Devuelve false si ya estaba: no cambia nada.
func (p *Package) AddBuild(build Build, now time.Time) (bool, error) {
	if !p.s.State.IsInitial {
		return false, ErrNotInAssembly
	}
	if p.hasBuild(build.Ref.Code) {
		return false, nil
	}
	if build.SystemKey != p.s.System.Key {
		return false, ErrBuildOtherSystem
	}
	p.s.Builds = append(slices.Clone(p.s.Builds), build.Ref)
	slices.SortFunc(p.s.Builds, func(a, b BuildRef) int { return strings.Compare(a.Code, b.Code) })
	p.s.UpdatedAt = now
	return true, nil
}

// RemoveBuild quita un BUILD. Devuelve false si no estaba: no cambia nada.
func (p *Package) RemoveBuild(code string, now time.Time) (bool, error) {
	if !p.s.State.IsInitial {
		return false, ErrNotInAssembly
	}
	if !p.hasBuild(code) {
		return false, nil
	}
	p.s.Builds = slices.DeleteFunc(slices.Clone(p.s.Builds), func(b BuildRef) bool { return b.Code == code })
	p.s.UpdatedAt = now
	return true, nil
}

// AddRequirement agrega un REQ del mismo sistema que esté «Aprobado QA» o «En paquete».
// Devuelve false si ya estaba: no cambia nada.
func (p *Package) AddRequirement(requirement Requirement, now time.Time) (bool, error) {
	if !p.s.State.IsInitial {
		return false, ErrNotInAssembly
	}
	if p.hasRequirement(requirement.Ref.Code) {
		return false, nil
	}
	if requirement.SystemKey != p.s.System.Key {
		return false, ErrRequirementOtherSystem
	}
	if key := requirement.Ref.State.Key; key != RequirementApprovedQA && key != RequirementInPackage {
		return false, ErrRequirementNotApproved
	}
	p.s.Requirements = append(slices.Clone(p.s.Requirements), requirement.Ref)
	slices.SortFunc(p.s.Requirements, func(a, b RequirementRef) int { return strings.Compare(a.Code, b.Code) })
	p.s.UpdatedAt = now
	return true, nil
}

// RemoveRequirement quita un REQ; su estado no cambia. Devuelve false si no estaba.
func (p *Package) RemoveRequirement(code string, now time.Time) (bool, error) {
	if !p.s.State.IsInitial {
		return false, ErrNotInAssembly
	}
	if !p.hasRequirement(code) {
		return false, nil
	}
	p.s.Requirements = slices.DeleteFunc(slices.Clone(p.s.Requirements), func(r RequirementRef) bool { return r.Code == code })
	p.s.UpdatedAt = now
	return true, nil
}

func (p Package) hasBuild(code string) bool {
	return slices.ContainsFunc(p.s.Builds, func(b BuildRef) bool { return b.Code == code })
}

func (p Package) hasRequirement(code string) bool {
	return slices.ContainsFunc(p.s.Requirements, func(r RequirementRef) bool { return r.Code == code })
}

func validTitle(raw string, errs *axis.FieldErrors) string {
	title := strings.TrimSpace(raw)
	switch {
	case title == "":
		errs.Add(axis.Required("title", "Escribe un título."))
	case utf8.RuneCountInString(title) > maxTitleLength:
		errs.Add(axis.InvalidField("title", "El título no puede pasar de 200 caracteres."))
	}
	return title
}
