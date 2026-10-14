package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const consultaUnidadTrabajoCronosSQL = `SELECT vec_personal.consultar_unidad_trabajo_propia_cronos_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioUnidadTrabajoCronosPostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioUnidadTrabajoCronosPostgreSQL(pool *pgxpool.Pool) (*RepositorioUnidadTrabajoCronosPostgreSQL, error) {
	return nuevoRepositorioUnidadTrabajoCronos(pool)
}
func nuevoRepositorioUnidadTrabajoCronos(pool iniciadorRegistroEmpleadoB2) (*RepositorioUnidadTrabajoCronosPostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	return &RepositorioUnidadTrabajoCronosPostgreSQL{pool}, nil
}

func (r *RepositorioUnidadTrabajoCronosPostgreSQL) ConsultarUnidadTrabajoPropiaCronos(ctx context.Context, o ports.OrdenUnidadTrabajoCronos) (ports.ResultadoUnidadTrabajoPropiaCronos, error) {
	var cero ports.ResultadoUnidadTrabajoPropiaCronos
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	m, err := domain.NuevoMaterialUnidadTrabajoCronos(o.Material.Actor(), o.Material.Fecha(), o.Material.ConocidoEn())
	if err != nil || !bytes.Equal(m.Canonico(), o.Material.Canonico()) || !application.AutorizacionUnidadTrabajoCronosValida(m, o.Autorizacion) {
		return cero, domain.ErrUnidadTrabajoCronosInvalida
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaUnidadTrabajoCronosSQL, m.Canonico(), o.Autorizacion, 8<<10, func(b []byte) (ports.ResultadoUnidadTrabajoPropiaCronos, error) {
		return decodificarUnidadTrabajoCronos(b, o)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return cero, err
		}
		if errors.Is(err, errRegistroEmpleadoB2Denegado) {
			return cero, domain.ErrUnidadTrabajoCronosDenegada
		}
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	return resultado, nil
}

func decodificarUnidadTrabajoCronos(b []byte, o ports.OrdenUnidadTrabajoCronos) (ports.ResultadoUnidadTrabajoPropiaCronos, error) {
	var cero ports.ResultadoUnidadTrabajoPropiaCronos
	if len(b) == 0 || verificarJSONOrganizacionHistorica(b) != nil {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	var raiz map[string]json.RawMessage
	if json.Unmarshal(b, &raiz) != nil || !clavesRegistroB2(raiz, []string{"unidad", "cobertura", "evidencia"}, nil) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	var e map[string]json.RawMessage
	if json.Unmarshal(raiz["evidencia"], &e) != nil || !clavesRegistroB2(e, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	if !bytes.Equal(bytes.TrimSpace(raiz["unidad"]), []byte("null")) {
		var u map[string]json.RawMessage
		if json.Unmarshal(raiz["unidad"], &u) != nil || !clavesRegistroB2(u, []string{"empleado_ref", "relacion_ref", "organismo_ref", "unidad_clave", "regimen_ref", "desde", "hasta_exclusivo", "fuente_ref", "fuente_version_ref", "version"}, nil) {
			return cero, domain.ErrUnidadTrabajoCronosNoDisponible
		}
	}
	var resultado ports.ResultadoUnidadTrabajoPropiaCronos
	if decodificarJSONRegistroB2(b, &resultado) != nil || !application.ResultadoUnidadTrabajoCronosValido(o.Material, o.Autorizacion, resultado) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	return resultado, nil
}

// El montaje debe ejecutar este preflight en la cuenta segregada de Personal.
func PreflightEjecutorUnidadTrabajoCronos(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `SELECT COALESCE(has_function_privilege(to_regprocedure('vec_personal.consultar_unidad_trabajo_propia_cronos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
 AND NOT has_table_privilege('vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 AND NOT has_table_privilege('vec_personal.recibo_unidad_trabajo_cronos','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`
	if pool == nil || preflightFichaPropia(ctx, pool, q) != nil {
		return domain.ErrUnidadTrabajoCronosNoDisponible
	}
	return nil
}

var _ ports.RepositorioUnidadTrabajoCronos = (*RepositorioUnidadTrabajoCronosPostgreSQL)(nil)
