package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const normalizarComprobacionSQL = `SELECT p::text, encode(sha256(convert_to(p::text,'UTF8')),'hex') FROM (SELECT $1::jsonb AS p) s`
const registrarComprobacionSQL = `SELECT vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const ajustesComprobacionSQL = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

var referenciaReciboComprobacion = regexp.MustCompile(`^comprobacion:[0-9a-f]{64}$`)
var errRollbackComprobacion = errors.New("rollback de comprobacion fallido")
var errCommitComprobacionIndeterminado = errors.New("commit de comprobacion indeterminado")

type RepositorioComprobacionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioComprobacion = (*RepositorioComprobacionPostgreSQL)(nil)

func NuevoRepositorioComprobacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioComprobacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrComprobacionDependencia
	}
	return &RepositorioComprobacionPostgreSQL{pool}, nil
}

func (r *RepositorioComprobacionPostgreSQL) NormalizarComprobacion(ctx context.Context, q ports.SolicitudComprobacion) (string, string, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return "", "", ports.ErrComprobacionDependencia
	}
	b, err := json.Marshal(q)
	if err != nil || len(b) > 49152 {
		return "", "", ports.ErrComprobacionInvalida
	}
	var canon, huella string
	limite, cancelar := context.WithTimeout(ctx, 15*time.Second)
	defer cancelar()
	err = r.pool.QueryRow(limite, normalizarComprobacionSQL, string(b)).Scan(&canon, &huella)
	if err != nil {
		return "", "", errorComprobacionSQL(err)
	}
	sum := sha256.Sum256([]byte(canon))
	if hex.EncodeToString(sum[:]) != huella || len(canon) > 49152 {
		return "", "", ports.ErrComprobacionDependencia
	}
	return canon, huella, nil
}

func (r *RepositorioComprobacionPostgreSQL) RegistrarComprobacion(ctx context.Context, i ports.IntencionComprobacion) (ports.ReciboComprobacion, error) {
	var cero ports.ReciboComprobacion
	if r == nil || r.pool == nil || ctx == nil || i.CanonPostgreSQL == "" || i.Material.ValidarEstructura() != nil {
		return cero, ports.ErrComprobacionDependencia
	}
	sum := sha256.Sum256([]byte(i.CanonPostgreSQL))
	if hex.EncodeToString(sum[:]) != i.HuellaSHA256 || !intencionComprobacionLigada(i) {
		return cero, ports.ErrComprobacionDependencia
	}
	return registrarComprobacionConRecuperacion(ctx, i, r.registrarUnaComprobacion)
}

func registrarComprobacionConRecuperacion(ctx context.Context, i ports.IntencionComprobacion, ejecutar func(context.Context, ports.IntencionComprobacion) (ports.ReciboComprobacion, error)) (ports.ReciboComprobacion, error) {
	var cero ports.ReciboComprobacion
	var err error
	for intento := 0; intento < 3; intento++ {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		var recibo ports.ReciboComprobacion
		recibo, err = ejecutar(ctx, i)
		if err == nil {
			return recibo, nil
		}
		if errors.Is(err, errCommitComprobacionIndeterminado) {
			if errors.Is(err, errRollbackComprobacion) {
				slog.Warn("bolsa: rollback anterior no confirmado")
			}
			// Un COMMIT sin respuesta puede haber escrito. B133 comprueba de nuevo
			// la misma decisión, clave y material, y devuelve el recibo original.
			limite, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
			recuperado, fallo := ejecutar(limite, i)
			cancelar()
			if fallo != nil || !reciboComprobacionValido(recuperado, i.Solicitud) {
				if fallo == nil {
					fallo = ports.ErrComprobacionDependencia
				}
				return cero, fmt.Errorf("%w: %w", ports.ErrComprobacionResultadoNoConfirmado, errors.Join(err, fallo))
			}
			return recuperado, nil
		}
		if errors.Is(err, errRollbackComprobacion) || !reintentarComprobacion(err) {
			return cero, errorComprobacionSQL(err)
		}
	}
	return cero, errorComprobacionSQL(err)
}

func (r *RepositorioComprobacionPostgreSQL) registrarUnaComprobacion(ctx context.Context, i ports.IntencionComprobacion) (out ports.ReciboComprobacion, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return out, err
	}
	confirmado := false
	defer func() {
		if confirmado {
			return
		}
		c, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if fallo := tx.Rollback(c); fallo != nil && !errors.Is(fallo, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("%w: %w", errRollbackComprobacion, fallo))
		}
	}()
	if _, err = tx.Exec(ctx, ajustesComprobacionSQL); err != nil {
		return out, err
	}
	m := i.Material
	var b []byte
	err = tx.QueryRow(ctx, registrarComprobacionSQL, i.CanonPostgreSQL, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&b)
	if err != nil {
		return out, err
	}
	if len(b) == 0 || len(b) > 4096 {
		return out, ports.ErrComprobacionDependencia
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 11 {
		return out, ports.ErrComprobacionDependencia
	}
	for _, clave := range []string{"comprobacion_ref", "version", "version_esperada", "resultado", "motivo_clave", "version_politica_ofertas", "recibo_ref", "evento_ref", "auditoria_ref", "registrada_en", "reutilizada"} {
		if v, ok := campos[clave]; !ok || string(v) == "null" {
			return out, ports.ErrComprobacionDependencia
		}
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&out) != nil || !errors.Is(d.Decode(new(any)), io.EOF) || !reciboComprobacionValido(out, i.Solicitud) {
		return ports.ReciboComprobacion{}, ports.ErrComprobacionDependencia
	}
	if err = tx.Commit(ctx); err != nil {
		if commitComprobacionIndeterminado(err) {
			return ports.ReciboComprobacion{}, fmt.Errorf("%w: %w", errCommitComprobacionIndeterminado, err)
		}
		return ports.ReciboComprobacion{}, err
	}
	confirmado = true
	return out, nil
}

func commitComprobacionIndeterminado(err error) bool {
	if errors.Is(err, pgx.ErrTxCommitRollback) {
		return false
	}
	var p *pgconn.PgError
	return !errors.As(err, &p)
}

// Compara el contenido semántico, sin confundir JSON de Go con jsonb::text.
func intencionComprobacionLigada(i ports.IntencionComprobacion) bool {
	b, err := json.Marshal(i.Solicitud)
	if err != nil {
		return false
	}
	var enviada, normalizada any
	if json.Unmarshal(b, &enviada) != nil || json.Unmarshal([]byte(i.CanonPostgreSQL), &normalizada) != nil {
		return false
	}
	return reflect.DeepEqual(enviada, normalizada)
}

func reciboComprobacionValido(r ports.ReciboComprobacion, q ports.SolicitudComprobacion) bool {
	return referenciaReciboComprobacion.MatchString(r.ComprobacionRef) && r.ReciboRef == "recibo:"+r.ComprobacionRef && r.EventoRef == "evento:"+r.ComprobacionRef && r.AuditoriaRef != "" && len(r.AuditoriaRef) <= 256 && r.Version == q.VersionEsperada+1 && r.VersionEsperada == q.VersionEsperada && r.VersionPoliticaOfertas == q.Contexto.VersionPoliticaOfertas && (r.Resultado == "pendiente" || r.Resultado == "no_aplicable") && r.MotivoClave != "" && len(r.MotivoClave) <= 80 && !r.RegistradaEn.IsZero() && r.RegistradaEn.Nanosecond()%1000 == 0
}

func reintentarComprobacion(err error) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && (p.Code == "40001" || p.Code == "40P01") && p.Routine != "exec_stmt_raise" && !strings.Contains(p.Where, "RAISE")
}

func errorComprobacionSQL(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errRollbackComprobacion) {
		return fmt.Errorf("%w: %w", ports.ErrComprobacionDependencia, err)
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "VBC01":
			return fmt.Errorf("%w: %w", ports.ErrComprobacionConflicto, err)
		case "42501":
			return fmt.Errorf("%w: %w", vd.ErrAutorizacionDenegada, err)
		case "22023", "22007", "22008":
			return fmt.Errorf("%w: %w", ports.ErrComprobacionInvalida, err)
		}
	}
	return fmt.Errorf("%w: %w", ports.ErrComprobacionDependencia, err)
}
