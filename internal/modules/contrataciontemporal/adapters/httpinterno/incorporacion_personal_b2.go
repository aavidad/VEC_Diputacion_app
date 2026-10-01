package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// El mismo manejador se monta en las dos rutas exactas. La frontera confiable
// liga método/ruta al perfil actual antes de invocar el servicio nominal.
// No registra rutas, no construye identidad y no consume recibos del cliente.
func NuevoManejadorIncorporacionPersonalB2(a AutoridadServidorIncorporacionEjercicioV2, e EjecutorIncorporacionPersonalB2) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ErrManejadorIncorporacionPersonalB2
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaHTTPB2Exacta(r) {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		if r.Method != http.MethodPost && !(r.URL.Path == RutaPlanB2 && r.Method == http.MethodGet) {
			allow := "POST"
			if r.URL.Path == RutaPlanB2 {
				allow = "GET, POST"
			}
			w.Header().Set("Allow", allow)
			errorHTTPB2(w, r, 405, "metodo_no_permitido")
			return
		}
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if !cabecerasPropuestaFormalizacionPermitidas(r) || !acceptCompatibleJSON(r.Header) {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		var plan EntradaPlanB2
		var confirmacion EntradaConfirmacionB2
		var exp string
		var err error
		switch {
		case r.Method == http.MethodGet:
			exp, err = leerConsultaIncorporacionEjercicioV2(r)
		case r.URL.Path == RutaPlanB2:
			err = leerCuerpoHTTPB2(w, r, &plan, []string{"expediente_ref", "version_expediente", "puesto_ref", "plaza_ref", "version_plantilla_ref", "version_rpt_ref", "regimen", "modalidad", "clase_ocupacion", "desde", "hasta", "motivo_clave", "documento_ref", "documento_sha256", "clave_idempotencia"})
			exp = plan.ExpedienteRef
		default:
			err = leerCuerpoHTTPB2(w, r, &confirmacion, []string{"expediente_ref", "plan_ref", "version_plan", "clave_idempotencia"})
			exp = confirmacion.ExpedienteRef
		}
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if err != nil {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		switch {
		case r.Method == http.MethodGet:
			if !referenciaConsultaHTTPB2(exp) {
				errorHTTPB2(w, r, 400, "peticion_no_valida")
				return
			}
		case r.URL.Path == RutaPlanB2:
			if plan.Validar() != nil {
				errorHTTPB2(w, r, 422, "contenido_no_valido")
				return
			}
		default:
			if confirmacion.Validar() != nil {
				errorHTTPB2(w, r, 422, "contenido_no_valido")
				return
			}
		}
		if err = a.ResolverContextoIncorporacionEjercicioV2(r.Context()); err != nil {
			errorOperacionHTTPB2(w, r, err)
			return
		}
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if r.URL.Path == RutaConfirmacionB2 {
			out, err := e.Confirmar(r.Context(), confirmacion)
			if r.Context().Err() != nil {
				errorOperacionHTTPB2(w, r, r.Context().Err())
				return
			}
			if err != nil {
				if !reflect.ValueOf(out).IsZero() {
					err = ErrManejadorIncorporacionPersonalB2
				}
				errorOperacionHTTPB2(w, r, err)
				return
			}
			if !reciboHTTPB2Valido(out, exp) || out.PlanRef != confirmacion.PlanRef || out.PlanVersion != confirmacion.VersionPlan {
				errorOperacionHTTPB2(w, r, ErrManejadorIncorporacionPersonalB2)
				return
			}
			responderJSONCobertura(w, r, 200, struct {
				Data ReciboIncorporacionPersonalB2HTTP `json:"data"`
			}{out})
			return
		}
		var out ProyeccionIncorporacionPersonalB2HTTP
		if r.Method == http.MethodGet {
			out, err = e.Consultar(r.Context(), exp)
		} else {
			out, err = e.Preparar(r.Context(), plan)
		}
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if err != nil {
			if !reflect.ValueOf(out).IsZero() {
				err = ErrManejadorIncorporacionPersonalB2
			}
			errorOperacionHTTPB2(w, r, err)
			return
		}
		if !proyeccionHTTPB2Valida(out, exp) || (r.Method == http.MethodPost && (out.Plan == nil || out.Plan.Intencion != plan)) {
			errorOperacionHTTPB2(w, r, ErrManejadorIncorporacionPersonalB2)
			return
		}
		// Normalizar listas vacías conserva un wire estable para la UI.
		out = copiarProyeccionHTTPB2(out)
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		responderJSONCobertura(w, r, 200, struct {
			Data ProyeccionIncorporacionPersonalB2HTTP `json:"data"`
		}{out})
	}), nil
}

func rutaHTTPB2Exacta(r *http.Request) bool {
	return r != nil && r.URL != nil && (r.URL.Path == RutaPlanB2 || r.URL.Path == RutaConfirmacionB2) && r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery && (r.Method == http.MethodGet || r.URL.RawQuery == "") && r.URL.EscapedPath() == r.URL.Path
}
func referenciaConsultaHTTPB2(s string) bool {
	return domain.ReferenciaOpacaValida(s)
}
func copiarProyeccionHTTPB2(v ProyeccionIncorporacionPersonalB2HTTP) ProyeccionIncorporacionPersonalB2HTTP {
	v.Prerrequisitos = append([]PrerrequisitoB2{}, v.Prerrequisitos...)
	v.Opciones.Vacantes = append([]OpcionVacanteB2{}, v.Opciones.Vacantes...)
	v.Opciones.Regimenes = append([]OpcionCatalogoB2{}, v.Opciones.Regimenes...)
	v.Opciones.Modalidades = append([]OpcionCatalogoB2{}, v.Opciones.Modalidades...)
	v.Opciones.ClasesOcupacion = append([]OpcionClaseOcupacionB2{}, v.Opciones.ClasesOcupacion...)
	v.Opciones.Motivos = append([]string{}, v.Opciones.Motivos...)
	v.Opciones.Documentos = append([]OpcionDocumentoB2{}, v.Opciones.Documentos...)
	if v.Plan != nil {
		p := *v.Plan
		v.Plan = &p
	}
	if v.Recibo != nil {
		r := *v.Recibo
		v.Recibo = &r
	}
	return v
}
func errorOperacionHTTPB2(w http.ResponseWriter, r *http.Request, err error) {
	status, codigo := 503, "servicio_no_disponible"
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
	case errors.Is(err, ErrPeticionIncorporacionPersonalB2):
		status, codigo = 422, "contenido_no_valido"
	case errors.Is(err, ErrDenegadaIncorporacionPersonalB2), errors.Is(err, ports.ErrAutorizacionDenegada), errors.Is(err, ports.ErrDenegadaIncorporacionAplicacion):
		status, codigo = 403, "acceso_denegado"
	case errors.Is(err, ErrConflictoIncorporacionPersonalB2):
		status, codigo = 409, "conflicto"
	case errors.Is(err, ErrPreparacionPendienteIncorporacionPersonalB2):
		status, codigo = 409, "preparacion_pendiente"
	}
	errorHTTPB2(w, r, status, codigo, err)
}
func errorHTTPB2(w http.ResponseWriter, r *http.Request, status int, codigo string, causas ...error) {
	responderJSONCobertura(w, r, status, map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.contratacion_temporal.incorporacion_personal_b2.error." + codigo, "correlacion_ref": nuevaCorrelacionCobertura()}}, causas...)
}
