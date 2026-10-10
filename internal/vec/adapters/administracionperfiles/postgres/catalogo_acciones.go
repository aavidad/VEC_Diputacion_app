package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// FuenteCatalogoAcciones consulta una publicación histórica exacta. La función
// SQL nominal debe verificar que el registro incluye el censo completo de
// entradas y perfiles; el adaptador no puede reconstruirlo desde una petición.
type FuenteCatalogoAcciones struct{ pool conexion }

var _ ports.FuenteCatalogoAccionesAdministracionV1 = (*FuenteCatalogoAcciones)(nil)

const funcionCatalogoAcciones = "vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text)"

const acreditarFuenteCatalogoAccionesSQL = `SELECT COALESCE(
	pg_catalog.to_regprocedure('` + funcionCatalogoAcciones + `') IS NOT NULL
	AND pg_catalog.has_function_privilege(current_user,
		pg_catalog.to_regprocedure('` + funcionCatalogoAcciones + `'),'EXECUTE'),false)`

const leerCatalogoAccionesSQL = `SELECT catalogo_canon,paquete_canon FROM vec_autorizacion.resolver_catalogo_acciones_administracion_v2($1::text,$2::integer,$3::text)`

const maximoBytesCatalogoAcciones = 16 << 20

// NuevaFuenteCatalogoAcciones no publica nada ni usa un catálogo local. Si la
// función nominal o su ACL faltan, no entrega una fuente productiva.
func NuevaFuenteCatalogoAcciones(ctx context.Context, pool *pgxpool.Pool) (*FuenteCatalogoAcciones, error) {
	if ctx == nil || pool == nil {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nuevaFuenteCatalogoAcciones(ctx, pool)
}

func nuevaFuenteCatalogoAcciones(ctx context.Context, pool conexion) (*FuenteCatalogoAcciones, error) {
	if ctx == nil || ausente(pool) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarFuenteCatalogoAccionesSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &FuenteCatalogoAcciones{pool: pool}, nil
}

func (f *FuenteCatalogoAcciones) ObtenerCatalogoAccionesAdministracionV1(ctx context.Context, ref string, version int, huella string) (domain.CatalogoAccionesAdministracionV1, error) {
	var vacio domain.CatalogoAccionesAdministracionV1
	if f == nil || ausente(f.pool) || ctx == nil || ref == "" || version < 1 || len(huella) != 64 {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	var canon, paquete []byte
	if err := f.pool.QueryRow(ctx, leerCatalogoAccionesSQL, ref, version, huella).Scan(&canon, &paquete); err != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, ports.ConClaseVersionBolsa("catalogo_consulta", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	// La clase sólo distingue la comprobación para el registro técnico; el
	// error sigue siendo el mismo para todos los consumidores.
	catalogo, err := decodificarCatalogoAccionesPublicado(canon, ref, version, huella)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("catalogo_canon", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	if ValidarPaqueteCatalogoAccionesV2(paquete, catalogo,
		catalogo.FuenteRef, catalogo.FuenteVersion, catalogo.FuenteHuellaSHA256) != nil {
		return vacio, ports.ConClaseVersionBolsa("catalogo_paquete", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	return catalogo, nil
}

func decodificarCatalogoAccionesPublicado(canon []byte, ref string, version int, huella string) (domain.CatalogoAccionesAdministracionV1, error) {
	var vacio domain.CatalogoAccionesAdministracionV1
	if len(canon) == 0 || len(canon) > maximoBytesCatalogoAcciones {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var catalogo domain.CatalogoAccionesAdministracionV1
	dec := json.NewDecoder(bytes.NewReader(canon))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&catalogo); err != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if catalogo.Entradas == nil || len(catalogo.Perfiles) == 0 || catalogo.Validar() != nil ||
		catalogo.Referencia != ref || catalogo.Version != version {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	recodificado, err := json.Marshal(catalogo)
	if err != nil || !bytes.Equal(canon, recodificado) {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	calculada, err := catalogo.HuellaSHA256()
	if err != nil || calculada != huella {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return catalogo, nil
}
