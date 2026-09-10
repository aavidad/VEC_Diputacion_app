package ports

import (
	"context"
	"errors"
	"regexp"
	"vec-diputacion-granada/internal/vec/domain"
)

var ErrLecturaAutenticacionOriginalV1 = errors.New("vec: autenticacion original no recuperable")
var huellaAutenticacionOriginalV1 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Selector de historia propietaria; no es autoridad para una acción actual.
type SolicitudLecturaAutenticacionOriginalV1 struct {
	AutenticacionRef, SesionRef, HuellaSHA256 string
}

func (s SolicitudLecturaAutenticacionOriginalV1) Validar() error {
	if (domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: s.AutenticacionRef, SesionRef: s.SesionRef}).Validar() != nil || !huellaAutenticacionOriginalV1.MatchString(s.HuellaSHA256) {
		return ErrLecturaAutenticacionOriginalV1
	}
	return nil
}

func (s SolicitudLecturaAutenticacionOriginalV1) ValidarResultado(r domain.AutenticacionRevalidadaV1) error {
	if s.Validar() != nil || r.AutenticacionRef != s.AutenticacionRef || r.SesionRef != s.SesionRef || r.AutenticacionHuellaSHA256 != s.HuellaSHA256 || r.Validar() != nil {
		return ErrLecturaAutenticacionOriginalV1
	}
	return nil
}

// Fuente propietaria read-only que relee el control fijado en el consumo original,
// nunca el último control. Un DTO válido no acredita almacenamiento ni permiso.
type LectorAutenticacionOriginalV1 interface {
	LeerAutenticacionOriginalV1(context.Context, SolicitudLecturaAutenticacionOriginalV1) (domain.AutenticacionRevalidadaV1, error)
}
