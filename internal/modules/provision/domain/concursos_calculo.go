package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

// Calcular no modifica su entrada. Una configuración no aprobada produce una
// simulación local; ninguna puntuación equivale a admisión o adjudicación.
func Calcular(c Configuracion, e Entrada) (Resultado, error) {
	if err := ValidarConfiguracion(c); err != nil {
		return Resultado{}, err
	}
	if err := ValidarEntrada(e); err != nil {
		return Resultado{}, err
	}
	if err := validarSolapes(c, e); err != nil {
		return Resultado{}, err
	}
	c, e = canonicos(c, e)
	salida := Resultado{Estado: "simulacion_local_sin_efectos", VersionMotor: VersionMotor, ConvocatoriaRef: c.ConvocatoriaRef, VersionReglas: c.Version, InstantaneaRef: e.InstantaneaRef, PuestoRef: e.PuestoRef, Completo: true, Incidencias: []string{}, Desglose: []Desglose{}, MaximoTotal: c.MaximoTotal}
	salida.HuellaReglas = huella(c)
	salida.HuellaEntrada = huella(e)
	for _, r := range c.Reglas {
		desglose := Desglose{ReglaID: r.ID, Familia: r.Familia, ReferenciaBase: r.ReferenciaBase, Estado: "calculado", Detalles: []Detalle{}, Maximo: r.Maximo}
		if !contiene(e.Disponibles, r.Familia) || (r.Familia == Grado && e.GradoPersonal == nil) {
			desglose.Estado = "pendiente_dato"
			salida.Completo = false
			salida.Incidencias = append(salida.Incidencias, "dato_no_disponible:"+r.ID)
		} else {
			var err error
			switch r.Familia {
			case Grado:
				desglose.Detalles, desglose.Bruto, err = calcularGrado(r, e)
			case ValoracionTrabajo, Antiguedad, Permanencia:
				desglose.Detalles, desglose.Bruto, err = calcularPeriodos(c, r, e)
			case Cursos:
				desglose.Detalles, desglose.Bruto, err = calcularCursos(c, r, e)
			case Titulaciones:
				desglose.Detalles, desglose.Bruto, err = calcularTitulos(c, r, e)
			}
			if err != nil {
				if nominal, ok := err.(*Error); ok && nominal.Codigo == "politica_pendiente" && r.Familia == Permanencia {
					desglose.Estado = "pendiente_politica"
					salida.Completo = false
					salida.Incidencias = append(salida.Incidencias, "politica_no_admitida:"+r.ID)
					salida.Desglose = append(salida.Desglose, desglose)
					continue
				}
				return Resultado{}, err
			}
			desglose.Resultado = tope(desglose.Bruto, r.Maximo)
			salida.Bruto, err = salida.Bruto.Sumar(desglose.Resultado)
			if err != nil {
				return Resultado{}, err
			}
		}
		salida.Desglose = append(salida.Desglose, desglose)
	}
	if c.CoberturaRequerida == "seis_familias_concurso_v1" {
		for _, familia := range []Familia{ValoracionTrabajo, Grado, Antiguedad, Permanencia, Titulaciones, Cursos} {
			presente := false
			for _, regla := range c.Reglas {
				if regla.Familia == familia {
					presente = true
					break
				}
			}
			if !presente {
				salida.Completo = false
				salida.Incidencias = append(salida.Incidencias, "regla_no_configurada:"+string(familia))
				salida.Desglose = append(salida.Desglose, Desglose{Familia: familia, Estado: "pendiente_regla", Detalles: []Detalle{}})
			}
		}
	}
	if salida.Completo {
		total := tope(salida.Bruto, c.MaximoTotal)
		salida.Total = &total
	}
	salida.HuellaResultado = huella(salida)
	return salida, nil
}
func tope(p, max b.Puntos) b.Puntos {
	cmp, _ := p.Comparar(max)
	if cmp > 0 {
		return max
	}
	return p
}
func entero(n int64) b.Racional { r, _ := b.NuevoRacional(n, 1); return r }
func huella(v any) string {
	datos, _ := json.Marshal(v)
	sum := sha256.Sum256(datos)
	return hex.EncodeToString(sum[:])
}
func tramoPara(r Regla, origen, destino int) (Tramo, error) {
	dif := destino - origen
	if r.Diferencia == "hecho_menos_puesto" {
		dif = -dif
	}
	for _, t := range r.Tramos {
		if dif >= t.MinDiferencia && dif <= t.MaxDiferencia {
			return t, nil
		}
	}
	return Tramo{}, fallo("diferencia_fuera_tabla", r.ID)
}
func calcularGrado(r Regla, e Entrada) ([]Detalle, b.Puntos, error) {
	t, err := tramoPara(r, *e.GradoPersonal, e.NivelPuesto)
	if err != nil {
		return nil, b.Puntos{}, err
	}
	neto := tope(t.Coeficiente, t.Maximo)
	return []Detalle{{HechoID: "grado_personal", EvidenciaRef: e.GradoEvidenciaRef, TramoID: t.ID, Motivo: "computado", Unidades: entero(1), FactorJornada: entero(1), Coeficiente: t.Coeficiente, Bruto: t.Coeficiente, Maximo: t.Maximo, Resultado: neto}}, neto, nil
}
func canonicos(c Configuracion, e Entrada) (Configuracion, Entrada) {
	// Copia de todas las colecciones que se ordenan: ni slice ni mapa del llamante
	// se modifican. La huella no depende del orden de transporte de los hechos.
	c.Reglas = append([]Regla(nil), c.Reglas...)
	sort.Slice(c.Reglas, func(i, j int) bool { return c.Reglas[i].ID < c.Reglas[j].ID })
	for i := range c.Reglas {
		r := &c.Reglas[i]
		r.Tramos = append([]Tramo(nil), r.Tramos...)
		sort.Slice(r.Tramos, func(i, j int) bool { return r.Tramos[i].ID < r.Tramos[j].ID })
		r.Tipos = append([]string(nil), r.Tipos...)
		sort.Strings(r.Tipos)
	}
	e.Disponibles = append([]Familia(nil), e.Disponibles...)
	sort.Slice(e.Disponibles, func(i, j int) bool { return e.Disponibles[i] < e.Disponibles[j] })
	e.Periodos = append([]Periodo{}, e.Periodos...)
	sort.Slice(e.Periodos, func(i, j int) bool { return e.Periodos[i].ID < e.Periodos[j].ID })
	for i := range e.Periodos {
		e.Periodos[i].Familias = append([]Familia(nil), e.Periodos[i].Familias...)
		sort.Slice(e.Periodos[i].Familias, func(a, z int) bool { return e.Periodos[i].Familias[a] < e.Periodos[i].Familias[z] })
	}
	e.Cursos = append([]Curso{}, e.Cursos...)
	sort.Slice(e.Cursos, func(i, j int) bool { return e.Cursos[i].ID < e.Cursos[j].ID })
	e.Cursos = deduplicarCursos(e.Cursos)
	e.Titulaciones = append([]Titulo{}, e.Titulaciones...)
	sort.Slice(e.Titulaciones, func(i, j int) bool { return e.Titulaciones[i].ID < e.Titulaciones[j].ID })
	e.Titulaciones = deduplicarTitulos(e.Titulaciones)
	return c, e
}
func deduplicarCursos(xs []Curso) []Curso {
	out := []Curso{}
	for _, x := range xs {
		if len(out) == 0 || out[len(out)-1].ID != x.ID {
			out = append(out, x)
		}
	}
	return out
}
func deduplicarTitulos(xs []Titulo) []Titulo {
	out := []Titulo{}
	for _, x := range xs {
		if len(out) == 0 || out[len(out)-1].ID != x.ID {
			out = append(out, x)
		}
	}
	return out
}
