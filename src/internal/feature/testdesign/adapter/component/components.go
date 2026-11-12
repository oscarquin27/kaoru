// Package component implementa el puerto Components con los casos de uso de component.
package component

import (
	"context"

	componentuc "kaoru/internal/feature/component/application/usecase"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
)

// Components valida componentes en el feature component.
type Components struct {
	resolve *componentuc.ResolveComponents
}

var _ port.Components = (*Components)(nil)

// New crea el adaptador.
func New(resolve *componentuc.ResolveComponents) *Components { return &Components{resolve: resolve} }

// Resolve conserva el código y los campos del error de component.
func (c *Components) Resolve(ctx context.Context, systemKey string, codes []string) ([]domain.ComponentRef, error) {
	out, err := c.resolve.Execute(ctx, componentuc.ResolveComponentsInput{SystemKey: systemKey, Codes: codes})
	if err != nil {
		return nil, err
	}
	refs := make([]domain.ComponentRef, len(out))
	for i, r := range out {
		refs[i] = domain.ComponentRef{Id: r.Id, Code: r.Code, Name: r.Name, Kind: string(r.Kind)}
	}
	return refs, nil
}
