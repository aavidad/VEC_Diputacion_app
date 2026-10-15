package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La oferta usa el servicio de Bolsa y su autorización propia bajo la ruta
// CT real. La captura congelada procede de la reserva interna, nunca de HTTP.
type publicadorOfertaBolsaCTDesarrollo struct {
	puente *puenteBolsaLlamamientoDesarrollo
}

var _ ct.PublicadorOfertaBolsaCT = publicadorOfertaBolsaCTDesarrollo{}

func (p publicadorOfertaBolsaCTDesarrollo) PublicarORecuperar(ctx context.Context,
	i ct.IntencionSeleccionOfertaCT, c ct.ComandoOfertaBolsaCT,
) (ct.OfertaPublicadaSeleccionCT, error) {
	vacio := ct.OfertaPublicadaSeleccionCT{}
	if ctx == nil || ctx.Err() != nil || c.ValidarPara(i) != nil ||
		p.puente == nil || p.puente.operacionNativa == nil ||
		p.puente.operacionNativa.ofertas == nil || p.puente.operacionNativa.preparadorOfertas == nil {
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
	var origen bolsa.OrigenOfertaCT
	decoder := json.NewDecoder(bytes.NewReader(c.OrigenCT))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&origen) != nil || decoder.Decode(&struct{}{}) != io.EOF ||
		origen.ValidarPara(c.BolsaRef, c.NumeroPlazas) != nil ||
		origen.OrganizacionRef != i.OrganizacionRef || origen.ExpedienteRef != i.ExpedienteRef ||
		origen.VersionExpediente != i.VersionExpediente || origen.ClaveCT != i.ClaveIdempotencia ||
		origen.CategoriaRef != i.CategoriaRef || origen.NecesidadRef != c.NecesidadRef {
		return vacio, ct.ErrVinculoOfertaBolsaCTInvalido
	}
	datos := dominiobolsa.DatosOferta{Categoria: c.Datos.Categoria, Centro: c.Datos.Centro,
		FechaInicio: c.Datos.FechaInicio, FechaFin: c.Datos.FechaFin, Descripcion: c.Datos.Descripcion}
	q, err := p.puente.operacionNativa.preparadorOfertas.PrepararSolicitudPublicarOferta(captura.ctx,
		bolsahttp.EntradaPublicarOferta{BolsaRef: c.BolsaRef, Datos: datos,
			NumeroPlazas: c.NumeroPlazas, ClaveIdempotencia: c.ClaveBolsa})
	if err != nil {
		return vacio, err
	}
	q.OrigenCT = &origen
	oferta, err := p.puente.operacionNativa.ofertas.PublicarOferta(captura.ctx, q)
	if err != nil {
		return vacio, err
	}
	salida := ct.OfertaPublicadaSeleccionCT{ClaveBolsa: c.ClaveBolsa, BolsaRef: oferta.BolsaRef,
		OfertaRef: oferta.OfertaRef, ReciboOfertaRef: oferta.ReciboRef, NumeroPlazas: oferta.NumeroPlazas,
		Datos: ct.DatosOfertaSeleccionCT{Categoria: oferta.Datos.Categoria, Centro: oferta.Datos.Centro,
			FechaInicio: oferta.Datos.FechaInicio, FechaFin: oferta.Datos.FechaFin, Descripcion: oferta.Datos.Descripcion},
		PublicadaEn: oferta.PublicadaEn.UTC(), VenceAntesDe: oferta.VenceAntesDe.UTC()}
	if err := salida.ValidarPara(c); err != nil {
		return vacio, err
	}
	return salida, nil
}
