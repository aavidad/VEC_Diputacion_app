package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const consultaProgramacionLaboralC4 = `SELECT vec_cronos_v1.consultar_programacion_laboral_c4_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioProgramacionLaboralC4 struct{ db iniciadorMarcaje }

func NuevoRepositorioProgramacionLaboralC4(pool *pgxpool.Pool) (*RepositorioProgramacionLaboralC4, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioProgramacionLaboralC4{db: pool}, nil
}

func PreflightProgramacionLaboralC4(ctx context.Context, pool *pgxpool.Pool) error {
	if ctx == nil || pool == nil {
		return ports.ErrDependenciaNoDisponible
	}
	var valida bool
	err := pool.QueryRow(ctx, `SELECT
	 COALESCE(has_function_privilege(to_regprocedure('vec_cronos_v1.consultar_programacion_laboral_c4_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE'),false)
	 AND NOT has_table_privilege('vec_cronos_v1.programacion_jornada','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
	 AND NOT has_table_privilege('vec_cronos_v1.programacion_contexto_c4','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`).Scan(&valida)
	if err != nil || !valida {
		return ports.ErrDependenciaNoDisponible
	}
	return nil
}

func (r *RepositorioProgramacionLaboralC4) ConsultarProgramacionC4(ctx context.Context, orden ports.OrdenConsultaSaldo, fecha string) (*application.ProgramacionLaboralC4, error) {
	if r == nil || r.db == nil || ctx == nil || !fechaCronosC4Canonica(fecha) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return nil, ports.ErrDependenciaNoDisponible
	}
	material := domain.MaterialConsultaSaldoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		EmpleadoRef: empleados[0], Desde: fecha, Hasta: fecha, ZonaHoraria: "Europe/Madrid"}
	canonico, err := material.Canonico()
	if err != nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaSaldoPropio(material)
	if err != nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialConsultaSaldoPropio(ctx, material)
	if err != nil {
		return nil, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaSaldoPropio, application.AccionConsultarSaldoPropio, recurso) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarLecturaV3(ctx, r.db, consultaProgramacionLaboralC4, canonico, v3)
	if err != nil {
		return nil, err
	}
	defer clear(bruto)
	return decodificarProgramacionC4(bruto, empleados[0], fecha)
}

func fechaCronosC4Canonica(s string) bool {
	fecha, err := time.Parse("2006-01-02", s)
	return err == nil && fecha.Format("2006-01-02") == s && fecha.Year() >= 1900 && fecha.Year() <= 2100
}

func decodificarProgramacionC4(bruto []byte, empleado, fecha string) (*application.ProgramacionLaboralC4, error) {
	if len(bruto) == 0 || len(bruto) > 16<<10 {
		return nil, ports.ErrDependenciaNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	var x struct {
		Estado             string    `json:"estado"`
		Fecha              string    `json:"fecha"`
		EmpleadoRef        string    `json:"empleado_ref"`
		RelacionRef        string    `json:"relacion_ref"`
		CentroRef          string    `json:"centro_ref"`
		RegimenRef         string    `json:"regimen_ref"`
		ProgramacionRef    string    `json:"programacion_ref"`
		TurnoRef           string    `json:"turno_ref"`
		PoliticaVersionRef string    `json:"politica_version_ref"`
		FuenteRef          string    `json:"fuente_ref"`
		Version            int64     `json:"version"`
		FuenteVersion      int64     `json:"fuente_version"`
		InicioUTC          time.Time `json:"inicio_utc"`
		FinUTC             time.Time `json:"fin_utc"`
		PublicadaEn        time.Time `json:"publicada_en"`
		MinutosProgramados int64     `json:"minutos_programados"`
	}
	if dec.Decode(&x) != nil || !errors.Is(dec.Decode(new(any)), io.EOF) || x.Fecha != fecha {
		return nil, ports.ErrDependenciaNoDisponible
	}
	if x.Estado == "no_determinable" {
		var forma map[string]json.RawMessage
		if json.Unmarshal(bruto, &forma) != nil || len(forma) != 2 || forma["estado"] == nil || forma["fecha"] == nil {
			return nil, ports.ErrDependenciaNoDisponible
		}
		return nil, nil
	}
	if x.Estado != "programado" || x.EmpleadoRef != empleado || x.RelacionRef == "" || x.CentroRef == "" || x.RegimenRef == "" || x.ProgramacionRef == "" || x.TurnoRef == "" || x.PoliticaVersionRef == "" || x.FuenteRef == "" || x.Version < 1 || x.FuenteVersion < 1 || x.MinutosProgramados < 0 || x.MinutosProgramados > 1440 || x.InicioUTC.IsZero() || x.FinUTC.IsZero() || x.PublicadaEn.IsZero() {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &application.ProgramacionLaboralC4{Fecha: x.Fecha, EmpleadoRef: x.EmpleadoRef, RelacionRef: x.RelacionRef, CentroRef: x.CentroRef, RegimenRef: x.RegimenRef,
		ProgramacionRef: x.ProgramacionRef, TurnoRef: x.TurnoRef, PoliticaVersionRef: x.PoliticaVersionRef, FuenteRef: x.FuenteRef,
		Version: x.Version, FuenteVersion: x.FuenteVersion, InicioUTC: x.InicioUTC.UTC(), FinUTC: x.FinUTC.UTC(), PublicadaEn: x.PublicadaEn.UTC(),
		MinutosProgramados: x.MinutosProgramados}, nil
}

var _ application.FuenteProgramacionLaboralC4 = (*RepositorioProgramacionLaboralC4)(nil)
