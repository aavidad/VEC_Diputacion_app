package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaConsultaSeguimientoV2                 = "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento"
	maximoRespuestaConsultaSeguimientoV2Bytes = 512 * 1024
	maximasActuacionesConsultaSeguimientoV2   = 512
	maximosDocumentosActuacionSeguimientoV2   = 32
)

// ConsultorSeguimientoIncorporacionV2 expone solamente la lectura pública del
// seguimiento que produjo la incorporación original.
type ConsultorSeguimientoIncorporacionV2 interface {
	ConsultarSeguimientoIncorporacionV2(context.Context, string) (ports.VistaSeguimientoIncorporacionV2, error)
}

// NuevoManejadorConsultaSeguimientoV2 no registra rutas ni construye la
// identidad: la autoridad se resuelve desde el contexto sellado del servidor.
func NuevoManejadorConsultaSeguimientoV2(
	a AutoridadServidorIncorporacionEjercicioV2,
	e ConsultorSeguimientoIncorporacionV2,
) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ErrManejadorIncorporacionEjercicioV2
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaConsultaSeguimientoV2Exacta(r) {
			errorHTTPIncorporacionEjercicioV2(w, http.StatusBadRequest, "peticion_no_valida")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			errorHTTPIncorporacionEjercicioV2(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
			return
		}
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		expediente, err := leerConsultaIncorporacionEjercicioV2(r)
		if err != nil || !domain.ReferenciaOpacaValida(expediente) ||
			!cabecerasPropuestaFormalizacionPermitidas(r) || !acceptCompatibleJSON(r.Header) {
			errorHTTPIncorporacionEjercicioV2(w, http.StatusBadRequest, "peticion_no_valida")
			return
		}
		if err = a.ResolverContextoIncorporacionEjercicioV2(r.Context()); err != nil {
			errorOperacionIncorporacionEjercicioV2(w, err)
			return
		}
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		vista, err := e.ConsultarSeguimientoIncorporacionV2(r.Context(), expediente)
		if r.Context().Err() != nil {
			errorOperacionIncorporacionEjercicioV2(w, r.Context().Err())
			return
		}
		if err != nil {
			if !reflect.ValueOf(vista).IsZero() {
				err = ErrManejadorIncorporacionEjercicioV2
			}
			errorOperacionIncorporacionEjercicioV2(w, err)
			return
		}
		if !vistaSeguimientoIncorporacionV2HTTPValida(vista, expediente) {
			errorOperacionIncorporacionEjercicioV2(w, ErrManejadorIncorporacionEjercicioV2)
			return
		}
		vista = copiarVistaSeguimientoIncorporacionV2HTTP(vista)
		if !responderVistaSeguimientoIncorporacionV2(w, vista) {
			errorOperacionIncorporacionEjercicioV2(w, ErrManejadorIncorporacionEjercicioV2)
		}
	}), nil
}

func rutaConsultaSeguimientoV2Exacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaConsultaSeguimientoV2 &&
		r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil &&
		r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery &&
		r.URL.EscapedPath() == r.URL.Path
}

func vistaSeguimientoIncorporacionV2HTTPValida(v ports.VistaSeguimientoIncorporacionV2, expediente string) bool {
	if v.Esquema != "vec.contratacion-temporal.seguimiento-incorporacion.v2" ||
		v.Alcance != "original_incorporacion" || v.ExpedienteRef != expediente ||
		!domain.ReferenciaOpacaValida(v.ExpedienteRef) || !domain.ReferenciaOpacaValida(v.ReciboIncorporacionRef) ||
		!domain.ReferenciaOpacaValida(v.SeguimientoRef) || v.VersionExpediente == 0 ||
		v.VersionSeguimiento == 0 || !v.EstadoClave.Valida() || v.Periodo.Validar() != nil ||
		!domain.InstanteUTCCanonico(v.RegistradoEn) || !v.EjercicioSintetico || v.FirmaOficial ||
		v.EficaciaAdministrativa || len(v.Actuaciones) == 0 || len(v.Actuaciones) > maximasActuacionesConsultaSeguimientoV2 {
		return false
	}
	for _, actuacion := range v.Actuaciones {
		if !domain.ReferenciaOpacaValida(actuacion.ActuacionRef) || !actuacion.TransicionClave.Valida() ||
			!actuacion.EstadoOrigen.Valida() || !actuacion.EstadoDestino.Valida() ||
			!domain.InstanteUTCCanonico(actuacion.EfectivoEn) || !domain.InstanteUTCCanonico(actuacion.RegistradaEn) {
			return false
		}
		if len(actuacion.Documentos) > maximosDocumentosActuacionSeguimientoV2 {
			return false
		}
		for _, documento := range actuacion.Documentos {
			if !documento.TipoClave.Valida() || !domain.ReferenciaOpacaValida(documento.Referencia) {
				return false
			}
		}
	}
	return true
}

func copiarVistaSeguimientoIncorporacionV2HTTP(v ports.VistaSeguimientoIncorporacionV2) ports.VistaSeguimientoIncorporacionV2 {
	v.Actuaciones = append([]ports.ActuacionVisibleSeguimientoIncorporacionV2{}, v.Actuaciones...)
	for i := range v.Actuaciones {
		v.Actuaciones[i].Documentos = append([]domain.DocumentoSeguimiento{}, v.Actuaciones[i].Documentos...)
	}
	return v
}

// responderVistaSeguimientoIncorporacionV2 serializa por completo antes de
// enviar cabeceras: no hay truncado ni respuesta parcial ante exceso.
func responderVistaSeguimientoIncorporacionV2(w http.ResponseWriter, vista ports.VistaSeguimientoIncorporacionV2) bool {
	contenido, err := json.Marshal(struct {
		Data ports.VistaSeguimientoIncorporacionV2 `json:"data"`
	}{vista})
	if err != nil || len(contenido) > maximoRespuestaConsultaSeguimientoV2Bytes {
		return false
	}
	aplicarCabecerasCobertura(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contenido)
	return true
}
