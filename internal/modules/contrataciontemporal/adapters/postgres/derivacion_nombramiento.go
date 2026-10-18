package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var ErrCerrojoDerivacionVS001 = errors.New("VS001: no se pudo liberar el cerrojo de la oferta")
var ErrReversionDerivacionVS001 = errors.New("VS001: no se pudo revertir la transaccion de derivacion")
var ErrDescarteDerivacionVS001 = errors.New("VS001: no se pudo cerrar la conexion reservada")

// La sesión se conserva durante toda la transacción: B135 relee la plaza bajo
// el cerrojo y el snapshot SERIALIZABLE se adquiere después de obtenerlo.
type sesionDerivacionNombramiento interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Release()
	Discard() error
}

type conexionDerivacionNombramiento struct{ *pgxpool.Conn }

func (c conexionDerivacionNombramiento) Discard() error {
	conexion := c.Hijack()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return conexion.Close(ctx)
}

type RegistroDerivacionNombramientoPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.TransaccionDerivacionNombramiento = (*RegistroDerivacionNombramientoPostgreSQL)(nil)

func NuevoRegistroDerivacionNombramientoPostgreSQL(pool *pgxpool.Pool) (*RegistroDerivacionNombramientoPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrDerivacionNombramientoNoDisponible
	}
	return &RegistroDerivacionNombramientoPostgreSQL{pool}, nil
}

func (r *RegistroDerivacionNombramientoPostgreSQL) ConfirmarDerivacionNombramiento(
	ctx context.Context, s ports.SolicitudDerivacionNombramiento,
	a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.ResultadoDerivacionNombramiento, error) {
	vacio := ports.ResultadoDerivacionNombramiento{}
	if r == nil || r.pool == nil || ctx == nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	canon, err := s.Canon()
	if err != nil {
		return vacio, err
	}
	recurso, err := s.Recurso()
	if err != nil {
		return vacio, err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || c.Operacion() != string(domain.AccionDerivarNombramiento) ||
		c.EfectoRef() != s.HijoRef || c.EfectoHuellaSHA256() != h ||
		c.AudienciaConsumo() != ports.AudienciaDerivacionNombramiento {
		return vacio, ports.ErrDerivacionNombramientoDenegada
	}
	actor, err := principalDerivacionNombramiento(a, s.HijoRef, h)
	if err != nil {
		return vacio, err
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return vacio, errorDerivacionNombramiento(ctx, err)
	}
	return ejecutarDerivacionNombramiento(ctx, conexionDerivacionNombramiento{conn}, s, canon, a, actor)
}

func principalDerivacionNombramiento(a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, hijoRef, huella string) (string, error) {
	decision := a.DecisionCanonica()
	defer borrarBytes(decision)
	var d struct {
		DecisionRef     string `json:"decision_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		RecursoRef      string `json:"recurso_ref"`
		Contexto        string `json:"contexto_recurso_huella_sha256"`
		Accion          string `json:"accion"`
		Finalidad       string `json:"finalidad"`
	}
	c := a.ResumenCapacidad()
	suma := sha256.Sum256(decision)
	if json.Unmarshal(decision, &d) != nil || d.DecisionRef != c.DecisionRef() ||
		hex.EncodeToString(suma[:]) != c.DecisionHuellaSHA256() ||
		!domain.ReferenciaOpacaValida(d.PrincipalID) || !domain.ReferenciaOpacaValida(d.PerfilActivoRef) ||
		d.RecursoRef != hijoRef || d.Contexto != huella ||
		d.Accion != string(domain.AccionDerivarNombramiento) || d.Finalidad != "gestionar_contratacion_temporal" {
		return "", ports.ErrDerivacionNombramientoDenegada
	}
	return d.PrincipalID, nil
}

func ejecutarDerivacionNombramiento(ctx context.Context, conn sesionDerivacionNombramiento,
	s ports.SolicitudDerivacionNombramiento, canon []byte,
	a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, actor string,
) (resultado ports.ResultadoDerivacionNombramiento, fallo error) {
	// Si el unlock falla, se retira la conexión del pool. Nunca se devuelve
	// una sesión con un advisory lock vivo a otra petición.
	locked := false
	descartar := false
	defer func() {
		if descartar {
			if err := conn.Discard(); err != nil {
				fallo = errors.Join(fallo, ErrDescarteDerivacionVS001)
			}
			resultado, fallo = ports.ResultadoDerivacionNombramiento{}, errors.Join(fallo, ErrCerrojoDerivacionVS001)
			return
		}
		if !locked {
			conn.Release()
			return
		}
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var liberado bool
		err := conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`,
			"vec_bolsa_llamamientos:resolucion-oferta:"+s.OfertaRef).Scan(&liberado)
		if err != nil || !liberado {
			if err := conn.Discard(); err != nil {
				fallo = errors.Join(fallo, ErrDescarteDerivacionVS001)
			}
			resultado, fallo = ports.ResultadoDerivacionNombramiento{}, errors.Join(fallo, ErrCerrojoDerivacionVS001)
			return
		}
		conn.Release()
	}()
	var tomado bool
	err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,0))`,
		"vec_bolsa_llamamientos:resolucion-oferta:"+s.OfertaRef).Scan(&tomado)
	if err != nil {
		// La respuesta del SELECT pudo perderse después de tomar el cerrojo.
		// Al no conocer el estado, solo cerrar la sesión garantiza liberarlo.
		descartar = true
		return resultado, errorDerivacionNombramiento(ctx, err)
	}
	if !tomado {
		return resultado, ports.ErrDerivacionNombramientoConcurrente
	}
	locked = true
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return resultado, errorDerivacionNombramiento(ctx, err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			descartar = true
			fallo = errors.Join(fallo, ErrReversionDerivacionVS001)
		}
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return resultado, errorDerivacionNombramiento(ctx, err)
	}
	personaVersion, perfilVersion := a.PersonaVersion(), a.PerfilVersion()
	if personaVersion > maximoVersionEnteraPostgreSQL || perfilVersion > maximoVersionEnteraPostgreSQL {
		return resultado, ports.ErrDerivacionNombramientoDenegada
	}
	var salida string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.derivar_nombramiento_ct234_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(canon), a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		int64(personaVersion), int64(perfilVersion), a.PayloadVECAD3(), a.SobreCOSESign1(),
		a.EvidenciaVerificacion(), a.RaizPublicaSPKI()).Scan(&salida)
	if err != nil {
		return resultado, errorDerivacionNombramiento(ctx, err)
	}
	// El hijo incluye la instrucción compartida y, por ello, su recibo puede
	// ocupar más que el material de entrada limitado a 256 KiB.
	if len(salida) < 2 || len(salida) > 1<<20 ||
		decodificarJSONEstricto([]byte(salida), &resultado) != nil || resultado.ValidarPara(s) != nil ||
		resultado.Expediente.Actuaciones[0].ActorRef != actor {
		return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoNoConfiable
	}
	if err = ctx.Err(); err != nil {
		return ports.ResultadoDerivacionNombramiento{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoIndeterminada
	}
	return resultado, nil
}

func errorDerivacionNombramiento(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "22023":
			return ports.ErrDerivacionNombramientoInvalida
		case "42501":
			return ports.ErrDerivacionNombramientoDenegada
		case "23505":
			return ports.ErrDerivacionNombramientoConflicto
		case "40001", "40P01":
			return ports.ErrDerivacionNombramientoConcurrente
		}
	}
	return ports.ErrDerivacionNombramientoNoDisponible
}
