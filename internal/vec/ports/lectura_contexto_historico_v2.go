package ports

import (
	"context"
	"errors"
	"regexp"
	"vec-diputacion-granada/internal/vec/domain"
)

var ErrLecturaContextoOriginalV2 = errors.New("vec: contexto original no recuperable")

const MaximoBytesContextoOriginalV2 = 65536

var registroContextoOriginalV2 = regexp.MustCompile(`^rca_[A-Za-z0-9_-]{24,128}$`)
var huellaContextoOriginalV2 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Selector de un original propietario; no autoriza resolver ni actuar hoy.
type SolicitudLecturaContextoOriginalV2 struct {
	RegistroContextoRef               string
	HuellaSHA256                      string
	ManifiestoProcedenciaHuellaSHA256 string
}

func (s SolicitudLecturaContextoOriginalV2) Validar() error {
	if !registroContextoOriginalV2.MatchString(s.RegistroContextoRef) || !huellaContextoOriginalV2.MatchString(s.HuellaSHA256) || !huellaContextoOriginalV2.MatchString(s.ManifiestoProcedenciaHuellaSHA256) {
		return ErrLecturaContextoOriginalV2
	}
	return nil
}

// ValidarResultado coteja identidad y canon original, nunca vigencia actual.
// Las cotas se verifican antes de clonar/recanonizar material recibido.
func (s SolicitudLecturaContextoOriginalV2) ValidarResultado(r domain.ResultadoContextoActorRegistradoV2) error {
	if s.Validar() != nil || r.RegistroContextoRef != s.RegistroContextoRef || r.HuellaSHA256 != s.HuellaSHA256 || r.ManifiestoProcedenciaHuellaSHA256 != s.ManifiestoProcedenciaHuellaSHA256 || len(r.RepresentacionCanonica) == 0 || len(r.RepresentacionCanonica) > MaximoBytesContextoOriginalV2 || len(r.ManifiestoProcedenciaCanonico) == 0 || len(r.ManifiestoProcedenciaCanonico) > MaximoBytesContextoOriginalV2 {
		return ErrLecturaContextoOriginalV2
	}
	if r.Validar() != nil {
		return ErrLecturaContextoOriginalV2
	}
	return nil
}

// Implementación propietaria: una fila exacta, sin registrar/resolver/revalidar
// fuentes actuales. Un DTO válido no prueba almacenamiento ni autoridad nueva.
type LectorContextoOriginalV2 interface {
	LeerContextoOriginalV2(context.Context, SolicitudLecturaContextoOriginalV2) (domain.ResultadoContextoActorRegistradoV2, error)
}
