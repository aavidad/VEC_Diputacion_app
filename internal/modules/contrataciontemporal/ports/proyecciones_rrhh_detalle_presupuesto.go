package ports

import "unicode/utf8"

const (
	// limiteBytesEntradaDetalleRRHHMinimizada coincide con el límite del
	// contrato protegido de resultado. Todo valor aceptado tiene una
	// representación JSON cerrada cuya cota superior cabe en 256 KiB.
	limiteBytesEntradaDetalleRRHHMinimizada uint64 = 256 * 1024

	presupuestoEnteroJSONRRHH   uint64 = 20
	presupuestoBooleanoJSONRRHH uint64 = 5
	presupuestoInstanteJSONRRHH uint64 = 64

	// Esta constante replica únicamente la parte mínima e incondicional que
	// medidorPresupuestoDetalleRRHH.hito añade por elemento. Permite rechazar
	// cardinalidades imposibles antes incluso de recorrer la colección.
	presupuestoMinimoHitoDetalleRRHH = uint64(
		len(`{"secuencia":`) + 20 +
			len(`,"version_expediente":`) + 20 +
			len(`,"accion_clave":`) + 2 +
			len(`,"realizada_en":`) + 64 +
			len(`,"fase_origen":`) + 2 +
			len(`,"fase_destino":`) + 2 +
			len(`,"estado_origen":`) + 2 +
			len(`,"estado_destino":`) + 2 +
			len(`}`),
	)
	limiteMaximoHitosPorPresupuestoRRHH = int(
		limiteBytesEntradaDetalleRRHHMinimizada /
			presupuestoMinimoHitoDetalleRRHH,
	)
)

type medidorPresupuestoDetalleRRHH struct {
	restante uint64
	excedido bool
}

func nuevoMedidorPresupuestoDetalleRRHH() medidorPresupuestoDetalleRRHH {
	return medidorPresupuestoDetalleRRHH{
		restante: limiteBytesEntradaDetalleRRHHMinimizada,
	}
}

func (m *medidorPresupuestoDetalleRRHH) sumar(bytes uint64) {
	if m.excedido {
		return
	}
	if bytes > m.restante {
		m.restante = 0
		m.excedido = true
		return
	}
	m.restante -= bytes
}

func (m *medidorPresupuestoDetalleRRHH) exceder() {
	m.restante = 0
	m.excedido = true
}

func (m *medidorPresupuestoDetalleRRHH) literal(valor string) {
	m.sumar(uint64(len(valor)))
}

// cadena suma exactamente el máximo que encoding/json necesita para una
// cadena: comillas, escapes HTML, controles, U+2028/U+2029 y sustitución de
// UTF-8 inválido. No crea una segunda representación ni acepta JSON abierto.
func (m *medidorPresupuestoDetalleRRHH) cadena(valor string) {
	m.sumar(2)
	for indice := 0; indice < len(valor) && !m.excedido; {
		actual := valor[indice]
		if actual < utf8.RuneSelf {
			switch {
			case actual == '"' || actual == '\\' ||
				actual == '\b' || actual == '\f' ||
				actual == '\n' || actual == '\r' || actual == '\t':
				m.sumar(2)
			case actual < 0x20 || actual == '<' || actual == '>' ||
				actual == '&':
				m.sumar(6)
			default:
				m.sumar(1)
			}
			indice++
			continue
		}
		caracter, ancho := utf8.DecodeRuneInString(valor[indice:])
		if caracter == utf8.RuneError && ancho == 1 {
			m.sumar(6)
			indice++
			continue
		}
		if caracter == '\u2028' || caracter == '\u2029' {
			m.sumar(6)
		} else {
			m.sumar(uint64(ancho))
		}
		indice += ancho
	}
}

func (m *medidorPresupuestoDetalleRRHH) entero() {
	m.sumar(presupuestoEnteroJSONRRHH)
}

func (m *medidorPresupuestoDetalleRRHH) booleano() {
	m.sumar(presupuestoBooleanoJSONRRHH)
}

func (m *medidorPresupuestoDetalleRRHH) instante() {
	m.sumar(presupuestoInstanteJSONRRHH)
}

func (m *medidorPresupuestoDetalleRRHH) resumen(
	r ResumenExpedienteRRHH,
) {
	m.literal(`{"expediente_ref":`)
	m.cadena(r.ExpedienteRef)
	m.literal(`,"organizacion_ref":`)
	m.cadena(r.OrganizacionRef)
	m.literal(`,"numero_visible":`)
	m.cadena(r.NumeroVisible)
	m.literal(`,"version":`)
	m.entero()
	m.literal(`,"flujo_ref":`)
	m.cadena(r.FlujoRef)
	m.literal(`,"flujo_version":`)
	m.entero()
	m.literal(`,"flujo_huella_sha256":`)
	m.cadena(r.FlujoHuella)
	m.literal(`,"fase_clave":`)
	m.cadena(string(r.FaseClave))
	m.literal(`,"estado_clave":`)
	m.cadena(string(r.EstadoClave))
	m.literal(`,"centro_ref":`)
	m.cadena(r.CentroRef)
	m.literal(`,"categoria_ref":`)
	m.cadena(r.CategoriaRef)
	m.literal(`,"modalidad_clave":`)
	m.cadena(string(r.ModalidadClave))
	m.literal(`,"unidad_ref":`)
	m.cadena(r.UnidadRef)
	m.literal(`,"creado_en":`)
	m.instante()
	m.literal(`,"actualizado_en":`)
	m.instante()
	m.literal(`}`)
}

func (m *medidorPresupuestoDetalleRRHH) solicitud(
	s SolicitudOperativaRRHH,
) {
	m.literal(`{"grupo_subgrupo":`)
	m.cadena(s.GrupoSubgrupo)
	m.literal(`,"motivo_clave":`)
	m.cadena(string(s.MotivoClave))
	m.literal(`,"periodo_inicio":`)
	m.instante()
	m.literal(`,"periodo_fin":`)
	m.instante()
	m.literal(`}`)
}

func (m *medidorPresupuestoDetalleRRHH) analisis(
	a *AnalisisOperativoRRHH,
) {
	if a == nil {
		m.literal(`null`)
		return
	}
	m.literal(`{"modalidad_clave":`)
	m.cadena(string(a.ModalidadClave))
	m.literal(`,"categoria_ref":`)
	m.cadena(a.CategoriaRef)
	m.literal(`,"causa_clave":`)
	m.cadena(string(a.CausaClave))
	m.literal(`,"periodo_inicio":`)
	m.instante()
	m.literal(`,"periodo_fin":`)
	m.instante()
	m.literal(`,"porcentaje_jornada":`)
	m.entero()
	m.literal(`,"resultado_rc":`)
	m.cadena(string(a.ResultadoRC))
	m.literal(`,"coste_previsto":`)
	if a.CostePrevisto == nil {
		m.literal(`null`)
	} else {
		m.literal(`{"centimos":`)
		m.entero()
		m.literal(`,"moneda":`)
		m.cadena(a.CostePrevisto.Moneda)
		m.literal(`}`)
	}
	m.literal(`,"fuente_coste_ref":`)
	m.cadena(a.FuenteCosteRef)
	m.literal(`}`)
}

func (m *medidorPresupuestoDetalleRRHH) cobertura(
	c *CoberturaOperativaRRHH,
) {
	if c == nil {
		m.literal(`null`)
		return
	}
	m.literal(`{"via_clave":`)
	m.cadena(string(c.ViaClave))
	m.literal(`,"decision_gobernada":`)
	m.booleano()
	m.literal(`,"procedimiento_ref":`)
	m.cadena(c.ProcedimientoRef)
	m.literal(`,"bolsa_ref":`)
	m.cadena(c.BolsaRef)
	m.literal(`,"comprobaciones":[`)
	for indice, comprobacion := range c.Comprobaciones {
		if m.excedido {
			break
		}
		if indice > 0 {
			m.literal(`,`)
		}
		m.literal(`{"clave":`)
		m.cadena(string(comprobacion.Clave))
		m.literal(`,"resultado":`)
		m.cadena(string(comprobacion.Resultado))
		m.literal(`}`)
	}
	m.literal(`]}`)
}

func (m *medidorPresupuestoDetalleRRHH) asignacion(
	a *AsignacionOperativaRRHH,
) {
	if a == nil {
		m.literal(`null`)
		return
	}
	m.literal(`{"unidad_ref":`)
	m.cadena(a.UnidadRef)
	m.literal(`,"asignada_en":`)
	m.instante()
	m.literal(`,"motivo_clave":`)
	m.cadena(string(a.MotivoClave))
	m.literal(`}`)
}

func (m *medidorPresupuestoDetalleRRHH) hito(h HitoExpedienteRRHH) {
	m.literal(`{"secuencia":`)
	m.entero()
	m.literal(`,"version_expediente":`)
	m.entero()
	m.literal(`,"accion_clave":`)
	m.cadena(string(h.AccionClave))
	m.literal(`,"realizada_en":`)
	m.instante()
	m.literal(`,"fase_origen":`)
	m.cadena(string(h.FaseOrigen))
	m.literal(`,"fase_destino":`)
	m.cadena(string(h.FaseDestino))
	m.literal(`,"estado_origen":`)
	m.cadena(string(h.EstadoOrigen))
	m.literal(`,"estado_destino":`)
	m.cadena(string(h.EstadoDestino))
	m.literal(`}`)
}

func presupuestoEntradaDetalleRRHHMinimizada(
	resumen ResumenExpedienteRRHH,
	solicitud SolicitudOperativaRRHH,
	analisis *AnalisisOperativoRRHH,
	referenciaAnalisis ReferenciaHitoAnalisisRRHH,
	cobertura *CoberturaOperativaRRHH,
	referenciaCobertura ReferenciaHitoCoberturaRRHH,
	asignacion *AsignacionOperativaRRHH,
	referenciaAsignacion ReferenciaHitoAsignacionRRHH,
	hitos []HitoExpedienteRRHH,
) (uint64, bool) {
	if cobertura != nil && len(cobertura.Comprobaciones) > 32 {
		return limiteBytesEntradaDetalleRRHHMinimizada + 1, false
	}
	m := nuevoMedidorPresupuestoDetalleRRHH()
	if len(hitos) > limiteMaximoHitosPorPresupuestoRRHH {
		m.exceder()
		return limiteBytesEntradaDetalleRRHHMinimizada + 1, false
	}
	m.literal(`{"resumen":`)
	m.resumen(resumen)
	m.literal(`,"solicitud":`)
	m.solicitud(solicitud)
	m.literal(`,"analisis":`)
	m.analisis(analisis)
	m.literal(`,"referencia_analisis":`)
	m.entero()
	m.literal(`,"cobertura":`)
	m.cobertura(cobertura)
	m.literal(`,"referencia_cobertura":`)
	m.entero()
	m.literal(`,"asignacion":`)
	m.asignacion(asignacion)
	m.literal(`,"referencia_asignacion":`)
	m.entero()
	m.literal(`,"hitos":[`)
	for indice, hito := range hitos {
		if m.excedido {
			break
		}
		if indice > 0 {
			m.literal(`,`)
		}
		m.hito(hito)
	}
	m.literal(`]}`)
	if m.excedido {
		return limiteBytesEntradaDetalleRRHHMinimizada + 1, false
	}
	return limiteBytesEntradaDetalleRRHHMinimizada - m.restante, true
}
