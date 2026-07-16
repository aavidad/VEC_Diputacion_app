package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type fuenteBaremacionPrueba struct {
	criterio       puertosbolsa.CriterioBaremacionConfiable
	evidencia      puertosbolsa.EvidenciaBaremacionConfiable
	representacion puertosbolsa.RepresentacionBaremacionConfiable
}

func (f *fuenteBaremacionPrueba) ObtenerCriterio(
	_ context.Context,
	s puertosbolsa.SolicitudObtenerCriterioBaremacion,
) (puertosbolsa.CriterioBaremacionConfiable, error) {
	if f.criterio.ValidarPara(s) != nil {
		return puertosbolsa.CriterioBaremacionConfiable{}, puertosbolsa.ErrCriterioBaremacionNoVigente
	}
	return f.criterio, nil
}

func (f *fuenteBaremacionPrueba) ObtenerEvidencia(
	_ context.Context,
	s puertosbolsa.SolicitudObtenerEvidenciaBaremacion,
) (puertosbolsa.EvidenciaBaremacionConfiable, error) {
	if f.evidencia.ValidarPara(s) != nil {
		return puertosbolsa.EvidenciaBaremacionConfiable{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return f.evidencia.Clonar()
}

func (f *fuenteBaremacionPrueba) ObtenerRepresentacion(
	_ context.Context,
	s puertosbolsa.SolicitudObtenerRepresentacionBaremacion,
) (puertosbolsa.RepresentacionBaremacionConfiable, error) {
	if f.representacion.ValidarPara(s) != nil {
		return puertosbolsa.RepresentacionBaremacionConfiable{}, puertosbolsa.ErrRepresentacionBaremacionNoConfiable
	}
	return f.representacion, nil
}

type calculadorBaremacionPrueba struct {
	resultado      puertosbolsa.ResultadoCalculoOficial
	recuperaciones int
}

func (c *calculadorBaremacionPrueba) CalcularPuntuacionOficial(
	context.Context,
	puertosbolsa.SolicitudCalcularPuntuacionOficial,
) (puertosbolsa.ResultadoCalculoOficial, error) {
	return puertosbolsa.ResultadoCalculoOficial{}, puertosbolsa.ErrCalculoOficialNoDisponible
}

func (c *calculadorBaremacionPrueba) RecuperarCalculoOficial(
	_ context.Context,
	s puertosbolsa.SolicitudRecuperarCalculoOficial,
) (puertosbolsa.ResultadoCalculoOficial, error) {
	c.recuperaciones++
	if s.Validar() != nil || s.CalculoRef != c.resultado.Calculo.CalculoRef ||
		s.HuellaResultado != c.resultado.Calculo.HuellaResultadoSHA256 {
		return puertosbolsa.ResultadoCalculoOficial{}, puertosbolsa.ErrCalculoOficialNoReproducible
	}
	return c.resultado.Clonar()
}

type politicasBaremacionPrueba struct {
	politica puertosbolsa.PoliticaFirmaBaremacion
}

func (p *politicasBaremacionPrueba) ObtenerPoliticaFirma(
	_ context.Context,
	s puertosbolsa.SolicitudObtenerPoliticaFirma,
) (puertosbolsa.PoliticaFirmaBaremacion, error) {
	if p.politica.ValidarPara(s) != nil {
		return puertosbolsa.PoliticaFirmaBaremacion{}, puertosbolsa.ErrPoliticaFirmaNoVigente
	}
	return p.politica, nil
}

type codificadorBaremacionPrueba struct{}

func (codificadorBaremacionPrueba) CodificarDecision(
	_ context.Context,
	s puertosbolsa.SolicitudCodificarDecisionCanonica,
) (puertosbolsa.CodificacionCanonicaDecision, error) {
	if s.Validar() != nil {
		return puertosbolsa.CodificacionCanonicaDecision{}, puertosbolsa.ErrCodificacionCanonicaNoDisponible
	}
	contenido := []byte("%PDF-1.7\ndecision-baremacion-canonica")
	carga, _ := puertosbolsa.NuevaCargaProtegida(contenido)
	huellaContenido, _ := s.Contenido.HuellaContenidoSHA256()
	huellaDocumento := sha256.Sum256(contenido)
	p := s.Contexto.Proyeccion()
	resultado := puertosbolsa.CodificacionCanonicaDecision{
		Carga: carga, ProcesoRef: s.Contenido.ProcesoRef, SolicitudRef: s.Contenido.SolicitudRef,
		SujetoRef: s.Contenido.SujetoRef, BaremacionMeritoRef: s.Contenido.BaremacionMeritoRef,
		DecisionRef: s.Contenido.ID, VersionBaremacion: s.Contenido.VersionBaremacion,
		PrincipalRef: p.PrincipalRef, PerfilActorClave: p.PerfilActorClave,
		AutorizacionDecisionRef:     s.AutorizacionDecision.Proyeccion().AutorizacionRef,
		AutorizacionCodificacionRef: p.AutorizacionRef, FinalidadClave: p.FinalidadClave,
		CorrelacionRef: p.CorrelacionRef, FormatoClave: s.Politica.FormatoFirmaClave,
		MIME: "application/pdf", HuellaContenidoSHA256: huellaContenido,
		HuellaDocumentoSHA256: hex.EncodeToString(huellaDocumento[:]), VersionCodificador: "codificador-pdf-v1",
	}
	if resultado.ValidarPara(s) != nil {
		return puertosbolsa.CodificacionCanonicaDecision{}, puertosbolsa.ErrCodificacionCanonicaNoDisponible
	}
	return resultado, nil
}
