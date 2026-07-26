package ports

import (
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type OrdenConsultaCuadroRRHH struct {
	bloqueoSerializacionConsultaRRHH
	contexto       ContextoConsultaRRHH
	capacidad      CapacidadConsultaRRHH
	solicitud      SolicitudCuadroRRHH
	consultaHuella string
	instante       time.Time
}

func NuevaOrdenConsultaCuadroRRHH(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	instante time.Time,
) (OrdenConsultaCuadroRRHH, error) {
	huella, err := huellaSolicitudCuadroRRHH(solicitud)
	if err != nil || capacidad.validaPara(
		contexto, DominioHuellaConsultaCuadroRRHH, huella,
		AccionConsultarCuadroRRHH, FinalidadConsultarCuadroRRHH, "", instante,
	) != nil {
		return OrdenConsultaCuadroRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return OrdenConsultaCuadroRRHH{
		contexto: contexto, capacidad: capacidad,
		solicitud: solicitud, consultaHuella: huella, instante: instante,
	}, nil
}

func (o OrdenConsultaCuadroRRHH) Contexto() ContextoConsultaRRHH {
	return o.contexto
}
func (o OrdenConsultaCuadroRRHH) Capacidad() CapacidadConsultaRRHH {
	return o.capacidad
}
func (o OrdenConsultaCuadroRRHH) Solicitud() SolicitudCuadroRRHH {
	return o.solicitud
}
func (o OrdenConsultaCuadroRRHH) ConsultaHuellaSHA256() string {
	return o.consultaHuella
}
func (o OrdenConsultaCuadroRRHH) Instante() time.Time { return o.instante }
func (o OrdenConsultaCuadroRRHH) ExportacionParaSQL() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	huella, err := huellaSolicitudCuadroRRHH(o.solicitud)
	if err != nil || huella != o.consultaHuella ||
		o.capacidad.validaPara(
			o.contexto, DominioHuellaConsultaCuadroRRHH, huella,
			AccionConsultarCuadroRRHH, FinalidadConsultarCuadroRRHH,
			"", o.instante,
		) != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrOrdenConsultaRRHHInvalida
	}
	return o.capacidad.material.exportacionParaSQL()
}

type OrdenConsultaDetalleRRHH struct {
	bloqueoSerializacionConsultaRRHH
	contexto       ContextoConsultaRRHH
	capacidad      CapacidadConsultaRRHH
	solicitud      SolicitudDetalleRRHH
	consultaHuella string
	instante       time.Time
}

func NuevaOrdenConsultaDetalleRRHH(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
	instante time.Time,
) (OrdenConsultaDetalleRRHH, error) {
	huella, err := huellaSolicitudDetalleRRHH(solicitud)
	if err != nil || capacidad.validaPara(
		contexto, DominioHuellaConsultaDetalleRRHH, huella,
		AccionConsultarDetalleRRHH, FinalidadConsultarDetalleRRHH,
		solicitud.expedienteRef, instante,
	) != nil {
		return OrdenConsultaDetalleRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return OrdenConsultaDetalleRRHH{
		contexto: contexto, capacidad: capacidad,
		solicitud: solicitud, consultaHuella: huella, instante: instante,
	}, nil
}

func (o OrdenConsultaDetalleRRHH) Contexto() ContextoConsultaRRHH {
	return o.contexto
}
func (o OrdenConsultaDetalleRRHH) Capacidad() CapacidadConsultaRRHH {
	return o.capacidad
}
func (o OrdenConsultaDetalleRRHH) Solicitud() SolicitudDetalleRRHH {
	return o.solicitud
}
func (o OrdenConsultaDetalleRRHH) ConsultaHuellaSHA256() string {
	return o.consultaHuella
}
func (o OrdenConsultaDetalleRRHH) Instante() time.Time { return o.instante }
func (o OrdenConsultaDetalleRRHH) ExportacionParaSQL() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	huella, err := huellaSolicitudDetalleRRHH(o.solicitud)
	if err != nil || huella != o.consultaHuella ||
		o.capacidad.validaPara(
			o.contexto, DominioHuellaConsultaDetalleRRHH, huella,
			AccionConsultarDetalleRRHH, FinalidadConsultarDetalleRRHH,
			o.solicitud.expedienteRef, o.instante,
		) != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrOrdenConsultaRRHHInvalida
	}
	return o.capacidad.material.exportacionParaSQL()
}
