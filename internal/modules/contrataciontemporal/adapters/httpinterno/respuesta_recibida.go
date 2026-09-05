package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaRegistroRespuestaRecibida = "/api/vec/contratacion-temporal/llamamientos/respuestas/registro"
const EsquemaRegistroRespuestaRecibida = "vec.contratacion-temporal.respuesta-recibida-llamamiento.v1"

type EjecutorRespuestaRecibida interface {
	Registrar(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error)
}

type respuestaRecibidaJSON struct {
	ClaveIdempotencia           string `json:"clave_idempotencia"`
	OrganizacionRef             string `json:"organizacion_ref"`
	ExpedienteRef               string `json:"expediente_ref"`
	LlamamientoRef              string `json:"llamamiento_ref"`
	ComunicacionRef             string `json:"comunicacion_ref"`
	VersionComunicacionEsperada uint64 `json:"version_comunicacion_esperada"`
	Respuesta                   string `json:"respuesta"`
	CorreoRef                   string `json:"correo_ref"`
	CorreoSHA256                string `json:"correo_sha256"`
	RecibidaEn                  string `json:"recibida_en"`
}

type respuestaRecibidaSalidaJSON struct {
	respuestaRecibidaJSON
	Esquema         string `json:"esquema"`
	JustificanteRef string `json:"justificante_ref"`
	ReciboRef       string `json:"recibo_ref"`
	AuditoriaRef    string `json:"auditoria_ref"`
	RegistradaEn    string `json:"registrada_en"`
	Estado          string `json:"estado"`
}

// Composición protege esta ruta con identidad y autorización. HTTP recibe
// una declaración y nunca toma actor, perfil o permisos del formulario.
func NuevoManejadorRespuestaRecibida(e EjecutorRespuestaRecibida) (http.Handler, error) {
	if dependenciaNula(e) {
		return nil, errors.New("contratacion temporal http: ejecutor de respuesta recibida no disponible")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallo := func(estado int, codigo string) {
			responderErrorComunicacionLlamamiento(w, errorPublicoCobertura{estado: estado, codigo: codigo,
				claveI18n: "api.contratacion_temporal.respuesta_recibida.error." + codigo})
		}
		if r == nil || r.URL == nil || r.URL.Path != RutaRegistroRespuestaRecibida ||
			r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
			r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
			fallo(http.StatusNotFound, "recurso_no_encontrado")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			fallo(http.StatusMethodNotAllowed, "metodo_no_permitido")
			return
		}
		if p := validarMetadatosComunicacionLlamamiento(r); p != nil {
			fallo(p.estado, p.codigo)
			return
		}
		var entrada respuestaRecibidaJSON
		if err := decodificarComunicacionLlamamiento(w, r, &entrada); err != nil {
			p := errorEntradaComunicacionLlamamiento(err)
			fallo(p.estado, p.codigo)
			return
		}
		fecha, err := time.Parse(time.RFC3339Nano, entrada.RecibidaEn)
		s := ports.SolicitudRegistrarRespuestaRecibida{
			ClaveIdempotencia: entrada.ClaveIdempotencia, OrganizacionRef: entrada.OrganizacionRef,
			ExpedienteRef: entrada.ExpedienteRef, LlamamientoRef: entrada.LlamamientoRef,
			ComunicacionRef: entrada.ComunicacionRef, VersionComunicacionEsperada: entrada.VersionComunicacionEsperada,
			Respuesta: ports.RespuestaLlamamiento(entrada.Respuesta), CorreoRef: entrada.CorreoRef,
			CorreoSHA256: entrada.CorreoSHA256, RecibidaEn: fecha,
		}
		if err != nil || s.Validar() != nil {
			fallo(http.StatusUnprocessableEntity, "contenido_no_valido")
			return
		}
		resultado, err := e.Registrar(r.Context(), s)
		if r.Context().Err() != nil {
			err = r.Context().Err()
		}
		if err != nil {
			estado, codigo := errorRespuestaRecibidaHTTP(err)
			fallo(estado, codigo)
			return
		}
		if resultado.ValidarPara(s) != nil {
			fallo(http.StatusBadGateway, "resultado_no_confiable")
			return
		}
		// Fechas y referencias provienen del recibo durable verificado, no se
		// declara aceptación terminal ni se añaden datos de la persona candidata.
		entrada.RecibidaEn = resultado.Solicitud.RecibidaEn.Format(time.RFC3339Nano)
		salida := respuestaRecibidaSalidaJSON{respuestaRecibidaJSON: entrada,
			Esquema: EsquemaRegistroRespuestaRecibida, JustificanteRef: resultado.JustificanteRef,
			ReciboRef: resultado.ReciboRef, AuditoriaRef: resultado.AuditoriaRef,
			RegistradaEn: resultado.RegistradaEn.Format(time.RFC3339Nano), Estado: resultado.Estado}
		estado := http.StatusCreated
		if resultado.Estado == "replay_registrada_por_rrhh" {
			estado = http.StatusOK
		}
		responderJSONCobertura(w, estado, struct {
			Data respuestaRecibidaSalidaJSON `json:"data"`
		}{salida})
	}), nil
}

func errorRespuestaRecibidaHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout, "peticion_cancelada"
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "plazo_agotado"
	case errors.Is(err, application.ErrSolicitudRespuestaRecibidaInvalida):
		return http.StatusUnprocessableEntity, "contenido_no_valido"
	case errors.Is(err, application.ErrRespuestaRecibidaDenegada):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, application.ErrClaveRespuestaRecibidaEnColision):
		return http.StatusConflict, "clave_idempotencia_reutilizada"
	case errors.Is(err, application.ErrVersionRespuestaRecibidaEnConflicto):
		return http.StatusConflict, "version_en_conflicto"
	case errors.Is(err, application.ErrResultadoRespuestaRecibidaNoConfiable):
		return http.StatusBadGateway, "resultado_no_confiable"
	default:
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
}
