package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ServicioFichaPropia entrega a la persona empleada su propia ficha: sus
// relaciones de servicio y servicios reconocidos. El empleado procede del
// contexto de actor; la autorización es una concesión V3 nueva por consulta.
type ServicioFichaPropia struct {
	autorizador ports.ProveedorAutorizacionFichaPropia
	repositorio ports.RepositorioFichaPropia
}

func NuevoServicioFichaPropia(a ports.ProveedorAutorizacionFichaPropia, r ports.RepositorioFichaPropia) (*ServicioFichaPropia, error) {
	if nulo(a) || nulo(r) {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	return &ServicioFichaPropia{a, r}, nil
}

func (s *ServicioFichaPropia) Consultar(ctx context.Context, solicitud domain.SolicitudFichaPropia) (ports.ResultadoFichaPropia, error) {
	var vacio ports.ResultadoFichaPropia
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrFichaPropiaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialFichaPropia(solicitud)
	if err != nil {
		if errors.Is(err, domain.ErrFichaPropiaSinEmpleado) || errors.Is(err, domain.ErrFichaPropiaAmbigua) {
			return vacio, err
		}
		return vacio, domain.ErrFichaPropiaInvalida
	}
	autorizacion, err := s.autorizador.AutorizarFichaPropia(ctx, material)
	if err != nil {
		return vacio, errorFichaPropiaOpaco(ctx, err)
	}
	if !autorizacionFichaPropiaValida(material, autorizacion) {
		return vacio, domain.ErrFichaPropiaNoDisponible
	}
	resultado, err := s.repositorio.ConsultarFichaPropia(ctx, ports.OrdenFichaPropia{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorFichaPropiaOpaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if resultado.Ficha.ValidarPara(material) != nil || !evidenciaFichaPropiaValida(autorizacion, resultado.Evidencia) {
		return vacio, domain.ErrFichaPropiaNoDisponible
	}
	return resultado, nil
}

func autorizacionFichaPropiaValida(m domain.MaterialFichaPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && a.ValidarEstructura() == nil &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionFichaPropia && x.AudienciaConsumo() == domain.AudienciaFichaPropia &&
		x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}

func evidenciaFichaPropiaValida(a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e ports.EvidenciaRegistroEmpleadoB2) bool {
	x := a.ResumenCapacidad()
	_, offset := e.ConsultadaEn.Zone()
	return e.ReciboRef != "" && len(e.ReciboRef) <= 160 && e.AuditoriaRef != "" && len(e.AuditoriaRef) <= 160 &&
		huellaRegistroB2.MatchString(e.ConsumoHuellaSHA256) && e.DecisionRef == x.DecisionRef() && e.EfectoRef == x.EfectoRef() &&
		!e.ConsultadaEn.IsZero() && offset == 0 && e.ConsultadaEn.Nanosecond()%1000 == 0 &&
		!e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func errorFichaPropiaOpaco(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, domain.ErrFichaPropiaDenegada) {
		return domain.ErrFichaPropiaDenegada
	}
	if errors.Is(err, domain.ErrFichaPropiaExcedeLimite) {
		return domain.ErrFichaPropiaExcedeLimite
	}
	return domain.ErrFichaPropiaNoDisponible
}
