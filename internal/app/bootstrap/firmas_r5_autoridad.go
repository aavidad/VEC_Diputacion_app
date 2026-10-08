package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// El registrador externo usa una sola identidad y asignación publicada en
// original, preparación y registro. El cuerpo HTTP no selecciona ese perfil.
type autoridadFirmaR5Externa struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
	perfil  *perfilFijoCTDesarrollo
	reloj   relojContratacionTemporalDesarrollo
}

var (
	_ httpinterno.AutoridadCanalRegistroFirmaVec     = (*autoridadFirmaR5Externa)(nil)
	_ httpinterno.AutoridadCanalRegistroFirmaExterna = (*autoridadFirmaR5Externa)(nil)
	_ httpinterno.AutoridadContextoCanalCircuitoRRHH = (*autoridadFirmaR5Externa)(nil)
	_ ports.ResolutorContextoAutorizacionAltaV3      = (*autoridadFirmaR5Externa)(nil)
)

func (a *autoridadFirmaR5Externa) contexto(ctx context.Context, ruta string) (ports.ContextoAutorizacionAltaV3, error) {
	var cero ports.ContextoAutorizacionAltaV3
	if a == nil || a.soporte == nil || a.perfil == nil || contextoInterfazNulo(ctx) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if a.perfil.clave != clavePerfilFijoFirmaExternaV2CTDesarrollo || a.perfil.metodo != http.MethodPost ||
		!a.perfil.atiende(ruta) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != ruta || capacidad.metodo != http.MethodPost ||
		a.soporte.perfilFijoParaContexto(ctx, ruta) != a.perfil ||
		!certificadoConsultaReciboRespuestaVigente(capacidad, a.reloj.Ahora()) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	operativo, err := a.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return cero, fmt.Errorf("%w: %w", ports.ErrRegistroFirmaDocumentoNoDisponible, err)
	}
	if !contextoRegistradoPerfilFijoCTDesarrollo(operativo.Resultado.Contexto, a.perfil) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	_, estado := a.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, a.perfil)
	if estado == perfilFijoConsumoFuenteNoDisponible {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if estado != perfilFijoConsumoVigente {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return operativo, nil
}

func (a *autoridadFirmaR5Externa) ResolverOrganizacionFirmaVec(ctx context.Context) (string, error) {
	if _, err := a.contexto(ctx, httpinterno.RutaOriginalFirmableCT); err != nil {
		return "", err
	}
	return organizacionAltaContratacionTemporalDesarrollo, nil
}

func (a *autoridadFirmaR5Externa) ResolverOrganizacionFirmaExterna(ctx context.Context) (string, error) {
	if _, err := a.contexto(ctx, httpinterno.RutaRegistroFirmaExterna); err != nil {
		return "", err
	}
	return organizacionAltaContratacionTemporalDesarrollo, nil
}

func (a *autoridadFirmaR5Externa) ResolverContextoCanalCircuitoRRHH(ctx context.Context) (httpinterno.ContextoCanalCircuitoRRHH, error) {
	operativo, err := a.contexto(ctx, httpinterno.RutaPreflightFirmaR5)
	if err != nil {
		return httpinterno.ContextoCanalCircuitoRRHH{}, err
	}
	v, err := operativo.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalCircuitoRRHH{}, fmt.Errorf("%w: %w", ports.ErrFirmaDocumentoDenegada, err)
	}
	return httpinterno.ContextoCanalCircuitoRRHH{
		AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (a *autoridadFirmaR5Externa) ResolverContextoAutorizacionAltaV3(ctx context.Context,
	s ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	operativo, err := a.contexto(ctx, httpinterno.RutaPreflightFirmaR5)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	if err := operativo.ValidarPara(s, a.reloj.Ahora()); err != nil {
		return ports.ContextoAutorizacionAltaV3{}, fmt.Errorf("%w: %w", ports.ErrFirmaDocumentoDenegada, err)
	}
	return operativo, nil
}
