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

const consultaHistoriaRelacionesPropiaSQL = `SELECT vec_personal.consultar_historia_relaciones_propias_empleado_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioHistoriaRelacionesPropiaPostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioHistoriaRelacionesPropiaPostgreSQL(pool *pgxpool.Pool) (*RepositorioHistoriaRelacionesPropiaPostgreSQL, error) {
	return nuevoRepositorioHistoriaRelacionesPropia(pool)
}
func nuevoRepositorioHistoriaRelacionesPropia(pool iniciadorRegistroEmpleadoB2) (*RepositorioHistoriaRelacionesPropiaPostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return &RepositorioHistoriaRelacionesPropiaPostgreSQL{pool}, nil
}
func (r *RepositorioHistoriaRelacionesPropiaPostgreSQL) ConsultarHistoriaRelacionesPropia(ctx context.Context, o ports.OrdenHistoriaRelacionesPropia) (ports.ResultadoHistoriaRelacionesPropia, error) {
	var cero ports.ResultadoHistoriaRelacionesPropia
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	m, err := domain.NuevoMaterialHistoriaRelacionesPropia(domain.SolicitudHistoriaRelacionesPropia{Actor: o.Material.Actor(), Corte: o.Material.Corte()})
	if err != nil || !bytes.Equal(m.Canonico(), o.Material.Canonico()) || !application.AutorizacionHistoriaRelacionesPropiaValida(m, o.Autorizacion) {
		return cero, domain.ErrHistoriaRelacionesPropiaInvalida
	}
	// El helper consume los once argumentos nominales; no lee la ficha RRHH.
	var errorRespuesta error
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaHistoriaRelacionesPropiaSQL, m.Canonico(), o.Autorizacion, maxRespuestaFichaEmpleadoB2, func(b []byte) (ports.ResultadoHistoriaRelacionesPropia, error) {
		resultado, e := decodificarHistoriaRelacionesPropia(b, o)
		errorRespuesta = e
		return resultado, e
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return cero, err
		}
		if errors.Is(err, errRegistroEmpleadoB2Denegado) {
			return cero, domain.ErrHistoriaRelacionesPropiaDenegada
		}
		if errors.Is(err, errRegistroEmpleadoB2ExcedeLimite) || errors.Is(errorRespuesta, domain.ErrHistoriaRelacionesPropiaExcedeLimite) {
			return cero, domain.ErrHistoriaRelacionesPropiaExcedeLimite
		}
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return resultado, nil
}
func decodificarHistoriaRelacionesPropia(b []byte, o ports.OrdenHistoriaRelacionesPropia) (ports.ResultadoHistoriaRelacionesPropia, error) {
	var cero ports.ResultadoHistoriaRelacionesPropia
	if verificarJSONOrganizacionHistorica(b) != nil {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	objeto := func(b []byte, obligatorias, opcionales []string) (map[string]json.RawMessage, bool) {
		var m map[string]json.RawMessage
		if json.Unmarshal(b, &m) != nil || !clavesRegistroB2(m, obligatorias, opcionales) {
			return nil, false
		}
		for _, v := range m {
			if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return nil, false
			}
		}
		return m, true
	}
	top, ok := objeto(b, []string{"historia", "evidencia"}, nil)
	if !ok {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	h, ok := objeto(top["historia"], []string{"empleado_ref", "corte", "cobertura", "revisiones"}, nil)
	if !ok {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	if _, ok = objeto(h["corte"], []string{"efectos_desde", "efectos_hasta", "conocido_en"}, nil); !ok {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	if _, ok = objeto(top["evidencia"], []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil); !ok {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	var filas []json.RawMessage
	if len(h["revisiones"]) == 0 || h["revisiones"][0] != '[' || json.Unmarshal(h["revisiones"], &filas) != nil {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	if len(filas) > domain.LimiteHistoriaRelacionesPropia {
		return cero, domain.ErrHistoriaRelacionesPropiaExcedeLimite
	}
	for _, fila := range filas {
		f, ok := objeto(fila, []string{"relacion_ref", "estado", "regimen", "modalidad", "unidad", "puesto", "situacion", "traza"}, nil)
		if !ok {
			return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
		}
		if _, ok = objeto(f["traza"], []string{"desde", "registrada_en", "version", "acto_ref", "fuente_ref", "fuente_version"}, []string{"hasta"}); !ok {
			return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
		}
	}
	var resultado ports.ResultadoHistoriaRelacionesPropia
	if decodificarJSONRegistroB2(b, &resultado) != nil || !evidenciaRegistroB2Valida(resultado.Evidencia, o.Autorizacion) || !application.ResultadoHistoriaRelacionesPropiaValido(o.Material, o.Autorizacion, resultado) {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return resultado, nil
}

// Únicamente habilita la fachada nominal. La cuenta debe ser la segregada de
// ficha propia ya validada por composición, sin lectura ni DML directo.
func PreflightEjecutorHistoriaRelacionesPropia(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `SELECT COALESCE(has_function_privilege(to_regprocedure('vec_personal.consultar_historia_relaciones_propias_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
 AND NOT has_table_privilege('vec_personal.servicio_reconocido_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 AND NOT has_table_privilege('vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`
	if pool == nil || preflightFichaPropia(ctx, pool, q) != nil {
		return domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return nil
}

var _ ports.RepositorioHistoriaRelacionesPropia = (*RepositorioHistoriaRelacionesPropiaPostgreSQL)(nil)
