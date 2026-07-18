package importacionconvoca

// ClonarLote crea una copia defensiva completa para las fronteras de
// persistencia. Evita que slices o punteros devueltos alteren el acta durable.
func ClonarLote(origen LoteValidado) LoteValidado {
	destino := origen
	destino.Acta.Incidencias = append([]Incidencia(nil), origen.Acta.Incidencias...)
	destino.Aceptadas = make([]FilaAceptada, len(origen.Aceptadas))
	for i := range origen.Aceptadas {
		destino.Aceptadas[i] = clonarFila(origen.Aceptadas[i])
	}
	return destino
}

func clonarFila(origen FilaAceptada) FilaAceptada {
	destino := origen
	if origen.Resumen != nil {
		resumen := *origen.Resumen
		destino.Resumen = &resumen
	}
	if origen.Detalle != nil {
		detalle := *origen.Detalle
		destino.Detalle = &detalle
	}
	return destino
}
