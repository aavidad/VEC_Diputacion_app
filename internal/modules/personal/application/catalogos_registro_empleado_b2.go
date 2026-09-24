package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioCatalogosRegistroEmpleadoB2 struct {
	autorizador ports.ProveedorAutorizacionCatalogosRegistroEmpleadoB2
	repositorio ports.RepositorioCatalogosRegistroEmpleadoB2
}

func NuevoServicioCatalogosRegistroEmpleadoB2(a ports.ProveedorAutorizacionCatalogosRegistroEmpleadoB2, r ports.RepositorioCatalogosRegistroEmpleadoB2) (*ServicioCatalogosRegistroEmpleadoB2, error) {
	if nulo(a) || nulo(r) {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &ServicioCatalogosRegistroEmpleadoB2{a, r}, nil
}

func (s *ServicioCatalogosRegistroEmpleadoB2) Consultar(ctx context.Context, solicitud domain.SolicitudConsultaCatalogoEmpleadoB2) (ports.ResultadoConsultaCatalogoEmpleadoB2, error) {
	var vacio ports.ResultadoConsultaCatalogoEmpleadoB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	m, err := domain.NuevoMaterialConsultaCatalogoEmpleadoB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	a, err := s.autorizador.AutorizarCatalogoRegistroEmpleadoB2(ctx, m)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionCatalogoEmpleadoB2Valida(m, a) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.ConsultarRRHH(ctx, ports.OrdenCatalogoEmpleadoB2{Material: m, Autorizacion: a})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !consultaCatalogoEmpleadoB2Valida(m, a, resultado) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}

func (s *ServicioCatalogosRegistroEmpleadoB2) Cambiar(ctx context.Context, solicitud domain.SolicitudCambioCatalogoEmpleadoB2) (ports.ResultadoCambioCatalogoEmpleadoB2, error) {
	var vacio ports.ResultadoCambioCatalogoEmpleadoB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	m, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	a, err := s.autorizador.AutorizarCatalogoRegistroEmpleadoB2(ctx, m)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionCatalogoEmpleadoB2Valida(m, a) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.CambiarRRHH(ctx, ports.OrdenCatalogoEmpleadoB2{Material: m, Autorizacion: a})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !cambioCatalogoEmpleadoB2Valido(m, a, solicitud, resultado) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}

func autorizacionCatalogoEmpleadoB2Valida(m domain.MaterialCatalogoEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	recurso := m.Recurso()
	huella, err := m.HuellaSHA256()
	r := a.ResumenCapacidad()
	accion, audiencia := accionAudienciaCatalogoEmpleadoB2(m.Operacion())
	return err == nil && a.ValidarEstructura() == nil &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		r.Operacion() == accion && r.AudienciaConsumo() == audiencia && r.EfectoRef() == recurso.Referencia && r.EfectoHuellaSHA256() == huella
}

func accionAudienciaCatalogoEmpleadoB2(operacion string) (string, string) {
	switch operacion {
	case "consultar":
		return domain.AccionConsultarCatalogoEmpleadoB2, domain.AudienciaConsultarCatalogoEmpleadoB2
	case "publicar":
		return domain.AccionPublicarCatalogoEmpleadoB2, domain.AudienciaPublicarCatalogoEmpleadoB2
	case "retirar":
		return domain.AccionRetirarCatalogoEmpleadoB2, domain.AudienciaRetirarCatalogoEmpleadoB2
	default:
		return "", ""
	}
}

func consultaCatalogoEmpleadoB2Valida(m domain.MaterialCatalogoEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoConsultaCatalogoEmpleadoB2) bool {
	if r.OrganismoRef != m.OrganismoRef() || len(r.Entradas) > m.Limite() || len(r.Entradas) > 100 {
		return false
	}
	capacidad := a.ResumenCapacidad()
	if r.Evidencia.DecisionRef != capacidad.DecisionRef() || r.Evidencia.EfectoRef != m.Recurso().Referencia ||
		!huellaRegistroB2.MatchString(r.Evidencia.ConsumoHuellaSHA256) || r.Evidencia.AuditoriaRef == "" ||
		!instanteCatalogoEmpleadoB2Valido(r.Evidencia.ConsultadaEn) ||
		r.Evidencia.ConsultadaEn.Before(capacidad.EmitidaEn()) || !r.Evidencia.ConsultadaEn.Before(capacidad.ExpiraEn()) {
		return false
	}
	ultimoRef, ultimaVersion := m.CursorRef(), m.CursorVersion()
	for _, entrada := range r.Entradas {
		if entrada.Validar() != nil || entrada.OrganismoRef != m.OrganismoRef() || entrada.Tipo != m.Tipo() ||
			(m.Estado() != "" && entrada.Estado != m.Estado()) ||
			entrada.Ref < ultimoRef || (entrada.Ref == ultimoRef && entrada.Version <= ultimaVersion) {
			return false
		}
		ultimoRef, ultimaVersion = entrada.Ref, entrada.Version
	}
	if r.CursorSiguiente != nil {
		if len(r.Entradas) == 0 || r.CursorSiguiente.Ref != ultimoRef || r.CursorSiguiente.Version != ultimaVersion {
			return false
		}
	}
	return true
}

func cambioCatalogoEmpleadoB2Valido(m domain.MaterialCatalogoEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, solicitud domain.SolicitudCambioCatalogoEmpleadoB2, r ports.ResultadoCambioCatalogoEmpleadoB2) bool {
	e := r.Entrada
	if e.Validar() != nil || e.OrganismoRef != m.OrganismoRef() || e.Tipo != m.Tipo() || e.Ref != m.Ref() ||
		e.Version != m.Version() || e.Revision != m.Revision() || e.Denominacion != solicitud.Denominacion ||
		e.HuellaSHA256 != solicitud.HuellaSHA256 || e.VigenteDesde != solicitud.VigenteDesde || e.VigenteHasta != solicitud.VigenteHasta ||
		(e.Estado != "publicada" && m.Operacion() == "publicar") || (e.Estado != "retirada" && m.Operacion() == "retirar") {
		return false
	}
	capacidad := a.ResumenCapacidad()
	actual := r.AccesoActual
	if (actual.EstadoReplay != "registrado" && actual.EstadoReplay != "replay") ||
		actual.DecisionRef != capacidad.DecisionRef() || actual.AuditoriaRef == "" ||
		!huellaRegistroB2.MatchString(actual.ConsumoHuellaSHA256) || !instanteCatalogoEmpleadoB2Valido(actual.RegistradoEn) ||
		actual.RegistradoEn.Before(capacidad.EmitidaEn()) || !actual.RegistradoEn.Before(capacidad.ExpiraEn()) {
		return false
	}
	return r.Recibo.DecisionRef != "" && r.Recibo.AuditoriaRef != "" &&
		huellaRegistroB2.MatchString(r.Recibo.ConsumoHuellaSHA256) && instanteCatalogoEmpleadoB2Valido(r.Recibo.RegistradoEn)
}

func instanteCatalogoEmpleadoB2Valido(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}
