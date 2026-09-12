package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionReservarIntentoCorreoLlamamiento    = "vec_contratacion_temporal.reservar_intento_despacho_correo_llamamiento_v1"
	funcionRegistrarResultadoCorreoLlamamiento = "vec_contratacion_temporal.registrar_resultado_intento_despacho_correo_llamamiento_v1"
	maximoRespuestaReservaCorreoLlamamiento    = 16 * 1024
)

var ErrRegistroCorreoLlamamientoNoDisponible = errors.New("contratacion temporal: registro PostgreSQL de correo de llamamiento no disponible")

// RegistroIntentosCorreoLlamamientoPostgreSQL consume la autorización y reserva
// la barrera durable antes de que la aplicación pueda invocar SMTP. No reintenta
// COMMIT: si su resultado es incierto no entrega la capacidad de finalización.
type RegistroIntentosCorreoLlamamientoPostgreSQL struct{ pool iniciadorTransacciones }

var _ ports.RegistroIntentosCorreoLlamamiento = (*RegistroIntentosCorreoLlamamientoPostgreSQL)(nil)

func NuevoRegistroIntentosCorreoLlamamientoPostgreSQL(pool *pgxpool.Pool) (*RegistroIntentosCorreoLlamamientoPostgreSQL, error) {
	return nuevoRegistroIntentosCorreoLlamamientoPostgreSQL(pool)
}

func nuevoRegistroIntentosCorreoLlamamientoPostgreSQL(pool iniciadorTransacciones) (*RegistroIntentosCorreoLlamamientoPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ErrRegistroCorreoLlamamientoNoDisponible
	}
	return &RegistroIntentosCorreoLlamamientoPostgreSQL{pool: pool}, nil
}

func (r *RegistroIntentosCorreoLlamamientoPostgreSQL) ReservarIntentoCorreoLlamamiento(ctx context.Context, solicitud ports.SolicitudDespacharCorreoLlamamiento, capacidad ports.CapacidadDespachoCorreoLlamamiento) (ports.ReservaIntentoCorreoLlamamiento, ports.CapacidadFinalizacionIntentoCorreoLlamamiento, error) {
	cero := ports.ReservaIntentoCorreoLlamamiento{}
	if ctx == nil || r == nil || dependenciaNula(r.pool) || solicitud.Validar() != nil || ctx.Err() != nil {
		if ctx != nil && ctx.Err() != nil {
			return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, ctx.Err()
		}
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, ErrRegistroCorreoLlamamientoNoDisponible
	}
	material := capacidad.ExportarMaterialParaConsumidor()
	if material.ValidarEstructura() != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	peticion, err := json.Marshal(solicitud)
	if err != nil || len(peticion) == 0 || len(peticion) > maximoRespuestaReservaCorreoLlamamiento {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, ErrRegistroCorreoLlamamientoNoDisponible
	}
	entrada := entradasMaterialCorreoLlamamiento(material)
	defer entrada.borrar()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, errorRegistroCorreoLlamamiento(ctx)
	}
	defer revertirTransaccion(tx)
	if err = configurarSesionCorreoLlamamiento(ctx, tx); err != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, errorRegistroCorreoLlamamiento(ctx)
	}
	var contenido string
	err = tx.QueryRow(ctx, "SELECT "+funcionReservarIntentoCorreoLlamamiento+"($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text",
		string(peticion), entrada.capacidad, entrada.decision, entrada.motivo, entrada.contexto,
		int64(material.PersonaVersion()), int64(material.PerfilVersion()), entrada.payload, entrada.sobre, entrada.evidencia, entrada.raiz).Scan(&contenido)
	if err != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, errorRegistroCorreoLlamamiento(ctx)
	}
	reserva, secreto, err := decodificarReservaCorreoLlamamiento([]byte(contenido), solicitud)
	if err != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	defer borrarBytes(secreto)
	if err = tx.Commit(ctx); err != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, errorRegistroCorreoLlamamiento(ctx)
	}
	// La capacidad sólo cruza la frontera después de un COMMIT confirmado.
	if reserva.YaReservado {
		return reserva, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, nil
	}
	finalizacion, err := ports.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(reserva, secreto)
	if err != nil {
		return cero, ports.CapacidadFinalizacionIntentoCorreoLlamamiento{}, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	return reserva, finalizacion, nil
}

func (r *RegistroIntentosCorreoLlamamientoPostgreSQL) RegistrarResultadoIntentoCorreoLlamamiento(ctx context.Context, reserva ports.ReservaIntentoCorreoLlamamiento, finalizacion ports.CapacidadFinalizacionIntentoCorreoLlamamiento, estado ports.EstadoCorreoLlamamiento, plantilla string) error {
	if ctx == nil || r == nil || dependenciaNula(r.pool) || reserva.Validar() != nil || finalizacion.ValidarPara(reserva) != nil || !estado.EsResultado() || plantilla != ports.PlantillaCorreoLlamamientoV1 {
		return ErrRegistroCorreoLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	secreto := finalizacion.ExportarSecretoParaConsumidor()
	defer borrarBytes(secreto)
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errorRegistroCorreoLlamamiento(ctx)
	}
	defer revertirTransaccion(tx)
	if err = configurarSesionCorreoLlamamiento(ctx, tx); err != nil {
		return errorRegistroCorreoLlamamiento(ctx)
	}
	_, err = tx.Exec(ctx, "SELECT "+funcionRegistrarResultadoCorreoLlamamiento+"($1::text,$2::text,$3,$4::text,$5::text)", reserva.IntentoRef, reserva.SolicitudHuella, secreto, estadoCorreoLlamamientoSQL(estado), plantilla)
	if err != nil {
		return errorRegistroCorreoLlamamiento(ctx)
	}
	if err = tx.Commit(ctx); err != nil {
		return errorRegistroCorreoLlamamiento(ctx)
	}
	return nil
}

func errorRegistroCorreoLlamamiento(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrRegistroCorreoLlamamientoNoDisponible
}

type entradasCorreoLlamamiento struct{ capacidad, decision, motivo, contexto, payload, sobre, evidencia, raiz []byte }

func entradasMaterialCorreoLlamamiento(m interface {
	CapacidadCanonica() []byte
	DecisionCanonica() []byte
	MotivoCanonico() []byte
	ContextoActorCanonico() []byte
	PayloadVECAD3() []byte
	SobreCOSESign1() []byte
	EvidenciaVerificacion() []byte
	RaizPublicaSPKI() []byte
}) entradasCorreoLlamamiento {
	return entradasCorreoLlamamiento{m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
}
func (e entradasCorreoLlamamiento) borrar() {
	for _, b := range [][]byte{e.capacidad, e.decision, e.motivo, e.contexto, e.payload, e.sobre, e.evidencia, e.raiz} {
		borrarBytes(b)
	}
}

func configurarSesionCorreoLlamamiento(ctx context.Context, tx pgx.Tx) error {
	if dependenciaNula(tx) {
		return ErrRegistroCorreoLlamamientoNoDisponible
	}
	_, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`)
	return err
}

type reservaCorreoLlamamientoDTO struct {
	IntentoRef            string
	MessageID             string
	FechaOrigen           time.Time
	SolicitudHuella       string
	Estado                string
	YaReservado           bool
	CapacidadFinalizacion string
}

func decodificarReservaCorreoLlamamiento(b []byte, solicitud ports.SolicitudDespacharCorreoLlamamiento) (ports.ReservaIntentoCorreoLlamamiento, []byte, error) {
	if len(b) == 0 || len(b) > maximoRespuestaReservaCorreoLlamamiento {
		return ports.ReservaIntentoCorreoLlamamiento{}, nil, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	var dto reservaCorreoLlamamientoDTO
	if decodificarJSONEstricto(b, &dto) != nil {
		return ports.ReservaIntentoCorreoLlamamiento{}, nil, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	reserva := ports.ReservaIntentoCorreoLlamamiento{IntentoRef: dto.IntentoRef, MessageID: dto.MessageID, FechaOrigen: normalizarInstantePostgreSQL(dto.FechaOrigen), SolicitudHuella: dto.SolicitudHuella, Estado: estadoCorreoLlamamientoDesdeSQL(dto.Estado), YaReservado: dto.YaReservado}
	if reserva.ValidarPara(solicitud) != nil {
		return ports.ReservaIntentoCorreoLlamamiento{}, nil, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	if reserva.YaReservado {
		if dto.CapacidadFinalizacion != "" {
			return ports.ReservaIntentoCorreoLlamamiento{}, nil, ports.ErrResultadoCorreoLlamamientoNoConfiable
		}
		return reserva, nil, nil
	}
	secreto, err := hex.DecodeString(dto.CapacidadFinalizacion)
	if err != nil || len(secreto) != 32 {
		borrarBytes(secreto)
		return ports.ReservaIntentoCorreoLlamamiento{}, nil, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	return reserva, secreto, nil
}

func estadoCorreoLlamamientoDesdeSQL(v string) ports.EstadoCorreoLlamamiento {
	switch v {
	case "iniciado":
		return ports.CorreoLlamamientoIniciado
	case "no_aceptado_transitorio":
		return ports.CorreoLlamamientoNoAceptadoTransitorio
	case "no_aceptado_permanente":
		return ports.CorreoLlamamientoNoAceptadoPermanente
	case "indeterminado":
		return ports.CorreoLlamamientoIndeterminado
	case "aceptado_por_relay":
		return ports.CorreoLlamamientoAceptadoPorRelay
	default:
		return 0
	}
}
func estadoCorreoLlamamientoSQL(v ports.EstadoCorreoLlamamiento) string {
	switch v {
	case ports.CorreoLlamamientoNoAceptadoTransitorio:
		return "no_aceptado_transitorio"
	case ports.CorreoLlamamientoNoAceptadoPermanente:
		return "no_aceptado_permanente"
	case ports.CorreoLlamamientoIndeterminado:
		return "indeterminado"
	case ports.CorreoLlamamientoAceptadoPorRelay:
		return "aceptado_por_relay"
	default:
		return ""
	}
}
