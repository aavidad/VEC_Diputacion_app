package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrCandidateIDRequired     = errors.New("candidate id is required")
	ErrCandidateDNIRequired    = errors.New("candidate dni is required")
	ErrCandidateNombreRequired = errors.New("candidate nombre is required")
	ErrCandidateEmailRequired  = errors.New("candidate email is required")

	ErrMeritIDRequired   = errors.New("merit id is required")
	ErrMeritTypeInvalid  = errors.New("merit type is invalid")
	ErrMeritStateInvalid = errors.New("merit state is invalid")
	ErrMeritDataInvalid  = errors.New("merit data is invalid")
	ErrMeritTransition   = errors.New("merit transition is invalid")
)

type MeritType string

const (
	MeritTypeExperienciaMismaCategoria MeritType = "experiencia_misma_categoria"
	MeritTypeExperienciaOtraCategoria  MeritType = "experiencia_otra_categoria"
	MeritTypeFormacionTitulo           MeritType = "formacion_titulo"
	MeritTypeFormacionCurso            MeritType = "formacion_curso"
	MeritTypeOtros                     MeritType = "otros"

	MeritTypeExperienceSameCategory  = MeritTypeExperienciaMismaCategoria
	MeritTypeExperienceOtherCategory = MeritTypeExperienciaOtraCategoria
	MeritTypeTrainingDegree          = MeritTypeFormacionTitulo
	MeritTypeTrainingCourse          = MeritTypeFormacionCurso
	MeritTypeOther                   = MeritTypeOtros
)

func (t MeritType) IsValid() bool {
	switch t {
	case MeritTypeExperienciaMismaCategoria,
		MeritTypeExperienciaOtraCategoria,
		MeritTypeFormacionTitulo,
		MeritTypeFormacionCurso,
		MeritTypeOtros:
		return true
	default:
		return false
	}
}

type MeritState string

const (
	MeritStateBorrador    MeritState = "Borrador"
	MeritStatePresentado  MeritState = "Presentado"
	MeritStateValidado    MeritState = "Validado"
	MeritStateRechazado   MeritState = "Rechazado"
	MeritStateSubsanacion MeritState = "Subsanacion"

	MeritStateDraft      = MeritStateBorrador
	MeritStateSubmitted  = MeritStatePresentado
	MeritStateValidated  = MeritStateValidado
	MeritStateRejected   = MeritStateRechazado
	MeritStateCorrection = MeritStateSubsanacion
)

func (s MeritState) IsValid() bool {
	switch s {
	case MeritStateBorrador,
		MeritStatePresentado,
		MeritStateValidado,
		MeritStateRechazado,
		MeritStateSubsanacion:
		return true
	default:
		return false
	}
}

type MeritData struct {
	Meses       int
	Horas       int
	PuntosFijos float64
}

func (d MeritData) Validate() error {
	if d.Meses < 0 || d.Horas < 0 || d.PuntosFijos < 0 {
		return ErrMeritDataInvalid
	}
	return nil
}

type Merit struct {
	ID     string
	Tipo   MeritType
	Datos  MeritData
	Estado MeritState
}

func NewMerit(id string, tipo MeritType, datos MeritData) (Merit, error) {
	merit := Merit{
		ID:     strings.TrimSpace(id),
		Tipo:   tipo,
		Datos:  datos,
		Estado: MeritStateBorrador,
	}
	if err := merit.Validate(); err != nil {
		return Merit{}, err
	}
	return merit, nil
}

func (m Merit) Validate() error {
	switch {
	case strings.TrimSpace(m.ID) == "":
		return ErrMeritIDRequired
	case !m.Tipo.IsValid():
		return ErrMeritTypeInvalid
	case !m.Estado.IsValid():
		return ErrMeritStateInvalid
	default:
		return m.Datos.Validate()
	}
}

func (m Merit) CanTransition(to MeritState) bool {
	if !m.Estado.IsValid() || !to.IsValid() {
		return false
	}
	for _, allowed := range allowedMeritTransitions[m.Estado] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (m *Merit) Transition(to MeritState) error {
	if m == nil {
		return fmt.Errorf("%w: nil merit", ErrMeritTransition)
	}
	if !to.IsValid() {
		return fmt.Errorf("%w: target state %q", ErrMeritStateInvalid, to)
	}
	if err := m.Validate(); err != nil {
		return err
	}
	if !m.CanTransition(to) {
		return fmt.Errorf("%w: %s -> %s", ErrMeritTransition, m.Estado, to)
	}
	m.Estado = to
	return nil
}

var allowedMeritTransitions = map[MeritState][]MeritState{
	MeritStateBorrador:    {MeritStatePresentado},
	MeritStatePresentado:  {MeritStateValidado, MeritStateRechazado, MeritStateSubsanacion},
	MeritStateSubsanacion: {MeritStatePresentado},
}
