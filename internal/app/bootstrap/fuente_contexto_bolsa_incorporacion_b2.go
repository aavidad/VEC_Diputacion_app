package bootstrap

import (
	"context"
	"errors"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type ConsultaVinculoIncorporacionB2 interface {
	Consultar(context.Context, ct.ConsultaVinculoCategoriaRPT) (ct.LecturaVinculoCategoriaRPT, error)
}

type AutoridadPublicacionIncorporacionB2 interface {
	LecturaRPT(context.Context, dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// Las dependencias resuelven sesión, perfil y contexto desde la frontera
// confiable de la petición. El contrato CT nunca aporta identidad ni permisos.
type ConfiguracionFuentesContextoIncorporacionB2 struct {
	Bolsa        bp.ConsultaPersonaAceptacionCT
	ActorBolsa   func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error)
	Vinculos     ConsultaVinculoIncorporacionB2
	Publicacion  ct.FuentePublicacionCategoriaRPT
	AutoridadRPT AutoridadPublicacionIncorporacionB2
}

type FuentesContextoIncorporacionB2 struct {
	c ConfiguracionFuentesContextoIncorporacionB2
}

type clavePersonaBolsaB2 struct {
	fuente   *FuentesContextoIncorporacionB2
	selector bp.SelectorPersonaAceptacionCT
}

// Estos DTO de Bolsa sólo contienen valores; la captura no conserva sus
// punteros ni material de autorización. Cada lectura devuelve valores propios.
type lecturaPersonaBolsaB2 struct {
	aceptacion bp.AceptacionPersonaCT
	persona    bp.PersonaAceptadaCT
	vinculo    bp.VinculoPersonaAceptadaCT
	evidencia  bp.EvidenciaConsultaPersonaAceptacionCT
}

func (l lecturaPersonaBolsaB2) resultado() bp.ResultadoConsultaPersonaAceptacionCT {
	return bp.ResultadoConsultaPersonaAceptacionCT{Estado: "acreditado", Aceptacion: &l.aceptacion, Persona: &l.persona, Vinculo: &l.vinculo, Evidencia: l.evidencia}
}

func (f *FuentesContextoIncorporacionB2) consultarPersonaBolsaB2(ctx context.Context, x dom.SelectorBolsaPlanB2) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
	var cero bp.ResultadoConsultaPersonaAceptacionCT
	if f == nil || ctx == nil || dependenciaBootstrapNula(f.c.Bolsa) || f.c.ActorBolsa == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	holder, _ := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if holder != nil {
		// El lector real vuelve al resolutor nominal al autorizar. Su mutex
		// queda libre mientras esta lectura consume y audita su propia V3.
		holder.mu.Lock()
		if holder.muPersonasB2 == nil {
			holder.muPersonasB2 = make(chan struct{}, 1)
		}
		mutexPersona := holder.muPersonasB2
		holder.mu.Unlock()
		select {
		case mutexPersona <- struct{}{}:
			defer func() { <-mutexPersona }()
		case <-ctx.Done():
			return cero, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	actor, err := f.c.ActorBolsa(ctx)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	selector := bp.SelectorPersonaAceptacionCT{UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef,
		AceptacionOperacionRef: x.AceptacionOperacionRef, AceptacionRegistroSHA256: x.AceptacionRegistroSHA256,
		AperturaOperacionRef: x.AperturaOperacionRef, AperturaRegistroSHA256: x.AperturaRegistroSHA256,
		LlamamientoRef: x.LlamamientoRef, PropuestaRef: x.PropuestaRef}
	consultar := func() (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		return f.c.Bolsa.ConsultarPersonaAceptacionCT(ctx, bp.SolicitudConsultaPersonaAceptacionCT{ActorConfiable: actor, Selector: selector})
	}
	if holder == nil {
		return consultar()
	}
	holder.mu.Lock()
	propietario := holder.propietario
	holder.mu.Unlock()
	if propietario == nil || propietario.soporte == nil {
		return consultar()
	}
	clave := clavePersonaBolsaB2{f, selector}
	if lectura, existe := holder.personasB2[clave]; existe {
		if propietario.soporte.reloj.Ahora().Before(lectura.vinculo.VigenteHasta) {
			return lectura.resultado(), nil
		}
		delete(holder.personasB2, clave)
	}
	r, err := consultar()
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.Estado == "acreditado" && r.Aceptacion != nil && r.Persona != nil && r.Vinculo != nil &&
		r.Persona.Version > 0 && r.Vinculo.ProcedenciaVersion > 0 && propietario.soporte.reloj.Ahora().Before(r.Vinculo.VigenteHasta) &&
		r.Aceptacion.OperacionRef == selector.AceptacionOperacionRef && r.Aceptacion.RegistroSHA256 == selector.AceptacionRegistroSHA256 &&
		r.Aceptacion.AperturaOperacionRef == selector.AperturaOperacionRef && r.Aceptacion.AperturaRegistroSHA256 == selector.AperturaRegistroSHA256 && r.Aceptacion.LlamamientoRef == selector.LlamamientoRef {
		if holder.personasB2 == nil {
			holder.personasB2 = map[clavePersonaBolsaB2]lecturaPersonaBolsaB2{}
		}
		holder.personasB2[clave] = lecturaPersonaBolsaB2{*r.Aceptacion, *r.Persona, *r.Vinculo, r.Evidencia}
	}
	return r, nil
}

func NuevasFuentesContextoIncorporacionB2(c ConfiguracionFuentesContextoIncorporacionB2) (*FuentesContextoIncorporacionB2, error) {
	if dependenciaBootstrapNula(c.Bolsa) || c.ActorBolsa == nil || dependenciaBootstrapNula(c.Vinculos) ||
		dependenciaBootstrapNula(c.Publicacion) || dependenciaBootstrapNula(c.AutoridadRPT) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return &FuentesContextoIncorporacionB2{c: c}, nil
}

func (f *FuentesContextoIncorporacionB2) LeerPersonaSeleccionada(ctx context.Context, c inc.ContratoPlanNominal) (inc.PersonaSeleccionadaBolsa, error) {
	var cero inc.PersonaSeleccionadaBolsa
	if f == nil || ctx == nil || dependenciaBootstrapNula(f.c.Bolsa) || f.c.ActorBolsa == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	x := c.SelectorBolsa
	if c.Protocolo != inc.ProtocoloPersonalB2V1 || x.CategoriaRef != c.CategoriaRef || x.LlamamientoRef != c.LlamamientoRef {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	// Traducción literal de los nueve selectores sellados por CT155. Bolsa
	// conserva la validación, la autorización vigente y la identidad aceptante.
	r, err := f.consultarPersonaBolsaB2(ctx, x)
	if err != nil {
		return cero, errorFuenteContextoIncorporacionB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.Estado == "pendiente" || r.Estado == "no_encontrada" {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	if r.Estado != "acreditado" || r.Aceptacion == nil || r.Persona == nil || r.Vinculo == nil ||
		r.Persona.Version < 1 || r.Vinculo.ProcedenciaVersion < 1 {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if r.Aceptacion.OperacionRef != x.AceptacionOperacionRef || r.Aceptacion.RegistroSHA256 != x.AceptacionRegistroSHA256 ||
		r.Aceptacion.AperturaOperacionRef != x.AperturaOperacionRef || r.Aceptacion.AperturaRegistroSHA256 != x.AperturaRegistroSHA256 ||
		r.Aceptacion.LlamamientoRef != x.LlamamientoRef || r.Aceptacion.ReciboRef != c.SeleccionReciboRef {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	return inc.PersonaSeleccionadaBolsa{
		OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		AceptacionRef: c.AceptacionRef, LlamamientoRef: r.Aceptacion.LlamamientoRef,
		SeleccionRef: c.SeleccionRef, VersionSeleccion: c.VersionSeleccion, ReciboRef: r.Aceptacion.ReciboRef,
		PersonaRef: r.Persona.Ref, PersonaVersion: uint64(r.Persona.Version),
		FuenteRef: r.Vinculo.ProcedenciaRef, FuenteVersion: r.Vinculo.ProcedenciaVersion, FuenteSHA256: r.Vinculo.ProcedenciaSHA256,
	}, nil
}

func (f *FuentesContextoIncorporacionB2) LeerPuestoRPT(ctx context.Context, c inc.ContratoPlanNominal) (inc.PuestoRPTNominal, error) {
	var cero inc.PuestoRPTNominal
	if f == nil || ctx == nil || dependenciaBootstrapNula(f.c.Vinculos) ||
		dependenciaBootstrapNula(f.c.Publicacion) || dependenciaBootstrapNula(f.c.AutoridadRPT) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if c.Protocolo != inc.ProtocoloPersonalB2V1 {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	q, err := ct.NuevaConsultaVinculoCategoriaRPT(c.OrganizacionRef, c.ExpedienteRef)
	if err != nil {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	l, err := f.c.Vinculos.Consultar(ctx, q)
	if err != nil {
		return cero, errorFuenteContextoIncorporacionB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if l.Vinculo == nil {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	v := l.Vinculo
	d := c.DatosPersonal
	p := dom.PublicacionCategoriaRPT{CatalogoID: d.CatalogoRPTID, ModuloID: d.ModuloRPTID,
		CatalogoVersion: d.CatalogoRPTVersion, CatalogoHuella: d.CatalogoRPTHuellaSHA256,
		CategoriaID: d.CategoriaID, CategoriaClave: d.CategoriaID}
	if l.OrganizacionRef != c.OrganizacionRef || l.ExpedienteRef != c.ExpedienteRef ||
		l.Analisis.VersionExpediente != c.VersionExpediente || l.Analisis.CategoriaRef != c.CategoriaRef ||
		v.Revision != c.VinculoRevision || v.ReciboRef != c.VinculoReciboRef || v.Publicacion() != p ||
		!p.CorrespondeA(c.CategoriaRef) || !v.Prospectivo || v.AcreditaProcedenciaHistorica {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	m, err := f.c.AutoridadRPT.LecturaRPT(ctx, p)
	if err != nil {
		return cero, errorFuenteContextoIncorporacionB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	publicada, err := f.c.Publicacion.ConsultarPublicacionCategoriaRPT(ctx, p, m)
	if err != nil {
		return cero, errorFuenteContextoIncorporacionB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if publicada != p {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	// El vínculo CT154 conserva la evidencia que acreditó esta publicación.
	// La lectura RPT actual consume su propia autorización.
	// Personal valida la fuente estructural y la pareja plaza/puesto elegida.
	return inc.PuestoRPTNominal{
		OrganizacionRef: l.OrganizacionRef, ExpedienteRef: l.ExpedienteRef, Fuente: c.FuenteRPT,
		CategoriaRef: l.Analisis.CategoriaRef, PuestoRef: c.PuestoRef, PlazaRef: c.PlazaRef,
		VinculoRevision: v.Revision, VinculoReciboRef: v.ReciboRef,
		Prospectivo: v.Prospectivo, AcreditaProcedenciaHistorica: v.AcreditaProcedenciaHistorica,
	}, nil
}

func errorFuenteContextoIncorporacionB2(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, e := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, e) {
			return e
		}
	}
	// Una caída de la autoridad prevalece sobre una denegación unida al error.
	for _, e := range []error{bp.ErrConsultaPersonaAceptacionCTNoDisponible, ct.ErrVinculoCategoriaRPTNoDisponible,
		vp.ErrLecturaRPTNoDisponible, vp.ErrLecturaRPTNoConfiable, vp.ErrFuenteAutorizacionNoDisponible,
		vp.ErrRegistroDecisionNoDisponible, vp.ErrRegistroDenegacionNoDisponible,
		vp.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, vp.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
		vp.ErrInstantaneaAutorizacionObsoleta} {
		if errors.Is(err, e) {
			return ct.ErrComposicionIncorporacionAplicacion
		}
	}
	for _, e := range []error{bp.ErrConsultaPersonaAceptacionCTDenegada, ct.ErrVinculoCategoriaRPTDenegado,
		vp.ErrLecturaRPTDenegada, ct.ErrAutorizacionDenegada, ct.ErrDenegadaIncorporacionAplicacion,
		core.ErrAutorizacionDenegada, core.ErrPermissionDenied, vp.ErrDenegacionExplicitaAutorizacionLigadaV3} {
		if errors.Is(err, e) {
			return ct.ErrDenegadaIncorporacionAplicacion
		}
	}
	if errors.Is(err, ct.ErrVinculoCategoriaRPTNoEncontrado) {
		return ct.ErrPreparacionIncorporacionPendiente
	}
	if errors.Is(err, ct.ErrVinculoCategoriaRPTConflicto) || errors.Is(err, bp.ErrConsultaPersonaAceptacionCTInvalida) ||
		errors.Is(err, ct.ErrVinculoCategoriaRPTInvalido) || errors.Is(err, vp.ErrLecturaRPTInvalida) {
		return ct.ErrConflictoIncorporacionAplicacion
	}
	return ct.ErrComposicionIncorporacionAplicacion
}

var _ inc.FuentePersonaSeleccionadaBolsa = (*FuentesContextoIncorporacionB2)(nil)
var _ inc.FuentePuestoRPTNominal = (*FuentesContextoIncorporacionB2)(nil)
