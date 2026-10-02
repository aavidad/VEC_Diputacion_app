package calculomeritos

import (
	"bytes"
	"encoding/json"
	"sort"
)

// CambioReglas describe una diferencia exacta. Una clave renombrada produce
// baja y alta; el comparador no deduce equivalencias ni efectos juridicos.
type CambioReglas struct {
	Ambito   string          `json:"ambito"`
	Clave    string          `json:"clave,omitempty"`
	Campo    string          `json:"campo"`
	Tipo     string          `json:"tipo"`
	Anterior json.RawMessage `json:"anterior"`
	Nuevo    json.RawMessage `json:"nuevo"`
}

type DiferenciaReglas struct {
	Esquema         string         `json:"esquema"`
	Alcance         string         `json:"alcance"`
	ConvocatoriaRef string         `json:"convocatoria_ref"`
	ExpedienteRef   string         `json:"expediente_ref"`
	Anterior        Dependencia    `json:"anterior"`
	Nuevo           Dependencia    `json:"nuevo"`
	Cambios         []CambioReglas `json:"cambios"`
}

// CompararConjuntos valida y compara dos instantaneas del mismo proceso.
// Versiones no consecutivas son admisibles; no acredita sucesion administrativa.
func CompararConjuntos(anterior, nuevo Conjunto) (DiferenciaReglas, error) {
	a, err := anterior.RepresentacionCanonica()
	if err != nil {
		return DiferenciaReglas{}, err
	}
	n, err := nuevo.RepresentacionCanonica()
	if err != nil {
		return DiferenciaReglas{}, err
	}
	if anterior.Referencia != nuevo.Referencia || anterior.ConvocatoriaRef != nuevo.ConvocatoriaRef || anterior.ExpedienteRef != nuevo.ExpedienteRef {
		return DiferenciaReglas{}, fallo("identidad_comparacion_incompatible")
	}
	if nuevo.Version < anterior.Version {
		return DiferenciaReglas{}, fallo("orden_versiones_invalido")
	}
	if nuevo.Version == anterior.Version && !bytes.Equal(a, n) {
		return DiferenciaReglas{}, fallo("version_contradictoria")
	}
	dependencias := []Dependencia{
		{anterior.Referencia, anterior.Version, HuellaSHA256(a)},
		{nuevo.Referencia, nuevo.Version, HuellaSHA256(n)},
	}
	for _, c := range []Conjunto{anterior, nuevo} {
		dependencias = append(dependencias, c.Bases)
		for _, s := range c.Secciones {
			dependencias = append(dependencias, s.Definicion)
		}
		for _, r := range c.Reglas {
			dependencias = append(dependencias, r.Definicion, r.Catalogo)
		}
	}
	if err := validarConsistenciaDependencias(dependencias); err != nil {
		return DiferenciaReglas{}, err
	}
	d := DiferenciaReglas{
		Esquema: "vec.bolsa.diferencia_reglas_meritos.v1", Alcance: "comparacion_reglas",
		ConvocatoriaRef: anterior.ConvocatoriaRef, ExpedienteRef: anterior.ExpedienteRef,
		Anterior: Dependencia{anterior.Referencia, anterior.Version, HuellaSHA256(a)},
		Nuevo:    Dependencia{nuevo.Referencia, nuevo.Version, HuellaSHA256(n)}, Cambios: []CambioReglas{},
	}
	// Campos tipados del conjunto, sin identidad, version ni colecciones.
	for _, par := range []struct {
		campo string
		a, n  any
	}{
		{"bases", anterior.Bases, nuevo.Bases},
		{"fecha_corte_inclusiva", anterior.FechaCorteInclusiva, nuevo.FechaCorteInclusiva},
		{"duplicados", anterior.Duplicados, nuevo.Duplicados},
		{"momento_redondeo", anterior.MomentoRedondeo, nuevo.MomentoRedondeo},
		{"seleccion_elementos", anterior.SeleccionElementos, nuevo.SeleccionElementos},
		{"maximo_total", anterior.MaximoTotal, nuevo.MaximoTotal},
	} {
		if err := d.registrar("conjunto", "", par.campo, par.a, par.n); err != nil {
			return DiferenciaReglas{}, err
		}
	}
	// Solo se proyectan tipos ya validados de esta familia, no JSON arbitrario.
	for _, ambito := range []string{"seccion", "regla"} {
		previos, nuevos := map[string]any{}, map[string]any{}
		if ambito == "seccion" {
			for _, s := range anterior.Secciones {
				previos[s.Clave] = s
			}
			for _, s := range nuevo.Secciones {
				nuevos[s.Clave] = s
			}
		} else {
			for _, r := range anterior.Reglas {
				previos[r.Clave] = r
			}
			for _, r := range nuevo.Reglas {
				nuevos[r.Clave] = r
			}
		}
		claves := map[string]bool{}
		for k := range previos {
			claves[k] = true
		}
		for k := range nuevos {
			claves[k] = true
		}
		for k := range claves {
			p, pExiste := previos[k]
			n, nExiste := nuevos[k]
			if !pExiste || !nExiste {
				if err := d.registrar(ambito, k, "elemento", p, n); err != nil {
					return DiferenciaReglas{}, err
				}
				continue
			}
			pb, err := json.Marshal(p)
			if err != nil {
				return DiferenciaReglas{}, err
			}
			nb, err := json.Marshal(n)
			if err != nil {
				return DiferenciaReglas{}, err
			}
			var pc, nc map[string]json.RawMessage
			if err := json.Unmarshal(pb, &pc); err != nil {
				return DiferenciaReglas{}, err
			}
			if err := json.Unmarshal(nb, &nc); err != nil {
				return DiferenciaReglas{}, err
			}
			for campo, valor := range pc {
				if err := d.registrar(ambito, k, campo, valor, nc[campo]); err != nil {
					return DiferenciaReglas{}, err
				}
			}
		}
	}
	sort.Slice(d.Cambios, func(i, j int) bool {
		a, b := d.Cambios[i], d.Cambios[j]
		if a.Ambito != b.Ambito {
			return a.Ambito < b.Ambito
		}
		if a.Clave != b.Clave {
			return a.Clave < b.Clave
		}
		return a.Campo < b.Campo
	})
	return d, nil
}

func (d *DiferenciaReglas) registrar(ambito, clave, campo string, a, n any) error {
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
	d.Cambios = append(d.Cambios, CambioReglas{ambito, clave, campo, tipo, previo, nuevo})
	return nil
}
