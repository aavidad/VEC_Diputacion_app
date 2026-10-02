package domain

// PreparacionRelacionParaRPT minimiza la ficha B2 que el caso de uso ya ha
// validado y consultado con su autorización. Esta proyección no concede acceso
// ni acredita empleo oficial, ocupación, reserva o disponibilidad de una plaza.
type PreparacionRelacionParaRPT struct {
	Esquema      string                       `json:"esquema"`
	Uso          string                       `json:"uso"`
	EmpleadoRef  string                       `json:"empleado_ref"`
	VersionFicha int64                        `json:"version_ficha"`
	Corte        CorteEmpleadoB2              `json:"corte"`
	Cobertura    string                       `json:"cobertura"`
	EstadoRPT    string                       `json:"estado_rpt"`
	Relaciones   []RelacionPreparacionParaRPT `json:"relaciones"`
}

type RelacionPreparacionParaRPT struct {
	RelacionRef string          `json:"relacion_ref"`
	Estado      string          `json:"estado"`
	EnIntervalo bool            `json:"en_intervalo"`
	Traza       TrazaEmpleadoB2 `json:"traza"`
}

// PrepararRelacionParaRPT selecciona la última revisión conocida por relación,
// incluso cuando esa revisión haya finalizado o suspendido la relación. Nunca
// recupera una revisión antigua porque su intervalo parezca vigente. Conserva
// el orden de primera aparición de las relaciones en la ficha consultada.
// El llamante debe mantener esta preparación dentro de la respuesta autorizada
// B2; no se puede usar como fuente de oficio o permiso para otro consumidor.
func PrepararRelacionParaRPT(f FichaEmpleadoB2) (PreparacionRelacionParaRPT, error) {
	var cero PreparacionRelacionParaRPT
	if !ReferenciaPersonaValida(f.PersonaRef) || !ReferenciaEmpleadoValida(f.EmpleadoRef) ||
		!patronReferenciaB2.MatchString(f.OrganismoRef) || f.Version < 1 || f.Corte.Validar() != nil ||
		f.EficaciaAdministrativa || f.FirmaOficial || len(f.Relaciones) > 200 {
		return cero, ErrRegistroEmpleadoB2Invalido
	}
	r := PreparacionRelacionParaRPT{
		Esquema: "vec.personal.preparacion-relacion-rpt.v1", Uso: "preparacion",
		EmpleadoRef:  f.EmpleadoRef,
		VersionFicha: f.Version, Corte: f.Corte, Cobertura: "no_acreditada",
		EstadoRPT: "pendiente_fuente_rpt", Relaciones: make([]RelacionPreparacionParaRPT, 0),
	}
	indices := make(map[string]int)
	revisiones := make(map[string]map[int64]struct{})
	for _, relacion := range f.Relaciones {
		if !ReferenciaRelacionValida(relacion.RelacionRef) || relacion.OrganismoRef != f.OrganismoRef ||
			!estadoRelacionB2Valido(relacion.Estado) || relacion.Traza.ValidarEn(f.Corte) != nil {
			return cero, ErrRegistroEmpleadoB2Invalido
		}
		if revisiones[relacion.RelacionRef] == nil {
			revisiones[relacion.RelacionRef] = make(map[int64]struct{})
		}
		if _, repetida := revisiones[relacion.RelacionRef][relacion.Traza.Version]; repetida {
			return cero, ErrRegistroEmpleadoB2Invalido
		}
		revisiones[relacion.RelacionRef][relacion.Traza.Version] = struct{}{}
		nueva := RelacionPreparacionParaRPT{
			RelacionRef: relacion.RelacionRef, Estado: relacion.Estado, Traza: relacion.Traza,
			EnIntervalo: !f.Corte.VigenteEn.AntesDe(relacion.Traza.Desde) &&
				(relacion.Traza.Hasta == "" || f.Corte.VigenteEn.AntesDe(relacion.Traza.Hasta)),
		}
		if indice, existe := indices[relacion.RelacionRef]; existe {
			anterior := r.Relaciones[indice].Traza
			if relacion.Traza.RegistradaEn.After(anterior.RegistradaEn) ||
				(relacion.Traza.RegistradaEn.Equal(anterior.RegistradaEn) && relacion.Traza.Version > anterior.Version) {
				r.Relaciones[indice] = nueva
			}
		} else {
			indices[relacion.RelacionRef] = len(r.Relaciones)
			r.Relaciones = append(r.Relaciones, nueva)
		}
	}
	return r, nil
}
