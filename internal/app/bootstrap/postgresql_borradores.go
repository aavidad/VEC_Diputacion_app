package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

type poolOperativoPostgreSQLBorradores interface {
	Ping(context.Context) error
	QueryRow(context.Context, string, ...any) pgx.Row
	Close()
}

type fabricaPoolPostgreSQLBorradores func(
	context.Context,
	*pgxpool.Config,
) (poolOperativoPostgreSQLBorradores, error)

// PoolsPostgreSQLBorradores mantiene una credencial distinta por capacidad.
// Los adaptadores reciben unicamente el pool nominal que necesitan; no existe
// un pool propietario compartido ni una credencial con la union de permisos.
type PoolsPostgreSQLBorradores struct {
	ejecutorConsulta  poolOperativoPostgreSQLBorradores
	proyectorGobierno poolOperativoPostgreSQLBorradores
	verificadorRecibo poolOperativoPostgreSQLBorradores
	cerrarUnaVez      sync.Once
}

// NuevosPoolsPostgreSQLBorradores analiza, abre, sondea y reidentifica los
// tres pools. Ante cualquier fallo cierra tambien los ya creados.
func NuevosPoolsPostgreSQLBorradores(
	ctx context.Context,
	cfg config.Config,
) (*PoolsPostgreSQLBorradores, error) {
	return nuevosPoolsPostgreSQLBorradores(ctx, cfg, crearPoolPostgreSQLBorradores)
}

func nuevosPoolsPostgreSQLBorradores(
	ctx context.Context,
	cfg config.Config,
	crear fabricaPoolPostgreSQLBorradores,
) (*PoolsPostgreSQLBorradores, error) {
	if ctx == nil || crear == nil {
		return nil, ErrConfiguracionPoolPostgreSQLBorradoresInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(ErrConexionPostgreSQLBorradoresNoDisponible, err)
	}
	ejecutor, proyector, verificador, err := cfg.Normalize().BolsaBorradoresPostgreSQL.DSNSeparados()
	if err != nil {
		return nil, err
	}
	dsn := [3]string{ejecutor, proyector, verificador}
	pools := &PoolsPostgreSQLBorradores{}
	usuarios := make(map[string]struct{}, len(dsn))
	for indice, perfil := range perfilesPoolPostgreSQLBorradores {
		configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(dsn[indice], perfil)
		if err != nil {
			pools.Close()
			return nil, err
		}
		pool, err := crear(ctx, configuracion)
		pools.asignar(indice, pool)
		if err != nil || dependenciaPoolPostgreSQLBorradoresNula(pool) {
			pools.Close()
			return nil, errorPostgreSQLBorradoresCerrado(ctx, ErrConexionPostgreSQLBorradoresNoDisponible)
		}
		ctxSonda, cancelar := context.WithTimeout(ctx, duracionSondaPostgreSQLBorradores)
		err = pool.Ping(ctxSonda)
		cancelar()
		if err != nil {
			pools.Close()
			if errors.Is(err, ErrIdentidadPostgreSQLBorradoresInvalida) {
				return nil, errorPostgreSQLBorradoresCerrado(ctx, ErrIdentidadPostgreSQLBorradoresInvalida)
			}
			return nil, errorPostgreSQLBorradoresCerrado(ctx, ErrConexionPostgreSQLBorradoresNoDisponible)
		}
		usuario, err := comprobarIdentidadPoolPostgreSQLBorradores(ctx, pool, perfil.rolEsperado)
		if err != nil {
			pools.Close()
			return nil, err
		}
		if _, repetido := usuarios[usuario]; repetido {
			pools.Close()
			return nil, ErrIdentidadPostgreSQLBorradoresInvalida
		}
		usuarios[usuario] = struct{}{}
	}
	return pools, nil
}

func dependenciaPoolPostgreSQLBorradoresNula(pool any) bool {
	if pool == nil {
		return true
	}
	valor := reflect.ValueOf(pool)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func crearPoolPostgreSQLBorradores(
	ctx context.Context,
	configuracion *pgxpool.Config,
) (poolOperativoPostgreSQLBorradores, error) {
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		return nil, ErrConexionPostgreSQLBorradoresNoDisponible
	}
	return pool, nil
}

func (p *PoolsPostgreSQLBorradores) asignar(
	indice int,
	pool poolOperativoPostgreSQLBorradores,
) {
	switch indice {
	case 0:
		p.ejecutorConsulta = pool
	case 1:
		p.proyectorGobierno = pool
	case 2:
		p.verificadorRecibo = pool
	}
}

func (p *PoolsPostgreSQLBorradores) EjecutorConsulta() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	pool, _ := p.ejecutorConsulta.(*pgxpool.Pool)
	return pool
}

func (p *PoolsPostgreSQLBorradores) ProyectorGobierno() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	pool, _ := p.proyectorGobierno.(*pgxpool.Pool)
	return pool
}

func (p *PoolsPostgreSQLBorradores) VerificadorRecibo() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	pool, _ := p.verificadorRecibo.(*pgxpool.Pool)
	return pool
}

// Close es idempotente y cierra en orden inverso a la adquisicion.
func (p *PoolsPostgreSQLBorradores) Close() {
	if p == nil {
		return
	}
	p.cerrarUnaVez.Do(func() {
		for _, pool := range []poolOperativoPostgreSQLBorradores{
			p.verificadorRecibo, p.proyectorGobierno, p.ejecutorConsulta,
		} {
			if !dependenciaPoolPostgreSQLBorradoresNula(pool) {
				pool.Close()
			}
		}
	})
}
