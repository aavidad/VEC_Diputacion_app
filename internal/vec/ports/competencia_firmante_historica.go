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
// central; RecursoActual lo resuelve el servidor para el PDP del consultante.
// Accion y finalidad son de esta lectura, no las del firmante historico.
type SolicitudRecuperacionCompetenciaFirmanteHistoricaV1 struct {
	RegistroRef       string
	CanonHuellaSHA256 string
	Actor             domain.ContextoActor
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
	Vinculo           domain.VinculoAutenticacionActorV2
	RecursoActual     domain.RecursoAutorizable
	SelectorHistorico domain.SelectorHistoricoCompetenciaFirmanteV1
	DescriptorLectura domain.DescriptorLecturaCompetenciaFirmanteHistoricaV1
	Accion            string
	Finalidad         string
	Motivo            domain.ReferenciaEntradaCatalogo
	CorrelacionRef    string
}

func (s SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) ValidarEn(en time.Time) error {
	if !en.IsZero() && en.Location() == time.UTC && en.Nanosecond()%1_000 == 0 &&
		s.Actor.Validar() == nil && s.ResultadoContexto.Validar() == nil &&
		s.Vinculo.VigenteEn(en, s.ResultadoContexto) &&
		s.RecursoActual.Validar() == nil && len(s.RecursoActual.Ambitos) > 0 &&
		s.SelectorHistorico.Validar() == nil &&
		s.DescriptorLectura.Validar() == nil &&
		domain.ReferenciaMotivoAutorizacionV2Valida(s.Motivo) &&
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
		domain.ValidarLecturaCompetenciaFirmanteHistoricaV1(c, s.RegistroRef,
			s.SelectorHistorico, s.DescriptorLectura, s.RecursoActual) != nil {
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

// La implementacion obtiene el selector del efecto durable identificado por
// RegistroRef y comprueba la version/huella del DescriptorLectura publicado;
// no confia en los valores de la solicitud aislada. Consume PDP V3 vigente
// para RecursoActual, Accion y
// Finalidad del consultante y audita antes de devolver bytes o metadatos.
// Nunca sustituye ese recurso por el de competencia historica. No emite HTTP.
type LectorCompetenciaFirmanteHistoricaV1 interface {
	RecuperarCompetenciaFirmanteHistoricaV1(context.Context, SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) (domain.CanonCompetenciaFirmanteHistoricaV1, error)
}
