package domain

import "time"

type ResultadoValidacionRC string

const (
	RCValidada    ResultadoValidacionRC = "validada"
	RCNoValidada  ResultadoValidacionRC = "no_validada"
	RCNoRequerida ResultadoValidacionRC = "no_requerida"
)

func (r ResultadoValidacionRC) valido() bool {
	return r == RCValidada || r == RCNoValidada || r == RCNoRequerida
}

type ValidacionRC struct {
	Resultado    ResultadoValidacionRC `json:"resultado"`
	FuenteRef    string                `json:"fuente_ref"`
	ReciboRef    string                `json:"recibo_ref"`
	ValidadaEn   time.Time             `json:"validada_en"`
	Numero       string                `json:"numero,omitempty"`
	Importe      Importe               `json:"importe,omitempty"`
	DocumentoRef string                `json:"documento_ref,omitempty"`
	Motivo       string                `json:"motivo,omitempty"`
}

func (v ValidacionRC) Validar() error {
	if !v.Resultado.valido() || !referenciaValida(v.FuenteRef) ||
		!referenciaValida(v.ReciboRef) || !instanteCanonico(v.ValidadaEn) ||
		!textoValido(v.Motivo, 1000, true) {
		return ErrDatoInvalido
	}
	if v.Resultado == RCValidada {
		if !referenciaValida(v.Numero) || v.Importe.Validar(false) != nil ||
			!referenciaValida(v.DocumentoRef) {
			return ErrDatoInvalido
		}
		return nil
	}
	if v.Numero != "" || v.Importe != (Importe{}) || v.DocumentoRef != "" ||
		v.Motivo == "" {
		return ErrDatoInvalido
	}
	return nil
}

type AnalisisRRHH struct {
	ModalidadClave    ClaveCatalogo   `json:"modalidad_clave"`
	CategoriaRef      string          `json:"categoria_ref"`
	GrupoSubgrupo     string          `json:"grupo_subgrupo"`
	CausaClave        ClaveCatalogo   `json:"causa_clave"`
	Periodo           PeriodoPrevisto `json:"periodo"`
	PorcentajeJornada uint16          `json:"porcentaje_jornada"`
	ValidacionRC      ValidacionRC    `json:"validacion_rc"`
	CostePrevisto     *Importe        `json:"coste_previsto,omitempty"`
	FuenteCosteRef    string          `json:"fuente_coste_ref,omitempty"`
	Observaciones     string          `json:"observaciones,omitempty"`
}

func (a AnalisisRRHH) Validar() error {
	if !a.ModalidadClave.Valida() || !referenciaValida(a.CategoriaRef) ||
		!grupoValido(a.GrupoSubgrupo) || !a.CausaClave.Valida() ||
		a.Periodo.Validar() != nil || a.PorcentajeJornada == 0 ||
		a.PorcentajeJornada > 10000 || a.ValidacionRC.Validar() != nil ||
		!textoValido(a.Observaciones, 4000, true) {
		return ErrDatoInvalido
	}
	if a.CostePrevisto == nil {
		if a.FuenteCosteRef != "" {
			return ErrDatoInvalido
		}
		return nil
	}
	if a.CostePrevisto.Validar(false) != nil || !referenciaValida(a.FuenteCosteRef) {
		return ErrDatoInvalido
	}
	return nil
}

func (a AnalisisRRHH) clonar() AnalisisRRHH {
	if a.CostePrevisto != nil {
		importe := *a.CostePrevisto
		a.CostePrevisto = &importe
	}
	return a
}
