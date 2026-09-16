package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const RutaFichaGINPIXV2 = "/api/vec/contratacion-temporal/incorporaciones-ejercicio/ficha-ginpix"

type AutoridadFichaGINPIXV2 interface{ ResolverContextoFichaGINPIXV2(context.Context) error }
type PreparadorFichaGINPIXV2 interface {
	Preparar(context.Context, string) (ginpixfichero.PreparacionExportacion, error)
}

func NuevoManejadorFichaGINPIXV2(a AutoridadFichaGINPIXV2, p PreparadorFichaGINPIXV2) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(p) {
		return nil, incorporacionejercicio.ErrFichaGINPIXV2NoDisponible
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaFichaGINPIXV2Exacta(r) {
			errorFichaGINPIXV2(w, r, 400, "peticion_no_valida")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			errorFichaGINPIXV2(w, r, 405, "metodo_no_permitido")
			return
		}
		exp, err := leerConsultaFichaGINPIXV2(r)
		if err != nil || !domain.ReferenciaOpacaValida(exp) || !cabecerasPropuestaFormalizacionPermitidas(r) {
			errorFichaGINPIXV2(w, r, 400, "peticion_no_valida")
			return
		}
		if err = a.ResolverContextoFichaGINPIXV2(r.Context()); err != nil {
			errorOperacionFichaGINPIXV2(w, r, err)
			return
		}
		if errContexto := r.Context().Err(); errContexto != nil {
			errorFichaGINPIXV2(w, r, 503, "servicio_no_disponible", errContexto)
			return
		}
		out, err := p.Preparar(r.Context(), exp)
		if err != nil {
			errorOperacionFichaGINPIXV2(w, r, err)
			return
		}
		metadatos, err := out.Metadatos()
		if err != nil || metadatos.ExpedienteRef != exp || metadatos.VersionExpediente == 0 || !domain.ReferenciaOpacaValida(metadatos.IncorporacionRef) {
			errorFichaGINPIXV2(w, r, 503, "servicio_no_disponible", err)
			return
		}
		contenido, err := out.Contenido()
		if err != nil || len(contenido) == 0 || len(contenido) > ginpixfichero.MaximoBytesFicheroGINPIX {
			errorFichaGINPIXV2(w, r, 503, "servicio_no_disponible", err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=ficha-ginpix-ejercicio.json")
		w.Header().Set("Content-Length", stringEnteroFichaGINPIXV2(len(contenido)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(contenido)
	}), nil
}
func leerConsultaFichaGINPIXV2(r *http.Request) (string, error) {
	if r == nil || r.URL == nil || len(r.URL.RawQuery) > 600 || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return "", errors.New("invalida")
	}
	if r.Body != nil && r.Body != http.NoBody {
		b, e := io.ReadAll(io.LimitReader(r.Body, 1))
		if e != nil || len(b) != 0 {
			return "", errors.New("invalida")
		}
	}
	q, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil || len(q) != 1 || len(q["expediente_ref"]) != 1 || q.Get("expediente_ref") == "" {
		return "", errors.New("invalida")
	}
	return q.Get("expediente_ref"), nil
}
func rutaFichaGINPIXV2Exacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaFichaGINPIXV2 && r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery && r.URL.EscapedPath() == r.URL.Path
}
func errorOperacionFichaGINPIXV2(w http.ResponseWriter, peticion *http.Request, err error) {
	status, codigo := 503, "servicio_no_disponible"
	if errors.Is(err, incorporacionejercicio.ErrFichaGINPIXV2Denegada) {
		status, codigo = 403, "acceso_denegado"
	}
	if errors.Is(err, incorporacionejercicio.ErrFichaGINPIXV2Conflicto) {
		status, codigo = 409, "recibo_no_confirmado"
	}
	errorFichaGINPIXV2(w, peticion, status, codigo, err)
}
func errorFichaGINPIXV2(w http.ResponseWriter, peticion *http.Request, status int, codigo string, causas ...error) {
	responderJSONCobertura(w, peticion, status, map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.contratacion_temporal.ficha_ginpix.error." + codigo, "correlacion_ref": nuevaCorrelacionCobertura()}}, causas...)
}
func stringEnteroFichaGINPIXV2(v int) string { return strconv.Itoa(v) }
