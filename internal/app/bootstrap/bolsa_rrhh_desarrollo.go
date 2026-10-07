package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const (
	rutaBolsasRRHHDesarrollo        = "/api/vec/bolsa/bolsas"
	prefijoCandidatosRRHHDesarrollo = rutaBolsasRRHHDesarrollo
	// B12/estadísticas (Peticion.pdf p.3 «cuadro de control» y p.1 punto 7
	// «estadísticas y explotación»): lectura agregada del mismo conjunto que
	// sirve el cuadro, sin consultas ni tablas propias.
	rutaEstadisticasBolsaRRHHDesarrollo = "/api/vec/bolsa/estadisticas"
	rutaAvisosBolsaRRHHDesarrollo       = "/api/vec/bolsa/avisos"
)

type datasetBolsasRRHHDesarrollo struct {
	GeneradoEn string `json:"generado_en"`
	Bolsas     []struct {
		Referencia          string                      `json:"bolsa_ref"`
		CategoriaRef        string                      `json:"categoria_ref"`
		Categoria           string                      `json:"categoria"`
		TipoLista           string                      `json:"tipo_lista"`
		VigenteDesde        string                      `json:"vigente_desde"`
		VigenteHasta        *string                     `json:"vigente_hasta"`
		LlamamientosEnCurso int                         `json:"llamamientos_en_curso"`
		PoliticaOrden       politicaOrdenRRHHDesarrollo `json:"politica_orden"`
	} `json:"bolsas"`
	Candidaturas []struct {
		Referencia  string  `json:"candidatura_ref"`
		BolsaRef    string  `json:"bolsa_ref"`
		Orden       *int    `json:"orden"`
		OrdenActa   int     `json:"orden_acta"`
		RazonOrden  string  `json:"razon_orden"`
		Nombre      string  `json:"nombre_visible"`
		Documento   string  `json:"documento_enmascarado"`
		Estado      string  `json:"estado_clave"`
		EstadoDesde string  `json:"estado_desde"`
		Disponible  *string `json:"disponible_desde"`
	} `json:"candidaturas"`
	Llamamientos []struct {
		Referencia  string `json:"llamamiento_ref"`
		Candidatura string `json:"candidatura_ref"`
		Comunicado  string `json:"comunicado_en"`
		Canal       string `json:"canal"`
		Resultado   string `json:"resultado"`
	} `json:"llamamientos"`
	Contactos []dominiobolsa.ContactoParticipacion `json:"contactos"`
	// Marcas (Bolsa 000041) por participación; nil si no están compuestas.
	Marcas map[string]dominiobolsa.MarcasParticipacion `json:"-"`
	// PoliticaIntentos, si el catálogo la tiene, permite rotular la baja
	// propuesta por intentos agotados como «en revisión».
	PoliticaIntentos *dominiobolsa.PoliticaIntentosTelefonicos `json:"-"`
}

type politicaOrdenRRHHDesarrollo struct {
	Referencia, Criterio, TipoLista, Reposicion, Rotulo, Actor, VigenteDesde string
	Version                                                                  uint64
	Provisional                                                              bool
}

type bolsasRRHHDesarrollo struct {
	cargar func(context.Context) (datasetBolsasRRHHDesarrollo, error)
	// resumen y cargarBolsa acotan la lectura (ver alcanceCargaBolsasRRHH);
	// si faltan, se usa cargar con todo el detalle.
	resumen     func(context.Context) (datasetBolsasRRHHDesarrollo, error)
	cargarBolsa func(context.Context, string) (datasetBolsasRRHHDesarrollo, error)
	mutar       http.Handler
	invalidar   func()
	contactos   lectorContactosBolsaDesarrollo
	avisos      *bolsaapplication.ServicioAvisosRRHH
}

type lectorContactosBolsaDesarrollo interface {
	ListarContactosBolsa(context.Context, string, string, int) (puertosbolsa.PaginaContactosParticipacion, error)
}

func nuevasRutasBolsasRRHHDesarrollo(cfg config.Config) ([]vechttp.RutaExacta, []vechttp.RutaColeccion, error) {
	return nuevasRutasBolsasRRHHDesarrolloConFuente(cfg, nil)
}

func nuevasRutasBolsasRRHHDesarrolloConFuente(_ config.Config, fuente *fuenteConstituidaRRHHDesarrollo, mutadores ...http.Handler) ([]vechttp.RutaExacta, []vechttp.RutaColeccion, error) {
	var cargar func(context.Context) (datasetBolsasRRHHDesarrollo, error)
	var invalidar func()
	if fuente != nil {
		cargar = fuente.cargar
		invalidar = fuente.invalidar
	}
	manejador := nuevoManejadorBolsasRRHHDesarrollo(cargar)
	if fuente != nil {
		manejador.avisos = fuente.avisos
		manejador.resumen = fuente.cargarResumen
		manejador.cargarBolsa = fuente.cargarBolsa
	}
	if len(mutadores) == 1 {
		manejador.mutar = mutadores[0]
		manejador.invalidar = invalidar
		manejador.contactos, _ = mutadores[0].(lectorContactosBolsaDesarrollo)
	}
	return []vechttp.RutaExacta{
			{Ruta: rutaBolsasRRHHDesarrollo, Manejador: manejador},
			{Ruta: rutaEstadisticasBolsaRRHHDesarrollo, Manejador: manejador},
			{Ruta: rutaAvisosBolsaRRHHDesarrollo, Manejador: manejador},
			{Ruta: bolsahttp.RutaSolicitudesDocumentalesPendientesRRHH, Manejador: manejador},
		},
		[]vechttp.RutaColeccion{{Prefijo: prefijoCandidatosRRHHDesarrollo, Manejador: manejador}}, nil
}

func nuevoManejadorBolsasRRHHDesarrollo(cargar func(context.Context) (datasetBolsasRRHHDesarrollo, error)) *bolsasRRHHDesarrollo {
	if cargar == nil {
		cargar = func(context.Context) (datasetBolsasRRHHDesarrollo, error) {
			return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
		}
	}
	return &bolsasRRHHDesarrollo{cargar: cargar}
}

func (h *bolsasRRHHDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	esMutacionSituacion := h != nil && h.mutar != nil && r != nil && r.Method == http.MethodPost
	if esMutacionSituacion {
		_, _, esMutacionSituacion = bolsahttp.ReferenciasRutaSituacionParticipacion(r)
	}
	esOperacion := h != nil && h.mutar != nil && r != nil && (r.Method == http.MethodPost || r.Method == http.MethodGet)
	if esOperacion {
		_, _, esOperacion = bolsahttp.ReferenciasRutaOperacionesSituacion(r)
	}
	esContacto := h != nil && h.mutar != nil && r != nil && (r.Method == http.MethodPost || r.Method == http.MethodGet)
	if esContacto {
		_, _, esContacto = bolsahttp.ReferenciasRutaContactosParticipacion(r)
	}
	esDatosContacto := h != nil && h.mutar != nil && r != nil && (r.Method == http.MethodPost || r.Method == http.MethodGet)
	if esDatosContacto {
		_, _, _, esDatosContacto = bolsahttp.ReferenciasRutaDatosContactoParticipacion(r)
	}
	esContratos := h != nil && h.mutar != nil && r != nil && r.Method == http.MethodGet
	if esContratos {
		_, _, esContratos = bolsahttp.ReferenciasRutaContratosParticipacion(r)
	}
	esSolicitudesDocumentales := h != nil && h.mutar != nil && r != nil && r.Method == http.MethodGet && r.URL != nil && r.URL.Path == bolsahttp.RutaSolicitudesDocumentalesPendientesRRHH
	esSancion := h != nil && h.mutar != nil && r != nil && (r.Method == http.MethodPost || r.Method == http.MethodGet)
	if esSancion {
		_, _, _, esSancion = bolsahttp.ReferenciasRutaSancionesParticipacion(r)
	}
	cabeceras := http.Header(nil)
	if r != nil {
		cabeceras = r.Header
		if esMutacionSituacion || esOperacion || esContacto || esDatosContacto || esContratos || esSancion || esSolicitudesDocumentales {
			cabeceras = r.Header.Clone()
			cabeceras.Del("Idempotency-Key")
		}
	}
	if h == nil || r == nil || r.URL == nil || r.URL.RawPath != "" || len(r.TransferEncoding) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(cabeceras) {
		responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	if esMutacionSituacion || esOperacion || esContacto || esDatosContacto || esContratos || esSancion || esSolicitudesDocumentales {
		h.mutar.ServeHTTP(w, r)
		if h.invalidar != nil {
			h.invalidar()
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderBolsaRRHHDesarrollo(w, http.StatusMethodNotAllowed, map[string]string{"codigo": "metodo_no_permitido"})
		return
	}
	if r.URL.Path == rutaEstadisticasBolsaRRHHDesarrollo {
		if r.URL.RawQuery != "" || r.ContentLength != 0 {
			responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
			return
		}
		vista, ok := h.vistaResumen(w, r)
		if !ok {
			return
		}
		responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": vista.respuestaEstadisticas()}, r.Method == http.MethodHead)
		return
	}
	if r.URL.Path == rutaAvisosBolsaRRHHDesarrollo {
		h.responderAvisos(w, r)
		return
	}
	if r.URL.Path == rutaBolsasRRHHDesarrollo {
		if r.URL.RawQuery != "" || r.ContentLength != 0 {
			responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
			return
		}
		vista, ok := h.vistaResumen(w, r)
		if !ok {
			return
		}
		responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": vista.respuestaBolsas()}, r.Method == http.MethodHead)
		return
	}
	bolsaRef, ok := referenciaBolsaCandidatos(r.URL.Path)
	if !ok || r.ContentLength != 0 {
		responderBolsaRRHHDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
		return
	}
	consulta, ok := consultaCandidatos(r.URL.RawQuery)
	if !ok {
		responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	vista, ok := h.vistaBolsa(w, r, bolsaRef)
	if !ok {
		return
	}
	if h.contactos == nil && h.mutar != nil {
		responderFalloBolsaRRHHDesarrollo(w, r, "contactos", ErrComposicionDesarrolloIncompleta)
		return
	}
	if h.contactos != nil {
		if err := h.cargarContactos(r.Context(), vista, bolsaRef); err != nil {
			responderFalloBolsaRRHHDesarrollo(w, r, "contactos", err)
			return
		}
	}
	respuesta, encontrada := vista.respuestaCandidatos(bolsaRef, consulta)
	if !encontrada {
		responderBolsaRRHHDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
		return
	}
	responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": respuesta}, r.Method == http.MethodHead)
}

func (h *bolsasRRHHDesarrollo) responderAvisos(w http.ResponseWriter, r *http.Request) {
	consulta, ok := consultaAvisosRRHH(r.URL.RawQuery)
	if !ok || r.ContentLength != 0 {
		responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	if h == nil || h.avisos == nil {
		responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		return
	}
	pagina, err := h.avisos.Consultar(r.Context(), consulta)
	if err != nil {
		responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		return
	}
	items := make([]map[string]any, 0, len(pagina.Avisos))
	for _, aviso := range pagina.Avisos {
		items = append(items, map[string]any{
			"tipo": aviso.Tipo, "bolsa": aviso.BolsaRef, "referencia": aviso.Referencia,
			"detalle": aviso.Detalle, "fecha": aviso.Fecha.UTC().Format(time.RFC3339Nano),
		})
	}
	responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": "vec.bolsa.rrhh.avisos.v1", "generado_en": pagina.GeneradaEn.UTC().Format(time.RFC3339Nano),
		"provisionalidad": pagina.Provisionalidad, "items": items,
		"conteos":    pagina.Conteos,
		"paginacion": map[string]any{"limite": consulta.Limite, "desde": pagina.Desde, "hasta": pagina.Hasta, "total": pagina.Total, "cursor_siguiente": pagina.CursorSiguiente},
	}}, r.Method == http.MethodHead)
}

func consultaAvisosRRHH(cruda string) (bolsaapplication.ConsultaAvisos, bool) {
	resultado := bolsaapplication.ConsultaAvisos{Limite: 20}
	if cruda == "" {
		return resultado, true
	}
	valores, err := url.ParseQuery(cruda)
	if err != nil || len(valores) > 2 {
		return resultado, false
	}
	for clave, lista := range valores {
		if len(lista) != 1 {
			return resultado, false
		}
		valor := strings.TrimSpace(lista[0])
		switch clave {
		case "cursor":
			if valor == "" || len(valor) > 512 {
				return resultado, false
			}
			resultado.Cursor = valor
		case "limite":
			limite, e := strconv.Atoi(valor)
			if e != nil || limite < 1 || limite > 100 {
				return resultado, false
			}
			resultado.Limite = limite
		default:
			return resultado, false
		}
	}
	return resultado, true
}

// cargarContactos lee los contactos de la bolsa por páginas. Cada página
// consume su propia autorización V3, así que una página incompleta es la
// última: pedir otra solo para recibirla vacía duplicaba la autorización.
func (h *bolsasRRHHDesarrollo) cargarContactos(ctx context.Context, vista *bolsasRRHHDesarrolloDatos, bolsa string) error {
	const porPagina = 100
	cursor := ""
	for pagina := 0; pagina < 100; pagina++ {
		p, err := h.contactos.ListarContactosBolsa(ctx, bolsa, cursor, porPagina)
		if err != nil {
			return err
		}
		vista.datos.Contactos = append(vista.datos.Contactos, p.Contactos...)
		if p.CursorSiguiente == "" || len(p.Contactos) < porPagina {
			return nil
		}
		if p.CursorSiguiente == cursor {
			return errContactosBolsaRRHHIncoherentes
		}
		cursor = p.CursorSiguiente
	}
	return errContactosBolsaRRHHIncoherentes
}

func (h *bolsasRRHHDesarrollo) vistaDurable(w http.ResponseWriter, r *http.Request) (*bolsasRRHHDesarrolloDatos, bool) {
	datos, err := h.cargar(r.Context())
	if err != nil {
		responderFalloBolsaRRHHDesarrollo(w, r, "carga", err)
		return nil, false
	}
	return &bolsasRRHHDesarrolloDatos{datos: datos}, true
}

// vistaResumen sirve el cuadro y las estadísticas: cada petición hace su
// propia lectura, sin compartirla con otras peticiones.
func (h *bolsasRRHHDesarrollo) vistaResumen(w http.ResponseWriter, r *http.Request) (*bolsasRRHHDesarrolloDatos, bool) {
	if h.resumen == nil {
		return h.vistaDurable(w, r)
	}
	datos, err := h.resumen(r.Context())
	if err != nil {
		responderFalloBolsaRRHHDesarrollo(w, r, "resumen", err)
		return nil, false
	}
	return &bolsasRRHHDesarrolloDatos{datos: datos}, true
}

// vistaBolsa lee solo la bolsa pedida, con nombres y marcas.
func (h *bolsasRRHHDesarrollo) vistaBolsa(w http.ResponseWriter, r *http.Request, bolsaRef string) (*bolsasRRHHDesarrolloDatos, bool) {
	if h.cargarBolsa == nil {
		return h.vistaDurable(w, r)
	}
	datos, err := h.cargarBolsa(r.Context(), bolsaRef)
	if err != nil {
		responderFalloBolsaRRHHDesarrollo(w, r, "bolsa", err)
		return nil, false
	}
	return &bolsasRRHHDesarrolloDatos{datos: datos}, true
}

type consultaCandidatosRRHH struct {
	estado, texto, cursor string
	limite                int
}

func consultaCandidatos(cruda string) (consultaCandidatosRRHH, bool) {
	resultado := consultaCandidatosRRHH{limite: 50}
	if cruda == "" {
		return resultado, true
	}
	valores, err := url.ParseQuery(cruda)
	if err != nil || len(valores) > 4 {
		return resultado, false
	}
	for clave, valoresClave := range valores {
		if len(valoresClave) != 1 {
			return resultado, false
		}
		valor := strings.TrimSpace(valoresClave[0])
		switch clave {
		case "estado":
			if valor != "" && !estadoBolsaVisible(valor) {
				return resultado, false
			}
			resultado.estado = valor
		case "texto":
			if len(valor) > 100 {
				return resultado, false
			}
			resultado.texto = strings.ToLower(valor)
		case "cursor":
			if len(valor) > 256 {
				return resultado, false
			}
			resultado.cursor = valor
		case "limite":
			n, e := strconv.Atoi(valor)
			if e != nil || n < 1 || n > 100 {
				return resultado, false
			}
			resultado.limite = n
		default:
			return resultado, false
		}
	}
	return resultado, true
}

func rutaBolsasCandidatosRRHHDesarrollo(ruta string) bool {
	_, valida := referenciaBolsaCandidatos(ruta)
	return valida
}

func rutaBolsasOperacionesRRHHDesarrollo(ruta string) bool {
	if !strings.HasPrefix(ruta, rutaBolsasRRHHDesarrollo+"/") {
		return false
	}
	partes := strings.Split(strings.TrimPrefix(ruta, rutaBolsasRRHHDesarrollo+"/"), "/")
	return len(partes) == 4 && partes[0] != "" && partes[1] == "candidatos" && partes[2] != "" && (partes[3] == "operaciones" || partes[3] == "contratos")
}

func referenciaBolsaCandidatos(ruta string) (string, bool) {
	const sufijo = "/candidatos"
	if !strings.HasPrefix(ruta, prefijoCandidatosRRHHDesarrollo+"/") || !strings.HasSuffix(ruta, sufijo) {
		return "", false
	}
	ref := strings.TrimSuffix(strings.TrimPrefix(ruta, prefijoCandidatosRRHHDesarrollo+"/"), sufijo)
	return ref, ref != "" && !strings.Contains(ref, "/") && ref == strings.TrimSpace(ref)
}

type bolsasRRHHDesarrolloDatos struct{ datos datasetBolsasRRHHDesarrollo }

func (h *bolsasRRHHDesarrolloDatos) respuestaBolsas() map[string]any {
	bolsas := make([]map[string]any, 0, len(h.datos.Bolsas))
	conteos := h.contarEstadosPorBolsa()
	for _, bolsa := range h.datos.Bolsas {
		conteo := conteos[bolsa.Referencia]
		if conteo == nil {
			conteo = mapaEstadosVacio()
		}
		bolsas = append(bolsas, salidaBolsaRRHH(bolsa.Referencia, bolsa.CategoriaRef, bolsa.Categoria, bolsa.TipoLista, bolsa.VigenteDesde, bolsa.VigenteHasta, conteo, bolsa.LlamamientosEnCurso, bolsa.PoliticaOrden))
	}
	return map[string]any{"esquema": "vec.bolsa.rrhh.bolsas.v1", "generado_en": instanteBolsasRRHH(h.datos.GeneradoEn), "bolsas": bolsas}
}

func (h *bolsasRRHHDesarrolloDatos) respuestaCandidatos(ref string, consulta consultaCandidatosRRHH) (map[string]any, bool) {
	var bolsa *struct {
		Referencia          string                      `json:"bolsa_ref"`
		CategoriaRef        string                      `json:"categoria_ref"`
		Categoria           string                      `json:"categoria"`
		TipoLista           string                      `json:"tipo_lista"`
		VigenteDesde        string                      `json:"vigente_desde"`
		VigenteHasta        *string                     `json:"vigente_hasta"`
		LlamamientosEnCurso int                         `json:"llamamientos_en_curso"`
		PoliticaOrden       politicaOrdenRRHHDesarrollo `json:"politica_orden"`
	}
	for indice := range h.datos.Bolsas {
		if h.datos.Bolsas[indice].Referencia == ref {
			bolsa = &h.datos.Bolsas[indice]
			break
		}
	}
	if bolsa == nil {
		return nil, false
	}
	todos := make([]bolsaapplication.CandidatoTurno, 0)
	contactosBolsa := make([]dominiobolsa.ContactoParticipacion, 0)
	for _, candidata := range h.datos.Candidaturas {
		if candidata.BolsaRef == ref {
			todos = append(todos, bolsaapplication.CandidatoTurno{ParticipacionRef: candidata.Referencia, NombreVisible: candidata.Nombre, Orden: candidata.Orden, OrdenActa: candidata.OrdenActa, Estado: candidata.Estado})
		}
	}
	for _, contacto := range h.datos.Contactos {
		if contacto.BolsaRef == ref {
			contactosBolsa = append(contactosBolsa, contacto)
		}
	}
	turno := bolsaapplication.ProyectarTurnoCandidatos(bolsa.PoliticaOrden.Referencia, bolsa.PoliticaOrden.Version, bolsa.PoliticaOrden.Provisional, todos, contactosBolsa)
	candidatas := make([]bolsaapplication.CandidatoTurno, 0)
	indices := make(map[string]int)
	for indice, candidata := range h.datos.Candidaturas {
		if candidata.BolsaRef != ref || (consulta.estado != "" && estadoBolsaCanonico(candidata.Estado) != consulta.estado) || (consulta.texto != "" && !strings.Contains(strings.ToLower(candidata.Nombre+" "+candidata.Documento), consulta.texto)) {
			continue
		}
		candidatas = append(candidatas, bolsaapplication.CandidatoTurno{ParticipacionRef: candidata.Referencia, NombreVisible: candidata.Nombre, Orden: candidata.Orden, OrdenActa: candidata.OrdenActa, Estado: candidata.Estado})
		indices[candidata.Referencia] = indice
	}
	candidatas = bolsaapplication.OrdenarCandidatosTurno(candidatas)
	inicio := 0
	if consulta.cursor != "" {
		for i, candidata := range candidatas {
			if candidata.ParticipacionRef == consulta.cursor {
				inicio = i + 1
				break
			}
		}
		if inicio == 0 {
			return nil, false
		}
	}
	fin := inicio + consulta.limite
	if fin > len(candidatas) {
		fin = len(candidatas)
	}
	salida := make([]map[string]any, 0, fin-inicio)
	for _, candidata := range candidatas[inicio:fin] {
		salida = append(salida, h.salidaCandidata(h.datos.Candidaturas[indices[candidata.ParticipacionRef]]))
	}
	conteo := mapaEstadosVacio()
	for _, candidata := range h.datos.Candidaturas {
		if candidata.BolsaRef == ref {
			conteo[estadoBolsaCanonico(candidata.Estado)]++
		}
	}
	hayMas := fin < len(candidatas)
	var siguiente any = nil
	if hayMas {
		siguiente = candidatas[fin-1].ParticipacionRef
	}
	contactos := make([]map[string]any, 0)
	for _, c := range h.datos.Contactos {
		if c.BolsaRef == ref {
			contactos = append(contactos, map[string]any{"contacto_ref": c.ContactoRef, "participacion_ref": c.ParticipacionRef, "llamamiento_ref": nuloBootstrap(c.LlamamientoRef), "canal": c.Canal, "instante": c.Instante.UTC().Format(time.RFC3339Nano), "actor_ref": c.Actor, "resultado": c.Resultado, "anotacion": c.Anotacion})
		}
	}
	return map[string]any{"esquema": "vec.bolsa.rrhh.candidatos.v2", "generado_en": instanteBolsasRRHH(h.datos.GeneradoEn), "bolsa": salidaBolsaRRHH(bolsa.Referencia, bolsa.CategoriaRef, bolsa.Categoria, bolsa.TipoLista, bolsa.VigenteDesde, bolsa.VigenteHasta, conteo, bolsa.LlamamientosEnCurso, bolsa.PoliticaOrden), "candidatos": salida, "contactos": contactos, "turno": salidaTurnoCandidatos(turno), "hay_mas": hayMas, "cursor_siguiente": siguiente}, true
}

func salidaTurnoCandidatos(turno bolsaapplication.TurnoCandidatos) map[string]any {
	var ultimo any
	if turno.UltimoLlamado != nil {
		candidato, contacto := turno.UltimoLlamado.Candidato, turno.UltimoLlamado.Contacto
		ultimo = map[string]any{"participacion_ref": candidato.ParticipacionRef, "nombre_visible": candidato.NombreVisible, "orden": candidato.Orden, "comunicado_en": contacto.Instante.UTC().Format(time.RFC3339Nano), "canal": contacto.Canal, "resultado": contacto.Resultado}
	}
	var siguiente any
	if turno.Siguiente != nil {
		candidato := turno.Siguiente
		siguiente = map[string]any{"participacion_ref": candidato.ParticipacionRef, "nombre_visible": candidato.NombreVisible, "orden": candidato.Orden}
	}
	return map[string]any{"politica_ref": turno.PoliticaRef, "politica_version": turno.PoliticaVersion, "provisional": turno.Provisional, "ultimo_llamado": ultimo, "siguiente": siguiente, "estado_siguiente": turno.EstadoSiguiente}
}

func nuloBootstrap(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func salidaBolsaRRHH(referencia, categoriaRef, categoria, tipo, desde string, hasta *string, conteo map[string]int, llamamientos int, politica ...politicaOrdenRRHHDesarrollo) map[string]any {
	var criterio any = nil
	if len(politica) == 1 {
		p := politica[0]
		criterio = map[string]any{"politica_ref": p.Referencia, "version": p.Version, "criterio": p.Criterio, "tipo_lista": p.TipoLista, "reposicion": p.Reposicion, "provisional": p.Provisional, "rotulo": p.Rotulo, "actor": p.Actor, "vigente_desde": p.VigenteDesde}
	}
	return map[string]any{"bolsa_ref": referencia, "categoria_clave": strings.TrimPrefix(categoriaRef, "categoria:rpt:"), "categoria": categoria, "tipo_lista": tipo, "vigente_desde": desde, "vigente_hasta": hasta, "total": conteo["disponible"] + conteo["trabajando"] + conteo["no_disponible"] + conteo["excluido"] + conteo["renuncia"] + conteo["pendiente_incorporacion"] + conteo["disponible_desde"] + conteo["en_revision"], "por_estado": conteo, "llamamientos_en_curso": llamamientos, "politica_orden": criterio}
}

func (h *bolsasRRHHDesarrolloDatos) salidaCandidata(candidata struct {
	Referencia  string  `json:"candidatura_ref"`
	BolsaRef    string  `json:"bolsa_ref"`
	Orden       *int    `json:"orden"`
	OrdenActa   int     `json:"orden_acta"`
	RazonOrden  string  `json:"razon_orden"`
	Nombre      string  `json:"nombre_visible"`
	Documento   string  `json:"documento_enmascarado"`
	Estado      string  `json:"estado_clave"`
	EstadoDesde string  `json:"estado_desde"`
	Disponible  *string `json:"disponible_desde"`
}) map[string]any {
	var llamada any = nil
	contactos := 0
	contactosBolsa := make([]dominiobolsa.ContactoParticipacion, 0)
	for _, contacto := range h.datos.Contactos {
		if contacto.BolsaRef == candidata.BolsaRef && contacto.ParticipacionRef == candidata.Referencia {
			contactos++
			contactosBolsa = append(contactosBolsa, contacto)
		}
	}
	candidatoTurno := bolsaapplication.CandidatoTurno{ParticipacionRef: candidata.Referencia}
	if ultimo := bolsaapplication.UltimoLlamadoDesdeContactos([]bolsaapplication.CandidatoTurno{candidatoTurno}, contactosBolsa); ultimo != nil {
		contacto := ultimo.Contacto
		llamada = map[string]any{"llamamiento_ref": contacto.LlamamientoRef, "comunicado_en": contacto.Instante.UTC().Format(time.RFC3339Nano), "canal": contacto.Canal, "resultado": contacto.Resultado}
	}
	salida := map[string]any{"participacion_ref": candidata.Referencia, "orden": candidata.Orden, "orden_acta": candidata.OrdenActa, "razon_orden": candidata.RazonOrden, "nombre_visible": candidata.Nombre, "documento_enmascarado": candidata.Documento, "estado_clave": estadoBolsaCanonico(candidata.Estado), "estado_desde": candidata.EstadoDesde, "disponible_desde": candidata.Disponible, "ultimo_llamamiento": llamada, "contactos_total": contactos}
	if h.datos.Marcas != nil {
		salida["marcas"] = h.salidaMarcas(candidata.Referencia, candidata.Estado)
	}
	return salida
}

// respuestaEstadisticas agrega lo que el cuadro ya muestra: bolsas vigentes y
// sustituidas, personas por estado, y llamamientos por canal y resultado. No
// inventa periodos: el conjunto es el vigente, con su instante de generación.
func (h *bolsasRRHHDesarrolloDatos) respuestaEstadisticas() map[string]any {
	porEstado := mapaEstadosVacio()
	porBolsa := make([]map[string]any, 0, len(h.datos.Bolsas))
	conteos := h.contarEstadosPorBolsa()
	vigentes, sustituidas := 0, 0
	for _, bolsa := range h.datos.Bolsas {
		conteo := conteos[bolsa.Referencia]
		if conteo == nil {
			conteo = mapaEstadosVacio()
		}
		for estado, n := range conteo {
			porEstado[estado] += n
		}
		total := 0
		for _, n := range conteo {
			total += n
		}
		if bolsa.VigenteHasta == nil {
			vigentes++
		} else {
			sustituidas++
		}
		porBolsa = append(porBolsa, map[string]any{
			"bolsa_ref": bolsa.Referencia, "categoria": bolsa.Categoria, "tipo_lista": bolsa.TipoLista,
			"vigente": bolsa.VigenteHasta == nil, "total": total, "por_estado": conteo,
		})
	}
	personas := 0
	for _, n := range porEstado {
		personas += n
	}
	canales := map[string]int{}
	resultados := map[string]int{}
	for _, llamamiento := range h.datos.Llamamientos {
		if llamamiento.Canal != "" {
			canales[llamamiento.Canal]++
		}
		if llamamiento.Resultado != "" {
			resultados[llamamiento.Resultado]++
		}
	}
	return map[string]any{
		"esquema": "vec.bolsa.rrhh.estadisticas.v1", "generado_en": instanteBolsasRRHH(h.datos.GeneradoEn),
		"bolsas":       map[string]any{"total": len(h.datos.Bolsas), "vigentes": vigentes, "sustituidas": sustituidas},
		"personas":     map[string]any{"total": personas, "por_estado": porEstado},
		"llamamientos": map[string]any{"total": len(h.datos.Llamamientos), "por_canal": canales, "por_resultado": resultados},
		"por_bolsa":    porBolsa,
	}
}

// contarEstadosPorBolsa recorre las candidaturas una sola vez. En una carga
// importada con muchas bolsas, repetir todo el recorrido para cada bolsa
// convertía la proyección del cuadro y las estadísticas en O(bolsas × personas).
func (h *bolsasRRHHDesarrolloDatos) contarEstadosPorBolsa() map[string]map[string]int {
	conteos := make(map[string]map[string]int, len(h.datos.Bolsas))
	for _, candidata := range h.datos.Candidaturas {
		conteo := conteos[candidata.BolsaRef]
		if conteo == nil {
			conteo = mapaEstadosVacio()
			conteos[candidata.BolsaRef] = conteo
		}
		conteo[estadoBolsaCanonico(candidata.Estado)]++
	}
	return conteos
}

func mapaEstadosVacio() map[string]int {
	return map[string]int{"disponible": 0, "no_disponible": 0, "trabajando": 0, "pendiente_incorporacion": 0, "renuncia": 0, "excluido": 0, "disponible_desde": 0, "en_revision": 0}
}
func estadoBolsaCanonico(origen string) string { return origen }
func estadoBolsaVisible(estado string) bool    { _, ok := mapaEstadosVacio()[estado]; return ok }
func instanteBolsasRRHH(valor string) string {
	if _, err := time.Parse(time.RFC3339, valor); err == nil {
		return valor
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func responderBolsaRRHHDesarrollo(w http.ResponseWriter, estado int, valor any, soloCabecera ...bool) {
	b, _ := json.Marshal(valor)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(estado)
	if estado != http.StatusNoContent && (len(soloCabecera) == 0 || !soloCabecera[0]) {
		_, _ = w.Write(b)
	}
}
