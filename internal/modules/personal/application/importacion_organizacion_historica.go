package application

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioImportacionOrganizacion struct {
	autorizador ports.AutorizadorImportacionOrganizacion
	catalogos   ports.VerificadorCatalogosImportacionOrganizacion
	politica    ports.PoliticaFuentesOrganizacionHistorica
	repositorio ports.RepositorioImportacionOrganizacion
}

func NuevoServicioImportacionOrganizacion(a ports.AutorizadorImportacionOrganizacion, c ports.VerificadorCatalogosImportacionOrganizacion,
	p ports.PoliticaFuentesOrganizacionHistorica, r ports.RepositorioImportacionOrganizacion) (*ServicioImportacionOrganizacion, error) {
	if nulaImportacion(a) || nulaImportacion(c) || nulaImportacion(p) || nulaImportacion(r) {
		return nil, domain.ErrImportacionOrganizacionNoDisponible
	}
	return &ServicioImportacionOrganizacion{a, c, p, r}, nil
}

func (s *ServicioImportacionOrganizacion) Preparar(ctx context.Context, solicitud domain.SolicitudImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	if solicitud.Fase != domain.FasePrepararOrganizacion {
		return ports.ReciboImportacionOrganizacion{}, domain.ErrImportacionOrganizacionInvalida
	}
	return s.ejecutar(ctx, solicitud)
}
func (s *ServicioImportacionOrganizacion) Conciliar(ctx context.Context, solicitud domain.SolicitudImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	if solicitud.Fase != domain.FaseConciliarOrganizacion {
		return ports.ReciboImportacionOrganizacion{}, domain.ErrImportacionOrganizacionInvalida
	}
	return s.ejecutar(ctx, solicitud)
}
func (s *ServicioImportacionOrganizacion) Publicar(ctx context.Context, solicitud domain.SolicitudImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	if solicitud.Fase != domain.FasePublicarOrganizacion {
		return ports.ReciboImportacionOrganizacion{}, domain.ErrImportacionOrganizacionInvalida
	}
	return s.ejecutar(ctx, solicitud)
}

func (s *ServicioImportacionOrganizacion) ejecutar(ctx context.Context, solicitud domain.SolicitudImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	var vacio ports.ReciboImportacionOrganizacion
	if ctx == nil || s == nil || nulaImportacion(s.autorizador) || nulaImportacion(s.catalogos) || nulaImportacion(s.politica) || nulaImportacion(s.repositorio) {
		return vacio, domain.ErrImportacionOrganizacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialImportacionOrganizacion(solicitud)
	if err != nil {
		return vacio, domain.ErrImportacionOrganizacionInvalida
	}
	autorizacion, err := s.autorizador.AutorizarImportacionOrganizacion(ctx, material)
	if err != nil {
		return vacio, errorImportacionOpaco(ctx, err)
	}
	if !autorizacionImportacionValida(material, autorizacion) {
		return vacio, domain.ErrImportacionOrganizacionDenegada
	}
	var acreditacion ports.AcreditacionFuenteOrganizacionHistorica
	if solicitud.Fase != domain.FasePrepararOrganizacion {
		if err := s.catalogos.ValidarReferencias(ctx, solicitud.Manifiesto.CatalogoUnidades, solicitud.Manifiesto.CatalogoClasificaciones); err != nil {
			return vacio, errorImportacionOpaco(ctx, err)
		}
	} else if err := s.catalogos.ValidarHechos(ctx, solicitud.Manifiesto.CatalogoUnidades, solicitud.Manifiesto.CatalogoClasificaciones, solicitud.Hechos); err != nil {
		return vacio, errorImportacionOpaco(ctx, err)
	}
	if solicitud.Fase == domain.FasePublicarOrganizacion {
		acreditacion, err = s.politica.AcreditarPublicacion(ctx, solicitud.Manifiesto)
		if err != nil {
			return vacio, errorImportacionOpaco(ctx, err)
		}
		if !acreditacionImportacionValida(solicitud.Manifiesto, acreditacion) {
			return vacio, domain.ErrImportacionOrganizacionDenegada
		}
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recibo, err := s.repositorio.Ejecutar(ctx, ports.OrdenImportacionOrganizacion{Material: material, Autorizacion: autorizacion, Acreditacion: acreditacion})
	if err != nil {
		return vacio, errorImportacionOpaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !reciboImportacionValido(material, autorizacion, recibo) {
		return vacio, domain.ErrImportacionOrganizacionNoDisponible
	}
	return recibo, nil
}

func autorizacionImportacionValida(m domain.MaterialImportacionOrganizacion, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	s := m.Solicitud()
	r := m.Recurso()
	h, err := r.HuellaContextoAutorizacionSHA256()
	x := a.ResumenCapacidad()
	return err == nil && a.ValidarEstructura() == nil && a.PersonaVersion() == s.Actor.Instantanea.PersonaVersion &&
		a.PerfilVersion() == s.Actor.Instantanea.PerfilVersion && x.Operacion() == s.Fase.Accion() &&
		x.AudienciaConsumo() == domain.AudienciaImportacionOrganizacion && x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}

func acreditacionImportacionValida(m domain.ManifiestoImportacionOrganizacion, a ports.AcreditacionFuenteOrganizacionHistorica) bool {
	h, err := m.HuellaSHA256()
	return err == nil && a.ManifiestoHuellaSHA256 == h && a.OrganismoRef == m.OrganismoRef && a.Tipo == m.Tipo && a.FuenteRef == m.FuenteRef &&
		a.FuenteVersion == m.FuenteVersion && a.FuenteHuellaSHA256 == m.FuenteHuellaSHA256 &&
		a.DiccionarioRef == m.DiccionarioRef && a.ActoRef == m.ActoRef && a.CustodiaRef == m.CustodiaRef &&
		referenciaAcreditacionValida(a.AcreditacionRef) && huellaAcreditacionValida(a.AcreditacionHuellaSHA256) &&
		domain.InstanteImportacionValido(a.AcreditadaEn)
}

func reciboImportacionValido(m domain.MaterialImportacionOrganizacion, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ReciboImportacionOrganizacion) bool {
	s := m.Solicitud()
	x := a.ResumenCapacidad()
	if !referenciaAcreditacionValida(r.ReciboRef) || !referenciaAcreditacionValida(r.LoteRef) ||
		r.Fase != s.Fase || r.RevisionAnterior != s.RevisionEsperada || r.RevisionNueva != s.RevisionEsperada+1 ||
		r.ClaveIdempotencia != s.ClaveIdempotencia || r.MaterialHuellaSHA256 != m.HuellaSHA256() ||
		r.FuenteHuellaSHA256 != s.Manifiesto.FuenteHuellaSHA256 || r.ActorRef != s.Actor.Principal.ID ||
		!referenciaAcreditacionValida(r.DecisionRef) || !referenciaAcreditacionValida(r.AuditoriaRef) || !domain.InstanteImportacionValido(r.RegistradoEn) {
		return false
	}
	// Un replay devuelve el recibo y la decisión originales, que pueden ser
	// anteriores a la autorización emitida para este intento de recuperación.
	if !r.Replay && (r.DecisionRef != x.DecisionRef() || r.RegistradoEn.Before(x.EmitidaEn()) || !r.RegistradoEn.Before(x.ExpiraEn())) {
		return false
	}
	if s.Fase == domain.FasePrepararOrganizacion {
		return r.Estado == "preparacion_no_autoritativa"
	}
	if r.LoteRef != s.LoteRef {
		return false
	}
	if s.Fase == domain.FaseConciliarOrganizacion {
		return r.Estado == "conciliacion_pendiente" || r.Estado == "conciliada"
	}
	return r.Estado == "publicada"
}

func referenciaAcreditacionValida(v string) bool {
	if len(v) < 3 || len(v) > 256 {
		return false
	}
	for _, c := range v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func huellaAcreditacionValida(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}
func nulaImportacion(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Pointer || r.Kind() == reflect.Interface || r.Kind() == reflect.Func || r.Kind() == reflect.Map || r.Kind() == reflect.Slice) && r.IsNil()
}
func errorImportacionOpaco(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, domain.ErrImportacionOrganizacionConflicto) {
		return domain.ErrImportacionOrganizacionConflicto
	}
	if errors.Is(err, domain.ErrImportacionOrganizacionDenegada) {
		return domain.ErrImportacionOrganizacionDenegada
	}
	return domain.ErrImportacionOrganizacionNoDisponible
}
