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

const consultaHistoriaServiciosPropiaSQL = `SELECT vec_personal.consultar_historia_servicios_propios_empleado_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioHistoriaServiciosPropiaPostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioHistoriaServiciosPropiaPostgreSQL(pool *pgxpool.Pool) (*RepositorioHistoriaServiciosPropiaPostgreSQL, error) {
	return nuevoRepositorioHistoriaServiciosPropia(pool)
}
func nuevoRepositorioHistoriaServiciosPropia(pool iniciadorRegistroEmpleadoB2) (*RepositorioHistoriaServiciosPropiaPostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return &RepositorioHistoriaServiciosPropiaPostgreSQL{pool}, nil
}
func (r *RepositorioHistoriaServiciosPropiaPostgreSQL) ConsultarHistoriaServiciosPropia(ctx context.Context, o ports.OrdenHistoriaServiciosPropia) (ports.ResultadoHistoriaServiciosPropia, error) {
	var cero ports.ResultadoHistoriaServiciosPropia
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	m, err := domain.NuevoMaterialHistoriaServiciosPropia(domain.SolicitudHistoriaServiciosPropia{Actor: o.Material.Actor(), Corte: o.Material.Corte()})
	if err != nil || !bytes.Equal(m.Canonico(), o.Material.Canonico()) || !application.AutorizacionHistoriaServiciosPropiaValida(m, o.Autorizacion) {
		return cero, domain.ErrHistoriaServiciosPropiaInvalida
	}
	// El helper consume los once argumentos nominales; no lee la ficha RRHH.
	var errorRespuesta error
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaHistoriaServiciosPropiaSQL, m.Canonico(), o.Autorizacion, maxRespuestaFichaEmpleadoB2, func(b []byte) (ports.ResultadoHistoriaServiciosPropia, error) {
		resultado, e := decodificarHistoriaServiciosPropia(b, o)
		errorRespuesta = e
		return resultado, e
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return cero, err
		}
		if errors.Is(err, errRegistroEmpleadoB2Denegado) {
			return cero, domain.ErrHistoriaServiciosPropiaDenegada
		}
		if errors.Is(err, errRegistroEmpleadoB2ExcedeLimite) || errors.Is(errorRespuesta, domain.ErrHistoriaServiciosPropiaExcedeLimite) {
			return cero, domain.ErrHistoriaServiciosPropiaExcedeLimite
		}
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return resultado, nil
}
func decodificarHistoriaServiciosPropia(b []byte, o ports.OrdenHistoriaServiciosPropia) (ports.ResultadoHistoriaServiciosPropia, error) {
	var cero ports.ResultadoHistoriaServiciosPropia
	if verificarJSONOrganizacionHistorica(b) != nil {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
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
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	h, ok := objeto(top["historia"], []string{"empleado_ref", "corte", "cobertura", "revisiones"}, nil)
	if !ok {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	if _, ok = objeto(h["corte"], []string{"efectos_desde", "efectos_hasta", "conocido_en"}, nil); !ok {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	if _, ok = objeto(top["evidencia"], []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil); !ok {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	var filas []json.RawMessage
	if len(h["revisiones"]) == 0 || h["revisiones"][0] != '[' || json.Unmarshal(h["revisiones"], &filas) != nil {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	if len(filas) > domain.LimiteHistoriaServiciosPropia {
		return cero, domain.ErrHistoriaServiciosPropiaExcedeLimite
	}
	for _, fila := range filas {
		f, ok := objeto(fila, []string{"servicio_ref", "relacion_ref", "periodo_desde", "periodo_hasta", "dias_reconocidos", "estado", "clase", "traza"}, nil)
		if !ok {
			return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
		}
		if _, ok = objeto(f["traza"], []string{"desde", "registrada_en", "version", "acto_ref", "fuente_ref", "fuente_version"}, []string{"hasta"}); !ok {
			return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
		}
	}
	var resultado ports.ResultadoHistoriaServiciosPropia
	if decodificarJSONRegistroB2(b, &resultado) != nil || !evidenciaRegistroB2Valida(resultado.Evidencia, o.Autorizacion) || !application.ResultadoHistoriaServiciosPropiaValido(o.Material, o.Autorizacion, resultado) {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return resultado, nil
}

// Únicamente habilita la fachada nominal. La cuenta debe ser la segregada de
// ficha propia ya validada por composición, sin lectura ni DML directo.
func PreflightEjecutorHistoriaServiciosPropia(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `SELECT COALESCE(has_function_privilege(to_regprocedure('vec_personal.consultar_historia_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
 AND NOT has_table_privilege('vec_personal.servicio_reconocido_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 AND NOT has_table_privilege('vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`
	if pool == nil || preflightFichaPropia(ctx, pool, q) != nil {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return nil
}

var _ ports.RepositorioHistoriaServiciosPropia = (*RepositorioHistoriaServiciosPropiaPostgreSQL)(nil)
