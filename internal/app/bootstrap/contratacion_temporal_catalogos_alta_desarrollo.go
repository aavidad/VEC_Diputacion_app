package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const (
	rutaCatalogosAltaContratacionTemporalDesarrollo = "/api/vec/contratacion-temporal/catalogos-alta"
	esquemaCatalogosAltaContratacionTemporal        = "vec.contratacion_temporal.catalogos_alta.v1"
	contactoAltaContratacionTemporalDesarrollo      = "contacto:desarrollo:001"
	grupoSubgrupoAltaContratacionTemporalDesarrollo = "C2"
)

var errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles = errors.New(
	"contratacion temporal: catalogos de alta de desarrollo no disponibles",
)

type opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo struct {
	Referencia string `json:"referencia"`
	Etiqueta   string `json:"etiqueta"`
}

type opcionClaveCatalogosAltaContratacionTemporalDesarrollo struct {
	Clave    string `json:"clave"`
	Etiqueta string `json:"etiqueta"`
}

type centroCatalogosAltaContratacionTemporalDesarrollo struct {
	Referencia string                                                        `json:"referencia"`
	Etiqueta   string                                                        `json:"etiqueta"`
	Contactos  []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo `json:"contactos"`
}

type categoriaCatalogosAltaContratacionTemporalDesarrollo struct {
	Referencia      string                                                   `json:"referencia"`
	Etiqueta        string                                                   `json:"etiqueta"`
	GruposSubgrupos []opcionClaveCatalogosAltaContratacionTemporalDesarrollo `json:"grupos_subgrupos"`
}

type catalogosAltaContratacionTemporalDesarrollo struct {
	Esquema    string                                                        `json:"esquema"`
	Centros    []centroCatalogosAltaContratacionTemporalDesarrollo           `json:"centros"`
	Categorias []categoriaCatalogosAltaContratacionTemporalDesarrollo        `json:"categorias"`
	Motivos    []opcionClaveCatalogosAltaContratacionTemporalDesarrollo      `json:"motivos"`
	Documentos []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo `json:"documentos"`
}

type respuestaCatalogosAltaContratacionTemporalDesarrollo struct {
	Data catalogosAltaContratacionTemporalDesarrollo `json:"data"`
}

// catalogosAlta devuelve exclusivamente opciones aceptadas por el soporte de
// alta que comparte este origen. Son datos efimeros, no autoritativos y nunca
// se registran en la composicion productiva.
func (o *origenConsultasContratacionTemporalDesarrollo) catalogosAlta() (
	catalogosAltaContratacionTemporalDesarrollo,
	error,
) {
	if o == nil {
		return catalogosAltaContratacionTemporalDesarrollo{},
			errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.autoridad != AutoridadNoAutoritativa {
		return catalogosAltaContratacionTemporalDesarrollo{},
			errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles
	}
	return catalogosAltaContratacionTemporalDesarrollo{
		Esquema: esquemaCatalogosAltaContratacionTemporal,
		Centros: []centroCatalogosAltaContratacionTemporalDesarrollo{{
			Referencia: centroAltaContratacionTemporalDesarrollo,
			Etiqueta:   "Centro de desarrollo (no autoritativo)",
			Contactos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{{
				Referencia: contactoAltaContratacionTemporalDesarrollo,
				Etiqueta:   "Contacto de desarrollo (no autoritativo)",
			}},
		}},
		Categorias: []categoriaCatalogosAltaContratacionTemporalDesarrollo{{
			Referencia: categoriaAltaContratacionTemporalDesarrollo,
			Etiqueta:   "Categoría C2 de desarrollo (no autoritativa)",
			GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
				Clave:    grupoSubgrupoAltaContratacionTemporalDesarrollo,
				Etiqueta: "Grupo C2 de desarrollo (no autoritativo)",
			}},
		}},
		Motivos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
			Clave:    string(motivoAltaContratacionTemporalDesarrollo),
			Etiqueta: "Sustitución de desarrollo (no autoritativa)",
		}},
		Documentos: make([]opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo, 0),
	}, nil
}

type manejadorCatalogosAltaContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func nuevaRutaCatalogosAltaContratacionTemporalDesarrollo(
	origen *origenConsultasContratacionTemporalDesarrollo,
) (vechttp.RutaExacta, error) {
	if origen == nil || origen.autoridad != AutoridadNoAutoritativa {
		return vechttp.RutaExacta{}, ErrActivacionDesarrolloInvalida
	}
	return vechttp.RutaExacta{
		Ruta: rutaCatalogosAltaContratacionTemporalDesarrollo,
		Manejador: &manejadorCatalogosAltaContratacionTemporalDesarrollo{
			origen: origen,
		},
	}, nil
}

func (m *manejadorCatalogosAltaContratacionTemporalDesarrollo) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	if m == nil || m.origen == nil || r == nil || r.URL == nil {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(
			w, r, http.StatusServiceUnavailable, "servicio_no_disponible",
		)
		return
	}
	if r.URL.Path != rutaCatalogosAltaContratacionTemporalDesarrollo ||
		r.URL.RawQuery != "" || r.ContentLength != 0 || len(r.TransferEncoding) != 0 ||
		cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(
			w, r, http.StatusBadRequest, "solicitud_invalida",
		)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(
			w, r, http.StatusMethodNotAllowed, "metodo_no_permitido",
		)
		return
	}
	catalogos, err := m.origen.catalogosAlta()
	if err != nil {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(
			w, r, http.StatusServiceUnavailable, "servicio_no_disponible",
		)
		return
	}
	contenido, err := json.Marshal(respuestaCatalogosAltaContratacionTemporalDesarrollo{
		Data: catalogos,
	})
	if err != nil {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(
			w, r, http.StatusServiceUnavailable, "servicio_no_disponible",
		)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(contenido)
	}
}

func cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(
	cabeceras http.Header,
) bool {
	for nombre := range cabeceras {
		minusculas := strings.ToLower(nombre)
		switch {
		case minusculas == "authorization",
			minusculas == "cookie",
			minusculas == "set-cookie",
			minusculas == "proxy-authorization",
			minusculas == "proxy-connection",
			minusculas == "forwarded",
			minusculas == "remote-user",
			minusculas == "x-remote-user",
			minusculas == "x-forwarded-user",
			minusculas == "idempotency-key",
			minusculas == "content-encoding",
			minusculas == "trailer",
			minusculas == "te",
			minusculas == "expect",
			cabeceraAutoridadLibreCatalogosAltaContratacionTemporalDesarrollo(minusculas),
			minusculas == "x-http-method-override",
			strings.Contains(minusculas, "role"),
			strings.HasPrefix(minusculas, "x-auth-"),
			strings.HasPrefix(minusculas, "x-vec-"),
			strings.HasPrefix(minusculas, "x-forwarded-"),
			strings.HasPrefix(minusculas, "x-envoy-"):
			return true
		}
	}
	return false
}

func cabeceraAutoridadLibreCatalogosAltaContratacionTemporalDesarrollo(nombre string) bool {
	switch nombre {
	case "actor", "user", "usuario", "identity", "identidad", "profile", "perfil",
		"organization", "organizacion", "session", "sesion", "account", "cuenta",
		"permissions", "permission", "permisos", "permiso",
		"x-actor", "x-user", "x-usuario", "x-identity", "x-identidad",
		"x-profile", "x-perfil", "x-organization", "x-organizacion",
		"x-session", "x-sesion", "x-account", "x-cuenta",
		"x-permissions", "x-permission", "x-permisos", "x-permiso":
		return true
	default:
		return false
	}
}

func prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w http.ResponseWriter) {
	for _, cabecera := range []string{
		"Set-Cookie",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Expose-Headers",
		"Content-Encoding",
		"Location",
		"Retry-After",
	} {
		w.Header().Del(cabecera)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; base-uri 'none'; frame-ancestors 'none'",
	)
}

func responderErrorCatalogosAltaContratacionTemporalDesarrollo(
	w http.ResponseWriter,
	r *http.Request,
	estado int,
	codigo string,
) {
	contenido, err := json.Marshal(map[string]any{
		"error": map[string]string{
			"codigo":     codigo,
			"clave_i18n": "api.contratacion_temporal.catalogos_alta.error." + codigo,
		},
	})
	if err != nil {
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
		estado = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	if r == nil || r.Method != http.MethodHead {
		_, _ = w.Write(contenido)
	}
}
