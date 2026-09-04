package postgres

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionPrepararFiscalizacion        = "vec_contratacion_temporal.preparar_fiscalizacion_v1"
	esquemaPrepararFiscalizacion        = "vec.contratacion-temporal.preparar-fiscalizacion.v1"
	maximoIntentosPrepararFiscalizacion = 3
)

var _ ports.PreparadorFiscalizacionIdempotente = (*PreparadorFiscalizacionPostgreSQL)(nil)

type PreparadorFiscalizacionPostgreSQL struct {
	pool      iniciadorTransacciones
	generador ports.GeneradorReferenciasFiscalizacion
}

func NuevoPreparadorFiscalizacionPostgreSQL(
	pool *pgxpool.Pool,
	generador ports.GeneradorReferenciasFiscalizacion,
) (*PreparadorFiscalizacionPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(generador) {
		return nil, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return &PreparadorFiscalizacionPostgreSQL{pool: pool, generador: generador}, nil
}

type referenciasPrepararFiscalizacionV1 struct {
	ReservaRef       string `json:"reserva_ref"`
	FiscalizacionRef string `json:"fiscalizacion_ref"`
	ReciboRef        string `json:"recibo_ref"`
	EventoRef        string `json:"evento_ref"`
	RetornoRef       string `json:"retorno_ref"`
}

func nuevasReferenciasPrepararFiscalizacionV1(
	referencias ports.ReferenciasEfectoFiscalizacion,
) referenciasPrepararFiscalizacionV1 {
	return referenciasPrepararFiscalizacionV1{
		ReservaRef:       referencias.ReservaRef,
		FiscalizacionRef: referencias.FiscalizacionRef,
		ReciboRef:        referencias.ReciboRef,
		EventoRef:        referencias.EventoRef,
		RetornoRef:       referencias.RetornoRef,
	}
}

func (r referenciasPrepararFiscalizacionV1) puertos() ports.ReferenciasEfectoFiscalizacion {
	return ports.ReferenciasEfectoFiscalizacion{
		ReservaRef: r.ReservaRef, FiscalizacionRef: r.FiscalizacionRef,
		ReciboRef: r.ReciboRef, EventoRef: r.EventoRef, RetornoRef: r.RetornoRef,
	}
}

type operacionPrepararFiscalizacionV1 struct {
	Esquema               string                             `json:"esquema"`
	Operacion             string                             `json:"operacion"`
	SellosHMAC            sellosPrepararAltaV2               `json:"sellos_hmac"`
	OrganizacionRef       string                             `json:"organizacion_ref"`
	ExpedienteRef         string                             `json:"expediente_ref"`
	VersionExpediente     uint64                             `json:"version_expediente"`
	ActorRef              string                             `json:"actor_ref"`
	PerfilRef             string                             `json:"perfil_ref"`
	Resultado             domain.ResultadoFiscalizacion      `json:"resultado"`
	Observaciones         string                             `json:"observaciones"`
	ReferenciasCandidatas referenciasPrepararFiscalizacionV1 `json:"referencias_candidatas"`
}

func nuevaOperacionPrepararFiscalizacion(
	solicitud ports.SolicitudPrepararFiscalizacion,
	referencias ports.ReferenciasEfectoFiscalizacion,
) (operacionPrepararFiscalizacionV1, error) {
	if solicitud.Validar() != nil || referencias.ValidarPara(solicitud.Material.Resultado) != nil {
		return operacionPrepararFiscalizacionV1{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	sellos, err := nuevosSellosPrepararAltaV2(
		solicitud.AmbitosHMAC, solicitud.HuellasPeticionHMAC,
	)
	if err != nil {
		return operacionPrepararFiscalizacionV1{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	material := solicitud.Material
	return operacionPrepararFiscalizacionV1{
		Esquema:    esquemaPrepararFiscalizacion,
		Operacion:  ports.OperacionRegistrarResultadoFiscalizacion,
		SellosHMAC: sellos, OrganizacionRef: material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef, PerfilRef: material.PerfilRef,
		Resultado: material.Resultado, Observaciones: material.Observaciones,
		ReferenciasCandidatas: nuevasReferenciasPrepararFiscalizacionV1(referencias),
	}, nil
}

func (p *PreparadorFiscalizacionPostgreSQL) PrepararFiscalizacion(
	ctx context.Context,
	solicitud ports.SolicitudPrepararFiscalizacion,
) (ports.PreparacionFiscalizacion, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		dependenciaNula(p.generador) || solicitud.Validar() != nil {
		return ports.PreparacionFiscalizacion{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionFiscalizacion{}, err
	}
	referencias, err := p.generador.GenerarReferenciasFiscalizacion(
		ctx, solicitud.Material.Resultado,
	)
	if err != nil || referencias.ValidarPara(solicitud.Material.Resultado) != nil {
		return ports.PreparacionFiscalizacion{}, errorDependenciaFiscalizacion(ctx)
	}
	operacion, err := nuevaOperacionPrepararFiscalizacion(solicitud, referencias)
	if err != nil {
		return ports.PreparacionFiscalizacion{}, err
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	defer borrarBytes(contenido)

	for intento := 1; intento <= maximoIntentosPrepararFiscalizacion; intento++ {
		preparacion, causa := p.prepararEnTransaccion(ctx, solicitud, operacion, contenido)
		if causa == nil {
			return preparacion, nil
		}
		if ctx.Err() != nil {
			return ports.PreparacionFiscalizacion{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(causa) ||
			intento == maximoIntentosPrepararFiscalizacion {
			return ports.PreparacionFiscalizacion{},
				normalizarErrorPreparacionFiscalizacion(ctx, causa)
		}
	}
	return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
}

func (p *PreparadorFiscalizacionPostgreSQL) prepararEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudPrepararFiscalizacion,
	operacion operacionPrepararFiscalizacionV1,
	contenido []byte,
) (ports.PreparacionFiscalizacion, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return ports.PreparacionFiscalizacion{}, err
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
		return ports.PreparacionFiscalizacion{}, err
	}
	fila := filaPreparacionFiscalizacion{}
	err = tx.QueryRow(ctx, `SELECT resultado, expediente_json, reserva_ref,
		fiscalizacion_ref, recibo_ref, evento_ref, retorno_ref, ambito_hmac,
		huella_peticion_hmac, organizacion_ref, expediente_ref,
		version_expediente, actor_ref, perfil_ref, resultado_fiscalizacion,
		observaciones, estado, recibo_json::text
		FROM `+funcionPrepararFiscalizacion+`($1::jsonb)`, contenido).Scan(
		&fila.resultado, &fila.expedienteJSON, &fila.reservaRef,
		&fila.fiscalizacionRef, &fila.reciboRef, &fila.eventoRef,
		&fila.retornoRef, &fila.ambitoHMAC, &fila.huellaPeticionHMAC,
		&fila.organizacionRef, &fila.expedienteRef, &fila.versionExpediente,
		&fila.actorRef, &fila.perfilRef, &fila.resultadoFiscalizacion,
		&fila.observaciones, &fila.estado, &fila.reciboJSON,
	)
	if err != nil {
		return ports.PreparacionFiscalizacion{}, err
	}
	if fila.resultado == "idempotencia_reutilizada" {
		return ports.PreparacionFiscalizacion{}, ports.ErrClaveIdempotenciaUsada
	}
	preparacion, err := fila.restaurar(solicitud, operacion)
	if err != nil {
		return ports.PreparacionFiscalizacion{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionFiscalizacion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.PreparacionFiscalizacion{}, err
	}
	return preparacion, nil
}

type filaPreparacionFiscalizacion struct {
	resultado              string
	expedienteJSON         string
	reservaRef             string
	fiscalizacionRef       string
	reciboRef              string
	eventoRef              string
	retornoRef             string
	ambitoHMAC             string
	huellaPeticionHMAC     string
	organizacionRef        string
	expedienteRef          string
	versionExpediente      int64
	actorRef               string
	perfilRef              string
	resultadoFiscalizacion string
	observaciones          string
	estado                 string
	reciboJSON             pgtype.Text
}

func (f filaPreparacionFiscalizacion) restaurar(
	solicitud ports.SolicitudPrepararFiscalizacion,
	operacion operacionPrepararFiscalizacionV1,
) (ports.PreparacionFiscalizacion, error) {
	if f.versionExpediente != 5 ||
		!operacion.SellosHMAC.contienePar(f.ambitoHMAC, f.huellaPeticionHMAC) {
		return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	var expediente domain.Expediente
	if decodificarJSONEstricto([]byte(f.expedienteJSON), &expediente) != nil ||
		expediente.Validar() != nil {
		return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	preparacion := ports.PreparacionFiscalizacion{
		Expediente: expediente,
		Referencias: ports.ReferenciasEfectoFiscalizacion{
			ReservaRef: f.reservaRef, FiscalizacionRef: f.fiscalizacionRef,
			ReciboRef: f.reciboRef, EventoRef: f.eventoRef, RetornoRef: f.retornoRef,
		},
		AmbitoIdempotenciaHMAC: f.ambitoHMAC,
		HuellaPeticionHMAC:     f.huellaPeticionHMAC,
		Material: ports.MaterialHuellaFiscalizacion{
			OrganizacionRef: f.organizacionRef, ExpedienteRef: f.expedienteRef,
			VersionExpediente: uint64(f.versionExpediente),
			ActorRef:          f.actorRef, PerfilRef: f.perfilRef,
			Resultado:     domain.ResultadoFiscalizacion(f.resultadoFiscalizacion),
			Observaciones: f.observaciones,
		},
	}
	switch f.estado {
	case string(ports.PreparacionFiscalizacionPreparada):
		if f.resultado != "preparada" || f.reciboJSON.Valid ||
			preparacion.Referencias != operacion.ReferenciasCandidatas.puertos() ||
			!hmac.Equal([]byte(f.ambitoHMAC),
				[]byte(operacion.SellosHMAC.Activo.AmbitoHMAC)) ||
			!hmac.Equal([]byte(f.huellaPeticionHMAC),
				[]byte(operacion.SellosHMAC.Activo.HuellaPeticionHMAC)) {
			return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
		}
		preparacion.Estado = ports.PreparacionFiscalizacionPreparada
	case string(ports.PreparacionFiscalizacionConfirmada):
		if f.resultado != "confirmada" || !f.reciboJSON.Valid {
			return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
		}
		recibo, err := decodificarReciboFiscalizacion(f.reciboJSON.String)
		if err != nil {
			return ports.PreparacionFiscalizacion{}, err
		}
		preparacion.Estado = ports.PreparacionFiscalizacionConfirmada
		preparacion.ReciboConfirmado = &recibo
	default:
		return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	if preparacion.ValidarPara(solicitud) != nil {
		return ports.PreparacionFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return preparacion, nil
}

func normalizarErrorPreparacionFiscalizacion(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrClaveIdempotenciaUsada) ||
		errors.Is(causa, ports.ErrPreparacionFiscalizacionInvalida) {
		return causa
	}
	return ports.ErrPersistenciaFiscalizacionNoDisponible
}

func errorDependenciaFiscalizacion(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrPersistenciaFiscalizacionNoDisponible
}
