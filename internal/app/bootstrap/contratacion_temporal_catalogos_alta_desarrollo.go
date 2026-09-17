package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	personalcatalogos "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
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

var gruposPorCategoriaSinteticaDesarrollo = map[string]string{
	categoriaAltaContratacionTemporalDesarrollo: grupoSubgrupoAltaContratacionTemporalDesarrollo,
	"categoria:desarrollo:c1":                   "C1",
	"categoria:desarrollo:a1":                   "A1",
	"categoria:desarrollo:a2":                   "A2",
	"categoria:desarrollo:b":                    "B",
	"categoria:desarrollo:ap":                   "AP",
}

var categoriasSinteticasDesarrollo = []categoriaCatalogosAltaContratacionTemporalDesarrollo{
	{
		Referencia: "categoria:desarrollo:c2",
		Etiqueta:   "Auxiliar administrativo/a",
		GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
			{Clave: "C2", Etiqueta: "Grupo C2"},
		},
	},
	{
		Referencia: "categoria:desarrollo:c1",
		Etiqueta:   "Administrativo/a",
		GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
			{Clave: "C1", Etiqueta: "Grupo C1"},
		},
	},
	{
		Referencia: "categoria:desarrollo:a1",
		Etiqueta:   "Técnico/a de administración general",
		GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
			{Clave: "A1", Etiqueta: "Grupo A1"},
		},
	},
	{
		Referencia: "categoria:desarrollo:a2",
		Etiqueta:   "Técnico/a medio/a",
		GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
			{Clave: "A2", Etiqueta: "Grupo A2"},
		},
	},
	{
		Referencia: "categoria:desarrollo:b",
		Etiqueta:   "Técnico/a especialista",
		GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
			{Clave: "B", Etiqueta: "Grupo B"},
		},
	},
	{
		Referencia: "categoria:desarrollo:ap",
		Etiqueta:   "Operario/a de servicios",
		GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
			{Clave: "AP", Etiqueta: "Grupo AP"},
		},
	},
}

func categoriaDeCatalogoDesarrollo(ref string) bool {
	_, ok := gruposPorCategoriaSinteticaDesarrollo[ref]
	return ok
}

func grupoSubgrupoDeCatalogoValido(categoriaRef string, grupoSubgrupo string) bool {
	esperado, ok := gruposPorCategoriaSinteticaDesarrollo[categoriaRef]
	return ok && esperado == grupoSubgrupo
}

func categoriaYGrupoDeCatalogoDesarrolloValidos(catalogo *catalogosAltaContratacionTemporalDesarrollo, referencia, grupo string) bool {
	if catalogo == nil {
		return grupoSubgrupoDeCatalogoValido(referencia, grupo)
	}
	for _, categoria := range catalogo.Categorias {
		if categoria.Referencia != referencia {
			continue
		}
		for _, candidato := range categoria.GruposSubgrupos {
			if candidato.Clave == grupo {
				return true
			}
		}
	}
	// Las categorías sintéticas siguen valiendo para los expedientes que ya las usan
	// aunque el catálogo publicado sea el de la RPT.
	return grupoSubgrupoDeCatalogoValido(referencia, grupo)
}

func nuevoCatalogoDesarrollo(rutaFuente, rutaRPT string) (*catalogosAltaContratacionTemporalDesarrollo, error) {
	if strings.TrimSpace(rutaFuente) != "" {
		return construirCatalogosAltaDesarrollo(rutaFuente, rutaRPT)
	}
	return &catalogosAltaContratacionTemporalDesarrollo{
		Esquema: esquemaCatalogosAltaContratacionTemporal,
		Centros: []centroCatalogosAltaContratacionTemporalDesarrollo{{
			Referencia: centroAltaContratacionTemporalDesarrollo,
			Etiqueta:   "Centro solicitante",
			Contactos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{{
				Referencia: contactoAltaContratacionTemporalDesarrollo, Etiqueta: "Contacto del centro",
			}},
		}},
		Categorias: append([]categoriaCatalogosAltaContratacionTemporalDesarrollo(nil), categoriasSinteticasDesarrollo...),
		Motivos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
			Clave: string(motivoAltaContratacionTemporalDesarrollo), Etiqueta: "Sustitución temporal",
		}},
		Documentos: make([]opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo, 0),
	}, nil
}

func construirCatalogosAltaDesarrollo(rutaFuente, rutaRPT string) (*catalogosAltaContratacionTemporalDesarrollo, error) {
	if strings.TrimSpace(rutaFuente) == "" {
		return nil, nil
	}
	fuente, err := fichero.NuevaConsultaCatalogos(rutaFuente)
	if err != nil {
		return nil, err
	}
	consulta, err := personalcatalogos.NuevaConsultaEstructuraOrganizativa(
		fuente, "estructura-organizativa-dipgra", 1,
	)
	if err != nil {
		return nil, err
	}
	datos, err := consulta.Obtener(context.Background())
	if err != nil {
		return nil, err
	}
	var centros []centroCatalogosAltaContratacionTemporalDesarrollo
	for _, u := range datos.Unidades {
		if u.Tipo == "centro" {
			cod := u.CodigoFuente
			if cod == "" {
				cod = u.Clave
			}
			centros = append(centros, centroCatalogosAltaContratacionTemporalDesarrollo{
				Referencia: "centro:rpt:" + cod,
				Etiqueta:   u.Etiqueta,
				Contactos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{
					{
						Referencia: "contacto:rpt:" + cod,
						Etiqueta:   "Contacto del centro",
					},
				},
			})
		}
	}
	categorias := make([]categoriaCatalogosAltaContratacionTemporalDesarrollo, len(categoriasSinteticasDesarrollo))
	copy(categorias, categoriasSinteticasDesarrollo)
	if strings.TrimSpace(rutaRPT) != "" {
		// Con la RPT pública configurada, las categorías son las reales de la
		// Diputación; las sintéticas quedan solo para los expedientes que ya las usan.
		categoriasRPT, err := cargarCategoriasRPTDesarrollo(rutaRPT)
		if err != nil {
			return nil, err
		}
		// Las sintéticas se publican al final para que los expedientes que ya las
		// usan sigan mostrando su nombre en cuadro, detalle y documentos.
		categorias = append(categoriasRPT, categoriasSinteticasDesarrollo...)
	}
	motivos := []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
		{
			Clave:    string(motivoAltaContratacionTemporalDesarrollo),
			Etiqueta: "Sustitución temporal",
		},
	}
	return &catalogosAltaContratacionTemporalDesarrollo{
		Esquema:    esquemaCatalogosAltaContratacionTemporal,
		Centros:    centros,
		Categorias: categorias,
		Motivos:    motivos,
		Documentos: make([]opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo, 0),
	}, nil
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
	if o.catalogoDesarrollo != nil {
		return *o.catalogoDesarrollo, nil
	}
	catalogos, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		return catalogosAltaContratacionTemporalDesarrollo{}, errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles
	}
	return *catalogos, nil
}

func (o *origenConsultasContratacionTemporalDesarrollo) centroDeCatalogo(ref string) bool {
	if ref == centroAltaContratacionTemporalDesarrollo {
		return true
	}
	if o == nil {
		return false
	}
	catalogos, err := o.catalogosAlta()
	if err != nil {
		return false
	}
	for _, c := range catalogos.Centros {
		if c.Referencia == ref {
			return true
		}
	}
	return false
}

func (o *origenConsultasContratacionTemporalDesarrollo) categoriaDeCatalogo(ref string) bool {
	if ref == categoriaAltaContratacionTemporalDesarrollo {
		return true
	}
	if o == nil {
		return false
	}
	catalogos, err := o.catalogosAlta()
	if err != nil {
		return false
	}
	for _, cat := range catalogos.Categorias {
		if cat.Referencia == ref {
			return true
		}
	}
	return false
}

func (o *origenConsultasContratacionTemporalDesarrollo) referenciasCentros() []string {
	res := []string{centroAltaContratacionTemporalDesarrollo}
	if o == nil {
		return res
	}
	catalogos, err := o.catalogosAlta()
	if err != nil {
		return res
	}
	visto := map[string]bool{centroAltaContratacionTemporalDesarrollo: true}
	for _, c := range catalogos.Centros {
		if !visto[c.Referencia] {
			visto[c.Referencia] = true
			res = append(res, c.Referencia)
		}
	}
	return res
}

func (o *origenConsultasContratacionTemporalDesarrollo) referenciasCategorias() []string {
	res := []string{categoriaAltaContratacionTemporalDesarrollo}
	if o == nil {
		return res
	}
	catalogos, err := o.catalogosAlta()
	if err != nil {
		return res
	}
	visto := map[string]bool{categoriaAltaContratacionTemporalDesarrollo: true}
	for _, cat := range catalogos.Categorias {
		if !visto[cat.Referencia] {
			visto[cat.Referencia] = true
			res = append(res, cat.Referencia)
		}
	}
	return res
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

// etiquetasReferenciasCatalogosAlta devuelve el nombre con el que RRHH conoce
// una referencia de centro, categoría o contacto del catálogo de alta; cadena
// vacía si no la conoce. Solo sirve a la presentación (documentos): no cambia
// el catálogo ni concede nada.
func (o *origenConsultasContratacionTemporalDesarrollo) etiquetasReferenciasCatalogosAlta() informejuridico.EtiquetadorReferencias {
	return func(referencia string) string {
		if o == nil {
			return ""
		}
		if referencia == centroAltaContratacionTemporalDesarrollo {
			return "Centro solicitante"
		}
		if referencia == contactoAltaContratacionTemporalDesarrollo {
			return "Contacto del centro"
		}
		catalogos, err := o.catalogosAlta()
		if err != nil {
			if referencia == categoriaAltaContratacionTemporalDesarrollo {
				return "Categoría C2"
			}
			return ""
		}
		for _, centro := range catalogos.Centros {
			if centro.Referencia == referencia {
				return centro.Etiqueta
			}
			for _, contacto := range centro.Contactos {
				if contacto.Referencia == referencia {
					return contacto.Etiqueta
				}
			}
		}
		for _, categoria := range catalogos.Categorias {
			if categoria.Referencia == referencia {
				return categoria.Etiqueta
			}
		}
		if referencia == categoriaAltaContratacionTemporalDesarrollo {
			return "Categoría C2"
		}
		return ""
	}
}
