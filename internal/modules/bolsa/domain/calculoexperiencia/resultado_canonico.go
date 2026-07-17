package calculoexperiencia

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

type materialContratoMotorResultadoV1 struct {
	Contrato string `json:"contrato"`
	Version  uint64 `json:"version"`
}

type materialHuellaPlanResultadoV1 struct {
	Esquema  string                   `json:"esquema"`
	Motor    materialMotorResultadoV1 `json:"motor"`
	Conjunto materialReferencia       `json:"conjunto"`
}

func vinculoMotorResultadoExperienciaV1() (VinculoMotorResultadoExperienciaV1, error) {
	contenido, err := json.Marshal(materialContratoMotorResultadoV1{
		Contrato: contratoMotorResultadoV1,
		Version:  versionMotorResultadoV1,
	})
	if err != nil {
		return VinculoMotorResultadoExperienciaV1{},
			nuevoError("resultado.motor", CodigoValorInvalido)
	}
	huella := sha256.Sum256(contenido)
	return VinculoMotorResultadoExperienciaV1{
		contrato:             contratoMotorResultadoV1,
		version:              versionMotorResultadoV1,
		huellaContratoSHA256: hex.EncodeToString(huella[:]),
	}, nil
}

func huellaPlanResultadoExperienciaV1(
	motor VinculoMotorResultadoExperienciaV1,
	conjunto reglasbaremo.ReferenciaVersionada,
) (string, error) {
	if validarReferenciaVersionada(conjunto, "resultado.conjunto") != nil {
		return "", nuevoError("resultado.plan", CodigoValorNoCanonico)
	}
	contenido, err := json.Marshal(materialHuellaPlanResultadoV1{
		Esquema: esquemaPlanResultadoV1,
		Motor: materialMotorResultadoV1{
			Contrato: motor.contrato, Version: motor.version,
			HuellaContratoSHA256: motor.huellaContratoSHA256,
		},
		Conjunto: materializarReferencia(conjunto),
	})
	if err != nil {
		return "", nuevoError("resultado.plan", CodigoValorInvalido)
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

// RepresentacionCanonica devuelve el unico JSON V1 admitido. No incorpora
// datos de ejecucion: simulacion y calculo oficial comparten bytes si sus dos
// instantaneas semanticas son identicas.
func (r ResultadoExperienciaV1) RepresentacionCanonica() ([]byte, error) {
	if err := r.Validar(); err != nil {
		return nil, err
	}
	material := materializarResultadoExperienciaV1(r)
	contenido, err := json.Marshal(material)
	if err != nil {
		return nil, nuevoError("resultado.representacion_canonica", CodigoValorInvalido)
	}
	if len(contenido) == 0 || len(contenido) > maximoBytesResultadoV1 {
		return nil, nuevoError("resultado.representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

func (r ResultadoExperienciaV1) MarshalJSON() ([]byte, error) {
	return r.RepresentacionCanonica()
}

func (r ResultadoExperienciaV1) HuellaSHA256() (string, error) {
	contenido, err := r.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func materializarResultadoExperienciaV1(r ResultadoExperienciaV1) materialResultadoExperienciaV1 {
	material := materialResultadoExperienciaV1{
		Esquema:      esquemaResultadoExperienciaV1,
		Estado:       r.estado,
		Fase:         r.fase,
		Vinculos:     materializarVinculosResultadoV1(r.vinculos),
		Seleccion:    materializarSeleccionCanonicaResultadoV1(r.seleccion),
		Intervalos:   make(materialesIntervalosResultadoV1, len(r.intervalos)),
		Aplicaciones: make(materialesCalculosResultadoV1, len(r.aplicaciones)),
		Reglas:       make(materialesReglasResultadoV1, len(r.reglas)),
		Secciones:    make(materialesSeccionesResultadoV1, len(r.secciones)),
		Bloqueos:     make(materialesBloqueosResultadoV1, len(r.bloqueos)),
	}
	if r.tieneTotal {
		total := r.total
		material.Total = &total
	}
	for indice, intervalo := range r.intervalos {
		material.Intervalos[indice] = materializarIntervaloResultadoV1(intervalo)
	}
	for indice, aplicacion := range r.aplicaciones {
		material.Aplicaciones[indice] = materializarAplicacionResultadoV1(aplicacion)
	}
	for indice, regla := range r.reglas {
		material.Reglas[indice] = materializarReglaResultadoV1(regla)
	}
	for indice, seccion := range r.secciones {
		material.Secciones[indice] = materializarSeccionResultadoV1(seccion)
	}
	for indice, bloqueo := range r.bloqueos {
		material.Bloqueos[indice] = materializarBloqueoResultadoV1(bloqueo)
	}
	return material
}

func materializarVinculosResultadoV1(v VinculosResultadoExperienciaV1) materialVinculosResultadoV1 {
	return materialVinculosResultadoV1{
		Motor: materialMotorResultadoV1{
			Contrato: v.motor.contrato, Version: v.motor.version,
			HuellaContratoSHA256: v.motor.huellaContratoSHA256,
		},
		Plan: materialPlanResultadoV1{
			Esquema: v.plan.esquema, HuellaSHA256: v.plan.huellaSHA256,
		},
		Conjunto: materializarReferencia(v.conjunto),
		Entrada: materialEntradaResultadoV1{
			Instantanea:           materializarReferencia(v.entrada.instantanea),
			HuellaContenidoSHA256: v.entrada.huellaContenidoSHA256,
		},
		FechaCorte: v.fechaCorte,
	}
}

func materializarSeleccionCanonicaResultadoV1(
	s SeleccionResultadoExperienciaV1,
) materialSeleccionResultadoV1 {
	material := materialSeleccionResultadoV1{
		Aplicaciones:    make(materialesSeleccionAplicacionesV1, len(s.aplicaciones)),
		Descartes:       make(materialesSeleccionDescartesV1, len(s.descartes)),
		SinCoincidencia: make(materialesSinCoincidenciaV1, len(s.sinCoincidencia)),
		Evaluaciones:    s.evaluaciones,
	}
	for indice, aplicacion := range s.aplicaciones {
		material.Aplicaciones[indice] = materialAplicacionSeleccionV1{
			Tramo: materializarReferencia(aplicacion.tramo), Regla: aplicacion.reglaClave,
			Grupo: aplicacion.grupoClave, Seccion: aplicacion.seccionClave,
			Prioridad: aplicacion.prioridad, Razon: aplicacion.razon,
		}
	}
	for indice, descarte := range s.descartes {
		material.Descartes[indice] = materialDescarteSeleccionV1{
			Tramo: materializarReferencia(descarte.tramo), Regla: descarte.reglaClave,
			Grupo: descarte.grupoClave, ReglaSeleccionada: descarte.reglaSeleccionada,
			Razon: descarte.razon,
		}
	}
	for indice, ausencia := range s.sinCoincidencia {
		material.SinCoincidencia[indice] = materialSinCoincidenciaV1{
			Tramo: materializarReferencia(ausencia.tramo), Razon: ausencia.razon,
		}
	}
	return material
}

func materializarIntervaloResultadoV1(
	i IntervaloAplicacionResultadoExperienciaV1,
) materialIntervaloAplicacionV1 {
	periodo := materialPeriodo{Modo: i.periodo.modo, Desde: i.periodo.desde}
	if fin, presente := i.periodo.FinInformado(); presente {
		periodo.FinInformado = &fin
	}
	material := materialIntervaloAplicacionV1{
		Tramo: materializarReferencia(i.tramo), Regla: i.reglaClave,
		Periodo: periodo, Extremo: i.extremo, Razon: i.razon,
	}
	if i.tieneEfectivo {
		material.Efectivo = &materialIntervaloEfectivoV1{
			Desde: i.efectivo.Desde(), HastaExclusivo: i.efectivo.Hasta(), Dias: i.dias,
		}
	}
	return material
}

func materializarAplicacionResultadoV1(
	a AplicacionCalculadaResultadoExperienciaV1,
) materialAplicacionCalculadaV1 {
	puntuacion := materialPuntuacionV1{Bruto: a.puntuacion.bruto.texto()}
	if a.puntuacion.tieneRedondeado {
		redondeado := a.puntuacion.redondeado.texto()
		puntuacion.Redondeado = &redondeado
	}
	return materialAplicacionCalculadaV1{
		Tramo: materializarReferencia(a.tramo), Regla: a.reglaClave,
		Jornada: materialJornadaV1{
			Origen: a.jornada.origen, Modo: a.jornada.modo, Factor: a.jornada.factor.texto(),
			AtestacionPresente: a.jornada.atestacionPresente,
			AtestacionUsada:    a.jornada.atestacionUsada, Razon: a.jornada.razon,
		},
		Unidades: materialUnidadesV1{
			Exactas: a.unidades.exactas.texto(), Aportadas: a.unidades.aportadas.texto(),
			Resto: a.unidades.resto.texto(), Frontera: a.unidades.frontera,
		},
		Puntuacion: puntuacion,
	}
}

func materializarTopeResultadoV1(t TopeResultadoExperienciaV1) materialTopeV1 {
	material := materialTopeV1{
		Antes: t.antes.texto(), Despues: t.despues.texto(), Aplicado: t.aplicado,
	}
	if t.limitado {
		limite := t.limite.texto()
		material.Limite = &limite
	}
	return material
}

func materializarReglaResultadoV1(r ResultadoReglaExperienciaV1) materialReglaResultadoV1 {
	return materialReglaResultadoV1{
		Seccion: r.seccionClave, Regla: r.reglaClave,
		UnidadesAgregadas:  r.unidadesAgregadas.texto(),
		UnidadesTrasRestos: r.unidadesTrasRestos.texto(), RestoRegla: r.restoRegla.texto(),
		TopeUnidades: materializarTopeResultadoV1(r.topeUnidades), Coeficiente: r.coeficiente,
		Bruto: r.bruto.texto(),
		Redondeo: materialRedondeoV1{
			Momento: r.redondeo.momento, Modo: r.redondeo.modo,
			Entrada: r.redondeo.entrada.texto(), Salida: r.redondeo.salida.texto(),
		},
		TopePuntos:    materializarTopeResultadoV1(r.topePuntos),
		PuntosFinales: r.puntosFinales.texto(),
	}
}

func materializarSeccionResultadoV1(
	s SubtotalSeccionResultadoExperienciaV1,
) materialSeccionResultadoV1 {
	return materialSeccionResultadoV1{
		Seccion: s.seccionClave, AntesTope: s.antesTope.texto(),
		Tope: materializarTopeResultadoV1(s.tope), PuntosFinales: s.puntosFinales,
	}
}

func materializarBloqueoResultadoV1(b BloqueoResultadoExperienciaV1) materialBloqueoResultadoV1 {
	material := materialBloqueoResultadoV1{
		Codigo: b.codigo, Tramos: make(materialesReferenciasBloqueoV1, len(b.tramos)),
		Reglas: make(materialesReglasBloqueoV1, len(b.reglas)),
		Grupo:  b.grupoClave, Seccion: b.seccionClave, ClaveGobernada: b.claveGobernada,
	}
	copy(material.Reglas, b.reglas)
	for indice, tramo := range b.tramos {
		material.Tramos[indice] = materializarReferencia(tramo)
	}
	if b.tieneValorExacto {
		valor := b.valorExacto.texto()
		material.ValorExacto = &valor
	}
	return material
}
