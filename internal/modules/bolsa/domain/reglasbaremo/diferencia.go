package reglasbaremo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"vec-diputacion-granada/internal/shared/baremacion"
)

// TipoValorCambioExperiencia cierra las variantes de este contrato de salida.
type TipoValorCambioExperiencia string

// ValorCambioExperiencia usa exactamente una variante. Los punteros conservan
// el cero; la ausencia de un lado se representa fuera de la union, con nil.
// No es configuracion de entrada ni un interprete de datos JSON.
type ValorCambioExperiencia struct {
	Tipo           TipoValorCambioExperiencia          `json:"tipo"`
	Texto          *string                             `json:"texto,omitempty"`
	Entero         *uint32                             `json:"entero,omitempty"`
	Puntos         *baremacion.Puntos                  `json:"puntos,omitempty"`
	Fecha          *baremacion.FechaCivil              `json:"fecha,omitempty"`
	Referencia     *materialReferencia                 `json:"referencia,omitempty"`
	Criterios      *[]materialCriterio                 `json:"criterios,omitempty"`
	UnidadTemporal *materialPoliticaUnidadTemporal     `json:"unidad_temporal,omitempty"`
	Jornada        *materialPoliticaJornada            `json:"jornada,omitempty"`
	Restos         *materialPoliticaRestos             `json:"restos,omitempty"`
	Redondeo       *materialPoliticaRedondeo           `json:"redondeo,omitempty"`
	Coincidencia   *materialPoliticaCoincidenciaReglas `json:"coincidencia_reglas,omitempty"`
	Solape         *materialPoliticaSolape             `json:"solape,omitempty"`
	Reparto        *materialPoliticaRepartoExceso      `json:"reparto_exceso,omitempty"`
	LimiteUnidades *materialLimiteUnidades             `json:"maximo_unidades,omitempty"`
	LimitePuntos   *materialLimitePuntos               `json:"maximo_puntos,omitempty"`
	Seccion        *materialSeccion                    `json:"seccion,omitempty"`
	Grupo          *materialGrupoConcurrencia          `json:"grupo,omitempty"`
	Regla          *materialReglaExperiencia           `json:"regla,omitempty"`
}

func (v ValorCambioExperiencia) MarshalJSON() ([]byte, error) {
	activas := 0
	for _, activa := range []bool{v.Texto != nil, v.Entero != nil, v.Puntos != nil, v.Fecha != nil, v.Referencia != nil, v.Criterios != nil, v.UnidadTemporal != nil, v.Jornada != nil, v.Restos != nil, v.Redondeo != nil, v.Coincidencia != nil, v.Solape != nil, v.Reparto != nil, v.LimiteUnidades != nil, v.LimitePuntos != nil, v.Seccion != nil, v.Grupo != nil, v.Regla != nil} {
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
	case "puntos":
		valida = v.Puntos != nil
	case "fecha":
		valida = v.Fecha != nil
	case "referencia":
		valida = v.Referencia != nil
	case "criterios":
		valida = v.Criterios != nil
	case "unidad_temporal":
		valida = v.UnidadTemporal != nil
	case "jornada":
		valida = v.Jornada != nil
	case "restos":
		valida = v.Restos != nil
	case "redondeo":
		valida = v.Redondeo != nil
	case "coincidencia_reglas":
		valida = v.Coincidencia != nil
	case "solape":
		valida = v.Solape != nil
	case "reparto_exceso":
		valida = v.Reparto != nil
	case "maximo_unidades":
		valida = v.LimiteUnidades != nil
	case "maximo_puntos":
		valida = v.LimitePuntos != nil
	case "seccion":
		valida = v.Seccion != nil
	case "grupo":
		valida = v.Grupo != nil
	case "regla":
		valida = v.Regla != nil
	}
	if activas != 1 || !valida {
		return nil, nuevoError("comparacion.valor", CodigoValorInvalido)
	}
	type materialValor ValorCambioExperiencia
	return json.Marshal(materialValor(v))
}

// CambioReglasExperiencia conserva campos tipados o un elemento completo en
// alta/baja. Criterios y politicas se muestran integros, sin interpretarlos.
type CambioReglasExperiencia struct {
	Ambito   string                  `json:"ambito"`
	Clave    string                  `json:"clave,omitempty"`
	Campo    string                  `json:"campo"`
	Tipo     string                  `json:"tipo"`
	Anterior *ValorCambioExperiencia `json:"anterior"`
	Nuevo    *ValorCambioExperiencia `json:"nuevo"`
}

type DiferenciaReglasExperiencia struct {
	Esquema         string                    `json:"esquema"`
	Alcance         string                    `json:"alcance"`
	ConvocatoriaRef string                    `json:"convocatoria_ref"`
	ExpedienteRef   string                    `json:"expediente_ref"`
	Anterior        materialReferencia        `json:"anterior"`
	Nuevo           materialReferencia        `json:"nuevo"`
	Cambios         []CambioReglasExperiencia `json:"cambios"`
}

// CompararConjuntos no calcula ni publica reglas. Exige el mismo proceso y
// una version mayor o identica byte a byte, sin acreditar sucesion juridica.
func CompararConjuntos(anterior, nuevo ConjuntoReglasBaremo) (DiferenciaReglasExperiencia, error) {
	a, err := anterior.RepresentacionCanonica()
	if err != nil {
		return DiferenciaReglasExperiencia{}, err
	}
	n, err := nuevo.RepresentacionCanonica()
	if err != nil {
		return DiferenciaReglasExperiencia{}, err
	}
	ia, in := anterior.identidad, nuevo.identidad
	if ia.referencia != in.referencia || ia.convocatoriaRef != in.convocatoriaRef || ia.expedienteRef != in.expedienteRef || in.version < ia.version {
		return DiferenciaReglasExperiencia{}, nuevoError("comparacion.identidad_version", CodigoInvarianteQuebrada)
	}
	if ia.version == in.version && !bytes.Equal(a, n) {
		return DiferenciaReglasExperiencia{}, nuevoError("comparacion.version", CodigoHuellaNoCoincide)
	}
	huella := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	ra := materialReferencia{ia.referencia, ia.version, huella(a)}
	rn := materialReferencia{in.referencia, in.version, huella(n)}
	if err := validarDependenciasComparacionExperiencia(anterior, nuevo, ra, rn); err != nil {
		return DiferenciaReglasExperiencia{}, err
	}
	d := DiferenciaReglasExperiencia{Esquema: "vec.bolsa.diferencia_reglas_experiencia.v1", Alcance: "comparacion_reglas", ConvocatoriaRef: ia.convocatoriaRef, ExpedienteRef: ia.expedienteRef, Anterior: ra, Nuevo: rn, Cambios: []CambioReglasExperiencia{}}
	previos, nuevos := proyectarComparacionExperiencia(anterior), proyectarComparacionExperiencia(nuevo)
	claves := map[claveComparacionExperiencia]bool{}
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
			if err := d.registrar(k, "elemento", p.material, n.material); err != nil {
				return DiferenciaReglasExperiencia{}, err
			}
			continue
		}
		for i, campo := range p.campos {
			if err := d.registrar(k, campo.nombre, campo.valor, n.campos[i].valor); err != nil {
				return DiferenciaReglasExperiencia{}, err
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

type claveComparacionExperiencia struct{ ambito, clave string }
type campoComparacionExperiencia struct {
	nombre string
	valor  *ValorCambioExperiencia
}
type objetoComparacionExperiencia struct {
	material *ValorCambioExperiencia
	campos   []campoComparacionExperiencia
}

// La proyeccion enumera campos de tipos cerrados; no recorre JSON arbitrario
// ni ejecuta expresiones. Orden, prioridad y politicas conservan su semantica.
func proyectarComparacionExperiencia(c ConjuntoReglasBaremo) map[claveComparacionExperiencia]objetoComparacionExperiencia {
	p := map[claveComparacionExperiencia]objetoComparacionExperiencia{
		{ambito: "conjunto"}: {campos: []campoComparacionExperiencia{{"bases", &ValorCambioExperiencia{Tipo: "referencia", Referencia: &materialReferencia{c.bases.referencia, c.bases.version, c.bases.huellaSHA256}}}, {"fecha_corte_inclusiva", &ValorCambioExperiencia{Tipo: "fecha", Fecha: &c.fechaCorteInclusiva}}}},
	}
	for _, s := range c.secciones {
		m := materialSeccion{s.clave, materialDeReferencia(s.definicion), s.orden, s.puntosMinimos, s.puntosMaximos}
		p[claveComparacionExperiencia{"seccion", s.clave}] = objetoComparacionExperiencia{&ValorCambioExperiencia{Tipo: "seccion", Seccion: &m}, []campoComparacionExperiencia{
			{"definicion", &ValorCambioExperiencia{Tipo: "referencia", Referencia: &m.Definicion}}, {"orden", &ValorCambioExperiencia{Tipo: "entero", Entero: &m.Orden}}, {"puntos_minimos", &ValorCambioExperiencia{Tipo: "puntos", Puntos: &m.PuntosMinimos}}, {"puntos_maximos", &ValorCambioExperiencia{Tipo: "puntos", Puntos: &m.PuntosMaximos}},
		}}
	}
	for _, g := range c.gruposConcurrencia {
		m := materialDeGrupoConcurrencia(g)
		var reparto *ValorCambioExperiencia
		if m.RepartoExceso != nil {
			reparto = &ValorCambioExperiencia{Tipo: "reparto_exceso", Reparto: m.RepartoExceso}
		}
		p[claveComparacionExperiencia{"grupo", g.clave}] = objetoComparacionExperiencia{&ValorCambioExperiencia{Tipo: "grupo", Grupo: &m}, []campoComparacionExperiencia{
			{"definicion", &ValorCambioExperiencia{Tipo: "referencia", Referencia: &m.Definicion}}, {"orden", &ValorCambioExperiencia{Tipo: "entero", Entero: &m.Orden}}, {"coincidencia_reglas", &ValorCambioExperiencia{Tipo: "coincidencia_reglas", Coincidencia: &m.CoincidenciaReglas}}, {"solape", &ValorCambioExperiencia{Tipo: "solape", Solape: &m.Solape}}, {"reparto_exceso", reparto},
		}}
	}
	for _, r := range c.reglasExperiencia {
		m := materialDeRegla(r)
		p[claveComparacionExperiencia{"regla", r.clave}] = objetoComparacionExperiencia{&ValorCambioExperiencia{Tipo: "regla", Regla: &m}, []campoComparacionExperiencia{
			{"definicion", &ValorCambioExperiencia{Tipo: "referencia", Referencia: &m.Definicion}}, {"seccion_clave", &ValorCambioExperiencia{Tipo: "texto", Texto: &m.SeccionClave}}, {"orden", &ValorCambioExperiencia{Tipo: "entero", Entero: &m.Orden}}, {"criterios", &ValorCambioExperiencia{Tipo: "criterios", Criterios: &m.Criterios}},
			{"grupo_concurrencia_clave", &ValorCambioExperiencia{Tipo: "texto", Texto: &m.GrupoConcurrenciaClave}}, {"prioridad_concurrencia", &ValorCambioExperiencia{Tipo: "entero", Entero: &m.PrioridadConcurrencia}},
			{"unidad_temporal", &ValorCambioExperiencia{Tipo: "unidad_temporal", UnidadTemporal: &m.UnidadTemporal}}, {"jornada", &ValorCambioExperiencia{Tipo: "jornada", Jornada: &m.Jornada}}, {"restos", &ValorCambioExperiencia{Tipo: "restos", Restos: &m.Restos}}, {"redondeo", &ValorCambioExperiencia{Tipo: "redondeo", Redondeo: &m.Redondeo}},
			{"puntos_por_unidad", &ValorCambioExperiencia{Tipo: "puntos", Puntos: &m.PuntosPorUnidad}}, {"maximo_unidades", &ValorCambioExperiencia{Tipo: "maximo_unidades", LimiteUnidades: &m.MaximoUnidades}}, {"maximo_puntos", &ValorCambioExperiencia{Tipo: "maximo_puntos", LimitePuntos: &m.MaximoPuntos}},
		}}
	}
	return p
}

func validarDependenciasComparacionExperiencia(a, n ConjuntoReglasBaremo, ra, rn materialReferencia) error {
	deps := []materialReferencia{ra, rn}
	for _, c := range []ConjuntoReglasBaremo{a, n} {
		deps = append(deps, materialDeReferencia(c.bases))
		for _, s := range c.secciones {
			deps = append(deps, materialDeReferencia(s.definicion))
		}
		for _, g := range c.gruposConcurrencia {
			deps = append(deps, materialDeReferencia(g.definicion))
		}
		for _, r := range c.reglasExperiencia {
			deps = append(deps, materialDeReferencia(r.definicion))
			for _, criterio := range r.criterios {
				deps = append(deps, materialDeReferencia(criterio.catalogo))
			}
		}
	}
	type identidad struct {
		ref     string
		version uint64
	}
	huellas := map[identidad]string{}
	for _, dep := range deps {
		k := identidad{dep.Referencia, dep.Version}
		if h, ok := huellas[k]; ok && h != dep.HuellaSHA256 {
			return nuevoError("comparacion.dependencia", CodigoHuellaNoCoincide)
		}
		huellas[k] = dep.HuellaSHA256
	}
	return nil
}

func (d *DiferenciaReglasExperiencia) registrar(k claveComparacionExperiencia, campo string, a, n *ValorCambioExperiencia) error {
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
	if campo == "elemento" && a == nil {
		tipo = "alta"
	}
	if campo == "elemento" && n == nil {
		tipo = "baja"
	}
	d.Cambios = append(d.Cambios, CambioReglasExperiencia{k.ambito, k.clave, campo, tipo, a, n})
	return nil
}
