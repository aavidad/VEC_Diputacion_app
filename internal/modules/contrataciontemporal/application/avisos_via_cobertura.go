package application

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Comprobaciones automáticas de la vía de cobertura (fase 3): bolsa agotada
// provisionalmente, propuesta de oferta al Servicio Andaluz de Empleo con su
// duración máxima y aviso de nueva convocatoria. Son solo propuestas: la vía
// la sigue decidiendo RRHH con la decisión de cobertura. Todos los valores
// (umbral, meses, años) proceden del catálogo de reglas; sin catálogo no se
// evalúa nada y la pantalla queda como hoy.

var ErrEvaluadorAvisosViaCoberturaInvalido = errors.New(
	"contratacion temporal: evaluador de avisos de via de cobertura invalido",
)

// Claves de las reglas que se consultan y del atributo del umbral.
const (
	AtributoUmbralDisponibles = "umbral_disponibles"
	maximoUmbralDisponibles   = 10_000
	formatoFechaCivilAviso    = "2006-01-02"
)

type ClaveAvisoViaCobertura string

const (
	AvisoBolsaAgotadaProvisionalmente ClaveAvisoViaCobertura = "bolsa_agotada_provisionalmente"
	AvisoPropuestaOfertaSAE           ClaveAvisoViaCobertura = "propuesta_oferta_sae"
	AvisoNuevaConvocatoria            ClaveAvisoViaCobertura = "nueva_convocatoria"
)

type EstadoAvisosViaCobertura string

const (
	EstadoAvisosEvaluados    EstadoAvisosViaCobertura = "evaluados"
	EstadoAvisosSinBolsa     EstadoAvisosViaCobertura = "sin_bolsa"
	EstadoAvisosNoDisponible EstadoAvisosViaCobertura = "no_disponible"
)

// Motivos del aviso de nueva convocatoria.
const (
	MotivoVigenciaSuperada = "vigencia_superada"
	MotivoBolsaAgotada     = "bolsa_agotada"
)

// ProcedenciaReglaAviso es lo que la ayuda «?» muestra de cada regla aplicada.
type ProcedenciaReglaAviso struct {
	Clave        string
	Etiqueta     string
	Descripcion  string
	Origen       string
	Articulo     string
	Norma        string
	ParteEjemplo string
	Referencia   string
	Ejemplo      bool
}

// AvisoViaCobertura es una comprobación que se ha cumplido. Los campos que no
// aplican a su clave quedan a cero.
type AvisoViaCobertura struct {
	Clave ClaveAvisoViaCobertura
	// Bolsa agotada provisionalmente.
	Disponibles int
	Integrantes int
	Umbral      int
	// Oferta al Servicio Andaluz de Empleo.
	DuracionMaximaMeses int
	FinMaximo           string
	FinPrevisto         string
	ExcedeDuracion      bool
	// Nueva convocatoria.
	Motivos       []string
	ConstituidaEn string
	VigenciaHasta string
	Reglas        []ProcedenciaReglaAviso
}

// ResultadoAvisosViaCobertura acompaña a la propuesta de cobertura.
type ResultadoAvisosViaCobertura struct {
	Estado     EstadoAvisosViaCobertura
	Avisos     []AvisoViaCobertura
	EvaluadaEn time.Time
}

// ReglasAvisosViaCobertura es el subconjunto del resolutor de reglas que usa
// el evaluador; en la composición es el catálogo de reglas de Bolsa.
type ReglasAvisosViaCobertura interface {
	Reglas(context.Context) ([]reglas.Regla, error)
	Vencimiento(context.Context, string, time.Time, string) (reglas.Regla, reglas.Vencimiento, error)
}

type EvaluadorAvisosViaCobertura struct {
	situacion ports.ConsultaSituacionBolsaCobertura
	reglas    ReglasAvisosViaCobertura
	reloj     ports.Reloj
}

func NuevoEvaluadorAvisosViaCobertura(
	situacion ports.ConsultaSituacionBolsaCobertura,
	resolutor ReglasAvisosViaCobertura,
	reloj ports.Reloj,
) (*EvaluadorAvisosViaCobertura, error) {
	if dependenciaNula(situacion) || dependenciaNula(resolutor) || dependenciaNula(reloj) {
		return nil, ErrEvaluadorAvisosViaCoberturaInvalido
	}
	return &EvaluadorAvisosViaCobertura{situacion: situacion, reglas: resolutor, reloj: reloj}, nil
}

// Evaluar devuelve false si no hay catálogo de reglas: el consumidor no
// muestra nada. Cualquier otro fallo devuelve el estado «no_disponible», que
// nunca se presenta como bolsa vigente ni como bolsa agotada.
func (e *EvaluadorAvisosViaCobertura) Evaluar(
	ctx context.Context,
	categoriaRef string,
	periodo domain.PeriodoPrevisto,
) (ResultadoAvisosViaCobertura, bool) {
	if e == nil || ctx == nil || dependenciaNula(e.situacion) || dependenciaNula(e.reglas) || dependenciaNula(e.reloj) {
		return ResultadoAvisosViaCobertura{}, false
	}
	ahora := e.reloj.Ahora().UTC().Truncate(time.Microsecond)
	noDisponible := ResultadoAvisosViaCobertura{Estado: EstadoAvisosNoDisponible, EvaluadaEn: ahora}
	vigentes, err := e.reglas.Reglas(ctx)
	if errors.Is(err, reglas.ErrReglasNoConfiguradas) {
		return ResultadoAvisosViaCobertura{}, false
	}
	if err != nil {
		registrarAvisosViaNoDisponibles("reglas", err)
		return noDisponible, true
	}
	agotamiento, conAgotamiento := reglaPorClave(vigentes, reglas.BolsaAgotamiento)
	_, conSAE := reglaPorClave(vigentes, reglas.BolsaSAEDuracionMaxima)
	_, conVigencia := reglaPorClave(vigentes, reglas.BolsaVigencia)
	if !conAgotamiento && !conSAE && !conVigencia {
		return ResultadoAvisosViaCobertura{}, false
	}
	if !domain.ReferenciaOpacaValida(categoriaRef) || periodo.Validar() != nil {
		return noDisponible, true
	}
	situacion, err := e.situacion.SituacionBolsaCobertura(ctx, categoriaRef)
	if err != nil || situacion.Validar() != nil {
		registrarAvisosViaNoDisponibles("situacion_bolsa", err)
		return noDisponible, true
	}
	if !situacion.Existe {
		return ResultadoAvisosViaCobertura{Estado: EstadoAvisosSinBolsa, EvaluadaEn: ahora}, true
	}
	resultado := ResultadoAvisosViaCobertura{Estado: EstadoAvisosEvaluados, EvaluadaEn: ahora}
	agotada := false
	if conAgotamiento {
		umbral, ok := umbralDisponibles(agotamiento)
		if !ok {
			registrarAvisosViaNoDisponibles("umbral", reglas.ErrReglaInvalida)
			return noDisponible, true
		}
		if situacion.Disponibles <= umbral {
			agotada = true
			resultado.Avisos = append(resultado.Avisos, AvisoViaCobertura{
				Clave: AvisoBolsaAgotadaProvisionalmente, Disponibles: situacion.Disponibles,
				Integrantes: situacion.Integrantes, Umbral: umbral,
				Reglas: []ProcedenciaReglaAviso{procedenciaDe(agotamiento)},
			})
		}
	}
	if agotada && conSAE {
		aviso, ok := e.avisoSAE(ctx, periodo, agotamiento)
		if !ok {
			return noDisponible, true
		}
		resultado.Avisos = append(resultado.Avisos, aviso)
	}
	vencida, vigenciaHasta, reglaVigencia := false, "", reglas.Regla{}
	if conVigencia {
		regla, vencimiento, err := e.reglas.Vencimiento(ctx, reglas.BolsaVigencia, situacion.ConstituidaEn, reglas.MunicipioSedeDiputacion)
		if err != nil || vencimiento.UltimoDia == "" || vencimiento.VenceAntesDe.IsZero() {
			registrarAvisosViaNoDisponibles("vigencia", err)
			return noDisponible, true
		}
		vencida = !ahora.Before(vencimiento.VenceAntesDe)
		vigenciaHasta, reglaVigencia = vencimiento.UltimoDia, regla
	}
	if vencida || agotada {
		aviso := AvisoViaCobertura{
			Clave: AvisoNuevaConvocatoria, VigenciaHasta: vigenciaHasta,
			ConstituidaEn: situacion.ConstituidaEn.In(zonaAvisosVia()).Format(formatoFechaCivilAviso),
		}
		if vencida {
			aviso.Motivos = append(aviso.Motivos, MotivoVigenciaSuperada)
			aviso.Reglas = append(aviso.Reglas, procedenciaDe(reglaVigencia))
		}
		if agotada {
			aviso.Motivos = append(aviso.Motivos, MotivoBolsaAgotada)
			aviso.Reglas = append(aviso.Reglas, procedenciaDe(agotamiento))
		}
		resultado.Avisos = append(resultado.Avisos, aviso)
	}
	return resultado, true
}

func (e *EvaluadorAvisosViaCobertura) avisoSAE(
	ctx context.Context,
	periodo domain.PeriodoPrevisto,
	agotamiento reglas.Regla,
) (AvisoViaCobertura, bool) {
	regla, vencimiento, err := e.reglas.Vencimiento(ctx, reglas.BolsaSAEDuracionMaxima, periodo.Inicio, reglas.MunicipioSedeDiputacion)
	if err != nil || regla.Unidad != reglas.UnidadMeses || vencimiento.UltimoDia == "" {
		registrarAvisosViaNoDisponibles("duracion_sae", err)
		return AvisoViaCobertura{}, false
	}
	finPrevisto := periodo.Fin.UTC().Format(formatoFechaCivilAviso)
	return AvisoViaCobertura{
		Clave: AvisoPropuestaOfertaSAE, DuracionMaximaMeses: regla.Cantidad,
		FinMaximo: vencimiento.UltimoDia, FinPrevisto: finPrevisto,
		// Las fechas civiles AAAA-MM-DD se comparan como texto sin ambigüedad.
		ExcedeDuracion: finPrevisto > vencimiento.UltimoDia,
		Reglas:         []ProcedenciaReglaAviso{procedenciaDe(regla), procedenciaDe(agotamiento)},
	}, true
}

func reglaPorClave(vigentes []reglas.Regla, clave string) (reglas.Regla, bool) {
	for _, regla := range vigentes {
		if regla.Clave == clave {
			return regla, true
		}
	}
	return reglas.Regla{}, false
}

// umbralDisponibles lee el atributo del catálogo: la bolsa se considera
// agotada provisionalmente cuando las personas disponibles no lo superan.
func umbralDisponibles(regla reglas.Regla) (int, bool) {
	texto, existe := regla.Atributos[AtributoUmbralDisponibles]
	if !existe {
		return 0, false
	}
	umbral, errNumero := strconv.Atoi(texto)
	valido := errNumero == nil && umbral >= 0 && umbral <= maximoUmbralDisponibles && strconv.Itoa(umbral) == texto
	return umbral, valido
}

// registrarAvisosViaNoDisponibles deja constancia de que las comprobaciones
// no se pudieron hacer; la pantalla lo muestra como «no disponible».
func registrarAvisosViaNoDisponibles(etapa string, err error) {
	slog.Warn("contratacion temporal: avisos de via de cobertura no disponibles", "etapa", etapa, "error", err)
}

func procedenciaDe(regla reglas.Regla) ProcedenciaReglaAviso {
	return ProcedenciaReglaAviso{
		Clave: regla.Clave, Etiqueta: regla.Etiqueta, Descripcion: regla.Descripcion,
		Origen: string(regla.Origen), Articulo: regla.Articulo, Norma: regla.Norma,
		ParteEjemplo: regla.ParteEjemplo, Referencia: regla.Referencia, Ejemplo: regla.EsEjemplo(),
	}
}

func zonaAvisosVia() *time.Location {
	if zona, err := time.LoadLocation("Europe/Madrid"); err == nil {
		return zona
	}
	return time.UTC
}

// copiarAvisosViaCobertura evita compartir listas mutables con el adaptador.
func copiarAvisosViaCobertura(entrada *ResultadoAvisosViaCobertura) *ResultadoAvisosViaCobertura {
	if entrada == nil {
		return nil
	}
	copia := *entrada
	copia.Avisos = make([]AvisoViaCobertura, len(entrada.Avisos))
	for indice, aviso := range entrada.Avisos {
		aviso.Motivos = append([]string(nil), aviso.Motivos...)
		aviso.Reglas = append([]ProcedenciaReglaAviso(nil), aviso.Reglas...)
		copia.Avisos[indice] = aviso
	}
	return &copia
}

// avisosViaPresentacion guarda el evaluador opcional de la presentación. Se
// configura una sola vez al componer, antes de servir peticiones.
type avisosViaPresentacion struct {
	evaluador atomic.Pointer[EvaluadorAvisosViaCobertura]
}

// ConfigurarAvisosVia activa las comprobaciones automáticas de la vía. Sin
// esta llamada la propuesta se presenta como hasta ahora.
func (s *ServicioPresentacionPropuestaCobertura) ConfigurarAvisosVia(evaluador *EvaluadorAvisosViaCobertura) error {
	if s == nil || evaluador == nil || !s.avisosVia.evaluador.CompareAndSwap(nil, evaluador) {
		return ErrEvaluadorAvisosViaCoberturaInvalido
	}
	return nil
}

// evaluarAvisosVia nunca hace fallar la propuesta: los avisos son orientativos
// y su indisponibilidad se muestra como tal.
func (s *ServicioPresentacionPropuestaCobertura) evaluarAvisosVia(
	ctx context.Context,
	analisis *domain.AnalisisRRHH,
) *ResultadoAvisosViaCobertura {
	evaluador := s.avisosVia.evaluador.Load()
	if evaluador == nil || analisis == nil {
		return nil
	}
	resultado, evaluado := evaluador.Evaluar(ctx, strings.TrimSpace(analisis.CategoriaRef), analisis.Periodo)
	if !evaluado {
		return nil
	}
	return &resultado
}
