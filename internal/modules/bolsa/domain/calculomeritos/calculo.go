package calculomeritos

import (
	"encoding/json"
	"math/big"
	"sort"

	"vec-diputacion-granada/internal/shared/baremacion"
)

// Resultado conserva todos los escalones del cálculo. En bloqueado no existe
// Total ni puntos parciales; las incidencias identifican el hecho afectado.
type Resultado struct {
	Esquema         string             `json:"esquema"`
	Motor           string             `json:"motor"`
	ConvocatoriaRef string             `json:"convocatoria_ref"`
	CategoriaRef    string             `json:"categoria_ref,omitempty"`
	PlazaRef        string             `json:"plaza_ref,omitempty"`
	Conjunto        Dependencia        `json:"conjunto"`
	Bases           *Dependencia       `json:"bases,omitempty"`
	Entrada         Dependencia        `json:"entrada"`
	Estado          string             `json:"estado"`
	Incidencias     []Incidencia       `json:"incidencias"`
	Meritos         []DetalleMerito    `json:"meritos"`
	Reglas          []DetalleRegla     `json:"reglas"`
	Secciones       []DetalleSeccion   `json:"secciones"`
	SumaSecciones   *baremacion.Puntos `json:"suma_secciones,omitempty"`
	MaximoTotal     *baremacion.Puntos `json:"maximo_total,omitempty"`
	Total           *baremacion.Puntos `json:"total,omitempty"`
}
type Incidencia struct {
	MeritoRef string `json:"merito_ref"`
	Codigo    string `json:"codigo"`
}
type DetalleMerito struct {
	MeritoRef           string             `json:"merito_ref"`
	ReglaClave          string             `json:"regla_clave,omitempty"`
	ReglaDefinicion     *Dependencia       `json:"regla_definicion,omitempty"`
	Codigo              string             `json:"codigo"`
	Validez             string             `json:"validez,omitempty"`
	Corte               string             `json:"corte,omitempty"`
	CodigoPuntos        string             `json:"codigo_puntos,omitempty"`
	PuntosProvisionales *baremacion.Puntos `json:"puntos_provisionales,omitempty"`
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
	MaximoPuntos       *baremacion.Puntos      `json:"maximo_puntos,omitempty"`
	Puntos             baremacion.Puntos       `json:"puntos"`
	UnidadesRecientes  *baremacion.Racional    `json:"unidades_recientes,omitempty"`
	TarifaReciente     *baremacion.Puntos      `json:"tarifa_reciente_por_unidad,omitempty"`
}
type DetalleSeccion struct {
	Clave        string             `json:"clave"`
	Definicion   Dependencia        `json:"definicion"`
	SumaReglas   baremacion.Puntos  `json:"suma_reglas"`
	MaximoPuntos *baremacion.Puntos `json:"maximo_puntos,omitempty"`
	Puntos       baremacion.Puntos  `json:"puntos"`
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
	v2 := c.Esquema == EsquemaReglasV2
	if v2 != (e.Esquema == EsquemaEntradaV2) {
		return Resultado{}, fallo("versiones_incompatibles")
	}
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
		Estado: "completado", Incidencias: []Incidencia{}, Meritos: []DetalleMerito{}, Reglas: []DetalleRegla{}, Secciones: []DetalleSeccion{}, MaximoTotal: c.MaximoTotal}
	if v2 {
		r.Esquema = EsquemaResultadoV2
		r.Motor = "vec.bolsa.motor_meritos.v2"
		r.CategoriaRef = c.CategoriaRef
		r.PlazaRef = c.PlazaRef
		r.Bases = &c.Bases
	}
	reglas := make(map[string]Regla)
	for _, regla := range c.Reglas {
		reglas[regla.Familia+":"+regla.Clase] = regla
	}
	hechos, evidencias, origenes := make(map[string]bool), make(map[string]bool), make(map[string]bool)
	candidatos := make(map[string][]int)
	bloqueo := func(m Merito, codigo string) {
		r.Incidencias = append(r.Incidencias, Incidencia{m.Referencia, codigo})
		r.Estado = "bloqueado"
	}
	for _, m := range e.Meritos {
		if hechos[m.Hecho.Referencia] || evidencias[m.Evidencia.Referencia] || (v2 && m.OrigenRef != "" && origenes[m.OrigenRef]) {
			bloqueo(m, "merito_duplicado")
		}
		hechos[m.Hecho.Referencia] = true
		evidencias[m.Evidencia.Referencia] = true
		if m.OrigenRef != "" {
			origenes[m.OrigenRef] = true
		}
	}
	for _, m := range e.Meritos {
		detalle := DetalleMerito{MeritoRef: m.Referencia}
		if v2 {
			detalle.Corte = c.FechaCorteInclusiva.String()
			if regla, ok := reglas[m.Familia+":"+m.Clase]; ok {
				detalle.ReglaClave = regla.Clave
				detalle.ReglaDefinicion = &regla.Definicion
			}
		}
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
			case v2 && m.Familia == "formacion" && !conceptoRelacionado(regla.ConceptosRelacionados, m.ConceptoRef):
				detalle.Codigo = "sin_relacion_publicada_plaza"
				if r.Estado != "bloqueado" {
					r.Estado = "provisional"
				}
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
				if !v2 && m.VigenteHasta != nil {
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
				case v2 && regla.Temporal != nil && fechaAnterior(*m.FechaObtencion, regla.Temporal.DesdeBonificacion) && regla.Temporal.EfectoAnterior == "excluir":
					detalle.Codigo = "fuera_ventana_publicada"
				default:
					detalle.Codigo = "admitido"
					candidatos[regla.Clave] = append(candidatos[regla.Clave], len(r.Meritos))
				}
			}
		}
		if v2 {
			switch detalle.Codigo {
			case "admitido":
				detalle.Validez = "aplicable_segun_regla"
			case "regla_ausente", "catalogo_incompatible", "acreditacion_pendiente", "aplicabilidad_pendiente", "dato_pendiente":
				detalle.Validez = "pendiente_revision"
			default:
				detalle.Validez = "no_aplicable_segun_regla"
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
		// La política de bases fija la prioridad. Una referencia solo ordena
		// resultados iguales cuando ningún tope obliga a elegir entre ellos.
		sort.Slice(indices, func(i, j int) bool {
			cmp := compararPrioridad(c.SeleccionElementos, regla, e.Meritos[indices[i]], e.Meritos[indices[j]])
			if cmp != 0 {
				return cmp > 0
			}
			return e.Meritos[indices[i]].Referencia < e.Meritos[indices[j]].Referencia
		})
		if v2 && prioridadAmbiguaBajoTope(c.SeleccionElementos, regla, e.Meritos, indices) {
			r.Estado = "bloqueado"
			r.Incidencias = append(r.Incidencias, Incidencia{e.Meritos[indices[0]].Referencia, "prioridad_bases_insuficiente"})
			r.Reglas, r.Secciones = []DetalleRegla{}, []DetalleSeccion{}
			r.SumaSecciones, r.Total = nil, nil
			for i := range r.Meritos {
				r.Meritos[i].PuntosProvisionales = nil
				r.Meritos[i].CodigoPuntos = ""
				if r.Meritos[i].Codigo == "admitido" {
					r.Meritos[i].Validez = "pendiente_revision"
				}
			}
			return r, nil
		}
		unidades := cero
		antiguas, recientes := cero, cero
		admitidos := uint32(0)
		for _, idx := range indices {
			if regla.MaximoElementos != nil && admitidos >= *regla.MaximoElementos {
				r.Meritos[idx].Codigo = "maximo_elementos"
				if v2 {
					r.Meritos[idx].Validez = "no_aplicable_segun_regla"
				}
				continue
			}
			unidades, err = unidades.Sumar(*e.Meritos[idx].Unidades)
			if err != nil {
				return Resultado{}, err
			}
			admitidos++
			if v2 {
				consumidas := unidades
				if regla.Temporal != nil {
					consumidas, err = antiguas.Sumar(recientes)
					if err != nil {
						return Resultado{}, err
					}
				} else {
					consumidas, err = consumidas.Restar(*e.Meritos[idx].Unidades)
					if err != nil {
						return Resultado{}, err
					}
				}
				computar := *e.Meritos[idx].Unidades
				if regla.MaximoUnidades != nil {
					resto, restaErr := regla.MaximoUnidades.Restar(consumidas)
					if restaErr != nil {
						return Resultado{}, restaErr
					}
					if cmpResto, _ := resto.Comparar(cero); cmpResto <= 0 {
						r.Meritos[idx].Codigo = "maximo_unidades"
						r.Meritos[idx].Validez = "no_aplicable_segun_regla"
						continue
					}
					if cmp, _ := computar.Comparar(resto); cmp > 0 {
						computar = resto
					}
				}
				m := e.Meritos[idx]
				tarifa := *regla.PuntosPorUnidad
				if regla.Temporal != nil {
					if fechaAnterior(regla.Temporal.DesdeBonificacion, *m.FechaObtencion) || fechasIguales(regla.Temporal.DesdeBonificacion, *m.FechaObtencion) {
						recientes, err = recientes.Sumar(computar)
						tarifa = regla.Temporal.PuntosRecientes
					} else {
						antiguas, err = antiguas.Sumar(computar)
					}
				}
				if err != nil {
					return Resultado{}, err
				}
				// El desglose solo publica micropuntos individuales exactos.
				// Las fracciones se acumulan antes del único redondeo de regla.
				exactos := productoPuntosRacional(tarifa, computar)
				if exactos.IsInt() && exactos.Num().Cmp(big.NewInt(baremacion.MaximoMicropuntos)) <= 0 {
					p, pErr := baremacion.PuntosDesdeMicropuntos(exactos.Num().Int64())
					if pErr != nil {
						return Resultado{}, pErr
					}
					r.Meritos[idx].PuntosProvisionales = &p
					r.Meritos[idx].CodigoPuntos = "antes_topes"
				} else {
					r.Meritos[idx].CodigoPuntos = "acumulado_por_regla"
				}
			}
		}
		computadas := unidades
		if regla.MaximoUnidades != nil {
			cmp, _ := computadas.Comparar(*regla.MaximoUnidades)
			if cmp > 0 {
				computadas = *regla.MaximoUnidades
			}
		}
		var bruto baremacion.Puntos
		if v2 && regla.Temporal != nil {
			bruto, err = redondearDosTarifas(*regla.PuntosPorUnidad, antiguas, regla.Temporal.PuntosRecientes, recientes, regla.Redondeo)
		} else {
			bruto, err = regla.PuntosPorUnidad.MultiplicarRedondeado(computadas, regla.Redondeo)
		}
		if err != nil {
			return Resultado{}, err
		}
		puntos, err := aplicarTopeOpcional(bruto, regla.MaximoPuntos)
		if err != nil {
			return Resultado{}, err
		}
		detalleRegla := DetalleRegla{Clave: regla.Clave, SeccionClave: regla.SeccionClave, Definicion: regla.Definicion, ElementosAdmitidos: admitidos, UnidadesAdmitidas: unidades, UnidadesComputadas: computadas, PuntosPorUnidad: *regla.PuntosPorUnidad, Redondeo: regla.Redondeo, BrutoRedondeado: bruto, MaximoPuntos: regla.MaximoPuntos, Puntos: puntos}
		if v2 && regla.Temporal != nil {
			detalleRegla.UnidadesRecientes = &recientes
			detalleRegla.TarifaReciente = &regla.Temporal.PuntosRecientes
		}
		r.Reglas = append(r.Reglas, detalleRegla)
		puntosSeccion[regla.SeccionClave], err = puntosSeccion[regla.SeccionClave].Sumar(puntos)
		if err != nil {
			return Resultado{}, err
		}
	}
	suma := baremacion.Puntos{}
	for _, s := range c.Secciones {
		bruto := puntosSeccion[s.Clave]
		puntos, err := aplicarTopeOpcional(bruto, s.MaximoPuntos)
		if err != nil {
			return Resultado{}, err
		}
		r.Secciones = append(r.Secciones, DetalleSeccion{s.Clave, s.Definicion, bruto, s.MaximoPuntos, puntos})
		suma, err = suma.Sumar(puntos)
		if err != nil {
			return Resultado{}, err
		}
	}
	total, err := aplicarTopeOpcional(suma, c.MaximoTotal)
	if err != nil {
		return Resultado{}, err
	}
	r.SumaSecciones = &suma
	r.Total = &total
	return r, nil
}

func conceptoRelacionado(conceptos []string, concepto string) bool {
	indice := sort.SearchStrings(conceptos, concepto)
	return indice < len(conceptos) && conceptos[indice] == concepto
}

func fechaAnterior(primera, segunda baremacion.FechaCivil) bool {
	cmp, _ := primera.Comparar(segunda)
	return cmp < 0
}

func fechasIguales(primera, segunda baremacion.FechaCivil) bool {
	cmp, _ := primera.Comparar(segunda)
	return cmp == 0
}

func aplicarTopeOpcional(puntos baremacion.Puntos, maximo *baremacion.Puntos) (baremacion.Puntos, error) {
	if maximo == nil {
		return puntos, nil
	}
	resultado, err := baremacion.AplicarTope(puntos, *maximo)
	if err != nil {
		return baremacion.Puntos{}, err
	}
	return resultado.Resultado(), nil
}

func racionalGrande(unidades baremacion.Racional) *big.Rat {
	return new(big.Rat).SetFrac(big.NewInt(unidades.Numerador()), big.NewInt(unidades.Denominador()))
}

func productoPuntosRacional(tarifa baremacion.Puntos, unidades baremacion.Racional) *big.Rat {
	numerador := new(big.Int).Mul(big.NewInt(tarifa.Micropuntos()), big.NewInt(unidades.Numerador()))
	return new(big.Rat).SetFrac(numerador, big.NewInt(unidades.Denominador()))
}

func puntuacionPotencial(regla Regla, merito Merito) *big.Rat {
	tarifa := *regla.PuntosPorUnidad
	if regla.Temporal != nil && !fechaAnterior(*merito.FechaObtencion, regla.Temporal.DesdeBonificacion) {
		tarifa = regla.Temporal.PuntosRecientes
	}
	return productoPuntosRacional(tarifa, *merito.Unidades)
}

func compararPrioridad(modo string, regla Regla, izquierda, derecha Merito) int {
	if modo == "mayor_puntuacion" {
		return puntuacionPotencial(regla, izquierda).Cmp(puntuacionPotencial(regla, derecha))
	}
	cmp, _ := izquierda.Unidades.Comparar(*derecha.Unidades)
	return cmp
}

// prioridadAmbiguaBajoTope rechaza un grupo de igual prioridad cuando el
// límite obligaría a escoger una identidad. No consulta ni usa referencias.
func prioridadAmbiguaBajoTope(modo string, regla Regla, meritos []Merito, indices []int) bool {
	consumidas := new(big.Rat)
	var maximo *big.Rat
	if regla.MaximoUnidades != nil {
		maximo = racionalGrande(*regla.MaximoUnidades)
	}
	for inicio := 0; inicio < len(indices); {
		if regla.MaximoElementos != nil && inicio >= int(*regla.MaximoElementos) {
			break
		}
		if maximo != nil && consumidas.Cmp(maximo) >= 0 {
			break
		}
		fin := inicio + 1
		for fin < len(indices) && compararPrioridad(modo, regla, meritos[indices[inicio]], meritos[indices[fin]]) == 0 {
			fin++
		}
		grupo := new(big.Rat)
		for _, idx := range indices[inicio:fin] {
			grupo.Add(grupo, racionalGrande(*meritos[idx].Unidades))
		}
		if fin-inicio > 1 {
			if regla.MaximoElementos != nil && fin > int(*regla.MaximoElementos) {
				return true
			}
			if maximo != nil && new(big.Rat).Add(consumidas, grupo).Cmp(maximo) > 0 {
				return true
			}
		}
		consumidas.Add(consumidas, grupo)
		inicio = fin
	}
	return false
}

// redondearDosTarifas suma las dos tasas exactas y aplica una sola decisión
// de redondeo, tal como declara el conjunto en momento_redondeo=por_regla.
func redondearDosTarifas(base baremacion.Puntos, unidadesBase baremacion.Racional,
	reciente baremacion.Puntos, unidadesRecientes baremacion.Racional, modo baremacion.ModoRedondeo) (baremacion.Puntos, error) {
	total := new(big.Rat).Add(productoPuntosRacional(base, unidadesBase), productoPuntosRacional(reciente, unidadesRecientes))
	cociente, resto := new(big.Int), new(big.Int)
	cociente.QuoRem(total.Num(), total.Denom(), resto)
	incrementar := false
	switch modo {
	case baremacion.RedondeoExacto:
		if resto.Sign() != 0 {
			return baremacion.Puntos{}, baremacion.ErrResultadoNoExacto
		}
	case baremacion.RedondeoTruncar:
	case baremacion.RedondeoHaciaArriba:
		incrementar = resto.Sign() != 0
	case baremacion.RedondeoMitadArriba:
		incrementar = new(big.Int).Lsh(resto, 1).Cmp(total.Denom()) >= 0
	case baremacion.RedondeoMitadAlPar:
		cmp := new(big.Int).Lsh(resto, 1).Cmp(total.Denom())
		incrementar = cmp > 0 || cmp == 0 && cociente.Bit(0) == 1
	default:
		return baremacion.Puntos{}, fallo("redondeo_invalido")
	}
	if incrementar {
		cociente.Add(cociente, big.NewInt(1))
	}
	if cociente.Sign() < 0 || cociente.Cmp(big.NewInt(baremacion.MaximoMicropuntos)) > 0 {
		return baremacion.Puntos{}, baremacion.ErrDesbordamiento
	}
	return baremacion.PuntosDesdeMicropuntos(cociente.Int64())
}
