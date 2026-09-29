package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// La lectura previa al alta reutiliza el resolutor durable O4-04B del rol
// ejecutor. La referencia de sondeo es interna y nunca representa un alta ni
// cruza a HTTP, al PDP o a la auditoría funcional de RRHH.
type fuentePreparacionCoberturaVigenteDesarrollo struct {
	resolutor cobertura.ResolutorGobiernoOperacionCobertura
	reloj     cobertura.RelojGobiernoOperacionCobertura
}

var _ application.FuenteCatalogoCoberturaVigente = (*fuentePreparacionCoberturaVigenteDesarrollo)(nil)

func nuevaFuentePreparacionCoberturaVigenteDesarrollo(
	resolutor cobertura.ResolutorGobiernoOperacionCobertura,
	reloj cobertura.RelojGobiernoOperacionCobertura,
) (*fuentePreparacionCoberturaVigenteDesarrollo, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(resolutor) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return &fuentePreparacionCoberturaVigenteDesarrollo{
		resolutor: resolutor, reloj: reloj,
	}, nil
}

func (f *fuentePreparacionCoberturaVigenteDesarrollo) ConsultarCatalogoViasCoberturaVigente(
	ctx context.Context,
	organizacionRef string,
) (domain.CatalogoViasCobertura, error) {
	if ctx == nil || f == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(f.resolutor) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(f.reloj) ||
		organizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return domain.CatalogoViasCobertura{}, application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoViasCobertura{}, err
	}
	solicitud, err := cobertura.NuevaSolicitudGobiernoDecisionCobertura(
		organizacionRef, expedienteSondeoGobiernoCT, 1,
	)
	if err != nil {
		return domain.CatalogoViasCobertura{}, application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(ctx, f.reloj, f.resolutor, solicitud)
	if err != nil {
		return domain.CatalogoViasCobertura{}, application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	datos, err := gobierno.DesplegarPara(ctx, f.reloj, solicitud)
	if err != nil || datos.Catalogo.Canon() != domain.CanonHuellaCatalogoCoberturaV2() ||
		!datos.PoliticaActuacion.Catalogo.CoincideExactamente(datos.Catalogo.Identidad()) {
		return domain.CatalogoViasCobertura{}, application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	catalogo, err := domain.RestaurarCatalogoViasCobertura(datos.Catalogo.Publicacion())
	if err != nil {
		return domain.CatalogoViasCobertura{}, application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return catalogo, nil
}
