package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	"vec-diputacion-granada/internal/shared/postgresql"
)

// RepositorioInscripcionesPostgreSQL mantiene separados los tres privilegios
// técnicos. Cada método elige su pool de forma fija; nunca acepta el rol del
// cliente como argumento SQL.
type RepositorioInscripcionesPostgreSQL struct {
	externo iniciadorTransacciones
	interno iniciadorTransacciones
	lector  iniciadorTransacciones
}

var _ inscripcion.Repositorio = (*RepositorioInscripcionesPostgreSQL)(nil)

func NuevoRepositorioInscripcionesPostgreSQL(externo, interno, lector *pgxpool.Pool) (*RepositorioInscripcionesPostgreSQL, error) {
	return nuevoRepositorioInscripcionesPostgreSQL(externo, interno, lector)
}

func nuevoRepositorioInscripcionesPostgreSQL(externo, interno, lector iniciadorTransacciones) (*RepositorioInscripcionesPostgreSQL, error) {
	if valorNulo(externo) || valorNulo(interno) || valorNulo(lector) {
		return nil, inscripcion.ErrNoDisponible
	}
	return &RepositorioInscripcionesPostgreSQL{externo: externo, interno: interno, lector: lector}, nil
}

func capturaEscrituraInscripcion(actor inscripcion.Actor, accion, audiencia, canal string) ([]byte, error) {
	if !actor.EscrituraValida() || actor.MaterialEscritura == nil {
		return nil, inscripcion.ErrAccesoDenegado
	}
	vinculo, err := actor.Vinculo.Datos()
	if err != nil || string(vinculo.Superficie) != canal || vinculo.SesionRef != actor.SesionRef ||
		vinculo.PrincipalID != actor.PersonaRef || vinculo.PerfilActivoRef != actor.PerfilRef ||
		vinculo.CuentaRef != actor.ResultadoContexto.Contexto.Instantanea.CuentaRef {
		return nil, inscripcion.ErrAccesoDenegado
	}
	resumen := actor.MaterialEscritura.ResumenCapacidad()
	if resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia ||
		!resumen.ExpiraEn().After(time.Now().UTC()) {
		return nil, inscripcion.ErrAccesoDenegado
	}
	captura, err := json.Marshal(struct {
		PersonaRef string `json:"persona_ref"`
		PerfilRef  string `json:"perfil_ref"`
		CuentaRef  string `json:"cuenta_ref"`
		Canal      string `json:"canal"`
		Idioma     string `json:"idioma"`
	}{actor.PersonaRef, actor.PerfilRef, vinculo.CuentaRef, canal, idiomaInscripcion(actor.Idioma)})
	if err != nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return captura, nil
}

func idiomaInscripcion(idioma string) string {
	if idioma == "" {
		return "es"
	}
	return idioma
}

func argumentosV3Inscripcion(actor inscripcion.Actor) []any {
	m := actor.MaterialEscritura
	return []any{
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	}
}

// transaccionInscripcion abre una instantánea SERIALIZABLE escribible incluso
// para GET: el asiento de lectura se confirma en esta misma transacción. La
// proyección se entrega al llamador sólo después de COMMIT.
func transaccionInscripcion(ctx context.Context, pool iniciadorTransacciones, operacion func(pgx.Tx) ([]byte, error), validar func([]byte) error) ([]byte, error) {
	if ctx == nil || valorNulo(pool) || operacion == nil || validar == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var resultado []byte
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		resultado = nil
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return err
		}
		defer revertir(tx)
		if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
			return err
		}
		contenido, err := operacion(tx)
		if err != nil {
			return err
		}
		if len(contenido) == 0 || len(contenido) > 2*1024*1024 || validar(contenido) != nil {
			return inscripcion.ErrNoDisponible
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		resultado = contenido
		return nil
	})
	if err != nil {
		return nil, errorInscripcionPostgreSQL(ctx, err)
	}
	return resultado, nil
}

func decodificarInscripcionEstricta(contenido []byte, destino any) error {
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	if err := lector.Decode(destino); err != nil {
		return inscripcion.ErrNoDisponible
	}
	if err := lector.Decode(new(any)); !errors.Is(err, io.EOF) {
		return inscripcion.ErrNoDisponible
	}
	return nil
}

func errorInscripcionPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501":
			return inscripcion.ErrAccesoDenegado
		case "B9601":
			return inscripcion.ErrCatalogoCambiado
		case "B9602":
			return inscripcion.ErrPlazoCerrado
		case "B9603":
			return inscripcion.ErrClaveConflicto
		case "B9604":
			return inscripcion.ErrSolicitudExistente
		case "B9605":
			return inscripcion.ErrDeclaracionInvalida
		case "22023":
			return inscripcion.ErrSolicitudInvalida
		case "23505":
			return inscripcion.ErrConflicto
		}
	}
	return inscripcion.ErrNoDisponible
}
