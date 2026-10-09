package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	"vec-diputacion-granada/internal/shared/postgresql"
)

// RepositorioInscripcionesPostgreSQL separa escrituras externa/interna de tres
// lectores nominales. El pool lector depende sólo del canal y la acción
// validados por el servidor, nunca de un selector recibido por HTTP.
type RepositorioInscripcionesPostgreSQL struct {
	externo        iniciadorTransacciones
	interno        iniciadorTransacciones
	lectorExterno  iniciadorTransacciones
	lectorEmpleado iniciadorTransacciones
	lectorRRHH     iniciadorTransacciones
}

var _ inscripcion.Repositorio = (*RepositorioInscripcionesPostgreSQL)(nil)

func NuevoRepositorioInscripcionesPostgreSQL(
	externo, interno, lectorExterno, lectorEmpleado, lectorRRHH *pgxpool.Pool,
) (*RepositorioInscripcionesPostgreSQL, error) {
	return nuevoRepositorioInscripcionesPostgreSQL(externo, interno, lectorExterno, lectorEmpleado, lectorRRHH)
}

// La superficie externa sólo recibe el ejecutor propio y su lector. Los
// métodos internos permanecen cerrados porque no existe pool interno.
func NuevoRepositorioInscripcionesExternoPostgreSQL(
	externo, lectorExterno *pgxpool.Pool,
) (*RepositorioInscripcionesPostgreSQL, error) {
	return nuevoRepositorioInscripcionesExternoPostgreSQL(externo, lectorExterno)
}

// La superficie interna consume su ejecutor y lectores de empleado/RRHH.
// Nunca abre el ejecutor externo ni su lector en ese proceso.
func NuevoRepositorioInscripcionesInternoPostgreSQL(
	interno, lectorEmpleado, lectorRRHH *pgxpool.Pool,
) (*RepositorioInscripcionesPostgreSQL, error) {
	return nuevoRepositorioInscripcionesInternoPostgreSQL(interno, lectorEmpleado, lectorRRHH)
}

func nuevoRepositorioInscripcionesExternoPostgreSQL(externo, lectorExterno iniciadorTransacciones) (*RepositorioInscripcionesPostgreSQL, error) {
	if valorNulo(externo) || valorNulo(lectorExterno) || mismaFuenteInscripcion(externo, lectorExterno) {
		return nil, inscripcion.ErrNoDisponible
	}
	return &RepositorioInscripcionesPostgreSQL{externo: externo, lectorExterno: lectorExterno}, nil
}

func nuevoRepositorioInscripcionesInternoPostgreSQL(interno, lectorEmpleado, lectorRRHH iniciadorTransacciones) (*RepositorioInscripcionesPostgreSQL, error) {
	if valorNulo(interno) || valorNulo(lectorEmpleado) || valorNulo(lectorRRHH) ||
		mismaFuenteInscripcion(interno, lectorEmpleado) || mismaFuenteInscripcion(interno, lectorRRHH) ||
		mismaFuenteInscripcion(lectorEmpleado, lectorRRHH) {
		return nil, inscripcion.ErrNoDisponible
	}
	return &RepositorioInscripcionesPostgreSQL{interno: interno, lectorEmpleado: lectorEmpleado, lectorRRHH: lectorRRHH}, nil
}

func nuevoRepositorioInscripcionesPostgreSQL(
	externo, interno, lectorExterno, lectorEmpleado, lectorRRHH iniciadorTransacciones,
) (*RepositorioInscripcionesPostgreSQL, error) {
	fuentes := []iniciadorTransacciones{externo, interno, lectorExterno, lectorEmpleado, lectorRRHH}
	for i, fuente := range fuentes {
		if valorNulo(fuente) {
			return nil, inscripcion.ErrNoDisponible
		}
		for j := 0; j < i; j++ {
			if mismaFuenteInscripcion(fuente, fuentes[j]) {
				return nil, inscripcion.ErrNoDisponible
			}
		}
	}
	return &RepositorioInscripcionesPostgreSQL{
		externo: externo, interno: interno, lectorExterno: lectorExterno,
		lectorEmpleado: lectorEmpleado, lectorRRHH: lectorRRHH,
	}, nil
}

func mismaFuenteInscripcion(a, b iniciadorTransacciones) bool {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if !va.IsValid() || !vb.IsValid() || va.Type() != vb.Type() {
		return false
	}
	if va.Kind() == reflect.Pointer {
		return va.Pointer() == vb.Pointer()
	}
	if va.Type().Comparable() {
		return a == b
	}
	return true
}

func capturaEscrituraInscripcion(actor inscripcion.Actor, accion, audiencia, canal string) ([]byte, error) {
	if !actor.EscrituraValida() || actor.MaterialEscritura == nil {
		return nil, inscripcion.ErrAccesoDenegado
	}
	vinculo, err := actor.Vinculo.Datos()
	if err != nil || actor.Canal != canal || string(vinculo.Superficie) != canal || vinculo.SesionRef != actor.SesionRef ||
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
	var etapa string
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		resultado = nil
		etapa = "abrir"
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return err
		}
		defer revertir(tx)
		etapa = "configurar"
		if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
			return err
		}
		etapa = "ejecutar"
		contenido, err := operacion(tx)
		if err != nil {
			return err
		}
		etapa = "validar_proyeccion"
		if len(contenido) == 0 || len(contenido) > 2*1024*1024 || validar(contenido) != nil {
			return inscripcion.ErrNoDisponible
		}
		etapa = "confirmar"
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		resultado = contenido
		return nil
	})
	if err != nil {
		return nil, errorInscripcionPostgreSQL(ctx, err, etapa)
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

// falloInscripcionPostgreSQL sólo expone etapa y SQLSTATE cerrados. El texto
// del servidor, sus detalles, pistas y parámetros nunca salen de esta capa.
type falloInscripcionPostgreSQL struct {
	nominal  error
	etapa    string
	sqlstate string
}

func (e falloInscripcionPostgreSQL) Error() string {
	return "bolsa inscripción: fallo de persistencia"
}
func (e falloInscripcionPostgreSQL) Unwrap() error { return e.nominal }
func (e falloInscripcionPostgreSQL) DiagnosticoInscripcion() (string, string) {
	return e.etapa, e.sqlstate
}

func errorInscripcionPostgreSQL(ctx context.Context, err error, etapa string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	nominal := inscripcion.ErrNoDisponible
	sqlstate := ""
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if codigoInscripcionSeguro(pgErr.Code) {
			sqlstate = pgErr.Code
		}
		switch pgErr.Code {
		case "42501":
			nominal = inscripcion.ErrAccesoDenegado
		case "B9601":
			nominal = inscripcion.ErrCatalogoCambiado
		case "B9602":
			nominal = inscripcion.ErrPlazoCerrado
		case "B9603":
			nominal = inscripcion.ErrClaveConflicto
		case "B9604":
			nominal = inscripcion.ErrSolicitudExistente
		case "B9605":
			nominal = inscripcion.ErrDeclaracionInvalida
		case "B9606":
			nominal = inscripcion.ErrRequisitoInvalido
		case "B9607":
			// La publicación supera el contrato de la pantalla de inscripción.
			nominal = inscripcion.ErrNoDisponible
		case "B9701":
			nominal = inscripcion.ErrVinculoIdentidadPendiente
		case "B9702":
			nominal = inscripcion.ErrActaNoDisponible
		case "B9703":
			nominal = inscripcion.ErrConflicto
		case "22023":
			nominal = inscripcion.ErrSolicitudInvalida
		case "23505":
			nominal = inscripcion.ErrConflicto
		}
	}
	if etapa != "abrir" && etapa != "configurar" && etapa != "ejecutar" &&
		etapa != "validar_proyeccion" && etapa != "confirmar" {
		etapa = "desconocida"
	}
	return falloInscripcionPostgreSQL{nominal: nominal, etapa: etapa, sqlstate: sqlstate}
}

func codigoInscripcionSeguro(codigo string) bool {
	if len(codigo) != 5 {
		return false
	}
	for _, caracter := range codigo {
		if caracter < '0' || caracter > '9' && caracter < 'A' || caracter > 'Z' {
			return false
		}
	}
	return true
}
