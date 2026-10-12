package internagobierno

import (
	"context"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type claveContextoOriginalCT struct{}
type contextoOriginalCTSellado struct {
	fuente   *FuenteF1
	peticion inc.PeticionAutoridad
	valor    ct.ContextoAutorizacionAltaV3
}

// VincularContextoOriginalCT conserva la única prueba F1 tras el mTLS de la
// petición. El cliente no puede aportar actor, perfil ni organización.
func (f *FuenteF1) VincularContextoOriginalCT(ctx context.Context) (context.Context, error) {
	if f == nil || ctx == nil || ctx.Err() != nil || ctx.Value(claveContextoOriginalCT{}) != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	peticion, r, err := f.ResolverPeticionYContexto(ctx)
	if err != nil || r.Resultado.Validar() != nil || r.Vinculo.ValidarPara(r.Resultado) != nil ||
		!r.Vinculo.VigenteEn(f.reloj.Ahora(), r.Resultado) {
		return nil, ErrGobiernoInternoNoDisponible
	}
	clon, err := r.Resultado.Clonar()
	if err != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	r.Resultado = clon
	return context.WithValue(ctx, claveContextoOriginalCT{}, contextoOriginalCTSellado{fuente: f, peticion: peticion, valor: r}), nil
}

// PeticionYContextoVinculadoOriginalCT permite que Detalle actual e histórico
// consuman la misma resolución F1 que el puente, sin segunda cápsula.
func (f *FuenteF1) PeticionYContextoVinculadoOriginalCT(ctx context.Context) (inc.PeticionAutoridad, ct.ContextoAutorizacionAltaV3, error) {
	var vacia inc.PeticionAutoridad
	r, err := f.ContextoVinculadoOriginalCT(ctx)
	if err != nil {
		return vacia, ct.ContextoAutorizacionAltaV3{}, err
	}
	sello := ctx.Value(claveContextoOriginalCT{}).(contextoOriginalCTSellado)
	return sello.peticion, r, nil
}

func (f *FuenteF1) ContextoVinculadoOriginalCT(ctx context.Context) (ct.ContextoAutorizacionAltaV3, error) {
	var vacio ct.ContextoAutorizacionAltaV3
	if f == nil || f.reloj == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	sello, ok := ctx.Value(claveContextoOriginalCT{}).(contextoOriginalCTSellado)
	if !ok || sello.fuente != f || sello.valor.Resultado.Validar() != nil ||
		sello.valor.Vinculo.ValidarPara(sello.valor.Resultado) != nil ||
		!sello.valor.Vinculo.VigenteEn(f.reloj.Ahora(), sello.valor.Resultado) {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	r := sello.valor
	clon, err := r.Resultado.Clonar()
	if err != nil {
		return vacio, ErrGobiernoInternoNoDisponible
	}
	r.Resultado = clon
	return r, nil
}

func (f *FuenteF1) ResolverOrganizacionFirmaVec(ctx context.Context) (string, error) {
	r, err := f.ContextoVinculadoOriginalCT(ctx)
	if err != nil {
		return "", ErrGobiernoInternoNoDisponible
	}
	actor := r.Resultado.Contexto
	nominal, ok := f.porCuenta[actor.Instantanea.CuentaRef]
	sello := ctx.Value(claveContextoOriginalCT{}).(contextoOriginalCTSellado)
	if !ok || nominal.PerfilActivoRef != actor.PerfilActivoRef || nominal.OrganizacionRef == "" ||
		sello.peticion.PreparacionCT.OrganizacionRef != nominal.OrganizacionRef ||
		sello.peticion.PreparacionCT.ActorRef != actor.PersonaRef {
		return "", ErrGobiernoInternoNoDisponible
	}
	return nominal.OrganizacionRef, nil
}
