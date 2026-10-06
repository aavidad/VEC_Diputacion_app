package bootstrap

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Consulta (44 campos) y recuperación (48 campos) del registro nominal de
// firmas V2 con la persona RRHH autenticada por mTLS. La fuente nominal es la
// misma de las demás rutas CT: sesión registrada y revalidada, contexto de
// actor registrado y perfil fijo propio consumido tal como está publicado.
// Ningún dato de la petición elige actor, perfil, organización ni candidato.
// Solo se compone con VEC_CT_FIRMAS_R5_V2_ENABLED=true y la doble llave de
// desarrollo; sin él las rutas no existen.
const (
	envCTFirmasR5V2Enabled                = "VEC_CT_FIRMAS_R5_V2_ENABLED"
	clavePerfilFijoFirmasR5V2CTDesarrollo = "firmas_r5_v2"
	canalIntentosFirmasR5V2CTDesarrollo   = string(dominiovec.SuperficieAutenticacionInternaCorporativaV1)
)

var errFirmasR5V2CTDesarrolloNoDisponible = errors.New("contratacion temporal: consulta y recuperacion de firmas V2 no disponibles")

func rutaFirmasR5V2CTDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaConsultaFirmasR5V2 || ruta == httpinterno.RutaRecuperacionFirmasR5V2
}

// accionFirmasR5V2CTDesarrollo liga cada ruta con su única acción: la ruta de
// consulta nunca obtiene una decisión de recuperación ni a la inversa.
func accionFirmasR5V2CTDesarrollo(ruta string) (string, bool) {
	switch ruta {
	case httpinterno.RutaConsultaFirmasR5V2:
		return ports.AccionConsultarFirmasR5V2, true
	case httpinterno.RutaRecuperacionFirmasR5V2:
		return ports.AccionRecuperarFirmasR5V2, true
	}
	return "", false
}

func motivoFirmasR5V2CTDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_firmas_r5_v2_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("firmas-r5-v2-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "firmas-r5-v2-ct"),
	}
}

func descriptoresMaterialFirmasR5V2CTDesarrollo() (consulta, recuperacion descriptorMaterialConsumidorV3Desarrollo) {
	return descriptorMaterialConsumidorV3Desarrollo{
			Audiencia: ports.AudienciaConsultaFirmasR5V2, Dominio: "vec.ct.firmas-r5-v2.consulta.capacidad-v3",
			Prefijo: "clave:capacidad:ct-firmas-r5-v2-consulta:", ProveedorNominal: proveedorMaterialContratacionTemporal,
		}, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia: ports.AudienciaRecuperacionFirmasR5V2, Dominio: "vec.ct.firmas-r5-v2.recuperacion.capacidad-v3",
			Prefijo: "clave:capacidad:ct-firmas-r5-v2-recuperacion:", ProveedorNominal: proveedorMaterialContratacionTemporal,
		}
}

// Las dos concesiones son de lectura y llevan exactamente los campos que
// exigen AD162 (consulta) y AD178 (recuperación), sin obligaciones.
func nuevaInstantaneaFirmasR5V2CTDesarrollo(principalID, perfilRef string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	concesion := func(accion string, campos []string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{Accion: accion, ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoConsultaFirmasR5, Finalidades: []string{ports.FinalidadFirmaDocumento},
			GarantiaMinima: dominiovec.AuthAssuranceHigh, CamposPermitidos: campos}
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		"firmas_r5_v2_lector_ct_desarrollo", "Consulta y recuperación del registro de firmas V2 de desarrollo",
		"firmas-r5-v2-lector-ct-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			concesion(ports.AccionConsultarFirmasR5V2, ports.CamposConsultaFirmasR5V2()),
			concesion(ports.AccionRecuperarFirmasR5V2, ports.CamposRecuperacionFirmasV2()),
		},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

// El PDP sólo admite la acción de la ruta sellada, el motivo propio y un
// recurso de expediente con la organización del perfil y la huella del
// material. El emisor V2 ya liga ese recurso al material exacto, y CT172 o
// CT175 recalculan la huella en la transacción del consumo.
func solicitudAutorizacionFirmasR5V2CTDesarrolloValida(ruta string, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	accion, ok := accionFirmasR5V2CTDesarrollo(ruta)
	r := datos.Recurso
	return ok && datos.Accion == accion && datos.Finalidad == ports.FinalidadFirmaDocumento &&
		datos.ReferenciaMotivo == motivoFirmasR5V2CTDesarrollo() && r.Validar() == nil &&
		ctdomain.ReferenciaOpacaValida(r.Referencia) && r.ModuloID == ports.ModuloContratacion &&
		r.Tipo == ports.TipoRecursoConsultaFirmasR5 && len(r.Ambitos) == 1 &&
		r.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		len(r.Atributos) == 1 && ctdomain.HuellaSHA256FirmaValida(r.Atributos["material_sha256"])
}

// fuenteNominalFirmasR5V2CTDesarrollo es la fuente nominal de sesión, perfil
// y certificado de las decisiones V3 de firma V2 (punto 3 del plan de firmas).
// Se revalida en cada llamada; el soporte cachea la sesión por petición.
type fuenteNominalFirmasR5V2CTDesarrollo struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
	perfil  *perfilFijoCTDesarrollo
	reloj   relojContratacionTemporalDesarrollo
	proceso string
	// rutas de este perfil y su acción auditada, y el motivo de sus
	// decisiones: los de R5 V2 o los de la firma V2 externa.
	accionDeRuta func(string) (string, bool)
	motivo       dominiovec.ReferenciaEntradaCatalogo
	// consultaPrevia: la ruta consulta las firmas R5 V2 antes de su acción
	// (vía externa), y esa consulta también se audita.
	consultaPrevia bool
}

// nuevaFuenteNominalFirmasR5V2CTDesarrollo: la fuente de las rutas de consulta
// y recuperación R5 V2.
func nuevaFuenteNominalFirmasR5V2CTDesarrollo(s *soporteAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo,
	reloj relojContratacionTemporalDesarrollo, proceso string) *fuenteNominalFirmasR5V2CTDesarrollo {
	return &fuenteNominalFirmasR5V2CTDesarrollo{soporte: s, perfil: p, reloj: reloj, proceso: proceso,
		accionDeRuta: accionFirmasR5V2CTDesarrollo, motivo: motivoFirmasR5V2CTDesarrollo()}
}

var (
	_ firmaemisorv2.FuenteContextoActorFirmaV2 = (*fuenteNominalFirmasR5V2CTDesarrollo)(nil)
	_ consultafirmasv2.FuenteContexto          = (*fuenteNominalFirmasR5V2CTDesarrollo)(nil)
)

// canal acredita la frontera mTLS sellada de una de las dos rutas, su método
// y el certificado vigente. No consulta la base.
func (f *fuenteNominalFirmasR5V2CTDesarrollo) canal(ctx context.Context) (capacidadConsultaContratacionTemporalDesarrollo, bool) {
	if f == nil || f.soporte == nil || f.perfil == nil || f.accionDeRuta == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return capacidadConsultaContratacionTemporalDesarrollo{}, false
	}
	capacidad, valida := f.soporte.capacidadValida(ctx)
	_, rutaPropia := f.accionDeRuta(capacidad.ruta)
	return capacidad, valida && rutaPropia && capacidad.metodo == http.MethodPost &&
		f.soporte.perfilFijoParaContexto(ctx, capacidad.ruta) == f.perfil &&
		certificadoConsultaReciboRespuestaVigente(capacidad, f.reloj.Ahora())
}

// operativo devuelve la sesión registrada del perfil fijo de esta petición.
// Una caída de la sesión o de la base es indisponibilidad, nunca permiso.
func (f *fuenteNominalFirmasR5V2CTDesarrollo) operativo(ctx context.Context) (ports.ContextoAutorizacionAltaV3, capacidadConsultaContratacionTemporalDesarrollo, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	capacidad, valida := f.canal(ctx)
	if !valida {
		if ctx != nil && ctx.Err() != nil {
			return vacio, capacidad, ctx.Err()
		}
		return vacio, capacidad, ports.ErrFirmaDocumentoDenegada
	}
	operativo, err := f.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, capacidad, ctx.Err()
		}
		if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
			return vacio, capacidad, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return vacio, capacidad, ports.ErrFirmaDocumentoDenegada
	}
	if !contextoRegistradoPerfilFijoCTDesarrollo(operativo.Resultado.Contexto, f.perfil) {
		return vacio, capacidad, ports.ErrFirmaDocumentoDenegada
	}
	return operativo, capacidad, nil
}

// contextoRegistradoPerfilFijoCTDesarrollo exige que la sesión revalidada sea la
// del perfil fijo: mismo perfil, persona, principal y cuenta registrados. El
// identificador del certificado mTLS es otro (lo coteja capacidadValida) y no
// se compara con la persona registrada.
func contextoRegistradoPerfilFijoCTDesarrollo(c dominiovec.ContextoActor, perfil *perfilFijoCTDesarrollo) bool {
	if perfil == nil {
		return false
	}
	propio := perfil.contexto.Resultado.Contexto
	return c.PerfilActivoRef != "" && c.PerfilActivoRef == perfil.perfilRef() &&
		c.PersonaRef != "" && c.PersonaRef == propio.PersonaRef &&
		c.Principal.ID != "" && c.Principal.ID == propio.Principal.ID &&
		c.Instantanea.CuentaRef == propio.Instantanea.CuentaRef
}

// contexto exige además que la asignación publicada del perfil siga siendo
// exactamente la plantilla: una revocación cierra la ruta antes de emitir.
func (f *fuenteNominalFirmasR5V2CTDesarrollo) contexto(ctx context.Context) (ports.ContextoAutorizacionAltaV3, capacidadConsultaContratacionTemporalDesarrollo, error) {
	operativo, capacidad, err := f.operativo(ctx)
	if err != nil {
		return operativo, capacidad, err
	}
	_, estado := f.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, f.perfil)
	switch {
	case ctx.Err() != nil:
		return ports.ContextoAutorizacionAltaV3{}, capacidad, ctx.Err()
	case estado == perfilFijoConsumoFuenteNoDisponible:
		return ports.ContextoAutorizacionAltaV3{}, capacidad, ports.ErrRegistroFirmaDocumentoNoDisponible
	case estado != perfilFijoConsumoVigente:
		return ports.ContextoAutorizacionAltaV3{}, capacidad, ports.ErrFirmaDocumentoDenegada
	}
	return operativo, capacidad, nil
}

// RevalidarContextoActorFirmaV2 entrega al emisor V2 la sesión, el contexto
// registrado y la huella del certificado verificado en este canal mTLS.
func (f *fuenteNominalFirmasR5V2CTDesarrollo) RevalidarContextoActorFirmaV2(ctx context.Context) (firmaemisorv2.ContextoActorFirmaV2, error) {
	operativo, capacidad, err := f.contexto(ctx)
	if err != nil {
		return firmaemisorv2.ContextoActorFirmaV2{}, err
	}
	resultado, err := operativo.Resultado.Clonar()
	if err != nil {
		return firmaemisorv2.ContextoActorFirmaV2{}, ports.ErrFirmaDocumentoDenegada
	}
	return firmaemisorv2.ContextoActorFirmaV2{Resultado: resultado, Vinculo: operativo.Vinculo,
		CertificadoCanalSHA256: capacidad.principal.Attributes["certificate_sha256"]}, nil
}

// El candidato es la persona del contexto registrado y revalidado: quien
// consulta antes de firmar con su propio certificado. La organización es el
// ámbito único del perfil. La petición no aporta ninguno de los dos. Vale sólo
// para estas lecturas: al preparar una firma el candidato sale de la evidencia
// de competencia, no de esta fuente.
func (f *fuenteNominalFirmasR5V2CTDesarrollo) ResolverContextoConsultaFirmasR5V2(ctx context.Context) (consultafirmasv2.Contexto, error) {
	operativo, _, err := f.contexto(ctx)
	if err != nil {
		return consultafirmasv2.Contexto{}, err
	}
	return consultafirmasv2.Contexto{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		FirmantePrincipalCandidatoRef: operativo.Resultado.Contexto.PersonaRef}, nil
}

// CrearOrdenIntentoFirma audita denegaciones y errores con la identidad de la
// sesión registrada. No exige perfil consumible: una revocación también se
// audita. Sin sesión registrada no hay orden y el manejador responde 503.
func (f *fuenteNominalFirmasR5V2CTDesarrollo) CrearOrdenIntentoFirma(ctx context.Context, intento, accion, recurso string,
	resultado dominiovec.ResultadoIntentoAuditoria,
) (puertosvec.OrdenIntentoAuditoria, error) {
	var cero puertosvec.OrdenIntentoAuditoria
	esperada, ok := "", false
	if capacidad, valida := f.canal(ctx); valida {
		esperada, ok = f.accionDeRuta(capacidad.ruta)
	}
	if !ok || (accion != esperada && !(f.consultaPrevia && accion == ports.AccionConsultarFirmasR5V2)) || f.proceso == "" {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	operativo, _, err := f.operativo(ctx)
	if err != nil {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if recurso == "" {
		recurso = organizacionAltaContratacionTemporalDesarrollo
	}
	referencia, err := puertosvec.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	correlacion, err := referencia.ValorCanonico()
	if err != nil {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return puertosvec.NuevaOrdenIntentoAuditoria(intento, operativo.Resultado, operativo.Vinculo, dominiovec.DatosIntentoAuditoria{
		Accion: accion, ModuloID: ports.ModuloContratacion, RecursoRef: recurso, FinalidadRef: ports.FinalidadFirmaDocumento,
		Resultado: resultado, Motivo: f.motivo, Proceso: f.proceso,
		Canal: canalIntentosFirmasR5V2CTDesarrollo, CorrelacionRef: correlacion,
	})
}

// emisorFirmasR5V2CTDesarrollo elige el emisor común de la audiencia de cada
// acción y entrega al PDP CT la solicitud exacta que va a evaluar. Una acción
// sin emisor se deniega.
type emisorFirmasR5V2CTDesarrollo struct {
	porAccion map[string]*emisorMaterialRenovableCTDesarrollo
}

func (e *emisorFirmasR5V2CTDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3,
	c dominiovec.ResultadoContextoActorRegistradoV2,
) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := s.Datos()
	var emisor *emisorMaterialRenovableCTDesarrollo
	if e != nil && err == nil && ctx != nil {
		emisor = e.porAccion[datos.Accion]
	}
	if emisor == nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ports.ErrFirmaDocumentoDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
}

// rutasFirmasR5V2CTDesarrollo agrupa lo que la raíz registra y cierra.
type rutasFirmasR5V2CTDesarrollo struct {
	rutas  []vechttp.RutaExacta
	cerrar func()
}

// nuevasRutasFirmasR5V2CTDesarrollo compone las dos rutas. Devuelve nil sin
// error con el selector apagado. Publica el catálogo de motivos, el perfil
// fijo (sólo si falta) y las claves de capacidad de sus dos audiencias.
func nuevasRutasFirmasR5V2CTDesarrollo(cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) (*rutasFirmasR5V2CTDesarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envCTFirmasR5V2Enabled)
	if err != nil || !activo {
		return nil, err
	}
	if alta == nil || alta.soporte == nil || alta.autorizador == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil ||
		alta.postgresql.proveedorMaterial == nil || derivador == nil || !derivador.valido() {
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	s := alta.soporte
	ahora := reloj.Ahora()
	principal := dominiovec.Principal{
		ID: s.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": s.certificadoSHA256,
		},
	}
	fijo, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmasR5V2CTDesarrollo,
		[]string{httpinterno.RutaConsultaFirmasR5V2, httpinterno.RutaRecuperacionFirmasR5V2},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaFirmasR5V2CTDesarrollo(principalID, perfilRef, ahora)
		})
	if err != nil {
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	fijo.metodo = http.MethodPost
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{motivoFirmasR5V2CTDesarrollo()}, desde) != nil {
		log.Print("contratacion temporal: firmas V2 no disponibles; etapa=catalogo_motivos")
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	if err := s.registrarPerfilFijoCTDesarrollo(fijo); err != nil {
		return nil, err
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, alta.postgresql.gobierno, s,
		aprobacionProvisionPerfilesRRHHDesdeConfig(cfg), fijo); err != nil {
		log.Print("contratacion temporal: firmas V2 no disponibles; etapa=perfil_fijo")
		return nil, err
	}
	dConsulta, dRecuperacion := descriptoresMaterialFirmasR5V2CTDesarrollo()
	emisores := make([]*emisorMaterialRenovableCTDesarrollo, 0, 2)
	for _, d := range []descriptorMaterialConsumidorV3Desarrollo{dConsulta, dRecuperacion} {
		material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
		if err != nil {
			return nil, errFirmasR5V2CTDesarrolloNoDisponible
		}
		material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
		proveedor, err := nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, alta.postgresql.gobierno, material, s, reloj, d)
		material.borrarCopiasEfimeras()
		if err != nil {
			log.Print("contratacion temporal: firmas V2 no disponibles; etapa=material_" + d.Audiencia)
			return nil, errFirmasR5V2CTDesarrolloNoDisponible
		}
		emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(alta.autorizador, proveedor)
		if err != nil {
			return nil, errFirmasR5V2CTDesarrolloNoDisponible
		}
		emisores = append(emisores, emisor)
	}
	reservados := []string{alta.postgresql.gobierno.Config().ConnConfig.User,
		alta.postgresql.registroAutorizacion.Config().ConnConfig.User, alta.postgresql.ejecucion.Config().ConnConfig.User}
	intentos, proceso, cerrar, err := AbrirRegistradorIntentosAuditoriaDesarrollo(ctx, cfg, alta.postgresql.ejecucion, reservados)
	if err != nil {
		log.Print("contratacion temporal: firmas V2 no disponibles; etapa=registrador_intentos")
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	compuestas := false
	defer func() {
		if !compuestas {
			cerrar()
		}
	}()
	fuente := nuevaFuenteNominalFirmasR5V2CTDesarrollo(s, fijo, reloj, proceso)
	emisor, err := firmaemisorv2.NuevoEmisor(fuente, &emisorFirmasR5V2CTDesarrollo{porAccion: map[string]*emisorMaterialRenovableCTDesarrollo{
		ports.AccionConsultarFirmasR5V2: emisores[0], ports.AccionRecuperarFirmasR5V2: emisores[1]}},
		motivoFirmasR5V2CTDesarrollo(), reloj)
	if err != nil {
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	registro, err := postgresct.NuevoRegistroFirmasVerificadasPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	consulta, err := httpinterno.NuevoManejadorConsultaFirmasR5V2(fuente, emisor, registro, intentos, fuente)
	if err != nil {
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	recuperacion, err := httpinterno.NuevoManejadorRecuperacionFirmasR5V2(fuente, emisor, registro, intentos, fuente)
	if err != nil {
		return nil, errFirmasR5V2CTDesarrolloNoDisponible
	}
	compuestas = true
	return &rutasFirmasR5V2CTDesarrollo{cerrar: cerrar, rutas: []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaConsultaFirmasR5V2, Manejador: consulta},
		{Ruta: httpinterno.RutaRecuperacionFirmasR5V2, Manejador: recuperacion},
	}}, nil
}
