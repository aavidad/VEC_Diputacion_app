package incorporacionejercicio

import (
	"time"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

func (p PlanPreparacionDurableV2) Validar() error {
	for _, ref := range []string{p.OrganizacionRef, p.UnidadRef, p.SeguimientoRef, p.RelacionRef} {
		if !dom.ReferenciaOpacaValida(ref) {
			return ct.ErrComposicionIncorporacionAplicacion
		}
	}
	f := ct.ReferenciaVersionadaPersonalRPT{Referencia: p.FuentePersonal.Referencia, Version: p.FuentePersonal.Version, HuellaSHA256: p.FuentePersonal.HuellaSHA256}
	if p.SolicitudPersonal.Validar() != nil || f.Validar() != nil || p.Definicion.Validar() != nil || p.VersionExpedienteRaiz == 0 || p.VersionExpedienteRaiz > ct.MaximoEnteroSeguroOperacionAnalisis ||
		p.VersionSeguimientoEsperada != 0 || p.Periodo.Validar() != nil || !p.MotivoClave.Valida() || len(p.Documentos) > 32 {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	if _, err := core.HuellaSHA256MotivoAutorizacionV2(p.MotivoV3); err != nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	refs := map[string]bool{}
	for _, d := range p.Documentos {
		if !d.TipoClave.Valida() || !dom.ReferenciaOpacaValida(d.Referencia) || refs[d.Referencia] {
			return ct.ErrComposicionIncorporacionAplicacion
		}
		refs[d.Referencia] = true
	}
	return nil
}

func validarInicialPreparacion(p PlanPreparacionDurableV2, pub dom.PublicacionDefinicionSeguimiento, estado dom.EstadoPersistidoSeguimiento, versionRaiz uint64, ahora time.Time) error {
	def, err := dom.RestaurarDefinicionSeguimiento(pub)
	if err != nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	s, err := dom.RehidratarSeguimiento(def, estado)
	if err != nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	if !def.VigenteEn(ahora) || estado.ActualizadoEn.After(ahora) || !def.Referencia().Coincide(p.Definicion) || versionRaiz != p.VersionExpedienteRaiz || estado.Referencia != p.SeguimientoRef || estado.OrganizacionRef != p.OrganizacionRef ||
		estado.ExpedienteRef != p.SolicitudPersonal.ExpedienteRef || estado.RelacionRef != p.RelacionRef || s.Version() != p.VersionSeguimientoEsperada || s.Version() != 0 ||
		p.Periodo.Desde.Before(estado.PeriodoPrevisto.Desde) || p.Periodo.Hasta.After(estado.PeriodoPrevisto.Hasta) {
		return ct.ErrConflictoIncorporacionAplicacion
	}
	// Leer el catálogo existente, sin aplicar una transición con recibos o
	// actuaciones inventadas para "ensayar" dentro de una consulta.
	for _, tr := range def.Publicacion().Transiciones {
		if tr.Clave != ct.TransicionConfirmarIncorporacion {
			continue
		}
		if tr.Origen != s.EstadoActual() || tr.EfectoPeriodo != dom.EfectoPeriodoAbrir || !tr.RequierePeriodo || tr.Calendario != nil {
			return ct.ErrConflictoIncorporacionAplicacion
		}
		permitido := false
		for _, m := range tr.MotivosPermitidos {
			if m == p.MotivoClave {
				permitido = true
			}
		}
		if !permitido {
			return ct.ErrConflictoIncorporacionAplicacion
		}
		tipos := map[dom.ClaveCatalogo]bool{}
		for _, d := range p.Documentos {
			conocido := false
			for _, r := range tr.Documentos {
				if r.TipoClave == d.TipoClave {
					conocido = true
				}
			}
			if !conocido {
				return ct.ErrConflictoIncorporacionAplicacion
			}
			tipos[d.TipoClave] = true
		}
		for _, r := range tr.Documentos {
			if r.Obligatorio && !tipos[r.TipoClave] {
				return ct.ErrConflictoIncorporacionAplicacion
			}
		}
		return nil
	}
	return ct.ErrConflictoIncorporacionAplicacion
}

func documentosIntencionExactos(documentos []dom.DocumentoSeguimiento, refs []string) bool {
	if len(documentos) != len(refs) {
		return false
	}
	vistos := map[string]bool{}
	for _, r := range refs {
		if vistos[r] {
			return false
		}
		vistos[r] = true
	}
	for _, d := range documentos {
		if !vistos[d.Referencia] {
			return false
		}
		delete(vistos, d.Referencia)
	}
	return len(vistos) == 0
}

func selectorPersonalDelPlan(s lector.Selector, p PlanPreparacionDurableV2) bool {
	return s.OrganizacionRef == p.OrganizacionRef && s.ExpedienteRef == p.SolicitudPersonal.ExpedienteRef && s.SolicitudRef == p.SolicitudPersonal.SolicitudRef &&
		s.VersionExpediente == p.SolicitudPersonal.VersionExpediente && s.RelacionRef == p.RelacionRef
}

func originalPersonalDelPlan(r lector.Resultado, s lector.Selector, p PlanPreparacionDurableV2, t time.Time) bool {
	o := r.Registro
	// El consumidor propietario coteja organización y canon del material;
	// ese campo no se duplica en el DTO reducido RegistroPersonalEjercicio.
	if o.ValidarEstructuraPara(p.SolicitudPersonal, t) != nil || o.Solicitud != p.SolicitudPersonal ||
		o.MaterialSHA256 != s.MaterialSHA256 || o.Resultado.ResultadoRef != s.ResultadoRef || o.Resultado.ReciboRef != s.ReciboRef || o.Resultado.RelacionRef != s.RelacionRef || o.Resultado.OcupacionRef != s.OcupacionRef ||
		!dom.InstanteUTCCanonico(r.LeidaEn) || r.LeidaEn.After(t) || !dom.ReferenciaOpacaValida(r.DecisionLecturaRef) || !dom.ReferenciaOpacaValida(r.AuditoriaLecturaRef) {
		return false
	}
	return true
}

func preparacionOriginalRestaurada(h hist.Restauracion, sel hist.Selector, org, exp, solicitud string) (ct.PreparacionIncorporacionAplicacionV2, *ct.ReciboIncorporacionAplicacionV2, error) {
	var z ct.PreparacionIncorporacionAplicacionV2
	r := h.Historia.ReciboOriginal()
	d, err := h.OrdenOriginal.Material().Datos()
	if err != nil || r.ValidarPara(h.OrdenOriginal) != nil || r.Transicion.Periodo == nil || r.Transicion.ReciboRef != sel.ReciboRef || r.MaterialOriginalSHA256 != sel.MaterialSHA256 || r.IntencionSHA256 != sel.IntencionSHA256 ||
		d.Preparacion.OrganizacionRef != org || d.Confirmacion.SolicitudPersonal.ExpedienteRef != exp || d.Confirmacion.SolicitudPersonal.SolicitudRef != solicitud {
		return z, nil, ct.ErrComposicionIncorporacionAplicacion
	}
	pub, anterior, posterior := h.Historia.EvidenciaSeguimiento()
	if _, err := ct.NuevaHistoriaRegistroIncorporacionV2(h.OrdenOriginal, r, pub, anterior, posterior); err != nil {
		return z, nil, ct.ErrComposicionIncorporacionAplicacion
	}
	out := reciboAplicacionDesdeRegistro(d.Personal, r)
	return ct.PreparacionIncorporacionAplicacionV2{SolicitudPersonal: d.Confirmacion.SolicitudPersonal, VersionActualExpediente: d.VersionActualExpediente,
		VersionSeguimientoEsperada: d.Confirmacion.VersionSeguimientoEsperada, Periodo: d.Confirmacion.PeriodoIncorporacion, MotivoClave: d.Confirmacion.MotivoClave,
		Documentos: append([]dom.DocumentoSeguimiento(nil), d.Confirmacion.Documentos...), MotivoV3: d.MotivoV3}, &out, nil
}

func reciboAplicacionDesdeRegistro(r ct.RegistroPersonalEjercicio, recibo ct.ReciboRegistroIncorporacionV2) ct.ReciboIncorporacionAplicacionV2 {
	return ct.ReciboIncorporacionAplicacionV2{Esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
		ExpedienteRef: r.Solicitud.ExpedienteRef, SolicitudPersonalRef: r.Solicitud.SolicitudRef, RelacionRef: r.Resultado.RelacionRef,
		ReciboRef: recibo.Transicion.ReciboRef, ActuacionRef: recibo.Transicion.ActuacionRef, RegistradaEn: recibo.Transicion.RegistradaEn, Periodo: *recibo.Transicion.Periodo,
		VersionSolicitudPersonal: recibo.VersionSolicitudPersonal, VersionActualExpediente: recibo.VersionActualExpediente, SeguimientoRef: recibo.SeguimientoRef,
		VersionSeguimientoAnterior: recibo.VersionSeguimientoAnterior, VersionSeguimientoResultante: recibo.VersionSeguimientoResultante,
		AuditoriaRef: recibo.AuditoriaCTRef, OutboxRef: recibo.OutboxCTRef, EjercicioSintetico: recibo.EjercicioSintetico, FirmaOficial: recibo.FirmaOficial, EficaciaAdministrativa: recibo.EficaciaAdministrativa}
}
