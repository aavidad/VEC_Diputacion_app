package bootstrap

import (
	"context"
	"log"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// avisosViaCoberturaConfigurable es la parte del presentador de cobertura que
// activa las comprobaciones automáticas de la vía.
type avisosViaCoberturaConfigurable interface {
	ConfigurarAvisosVia(*application.EvaluadorAvisosViaCobertura) error
}

// configurarAvisosViaCoberturaDesarrollo activa en la fase 3 las propuestas de
// bolsa agotada, oferta al Servicio Andaluz de Empleo y nueva convocatoria.
// Sin catálogo de reglas de Bolsa o sin bolsas constituidas legibles no se
// activa nada y la propuesta de cobertura se presenta como hasta ahora.
func configurarAvisosViaCoberturaDesarrollo(
	autoridad *autoridadConsultasContratacionTemporalDesarrollo,
	reglasBolsa *reglas.Resolutor,
	fuente *fuenteConstituidaRRHHDesarrollo,
) {
	if autoridad == nil || fuente == nil {
		return
	}
	// Fuente de cobertura y avisos comparten la misma lectura memorizada:
	// una sola carga de Bolsa por propuesta.
	var situacion ports.ConsultaSituacionBolsaCobertura = situacionBolsaCoberturaDesarrollo{fuente: fuente}
	if autoridad.situacionBolsaCobertura != nil {
		autoridad.situacionBolsaCobertura.fijar(situacion)
		situacion = autoridad.situacionBolsaCobertura
	}
	if autoridad.presentadorCobertura == nil || reglasBolsa == nil {
		return
	}
	evaluador, err := application.NuevoEvaluadorAvisosViaCobertura(
		situacion, reglasBolsa, relojSistemaAvisosViaDesarrollo{},
	)
	if err == nil {
		err = autoridad.presentadorCobertura.ConfigurarAvisosVia(evaluador)
	}
	if err != nil {
		log.Printf("contratacion temporal: avisos de via de cobertura no compuestos")
	}
}

type relojSistemaAvisosViaDesarrollo struct{}

func (relojSistemaAvisosViaDesarrollo) Ahora() time.Time { return time.Now() }

// situacionBolsaCoberturaDesarrollo traduce las lecturas propias de Bolsa
// (bolsas constituidas y situación vigente de cada participación) al resumen
// que necesita Contratación temporal: solo recuentos y fecha de constitución.
type situacionBolsaCoberturaDesarrollo struct {
	fuente *fuenteConstituidaRRHHDesarrollo
	ahora  func() time.Time
}

func (s situacionBolsaCoberturaDesarrollo) SituacionBolsaCobertura(
	ctx context.Context,
	categoriaRef string,
) (ports.SituacionBolsaCobertura, error) {
	if s.fuente == nil || ctx == nil || categoriaRef == "" {
		return ports.SituacionBolsaCobertura{}, ports.ErrSituacionBolsaCoberturaNoDisponible
	}
	// Solo hacen falta recuentos por estado: el resumen del cuadro de Bolsa
	// (una lectura de conjunto con B82/B85) basta y no descifra el acta.
	// Un fallo queda «no disponible»; nunca se sirve una lectura anterior.
	datos, err := s.fuente.cargarResumen(ctx)
	if err != nil {
		log.Printf("contratacion temporal: situacion de bolsa no legible; causa=%s", causaFalloPostgreSQLCTDesarrollo(err))
		return ports.SituacionBolsaCobertura{}, ports.ErrSituacionBolsaCoberturaNoDisponible
	}
	ahora := time.Now()
	if s.ahora != nil {
		ahora = s.ahora()
	}
	return resumirSituacionBolsaCobertura(datos, categoriaRef, ahora)
}

func resumirSituacionBolsaCobertura(
	datos datasetBolsasRRHHDesarrollo,
	categoriaRef string,
	ahora time.Time,
) (ports.SituacionBolsaCobertura, error) {
	var situacion ports.SituacionBolsaCobertura
	bolsas := map[string]struct{}{}
	for _, bolsa := range datos.Bolsas {
		if bolsa.CategoriaRef != categoriaRef {
			continue
		}
		desde, err := time.Parse(time.RFC3339, bolsa.VigenteDesde)
		if err != nil {
			return ports.SituacionBolsaCobertura{}, ports.ErrSituacionBolsaCoberturaNoDisponible
		}
		bolsas[bolsa.Referencia] = struct{}{}
		if !situacion.Existe || desde.After(situacion.ConstituidaEn) {
			situacion.Existe, situacion.BolsaRef, situacion.ConstituidaEn = true, bolsa.Referencia, desde
		}
	}
	// El dataset de desarrollo no acredita la ausencia de una bolsa para una
	// categoría sin referencia exacta. La falta de vínculo queda no disponible.
	if len(bolsas) == 0 {
		return ports.SituacionBolsaCobertura{}, ports.ErrSituacionBolsaCoberturaNoDisponible
	}
	for _, candidatura := range datos.Candidaturas {
		if _, propia := bolsas[candidatura.BolsaRef]; !propia {
			continue
		}
		situacion.Integrantes++
		if candidaturaDisponibleCobertura(candidatura.Estado, candidatura.Disponible, ahora) {
			situacion.Disponibles++
		}
	}
	return situacion, situacion.Validar()
}

// candidaturaDisponibleCobertura aplica el art. 9 del Reglamento: puede ser
// llamada quien está disponible o ya ha alcanzado su fecha de disponibilidad.
func candidaturaDisponibleCobertura(estado string, disponibleDesde *string, ahora time.Time) bool {
	switch estado {
	case dominiobolsa.SituacionDisponible:
		return true
	case dominiobolsa.SituacionDisponibleDesde:
		if disponibleDesde == nil {
			return false
		}
		desde, err := time.Parse(time.RFC3339, *disponibleDesde)
		return err == nil && !ahora.Before(desde)
	}
	return false
}
