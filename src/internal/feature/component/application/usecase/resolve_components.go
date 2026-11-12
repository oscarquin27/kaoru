package usecase

import (
	"context"
	"slices"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/component/domain"
)

// ResolveComponents convierte códigos de COMP en referencias para declarar impacto.
// Lo usan los features dueños del REQ y del TC.
type ResolveComponents struct {
	d Deps
}

// ResolveComponentsInput es la entrada de ResolveComponents: los códigos y el sistema al que deben pertenecer.
type ResolveComponentsInput struct {
	SystemKey string
	Codes     []string
}

// ComponentRef es un COMP visto desde la entidad que declara su impacto.
type ComponentRef struct {
	Id   string
	Code string
	Name string
	Kind domain.Kind
}

// Execute devuelve una referencia por cada código distinto, en el orden en que llegaron.
// Si un COMP no existe, es de otro sistema o está retirado, devuelve un INVALID_REQUEST en el campo `components`.
func (uc *ResolveComponents) Execute(ctx context.Context, in ResolveComponentsInput) ([]ComponentRef, error) {
	codes := make([]string, 0, len(in.Codes))
	for _, code := range in.Codes {
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}
	components, err := uc.d.Repository.FindByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	found := make(map[string]domain.Component, len(components))
	for _, c := range components {
		found[c.Snapshot().Code.String()] = c
	}
	refs := make([]ComponentRef, len(codes))
	for i, code := range codes {
		component, ok := found[code]
		s := component.Snapshot()
		switch {
		case !ok:
			return nil, invalidComponents("El componente " + code + " no existe.")
		case s.System.Key != in.SystemKey:
			return nil, invalidComponents("El componente " + code + " es de otro sistema.")
		case component.Retired():
			return nil, invalidComponents("El componente " + code + " está retirado.")
		}
		refs[i] = ComponentRef{Id: s.Id, Code: code, Name: s.Name, Kind: s.Kind}
	}
	return refs, nil
}

func invalidComponents(message string) error {
	return axis.Invalid(axis.InvalidField("components", message))
}
