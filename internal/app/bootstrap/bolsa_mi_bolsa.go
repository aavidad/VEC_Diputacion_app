package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"time"

	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errMiBolsaNoDisponible = errors.New("bolsa: consulta personal de desarrollo no disponible")

func motivoMiBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_mi_bolsa_desarrollo", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-mi-bolsa-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-mi-bolsa-consultar"),
	}
}

// motivoPortalMiBolsaDesarrollo motiva las acciones propias del candidato
// (AD3-84); la consulta conserva su propio motivo.
func motivoPortalMiBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_portal_mi_bolsa_desarrollo", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-portal-mi-bolsa-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-mi-bolsa-portal"),
	}
}

// El material sólo asigna una cuenta y un perfil a una hoja mTLS. El vínculo
// candidato procede siempre del resolutor durable de contexto_actor_v1.
type preparadorMiBolsaDesarrollo struct {
	sello     *selloConsultasContratacionTemporalDesarrollo
	identidad *identidadCandidatoBolsaDesarrollo
	sesion    *proveedorSesionConsultaRRHHDesarrollo
	reloj     relojContratacionTemporalDesarrollo
}

func (p *preparadorMiBolsaDesarrollo) PrepararMiBolsa(r *http.Request) (mibolsa.Orden, error) {
	if p == nil || r == nil || p.sello == nil || p.identidad == nil || p.sesion == nil {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	capacidad, ok := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !ok || capacidad.sello != p.sello || r.URL == nil || capacidad.ruta != r.URL.Path || !bolsapersonal.EsRutaPortal(capacidad.ruta) ||
		capacidad.principal.ID != p.identidad.identidad.principal.ID ||
		capacidad.principal.Attributes["certificate_sha256"] != p.identidad.identidad.principal.Attributes["certificate_sha256"] ||
		len(capacidad.principal.Roles) != 1 || capacidad.principal.Roles[0] != "candidato_bolsa" ||
		capacidad.certificadoVerificadoEn.IsZero() || capacidad.certificadoVerificadoEn.After(ahora) ||
		!ahora.Before(capacidad.certificadoValidoHasta) || !ahora.Before(p.identidad.validoHasta) {
		return mibolsa.Orden{}, bolsapersonal.ErrAutenticacionAusente
	}
	registrado, err := p.sesion.ResolverContexto(r.Context())
	if err != nil || registrado.Vinculo.ValidarPara(registrado.Resultado) != nil ||
		registrado.Resultado.Contexto.PersonaRef != p.identidad.personaRef ||
		registrado.Resultado.Contexto.PerfilActivoRef != p.identidad.perfilRef {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	candidatos := 0
	for _, v := range registrado.Resultado.Contexto.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			candidatos++
			if v.Referencia != p.identidad.candidatoRef {
				return mibolsa.Orden{}, errMiBolsaNoDisponible
			}
		}
	}
	if candidatos != 1 {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	motivo := motivoMiBolsaDesarrollo()
	if capacidad.ruta != bolsapersonal.RutaMiBolsa {
		motivo = motivoPortalMiBolsaDesarrollo()
	}
	return mibolsa.Orden{ResultadoContexto: registrado.Resultado, Vinculo: registrado.Vinculo, Motivo: motivo, Correlacion: correlacion}, nil
}

func vincularCertificadoMiBolsaDesarrollo(ctx context.Context, resolutor dominiovec.ResolutorContextoActorRegistradoV2, identidad *identidadCandidatoBolsaDesarrollo, reloj relojContratacionTemporalDesarrollo, ahora, verificadoEn, validoHasta time.Time) (dominiovec.VinculoAutenticacionActorV2, dominiovec.ResultadoContextoActorRegistradoV2, error) {
	vacio := dominiovec.ResultadoContextoActorRegistradoV2{}
	if ctx == nil || resolutor == nil || identidad == nil || ahora.IsZero() ||
		verificadoEn.IsZero() || verificadoEn.After(ahora) || !ahora.Before(validoHasta) {
		return dominiovec.VinculoAutenticacionActorV2{}, vacio, errMiBolsaNoDisponible
	}
	var aleatorio [24]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return dominiovec.VinculoAutenticacionActorV2{}, vacio, errMiBolsaNoDisponible
	}
	base := identidad.identidad.principal.ID + "\x00" + identidad.identidad.principal.Attributes["certificate_sha256"] + "\x00" + hex.EncodeToString(aleatorio[:])
	hasta := ahora.Add(5 * time.Minute)
	if validoHasta.Before(hasta) {
		hasta = validoHasta
	}
	if !hasta.After(ahora) {
		return dominiovec.VinculoAutenticacionActorV2{}, vacio, errMiBolsaNoDisponible
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:          referenciaAltaContratacionTemporalDesarrollo("aut_", base+"\x00aut"),
		AutenticacionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(base + "\x00aut"),
		AsercionRef:               referenciaAltaContratacionTemporalDesarrollo("ase_", base+"\x00ase"),
		SesionRef:                 referenciaAltaContratacionTemporalDesarrollo("ses_", base+"\x00ses"),
		ControlSesionRef:          referenciaAltaContratacionTemporalDesarrollo("cse_", base+"\x00cse"),
		ControlSesionRevision:     1,
		ControlSesionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(base + "\x00cse"),
		CuentaRef:                 identidad.cuentaRef, CuentaOrdinariaRef: identidad.cuentaRef,
		Superficie:      dominiovec.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado: dominiovec.AuthMethodCertificate, GarantiaObservada: dominiovec.AuthAssuranceHigh,
		PoliticaGarantiaRef:          referenciaAltaContratacionTemporalDesarrollo("pga_", "bolsa-mi-bolsa-mtls-desarrollo-v1"),
		PoliticaGarantiaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("bolsa-mi-bolsa-mtls-desarrollo-v1"),
		AutenticacionVerificadaEn:    verificadoEn,
		SesionEmitidaEn:              verificadoEn,
		SesionRevalidadaEn:           verificadoEn,
		SesionValidaHasta:            hasta,
	}
	vinculo, resultado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
		ctx, revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutor,
		dominiovec.SolicitudContextoActor{Cuenta: dominiovec.CuentaAutenticadaContextoActor{CuentaRef: identidad.cuentaRef, Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh}, PerfilActivoRef: identidad.perfilRef},
		reloj,
	)
	if err != nil || vinculo.ValidarPara(resultado) != nil ||
		resultado.Contexto.PersonaRef != identidad.personaRef ||
		resultado.Contexto.PerfilActivoRef != identidad.perfilRef {
		return dominiovec.VinculoAutenticacionActorV2{}, vacio, errMiBolsaNoDisponible
	}
	candidatos := 0
	for _, v := range resultado.Contexto.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			candidatos++
			if v.Referencia != identidad.candidatoRef {
				return dominiovec.VinculoAutenticacionActorV2{}, vacio, errMiBolsaNoDisponible
			}
		}
	}
	if candidatos != 1 {
		return dominiovec.VinculoAutenticacionActorV2{}, vacio, errMiBolsaNoDisponible
	}
	return vinculo, resultado, nil
}

type politicaMiBolsaDesarrollo struct {
	instantanea dominiovec.InstantaneaAutorizacion
	registro    registroDecisionesAnalisisContratacionTemporalDesarrollo
	motivo      dominiovec.ReferenciaEntradaCatalogo
	// motivoPortal solo existe si están compuestas las acciones propias.
	motivoPortal *dominiovec.ReferenciaEntradaCatalogo
}

// motivoDe devuelve el motivo que corresponde a la acción, o falso si la
// política no la admite.
func (p *politicaMiBolsaDesarrollo) motivoDe(accion string) (dominiovec.ReferenciaEntradaCatalogo, bool) {
	if accion == puertosbolsa.AccionConsultarMiBolsa {
		return p.motivo, true
	}
	if p.motivoPortal == nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, false
	}
	for _, par := range puertosbolsa.AccionesPortalCandidato() {
		if par[0] == accion {
			return *p.motivoPortal, true
		}
	}
	return dominiovec.ReferenciaEntradaCatalogo{}, false
}

func (p *politicaMiBolsaDesarrollo) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || p.instantanea.Validar() != nil ||
		principal != p.instantanea.AsignacionPerfil.PrincipalID || perfil != p.instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.instantanea), nil
}
func (p *politicaMiBolsaDesarrollo) ValidarReferenciaMotivoAutorizacionV2(ctx context.Context, motivo dominiovec.ReferenciaEntradaCatalogo, ahora time.Time) error {
	if p == nil || ctx == nil || ctx.Err() != nil || (p.motivo != motivo && (p.motivoPortal == nil || *p.motivoPortal != motivo)) || !p.instantanea.AsignacionPerfil.VigenteEn(ahora) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}
func (p *politicaMiBolsaDesarrollo) ordenValida(ctx context.Context, orden interface {
	Datos() (puertosvec.DatosOrdenRegistroAutorizacionLigadaV3, error)
}) bool {
	if p == nil || ctx == nil || ctx.Err() != nil || orden == nil {
		return false
	}
	if p.instantanea.Validar() != nil || len(p.instantanea.AsignacionPerfil.Ambitos) != 1 ||
		len(p.instantanea.AsignacionPerfil.Ambitos[0].Valores) != 1 {
		return false
	}
	datos, err := orden.Datos()
	if err != nil || datos.ResultadoContexto.Validar() != nil || datos.ResultadoContexto.Contexto.PersonaRef != p.instantanea.AsignacionPerfil.PrincipalID ||
		datos.ResultadoContexto.Contexto.PerfilActivoRef != p.instantanea.AsignacionPerfil.PerfilActivoRef {
		return false
	}
	solicitud, err := datos.Solicitud.Datos()
	motivo, admitida := p.motivoDe(solicitud.Accion)
	return err == nil && admitida && datos.ReferenciaMotivo == motivo && solicitud.ReferenciaMotivo == motivo &&
		solicitud.Recurso.Ambitos["candidato_ref"] == p.instantanea.AsignacionPerfil.Ambitos[0].Valores[0]
}
func (p *politicaMiBolsaDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	if !p.ordenValida(ctx, orden) || p.registro == nil {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	return p.registro.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}
func (p *politicaMiBolsaDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	if !p.ordenValida(ctx, orden) || p.registro == nil {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return p.registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

type autorizadorMiBolsaDesarrollo struct {
	delegado  *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
	identidad *identidadCandidatoBolsaDesarrollo
	sello     *selloConsultasContratacionTemporalDesarrollo
}

func (a *autorizadorMiBolsaDesarrollo) ExigirSolicitudLigadaV3(ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	if a == nil || a.delegado == nil || a.identidad == nil || ctx == nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	datos, err := solicitud.Datos()
	metodo := http.MethodPost
	if capacidad.ruta == bolsapersonal.RutaMiBolsa {
		metodo = http.MethodGet
	}
	if !ok || a.sello == nil || capacidad.sello != a.sello || !bolsapersonal.EsRutaPortal(capacidad.ruta) ||
		capacidad.principal.ID != a.identidad.identidad.principal.ID ||
		capacidad.principal.Attributes["certificate_sha256"] != a.identidad.identidad.principal.Attributes["certificate_sha256"] ||
		len(capacidad.principal.Roles) != 1 || capacidad.principal.Roles[0] != "candidato_bolsa" || err != nil ||
		!slices.Contains(bolsapersonal.AccionPortalEn(metodo, capacidad.ruta), datos.Accion) ||
		datos.Recurso.Referencia != "mi-bolsa:"+a.identidad.candidatoRef ||
		datos.Recurso.Ambitos["candidato_ref"] != a.identidad.candidatoRef {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

type proveedorMiBolsaDesarrollo struct {
	delegado *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (p proveedorMiBolsaDesarrollo) EmitirMaterialMiBolsa(ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	material, err := emitirMaterialNominalMiBolsaDesarrollo(ctx, p.delegado, motivoMiBolsaDesarrollo(), solicitud, resultado, decision, confirmacion)
	if err != nil {
		return nil, puertosbolsa.ErrMaterialMiBolsaNoDisponible
	}
	return material, nil
}

// proveedorPortalMiBolsaDesarrollo emite el material de cada acción propia
// con el proveedor de su audiencia; una acción sin proveedor no se atiende.
type proveedorPortalMiBolsaDesarrollo struct {
	porAccion map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (p proveedorPortalMiBolsaDesarrollo) EmitirMaterialPortalCandidato(ctx context.Context, accion string, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	delegado := p.porAccion[accion]
	if delegado == nil {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	material, err := emitirMaterialNominalMiBolsaDesarrollo(ctx, delegado, motivoPortalMiBolsaDesarrollo(), solicitud, resultado, decision, confirmacion)
	if err != nil {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	return material, nil
}

func emitirMaterialNominalMiBolsaDesarrollo(ctx context.Context, delegado *proveedorMaterialAltaContratacionTemporalDesarrollo, motivo dominiovec.ReferenciaEntradaCatalogo, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if delegado == nil || delegado.atestador == nil || delegado.confianza == nil || delegado.emisor == nil {
		return nil, errMiBolsaNoDisponible
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, resultado)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return nil, errMiBolsaNoDisponible
	}
	atestacion, err := delegado.atestador.Atestar(ctx, decision, motivo, resultado)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	prueba, err := delegado.Verificar(ctx, solicitud, decision, motivo, resultado, atestacion)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	capacidad, err := delegado.emisor.Emitir(ctx, solicitud, decision, motivo, resultado, atestacion, prueba)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	material, err := confianzaatestacion.NuevoMaterialConsumoAutorizacionAtestadaV3(solicitud, decision, motivo, resultado, atestacion, prueba, capacidad, delegado.raiz)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	return material, nil
}

// concesionesPortalMiBolsaDesarrollo concede las acciones propias AD3-84:
// sin campos ni obligaciones, con la misma garantía que la consulta.
func concesionesPortalMiBolsaDesarrollo() []dominiovec.ConcesionRol {
	concesiones := make([]dominiovec.ConcesionRol, 0, 4)
	for _, par := range puertosbolsa.AccionesPortalCandidato() {
		tipo := puertosbolsa.TipoRecursoMiBolsa
		if par[0] == puertosbolsa.AccionManifestarDisposicionPropia {
			tipo = puertosbolsa.TipoRecursoOfertaBolsa
		}
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: par[0], ModuloID: puertosbolsa.ModuloMiBolsa, TipoRecurso: tipo,
			Finalidades: []string{puertosbolsa.FinalidadPortalCandidato}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		})
	}
	return concesiones
}

func nuevaInstantaneaMiBolsaDesarrollo(identidad *identidadCandidatoBolsaDesarrollo, ahora time.Time, portal ...bool) (dominiovec.InstantaneaAutorizacion, error) {
	if identidad == nil || identidad.candidatoRef == "" || identidad.personaRef == "" || identidad.perfilRef == "" {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	rol := dominiovec.VersionRol{
		RolID: "candidato_bolsa_consulta_propia_desarrollo", Version: 1,
		Nombre: "Consulta propia de bolsa en desarrollo", Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: puertosbolsa.AccionConsultarMiBolsa, ModuloID: puertosbolsa.ModuloMiBolsa,
			TipoRecurso:      puertosbolsa.TipoRecursoMiBolsa,
			Finalidades:      []string{puertosbolsa.FinalidadMiBolsa},
			CamposPermitidos: []string{puertosbolsa.CampoMiBolsa}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: desde,
	}
	if len(portal) == 1 && portal[0] {
		// Rol distinto (no una versión nueva del de consulta): la asignación
		// sube de versión al cambiar de rol y la historia anterior se conserva.
		rol.RolID, rol.Nombre = "candidato_bolsa_portal_propio_desarrollo", "Consulta y acciones propias de bolsa en desarrollo"
		rol.Concesiones = append(rol.Concesiones, concesionesPortalMiBolsaDesarrollo()...)
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: referenciaAltaContratacionTemporalDesarrollo("asg_", identidad.personaRef+"\x00"+identidad.perfilRef+"\x00bolsa-mi-bolsa-v1"),
		Version:      1, PerfilActivoRef: identidad.perfilRef, PrincipalID: identidad.personaRef,
		VersionRolRef: rol.Referencia(), Estado: dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:      []dominiovec.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{identidad.candidatoRef}}},
		VigenteDesde: desde, VigenteHasta: hasta,
		EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: desde,
	}
	huella, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: rol,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: rol.Referencia(), Revision: 1, Estado: dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: rol.PublicadaPor, ActualizadoEn: desde,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	return instantanea, nil
}

func nuevaRutaMiBolsaDesarrollo(
	ctx context.Context, identidad *identidadCandidatoBolsaDesarrollo,
	sello *selloConsultasContratacionTemporalDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	identidadCT *proveedorSesionConsultaRRHHDesarrollo,
	fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
	campos puertosbolsa.CamposPortalMiBolsa,
	portal puertosbolsa.ReglasPortalCandidato,
) ([]vechttp.RutaExacta, error) {
	if ctx == nil || identidad == nil || sello == nil || alta == nil || alta.soporte == nil ||
		alta.postgresql.bolsa == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.proveedorMaterialMiBolsa == nil || identidadCT == nil ||
		identidadCT.resolutor == nil || derivador == nil || !derivador.valido() ||
		alta.soporte.registroDecisionesAnalisis == nil ||
		(portal != nil && len(alta.postgresql.proveedoresMaterialPortal) != 3) {
		return nil, errMiBolsaNoDisponible
	}
	// ResolverRegistrado escribe el recibo rca_ mediante contexto_actor_v1.
	// La fuente privada sólo aporta las referencias a la consulta cerrada.
	vinculo, resultado, err := vincularCertificadoMiBolsaDesarrollo(ctx, identidadCT.resolutor, identidad, reloj, reloj.Ahora().UTC().Truncate(time.Microsecond), identidad.verificadoEn, identidad.validoHasta)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: identidad.identidad.principal.ID,
		certificadoSHA256: identidad.identidad.principal.Attributes["certificate_sha256"],
		candidatoBolsa:    true, contexto: ctports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado},
		contextoEsperadoRegistrado: resultado, reloj: reloj,
	}
	// La cuenta técnica usa el alias seudónimo de identidad ya existente. Su
	// perfil de arranque es sólo un fixture para el proyector de cuenta; el
	// contexto personal real se resolvió arriba y se vuelve a resolver por GET.
	semillaCuenta, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(identidad.identidad.principal, reloj.Ahora())
	if err != nil || semillaCuenta.Resultado.Contexto.Instantanea.CuentaRef != identidad.cuentaRef ||
		semillaCuenta.Resultado.Contexto.PersonaRef != identidad.personaRef {
		return nil, errMiBolsaNoDisponible
	}
	soporteCuenta := &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: identidad.identidad.principal.ID,
		certificadoSHA256: identidad.identidad.principal.Attributes["certificate_sha256"],
		contexto:          semillaCuenta, reloj: reloj,
	}
	seudonimizador := &seudonimizadorSesionDesarrollo{derivador: derivador}
	seudonimos, err := seudonimizador.SeudonimizarAlta(ctx, postgresidentidad.IdentificadoresAlta{
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID:       "preparacion-cuenta", SesionID: "preparacion-alias",
		CuentaID: "desarrollo:" + identidad.cuentaRef, SujetoID: identidad.identidad.principal.ID,
	})
	if err != nil || prepararCuentaNominalConsultasDesarrollo(ctx, alta.postgresql.gobierno, soporteCuenta, seudonimos) != nil {
		return nil, errMiBolsaNoDisponible
	}
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(
		soporte, identidadCT.registro, identidadCT.revalidador, reloj, identidadCT.resolutor, fronteras,
	)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	autoridad := autoridadPostgreSQLDesarrollo{
		pool: alta.postgresql.gobierno, vinculo: vinculo,
		prefijoBloqueo: "vec:bolsa:mi-bolsa:autorizacion:",
		actoControlRol: "acto:bolsa:mi-bolsa:control-rol:v1",
		actoAsignacion: "acto:bolsa:mi-bolsa:asignacion:v1",
		actoSesion:     "acto:bolsa:mi-bolsa:sesion:v1",
	}
	semilla, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, reloj.Ahora(), portal != nil)
	if err != nil || !autoridad.validaConfiguracion() {
		return nil, errMiBolsaNoDisponible
	}
	preparada, err := autoridad.prepararInstantanea(ctx, semilla, true)
	if err != nil || preparada.Validar() != nil || autoridad.publicarInstantanea(ctx, preparada) != nil {
		return nil, errMiBolsaNoDisponible
	}
	politica := &politicaMiBolsaDesarrollo{instantanea: preparada, registro: alta.soporte.registroDecisionesAnalisis, motivo: motivoMiBolsaDesarrollo()}
	// Instante fijo de la ventana sintética, como el resto de catálogos: el
	// replay del publicador exige el mismo publicado_en y el reloj rompería
	// cualquier arranque posterior al primero.
	desdeMotivos, _, vigenteMotivos := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigenteMotivos || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{politica.motivo}, desdeMotivos) != nil {
		return nil, errMiBolsaNoDisponible
	}
	if portal != nil {
		motivoPortal := motivoPortalMiBolsaDesarrollo()
		if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{motivoPortal}, reloj.Ahora()) != nil {
			return nil, errMiBolsaNoDisponible
		}
		politica.motivoPortal = &motivoPortal
	}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		politica, politica, politica, politica, reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	consulta, err := postgresbolsa.NuevaConsultaMiBolsaPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	autorizadorPropio := &autorizadorMiBolsaDesarrollo{delegado: autorizador, identidad: identidad, sello: sello}
	servicio, err := mibolsa.Nuevo(consulta, autorizadorPropio, proveedorMiBolsaDesarrollo{delegado: alta.postgresql.proveedorMaterialMiBolsa}, reloj)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	if portal != nil {
		if servicio, err = servicio.ConReglasPortal(portal); err != nil {
			return nil, errMiBolsaNoDisponible
		}
	}
	preparador := &preparadorMiBolsaDesarrollo{sello: sello, identidad: identidad, sesion: sesion, reloj: reloj}
	var consultaHTTP http.Handler
	if campos == nil {
		consultaHTTP, err = bolsapersonal.Nuevo(preparador, servicio)
	} else {
		consultaHTTP, err = bolsapersonal.NuevoConCampos(preparador, servicio, campos)
	}
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	rutas := []vechttp.RutaExacta{{Ruta: bolsapersonal.RutaMiBolsa, Manejador: consultaHTTP}}
	if portal == nil {
		return rutas, nil
	}
	registro, err := postgresbolsa.NuevoRegistroPortalCandidatoPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	acciones, err := mibolsa.NuevoPortal(registro, autorizadorPropio, proveedorPortalMiBolsaDesarrollo{porAccion: alta.postgresql.proveedoresMaterialPortal}, portal, reloj)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	for _, ruta := range []string{bolsapersonal.RutaMiBolsaSolicitudes, bolsapersonal.RutaMiBolsaRespuestas} {
		manejador, err := bolsapersonal.NuevoPortal(ruta, preparador, acciones)
		if err != nil {
			return nil, errMiBolsaNoDisponible
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: ruta, Manejador: manejador})
	}
	return rutas, nil
}
