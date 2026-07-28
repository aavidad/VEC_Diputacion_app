package ports

// ReferenciaHitoAnalisisRRHH identifica, de forma nominal, la actuación
// durable que materializó la proyección vigente del análisis. Su valor no es
// serializable ni intercambiable con los vínculos de otros bloques.
type ReferenciaHitoAnalisisRRHH struct {
	bloqueoSerializacionConsultaRRHH
	secuencia uint64
}

// NuevaReferenciaHitoAnalisisRRHH construye una referencia uno-basada. La
// primera actuación siempre corresponde al alta, por lo que un análisis solo
// puede vincularse desde la segunda.
func NuevaReferenciaHitoAnalisisRRHH(
	secuencia uint64,
) (ReferenciaHitoAnalisisRRHH, error) {
	if !secuenciaHitoOperativoRRHHValida(secuencia) {
		return ReferenciaHitoAnalisisRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return ReferenciaHitoAnalisisRRHH{secuencia: secuencia}, nil
}

// ReferenciaHitoCoberturaRRHH identifica, de forma nominal, la actuación
// durable que materializó la proyección vigente de cobertura.
type ReferenciaHitoCoberturaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	secuencia uint64
}

// NuevaReferenciaHitoCoberturaRRHH construye la referencia uno-basada a la
// actuación de cobertura seleccionada por la consulta durable.
func NuevaReferenciaHitoCoberturaRRHH(
	secuencia uint64,
) (ReferenciaHitoCoberturaRRHH, error) {
	if !secuenciaHitoOperativoRRHHValida(secuencia) {
		return ReferenciaHitoCoberturaRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return ReferenciaHitoCoberturaRRHH{secuencia: secuencia}, nil
}

// ReferenciaHitoAsignacionRRHH identifica, de forma nominal, la actuación
// durable que materializó la proyección vigente de asignación.
type ReferenciaHitoAsignacionRRHH struct {
	bloqueoSerializacionConsultaRRHH
	secuencia uint64
}

// NuevaReferenciaHitoAsignacionRRHH construye la referencia uno-basada a la
// actuación de asignación vigente seleccionada por la consulta durable.
func NuevaReferenciaHitoAsignacionRRHH(
	secuencia uint64,
) (ReferenciaHitoAsignacionRRHH, error) {
	if !secuenciaHitoOperativoRRHHValida(secuencia) {
		return ReferenciaHitoAsignacionRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return ReferenciaHitoAsignacionRRHH{secuencia: secuencia}, nil
}

func secuenciaHitoOperativoRRHHValida(secuencia uint64) bool {
	return secuencia >= 2 && secuencia <= versionMaximaJSONSegura
}

// EntradaDetalleExpedienteRRHHMinimizada es el único transporte nominal desde
// un adaptador de lectura hacia el constructor de la proyección. No admite
// agregados, personas, documentos, observaciones, campos libres ni JSON.
//
// El adaptador PostgreSQL debe decodificar su contrato estricto en tipos
// propios, construir las referencias nominales y entregar únicamente estos
// datos reducidos.
type EntradaDetalleExpedienteRRHHMinimizada struct {
	bloqueoSerializacionConsultaRRHH
	resumen              ResumenExpedienteRRHH
	solicitud            SolicitudOperativaRRHH
	analisis             *AnalisisOperativoRRHH
	referenciaAnalisis   ReferenciaHitoAnalisisRRHH
	cobertura            *CoberturaOperativaRRHH
	referenciaCobertura  ReferenciaHitoCoberturaRRHH
	asignacion           *AsignacionOperativaRRHH
	referenciaAsignacion ReferenciaHitoAsignacionRRHH
	hitos                []HitoExpedienteRRHH
}

// NuevaEntradaDetalleExpedienteRRHHMinimizada crea una instantánea defensiva
// de la proyección reducida. Las referencias nulas solo son válidas cuando el
// bloque correspondiente también es nulo; la validación completa se repite al
// construir el detalle junto con su recibo de lectura. Antes de copiar,
// comprueba que la cota JSON cerrada de toda la entrada no supera 256 KiB.
func NuevaEntradaDetalleExpedienteRRHHMinimizada(
	resumen ResumenExpedienteRRHH,
	solicitud SolicitudOperativaRRHH,
	analisis *AnalisisOperativoRRHH,
	referenciaAnalisis ReferenciaHitoAnalisisRRHH,
	cobertura *CoberturaOperativaRRHH,
	referenciaCobertura ReferenciaHitoCoberturaRRHH,
	asignacion *AsignacionOperativaRRHH,
	referenciaAsignacion ReferenciaHitoAsignacionRRHH,
	hitos []HitoExpedienteRRHH,
) (EntradaDetalleExpedienteRRHHMinimizada, error) {
	if _, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		resumen, solicitud, analisis, referenciaAnalisis,
		cobertura, referenciaCobertura,
		asignacion, referenciaAsignacion, hitos,
	); !cabe || validarComponentesEntradaDetalleRRHHMinimizada(
		resumen, solicitud, analisis, referenciaAnalisis,
		cobertura, referenciaCobertura,
		asignacion, referenciaAsignacion, hitos,
	) != nil {
		return EntradaDetalleExpedienteRRHHMinimizada{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	entrada := EntradaDetalleExpedienteRRHHMinimizada{
		resumen: resumen, solicitud: solicitud,
		analisis:             clonarAnalisisOperativoMinimizadoRRHH(analisis),
		referenciaAnalisis:   referenciaAnalisis,
		cobertura:            clonarCoberturaOperativaMinimizadaRRHH(cobertura),
		referenciaCobertura:  referenciaCobertura,
		asignacion:           clonarAsignacionOperativaMinimizadaRRHH(asignacion),
		referenciaAsignacion: referenciaAsignacion,
		hitos:                clonarHitosRRHH(hitos),
	}
	if validarComponentesEntradaDetalleRRHHMinimizada(
		entrada.resumen, entrada.solicitud,
		entrada.analisis, entrada.referenciaAnalisis,
		entrada.cobertura, entrada.referenciaCobertura,
		entrada.asignacion, entrada.referenciaAsignacion,
		entrada.hitos,
	) != nil {
		return EntradaDetalleExpedienteRRHHMinimizada{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return entrada, nil
}

func validarComponentesEntradaDetalleRRHHMinimizada(
	resumen ResumenExpedienteRRHH,
	solicitud SolicitudOperativaRRHH,
	analisis *AnalisisOperativoRRHH,
	referenciaAnalisis ReferenciaHitoAnalisisRRHH,
	cobertura *CoberturaOperativaRRHH,
	referenciaCobertura ReferenciaHitoCoberturaRRHH,
	asignacion *AsignacionOperativaRRHH,
	referenciaAsignacion ReferenciaHitoAsignacionRRHH,
	hitos []HitoExpedienteRRHH,
) error {
	if resumen.Validar() != nil || solicitud.validar() != nil ||
		len(hitos) == 0 ||
		uint64(len(hitos)) != resumen.Version ||
		!referenciaHitoDetalleRRHHCoherente(
			analisis != nil, referenciaAnalisis.secuencia,
		) ||
		!referenciaHitoDetalleRRHHCoherente(
			cobertura != nil, referenciaCobertura.secuencia,
		) ||
		!referenciaHitoDetalleRRHHCoherente(
			asignacion != nil, referenciaAsignacion.secuencia,
		) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if analisis != nil && analisis.validar() != nil ||
		cobertura != nil && cobertura.validar() != nil ||
		asignacion != nil && asignacion.validar() != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	for _, secuencia := range []uint64{
		referenciaAnalisis.secuencia,
		referenciaCobertura.secuencia,
		referenciaAsignacion.secuencia,
	} {
		if secuencia > uint64(len(hitos)) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	}
	return nil
}

func (e EntradaDetalleExpedienteRRHHMinimizada) validarReferencias() error {
	if _, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		e.resumen, e.solicitud, e.analisis, e.referenciaAnalisis,
		e.cobertura, e.referenciaCobertura,
		e.asignacion, e.referenciaAsignacion, e.hitos,
	); !cabe {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return validarComponentesEntradaDetalleRRHHMinimizada(
		e.resumen, e.solicitud, e.analisis, e.referenciaAnalisis,
		e.cobertura, e.referenciaCobertura,
		e.asignacion, e.referenciaAsignacion, e.hitos,
	)
}

func referenciaHitoDetalleRRHHCoherente(
	bloquePresente bool,
	secuencia uint64,
) bool {
	if !bloquePresente {
		return secuencia == 0
	}
	return secuenciaHitoOperativoRRHHValida(secuencia)
}

// NuevoDetalleExpedienteRRHHMinimizado reconstruye solo las invariantes
// privadas de la proyección. El recibo debe acreditar exactamente el
// expediente y la versión entregados; no se admite una vía alternativa sin
// registro de lectura.
func NuevoDetalleExpedienteRRHHMinimizado(
	entrada EntradaDetalleExpedienteRRHHMinimizada,
	lectura ReciboLecturaRRHH,
) (DetalleExpedienteRRHH, error) {
	if entrada.validarReferencias() != nil {
		return DetalleExpedienteRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	detalle := DetalleExpedienteRRHH{
		Resumen: entrada.resumen, Solicitud: entrada.solicitud,
		Analisis:   clonarAnalisisOperativoMinimizadoRRHH(entrada.analisis),
		Cobertura:  clonarCoberturaOperativaMinimizadaRRHH(entrada.cobertura),
		Asignacion: clonarAsignacionOperativaMinimizadaRRHH(entrada.asignacion),
		Hitos:      clonarHitosRRHH(entrada.hitos), Lectura: lectura,
	}
	if detalle.Analisis != nil {
		detalle.Analisis.vinculo = vinculoDesdeHitoMinimizadoRRHH(
			detalle.Hitos, entrada.referenciaAnalisis.secuencia,
		)
		detalle.bloques |= bloqueAnalisisRRHH
	}
	if detalle.Cobertura != nil {
		detalle.Cobertura.vinculo = vinculoDesdeHitoMinimizadoRRHH(
			detalle.Hitos, entrada.referenciaCobertura.secuencia,
		)
		detalle.bloques |= bloqueCoberturaRRHH
	}
	if detalle.Asignacion != nil {
		detalle.Asignacion.vinculo = vinculoDesdeHitoMinimizadoRRHH(
			detalle.Hitos, entrada.referenciaAsignacion.secuencia,
		)
		detalle.bloques |= bloqueAsignacionRRHH
	}
	detalle.huella = calcularHuellaDetalleRRHH(detalle)
	if detalle.validarEstructura() != nil {
		return DetalleExpedienteRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	return detalle, nil
}

func vinculoDesdeHitoMinimizadoRRHH(
	hitos []HitoExpedienteRRHH,
	secuencia uint64,
) vinculoHitoOperativoRRHH {
	if secuencia == 0 || secuencia > uint64(len(hitos)) {
		return vinculoHitoOperativoRRHH{}
	}
	hito := hitos[secuencia-1]
	if hito.Secuencia != secuencia {
		return vinculoHitoOperativoRRHH{}
	}
	return vinculoHitoOperativoRRHH{
		secuencia: hito.Secuencia, versionExpediente: hito.VersionExpediente,
		accionClave: hito.AccionClave, faseDestino: hito.FaseDestino,
		realizadaEn: hito.RealizadaEn,
	}
}

func clonarHitosRRHH(
	hitos []HitoExpedienteRRHH,
) []HitoExpedienteRRHH {
	return append([]HitoExpedienteRRHH(nil), hitos...)
}

func clonarAnalisisOperativoMinimizadoRRHH(
	origen *AnalisisOperativoRRHH,
) *AnalisisOperativoRRHH {
	if origen == nil {
		return nil
	}
	copia := *origen
	copia.vinculo = vinculoHitoOperativoRRHH{}
	if origen.CostePrevisto != nil {
		coste := *origen.CostePrevisto
		copia.CostePrevisto = &coste
	}
	return &copia
}

func clonarCoberturaOperativaMinimizadaRRHH(
	origen *CoberturaOperativaRRHH,
) *CoberturaOperativaRRHH {
	if origen == nil {
		return nil
	}
	copia := *origen
	copia.vinculo = vinculoHitoOperativoRRHH{}
	copia.Comprobaciones = append(
		[]ComprobacionOperativaRRHH(nil),
		origen.Comprobaciones...,
	)
	return &copia
}

func clonarAsignacionOperativaMinimizadaRRHH(
	origen *AsignacionOperativaRRHH,
) *AsignacionOperativaRRHH {
	if origen == nil {
		return nil
	}
	copia := *origen
	copia.vinculo = vinculoHitoOperativoRRHH{}
	return &copia
}
