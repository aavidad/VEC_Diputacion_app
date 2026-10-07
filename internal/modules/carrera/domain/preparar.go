package domain

import (
	"sort"
	"strings"
	"time"
)

// Preparar comprueba integridad de datos declarados. No concede elegibilidad ni
// ejecuta reconocimiento, resolución, inscripción o efectos retributivos.
func Preparar(e Escenario) (Preparacion, error) {
	if e.Alcance != AlcanceSintetico || !texto(e.Version) || !fuenteCompleta(e.Fuente) || len(e.Casos) == 0 || len(e.Casos) > 64 {
		return Preparacion{}, ErrEscenario
	}
	out := Preparacion{Alcance: e.Alcance, Version: e.Version, Fuente: e.Fuente, Casos: make([]PreparacionCaso, 0, len(e.Casos)), Pendientes: []string{}}
	vistos := make(map[string]bool)
	for _, c := range e.Casos {
		if !texto(c.Referencia) || vistos[c.Referencia] || !limites(c) {
			return Preparacion{}, ErrEscenario
		}
		vistos[c.Referencia] = true
		if c.Via != "grado" && c.Via != "progresion" && c.Via != "promocion" {
			return Preparacion{}, ErrEscenario
		}
		p := prepararCaso(c)
		out.Casos = append(out.Casos, p)
		for _, key := range p.Pendientes {
			if !contiene(out.Pendientes, key) {
				out.Pendientes = append(out.Pendientes, key)
			}
		}
	}
	return out, nil
}

func prepararCaso(c Caso) PreparacionCaso {
	p := PreparacionCaso{Referencia: c.Referencia, Via: c.Via, PersonaNombre: c.PersonaNombre, Regimen: c.Regimen, EstadoGlobal: "pendiente", NivelPuesto: copiarNumero(c.NivelPuesto), GradoPersonal: copiarNumero(c.GradoPersonal), Fuentes: append([]Fuente{}, c.Fuentes...), Comprobaciones: []Comprobacion{}, Pendientes: []string{}}
	p.Antecedentes = copiarAntecedentes(c)
	add := func(key, estado, motivo string) {
		p.Comprobaciones = append(p.Comprobaciones, Comprobacion{"carrera.comprobacion." + key, estado, "carrera.motivo." + motivo})
		if estado != "disponible" {
			p.Pendientes = append(p.Pendientes, "carrera.pendiente."+key)
		}
	}
	check := func(key string, ok bool, motivo string) {
		if ok {
			add(key, "disponible", "disponible")
		} else {
			add(key, "pendiente", motivo)
		}
	}
	datos := texto(c.PersonaNombre) && texto(c.Regimen)
	if c.Via == "grado" {
		datos = datos && numero(c.NivelPuesto) && numero(c.GradoPersonal) && texto(c.GrupoSubgrupo)
	}
	if c.Via == "progresion" {
		datos = datos && texto(c.GrupoProfesional)
	}
	if c.Via == "promocion" {
		datos = datos && (texto(c.GrupoSubgrupo) || texto(c.GrupoProfesional))
	}
	check("datos", datos, "datos_incompletos")
	fuentes := len(c.Fuentes) > 0
	vistos := make(map[string]bool)
	for _, f := range c.Fuentes {
		fuentes = fuentes && fuenteCompleta(f) && !vistos[f.Referencia]
		vistos[f.Referencia] = true
	}
	check("fuentes", fuentes, "fuente_incompleta")
	politica := texto(c.Politica.Referencia) && texto(c.Politica.Version) && texto(c.Politica.AprobacionReferencia) && fuenteExiste(c.Fuentes, c.Politica.Fuente, c.Politica.Version)
	if !politica || len(c.Politica.Regimenes) == 0 || !texto(c.Regimen) {
		add("compatibilidad", "pendiente", "regimen_pendiente")
	} else if !contiene(c.Politica.Regimenes, c.Regimen) {
		add("compatibilidad", "incompatible", "regimen_incompatible")
	} else {
		add("compatibilidad", "disponible", "disponible")
	}
	pm := cotejarPeriodos(c)
	check("periodos", pm == "disponible", pm)
	evidencias := len(c.Evidencias) > 0
	for _, ev := range c.Evidencias {
		evidencias = evidencias && evidenciaCompleta(ev, c.Fuentes)
	}
	check("evidencias", evidencias, "evidencia_incompleta")
	// La aprobación aportada solo permite preparar la referencia; no se autentica
	// aquí y nunca activa una regla jurídica ni cambia el resultado global.
	if politica {
		add("politica", "disponible", "politica_preparada")
	} else {
		add("politica", "pendiente", "politica_pendiente")
	}
	if c.Via == "progresion" {
		check("convenio", c.ConvenioConsolidado && evidenciaCompleta(c.Convenio, c.Fuentes), "convenio_pendiente")
		check("curso_prueba", evidenciaCompleta(c.Curso, c.Fuentes) && evidenciaCompleta(c.Prueba, c.Fuentes) && texto(c.CursoAprobacion) && texto(c.PruebaAprobacion), "curso_prueba_pendiente")
	}
	if c.Via == "promocion" {
		check("convocatoria", evidenciaCompleta(c.Convocatoria, c.Fuentes), "convocatoria_pendiente")
	}
	add("reconocimiento", "pendiente", "resolucion_necesaria")
	add("registro", "pendiente", "registro_necesario")
	return p
}

func cotejarPeriodos(c Caso) string {
	if len(c.Periodos) == 0 {
		return "periodos_incompletos"
	}
	type intervalo struct{ inicio, fin time.Time }
	fechas := make([]intervalo, 0, len(c.Periodos))
	for _, p := range c.Periodos {
		ini, err1 := time.Parse(time.DateOnly, p.Inicio)
		fin, err2 := time.Parse(time.DateOnly, p.Fin)
		if err1 != nil || err2 != nil || ini.Year() < 1 || fin.Year() < 1 || fin.Before(ini) {
			return "fecha_invalida"
		}
		if !evidenciaCompleta(p.Evidencia, c.Fuentes) {
			return "evidencia_incompleta"
		}
		fechas = append(fechas, intervalo{ini, fin})
	}
	sort.Slice(fechas, func(i, j int) bool { return fechas[i].inicio.Before(fechas[j].inicio) })
	for i := 1; i < len(fechas); i++ {
		if !fechas[i].inicio.After(fechas[i-1].fin) {
			return "periodos_solapados"
		}
	}
	return "disponible"
}
func texto(s string) bool          { return len(s) <= 1024 && strings.TrimSpace(s) != "" }
func fuenteCompleta(f Fuente) bool { return texto(f.Referencia) && texto(f.Version) }
func fuenteExiste(fs []Fuente, ref, version string) bool {
	for _, f := range fs {
		if f.Referencia == ref && f.Version == version && fuenteCompleta(f) {
			return true
		}
	}
	return false
}
func evidenciaCompleta(e Evidencia, fs []Fuente) bool {
	return texto(e.Referencia) && fuenteExiste(fs, e.Fuente, e.Version)
}
func numero(n *int) bool { return n != nil && *n >= 0 }
func copiarNumero(n *int) *int {
	if n == nil {
		return nil
	}
	v := *n
	return &v
}
func copiarAntecedentes(c Caso) Antecedentes {
	politica := c.Politica
	politica.Regimenes = append([]string{}, c.Politica.Regimenes...)
	return Antecedentes{
		GrupoSubgrupo: c.GrupoSubgrupo, GrupoProfesional: c.GrupoProfesional,
		Periodos: append([]Periodo{}, c.Periodos...), Evidencias: append([]Evidencia{}, c.Evidencias...),
		Politica: politica, Convenio: c.Convenio, ConvenioConsolidado: c.ConvenioConsolidado,
		Curso: c.Curso, Prueba: c.Prueba, Convocatoria: c.Convocatoria,
		CursoAprobacion: c.CursoAprobacion, PruebaAprobacion: c.PruebaAprobacion,
	}
}
func contiene(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
