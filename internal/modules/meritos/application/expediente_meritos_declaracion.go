package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// DeclaracionConComprobante vincula una declaracion RUM a una version exacta
// cuyo original ya fue comprobado por Documentos. No habilita acreditacion ni
// deduce puntuacion por la sola presencia del archivo.
type DeclaracionConComprobante struct {
	comprobantes ports.ConsultaComprobanteMeritoConfirmado
	meritos      *Servicio
}

func NuevaDeclaracionConComprobante(c ports.ConsultaComprobanteMeritoConfirmado, m *Servicio) (*DeclaracionConComprobante, error) {
	if nulo(c) || m == nil {
		return nil, ports.ErrComprobanteMeritoNoDisponible
	}
	return &DeclaracionConComprobante{comprobantes: c, meritos: m}, nil
}

// Declarar consulta Documentos por persona autenticada antes de escribir RUM.
// El adaptador de Documentos autoriza antes de leer y comprueba custodia,
// antivirus y original. Un fallo posterior de RUM permite repetir la consulta.
func (s *DeclaracionConComprobante) Declarar(ctx context.Context, solicitud Solicitud, comprobanteRef string) (ports.Recibo, error) {
	if s == nil || ctx == nil || nulo(s.comprobantes) || s.meritos == nil {
		return ports.Recibo{}, ports.ErrComprobanteMeritoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.Recibo{}, err
	}
	if solicitud.validarContexto() != nil || solicitud.validarComando() != nil || !domain.ReferenciaValida(comprobanteRef) || solicitud.VersionEsperada != 0 ||
		solicitud.Hecho.Version != 1 || solicitud.Hecho.Estado != domain.Declarado || solicitud.Hecho.Revision != nil ||
		len(solicitud.Hecho.Evidencias) != 0 {
		return ports.Recibo{}, ErrSolicitud
	}
	persona := solicitud.Contexto.Contexto.PersonaRef
	if solicitud.Hecho.PersonaRef != persona {
		return ports.Recibo{}, ErrSolicitud
	}
	documento, err := s.comprobantes.ConsultarVersionConfirmadaPropia(ctx, persona, comprobanteRef)
	if err != nil {
		return ports.Recibo{}, err
	}
	if documento.Validar() != nil || !domain.ReferenciaValida(documento.ID) {
		return ports.Recibo{}, ports.ErrComprobanteMeritoNoDisponible
	}
	solicitud.Hecho.Evidencias = []vec.ReferenciaDocumento{documento}
	return s.meritos.Declarar(ctx, solicitud)
}
