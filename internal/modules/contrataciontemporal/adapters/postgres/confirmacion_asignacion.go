package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionConfirmarAsignacion        = "vec_contratacion_temporal.confirmar_asignacion_v1"
	maximoIntentosConfirmarAsignacion = 3
	// La clave VEC-AD-3 publicada pertenece al consumidor transaccional de CT.
	// El nombre histórico conserva "alta", pero el efecto y la operación quedan
	// ligados de forma independiente por la capacidad y por esta transacción.
	audienciaConfirmarAsignacionV1 = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
)

var _ ports.TransaccionAsignaciones = (*TransaccionAsignacionesPostgreSQL)(nil)

type TransaccionAsignacionesPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor proveedorMaterialConfirmacionAsignacion
}

type proveedorMaterialConfirmacionAsignacion interface {
	ProveerMaterialConfirmacionAsignacion(
		context.Context,
		ports.OrdenConfirmarAsignacion,
	) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func NuevaTransaccionAsignacionesPostgreSQL(
	pool *pgxpool.Pool,
	proveedor proveedorMaterialConfirmacionAsignacion,
) (*TransaccionAsignacionesPostgreSQL, error) {
	return nuevaTransaccionAsignacionesPostgreSQL(pool, proveedor)
}

func nuevaTransaccionAsignacionesPostgreSQL(
	pool iniciadorTransacciones,
	proveedor proveedorMaterialConfirmacionAsignacion,
) (*TransaccionAsignacionesPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrPersistenciaAsignacionNoDisponible
	}
	return &TransaccionAsignacionesPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

type entradasConfirmarAsignacion struct {
	contenido       []byte
	capacidad       []byte
	decision        []byte
	motivo          []byte
	contextoActor   []byte
	personaVersion  int64
	perfilVersion   int64
	payloadVECAD3   []byte
	sobreCOSESign1  []byte
	evidencia       []byte
	raizPublicaSPKI []byte
}

func (e *entradasConfirmarAsignacion) borrar() {
	if e == nil {
		return
	}
	for _, contenido := range [][]byte{
		e.contenido, e.capacidad, e.decision, e.motivo, e.contextoActor,
		e.payloadVECAD3, e.sobreCOSESign1, e.evidencia, e.raizPublicaSPKI,
	} {
		borrarBytes(contenido)
	}
}

func (t *TransaccionAsignacionesPostgreSQL) ConfirmarAsignacion(
	ctx context.Context,
	orden ports.OrdenConfirmarAsignacion,
) (ports.ReciboAsignacion, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) || dependenciaNula(t.proveedor) {
		return ports.ReciboAsignacion{}, ports.ErrOrdenAsignacionInvalida
	}
	material, err := t.proveedor.ProveerMaterialConfirmacionAsignacion(ctx, orden)
	if err != nil {
		return ports.ReciboAsignacion{}, errorDependenciaAsignacion(ctx)
	}
	entradas, err := prepararEntradasConfirmarAsignacion(orden, material)
	if err != nil {
		return ports.ReciboAsignacion{}, err
	}
	defer entradas.borrar()
	for intento := 1; intento <= maximoIntentosConfirmarAsignacion; intento++ {
		recibo, causa := t.confirmarEnTransaccion(ctx, orden, entradas)
		if causa == nil {
			return recibo, nil
		}
		if ctx.Err() != nil {
			return ports.ReciboAsignacion{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(causa) || intento == maximoIntentosConfirmarAsignacion {
			return ports.ReciboAsignacion{}, normalizarErrorConfirmacionAsignacion(ctx, causa)
		}
	}
	return ports.ReciboAsignacion{}, ports.ErrPersistenciaAsignacionNoDisponible
}

func prepararEntradasConfirmarAsignacion(
	orden ports.OrdenConfirmarAsignacion,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (entradasConfirmarAsignacion, error) {
	evidenciaOrden, err := orden.Datos()
	if err != nil || material.ValidarEstructura() != nil {
		return entradasConfirmarAsignacion{}, ports.ErrOrdenAsignacionInvalida
	}
	solicitud, errSolicitud := evidenciaOrden.SolicitudV3.Datos()
	confirmacion, errConfirmacion := evidenciaOrden.ConfirmacionV3.Datos()
	vinculo, errVinculo := solicitud.VinculoAutenticacionActor.Datos()
	huellaRecurso, errRecurso := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if errSolicitud != nil || errConfirmacion != nil || errVinculo != nil ||
		errRecurso != nil || resumen.ValidarEstructura() != nil ||
		resumen.DecisionRef() != confirmacion.DecisionRef ||
		resumen.DecisionHuellaSHA256() != confirmacion.DecisionHuellaSHA256 ||
		resumen.ContextoRef() != vinculo.RegistroContextoRef ||
		resumen.ContextoHuellaSHA256() != vinculo.ContextoActorHuellaSHA256 ||
		resumen.Operacion() != solicitud.Accion ||
		resumen.EfectoRef() != evidenciaOrden.Material.ExpedienteRef ||
		resumen.EfectoHuellaSHA256() != huellaRecurso ||
		resumen.AudienciaConsumo() != audienciaConfirmarAsignacionV1 ||
		!capacidadBreveContenidaEnConcesion(resumen, confirmacion) {
		return entradasConfirmarAsignacion{}, ports.ErrPersistenciaAsignacionNoDisponible
	}
	contenido, err := codificarOperacionConfirmarAsignacion(orden)
	if err != nil {
		return entradasConfirmarAsignacion{}, err
	}
	return entradasConfirmarAsignacion{
		contenido: contenido,
		capacidad: material.CapacidadCanonica(), decision: material.DecisionCanonica(),
		motivo: material.MotivoCanonico(), contextoActor: material.ContextoActorCanonico(),
		personaVersion: int64(material.PersonaVersion()), perfilVersion: int64(material.PerfilVersion()),
		payloadVECAD3: material.PayloadVECAD3(), sobreCOSESign1: material.SobreCOSESign1(),
		evidencia: material.EvidenciaVerificacion(), raizPublicaSPKI: material.RaizPublicaSPKI(),
	}, nil
}

func (t *TransaccionAsignacionesPostgreSQL) confirmarEnTransaccion(
	ctx context.Context,
	orden ports.OrdenConfirmarAsignacion,
	entradas entradasConfirmarAsignacion,
) (ports.ReciboAsignacion, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ports.ReciboAsignacion{}, err
	}
	defer revertirTransaccion(tx)
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		return ports.ReciboAsignacion{}, err
	}
	var ahora time.Time
	if err = tx.QueryRow(ctx,
		`SELECT date_trunc('microseconds', clock_timestamp())`,
	).Scan(&ahora); err != nil {
		return ports.ReciboAsignacion{}, err
	}
	ahora = normalizarInstantePostgreSQL(ahora)
	if err = orden.ValidarDentroDeTransaccion(ahora); err != nil {
		return ports.ReciboAsignacion{}, fmt.Errorf("%w: vigencia precommit", err)
	}
	var reciboJSON string
	err = tx.QueryRow(ctx, `SELECT recibo_json::text FROM `+
		funcionConfirmarAsignacion+`($1::jsonb,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		entradas.contenido, entradas.capacidad, entradas.decision, entradas.motivo,
		entradas.contextoActor, entradas.personaVersion, entradas.perfilVersion,
		entradas.payloadVECAD3, entradas.sobreCOSESign1, entradas.evidencia,
		entradas.raizPublicaSPKI,
	).Scan(&reciboJSON)
	if err != nil {
		return ports.ReciboAsignacion{}, err
	}
	recibo, err := decodificarReciboConfirmacionAsignacion(reciboJSON)
	if err != nil ||
		recibo.ValidarParaOrden(orden) != nil {
		return ports.ReciboAsignacion{}, ports.ErrResultadoAsignacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboAsignacion{}, err
	}
	return recibo, nil
}

func decodificarReciboConfirmacionAsignacion(
	contenido string,
) (ports.ReciboAsignacion, error) {
	var recibo ports.ReciboAsignacion
	if decodificarJSONEstricto([]byte(contenido), &recibo) != nil {
		return ports.ReciboAsignacion{}, ports.ErrResultadoAsignacionNoConfiable
	}
	return normalizarReciboAsignacionPostgreSQL(recibo), nil
}

func normalizarReciboAsignacionPostgreSQL(
	recibo ports.ReciboAsignacion,
) ports.ReciboAsignacion {
	recibo.ConfirmadaEn = normalizarInstantePostgreSQL(recibo.ConfirmadaEn)
	return recibo
}

func normalizarErrorConfirmacionAsignacion(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrOrdenAsignacionInvalida) ||
		errors.Is(causa, ports.ErrResultadoAsignacionNoConfiable) {
		return causa
	}
	return ports.ErrPersistenciaAsignacionNoDisponible
}
