package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Ofertas publicadas de Bolsa (art. 8.1 del Reglamento). Comparten sesión,
// política, motivo y material con la emisión B7: publicar y resolver son
// modalidades del mismo llamamiento, sin acción ni consumidor nuevos.
const (
	claveFronteraPublicarOfertaBolsa  = "bolsa-bof-oferta-publicar"
	claveFronteraConsultarOfertaBolsa = "bolsa-bof-oferta-consultar"
	claveFronteraResolverOfertaBolsa  = "bolsa-bof-oferta-resolver"
)

func descriptoresFronterasOfertasBolsaDesarrollo(perfilActivoRef string) []descriptorFronteraComunDesarrollo {
	return []descriptorFronteraComunDesarrollo{
		{Clave: claveFronteraPublicarOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaOfertasPublicadas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraConsultarOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaOfertasPublicadas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraResolverOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaResolucionesOferta, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
	}
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudPublicarOferta(ctx context.Context, entrada bolsahttp.EntradaPublicarOferta) (puertosbolsa.SolicitudPublicarOferta, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudPublicarOferta{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudPublicarOferta{}, err
	}
	return puertosbolsa.SolicitudPublicarOferta{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, Datos: entrada.Datos, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoEmitirLlamamientoBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudResolverOferta(ctx context.Context, entrada bolsahttp.EntradaResolverOferta) (puertosbolsa.SolicitudResolverOferta, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudResolverOferta{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudResolverOferta{}, err
	}
	return puertosbolsa.SolicitudResolverOferta{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, OfertaRef: entrada.OfertaRef, ParticipacionRef: entrada.ParticipacionRef, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoEmitirLlamamientoBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudConsultarOfertas(ctx context.Context, bolsaRef string, limite int) (puertosbolsa.SolicitudConsultarOfertas, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConsultarOfertas{}, err
	}
	return puertosbolsa.SolicitudConsultarOfertas{ContextoActor: contexto.Resultado.Contexto, BolsaRef: bolsaRef, Limite: limite}, nil
}

var _ bolsahttp.PreparadorOfertasPublicadas = (*preparadorBorradorLlamamientoDesarrollo)(nil)

// calculadoraPlazoOfertaDesarrollo resuelve la regla b10 del catálogo de
// reglas de Bolsa. El resolutor se fija una sola vez al componer, antes de
// servir; sin él no hay plazo y la publicación se rechaza.
type calculadoraPlazoOfertaDesarrollo struct {
	resolutor atomic.Pointer[reglas.Resolutor]
}

func (c *calculadoraPlazoOfertaDesarrollo) fijar(resolutor *reglas.Resolutor) {
	if c != nil && resolutor != nil {
		c.resolutor.CompareAndSwap(nil, resolutor)
	}
}

func (c *calculadoraPlazoOfertaDesarrollo) PlazoDisposicion(ctx context.Context, publicadaEn time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	if c == nil || c.resolutor.Load() == nil {
		return puertosbolsa.PlazoOferta{}, time.Time{}, puertosbolsa.ErrPlazoOfertaNoConfigurado
	}
	regla, vencimiento, err := c.resolutor.Load().Vencimiento(ctx, reglas.BolsaPlazoPublicacion, publicadaEn, "")
	if err != nil {
		if errors.Is(err, reglas.ErrReglaNoEncontrada) || errors.Is(err, reglas.ErrReglasNoConfiguradas) {
			return puertosbolsa.PlazoOferta{}, time.Time{}, errors.Join(puertosbolsa.ErrPlazoOfertaNoConfigurado, err)
		}
		return puertosbolsa.PlazoOferta{}, time.Time{}, errors.Join(puertosbolsa.ErrOfertaNoDisponible, err)
	}
	return puertosbolsa.PlazoOferta{
		ReglaRef: regla.Referencia, HuellaCatalogo: regla.HuellaCatalogo, Unidad: string(regla.Unidad),
		Cantidad: regla.Cantidad, Computo: string(regla.Computo), UltimoDia: vencimiento.UltimoDia,
		Ejemplo: regla.EsEjemplo(), Articulo: regla.Articulo, Calendarios: vencimiento.Calendarios,
	}, vencimiento.VenceAntesDe, nil
}

var _ puertosbolsa.CalculadoraPlazoOferta = (*calculadoraPlazoOfertaDesarrollo)(nil)
