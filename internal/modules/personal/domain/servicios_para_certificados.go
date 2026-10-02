package domain

import "errors"

var ErrPreparacionServiciosInvalida = errors.New("personal.servicios_certificados.preparacion_invalida")

// PreparacionServiciosParaCertificados revisa los hechos de ejercicio de B2.
// No acredita cobertura de oficio, antigüedad, firma ni emisión de certificados.
type PreparacionServiciosParaCertificados struct {
	Estado                 string                              `json:"estado"`
	EmpleadoRef            string                              `json:"empleado_ref"`
	Corte                  CorteEmpleadoB2                     `json:"corte"`
	Version                int64                               `json:"version"`
	Cobertura              string                              `json:"cobertura"`
	EficaciaAdministrativa bool                                `json:"eficacia_administrativa"`
	FirmaOficial           bool                                `json:"firma_oficial"`
	Servicios              []ServicioPreparadoParaCertificados `json:"servicios"`
}

type ServicioPreparadoParaCertificados struct {
	ServicioRef       string          `json:"servicio_ref"`
	RelacionRef       string          `json:"relacion_ref"`
	PeriodoDesde      FechaCivil      `json:"periodo_desde"`
	PeriodoHasta      FechaCivil      `json:"periodo_hasta"`
	Estado            string          `json:"estado"`
	Clase             string          `json:"clase"`
	Traza             TrazaEmpleadoB2 `json:"traza"`
	SeleccionTemporal string          `json:"seleccion_temporal"`
	Solapado          bool            `json:"solapado"`
	Faltantes         []string        `json:"faltantes"`
}

// PrepararServiciosParaCertificados es una proyección pura. Su consumidor debe
// validar antes el resultado B2 y la evidencia de su lectura autorizada; llamar
// a esta función no concede acceso a ningún hecho. Los periodos se conservan,
// sin recortarlos ni convertir estados declarados en reconocimientos.
func PrepararServiciosParaCertificados(f FichaEmpleadoB2) (PreparacionServiciosParaCertificados, error) {
	var vacio PreparacionServiciosParaCertificados
	if f.EficaciaAdministrativa || f.FirmaOficial || f.Corte.Validar() != nil || f.Version < 1 ||
		!ReferenciaEmpleadoValida(f.EmpleadoRef) || !patronReferenciaB2.MatchString(f.OrganismoRef) || len(f.Servicios) > 200 {
		return vacio, ErrPreparacionServiciosInvalida
	}
	p := PreparacionServiciosParaCertificados{Estado: "preparacion_sintetica", EmpleadoRef: f.EmpleadoRef, Corte: f.Corte, Version: f.Version,
		Cobertura: "no_acreditada", Servicios: make([]ServicioPreparadoParaCertificados, 0, len(f.Servicios))}
	// B2 conserva revisiones, también las sustituidas. Elegir primero la última
	// conocida impide recuperar una revisión antigua porque la nueva ya cesó.
	ultimas := make(map[string]int)
	for i, s := range f.Servicios {
		if !instanteRegistroB2Valido(s.Traza.RegistradaEn) || s.Traza.RegistradaEn.After(f.Corte.ConocidoEn) {
			continue
		}
		anterior, existe := ultimas[s.ServicioRef]
		if !existe || s.Traza.RegistradaEn.After(f.Servicios[anterior].Traza.RegistradaEn) ||
			(s.Traza.RegistradaEn.Equal(f.Servicios[anterior].Traza.RegistradaEn) && s.Traza.Version > f.Servicios[anterior].Traza.Version) {
			ultimas[s.ServicioRef] = i
		}
	}
	for i, s := range f.Servicios {
		if !patronReferenciaB2.MatchString(s.ServicioRef) || !ReferenciaRelacionValida(s.RelacionRef) ||
			(s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") {
			return vacio, ErrPreparacionServiciosInvalida
		}
		fila := ServicioPreparadoParaCertificados{ServicioRef: s.ServicioRef, RelacionRef: s.RelacionRef,
			PeriodoDesde: s.PeriodoDesde, PeriodoHasta: s.PeriodoHasta, Estado: s.Estado, Traza: s.Traza,
			SeleccionTemporal: "pendiente", Faltantes: make([]string, 0)}
		if !patronReferenciaB2.MatchString(s.Traza.ActoRef) {
			fila.Faltantes = append(fila.Faltantes, "acto")
		}
		if !patronReferenciaB2.MatchString(s.Traza.FuenteRef) {
			fila.Faltantes = append(fila.Faltantes, "fuente")
		}
		if s.Traza.FuenteVersion < 1 {
			fila.Faltantes = append(fila.Faltantes, "version_fuente")
		}
		if s.Traza.Version < 1 {
			fila.Faltantes = append(fila.Faltantes, "version_hecho")
		}
		clase := s.CatalogoSnapshot.ClaseServicio
		if clase == nil || !clase.validar("clase_servicio", f.OrganismoRef, s.ClaseRef, s.Traza.Desde) {
			fila.Faltantes = append(fila.Faltantes, "clase")
		} else {
			fila.Clase = clase.Denominacion
		}
		periodoValido := s.PeriodoDesde.Validar() == nil && s.PeriodoHasta.Validar() == nil && !s.PeriodoHasta.AntesDe(s.PeriodoDesde)
		if !periodoValido {
			fila.Faltantes = append(fila.Faltantes, "periodo")
		}
		trazaValida := s.Traza.Desde.Validar() == nil && (s.Traza.Hasta == "" || (s.Traza.Hasta.Validar() == nil && s.Traza.Desde.AntesDe(s.Traza.Hasta))) && instanteRegistroB2Valido(s.Traza.RegistradaEn)
		if !trazaValida {
			fila.Faltantes = append(fila.Faltantes, "registro")
		}
		if len(fila.Faltantes) == 0 {
			fila.SeleccionTemporal = "incluido"
			if f.Corte.VigenteEn.AntesDe(s.Traza.Desde) || (s.Traza.Hasta != "" && !f.Corte.VigenteEn.AntesDe(s.Traza.Hasta)) ||
				s.Traza.RegistradaEn.After(f.Corte.ConocidoEn) || f.Corte.VigenteEn.AntesDe(s.PeriodoHasta) {
				fila.SeleccionTemporal = "fuera_corte"
			}
		}
		if ultima, existe := ultimas[s.ServicioRef]; existe && ultima != i && instanteRegistroB2Valido(s.Traza.RegistradaEn) && !s.Traza.RegistradaEn.After(f.Corte.ConocidoEn) {
			fila.SeleccionTemporal = "sustituido"
		}
		p.Servicios = append(p.Servicios, fila)
	}
	// Un extremo común se señala de forma conservadora para revisión. No se
	// decide aquí si ese día es computable ni la regla de cómputo de antigüedad.
	for i := range p.Servicios {
		a := &p.Servicios[i]
		if a.Estado != "reconocido" || a.SeleccionTemporal != "incluido" {
			continue
		}
		for j := i + 1; j < len(p.Servicios); j++ {
			b := &p.Servicios[j]
			if b.Estado == "reconocido" && b.SeleccionTemporal == "incluido" &&
				!a.PeriodoHasta.AntesDe(b.PeriodoDesde) && !b.PeriodoHasta.AntesDe(a.PeriodoDesde) {
				a.Solapado, b.Solapado = true, true
			}
		}
	}
	return p, nil
}
