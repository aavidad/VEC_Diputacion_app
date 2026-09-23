// Package postgres implements only Cronos-owned durable functions.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type RepositorioMarcajes struct {
	db        iniciadorMarcaje
	auditoria ports.RegistroResultadoEjecucionMarcaje
}

func NuevoRepositorioMarcajes(pool *pgxpool.Pool, auditoria ports.RegistroResultadoEjecucionMarcaje) (*RepositorioMarcajes, error) {
	if pool == nil || auditoria == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioMarcajes{db: pool, auditoria: auditoria}, nil
}

func (r *RepositorioMarcajes) RegistrarOriginalAutorizado(ctx context.Context, original domain.MarcajeOriginal, material domain.MaterialAutorizacionMarcajePropio, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	if r == nil || r.db == nil || r.auditoria == nil || ctx == nil || original.Validate() != nil || material.Validar() != nil || v3.ValidarEstructura() != nil || original.EmpleadoRef != material.EmpleadoRef || original.ClaveOperacion != material.ClaveOperacion || original.Movimiento != material.Movimiento || !original.InstanteUTC.Equal(material.InstanteUTC) || original.CanalAcreditado != material.Canal {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := material.Canonico()
	if err != nil || !resumenMarcajeValido(v3, material, canonico) {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	evento := ports.ResultadoEjecucionMarcaje{
		DecisionRef: v3.ResumenCapacidad().DecisionRef(), ContextoRef: v3.ResumenCapacidad().ContextoRef(),
		ActorRef: material.ActorRef, PerfilRef: material.PerfilRef,
		Accion: application.AccionRegistrarMarcajePropio, RecursoRef: "marcaje:cronos:" + material.ClaveOperacion,
	}
	secretos := [][]byte{v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(), v3.ContextoActorCanonico(), v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			clear(b)
		}
	}()
	return r.ejecutarTransaccionMarcaje(ctx, evento, func(tx pgx.Tx) (ports.ReciboMarcajePropio, error) {
		if _, err := tx.Exec(ctx, "SELECT set_config('search_path','pg_catalog',true),set_config('timezone','UTC',true),set_config('row_security','on',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','10s',true)"); err != nil {
			return ports.ReciboMarcajePropio{}, errorSeguro(ctx, err)
		}
		var bruto []byte
		err := tx.QueryRow(ctx, `SELECT vec_cronos_v1.registrar_marcaje_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`, string(canonico), secretos[0], secretos[1], secretos[2], secretos[3], v3.PersonaVersion(), v3.PerfilVersion(), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&bruto)
		defer clear(bruto)
		if err != nil {
			return ports.ReciboMarcajePropio{}, errorSeguro(ctx, err)
		}
		var recibido reciboSQL
		if decodificarRecibo(bruto, &recibido) != nil || !reciboValido(recibido, original) {
			return ports.ReciboMarcajePropio{}, errReciboMarcajeInvalido
		}
		return ports.ReciboMarcajePropio{Referencia: recibido.Referencia, InstanteUTC: recibido.InstanteUTC.UTC(), MarcajeOriginalRef: recibido.MarcajeOriginalRef, Replay: *recibido.Replay}, nil
	})
}

func errorSeguro(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "PC002":
			return ports.ErrClaveOperacionEnConflicto
		case "PC003", "42501":
			return ports.ErrDependenciaNoDisponible
		}
	}
	return ports.ErrDependenciaNoDisponible
}

func resumenMarcajeValido(v vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, m domain.MaterialAutorizacionMarcajePropio, canonico []byte) bool {
	recurso, err := application.RecursoMarcajePropio(m)
	if err != nil || len(canonico) == 0 {
		return false
	}
	esperado, err := recurso.HuellaContextoAutorizacionSHA256()
	r := v.ResumenCapacidad()
	return err == nil && v.ValidarEstructura() == nil &&
		r.AudienciaConsumo() == application.AudienciaMarcajePropio &&
		r.Operacion() == application.AccionRegistrarMarcajePropio &&
		r.EfectoRef() == recurso.Referencia && r.EfectoHuellaSHA256() == esperado
}

type reciboSQL struct {
	Referencia         string    `json:"referencia"`
	InstanteUTC        time.Time `json:"instante_utc"`
	MarcajeOriginalRef string    `json:"marcaje_original_ref"`
	Replay             *bool     `json:"replay"`
}

var referenciaRecibo = regexp.MustCompile(`^recibo:cronos:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func decodificarRecibo(b []byte, r *reciboSQL) error {
	if len(b) == 0 || len(b) > 4096 {
		return ports.ErrDependenciaNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(r) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return ports.ErrDependenciaNoDisponible
	}
	return nil
}
func reciboValido(r reciboSQL, m domain.MarcajeOriginal) bool {
	_, offset := r.InstanteUTC.Zone()
	if !referenciaRecibo.MatchString(r.Referencia) || r.MarcajeOriginalRef != "marcaje:cronos:"+m.ClaveOperacion ||
		r.Replay == nil || r.InstanteUTC.IsZero() || offset != 0 || r.InstanteUTC.Nanosecond()%1000 != 0 {
		return false
	}
	if *r.Replay {
		return !r.InstanteUTC.After(m.InstanteUTC)
	}
	return r.InstanteUTC.Equal(m.InstanteUTC)
}
