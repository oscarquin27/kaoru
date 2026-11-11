// Package domain modela el RISK: el nivel de riesgo de un REQ, que define cuánta prueba necesita.
package domain

import (
	"slices"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	shared "kaoru/internal/shared/domain"
)

// Level es el nivel de riesgo.
type Level string

// Niveles de riesgo.
const (
	LevelHigh   Level = "high"
	LevelMedium Level = "medium"
	LevelLow    Level = "low"
)

// Characteristic es una característica de calidad de ISO/IEC 25010.
type Characteristic string

// Características de calidad (ISO/IEC 25010:2011).
var characteristics = []Characteristic{
	"compatibility", "functional_suitability", "maintainability", "performance_efficiency",
	"portability", "reliability", "security", "usability",
}

// Snapshot es el contenido de un RISK.
type Snapshot struct {
	Id              string
	Code            shared.Code
	RequirementId   string
	Level           Level
	Justification   string
	Characteristics []Characteristic
	AssessedBy      shared.UserId
	AssessedByName  string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Assessment es el RISK vigente de un REQ.
type Assessment struct {
	s Snapshot
}

// Input son los datos de una evaluación de riesgo.
type Input struct {
	Level           string
	Justification   string
	Characteristics []string
	AssessedBy      shared.UserId
	Now             time.Time
}

// New valida y crea el RISK de un REQ.
func New(id string, code shared.Code, requirementId string, in Input) (Assessment, error) {
	level, justification, chars, err := validate(in)
	if err != nil {
		return Assessment{}, err
	}
	return Assessment{s: Snapshot{
		Id: id, Code: code, RequirementId: requirementId, Level: level, Justification: justification,
		Characteristics: chars, AssessedBy: in.AssessedBy, CreatedAt: in.Now, UpdatedAt: in.Now,
	}}, nil
}

// Rehydrate arma un RISK ya guardado. Solo lo usa el adaptador de persistencia.
func Rehydrate(s Snapshot) Assessment { return Assessment{s: s} }

// Snapshot devuelve el contenido del RISK.
func (a Assessment) Snapshot() Snapshot { return a.s }

// FieldChange es un campo que cambió, para la auditoría.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// Reassess reemplaza la evaluación. Devuelve solo los campos que cambiaron de verdad.
func (a *Assessment) Reassess(in Input) ([]FieldChange, error) {
	level, justification, chars, err := validate(in)
	if err != nil {
		return nil, err
	}
	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("level", string(a.s.Level), string(level))
	add("justification", a.s.Justification, justification)
	add("characteristics", join(a.s.Characteristics), join(chars))
	if len(changes) > 0 {
		a.s.Level, a.s.Justification, a.s.Characteristics = level, justification, chars
		a.s.AssessedBy, a.s.UpdatedAt = in.AssessedBy, in.Now
	}
	return changes, nil
}

func validate(in Input) (Level, string, []Characteristic, error) {
	var errs axis.FieldErrors
	level := Level(in.Level)
	switch {
	case in.Level == "":
		errs.Add(axis.Required("level", "Elige el nivel de riesgo."))
	case level != LevelHigh && level != LevelMedium && level != LevelLow:
		errs.Add(axis.InvalidField("level", "El nivel debe ser alto, medio o bajo."))
	}
	justification := strings.TrimSpace(in.Justification)
	if justification == "" {
		errs.Add(axis.Required("justification", "Explica por qué ese nivel."))
	}
	chars := make([]Characteristic, 0, len(in.Characteristics))
	for _, raw := range in.Characteristics {
		c := Characteristic(raw)
		if !slices.Contains(characteristics, c) {
			errs.Add(axis.InvalidField("characteristics", "«"+raw+"» no es una característica de ISO/IEC 25010."))
			break
		}
		if !slices.Contains(chars, c) {
			chars = append(chars, c)
		}
	}
	slices.Sort(chars)
	return level, justification, chars, errs.Err()
}

func join(chars []Characteristic) string {
	parts := make([]string, len(chars))
	for i, c := range chars {
		parts[i] = string(c)
	}
	return strings.Join(parts, ",")
}
