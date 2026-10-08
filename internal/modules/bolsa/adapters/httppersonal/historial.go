package httppersonal

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaMiBolsaHistorial = "/api/vec/bolsa/mi-bolsa/historial"

type ConsultorHistorial interface {
	ConsultarHistorial(context.Context, mibolsa.Orden, int) (puertosbolsa.PaginaHistorialMiBolsa, error)
}

type HandlerHistorial struct {
	preparador Preparador
	consultor  ConsultorHistorial
}

func NuevoHistorial(preparador Preparador, consultor ConsultorHistorial) (http.Handler, error) {
	if nula(preparador) || nula(consultor) {
		return nil, ErrDependenciaNoDisponible
	}
	return &HandlerHistorial{preparador: preparador, consultor: consultor}, nil
}

func (h *HandlerHistorial) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || nula(h.preparador) || nula(h.consultor) {
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
		return
	}
	if r.URL == nil || r.URL.Path != RutaMiBolsaHistorial || r.URL.RawPath != "" || r.URL.EscapedPath() != RutaMiBolsaHistorial {
		responder(w, 404, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responder(w, 405, errorRespuesta{"metodo_no_permitido"})
		return
	}
	pagina, ok := paginaHistorial(r.URL.RawQuery)
	if !ok || r.URL.ForceQuery || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || !cuerpoAusente(r) || cabeceraProhibida(r.Header) {
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
		return
	}
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderError(w, err)
		return
	}
	p, err := h.consultor.ConsultarHistorial(r.Context(), orden, pagina)
	if err != nil {
		publicarAcuseLecturaFallida(w, err)
		responderErrorHistorial(w, err)
		return
	}
	responder(w, 200, respuestaHistorial(p))
}

func paginaHistorial(query string) (int, bool) {
	if query == "" {
		return 1, true
	}
	if !strings.HasPrefix(query, "pagina=") {
		return 0, false
	}
	valor := strings.TrimPrefix(query, "pagina=")
	if valor == "" || len(valor) > 5 || valor[0] == '0' {
		return 0, false
	}
	for _, c := range valor {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(valor)
	return n, err == nil && n >= 1 && n <= puertosbolsa.MaximaPaginaHistorialMiBolsa
}

func instanteHistorial(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

func opcionalHistorial(t *time.Time) any {
	if t == nil {
		return nil
	}
	return instanteHistorial(*t)
}

func respuestaHistorial(p puertosbolsa.PaginaHistorialMiBolsa) any {
	items := make([]map[string]any, 0, len(p.Items))
	for _, h := range p.Items {
		item := map[string]any{"clase": h.Clase, "bolsa": h.Bolsa, "categoria": h.Categoria, "ocurrido_en": instanteHistorial(h.OcurridoEn)}
		switch h.Clase {
		case "contrato_bolsa":
			item["tipo"], item["inicio"], item["fin_previsto"], item["modalidad_clave"], item["procedencia"] = h.Tipo, opcionalHistorial(h.Inicio), opcionalHistorial(h.FinPrevisto), h.ModalidadClave, h.Procedencia
		case "llamamiento":
			item["canal"], item["resultado"] = h.Canal, h.Resultado
		case "renuncia":
			item["respuesta"], item["modo"], item["estado"] = h.Respuesta, h.Modo, h.Estado
		}
		items = append(items, item)
	}
	return map[string]any{"data": map[string]any{
		"esquema":         puertosbolsa.EsquemaHistorialMiBolsa,
		"consultada_en":   instanteHistorial(p.ConsultadaEn),
		"campos_visibles": puertosbolsa.CamposHistorialMiBolsa(),
		"historial":       map[string]any{"pagina": p.Pagina, "tamano": p.Tamano, "hay_mas": p.HayMas, "items": items},
	}}
}

func responderErrorHistorial(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, puertosbolsa.ErrConsultaHistorialMiBolsaInvalida):
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
	case errors.Is(err, puertosbolsa.ErrHistorialMiBolsaNoDisponible), errors.Is(err, puertosbolsa.ErrResultadoHistorialMiBolsaInvalido):
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responder(w, 403, errorRespuesta{"acceso_denegado"})
	default:
		responderError(w, err)
	}
}
