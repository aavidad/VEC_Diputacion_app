package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"log"
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// La semilla publicada por el preparador solo fija las coordenadas de la
// identidad. El esperado operativo se lee de la autoridad registrada antes
// de atender peticiones y conserva también vínculos, versiones y procedencia.
func contextoEsperadoRegistradoDesarrollo(
	ctx context.Context, resolutor dominiovec.ResolutorContextoActorRegistradoV2,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	if soporte == nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, ports.ErrConsultaRRHHNoDisponible
	}
	return contextoEsperadoRegistradoParaSemillaDesarrollo(ctx, resolutor, soporte, soporte.contexto.Resultado)
}

func contextoEsperadoRegistradoParaSemillaDesarrollo(
	ctx context.Context, resolutor dominiovec.ResolutorContextoActorRegistradoV2,
	soporte *soporteAltaContratacionTemporalDesarrollo, semilla dominiovec.ResultadoContextoActorRegistradoV2,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	fallo := ports.ErrConsultaRRHHNoDisponible
	if ctx == nil || ctx.Err() != nil || resolutor == nil || soporte == nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	if err := semilla.Validar(); err != nil {
		log.Print("contratacion temporal: contexto esperado no disponible; etapa=validar_semilla_contexto")
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	if soporte.principalID == "" || !huellaSHA256ValidaContratacionTemporalDesarrollo(soporte.certificadoSHA256) {
		log.Print("contratacion temporal: contexto esperado no disponible; etapa=validar_soporte_contexto")
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	actor := semilla.Contexto
	registrado, err := resolutor.ResolverContextoActorRegistradoV2(ctx, dominiovec.SolicitudContextoActor{
		Cuenta: dominiovec.CuentaAutenticadaContextoActor{
			CuentaRef: actor.Instantanea.CuentaRef,
			Metodo:    dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh,
		},
		PerfilActivoRef: actor.PerfilActivoRef,
	})
	if err != nil {
		log.Print("contratacion temporal: contexto esperado no disponible; etapa=resolver_contexto_registrado")
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	if err := registrado.Validar(); err != nil {
		log.Print("contratacion temporal: contexto esperado no disponible; etapa=validar_contexto_registrado")
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	if registrado.AutoridadEfectiva != dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		registrado.Contexto.Principal.AuthMethod != dominiovec.AuthMethodCertificate ||
		registrado.Contexto.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh ||
		registrado.Contexto.Instantanea.CuentaRef != actor.Instantanea.CuentaRef ||
		registrado.Contexto.Instantanea.PersonaRef != actor.Instantanea.PersonaRef ||
		registrado.Contexto.Instantanea.PerfilActivoRef != actor.Instantanea.PerfilActivoRef ||
		registrado.Contexto.Instantanea.VinculoRef != actor.Instantanea.VinculoRef ||
		registrado.Contexto.PersonaRef != actor.PersonaRef ||
		registrado.Contexto.PerfilActivoRef != actor.PerfilActivoRef ||
		registrado.Contexto.Principal.ID != actor.Principal.ID ||
		!registrado.Contexto.Instantanea.VigenteEn(registrado.ResueltoEnAutoritativo) {
		log.Print("contratacion temporal: contexto esperado no disponible; etapa=identidad_contexto_registrado")
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	for _, vinculo := range registrado.Contexto.Instantanea.Vinculos {
		if !vinculo.VigenteEn(registrado.ResueltoEnAutoritativo) {
			log.Print("contratacion temporal: contexto esperado no disponible; etapa=vigencia_vinculo_registrado")
			return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
		}
	}
	return registrado.Clonar()
}

// Un recibo fresco tiene otra referencia e instante. Todo lo demás debe ser
// idéntico a la autoridad fijada al componer la instancia.
func mismoContextoEsperadoRegistradoDesarrollo(
	esperado, actual dominiovec.ResultadoContextoActorRegistradoV2,
) bool {
	if esperado.Validar() != nil || actual.Validar() != nil ||
		esperado.AutoridadEfectiva != actual.AutoridadEfectiva ||
		!bytes.Equal(esperado.ManifiestoProcedenciaCanonico, actual.ManifiestoProcedenciaCanonico) ||
		esperado.ManifiestoProcedenciaHuellaSHA256 != actual.ManifiestoProcedenciaHuellaSHA256 {
		return false
	}
	actor, err := actual.Contexto.Clonar()
	if err != nil {
		return false
	}
	actor.ResueltoEn = esperado.Contexto.ResueltoEn
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	return err == nil && bytes.Equal(canon, esperado.RepresentacionCanonica)
}

type contextoOperacionCTDesarrollo struct {
	mu       sync.Mutex
	soporte  *soporteAltaContratacionTemporalDesarrollo
	contexto ports.ContextoAutorizacionAltaV3
	err      error
}

type proveedorSesionOperativaCTDesarrollo interface {
	ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error)
}

// Comparte las autoridades de sesión y contexto, pero conserva base, clave
// efímera y perfil CT130 propios. El proveedor exige la misma capacidad mTLS.
func nuevaSesionReincorporacionTitularDesarrollo(
	base *proveedorSesionConsultaRRHHDesarrollo,
	esperado dominiovec.ResultadoContextoActorRegistradoV2,
) (proveedorSesionOperativaCTDesarrollo, error) {
	return nuevaSesionPerfilAdicionalCTDesarrollo(base, esperado)
}

func nuevaSesionPerfilAdicionalCTDesarrollo(
	base *proveedorSesionConsultaRRHHDesarrollo,
	esperado dominiovec.ResultadoContextoActorRegistradoV2,
) (proveedorSesionOperativaCTDesarrollo, error) {
	if base == nil || base.soporte == nil || esperado.Validar() != nil ||
		esperado.Contexto.PerfilActivoRef == base.base.Contexto.PerfilActivoRef ||
		esperado.Contexto.Principal.ID != base.base.Contexto.Principal.ID ||
		esperado.Contexto.Instantanea.CuentaRef != base.base.Contexto.Instantanea.CuentaRef ||
		esperado.Contexto.PersonaRef != base.base.Contexto.PersonaRef {
		return nil, ports.ErrAutorizacionDenegada
	}
	clon, err := esperado.Clonar()
	if err != nil {
		return nil, ports.ErrAutorizacionDenegada
	}
	proveedor := &proveedorSesionConsultaRRHHDesarrollo{
		soporte: base.soporte, registro: base.registro, revalidador: base.revalidador,
		reloj: base.reloj, resolutor: base.resolutor, base: clon,
		fronteras: base.fronteras, superficie: base.superficie,
	}
	if _, err := rand.Read(proveedor.clave[:]); err != nil {
		return nil, ports.ErrAutorizacionDenegada
	}
	return proveedor, nil
}

func rutaReincorporacionTitularDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaReincorporacionesTitular || ruta == httpinterno.RutaCapacidadReincorporacionTitular
}

func rutaSesionOperativaCTDesarrollo(ruta string) bool {
	return rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta) ||
		ruta == httpinterno.RutaResultadoCobertura
}

// Todas las decisiones de una petición usan la misma sesión revalidada y el
// mismo recibo registrado. El holder nace exclusivamente en la frontera mTLS.
func (s *soporteAltaContratacionTemporalDesarrollo) contextoOperativoDesarrollo(
	ctx context.Context,
) (ports.ContextoAutorizacionAltaV3, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	if s == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || capacidad.contextoOperacion == nil ||
		!rutaSesionOperativaCTDesarrollo(capacidad.ruta) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	s.mu.Lock()
	esperado := s.contextoEsperadoRegistrado
	sesion := s.sesionOperativa
	if rutaPerfilCoberturaCTDesarrollo(capacidad.ruta) {
		esperado = s.contextoEsperadoRegistradoCobertura
		sesion = s.sesionOperativaCobertura
	}
	if rutaReincorporacionTitularDesarrollo(capacidad.ruta) {
		if s.reincorporacionTitular == nil {
			s.mu.Unlock()
			return vacio, ports.ErrAutorizacionDenegada
		}
		esperado = s.reincorporacionTitular.contextoEsperadoRegistrado
		sesion = s.reincorporacionTitular.sesionOperativa
	}
	s.mu.Unlock()
	if esperado.Validar() != nil || sesion == nil {
		if rutaConsultaRespuestaCTDesarrollo(capacidad.ruta) {
			return vacio, ports.ErrConsultaRRHHNoDisponible
		}
		return vacio, ports.ErrAutorizacionDenegada
	}
	holder := capacidad.contextoOperacion
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.soporte == nil {
		holder.soporte = s
		comun, err := sesion.ResolverContexto(ctx)
		if err == nil && comun.Vinculo.ValidarPara(comun.Resultado) == nil &&
			mismoContextoEsperadoRegistradoDesarrollo(esperado, comun.Resultado) {
			holder.contexto = ports.ContextoAutorizacionAltaV3{Vinculo: comun.Vinculo, Resultado: comun.Resultado}
		} else if err != nil && rutaConsultaRespuestaCTDesarrollo(capacidad.ruta) &&
			(errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
			holder.err = ports.ErrConsultaRRHHNoDisponible
		} else {
			holder.err = ports.ErrAutorizacionDenegada
		}
	}
	datos, err := holder.contexto.Vinculo.Datos()
	if rutaConsultaRespuestaCTDesarrollo(capacidad.ruta) && errors.Is(holder.err, ports.ErrConsultaRRHHNoDisponible) {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	if holder.soporte != s || holder.err != nil || err != nil ||
		holder.contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: datos.AutenticacionRef,
			SesionRef:        datos.SesionRef,
			PerfilRef:        esperado.Contexto.PerfilActivoRef,
		}, s.reloj.Ahora()) != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	resultado, err := holder.contexto.Resultado.Clonar()
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: holder.contexto.Vinculo, Resultado: resultado}, nil
}
