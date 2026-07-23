package domain

import (
	"strings"
	"time"
)

const (
	// JornadaCompletaDiezmilesimas representa el 100 % sin recurrir a coma
	// flotante. Las bases y acuerdos determinan las jornadas admitidas; el
	// dominio solo impide valores nulos o superiores a una jornada completa.
	JornadaCompletaDiezmilesimas JornadaDiezmilesimas = 10_000

	// Estos límites son técnicos, no reglas laborales. Evitan que un periodo o
	// importe hostil provoque recorridos o multiplicaciones no representables.
	maximoAniosPeriodoAnalisis              = 100
	maximoCentimosCalculablesAnalisis int64 = 922_337_203_685_477
)

// JornadaDiezmilesimas conserva de forma exacta cualquier jornada entre
// 0,01 % y 100 %. Su interpretación laboral procede del catálogo versionado.
type JornadaDiezmilesimas uint16

func (j JornadaDiezmilesimas) Validar() error {
	if j == 0 || j > JornadaCompletaDiezmilesimas {
		return ErrDatoInvalido
	}
	return nil
}

type ResultadoValidacionRC string

const (
	RCValidada    ResultadoValidacionRC = "validada"
	RCNoRequerida ResultadoValidacionRC = "no_requerida"
	RCRechazada   ResultadoValidacionRC = "rechazada"
)

func (r ResultadoValidacionRC) valido() bool {
	return r == RCValidada || r == RCNoRequerida || r == RCRechazada
}

type ValidacionRC struct {
	Resultado           ResultadoValidacionRC `json:"resultado"`
	EntradaRef          string                `json:"entrada_ref"`
	HuellaEntradaSHA256 string                `json:"huella_entrada_sha256"`
	FuenteRef           string                `json:"fuente_ref"`
	ReciboRef           string                `json:"recibo_ref"`
	ValidadaEn          time.Time             `json:"validada_en"`
	FechaRC             time.Time             `json:"fecha_rc,omitempty"`
	Numero              string                `json:"numero,omitempty"`
	Importe             Importe               `json:"importe,omitempty"`
	DocumentoRef        string                `json:"documento_ref,omitempty"`
	Motivo              string                `json:"motivo,omitempty"`
}

func (v ValidacionRC) Validar() error {
	if !v.Resultado.valido() || !referenciaValida(v.EntradaRef) ||
		!huellaEntradaValida(v.HuellaEntradaSHA256) || !referenciaValida(v.FuenteRef) ||
		!referenciaValida(v.ReciboRef) || !instanteCanonico(v.ValidadaEn) ||
		!textoValido(v.Motivo, 1000, true) {
		return ErrDatoInvalido
	}
	if v.Resultado == RCValidada {
		if !fechaCivilCanonica(v.FechaRC) || !referenciaValida(v.Numero) ||
			!importeCalculable(v.Importe) ||
			!referenciaValida(v.DocumentoRef) {
			return ErrDatoInvalido
		}
		return nil
	}
	if !v.FechaRC.IsZero() || v.Numero != "" || v.Importe != (Importe{}) ||
		v.DocumentoRef != "" || v.Motivo == "" {
		return ErrDatoInvalido
	}
	return nil
}

// HabilitaAvance distingue una decisión negativa registrable de una decisión
// que permite continuar el expediente. La ausencia de ValidacionRC representa
// el estado pendiente y, al ser inválida, tampoco habilita el avance.
func (v ValidacionRC) HabilitaAvance() bool {
	return v.Validar() == nil && (v.Resultado == RCValidada || v.Resultado == RCNoRequerida)
}

type AnalisisRRHH struct {
	ModalidadClave    ClaveCatalogo        `json:"modalidad_clave"`
	CategoriaRef      string               `json:"categoria_ref"`
	GrupoSubgrupo     string               `json:"grupo_subgrupo"`
	CausaClave        ClaveCatalogo        `json:"causa_clave"`
	Periodo           PeriodoPrevisto      `json:"periodo"`
	PorcentajeJornada JornadaDiezmilesimas `json:"porcentaje_jornada"`
	ValidacionRC      ValidacionRC         `json:"validacion_rc"`
	CostePrevisto     *Importe             `json:"coste_previsto,omitempty"`
	FuenteCosteRef    string               `json:"fuente_coste_ref,omitempty"`
	Observaciones     string               `json:"observaciones,omitempty"`
}

func (a AnalisisRRHH) Validar() error {
	if !a.ModalidadClave.Valida() || !referenciaValida(a.CategoriaRef) ||
		!grupoValido(a.GrupoSubgrupo) || !a.CausaClave.Valida() ||
		!periodoAnalisisValido(a.Periodo) || a.PorcentajeJornada.Validar() != nil ||
		a.ValidacionRC.Validar() != nil ||
		!textoValido(a.Observaciones, 4000, true) {
		return ErrDatoInvalido
	}
	if a.CostePrevisto == nil {
		if a.FuenteCosteRef != "" {
			return ErrDatoInvalido
		}
		return nil
	}
	if !importeCalculable(*a.CostePrevisto) || !referenciaValida(a.FuenteCosteRef) {
		return ErrDatoInvalido
	}
	if a.ValidacionRC.Resultado == RCValidada &&
		a.CostePrevisto.Centimos > a.ValidacionRC.Importe.Centimos {
		return ErrDatoInvalido
	}
	return nil
}

// HabilitaAvance indica si el análisis es íntegro y su decisión presupuestaria
// permite pasar a cobertura. Una RC rechazada conserva su evidencia, pero
// devuelve false.
func (a AnalisisRRHH) HabilitaAvance() bool {
	return a.Validar() == nil && a.ValidacionRC.HabilitaAvance()
}

func periodoAnalisisValido(periodo PeriodoPrevisto) bool {
	if periodo.Validar() != nil {
		return false
	}
	return !periodo.Fin.After(periodo.Inicio.AddDate(maximoAniosPeriodoAnalisis, 0, 0))
}

func importeCalculable(importe Importe) bool {
	return importe.Validar(false) == nil &&
		importe.Centimos <= maximoCentimosCalculablesAnalisis
}

func huellaEntradaValida(huella string) bool {
	return patronHuella.MatchString(huella) && huella != strings.Repeat("0", 64)
}

func (a AnalisisRRHH) clonar() AnalisisRRHH {
	if a.CostePrevisto != nil {
		importe := *a.CostePrevisto
		a.CostePrevisto = &importe
	}
	return a
}

// Clonar valida y entrega una copia defensiva. El importe opcional nunca
// comparte su puntero con el valor aportado por quien invoca el dominio.
func (a AnalisisRRHH) Clonar() (AnalisisRRHH, error) {
	if a.Validar() != nil {
		return AnalisisRRHH{}, ErrDatoInvalido
	}
	return a.clonar(), nil
}
