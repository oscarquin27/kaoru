// Package domain modela los flujos configurables: estados, transiciones, permisos y reglas.
package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"kaoru/internal/core/pkg/axis"
)

// EntityType es el tipo de entidad que tiene un ciclo de vida.
type EntityType string

// Tipos de entidad con flujo.
const (
	EntityRequirement EntityType = "requirement"
	EntityChange      EntityType = "change"
	EntityDefect      EntityType = "defect"
)

// Errores de dominio.
var (
	ErrUnknownEntityType = axis.New(axis.NotFoundResource, "No existe un flujo para ese tipo de entidad.")
	ErrPermissionDenied  = axis.New(axis.PermissionDenied, "Tu rol no tiene permiso para esta acción.")
	ErrNoInitialState    = axis.New(axis.PreconditionFailed, "El flujo no tiene un estado inicial configurado.")
	ErrUnknownState      = axis.New(axis.InvalidRequest, "El estado no existe en este flujo.")
	ErrStateNotFound     = axis.New(axis.NotFoundResource, "El estado no existe en este flujo.")
	ErrTransitionMissing = axis.New(axis.NotFoundResource, "La transición no existe en este flujo.")
)

// ParseEntityType valida el tipo de entidad.
func ParseEntityType(raw string) (EntityType, error) {
	switch t := EntityType(raw); t {
	case EntityRequirement, EntityChange, EntityDefect:
		return t, nil
	default:
		return "", ErrUnknownEntityType
	}
}

// State es un estado de un flujo.
type State struct {
	Id   string
	Key  string
	Name string
	// Position es el orden en la línea de vida; nil en un estado lateral.
	Position  *int
	IsInitial bool
	IsFinal   bool
}

// Rule es una regla de bloqueo de una transición. El feature dueño de la entidad la evalúa.
type Rule struct {
	Id     string
	Key    string
	Params map[string]any
	// Route limita la regla a una ruta del CHG; vacío aplica a todas.
	Route string
	// Blocks en falso significa que la regla solo avisa.
	Blocks  bool
	Enabled bool
}

// Transition es un paso permitido entre dos estados.
type Transition struct {
	From           string
	To             string
	Permission     string
	RequiresReason bool
	Rules          []Rule
}

// Grant es un permiso asignado a un rol.
type Grant struct {
	Role       string
	Permission string
}

// Allows dice si alguno de los permisos concedidos es el que se pide.
func Allows(grants []Grant, permission string) bool {
	return slices.ContainsFunc(grants, func(g Grant) bool { return g.Permission == permission })
}

// Workflow es el flujo completo de un tipo de entidad.
type Workflow struct {
	EntityType  EntityType
	States      []State
	Transitions []Transition
}

// Initial devuelve el estado en el que nace una entidad.
func (w Workflow) Initial() (State, error) {
	for _, s := range w.States {
		if s.IsInitial {
			return s, nil
		}
	}
	return State{}, ErrNoInitialState
}

// State busca un estado por su clave.
func (w Workflow) State(key string) (State, error) {
	for _, s := range w.States {
		if s.Key == key {
			return s, nil
		}
	}
	return State{}, ErrUnknownState
}

// Transition devuelve la transición entre dos estados, o INVALID_ACTION si el flujo no la permite.
func (w Workflow) Transition(from, to string) (Transition, error) {
	target, err := w.State(to)
	if err != nil {
		return Transition{}, axis.Invalid(axis.InvalidField("to", ErrUnknownState.Error()))
	}
	for _, t := range w.Transitions {
		if t.From == from && t.To == to {
			return t, nil
		}
	}
	origin, _ := w.State(from)
	return Transition{}, axis.New(axis.InvalidAction,
		fmt.Sprintf("No se puede pasar de «%s» a «%s».", origin.Name, target.Name))
}

// Roles que Kaoru reconoce. Nacen en Keycloak; aquí solo se les asignan permisos.
var Roles = []string{"change-manager", "qa-admin", "qa", "dev", "ops", "auditor"}

const (
	// PermissionConfigure es el permiso para configurar flujos, reglas y permisos.
	PermissionConfigure = "admin.configure"
	// RoleAdmin es el rol que no puede quedarse sin PermissionConfigure.
	RoleAdmin = "qa-admin"
)

// Permission es un permiso asignable a roles.
type Permission struct {
	Key         string
	Description string
}

// ValidateRolePermissions comprueba un cambio de permisos de un rol: el rol y los permisos
// deben existir, y el rol administrador no puede perder el permiso de configurar,
// porque nadie podría devolvérselo desde la aplicación.
func ValidateRolePermissions(role string, permissions []string, known []Permission) ([]string, error) {
	if !slices.Contains(Roles, role) {
		return nil, axis.New(axis.NotFoundResource, "El rol no existe.")
	}
	out := make([]string, 0, len(permissions))
	for _, key := range permissions {
		if !slices.ContainsFunc(known, func(p Permission) bool { return p.Key == key }) {
			return nil, axis.Invalid(axis.InvalidField("permissions", "El permiso «"+key+"» no existe."))
		}
		if !slices.Contains(out, key) {
			out = append(out, key)
		}
	}
	if role == RoleAdmin && !slices.Contains(out, PermissionConfigure) {
		return nil, axis.New(axis.InvalidAction, "El rol qa-admin no puede quedarse sin el permiso de configurar.")
	}
	slices.Sort(out)
	return out, nil
}

// maxStateName es el largo máximo del nombre de un estado: debe caber en una etiqueta.
const maxStateName = 60

var stateKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,39}$`)

func validStateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", axis.Invalid(axis.Required("name", "El nombre es obligatorio."))
	}
	if len([]rune(name)) > maxStateName {
		return "", axis.Invalid(axis.InvalidField("name", fmt.Sprintf("El nombre admite hasta %d caracteres.", maxStateName)))
	}
	return name, nil
}

func validPosition(position *int) error {
	if position != nil && *position < 1 {
		return axis.Invalid(axis.InvalidField("position", "La posición en la línea de vida empieza en 1."))
	}
	return nil
}

// NewState valida un estado que se agrega al flujo. Nunca nace como inicial:
// el estado inicial lo fija la instalación.
func (w Workflow) NewState(key, name string, position *int, isFinal bool) (State, error) {
	if !stateKeyPattern.MatchString(key) {
		return State{}, axis.Invalid(axis.InvalidField("key",
			"La clave va en minúsculas, empieza con una letra y admite letras, números y guion bajo (2 a 40)."))
	}
	if _, err := w.State(key); err == nil {
		return State{}, axis.New(axis.DuplicateResource, "Ya existe un estado con esa clave en este flujo.")
	}
	name, err := validStateName(name)
	if err != nil {
		return State{}, err
	}
	if err := validPosition(position); err != nil {
		return State{}, err
	}
	return State{Key: key, Name: name, Position: position, IsFinal: isFinal}, nil
}

// StateEdit son los cambios a un estado; nil deja el nombre como está y
// PositionSet en falso deja la posición como está.
type StateEdit struct {
	Name        *string
	PositionSet bool
	Position    *int
}

// EditState devuelve el estado con los cambios aplicados. La clave no cambia:
// es la que referencian las transiciones y el historial.
func (w Workflow) EditState(key string, edit StateEdit) (State, error) {
	state, err := w.State(key)
	if err != nil {
		return State{}, ErrStateNotFound
	}
	if edit.Name != nil {
		if state.Name, err = validStateName(*edit.Name); err != nil {
			return State{}, err
		}
	}
	if edit.PositionSet {
		if err := validPosition(edit.Position); err != nil {
			return State{}, err
		}
		state.Position = edit.Position
	}
	return state, nil
}

// RemovableState dice si un estado se puede quitar: no puede ser el inicial ni tener transiciones.
// Que ningún registro esté en ese estado lo garantiza la base.
func (w Workflow) RemovableState(key string) (State, error) {
	state, err := w.State(key)
	if err != nil {
		return State{}, ErrStateNotFound
	}
	if state.IsInitial {
		return State{}, axis.New(axis.InvalidAction, "El estado inicial no se puede quitar.")
	}
	if slices.ContainsFunc(w.Transitions, func(t Transition) bool { return t.From == key || t.To == key }) {
		return State{}, axis.New(axis.InvalidAction, "Quita primero las transiciones que entran o salen de este estado.")
	}
	return state, nil
}

// FindTransition busca la transición entre dos estados.
func (w Workflow) FindTransition(from, to string) (Transition, bool) {
	i := slices.IndexFunc(w.Transitions, func(t Transition) bool { return t.From == from && t.To == to })
	if i < 0 {
		return Transition{}, false
	}
	return w.Transitions[i], true
}

// NewTransition valida una transición que se agrega o se ajusta: los dos estados deben existir
// y ser distintos, y el permiso debe existir.
func (w Workflow) NewTransition(from, to, permission string, requiresReason bool, known []Permission) (Transition, error) {
	if _, err := w.State(from); err != nil {
		return Transition{}, ErrStateNotFound
	}
	if _, err := w.State(to); err != nil {
		return Transition{}, ErrStateNotFound
	}
	if from == to {
		return Transition{}, axis.New(axis.InvalidAction, "Una transición va de un estado a otro distinto.")
	}
	if permission == "" {
		return Transition{}, axis.Invalid(axis.Required("permission", "El permiso es obligatorio."))
	}
	if !slices.ContainsFunc(known, func(p Permission) bool { return p.Key == permission }) {
		return Transition{}, axis.Invalid(axis.InvalidField("permission", "El permiso «"+permission+"» no existe."))
	}
	return Transition{From: from, To: to, Permission: permission, RequiresReason: requiresReason}, nil
}
