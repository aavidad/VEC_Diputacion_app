package application

import (
	"context"
	"regexp"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const EsquemaHechosIncorporacionCT = "vec.personal.hechos-incorporacion-ct.v1"

var referenciaHechosIncorporacionCT = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$|^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type ServicioConsultaIncorporacionCT struct {
	fuente ports.FuenteFichaIncorporacionCT
}

func NuevoServicioConsultaIncorporacionCT(f ports.FuenteFichaIncorporacionCT) (*ServicioConsultaIncorporacionCT, error) {
	if nulo(f) {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &ServicioConsultaIncorporacionCT{fuente: f}, nil
}

func (s *ServicioConsultaIncorporacionCT) ConsultarHechosIncorporacionCT(ctx context.Context, solicitud ports.SolicitudHechosIncorporacionCT) (ports.HechosIncorporacionCT, error) {
	var cero ports.HechosIncorporacionCT
	if s == nil || ctx == nil || nulo(s.fuente) {
		return cero, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	x := solicitud.Seleccion
	if !domain.ReferenciaPersonaValida(x.PersonaRef) || !domain.ReferenciaRelacionValida(x.RelacionRef) ||
		x.VersionEmpleado < 1 || x.VersionRelacion < 1 || x.VersionOcupacion < 1 ||
		!referenciaHechosIncorporacionCT.MatchString(x.OcupacionRef) || !referenciaHechosIncorporacionCT.MatchString(x.UnidadRef) ||
		!referenciaHechosIncorporacionCT.MatchString(x.PuestoRef) || !referenciaHechosIncorporacionCT.MatchString(x.PlazaRef) {
		return cero, domain.ErrRegistroEmpleadoB2Invalido
	}
	c := domain.SolicitudFichaEmpleadoB2{EmpleadoRef: x.EmpleadoRef, OrganismoRef: x.OrganismoRef, Corte: x.Corte, Actor: solicitud.Actor}
	m, err := domain.NuevoMaterialFichaEmpleadoB2(c)
	if err != nil {
		return cero, domain.ErrRegistroEmpleadoB2Invalido
	}
	// Pasar una copia del actor impide que la dependencia altere el contexto del llamante.
	c.Actor = m.Actor()
	r, err := s.fuente.ConsultarFicha(ctx, c)
	if err != nil {
		return cero, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	f := r.Ficha
	if f.ValidarPara(m) != nil || f.PersonaRef != x.PersonaRef || !evidenciaHechosIncorporacionCT(r.Evidencia, x.EmpleadoRef) {
		return cero, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if f.Version != x.VersionEmpleado {
		return cero, domain.ErrRegistroEmpleadoB2Conflicto
	}
	var relacion domain.RelacionRegistroEmpleadoB2
	var ocupacion domain.OcupacionEmpleadoB2
	nr, no := 0, 0
	for _, v := range f.Relaciones {
		if v.RelacionRef == x.RelacionRef && v.Traza.Version == x.VersionRelacion {
			relacion = v
			nr++
		}
	}
	for _, v := range f.Ocupaciones {
		if v.OcupacionRef == x.OcupacionRef && v.Traza.Version == x.VersionOcupacion {
			ocupacion = v
			no++
		}
	}
	// B2 devuelve historia completa conocida. Una revisión posterior desplaza
	// la anterior aunque aquella conserve un intervalo aparentemente vigente.
	for _, v := range f.Relaciones {
		if v.RelacionRef == x.RelacionRef && trazaPosteriorIncorporacionCT(v.Traza, relacion.Traza) {
			return cero, domain.ErrRegistroEmpleadoB2Conflicto
		}
	}
	for _, v := range f.Ocupaciones {
		if v.OcupacionRef == x.OcupacionRef && trazaPosteriorIncorporacionCT(v.Traza, ocupacion.Traza) {
			return cero, domain.ErrRegistroEmpleadoB2Conflicto
		}
	}
	if nr != 1 || no != 1 || relacion.UnidadRef != x.UnidadRef || ocupacion.UnidadRef != x.UnidadRef ||
		ocupacion.RelacionRef != x.RelacionRef || ocupacion.PuestoRef != x.PuestoRef || ocupacion.PlazaRef != x.PlazaRef ||
		relacion.Estado != "vigente" || ocupacion.Estado != "vigente" || ocupacion.Clase == "reserva" ||
		!trazaVigenteIncorporacionCT(relacion.Traza, x.Corte.VigenteEn) || !trazaVigenteIncorporacionCT(ocupacion.Traza, x.Corte.VigenteEn) {
		return cero, domain.ErrRegistroEmpleadoB2Conflicto
	}
	return ports.HechosIncorporacionCT{Esquema: EsquemaHechosIncorporacionCT, Seleccion: x,
		Relacion: relacion.Traza, Ocupacion: ocupacion.Traza, RegimenRef: relacion.RegimenRef,
		ModalidadRef: ocupacion.ModalidadRef, ClaseOcupacion: ocupacion.Clase, Evidencia: r.Evidencia}, nil
}

func trazaVigenteIncorporacionCT(t domain.TrazaEmpleadoB2, en domain.FechaCivil) bool {
	return !en.AntesDe(t.Desde) && (t.Hasta == "" || en.AntesDe(t.Hasta))
}

func trazaPosteriorIncorporacionCT(a, b domain.TrazaEmpleadoB2) bool {
	return a.RegistradaEn.After(b.RegistradaEn) || (a.RegistradaEn.Equal(b.RegistradaEn) && a.Version > b.Version)
}

func evidenciaHechosIncorporacionCT(e ports.EvidenciaRegistroEmpleadoB2, empleado string) bool {
	_, offset := e.ConsultadaEn.Zone()
	return e.ReciboRef != "" && len(e.ReciboRef) <= 160 && e.DecisionRef != "" && len(e.DecisionRef) <= 160 &&
		e.AuditoriaRef != "" && len(e.AuditoriaRef) <= 160 && e.EfectoRef == empleado && huellaRegistroB2.MatchString(e.ConsumoHuellaSHA256) &&
		!e.ConsultadaEn.IsZero() && offset == 0 && e.ConsultadaEn.Nanosecond()%1000 == 0
}

var _ ports.ConsultaHechosIncorporacionCT = (*ServicioConsultaIncorporacionCT)(nil)
