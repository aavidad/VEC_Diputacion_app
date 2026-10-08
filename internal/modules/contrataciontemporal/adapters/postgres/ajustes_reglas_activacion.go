package postgres

import (
	"context"
	"errors"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	"vec-diputacion-granada/internal/vec/reglas"
)

// CT191 concede EXECUTE al rol ejecutor. No se lee la tabla de activaciones
// desde la aplicación: esta fachada comprueba también la base publicada.
const leerActivacionBaseAjustesCTSQL = `SELECT estado,secuencia,catalogo_id,version,huella_sha256,aprobacion_ref FROM vec_contratacion_temporal.leer_activacion_regla_base_v1()`

func (r *RepositorioAjustesReglasCT) LeerActivacion(ctx context.Context) (app.ActivacionBase, error) {
	if r == nil || dependenciaNula(r.pool) || ctx == nil || ctx.Err() != nil {
		return app.ActivacionBase{}, app.ErrNoDisponible
	}
	var estado string
	var secuencia, version *int64
	var catalogoID, huella, aprobacion *string
	err := r.pool.QueryRow(ctx, leerActivacionBaseAjustesCTSQL).Scan(
		&estado, &secuencia, &catalogoID, &version, &huella, &aprobacion)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return app.ActivacionBase{}, err
		}
		return app.ActivacionBase{}, app.ErrNoDisponible
	}
	resultado := app.ActivacionBase{Estado: estado}
	switch estado {
	case "sin_publicar":
		if secuencia != nil || catalogoID != nil || version != nil || huella != nil || aprobacion != nil {
			return app.ActivacionBase{}, app.ErrNoDisponible
		}
	case "inactiva":
		if secuencia == nil || *secuencia < 1 || *secuencia > maximoVersionAjustesCT ||
			catalogoID != nil || version != nil || huella != nil || aprobacion != nil {
			return app.ActivacionBase{}, app.ErrNoDisponible
		}
		resultado.Secuencia = *secuencia
	case "activa":
		if secuencia == nil || *secuencia < 1 || *secuencia > maximoVersionAjustesCT ||
			catalogoID == nil || *catalogoID != reglas.CatalogoContratacionTemporal ||
			version == nil || *version < 1 || *version > maximoVersionAjustesCT ||
			huella == nil || !huellaSHA256CTValida(*huella) || aprobacion == nil || *aprobacion == "" {
			return app.ActivacionBase{}, app.ErrNoDisponible
		}
		resultado.Secuencia, resultado.CatalogoID, resultado.Version = *secuencia, *catalogoID, int(*version)
		resultado.HuellaSHA256, resultado.AprobacionRef = *huella, *aprobacion
	default:
		return app.ActivacionBase{}, app.ErrNoDisponible
	}
	return resultado, nil
}

// La fachada se consulta fuera de la transacción de CT148 para la pantalla y
// el preflight. CT191 repite la guarda dentro de la transacción de escritura.
