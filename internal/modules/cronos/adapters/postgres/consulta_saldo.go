package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// RepositorioConsultaSaldo lee el libro de saldo de la persona con un LOGIN
// que sólo hereda vec_cronos_v1_ejecutor. Cada lectura obtiene una decisión
// V3 nueva (AD3-53) que cronos_v1 000007 consume, audita y usa para fijar la
// RLS en la misma transacción que lee.
type RepositorioConsultaSaldo struct {
	db iniciadorMarcaje
}

func NuevoRepositorioConsultaSaldo(pool *pgxpool.Pool) (*RepositorioConsultaSaldo, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioConsultaSaldo{db: pool}, nil
}

func (r *RepositorioConsultaSaldo) ConsultarFuenteSaldo(ctx context.Context, orden ports.OrdenConsultaSaldo, empleado, desde, hasta, zona string) (ports.FuenteSaldo, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != empleado {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	material := domain.MaterialConsultaSaldoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Desde: desde, Hasta: hasta, ZonaHoraria: zona}
	canonico, err := material.Canonico()
	if err != nil {
		return ports.FuenteSaldo{}, ports.ErrConsultaSaldoInvalida
	}
	recurso, err := application.RecursoConsultaSaldoPropio(material)
	if err != nil {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialConsultaSaldoPropio(ctx, material)
	if err != nil {
		return ports.FuenteSaldo{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaSaldoPropio, application.AccionConsultarSaldoPropio, recurso) {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarLecturaV3(ctx, r.db, consultaSaldoPropio, canonico, v3)
	if err != nil {
		return ports.FuenteSaldo{}, err
	}
	defer clear(bruto)
	return decodificarFuenteSaldoInterna(bruto)
}

var _ ports.RepositorioConsultaSaldo = (*RepositorioConsultaSaldo)(nil)

// decodificarFuenteSaldoInterna prepares the SQL projection for a future
// authorized reader. It does not itself grant or perform a database read.
func decodificarFuenteSaldoInterna(bruto []byte) (ports.FuenteSaldo, error) {
	if len(bruto) == 0 || len(bruto) > 4*1024*1024 {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	var sql struct {
		EmpleadoRef string `json:"empleado_ref"`
		Desde       string `json:"desde"`
		Hasta       string `json:"hasta"`
		ZonaHoraria string `json:"zona_horaria"`
		Completo    *bool  `json:"completo"`
		Jornadas    []struct {
			ProgramacionRef    string `json:"programacion_ref"`
			Fecha              string `json:"fecha"`
			TurnoRef           string `json:"turno_ref"`
			PoliticaVersionRef string `json:"politica_version_ref"`
			FuenteRef          string `json:"fuente_ref"`
			ZonaHoraria        string `json:"zona_horaria"`
			MinutosPrevistos   int64  `json:"minutos_previstos"`
			Version            int64  `json:"version"`
		} `json:"jornadas"`
		Marcajes []struct {
			MarcajeRef  string    `json:"marcaje_ref"`
			Movimiento  string    `json:"movimiento"`
			InstanteUTC time.Time `json:"instante_utc"`
			Canal       struct {
				PoliticaVersionRef string `json:"politica_version_ref"`
				CanalRef           string `json:"canal_ref"`
				OrigenRef          string `json:"origen_ref"`
				CalidadRef         string `json:"calidad_ref"`
			} `json:"canal"`
			TipoOrigen *string `json:"tipo_origen"`
		} `json:"marcajes"`
		MovimientosSaldo []struct {
			Fecha              string   `json:"fecha"`
			Tipo               string   `json:"tipo"`
			DeltaMicrosegundos int64    `json:"delta_microsegundos"`
			Fuentes            []string `json:"fuentes"`
		} `json:"movimientos_saldo"`
	}
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	if dec.Decode(&sql) != nil || dec.Decode(&struct{}{}) != io.EOF || sql.Completo == nil || sql.EmpleadoRef == "" || sql.ZonaHoraria == "" {
		return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
	}
	f := ports.FuenteSaldo{EmpleadoRef: sql.EmpleadoRef, Desde: sql.Desde, Hasta: sql.Hasta, ZonaHoraria: sql.ZonaHoraria, Completo: *sql.Completo, Jornadas: make([]ports.JornadaPrevista, 0, len(sql.Jornadas)), Marcajes: make([]ports.MarcajeSaldo, 0, len(sql.Marcajes)), MovimientosSaldo: make([]ports.MovimientoSaldo, 0, len(sql.MovimientosSaldo))}
	for _, j := range sql.Jornadas {
		if j.ProgramacionRef == "" || j.FuenteRef == "" || j.Version < 1 || j.ZonaHoraria != sql.ZonaHoraria {
			return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
		}
		f.Jornadas = append(f.Jornadas, ports.JornadaPrevista{Fecha: j.Fecha, TurnoRef: j.TurnoRef, PoliticaVersionRef: j.PoliticaVersionRef, MinutosPrevistos: j.MinutosPrevistos})
	}
	for _, m := range sql.Marcajes {
		f.Marcajes = append(f.Marcajes, ports.MarcajeSaldo{MarcajeRef: m.MarcajeRef, Movimiento: domain.PunchKind(m.Movimiento), InstanteUTC: m.InstanteUTC.UTC(), Canal: ports.CanalSaldo{PoliticaVersionRef: m.Canal.PoliticaVersionRef, CanalRef: m.Canal.CanalRef, OrigenRef: m.Canal.OrigenRef, CalidadRef: m.Canal.CalidadRef}, OrigenRef: m.Canal.OrigenRef, TipoOrigen: m.TipoOrigen})
	}
	for _, m := range sql.MovimientosSaldo {
		f.MovimientosSaldo = append(f.MovimientosSaldo, ports.MovimientoSaldo{Fecha: m.Fecha, Tipo: m.Tipo, DeltaMicrosegundos: m.DeltaMicrosegundos, Fuentes: m.Fuentes})
	}
	return f, nil
}
