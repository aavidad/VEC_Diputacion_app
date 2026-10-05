package observabilidad

import (
	"context"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	_ ports.EmisorResultadosTecnicosConContexto       = (*EmisorJSONLines)(nil)
	_ ports.ConsultaMetricasEmisionResultadosTecnicos = (*EmisorJSONLines)(nil)
)

// EmitirResultadoConContexto registra solo un resultado ya observado. Una
// cancelación del llamante no borra su correlación privada. Sin ella se cuenta
// la pérdida y jamás se crea otra referencia que parezca ligada al efecto.
func (e *EmisorJSONLines) EmitirResultadoConContexto(ctx context.Context, solicitud domain.SolicitudResultadoTecnico) {
	if e == nil {
		return
	}
	e.enVuelo.Add(1)
	defer e.enVuelo.Add(-1)
	if e.cerrado.Load() {
		e.resultadosDescartados.Add(1)
		return
	}
	clasificacion, err := domain.ClasificarResultadoTecnico(solicitud)
	if err != nil {
		e.contarResultadoInvalido(err)
		return
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		e.resultadosSinCorrelacion.Add(1)
		return
	}
	referencia, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		e.contarCorrelacionResultadoNoDisponible(err)
		return
	}
	select {
	case e.cola <- elementoCola{
		clasificacionResultado: clasificacion, esResultado: true,
		instante: e.reloj(), correlacion: correlacion, referencia: referencia,
	}:
		e.resultadosAceptados.Add(1)
	default:
		e.resultadosDescartados.Add(1)
	}
}

func (e *EmisorJSONLines) MetricasResultadosTecnicos() ports.MetricasEmisionResultadosTecnicos {
	if e == nil {
		return ports.MetricasEmisionResultadosTecnicos{}
	}
	return ports.MetricasEmisionResultadosTecnicos{
		Aceptados:       e.resultadosAceptados.Load(),
		Descartados:     e.resultadosDescartados.Load(),
		Invalidos:       e.resultadosInvalidos.Load(),
		SinCorrelacion:  e.resultadosSinCorrelacion.Load(),
		Escritos:        e.resultadosEscritos.Load(),
		FallosEscritura: e.resultadosFallosEscritura.Load(),
	}
}

// lineaResultado no incorpora mensaje, error, identidad ni recurso.
type lineaResultado struct {
	Esquema        string `json:"esquema"`
	Instante       string `json:"instante"`
	Resultado      string `json:"resultado"`
	Nivel          string `json:"nivel"`
	Componente     string `json:"componente"`
	Etapa          string `json:"etapa"`
	Entorno        string `json:"entorno"`
	VersionBinario string `json:"version_binario"`
	Correlacion    string `json:"correlacion"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func (e *EmisorJSONLines) escribirResultado(
	clasificacion domain.ClasificacionResultadoTecnico, instante time.Time,
	correlacion string, referencia domain.ReferenciaCorrelacionAutorizacionV2,
) {
	resultado, err := domain.NuevoResultadoTecnico(
		clasificacion, instante, e.entorno, e.version, correlacion, referencia,
	)
	if err != nil {
		e.contarResultadoInvalido(err)
		return
	}
	linea := lineaResultado{
		Esquema: resultado.Esquema, Instante: resultado.Instante.Format(formatoInstante),
		Resultado: string(resultado.Resultado), Nivel: string(resultado.Nivel),
		Componente: string(resultado.Componente), Etapa: string(resultado.Etapa),
		Entorno: string(resultado.Entorno), VersionBinario: resultado.VersionBinario,
		Correlacion: resultado.Correlacion, CorrelacionRef: resultado.CorrelacionRef,
	}
	datos, err := json.Marshal(linea)
	if err != nil {
		e.contarFalloEscrituraResultado(err)
		return
	}
	if !e.escribirLinea(datos) {
		e.resultadosFallosEscritura.Add(1)
		return
	}
	e.resultadosEscritos.Add(1)
}

func (e *EmisorJSONLines) contarResultadoInvalido(err error) {
	if err != nil {
		e.resultadosInvalidos.Add(1)
	}
}

func (e *EmisorJSONLines) contarCorrelacionResultadoNoDisponible(err error) {
	if err != nil {
		e.resultadosSinCorrelacion.Add(1)
	}
}

func (e *EmisorJSONLines) contarFalloEscrituraResultado(err error) {
	if err != nil {
		e.resultadosFallosEscritura.Add(1)
	}
}
