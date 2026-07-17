package calculoexperiencia

func (m *materialesSeleccionAplicacionesV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.seleccion.aplicaciones", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialAplicacionSeleccionV1](
		datos, maximoAplicacionesResultadoV1, "resultado.seleccion.aplicaciones",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesSeleccionDescartesV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.seleccion.descartes", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialDescarteSeleccionV1](
		datos, maximoDescartesResultadoV1, "resultado.seleccion.descartes",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesSinCoincidenciaV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.seleccion.sin_coincidencia", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialSinCoincidenciaV1](
		datos, maximoSinCoincidenciaResultadoV1, "resultado.seleccion.sin_coincidencia",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesIntervalosResultadoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.intervalos", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialIntervaloAplicacionV1](
		datos, maximoAplicacionesResultadoV1, "resultado.intervalos",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesCalculosResultadoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.aplicaciones", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialAplicacionCalculadaV1](
		datos, maximoAplicacionesResultadoV1, "resultado.aplicaciones",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesReglasResultadoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.reglas", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialReglaResultadoV1](
		datos, maximoReglasResultadoV1, "resultado.reglas",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesSeccionesResultadoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.secciones", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialSeccionResultadoV1](
		datos, maximoSeccionesResultadoV1, "resultado.secciones",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesBloqueosResultadoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.bloqueos", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialBloqueoResultadoV1](
		datos, maximoBloqueosResultadoV1, "resultado.bloqueos",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesReferenciasBloqueoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.bloqueo.tramos", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[materialReferencia](
		datos, maximoReferenciasBloqueoV1, "resultado.bloqueo.tramos",
	)
	if err == nil {
		*m = resultado
	}
	return err
}

func (m *materialesReglasBloqueoV1) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("resultado.bloqueo.reglas", CodigoValorNoCanonico)
	}
	resultado, err := decodificarColeccionResultadoV1[string](
		datos, maximoReferenciasBloqueoV1, "resultado.bloqueo.reglas",
	)
	if err == nil {
		*m = resultado
	}
	return err
}
