package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type manejadorPeticionCentroDesarrollo struct {
	proveedor *proveedorPeticionCentroDesarrollo
	servicio  *application.ServicioPeticionCentro
	bandeja   ports.ConsultaBandejaPeticionesCentro
}

func rutasHTTPPeticionCentroDesarrollo(p *proveedorPeticionCentroDesarrollo, r *postgresct.RepositorioPeticionesCentroPostgreSQL) ([]vechttp.RutaExacta, error) {
	s, err := application.NuevoServicioPeticionCentro(p, r, p.reloj)
	if err != nil {
		return nil, err
	}
	m := &manejadorPeticionCentroDesarrollo{proveedor: p, servicio: s, bandeja: r}
	return []vechttp.RutaExacta{{Ruta: rutaOperacionesPeticionCentro, Manejador: m}, {Ruta: rutaBandejaPeticionCentro, Manejador: m}, {Ruta: rutaContextoPeticionCentro, Manejador: m}}, nil
}

func (m *manejadorPeticionCentroDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	fallo := func(estado int, codigo string) {
		w.WriteHeader(estado)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.contratacion_temporal.peticion_centro.error." + codigo}})
	}
	if m == nil || m.proveedor == nil || m.servicio == nil || m.bandeja == nil || r == nil || r.URL == nil {
		fallo(503, "servicio_no_disponible")
		return
	}
	if !rutaPeticionCentroDesarrollo(r.URL.Path) || r.URL.RawPath != "" || r.URL.RawQuery != "" || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		fallo(400, "solicitud_invalida")
		return
	}
	escritura := r.URL.Path == rutaOperacionesPeticionCentro
	metodo := http.MethodGet
	if escritura {
		metodo = http.MethodPost
	}
	if r.Method != metodo {
		w.Header().Set("Allow", metodo)
		fallo(405, "metodo_no_permitido")
		return
	}
	a, err := m.proveedor.identidad(r.Context())
	if err != nil {
		fallo(403, "operacion_denegada")
		return
	}
	responder := func(data any) { _ = json.NewEncoder(w).Encode(map[string]any{"data": data}) }
	if !escritura {
		if r.ContentLength != 0 {
			fallo(400, "solicitud_invalida")
			return
		}
		if r.URL.Path == rutaContextoPeticionCentro {
			data, err := m.contexto(r, a)
			if err != nil {
				fallo(503, "servicio_no_disponible")
				return
			}
			responder(data)
			return
		}
		datos, err := m.bandeja.ListarPeticiones(r.Context(), a.actor)
		if err != nil {
			fallo(503, "servicio_no_disponible")
			return
		}
		responder(map[string]any{"peticiones": datos, "limite": 50})
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 0 && (len(params) != 1 || !strings.EqualFold(params["charset"], "utf-8")) || r.ContentLength > 64*1024 || r.Body == nil {
		fallo(400, "solicitud_invalida")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64*1024))
	if err != nil || validarClavesJSONUnicas(b) != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	defer clear(b)
	var comando ports.ComandoPeticionCentro
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&comando) != nil || d.Decode(&struct{}{}) != io.EOF || comando.Validar() != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	if comando.Operacion == ports.OperacionPresentarPeticionCentro && a.principal.Roles[0] != "solicitante_centro" || comando.Operacion == ports.OperacionRatificarPeticionCentro && a.principal.Roles[0] != "ratificador_centro" {
		fallo(403, "operacion_denegada")
		return
	}
	recibo, err := m.servicio.Ejecutar(r.Context(), comando)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPeticionCentroInvalida):
			fallo(400, "solicitud_invalida")
		case errors.Is(err, domain.ErrRatificacionCentroDenegada):
			fallo(403, "operacion_denegada")
		case errors.Is(err, ports.ErrClavePeticionCentroUsada), errors.Is(err, domain.ErrVersionPeticionCentroEnConflicto):
			fallo(409, "peticion_en_conflicto")
		default:
			fallo(503, "servicio_no_disponible")
		}
		return
	}
	responder(recibo)
}

// Los nombres sirven para presentar la identidad ya autenticada. No vuelven
// en el comando ni se utilizan como permiso para ratificar.
func (m *manejadorPeticionCentroDesarrollo) contexto(r *http.Request, a *identidadPeticionCentroDesarrollo) (any, error) {
	c, err := m.proveedor.catalogoParaActor(r.Context(), a.actor)
	if err != nil {
		return nil, err
	}
	etiquetas := make(map[string]string, len(c.Entradas))
	for _, e := range c.Entradas {
		etiquetas[e.Clave] = e.Etiqueta
	}
	catalogos, err := nuevoOrigenConsultasContratacionTemporalDesarrollo().catalogosAlta()
	if err != nil {
		return nil, err
	}
	catalogos.Centros = []centroCatalogosAltaContratacionTemporalDesarrollo{{Referencia: a.actor.CentroRef, Etiqueta: etiquetas[a.actor.CentroRef], Contactos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{{Referencia: contactoAltaContratacionTemporalDesarrollo, Etiqueta: "Contacto sintético del centro"}}}}
	actor := map[string]any{"referencia": a.actor.ActorRef, "nombre": a.principal.DisplayName, "cargo": etiquetas[a.actor.PuestoRef], "centro": etiquetas[a.actor.CentroRef], "puede_presentar": a.principal.Roles[0] == "solicitante_centro", "puede_ratificar": a.principal.Roles[0] == "ratificador_centro"}
	if rat, ok := m.proveedor.actores[a.adscripcion.RatificadorSubject]; ok && actorPeticionCentroPerteneceCatalogo(c, rat.actor) {
		actor["ratificador_nombre"] = rat.principal.DisplayName
		actor["ratificador_cargo"] = etiquetas[rat.actor.PuestoRef]
	}
	// Solo etiquetas de la relación nominal configurada, no un directorio de
	// usuarios del centro. No se aceptan de vuelta como identidad ni permiso.
	intervinientes := make(map[string]map[string]string)
	for _, otro := range m.proveedor.actores {
		if otro.actor.CentroRef == a.actor.CentroRef && actorPeticionCentroPerteneceCatalogo(c, otro.actor) && (otro.actor == a.actor ||
			a.adscripcion.RatificadorSubject == otro.principal.ID ||
			otro.adscripcion.RatificadorSubject == a.principal.ID) {
			intervinientes[otro.actor.ActorRef] = map[string]string{"nombre": otro.principal.DisplayName, "cargo": etiquetas[otro.actor.PuestoRef], "puesto_ref": otro.actor.PuestoRef}
		}
	}
	return map[string]any{"actor": actor, "intervinientes": intervinientes, "catalogos": catalogos, "entorno": "desarrollo_sintetico", "catalogo_revision": c.Revision}, nil
}
