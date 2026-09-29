// Package httpapi expone la ficha propia de la persona aspirante en el área
// personal del portal externo. La persona y su identidad salen siempre de la
// frontera del portal; el JSON solo aporta operación, clave, versión, motivo
// y datos de contacto.
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

	"vec-diputacion-granada/internal/modules/aspirantes/application"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
)

// RutaMiFicha es la única ruta: GET consulta y POST da de alta o rectifica.
const RutaMiFicha = "/api/vec/aspirantes/area-personal/mi-ficha"

const limitePeticion int64 = 4 << 10

// ResolverOrdenFicha entrega la orden que la frontera del portal externo fijó
// para la petición en curso (sesión, vínculo V2 e identidad del certificado).
type ResolverOrdenFicha interface {
	ResolverOrdenFicha(context.Context) (ports.OrdenFicha, error)
}

// AuditorDenegacion anota cada 401/403 en la frontera. Si no puede anotarlo,
// la respuesta es 503: una denegación sin rastro no se entrega.
type AuditorDenegacion interface {
	AuditarDenegacion(context.Context, int) error
}

type ServicioFicha interface {
	Consultar(context.Context, ports.OrdenFicha) (application.VistaFichaPropia, error)
	Alta(context.Context, ports.OrdenFicha, application.PeticionFicha) (ports.ReciboFicha, error)
	Rectificar(context.Context, ports.OrdenFicha, application.PeticionFicha) (ports.ReciboFicha, error)
}

type Manejador struct {
	servicio ServicioFicha
	orden    ResolverOrdenFicha
	auditor  AuditorDenegacion
}

func NuevoManejador(servicio ServicioFicha, orden ResolverOrdenFicha, auditor AuditorDenegacion) (*Manejador, error) {
	if servicio == nil || orden == nil || auditor == nil {
		return nil, ports.ErrNoDisponible
	}
	return &Manejador{servicio: servicio, orden: orden, auditor: auditor}, nil
}

type peticionHTTP struct {
	Operacion       string            `json:"operacion"`
	ClaveOperacion  string            `json:"clave_operacion"`
	VersionEsperada uint64            `json:"version_esperada"`
	Motivo          string            `json:"motivo"`
	Campos          map[string]string `json:"campos"`
}

type reciboHTTP struct {
	ReciboRef string `json:"recibo_ref"`
	Version   uint64 `json:"version"`
	FechaUTC  string `json:"fecha_utc"`
	Replay    bool   `json:"replay"`
}

// camposExactos exige los campos de cada operación y ninguno más.
func camposExactos(b []byte) (string, bool) {
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return "", false
	}
	var operacion string
	if json.Unmarshal(top["operacion"], &operacion) != nil {
		return "", false
	}
	requeridos := map[string][]string{
		"alta":       {"operacion", "clave_operacion", "version_esperada", "campos"},
		"rectificar": {"operacion", "clave_operacion", "version_esperada", "motivo", "campos"},
	}[operacion]
	if requeridos == nil || len(top) != len(requeridos) {
		return "", false
	}
	for _, clave := range requeridos {
		if v, ok := top[clave]; !ok || bytes.Equal(v, []byte("null")) {
			return "", false
		}
	}
	return operacion, true
}

func esJSON(valor string) bool {
	tipo, parametros, err := mime.ParseMediaType(valor)
	return err == nil && tipo == "application/json" && len(parametros) <= 1 &&
		(len(parametros) == 0 || strings.EqualFold(parametros["charset"], "utf-8"))
}

// sinClavesDuplicadas rechaza `{"a":1,"a":2}`, que Go aceptaría quedándose
// con el último valor.
func sinClavesDuplicadas(b []byte) bool {
	d := json.NewDecoder(bytes.NewReader(b))
	var leer func(profundidad int) bool
	leer = func(profundidad int) bool {
		if profundidad > 4 {
			return false
		}
		t, err := d.Token()
		if err != nil {
			return false
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return true
		}
		switch delim {
		case '{':
			vistas := map[string]bool{}
			for d.More() {
				clave, err := d.Token()
				texto, esTexto := clave.(string)
				if err != nil || !esTexto || vistas[texto] {
					return false
				}
				vistas[texto] = true
				if !leer(profundidad + 1) {
					return false
				}
			}
		case '[':
			for d.More() {
				if !leer(profundidad + 1) {
					return false
				}
			}
		default:
			return false
		}
		cierre, err := d.Token()
		return err == nil && (cierre == json.Delim('}') || cierre == json.Delim(']'))
	}
	return leer(0) && func() bool { _, err := d.Token(); return err == io.EOF }()
}

func (m *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.servicio == nil || m.orden == nil || m.auditor == nil || r == nil || r.URL == nil {
		responderError(w, ports.ErrNoDisponible)
		return
	}
	if r.URL.Path != RutaMiFicha || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		responderError(w, ports.ErrInvalida)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responder(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido", "clave_i18n": "api.aspirantes.ficha.error.metodo_no_permitido"}})
		return
	}
	orden, err := m.orden.ResolverOrdenFicha(r.Context())
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	if r.Method == http.MethodGet {
		if r.ContentLength != 0 {
			responderError(w, ports.ErrInvalida)
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
	if r.Body == nil || r.ContentLength > limitePeticion || !esJSON(r.Header.Get("Content-Type")) {
		responderError(w, ports.ErrInvalida)
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limitePeticion))
	defer clear(b)
	if err != nil || !sinClavesDuplicadas(b) {
		responderError(w, ports.ErrInvalida)
		return
	}
	operacion, ok := camposExactos(b)
	if !ok {
		responderError(w, ports.ErrInvalida)
		return
	}
	var p peticionHTTP
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil || dec.Decode(&struct{}{}) != io.EOF || p.Campos == nil {
		responderError(w, ports.ErrInvalida)
		return
	}
	peticion := application.PeticionFicha{ClaveOperacion: p.ClaveOperacion, VersionEsperada: p.VersionEsperada, Motivo: p.Motivo, Campos: p.Campos}
	var recibo ports.ReciboFicha
	if operacion == "alta" {
		recibo, err = m.servicio.Alta(r.Context(), orden, peticion)
	} else {
		recibo, err = m.servicio.Rectificar(r.Context(), orden, peticion)
	}
	clear(p.Campos)
	if err != nil {
		m.responderError(w, r, err)
		return
	}
	estado := http.StatusCreated
	if recibo.Replay {
		estado = http.StatusOK
	}
	responder(w, estado, map[string]any{"data": reciboHTTP{ReciboRef: recibo.ReciboRef, Version: recibo.Version,
		FechaUTC: recibo.FechaUTC.UTC().Format("2006-01-02T15:04:05.000000Z"), Replay: recibo.Replay}})
}

func (m *Manejador) responderError(w http.ResponseWriter, r *http.Request, err error) {
	estado, _ := clasificar(err)
	if (estado == http.StatusUnauthorized || estado == http.StatusForbidden) && m.auditor.AuditarDenegacion(r.Context(), estado) != nil {
		responderError(w, ports.ErrNoDisponible)
		return
	}
	responderError(w, err)
}

func clasificar(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrNoAutenticado):
		return http.StatusUnauthorized, "no_autenticado"
	case errors.Is(err, ports.ErrProhibido):
		return http.StatusForbidden, "prohibido"
	case errors.Is(err, ports.ErrSinFicha):
		return http.StatusNotFound, "sin_ficha"
	case errors.Is(err, ports.ErrFichaExistente):
		return http.StatusConflict, "ficha_existente"
	case errors.Is(err, ports.ErrConflicto):
		return http.StatusConflict, "conflicto"
	case errors.Is(err, ports.ErrInvalida):
		return http.StatusUnprocessableEntity, "peticion_invalida"
	}
	return http.StatusServiceUnavailable, "no_disponible"
}

func responderError(w http.ResponseWriter, err error) {
	estado, codigo := clasificar(err)
	responder(w, estado, map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.aspirantes.ficha.error." + codigo}})
}

func responder(w http.ResponseWriter, estado int, cuerpo any) {
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(cuerpo)
}
