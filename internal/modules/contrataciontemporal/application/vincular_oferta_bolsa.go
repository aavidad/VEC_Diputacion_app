package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ServicioVinculoOfertaBolsa consume la asociación propia de Bolsa y acepta
// la versión CT únicamente si el resultado coincide con la transición del
// dominio. La transacción confirma ambos permisos y ambos efectos juntos.
type ServicioVinculoOfertaBolsa struct {
	preparador  ports.PreparadorVinculoOfertaBolsa
	transaccion ports.TransaccionVinculoOfertaBolsa
}

func NuevoServicioVinculoOfertaBolsa(p ports.PreparadorVinculoOfertaBolsa,
	t ports.TransaccionVinculoOfertaBolsa) (*ServicioVinculoOfertaBolsa, error) {
	if dependenciaNula(p) || dependenciaNula(t) {
		return nil, ports.ErrVinculoOfertaBolsaNoDisponible
	}
	return &ServicioVinculoOfertaBolsa{p, t}, nil
}

func (s *ServicioVinculoOfertaBolsa) Vincular(ctx context.Context,
	solicitud ports.SolicitudVincularOfertaBolsa) (ports.ResultadoVinculoOfertaBolsa, error) {
	vacio := ports.ResultadoVinculoOfertaBolsa{}
	if s == nil || ctx == nil || dependenciaNula(s.preparador) || dependenciaNula(s.transaccion) {
		return vacio, ports.ErrVinculoOfertaBolsaNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	preparacion, err := s.preparador.PrepararVinculoOfertaBolsa(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if preparacion.ValidarPara(solicitud) != nil {
		return vacio, ports.ErrVinculoOfertaBolsaDenegado
	}
	resultado, err := s.transaccion.VincularOfertaBolsa(ctx, preparacion)
	if err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !resultadoVinculoOfertaBolsaExacto(solicitud, preparacion, resultado) {
		return vacio, ports.ErrVinculoOfertaBolsaNoConfiable
	}
	return resultado, nil
}

func resultadoVinculoOfertaBolsaExacto(s ports.SolicitudVincularOfertaBolsa,
	p ports.PreparacionVinculoOfertaBolsa, r ports.ResultadoVinculoOfertaBolsa) bool {
	if r.Anterior.Validar() != nil || r.Siguiente.Validar() != nil ||
		r.Anterior.OrganizacionRef != s.OrganizacionRef ||
		r.Anterior.Referencia != s.ExpedienteRef || r.Anterior.Version != s.VersionEsperada ||
		r.Anterior.Flujo != p.Definicion.Flujo || r.Anterior.Circuito == nil ||
		r.Anterior.Analisis == nil || r.Anterior.Asignacion == nil || r.Anterior.ViaCobertura == nil ||
		r.Anterior.ViaCobertura.BolsaRef != s.BolsaRef ||
		r.Anterior.Asignacion.UnidadRef != p.UnidadRef ||
		r.OfertaRef != s.OfertaRef || r.BolsaRef != s.BolsaRef ||
		r.NumeroPlaza != s.NumeroPlaza ||
		!domain.ReferenciaOpacaValida(r.ReciboPublicacionRef) ||
		!huellaOfertaBolsaValida(r.PublicacionSHA256) ||
		!domain.ReferenciaOpacaValida(r.ReciboAsociacionBolsaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboCTRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaCTRef) ||
		!domain.ReferenciaOpacaValida(r.EventoCTRef) ||
		!domain.InstanteUTCCanonico(r.PublicadaEn) ||
		!domain.InstanteUTCCanonico(r.AsociadaEn) ||
		!domain.InstanteUTCCanonico(r.RegistradaEn) ||
		r.PublicadaEn.After(r.AsociadaEn) || r.AsociadaEn.After(r.RegistradaEn) {
		return false
	}
	credito := r.Anterior.Circuito.Hitos
	if len(credito) == 0 || credito[len(credito)-1].Tipo != domain.HitoCreditoComprobado ||
		r.PublicadaEn.Before(credito[len(credito)-1].RegistradoEn) {
		return false
	}
	actuacion := domain.DatosActuacion{
		AccionClave: domain.AccionVincularOfertaBolsa,
		ActorRef:    p.ActorRef, UnidadRef: p.UnidadRef, ReciboRef: r.ReciboCTRef,
		RealizadaEn: r.RegistradaEn, FaseDestino: r.Anterior.FaseActual,
		EstadoDestino: r.Anterior.EstadoActual,
	}
	esperado, err := r.Anterior.VincularOfertaBolsa(
		s.VersionEsperada, p.Definicion, s.OfertaRef, p.PerfilRef, actuacion)
	if err != nil {
		return false
	}
	esperadaJSON, err := json.Marshal(esperado)
	recibidaJSON, errRecibida := json.Marshal(r.Siguiente)
	return err == nil && errRecibida == nil && bytes.Equal(esperadaJSON, recibidaJSON)
}

func huellaOfertaBolsaValida(valor string) bool {
	b, err := hex.DecodeString(valor)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == valor
}
