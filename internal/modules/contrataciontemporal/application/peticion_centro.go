package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ServicioPeticionCentro tramita únicamente la petición previa a RRHH. No
// implementa otra alta de expediente ni altera las actuaciones ya registradas.
type ServicioPeticionCentro struct {
	autoridad   ports.AutoridadPeticionCentro
	repositorio ports.RepositorioPeticionesCentro
	reloj       ports.Reloj
}

func NuevoServicioPeticionCentro(a ports.AutoridadPeticionCentro, r ports.RepositorioPeticionesCentro, reloj ports.Reloj) (*ServicioPeticionCentro, error) {
	if dependenciaNula(a) || dependenciaNula(r) || dependenciaNula(reloj) {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	return &ServicioPeticionCentro{autoridad: a, repositorio: r, reloj: reloj}, nil
}

func (s *ServicioPeticionCentro) Ejecutar(ctx context.Context, entrada ports.ComandoPeticionCentro) (ports.ReciboPeticionCentro, error) {
	var vacio ports.ReciboPeticionCentro
	if s == nil || ctx == nil {
		return vacio, ports.ErrPeticionCentroNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	comando, err := entrada.Clonar()
	if err != nil {
		return vacio, err
	}
	actor, err := s.autoridad.ActorPeticionCentro(ctx)
	if err != nil {
		return vacio, err
	}
	previo, err := s.repositorio.ConsultarOperacion(ctx, actor, comando.ClaveIdempotencia)
	if err != nil {
		return vacio, err
	}
	var material ports.MaterialPeticionCentro
	if previo != nil {
		if !previo.MismaPeticion(actor, comando) {
			return vacio, ports.ErrClavePeticionCentroUsada
		}
		material = *previo
	} else {
		var peticion domain.PeticionCentro
		if comando.Operacion == ports.OperacionPresentarPeticionCentro {
			configuracion, err := s.autoridad.ConfiguracionPeticionCentro(ctx, actor)
			if err != nil {
				return vacio, err
			}
			if configuracion.Solicitante != actor {
				return vacio, domain.ErrRatificacionCentroDenegada
			}
			peticion, err = domain.NuevaPeticionCentro("peticion:centro:"+comando.ClaveIdempotencia, configuracion, *comando.Solicitud, s.reloj.Ahora())
			if err != nil {
				return vacio, err
			}
		} else {
			datos, err := s.repositorio.ObtenerPeticion(ctx, actor, comando.PeticionRef)
			if err != nil {
				return vacio, err
			}
			peticion, err = domain.RehidratarPeticionCentro(datos)
			if err != nil {
				return vacio, err
			}
			peticion, err = peticion.Ratificar(actor, comando.VersionEsperada, comando.Motivo, s.reloj.Ahora())
			if err != nil {
				return vacio, err
			}
		}
		material = ports.MaterialPeticionCentro{Comando: comando, Actor: actor, Peticion: peticion.Datos()}
	}
	if err := material.Validar(); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recibo, err := s.repositorio.ConfirmarPeticion(ctx, material)
	// El puerto devuelve un recibo únicamente tras COMMIT. La cancelación
	// concurrente no borra ese hecho ni lo convierte en una segunda actuación.
	if recibo.ValidarPara(material) == nil {
		return recibo, nil
	}
	if err != nil {
		return vacio, err
	}
	return vacio, ports.ErrReciboPeticionCentroNoConfiable
}
