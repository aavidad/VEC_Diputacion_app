package application

// ComprobacionCompletitudOrganizacion comprueba únicamente campos y decisiones
// declarados en el paquete. No acredita fuente, catálogos ni acto administrativo.
type ComprobacionCompletitudOrganizacion struct {
	Completa  bool                              `json:"completa"`
	Faltantes []FaltanteCompletitudOrganizacion `json:"faltantes"`
}

// Los valores son códigos del catálogo; nunca contienen hechos ni referencias.
type FaltanteCompletitudOrganizacion struct {
	Clave    string `json:"clave"`
	Esperado string `json:"esperado"`
	Actual   string `json:"actual"`
}

func ComprobarCompletitudPreparacionOrganizacion(p PaquetePreparacionOrganizacion) (PaquetePreparacionOrganizacion, InformeRevisionPreparacionOrganizacion, ComprobacionCompletitudOrganizacion) {
	normalizado, informe := PrepararPaqueteOrganizacion(p)
	r := ComprobacionCompletitudOrganizacion{Faltantes: []FaltanteCompletitudOrganizacion{}}
	agregar := func(clave, esperado, actual string) {
		r.Faltantes = append(r.Faltantes, FaltanteCompletitudOrganizacion{Clave: clave, Esperado: esperado, Actual: actual})
	}
	if !informe.Valido || informe.CoberturaConciliacion == nil || p.Manifiesto.Validar(true) != nil {
		agregar("revision_paquete", "valido", "invalido")
		return normalizado, informe, r
	}
	m := p.Manifiesto
	for _, campo := range []struct{ clave, valor string }{
		{"documento_ref", m.DocumentoRef}, {"custodia_ref", m.CustodiaRef},
		{"diccionario_ref", m.DiccionarioRef}, {"acto_ref", m.ActoRef},
		{"aprobada_en", string(m.AprobadaEn)}, {"publicada_en", string(m.PublicadaEn)},
		{"efectos_desde", string(m.EfectosDesde)},
	} {
		if campo.valor == "" {
			agregar(campo.clave, "presente", "ausente")
		}
	}
	// Validar(false) es el contrato de campos obligatorios del importador. Si
	// cambia, la comprobación cierra aunque no conozca todavía el campo nuevo.
	if m.Validar(false) != nil && len(r.Faltantes) == 0 {
		agregar("manifiesto_invalido", "valido", "invalido")
	}
	for _, resultado := range []string{"sin_decision", "pendiente", "descartada"} {
		if informe.CoberturaConciliacion.RecuentosHechos[resultado] > 0 || decisionAdicionalConResultado(informe.CoberturaConciliacion, resultado) {
			agregar("decision_"+resultado, "vinculada", resultado)
		}
	}
	if !informe.CoberturaConciliacion.Completa && len(r.Faltantes) == 0 {
		agregar("cobertura_conciliacion", "completa", "incompleta")
	}
	r.Completa = len(r.Faltantes) == 0
	return normalizado, informe, r
}

func decisionAdicionalConResultado(c *CoberturaConciliacionOrganizacion, resultado string) bool {
	for _, d := range c.DecisionesAdicionales {
		if d.Resultado == resultado {
			return true
		}
	}
	return false
}
