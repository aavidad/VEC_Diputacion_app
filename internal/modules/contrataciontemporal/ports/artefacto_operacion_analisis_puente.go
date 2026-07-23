package ports

import (
	"context"
	"errors"
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
// Su constructor exportado existe porque la raíz de composición vive en otro
// paquete Go, pero solo debe invocarse desde esa composición interna confiable.
// El modificador internal impide importarlo desde otros módulos; dentro de este
// módulo la revisión de dependencias debe impedir llamadas desde transportes.
// Ningún DTO, cabecera, cookie ni comando de cliente puede aportar confianza,
// raíces o credenciales.
type CapacidadPrepararArtefactoAnalisisO3 struct {
	solicitudes PreparadorSolicitudesFuentesAnalisisO3
	fuenteRC    FuentePresupuestaria
	calculador  CalculadorCostePersonal
	verificador VerificadorRespuestaFuenteAnalisis
	publicador  VerificadorPublicacionMotivoFuenteAnalisis
	consumidor  ConsumidorConjuntoFuentesAnalisisO3
	confianza   ConfianzaAutoridadesFuenteAnalisis
	reloj       RelojFuenteAnalisis
}

func NuevaCapacidadPrepararArtefactoAnalisisO3ParaComposicionInterna(
	solicitudes PreparadorSolicitudesFuentesAnalisisO3,
	fuenteRC FuentePresupuestaria,
	calculador CalculadorCostePersonal,
	verificador VerificadorRespuestaFuenteAnalisis,
	publicador VerificadorPublicacionMotivoFuenteAnalisis,
	consumidor ConsumidorConjuntoFuentesAnalisisO3,
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
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	solicitudes, err := c.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(operacion, solicitud)
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
		operacion,
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
			operacion,
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
	comprobadaEn := c.reloj.Ahora()
	if err := operacion.Err(); err != nil {
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
			operacion,
			evidenciaRC,
			evidenciaCoste,
			comprobadaEn,
		) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	preparadoEn := c.reloj.Ahora()
	if err := operacion.Err(); err != nil {
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
	return nuevoArtefactoAnalisisVerificadoDesdeFuentesO3(
		solicitud,
		evidenciaRC,
		evidenciaCoste,
		preparadoEn,
	)
}

func (c *CapacidadPrepararArtefactoAnalisisO3) ConsumirArtefactoAnalisisO3(
	ctx context.Context,
	solicitud SolicitudPrepararArtefactoAnalisis,
	artefacto ArtefactoAnalisisPreparado,
) (ArtefactoAnalisisPreparado, error) {
	if c == nil || ctx == nil ||
		validarArtefactoAnalisisPreparado(solicitud, artefacto) != nil ||
		artefacto.pruebas.reciboConjunto != nil ||
		dependenciaNulaFuenteAnalisis(c.consumidor) ||
		dependenciaNulaFuenteAnalisis(c.reloj) {
		return ArtefactoAnalisisPreparado{},
			ErrSolicitudArtefactoAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	comprobadaEn := c.reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ArtefactoAnalisisPreparado{},
			errorDisponibilidadFuente(
				ErrArtefactoAnalisisNoDisponible,
				err,
			)
	}
	rc := artefacto.pruebas.evidenciaRC
	coste := artefacto.pruebas.evidenciaCoste
	if rc.validarEn(comprobadaEn) != nil ||
		(coste.datos != nil && coste.validarEn(comprobadaEn) != nil) ||
		c.revalidarAutoridades(
			operacion,
			rc,
			coste,
			comprobadaEn,
		) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	recibo, err := c.consumidor.ConsumirConjuntoFuentesAnalisisO3(
		operacion,
		artefacto.pruebas.ordenConjunto,
	)
	if err != nil {
		if errContexto := operacion.Err(); errContexto != nil {
			return ArtefactoAnalisisPreparado{},
				errorDisponibilidadFuente(
					ErrConsumoFuenteAnalisisNoDisponible,
					errContexto,
				)
		}
		if errors.Is(err, ErrConjuntoFuentesAnalisisYaConsumido) {
			return ArtefactoAnalisisPreparado{},
				ErrConjuntoFuentesAnalisisYaConsumido
		}
		return ArtefactoAnalisisPreparado{},
			errorDisponibilidadFuente(
				ErrConsumoFuenteAnalisisNoDisponible,
				err,
			)
	}
	if recibo.ValidarPara(artefacto.pruebas.ordenConjunto) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	consumidaEn := c.reloj.Ahora()
	if rc.validarEn(consumidaEn) != nil ||
		(coste.datos != nil && coste.validarEn(consumidaEn) != nil) {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	return artefactoAnalisisConsumidoDesdeReciboConjuntoO3(
		solicitud,
		artefacto,
		recibo,
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
