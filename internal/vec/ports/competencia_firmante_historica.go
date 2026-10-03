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

// SelectorHistoricoCompetenciaFirmanteV1 procede del efecto propietario y su
// descriptor publicado. No se construye desde parametros libres del canal.
// Identifica el recurso exacto historico; el consultante usa RecursoActual.
type SelectorHistoricoCompetenciaFirmanteV1 struct {
	OrganizacionRef, UnidadRef, ExpedienteRef, DocumentoRef string
	ModuloID, TipoRecurso, RecursoRef                       string
	RecursoContextoSHA256                                   string
}

func (s SelectorHistoricoCompetenciaFirmanteV1) Validar() error {
	if referenciaCompetenciaHistoricaPuerto(s.OrganizacionRef, 512) &&
		referenciaCompetenciaHistoricaPuerto(s.UnidadRef, 512) &&
		referenciaCompetenciaHistoricaPuerto(s.ExpedienteRef, 512) &&
		referenciaCompetenciaHistoricaPuerto(s.DocumentoRef, 512) &&
		referenciaCompetenciaHistoricaPuerto(s.ModuloID, 128) &&
		referenciaCompetenciaHistoricaPuerto(s.TipoRecurso, 128) &&
		s.RecursoRef == s.DocumentoRef &&
		huellaCompetenciaHistoricaPuerto(s.RecursoContextoSHA256) {
		return nil
	}
	return ErrRecuperacionCompetenciaFirmanteHistoricaV1
}

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
	SelectorHistorico SelectorHistoricoCompetenciaFirmanteV1
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
	selector := s.SelectorHistorico
	if err != nil || huella != s.CanonHuellaSHA256 ||
		c.Recurso.RecursoAutorizableRef != selector.RecursoRef ||
		c.Recurso.OrganizacionRef != selector.OrganizacionRef ||
		c.Recurso.UnidadRef != selector.UnidadRef ||
		c.Recurso.ExpedienteRef != selector.ExpedienteRef ||
		c.Recurso.DocumentoRef != selector.DocumentoRef ||
		c.Recurso.ModuloID != selector.ModuloID ||
		c.Recurso.TipoRecurso != selector.TipoRecurso ||
		c.Recurso.RecursoContextoSHA256 != selector.RecursoContextoSHA256 {
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
// RegistroRef y lo compara con SelectorHistorico; no confia en el valor de la
// solicitud aislada. Consume PDP V3 vigente para RecursoActual, Accion y
// Finalidad del consultante y audita antes de devolver bytes o metadatos.
// Nunca sustituye ese recurso por el de competencia historica. No emite HTTP.
type LectorCompetenciaFirmanteHistoricaV1 interface {
	RecuperarCompetenciaFirmanteHistoricaV1(context.Context, SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) (domain.CanonCompetenciaFirmanteHistoricaV1, error)
}
