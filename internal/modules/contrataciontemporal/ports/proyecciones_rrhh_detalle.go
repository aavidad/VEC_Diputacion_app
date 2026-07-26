package ports

import (
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var patronGrupoDetalleRRHH = regexp.MustCompile(`^[A-Z][A-Z0-9/+.-]{0,19}$`)

type SolicitudOperativaRRHH struct {
	GrupoSubgrupo string               `json:"grupo_subgrupo"`
	MotivoClave   domain.ClaveCatalogo `json:"motivo_clave"`
	PeriodoInicio time.Time            `json:"periodo_inicio"`
	PeriodoFin    time.Time            `json:"periodo_fin"`
}

func (s SolicitudOperativaRRHH) validar() error {
	periodo := domain.PeriodoPrevisto{Inicio: s.PeriodoInicio, Fin: s.PeriodoFin}
	if !patronGrupoDetalleRRHH.MatchString(s.GrupoSubgrupo) ||
		!s.MotivoClave.Valida() || periodo.Validar() != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

type ImporteOperativoRRHH struct {
	Centimos int64  `json:"centimos"`
	Moneda   string `json:"moneda"`
}

func (i ImporteOperativoRRHH) validar() error {
	if (domain.Importe{Centimos: i.Centimos, Moneda: i.Moneda}).Validar(false) != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

type AnalisisOperativoRRHH struct {
	ModalidadClave    domain.ClaveCatalogo         `json:"modalidad_clave"`
	CategoriaRef      string                       `json:"categoria_ref"`
	CausaClave        domain.ClaveCatalogo         `json:"causa_clave"`
	PeriodoInicio     time.Time                    `json:"periodo_inicio"`
	PeriodoFin        time.Time                    `json:"periodo_fin"`
	PorcentajeJornada domain.JornadaDiezmilesimas  `json:"porcentaje_jornada"`
	ResultadoRC       domain.ResultadoValidacionRC `json:"resultado_rc"`
	CostePrevisto     *ImporteOperativoRRHH        `json:"coste_previsto,omitempty"`
	FuenteCosteRef    string                       `json:"fuente_coste_ref,omitempty"`
	vinculo           vinculoHitoOperativoRRHH
}

func (a AnalisisOperativoRRHH) validar() error {
	periodo := domain.PeriodoPrevisto{Inicio: a.PeriodoInicio, Fin: a.PeriodoFin}
	resultadoValido := a.ResultadoRC == domain.RCValidada ||
		a.ResultadoRC == domain.RCNoRequerida ||
		a.ResultadoRC == domain.RCRechazada
	if !a.ModalidadClave.Valida() ||
		!domain.ReferenciaOpacaValida(a.CategoriaRef) ||
		!a.CausaClave.Valida() || periodo.Validar() != nil ||
		a.PorcentajeJornada.Validar() != nil || !resultadoValido {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if a.CostePrevisto == nil {
		if a.FuenteCosteRef != "" {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		return nil
	}
	if a.CostePrevisto.validar() != nil ||
		!domain.ReferenciaOpacaValida(a.FuenteCosteRef) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

type ComprobacionOperativaRRHH struct {
	Clave     domain.ClaveCatalogo         `json:"clave"`
	Resultado domain.ResultadoComprobacion `json:"resultado"`
}

func (c ComprobacionOperativaRRHH) validar() error {
	resultadoValido := c.Resultado == domain.ComprobacionAfirmativa ||
		c.Resultado == domain.ComprobacionNegativa ||
		c.Resultado == domain.ComprobacionNoAplica ||
		c.Resultado == domain.ComprobacionNoConsta
	if !c.Clave.Valida() || !resultadoValido {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

type CoberturaOperativaRRHH struct {
	ViaClave          domain.ClaveCatalogo        `json:"via_clave"`
	DecisionGobernada bool                        `json:"decision_gobernada"`
	ProcedimientoRef  string                      `json:"procedimiento_ref,omitempty"`
	BolsaRef          string                      `json:"bolsa_ref,omitempty"`
	Comprobaciones    []ComprobacionOperativaRRHH `json:"comprobaciones,omitempty"`
	vinculo           vinculoHitoOperativoRRHH
}

func (c CoberturaOperativaRRHH) validar() error {
	if !c.ViaClave.Valida() {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if c.DecisionGobernada {
		if c.ProcedimientoRef != "" || c.BolsaRef != "" ||
			len(c.Comprobaciones) != 0 {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(c.ProcedimientoRef) ||
		(c.BolsaRef != "" && !domain.ReferenciaOpacaValida(c.BolsaRef)) ||
		len(c.Comprobaciones) == 0 || len(c.Comprobaciones) > 32 {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	vistas := make(map[domain.ClaveCatalogo]struct{}, len(c.Comprobaciones))
	for _, comprobacion := range c.Comprobaciones {
		if comprobacion.validar() != nil {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		if _, repetida := vistas[comprobacion.Clave]; repetida {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		vistas[comprobacion.Clave] = struct{}{}
	}
	return nil
}

type AsignacionOperativaRRHH struct {
	UnidadRef   string               `json:"unidad_ref"`
	AsignadaEn  time.Time            `json:"asignada_en"`
	MotivoClave domain.ClaveCatalogo `json:"motivo_clave,omitempty"`
	vinculo     vinculoHitoOperativoRRHH
}

func (a AsignacionOperativaRRHH) validar() error {
	if !domain.ReferenciaOpacaValida(a.UnidadRef) ||
		!domain.InstanteUTCCanonico(a.AsignadaEn) ||
		(a.MotivoClave != "" && !a.MotivoClave.Valida()) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

// HitoExpedienteRRHH conserva únicamente la secuencia durable de cada fase.
// Excluye actor, observaciones, recibos internos y documentos.
type HitoExpedienteRRHH struct {
	Secuencia         uint64                 `json:"secuencia"`
	VersionExpediente uint64                 `json:"version_expediente"`
	AccionClave       domain.ClaveCatalogo   `json:"accion_clave"`
	RealizadaEn       time.Time              `json:"realizada_en"`
	FaseOrigen        domain.ClaveFase       `json:"fase_origen,omitempty"`
	FaseDestino       domain.ClaveFase       `json:"fase_destino"`
	EstadoOrigen      domain.EstadoOperativo `json:"estado_origen"`
	EstadoDestino     domain.EstadoOperativo `json:"estado_destino"`
}

func (h HitoExpedienteRRHH) validar() error {
	if h.Secuencia < 1 || h.VersionExpediente < 1 ||
		h.Secuencia != h.VersionExpediente ||
		!h.AccionClave.Valida() ||
		!domain.InstanteUTCCanonico(h.RealizadaEn) ||
		!h.FaseDestino.Valida() || !h.EstadoOrigen.Valido() ||
		!h.EstadoDestino.Valido() ||
		(h.FaseOrigen != "" && !h.FaseOrigen.Valida()) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

// DetalleExpedienteRRHH es una proyección explícita de datos operativos. No
// expone el agregado ni campos libres o identificadores de personas.
type DetalleExpedienteRRHH struct {
	Resumen    ResumenExpedienteRRHH    `json:"resumen"`
	Solicitud  SolicitudOperativaRRHH   `json:"solicitud"`
	Analisis   *AnalisisOperativoRRHH   `json:"analisis,omitempty"`
	Cobertura  *CoberturaOperativaRRHH  `json:"cobertura,omitempty"`
	Asignacion *AsignacionOperativaRRHH `json:"asignacion,omitempty"`
	Hitos      []HitoExpedienteRRHH     `json:"hitos"`
	Lectura    ReciboLecturaRRHH        `json:"-"`
	huella     [32]byte
	bloques    uint8
}

// NuevoDetalleExpedienteRRHH valida el agregado completo antes de reducirlo a
// una proyección serializable. Ninguna referencia personal cruza este puerto.
func NuevoDetalleExpedienteRRHH(
	expediente domain.Expediente,
	lectura ReciboLecturaRRHH,
) (DetalleExpedienteRRHH, error) {
	if expediente.Validar() != nil {
		return DetalleExpedienteRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	detalle := DetalleExpedienteRRHH{
		Resumen: resumenDesdeExpedienteRRHH(expediente),
		Solicitud: SolicitudOperativaRRHH{
			GrupoSubgrupo: expediente.Solicitud.GrupoSubgrupo,
			MotivoClave:   expediente.Solicitud.MotivoClave,
			PeriodoInicio: expediente.Solicitud.Periodo.Inicio,
			PeriodoFin:    expediente.Solicitud.Periodo.Fin,
		},
		Hitos:   hitosDesdeExpedienteRRHH(expediente),
		Lectura: lectura,
	}
	if expediente.Analisis != nil {
		detalle.Analisis = analisisDesdeExpedienteRRHH(expediente)
		detalle.bloques |= bloqueAnalisisRRHH
		detalle.Resumen.CategoriaRef = expediente.Analisis.CategoriaRef
		detalle.Resumen.ModalidadClave = expediente.Analisis.ModalidadClave
	}
	if expediente.ViaCobertura != nil {
		detalle.Cobertura = coberturaDesdeExpedienteRRHH(expediente)
		detalle.bloques |= bloqueCoberturaRRHH
	}
	if expediente.Asignacion != nil {
		detalle.Asignacion = &AsignacionOperativaRRHH{
			UnidadRef:   expediente.Asignacion.UnidadRef,
			AsignadaEn:  expediente.Asignacion.AsignadaEn,
			MotivoClave: expediente.Asignacion.MotivoClave,
			vinculo:     vinculoDesdeAsignacionRRHH(*expediente.Asignacion),
		}
		detalle.bloques |= bloqueAsignacionRRHH
		detalle.Resumen.UnidadRef = expediente.Asignacion.UnidadRef
	}
	detalle.huella = calcularHuellaDetalleRRHH(detalle)
	if detalle.validarEstructura() != nil {
		return DetalleExpedienteRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	return detalle, nil
}

func resumenDesdeExpedienteRRHH(e domain.Expediente) ResumenExpedienteRRHH {
	return ResumenExpedienteRRHH{
		ExpedienteRef: e.Referencia, OrganizacionRef: e.OrganizacionRef,
		NumeroVisible: e.NumeroVisible, Version: e.Version,
		FlujoRef: e.Flujo.DefinicionRef, FlujoVersion: e.Flujo.Version,
		FlujoHuella: e.Flujo.HuellaSHA256, FaseClave: e.FaseActual,
		EstadoClave: e.EstadoActual, CentroRef: e.Solicitud.CentroRef,
		CategoriaRef: e.Solicitud.CategoriaRef,
		CreadoEn:     e.CreadoEn, ActualizadoEn: e.ActualizadoEn,
	}
}

func analisisDesdeExpedienteRRHH(e domain.Expediente) *AnalisisOperativoRRHH {
	a := *e.Analisis
	proyeccion := &AnalisisOperativoRRHH{
		ModalidadClave: a.ModalidadClave, CategoriaRef: a.CategoriaRef,
		CausaClave: a.CausaClave, PeriodoInicio: a.Periodo.Inicio,
		PeriodoFin: a.Periodo.Fin, PorcentajeJornada: a.PorcentajeJornada,
		ResultadoRC: a.ValidacionRC.Resultado, FuenteCosteRef: a.FuenteCosteRef,
		vinculo: vinculoDesdeAnalisisRRHH(e),
	}
	if a.CostePrevisto != nil {
		proyeccion.CostePrevisto = &ImporteOperativoRRHH{
			Centimos: a.CostePrevisto.Centimos, Moneda: a.CostePrevisto.Moneda,
		}
	}
	return proyeccion
}

func coberturaDesdeExpedienteRRHH(
	e domain.Expediente,
) *CoberturaOperativaRRHH {
	c := *e.ViaCobertura
	proyeccion := &CoberturaOperativaRRHH{
		ViaClave: c.ViaClave, DecisionGobernada: c.DecisionGobernada != nil,
		ProcedimientoRef: c.ProcedimientoRef, BolsaRef: c.BolsaRef,
		Comprobaciones: make([]ComprobacionOperativaRRHH, len(c.Comprobaciones)),
		vinculo:        vinculoDesdeCoberturaRRHH(e),
	}
	for i, comprobacion := range c.Comprobaciones {
		proyeccion.Comprobaciones[i] = ComprobacionOperativaRRHH{
			Clave: comprobacion.Clave, Resultado: comprobacion.Resultado,
		}
	}
	return proyeccion
}

func hitosDesdeExpedienteRRHH(e domain.Expediente) []HitoExpedienteRRHH {
	hitos := make([]HitoExpedienteRRHH, len(e.Actuaciones))
	for i, actuacion := range e.Actuaciones {
		hitos[i] = HitoExpedienteRRHH{
			Secuencia: actuacion.Secuencia, VersionExpediente: actuacion.VersionExpediente,
			AccionClave: actuacion.AccionClave, RealizadaEn: actuacion.RealizadaEn,
			FaseOrigen: actuacion.FaseOrigen, FaseDestino: actuacion.FaseDestino,
			EstadoOrigen: actuacion.EstadoOrigen, EstadoDestino: actuacion.EstadoDestino,
		}
	}
	return hitos
}
