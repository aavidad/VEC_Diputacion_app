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

const consultaLectorServiciosCertificadosSQL = `SELECT vec_personal.consultar_servicios_certificados_propios_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

// 200 servicios con referencias largas caben holgadamente; un exceso se trata
// como fuente no disponible, nunca como lista truncada.
const maxRespuestaLectorServiciosCertificados = 512 << 10

type RepositorioLectorServiciosCertificadosPostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioLectorServiciosCertificadosPostgreSQL(pool *pgxpool.Pool) (*RepositorioLectorServiciosCertificadosPostgreSQL, error) {
	return nuevoRepositorioLectorServiciosCertificados(pool)
}
func nuevoRepositorioLectorServiciosCertificados(pool iniciadorRegistroEmpleadoB2) (*RepositorioLectorServiciosCertificadosPostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrLectorServiciosCertificadosNoDisponible
	}
	return &RepositorioLectorServiciosCertificadosPostgreSQL{pool}, nil
}

func (r *RepositorioLectorServiciosCertificadosPostgreSQL) ConsultarServiciosParaCertificados(ctx context.Context, o ports.OrdenLectorServiciosCertificados) (ports.ResultadoServiciosParaCertificadosV2, error) {
	var cero ports.ResultadoServiciosParaCertificadosV2
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrLectorServiciosCertificadosNoDisponible
	}
	m, err := domain.NuevoMaterialLectorServiciosCertificados(o.Material.Solicitud())
	if err != nil || !bytes.Equal(m.Canonico(), o.Material.Canonico()) || !application.AutorizacionLectorServiciosCertificadosValida(m, o.Autorizacion) {
		return cero, domain.ErrLectorServiciosCertificadosInvalido
	}
	var errorRespuesta error
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaLectorServiciosCertificadosSQL, m.Canonico(), o.Autorizacion, maxRespuestaLectorServiciosCertificados,
		func(b []byte) (ports.ResultadoServiciosParaCertificadosV2, error) {
			v, e := decodificarLectorServiciosCertificados(b, o)
			errorRespuesta = e
			return v, e
		})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return cero, err
		case errors.Is(err, errRegistroEmpleadoB2Denegado):
			return cero, domain.ErrLectorServiciosCertificadosDenegado
		case errors.Is(err, errRegistroEmpleadoB2ExcedeLimite), errors.Is(errorRespuesta, domain.ErrLectorServiciosCertificadosExcedeLimite):
			return cero, domain.ErrLectorServiciosCertificadosExcedeLimite
		default:
			return cero, domain.ErrLectorServiciosCertificadosNoDisponible
		}
	}
	return resultado, nil
}

// Wire traduce la respuesta de Personal36; no añade campos ni reglas.
type servicioCertificadosWire struct {
	ServicioRef string `json:"servicio_ref"`
	RelacionRef string `json:"relacion_ref"`
	Version     int64  `json:"version"`
	Periodo     struct {
		Desde domain.FechaCivil `json:"desde"`
		Hasta domain.FechaCivil `json:"hasta"`
	} `json:"periodo"`
	DiasReconocidos int64  `json:"dias_reconocidos"`
	Estado          string `json:"estado"`
	ClaseRef        string `json:"clase_ref"`
	ClaseVersion    int64  `json:"clase_version"`
	Procedencia     struct {
		ActoRef       string                         `json:"acto_ref"`
		FuenteRef     string                         `json:"fuente_ref"`
		FuenteVersion string                         `json:"fuente_version"`
		Certeza       ports.CertezaPersonalNominalV1 `json:"certeza"`
	} `json:"procedencia"`
}
type resultadoServiciosCertificadosWire struct {
	Servicios struct {
		EmpleadoRef  string                           `json:"empleado_ref"`
		OrganismoRef string                           `json:"organismo_ref"`
		Version      int64                            `json:"version"`
		Corte        domain.CorteEmpleadoB2           `json:"corte"`
		Cobertura    ports.CoberturaPersonalNominalV1 `json:"cobertura"`
		Servicios    []servicioCertificadosWire       `json:"servicios"`
	} `json:"servicios"`
	Evidencia ports.EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

func decodificarLectorServiciosCertificados(b []byte, o ports.OrdenLectorServiciosCertificados) (ports.ResultadoServiciosParaCertificadosV2, error) {
	var cero ports.ResultadoServiciosParaCertificadosV2
	if verificarJSONOrganizacionHistorica(b) != nil {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	// Claves exactas en cada nivel y ningún null: una omisión no adquiere sentido.
	objeto := func(raw json.RawMessage, obligatorias []string) (map[string]json.RawMessage, bool) {
		var m map[string]json.RawMessage
		if json.Unmarshal(raw, &m) != nil || !clavesRegistroB2(m, obligatorias, nil) {
			return nil, false
		}
		for _, v := range m {
			if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return nil, false
			}
		}
		return m, true
	}
	raiz, ok := objeto(b, []string{"servicios", "evidencia"})
	if !ok {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	cuerpo, ok := objeto(raiz["servicios"], []string{"empleado_ref", "organismo_ref", "version", "corte", "cobertura", "servicios"})
	if !ok {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	if _, ok = objeto(cuerpo["corte"], []string{"vigente_en", "conocido_en"}); !ok {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	if _, ok = objeto(raiz["evidencia"], []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}); !ok {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	var filas []json.RawMessage
	if len(cuerpo["servicios"]) == 0 || cuerpo["servicios"][0] != '[' || json.Unmarshal(cuerpo["servicios"], &filas) != nil {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	if len(filas) > domain.LimiteLectorServiciosCertificados {
		return cero, domain.ErrLectorServiciosCertificadosExcedeLimite
	}
	for _, fila := range filas {
		f, ok := objeto(fila, []string{"servicio_ref", "relacion_ref", "version", "periodo", "dias_reconocidos", "estado", "clase_ref", "clase_version", "procedencia"})
		if !ok {
			return cero, errRegistroEmpleadoB2NoDisponible
		}
		if _, ok = objeto(f["periodo"], []string{"desde", "hasta"}); !ok {
			return cero, errRegistroEmpleadoB2NoDisponible
		}
		if _, ok = objeto(f["procedencia"], []string{"acto_ref", "fuente_ref", "fuente_version", "certeza"}); !ok {
			return cero, errRegistroEmpleadoB2NoDisponible
		}
	}
	var w resultadoServiciosCertificadosWire
	if decodificarJSONRegistroB2(b, &w) != nil {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	r := ports.ResultadoServiciosParaCertificadosV2{EmpleadoRef: w.Servicios.EmpleadoRef, OrganismoRef: w.Servicios.OrganismoRef, Version: w.Servicios.Version,
		Corte: w.Servicios.Corte, Cobertura: w.Servicios.Cobertura, Evidencia: w.Evidencia, Servicios: make([]ports.ServicioParaCertificadosV2, 0, len(w.Servicios.Servicios))}
	for _, s := range w.Servicios.Servicios {
		r.Servicios = append(r.Servicios, ports.ServicioParaCertificadosV2{DiasReconocidos: s.DiasReconocidos, ServicioParaCertificadosV1: ports.ServicioParaCertificadosV1{
			ServicioRef: s.ServicioRef, RelacionRef: s.RelacionRef, Version: s.Version, Estado: s.Estado, ClaseRef: s.ClaseRef, ClaseVersion: s.ClaseVersion,
			Periodo:     ports.PeriodoPersonalNominalV1{Desde: s.Periodo.Desde, Hasta: s.Periodo.Hasta},
			Procedencia: ports.ProcedenciaPersonalNominalV1{ActoRef: s.Procedencia.ActoRef, FuenteRef: s.Procedencia.FuenteRef, FuenteVersion: s.Procedencia.FuenteVersion, Certeza: s.Procedencia.Certeza}}})
	}
	if !application.ResultadoLectorServiciosCertificadosValido(o.Material, o.Autorizacion, r) || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

// PreflightEjecutorLectorServiciosCertificados comprueba que la cuenta sólo
// ejecuta la fachada nominal, sin lectura ni DML directo sobre la historia.
func PreflightEjecutorLectorServiciosCertificados(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `SELECT COALESCE(has_function_privilege(to_regprocedure('vec_personal.consultar_servicios_certificados_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
 AND NOT has_table_privilege('vec_personal.servicio_reconocido_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 AND NOT has_table_privilege('vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`
	if pool == nil || preflightFichaPropia(ctx, pool, q) != nil {
		return domain.ErrLectorServiciosCertificadosNoDisponible
	}
	return nil
}

var _ ports.RepositorioLectorServiciosCertificados = (*RepositorioLectorServiciosCertificadosPostgreSQL)(nil)
