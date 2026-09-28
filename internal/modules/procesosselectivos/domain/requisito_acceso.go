// Package domain contiene reglas propias de los procesos selectivos.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrRequisitoAccesoInvalido = errors.New("procesos selectivos: requisito de acceso invalido")

type TipoProceso string

const (
	ProcesoOPE            TipoProceso = "ope"
	ProcesoBolsaInmediata TipoProceso = "bolsa_inmediata"
	ProcesoSelectivoOtro  TipoProceso = "selectivo_otro"
)

type HitoCumplimiento string

const (
	HitoSolicitud      HitoCumplimiento = "solicitud"
	HitoFinPlazo       HitoCumplimiento = "fin_plazo"
	HitoInicioProceso  HitoCumplimiento = "inicio_proceso"
	HitoPrueba         HitoCumplimiento = "prueba"
	HitoIncorporacion  HitoCumplimiento = "incorporacion"
	HitoFechaExplicita HitoCumplimiento = "fecha_explicita"
)

type ClaseRequisito string

const (
	RequisitoTitulacion ClaseRequisito = "titulacion"
	RequisitoOtro       ClaseRequisito = "otro"
)

// RequisitoAcceso fija una condición de las bases, no una regla de puntuación.
// FechaLimite es una fecha civil inclusiva resuelta al publicar la versión.
// La fuente pública contiene el texto legal; CodigoEstructurado identifica
// un criterio aprobado, nunca se deduce del texto mediante IA.
type RequisitoAcceso struct {
	Referencia             string
	BasesRef               string
	BasesVersion           uint64
	BasesHuellaSHA256      string
	TipoProceso            TipoProceso
	Clase                  ClaseRequisito
	CodigoEstructurado     string
	Hito                   HitoCumplimiento
	FechaLimite            string
	PermiteTituloPendiente bool
	EvidenciaPrevisionRef  string
}

var fechaCivil = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var huellaSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (r RequisitoAcceso) Validar() error {
	if !referenciaValida(r.Referencia) || !referenciaValida(r.BasesRef) ||
		r.BasesVersion == 0 || !huellaSHA256.MatchString(r.BasesHuellaSHA256) ||
		(r.CodigoEstructurado != "" && !referenciaValida(r.CodigoEstructurado)) || !fechaValida(r.FechaLimite) {
		return ErrRequisitoAccesoInvalido
	}
	switch r.TipoProceso {
	case ProcesoOPE, ProcesoBolsaInmediata, ProcesoSelectivoOtro:
	default:
		return ErrRequisitoAccesoInvalido
	}
	switch r.Hito {
	case HitoSolicitud, HitoFinPlazo, HitoInicioProceso, HitoPrueba, HitoIncorporacion, HitoFechaExplicita:
	default:
		return ErrRequisitoAccesoInvalido
	}
	if r.Clase != RequisitoTitulacion && r.Clase != RequisitoOtro {
		return ErrRequisitoAccesoInvalido
	}
	if r.PermiteTituloPendiente {
		if r.Clase != RequisitoTitulacion || r.TipoProceso != ProcesoOPE || !referenciaValida(r.EvidenciaPrevisionRef) {
			return ErrRequisitoAccesoInvalido
		}
	} else if r.EvidenciaPrevisionRef != "" {
		return ErrRequisitoAccesoInvalido
	}
	return nil
}

type EstadoHecho string

const (
	HechoDeclarado  EstadoHecho = "declarado"
	HechoPendiente  EstadoHecho = "pendiente"
	HechoAcreditado EstadoHecho = "acreditado"
	HechoRechazado  EstadoHecho = "rechazado"
)

// HechoAcceso es una proyección mínima del expediente común de méritos.
// Su existencia no acredita que la fuente o la persona hayan sido autorizadas.
type HechoAcceso struct {
	CodigoEstructurado    string
	Estado                EstadoHecho
	CumplidoEn            string
	FuenteRef             string
	EvidenciaRef          string
	PrevisionEn           string
	EvidenciaPrevisionRef string
}

type EstadoEvaluacion string

const (
	EvaluacionCumple    EstadoEvaluacion = "cumple"
	EvaluacionNoCumple  EstadoEvaluacion = "no_cumple"
	EvaluacionPendiente EstadoEvaluacion = "pendiente"
)

type MotivoEvaluacion string

const (
	MotivoAcreditado       MotivoEvaluacion = "acreditado_en_hito"
	MotivoAcreditadoTarde  MotivoEvaluacion = "acreditado_despues_hito"
	MotivoRechazado        MotivoEvaluacion = "hecho_rechazado"
	MotivoFaltaDato        MotivoEvaluacion = "dato_ausente_o_sin_acreditar"
	MotivoTextoSinCriterio MotivoEvaluacion = "texto_sin_criterio_estructurado"
	MotivoPrevision        MotivoEvaluacion = "cumplimiento_previsto"
	MotivoPrevisionTarde   MotivoEvaluacion = "prevision_despues_hito"
)

type EvaluacionAcceso struct {
	Estado      EstadoEvaluacion
	Motivo      MotivoEvaluacion
	FuenteRef   string
	FechaHito   string
	Condicional bool
}

// EvaluarAcceso no decide admisión ni inscripción. La previsión permitida es
// condicional y nunca convierte una titulación futura en hecho acreditado.
func EvaluarAcceso(r RequisitoAcceso, hecho *HechoAcceso) (EvaluacionAcceso, error) {
	if err := r.Validar(); err != nil {
		return EvaluacionAcceso{}, err
	}
	resultado := EvaluacionAcceso{Estado: EvaluacionPendiente, Motivo: MotivoFaltaDato, FechaHito: r.FechaLimite}
	if r.CodigoEstructurado == "" {
		resultado.Motivo = MotivoTextoSinCriterio
		return resultado, nil
	}
	if hecho == nil {
		return resultado, nil
	}
	if hecho.CodigoEstructurado != r.CodigoEstructurado || !referenciaValida(hecho.FuenteRef) {
		return EvaluacionAcceso{}, ErrRequisitoAccesoInvalido
	}
	resultado.FuenteRef = hecho.FuenteRef
	switch hecho.Estado {
	case HechoAcreditado:
		if !fechaValida(hecho.CumplidoEn) || !referenciaValida(hecho.EvidenciaRef) {
			return EvaluacionAcceso{}, ErrRequisitoAccesoInvalido
		}
		if hecho.CumplidoEn <= r.FechaLimite {
			resultado.Estado, resultado.Motivo = EvaluacionCumple, MotivoAcreditado
		} else {
			resultado.Estado, resultado.Motivo = EvaluacionNoCumple, MotivoAcreditadoTarde
		}
	case HechoRechazado:
		resultado.Estado, resultado.Motivo = EvaluacionNoCumple, MotivoRechazado
	case HechoDeclarado, HechoPendiente:
		if hecho.PrevisionEn == "" && hecho.EvidenciaPrevisionRef == "" {
			return resultado, nil
		}
		if !fechaValida(hecho.PrevisionEn) || !referenciaValida(hecho.EvidenciaPrevisionRef) {
			return EvaluacionAcceso{}, ErrRequisitoAccesoInvalido
		}
		if !r.PermiteTituloPendiente || hecho.EvidenciaPrevisionRef != r.EvidenciaPrevisionRef {
			return resultado, nil
		}
		if hecho.PrevisionEn <= r.FechaLimite {
			resultado.Motivo, resultado.Condicional = MotivoPrevision, true
		} else {
			resultado.Estado, resultado.Motivo = EvaluacionNoCumple, MotivoPrevisionTarde
		}
	default:
		return EvaluacionAcceso{}, ErrRequisitoAccesoInvalido
	}
	return resultado, nil
}

func referenciaValida(valor string) bool {
	return valor != "" && valor == strings.TrimSpace(valor) && len(valor) <= 160 && !strings.ContainsAny(valor, "\r\n\t")
}

func fechaValida(valor string) bool {
	if !fechaCivil.MatchString(valor) {
		return false
	}
	fecha, err := time.Parse("2006-01-02", valor)
	return err == nil && fecha.Format("2006-01-02") == valor
}
