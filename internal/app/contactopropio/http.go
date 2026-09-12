package contactopropio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaContactoPropio       = "/api/vec/usuarios/contacto-propio"
	maximoCuerpoContacto     = 2 * 1024
	maximoCorreoContacto     = 254
	claveErrorPeticion       = "api.error.bad_request"
	claveErrorAccesoDenegado = "api.error.forbidden"
	claveErrorNoEncontrado   = "api.error.not_found"
)

var ErrManejadorContactoPropioInvalido = errors.New("contacto propio: manejador invalido")

// EjecutorContactoPropio es la única frontera de negocio del transporte. La
// cápsula de identidad y la autorización pertenecen al Servicio, nunca a HTTP.
type EjecutorContactoPropio interface {
	Guardar(context.Context, string, uint64) (ports.ReciboContactoUsuario, error)
}

type manejadorContactoPropio struct {
	ejecutor EjecutorContactoPropio
	catalogo *i18n.Catalog
}

type entradaContactoPropio struct {
	Correo          string `json:"correo"`
	VersionEsperada uint64 `json:"version_esperada"`
}

type reciboContactoPropio struct {
	ReciboRef string `json:"recibo_ref"`
	Version   uint64 `json:"version"`
}

type errorContactoPropio struct {
	Error string `json:"error"`
}

var _ http.Handler = (*manejadorContactoPropio)(nil)
var _ EjecutorContactoPropio = (*Servicio)(nil)

func NuevoManejador(ejecutor EjecutorContactoPropio, catalogo *i18n.Catalog) (http.Handler, error) {
	if dependenciaContactoPropioNula(ejecutor) || catalogo == nil {
		return nil, ErrManejadorContactoPropioInvalido
	}
	return &manejadorContactoPropio{ejecutor: ejecutor, catalogo: catalogo}, nil
}

// NuevasRutas declara, sin registrar, la ruta exacta que montará la
// composición exterior bajo la misma autoridad central del resto de VEC.
func NuevasRutas(servicio *Servicio, catalogo *i18n.Catalog) ([]httpapi.RutaExacta, error) {
	manejador, err := NuevoManejador(servicio, catalogo)
	if err != nil {
		return nil, ErrManejadorContactoPropioInvalido
	}
	return []httpapi.RutaExacta{{Ruta: RutaContactoPropio, Manejador: manejador}}, nil
}

func (h *manejadorContactoPropio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaContactoPropioNula(h.ejecutor) || h.catalogo == nil {
		responderContactoPropio(w, nil, http.StatusForbidden, claveErrorAccesoDenegado)
		return
	}
	if !rutaContactoPropioExacta(r) {
		responderContactoPropio(w, h.catalogo, http.StatusNotFound, claveErrorNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderContactoPropio(w, h.catalogo, http.StatusMethodNotAllowed, claveErrorPeticion)
		return
	}
	if !esJSONContactoPropio(r.Header.Get("Content-Type")) {
		responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
		return
	}
	entrada, err := leerEntradaContactoPropio(r.Body)
	if err != nil {
		responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
		return
	}
	recibo, err := h.ejecutor.Guardar(r.Context(), entrada.Correo, entrada.VersionEsperada)
	if err != nil {
		if errors.Is(err, ErrContactoPropioInvalido) {
			responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
			return
		}
		responderContactoPropio(w, h.catalogo, http.StatusForbidden, claveErrorAccesoDenegado)
		return
	}
	responderJSONContactoPropio(w, http.StatusCreated, reciboContactoPropio{
		ReciboRef: recibo.EvidenciaCentral.Referencia,
		Version:   recibo.Version,
	})
}

func rutaContactoPropioExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaContactoPropio &&
		r.URL.RawQuery == "" && r.URL.EscapedPath() == r.URL.Path
}

func esJSONContactoPropio(valor string) bool {
	tipo, _, err := mime.ParseMediaType(valor)
	return err == nil && tipo == "application/json"
}

func leerEntradaContactoPropio(cuerpo io.Reader) (entradaContactoPropio, error) {
	var entrada entradaContactoPropio
	if cuerpo == nil {
		return entrada, ErrContactoPropioInvalido
	}
	contenido, err := io.ReadAll(io.LimitReader(cuerpo, maximoCuerpoContacto+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoContacto {
		return entrada, ErrContactoPropioInvalido
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &campos); err != nil || len(campos) != 2 {
		return entrada, ErrContactoPropioInvalido
	}
	if _, ok := campos["correo"]; !ok {
		return entrada, ErrContactoPropioInvalido
	}
	if _, ok := campos["version_esperada"]; !ok {
		return entrada, ErrContactoPropioInvalido
	}
	if bytes.Equal(bytes.TrimSpace(campos["version_esperada"]), []byte("null")) {
		return entrada, ErrContactoPropioInvalido
	}
	if err := validarClavesJSONContactoPropio(contenido); err != nil {
		return entrada, err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil || decodificador.Decode(&struct{}{}) != io.EOF {
		return entradaContactoPropio{}, ErrContactoPropioInvalido
	}
	if strings.TrimSpace(entrada.Correo) == "" || len(entrada.Correo) > maximoCorreoContacto ||
		strings.ContainsAny(entrada.Correo, "\r\n") {
		return entradaContactoPropio{}, ErrContactoPropioInvalido
	}
	return entrada, nil
}

func validarClavesJSONContactoPropio(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	token, err := decodificador.Token()
	if err != nil || token != json.Delim('{') {
		return ErrContactoPropioInvalido
	}
	vistas := make(map[string]struct{}, 2)
	for decodificador.More() {
		token, err := decodificador.Token()
		clave, ok := token.(string)
		if err != nil || !ok {
			return ErrContactoPropioInvalido
		}
		if _, repetida := vistas[clave]; repetida {
			return ErrContactoPropioInvalido
		}
		vistas[clave] = struct{}{}
		var valor json.RawMessage
		if err := decodificador.Decode(&valor); err != nil {
			return ErrContactoPropioInvalido
		}
	}
	if token, err := decodificador.Token(); err != nil || token != json.Delim('}') {
		return ErrContactoPropioInvalido
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return ErrContactoPropioInvalido
	}
	return nil
}

func responderContactoPropio(w http.ResponseWriter, catalogo *i18n.Catalog, estado int, clave string) {
	responderJSONContactoPropio(w, estado, errorContactoPropio{Error: textoContactoPropio(catalogo, clave)})
}

func responderJSONContactoPropio(w http.ResponseWriter, estado int, respuesta any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(respuesta)
}

func textoContactoPropio(catalogo *i18n.Catalog, clave string) string {
	if texto, ok := catalogo.Message(i18n.DefaultLocale, clave); ok {
		return texto
	}
	return clave
}

func dependenciaContactoPropioNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
