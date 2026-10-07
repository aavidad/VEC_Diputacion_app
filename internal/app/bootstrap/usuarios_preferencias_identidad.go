package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	"vec-diputacion-granada/internal/shared/telemetria"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

type revalidadorPreferenciasMedido struct {
	core.RevalidadorAutenticacionActorV1
	medir bool
}

func (m revalidadorPreferenciasMedido) RevalidarAutenticacionActorV1(ctx context.Context, s core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	inicio := time.Now()
	r, err := m.RevalidadorAutenticacionActorV1.RevalidarAutenticacionActorV1(ctx, s)
	if m.medir {
		telemetria.RegistrarFase(ctx, telemetria.FaseSesion, time.Since(inicio), err)
	}
	return r, err
}

type resolutorPreferenciasMedido struct {
	core.ResolutorContextoActorRegistradoV2
	medir bool
}

func (m resolutorPreferenciasMedido) ResolverContextoActorRegistradoV2(ctx context.Context, s core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	inicio := time.Now()
	r, err := m.ResolutorContextoActorRegistradoV2.ResolverContextoActorRegistradoV2(ctx, s)
	if m.medir {
		telemetria.RegistrarFase(ctx, telemetria.FaseContexto, time.Since(inicio), err)
	}
	return r, err
}

func (a *autoridadPreferenciasUsuariosDesarrollo) resolverSesion(r *http.Request, cuenta cuentaUsuariosPreferenciasDesarrollo, ahora time.Time) (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error) {
	vacio := core.VinculoAutenticacionActorV2{}
	resultadoVacio := core.ResultadoContextoActorRegistradoV2{}
	if a == nil || a.base == nil || r == nil || r.TLS == nil || len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) == 0 ||
		(a.superficie != core.SuperficieAutenticacionExternaPersonalV1 && a.superficie != core.SuperficieAutenticacionInternaCorporativaV1) {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	medir := a.ruta == usuarioshttp.RutaMisPreferencias || a.ruta == usuarioshttp.RutaMisPreferenciasAreaPersonal
	asercion, err := nonceRutasDietas()
	if err != nil {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	sesion, err := nonceRutasDietas()
	if err != nil {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	hasta := ahora.Add(2 * time.Minute)
	if limite := r.TLS.VerifiedChains[0][0].NotAfter.UTC().Truncate(time.Microsecond); limite.Before(hasta) {
		hasta = limite
	}
	superficie := httpseguridad.Superficie(a.superficie)
	politica := "dev-certificado-mtls-v1;solo-sintetico;canal-privado-validado;garantia-alta-desarrollo;vigencia-120s;sin-kerberos;no-corporativa"
	alta := httpseguridad.AltaSesionAtomica{
		AsercionID: asercion, SesionID: sesion, SujetoID: cuenta.Sujeto, CuentaID: "desarrollo:" + cuenta.CuentaRef,
		Superficie: superficie, EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh,
		AutenticacionVerificadaEn: ahora, SesionEmitidaEn: ahora, AsercionExpiraEn: hasta,
		PoliticaGarantiaRef:          referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1"),
		PoliticaGarantiaHuellaSHA256: huellaRutasDietas(politica),
		AutenticacionHuellaSHA256:    huellaRutasDietas(a.base.instancia + "|" + asercion + "|" + sesion + "|" + cuenta.CertificadoSHA256 + "|" + cuenta.CuentaRef + "|" + cuenta.PerfilRef + "|" + r.URL.Path + "|" + r.Method + "|" + string(a.superficie) + "|" + ahora.Format(time.RFC3339Nano)),
	}
	inicioSesion := time.Now()
	confirmacion, err := a.base.registro.ConsumirAsercionYRegistrar(r.Context(), alta)
	if medir {
		telemetria.RegistrarFase(r.Context(), telemetria.FaseSesion, time.Since(inicioSesion), err)
	}
	if err != nil || confirmacion.ValidarPara(alta) != nil || confirmacion.CuentaRef != cuenta.CuentaRef {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{delegado: a.base.revalidador, alta: alta, confirmacion: confirmacion, reloj: a.reloj, superficie: superficie}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(r.Context(), revalidadorPreferenciasMedido{revalidador, medir},
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef},
		resolutorPreferenciasMedido{a.base.contextos, medir}, core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta.CuentaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}, PerfilActivoRef: cuenta.PerfilRef}, a.reloj)
	if err != nil {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.CuentaRef != cuenta.CuentaRef || datos.PerfilActivoRef != cuenta.PerfilRef || datos.CuentaPrivilegiada ||
		datos.Superficie != a.superficie || !vinculo.VigenteEn(a.reloj.Ahora(), resultado) ||
		resultado.Contexto.PersonaRef == "" || resultado.Contexto.PersonaRef != resultado.Contexto.Instantanea.PersonaRef {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	if errors.Is(r.Context().Err(), context.Canceled) {
		return vacio, resultadoVacio, errComposicionUsuariosPreferencias
	}
	return vinculo, resultado, nil
}
