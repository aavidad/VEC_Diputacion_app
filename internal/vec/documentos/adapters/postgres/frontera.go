package postgres

import (
	"context"
	"errors"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/postgresql"
)

// Valores cerrados de vec_documentos.denegacion_frontera (Documentos-4).
const (
	MotivoFronteraAutenticacion = "autenticacion_requerida"
	MotivoFronteraDenegado      = "acceso_denegado"
	MotivoFronteraDependencia   = "dependencia"
	RutaFronteraConsulta        = "/api/vec/documentos/expedientes/consultas"
	RutaFronteraDescarga        = "/api/vec/documentos/originales/descargas"
	RutaFronteraRegistroExterno = "/api/vec/documentos/externos/registros"
	RutaFronteraOtra            = "otra"
)

var (
	ErrFronteraNoDisponible  = errors.New("documentos: registro de denegaciones no disponible")
	ErrOrdenFronteraInvalida = errors.New("documentos: denegación de frontera inválida")
	correlacionFrontera      = regexp.MustCompile(`^corr_([0-9a-f]{32}|no_disponible)$`)
	actorFrontera            = regexp.MustCompile(`^[A-Za-z0-9:_-]{3,256}$`)
)

// OrdenDenegacionFrontera contiene solo valores cerrados y referencias opacas.
// ActorRef queda vacío si la identidad no llegó a acreditarse.
type OrdenDenegacionFrontera struct {
	CorrelacionRef, Motivo, Ruta, Metodo, ActorRef string
}

func (o OrdenDenegacionFrontera) Validar() error {
	switch o.Motivo {
	case MotivoFronteraAutenticacion, MotivoFronteraDenegado, MotivoFronteraDependencia:
	default:
		return ErrOrdenFronteraInvalida
	}
	switch o.Ruta {
	case RutaFronteraConsulta, RutaFronteraDescarga, RutaFronteraRegistroExterno, RutaFronteraOtra:
	default:
		return ErrOrdenFronteraInvalida
	}
	if (o.Metodo != "POST" && o.Metodo != "otro") || !correlacionFrontera.MatchString(o.CorrelacionRef) ||
		(o.ActorRef != "" && !actorFrontera.MatchString(o.ActorRef)) {
		return ErrOrdenFronteraInvalida
	}
	return nil
}

// RegistradorFrontera escribe con un LOGIN cuya única membresía es
// vec_documentos_auditor; la función SQL lo vuelve a comprobar.
type consultaFrontera interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type RegistradorFrontera struct{ db consultaFrontera }

func NuevoRegistradorFrontera(db *pgxpool.Pool) (*RegistradorFrontera, error) {
	if db == nil {
		return nil, ErrFronteraNoDisponible
	}
	return &RegistradorFrontera{db: db}, nil
}

// Preflight comprueba que el LOGIN puede registrar y no ejecutar fachadas
// documentales ni leer la tabla.
func (r *RegistradorFrontera) Preflight(ctx context.Context) error {
	if r == nil || r.db == nil || ctx == nil {
		return ErrFronteraNoDisponible
	}
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT has_function_privilege('vec_documentos.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE')
 AND NOT has_function_privilege('vec_documentos.listar_expediente_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_table_privilege('vec_documentos.denegacion_frontera','SELECT')`).Scan(&ok)
	if err != nil || !ok {
		return ErrFronteraNoDisponible
	}
	return nil
}

func (r *RegistradorFrontera) RegistrarDenegacion(ctx context.Context, o OrdenDenegacionFrontera) error {
	if r == nil || r.db == nil || ctx == nil {
		return ErrFronteraNoDisponible
	}
	if o.Validar() != nil {
		return ErrOrdenFronteraInvalida
	}
	// Cada llamada es una transacción implícita completa. Solo se repite si
	// PostgreSQL acredita el aborto; se conserva la correlación original.
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var ref string
		if err := r.db.QueryRow(ctx, `SELECT vec_documentos.registrar_denegacion_frontera_v1($1,$2,$3,$4,$5)`,
			o.CorrelacionRef, o.Motivo, o.Ruta, o.Metodo, o.ActorRef).Scan(&ref); err != nil {
			return err
		}
		if ref == "" {
			return ErrFronteraNoDisponible
		}
		return nil
	})
	if err != nil {
		return ErrFronteraNoDisponible
	}
	return nil
}
