package domain

// CopiarEntradaCiclo evita que una rectificación o el consumidor altere la
// instantánea conservada en una valoración anterior.
func CopiarEntradaCiclo(e Entrada) Entrada {
	if e.GradoPersonal != nil {
		v := *e.GradoPersonal
		e.GradoPersonal = &v
	}
	e.Disponibles = append([]Familia(nil), e.Disponibles...)
	if e.Periodos != nil {
		e.Periodos = append([]Periodo{}, e.Periodos...)
	}
	for i := range e.Periodos {
		e.Periodos[i].Familias = append([]Familia(nil), e.Periodos[i].Familias...)
		if e.Periodos[i].Hasta != nil {
			v := *e.Periodos[i].Hasta
			e.Periodos[i].Hasta = &v
		}
	}
	if e.Cursos != nil {
		e.Cursos = append([]Curso{}, e.Cursos...)
	}
	for i := range e.Cursos {
		if e.Cursos[i].VigenteHasta != nil {
			v := *e.Cursos[i].VigenteHasta
			e.Cursos[i].VigenteHasta = &v
		}
	}
	if e.Titulaciones != nil {
		e.Titulaciones = append([]Titulo{}, e.Titulaciones...)
	}
	return e
}

func CopiarConfiguracionCiclo(c Configuracion) Configuracion {
	c.Reglas = append([]Regla(nil), c.Reglas...)
	for i := range c.Reglas {
		r := &c.Reglas[i]
		r.Tramos = append([]Tramo(nil), r.Tramos...)
		r.Tipos = append([]string(nil), r.Tipos...)
		if r.Conversion != nil {
			v := *r.Conversion
			r.Conversion = &v
		}
		if r.HorasMinimas != nil {
			v := *r.HorasMinimas
			r.HorasMinimas = &v
		}
	}
	return c
}

func copiarResultadoCiclo(r Resultado) Resultado {
	r.Incidencias = append([]string{}, r.Incidencias...)
	r.Desglose = append([]Desglose{}, r.Desglose...)
	for i := range r.Desglose {
		r.Desglose[i].Detalles = append([]Detalle{}, r.Desglose[i].Detalles...)
	}
	if r.Total != nil {
		v := *r.Total
		r.Total = &v
	}
	return r
}
