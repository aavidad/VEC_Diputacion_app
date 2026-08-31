package ginpixapi

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// prepararOperacionRecuperable abre unicamente los sobres autenticos de
// O7-06A. Carga conserva el resultado inmutable O7-03; Orden y
// ReciboIncorporacion conservan la autoridad y el recibo O7-02 completos. Los
// Datos copiables solo se usan al final para cotejar todas las ligaduras.
func prepararOperacionRecuperable(
	solicitud ports.SolicitudOperacionGINPIX,
) (Preparacion, error) {
	if solicitud.Validar() != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	datos, errDatos := solicitud.Datos()
	orden, errOrden := solicitud.Orden()
	incorporacion, errIncorporacion := solicitud.ReciboIncorporacion()
	carga, errCarga := solicitud.Carga()
	if errDatos != nil || errOrden != nil || errIncorporacion != nil || errCarga != nil ||
		incorporacion.ValidarPara(orden) != nil || carga.Validar() != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}

	exportacion, err := ginpixfichero.PrepararExportacion(carga)
	if err != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	cuerpo, errCuerpo := exportacion.Contenido()
	huellaCuerpo, errHuella := exportacion.HuellaSHA256()
	metadatosFichero, errMetadatos := exportacion.Metadatos()
	if errCuerpo != nil || errHuella != nil || errMetadatos != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	metadatos := MetadatosOperacion{
		VersionExpediente:      metadatosFichero.VersionExpediente,
		ExpedienteRef:          metadatosFichero.ExpedienteRef,
		IncorporacionRef:       metadatosFichero.IncorporacionRef,
		ProcedenciaModeloRef:   metadatosFichero.ProcedenciaModeloRef,
		CorrelacionRef:         metadatosFichero.CorrelacionRef,
		IdempotenciaRef:        metadatosFichero.IdempotenciaRef,
		ModeloHuellaSHA256:     datos.ModeloHuellaSHA256,
		MapeoRef:               metadatosFichero.MapeoRef,
		MapeoVersion:           metadatosFichero.MapeoVersion,
		ProcedenciaMapeoRef:    metadatosFichero.ProcedenciaMapeoRef,
		MapeoHuellaSHA256:      datos.MapeoHuellaSHA256,
		CargaHuellaSHA256:      datos.CargaHuellaSHA256,
		CuerpoHuellaSHA256:     huellaCuerpo,
		ReciboIncorporacionRef: incorporacion.ReciboRef,
		ResultadoPersonalRef:   incorporacion.ResultadoPersonalRef,
		ReciboPersonalRef:      incorporacion.ReciboPersonalRef,
	}
	if !metadatosOperacionLigadosASolicitud(metadatos, datos) {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	preparacion := Preparacion{datos: &datosPreparacion{
		cuerpo: append([]byte(nil), cuerpo...), metadatos: metadatos,
		orden: orden, incorporacion: clonarReciboConfirmacionIncorporacion(incorporacion),
	}}
	if preparacion.Validar() != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	return preparacion, nil
}

func metadatosOperacionLigadosASolicitud(
	metadatos MetadatosOperacion,
	datos ports.DatosOperacionGINPIX,
) bool {
	return metadatos.VersionExpediente == datos.VersionExpediente &&
		metadatos.ExpedienteRef == datos.ExpedienteRef &&
		metadatos.IncorporacionRef == datos.IncorporacionRef &&
		metadatos.ReciboIncorporacionRef == datos.ReciboIncorporacionRef &&
		metadatos.ResultadoPersonalRef == datos.ResultadoPersonalRef &&
		metadatos.ReciboPersonalRef == datos.ReciboPersonalRef &&
		metadatos.CorrelacionRef == datos.CorrelacionRef &&
		metadatos.IdempotenciaRef == datos.IdempotenciaRef &&
		metadatos.ProcedenciaModeloRef == datos.ProcedenciaModeloRef &&
		metadatos.ModeloHuellaSHA256 == datos.ModeloHuellaSHA256 &&
		metadatos.MapeoRef == datos.MapeoRef &&
		metadatos.MapeoVersion == datos.MapeoVersion &&
		metadatos.ProcedenciaMapeoRef == datos.ProcedenciaMapeoRef &&
		metadatos.MapeoHuellaSHA256 == datos.MapeoHuellaSHA256 &&
		metadatos.CargaHuellaSHA256 == datos.CargaHuellaSHA256
}

func datosReciboExternoCompletos(recibo ReciboExterno) (DatosReciboExterno, bool) {
	datos, err := recibo.Datos()
	return datos, err == nil
}

func traducirReciboOperacionRecuperable(
	recibo ReciboExterno,
	solicitud ports.SolicitudOperacionGINPIX,
) (ports.ReciboExternoOperacionGINPIX, error) {
	datosRecibo, errRecibo := recibo.Datos()
	datosSolicitud, errSolicitud := solicitud.Datos()
	if errRecibo != nil || errSolicitud != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrReciboExternoOperacionGINPIXInvalido
	}
	traducido := ports.ReciboExternoOperacionGINPIX{
		ReciboExternoRef:             datosRecibo.ReciboExternoRef,
		EvidenciaExternaRef:          datosRecibo.EvidenciaExternaRef,
		EvidenciaExternaHuellaSHA256: datosRecibo.EvidenciaExternaHuellaSHA256,
		ClaveOperacionRef:            datosSolicitud.ClaveOperacionRef,
		VersionExpediente:            datosRecibo.VersionExpediente,
		ExpedienteRef:                datosRecibo.ExpedienteRef,
		IncorporacionRef:             datosRecibo.IncorporacionRef,
		ReciboIncorporacionRef:       datosRecibo.ReciboIncorporacionRef,
		ResultadoPersonalRef:         datosRecibo.ResultadoPersonalRef,
		ReciboPersonalRef:            datosRecibo.ReciboPersonalRef,
		CorrelacionRef:               datosRecibo.CorrelacionRef,
		IdempotenciaRef:              datosRecibo.IdempotenciaRef,
		ModeloHuellaSHA256:           datosRecibo.ModeloHuellaSHA256,
		MapeoRef:                     datosRecibo.MapeoRef,
		MapeoVersion:                 datosRecibo.MapeoVersion,
		MapeoHuellaSHA256:            datosRecibo.MapeoHuellaSHA256,
		CargaHuellaSHA256:            datosRecibo.CargaHuellaSHA256,
	}
	if traducido.ValidarPara(solicitud) != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrReciboExternoOperacionGINPIXInvalido
	}
	return traducido, nil
}

func errorEmisionOperacionRecuperable(err error, hayReciboCompleto bool) error {
	base := ports.ErrEmisionOperacionGINPIXNoIniciada
	if hayReciboCompleto || errors.Is(err, ErrOperacionAPIGINPIXIndeterminada) {
		base = ports.ErrEmisionOperacionGINPIXIndeterminada
	}
	return errorOperacionConCausalidad(base, err)
}

func errorConsultaOperacionRecuperable(err error) error {
	return errorOperacionConCausalidad(ports.ErrConsultaOperacionGINPIXNoDisponible, err)
}

func errorOperacionConCausalidad(base, err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return errors.Join(base, context.Canceled)
	case errors.Is(err, context.DeadlineExceeded):
		return errors.Join(base, context.DeadlineExceeded)
	default:
		return base
	}
}
