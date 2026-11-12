// Package changes implementa el puerto Changes con el caso de uso ListWindows de change.
package changes

import (
	"context"

	changeuc "kaoru/internal/feature/change/application/usecase"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
)

// Changes lee las ventanas de los CHG vigentes.
type Changes struct {
	list *changeuc.ListWindows
}

var _ port.Changes = (*Changes)(nil)

// New crea el adaptador.
func New(list *changeuc.ListWindows) *Changes { return &Changes{list: list} }

// Windows devuelve las ventanas por fecha de inicio.
func (c *Changes) Windows(ctx context.Context) ([]domain.Window, error) {
	out, err := c.list.Execute(ctx)
	if err != nil {
		return nil, err
	}
	windows := make([]domain.Window, 0, len(out))
	for _, o := range out {
		s := o.Change
		if s.WindowStart == nil || s.WindowEnd == nil {
			continue
		}
		windows = append(windows, domain.Window{
			ChangeCode: s.Code.String(), ChangeTitle: s.Title, System: s.System.Name, Route: string(s.Route),
			StateKey: s.State.Key, StateName: s.State.Name, Start: s.WindowStart.UTC(), End: s.WindowEnd.UTC(),
		})
	}
	return windows, nil
}
