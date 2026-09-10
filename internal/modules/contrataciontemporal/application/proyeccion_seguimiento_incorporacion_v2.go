package application

import (
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaVistaSeguimientoIncorporacionV2 = "vec.contratacion-temporal.seguimiento-incorporacion.v2"
	esquemaReciboIncorporacionEjercicioV2  = "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2"
	alcanceOriginalIncorporacion           = "original_incorporacion"
)

var ErrProyeccionSeguimientoIncorporacionV2Invalida = errors.New(
	"contratacion temporal: proyeccion de seguimiento de incorporacion V2 invalida",
)

// ProyectarSeguimientoIncorporacionV2 solo publica la evidencia ORIGINAL de la
// incorporación. Restaura antes de leer para no convertir un estado aportado
// arbitrariamente en una lectura válida ni atribuirle autoridad vigente.
func ProyectarSeguimientoIncorporacionV2(
	recibo ports.ReciboIncorporacionAplicacionV2,
	publicacion domain.PublicacionDefinicionSeguimiento,
	estado domain.EstadoPersistidoSeguimiento,
) (ports.VistaSeguimientoIncorporacionV2, error) {
	definicion, err := domain.RestaurarDefinicionSeguimiento(publicacion)
	if err != nil {
		return ports.VistaSeguimientoIncorporacionV2{}, ErrProyeccionSeguimientoIncorporacionV2Invalida
	}
	seguimiento, err := domain.RehidratarSeguimiento(definicion, estado)
	if err != nil {
		return ports.VistaSeguimientoIncorporacionV2{}, ErrProyeccionSeguimientoIncorporacionV2Invalida
	}
	original := seguimiento.Estado()
	if len(original.Actuaciones) == 0 {
		return ports.VistaSeguimientoIncorporacionV2{}, ErrProyeccionSeguimientoIncorporacionV2Invalida
	}
	hito := original.Actuaciones[len(original.Actuaciones)-1]
	if !reciboIncorporacionEjercicioV2Valido(recibo) ||
		recibo.ExpedienteRef != original.ExpedienteRef ||
		recibo.RelacionRef != original.RelacionRef ||
		recibo.SeguimientoRef != original.Referencia ||
		recibo.VersionSeguimientoResultante != original.Version ||
		recibo.Periodo != original.PeriodoPrevisto ||
		recibo.ActuacionRef != hito.ActuacionRef ||
		recibo.ReciboRef != hito.ReciboRef ||
		!recibo.RegistradaEn.Equal(hito.RegistradaEn) ||
		hito.Clase != domain.TransicionOrdinaria ||
		hito.TransicionClave != ports.TransicionConfirmarIncorporacion ||
		hito.Periodo == nil || *hito.Periodo != recibo.Periodo ||
		!hito.EfectivoEn.Equal(recibo.Periodo.Desde) {
		return ports.VistaSeguimientoIncorporacionV2{}, ErrProyeccionSeguimientoIncorporacionV2Invalida
	}

	actuaciones := make([]ports.ActuacionVisibleSeguimientoIncorporacionV2, len(original.Actuaciones))
	for i, actuacion := range original.Actuaciones {
		actuaciones[i] = ports.ActuacionVisibleSeguimientoIncorporacionV2{
			ActuacionRef: actuacion.ActuacionRef, TransicionClave: actuacion.TransicionClave,
			EstadoOrigen: actuacion.EstadoOrigen, EstadoDestino: actuacion.EstadoDestino,
			EfectivoEn: actuacion.EfectivoEn, RegistradaEn: actuacion.RegistradaEn,
			Documentos: append([]domain.DocumentoSeguimiento{}, actuacion.Documentos...),
		}
	}
	return ports.VistaSeguimientoIncorporacionV2{
		Esquema: esquemaVistaSeguimientoIncorporacionV2, Alcance: alcanceOriginalIncorporacion,
		ExpedienteRef: original.ExpedienteRef, VersionExpediente: recibo.VersionActualExpediente,
		ReciboIncorporacionRef: recibo.ReciboRef, SeguimientoRef: original.Referencia,
		VersionSeguimiento: original.Version, EstadoClave: original.EstadoActual,
		Periodo: original.PeriodoPrevisto, RegistradoEn: hito.RegistradaEn,
		Actuaciones: actuaciones, EjercicioSintetico: recibo.EjercicioSintetico,
		FirmaOficial: recibo.FirmaOficial, EficaciaAdministrativa: recibo.EficaciaAdministrativa,
	}, nil
}

func reciboIncorporacionEjercicioV2Valido(recibo ports.ReciboIncorporacionAplicacionV2) bool {
	if recibo.Esquema != esquemaReciboIncorporacionEjercicioV2 ||
		!recibo.EjercicioSintetico || recibo.FirmaOficial || recibo.EficaciaAdministrativa ||
		!domain.InstanteUTCCanonico(recibo.RegistradaEn) || recibo.Periodo.Validar() != nil ||
		!versionSeguimientoIncorporacionValida(recibo.VersionSolicitudPersonal, false) ||
		!versionSeguimientoIncorporacionValida(recibo.VersionActualExpediente, false) ||
		!versionSeguimientoIncorporacionValida(recibo.VersionSeguimientoAnterior, true) ||
		!versionSeguimientoIncorporacionValida(recibo.VersionSeguimientoResultante, false) ||
		recibo.VersionSeguimientoResultante != recibo.VersionSeguimientoAnterior+1 {
		return false
	}
	for _, referencia := range []string{
		recibo.ExpedienteRef, recibo.SolicitudPersonalRef, recibo.RelacionRef, recibo.ReciboRef,
		recibo.ActuacionRef, recibo.SeguimientoRef, recibo.AuditoriaRef, recibo.OutboxRef,
	} {
		if !domain.ReferenciaOpacaValida(referencia) {
			return false
		}
	}
	return true
}

func versionSeguimientoIncorporacionValida(version uint64, admiteCero bool) bool {
	return (admiteCero || version > 0) && version <= ports.MaximoEnteroSeguroOperacionAnalisis
}
