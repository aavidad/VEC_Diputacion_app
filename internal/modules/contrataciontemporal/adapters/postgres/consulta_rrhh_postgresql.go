package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.SesionConsultaRRHH = (*SesionConsultaRRHHPostgreSQL)(nil)

// SesionConsultaRRHHPostgreSQL ejecuta consumo VEC-AD-3, consulta minimizada y
// registro durable de acceso en una única transacción serializable. No
// interpreta errores SQL de negocio para evitar crear un oráculo de existencia.
type SesionConsultaRRHHPostgreSQL struct {
	pool       iniciadorTransacciones
	analizador analizadorCanonConsultaRRHH
}

// NuevaSesionConsultaRRHHPostgreSQL sólo acepta el pool nominal exclusivo de
// consultas RRHH. Otros adaptadores no pueden reutilizar accidentalmente sus
// privilegios.
func NuevaSesionConsultaRRHHPostgreSQL(
	pool *PoolConsultasRRHHPostgreSQL,
) (*SesionConsultaRRHHPostgreSQL, error) {
	if pool == nil || pool.iniciador == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return nuevaSesionConsultaRRHHPostgreSQL(
		pool.iniciador,
		analizadorCanonConsultaRRHHPostgreSQL{},
	)
}

func nuevaSesionConsultaRRHHPostgreSQL(
	pool iniciadorTransacciones,
	analizador analizadorCanonConsultaRRHH,
) (*SesionConsultaRRHHPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(analizador) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return &SesionConsultaRRHHPostgreSQL{
		pool: pool, analizador: analizador,
	}, nil
}

func (s *SesionConsultaRRHHPostgreSQL) ConsultarCuadroYRegistrar(
	ctx context.Context,
	orden ports.OrdenConsultaCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	if err := s.validarContexto(ctx); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	material, err := orden.ExportacionParaSQL()
	if err != nil || material.ValidarEstructura() != nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	argumentos, err := nuevosArgumentosMaterialConsultaRRHH(material)
	if err != nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	defer argumentos.limpiar()

	contexto, capacidad, solicitud :=
		orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	var salida salidaCuadroConsultaRRHH
	defer func() {
		clear(salida.contenidoCanonico)
		salida.cursorSiguiente = ""
	}()
	argumentosSQL := []any{
		contexto.OrganizacionRef(),
		string(capacidad.ClaseAmbito()),
		capacidad.AmbitoRef(),
		solicitud.Texto(),
		string(solicitud.EstadoClave()),
		string(solicitud.FaseClave()),
		int16(solicitud.Limite()),
		solicitud.Cursor(),
		argumentos.capacidadCanonica,
		argumentos.decisionCanonica,
		argumentos.motivoCanonico,
		argumentos.contextoActorCanonico,
		argumentos.personaVersion,
		argumentos.perfilVersion,
		argumentos.payloadVECAD3,
		argumentos.sobreCOSESign1,
		argumentos.evidenciaVerificacion,
		argumentos.raizPublicaSPKI,
	}
	return ejecutarConsultaRRHHEnTransaccion(
		ctx,
		s.pool,
		consultaCuadroRRHHPostgreSQL,
		argumentosSQL,
		destinosCuadroConsultaRRHH(&salida),
		func() (ports.PaginaCuadroRRHH, error) {
			recibo, err := salida.cierre.construirRecibo(contexto, capacidad)
			if err != nil {
				return ports.PaginaCuadroRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			_, _, total, err := salida.cierre.enterosSeguros()
			if err != nil {
				return ports.PaginaCuadroRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			pagina, err := s.analizador.analizarCuadro(
				salida.contenidoCanonico,
				salida.cursorSiguiente,
				salida.cierre.generadaEn,
				total,
			)
			if err != nil {
				return ports.PaginaCuadroRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			pagina.Lectura = recibo
			if pagina.ValidarParaEjecucionInterna(orden) != nil {
				return ports.PaginaCuadroRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			return pagina, nil
		},
	)
}

func (s *SesionConsultaRRHHPostgreSQL) ConsultarDetalleYRegistrar(
	ctx context.Context,
	orden ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	if err := s.validarContexto(ctx); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	material, err := orden.ExportacionParaSQL()
	if err != nil || material.ValidarEstructura() != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	argumentos, err := nuevosArgumentosMaterialConsultaRRHH(material)
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	defer argumentos.limpiar()

	contexto, capacidad, solicitud :=
		orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	var salida salidaDetalleConsultaRRHH
	defer clear(salida.contenidoCanonico)
	argumentosSQL := []any{
		contexto.OrganizacionRef(),
		string(capacidad.ClaseAmbito()),
		capacidad.AmbitoRef(),
		solicitud.ExpedienteRef(),
		int64(solicitud.VersionObservada()),
		argumentos.capacidadCanonica,
		argumentos.decisionCanonica,
		argumentos.motivoCanonico,
		argumentos.contextoActorCanonico,
		argumentos.personaVersion,
		argumentos.perfilVersion,
		argumentos.payloadVECAD3,
		argumentos.sobreCOSESign1,
		argumentos.evidenciaVerificacion,
		argumentos.raizPublicaSPKI,
	}
	return ejecutarConsultaRRHHEnTransaccion(
		ctx,
		s.pool,
		consultaDetalleRRHHPostgreSQL,
		argumentosSQL,
		destinosDetalleConsultaRRHH(&salida),
		func() (ports.DetalleExpedienteRRHH, error) {
			recibo, err := salida.cierre.construirRecibo(contexto, capacidad)
			if err != nil {
				return ports.DetalleExpedienteRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			_, version, _, err := salida.cierre.enterosSeguros()
			if err != nil {
				return ports.DetalleExpedienteRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			entrada, err := s.analizador.analizarDetalle(
				salida.contenidoCanonico,
				salida.cierre.generadaEn,
				salida.cierre.expedienteRef,
				version,
			)
			if err != nil {
				return ports.DetalleExpedienteRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
				entrada,
				recibo,
			)
			if err != nil || detalle.ValidarParaEjecucionInterna(orden) != nil {
				return ports.DetalleExpedienteRRHH{},
					ports.ErrResultadoConsultaRRHHNoConfiable
			}
			return detalle, nil
		},
	)
}

func ejecutarConsultaRRHHEnTransaccion[T any](
	ctx context.Context,
	pool iniciadorTransacciones,
	consulta string,
	argumentos []any,
	destinos []any,
	validar func() (T, error),
) (T, error) {
	var vacio T
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return vacio, normalizarErrorConsultaRRHH(ctx, err)
	}
	defer revertirTransaccion(tx)
	if err := tx.QueryRow(ctx, consulta, argumentos...).Scan(destinos...); err != nil {
		return vacio, normalizarErrorFilaConsultaRRHH(ctx, err)
	}
	resultado, err := validar()
	if err != nil {
		return vacio, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorConsultaRRHH(ctx, err)
	}
	return resultado, nil
}

func (s *SesionConsultaRRHHPostgreSQL) validarContexto(
	ctx context.Context,
) error {
	if ctx == nil || s == nil || dependenciaNula(s.pool) ||
		dependenciaNula(s.analizador) {
		return ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

type argumentosMaterialConsultaRRHH struct {
	capacidadCanonica     []byte
	decisionCanonica      []byte
	motivoCanonico        []byte
	contextoActorCanonico []byte
	personaVersion        int64
	perfilVersion         int64
	payloadVECAD3         []byte
	sobreCOSESign1        []byte
	evidenciaVerificacion []byte
	raizPublicaSPKI       []byte
}

func nuevosArgumentosMaterialConsultaRRHH(
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (argumentosMaterialConsultaRRHH, error) {
	if material.ValidarEstructura() != nil {
		return argumentosMaterialConsultaRRHH{},
			ports.ErrConsultaRRHHNoDisponible
	}
	return argumentosMaterialConsultaRRHH{
		capacidadCanonica:     material.CapacidadCanonica(),
		decisionCanonica:      material.DecisionCanonica(),
		motivoCanonico:        material.MotivoCanonico(),
		contextoActorCanonico: material.ContextoActorCanonico(),
		personaVersion:        int64(material.PersonaVersion()),
		perfilVersion:         int64(material.PerfilVersion()),
		payloadVECAD3:         material.PayloadVECAD3(),
		sobreCOSESign1:        material.SobreCOSESign1(),
		evidenciaVerificacion: material.EvidenciaVerificacion(),
		raizPublicaSPKI:       material.RaizPublicaSPKI(),
	}, nil
}

func (a *argumentosMaterialConsultaRRHH) limpiar() {
	if a == nil {
		return
	}
	for _, pieza := range [][]byte{
		a.capacidadCanonica,
		a.decisionCanonica,
		a.motivoCanonico,
		a.contextoActorCanonico,
		a.payloadVECAD3,
		a.sobreCOSESign1,
		a.evidenciaVerificacion,
		a.raizPublicaSPKI,
	} {
		clear(pieza)
	}
	*a = argumentosMaterialConsultaRRHH{}
}

func destinosCuadroConsultaRRHH(s *salidaCuadroConsultaRRHH) []any {
	return append(
		[]any{&s.contenidoCanonico, &s.cursorSiguiente},
		destinosCierreConsultaRRHH(&s.cierre)...,
	)
}

func destinosDetalleConsultaRRHH(s *salidaDetalleConsultaRRHH) []any {
	return append(
		[]any{&s.contenidoCanonico},
		destinosCierreConsultaRRHH(&s.cierre)...,
	)
}

func destinosCierreConsultaRRHH(s *salidaCierreConsultaRRHH) []any {
	return []any{
		&s.esquema,
		&s.accesoRef,
		&s.secuencia,
		&s.anteriorSHA256,
		&s.huellaSHA256,
		&s.vinculoIdentidadHuellaSHA256,
		&s.alcanceHuellaSHA256,
		&s.registradaEn,
		&s.auditoriaRef,
		&s.auditoriaHuellaSHA256,
		&s.consumoHuellaSHA256,
		&s.contenidoHuellaSHA256,
		&s.resultadoHuellaSHA256,
		&s.cursorHuellaSHA256,
		&s.generadaEn,
		&s.expedienteRef,
		&s.versionExpediente,
		&s.total,
		&s.reciboSelloSHA256,
	}
}

func normalizarErrorFilaConsultaRRHH(
	ctx context.Context,
	err error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var errorPostgreSQL *pgconn.PgError
	if errors.As(err, &errorPostgreSQL) {
		if errorPostgreSQL.Code == "42501" {
			return ports.ErrConsultaRRHHNoObservable
		}
		return ports.ErrConsultaRRHHNoDisponible
	}
	return ports.ErrResultadoConsultaRRHHNoConfiable
}

func normalizarErrorConsultaRRHH(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var errorPostgreSQL *pgconn.PgError
	if errors.As(err, &errorPostgreSQL) &&
		errorPostgreSQL.Code == "42501" {
		return ports.ErrConsultaRRHHNoObservable
	}
	return ports.ErrConsultaRRHHNoDisponible
}
