// Package http adapta exclusivamente la configuración SMTP administrativa.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const RutaConfiguracionCorreo = "/api/vec/administracion/configuracion-correo"

var ErrManejadorConfiguracionCorreoInvalido = errors.New("administracion http: manejador de configuracion de correo invalido")

// autoridadConfiguracionCorreo extrae una identidad ya acreditada desde la
// cápsula ligada al canal. No acepta headers, bearer ni valores del cuerpo.
type autoridadConfiguracionCorreo interface {
	PrincipalConfiguracionCorreo(context.Context) (vecdomain.Principal, error)
}

type ejecutorConfiguracionCorreo interface {
	Consultar(context.Context, vecdomain.Principal) (admindomain.VistaConfiguracionCorreo, error)
	Actualizar(context.Context, vecdomain.Principal, admindomain.ActualizacionConfiguracionCorreo) (admindomain.VistaConfiguracionCorreo, error)
}

type ManejadorConfiguracionCorreo struct {
	autoridad autoridadConfiguracionCorreo
	ejecutor  ejecutorConfiguracionCorreo
}

func NuevoManejadorConfiguracionCorreo(autoridad autoridadConfiguracionCorreo, ejecutor ejecutorConfiguracionCorreo) (*ManejadorConfiguracionCorreo, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorConfiguracionCorreoInvalido
	}
	return &ManejadorConfiguracionCorreo{autoridad: autoridad, ejecutor: ejecutor}, nil
}

func (h *ManejadorConfiguracionCorreo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.ejecutor) || !peticionExacta(r) {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, PUT")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Context().Err() != nil {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	principal, err := h.autoridad.PrincipalConfiguracionCorreo(r.Context())
	if err != nil {
		responderError(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	if r.Method == http.MethodGet {
		vista, err := h.ejecutor.Consultar(r.Context(), principal)
		responderResultado(w, vista, err)
		return
	}
	entrada, err := entradaDesdeHTTP(r)
	if err != nil {
		responderError(w, http.StatusBadRequest, "entrada_invalida")
		return
	}
	vista, err := h.ejecutor.Actualizar(r.Context(), principal, entrada)
	responderResultado(w, vista, err)
}

type entradaHTTP struct {
	Configurada        bool                                `json:"configurada"`
	Host               string                              `json:"host"`
	Puerto             uint16                              `json:"puerto"`
	NombreServidor     string                              `json:"server_name"`
	ReferenciaCA       string                              `json:"referencia_ca"`
	RemitenteFijo      string                              `json:"remitente_fijo"`
	Usuario            string                              `json:"usuario"`
	ModoTLS            admindomain.ModoTLSCorreo           `json:"modo_tls"`
	ModoAutenticacion  admindomain.ModoAutenticacionCorreo `json:"modo_autenticacion"`
	TiempoMaximoMillis int64                               `json:"tiempo_maximo_ms"`
	Secreto            *string                             `json:"secreto"`
	SecretoConfigurado bool                                `json:"secreto_configurado"`
	VersionEsperada    uint64                              `json:"version_esperada"`
}

func entradaDesdeHTTP(r *http.Request) (admindomain.ActualizacionConfiguracionCorreo, error) {
	if r == nil || r.Body == nil {
		return admindomain.ActualizacionConfiguracionCorreo{}, ErrManejadorConfiguracionCorreoInvalido
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 32*1024))
	dec.DisallowUnknownFields()
	var entrada entradaHTTP
	if err := dec.Decode(&entrada); err != nil || dec.Decode(&struct{}{}) != ioEOF {
		return admindomain.ActualizacionConfiguracionCorreo{}, ErrManejadorConfiguracionCorreoInvalido
	}
	vista := admindomain.VistaConfiguracionCorreo{Configurada: entrada.Configurada, Host: entrada.Host, Puerto: entrada.Puerto, NombreServidor: entrada.NombreServidor, ReferenciaCA: entrada.ReferenciaCA, RemitenteFijo: entrada.RemitenteFijo, Usuario: entrada.Usuario, ModoTLS: entrada.ModoTLS, ModoAutenticacion: entrada.ModoAutenticacion, TiempoMaximoMillis: entrada.TiempoMaximoMillis, SecretoConfigurado: entrada.SecretoConfigurado}
	actualizacion := admindomain.ActualizacionConfiguracionCorreo{VistaConfiguracionCorreo: vista, VersionEsperada: entrada.VersionEsperada}
	if entrada.Secreto != nil {
		secreto, err := admindomain.NuevoSecretoCorreo([]byte(*entrada.Secreto))
		if err != nil {
			return actualizacion, err
		}
		actualizacion.SecretoNuevo = &secreto
	}
	if actualizacion.Validar() != nil {
		return admindomain.ActualizacionConfiguracionCorreo{}, ErrManejadorConfiguracionCorreoInvalido
	}
	return actualizacion, nil
}

// ioEOF evita que el decodificador acepte un segundo objeto sin exponer el cuerpo.
var ioEOF = io.EOF

func responderResultado(w http.ResponseWriter, vista admindomain.VistaConfiguracionCorreo, err error) {
	if err == nil {
		responderJSON(w, http.StatusOK, map[string]any{"configuracion": vista})
		return
	}
	if errors.Is(err, adminapp.ErrConfiguracionCorreoConflicto) {
		responderError(w, http.StatusConflict, "conflicto_version")
		return
	}
	if errors.Is(err, adminapp.ErrConfiguracionCorreoNoDisponible) {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	responderError(w, http.StatusForbidden, "acceso_denegado")
}
func responderJSON(w http.ResponseWriter, estado int, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}
func responderError(w http.ResponseWriter, estado int, codigo string) {
	responderJSON(w, estado, map[string]any{"error": map[string]string{"codigo": codigo}})
}
func peticionExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaConfiguracionCorreo && r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && r.URL.EscapedPath() == r.URL.Path && !strings.Contains(r.URL.Path, "%")
}
func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return x.IsNil()
	}
	return false
}
