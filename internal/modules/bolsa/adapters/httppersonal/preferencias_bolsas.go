package httppersonal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaPreferenciasBolsas = RutaMiBolsa + "/preferencias"
const maximoCuerpoPreferenciasBolsas = 16 * 1024

type OrdenadorPreferenciasBolsas interface {
	Ordenar(context.Context, mibolsa.Orden, []string, uint64, string) (bolsa.ReciboPreferenciasBolsas, error)
}

// El montaje exige un Preparador real de Mi Bolsa y el servicio V3. Esta
// ruta no incluye GET mientras falte la autoridad nominal de lectura.
type HandlerPreferenciasBolsas struct {
	preparador Preparador
	ordenador  OrdenadorPreferenciasBolsas
}

func NuevoHandlerPreferenciasBolsas(preparador Preparador, ordenador OrdenadorPreferenciasBolsas) (http.Handler, error) {
	if nula(preparador) || nula(ordenador) {
		return nil, ErrDependenciaNoDisponible
	}
	return &HandlerPreferenciasBolsas{preparador: preparador, ordenador: ordenador}, nil
}

func (h *HandlerPreferenciasBolsas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || nula(h.preparador) || nula(h.ordenador) {
		responder(w, http.StatusServiceUnavailable, errorRespuesta{"servicio_no_disponible"})
		return
	}
	if r.URL == nil || r.URL.Path != RutaPreferenciasBolsas || r.URL.RawPath != "" ||
		r.URL.EscapedPath() != RutaPreferenciasBolsas {
		responder(w, http.StatusNotFound, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		responder(w, http.StatusMethodNotAllowed, errorRespuesta{"metodo_no_permitido"})
		return
	}
	medio, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || medio != "application/json" || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.ContentLength < 2 || r.ContentLength > maximoCuerpoPreferenciasBolsas ||
		len(r.TransferEncoding) != 0 || r.Body == nil || cabeceraProhibida(r.Header) {
		responder(w, http.StatusBadRequest, errorRespuesta{"peticion_no_permitida"})
		return
	}
	contenido, err := io.ReadAll(io.LimitReader(r.Body, maximoCuerpoPreferenciasBolsas+1))
	if err != nil || int64(len(contenido)) != r.ContentLength {
		responder(w, http.StatusBadRequest, errorRespuesta{"peticion_no_permitida"})
		return
	}
	orden, version, clave, err := decodificarOrdenPreferenciasBolsas(contenido)
	if err != nil {
		responder(w, http.StatusBadRequest, errorRespuesta{"peticion_no_permitida"})
		return
	}
	actor, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderError(w, err)
		return
	}
	recibo, err := h.ordenador.Ordenar(r.Context(), actor, orden, version, clave)
	if err != nil {
		responderErrorPreferenciasBolsas(w, err)
		return
	}
	estado := http.StatusCreated
	if recibo.Reutilizado {
		estado = http.StatusOK
	}
	responder(w, estado, struct {
		ReciboRef    string   `json:"recibo_ref"`
		Version      uint64   `json:"version"`
		Orden        []string `json:"orden"`
		RegistradoEn string   `json:"registrado_en"`
		Reutilizado  bool     `json:"reutilizado"`
	}{recibo.ReciboRef, recibo.Version, recibo.Orden, recibo.RegistradoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), recibo.Reutilizado})
}

// Exige tres claves exactas y rechaza duplicados antes de interpretar el
// cuerpo, evitando que una clave ocultada cambie el recibo de un reintento.
func decodificarOrdenPreferenciasBolsas(contenido []byte) ([]string, uint64, string, error) {
	d := json.NewDecoder(bytes.NewReader(contenido))
	primer, err := d.Token()
	if err != nil || primer != json.Delim('{') {
		return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
	}
	campos := make(map[string]json.RawMessage, 3)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
		}
		clave, ok := token.(string)
		if !ok || campos[clave] != nil ||
			clave != "orden" && clave != "version_esperada" && clave != "clave_idempotencia" {
			return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
		}
		var valor json.RawMessage
		if d.Decode(&valor) != nil {
			return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
		}
		campos[clave] = valor
	}
	cierre, err := d.Token()
	if err != nil || cierre != json.Delim('}') || len(campos) != 3 {
		return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
	}
	var orden []string
	var version uint64
	var clave string
	if json.Unmarshal(campos["orden"], &orden) != nil ||
		json.Unmarshal(campos["version_esperada"], &version) != nil ||
		json.Unmarshal(campos["clave_idempotencia"], &clave) != nil {
		return nil, 0, "", bolsa.ErrOrdenPreferenciasInvalido
	}
	return orden, version, clave, nil
}

func responderErrorPreferenciasBolsas(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responder(w, http.StatusForbidden, errorRespuesta{"acceso_denegado"})
	case errors.Is(err, bolsa.ErrPreferenciasDesactualizadas):
		responder(w, http.StatusConflict, errorRespuesta{"preferencias_desactualizadas"})
	case errors.Is(err, bolsa.ErrOrdenPreferenciasInvalido):
		responder(w, http.StatusUnprocessableEntity, errorRespuesta{"orden_invalido"})
	case errors.Is(err, bolsa.ErrPreferenciasNoDisponibles):
		responder(w, http.StatusServiceUnavailable, errorRespuesta{"servicio_no_disponible"})
	default:
		responderError(w, err)
	}
}
