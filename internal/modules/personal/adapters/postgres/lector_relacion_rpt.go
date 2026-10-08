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

const consultaLectorRelacionRPTSQL = `SELECT vec_personal.consultar_relacion_para_rpt_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const maxRespuestaLectorRelacionRPT = 16 << 10

type RepositorioLectorRelacionRPTPostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioLectorRelacionRPTPostgreSQL(pool *pgxpool.Pool) (*RepositorioLectorRelacionRPTPostgreSQL, error) {
	return nuevoRepositorioLectorRPT(pool)
}
func nuevoRepositorioLectorRPT(pool iniciadorRegistroEmpleadoB2) (*RepositorioLectorRelacionRPTPostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrLectorRelacionRPTNoDisponible
	}
	return &RepositorioLectorRelacionRPTPostgreSQL{pool}, nil
}
func (r *RepositorioLectorRelacionRPTPostgreSQL) ConsultarRelacionParaRPT(ctx context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
	var cero ports.ResultadoRelacionParaRPTV1
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrLectorRelacionRPTNoDisponible
	}
	m, err := domain.NuevoMaterialLectorRelacionRPT(o.Material.Solicitud())
	if err != nil || !bytes.Equal(m.Canonico(), o.Material.Canonico()) || !application.AutorizacionLectorRelacionRPTValida(m, o.Autorizacion) {
		return cero, domain.ErrLectorRelacionRPTInvalido
	}
	result, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaLectorRelacionRPTSQL, m.Canonico(), o.Autorizacion, maxRespuestaLectorRelacionRPT, func(b []byte) (ports.ResultadoRelacionParaRPTV1, error) { return decodificarLectorRelacionRPT(b, o) })
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return cero, err
		case errors.Is(err, errRegistroEmpleadoB2Denegado):
			return cero, domain.ErrLectorRelacionRPTDenegado
		default:
			return cero, domain.ErrLectorRelacionRPTNoDisponible
		}
	}
	return result, nil
}

// Wire sólo traduce el DTO nominal existente; no introduce otra ficha B2.
type relacionRPTWire struct {
	EmpleadoRef  string `json:"empleado_ref"`
	RelacionRef  string `json:"relacion_ref"`
	OrganismoRef string `json:"organismo_ref"`
	Version      int64  `json:"version"`
	Estado       string `json:"estado"`
	Periodo      struct {
		Desde domain.FechaCivil `json:"desde"`
		Hasta domain.FechaCivil `json:"hasta"`
	} `json:"periodo"`
	Procedencia struct {
		ActoRef       string                         `json:"acto_ref"`
		FuenteRef     string                         `json:"fuente_ref"`
		FuenteVersion string                         `json:"fuente_version"`
		Certeza       ports.CertezaPersonalNominalV1 `json:"certeza"`
	} `json:"procedencia"`
}
type resultadoRelacionRPTWire struct {
	Relacion  relacionRPTWire                   `json:"relacion"`
	Corte     domain.CorteEmpleadoB2            `json:"corte"`
	Cobertura ports.CoberturaPersonalNominalV1  `json:"cobertura"`
	Evidencia ports.EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

func decodificarLectorRelacionRPT(b []byte, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
	var cero ports.ResultadoRelacionParaRPTV1
	if verificarJSONOrganizacionHistorica(b) != nil {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	var root, relacion, corte, evidencia, periodo, procedencia map[string]json.RawMessage
	if json.Unmarshal(b, &root) != nil || !clavesRegistroB2(root, []string{"relacion", "corte", "cobertura", "evidencia"}, nil) ||
		json.Unmarshal(root["relacion"], &relacion) != nil || !clavesRegistroB2(relacion, []string{"empleado_ref", "relacion_ref", "organismo_ref", "version", "estado", "periodo", "procedencia"}, nil) ||
		json.Unmarshal(root["corte"], &corte) != nil || !clavesRegistroB2(corte, []string{"vigente_en", "conocido_en"}, nil) ||
		json.Unmarshal(root["evidencia"], &evidencia) != nil || !clavesRegistroB2(evidencia, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) ||
		json.Unmarshal(relacion["periodo"], &periodo) != nil || !clavesRegistroB2(periodo, []string{"desde", "hasta"}, nil) ||
		json.Unmarshal(relacion["procedencia"], &procedencia) != nil || !clavesRegistroB2(procedencia, []string{"acto_ref", "fuente_ref", "fuente_version", "certeza"}, nil) {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	// Hasta "" significa abierto. null u omisión no pueden obtener ese sentido.
	for _, obj := range []map[string]json.RawMessage{periodo, procedencia, corte, evidencia} {
		for _, value := range obj {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return cero, errRegistroEmpleadoB2NoDisponible
			}
		}
	}
	var w resultadoRelacionRPTWire
	if decodificarJSONRegistroB2(b, &w) != nil {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	v := w.Relacion
	r := ports.ResultadoRelacionParaRPTV1{
		Relacion: ports.RelacionParaRPTV1{EmpleadoRef: v.EmpleadoRef, RelacionRef: v.RelacionRef, OrganismoRef: v.OrganismoRef, Version: v.Version, Estado: v.Estado,
			Periodo:     ports.PeriodoPersonalNominalV1{Desde: v.Periodo.Desde, Hasta: v.Periodo.Hasta},
			Procedencia: ports.ProcedenciaPersonalNominalV1{ActoRef: v.Procedencia.ActoRef, FuenteRef: v.Procedencia.FuenteRef, FuenteVersion: v.Procedencia.FuenteVersion, Certeza: v.Procedencia.Certeza}},
		Corte: w.Corte, Cobertura: w.Cobertura, Evidencia: w.Evidencia,
	}
	if !application.ResultadoLectorRelacionRPTValido(o.Material, o.Autorizacion, r) || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

var _ ports.RepositorioLectorRelacionRPT = (*RepositorioLectorRelacionRPTPostgreSQL)(nil)
