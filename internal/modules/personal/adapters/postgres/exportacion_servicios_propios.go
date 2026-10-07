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

const exportacionServiciosPropiosSQL = `SELECT vec_personal.exportar_servicios_propios_empleado_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioExportacionServiciosPropiosPostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioExportacionServiciosPropiosPostgreSQL(pool *pgxpool.Pool) (*RepositorioExportacionServiciosPropiosPostgreSQL, error) {
	return nuevoRepositorioExportacionServiciosPropios(pool)
}
func nuevoRepositorioExportacionServiciosPropios(pool iniciadorRegistroEmpleadoB2) (*RepositorioExportacionServiciosPropiosPostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return &RepositorioExportacionServiciosPropiosPostgreSQL{pool}, nil
}
func (r *RepositorioExportacionServiciosPropiosPostgreSQL) ExportarServiciosPropios(ctx context.Context, o ports.OrdenExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	var cero ports.ResultadoExportacionServiciosPropios
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	m, err := domain.NuevoMaterialExportacionServiciosPropios(o.Material.Solicitud(), o.Material.Formato())
	if err != nil || !bytes.Equal(m.Canonico(), o.Material.Canonico()) || !application.AutorizacionExportacionServiciosPropiosValida(m, o.Autorizacion) {
		return cero, domain.ErrExportacionServiciosPropiosInvalida
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, exportacionServiciosPropiosSQL, m.Canonico(), o.Autorizacion, maxRespuestaFichaPropia, func(b []byte) (ports.ResultadoExportacionServiciosPropios, error) {
		return decodificarExportacionServiciosPropios(b, o)
	})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return cero, err
		case errors.Is(err, errRegistroEmpleadoB2Denegado):
			return cero, domain.ErrExportacionServiciosPropiosDenegada
		default:
			return cero, domain.ErrExportacionServiciosPropiosNoDisponible
		}
	}
	return resultado, nil
}
func decodificarExportacionServiciosPropios(b []byte, o ports.OrdenExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	var cero ports.ResultadoExportacionServiciosPropios
	if verificarJSONOrganizacionHistorica(b) != nil {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	var top, corte, evidencia map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil || !clavesRegistroB2(top, []string{"corte", "servicios", "evidencia"}, nil) || json.Unmarshal(top["corte"], &corte) != nil || !clavesRegistroB2(corte, []string{"vigente_en", "conocido_en"}, nil) || json.Unmarshal(top["evidencia"], &evidencia) != nil || !clavesRegistroB2(evidencia, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) || !filasExactasFichaPropia(top["servicios"], []string{"inicio", "fin", "clase", "dias", "estado"}) {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	var filas []map[string]json.RawMessage
	if json.Unmarshal(top["servicios"], &filas) != nil {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	for _, fila := range filas {
		for _, v := range fila {
			if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return cero, domain.ErrExportacionServiciosPropiosNoDisponible
			}
		}
	}
	for _, obj := range []map[string]json.RawMessage{corte, evidencia} {
		for _, v := range obj {
			if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return cero, domain.ErrExportacionServiciosPropiosNoDisponible
			}
		}
	}
	var wire struct {
		Corte     domain.CorteEmpleadoB2            `json:"corte"`
		Servicios []domain.ServicioFichaPropia      `json:"servicios"`
		Evidencia ports.EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
	}
	if decodificarJSONRegistroB2(b, &wire) != nil || o.Material.ValidarServicios(wire.Corte, wire.Servicios) != nil || !evidenciaRegistroB2Valida(wire.Evidencia, o.Autorizacion) || wire.Evidencia.ReciboRef != o.Material.Solicitud().ReciboRef {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	contenido, huella, err := serializarServiciosPropiosCSV(o.Material.Formato(), wire.Servicios)
	if err != nil {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	r := ports.ResultadoExportacionServiciosPropios{Corte: wire.Corte, ContenidoCSV: contenido, ContenidoSHA256: huella, NombreArchivo: o.Material.Formato().Datos().NombreArchivo, Evidencia: wire.Evidencia}
	if !application.ResultadoExportacionServiciosPropiosValido(o.Material, o.Autorizacion, r) {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return r, nil
}

var _ ports.RepositorioExportacionServiciosPropios = (*RepositorioExportacionServiciosPropiosPostgreSQL)(nil)

// El montaje verifica la fachada exportadora propia antes de publicar la ruta.
func PreflightEjecutorExportacionServiciosPropios(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `SELECT has_function_privilege('vec_personal.exportar_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND (SELECT count(*)=2 FROM pg_catalog.pg_attribute WHERE attrelid='vec_personal.recibo_ficha_propia_empleado'::regclass AND NOT attisdropped AND NOT attnotnull AND ((attname='vigente_en' AND atttypid='date'::regtype) OR (attname='conocido_en' AND atttypid='timestamptz'::regtype AND atttypmod=6)))
 AND COALESCE((SELECT pg_catalog.strpos(prosrc,'consultada_en,vigente_en,conocido_en)')>0 FROM pg_catalog.pg_proc WHERE oid='vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),false)`
	if pool == nil || preflightFichaPropia(ctx, pool, q) != nil {
		return domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return nil
}
