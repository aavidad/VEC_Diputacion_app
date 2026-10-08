package bootstrap

import (
	"context"

	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	vecapplication "vec-diputacion-granada/internal/vec/application"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// politicasConservacionR5Desarrollo mantiene las solicitudes exactas v1 ya
// guardadas mientras los nuevos originales CT usan v2. La versión de la
// solicitud determina cuál catálogo puede resolverla; no hay sustitución de
// política ni selección por una referencia aportada sin cotejo.
type politicasConservacionR5Desarrollo struct {
	actual, historico *conservacion.Catalogo
}

var _ vecports.ResolutorPoliticaConservacionDocumental = politicasConservacionR5Desarrollo{}

func catalogosDocumentosR5Desarrollo(reloj vecports.Reloj, activo bool) (*conservacion.Catalogo,
	vecports.ResolutorPoliticaConservacionDocumental, error,
) {
	v1, err := conservacion.NuevoCatalogoProvisional(reloj)
	if err != nil {
		return nil, nil, err
	}
	if !activo {
		return v1, v1, nil
	}
	v2, err := conservacion.NuevoCatalogoProvisionalV2(reloj)
	if err != nil {
		return nil, nil, err
	}
	return v2, politicasConservacionR5Desarrollo{actual: v2, historico: v1}, nil
}

func (p politicasConservacionR5Desarrollo) BuscarPoliticasConservacionDocumental(
	ctx context.Context, s vecports.SolicitudPoliticaConservacionDocumental,
) ([]vecports.PoliticaConservacionDocumental, error) {
	if p.actual == nil || p.historico == nil || ctx == nil || ctx.Err() != nil || s.Validar() != nil {
		return nil, vecports.ErrPoliticaConservacionDocumentalNoResuelta
	}
	actual, err := p.actual.BuscarPoliticasConservacionDocumental(ctx, s)
	if err != nil || len(actual) != 0 {
		return actual, err
	}
	return p.historico.BuscarPoliticasConservacionDocumental(ctx, s)
}

// montajeOriginalFirmaR5Desarrollo comparte la custodia y la autoridad de
// Documentos con las dos vías de firma. La lectura del objeto usa el PDP de
// CT de la misma petición, también para el PDF firmado anterior.
type montajeOriginalFirmaR5Desarrollo struct {
	original           *originalFirmableCTDesarrollo
	servicioOriginal   *vecapplication.ServicioOriginalFirmableCT
	servicioDocumentos *docapp.Servicio
	tipos              *ctadapters.TiposOriginalFirmableRRHH
}

// nuevoMontajeOriginalFirmaR5Desarrollo recibe la fuente RRHH ya autorizada
// por la composición de CT. El catálogo de Documentos debe publicar seis
// tipos originales distintos de los tipos reservados a PDF firmados.
func nuevoMontajeOriginalFirmaR5Desarrollo(
	fuente vecports.FuentePDFOriginalCT,
	documentos *autoridadDocumentosDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
) (*montajeOriginalFirmaR5Desarrollo, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(fuente) || documentos == nil || documentos.servicio == nil ||
		documentos.politicas == nil || !documentos.seudonimizador.valido() ||
		documentos.servicio.Repositorio == nil || documentos.servicio.Almacen == nil ||
		alta == nil || alta.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(alta.autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(documentos.reloj) {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	tipos, err := ctadapters.NuevosTiposOriginalFirmableRRHH(documentos.politicas)
	if err != nil {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	original, err := nuevoOriginalFirmableCTDesarrollo(pdpCTOriginalFirmableDesarrollo{alta: alta},
		documentos.seudonimizador, documentos.politicas, documentos.reloj)
	if err != nil {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	fabricaLectura, err := docautorizacion.NuevaFabricaContextoLecturaOriginalV3(original, documentos.reloj)
	if err != nil {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	servicioDocumentos := *documentos.servicio
	servicioDocumentos.ContextosLectura = fabricaLectura
	custodia, err := almacen.NuevaCustodiaDocumentosOriginalCT(
		servicioDocumentosOriginalCTDesarrollo{servicio: &servicioDocumentos}, original, original.mapear, tipos)
	if err != nil {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	servicioOriginal, err := vecapplication.NuevoServicioOriginalFirmableCT(fuente, custodia, tipos)
	if err != nil {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	return &montajeOriginalFirmaR5Desarrollo{
		original: original, servicioOriginal: servicioOriginal,
		servicioDocumentos: &servicioDocumentos, tipos: tipos,
	}, nil
}
