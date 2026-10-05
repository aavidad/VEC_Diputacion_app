package application

// La cobertura describe las decisiones declaradas del paquete validado.
// Completa conserva el criterio del importador para cubrir hechos; no afirma
// conciliación durable, acreditación de fuente, aprobación ni publicación.
type CoberturaConciliacionOrganizacion struct {
	Completa              bool                            `json:"completa"`
	RecuentosHechos       map[string]int                  `json:"recuentos_hechos"`
	Hechos                []CoberturaHechoOrganizacion    `json:"hechos"`
	DecisionesAdicionales []CoberturaDecisionOrganizacion `json:"decisiones_adicionales,omitempty"`
}

type CoberturaHechoOrganizacion struct {
	HechoRef      string `json:"hecho_ref"`
	Clase         string `json:"clase"`
	FilaFuenteRef string `json:"fila_fuente_ref"`
	ClaseDecision string `json:"clase_decision"`
	Resultado     string `json:"resultado"`
}

type CoberturaDecisionOrganizacion struct {
	Clase         string `json:"clase"`
	FilaFuenteRef string `json:"fila_fuente_ref"`
	Resultado     string `json:"resultado"`
}

// p ya está validado y normalizado. La identidad de decisión incluye clase y
// fila: dos clases de hecho pueden proceder de la misma fila de la fuente.
func revisarCoberturaConciliacionOrganizacion(p PaquetePreparacionOrganizacion) *CoberturaConciliacionOrganizacion {
	c := &CoberturaConciliacionOrganizacion{Completa: true,
		RecuentosHechos: map[string]int{"sin_decision": 0, "pendiente": 0, "descartada": 0, "vinculada": 0},
		Hechos:          make([]CoberturaHechoOrganizacion, 0, len(p.Hechos))}
	resultados := make(map[string]string, len(p.Decisiones))
	for _, d := range p.Decisiones {
		resultados[d.Clase+"\x00"+d.FilaFuenteRef] = d.Resultado
		if d.Resultado != "vinculada" {
			c.Completa = false
		}
		// Clasificación comprueba una referencia adicional y nunca sustituye la
		// decisión de unidad, puesto, plaza, dotación o vínculo de un hecho.
		if d.Clase == "clasificacion" {
			c.DecisionesAdicionales = append(c.DecisionesAdicionales, CoberturaDecisionOrganizacion{
				Clase: d.Clase, FilaFuenteRef: d.FilaFuenteRef, Resultado: d.Resultado})
		}
	}
	for _, h := range p.Hechos {
		claseDecision := h.Clase
		if h.Clase == "nodo" {
			claseDecision = "unidad"
		}
		resultado, existe := resultados[claseDecision+"\x00"+h.FilaFuenteRef]
		if !existe {
			resultado = "sin_decision"
		}
		if resultado != "vinculada" {
			c.Completa = false
		}
		c.RecuentosHechos[resultado]++
		c.Hechos = append(c.Hechos, CoberturaHechoOrganizacion{HechoRef: h.HechoRef, Clase: h.Clase,
			FilaFuenteRef: h.FilaFuenteRef, ClaseDecision: claseDecision, Resultado: resultado})
	}
	return c
}
