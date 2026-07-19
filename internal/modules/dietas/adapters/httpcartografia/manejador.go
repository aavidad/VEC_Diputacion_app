// Package httpcartografia contiene el unico adaptador HTTP del calculo de
// rutas. La API VEC autorizada y la superficie aislada de presentacion lo
// componen con envolventes distintas, pero comparten decodificacion, limites y
// proyeccion de respuesta.
package httpcartografia

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const (
	RutaPresentacion = "/api/presentacion/cartografia/rutas"
	maximoCuerpo     = int64(1 << 20)
)

type OpcionesManejador struct {
	// EnvolverEnDatos conserva el contrato historico de /api/vec. La
	// superficie de presentacion entrega el mismo DTO directamente.
	EnvolverEnDatos bool
}

type Manejador struct {
	calculador      dietasapp.CasoUsoCalculoRutas
	envolverEnDatos bool
}

type solicitudHTTP struct {
	Coordenadas  []coordenadaHTTP `json:"coordinates"`
	Alternativas int              `json:"alternatives"`
}

type coordenadaHTTP struct {
	Latitud  float64 `json:"lat"`
	Longitud float64 `json:"lon"`
	Nombre   string  `json:"name"`
}

type respuestaHTTP struct {
	Codigo       string     `json:"code"`
	Rutas        []rutaHTTP `json:"routes"`
	VersionGrafo string     `json:"data_version"`
	Motor        string     `json:"engine"`
	Ambito       string     `json:"route_scope"`
}

type rutaHTTP struct {
	Distancia float64       `json:"distance"`
	Duracion  float64       `json:"duration"`
	Tramos    []tramoHTTP   `json:"legs"`
	Geometria geometriaHTTP `json:"geometry"`
}

type tramoHTTP struct {
	Distancia float64 `json:"distance"`
	Duracion  float64 `json:"duration"`
}

type geometriaHTTP struct {
	Tipo        string      `json:"type"`
	Coordenadas [][]float64 `json:"coordinates"`
}

func NuevoManejador(calculador dietasapp.CasoUsoCalculoRutas, opciones OpcionesManejador) (*Manejador, error) {
	if calculador == nil {
		return nil, errors.New("http cartografia: calculador requerido")
	}
	return &Manejador{calculador: calculador, envolverEnDatos: opciones.EnvolverEnDatos}, nil
}

func (m *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararRespuestaJSON(w)
	if r == nil || r.URL == nil {
		escribirError(w, http.StatusBadRequest, "peticion de ruta invalida")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		escribirError(w, http.StatusMethodNotAllowed, "metodo no permitido")
		return
	}
	if r.URL.RawQuery != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
		escribirError(w, http.StatusBadRequest, "peticion de ruta invalida")
		return
	}
	if err := validarTipoPeticionJSON(r.Header.Get("Content-Type")); err != nil {
		escribirError(w, http.StatusUnsupportedMediaType, "se requiere application/json UTF-8")
		return
	}
	solicitud, estado, err := decodificarSolicitud(w, r)
	if err != nil {
		escribirError(w, estado, "peticion de ruta invalida")
		return
	}
	resultado, err := m.calculador.Calcular(r.Context(), solicitud)
	if err != nil {
		switch {
		case errors.Is(err, dietasports.ErrSolicitudRutaInvalida):
			escribirError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, dietasports.ErrMotorRutasNoDisponible):
			escribirError(w, http.StatusBadGateway, "no se pudo consultar el motor OSRM interno autorizado")
		default:
			escribirError(w, http.StatusBadGateway, "el motor OSRM interno devolvio una respuesta no valida")
		}
		return
	}
	respuesta := proyectarRespuesta(resultado)
	w.WriteHeader(http.StatusOK)
	if m.envolverEnDatos {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": respuesta})
		return
	}
	_ = json.NewEncoder(w).Encode(respuesta)
}

func decodificarSolicitud(w http.ResponseWriter, r *http.Request) (dietasports.SolicitudCalculoRuta, int, error) {
	if r.Body == nil {
		return dietasports.SolicitudCalculoRuta{}, http.StatusBadRequest, errors.New("cuerpo ausente")
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpo))
	if err != nil {
		var demasiadoGrande *http.MaxBytesError
		if errors.As(err, &demasiadoGrande) {
			return dietasports.SolicitudCalculoRuta{}, http.StatusRequestEntityTooLarge, err
		}
		return dietasports.SolicitudCalculoRuta{}, http.StatusBadRequest, err
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return dietasports.SolicitudCalculoRuta{}, http.StatusBadRequest, errors.New("JSON vacio o no UTF-8")
	}
	decodificador := json.NewDecoder(strings.NewReader(string(contenido)))
	decodificador.DisallowUnknownFields()
	var entrada solicitudHTTP
	if err := decodificador.Decode(&entrada); err != nil {
		return dietasports.SolicitudCalculoRuta{}, http.StatusBadRequest, err
	}
	var extra any
	if err := decodificador.Decode(&extra); !errors.Is(err, io.EOF) {
		return dietasports.SolicitudCalculoRuta{}, http.StatusBadRequest, errors.New("segundo documento JSON")
	}
	coordenadas := make([]dietasports.CoordenadaRuta, 0, len(entrada.Coordenadas))
	for _, coordenada := range entrada.Coordenadas {
		coordenadas = append(coordenadas, dietasports.CoordenadaRuta{
			Latitud: coordenada.Latitud, Longitud: coordenada.Longitud, Nombre: coordenada.Nombre,
		})
	}
	return dietasports.SolicitudCalculoRuta{
		Coordenadas: coordenadas, Alternativas: entrada.Alternativas,
	}, http.StatusOK, nil
}

func validarTipoPeticionJSON(valor string) error {
	tipo, parametros, err := mime.ParseMediaType(valor)
	if err != nil || !strings.EqualFold(tipo, "application/json") {
		return errors.New("tipo no admitido")
	}
	for nombre, parametro := range parametros {
		if !strings.EqualFold(nombre, "charset") || !strings.EqualFold(parametro, "utf-8") {
			return errors.New("parametro no admitido")
		}
	}
	return nil
}

func proyectarRespuesta(resultado dietasports.ResultadoCalculoRuta) respuestaHTTP {
	rutas := make([]rutaHTTP, 0, len(resultado.Alternativas))
	for _, alternativa := range resultado.Alternativas {
		geometria := make([][]float64, 0, len(alternativa.Geometria.Coordenadas))
		for _, punto := range alternativa.Geometria.Coordenadas {
			geometria = append(geometria, []float64{punto.Longitud, punto.Latitud})
		}
		tramos := make([]tramoHTTP, 0, len(alternativa.Tramos))
		for _, tramo := range alternativa.Tramos {
			tramos = append(tramos, tramoHTTP{Distancia: tramo.DistanciaMetros, Duracion: tramo.DuracionSegundos})
		}
		rutas = append(rutas, rutaHTTP{
			Distancia: alternativa.DistanciaMetros,
			Duracion:  alternativa.DuracionSegundos,
			Tramos:    tramos,
			Geometria: geometriaHTTP{Tipo: alternativa.Geometria.Tipo, Coordenadas: geometria},
		})
	}
	return respuestaHTTP{
		Codigo: "Ok", Rutas: rutas, VersionGrafo: resultado.VersionGrafo,
		Motor: resultado.Motor, Ambito: resultado.Ambito,
	}
}

func prepararRespuestaJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func escribirError(w http.ResponseWriter, estado int, mensaje string) {
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": mensaje})
}
