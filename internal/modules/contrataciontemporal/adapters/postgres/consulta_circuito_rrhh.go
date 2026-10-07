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

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionConsultaCircuitoRRHH      = "contratacion_temporal.circuito.consultar"
	TipoRecursoConsultaCircuitoRRHH = "expediente_circuito_rrhh"
	AudienciaConsultaCircuitoRRHH   = "vec_contratacion_temporal.circuito.consultar.v1"
	maximoRespuestaCircuitoRRHH     = 256 * 1024
)

type materialCanonicoConsultaCircuitoRRHH struct {
	ExpedienteRef     string `json:"expediente_ref"`
	OrganizacionRef   string `json:"organizacion_ref"`
	VersionExpediente uint64 `json:"version_expediente"`
}

func canonConsultaCircuitoRRHH(s ports.SolicitudConsultaCircuitoRRHH) ([]byte, error) {
	if s.Validar() != nil {
		return nil, ports.ErrConsultaCircuitoRRHHInvalida
	}
	return json.Marshal(materialCanonicoConsultaCircuitoRRHH{
		ExpedienteRef: s.ExpedienteRef, OrganizacionRef: s.OrganizacionRef,
		VersionExpediente: s.VersionObservada,
	})
}

func RecursoConsultaCircuitoRRHH(s ports.SolicitudConsultaCircuitoRRHH) (vecdomain.RecursoAutorizable, error) {
	contenido, err := canonConsultaCircuitoRRHH(s)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	defer borrarBytes(contenido)
	huella := sha256.Sum256(contenido)
	return vecdomain.RecursoAutorizable{
		Referencia: s.ExpedienteRef, ModuloID: "contratacion_temporal",
		Tipo:      TipoRecursoConsultaCircuitoRRHH,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(huella[:])},
	}, nil
}

type LectorCircuitoRRHHPostgreSQL struct {
	pool iniciadorTransacciones
}

var _ ports.LectorCircuitoRRHH = (*LectorCircuitoRRHHPostgreSQL)(nil)

func NuevoLectorCircuitoRRHHPostgreSQL(pool *pgxpool.Pool) (*LectorCircuitoRRHHPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	return &LectorCircuitoRRHHPostgreSQL{pool: pool}, nil
}

func (l *LectorCircuitoRRHHPostgreSQL) ConsultarCircuitoRRHH(ctx context.Context, material ports.MaterialConsultaCircuitoRRHH) (ports.ResultadoConsultaCircuitoRRHH, error) {
	vacia := ports.ResultadoConsultaCircuitoRRHH{}
	s := material.Solicitud
	if l == nil || ctx == nil || dependenciaNula(l.pool) || material.ValidarPara(s) != nil {
		return vacia, ports.ErrConsultaCircuitoRRHHDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	recurso, err := RecursoConsultaCircuitoRRHH(s)
	if err != nil {
		return vacia, ports.ErrConsultaCircuitoRRHHInvalida
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	a := material.Autorizacion
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil ||
		resumen.Operacion() != AccionConsultaCircuitoRRHH ||
		resumen.EfectoRef() != s.ExpedienteRef ||
		resumen.EfectoHuellaSHA256() != huella ||
		resumen.AudienciaConsumo() != AudienciaConsultaCircuitoRRHH {
		return vacia, ports.ErrConsultaCircuitoRRHHDenegada
	}
	contenido, err := canonConsultaCircuitoRRHH(s)
	if err != nil {
		return vacia, err
	}
	defer borrarBytes(contenido)
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, secreto := range secretos {
			borrarBytes(secreto)
		}
	}()
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, normalizarErrorConsultaCircuitoRRHH(ctx, err)
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacia, normalizarErrorConsultaCircuitoRRHH(ctx, err)
	}
	var circuitoJSON, flujoJSON []byte
	var version int64
	err = tx.QueryRow(ctx, `SELECT circuito_json,version_expediente,flujo_json
		FROM vec_contratacion_temporal.consultar_circuito_rrhh_v1(
		$1::text,$2::text,$3::bigint,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		// #nosec G115 -- Solicitud.Validar limita la versión a MaxInt64.
		s.OrganizacionRef, s.ExpedienteRef, int64(s.VersionObservada),
		secretos[0], secretos[1], secretos[2], secretos[3],
		// #nosec G115 -- ValidarEstructura limita ambas versiones al máximo exacto V3 (< MaxInt64).
		int64(a.PersonaVersion()),
		// #nosec G115 -- ValidarEstructura limita ambas versiones al máximo exacto V3 (< MaxInt64).
		int64(a.PerfilVersion()),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&circuitoJSON, &version, &flujoJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.Commit(ctx); err != nil {
			return vacia, normalizarErrorConsultaCircuitoRRHH(ctx, err)
		}
		if err = ctx.Err(); err != nil {
			return vacia, err
		}
		return vacia, domain.ErrVersionEnConflicto
	}
	if err != nil {
		return vacia, normalizarErrorConsultaCircuitoRRHH(ctx, err)
	}
	if version <= 0 || uint64(version) != s.VersionObservada ||
		len(circuitoJSON) == 0 || len(circuitoJSON) > maximoRespuestaCircuitoRRHH ||
		len(flujoJSON) == 0 || len(flujoJSON) > maximoRespuestaCircuitoRRHH {
		return vacia, ports.ErrResultadoCircuitoRRHHNoConfiable
	}
	resultado := ports.ResultadoConsultaCircuitoRRHH{
		ExpedienteRef: s.ExpedienteRef, VersionExpediente: uint64(version),
		TransicionesPermitidas: []domain.TransicionCircuitoRRHH{},
	}
	if decodificarJSONEstricto(flujoJSON, &resultado.Flujo) != nil ||
		decodificarJSONEstricto(circuitoJSON, &resultado.Circuito) != nil ||
		resultado.ValidarPara(s) != nil {
		return vacia, ports.ErrResultadoCircuitoRRHHNoConfiable
	}
	if err = ctx.Err(); err != nil {
		return vacia, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacia, normalizarErrorConsultaCircuitoRRHH(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return vacia, err
	}
	return resultado, nil
}

func normalizarErrorConsultaCircuitoRRHH(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return ports.ErrConsultaCircuitoRRHHDenegada
	}
	return ports.ErrConsultaCircuitoRRHHNoDisponible
}
