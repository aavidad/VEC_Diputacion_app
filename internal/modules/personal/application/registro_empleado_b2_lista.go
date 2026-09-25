package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

// ServicioEmpleadosRegistroB2 entrega a RRHH la lista de empleados del
// organismo para elegir una ficha. Cada página exige su concesión V3.
type ServicioEmpleadosRegistroB2 struct {
	autorizador ports.ProveedorAutorizacionRegistroEmpleadoB2
	repositorio ports.RepositorioEmpleadosRegistroB2
}

func NuevoServicioEmpleadosRegistroB2(a ports.ProveedorAutorizacionRegistroEmpleadoB2, r ports.RepositorioEmpleadosRegistroB2) (*ServicioEmpleadosRegistroB2, error) {
	if nulo(a) || nulo(r) {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &ServicioEmpleadosRegistroB2{a, r}, nil
}

func (s *ServicioEmpleadosRegistroB2) ConsultarEmpleados(ctx context.Context, solicitud domain.SolicitudEmpleadosB2) (ports.ResultadoEmpleadosB2, error) {
	var vacio ports.ResultadoEmpleadosB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialEmpleadosB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	autorizacion, err := s.autorizador.AutorizarConsultaRegistroEmpleadoB2(ctx, material)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionRegistroB2Valida(material, autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.ListarEmpleadosRRHH(ctx, ports.OrdenEmpleadosB2{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if resultado.Pagina.ValidarPara(material) != nil || !evidenciaRegistroB2Valida(material, autorizacion, resultado.Evidencia) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}
