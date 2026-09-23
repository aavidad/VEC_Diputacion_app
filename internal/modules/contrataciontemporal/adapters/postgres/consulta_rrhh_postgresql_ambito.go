package postgres

import (
	"context"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// SesionConsultaRRHHConAmbitoPostgreSQL solo acepta el pool nominal CA6/CT109.
// Su tipo es distinto del adaptador historico y nunca elige SQL legacy.
type SesionConsultaRRHHConAmbitoPostgreSQL struct {
	pool       *PoolConsultasRRHHConAmbitoPostgreSQL
	analizador analizadorCanonConsultaRRHH
}

var _ ports.SesionConsultaRRHH = (*SesionConsultaRRHHConAmbitoPostgreSQL)(nil)

func NuevaSesionConsultaRRHHConAmbitoPostgreSQL(pool *PoolConsultasRRHHConAmbitoPostgreSQL) (*SesionConsultaRRHHConAmbitoPostgreSQL, error) {
	if pool == nil || pool.iniciador == nil || pool.iniciador.dependencia != pool {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return &SesionConsultaRRHHConAmbitoPostgreSQL{pool: pool, analizador: analizadorCanonConsultaRRHHPostgreSQL{}}, nil
}

func (s *SesionConsultaRRHHConAmbitoPostgreSQL) validar(ctx context.Context) error {
	if ctx == nil || s == nil || s.pool == nil || s.pool.iniciador == nil ||
		s.pool.iniciador.dependencia != s.pool || dependenciaNula(s.analizador) {
		return ports.ErrConsultaRRHHNoDisponible
	}
	return ctx.Err()
}

func (s *SesionConsultaRRHHConAmbitoPostgreSQL) ConsultarCuadroYRegistrar(ctx context.Context, orden ports.OrdenConsultaCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	if err := s.validar(ctx); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	material, err := orden.ExportacionParaSQL()
	if err != nil || material.ValidarEstructura() != nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	args, err := nuevosArgumentosMaterialConsultaRRHH(material)
	if err != nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	defer args.limpiar()
	c, cap, q := orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	if cap.ClaseAmbito() != ports.AmbitoOrganizacionRRHH || cap.AmbitoRef() != c.OrganizacionRef() {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	proof, err := c.ExportarComprobanteAmbitoParaSQL()
	if err != nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	defer clear(proof)
	sqlArgs := append(argumentosSQLCuadroConsultaRRHH(c.OrganizacionRef(), string(cap.ClaseAmbito()), cap.AmbitoRef(), q, args), string(proof))
	var salida salidaCuadroConsultaRRHH
	defer func() { clear(salida.contenidoCanonico); salida.cursorSiguiente = "" }()
	return ejecutarConsultaRRHHEnTransaccion(ctx, s.pool.iniciador, consultaCuadroRRHHAmbitoPostgreSQL, sqlArgs,
		destinosCuadroConsultaRRHH(&salida), func() (ports.PaginaCuadroRRHH, error) {
			salida.cierre.normalizarInstantesSQL()
			totales, e := salida.construirTotales()
			if e != nil {
				return ports.PaginaCuadroRRHH{}, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: ports.ErrResultadoConsultaRRHHNoConfiable, Causa: e}
			}
			recibo, e := salida.cierre.construirRecibo(c, cap)
			if e != nil {
				return ports.PaginaCuadroRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			_, _, total, e := salida.cierre.enterosSeguros()
			if e != nil {
				return ports.PaginaCuadroRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			pagina, e := s.analizador.analizarCuadro(salida.contenidoCanonico, salida.cursorSiguiente, salida.cierre.generadaEn, total)
			if e != nil {
				return ports.PaginaCuadroRRHH{}, e
			}
			pagina.Lectura = recibo
			pagina.Totales = totales
			if pagina.ValidarParaEjecucionInterna(orden) != nil {
				return ports.PaginaCuadroRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			return pagina, nil
		})
}

func (s *SesionConsultaRRHHConAmbitoPostgreSQL) ConsultarDetalleYRegistrar(ctx context.Context, orden ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	if err := s.validar(ctx); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	material, err := orden.ExportacionParaSQL()
	if err != nil || material.ValidarEstructura() != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	args, err := nuevosArgumentosMaterialConsultaRRHH(material)
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	defer args.limpiar()
	c, cap, q := orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	if cap.ClaseAmbito() != ports.AmbitoOrganizacionRRHH || cap.AmbitoRef() != c.OrganizacionRef() {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	proof, err := c.ExportarComprobanteAmbitoParaSQL()
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	defer clear(proof)
	sqlArgs := append(argumentosSQLDetalleConsultaRRHH(c.OrganizacionRef(), string(cap.ClaseAmbito()), cap.AmbitoRef(), q, args), string(proof))
	var salida salidaDetalleConsultaRRHH
	defer clear(salida.contenidoCanonico)
	return ejecutarConsultaRRHHEnTransaccion(ctx, s.pool.iniciador, consultaDetalleRRHHAmbitoPostgreSQL, sqlArgs,
		destinosDetalleConsultaRRHH(&salida), func() (ports.DetalleExpedienteRRHH, error) {
			salida.cierre.normalizarInstantesSQL()
			recibo, e := salida.cierre.construirRecibo(c, cap)
			if e != nil {
				return ports.DetalleExpedienteRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			_, version, _, e := salida.cierre.enterosSeguros()
			if e != nil {
				return ports.DetalleExpedienteRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			entrada, e := s.analizador.analizarDetalle(salida.contenidoCanonico, salida.cierre.generadaEn, salida.cierre.expedienteRef, version)
			if e != nil {
				return ports.DetalleExpedienteRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			detalle, e := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, recibo)
			if e != nil || detalle.ValidarParaEjecucionInterna(orden) != nil {
				return ports.DetalleExpedienteRRHH{}, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			return detalle, nil
		})
}

// Sentencias cerradas para el pool nominal CA6/CT109.
const (
	consultaCuadroRRHHAmbitoPostgreSQL = `
SELECT contenido_canonico,
       cursor_siguiente,
       esquema,
       acceso_ref,
       secuencia::bigint,
       anterior_sha256,
       huella_sha256,
       vinculo_identidad_huella_sha256,
       alcance_huella_sha256,
       registrada_en,
       auditoria_vec_ref,
       auditoria_vec_huella_sha256,
       consumo_vec_huella_sha256,
       contenido_huella_sha256,
       resultado_huella_sha256,
       cursor_huella_sha256,
       generada_en,
       expediente_ref,
       version_expediente::bigint,
       total,
       recibo_sello_sha256,
       total_filtrado::bigint,
       en_tramitacion::bigint,
       con_incidencia::bigint,
       en_llamamiento::bigint
  FROM vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(
       ROW($1::text, $2::text, $3::text)::
           vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       ROW($4::text, $5::text, $6::text, $7::smallint, $8::text)::
           vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
       $19::jsonb,
       $9::bytea,
       $10::bytea,
       $11::bytea,
       $12::bytea,
       $13::numeric,
       $14::numeric,
       $15::bytea,
       $16::bytea,
       $17::bytea,
       $18::bytea
  )`
	consultaDetalleRRHHAmbitoPostgreSQL = `
SELECT contenido_canonico,
       esquema,
       acceso_ref,
       secuencia::bigint,
       anterior_sha256,
       huella_sha256,
       vinculo_identidad_huella_sha256,
       alcance_huella_sha256,
       registrada_en,
       auditoria_vec_ref,
       auditoria_vec_huella_sha256,
       consumo_vec_huella_sha256,
       contenido_huella_sha256,
       resultado_huella_sha256,
       cursor_huella_sha256,
       generada_en,
       expediente_ref,
       version_expediente::bigint,
       total,
       recibo_sello_sha256
  FROM vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(
       ROW($1::text, $2::text, $3::text)::
           vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       ROW($4::text, $5::numeric)::
           vec_contratacion_temporal.consulta_detalle_rrhh_v1,
       $16::jsonb,
       $6::bytea,
       $7::bytea,
       $8::bytea,
       $9::bytea,
       $10::numeric,
       $11::numeric,
       $12::bytea,
       $13::bytea,
       $14::bytea,
       $15::bytea
  )`
)
