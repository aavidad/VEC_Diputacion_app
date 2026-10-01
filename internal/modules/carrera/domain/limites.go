package domain

// Límites técnicos de cardinalidad y longitud, sin umbrales de derecho material.
func limites(c Caso) bool {
	if len(c.Fuentes) > 32 || len(c.Periodos) > 128 || len(c.Evidencias) > 128 || len(c.Politica.Regimenes) > 32 {
		return false
	}
	ss := []string{c.PersonaNombre, c.Regimen, c.GrupoSubgrupo, c.GrupoProfesional, c.Politica.Referencia, c.Politica.Version, c.Politica.Fuente, c.Politica.AprobacionReferencia, c.CursoAprobacion, c.PruebaAprobacion}
	for _, f := range c.Fuentes {
		ss = append(ss, f.Referencia, f.Version)
	}
	for _, p := range c.Periodos {
		ss = append(ss, p.Inicio, p.Fin, p.Evidencia.Referencia, p.Evidencia.Fuente, p.Evidencia.Version)
	}
	for _, ev := range append(append([]Evidencia{}, c.Evidencias...), c.Convenio, c.Curso, c.Prueba, c.Convocatoria) {
		ss = append(ss, ev.Referencia, ev.Fuente, ev.Version)
	}
	ss = append(ss, c.Politica.Regimenes...)
	for _, s := range ss {
		if len(s) > 1024 {
			return false
		}
	}
	return true
}
