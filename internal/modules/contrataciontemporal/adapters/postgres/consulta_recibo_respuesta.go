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
	AccionConsultaReciboRespuesta      = "contratacion_temporal.llamamiento.respuesta.consultar_recibo"
	TipoRecursoConsultaReciboRespuesta = "respuesta_recibida_comunicacion_ct"
)

type ProveedorConsultaReciboRespuesta interface {
	AutorizarConsultaReciboRespuesta(context.Context, ports.SolicitudConsultaReciboRespuesta) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoConsultaReciboRespuesta(s ports.SolicitudConsultaReciboRespuesta) (dominiovec.RecursoAutorizable, error) {
	if err := s.Validar(); err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, ports.ErrConsultaReciboRespuestaInvalida
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: s.ComunicacionRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoConsultaReciboRespuesta,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

type LectorReciboRespuestaPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorConsultaReciboRespuesta
}

var _ ports.LectorReciboRespuesta = (*LectorReciboRespuestaPostgreSQL)(nil)

func NuevoLectorReciboRespuestaPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorConsultaReciboRespuesta) (*LectorReciboRespuestaPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrConsultaReciboRespuestaFallo
	}
	return &LectorReciboRespuestaPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (l *LectorReciboRespuestaPostgreSQL) ConsultarReciboRespuesta(ctx context.Context, s ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	vacio := ports.ReciboRespuestaConsultado{}
	if l == nil || ctx == nil || dependenciaNula(l.pool) || dependenciaNula(l.proveedor) {
		return vacio, ports.ErrConsultaReciboRespuestaFallo
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recurso, err := RecursoConsultaReciboRespuesta(s)
	if err != nil {
		return vacio, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	a, err := l.proveedor.AutorizarConsultaReciboRespuesta(ctx, s)
	if err != nil {
		return vacio, normalizarErrorConsultaReciboRespuesta(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	r := a.ResumenCapacidad()
	if a.ValidarEstructura() != nil || r.Operacion() != AccionConsultaReciboRespuesta ||
		r.EfectoRef() != s.ComunicacionRef || r.EfectoHuellaSHA256() != huella ||
		r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	contenido, err := json.Marshal(s)
	if err != nil {
		return vacio, ports.ErrConsultaReciboRespuestaInvalida
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, normalizarErrorConsultaReciboRespuesta(ctx, err)
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, normalizarErrorConsultaReciboRespuesta(ctx, err)
	}
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var resultado string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`, string(contenido),
		secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&resultado)
	if err != nil {
		return vacio, normalizarErrorConsultaReciboRespuesta(ctx, err)
	}
	var envoltorio struct {
		Encontrado bool                            `json:"Encontrado"`
		Recibo     ports.ReciboRespuestaConsultado `json:"Recibo"`
	}
	if len(resultado) == 0 || len(resultado) > 4096 || decodificarJSONEstricto([]byte(resultado), &envoltorio) != nil {
		return vacio, ports.ErrReciboRespuestaNoConfiable
	}
	// La auditoría de lectura se conserva también para una búsqueda sin fila.
	if envoltorio.Encontrado {
		envoltorio.Recibo.RegistradaEn = envoltorio.Recibo.RegistradaEn.UTC()
		if envoltorio.Recibo.ValidarPara(s) != nil {
			return vacio, ports.ErrReciboRespuestaNoConfiable
		}
	} else if envoltorio.Recibo != vacio {
		return vacio, ports.ErrReciboRespuestaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorConsultaReciboRespuesta(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !envoltorio.Encontrado {
		return vacio, ports.ErrReciboRespuestaNoEncontrado
	}
	return envoltorio.Recibo, nil
}

func normalizarErrorConsultaReciboRespuesta(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, e := range []error{context.Canceled, context.DeadlineExceeded, ports.ErrConsultaReciboRespuestaInvalida,
		ports.ErrReciboRespuestaNoEncontrado, ports.ErrConsultaReciboRespuestaDenegada, ports.ErrReciboRespuestaNoConfiable} {
		if errors.Is(err, e) {
			return e
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P1390":
			return ports.ErrConsultaReciboRespuestaInvalida
		case "P1393", "42501":
			return ports.ErrConsultaReciboRespuestaDenegada
		}
	}
	return ports.ErrConsultaReciboRespuestaFallo
}
