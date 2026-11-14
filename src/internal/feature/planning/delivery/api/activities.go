package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/planning/application/usecase"
	"kaoru/internal/feature/planning/domain"
)

type activityChangeDTO struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	System string `json:"system"`
}

type activitySummaryDTO struct {
	Code      string             `json:"code"`
	Kind      string             `json:"kind"`
	Title     string             `json:"title"`
	Change    *activityChangeDTO `json:"change,omitempty"`
	StartDate string             `json:"startDate"`
	EndDate   string             `json:"endDate"`
	Status    string             `json:"status"`
	Operators []userRefDTO       `json:"operators"`
}

type activityRefDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

type overlapDTO struct {
	Operator userRefDTO     `json:"operator"`
	Activity activityRefDTO `json:"activity"`
	Other    activityRefDTO `json:"other"`
	From     string         `json:"from"`
	To       string         `json:"to"`
	Days     int            `json:"days"`
}

type activityDTO struct {
	activitySummaryDTO
	Notes       string       `json:"notes"`
	CancelledAt *time.Time   `json:"cancelledAt,omitempty"`
	Overlaps    []overlapDTO `json:"overlaps"`
	CreatedBy   userRefDTO   `json:"createdBy"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type activityInputDTO struct {
	Kind      string   `json:"kind"`
	Title     string   `json:"title"`
	Change    string   `json:"change"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Operators []string `json:"operators"`
	Notes     string   `json:"notes"`
}

type activityPatchDTO struct {
	Kind      *string   `json:"kind"`
	Title     *string   `json:"title"`
	Change    *string   `json:"change"`
	StartDate *string   `json:"startDate"`
	EndDate   *string   `json:"endDate"`
	Operators *[]string `json:"operators"`
	Notes     *string   `json:"notes"`
	Cancelled *bool     `json:"cancelled"`
}

type scheduleWindowDTO struct {
	Change codeTitleDTO `json:"change"`
	System string       `json:"system"`
	Start  time.Time    `json:"start"`
	End    time.Time    `json:"end"`
}

type operatorPlanDTO struct {
	Operator   userRefDTO           `json:"operator"`
	Activities []activitySummaryDTO `json:"activities"`
	Overlaps   []overlapDTO         `json:"overlaps"`
}

type projectPlanDTO struct {
	Change         activityChangeDTO    `json:"change"`
	Certifications []activitySummaryDTO `json:"certifications"`
}

type scheduleDTO struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Summary struct {
		Activities int `json:"activities"`
		Operators  int `json:"operators"`
		Overlaps   int `json:"overlaps"`
	} `json:"summary"`
	Operators  []operatorPlanDTO    `json:"operators"`
	Unassigned []activitySummaryDTO `json:"unassigned"`
	Projects   []projectPlanDTO     `json:"projects"`
	Windows    []scheduleWindowDTO  `json:"windows"`
	Milestones []milestoneDTO       `json:"milestones"`
}

func (h *Handler) listActivities(c *gin.Context) {
	page, size, err := httpserver.PageParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	from, to, err := dateParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.ListActivities.Execute(c.Request.Context(), usecase.ListActivitiesInput{
		Operator: c.Query("operator"), ChangeCode: c.Query("change"), Kind: c.Query("kind"), Status: c.Query("status"),
		From: from, To: to, Page: page, Size: size,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]activitySummaryDTO, len(out.Activities))
	for i, a := range out.Activities {
		items[i] = toSummaryDTO(a.Activity, a.Status)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "size": size, "total": out.Total})
}

func (h *Handler) createActivity(c *gin.Context) {
	var body activityInputDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.CreateActivity.Execute(c.Request.Context(), usecase.CreateActivityInput{
		Actor: actorOf(c), Kind: body.Kind, Title: body.Title, Change: body.Change, Start: body.StartDate, End: body.EndDate,
		Operators: body.Operators, Notes: body.Notes,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, toActivityDTO(out))
}

func (h *Handler) getActivity(c *gin.Context) {
	out, err := h.uc.GetActivity.Execute(c.Request.Context(), c.Param("code"))
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityDTO(out))
}

func (h *Handler) updateActivity(c *gin.Context) {
	var body activityPatchDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.UpdateActivity.Execute(c.Request.Context(), usecase.UpdateActivityInput{
		Actor: actorOf(c), Code: c.Param("code"), Kind: body.Kind, Title: body.Title, Change: body.Change,
		Start: body.StartDate, End: body.EndDate, Operators: body.Operators, Notes: body.Notes, Cancelled: body.Cancelled,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityDTO(out))
}

func (h *Handler) schedule(c *gin.Context) {
	from, to, err := dateParams(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.GetSchedule.Execute(c.Request.Context(), usecase.GetScheduleInput{From: from, To: to, Operator: c.Query("operator")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toScheduleDTO(out))
}

// dateParams lee from y to como días, ambos incluidos.
func dateParams(c *gin.Context) (from, to *time.Time, err error) {
	var errs axis.FieldErrors
	parse := func(field string) *time.Time {
		raw := c.Query(field)
		if raw == "" {
			return nil
		}
		day, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			errs.Add(axis.InvalidField(field, "La fecha debe tener el formato 2026-11-30."))
			return nil
		}
		return &day
	}
	from, to = parse("from"), parse("to")
	return from, to, errs.Err()
}

func toUserDTO(u domain.UserRef) userRefDTO { return userRefDTO{ID: string(u.Id), DisplayName: u.Name} }

func toSummaryDTO(s domain.ActivitySnapshot, status domain.Status) activitySummaryDTO {
	dto := activitySummaryDTO{
		Code: s.Code, Kind: string(s.Kind), Title: s.Title, StartDate: s.Start.Format(domain.DateLayout),
		EndDate: s.End.Format(domain.DateLayout), Status: string(status), Operators: make([]userRefDTO, len(s.Operators)),
	}
	if s.Change != nil {
		dto.Change = &activityChangeDTO{Code: s.Change.Code, Title: s.Change.Title, System: s.Change.System}
	}
	for i, op := range s.Operators {
		dto.Operators[i] = toUserDTO(op)
	}
	return dto
}

func toOverlapDTO(o domain.Overlap) overlapDTO {
	ref := func(r domain.ActivityRef) activityRefDTO {
		return activityRefDTO{Code: r.Code, Title: r.Title, Kind: string(r.Kind)}
	}
	return overlapDTO{
		Operator: toUserDTO(o.Operator), Activity: ref(o.First), Other: ref(o.Second),
		From: o.From.Format(domain.DateLayout), To: o.To.Format(domain.DateLayout), Days: o.Days(),
	}
}

func toOverlapDTOs(list []domain.Overlap) []overlapDTO {
	out := make([]overlapDTO, len(list))
	for i, o := range list {
		out[i] = toOverlapDTO(o)
	}
	return out
}

func toActivityDTO(o usecase.ActivityOutput) activityDTO {
	s := o.Activity
	return activityDTO{
		activitySummaryDTO: toSummaryDTO(s, o.Status), Notes: s.Notes, CancelledAt: s.CancelledAt,
		Overlaps: toOverlapDTOs(o.Overlaps), CreatedBy: toUserDTO(s.CreatedBy), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func toScheduleDTO(o usecase.ScheduleOutput) scheduleDTO {
	s, now := o.Schedule, o.Now
	summaries := func(list []domain.Activity) []activitySummaryDTO {
		out := make([]activitySummaryDTO, len(list))
		for i, a := range list {
			out[i] = toSummaryDTO(a.Snapshot(), a.Status(now))
		}
		return out
	}
	dto := scheduleDTO{
		From: s.Range.From.Format(domain.DateLayout), To: s.Range.To.Format(domain.DateLayout),
		Operators: make([]operatorPlanDTO, len(s.Operators)), Unassigned: summaries(s.Unassigned),
		Projects: make([]projectPlanDTO, len(s.Projects)), Windows: make([]scheduleWindowDTO, len(s.Windows)),
		Milestones: make([]milestoneDTO, len(s.Milestones)),
	}
	dto.Summary.Activities, dto.Summary.Operators, dto.Summary.Overlaps = s.ActivityCount(), len(s.Operators), s.OverlapCount()
	for i, op := range s.Operators {
		dto.Operators[i] = operatorPlanDTO{Operator: toUserDTO(op.Operator), Activities: summaries(op.Activities), Overlaps: toOverlapDTOs(op.Overlaps)}
	}
	for i, p := range s.Projects {
		dto.Projects[i] = projectPlanDTO{
			Change:         activityChangeDTO{Code: p.Change.Code, Title: p.Change.Title, System: p.Change.System},
			Certifications: summaries(p.Certifications),
		}
	}
	for i, w := range s.Windows {
		dto.Windows[i] = scheduleWindowDTO{Change: codeTitleDTO{Code: w.ChangeCode, Title: w.ChangeTitle}, System: w.System, Start: w.Start, End: w.End}
	}
	for i, m := range s.Milestones {
		dto.Milestones[i] = toMilestoneDTO(usecase.MilestoneOutput{Milestone: m.Snapshot(), DaysLate: m.DaysLate(now)})
	}
	return dto
}
