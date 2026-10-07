package domain

import "slices"

// CopiarProceso entrega un grafo independiente para conservar la definición
// preparada aunque el llamante edite después sus colecciones o reglas.
func CopiarProceso(p ProcesoProvision) ProcesoProvision {
	p.Puestos = slices.Clone(p.Puestos)
	for i := range p.Puestos {
		p.Puestos[i].Requisitos = slices.Clone(p.Puestos[i].Requisitos)
	}
	p.Configuracion.Reglas = slices.Clone(p.Configuracion.Reglas)
	for i := range p.Configuracion.Reglas {
		r := &p.Configuracion.Reglas[i]
		r.Tramos = slices.Clone(r.Tramos)
		r.Tipos = slices.Clone(r.Tipos)
		r.Conversion = copiarValorProceso(r.Conversion)
		r.HorasMinimas = copiarValorProceso(r.HorasMinimas)
	}
	return p
}

// CopiarSolicitud conserva cada proyección de la instantánea sin compartir
// datos mutables con la petición recibida.
func CopiarSolicitud(s SolicitudProvision) SolicitudProvision {
	s.Preferencias = slices.Clone(s.Preferencias)
	s.Valoraciones = slices.Clone(s.Valoraciones)
	for i := range s.Valoraciones {
		v := &s.Valoraciones[i]
		v.Requisitos = slices.Clone(v.Requisitos)
		v.Entrada = copiarEntradaProceso(v.Entrada)
	}
	return s
}

func copiarEntradaProceso(e Entrada) Entrada {
	e.GradoPersonal = copiarValorProceso(e.GradoPersonal)
	e.Disponibles = slices.Clone(e.Disponibles)
	e.Periodos = slices.Clone(e.Periodos)
	for i := range e.Periodos {
		e.Periodos[i].Hasta = copiarValorProceso(e.Periodos[i].Hasta)
		e.Periodos[i].Familias = slices.Clone(e.Periodos[i].Familias)
	}
	e.Cursos = slices.Clone(e.Cursos)
	for i := range e.Cursos {
		e.Cursos[i].VigenteHasta = copiarValorProceso(e.Cursos[i].VigenteHasta)
	}
	e.Titulaciones = slices.Clone(e.Titulaciones)
	return e
}

// Los punteros de estos contratos apuntan sólo a valores escalares o structs
// sin colecciones ni otros punteros: int, fechas, racionales y conversión.
func copiarValorProceso[T any](p *T) *T {
	if p == nil {
		return nil
	}
	copia := *p
	return &copia
}
