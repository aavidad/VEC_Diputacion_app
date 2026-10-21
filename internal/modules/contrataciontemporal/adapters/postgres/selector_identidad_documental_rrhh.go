package postgres

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ports.SesionSelectorIdentidadDocumentalRRHH = (*SesionConsultaRRHHPostgreSQL)(nil)

func (s *SesionConsultaRRHHPostgreSQL) ConsultarSelectorYRegistrarAcceso(ctx context.Context, orden ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, domain.SelectorBolsaPlanB2Nativo, error) {
	dz, sz := ports.DetalleExpedienteRRHH{}, domain.SelectorBolsaPlanB2Nativo{}
	if err := s.validarContexto(ctx); err != nil {
		return dz, sz, err
	}
	ampliado, err := orden.Capacidad().ContratoSelectoresRecuperacionFirmas()
	if err != nil || ampliado || s.modo != modoConsultaDetalleRRHHOrdinaria ||
		orden.Solicitud().VersionObservada() != 0 {
		return dz, sz, ports.ErrConsultaRRHHNoDisponible
	}
	material, err := orden.ExportacionParaSQL()
	if err != nil || material.ValidarEstructura() != nil {
		return dz, sz, ports.ErrConsultaRRHHNoDisponible
	}
	args, err := nuevosArgumentosMaterialConsultaRRHH(material)
	if err != nil {
		return dz, sz, err
	}
	defer args.limpiar()
	c, cap, q := orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	var salida salidaDetalleConsultaRRHH
	defer clear(salida.contenidoCanonico)
	var raw string
	destinos := append(destinosDetalleConsultaRRHH(&salida), &raw)
	type lectura struct {
		detalle  ports.DetalleExpedienteRRHH
		selector domain.SelectorBolsaPlanB2Nativo
	}
	r, err := ejecutarConsultaRRHHEnTransaccion(ctx, s.pool, consultaSelectorIdentidadDocumentalRRHHPostgreSQL,
		argumentosSQLDetalleConsultaRRHH(c.OrganizacionRef(), string(cap.ClaseAmbito()), cap.AmbitoRef(), q, args), destinos,
		func() (lectura, error) {
			z := lectura{}
			if ctx.Err() != nil {
				return z, ctx.Err()
			}
			salida.cierre.normalizarInstantesSQL()
			recibo, e := salida.cierre.construirRecibo(c, cap)
			if e != nil {
				return z, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			_, version, _, e := salida.cierre.enterosSeguros()
			if e != nil {
				return z, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			entrada, e := s.analizador.analizarDetalle(salida.contenidoCanonico, salida.cierre.generadaEn, salida.cierre.expedienteRef, version)
			if e != nil {
				return z, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			d, e := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, recibo)
			if e != nil || d.ValidarParaEjecucionInterna(orden) != nil {
				return z, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			selector, e := decodificarSelectorIdentidadDocumentalRRHH(raw, q.ExpedienteRef(), c.OrganizacionRef())
			if e != nil {
				return z, e
			}
			return lectura{d, selector}, nil
		})
	if ctx.Err() != nil {
		return dz, sz, ctx.Err()
	}
	if err != nil {
		return dz, sz, err
	}
	return r.detalle, r.selector, nil
}

func decodificarSelectorIdentidadDocumentalRRHH(raw, expediente, organizacion string) (domain.SelectorBolsaPlanB2Nativo, error) {
	z := domain.SelectorBolsaPlanB2Nativo{}
	var selector domain.SelectorBolsaPlanB2Nativo
	if len(raw) == 0 || len(raw) > 2048 ||
		decodificarJSONEstricto([]byte(raw), &selector) != nil || !selector.Valido() ||
		selector.ExpedienteRef != expediente || selector.OrganizacionRef != organizacion {
		return z, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return selector, nil
}

const consultaSelectorIdentidadDocumentalRRHHPostgreSQL = `
SELECT contenido_canonico,esquema,acceso_ref,secuencia::bigint,anterior_sha256,
       huella_sha256,vinculo_identidad_huella_sha256,alcance_huella_sha256,
       registrada_en,auditoria_vec_ref,auditoria_vec_huella_sha256,
       consumo_vec_huella_sha256,contenido_huella_sha256,resultado_huella_sha256,
       cursor_huella_sha256,generada_en,expediente_ref,version_expediente::bigint,
       total,recibo_sello_sha256,selector::text
  FROM vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(
       ROW($1::text,$2::text,$3::text)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       ROW($4::text,$5::numeric)::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
       $6::bytea,$7::bytea,$8::bytea,$9::bytea,$10::numeric,$11::numeric,
       $12::bytea,$13::bytea,$14::bytea,$15::bytea)`
