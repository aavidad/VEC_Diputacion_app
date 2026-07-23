package ports

import (
	"context"
	"time"
)

// SolicitudesFuentesAnalisisO3 agrupa peticiones ya selladas por la
// infraestructura interna. No forma parte del DTO de usuario.
type SolicitudesFuentesAnalisisO3 struct {
	ValidacionRC SolicitudValidarRC
	CalculoCoste *SolicitudCalcularCoste
}

func (s SolicitudesFuentesAnalisisO3) ValidarPara(
	solicitud SolicitudPrepararArtefactoAnalisis,
) error {
	datosRC, errRC := s.ValidacionRC.Datos()
	if solicitud.Validar() != nil || errRC != nil ||
		datosRC.OrganizacionRef != solicitud.OrganizacionRef ||
		datosRC.ExpedienteRef != solicitud.ExpedienteRef ||
		datosRC.VersionExpediente != solicitud.VersionExpediente ||
		datosRC.Entrada.Referencia !=
			solicitud.DatosFuncionales.EntradaRC.Referencia ||
		datosRC.Entrada.HuellaSHA256 !=
			solicitud.DatosFuncionales.EntradaRC.HuellaSHA256 ||
		datosRC.SolicitadaEn.Before(solicitud.SolicitadaEn) {
		return ErrSolicitudArtefactoAnalisisInvalida
	}
	if s.CalculoCoste == nil {
		return nil
	}
	datosCoste, errCoste := s.CalculoCoste.Datos()
	if errCoste != nil ||
		datosCoste.OrganizacionRef != solicitud.OrganizacionRef ||
		datosCoste.ExpedienteRef != solicitud.ExpedienteRef ||
		datosCoste.VersionExpediente != solicitud.VersionExpediente ||
		datosCoste.CategoriaRef !=
			solicitud.DatosFuncionales.CategoriaRef ||
		datosCoste.GrupoSubgrupo !=
			solicitud.DatosFuncionales.GrupoSubgrupo ||
		datosCoste.ModalidadClave !=
			solicitud.DatosFuncionales.ModalidadClave ||
		datosCoste.CausaClave != solicitud.DatosFuncionales.CausaClave ||
		!datosCoste.Periodo.Inicio.Equal(
			solicitud.DatosFuncionales.Periodo.Inicio,
		) ||
		!datosCoste.Periodo.Fin.Equal(
			solicitud.DatosFuncionales.Periodo.Fin,
		) ||
		datosCoste.Jornada !=
			solicitud.DatosFuncionales.PorcentajeJornada ||
		datosCoste.SolicitadaEn.Before(solicitud.SolicitadaEn) {
		return ErrSolicitudArtefactoAnalisisInvalida
	}
	return nil
}

type PreparadorSolicitudesFuentesAnalisisO3 interface {
	PrepararSolicitudesFuentesAnalisisO3(
		context.Context,
		SolicitudPrepararArtefactoAnalisis,
	) (SolicitudesFuentesAnalisisO3, error)
}

// CapacidadPrepararArtefactoAnalisisO3 es la única acuñadora del artefacto.
// Se inyecta desde la composición interna con confianza y autoridades O3-03;
// ningún comando de cliente puede construirla ni aportar sus credenciales.
type CapacidadPrepararArtefactoAnalisisO3 struct {
	solicitudes PreparadorSolicitudesFuentesAnalisisO3
	fuenteRC    FuentePresupuestaria
	calculador  CalculadorCostePersonal
	verificador VerificadorRespuestaFuenteAnalisis
	publicador  VerificadorPublicacionMotivoFuenteAnalisis
	consumidor  ConsumidorRespuestaFuenteAnalisis
	confianza   ConfianzaAutoridadesFuenteAnalisis
	reloj       RelojFuenteAnalisis
}

func NuevaCapacidadPrepararArtefactoAnalisisO3(
	solicitudes PreparadorSolicitudesFuentesAnalisisO3,
	fuenteRC FuentePresupuestaria,
	calculador CalculadorCostePersonal,
	verificador VerificadorRespuestaFuenteAnalisis,
	publicador VerificadorPublicacionMotivoFuenteAnalisis,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
) (*CapacidadPrepararArtefactoAnalisisO3, error) {
	if dependenciaNulaFuenteAnalisis(solicitudes) ||
		dependenciaNulaFuenteAnalisis(fuenteRC) ||
		dependenciaNulaFuenteAnalisis(calculador) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(publicador) ||
		dependenciaNulaFuenteAnalisis(consumidor) ||
		dependenciaNulaFuenteAnalisis(reloj) ||
		confianza.organizacionRef == "" ||
		confianza.audiencia == "" {
		return nil, ErrSolicitudArtefactoAnalisisInvalida
	}
	return &CapacidadPrepararArtefactoAnalisisO3{
		solicitudes: solicitudes,
		fuenteRC:    fuenteRC,
		calculador:  calculador,
		verificador: verificador,
		publicador:  publicador,
		consumidor:  consumidor,
		confianza:   confianza,
		reloj:       reloj,
	}, nil
}

func (c *CapacidadPrepararArtefactoAnalisisO3) PrepararArtefactoAnalisis(
	ctx context.Context,
	solicitud SolicitudPrepararArtefactoAnalisis,
) (ArtefactoAnalisisPreparado, error) {
	if c == nil || ctx == nil || solicitud.Validar() != nil ||
		dependenciaNulaFuenteAnalisis(c.solicitudes) ||
		dependenciaNulaFuenteAnalisis(c.fuenteRC) ||
		dependenciaNulaFuenteAnalisis(c.calculador) ||
		dependenciaNulaFuenteAnalisis(c.verificador) ||
		dependenciaNulaFuenteAnalisis(c.publicador) ||
		dependenciaNulaFuenteAnalisis(c.consumidor) ||
		dependenciaNulaFuenteAnalisis(c.reloj) {
		return ArtefactoAnalisisPreparado{},
			ErrSolicitudArtefactoAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ArtefactoAnalisisPreparado{},
			errorDisponibilidadFuente(
				ErrArtefactoAnalisisNoDisponible,
				err,
			)
	}
	solicitudes, err := c.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(ctx, solicitud)
	if err != nil {
		return ArtefactoAnalisisPreparado{},
			errorDisponibilidadFuente(
				ErrArtefactoAnalisisNoDisponible,
				err,
			)
	}
	if solicitudes.ValidarPara(solicitud) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	evidenciaRC, err := VerificarValidacionRCConFuenteO3(
		ctx,
		c.fuenteRC,
		c.verificador,
		c.publicador,
		c.confianza,
		c.reloj,
		solicitudes.ValidacionRC,
	)
	if err != nil {
		return ArtefactoAnalisisPreparado{}, err
	}
	var evidenciaCoste EvidenciaCalculoCosteVerificadaO3
	if solicitudes.CalculoCoste != nil {
		evidenciaCoste, err = VerificarCalculoCosteConFuenteO3(
			ctx,
			c.calculador,
			c.verificador,
			c.confianza,
			c.reloj,
			*solicitudes.CalculoCoste,
		)
		if err != nil {
			return ArtefactoAnalisisPreparado{}, err
		}
	}
	return c.consumirYAcuñar(
		ctx,
		solicitud,
		evidenciaRC,
		evidenciaCoste,
	)
}

func (c *CapacidadPrepararArtefactoAnalisisO3) consumirYAcuñar(
	ctx context.Context,
	solicitud SolicitudPrepararArtefactoAnalisis,
	evidenciaRC EvidenciaValidacionRCVerificadaO3,
	evidenciaCoste EvidenciaCalculoCosteVerificadaO3,
) (ArtefactoAnalisisPreparado, error) {
	comprobadaEn := c.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return ArtefactoAnalisisPreparado{},
			errorDisponibilidadFuente(
				ErrArtefactoAnalisisNoDisponible,
				err,
			)
	}
	if evidenciaRC.validarEn(comprobadaEn) != nil ||
		(evidenciaCoste.datos != nil &&
			evidenciaCoste.validarEn(comprobadaEn) != nil) ||
		c.revalidarAutoridades(
			ctx,
			evidenciaRC,
			evidenciaCoste,
			comprobadaEn,
		) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	reciboRC, err := consumirEvidenciaFuenteAnalisisO3(
		ctx,
		c.consumidor,
		evidenciaRC.datos.orden,
	)
	if err != nil {
		return ArtefactoAnalisisPreparado{}, err
	}
	var reciboCoste *ReciboConsumoRespuestaFuenteAnalisis
	if evidenciaCoste.datos != nil {
		recibo, errConsumo := consumirEvidenciaFuenteAnalisisO3(
			ctx,
			c.consumidor,
			evidenciaCoste.datos.orden,
		)
		if errConsumo != nil {
			return ArtefactoAnalisisPreparado{}, errConsumo
		}
		reciboCoste = &recibo
	}
	preparadoEn := c.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return ArtefactoAnalisisPreparado{},
			errorDisponibilidadFuente(
				ErrArtefactoAnalisisNoDisponible,
				err,
			)
	}
	if evidenciaRC.validarEn(preparadoEn) != nil ||
		(evidenciaCoste.datos != nil &&
			evidenciaCoste.validarEn(preparadoEn) != nil) {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	return nuevoArtefactoAnalisisPreparadoDesdeFuentesO3(
		solicitud,
		evidenciaRC,
		reciboRC,
		evidenciaCoste,
		reciboCoste,
		preparadoEn,
	)
}

func (c *CapacidadPrepararArtefactoAnalisisO3) revalidarAutoridades(
	ctx context.Context,
	rc EvidenciaValidacionRCVerificadaO3,
	coste EvidenciaCalculoCosteVerificadaO3,
	comprobadaEn time.Time,
) error {
	if rc.datos == nil {
		return ErrArtefactoAnalisisNoConfiable
	}
	datosSolicitudRC, err := rc.datos.solicitud.Datos()
	materialRC := materialDesafioSolicitudFuenteAnalisis(
		rc.datos.solicitud.datosCanonicos(),
		datosSolicitudRC.HuellaPeticionHMAC,
	)
	if err != nil || len(materialRC) == 0 {
		return ErrArtefactoAnalisisNoConfiable
	}
	fuenteRC, err := presentarYVerificarAutoridadFuenteAnalisis(
		ctx, c.fuenteRC, c.confianza, materialRC,
		RolFuentePresupuestaria, comprobadaEn,
	)
	if err != nil {
		return err
	}
	verificadorRC, err := presentarYVerificarAutoridadFuenteAnalisis(
		ctx, c.verificador, c.confianza, materialRC,
		RolVerificadorRespuesta, comprobadaEn,
	)
	if err != nil {
		return err
	}
	publicadorRC, err := presentarYVerificarAutoridadFuenteAnalisis(
		ctx, c.publicador, c.confianza, materialRC,
		RolPublicadorCatalogo, comprobadaEn,
	)
	if err != nil ||
		!identidadesAutoridadFuenteAnalisisIguales(
			fuenteRC,
			rc.datos.identidadFuente,
		) ||
		!identidadesAutoridadFuenteAnalisisIguales(
			verificadorRC,
			rc.datos.identidadVerificador,
		) ||
		!identidadesAutoridadFuenteAnalisisIguales(
			publicadorRC,
			rc.datos.identidadPublicador,
		) {
		return ErrArtefactoAnalisisNoConfiable
	}
	if coste.datos == nil {
		return nil
	}
	datosSolicitudCoste, err := coste.datos.solicitud.Datos()
	materialCoste := materialDesafioSolicitudFuenteAnalisis(
		coste.datos.solicitud.datosCanonicos(),
		datosSolicitudCoste.HuellaPeticionHMAC,
	)
	if err != nil || len(materialCoste) == 0 {
		return ErrArtefactoAnalisisNoConfiable
	}
	fuenteCoste, err := presentarYVerificarAutoridadFuenteAnalisis(
		ctx, c.calculador, c.confianza, materialCoste,
		RolCalculadorCoste, comprobadaEn,
	)
	if err != nil {
		return err
	}
	verificadorCoste, err := presentarYVerificarAutoridadFuenteAnalisis(
		ctx, c.verificador, c.confianza, materialCoste,
		RolVerificadorRespuesta, comprobadaEn,
	)
	if err != nil ||
		!identidadesAutoridadFuenteAnalisisIguales(
			fuenteCoste,
			coste.datos.identidadFuente,
		) ||
		!identidadesAutoridadFuenteAnalisisIguales(
			verificadorCoste,
			coste.datos.identidadVerificador,
		) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

var _ PreparadorArtefactoAnalisisO3 = (*CapacidadPrepararArtefactoAnalisisO3)(nil)
