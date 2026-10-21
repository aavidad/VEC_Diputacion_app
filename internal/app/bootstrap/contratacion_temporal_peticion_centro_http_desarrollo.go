package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/telemetria"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type manejadorPeticionCentroDesarrollo struct {
	proveedor *proveedorPeticionCentroDesarrollo
	servicio  *application.ServicioPeticionCentro
	bandeja   ports.ConsultaBandejaPeticionesCentro
	catalogo  *catalogosAltaContratacionTemporalDesarrollo
}

type vinculoCategoriaRPTCentroJSON struct {
	CategoriaRef      string `json:"categoria_ref"`
	RPTCategoriaClave string `json:"rpt_categoria_clave"`
}

type vinculoCentroRPTCentroJSON struct {
	CentroRef       string `json:"centro_ref"`
	RPTCentroCodigo string `json:"rpt_centro_codigo"`
}

type vinculosRPTCentroJSON struct {
	Categorias []vinculoCategoriaRPTCentroJSON `json:"categorias"`
	Centros    []vinculoCentroRPTCentroJSON    `json:"centros"`
}

func rutasHTTPPeticionCentroDesarrollo(p *proveedorPeticionCentroDesarrollo, r *postgresct.RepositorioPeticionesCentroPostgreSQL, catalogos ...*catalogosAltaContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	catalogo, err := catalogoAnalisisDesarrollo(catalogos...)
	if err != nil {
		return nil, err
	}
	s, err := application.NuevoServicioPeticionCentro(p, r, p.reloj, catalogo)
	if err != nil {
		return nil, err
	}
	if p != nil && p.alta != nil && p.alta.servicio != nil {
		// El aviso de fuente de necesidades no configurada sale una sola vez,
		// donde la composición decide la fuente (avisarFuenteNecesidadesAlta-
		// NoConfigurada); montar estas rutas no lo repite.
		if err := s.ConfigurarPreparadorNecesidad(p.alta.servicio); err != nil {
			return nil, err
		}
	}
	m := &manejadorPeticionCentroDesarrollo{proveedor: p, servicio: s, bandeja: r, catalogo: catalogo}
	return []vechttp.RutaExacta{{Ruta: rutaOperacionesPeticionCentro, Manejador: m}, {Ruta: rutaBandejaPeticionCentro, Manejador: m}, {Ruta: rutaContextoPeticionCentro, Manejador: m}}, nil
}

// avisarFuenteNecesidadesAltaNoConfigurada es el único emisor del aviso: la
// composición lo llama una vez por arranque, en la rama que deja el alta de
// necesidades sin fuente.
func avisarFuenteNecesidadesAltaNoConfigurada(rutaRPT string) {
	avisarErrorCatalogadoNecesidadesAltaArranque(
		errFuenteNecesidadesAltaNoConfigurada,
		"VEC_CT_NECESIDADES_ALTA_SOURCE_PATH",
	)
	if strings.TrimSpace(rutaRPT) == "" {
		avisarErrorCatalogadoNecesidadesAltaArranque(
			errCatalogoRPTNecesidadesAltaNoConfigurado,
			"VEC_RPT_CATALOGO_PATH",
		)
	}
}

func avisarErrorCatalogadoNecesidadesAltaArranque(err error, requisito string) {
	slog.Warn(
		"fuente de necesidades de alta no configurada al arrancar",
		"componente", ComponenteFalloArranque(err),
		"causa", "fuente_necesidades_no_disponible",
		"requisito", requisito,
		"error.type", telemetria.ClaseErrorArranque(err),
		"error.message", telemetria.MensajeErrorArranque(err),
	)
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
		fallo(falloPeticionCentroDesarrollo(r, err))
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
				fallo(falloPeticionCentroDesarrollo(r, err))
				return
			}
			responder(data)
			return
		}
		datos, err := m.bandeja.ListarPeticiones(r.Context(), a.actor)
		if err != nil {
			fallo(falloPeticionCentroDesarrollo(r, err))
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
	if d.Decode(&comando) != nil || d.Decode(&struct{}{}) != io.EOF ||
		validarFormaNecesidadPeticionCentroHTTP(b) != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	if comando.NecesidadEntrada != nil && comando.Solicitud != nil {
		comando.NecesidadEntrada.Periodo = comando.Solicitud.Periodo
	}
	if comando.Validar() != nil {
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
		case errors.Is(err, domain.ErrPeticionCentroInvalida), errors.Is(err, application.ErrSolicitudRegistroInvalida):
			fallo(400, "solicitud_invalida")
		case errors.Is(err, ports.ErrClavePeticionCentroUsada), errors.Is(err, domain.ErrVersionPeticionCentroEnConflicto):
			fallo(409, "peticion_en_conflicto")
		default:
			fallo(falloPeticionCentroDesarrollo(r, err))
		}
		return
	}
	responder(recibo)
}

func validarFormaNecesidadPeticionCentroHTTP(contenido []byte) error {
	var sobre struct {
		Necesidad json.RawMessage `json:"necesidad"`
	}
	if err := json.Unmarshal(contenido, &sobre); err != nil {
		return err
	}
	if len(sobre.Necesidad) == 0 {
		return nil // El caso de uso valida el sobre legado o el esquema v3.
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(sobre.Necesidad, &campos) != nil || len(campos) != 7 {
		return domain.ErrPeticionCentroInvalida
	}
	for _, clave := range []string{"esquema", "catalogo_ref", "catalogo_version", "catalogo_huella_sha256", "causa_clave", "jornada_minutos", "campos"} {
		if _, ok := campos[clave]; !ok {
			return domain.ErrPeticionCentroInvalida
		}
	}
	return nil
}

// Las dependencias pueden envolver una denegación genérica. Se comprueba
// primero su causa concreta y solo se informa una denegación real como 403.
// El registro nunca contiene err: podría incluir referencias de la petición.
func falloPeticionCentroDesarrollo(r *http.Request, err error) (int, string) {
	// identidad() normaliza el contexto vencido como denegación. La petición
	// conserva la causa para responder 503 sin abrir la operación.
	causa := causaFalloPeticionCentroDesarrollo(errors.Join(err, r.Context().Err()))
	if causa == "autorizacion_denegada" {
		slog.Warn("petición de centro denegada", "ruta", r.URL.Path, "metodo", r.Method, "causa", causa)
		return http.StatusForbidden, "operacion_denegada"
	}
	slog.Error("petición de centro no disponible", "ruta", r.URL.Path, "metodo", r.Method, "causa", causa)
	return http.StatusServiceUnavailable, "servicio_no_disponible"
}

func causaFalloPeticionCentroDesarrollo(err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "peticion_cancelada_o_vencida"
	case errors.Is(err, personalports.ErrCambioOrganizacionNoDisponible), errors.Is(err, vecports.ErrCatalogoNoEncontrado):
		return "catalogo_organizacion_no_disponible"
	}
	// La recepción RRHH ya cataloga los fallos del autorizador V3 y del
	// registro. Sus causas de indisponibilidad preceden a la denegación.
	if causa := causaFalloEntregaPeticionDesarrollo(err); causa != "no_clasificada" {
		return causa
	}
	if errors.Is(err, domain.ErrRatificacionCentroDenegada) {
		return "autorizacion_denegada"
	}
	return "no_clasificada"
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
	if m.catalogo == nil {
		return nil, errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles
	}
	contextoCatalogos, err := datosCatalogosAltaV2Desde(*m.catalogo)
	if err != nil {
		return nil, err
	}
	catalogos := *m.catalogo
	catalogos.Centros = []centroCatalogosAltaContratacionTemporalDesarrollo{{Referencia: a.actor.CentroRef, Etiqueta: etiquetas[a.actor.CentroRef], Contactos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{{Referencia: contactoAltaContratacionTemporalDesarrollo, Etiqueta: etiquetaContactoAltaContratacionTemporalDesarrollo}}}}
	contextoCatalogos.Centros = catalogos.Centros
	contextoCatalogos.NumeroExpedienteMOAD = nil
	contextoCatalogos.PreparacionVias = nil
	vinculos := vinculosRPTCentroJSON{Categorias: []vinculoCategoriaRPTCentroJSON{}, Centros: []vinculoCentroRPTCentroJSON{}}
	for _, categoria := range m.catalogo.Categorias {
		if categoria.rptClave != "" {
			vinculos.Categorias = append(vinculos.Categorias, vinculoCategoriaRPTCentroJSON{categoria.Referencia, categoria.rptClave})
		}
	}
	if codigo := m.catalogo.centrosCodigoFuente[a.actor.CentroRef]; codigo != "" {
		vinculos.Centros = append(vinculos.Centros, vinculoCentroRPTCentroJSON{a.actor.CentroRef, codigo})
	}
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
	return map[string]any{"actor": actor, "intervinientes": intervinientes, "catalogos": contextoCatalogos,
		"vinculos_rpt": vinculos, "jornada_referencia_minutos": contextoCatalogos.Necesidades.JornadaReferenciaMinutos,
		"entorno": "desarrollo_sintetico", "catalogo_revision": c.Revision}, nil
}
