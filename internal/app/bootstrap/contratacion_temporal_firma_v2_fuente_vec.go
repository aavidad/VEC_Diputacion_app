package bootstrap

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal/firmavec"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// El bootstrap conserva las firmas antiguas. El contenedor AUT56 y la fuente
// nominal viven en el paquete neutral consumible por la raíz interna.
type contenedorCompetenciaFirmaVecV2 = firmavec.CompetenciaFirmaVecV2
type fuenteNominalFirmaVecV2 = firmavec.FuenteNominalFirmaVecV2
type fuenteCompetenciaFirmaVecV2 = firmavec.FuenteCompetenciaFirmaVecV2

func prepararPeticionFirmaVecV2(r *http.Request) (*http.Request, error) {
	return firmavec.PrepararPeticionFirmaVecV2(r)
}

func competenciaFirmaVecV2DesdeContexto(ctx context.Context) (*contenedorCompetenciaFirmaVecV2, bool) {
	return firmavec.CompetenciaDesdeContextoFirmaVecV2(ctx)
}

func nuevaFuenteCompetenciaFirmaVecV2(delegada ports.FuenteCompetenciaFirmante) (*fuenteCompetenciaFirmaVecV2, error) {
	return firmavec.NuevaFuenteCompetenciaFirmaVecV2(delegada, rolFirmaExternaRegistroCTDesarrollo)
}

// La cápsula pertenece a la autoridad CT existente y sólo la abre su petición.
// El paquete neutral no recibe un vínculo libre ni una selección del cuerpo.
type sesionAcreditadaFirmaVecV2 struct {
	autoridad *autoridadSesionFirmanteV2
	capsula   *capsulaSesionFirmanteV2
}

func (a *autoridadSesionFirmanteV2) AcreditarSesionFirmaVecV2(r *http.Request, seleccion ports.SeleccionFirmanteV2,
	personaCA25, huellaAUT56 string,
) (firmavec.SesionFirmanteAcreditadaV2, error) {
	c, err := a.acreditar(r, seleccion, personaCA25, huellaAUT56)
	if err != nil {
		return nil, err
	}
	return &sesionAcreditadaFirmaVecV2{autoridad: a, capsula: c}, nil
}

func (s *sesionAcreditadaFirmaVecV2) AbrirSesionFirmaVecV2(ctx context.Context, r *http.Request) (
	core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error,
) {
	if s == nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, errSesionFirmanteV2Denegada
	}
	return s.autoridad.abrirConContexto(ctx, r, s.capsula)
}

func (s *sesionAcreditadaFirmaVecV2) RevalidarSesionFirmaVecV2(ctx context.Context, r *http.Request) (
	core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error,
) {
	if s == nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, errSesionFirmanteV2Denegada
	}
	return s.autoridad.revalidarConContexto(ctx, r, s.capsula)
}

// El predicado común vive con el emisor neutral de la ruta exacta.
func solicitudAutorizacionFirmaVecV2CTValida(datos core.DatosSolicitudAutorizacionLigadaV3,
	motivo core.ReferenciaEntradaCatalogo) bool {
	return firmavec.SolicitudAutorizacionFirmaVecV2CTValida(datos, motivo)
}
