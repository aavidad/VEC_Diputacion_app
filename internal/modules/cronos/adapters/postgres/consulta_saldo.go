package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// RepositorioConsultaSaldo is deliberately unavailable until a nominal
// authorization consumer can audit and read Cronos in one durable boundary.
// The internal SQL projection has no EXECUTE grant for the application role.
type RepositorioConsultaSaldo struct{}

func NuevoRepositorioConsultaSaldo(_ *pgxpool.Pool) (*RepositorioConsultaSaldo, error) {
	return nil, ports.ErrDependenciaNoDisponible
}

func (*RepositorioConsultaSaldo) ConsultarFuenteSaldo(context.Context, ports.OrdenConsultaSaldo, string, string, string, string) (ports.FuenteSaldo, error) {
	return ports.FuenteSaldo{}, ports.ErrDependenciaNoDisponible
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
		f.Marcajes = append(f.Marcajes, ports.MarcajeSaldo{MarcajeRef: m.MarcajeRef, Movimiento: domain.PunchKind(m.Movimiento), InstanteUTC: m.InstanteUTC, Canal: ports.CanalSaldo{PoliticaVersionRef: m.Canal.PoliticaVersionRef, CanalRef: m.Canal.CanalRef, OrigenRef: m.Canal.OrigenRef, CalidadRef: m.Canal.CalidadRef}, OrigenRef: m.Canal.OrigenRef, TipoOrigen: m.TipoOrigen})
	}
	for _, m := range sql.MovimientosSaldo {
		f.MovimientosSaldo = append(f.MovimientosSaldo, ports.MovimientoSaldo{Fecha: m.Fecha, Tipo: m.Tipo, DeltaMicrosegundos: m.DeltaMicrosegundos, Fuentes: m.Fuentes})
	}
	return f, nil
}
