package ports

import (
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type pruebasArtefactoAnalisisO3 struct {
	evidenciaRC    EvidenciaValidacionRCVerificadaO3
	evidenciaCoste EvidenciaCalculoCosteVerificadaO3
	ordenConjunto  OrdenConsumoConjuntoFuentesAnalisisO3
	reciboConjunto *ReciboConsumoConjuntoFuentesAnalisisO3
}

// PruebasArtefactoAnalisisO3 transporta a O3-04 las respuestas, las
// confirmaciones y las órdenes exactas ya consumidas. No permite reconstruir
// ni serializar un ArtefactoAnalisisPreparado.
type PruebasArtefactoAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	SolicitudRC             SolicitudValidarRC
	ResultadoRC             ResultadoValidacionRC
	ConfirmacionRC          ConfirmacionRespuestaFuenteAnalisis
	ConfirmacionPublicacion *ConfirmacionPublicacionMotivoFuenteAnalisis
	OrdenConsumoRC          OrdenConsumoRespuestaFuenteAnalisis
	ReciboConsumoRC         ReciboConsumoRespuestaFuenteAnalisis
	SolicitudCoste          *SolicitudCalcularCoste
	ResultadoCoste          *ResultadoCalculoCoste
	ConfirmacionCoste       *ConfirmacionRespuestaFuenteAnalisis
	OrdenConsumoCoste       *OrdenConsumoRespuestaFuenteAnalisis
	ReciboConsumoCoste      *ReciboConsumoRespuestaFuenteAnalisis
	OrdenConsumoConjunto    OrdenConsumoConjuntoFuentesAnalisisO3
	ReciboConsumoConjunto   *ReciboConsumoConjuntoFuentesAnalisisO3
}

func nuevoArtefactoAnalisisVerificadoDesdeFuentesO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
	evidenciaRC EvidenciaValidacionRCVerificadaO3,
	evidenciaCoste EvidenciaCalculoCosteVerificadaO3,
	preparadoEn time.Time,
) (ArtefactoAnalisisPreparado, error) {
	ordenConjunto, err := nuevaOrdenConsumoConjuntoFuentesAnalisisO3(
		solicitud,
		evidenciaRC,
		evidenciaCoste,
	)
	if err != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	pruebas := pruebasArtefactoAnalisisO3{
		evidenciaRC:    evidenciaRC,
		evidenciaCoste: evidenciaCoste,
		ordenConjunto:  ordenConjunto,
	}
	datos, err := datosArtefactoAnalisisDesdePruebasO3(
		solicitud,
		pruebas,
		preparadoEn,
	)
	if err != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	artefacto := ArtefactoAnalisisPreparado{
		datos:   &datos,
		pruebas: &pruebas,
	}
	if validarArtefactoAnalisisPreparado(solicitud, artefacto) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	return artefacto, nil
}

func artefactoAnalisisConsumidoDesdeReciboConjuntoO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
	artefacto ArtefactoAnalisisPreparado,
	recibo ReciboConsumoConjuntoFuentesAnalisisO3,
) (ArtefactoAnalisisPreparado, error) {
	if validarArtefactoAnalisisPreparado(solicitud, artefacto) != nil ||
		artefacto.pruebas.reciboConjunto != nil ||
		recibo.ValidarPara(artefacto.pruebas.ordenConjunto) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	pruebas := *artefacto.pruebas
	reciboClonado := clonarReciboConsumoConjuntoFuentesAnalisisO3(recibo)
	pruebas.reciboConjunto = &reciboClonado
	datos, err := datosArtefactoAnalisisDesdePruebasO3(
		solicitud,
		pruebas,
		artefacto.datos.PreparadoEn,
	)
	if err != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	consumido := ArtefactoAnalisisPreparado{
		datos:   &datos,
		pruebas: &pruebas,
	}
	if validarArtefactoAnalisisConsumidoO3(
		solicitud,
		consumido,
	) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	return consumido, nil
}

func validarArtefactoAnalisisPreparado(
	solicitud SolicitudPrepararArtefactoAnalisis,
	artefacto ArtefactoAnalisisPreparado,
) error {
	if artefacto.datos == nil || artefacto.pruebas == nil ||
		validarDatosArtefactoAnalisis(
			solicitud,
			*artefacto.datos,
		) != nil {
		return ErrArtefactoAnalisisNoConfiable
	}
	esperados, err := datosArtefactoAnalisisDesdePruebasO3(
		solicitud,
		*artefacto.pruebas,
		artefacto.datos.PreparadoEn,
	)
	if err != nil || !reflect.DeepEqual(
		esperados,
		*artefacto.datos,
	) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func validarArtefactoAnalisisConsumidoO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
	artefacto ArtefactoAnalisisPreparado,
) error {
	if validarArtefactoAnalisisPreparado(solicitud, artefacto) != nil ||
		artefacto.pruebas.reciboConjunto == nil ||
		artefacto.pruebas.reciboConjunto.ValidarPara(
			artefacto.pruebas.ordenConjunto,
		) != nil {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func (a ArtefactoAnalisisPreparado) PruebasParaO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
) (PruebasArtefactoAnalisisO3, error) {
	if validarArtefactoAnalisisConsumidoO3(solicitud, a) != nil {
		return PruebasArtefactoAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	reciboConjunto := clonarReciboConsumoConjuntoFuentesAnalisisO3(
		*a.pruebas.reciboConjunto,
	)
	pruebas := PruebasArtefactoAnalisisO3{
		SolicitudRC:           a.pruebas.evidenciaRC.datos.solicitud,
		ResultadoRC:           a.pruebas.evidenciaRC.datos.resultado,
		ConfirmacionRC:        a.pruebas.evidenciaRC.datos.confirmacion,
		OrdenConsumoRC:        a.pruebas.evidenciaRC.datos.orden,
		ReciboConsumoRC:       reciboConjunto.ReciboRC,
		OrdenConsumoConjunto:  a.pruebas.ordenConjunto,
		ReciboConsumoConjunto: &reciboConjunto,
	}
	if confirmacion := a.pruebas.evidenciaRC.datos.confirmacionMotivo; confirmacion != nil {
		clon := confirmacion.clonar()
		pruebas.ConfirmacionPublicacion = &clon
	}
	if a.pruebas.evidenciaCoste.datos != nil {
		solicitudCoste :=
			a.pruebas.evidenciaCoste.datos.solicitud
		resultadoCoste :=
			a.pruebas.evidenciaCoste.datos.resultado
		confirmacionCoste :=
			a.pruebas.evidenciaCoste.datos.confirmacion
		ordenCoste :=
			a.pruebas.evidenciaCoste.datos.orden
		reciboCoste := *reciboConjunto.ReciboCoste
		pruebas.SolicitudCoste = &solicitudCoste
		pruebas.ResultadoCoste = &resultadoCoste
		pruebas.ConfirmacionCoste = &confirmacionCoste
		pruebas.OrdenConsumoCoste = &ordenCoste
		pruebas.ReciboConsumoCoste = &reciboCoste
	}
	return pruebas, nil
}

func datosArtefactoAnalisisDesdePruebasO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
	pruebas pruebasArtefactoAnalisisO3,
	preparadoEn time.Time,
) (DatosArtefactoAnalisis, error) {
	if solicitud.Validar() != nil ||
		!instanteSeguroOperacionAnalisis(preparadoEn) ||
		preparadoEn.Before(solicitud.SolicitadaEn) ||
		pruebas.evidenciaRC.validarEn(preparadoEn) != nil {
		return DatosArtefactoAnalisis{},
			ErrArtefactoAnalisisNoConfiable
	}
	ordenEsperada, errOrden := nuevaOrdenConsumoConjuntoFuentesAnalisisO3(
		solicitud,
		pruebas.evidenciaRC,
		pruebas.evidenciaCoste,
	)
	datosOrdenEsperada, errDatosEsperados := ordenEsperada.Datos()
	datosOrdenActual, errDatosActuales := pruebas.ordenConjunto.Datos()
	if errOrden != nil || errDatosEsperados != nil ||
		errDatosActuales != nil ||
		!reflect.DeepEqual(datosOrdenEsperada, datosOrdenActual) {
		return DatosArtefactoAnalisis{},
			ErrArtefactoAnalisisNoConfiable
	}
	if pruebas.reciboConjunto != nil &&
		pruebas.reciboConjunto.ValidarPara(
			pruebas.ordenConjunto,
		) != nil {
		return DatosArtefactoAnalisis{},
			ErrArtefactoAnalisisNoConfiable
	}
	validacion, err := materializarValidacionRCEvidenciaO3(
		pruebas.evidenciaRC,
	)
	datosSolicitudRC, errSolicitudRC :=
		pruebas.evidenciaRC.datos.solicitud.Datos()
	datosResultadoRC, errResultadoRC :=
		pruebas.evidenciaRC.datos.resultado.Datos()
	datosConfirmacionRC, errConfirmacionRC :=
		pruebas.evidenciaRC.datos.confirmacion.Datos()
	if err != nil || errSolicitudRC != nil || errResultadoRC != nil ||
		errConfirmacionRC != nil {
		return DatosArtefactoAnalisis{},
			ErrArtefactoAnalisisNoConfiable
	}
	datos := DatosArtefactoAnalisis{
		ArtefactoRef:      solicitud.ArtefactoRef,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionExpediente,
		DatosFuncionales:  solicitud.DatosFuncionales,
		ResultadoRC:       validacion.Resultado,
		FuenteRCRef:       validacion.FuenteRef,
		ReciboRCRef:       validacion.ReciboRef,
		ValidadaEn:        validacion.ValidadaEn,
		FechaRC:           clonarTiempo(validacion.FechaRC),
		NumeroRC:          validacion.Numero,
		ImporteRC:         clonarImporte(validacion.Importe),
		DocumentoRCRef:    validacion.DocumentoRef,
		MotivoRC: motivoRCGobernadoDesdeResultadoO3(
			datosResultadoRC,
		),
		PeticionRCRef:          datosSolicitudRC.PeticionRef,
		HuellaPeticionRCHMAC:   datosSolicitudRC.HuellaPeticionHMAC,
		HuellaRespuestaRC:      datosResultadoRC.HuellaRespuestaSHA256,
		SelloRespuestaRCHMAC:   datosResultadoRC.Atestacion.SelloHMAC,
		GeneracionRespuestaRC:  datosResultadoRC.Atestacion.Metadatos.Generacion,
		ConfirmadaRCEn:         datosConfirmacionRC.VerificadaEn,
		RespuestaRCValidaHasta: datosConfirmacionRC.ValidaHasta,
		AutoridadFuenteRC: vinculoAutoridadFuenteAnalisisO3(
			pruebas.evidenciaRC.datos.identidadFuente,
		),
		AutoridadVerificadorRC: vinculoAutoridadFuenteAnalisisO3(
			pruebas.evidenciaRC.datos.identidadVerificador,
		),
		AutoridadPublicadorRC: vinculoAutoridadFuenteAnalisisO3(
			pruebas.evidenciaRC.datos.identidadPublicador,
		),
		PreparadoEn: preparadoEn,
	}
	if pruebas.evidenciaRC.datos.confirmacionMotivo != nil {
		confirmacion, errMotivo :=
			pruebas.evidenciaRC.datos.confirmacionMotivo.Datos()
		if errMotivo != nil {
			return DatosArtefactoAnalisis{},
				ErrArtefactoAnalisisNoConfiable
		}
		datos.PublicacionMotivoRef = confirmacion.PublicacionRef
		datos.ReciboVerificacionMotivoRef =
			confirmacion.ReciboVerificacionRef
	}
	if pruebas.evidenciaCoste.datos != nil {
		if pruebas.evidenciaCoste.validarEn(preparadoEn) != nil {
			return DatosArtefactoAnalisis{},
				ErrArtefactoAnalisisNoConfiable
		}
		incorporarCosteArtefactoAnalisisO3(
			&datos,
			pruebas.evidenciaCoste,
		)
	}
	if pruebas.reciboConjunto != nil {
		datos.ConsumoRCRef =
			pruebas.reciboConjunto.ReciboRC.ConsumoRef
		datos.ConsumidaRCEn =
			pruebas.reciboConjunto.ReciboRC.ConsumidaEn
		if pruebas.reciboConjunto.ReciboCoste != nil {
			datos.ConsumoCosteRef =
				pruebas.reciboConjunto.ReciboCoste.ConsumoRef
			datos.ConsumidaCosteEn =
				pruebas.reciboConjunto.ReciboCoste.ConsumidaEn
		}
	}
	huella, err := huellaArtefactoAnalisisO3(datos)
	if err != nil {
		return DatosArtefactoAnalisis{},
			ErrArtefactoAnalisisNoConfiable
	}
	datos.ArtefactoHuellaSHA256 = huella
	if validarDatosArtefactoAnalisis(solicitud, datos) != nil {
		return DatosArtefactoAnalisis{},
			ErrArtefactoAnalisisNoConfiable
	}
	return datos, nil
}

func incorporarCosteArtefactoAnalisisO3(
	destino *DatosArtefactoAnalisis,
	evidencia EvidenciaCalculoCosteVerificadaO3,
) {
	resultado, _ := evidencia.datos.resultado.Datos()
	solicitud, _ := evidencia.datos.solicitud.Datos()
	confirmacion, _ := evidencia.datos.confirmacion.Datos()
	importe := resultado.Importe
	destino.CostePrevisto = &importe
	destino.FuenteCosteRef = resultado.FuenteRef
	destino.ReciboCosteRef = resultado.ReciboRef
	destino.CalculadoEn = resultado.CalculadoEn
	destino.PeticionCosteRef = solicitud.PeticionRef
	destino.HuellaPeticionCosteHMAC = solicitud.HuellaPeticionHMAC
	destino.HuellaRespuestaCoste = resultado.HuellaRespuestaSHA256
	destino.SelloRespuestaCosteHMAC = resultado.Atestacion.SelloHMAC
	destino.GeneracionRespuestaCoste =
		resultado.Atestacion.Metadatos.Generacion
	destino.ConfirmadaCosteEn = confirmacion.VerificadaEn
	destino.RespuestaCosteValidaHasta = confirmacion.ValidaHasta
	destino.AutoridadFuenteCoste = vinculoAutoridadFuenteAnalisisO3(
		evidencia.datos.identidadFuente,
	)
	destino.AutoridadVerificadorCoste = vinculoAutoridadFuenteAnalisisO3(
		evidencia.datos.identidadVerificador,
	)
}

func motivoRCGobernadoDesdeResultadoO3(
	resultado DatosResultadoValidacionRC,
) MotivoRCGobernado {
	if resultado.Validacion.Resultado == domain.RCValidada {
		return MotivoRCGobernado{}
	}
	motivo, err := resultado.Motivo.Datos()
	if err != nil {
		return MotivoRCGobernado{}
	}
	return MotivoRCGobernado{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           motivo.CatalogoRef,
			CatalogoVersion:      int(motivo.CatalogoVersion),
			CatalogoHuellaSHA256: motivo.CatalogoHuella,
			EntradaClave:         string(motivo.EntradaClave),
		},
		ClaveMensajeI18N: motivo.ClaveMensajeI18N,
	}
}
