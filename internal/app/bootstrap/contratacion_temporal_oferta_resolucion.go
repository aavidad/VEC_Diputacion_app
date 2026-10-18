package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ct.ResolutorCierreOfertaBolsaCT = publicadorOfertaBolsaCTDesarrollo{}

func (p publicadorOfertaBolsaCTDesarrollo) Resolver(ctx context.Context,
	i ct.IntencionSeleccionOfertaCT, c ct.ComandoOfertaBolsaCT,
) (ct.ResultadoCierreOfertaBolsaCT, error) {
	vacio := ct.ResultadoCierreOfertaBolsaCT{}
	if ctx == nil || ctx.Err() != nil || c.ValidarPara(i) != nil || p.puente == nil ||
		p.puente.operacionNativa == nil || p.puente.operacionNativa.preparadorOfertas == nil ||
		dependenciaBootstrapNula(p.resolucion) || dependenciaBootstrapNula(p.autoridadCierre) {
		return vacio, ct.ErrIntegracionBolsaNoDisponible
	}
	captura, err := p.puente.capturaOperacionNativa(ctx)
	if err != nil {
		return vacio, err
	}
	if captura.ct.Resultado.Contexto.PersonaRef != i.ActorRef ||
		captura.bolsa.Resultado.Contexto.PersonaRef != i.ActorRef ||
		captura.ct.Resultado.Contexto.Instantanea.CuentaRef != i.ResultadoContexto.Contexto.Instantanea.CuentaRef {
		return vacio, ct.ErrAutorizacionDenegada
	}
	ambitos, err := p.puente.operacionNativa.preparadorOfertas.ResolverContextoContactosBolsa(
		captura.ctx, captura.bolsa.Resultado.Contexto, c.BolsaRef)
	if err != nil {
		return vacio, err
	}
	comando, err := json.Marshal(c)
	if err != nil {
		return vacio, err
	}
	canon, err := bolsa.CanonCierreIntencionOfertaCT(i.ActorRef, comando)
	if err != nil {
		return vacio, err
	}
	material, err := p.autoridadCierre.MaterialCierreBolsa(ctx, i, c, canon, ambitos.UnidadRef, ambitos.AmbitoRef)
	if err != nil {
		return vacio, err
	}
	r, err := p.resolucion.Resolver(ctx, bolsa.ComandoResolverIntencionOfertaCT{
		ActorRef: i.ActorRef, UnidadRef: ambitos.UnidadRef, AmbitoRef: ambitos.AmbitoRef,
		Comando: comando, MaterialCanonico: canon, Material: material})
	if err != nil {
		return vacio, err
	}
	return proyectarResultadoResolucionOfertaCT(i, c, r)
}

func proyectarResultadoResolucionOfertaCT(i ct.IntencionSeleccionOfertaCT, c ct.ComandoOfertaBolsaCT,
	r bolsa.ResultadoResolverIntencionOfertaCT,
) (ct.ResultadoCierreOfertaBolsaCT, error) {
	vacio := ct.ResultadoCierreOfertaBolsaCT{}
	salida := ct.ResultadoCierreOfertaBolsaCT{Situacion: ct.SituacionCierreOfertaBolsaCT(r.Situacion)}
	switch r.Situacion {
	case bolsa.ResultadoIntencionOfertaPublicada:
		if r.Oferta == nil || len(r.Cierre) != 0 {
			return vacio, ct.ErrCierreOfertaBolsaCTInvalido
		}
		oferta, err := proyectarOfertaBolsaSeleccionCT(c, *r.Oferta)
		if err != nil {
			return vacio, err
		}
		salida.Oferta = &oferta
	case bolsa.ResultadoIntencionOfertaCerrada:
		if r.Oferta != nil || len(r.Cierre) == 0 {
			return vacio, ct.ErrCierreOfertaBolsaCTInvalido
		}
		var cierre ct.ReciboCierreOfertaCT
		decoder := json.NewDecoder(bytes.NewReader(r.Cierre))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&cierre) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			return vacio, ct.ErrCierreOfertaBolsaCTInvalido
		}
		salida.Cierre = &cierre
	default:
		return vacio, ct.ErrCierreOfertaBolsaCTInvalido
	}
	if err := salida.ValidarPara(i, c); err != nil {
		return vacio, err
	}
	return salida, nil
}
