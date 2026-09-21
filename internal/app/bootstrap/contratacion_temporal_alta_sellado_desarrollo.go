package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

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
	if !principalContratacionTemporalDesarrolloValido(principal) {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
}

func nuevoContextoSinteticoContratacionTemporalDesarrollo(
	principal dominiovec.Principal,
	ahora time.Time,
) (ports.ContextoAutorizacionAltaV3, error) {
	return nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoSinteticoContratacionTemporalDesarrollo(),
	)
}

// discriminadorContextoSinteticoDesarrollo nunca llega desde un transporte ni
// una configuración abierta. Se limita a separar contextos sintéticos que
// comparten la misma cuenta y persona dentro del ensamblaje de desarrollo.
type discriminadorContextoSinteticoDesarrollo struct {
	perfil, vinculo, procedencia, registro string
	autenticacion, asercion, sesion        string
	controlSesion, politicaGarantia        string
}

func discriminadorContextoSinteticoContratacionTemporalDesarrollo() discriminadorContextoSinteticoDesarrollo {
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil", vinculo: "vinculo", procedencia: "procedencia", registro: "registro-contexto",
		autenticacion: "autenticacion", asercion: "asercion", sesion: "sesion",
		controlSesion: "control-sesion", politicaGarantia: "politica-garantia",
	}
}

func (d discriminadorContextoSinteticoDesarrollo) valido() bool {
	return d.perfil != "" && d.vinculo != "" && d.procedencia != "" && d.registro != "" &&
		d.autenticacion != "" && d.asercion != "" && d.sesion != "" &&
		d.controlSesion != "" && d.politicaGarantia != ""
}

func nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
	principal dominiovec.Principal,
	ahora time.Time,
	discriminador discriminadorContextoSinteticoDesarrollo,
) (ports.ContextoAutorizacionAltaV3, error) {
	if !principalSinteticoContratacionTemporalDesarrolloValido(principal) ||
		!domain.InstanteUTCCanonico(ahora) || !discriminador.valido() {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	resueltoEn := desde.Add(3 * time.Minute)
	base := principal.ID + "\x00" + principal.Attributes["certificate_sha256"]
	cuentaRef := referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta")
	personaRef := referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona")
	perfilRef := referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00"+discriminador.perfil)
	vinculoRef := referenciaAltaContratacionTemporalDesarrollo("vca_", base+"\x00"+discriminador.vinculo)
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
		VigenteDesde: desde,
		VigenteHasta: hasta,
		Vinculos:     []dominiovec.VinculoReferenciaContextoActor{},
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, resueltoEn)
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
			"prc_", base+"\x00"+discriminador.procedencia,
		),
		ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00" + discriminador.procedencia,
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
			"rca_", base+"\x00"+discriminador.registro,
		),
		Contexto: actor, RepresentacionCanonica: canon,
		HuellaSHA256:                      huellaContexto,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:                 dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            resueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: referenciaAltaContratacionTemporalDesarrollo(
			"aut_", base+"\x00"+discriminador.autenticacion,
		),
		AutenticacionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00" + discriminador.autenticacion,
		),
		AsercionRef: referenciaAltaContratacionTemporalDesarrollo(
			"ase_", base+"\x00"+discriminador.asercion,
		),
		SesionRef: referenciaAltaContratacionTemporalDesarrollo(
			"ses_", base+"\x00"+discriminador.sesion,
		),
		ControlSesionRef: referenciaAltaContratacionTemporalDesarrollo(
			"cse_", base+"\x00"+discriminador.controlSesion,
		),
		ControlSesionRevision: 1,
		ControlSesionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00" + discriminador.controlSesion,
		),
		CuentaRef: cuentaRef, CuentaOrdinariaRef: cuentaRef,
		Superficie:        dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:   dominiovec.AuthMethodCertificate,
		GarantiaObservada: dominiovec.AuthAssuranceHigh,
		PoliticaGarantiaRef: referenciaAltaContratacionTemporalDesarrollo(
			"pga_", base+"\x00"+discriminador.politicaGarantia,
		),
		PoliticaGarantiaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00" + discriminador.politicaGarantia,
		),
		AutenticacionVerificadaEn: desde,
		SesionEmitidaEn:           desde.Add(time.Minute),
		SesionValidaHasta:         hasta,
		SesionRevalidadaEn:        desde.Add(2 * time.Minute),
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
