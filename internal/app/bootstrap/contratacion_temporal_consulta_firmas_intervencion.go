package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	consultafirmas "vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const clavePerfilFijoLectorFirmasIntervencionCT = "lector_firmas_intervencion"

func nuevaInstantaneaLectorFirmasIntervencionCTDesarrollo(
	principalID, perfilRef string, ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef,
		ahora, "intervencion_firmas_lector_desarrollo",
		"Consulta de firmas de expediente para fiscalización de desarrollo",
		"intervencion-firmas-lector-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{{
			Accion: ports.AccionConsultarFirmasDocumento, ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoConsultaFirmasDocumento,
			Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
			CamposPermitidos: consultafirmas.CamposConsultaFirmasDocumento(),
		}}, []dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

// Este lector sólo acompaña el POST interno de fiscalización. Su perfil de
// lectura pertenece a la misma persona de Intervención, pero conserva sesión,
// vínculo y concesión distintos del perfil que registra la fiscalización.
type lectorFirmasIntervencionCTDesarrollo struct {
	canal       *soporteFiscalizacionContratacionTemporalDesarrollo
	puente      *soporteAltaContratacionTemporalDesarrollo
	perfil      *perfilFijoCTDesarrollo
	sesion      proveedorSesionOperativaCTDesarrollo
	esperado    dominiovec.ResultadoContextoActorRegistradoV2
	firma       *firmaDocumentoCTDesarrollo
	autorizador autorizadorLigadoContratacionTemporalDesarrollo
}

// nuevoLectorFirmasIntervencionCTDesarrollo publica una plantilla nueva sólo
// si no existe asignación. Una asignación alterada se conserva y deniega hasta
// provisión con preimagen aprobada y CAS por la autoridad existente.
func nuevoLectorFirmasIntervencionCTDesarrollo(
	ctx context.Context, firma *firmaDocumentoCTDesarrollo,
	canal *soporteFiscalizacionContratacionTemporalDesarrollo,
	base *proveedorSesionConsultaRRHHDesarrollo,
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
) (*lectorFirmasIntervencionCTDesarrollo, error) {
	fallo := ports.ErrRegistroFirmaDocumentoNoDisponible
	if ctx == nil || ctx.Err() != nil || firma == nil || firma.alta == nil ||
		firma.alta.postgresql.gobierno == nil || firma.alta.postgresql.proveedorMaterialConsultaFirmasDocumento == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(firma.lector) || canal == nil ||
		canal.puente == nil || canal.fijo == nil || base == nil || canal.sello == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(canal.reloj) {
		return nil, fallo
	}
	actor := canal.contexto.Resultado.Contexto
	vinculo, err := canal.contexto.Vinculo.Datos()
	if actor.Principal.ID != canal.principalID ||
		actor.Principal.Attributes["certificate_sha256"] != canal.certificadoSHA256 ||
		!principalIntervencionContratacionTemporalDesarrolloValido(actor.Principal) ||
		err != nil ||
		canal.contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: vinculo.AutenticacionRef,
			SesionRef:        vinculo.SesionRef,
			PerfilRef:        actor.PerfilActivoRef,
		}, canal.reloj.Ahora()) != nil {
		return nil, fallo
	}
	perfil, err := nuevoPerfilFijoCTDesarrollo(actor.Principal, canal.contexto, canal.reloj.Ahora(),
		clavePerfilFijoLectorFirmasIntervencionCT, []string{httpinterno.RutaResultadosFiscalizacion},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaLectorFirmasIntervencionCTDesarrollo(principalID, perfilRef, canal.reloj.Ahora())
		})
	if err != nil || perfil == nil || perfil.perfilRef() == canal.fijo.perfilRef() {
		return nil, fallo
	}
	perfil.metodo = http.MethodPost
	perfil.propioDelSoporte = true
	puente := &soporteAltaContratacionTemporalDesarrollo{
		sello: canal.sello, principalID: canal.principalID, certificadoSHA256: canal.certificadoSHA256,
		contexto: perfil.contexto, reloj: canal.reloj,
	}
	puente.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{
		pool: firma.alta.postgresql.gobierno, soporte: puente,
	}
	if err := publicarContextoPerfilFijoCTDesarrollo(ctx, firma.alta.postgresql.gobierno, perfil); err != nil {
		return nil, fallo
	}
	if _, err := asegurarPerfilFijoCTDesarrollo(ctx, firma.alta.postgresql.gobierno, puente, perfil,
		aprobacion, preimagenPropiaPerfilFijoCTDesarrollo(perfil, actoAsignacionPerfilFijoCTDesarrollo)); err != nil {
		return nil, fallo
	}
	sesion, err := nuevaSesionConsultaFirmasIntervencionCTDesarrollo(ctx, base, canal, puente, perfil)
	if err != nil {
		return nil, fallo
	}
	lector := &lectorFirmasIntervencionCTDesarrollo{canal: canal, puente: puente, perfil: perfil,
		sesion: sesion, esperado: sesion.base, firma: firma}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		lector, lector, lector, lector, canal.reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, fallo
	}
	lector.autorizador = autorizador
	return lector, nil
}

func (l *lectorFirmasIntervencionCTDesarrollo) capacidadValida(ctx context.Context) bool {
	if l == nil || l.canal == nil || !l.canal.capacidadValida(ctx) || ctx == nil || ctx.Err() != nil {
		return false
	}
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	return ok && c.metodo == http.MethodPost && c.ruta == httpinterno.RutaResultadosFiscalizacion
}

func (l *lectorFirmasIntervencionCTDesarrollo) contextoOperativo(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	if l == nil || l.perfil == nil || l.sesion == nil || l.esperado.Validar() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if !l.capacidadValida(ctx) {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	actual, err := l.sesion.ResolverContexto(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ports.ContextoAutorizacionAltaV3{}, ctx.Err()
		}
		if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
			return ports.ContextoAutorizacionAltaV3{}, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	if actual.Vinculo.ValidarPara(actual.Resultado) != nil ||
		!mismoContextoEsperadoRegistradoDesarrollo(l.esperado, actual.Resultado) ||
		actual.Resultado.Contexto.PerfilActivoRef != l.perfil.perfilRef() ||
		actual.Resultado.Contexto.Principal.ID != l.canal.principalID ||
		actual.Resultado.Contexto.Instantanea.CuentaRef != l.canal.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		actual.Resultado.Contexto.PersonaRef != l.canal.contexto.Resultado.Contexto.PersonaRef {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, nil
}

// ConsultarFirmas usa la misma fachada CT152 y su consumidor AD3-125 que la
// consulta nominal. Nunca recurre al perfil de escritura o al de RRHH.
func (l *lectorFirmasIntervencionCTDesarrollo) ConsultarFirmas(
	ctx context.Context, organizacion, expediente string,
) ([]ports.FirmaRegistrada, error) {
	if l == nil || l.firma == nil || l.firma.alta == nil ||
		l.autorizador == nil || dependenciaEsNulaContratacionTemporalDesarrollo(l.firma.lector) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if !l.capacidadValida(ctx) {
		return nil, ports.ErrAutorizacionDenegada
	}
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacion, ExpedienteRef: expediente}
	recurso, err := consultafirmas.RecursoConsultaFirmasDocumento(m)
	if err != nil || organizacion != organizacionAltaContratacionTemporalDesarrollo {
		return nil, ports.ErrAutorizacionDenegada
	}
	operativo, err := l.contextoOperativo(ctx)
	if err != nil {
		return nil, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	motivo := motivoConsultaFirmasDocumentoCTDesarrollo()
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivo,
		Accion: ports.AccionConsultarFirmasDocumento, Recurso: recurso,
		Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion,
	}
	ctx = context.WithValue(ctx, claveConsultaFirmasDocumentoCTDesarrollo{}, m)
	if !solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx, datos) {
		return nil, ports.ErrAutorizacionDenegada
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return nil, ports.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := l.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
			return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return nil, ports.ErrAutorizacionDenegada
	}
	proveedor := l.firma.alta.postgresql.proveedorMaterialConsultaFirmasDocumento
	if proveedor == nil {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	material, err := proveedor.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, motivo, operativo.Resultado)
	if err != nil {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	capacidad := ports.TransportarMaterialConsultaFirmasDocumento(material)
	r := material.ResumenCapacidad()
	ahora := l.canal.reloj.Ahora()
	if consultafirmas.ValidarCapacidadConsultaFirmasDocumento(capacidad, m) != nil ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return nil, ports.ErrAutorizacionDenegada
	}
	firmas, err := l.firma.lector.ConsultarFirmasAutorizadas(ctx, m, capacidad)
	if err != nil {
		if errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
			return nil, ports.ErrAutorizacionDenegada
		}
		return nil, err
	}
	return firmas, nil
}

func (l *lectorFirmasIntervencionCTDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context, principalID, perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	if !l.capacidadValida(ctx) || l.perfil == nil || l.puente == nil ||
		principalID != l.canal.principalID || perfilRef != l.perfil.perfilRef() ||
		!l.solicitudValida(ctx) {
		return dominiovec.InstantaneaAutorizacion{}, dominiovec.ErrAutorizacionDenegada
	}
	i, estado := l.puente.consumirPerfilFijoCTDesarrolloConEstado(ctx, l.perfil)
	if estado == perfilFijoConsumoFuenteNoDisponible {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	if estado != perfilFijoConsumoVigente || i.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, dominiovec.ErrAutorizacionDenegada
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(i), nil
}

func (l *lectorFirmasIntervencionCTDesarrollo) solicitudValida(ctx context.Context) bool {
	if ctx == nil || l == nil {
		return false
	}
	d, ok := ctx.Value(claveSolicitudAutorizacionContratacionTemporalDesarrollo{}).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
	return ok && solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx, d)
}

func (l *lectorFirmasIntervencionCTDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context, referencia dominiovec.ReferenciaEntradaCatalogo, instante time.Time,
) error {
	if !l.capacidadValida(ctx) || !l.solicitudValida(ctx) ||
		referencia != motivoConsultaFirmasDocumentoCTDesarrollo() ||
		!ctdomain.InstanteUTCCanonico(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func (l *lectorFirmasIntervencionCTDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if !l.capacidadValida(ctx) || !l.solicitudValida(ctx) || l.canal.registroDecisiones == nil {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	datos, err := orden.Datos()
	if err != nil || datos.ReferenciaMotivo != motivoConsultaFirmasDocumentoCTDesarrollo() ||
		datos.Decision.ValidarPara(datos.Solicitud) != nil || datos.ResultadoContexto.Validar() != nil ||
		datos.ResultadoContexto.Contexto.PerfilActivoRef != l.perfil.perfilRef() {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	return l.canal.registroDecisiones.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (l *lectorFirmasIntervencionCTDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if !l.capacidadValida(ctx) || !l.solicitudValida(ctx) || l.canal.registroDecisiones == nil {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	datos, err := orden.Datos()
	if err != nil || datos.ReferenciaMotivo != motivoConsultaFirmasDocumentoCTDesarrollo() {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return l.canal.registroDecisiones.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}
