package domain

import "time"

// DefectFilter acota los DEF por los que se mide. Un campo vacío no filtra; el período es el de su creación.
type DefectFilter struct {
	SystemKey string
	Severity  string
	Period    Period
}

// WorkflowState es un estado del flujo del DEF, en el orden de su línea de vida.
type WorkflowState struct {
	State   StateRef
	IsFinal bool
}

// Transition es un cambio de estado tal como lo guardó la auditoría.
type Transition struct {
	At   time.Time
	From string
	To   string
}

// DefectHistory es un DEF con su historia de estados, de la más antigua a la más reciente.
type DefectHistory struct {
	Id          string
	CreatedAt   time.Time
	Current     string
	Transitions []Transition
}

// segment es un tramo de la historia: un estado, desde cuándo y hasta cuándo.
type segment struct {
	state      string
	start, end time.Time
}

// segments parte la historia en tramos. El DEF nació en el estado del que salió su primera
// transición; si nunca cambió, en el actual. El último tramo llega hasta now.
func (h DefectHistory) segments(now time.Time) []segment {
	state, start := h.Current, h.CreatedAt
	if len(h.Transitions) > 0 {
		state = h.Transitions[0].From
	}
	out := make([]segment, 0, len(h.Transitions)+1)
	for _, t := range h.Transitions {
		out = append(out, segment{state: state, start: start, end: t.At})
		state, start = t.To, t.At
	}
	return append(out, segment{state: state, start: start, end: now})
}

// StateMetric resume un estado del flujo del DEF.
type StateMetric struct {
	State   StateRef
	IsFinal bool
	// Current son los DEF que están hoy en el estado.
	Current int
	// Defects son los DEF que pasaron por el estado; Visits, las veces que se entró.
	Defects int
	Visits  int
	// MedianDays y P90Days son del tiempo total que cada DEF pasó en el estado. nil en un estado final,
	// donde quedarse no es una demora, o si nadie pasó por él.
	MedianDays *float64
	P90Days    *float64
}

// DefectStateMetrics es el tiempo en estado de un conjunto de DEF.
type DefectStateMetrics struct {
	Total  int
	States []StateMetric
	// Resolved son los DEF que hoy están en un estado final; Resolution*, el tiempo desde su creación
	// hasta que entraron a ese estado.
	Resolved             int
	ResolutionMedianDays *float64
	ResolutionP90Days    *float64
}

// MeasureDefectStates calcula el tiempo en estado. Una clave que el flujo ya no tiene se muestra con
// su clave como nombre, al final, para no perder lo medido.
func MeasureDefectStates(states []WorkflowState, histories []DefectHistory, now time.Time) DefectStateMetrics {
	out := DefectStateMetrics{Total: len(histories)}
	index := make(map[string]int, len(states))
	for _, s := range states {
		index[s.State.Key] = len(out.States)
		out.States = append(out.States, StateMetric{State: s.State, IsFinal: s.IsFinal})
	}
	at := func(key string) int {
		if i, ok := index[key]; ok {
			return i
		}
		index[key] = len(out.States)
		out.States = append(out.States, StateMetric{State: StateRef{Key: key, Name: key}})
		return index[key]
	}
	times := map[int][]float64{}
	var resolutions []float64
	for _, h := range histories {
		spent := map[int]time.Duration{}
		for _, seg := range h.segments(now) {
			i := at(seg.state)
			out.States[i].Visits++
			spent[i] += max(0, seg.end.Sub(seg.start))
		}
		for i, d := range spent {
			out.States[i].Defects++
			times[i] = append(times[i], Days(d))
		}
		current := at(h.Current)
		out.States[current].Current++
		if out.States[current].IsFinal {
			entered := h.CreatedAt
			if n := len(h.Transitions); n > 0 {
				entered = h.Transitions[n-1].At
			}
			resolutions = append(resolutions, Days(max(0, entered.Sub(h.CreatedAt))))
		}
	}
	for i := range out.States {
		if out.States[i].IsFinal {
			continue
		}
		out.States[i].MedianDays = Median(times[i])
		out.States[i].P90Days = Percentile(times[i], 0.9)
	}
	out.Resolved = len(resolutions)
	out.ResolutionMedianDays = Median(resolutions)
	out.ResolutionP90Days = Percentile(resolutions, 0.9)
	return out
}
