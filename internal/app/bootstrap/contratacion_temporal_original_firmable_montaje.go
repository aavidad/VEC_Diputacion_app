package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecapplication "vec-diputacion-granada/internal/vec/application"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	envCTOriginalFirmableEnabled = "VEC_CT_ORIGINAL_FIRMABLE_ENABLED"
	clavePerfilOriginalDetalleCT = "original_firmable_ct_detalle"
)

var errMontajeOriginalFirmableCTNoDisponible = errors.New("contratacion temporal: original firmable no disponible")

// El original consume dos asignaciones ya publicadas: CT detalle tiene tres
// ámbitos y Documentos uno. Los perfiles pertenecen a la misma persona y
// cuenta; no atienden ninguna ruta directamente ni se eligen desde HTTP.
type perfilesNominalesOriginalCT struct {
	soporte    *soporteAltaContratacionTemporalDesarrollo
	detalle    *perfilFijoCTDesarrollo
	documentos *perfilFijoCTDesarrollo
}

type capturaContextosOriginalCT struct{ capturaPerfilesNominalesCT }

func (p *perfilesNominalesOriginalCT) resolver(ctx context.Context, perfil *perfilFijoCTDesarrollo) (ports.ContextoAutorizacionAltaV3, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	if p == nil || p.soporte == nil || ctx == nil || (perfil != p.detalle && perfil != p.documentos) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	c, valida := p.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaOriginalFirmableCT || c.metodo != http.MethodPost || c.original == nil ||
		c.vinculoCanalTLS == ([32]byte{}) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	return resolverContextoPerfilNominalCT(ctx, p.soporte, perfil, &c.original.capturaPerfilesNominalesCT, p)
}

func nuevoPerfilNominalOriginalCT(s *soporteAltaContratacionTemporalDesarrollo, principal core.Principal,
	clave string, ahora time.Time) (*perfilFijoCTDesarrollo, error) {
	if s == nil || principal.ID == "" || principal.ID != s.principalID ||
		principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh ||
		principal.Attributes["certificate_sha256"] != s.certificadoSHA256 || ahora.IsZero() {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	base, err := s.contexto.Vinculo.Datos()
	if err != nil || base.PrincipalID == "" || base.CuentaRef == "" || base.PerfilActivoRef == "" {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorPerfilFijoCTDesarrollo(clave))
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil || v.PrincipalID != base.PrincipalID || v.CuentaRef != base.CuentaRef ||
		v.CuentaOrdinariaRef != base.CuentaOrdinariaRef || v.CuentaPrivilegiada != base.CuentaPrivilegiada ||
		v.Superficie != base.Superficie || v.PerfilActivoRef == base.PerfilActivoRef ||
		contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	var plantilla core.InstantaneaAutorizacion
	switch clave {
	case clavePerfilOriginalDetalleCT:
		plantilla, err = nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora,
			clave, clave, clave, []core.ConcesionRol{{Accion: ports.AccionConsultarDetalleRRHH, ModuloID: ports.ModuloContratacion,
				TipoRecurso: ports.TipoRecursoExpediente, Finalidades: []string{ports.FinalidadConsultarDetalleRRHH},
				GarantiaMinima: core.AuthAssuranceHigh}}, []core.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
				{Clave: "clase_ambito", Valores: []string{string(ports.AmbitoOrganizacionRRHH)}},
				{Clave: "ambito_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			})
	case clavePerfilFijoOriginalFirmableCTDesarrollo:
		plantilla, err = instantaneaOriginalFirmableCTDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora)
	default:
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	if err != nil || plantilla.Validar() != nil || plantilla.AsignacionPerfil.PerfilActivoRef != v.PerfilActivoRef {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	return &perfilFijoCTDesarrollo{clave: clave, rutas: map[string]struct{}{}, contexto: contexto, plantilla: plantilla,
		actoSesion: actoSesionPerfilFijoCTDesarrollo}, nil
}

func nuevosPerfilesNominalesOriginalCT(s *soporteAltaContratacionTemporalDesarrollo,
	principal core.Principal, ahora time.Time) (*perfilesNominalesOriginalCT, error) {
	detalle, err := nuevoPerfilNominalOriginalCT(s, principal, clavePerfilOriginalDetalleCT, ahora)
	if err != nil {
		return nil, fmt.Errorf("%w: perfil detalle: %w", errMontajeOriginalFirmableCTNoDisponible, err)
	}
	documentos, err := nuevoPerfilNominalOriginalCT(s, principal, clavePerfilFijoOriginalFirmableCTDesarrollo, ahora)
	if err != nil {
		return nil, fmt.Errorf("%w: perfil documentos: %w", errMontajeOriginalFirmableCTNoDisponible, err)
	}
	if detalle.perfilRef() == documentos.perfilRef() || detalle.contexto.Resultado.Contexto.PersonaRef != documentos.contexto.Resultado.Contexto.PersonaRef ||
		detalle.contexto.Resultado.Contexto.Instantanea.CuentaRef != documentos.contexto.Resultado.Contexto.Instantanea.CuentaRef {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	return &perfilesNominalesOriginalCT{soporte: s, detalle: detalle, documentos: documentos}, nil
}

func originalFirmableCTSolicitado(cfg config.Config) (bool, error) {
	return selectorCapacidadRRHHDesarrollo(cfg, envCTOriginalFirmableEnabled)
}

type autoridadCanalOriginalCTDesarrollo struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
}

func (a autoridadCanalOriginalCTDesarrollo) ResolverOrganizacionFirmaVec(ctx context.Context) (string, error) {
	if a.soporte == nil || ctx == nil || ctx.Err() != nil || a.soporte.perfilesOriginal == nil {
		return "", errOriginalFirmableCTDenegado
	}
	c, valida := a.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaOriginalFirmableCT || c.metodo != http.MethodPost ||
		c.consultaRRHH == nil || c.contextoOperacion == nil || c.original == nil || c.vinculoCanalTLS == ([32]byte{}) {
		return "", errOriginalFirmableCTDenegado
	}
	return organizacionAltaContratacionTemporalDesarrollo, nil
}

// nuevaRutaOriginalFirmableCTDesarrollo reutiliza Servicio Documentos y sus
// recursos; sólo cambia la fábrica de lectura del objeto para el PDP CT de
// esta petición. El original no registra firma ni resolución CT.
func nuevaRutaOriginalFirmableCTDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo,
	consultas dependenciasConsultasRRHHDesarrollo, render ports.RenderizadorBorradorRRHH,
	documentos *autoridadDocumentosDesarrollo, seudonimos *seudonimizadorAlmacenDesarrollo,
	reloj ports.Reloj) ([]vechttp.RutaExacta, error) {
	if alta == nil || alta.soporte == nil || alta.soporte.perfilesOriginal == nil ||
		consultas.detalle == nil || consultas.originalPropuesta == nil || render == nil ||
		documentos == nil || documentos.servicioOriginal == nil || !seudonimos.valido() || reloj == nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	politicas, ok := documentos.servicioOriginal.Politicas.(*conservacion.Catalogo)
	if !ok || politicas == nil || documentos.servicioOriginal.Repositorio == nil || documentos.servicioOriginal.Almacen == nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	tipos, err := ctadapters.NuevosTiposOriginalFirmableRRHH(politicas)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	autoridad, err := nuevoOriginalFirmableCTDesarrollo(pdpCTOriginalFirmableDesarrollo{alta: alta}, seudonimos, politicas, reloj)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	lectura, err := docautorizacion.NuevaFabricaContextoLecturaOriginalV3(autoridad, reloj)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	copia := *documentos.servicioOriginal
	copia.ContextosLectura = lectura
	custodia, err := almacen.NuevaCustodiaDocumentosOriginalCT(servicioDocumentosOriginalCTDesarrollo{servicio: &copia},
		autoridad, autoridad.mapear, tipos)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	origen, err := ctapplication.NuevaFuenteOriginalFirmableRRHH(consultas.detalle, consultas.originalPropuesta, render, tipos)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	servicio, err := vecapplication.NuevoServicioOriginalFirmableCT(origen, custodia, tipos)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	h, err := httpinterno.NuevoManejadorOriginalFirmableCT(autoridadCanalOriginalCTDesarrollo{alta.soporte}, servicio)
	if err != nil {
		return nil, errMontajeOriginalFirmableCTNoDisponible
	}
	return []vechttp.RutaExacta{{Ruta: httpinterno.RutaOriginalFirmableCT, Manejador: h}}, nil
}
