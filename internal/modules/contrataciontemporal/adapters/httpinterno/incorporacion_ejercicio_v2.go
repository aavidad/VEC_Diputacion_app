package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// AutoridadServidorIncorporacionEjercicioV2 es obligatoria en ambos metodos.
// Su implementacion confiable verifica la ruta/metodo y autoridad ACTUAL desde
// el transporte/contexto sellado del servidor; nunca desde campos del body.
// No concede permiso por parsear JSON ni restaura identidad desde el navegador.
type AutoridadServidorIncorporacionEjercicioV2 interface {
	ResolverContextoIncorporacionEjercicioV2(context.Context) error
}

type EjecutorIncorporacionEjercicioV2 = ports.ServicioIncorporacionAplicacionV2

var ErrManejadorIncorporacionEjercicioV2 = errors.New("contratacion temporal http: incorporacion no disponible")

// No registra rutas ni compone pools. Ambos metodos exigen autoridad actual
// desde contexto sellado; GET llama exclusivamente al puerto de consulta.
func NuevoManejadorIncorporacionEjercicioV2(a AutoridadServidorIncorporacionEjercicioV2, e EjecutorIncorporacionEjercicioV2) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ErrManejadorIncorporacionEjercicioV2
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaIncorporacionEjercicioV2Exacta(r) {
			errorHTTPIncorporacionEjercicioV2(w, 400, "peticion_no_valida")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			errorHTTPIncorporacionEjercicioV2(w, 405, "metodo_no_permitido")
			return
		}
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		if !cabecerasPropuestaFormalizacionPermitidas(r) || !acceptCompatibleJSON(r.Header) {
			errorHTTPIncorporacionEjercicioV2(w, 400, "peticion_no_valida")
			return
		}
		var entrada EntradaIncorporacionEjercicioV2
		var expediente string
		var err error
		if r.Method == http.MethodGet {
			expediente, err = leerConsultaIncorporacionEjercicioV2(r)
		} else {
			entrada, err = leerEntradaIncorporacionEjercicioV2(w, r)
			expediente = entrada.ExpedienteRef
		}
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		if err != nil {
			errorHTTPIncorporacionEjercicioV2(w, 400, "peticion_no_valida")
			return
		}
		if !domain.ReferenciaOpacaValida(expediente) {
			if r.Method == http.MethodGet {
				errorHTTPIncorporacionEjercicioV2(w, 400, "peticion_no_valida")
			} else {
				errorHTTPIncorporacionEjercicioV2(w, 422, "contenido_no_valido")
			}
			return
		}
		if r.Method == http.MethodPost && entrada.Validar() != nil {
			errorHTTPIncorporacionEjercicioV2(w, 422, "contenido_no_valido")
			return
		}
		err = a.ResolverContextoIncorporacionEjercicioV2(r.Context())
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		if err != nil {
			errorOperacionIncorporacionEjercicioV2(w, err)
			return
		}
		if r.Method == http.MethodGet {
			out, err := e.Consultar(r.Context(), expediente)
			if r.Context().Err() != nil {
				errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
				return
			}
			if err != nil {
				if !reflect.ValueOf(out).IsZero() {
					err = ErrManejadorIncorporacionEjercicioV2
				}
				errorOperacionIncorporacionEjercicioV2(w, err)
				return
			}
			if !proyeccionIncorporacionHTTPValida(out, expediente) {
				errorOperacionIncorporacionEjercicioV2(w, ErrManejadorIncorporacionEjercicioV2)
				return
			}
			out = copiarProyeccionIncorporacionHTTP(out)
			if r.Context().Err() != nil {
				errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
				return
			}
			responderJSONCobertura(w, http.StatusOK, struct {
				Data ports.ProyeccionIncorporacionAplicacionV2 `json:"data"`
			}{out})
			return
		}
		out, err := e.Confirmar(r.Context(), entrada.Copia())
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		if err != nil {
			if !reflect.ValueOf(out).IsZero() {
				err = ErrManejadorIncorporacionEjercicioV2
			}
			errorOperacionIncorporacionEjercicioV2(w, err)
			return
		}
		if !reciboIncorporacionHTTPValido(out, expediente) || out.SolicitudPersonalRef != entrada.SolicitudPersonalRef ||
			out.VersionActualExpediente > entrada.VersionActualExpedienteObservada {
			errorOperacionIncorporacionEjercicioV2(w, ErrManejadorIncorporacionEjercicioV2)
			return
		}
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		// Recibo PUBLICO minimizado de B; no se serializa Material/Orden/Historia.
		responderJSONCobertura(w, http.StatusOK, struct {
			Data ports.ReciboIncorporacionAplicacionV2 `json:"data"`
		}{out})
	}), nil
}

func errorOperacionIncorporacionEjercicioV2(w http.ResponseWriter, err error) {
	status, codigo := 503, "servicio_no_disponible"
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
	case errors.Is(err, ports.ErrIntencionIncorporacionAplicacion):
		status, codigo = 422, "contenido_no_valido"
	case errors.Is(err, ports.ErrDenegadaIncorporacionAplicacion), errors.Is(err, ports.ErrAutorizacionDenegada), errors.Is(err, application.ErrConfirmacionIncorporacionDenegada):
		status, codigo = 403, "acceso_denegado"
	case errors.Is(err, ports.ErrConflictoIncorporacionAplicacion):
		status, codigo = 409, "conflicto"
	}
	errorHTTPIncorporacionEjercicioV2(w, status, codigo)
}

func rutaIncorporacionEjercicioV2Exacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaIncorporacionEjercicioV2 &&
		r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil &&
		r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery &&
		(r.Method == http.MethodGet || r.URL.RawQuery == "") && r.URL.EscapedPath() == r.URL.Path
}

func errorHTTPIncorporacionEjercicioV2(w http.ResponseWriter, status int, codigo string) {
	responderJSONCobertura(w, status, map[string]any{"error": map[string]string{
		"codigo":          codigo,
		"clave_i18n":      "api.contratacion_temporal.incorporacion_ejercicio.error." + codigo,
		"correlacion_ref": "corr_no_disponible",
	}})
}
