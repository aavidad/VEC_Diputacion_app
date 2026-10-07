package domain

import (
	"bytes"
	"encoding/json"
	"sort"

	b "vec-diputacion-granada/internal/shared/baremacion"
)

// ValorCambioConfiguracion contiene una sola variante tipada. Los punteros
// conservan ceros y false; nil en un lado de CambioConfiguracion significa ausencia.
type ValorCambioConfiguracion struct {
	Tipo       string        `json:"tipo"`
	Texto      *string       `json:"texto,omitempty"`
	Entero     *int64        `json:"entero,omitempty"`
	Booleano   *bool         `json:"booleano,omitempty"`
	Puntos     *b.Puntos     `json:"puntos,omitempty"`
	Fecha      *b.FechaCivil `json:"fecha,omitempty"`
	Racional   *b.Racional   `json:"racional,omitempty"`
	Tipos      *[]string     `json:"tipos,omitempty"`
	Conversion *Conversion   `json:"conversion,omitempty"`
	Regla      *Regla        `json:"regla,omitempty"`
	Tramo      *Tramo        `json:"tramo,omitempty"`
}

func (v ValorCambioConfiguracion) MarshalJSON() ([]byte, error) {
	activas := 0
	for _, activa := range []bool{v.Texto != nil, v.Entero != nil, v.Booleano != nil, v.Puntos != nil, v.Fecha != nil, v.Racional != nil, v.Tipos != nil, v.Conversion != nil, v.Regla != nil, v.Tramo != nil} {
		if activa {
			activas++
		}
	}
	valida := false
	switch v.Tipo {
	case "texto":
		valida = v.Texto != nil
	case "entero":
		valida = v.Entero != nil
	case "booleano":
		valida = v.Booleano != nil
	case "puntos":
		valida = v.Puntos != nil
	case "fecha":
		valida = v.Fecha != nil
	case "racional":
		valida = v.Racional != nil
	case "tipos":
		valida = v.Tipos != nil
	case "conversion":
		valida = v.Conversion != nil
	case "regla":
		valida = v.Regla != nil
	case "tramo":
		valida = v.Tramo != nil
	}
	if activas != 1 || !valida {
		return nil, fallo("valor_cambio_invalido", "comparacion")
	}
	type material ValorCambioConfiguracion
	return json.Marshal(material(v))
}

type CambioConfiguracion struct {
	Ambito   string                    `json:"ambito"`
	ReglaID  string                    `json:"regla_id,omitempty"`
	TramoID  string                    `json:"tramo_id,omitempty"`
	Campo    string                    `json:"campo"`
	Tipo     string                    `json:"tipo"`
	Anterior *ValorCambioConfiguracion `json:"anterior"`
	Nuevo    *ValorCambioConfiguracion `json:"nuevo"`
}

type ReferenciaConfiguracion struct {
	Version      string `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type DiferenciaConfiguracion struct {
	Esquema         string                  `json:"esquema"`
	Alcance         string                  `json:"alcance"`
	ConvocatoriaRef string                  `json:"convocatoria_ref"`
	Anterior        ReferenciaConfiguracion `json:"anterior"`
	Nuevo           ReferenciaConfiguracion `json:"nuevo"`
	Cambios         []CambioConfiguracion   `json:"cambios"`
}

type claveConfiguracion struct{ ambito, regla, tramo string }
type campoConfiguracion struct {
	nombre string
	valor  *ValorCambioConfiguracion
}
type elementoConfiguracion struct {
	material *ValorCambioConfiguracion
	campos   []campoConfiguracion
}

// CompararConfiguraciones valida dos configuraciones de la misma convocatoria.
// Version es un token opaco: no ordena ni acredita sucesión administrativa.
// No calcula puntos, consulta personas ni publica las configuraciones.
func CompararConfiguraciones(anterior, nuevo Configuracion) (DiferenciaConfiguracion, error) {
	if err := ValidarConfiguracion(anterior); err != nil {
		return DiferenciaConfiguracion{}, err
	}
	if err := ValidarConfiguracion(nuevo); err != nil {
		return DiferenciaConfiguracion{}, err
	}
	if anterior.ConvocatoriaRef != nuevo.ConvocatoriaRef {
		return DiferenciaConfiguracion{}, fallo("identidad_comparacion_incompatible", "convocatoria_ref")
	}
	anterior, _ = canonicos(anterior, Entrada{})
	nuevo, _ = canonicos(nuevo, Entrada{})
	anterior, nuevo = copiarConfiguracionComparacion(anterior), copiarConfiguracionComparacion(nuevo)
	ha, hn := huella(anterior), huella(nuevo)
	if anterior.Version == nuevo.Version && ha != hn {
		return DiferenciaConfiguracion{}, fallo("version_contradictoria", "version")
	}
	d := DiferenciaConfiguracion{
		Esquema: "vec.provision.diferencia_configuracion.v1", Alcance: "comparacion_reglas",
		ConvocatoriaRef: anterior.ConvocatoriaRef,
		Anterior:        ReferenciaConfiguracion{anterior.Version, ha}, Nuevo: ReferenciaConfiguracion{nuevo.Version, hn},
		Cambios: []CambioConfiguracion{},
	}
	previos, nuevos := proyectarConfiguracion(anterior), proyectarConfiguracion(nuevo)
	claves := map[claveConfiguracion]bool{}
	for k := range previos {
		claves[k] = true
	}
	for k := range nuevos {
		claves[k] = true
	}
	for k := range claves {
		a, existeA := previos[k]
		n, existeN := nuevos[k]
		if !existeA || !existeN {
			// Los tramos de una regla dada de alta/baja viajan dentro de esa regla.
			if k.ambito == "tramo" {
				_, reglaA := previos[claveConfiguracion{ambito: "regla", regla: k.regla}]
				_, reglaN := nuevos[claveConfiguracion{ambito: "regla", regla: k.regla}]
				if !reglaA || !reglaN {
					continue
				}
			}
			if err := d.registrar(k, "elemento", a.material, n.material); err != nil {
				return DiferenciaConfiguracion{}, err
			}
			continue
		}
		for i, campo := range a.campos {
			if err := d.registrar(k, campo.nombre, campo.valor, n.campos[i].valor); err != nil {
				return DiferenciaConfiguracion{}, err
			}
		}
	}
	sort.Slice(d.Cambios, func(i, j int) bool {
		a, n := d.Cambios[i], d.Cambios[j]
		if a.Ambito != n.Ambito {
			return a.Ambito < n.Ambito
		}
		if a.ReglaID != n.ReglaID {
			return a.ReglaID < n.ReglaID
		}
		if a.TramoID != n.TramoID {
			return a.TramoID < n.TramoID
		}
		return a.Campo < n.Campo
	})
	return d, nil
}

func (d *DiferenciaConfiguracion) registrar(k claveConfiguracion, campo string, a, n *ValorCambioConfiguracion) error {
	previo, err := json.Marshal(a)
	if err != nil {
		return err
	}
	nuevo, err := json.Marshal(n)
	if err != nil {
		return err
	}
	if bytes.Equal(previo, nuevo) {
		return nil
	}
	tipo := "modificacion"
	if a == nil {
		tipo = "alta"
	}
	if n == nil {
		tipo = "baja"
	}
	d.Cambios = append(d.Cambios, CambioConfiguracion{k.ambito, k.regla, k.tramo, campo, tipo, a, n})
	return nil
}

func copiarConfiguracionComparacion(c Configuracion) Configuracion {
	// canonicos ya copió las colecciones. Copiamos también los opcionales para
	// que editar el resultado no cambie la configuración del llamante.
	for i := range c.Reglas {
		r := &c.Reglas[i]
		if r.Conversion != nil {
			x := *r.Conversion
			r.Conversion = &x
		}
		if r.HorasMinimas != nil {
			x := *r.HorasMinimas
			r.HorasMinimas = &x
		}
	}
	return c
}
func textoConfiguracion(x string) *ValorCambioConfiguracion {
	return &ValorCambioConfiguracion{Tipo: "texto", Texto: &x}
}
func enteroConfiguracion(x int64) *ValorCambioConfiguracion {
	return &ValorCambioConfiguracion{Tipo: "entero", Entero: &x}
}
func puntosConfiguracion(x b.Puntos) *ValorCambioConfiguracion {
	return &ValorCambioConfiguracion{Tipo: "puntos", Puntos: &x}
}
func fechaConfiguracion(x b.FechaCivil) *ValorCambioConfiguracion {
	return &ValorCambioConfiguracion{Tipo: "fecha", Fecha: &x}
}
func booleanoConfiguracion(x bool) *ValorCambioConfiguracion {
	return &ValorCambioConfiguracion{Tipo: "booleano", Booleano: &x}
}
func conversionConfiguracion(x *Conversion) *ValorCambioConfiguracion {
	if x == nil {
		return nil
	}
	return &ValorCambioConfiguracion{Tipo: "conversion", Conversion: x}
}
func racionalConfiguracion(x *b.Racional) *ValorCambioConfiguracion {
	if x == nil {
		return nil
	}
	return &ValorCambioConfiguracion{Tipo: "racional", Racional: x}
}
func proyectarConfiguracion(c Configuracion) map[claveConfiguracion]elementoConfiguracion {
	out := map[claveConfiguracion]elementoConfiguracion{}
	out[claveConfiguracion{ambito: "configuracion"}] = elementoConfiguracion{campos: []campoConfiguracion{
		{"bases_ref", textoConfiguracion(c.BasesRef)},
		{"cobertura_requerida", textoConfiguracion(c.CoberturaRequerida)},
		{"fecha_corte", fechaConfiguracion(c.FechaCorte)},
		{"ventana_desde", fechaConfiguracion(c.VentanaDesde)},
		{"maximo_total", puntosConfiguracion(c.MaximoTotal)},
	}}
	for _, r := range c.Reglas {
		tipos := append([]string{}, r.Tipos...)
		out[claveConfiguracion{ambito: "regla", regla: r.ID}] = elementoConfiguracion{
			material: &ValorCambioConfiguracion{Tipo: "regla", Regla: &r},
			campos: []campoConfiguracion{
				{"familia", textoConfiguracion(string(r.Familia))},
				{"referencia_base", textoConfiguracion(r.ReferenciaBase)},
				{"ventana_desde", textoConfiguracion(r.VentanaDesde)},
				{"coeficiente", puntosConfiguracion(r.Coeficiente)},
				{"maximo", puntosConfiguracion(r.Maximo)},
				{"redondeo", textoConfiguracion(string(r.Redondeo))},
				{"diferencia", textoConfiguracion(r.Diferencia)},
				{"min_diferencia", enteroConfiguracion(int64(r.MinDiferencia))},
				{"max_diferencia", enteroConfiguracion(int64(r.MaxDiferencia))},
				{"agrupacion", textoConfiguracion(r.Agrupacion)},
				{"tipos", &ValorCambioConfiguracion{Tipo: "tipos", Tipos: &tipos}},
				{"conversion", conversionConfiguracion(r.Conversion)},
				{"jornada", textoConfiguracion(r.Jornada)},
				{"solapes", textoConfiguracion(r.Solapes)},
				{"horas_minimas", racionalConfiguracion(r.HorasMinimas)},
				{"excluir_requisito", booleanoConfiguracion(r.ExcluirRequisito)},
				{"seleccion_elementos", textoConfiguracion(r.SeleccionElementos)},
				{"maximo_elementos", enteroConfiguracion(int64(r.MaximoElementos))},
				{"permanencia_politica", textoConfiguracion(r.PermanenciaPolitica)},
				{"tipo_provisional", textoConfiguracion(r.TipoProvisional)},
				{"factor_provisional_numerador", enteroConfiguracion(r.FactorProvisionalNumerador)},
				{"factor_provisional_denominador", enteroConfiguracion(r.FactorProvisionalDenominador)},
			},
		}
		for _, t := range r.Tramos {
			out[claveConfiguracion{"tramo", r.ID, t.ID}] = elementoConfiguracion{
				material: &ValorCambioConfiguracion{Tipo: "tramo", Tramo: &t},
				campos: []campoConfiguracion{
					{"min_diferencia", enteroConfiguracion(int64(t.MinDiferencia))},
					{"max_diferencia", enteroConfiguracion(int64(t.MaxDiferencia))},
					{"coeficiente", puntosConfiguracion(t.Coeficiente)},
					{"maximo", puntosConfiguracion(t.Maximo)},
				},
			}
		}
	}
	return out
}
