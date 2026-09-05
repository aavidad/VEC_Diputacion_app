package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionRegistroRespuestaRecibida      = "contratacion_temporal.llamamiento.respuesta.registrar"
	TipoRecursoRegistroRespuestaRecibida = "respuesta_recibida_llamamiento_contratacion_temporal"
)

// ProveedorRegistroRespuestaRecibida acredita una decisión nueva, ligada al
// material completo, incluso para recuperar un recibo. No verifica el correo.
type ProveedorRegistroRespuestaRecibida interface {
	AutorizarRegistroRespuestaRecibida(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoRegistroRespuestaRecibida(s ports.SolicitudRegistrarRespuestaRecibida) (dominiovec.RecursoAutorizable, error) {
	if err := s.Validar(); err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: s.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoRegistroRespuestaRecibida,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

type RegistroRespuestasRecibidasPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorRegistroRespuestaRecibida
}

var _ ports.RegistroRespuestasRecibidas = (*RegistroRespuestasRecibidasPostgreSQL)(nil)

func NuevoRegistroRespuestasRecibidasPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorRegistroRespuestaRecibida) (*RegistroRespuestasRecibidasPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrRespuestaRecibidaNoDisponible
	}
	return &RegistroRespuestasRecibidasPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (t *RegistroRespuestasRecibidasPostgreSQL) RegistrarRespuestaRecibida(ctx context.Context, s ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
	vacio := ports.RespuestaRecibidaRegistrada{}
	if ctx == nil || t == nil || dependenciaNula(t.pool) || dependenciaNula(t.proveedor) {
		return vacio, ports.ErrRespuestaRecibidaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := s.Validar(); err != nil {
		return vacio, err
	}
	a, err := t.proveedor.AutorizarRegistroRespuestaRecibida(ctx, s)
	if err != nil {
		return vacio, normalizarErrorRespuestaRecibida(ctx, err)
	}
	recurso, err := RecursoRegistroRespuestaRecibida(s)
	huella, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	r := a.ResumenCapacidad()
	if err != nil || errHuella != nil || a.ValidarEstructura() != nil ||
		r.Operacion() != AccionRegistroRespuestaRecibida || r.EfectoRef() != s.ExpedienteRef ||
		r.EfectoHuellaSHA256() != huella || r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	contenido, err := json.Marshal(s)
	if err != nil {
		return vacio, ports.ErrSolicitudRespuestaRecibidaInvalida
	}
	recibo, err := t.registrar(ctx, s, contenido, a)
	if err != nil {
		return vacio, normalizarErrorRespuestaRecibida(ctx, err)
	}
	return recibo, nil
}

func (t *RegistroRespuestasRecibidasPostgreSQL) registrar(ctx context.Context, s ports.SolicitudRegistrarRespuestaRecibida, contenido []byte, a puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.RespuestaRecibidaRegistrada, error) {
	vacio := ports.RespuestaRecibidaRegistrada{}
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, err
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true), set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, err
	}
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var contenidoRecibo string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&contenidoRecibo)
	if err != nil {
		return vacio, err
	}
	var recibo ports.RespuestaRecibidaRegistrada
	if len(contenidoRecibo) == 0 || len(contenidoRecibo) > maximoMaterialComunicacionLlamamiento ||
		decodificarJSONEstricto([]byte(contenidoRecibo), &recibo) != nil {
		return vacio, ports.ErrResultadoRespuestaRecibidaNoConfiable
	}
	recibo.Solicitud.RecibidaEn = normalizarInstantePostgreSQL(recibo.Solicitud.RecibidaEn)
	recibo.RegistradaEn = normalizarInstantePostgreSQL(recibo.RegistradaEn)
	if err := recibo.ValidarPara(s); err != nil {
		return vacio, err
	}
	// No reintentar COMMIT incierto: el mismo intento con autorización nueva
	// recupera el recibo; no genera otra respuesta ni transiciona Bolsa.
	if err = tx.Commit(ctx); err != nil {
		return vacio, err
	}
	return recibo, nil
}

func normalizarErrorRespuestaRecibida(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, permitido := range []error{context.Canceled, context.DeadlineExceeded,
		ports.ErrOperacionRespuestaRecibidaDenegada, ports.ErrClaveRespuestaRecibidaUsada,
		ports.ErrVersionRespuestaRecibidaEnConflicto, ports.ErrSolicitudRespuestaRecibidaInvalida,
		ports.ErrResultadoRespuestaRecibidaNoConfiable} {
		if errors.Is(err, permitido) {
			return permitido
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501", "P0563":
			return ports.ErrOperacionRespuestaRecibidaDenegada
		case "P0560":
			return ports.ErrSolicitudRespuestaRecibidaInvalida
		case "P0561":
			return ports.ErrClaveRespuestaRecibidaUsada
		case "P0562":
			return ports.ErrVersionRespuestaRecibidaEnConflicto
		}
	}
	return ports.ErrRespuestaRecibidaNoDisponible
}
