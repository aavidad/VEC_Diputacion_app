package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

// ServicioCorreoAvisos entrega el correo activo de «Mis correos» de una persona
// candidata a quien emite un llamamiento, sólo dentro de la llamada que lo usa.
type ServicioCorreoAvisos struct {
	registro  ports.RegistroCorreoAvisos
	protector ports.ProtectorDireccionCorreo
}

func NuevoServicioCorreoAvisos(registro ports.RegistroCorreoAvisos, protector ports.ProtectorDireccionCorreo) (*ServicioCorreoAvisos, error) {
	if registro == nil || protector == nil {
		return nil, ports.ErrCorreosNoDisponible
	}
	return &ServicioCorreoAvisos{registro: registro, protector: protector}, nil
}

// ConCorreoActivoAvisos pide una V3 fresca, consume la lectura y, si la
// persona tiene correo activo verificado desde el área personal externa,
// llama a usar con esa dirección. La transacción ya está confirmada cuando
// usar se ejecuta. Sin correo activo no llama a usar y devuelve Encontrado
// false. Cualquier fallo es ErrCorreosNoDisponible o ErrCorreosProhibido y
// garantiza que usar no se ha llamado. Quien usa la dirección anota por su
// cuenta el resultado de lo que haga con ella.
func (s *ServicioCorreoAvisos) ConCorreoActivoAvisos(ctx context.Context, orden ports.OrdenCorreoAvisos, sol ports.SolicitudCorreoAvisos, usar func(string)) (ports.ResultadoCorreoAvisos, error) {
	vacio := ports.ResultadoCorreoAvisos{}
	if s == nil || s.registro == nil || s.protector == nil || ctx == nil || usar == nil || orden.Proveedor == nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	m, err := canonico.MaterialCorreoAvisos(sol)
	if err != nil {
		return vacio, ports.ErrCorreosInvalidos
	}
	recurso, material, err := canonico.RecursoCorreoAvisos(m)
	if err != nil {
		return vacio, ports.ErrCorreosInvalidos
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacio, ports.ErrCorreosInvalidos
	}
	v3, err := orden.Proveedor.ProveerMaterialCorreoAvisos(ctx, m, material)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if errors.Is(err, ports.ErrCorreosProhibido) {
		return vacio, ports.ErrCorreosProhibido
	}
	if err != nil || v3.ValidarEstructura() != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	resumen := v3.ResumenCapacidad()
	if resumen.Operacion() != ports.AccionV3CorreoAvisosLlamamiento || resumen.AudienciaConsumo() != ports.AudienciaCorreoAvisosLlamamientoInterna ||
		resumen.EfectoRef() != m.BolsaRef || resumen.EfectoHuellaSHA256() != huella {
		return vacio, ports.ErrCorreosProhibido
	}
	lectura, err := s.registro.LeerCorreoActivoAvisos(ctx, material, v3)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, ports.ErrCorreosProhibido) {
			return vacio, ports.ErrCorreosProhibido
		}
		return vacio, ports.ErrCorreosNoDisponible
	}
	if !lectura.Encontrado {
		return ports.ResultadoCorreoAvisos{}, nil
	}
	if lectura.PersonaRef == "" || !canonico.CorreoRefValida(lectura.CorreoRef) || lectura.Sobre.CorreoRef != lectura.CorreoRef {
		return vacio, ports.ErrCorreosNoDisponible
	}
	// La dirección se valida entera antes de usarla: un sobre que no abre o
	// una dirección no admitida se tratan como indisponibilidad y usar no se
	// llama. La llamada a usar queda fuera del callback de descifrado.
	var direccion string
	err = s.protector.ConDireccionCorreoDescifrada(ctx, lectura.PersonaRef, lectura.Sobre, func(claro []byte) error {
		if !domain.DireccionCorreoValida(string(claro)) {
			return ports.ErrCorreosNoDisponible
		}
		direccion = string(claro)
		return nil
	})
	if err != nil || direccion == "" {
		return vacio, ports.ErrCorreosNoDisponible
	}
	usar(direccion)
	return ports.ResultadoCorreoAvisos{Encontrado: true, CorreoRef: lectura.CorreoRef}, nil
}
