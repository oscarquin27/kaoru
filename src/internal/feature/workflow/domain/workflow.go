// Package domain modela los flujos configurables: estados, transiciones, permisos y reglas.
package domain

import (
	"fmt"
	"slices"

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
