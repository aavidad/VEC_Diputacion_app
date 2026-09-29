package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

const RutaMisPreferencias = "/api/vec/usuarios/mis-preferencias"
const RutaMisPreferenciasAreaPersonal = "/api/vec/usuarios/area-personal/mis-preferencias"
const limitePeticionPreferencias int64 = 16 << 10

// ResolverOrden recibe una identidad ya acreditada por la frontera del portal.
// El JSON nunca participa en la selección de persona, perfil o proveedor V3.
type ResolverOrden interface {
	ResolverOrdenPreferencias(context.Context) (ports.OrdenPreferencias, error)
}

// El dueño del handler registra sólo denegaciones del caso de uso tras pasar
// la frontera exacta. Una caída de auditoría se anuncia como indisponibilidad.
type AuditorDenegacion interface {
	AuditarDenegacionPreferencias(context.Context, int) error
}

type ManejadorPreferencias struct {
	servicio *application.ServicioPreferencias
	orden    ResolverOrden
	auditor  AuditorDenegacion
	ruta     string
}

func NuevoManejadorPreferencias(servicio *application.ServicioPreferencias, orden ResolverOrden, auditor AuditorDenegacion) (*ManejadorPreferencias, error) {
	return NuevoManejadorPreferenciasEnRuta(servicio, orden, auditor, RutaMisPreferencias)
}

func NuevoManejadorPreferenciasEnRuta(servicio *application.ServicioPreferencias, orden ResolverOrden, auditor AuditorDenegacion, ruta string) (*ManejadorPreferencias, error) {
	if servicio == nil || orden == nil || auditor == nil {
		return nil, ports.ErrNoDisponible
	}
	if ruta != RutaMisPreferencias && ruta != RutaMisPreferenciasAreaPersonal {
		return nil, ports.ErrNoDisponible
	}
	return &ManejadorPreferencias{servicio: servicio, orden: orden, auditor: auditor, ruta: ruta}, nil
}

func (m *ManejadorPreferencias) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.servicio == nil || m.orden == nil || r == nil || r.URL == nil {
		responderError(w, ports.ErrNoDisponible)
		return
	}
	if r.URL.Path != m.ruta || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		responderError(w, ports.ErrPeticionInvalida)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, PUT")
		responder(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido", "clave_i18n": "api.usuarios.preferencias.error.metodo_no_permitido"}})
		return
	}
	orden, err := m.orden.ResolverOrdenPreferencias(r.Context())
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	if r.Method == http.MethodGet {
		if r.ContentLength > 0 || r.Body != nil && r.ContentLength != 0 {
			responderError(w, ports.ErrPeticionInvalida)
			return
		}
		vista, err := m.servicio.Consultar(r.Context(), orden)
		if err != nil {
			m.responderError(w, r, err)
			return
		}
		responder(w, http.StatusOK, map[string]any{"data": vista})
		return
	}
	if r.Body == nil || r.ContentLength > limitePeticionPreferencias || !esJSON(r.Header.Get("Content-Type")) {
		responderError(w, ports.ErrPeticionInvalida)
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limitePeticionPreferencias))
	if err != nil || !jsonSinClavesDuplicadas(b) {
		responderError(w, ports.ErrPeticionInvalida)
		return
	}
	defer clear(b)
	var peticion ports.PeticionGuardarPreferencias
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&peticion) != nil || dec.Decode(&struct{}{}) != io.EOF || !camposObligatorios(b) {
		responderError(w, ports.ErrPeticionInvalida)
		return
	}
	recibo, err := m.servicio.Guardar(r.Context(), orden, peticion)
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	estado := http.StatusCreated
	if recibo.Replay {
		estado = http.StatusOK
	}
	responder(w, estado, map[string]any{"data": recibo})
}

func esJSON(valor string) bool {
	tipo, parametros, err := mime.ParseMediaType(valor)
	return err == nil && tipo == "application/json" && len(parametros) <= 1 &&
		(len(parametros) == 0 || strings.EqualFold(parametros["charset"], "utf-8"))
}

func camposObligatorios(b []byte) bool {
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil || len(top) != 4 {
		return false
	}
	for _, clave := range []string{"version_esperada", "catalogo_version_ref", "clave_operacion", "valores"} {
		if len(top[clave]) == 0 || bytes.Equal(top[clave], []byte("null")) {
			return false
		}
	}
	var valores map[string]json.RawMessage
	if json.Unmarshal(top["valores"], &valores) != nil || len(valores) != 8 {
		return false
	}
	for _, clave := range []string{"idioma", "tamano_texto", "alto_contraste", "tema", "inicio", "filas", "aviso_correo_tareas", "aviso_correo_plazos"} {
		if len(valores[clave]) == 0 || bytes.Equal(valores[clave], []byte("null")) {
			return false
		}
	}
	return true
}

func jsonSinClavesDuplicadas(b []byte) bool {
	d := json.NewDecoder(bytes.NewReader(b))
	if !leerValorJSON(d) {
		return false
	}
	return d.Decode(&struct{}{}) == io.EOF
}

func leerValorJSON(d *json.Decoder) bool {
	t, err := d.Token()
	if err == nil {
		return leerValorJSONDesdeToken(d, t)
	}
	return false
}

func leerValorJSONDesdeToken(d *json.Decoder, t json.Token) bool {
	delim, ok := t.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		vistas := map[string]bool{}
		for d.More() {
			clave, err := d.Token()
			texto, ok := clave.(string)
			if err == nil && ok && !vistas[texto] {
				vistas[texto] = true
				if !leerValorJSON(d) {
					return false
				}
				continue
			}
			return false
		}
		cierre, err := d.Token()
		return err == nil && cierre == json.Delim('}')
	case '[':
		for d.More() {
			if !leerValorJSON(d) {
				return false
			}
		}
		cierre, err := d.Token()
		return err == nil && cierre == json.Delim(']')
	default:
		return false
	}
}

func (m *ManejadorPreferencias) responderError(w http.ResponseWriter, r *http.Request, err error) {
	estado, _ := clasificarError(err)
	if estado == http.StatusUnauthorized || estado == http.StatusForbidden {
		if m.auditor.AuditarDenegacionPreferencias(r.Context(), estado) != nil {
			responderError(w, ports.ErrNoDisponible)
			return
		}
	}
	responderError(w, err)
}

func clasificarError(err error) (int, string) {
	estado, codigo := http.StatusServiceUnavailable, "no_disponible"
	switch {
	case errors.Is(err, ports.ErrNoAutenticado):
		estado, codigo = http.StatusUnauthorized, "no_autenticado"
	case errors.Is(err, ports.ErrProhibido):
		estado, codigo = http.StatusForbidden, "prohibido"
	case errors.Is(err, ports.ErrConflicto):
		estado, codigo = http.StatusConflict, "conflicto"
	case errors.Is(err, ports.ErrPeticionInvalida):
		estado, codigo = http.StatusUnprocessableEntity, "peticion_invalida"
	}
	return estado, codigo
}

func responderError(w http.ResponseWriter, err error) {
	estado, codigo := clasificarError(err)
	responder(w, estado, map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.usuarios.preferencias.error." + codigo}})
}

func responder(w http.ResponseWriter, estado int, cuerpo any) {
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(cuerpo)
}
