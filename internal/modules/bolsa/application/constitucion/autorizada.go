package constitucion

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// ServicioAutorizado constituye la bolsa de un acta ya importada cuando RRHH
// confirma la carga desde la pantalla (B1). Construye la constitución igual
// que Constituir, pero la persiste con el consumo de la decisión V3 de la
// carga en la misma transacción (B79). Los vínculos `can_* → participación`
// se registran después, igual que en la línea de órdenes.
type ServicioAutorizado struct {
	base       *Servicio
	autorizado ports.RepositorioConstitucionCargaConvoca
}

func NuevoServicioAutorizado(base *Servicio, autorizado ports.RepositorioConstitucionCargaConvoca) (*ServicioAutorizado, error) {
	if base == nil || base.recuperador == nil || base.repositorio == nil || base.derivador == nil || base.reloj == nil || autorizado == nil {
		return nil, ErrDependenciasRequeridas
	}
	return &ServicioAutorizado{base: base, autorizado: autorizado}, nil
}

// Constituir recupera el lote del acta, construye la constitución y la
// confirma consumiendo el material. El actor es el titular de la decisión.
func (s *ServicioAutorizado) Constituir(ctx context.Context, solicitud Solicitud, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error) {
	if ctx == nil || s == nil || s.base == nil {
		return ports.ReciboCargaConvoca{}, ErrDependenciasRequeridas
	}
	if solicitud.HuellaFicheroSHA256 == "" || solicitud.CategoriaRef == "" || solicitud.ActorRef == "" || material.ValidarEstructura() != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaInvalida
	}
	lote, _, existe, err := s.base.recuperador.RecuperarLote(ctx, solicitud.HuellaFicheroSHA256, solicitud.CategoriaRef)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	if !existe {
		return ports.ReciboCargaConvoca{}, ErrActaNoEncontrada
	}
	ahora := s.base.reloj().UTC().Truncate(time.Microsecond)
	constitucion, vinculos, pendientes, err := construirConstitucion(lote, solicitud.ActorRef, ahora, s.base.derivador)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	recibo, err := s.autorizado.ConstituirCargaConvocaAutorizada(ctx, constitucion, material)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	if len(vinculos) > 0 {
		recibo.Vinculos, err = s.base.repositorio.RegistrarVinculos(ctx, recibo.ActaRef, vinculos, ahora)
		if err != nil {
			return ports.ReciboCargaConvoca{}, err
		}
	}
	recibo.PendientesRevision = pendientes
	return recibo, nil
}
