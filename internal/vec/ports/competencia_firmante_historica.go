package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrRecuperacionCompetenciaFirmanteHistoricaV1 = errors.New("vec: competencia firmante historica no recuperable")

// La referencia selecciona una entrada previamente registrada por el
// propietario del efecto. Actor, resultado y vinculo proceden de identidad
// central; Recurso lo resuelve el servidor. Accion y finalidad son de esta
// lectura y nunca se sustituyen por las del firmante historico.
type SolicitudRecuperacionCompetenciaFirmanteHistoricaV1 struct {
	RegistroRef       string
	CanonHuellaSHA256 string
	Actor             domain.ContextoActor
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
	Vinculo           domain.VinculoAutenticacionActorV2
	Recurso           domain.RecursoAutorizable
	Accion            string
	Finalidad         string
	Motivo            domain.ReferenciaEntradaCatalogo
	CorrelacionRef    string
}

func (s SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) ValidarEn(en time.Time) error {
	if !en.IsZero() && en.Location() == time.UTC && en.Nanosecond()%1_000 == 0 &&
		s.Actor.Validar() == nil && s.ResultadoContexto.Validar() == nil &&
		s.Vinculo.VigenteEn(en, s.ResultadoContexto) &&
		s.Recurso.Validar() == nil && len(s.Recurso.Ambitos) > 0 &&
		s.Motivo.Validar() == nil &&
		referenciaCompetenciaHistoricaPuerto(s.RegistroRef, 512) &&
		huellaCompetenciaHistoricaPuerto(s.CanonHuellaSHA256) &&
		referenciaCompetenciaHistoricaPuerto(s.Accion, 256) &&
		referenciaCompetenciaHistoricaPuerto(s.Finalidad, 512) &&
		domain.ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		huella, err := s.Actor.HuellaSHA256VinculadaV2()
		if err == nil && huella == s.ResultadoContexto.HuellaSHA256 {
			return nil
		}
	}
	return ErrRecuperacionCompetenciaFirmanteHistoricaV1
}

// El resultado se contrasta con el selector y el recurso exacto. La fuente
// debe consumir autorizacion actual y auditar incluso el replay; cotejar este
// DTO por si solo no concede lectura.
func (s SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) ValidarResultado(
	c domain.CanonCompetenciaFirmanteHistoricaV1,
	en time.Time,
) error {
	if s.ValidarEn(en) != nil {
		return ErrRecuperacionCompetenciaFirmanteHistoricaV1
	}
	huella, err := c.HuellaSHA256()
	if err != nil || huella != s.CanonHuellaSHA256 ||
		c.Recurso.RecursoAutorizableRef != s.Recurso.Referencia ||
		c.Recurso.OrganizacionRef != s.Recurso.Ambitos["organizacion_ref"] ||
		c.Recurso.UnidadRef != s.Recurso.Ambitos["unidad_ref"] ||
		c.Recurso.ExpedienteRef != s.Recurso.Ambitos["expediente_ref"] ||
		c.Recurso.DocumentoRef != s.Recurso.Referencia {
		return ErrRecuperacionCompetenciaFirmanteHistoricaV1
	}
	actual, err := s.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || actual != c.Recurso.RecursoContextoSHA256 {
		return ErrRecuperacionCompetenciaFirmanteHistoricaV1
	}
	return nil
}

func referenciaCompetenciaHistoricaPuerto(valor string, maximo int) bool {
	if len(valor) == 0 || len(valor) > maximo || strings.ContainsRune(valor, '*') {
		return false
	}
	for _, r := range valor {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

func huellaCompetenciaHistoricaPuerto(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) || valor == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	b, err := hex.DecodeString(valor)
	return err == nil && len(b) == sha256.Size
}

// La implementacion lee historia propietaria y consume PDP V3 vigente para
// el actor consultante antes de devolver bytes o metadatos. No emite HTTP.
type LectorCompetenciaFirmanteHistoricaV1 interface {
	RecuperarCompetenciaFirmanteHistoricaV1(context.Context, SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) (domain.CanonCompetenciaFirmanteHistoricaV1, error)
}
