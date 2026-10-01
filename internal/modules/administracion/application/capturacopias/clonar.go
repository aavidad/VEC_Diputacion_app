package capturacopias

import (
	"slices"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func clonarModulos(ms []copias.Modulo) []copias.Modulo {
	r := slices.Clone(ms)
	for i := range r {
		r[i].Migraciones = slices.Clone(r[i].Migraciones)
	}
	return r
}
func clonar(i copias.Inventario) copias.Inventario {
	i.PostgreSQL.Bases = slices.Clone(i.PostgreSQL.Bases)
	i.PostgreSQL.Herramientas = slices.Clone(i.PostgreSQL.Herramientas)
	i.PostgreSQL.Extensiones = slices.Clone(i.PostgreSQL.Extensiones)
	i.PostgreSQL.Almacenes = slices.Clone(i.PostgreSQL.Almacenes)
	i.Release.Binarios = slices.Clone(i.Release.Binarios)
	i.Release.Componentes = slices.Clone(i.Release.Componentes)
	i.Release.EsquemaEsperado = clonarModulos(i.Release.EsquemaEsperado)
	i.Modulos = clonarModulos(i.Modulos)
	return i
}
