package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrScheduleProfileInvalid = errors.New("cronos schedule profile invalid")
	ErrWorkdayInvalid         = errors.New("cronos workday invalid")
)

type WorkMode string

const (
	WorkModeOnSite   WorkMode = "presencial"
	WorkModeTelework WorkMode = "teletrabajo"
)

type PunchKind string

const (
	PunchEntry      PunchKind = "entrada"
	PunchExit       PunchKind = "salida"
	PunchPauseStart PunchKind = "inicio_pausa"
	PunchPauseEnd   PunchKind = "fin_pausa"
)

type ScheduleProfile struct {
	ID               string
	Name             string
	Unit             string
	Flexible         bool
	AllowsTelework   bool
	RequiresCoverage bool
	DailyTarget      time.Duration
	WeeklyTarget     time.Duration
	EntryWindowStart time.Duration
	EntryWindowEnd   time.Duration
	CoreStart        time.Duration
	CoreEnd          time.Duration
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
}

func (p ScheduleProfile) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Name) == "" {
		return ErrScheduleProfileInvalid
	}
	if p.DailyTarget <= 0 || p.WeeklyTarget <= 0 {
		return ErrScheduleProfileInvalid
	}
	if p.Flexible && p.EntryWindowEnd > 0 && p.EntryWindowStart > p.EntryWindowEnd {
		return ErrScheduleProfileInvalid
	}
	if p.CoreEnd > 0 && p.CoreStart > p.CoreEnd {
		return ErrScheduleProfileInvalid
	}
	if !p.EffectiveTo.IsZero() && !p.EffectiveFrom.IsZero() && p.EffectiveTo.Before(p.EffectiveFrom) {
		return ErrScheduleProfileInvalid
	}
	return nil
}

type Punch struct {
	At      time.Time
	Kind    PunchKind
	Channel string
	Mode    WorkMode
}

type AuthorizedAbsence struct {
	Ref      string
	Kind     string
	Duration time.Duration
	Paid     bool
}

type Workday struct {
	EmployeeID       string
	Date             time.Time
	Age              int
	Profile          ScheduleProfile
	Punches          []Punch
	TeleworkDuration time.Duration
	Absences         []AuthorizedAbsence
}

type IncidentSeverity string

const (
	IncidentInfo     IncidentSeverity = "info"
	IncidentWarning  IncidentSeverity = "warning"
	IncidentBlocking IncidentSeverity = "blocking"
)

type Incident struct {
	Code     string
	Label    string
	Severity IncidentSeverity
}

type DayResult struct {
	EmployeeID          string
	Date                time.Time
	Age                 int
	ProfileID           string
	Theoretical         time.Duration
	Reduction           time.Duration
	AuthorizedAbsence   time.Duration
	Worked              time.Duration
	OnSiteWorked        time.Duration
	Telework            time.Duration
	Balance             time.Duration
	OpenEntry           bool
	FixedCoverageBreach bool
	Incidents           []Incident
}

func EvaluateWorkday(day Workday) (DayResult, error) {
	if strings.TrimSpace(day.EmployeeID) == "" || day.Date.IsZero() || day.Age < 0 {
		return DayResult{}, ErrWorkdayInvalid
	}
	if day.TeleworkDuration < 0 {
		return DayResult{}, ErrWorkdayInvalid
	}
	if err := day.Profile.Validate(); err != nil {
		return DayResult{}, err
	}
	absence, err := authorizedAbsenceDuration(day.Absences)
	if err != nil {
		return DayResult{}, err
	}
	onsite, firstEntry, lastExit, openEntry, incidents := evaluatePunches(day.Punches)
	reduction := DailyReductionForAge(day.Age)
	theoretical := day.Profile.DailyTarget - reduction - absence
	if theoretical < 0 {
		theoretical = 0
	}
	if day.TeleworkDuration > 0 && !day.Profile.AllowsTelework {
		incidents = append(incidents, Incident{
			Code:     "teletrabajo_no_autorizado",
			Label:    "Teletrabajo informado en perfil sin teletrabajo",
			Severity: IncidentWarning,
		})
	}
	coverageBreach := coverageBreach(day.Profile, firstEntry, lastExit, onsite, day.TeleworkDuration, reduction)
	if coverageBreach {
		incidents = append(incidents, Incident{
			Code:     "cobertura_obligatoria_incumplida",
			Label:    "La jornada no cubre el tramo obligatorio del perfil",
			Severity: IncidentBlocking,
		})
	}
	worked := onsite + day.TeleworkDuration
	return DayResult{
		EmployeeID:          day.EmployeeID,
		Date:                day.Date,
		Age:                 day.Age,
		ProfileID:           day.Profile.ID,
		Theoretical:         theoretical,
		Reduction:           reduction,
		AuthorizedAbsence:   absence,
		Worked:              worked,
		OnSiteWorked:        onsite,
		Telework:            day.TeleworkDuration,
		Balance:             worked - theoretical,
		OpenEntry:           openEntry,
		FixedCoverageBreach: coverageBreach,
		Incidents:           incidents,
	}, nil
}

func DailyReductionForAge(age int) time.Duration {
	switch {
	case age >= 64:
		return 2 * time.Hour
	case age >= 63:
		return time.Hour
	default:
		return 0
	}
}

func FormatHHMM(value time.Duration) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	minutes := int(value.Round(time.Minute) / time.Minute)
	return fmt.Sprintf("%s%02d:%02d", sign, minutes/60, minutes%60)
}

func Minutes(hours, minutes int) time.Duration {
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
}

func authorizedAbsenceDuration(absences []AuthorizedAbsence) (time.Duration, error) {
	var total time.Duration
	for _, absence := range absences {
		if absence.Duration < 0 {
			return 0, ErrWorkdayInvalid
		}
		total += absence.Duration
	}
	return total, nil
}

func evaluatePunches(punches []Punch) (worked time.Duration, firstEntry, lastExit time.Time, openEntry bool, incidents []Incident) {
	ordered := append([]Punch(nil), punches...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].At.Before(ordered[j].At)
	})
	var entryAt *time.Time
	var pauseAt *time.Time
	var pauseTotal time.Duration
	for _, punch := range ordered {
		if punch.At.IsZero() {
			incidents = append(incidents, incident("fichaje_sin_fecha", "Fichaje sin fecha/hora", IncidentBlocking))
			continue
		}
		switch punch.Kind {
		case PunchEntry:
			if entryAt != nil {
				incidents = append(incidents, incident("entrada_duplicada", "Entrada duplicada sin salida previa", IncidentBlocking))
				continue
			}
			at := punch.At
			entryAt = &at
			if firstEntry.IsZero() {
				firstEntry = at
			}
		case PunchPauseStart:
			if entryAt == nil {
				incidents = append(incidents, incident("pausa_sin_entrada", "Pausa sin entrada abierta", IncidentWarning))
				continue
			}
			if pauseAt != nil {
				incidents = append(incidents, incident("pausa_duplicada", "Inicio de pausa duplicado", IncidentWarning))
				continue
			}
			at := punch.At
			pauseAt = &at
		case PunchPauseEnd:
			if pauseAt == nil {
				incidents = append(incidents, incident("fin_pausa_sin_inicio", "Fin de pausa sin inicio", IncidentWarning))
				continue
			}
			if punch.At.Before(*pauseAt) {
				incidents = append(incidents, incident("pausa_invalida", "Pausa con fin anterior al inicio", IncidentBlocking))
				pauseAt = nil
				continue
			}
			pauseTotal += punch.At.Sub(*pauseAt)
			pauseAt = nil
		case PunchExit:
			if entryAt == nil {
				incidents = append(incidents, incident("salida_sin_entrada", "Salida sin entrada previa", IncidentBlocking))
				continue
			}
			if punch.At.Before(*entryAt) {
				incidents = append(incidents, incident("secuencia_invalida", "Salida anterior a la entrada", IncidentBlocking))
				entryAt = nil
				pauseAt = nil
				continue
			}
			if pauseAt != nil {
				pauseTotal += punch.At.Sub(*pauseAt)
				incidents = append(incidents, incident("pausa_abierta_cerrada", "Pausa abierta cerrada con la salida", IncidentWarning))
				pauseAt = nil
			}
			worked += punch.At.Sub(*entryAt)
			lastExit = punch.At
			entryAt = nil
		default:
			incidents = append(incidents, incident("tipo_fichaje_desconocido", "Tipo de fichaje desconocido", IncidentBlocking))
		}
	}
	if pauseAt != nil {
		incidents = append(incidents, incident("pausa_sin_fin", "Pausa abierta sin fin", IncidentWarning))
	}
	if entryAt != nil {
		openEntry = true
		incidents = append(incidents, incident("fichaje_sin_salida", "Entrada abierta sin salida", IncidentBlocking))
	}
	worked -= pauseTotal
	if worked < 0 {
		worked = 0
	}
	return worked, firstEntry, lastExit, openEntry, incidents
}

func coverageBreach(profile ScheduleProfile, firstEntry, lastExit time.Time, onsite, telework, reduction time.Duration) bool {
	if onsite == 0 && telework > 0 && profile.AllowsTelework && !profile.RequiresCoverage {
		return false
	}
	if firstEntry.IsZero() || lastExit.IsZero() {
		return profile.RequiresCoverage || !profile.Flexible
	}
	start := profile.CoreStart
	end := profile.CoreEnd
	if !profile.Flexible {
		if profile.EntryWindowStart > 0 {
			start = profile.EntryWindowStart
		}
		if profile.EntryWindowEnd > 0 {
			end = profile.EntryWindowEnd
		}
	}
	if start == 0 && end == 0 {
		return false
	}
	if profile.Flexible && reduction > 0 && end > 0 {
		end -= reduction
		if end < start {
			end = start
		}
	}
	first := timeOfDay(firstEntry)
	last := timeOfDay(lastExit)
	if start > 0 && first > start {
		return true
	}
	if end > 0 && last < end {
		return true
	}
	return false
}

func incident(code, label string, severity IncidentSeverity) Incident {
	return Incident{Code: code, Label: label, Severity: severity}
}

func timeOfDay(value time.Time) time.Duration {
	return time.Duration(value.Hour())*time.Hour +
		time.Duration(value.Minute())*time.Minute +
		time.Duration(value.Second())*time.Second
}
