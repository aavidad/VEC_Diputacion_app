package ports

import (
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type pruebasArtefactoAnalisisO3 struct {
	evidenciaRC    EvidenciaValidacionRCVerificadaO3
	reciboRC       ReciboConsumoRespuestaFuenteAnalisis
	evidenciaCoste EvidenciaCalculoCosteVerificadaO3
	reciboCoste    *ReciboConsumoRespuestaFuenteAnalisis
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
}

func nuevoArtefactoAnalisisPreparadoDesdeFuentesO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
	evidenciaRC EvidenciaValidacionRCVerificadaO3,
	reciboRC ReciboConsumoRespuestaFuenteAnalisis,
	evidenciaCoste EvidenciaCalculoCosteVerificadaO3,
	reciboCoste *ReciboConsumoRespuestaFuenteAnalisis,
	preparadoEn time.Time,
) (ArtefactoAnalisisPreparado, error) {
	pruebas := pruebasArtefactoAnalisisO3{
		evidenciaRC:    evidenciaRC,
		reciboRC:       reciboRC,
		evidenciaCoste: evidenciaCoste,
		reciboCoste:    reciboCoste,
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

func (a ArtefactoAnalisisPreparado) PruebasParaO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
) (PruebasArtefactoAnalisisO3, error) {
	if validarArtefactoAnalisisPreparado(solicitud, a) != nil {
		return PruebasArtefactoAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	pruebas := PruebasArtefactoAnalisisO3{
		SolicitudRC:             a.pruebas.evidenciaRC.datos.solicitud,
		ResultadoRC:             a.pruebas.evidenciaRC.datos.resultado,
		ConfirmacionRC:          a.pruebas.evidenciaRC.datos.confirmacion,
		ConfirmacionPublicacion: a.pruebas.evidenciaRC.datos.confirmacionMotivo,
		OrdenConsumoRC:          a.pruebas.evidenciaRC.datos.orden,
		ReciboConsumoRC:         a.pruebas.reciboRC,
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
		reciboCoste := *a.pruebas.reciboCoste
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
		pruebas.evidenciaRC.validarEn(preparadoEn) != nil ||
		pruebas.reciboRC.ValidarPara(
			pruebas.evidenciaRC.datos.orden,
		) != nil ||
		pruebas.reciboRC.ConsumidaEn.After(preparadoEn) {
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
		ConsumoRCRef:           pruebas.reciboRC.ConsumoRef,
		ConsumidaRCEn:          pruebas.reciboRC.ConsumidaEn,
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
	if pruebas.evidenciaCoste.datos == nil {
		if pruebas.reciboCoste != nil {
			return DatosArtefactoAnalisis{},
				ErrArtefactoAnalisisNoConfiable
		}
	} else {
		if pruebas.reciboCoste == nil ||
			pruebas.evidenciaCoste.validarEn(preparadoEn) != nil ||
			pruebas.reciboCoste.ValidarPara(
				pruebas.evidenciaCoste.datos.orden,
			) != nil ||
			pruebas.reciboCoste.ConsumidaEn.After(preparadoEn) {
			return DatosArtefactoAnalisis{},
				ErrArtefactoAnalisisNoConfiable
		}
		incorporarCosteArtefactoAnalisisO3(
			&datos,
			pruebas.evidenciaCoste,
			*pruebas.reciboCoste,
		)
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
	recibo ReciboConsumoRespuestaFuenteAnalisis,
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
	destino.ConsumoCosteRef = recibo.ConsumoRef
	destino.ConsumidaCosteEn = recibo.ConsumidaEn
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
