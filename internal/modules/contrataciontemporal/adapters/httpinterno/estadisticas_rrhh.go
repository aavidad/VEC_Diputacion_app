package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Estadísticas de contratación por periodo (C18):
// GET /api/vec/contratacion-temporal/estadisticas?periodo=anual|mensual|semanal&desde=AAAA-MM-DD&hasta=AAAA-MM-DD
// Agregados sin datos personales sobre el mismo corte publicado que el cuadro;
// el alcance sale de la identidad acreditada del lector, nunca de la petición.
const (
	RutaEstadisticasRRHH    = "/api/vec/contratacion-temporal/estadisticas"
	EsquemaEstadisticasRRHH = "vec.ct.estadisticas.v1"
	zonaEstadisticasRRHH    = "Europe/Madrid"
	fechaEstadisticasRRHH   = "2006-01-02"
)

var (
	ErrManejadorEstadisticasRRHHInvalido = errors.New("contratacion temporal http: manejador de estadisticas RRHH invalido")
	errorAccesoEstadisticasRRHHDenegado  = nuevoErrorConsultaRRHH(http.StatusForbidden, "acceso_denegado")
)

// ResolutorAlcanceEstadisticasRRHH devuelve el ámbito del lector RRHH
// acreditado en el contexto de la petición.
type ResolutorAlcanceEstadisticasRRHH interface {
	AlcanceEstadisticasRRHH(context.Context) (ports.AlcanceEstadisticasRRHH, error)
}

type manejadorEstadisticasRRHH struct {
	consultor ports.ConsultorEstadisticasRRHH
	alcance   ResolutorAlcanceEstadisticasRRHH
	reloj     func() time.Time
}

func NuevoManejadorEstadisticasRRHH(consultor ports.ConsultorEstadisticasRRHH, alcance ResolutorAlcanceEstadisticasRRHH, reloj func() time.Time) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(consultor) || dependenciaConsultaRRHHNula(alcance) || reloj == nil {
		return nil, ErrManejadorEstadisticasRRHHInvalido
	}
	return &manejadorEstadisticasRRHH{consultor: consultor, alcance: alcance, reloj: reloj}, nil
}

type totalesEstadisticasRRHHJSON struct {
	Altas           uint64 `json:"altas"`
	Llamamientos    uint64 `json:"llamamientos"`
	Formalizaciones uint64 `json:"formalizaciones"`
	Cierres         uint64 `json:"cierres"`
	Incidencias     uint64 `json:"incidencias"`
}

type serieEstadisticasRRHHJSON struct {
	Inicio          string `json:"inicio"`
	Altas           uint64 `json:"altas"`
	Llamamientos    uint64 `json:"llamamientos"`
	Formalizaciones uint64 `json:"formalizaciones"`
	Cierres         uint64 `json:"cierres"`
	Incidencias     uint64 `json:"incidencias"`
}

type estadisticasRRHHJSON struct {
	Esquema     string                      `json:"esquema"`
	Periodo     string                      `json:"periodo"`
	Desde       string                      `json:"desde"`
	Hasta       string                      `json:"hasta"`
	ZonaHoraria string                      `json:"zona_horaria"`
	CorteGlobal uint64                      `json:"corte_global"`
	Series      []serieEstadisticasRRHHJSON `json:"series"`
	Totales     totalesEstadisticasRRHHJSON `json:"totales"`
}

type envoltorioEstadisticasRRHH struct {
	Data estadisticasRRHHJSON `json:"data"`
}

func (h *manejadorEstadisticasRRHH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaConsultaRRHHNula(h.consultor) || dependenciaConsultaRRHHNula(h.alcance) || h.reloj == nil {
		responderErrorConsultaRRHH(w, r, nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaEstadisticasRRHHExacta(r) {
		responderErrorConsultaRRHH(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderErrorConsultaRRHH(w, r, nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderErrorConsultaRRHH(w, r, nil, errorPeticionConsultaRRHHNoValida)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, r, err, clasificarErrorConsultaRRHH(err))
		return
	}
	consulta, ok := consultaEstadisticasRRHHDesdeQuery(r.URL.RawQuery, h.reloj())
	if !ok || consulta.Validar() != nil {
		responderErrorConsultaRRHH(w, r, nil, errorPeticionConsultaRRHHNoValida)
		return
	}
	alcance, err := h.alcance.AlcanceEstadisticasRRHH(r.Context())
	if err != nil || alcance.Validar() != nil {
		responderErrorConsultaRRHH(w, r, err, errorAccesoEstadisticasRRHHDenegado)
		return
	}
	resultado, err := h.consultor.ConsultarEstadisticasRRHH(r.Context(), alcance, consulta)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, r, errContexto, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if err != nil {
		if errors.Is(err, ports.ErrEstadisticasRRHHInvalida) {
			responderErrorConsultaRRHH(w, r, nil, errorPeticionConsultaRRHHNoValida)
			return
		}
		responderErrorConsultaRRHH(w, r, err, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if r.Method == http.MethodHead {
		aplicarCabecerasCobertura(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	responderJSONConsultaRRHH(w, r, http.StatusOK, envoltorioEstadisticasRRHH{Data: proyectarEstadisticasRRHH(consulta, resultado)})
}

func proyectarEstadisticasRRHH(consulta ports.ConsultaEstadisticasRRHH, resultado ports.EstadisticasRRHH) estadisticasRRHHJSON {
	series := make([]serieEstadisticasRRHHJSON, 0, len(resultado.Series))
	for _, serie := range resultado.Series {
		series = append(series, serieEstadisticasRRHHJSON{
			Inicio: serie.Inicio.Format(fechaEstadisticasRRHH), Altas: serie.Altas, Llamamientos: serie.Llamamientos,
			Formalizaciones: serie.Formalizaciones, Cierres: serie.Cierres, Incidencias: serie.Incidencias,
		})
	}
	totales := resultado.Totales()
	return estadisticasRRHHJSON{
		Esquema: EsquemaEstadisticasRRHH, Periodo: consulta.Periodo,
		Desde: consulta.Desde.Format(fechaEstadisticasRRHH), Hasta: consulta.Hasta.Format(fechaEstadisticasRRHH),
		ZonaHoraria: zonaEstadisticasRRHH, CorteGlobal: resultado.CorteGlobal, Series: series,
		Totales: totalesEstadisticasRRHHJSON{
			Altas: totales.Altas, Llamamientos: totales.Llamamientos, Formalizaciones: totales.Formalizaciones,
			Cierres: totales.Cierres, Incidencias: totales.Incidencias,
		},
	}
}

// consultaEstadisticasRRHHDesdeQuery admite solo los tres parámetros, cada
// uno como mucho una vez y sin vacíos. Sin parámetros: los últimos doce meses
// (mensual), doce semanas (semanal) o cinco años (anual) hasta hoy en Madrid.
func consultaEstadisticasRRHHDesdeQuery(rawQuery string, ahora time.Time) (ports.ConsultaEstadisticasRRHH, bool) {
	valores, err := url.ParseQuery(rawQuery)
	if err != nil || strings.Contains(rawQuery, ";") {
		return ports.ConsultaEstadisticasRRHH{}, false
	}
	for clave, lista := range valores {
		if (clave != "periodo" && clave != "desde" && clave != "hasta") || len(lista) != 1 || lista[0] == "" {
			return ports.ConsultaEstadisticasRRHH{}, false
		}
	}
	consulta := ports.ConsultaEstadisticasRRHH{Periodo: ports.PeriodoEstadisticasRRHHMensual}
	if periodo := valores.Get("periodo"); periodo != "" {
		consulta.Periodo = periodo
	}
	zona, err := time.LoadLocation(zonaEstadisticasRRHH)
	if err != nil {
		return ports.ConsultaEstadisticasRRHH{}, false
	}
	hoy := ahora.In(zona)
	hoy = time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, time.UTC)
	consulta.Hasta = hoy
	if valor := valores.Get("hasta"); valor != "" {
		if consulta.Hasta, err = ports.FechaEstadisticasRRHH(valor); err != nil {
			return ports.ConsultaEstadisticasRRHH{}, false
		}
	}
	switch consulta.Periodo {
	case ports.PeriodoEstadisticasRRHHAnual:
		consulta.Desde = time.Date(consulta.Hasta.Year()-4, time.January, 1, 0, 0, 0, 0, time.UTC)
	case ports.PeriodoEstadisticasRRHHMensual:
		consulta.Desde = time.Date(consulta.Hasta.Year(), consulta.Hasta.Month()-11, 1, 0, 0, 0, 0, time.UTC)
	case ports.PeriodoEstadisticasRRHHSemanal:
		consulta.Desde = consulta.Hasta.AddDate(0, 0, -7*11)
	default:
		return ports.ConsultaEstadisticasRRHH{}, false
	}
	if valor := valores.Get("desde"); valor != "" {
		if consulta.Desde, err = ports.FechaEstadisticasRRHH(valor); err != nil {
			return ports.ConsultaEstadisticasRRHH{}, false
		}
	}
	return consulta, true
}

func rutaEstadisticasRRHHExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" ||
		r.URL.RawFragment != "" || r.URL.Path != RutaEstadisticasRRHH {
		return false
	}
	return r.URL.EscapedPath() == RutaEstadisticasRRHH && !strings.Contains(r.URL.Path, "%")
}
