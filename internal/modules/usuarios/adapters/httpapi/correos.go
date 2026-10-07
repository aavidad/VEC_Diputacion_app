package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	"vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

// Rutas exactas de «Mis correos», una por superficie. GET devuelve el
// conjunto propio; POST ejecuta una operación con clave idempotente.
const RutaMisCorreos = "/api/vec/usuarios/mis-correos"
const RutaMisCorreosAreaPersonal = "/api/vec/usuarios/area-personal/mis-correos"
const limitePeticionCorreos int64 = 4 << 10

// ResolverOrdenCorreos recibe una identidad ya acreditada por la frontera del
// portal. El JSON nunca participa en la selección de persona o superficie.
type ResolverOrdenCorreos interface {
	ResolverOrdenCorreos(context.Context) (ports.OrdenCorreos, error)
}

// AuditorIntentosConsultaCorreos recibe únicamente el resultado de la consulta
// interna tras capturar su identidad en la frontera. Nunca recibe el cuerpo.
type AuditorIntentosConsultaCorreos interface {
	PrepararIntentoConsultaCorreos(context.Context) error
	AuditarIntentoConsultaCorreos(context.Context, int) error
}

type ManejadorCorreos struct {
	servicio         *application.ServicioCorreos
	orden            ResolverOrdenCorreos
	auditor          AuditorDenegacion
	ruta             string
	intentosConsulta AuditorIntentosConsultaCorreos
}

func NuevoManejadorCorreosEnRuta(servicio *application.ServicioCorreos, orden ResolverOrdenCorreos, auditor AuditorDenegacion, ruta string) (*ManejadorCorreos, error) {
	if servicio == nil || orden == nil || auditor == nil || (ruta != RutaMisCorreos && ruta != RutaMisCorreosAreaPersonal) {
		return nil, ports.ErrCorreosNoDisponible
	}
	return &ManejadorCorreos{servicio: servicio, orden: orden, auditor: auditor, ruta: ruta}, nil
}

// El montaje interno usa este constructor estricto. POST y el constructor
// anterior mantienen su auditoría de frontera y sus contratos existentes.
func NuevoManejadorConsultaCorreosInternaConIntentos(servicio *application.ServicioCorreos, orden ResolverOrdenCorreos, auditor AuditorDenegacion, intentos AuditorIntentosConsultaCorreos) (*ManejadorCorreos, error) {
	if intentos == nil || reflect.ValueOf(intentos).Kind() == reflect.Pointer && reflect.ValueOf(intentos).IsNil() {
		return nil, ports.ErrCorreosNoDisponible
	}
	m, err := NuevoManejadorCorreosEnRuta(servicio, orden, auditor, RutaMisCorreos)
	if err != nil {
		return nil, err
	}
	m.intentosConsulta = intentos
	return m, nil
}

// peticionCorreosHTTP es el único cuerpo admitido. «operacion» elige el caso
// de uso; los demás campos se validan otra vez en la aplicación.
type peticionCorreosHTTP struct {
	Operacion       string `json:"operacion"`
	VersionEsperada uint64 `json:"version_esperada"`
	ClaveOperacion  string `json:"clave_operacion"`
	CorreoRef       string `json:"correo_ref"`
	Direccion       string `json:"direccion"`
	Codigo          string `json:"codigo"`
}

// camposCorreosValidos exige exactamente los campos de cada operación.
func camposCorreosValidos(b []byte) bool {
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return false
	}
	var operacion string
	if len(top["operacion"]) == 0 || json.Unmarshal(top["operacion"], &operacion) != nil {
		return false
	}
	extras := map[string][]string{"anadir": {"direccion"}, "reenviar": {"correo_ref"}, "activar": {"correo_ref"}, "retirar": {"correo_ref"}, "verificar": {"correo_ref", "codigo"}}
	requeridos, ok := extras[operacion]
	if !ok || len(top) != 3+len(requeridos) {
		return false
	}
	for _, clave := range append([]string{"version_esperada", "clave_operacion"}, requeridos...) {
		if len(top[clave]) == 0 || bytes.Equal(top[clave], []byte("null")) {
			return false
		}
	}
	return true
}

func (m *ManejadorCorreos) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.servicio == nil || m.orden == nil || r == nil || r.URL == nil {
		responderErrorCorreos(w, ports.ErrCorreosNoDisponible)
		return
	}
	if r.URL.Path != m.ruta || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		responderErrorCorreos(w, ports.ErrCorreosInvalidos)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responder(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido", "clave_i18n": "api.usuarios.correos.error.metodo_no_permitido"}})
		return
	}
	if r.Method == http.MethodGet && m.intentosConsulta != nil {
		if m.intentosConsulta.PrepararIntentoConsultaCorreos(r.Context()) != nil {
			responderErrorCorreos(w, ports.ErrCorreosNoDisponible)
			return
		}
	}
	orden, err := m.orden.ResolverOrdenCorreos(r.Context())
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	if r.Method == http.MethodGet {
		if r.ContentLength > 0 || r.Body != nil && r.ContentLength != 0 {
			m.responderError(w, r, ports.ErrCorreosInvalidos)
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
	if r.Body == nil || r.ContentLength > limitePeticionCorreos || !esJSON(r.Header.Get("Content-Type")) {
		responderErrorCorreos(w, ports.ErrCorreosInvalidos)
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limitePeticionCorreos))
	if err != nil || !jsonSinClavesDuplicadas(b) || !camposCorreosValidos(b) {
		responderErrorCorreos(w, ports.ErrCorreosInvalidos)
		return
	}
	defer clear(b)
	var peticion peticionCorreosHTTP
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&peticion) != nil || dec.Decode(&struct{}{}) != io.EOF {
		responderErrorCorreos(w, ports.ErrCorreosInvalidos)
		return
	}
	recibo, err := m.servicio.Operar(r.Context(), orden, peticion.Operacion, ports.PeticionCorreo{
		VersionEsperada: peticion.VersionEsperada, ClaveOperacion: peticion.ClaveOperacion,
		CorreoRef: peticion.CorreoRef, Direccion: peticion.Direccion, Codigo: peticion.Codigo,
	})
	peticion.Direccion, peticion.Codigo = "", ""
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

func (m *ManejadorCorreos) responderError(w http.ResponseWriter, r *http.Request, err error) {
	estado, _ := clasificarErrorCorreos(err)
	if r.Method == http.MethodGet && m.intentosConsulta != nil {
		if m.intentosConsulta.AuditarIntentoConsultaCorreos(r.Context(), estado) != nil {
			responderErrorCorreos(w, ports.ErrCorreosNoDisponible)
			return
		}
	} else if estado == http.StatusUnauthorized || estado == http.StatusForbidden {
		if m.auditor.AuditarDenegacionPreferencias(r.Context(), estado) != nil {
			responderErrorCorreos(w, ports.ErrCorreosNoDisponible)
			return
		}
	}
	responderErrorCorreos(w, err)
}

func clasificarErrorCorreos(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrCorreosNoAutenticado):
		return http.StatusUnauthorized, "no_autenticado"
	case errors.Is(err, ports.ErrCorreosProhibido):
		return http.StatusForbidden, "prohibido"
	case errors.Is(err, ports.ErrCorreosCodigoIncorrecto):
		return http.StatusUnprocessableEntity, "codigo_incorrecto"
	case errors.Is(err, ports.ErrCorreosCodigoCaducado):
		return http.StatusConflict, "codigo_caducado"
	case errors.Is(err, ports.ErrCorreosYaRegistrado):
		return http.StatusConflict, "ya_registrado"
	case errors.Is(err, ports.ErrCorreosMaximo):
		return http.StatusConflict, "maximo"
	case errors.Is(err, ports.ErrCorreosEnUso):
		return http.StatusConflict, "en_uso"
	case errors.Is(err, ports.ErrCorreosConflicto):
		return http.StatusConflict, "conflicto"
	case errors.Is(err, ports.ErrCorreosLimite):
		return http.StatusTooManyRequests, "limite"
	case errors.Is(err, ports.ErrCorreosInvalidos):
		return http.StatusUnprocessableEntity, "peticion_invalida"
	}
	return http.StatusServiceUnavailable, "no_disponible"
}

func responderErrorCorreos(w http.ResponseWriter, err error) {
	estado, codigo := clasificarErrorCorreos(err)
	cuerpo := map[string]any{"codigo": codigo, "clave_i18n": "api.usuarios.correos.error." + codigo}
	var incorrecto ports.CodigoIncorrecto
	if errors.As(err, &incorrecto) && incorrecto.IntentosRestantes >= 0 {
		cuerpo["intentos_restantes"] = incorrecto.IntentosRestantes
	}
	responder(w, estado, map[string]any{"error": cuerpo})
}
