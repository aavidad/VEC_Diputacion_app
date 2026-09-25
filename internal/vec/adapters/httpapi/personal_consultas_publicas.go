package httpapi

import (
	"errors"
	"net/http"
)

// Consultas públicas de solo lectura de Personal. Sirven catálogos sin personas,
// ocupantes ni datos de empleado: categorías profesionales gobernadas, la RPT
// publicada y la estructura organizativa de referencia. No resuelven identidad
// ni conceden permisos; la composición decide dónde se montan y cada handler
// solo atiende su ruta exacta con GET o HEAD.
const (
	RutaCategoriasProfesionalesPersonal       = rutaCategoriasProfesionalesPresentacion
	RutaRPTPublicaPersonal                    = rutaRPTPublicaPresentacion
	RutaEstructuraOrganizativaPublicaPersonal = rutaEstructuraOrganizativaPublicaPresentacion
)

var ErrConsultaPublicaPersonalInvalida = errors.New("httpapi: consulta publica de Personal invalida")

// NewHandlerCategoriasProfesionalesPublicas sirve el catálogo profesional
// gobernado tal como lo publica su fuente, sea o no de demostración: la marca
// viaja en la respuesta para que el cliente la muestre.
func NewHandlerCategoriasProfesionalesPublicas(consulta ConsultaCategoriasProfesionales) (http.Handler, error) {
	if dependenciaHTTPNula(consulta) {
		return nil, ErrConsultaPublicaPersonalInvalida
	}
	return handlerCategoriasProfesionales(consulta, false), nil
}

// NewHandlerRPTPublica sirve la proyección pública de la RPT inmovilizada por
// huella en su adaptador.
func NewHandlerRPTPublica(consulta ConsultaRPTPublica) (http.Handler, error) {
	if dependenciaHTTPNula(consulta) {
		return nil, ErrConsultaPublicaPersonalInvalida
	}
	return handlerRPTPublica(consulta), nil
}

// NewHandlerEstructuraOrganizativaPublica sirve la estructura organizativa de
// referencia, sin personas ni cadena de mando efectiva.
func NewHandlerEstructuraOrganizativaPublica(consulta ConsultaEstructuraOrganizativaPublica) (http.Handler, error) {
	if dependenciaHTTPNula(consulta) {
		return nil, ErrConsultaPublicaPersonalInvalida
	}
	return handlerEstructuraOrganizativaPublica(consulta), nil
}

func handlerCategoriasProfesionales(consulta ConsultaCategoriasProfesionales, exigirDemostracion bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path != rutaCategoriasProfesionalesPresentacion || r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Allow", "GET, HEAD")
			writeErrorCategoriasProfesionales(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		servirCategoriasProfesionales(w, r, consulta, exigirDemostracion)
	})
}

func handlerRPTPublica(consulta ConsultaRPTPublica) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			escribirRPTPublicaError(w, http.StatusServiceUnavailable, "rpt_publica_no_disponible")
			return
		}
		if r.URL.Path != rutaRPTPublicaPresentacion || r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			escribirRPTPublicaError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		servirRPTPublica(w, r, consulta)
	})
}

func handlerEstructuraOrganizativaPublica(consulta ConsultaEstructuraOrganizativaPublica) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			escribirEstructuraOrganizativaPublicaError(w, http.StatusServiceUnavailable, "estructura_organizativa_publica_no_disponible")
			return
		}
		if r.URL.Path != rutaEstructuraOrganizativaPublicaPresentacion || r.URL.RawPath != "" || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			escribirEstructuraOrganizativaPublicaError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		estructura, err := consulta.Obtener(r.Context())
		if err != nil || estructura.Validar() != nil {
			escribirEstructuraOrganizativaPublicaError(w, http.StatusServiceUnavailable, "estructura_organizativa_publica_no_disponible")
			return
		}
		escribirEstructuraOrganizativaPublicaJSON(w, http.StatusOK, map[string]any{"estructura_organizativa": estructura})
	})
}
