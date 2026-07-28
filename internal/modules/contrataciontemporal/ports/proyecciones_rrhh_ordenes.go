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
	filtrosHuella  string
	instante       time.Time
}

func NuevaOrdenConsultaCuadroRRHH(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	instante time.Time,
) (OrdenConsultaCuadroRRHH, error) {
	huella, err := huellaSolicitudCuadroRRHH(solicitud)
	filtrosHuella, errFiltros := huellaFiltrosCuadroRRHH(solicitud)
	if err != nil || errFiltros != nil || capacidad.validaPara(
		contexto, DominioHuellaConsultaCuadroRRHH, huella,
		AccionConsultarCuadroRRHH, FinalidadConsultarCuadroRRHH, "", instante,
	) != nil {
		return OrdenConsultaCuadroRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return OrdenConsultaCuadroRRHH{
		contexto: contexto, capacidad: capacidad,
		solicitud: solicitud, consultaHuella: huella,
		filtrosHuella: filtrosHuella, instante: instante,
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
func (o OrdenConsultaCuadroRRHH) FiltrosHuellaSHA256() string {
	return o.filtrosHuella
}
func (o OrdenConsultaCuadroRRHH) Instante() time.Time { return o.instante }

func (o OrdenConsultaCuadroRRHH) canonesParaExportacionSQL() (
	exportacionCanonicaRRHH,
	exportacionCanonicaRRHH,
	error,
) {
	consulta, err := canonSolicitudCuadroRRHH(o.solicitud)
	familia, errFamilia := canonFiltrosSolicitudCuadroRRHH(o.solicitud)
	if err != nil || errFamilia != nil ||
		consulta.huellaSHA256 != o.consultaHuella ||
		familia.huellaSHA256 != o.filtrosHuella ||
		o.capacidad.validaPara(
			o.contexto, DominioHuellaConsultaCuadroRRHH,
			consulta.huellaSHA256,
			AccionConsultarCuadroRRHH, FinalidadConsultarCuadroRRHH,
			"", o.instante,
		) != nil {
		return exportacionCanonicaRRHH{}, exportacionCanonicaRRHH{},
			ErrOrdenConsultaRRHHInvalida
	}
	return consulta, familia, nil
}

func (o OrdenConsultaCuadroRRHH) ExportacionParaSQL() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	if _, _, err := o.canonesParaExportacionSQL(); err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrOrdenConsultaRRHHInvalida
	}
	return o.capacidad.material.exportacionParaSQL()
}

func (o OrdenConsultaCuadroRRHH) ExportarConsultaCanonicaParaSQL() (
	ExportacionCanonicaConsultaCuadroRRHH,
	error,
) {
	consulta, _, err := o.canonesParaExportacionSQL()
	if err != nil {
		return ExportacionCanonicaConsultaCuadroRRHH{},
			ErrOrdenConsultaRRHHInvalida
	}
	return ExportacionCanonicaConsultaCuadroRRHH{consulta}, nil
}

func (o OrdenConsultaCuadroRRHH) ExportarFamiliaCanonicaParaSQL() (
	ExportacionCanonicaFamiliaCuadroRRHH,
	error,
) {
	_, familia, err := o.canonesParaExportacionSQL()
	if err != nil {
		return ExportacionCanonicaFamiliaCuadroRRHH{},
			ErrOrdenConsultaRRHHInvalida
	}
	return ExportacionCanonicaFamiliaCuadroRRHH{familia}, nil
}

func (o OrdenConsultaCuadroRRHH) ExportarAlcanceCanonicoParaSQL() (
	ExportacionCanonicaAlcanceRRHH,
	error,
) {
	if _, _, err := o.canonesParaExportacionSQL(); err != nil {
		return ExportacionCanonicaAlcanceRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	canon, err := canonAlcanceCapacidadRRHH(o.capacidad)
	if err != nil {
		return ExportacionCanonicaAlcanceRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return ExportacionCanonicaAlcanceRRHH{canon}, nil
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

func (o OrdenConsultaDetalleRRHH) canonParaExportacionSQL() (
	exportacionCanonicaRRHH,
	error,
) {
	consulta, err := canonSolicitudDetalleRRHH(o.solicitud)
	if err != nil || consulta.huellaSHA256 != o.consultaHuella ||
		o.capacidad.validaPara(
			o.contexto, DominioHuellaConsultaDetalleRRHH,
			consulta.huellaSHA256,
			AccionConsultarDetalleRRHH, FinalidadConsultarDetalleRRHH,
			o.solicitud.expedienteRef, o.instante,
		) != nil {
		return exportacionCanonicaRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return consulta, nil
}

func (o OrdenConsultaDetalleRRHH) ExportacionParaSQL() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	if _, err := o.canonParaExportacionSQL(); err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrOrdenConsultaRRHHInvalida
	}
	return o.capacidad.material.exportacionParaSQL()
}

func (o OrdenConsultaDetalleRRHH) ExportarConsultaCanonicaParaSQL() (
	ExportacionCanonicaConsultaDetalleRRHH,
	error,
) {
	consulta, err := o.canonParaExportacionSQL()
	if err != nil {
		return ExportacionCanonicaConsultaDetalleRRHH{},
			ErrOrdenConsultaRRHHInvalida
	}
	return ExportacionCanonicaConsultaDetalleRRHH{consulta}, nil
}

func (o OrdenConsultaDetalleRRHH) ExportarAlcanceCanonicoParaSQL() (
	ExportacionCanonicaAlcanceRRHH,
	error,
) {
	if _, err := o.canonParaExportacionSQL(); err != nil {
		return ExportacionCanonicaAlcanceRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	canon, err := canonAlcanceCapacidadRRHH(o.capacidad)
	if err != nil {
		return ExportacionCanonicaAlcanceRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return ExportacionCanonicaAlcanceRRHH{canon}, nil
}
