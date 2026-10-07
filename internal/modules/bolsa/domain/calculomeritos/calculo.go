package calculomeritos

import (
	"encoding/json"
	"sort"

	"vec-diputacion-granada/internal/shared/baremacion"
)

// Resultado conserva todos los escalones del cálculo. En bloqueado no existe
// Total ni puntos parciales; las incidencias identifican el hecho afectado.
type Resultado struct {
	Esquema         string             `json:"esquema"`
	Motor           string             `json:"motor"`
	ConvocatoriaRef string             `json:"convocatoria_ref"`
	Conjunto        Dependencia        `json:"conjunto"`
	Entrada         Dependencia        `json:"entrada"`
	Estado          string             `json:"estado"`
	Incidencias     []Incidencia       `json:"incidencias"`
	Meritos         []DetalleMerito    `json:"meritos"`
	Reglas          []DetalleRegla     `json:"reglas"`
	Secciones       []DetalleSeccion   `json:"secciones"`
	SumaSecciones   *baremacion.Puntos `json:"suma_secciones,omitempty"`
	MaximoTotal     baremacion.Puntos  `json:"maximo_total"`
	Total           *baremacion.Puntos `json:"total,omitempty"`
}
type Incidencia struct {
	MeritoRef string `json:"merito_ref"`
	Codigo    string `json:"codigo"`
}
type DetalleMerito struct {
	MeritoRef  string `json:"merito_ref"`
	ReglaClave string `json:"regla_clave,omitempty"`
	Codigo     string `json:"codigo"`
}
type DetalleRegla struct {
	Clave              string                  `json:"clave"`
	SeccionClave       string                  `json:"seccion_clave"`
	Definicion         Dependencia             `json:"definicion"`
	ElementosAdmitidos uint32                  `json:"elementos_admitidos"`
	UnidadesAdmitidas  baremacion.Racional     `json:"unidades_admitidas"`
	UnidadesComputadas baremacion.Racional     `json:"unidades_computadas"`
	PuntosPorUnidad    baremacion.Puntos       `json:"puntos_por_unidad"`
	Redondeo           baremacion.ModoRedondeo `json:"redondeo"`
	BrutoRedondeado    baremacion.Puntos       `json:"bruto_redondeado"`
	MaximoPuntos       baremacion.Puntos       `json:"maximo_puntos"`
	Puntos             baremacion.Puntos       `json:"puntos"`
}
type DetalleSeccion struct {
	Clave        string            `json:"clave"`
	Definicion   Dependencia       `json:"definicion"`
	SumaReglas   baremacion.Puntos `json:"suma_reglas"`
	MaximoPuntos baremacion.Puntos `json:"maximo_puntos"`
	Puntos       baremacion.Puntos `json:"puntos"`
}

func (r Resultado) RepresentacionCanonica() ([]byte, error) { return json.Marshal(r) }
func (r Resultado) HuellaSHA256() (string, error) {
	b, err := r.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	return HuellaSHA256(b), nil
}

// Calcular valida ambas instantáneas. La política de corte es inclusiva y los
// duplicados de hecho/evidencia se bloquean incluso si usan distintas clases.
func Calcular(c Conjunto, e Entrada) (Resultado, error) {
	cb, err := c.RepresentacionCanonica()
	if err != nil {
		return Resultado{}, err
	}
	eb, err := e.RepresentacionCanonica()
	if err != nil {
		return Resultado{}, err
	}
	dependencias := []Dependencia{c.Bases, {c.Referencia, c.Version, HuellaSHA256(cb)}, {e.Referencia, e.Version, HuellaSHA256(eb)}}
	for _, seccion := range c.Secciones {
		dependencias = append(dependencias, seccion.Definicion)
	}
	for _, regla := range c.Reglas {
		dependencias = append(dependencias, regla.Definicion, regla.Catalogo)
	}
	for _, merito := range e.Meritos {
		dependencias = append(dependencias, merito.Hecho, merito.Evidencia, merito.Catalogo)
	}
	if err := validarConsistenciaDependencias(dependencias); err != nil {
		return Resultado{}, err
	}
	r := Resultado{Esquema: EsquemaResultado, Motor: "vec.bolsa.motor_meritos.v1", ConvocatoriaRef: c.ConvocatoriaRef,
		Conjunto: Dependencia{c.Referencia, c.Version, HuellaSHA256(cb)}, Entrada: Dependencia{e.Referencia, e.Version, HuellaSHA256(eb)},
		Estado: "completado", Incidencias: []Incidencia{}, Meritos: []DetalleMerito{}, Reglas: []DetalleRegla{}, Secciones: []DetalleSeccion{}, MaximoTotal: *c.MaximoTotal}
	reglas := make(map[string]Regla)
	for _, regla := range c.Reglas {
		reglas[regla.Familia+":"+regla.Clase] = regla
	}
	hechos, evidencias := make(map[string]bool), make(map[string]bool)
	candidatos := make(map[string][]int)
	bloqueo := func(m Merito, codigo string) {
		r.Incidencias = append(r.Incidencias, Incidencia{m.Referencia, codigo})
		r.Estado = "bloqueado"
	}
	for _, m := range e.Meritos {
		if hechos[m.Hecho.Referencia] || evidencias[m.Evidencia.Referencia] {
			bloqueo(m, "merito_duplicado")
		}
		hechos[m.Hecho.Referencia] = true
		evidencias[m.Evidencia.Referencia] = true
	}
	for _, m := range e.Meritos {
		detalle := DetalleMerito{MeritoRef: m.Referencia}
		switch {
		case m.Uso == "requisito":
			detalle.Codigo = "requisito_no_puntua"
		case m.Estado == "rechazado":
			detalle.Codigo = "merito_rechazado"
		case m.Aplicabilidad == "no":
			detalle.Codigo = "no_aplicable"
		default:
			regla, ok := reglas[m.Familia+":"+m.Clase]
			switch {
			case !ok:
				detalle.Codigo = "regla_ausente"
				bloqueo(m, detalle.Codigo)
			case m.Catalogo != regla.Catalogo:
				detalle.Codigo = "catalogo_incompatible"
				bloqueo(m, detalle.Codigo)
			case m.Estado != "acreditado":
				detalle.Codigo = "acreditacion_pendiente"
				bloqueo(m, detalle.Codigo)
			case m.Aplicabilidad != "si":
				detalle.Codigo = "aplicabilidad_pendiente"
				bloqueo(m, detalle.Codigo)
			case m.FechaObtencion == nil || m.Unidades == nil:
				detalle.Codigo = "dato_pendiente"
				bloqueo(m, detalle.Codigo)
			default:
				detalle.ReglaClave = regla.Clave
				corte, _ := m.FechaObtencion.Comparar(c.FechaCorteInclusiva)
				caducado := false
				if m.VigenteHasta != nil {
					cmp, _ := m.VigenteHasta.Comparar(c.FechaCorteInclusiva)
					caducado = cmp < 0
				}
				cantidad, _ := m.Unidades.Comparar(*regla.MinimoUnidades)
				switch {
				case corte > 0:
					detalle.Codigo = "posterior_al_corte"
				case caducado:
					detalle.Codigo = "merito_caducado"
				case cantidad < 0:
					detalle.Codigo = "inferior_al_minimo"
				default:
					detalle.Codigo = "admitido"
					candidatos[regla.Clave] = append(candidatos[regla.Clave], len(r.Meritos))
				}
			}
		}
		r.Meritos = append(r.Meritos, detalle)
	}
	if r.Estado == "bloqueado" {
		return r, nil
	}
	cero, err := baremacion.NuevoRacional(0, 1)
	if err != nil {
		return Resultado{}, err
	}
	puntosSeccion := make(map[string]baremacion.Puntos)
	for _, regla := range c.Reglas {
		indices := candidatos[regla.Clave]
		// Mayor cantidad gana; en igualdad todos tienen idéntico efecto numérico.
		// El desempate por referencia solo fija el desglose reproducible.
		sort.Slice(indices, func(i, j int) bool {
			cmp, _ := e.Meritos[indices[i]].Unidades.Comparar(*e.Meritos[indices[j]].Unidades)
			if cmp != 0 {
				return cmp > 0
			}
			return e.Meritos[indices[i]].Referencia < e.Meritos[indices[j]].Referencia
		})
		unidades := cero
		admitidos := uint32(0)
		for _, idx := range indices {
			if admitidos >= *regla.MaximoElementos {
				r.Meritos[idx].Codigo = "maximo_elementos"
				continue
			}
			unidades, err = unidades.Sumar(*e.Meritos[idx].Unidades)
			if err != nil {
				return Resultado{}, err
			}
			admitidos++
		}
		computadas := unidades
		cmp, _ := computadas.Comparar(*regla.MaximoUnidades)
		if cmp > 0 {
			computadas = *regla.MaximoUnidades
		}
		bruto, err := regla.PuntosPorUnidad.MultiplicarRedondeado(computadas, regla.Redondeo)
		if err != nil {
			return Resultado{}, err
		}
		tope, err := baremacion.AplicarTope(bruto, *regla.MaximoPuntos)
		if err != nil {
			return Resultado{}, err
		}
		puntos := tope.Resultado()
		r.Reglas = append(r.Reglas, DetalleRegla{regla.Clave, regla.SeccionClave, regla.Definicion, admitidos, unidades, computadas, *regla.PuntosPorUnidad, regla.Redondeo, bruto, *regla.MaximoPuntos, puntos})
		puntosSeccion[regla.SeccionClave], err = puntosSeccion[regla.SeccionClave].Sumar(puntos)
		if err != nil {
			return Resultado{}, err
		}
	}
	suma := baremacion.Puntos{}
	for _, s := range c.Secciones {
		bruto := puntosSeccion[s.Clave]
		tope, err := baremacion.AplicarTope(bruto, *s.MaximoPuntos)
		if err != nil {
			return Resultado{}, err
		}
		puntos := tope.Resultado()
		r.Secciones = append(r.Secciones, DetalleSeccion{s.Clave, s.Definicion, bruto, *s.MaximoPuntos, puntos})
		suma, err = suma.Sumar(puntos)
		if err != nil {
			return Resultado{}, err
		}
	}
	tope, err := baremacion.AplicarTope(suma, *c.MaximoTotal)
	if err != nil {
		return Resultado{}, err
	}
	total := tope.Resultado()
	r.SumaSecciones = &suma
	r.Total = &total
	return r, nil
}
