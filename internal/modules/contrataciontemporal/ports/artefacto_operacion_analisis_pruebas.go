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
}

// PruebasArtefactoAnalisisO3 transporta a O3-04 las respuestas, las
// confirmaciones y la orden exacta todavía no consumida. No permite
// reconstruir ni serializar un ArtefactoAnalisisPreparado.
type PruebasArtefactoAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	SolicitudRC             SolicitudValidarRC
	ResultadoRC             ResultadoValidacionRC
	ConfirmacionRC          ConfirmacionRespuestaFuenteAnalisis
	ConfirmacionPublicacion *ConfirmacionPublicacionMotivoFuenteAnalisis
	OrdenConsumoRC          OrdenConsumoRespuestaFuenteAnalisis
	SolicitudCoste          *SolicitudCalcularCoste
	ResultadoCoste          *ResultadoCalculoCoste
	ConfirmacionCoste       *ConfirmacionRespuestaFuenteAnalisis
	OrdenConsumoCoste       *OrdenConsumoRespuestaFuenteAnalisis
	OrdenConsumoConjunto    OrdenConsumoConjuntoFuentesAnalisisO3
}

// NuevoArtefactoAnalisisVerificadoO3 es una fábrica mínima del tipo opaco.
// La orquestación de fuentes, autoridades y tiempos pertenece a application.
func NuevoArtefactoAnalisisVerificadoO3(
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
		SolicitudRC:          a.pruebas.evidenciaRC.datos.solicitud,
		ResultadoRC:          a.pruebas.evidenciaRC.datos.resultado,
		ConfirmacionRC:       a.pruebas.evidenciaRC.datos.confirmacion,
		OrdenConsumoRC:       a.pruebas.evidenciaRC.datos.orden,
		OrdenConsumoConjunto: a.pruebas.ordenConjunto,
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
		pruebas.SolicitudCoste = &solicitudCoste
		pruebas.ResultadoCoste = &resultadoCoste
		pruebas.ConfirmacionCoste = &confirmacionCoste
		pruebas.OrdenConsumoCoste = &ordenCoste
	}
	return pruebas, nil
}

func (a ArtefactoAnalisisPreparado) ValidarVigenciaEn(
	solicitud SolicitudPrepararArtefactoAnalisis,
	comprobadaEn time.Time,
) error {
	if validarArtefactoAnalisisPreparado(solicitud, a) != nil ||
		a.pruebas.evidenciaRC.validarEn(comprobadaEn) != nil ||
		(a.pruebas.evidenciaCoste.datos != nil &&
			a.pruebas.evidenciaCoste.validarEn(comprobadaEn) != nil) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
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
