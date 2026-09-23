package bootstrap

import (
	"context"
	"sync"

	"vec-diputacion-granada/internal/app/identidadcomun"
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
	fallo := ports.ErrConsultaRRHHNoDisponible
	if ctx == nil || ctx.Err() != nil || resolutor == nil || soporte == nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	semilla := soporte.contexto.Resultado
	if semilla.Validar() != nil || soporte.principalID == "" ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(soporte.certificadoSHA256) {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	registrado, err := identidadcomun.ResolverEsperadoRegistrado(ctx, resolutor, semilla)
	if err != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, fallo
	}
	return registrado, nil
}

// Un recibo fresco tiene otra referencia e instante. Todo lo demás debe ser
// idéntico a la autoridad fijada al componer la instancia.
func mismoContextoEsperadoRegistradoDesarrollo(
	esperado, actual dominiovec.ResultadoContextoActorRegistradoV2,
) bool {
	return identidadcomun.MismoContextoEsperadoRegistrado(esperado, actual)
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
	s.mu.Lock()
	esperado := s.contextoEsperadoRegistrado
	sesion := s.sesionOperativa
	s.mu.Unlock()
	if esperado.Validar() != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || sesion == nil || capacidad.contextoOperacion == nil ||
		!rutaSesionOperativaCTDesarrollo(capacidad.ruta) {
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
		} else {
			holder.err = ports.ErrAutorizacionDenegada
		}
	}
	datos, err := holder.contexto.Vinculo.Datos()
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
