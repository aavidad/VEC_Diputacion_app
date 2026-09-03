package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	contratacioncomposicion "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	organizacionAltaContratacionTemporalDesarrollo = "organizacion:desarrollo:dipgra"
	centroAltaContratacionTemporalDesarrollo       = "centro:desarrollo:001"
	categoriaAltaContratacionTemporalDesarrollo    = "categoria:desarrollo:c2"
	motivoAltaContratacionTemporalDesarrollo       = domain.ClaveCatalogo("sustitucion")
)

var errAltaContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: alta efimera de desarrollo no disponible",
)

type relojContratacionTemporalDesarrollo struct{}

func (relojContratacionTemporalDesarrollo) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// soporteAltaContratacionTemporalDesarrollo simula las fuentes corporativas
// solo para ejercitar los casos de uso reales. Todo su estado es efimero,
// no_autoritativo y queda aislado por la composicion de doble llave.
type soporteAltaContratacionTemporalDesarrollo struct {
	mu                sync.Mutex
	sello             *selloConsultasContratacionTemporalDesarrollo
	principalID       string
	certificadoSHA256 string
	contexto          ports.ContextoAutorizacionAltaV3
	flujo             ports.ConfiguracionAltaFlujo
	motivo            dominiovec.ReferenciaEntradaCatalogo
	instantanea       dominiovec.InstantaneaAutorizacion
	ambitos           ports.SelladorAmbitoIdempotencia
	reloj             relojContratacionTemporalDesarrollo
	concesiones       map[string]struct{}
}

func nuevaRutaAltaContratacionTemporalDesarrollo(
	cfg config.Config,
	identidad *resolvedorIdentidadDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	sello *selloConsultasContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (vechttp.RutaExacta, func(), error) {
	if identidad == nil || derivador == nil || !derivador.valido() || sello == nil ||
		!principalContratacionTemporalDesarrolloValido(identidad.principal) {
		return vechttp.RutaExacta{}, nil, ErrActivacionDesarrolloInvalida
	}
	ahora := reloj.Ahora()
	contexto, err := nuevoContextoAltaContratacionTemporalDesarrollo(
		identidad.principal, ahora,
	)
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	datosVinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	huellas, ambitos, err := nuevasCapacidadesHMACAltaContratacionTemporalDesarrollo(
		derivador,
	)
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	flujo := ports.ConfiguracionAltaFlujo{
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo",
			Version:       1,
			HuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FaseInicial:      domain.ClaveFase("solicitud"),
		UnidadInicialRef: "unidad:desarrollo:rrhh",
		AccionInicial:    domain.ClaveCatalogo("alta"),
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_", "crear-solicitud",
		),
	}
	instantanea, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		datosVinculo.PrincipalID, datosVinculo.PerfilActivoRef, ahora,
	)
	if err != nil || flujo.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vechttp.RutaExacta{}, nil, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: identidad.principal.ID,
		certificadoSHA256: identidad.principal.Attributes["certificate_sha256"],
		contexto:          contexto, flujo: flujo, motivo: motivo, instantanea: instantanea,
		ambitos: ambitos, reloj: reloj,
		concesiones: make(map[string]struct{}),
	}
	generador := seguridadvec.GeneradorReferenciasCriptograficas{}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		soporte, soporte, soporte, soporte, reloj, generador,
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico()
	candidaturas, transaccion, cerrarPostgreSQL, err :=
		nuevasDependenciasAltaPostgreSQLContratacionTemporalDesarrollo(
			cfg, derivador, soporte, reloj,
		)
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	servicio, err := application.NuevoServicioRegistroSolicitud(
		soporte, soporte, huellas, ambitos, soporte, generador,
		referencias, candidaturas,
		postgrescontratacion.NuevoDerivadorHuellaEfectoAltaCanonico(),
		autorizador, reloj, transaccion,
	)
	if err != nil {
		cerrarPostgreSQL()
		return vechttp.RutaExacta{}, nil, err
	}
	ruta, err := contratacioncomposicion.NuevaRutaAlta(soporte, servicio, reloj)
	if err != nil {
		cerrarPostgreSQL()
		return vechttp.RutaExacta{}, nil, err
	}
	return ruta, cerrarPostgreSQL, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) capacidadAltaValida(
	ctx context.Context,
) bool {
	if s == nil || ctx == nil || ctx.Err() != nil || s.sello == nil ||
		s.principalID == "" || s.certificadoSHA256 == "" {
		return false
	}
	capacidad, existe := ctx.Value(
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
	).(capacidadConsultaContratacionTemporalDesarrollo)
	return existe && capacidad.sello == s.sello &&
		capacidad.ruta == httpinterno.RutaAltaSolicitudes &&
		principalContratacionTemporalDesarrolloValido(capacidad.principal) &&
		capacidad.principal.ID == s.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.certificadoSHA256
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoCanalAlta(
	ctx context.Context,
) (application.SolicitudRegistrarExpediente, error) {
	if !s.capacidadAltaValida(ctx) {
		return application.SolicitudRegistrarExpediente{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return application.SolicitudRegistrarExpediente{}, ports.ErrAutorizacionDenegada
	}
	return application.SolicitudRegistrarExpediente{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoAutorizacionAltaV3(
	ctx context.Context,
	solicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	if !s.capacidadAltaValida(ctx) || solicitud.Validar() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil || solicitud.AutenticacionRef != vinculo.AutenticacionRef ||
		solicitud.SesionRef != vinculo.SesionRef ||
		solicitud.PerfilRef != vinculo.PerfilActivoRef {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	resultado, err := s.contexto.Resultado.Clonar()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: s.contexto.Vinculo, Resultado: resultado,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverFlujoAlta(
	ctx context.Context,
	solicitud ports.SolicitudResolverFlujo,
) (ports.ConfiguracionAltaFlujo, error) {
	if !s.capacidadAltaValida(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.CentroRef != centroAltaContratacionTemporalDesarrollo ||
		solicitud.CategoriaRef != categoriaAltaContratacionTemporalDesarrollo ||
		solicitud.MotivoClave != motivoAltaContratacionTemporalDesarrollo {
		return ports.ConfiguracionAltaFlujo{}, ports.ErrFlujoNoDisponible
	}
	return s.flujo, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverMotivoAutorizacionAltaV3(
	ctx context.Context,
	solicitud ports.SolicitudResolverMotivoAutorizacionAltaV3,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if !s.capacidadAltaValida(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.Flujo != s.flujo.Flujo ||
		solicitud.MotivoClave != motivoAltaContratacionTemporalDesarrollo {
		return dominiovec.ReferenciaEntradaCatalogo{}, ports.ErrMotivoAutorizacionNoDisponible
	}
	return s.motivo, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID string,
	perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	if !s.capacidadAltaValida(ctx) ||
		principalID != s.instantanea.AsignacionPerfil.PrincipalID ||
		perfilRef != s.instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea), nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) error {
	if !s.capacidadAltaValida(ctx) || referencia != s.motivo ||
		!domain.InstanteUTCCanonico(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context,
	orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if !s.capacidadAltaValida(ctx) || s.instantanea.Validar() != nil {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	datos, err := orden.Datos()
	if err != nil || datos.ReferenciaMotivo != s.motivo ||
		datos.ResultadoContexto.Validar() != nil ||
		datos.Decision.ValidarPara(datos.Solicitud) != nil {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	huella, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.Decision)
	desde, hasta, errVentana := datos.Decision.VentanaValidez()
	ahora := s.reloj.Ahora()
	if err != nil || errVentana != nil || ahora.Before(desde) || !ahora.Before(hasta) {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	s.mu.Lock()
	s.concesiones[huella] = struct{}{}
	s.mu.Unlock()
	return ahora, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context,
	orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if !s.capacidadAltaValida(ctx) {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	_, err := orden.Datos()
	return err
}

func nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	publicadaEn := ahora.Add(-48 * time.Hour)
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh_desarrollo", Version: 1,
		Nombre: "Tecnico RRHH de desarrollo",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion:         ports.AccionCrearSolicitud,
			ModuloID:       ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoExpediente,
			Finalidades:    []string{ports.FinalidadCrearSolicitud},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		PublicadaPor: "seguridad:desarrollo:no-autoritativa",
		PublicadaEn:  publicadaEn,
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: "asignacion-rrhh-desarrollo-no-autoritativa",
		Version:      1, PerfilActivoRef: perfilRef, PrincipalID: principalID,
		VersionRolRef: version.Referencia(),
		Estado:        dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos: []dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: []string{centroAltaContratacionTemporalDesarrollo}},
			{Clave: "categoria_ref", Valores: []string{categoriaAltaContratacionTemporalDesarrollo}},
		},
		VigenteDesde: ahora.Add(-time.Hour),
		VigenteHasta: ahora.Add(24 * time.Hour),
		EmitidaPor:   "identidad:desarrollo:no-autoritativa",
		EmitidaEn:    ahora.Add(-2 * time.Hour),
	}
	huellaPoliticas, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion,
		VersionRol:       version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: publicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaPoliticas,
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return instantanea, nil
}

func clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
	instantanea dominiovec.InstantaneaAutorizacion,
) dominiovec.InstantaneaAutorizacion {
	copia := instantanea
	copia.VersionRol.Concesiones = append(
		[]dominiovec.ConcesionRol(nil), instantanea.VersionRol.Concesiones...,
	)
	for indice := range copia.VersionRol.Concesiones {
		copia.VersionRol.Concesiones[indice].Finalidades = append(
			[]string(nil), instantanea.VersionRol.Concesiones[indice].Finalidades...,
		)
	}
	copia.AsignacionPerfil.Ambitos = append(
		[]dominiovec.AmbitoPerfil(nil), instantanea.AsignacionPerfil.Ambitos...,
	)
	for indice := range copia.AsignacionPerfil.Ambitos {
		copia.AsignacionPerfil.Ambitos[indice].Valores = append(
			[]string(nil), instantanea.AsignacionPerfil.Ambitos[indice].Valores...,
		)
	}
	return copia
}

type selladorHMACAltaContratacionTemporalDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
	indice    int
	ambito    bool
	dominio   string
}

func (s *selladorHMACAltaContratacionTemporalDesarrollo) SellarDatos(
	ctx context.Context,
	datos []byte,
) (string, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || len(datos) == 0 ||
		s.derivador == nil || !s.derivador.valido() {
		return "", seguridadcontratacion.ErrSelladoAltaNoDisponible
	}
	resultados, err := s.derivador.calcularHMAC(datos, datos)
	if err != nil {
		return "", err
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	if s.indice < 0 || s.indice >= len(resultados) {
		return "", seguridadcontratacion.ErrSelladoAltaNoDisponible
	}
	resultado := resultados[s.indice]
	valor := resultado.huellaSolicitud[:]
	if s.ambito {
		valor = resultado.localizador[:]
	}
	return fmt.Sprintf(
		"hmac-sha256:%s/v%d:%s",
		s.dominio, resultado.generacion, hex.EncodeToString(valor),
	), nil
}

func nuevasCapacidadesHMACAltaContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
) (
	ports.DerivadorHuellaAlta,
	ports.SelladorAmbitoIdempotencia,
	error,
) {
	huellaActiva, huellasRetenidas, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, "vec.contratacion-temporal.huella-peticion", false,
	)
	if err != nil {
		return nil, nil, err
	}
	ambitoActivo, ambitosRetenidos, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, "vec.contratacion-temporal.ambito-idempotencia", true,
	)
	if err != nil {
		return nil, nil, err
	}
	huellas, err := seguridadcontratacion.NuevoDerivadorHuellaAltaHMACRotable(
		huellaActiva, huellasRetenidas,
	)
	if err != nil {
		return nil, nil, err
	}
	ambitos, err := seguridadcontratacion.NuevoSelladorAmbitoIdempotenciaHMACRotable(
		ambitoActivo, ambitosRetenidos,
	)
	if err != nil {
		return nil, nil, err
	}
	return huellas, ambitos, nil
}

func configuracionesHMACAltaContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	dominio string,
	ambito bool,
) (
	seguridadcontratacion.ConfiguracionSelladorHMAC,
	[]seguridadcontratacion.ConfiguracionSelladorHMAC,
	error,
) {
	if derivador == nil || !derivador.valido() {
		return seguridadcontratacion.ConfiguracionSelladorHMAC{}, nil,
			seguridadcontratacion.ErrSelladoAltaNoDisponible
	}
	configuraciones := make(
		[]seguridadcontratacion.ConfiguracionSelladorHMAC,
		len(derivador.generaciones),
	)
	for indice, generacion := range derivador.generaciones {
		sellador := &selladorHMACAltaContratacionTemporalDesarrollo{
			derivador: derivador, indice: indice, ambito: ambito, dominio: dominio,
		}
		configuracion, err := seguridadcontratacion.NuevaConfiguracionSelladorHMAC(
			fmt.Sprintf("%s/v%d", dominio, generacion.generacion),
			sellador,
		)
		if err != nil {
			return seguridadcontratacion.ConfiguracionSelladorHMAC{}, nil, err
		}
		configuraciones[indice] = configuracion
	}
	return configuraciones[0], configuraciones[1:], nil
}

func nuevoContextoAltaContratacionTemporalDesarrollo(
	principal dominiovec.Principal,
	ahora time.Time,
) (ports.ContextoAutorizacionAltaV3, error) {
	if !principalContratacionTemporalDesarrolloValido(principal) ||
		!domain.InstanteUTCCanonico(ahora) {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	base := principal.ID + "\x00" + principal.Attributes["certificate_sha256"]
	cuentaRef := referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta")
	personaRef := referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona")
	perfilRef := referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00perfil")
	vinculoRef := referenciaAltaContratacionTemporalDesarrollo("vca_", base+"\x00vinculo")
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: cuentaRef,
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: vinculoRef, VinculoVersion: 1,
		CuentaRef: cuentaRef, CuentaVersion: 1,
		PersonaRef: personaRef, PersonaVersion: 1,
		PerfilActivoRef: perfilRef, PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour),
		VigenteHasta: ahora.Add(24 * time.Hour),
		Vinculos:     []dominiovec.VinculoReferenciaContextoActor{},
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-time.Minute))
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	huellaContexto, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	// La etiqueta de autoridad es una precondicion estructural del contrato V3;
	// las referencias que la acompañan siguen marcadas como desarrollo efimero
	// y nunca salen de la composicion protegida por doble llave y mTLS.
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: referenciaAltaContratacionTemporalDesarrollo(
			"prc_", base+"\x00procedencia",
		),
		ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00procedencia",
		),
		ProcedenciaAutoridad: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema:           dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuentaRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: personaRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: perfilRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: vinculoRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: referenciaAltaContratacionTemporalDesarrollo(
			"rca_", base+"\x00registro-contexto",
		),
		Contexto: actor, RepresentacionCanonica: canon,
		HuellaSHA256:                      huellaContexto,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:                 dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: referenciaAltaContratacionTemporalDesarrollo(
			"aut_", base+"\x00autenticacion",
		),
		AutenticacionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00autenticacion",
		),
		AsercionRef: referenciaAltaContratacionTemporalDesarrollo(
			"ase_", base+"\x00asercion",
		),
		SesionRef: referenciaAltaContratacionTemporalDesarrollo(
			"ses_", base+"\x00sesion",
		),
		ControlSesionRef: referenciaAltaContratacionTemporalDesarrollo(
			"cse_", base+"\x00control-sesion",
		),
		ControlSesionRevision: 1,
		ControlSesionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00control-sesion",
		),
		CuentaRef: cuentaRef, CuentaOrdinariaRef: cuentaRef,
		Superficie:        dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:   dominiovec.AuthMethodCertificate,
		GarantiaObservada: dominiovec.AuthAssuranceHigh,
		PoliticaGarantiaRef: referenciaAltaContratacionTemporalDesarrollo(
			"pga_", base+"\x00politica-garantia",
		),
		PoliticaGarantiaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00politica-garantia",
		),
		AutenticacionVerificadaEn: ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:           ahora.Add(-9 * time.Minute),
		SesionValidaHasta:         ahora.Add(24 * time.Hour),
		SesionRevalidadaEn:        ahora.Add(-2 * time.Minute),
	}
	solicitudAutenticacion := dominiovec.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: autenticacion.AutenticacionRef,
		SesionRef:        autenticacion.SesionRef,
	}
	solicitudContexto := dominiovec.SolicitudContextoActor{
		Cuenta: cuenta, PerfilActivoRef: perfilRef,
	}
	vinculo, resultadoClonado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		solicitudAutenticacion,
		resolutorContextoAltaContratacionTemporalDesarrollo{valor: resultado},
		solicitudContexto,
		relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora},
	)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: vinculo, Resultado: resultadoClonado,
	}, nil
}

type revalidadorAutenticacionAltaContratacionTemporalDesarrollo struct {
	valor dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionAltaContratacionTemporalDesarrollo) RevalidarAutenticacionActorV1(
	ctx context.Context,
	solicitud dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	if ctx == nil || ctx.Err() != nil ||
		solicitud.AutenticacionRef != r.valor.AutenticacionRef ||
		solicitud.SesionRef != r.valor.SesionRef {
		return dominiovec.AutenticacionRevalidadaV1{},
			dominiovec.ErrAutenticacionRevalidadaInvalida
	}
	return r.valor, nil
}

type resolutorContextoAltaContratacionTemporalDesarrollo struct {
	valor dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoAltaContratacionTemporalDesarrollo) ResolverContextoActorRegistradoV2(
	ctx context.Context,
	solicitud dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil ||
		solicitud.Cuenta.CuentaRef != r.valor.Contexto.Instantanea.CuentaRef ||
		solicitud.PerfilActivoRef != r.valor.Contexto.PerfilActivoRef {
		return dominiovec.ResultadoContextoActorRegistradoV2{},
			dominiovec.ErrVinculoAutenticacionActorV2Invalido
	}
	return r.valor.Clonar()
}

type relojFijoAltaContratacionTemporalDesarrollo struct {
	ahora time.Time
}

func (r relojFijoAltaContratacionTemporalDesarrollo) Ahora() time.Time {
	return r.ahora
}

func referenciaAltaContratacionTemporalDesarrollo(prefijo, material string) string {
	suma := sha256.Sum256([]byte("vec.ct.alta.desarrollo.v1\x00" + material))
	return prefijo + hex.EncodeToString(suma[:16])
}

func huellaAltaContratacionTemporalDesarrollo(material string) string {
	suma := sha256.Sum256([]byte("vec.ct.alta.desarrollo.v1\x00" + material))
	return hex.EncodeToString(suma[:])
}
