package postgres

import (
	"context"
	"errors"
	"math"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

const ajustesRegistroEmpleadoB2 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

const (
	consultaFichaEmpleadoB2SQL  = `SELECT vec_personal.consultar_registro_empleado_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaVacantesB2SQL       = `SELECT vec_personal.consultar_vacantes_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	registrarEmpleadoB2SQL      = `SELECT vec_personal.registrar_empleado_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	registrarHechoEmpleadoB2SQL = `SELECT vec_personal.registrar_hecho_empleado_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

var (
	errRegistroEmpleadoB2Invalido     = errors.New("personal: material de registro invalido")
	errRegistroEmpleadoB2Denegado     = errors.New("personal: acceso al registro denegado")
	errRegistroEmpleadoB2NoEncontrado = errors.New("personal: empleado no encontrado")
	errRegistroEmpleadoB2Cobertura    = errors.New("personal: cobertura de vacantes no acreditada")
	errRegistroEmpleadoB2Conflicto    = errors.New("personal: conflicto de registro")
	errRegistroEmpleadoB2NoDisponible = errors.New("personal: registro no disponible")
	errRegistroEmpleadoB2ExcedeLimite = errors.New("personal: respuesta de registro excede su límite")
)

type iniciadorRegistroEmpleadoB2 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RepositorioRegistroEmpleadoB2PostgreSQL struct {
	pool iniciadorRegistroEmpleadoB2
}

func NuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool *pgxpool.Pool) (*RepositorioRegistroEmpleadoB2PostgreSQL, error) {
	return nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
}

func nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool iniciadorRegistroEmpleadoB2) (*RepositorioRegistroEmpleadoB2PostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, errRegistroEmpleadoB2NoDisponible
	}
	return &RepositorioRegistroEmpleadoB2PostgreSQL{pool: pool}, nil
}

// El efecto de la función nominal comprende consumo V3, auditoría y lectura o
// escritura. La respuesta debe superar su contrato antes de confirmar.
func ejecutarRegistroEmpleadoB2[T any](ctx context.Context, pool iniciadorRegistroEmpleadoB2, consulta string, material []byte, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, limiteBytes int, decodificar func([]byte) (T, error)) (T, error) {
	var vacio T
	if ctx == nil || nuloRegistroEmpleadoB2(pool) || len(material) == 0 || a.ValidarEstructura() != nil || a.PersonaVersion() > math.MaxInt64 || a.PerfilVersion() > math.MaxInt64 || decodificar == nil {
		return vacio, errRegistroEmpleadoB2Invalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	piezas := [8][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, pieza := range piezas {
			for i := range pieza {
				pieza[i] = 0
			}
		}
	}()
	parametros := []any{string(material), piezas[0], piezas[1], piezas[2], piezas[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()), piezas[4], piezas[5], piezas[6], piezas[7]}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, normalizarErrorRegistroEmpleadoB2(ctx, err)
	}
	if tx == nil {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroEmpleadoB2); err != nil {
		return vacio, normalizarErrorRegistroEmpleadoB2(ctx, err)
	}
	var bruto []byte
	if err = tx.QueryRow(ctx, consulta, parametros...).Scan(&bruto); err != nil {
		return vacio, normalizarErrorRegistroEmpleadoB2(ctx, err)
	}
	if len(bruto) == 0 || len(bruto) > limiteBytes {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	resultado, err := decodificar(bruto)
	if err != nil {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorRegistroEmpleadoB2(ctx, err)
	}
	confirmada = true
	return resultado, nil
}

func normalizarErrorRegistroEmpleadoB2(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var pg *pgconn.PgError
	// P0002 procede de la acreditación B1 del objetivo dentro del acto SQL.
	// Ausencia, revocación y caducidad tienen la misma salida opaca.
	if errors.As(err, &pg) && (pg.Code == "42501" || pg.Code == "P0002") {
		return errRegistroEmpleadoB2Denegado
	}
	if errors.As(err, &pg) && pg.Code == "22023" {
		return errRegistroEmpleadoB2Invalido
	}
	if errors.As(err, &pg) && pg.Code == "P7404" {
		return errRegistroEmpleadoB2NoEncontrado
	}
	if errors.As(err, &pg) && pg.Code == "P7401" {
		return errRegistroEmpleadoB2Cobertura
	}
	// El catálogo B2 devuelve 23514 para versión ausente, retirada o fuera de
	// vigencia. Todas estas causas comparten el mismo conflicto opaco.
	if errors.As(err, &pg) && (pg.Code == "23505" || pg.Code == "23514") {
		return errRegistroEmpleadoB2Conflicto
	}
	// 54000: la función SQL rechaza una respuesta con más filas de las que
	// admite su contrato (p. ej. ficha propia de más de 200 relaciones).
	if errors.As(err, &pg) && pg.Code == "54000" {
		return errRegistroEmpleadoB2ExcedeLimite
	}
	return errRegistroEmpleadoB2NoDisponible
}

func nuloRegistroEmpleadoB2(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return x.IsNil()
	default:
		return false
	}
}
