package domain

import "fmt"

// Las referencias de este contrato son tokens opacos, no campos de nombre.
func referenciaProceso(s string) bool {
	if !referencia(s) {
		return false
	}
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case ':', '-', '_', '.', '/', '#':
		default:
			return false
		}
	}
	return true
}

// ValidarProceso prepara exclusivamente convocatorias borrador. No consulta
// RPT, autoriza publicación ni transforma el borrador en bases aprobadas.
func ValidarProceso(p ProcesoProvision) error {
	if p.SchemaVersion != VersionProceso || p.Estado != "borrador" || !referenciaProceso(p.Referencia) || !referenciaProceso(p.Version) {
		return fallo("proceso_invalido", "proceso")
	}
	if err := ValidarConfiguracion(p.Configuracion); err != nil {
		return err
	}
	if p.Configuracion.ConvocatoriaRef != p.Referencia {
		return fallo("proceso_no_coincide", "configuracion.convocatoria_ref")
	}
	if len(p.Puestos) == 0 || len(p.Puestos) > 100 {
		return fallo("puestos_invalidos", "puestos")
	}
	vistos, rpt := map[string]bool{}, map[string]bool{}
	for i, puesto := range p.Puestos {
		campo := fmt.Sprintf("puestos.%d", i)
		if !referenciaProceso(puesto.Referencia) || vistos[puesto.Referencia] || !referenciaProceso(puesto.RPTRef) || !referenciaProceso(puesto.RPTVersion) || rpt[puesto.RPTRef] || puesto.Nivel <= 0 || puesto.Nivel > 1000 {
			return fallo("puesto_ofertado_invalido", campo)
		}
		vistos[puesto.Referencia], rpt[puesto.RPTRef] = true, true
		if puesto.Requisitos == nil || len(puesto.Requisitos) > 100 {
			return fallo("requisitos_invalidos", campo+".requisitos")
		}
		rs := map[string]bool{}
		for _, r := range puesto.Requisitos {
			if !referenciaProceso(r.Referencia) || !referenciaProceso(r.Version) || rs[r.Referencia] {
				return fallo("requisito_invalido", campo+".requisitos")
			}
			rs[r.Referencia] = true
		}
	}
	return nil
}

// ValidarSolicitud conserva vínculo exacto a proceso, reglas, empleado,
// instantánea y puestos. No deduce acceso por la puntuación obtenida.
func ValidarSolicitud(p ProcesoProvision, s SolicitudProvision) error {
	if err := ValidarProceso(p); err != nil {
		return err
	}
	if !referenciaProceso(s.Referencia) || !referenciaProceso(s.EmpleadoRef) || s.ProcesoRef != p.Referencia || s.ProcesoVersion != p.Version || s.VersionReglas != p.Configuracion.Version {
		return fallo("solicitud_invalida", "solicitud")
	}
	i := s.Instantanea
	if !referenciaProceso(i.Referencia) || !referenciaProceso(i.Version) || !referenciaProceso(i.FuenteRef) || i.EmpleadoRef != s.EmpleadoRef {
		return fallo("instantanea_invalida", "instantanea")
	}
	if i.CondicionInterna != Cumple {
		return fallo("empleado_interno_no_confirmado", "instantanea.condicion_interna")
	}
	if len(s.Preferencias) == 0 || len(s.Preferencias) > len(p.Puestos) || len(s.Valoraciones) != len(s.Preferencias) {
		return fallo("preferencias_invalidas", "preferencias")
	}
	puestos := map[string]PuestoOfertado{}
	for _, puesto := range p.Puestos {
		puestos[puesto.Referencia] = puesto
	}
	preferencias := map[string]bool{}
	for n, pref := range s.Preferencias {
		if _, ok := puestos[pref.PuestoRef]; !ok || preferencias[pref.PuestoRef] || pref.Orden != n+1 {
			return fallo("preferencia_invalida", "preferencias")
		}
		preferencias[pref.PuestoRef] = true
	}
	valorados := map[string]bool{}
	for _, v := range s.Valoraciones {
		puesto, ok := puestos[v.PuestoRef]
		if !ok || !preferencias[v.PuestoRef] || valorados[v.PuestoRef] {
			return fallo("valoracion_puesto_invalida", "valoraciones")
		}
		valorados[v.PuestoRef] = true
		if v.Entrada.PuestoRef != puesto.Referencia || v.Entrada.NivelPuesto != puesto.Nivel || v.Entrada.InstantaneaRef != i.Referencia {
			return fallo("entrada_no_coincide", "valoraciones.entrada")
		}
		if err := ValidarEntrada(v.Entrada); err != nil {
			return err
		}
		if err := validarComprobaciones(puesto, v.Requisitos); err != nil {
			return err
		}
	}
	return nil
}

func validarComprobaciones(p PuestoOfertado, cs []ComprobacionRequisito) error {
	if cs == nil || len(cs) != len(p.Requisitos) {
		return fallo("comprobaciones_incompletas", "requisitos")
	}
	requisitos := map[string]string{}
	for _, r := range p.Requisitos {
		requisitos[r.Referencia] = r.Version
	}
	vistos := map[string]bool{}
	for _, c := range cs {
		version, ok := requisitos[c.RequisitoRef]
		if !ok || version != c.RequisitoVersion || vistos[c.RequisitoRef] || !referenciaProceso(c.MotivoCodigo) || !referenciaProceso(c.FuenteRef) {
			return fallo("comprobacion_invalida", "requisitos")
		}
		if c.Estado != Cumple && c.Estado != NoCumple && c.Estado != Pendiente {
			return fallo("estado_requisito_invalido", "requisitos.estado")
		}
		vistos[c.RequisitoRef] = true
	}
	return nil
}

func EstadoRequisitos(cs []ComprobacionRequisito) EstadoRequisito {
	estado := Cumple
	for _, c := range cs {
		if c.Estado == NoCumple {
			return NoCumple
		}
		if c.Estado != Cumple {
			estado = Pendiente
		}
	}
	return estado
}
