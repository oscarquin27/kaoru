package postgres

import "time"

// utc normaliza a UTC una fecha opcional leída de la base.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
