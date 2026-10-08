package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
)

const vigenciaCorteGlobalBolsas = 5 * time.Minute
const maximoBytesCortesGlobales = 64 << 20

type corteGlobalBolsasRRHH struct {
	ref    string
	creado time.Time
	bytes  int
	datos  bolsaapp.ConjuntoGlobalRRHH
}

// Conserva únicamente la proyección mínima ya leída por la ruta de resumen.
// Nunca conserva nombres/documentos ni sustituye la revalidación de la ruta:
// cada página pasa por la misma autoridad mTLS de /bolsas que el cuadro.
type cacheGlobalBolsasRRHH struct {
	mu     sync.Mutex
	cortes []corteGlobalBolsasRRHH
}

func (c *cacheGlobalBolsasRRHH) guardar(datos bolsaapp.ConjuntoGlobalRRHH, ahora time.Time) (string, error) {
	// Una recarga del mismo conjunto reutiliza el corte emitido. La hora de
	// lectura no cambia las filas ni justifica invalidar sus continuaciones.
	huella := datos
	huella.GeneradoEn = ""
	b, err := json.Marshal(huella)
	if err != nil {
		return "", err
	}
	if len(b) > maximoBytesCortesGlobales {
		return "", bolsaapp.ErrConsultaGlobalRRHHInvalida
	}
	hash := sha256.Sum256(b)
	ref := hex.EncodeToString(hash[:])
	c.mu.Lock()
	defer c.mu.Unlock()
	c.purgar(ahora)
	for _, corte := range c.cortes {
		if corte.ref == ref {
			return ref, nil
		}
	}
	bytes := len(b)
	for _, corte := range c.cortes {
		bytes += corte.bytes
	}
	// Nunca retirar un corte vigente para admitir otro. Si la memoria está
	// llena, la lectura nueva falla; los enlaces ya entregados siguen válidos.
	if bytes > maximoBytesCortesGlobales {
		return "", fmt.Errorf("bolsa: memoria de cortes esperado<=%d observado=%d", maximoBytesCortesGlobales, bytes)
	}
	c.cortes = append(c.cortes, corteGlobalBolsasRRHH{ref: ref, creado: ahora, bytes: len(b), datos: datos})
	return ref, nil
}

func (c *cacheGlobalBolsasRRHH) purgar(ahora time.Time) {
	for len(c.cortes) > 0 && !ahora.Before(c.cortes[0].creado.Add(vigenciaCorteGlobalBolsas)) {
		c.cortes[0] = corteGlobalBolsasRRHH{}
		c.cortes = c.cortes[1:]
	}
}

func (c *cacheGlobalBolsasRRHH) leer(ref string, ahora time.Time) (bolsaapp.ConjuntoGlobalRRHH, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.purgar(ahora)
	for _, corte := range c.cortes {
		if corte.ref == ref {
			return corte.datos, true
		}
	}
	return bolsaapp.ConjuntoGlobalRRHH{}, false
}

func conjuntoGlobalBolsas(datos datasetBolsasRRHHDesarrollo) (bolsaapp.ConjuntoGlobalRRHH, error) {
	c := bolsaapp.ConjuntoGlobalRRHH{GeneradoEn: datos.GeneradoEn, ListaLlamamientosDisponible: datos.LlamamientosGlobales != nil}
	categorias := make(map[string]string, len(datos.Bolsas))
	conteos := make(map[string]int, len(datos.Bolsas))
	for _, b := range datos.Bolsas {
		categorias[b.Referencia] = b.Categoria
		conteos[b.Referencia] = b.LlamamientosEnCurso
	}
	c.Participaciones = make([]bolsaapp.ParticipacionGlobalRRHH, 0, len(datos.Candidaturas))
	for _, p := range datos.Candidaturas {
		categoria, ok := categorias[p.BolsaRef]
		if !ok || !estadoBolsaVisible(p.Estado) || p.OrdenActa < 1 {
			return c, bolsaapp.ErrConsultaGlobalRRHHInvalida
		}
		var disponible *string
		if p.Disponible != nil {
			valor := *p.Disponible
			disponible = &valor
		}
		c.Participaciones = append(c.Participaciones, bolsaapp.ParticipacionGlobalRRHH{BolsaRef: p.BolsaRef, Categoria: categoria, ParticipacionRef: p.Referencia, OrdenActa: p.OrdenActa, Estado: p.Estado, EstadoDesde: p.EstadoDesde, DisponibleDesde: disponible})
	}
	c.Llamamientos = make([]bolsaapp.LlamamientoGlobalRRHH, 0, len(datos.LlamamientosGlobales))
	for _, l := range datos.LlamamientosGlobales {
		categoria, ok := categorias[l.BolsaRef]
		if !ok || l.LlamamientoRef == "" {
			return c, bolsaapp.ErrConsultaGlobalRRHHInvalida
		}
		l.Categoria = categoria
		c.Llamamientos = append(c.Llamamientos, l)
		conteos[l.BolsaRef]--
	}
	if c.ListaLlamamientosDisponible {
		for _, n := range conteos {
			if n != 0 {
				return c, bolsaapp.ErrConsultaGlobalRRHHInvalida
			}
		}
	}
	c.Ordenar()
	return c, nil
}

func (h *bolsasRRHHDesarrollo) adjuntarCorteGlobal(w http.ResponseWriter, r *http.Request, datos datasetBolsasRRHHDesarrollo, respuesta map[string]any) bool {
	if !datos.ResumenConjunto {
		return true
	} // conserva el montaje legado.
	conjunto, err := conjuntoGlobalBolsas(datos)
	if err != nil {
		responderFalloBolsaRRHHDesarrollo(w, r, "corte_global", err)
		return false
	}
	ref, err := h.global.guardar(conjunto, time.Now())
	if err != nil {
		responderFalloBolsaRRHHDesarrollo(w, r, "corte_global", err)
		return false
	}
	respuesta["corte_ref"] = ref
	if retenido, existe := h.global.leer(ref, time.Now()); existe {
		respuesta["generado_en"] = retenido.GeneradoEn
	}
	respuesta["lista_llamamientos_disponible"] = conjunto.ListaLlamamientosDisponible
	if llamamientos, ok := respuesta["llamamientos"].(map[string]any); ok {
		total := 0
		for _, b := range datos.Bolsas {
			total += b.LlamamientosEnCurso
		}
		llamamientos["en_curso_total"] = total
	}
	return true
}

func consultaGlobalBolsas(cruda string) (bolsaapp.ConsultaGlobalRRHH, string, bool) {
	q := bolsaapp.ConsultaGlobalRRHH{Limite: 50}
	if len(cruda) > 4096 {
		return q, "", false
	}
	v, err := url.ParseQuery(cruda)
	if err != nil || len(v) > 6 {
		return q, "", false
	}
	corte := ""
	for clave, valores := range v {
		if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
			return q, corte, false
		}
		valor := valores[0]
		switch clave {
		case "filtro":
			q.Filtro = valor
		case "bolsa":
			if len(valor) > 512 || strings.ContainsAny(valor, "/\x00\r\n") {
				return q, corte, false
			}
			q.BolsaRef = valor
		case "corte":
			b, err := hex.DecodeString(valor)
			if err != nil || len(b) != 32 || valor != strings.ToLower(valor) {
				return q, corte, false
			}
			corte = valor
		case "cursor", "limite":
			n, err := strconv.Atoi(valor)
			if err != nil || strconv.Itoa(n) != valor {
				return q, corte, false
			}
			if clave == "cursor" {
				q.Desde = n
			} else {
				q.Limite = n
			}
		default:
			return q, corte, false
		}
	}
	return q, corte, q.Validar() == nil && (q.Desde == 0 || corte != "")
}

func (h *bolsasRRHHDesarrollo) responderGlobal(w http.ResponseWriter, r *http.Request) {
	q, corte, ok := consultaGlobalBolsas(r.URL.RawQuery)
	if !ok {
		responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	if corte == "" {
		vista, disponible := h.vistaResumen(w, r)
		if !disponible {
			return
		}
		if !vista.datos.ResumenConjunto {
			responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
			return
		}
		respuesta := map[string]any{}
		if !h.adjuntarCorteGlobal(w, r, vista.datos, respuesta) {
			return
		}
		corte = respuesta["corte_ref"].(string)
	}
	conjunto, existe := h.global.leer(corte, time.Now())
	if !existe {
		responderBolsaRRHHDesarrollo(w, http.StatusConflict, map[string]string{"codigo": "corte_caducado"})
		return
	}
	if q.BolsaRef != "" {
		bolsaExiste := false
		for _, p := range conjunto.Participaciones {
			if p.BolsaRef == q.BolsaRef {
				bolsaExiste = true
				break
			}
		}
		if !bolsaExiste {
			responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
			return
		}
	}
	pagina, err := conjunto.Paginar(q)
	if err != nil {
		responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	var siguiente any
	if pagina.Hasta < pagina.Total {
		siguiente = strconv.Itoa(pagina.Hasta)
	}
	responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": "vec.bolsa.rrhh.global.v1", "generado_en": conjunto.GeneradoEn, "corte_ref": corte,
		"filtro": q.Filtro, "bolsa_ref": nuloBootstrap(q.BolsaRef), "total": pagina.Total,
		"desde": pagina.Desde, "hasta": pagina.Hasta, "hay_mas": siguiente != nil, "cursor_siguiente": siguiente, "items": pagina.Items,
	}}, r.Method == http.MethodHead)
}
