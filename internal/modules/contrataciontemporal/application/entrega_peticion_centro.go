package application

import (
	"context"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioEntregaPeticionCentro struct {
	repositorio ports.RepositorioEntregasPeticionCentro
	registro    ports.RegistradorExpedientePeticionCentro
}

func NuevoServicioEntregaPeticionCentro(r ports.RepositorioEntregasPeticionCentro, alta ports.RegistradorExpedientePeticionCentro) (*ServicioEntregaPeticionCentro, error) {
	if dependenciaNula(r) || dependenciaNula(alta) {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	return &ServicioEntregaPeticionCentro{r, alta}, nil
}

// Entregar coordina dos capacidades durables. Una interrupción conserva la
// preparación y su clave; nunca se declara confirmado solo por crear el alta.
func (s *ServicioEntregaPeticionCentro) Entregar(ctx context.Context, c ports.ComandoEntregarPeticionCentro) (ports.EntregaPeticionCentro, error) {
	var vacia ports.EntregaPeticionCentro
	if s == nil || ctx == nil {
		return vacia, ports.ErrPeticionCentroNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if err := c.Validar(); err != nil {
		return vacia, err
	}
	e, err := s.repositorio.PrepararEntrega(ctx, c)
	if err != nil {
		return vacia, err
	}
	if e.ValidarReserva() != nil || e.Peticion.Referencia != c.PeticionRef {
		return vacia, ports.ErrReciboPeticionCentroNoConfiable
	}
	if e.EstadoEntrega == "confirmada" {
		return e, nil
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	// Copia defensiva: el adaptador no puede modificar el original ratificado.
	copia := e
	copia.Peticion.Solicitud, err = e.Peticion.Solicitud.Clonar()
	if err != nil {
		return vacia, err
	}
	alta, err := s.registro.RegistrarExpedientePeticion(ctx, copia)
	if err != nil {
		return vacia, err
	}
	if alta.Recibo.ValidarEstructura() != nil || alta.Recibo.Version != 1 || !ports.SelloHMACSHA256Valido(alta.AmbitoHMAC) {
		return vacia, ports.ErrReciboPeticionCentroNoConfiable
	}
	confirmada, err := s.repositorio.ConfirmarEntrega(ctx, c, alta)
	if err != nil {
		return vacia, err
	}
	if confirmada.ValidarReserva() != nil || confirmada.EstadoEntrega != "confirmada" ||
		!reflect.DeepEqual(confirmada.Peticion, e.Peticion) || confirmada.ClaveAlta != e.ClaveAlta || confirmada.ActorRef != e.ActorRef || confirmada.PerfilRef != e.PerfilRef ||
		confirmada.AmbitoAltaHMAC != e.AmbitoAltaHMAC || !reflect.DeepEqual(*confirmada.ReciboAlta, alta.Recibo) {
		return vacia, ports.ErrReciboPeticionCentroNoConfiable
	}
	return confirmada, nil
}
