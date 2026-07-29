package httpinterno

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type envoltorioDetalleRRHH struct {
	Data detalleRRHHJSON `json:"data"`
}

type detalleRRHHJSON struct {
	Esquema    string                   `json:"esquema"`
	Resumen    resumenRRHHJSON          `json:"resumen"`
	Solicitud  solicitudRRHHJSON        `json:"solicitud"`
	Analisis   *analisisRRHHJSON        `json:"analisis,omitempty"`
	Cobertura  *coberturaRRHHJSON       `json:"cobertura,omitempty"`
	Asignacion *asignacionRRHHJSON      `json:"asignacion,omitempty"`
	Hitos      []hitoExpedienteRRHHJSON `json:"hitos"`
}

type solicitudRRHHJSON struct {
	GrupoSubgrupo string `json:"grupo_subgrupo"`
	MotivoClave   string `json:"motivo_clave"`
	PeriodoInicio string `json:"periodo_inicio"`
	PeriodoFin    string `json:"periodo_fin"`
}

type importeRRHHJSON struct {
	Centimos int64  `json:"centimos"`
	Moneda   string `json:"moneda"`
}

type analisisRRHHJSON struct {
	ModalidadClave    string           `json:"modalidad_clave"`
	CategoriaRef      string           `json:"categoria_ref"`
	CausaClave        string           `json:"causa_clave"`
	PeriodoInicio     string           `json:"periodo_inicio"`
	PeriodoFin        string           `json:"periodo_fin"`
	PorcentajeJornada uint16           `json:"porcentaje_jornada"`
	ResultadoRC       string           `json:"resultado_rc"`
	CostePrevisto     *importeRRHHJSON `json:"coste_previsto,omitempty"`
	FuenteCosteRef    string           `json:"fuente_coste_ref,omitempty"`
}

type comprobacionRRHHJSON struct {
	Clave     string `json:"clave"`
	Resultado string `json:"resultado"`
}

type coberturaRRHHJSON struct {
	ViaClave          string                 `json:"via_clave"`
	DecisionGobernada bool                   `json:"decision_gobernada"`
	ProcedimientoRef  string                 `json:"procedimiento_ref,omitempty"`
	BolsaRef          string                 `json:"bolsa_ref,omitempty"`
	Comprobaciones    []comprobacionRRHHJSON `json:"comprobaciones"`
}

type asignacionRRHHJSON struct {
	UnidadRef   string `json:"unidad_ref"`
	AsignadaEn  string `json:"asignada_en"`
	MotivoClave string `json:"motivo_clave,omitempty"`
}

type hitoExpedienteRRHHJSON struct {
	Secuencia         uint64 `json:"secuencia"`
	VersionExpediente uint64 `json:"version_expediente"`
	AccionClave       string `json:"accion_clave"`
	RealizadaEn       string `json:"realizada_en"`
	FaseOrigen        string `json:"fase_origen,omitempty"`
	FaseDestino       string `json:"fase_destino"`
	EstadoOrigen      string `json:"estado_origen"`
	EstadoDestino     string `json:"estado_destino"`
}

func proyectarDetalleRRHH(entrada ports.DetalleExpedienteRRHH) detalleRRHHJSON {
	salida := detalleRRHHJSON{
		Esquema: esquemaConsultaDetalleRRHH,
		Resumen: proyectarResumenRRHH(entrada.Resumen),
		Solicitud: solicitudRRHHJSON{
			GrupoSubgrupo: entrada.Solicitud.GrupoSubgrupo,
			MotivoClave:   string(entrada.Solicitud.MotivoClave),
			PeriodoInicio: instanteConsultaRRHH(entrada.Solicitud.PeriodoInicio),
			PeriodoFin:    instanteConsultaRRHH(entrada.Solicitud.PeriodoFin),
		},
		Hitos: make([]hitoExpedienteRRHHJSON, len(entrada.Hitos)),
	}
	if entrada.Analisis != nil {
		salida.Analisis = proyectarAnalisisRRHH(*entrada.Analisis)
	}
	if entrada.Cobertura != nil {
		salida.Cobertura = proyectarCoberturaRRHH(*entrada.Cobertura)
	}
	if entrada.Asignacion != nil {
		salida.Asignacion = &asignacionRRHHJSON{
			UnidadRef:   entrada.Asignacion.UnidadRef,
			AsignadaEn:  instanteConsultaRRHH(entrada.Asignacion.AsignadaEn),
			MotivoClave: string(entrada.Asignacion.MotivoClave),
		}
	}
	for indice, hito := range entrada.Hitos {
		salida.Hitos[indice] = hitoExpedienteRRHHJSON{
			Secuencia: hito.Secuencia, VersionExpediente: hito.VersionExpediente,
			AccionClave: string(hito.AccionClave),
			RealizadaEn: instanteConsultaRRHH(hito.RealizadaEn),
			FaseOrigen:  string(hito.FaseOrigen), FaseDestino: string(hito.FaseDestino),
			EstadoOrigen:  string(hito.EstadoOrigen),
			EstadoDestino: string(hito.EstadoDestino),
		}
	}
	return salida
}

// detalleConsultaRRHHPublicable comprueba la superficie que va a cruzar HTTP.
// La autoridad y el recibo V2 siguen validados por aplicación/persistencia y
// nunca se trasladan a esta proyección.
func detalleConsultaRRHHPublicable(
	entrada ports.DetalleExpedienteRRHH,
	solicitud ports.SolicitudDetalleRRHH,
) bool {
	if entrada.Resumen.Validar() != nil ||
		entrada.Resumen.ExpedienteRef != solicitud.ExpedienteRef() ||
		(solicitud.VersionObservada() != 0 &&
			entrada.Resumen.Version != solicitud.VersionObservada()) ||
		!domain.GrupoSubgrupoValido(entrada.Solicitud.GrupoSubgrupo) ||
		!entrada.Solicitud.MotivoClave.Valida() ||
		!intervaloConsultaRRHHCanonico(
			entrada.Solicitud.PeriodoInicio,
			entrada.Solicitud.PeriodoFin,
		) ||
		len(entrada.Hitos) == 0 ||
		uint64(len(entrada.Hitos)) != entrada.Resumen.Version {
		return false
	}
	if entrada.Analisis != nil &&
		!intervaloConsultaRRHHCanonico(
			entrada.Analisis.PeriodoInicio,
			entrada.Analisis.PeriodoFin,
		) {
		return false
	}
	if entrada.Asignacion != nil &&
		!domain.InstanteUTCCanonico(entrada.Asignacion.AsignadaEn) {
		return false
	}
	for indice, hito := range entrada.Hitos {
		if hito.Secuencia != uint64(indice+1) ||
			hito.VersionExpediente != hito.Secuencia ||
			!hito.AccionClave.Valida() ||
			!domain.InstanteUTCCanonico(hito.RealizadaEn) ||
			!hito.FaseDestino.Valida() ||
			(hito.FaseOrigen != "" && !hito.FaseOrigen.Valida()) ||
			!hito.EstadoOrigen.Valido() ||
			!hito.EstadoDestino.Valido() {
			return false
		}
	}
	return true
}

func intervaloConsultaRRHHCanonico(inicio, fin time.Time) bool {
	return domain.InstanteUTCCanonico(inicio) &&
		domain.InstanteUTCCanonico(fin) &&
		!fin.Before(inicio)
}

func proyectarAnalisisRRHH(entrada ports.AnalisisOperativoRRHH) *analisisRRHHJSON {
	salida := &analisisRRHHJSON{
		ModalidadClave: string(entrada.ModalidadClave),
		CategoriaRef:   entrada.CategoriaRef, CausaClave: string(entrada.CausaClave),
		PeriodoInicio:     instanteConsultaRRHH(entrada.PeriodoInicio),
		PeriodoFin:        instanteConsultaRRHH(entrada.PeriodoFin),
		PorcentajeJornada: uint16(entrada.PorcentajeJornada),
		ResultadoRC:       string(entrada.ResultadoRC),
		FuenteCosteRef:    entrada.FuenteCosteRef,
	}
	if entrada.CostePrevisto != nil {
		salida.CostePrevisto = &importeRRHHJSON{
			Centimos: entrada.CostePrevisto.Centimos,
			Moneda:   entrada.CostePrevisto.Moneda,
		}
	}
	return salida
}

func proyectarCoberturaRRHH(
	entrada ports.CoberturaOperativaRRHH,
) *coberturaRRHHJSON {
	salida := &coberturaRRHHJSON{
		ViaClave:          string(entrada.ViaClave),
		DecisionGobernada: entrada.DecisionGobernada,
		ProcedimientoRef:  entrada.ProcedimientoRef, BolsaRef: entrada.BolsaRef,
		Comprobaciones: make([]comprobacionRRHHJSON, len(entrada.Comprobaciones)),
	}
	for indice, comprobacion := range entrada.Comprobaciones {
		salida.Comprobaciones[indice] = comprobacionRRHHJSON{
			Clave:     string(comprobacion.Clave),
			Resultado: string(comprobacion.Resultado),
		}
	}
	return salida
}
