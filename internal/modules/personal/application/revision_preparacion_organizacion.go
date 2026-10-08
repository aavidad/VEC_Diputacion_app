package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

// PaquetePreparacionOrganizacion conserva el paquete de preparación existente.
// Las decisiones son declaraciones opcionales, sin aprobación ni efecto durable.
type PaquetePreparacionOrganizacion struct {
	Manifiesto domain.ManifiestoImportacionOrganizacion  `json:"manifiesto"`
	Hechos     []domain.HechoImportacionOrganizacion     `json:"hechos"`
	Decisiones []domain.DecisionConciliacionOrganizacion `json:"decisiones,omitempty"`
}

type InformeRevisionPreparacionOrganizacion struct {
	Valido                 bool                               `json:"valido"`
	Estado                 string                             `json:"estado"`
	ClaveError             string                             `json:"clave_error,omitempty"`
	Seccion                string                             `json:"seccion,omitempty"`
	FilaFallida            int                                `json:"fila_fallida,omitempty"`
	Hechos                 int                                `json:"hechos"`
	Decisiones             int                                `json:"decisiones"`
	RecuentosClase         map[string]int                     `json:"recuentos_clase"`
	RecuentosDecision      map[string]int                     `json:"recuentos_decision"`
	ManifiestoHuellaSHA256 string                             `json:"manifiesto_huella_sha256,omitempty"`
	PaqueteHuellaSHA256    string                             `json:"paquete_huella_sha256,omitempty"`
	PendientesPublicacion  []string                           `json:"pendientes_publicacion"`
	CoberturaConciliacion  *CoberturaConciliacionOrganizacion `json:"cobertura_conciliacion,omitempty"`
}

// RevisarPreparacionOrganizacion solo comprueba material en memoria. No consulta
// catálogos, acredita fuentes, autoriza actores ni invoca el importador durable.
func RevisarPreparacionOrganizacion(p PaquetePreparacionOrganizacion) InformeRevisionPreparacionOrganizacion {
	r := InformeRevisionPreparacionOrganizacion{Estado: "preparacion_no_autoritativa",
		Hechos: len(p.Hechos), Decisiones: len(p.Decisiones),
		RecuentosClase: map[string]int{}, RecuentosDecision: map[string]int{},
		PendientesPublicacion: []string{"verificacion_catalogos", "acreditacion_fuente", "conciliacion_autorizada", "aprobacion_separada", "publicacion_autorizada"}}
	fallo := func(clave, seccion string, fila int) InformeRevisionPreparacionOrganizacion {
		r.ClaveError, r.Seccion, r.FilaFallida = clave, seccion, fila
		return r
	}
	if p.Manifiesto.Validar(true) != nil {
		return fallo("manifiesto_invalido", "manifiesto", 0)
	}
	if len(p.Hechos) < 1 || len(p.Hechos) > 3000 {
		return fallo("cantidad_hechos_invalida", "hechos", 0)
	}
	if len(p.Decisiones) > 1000 {
		return fallo("cantidad_decisiones_invalida", "decisiones", 0)
	}
	vistos, origenes, filas := map[string]bool{}, map[string]bool{}, map[string]map[string]bool{}
	for i, h := range p.Hechos {
		if h.Validar(p.Manifiesto) != nil {
			return fallo("hecho_invalido", "hechos", i+1)
		}
		identidad, origen := h.Clase+"\x00"+h.HechoRef, h.Clase+"\x00"+h.FilaFuenteRef
		if vistos[identidad] || origenes[origen] {
			return fallo("hecho_duplicado", "hechos", i+1)
		}
		vistos[identidad], origenes[origen] = true, true
		if filas[h.FilaFuenteRef] == nil {
			filas[h.FilaFuenteRef] = map[string]bool{}
		}
		filas[h.FilaFuenteRef][h.Clase] = true
		r.RecuentosClase[h.Clase]++
	}
	vistas := map[string]bool{}
	for i, d := range p.Decisiones {
		if d.Validar() != nil {
			return fallo("decision_invalida", "decisiones", i+1)
		}
		clave := d.Clase + "\x00" + d.FilaFuenteRef
		if vistas[clave] {
			return fallo("decision_duplicada", "decisiones", i+1)
		}
		clasesHecho, existe := filas[d.FilaFuenteRef]
		if !existe {
			return fallo("decision_sin_fila", "decisiones", i+1)
		}
		if d.Clase != "clasificacion" && !clasesHecho[d.Clase] && !(d.Clase == "unidad" && clasesHecho["nodo"]) {
			return fallo("decision_invalida", "decisiones", i+1)
		}
		vistas[clave] = true
		r.RecuentosDecision[d.Resultado]++
		if d.Resultado == "pendiente" {
			r.PendientesPublicacion = append(r.PendientesPublicacion, "decisiones_pendientes")
		}
	}
	m := p.Manifiesto
	for _, campo := range []struct{ clave, valor string }{
		{"documento_ref", m.DocumentoRef}, {"custodia_ref", m.CustodiaRef}, {"diccionario_ref", m.DiccionarioRef}, {"acto_ref", m.ActoRef},
		{"aprobada_en", string(m.AprobadaEn)}, {"publicada_en", string(m.PublicadaEn)}, {"efectos_desde", string(m.EfectosDesde)},
	} {
		if campo.valor == "" {
			r.PendientesPublicacion = append(r.PendientesPublicacion, campo.clave)
		}
	}
	sort.Strings(r.PendientesPublicacion)
	r.PendientesPublicacion = compactarPendientes(r.PendientesPublicacion)
	manifiestoHuella, err := m.HuellaSHA256()
	if err != nil {
		return fallo("manifiesto_invalido", "manifiesto", 0)
	}
	p = normalizarPreparacionOrganizacion(p)
	material, err := json.Marshal(p)
	if err != nil {
		return fallo("paquete_invalido", "paquete", 0)
	}
	h := sha256.Sum256(material)
	r.ManifiestoHuellaSHA256 = manifiestoHuella
	r.PaqueteHuellaSHA256, r.Valido = hex.EncodeToString(h[:]), true
	r.CoberturaConciliacion = revisarCoberturaConciliacionOrganizacion(p)
	return r
}

func compactarPendientes(p []string) []string {
	n := 0
	for _, s := range p {
		if n == 0 || p[n-1] != s {
			p[n] = s
			n++
		}
	}
	return p[:n]
}

// PrepararPaqueteOrganizacion devuelve el material revisado en el mismo orden
// canónico que identifica PaqueteHuellaSHA256. Nunca devuelve material inválido.
// La revisión sigue siendo local y no acredita fuentes ni concede publicación.
func PrepararPaqueteOrganizacion(p PaquetePreparacionOrganizacion) (PaquetePreparacionOrganizacion, InformeRevisionPreparacionOrganizacion) {
	r := RevisarPreparacionOrganizacion(p)
	if !r.Valido {
		return PaquetePreparacionOrganizacion{}, r
	}
	return normalizarPreparacionOrganizacion(p), r
}

func normalizarPreparacionOrganizacion(p PaquetePreparacionOrganizacion) PaquetePreparacionOrganizacion {
	p.Hechos = append([]domain.HechoImportacionOrganizacion(nil), p.Hechos...)
	p.Decisiones = append([]domain.DecisionConciliacionOrganizacion(nil), p.Decisiones...)
	sort.Slice(p.Hechos, func(i, j int) bool {
		a, b := p.Hechos[i], p.Hechos[j]
		if a.Clase != b.Clase {
			return a.Clase < b.Clase
		}
		return a.HechoRef < b.HechoRef
	})
	sort.Slice(p.Decisiones, func(i, j int) bool {
		a, b := p.Decisiones[i], p.Decisiones[j]
		if a.Clase != b.Clase {
			return a.Clase < b.Clase
		}
		return a.FilaFuenteRef < b.FilaFuenteRef
	})
	return p
}
