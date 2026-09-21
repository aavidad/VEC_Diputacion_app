package bootstrap

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const (
	rutaBolsasRRHHDesarrollo        = "/api/vec/bolsa/bolsas"
	prefijoCandidatosRRHHDesarrollo = rutaBolsasRRHHDesarrollo
)

type datasetBolsasRRHHDesarrollo struct {
	GeneradoEn string `json:"generado_en"`
	Bolsas     []struct {
		Referencia   string  `json:"bolsa_ref"`
		CategoriaRef string  `json:"categoria_ref"`
		Categoria    string  `json:"categoria"`
		TipoLista    string  `json:"tipo_lista"`
		VigenteDesde string  `json:"vigente_desde"`
		VigenteHasta *string `json:"vigente_hasta"`
	} `json:"bolsas"`
	Candidaturas []struct {
		Referencia  string  `json:"candidatura_ref"`
		BolsaRef    string  `json:"bolsa_ref"`
		Orden       int     `json:"orden"`
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
}

type bolsasRRHHDesarrollo struct {
	cargar    func(context.Context) (datasetBolsasRRHHDesarrollo, error)
	mutar     http.Handler
	invalidar func()
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
	if len(mutadores) == 1 {
		manejador.mutar = mutadores[0]
		manejador.invalidar = invalidar
	}
	return []vechttp.RutaExacta{{Ruta: rutaBolsasRRHHDesarrollo, Manejador: manejador}},
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
	esContacto := h != nil && h.mutar != nil && r != nil && (r.Method == http.MethodPost || r.Method == http.MethodGet)
	if esContacto {
		_, _, esContacto = bolsahttp.ReferenciasRutaContactosParticipacion(r)
	}
	cabeceras := http.Header(nil)
	if r != nil {
		cabeceras = r.Header
		if esMutacionSituacion || esContacto {
			cabeceras = r.Header.Clone()
			cabeceras.Del("Idempotency-Key")
		}
	}
	if h == nil || r == nil || r.URL == nil || r.URL.RawPath != "" || len(r.TransferEncoding) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(cabeceras) {
		responderAreaPersonalDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	if esMutacionSituacion || esContacto {
		h.mutar.ServeHTTP(w, r)
		if h.invalidar != nil {
			h.invalidar()
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderAreaPersonalDesarrollo(w, http.StatusMethodNotAllowed, map[string]string{"codigo": "metodo_no_permitido"})
		return
	}
	if r.URL.Path == rutaBolsasRRHHDesarrollo {
		if r.URL.RawQuery != "" || r.ContentLength != 0 {
			responderAreaPersonalDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
			return
		}
		vista, ok := h.vistaDurable(r.Context(), w)
		if !ok {
			return
		}
		responderAreaPersonalDesarrollo(w, http.StatusOK, map[string]any{"data": vista.respuestaBolsas()}, r.Method == http.MethodHead)
		return
	}
	bolsaRef, ok := referenciaBolsaCandidatos(r.URL.Path)
	if !ok || r.ContentLength != 0 {
		responderAreaPersonalDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
		return
	}
	consulta, ok := consultaCandidatos(r.URL.RawQuery)
	if !ok {
		responderAreaPersonalDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	vista, ok := h.vistaDurable(r.Context(), w)
	if !ok {
		return
	}
	respuesta, encontrada := vista.respuestaCandidatos(bolsaRef, consulta)
	if !encontrada {
		responderAreaPersonalDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
		return
	}
	responderAreaPersonalDesarrollo(w, http.StatusOK, map[string]any{"data": respuesta}, r.Method == http.MethodHead)
}

func (h *bolsasRRHHDesarrollo) vistaDurable(ctx context.Context, w http.ResponseWriter) (*bolsasRRHHDesarrolloDatos, bool) {
	datos, err := h.cargar(ctx)
	if err != nil {
		responderAreaPersonalDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
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
	for _, bolsa := range h.datos.Bolsas {
		conteo := mapaEstadosVacio()
		for _, candidata := range h.datos.Candidaturas {
			if candidata.BolsaRef == bolsa.Referencia {
				conteo[estadoBolsaCanonico(candidata.Estado)]++
			}
		}
		bolsas = append(bolsas, salidaBolsaRRHH(bolsa.Referencia, bolsa.CategoriaRef, bolsa.Categoria, bolsa.TipoLista, bolsa.VigenteDesde, bolsa.VigenteHasta, conteo))
	}
	return map[string]any{"esquema": "vec.bolsa.rrhh.bolsas.v1", "generado_en": instanteBolsasRRHH(h.datos.GeneradoEn), "bolsas": bolsas}
}

func (h *bolsasRRHHDesarrolloDatos) respuestaCandidatos(ref string, consulta consultaCandidatosRRHH) (map[string]any, bool) {
	var bolsa *struct {
		Referencia   string  `json:"bolsa_ref"`
		CategoriaRef string  `json:"categoria_ref"`
		Categoria    string  `json:"categoria"`
		TipoLista    string  `json:"tipo_lista"`
		VigenteDesde string  `json:"vigente_desde"`
		VigenteHasta *string `json:"vigente_hasta"`
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
	candidatas := make([]int, 0)
	for indice, candidata := range h.datos.Candidaturas {
		if candidata.BolsaRef != ref || (consulta.estado != "" && estadoBolsaCanonico(candidata.Estado) != consulta.estado) || (consulta.texto != "" && !strings.Contains(strings.ToLower(candidata.Nombre+" "+candidata.Documento), consulta.texto)) {
			continue
		}
		candidatas = append(candidatas, indice)
	}
	sort.Slice(candidatas, func(i, j int) bool {
		return h.datos.Candidaturas[candidatas[i]].Orden < h.datos.Candidaturas[candidatas[j]].Orden
	})
	inicio := 0
	if consulta.cursor != "" {
		for i, indice := range candidatas {
			if h.datos.Candidaturas[indice].Referencia == consulta.cursor {
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
	for _, indice := range candidatas[inicio:fin] {
		salida = append(salida, h.salidaCandidata(h.datos.Candidaturas[indice]))
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
		siguiente = h.datos.Candidaturas[candidatas[fin-1]].Referencia
	}
	contactos := make([]map[string]any, 0)
	for _, c := range h.datos.Contactos {
		if c.BolsaRef == ref {
			contactos = append(contactos, map[string]any{"contacto_ref": c.ContactoRef, "participacion_ref": c.ParticipacionRef, "llamamiento_ref": nuloBootstrap(c.LlamamientoRef), "canal": c.Canal, "instante": c.Instante.UTC().Format(time.RFC3339Nano), "actor_ref": c.Actor, "resultado": c.Resultado, "anotacion": c.Anotacion})
		}
	}
	return map[string]any{"esquema": "vec.bolsa.rrhh.candidatos.v1", "generado_en": instanteBolsasRRHH(h.datos.GeneradoEn), "bolsa": salidaBolsaRRHH(bolsa.Referencia, bolsa.CategoriaRef, bolsa.Categoria, bolsa.TipoLista, bolsa.VigenteDesde, bolsa.VigenteHasta, conteo), "candidatos": salida, "contactos": contactos, "hay_mas": hayMas, "cursor_siguiente": siguiente}, true
}

func nuloBootstrap(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func salidaBolsaRRHH(referencia, categoriaRef, categoria, tipo, desde string, hasta *string, conteo map[string]int) map[string]any {
	return map[string]any{"bolsa_ref": referencia, "categoria_clave": strings.TrimPrefix(categoriaRef, "categoria:rpt:"), "categoria": categoria, "tipo_lista": tipo, "vigente_desde": desde, "vigente_hasta": hasta, "total": conteo["disponible"] + conteo["ocupado"] + conteo["no_disponible"] + conteo["excluido"] + conteo["renuncia_pendiente"], "por_estado": conteo}
}

func (h *bolsasRRHHDesarrolloDatos) salidaCandidata(candidata struct {
	Referencia  string  `json:"candidatura_ref"`
	BolsaRef    string  `json:"bolsa_ref"`
	Orden       int     `json:"orden"`
	Nombre      string  `json:"nombre_visible"`
	Documento   string  `json:"documento_enmascarado"`
	Estado      string  `json:"estado_clave"`
	EstadoDesde string  `json:"estado_desde"`
	Disponible  *string `json:"disponible_desde"`
}) map[string]any {
	var ultimo map[string]string
	for _, llamada := range h.datos.Llamamientos {
		if llamada.Candidatura == candidata.Referencia && (ultimo == nil || llamada.Comunicado > ultimo["comunicado_en"]) {
			ultimo = map[string]string{"llamamiento_ref": llamada.Referencia, "comunicado_en": llamada.Comunicado, "canal": llamada.Canal, "resultado": llamada.Resultado}
		}
	}
	var llamada any = nil
	if ultimo != nil {
		llamada = ultimo
	}
	contactos := 0
	for _, c := range h.datos.Contactos {
		if c.ParticipacionRef == candidata.Referencia {
			contactos++
		}
	}
	return map[string]any{"participacion_ref": candidata.Referencia, "orden": candidata.Orden, "nombre_visible": candidata.Nombre, "documento_enmascarado": candidata.Documento, "estado_clave": estadoBolsaCanonico(candidata.Estado), "estado_desde": candidata.EstadoDesde, "disponible_desde": candidata.Disponible, "ultimo_llamamiento": llamada, "contactos_total": contactos}
}

func mapaEstadosVacio() map[string]int {
	return map[string]int{"disponible": 0, "no_disponible": 0, "trabajando": 0, "pendiente_incorporacion": 0, "renuncia": 0, "excluido": 0, "disponible_desde": 0}
}
func estadoBolsaCanonico(origen string) string { return origen }
func estadoBolsaVisible(estado string) bool    { _, ok := mapaEstadosVacio()[estado]; return ok }
func instanteBolsasRRHH(valor string) string {
	if _, err := time.Parse(time.RFC3339, valor); err == nil {
		return valor
	}
	return time.Now().UTC().Format(time.RFC3339)
}
