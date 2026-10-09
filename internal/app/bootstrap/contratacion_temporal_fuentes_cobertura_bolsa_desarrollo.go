package bootstrap

import (
	"context"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Comprobaciones de la vía de bolsa que responde la situación real de Bolsa.
// El resto de comprobaciones (SAE, nueva convocatoria) siguen sin fuente y se
// responden «no consta».
const (
	procedenciaBolsaCoberturaCT          domain.ClaveCatalogo = "bolsa"
	comprobacionExisteBolsaVigenteCT     domain.ClaveCatalogo = "existe_bolsa_vigente"
	comprobacionCandidaturasDisponibleCT domain.ClaveCatalogo = "hay_candidaturas_disponibles"
	validezSituacionBolsaCoberturaCT                          = 15 * time.Second
)

// situacionBolsaCoberturaFijable enlaza la fuente de cobertura con las
// lecturas propias de Bolsa. Se compone vacía junto a Contratación temporal y
// se fija cuando la composición ya dispone de las bolsas constituidas. Sin
// enlace o si Bolsa no responde, la comprobación queda «no consta»: nunca se
// interpreta como bolsa vigente ni como bolsa agotada.
type situacionBolsaCoberturaFijable struct {
	mu        sync.Mutex
	consulta  ports.ConsultaSituacionBolsaCobertura
	ahora     func() time.Time
	memoria   map[string]situacionBolsaCoberturaMemorizada
	enCurso   map[string]*sync.Mutex
	validezDe time.Duration
}

type situacionBolsaCoberturaMemorizada struct {
	situacion ports.SituacionBolsaCobertura
	hasta     time.Time
}

func (s *situacionBolsaCoberturaFijable) fijar(consulta ports.ConsultaSituacionBolsaCobertura) {
	if s == nil || dependenciaEsNulaContratacionTemporalDesarrollo(consulta) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consulta == nil {
		s.consulta = consulta
	}
}

// situacion devuelve la situación de Bolsa de la categoría. La preparación de
// una propuesta pregunta por varias comprobaciones a la vez: se memoriza unos
// segundos por categoría para leer Bolsa una sola vez por propuesta.
func (s *situacionBolsaCoberturaFijable) situacion(
	ctx context.Context,
	categoriaRef string,
) (ports.SituacionBolsaCobertura, bool) {
	if s == nil || ctx == nil || categoriaRef == "" {
		return ports.SituacionBolsaCobertura{}, false
	}
	s.mu.Lock()
	consulta := s.consulta
	if consulta == nil {
		s.mu.Unlock()
		return ports.SituacionBolsaCobertura{}, false
	}
	if s.enCurso == nil {
		s.enCurso = make(map[string]*sync.Mutex)
	}
	cerrojo := s.enCurso[categoriaRef]
	if cerrojo == nil {
		cerrojo = &sync.Mutex{}
		s.enCurso[categoriaRef] = cerrojo
	}
	s.mu.Unlock()

	cerrojo.Lock()
	defer cerrojo.Unlock()
	if memorizada, ok := s.memorizada(categoriaRef); ok {
		return memorizada, true
	}
	situacion, err := consulta.SituacionBolsaCobertura(ctx, categoriaRef)
	if err != nil || situacion.Validar() != nil {
		return ports.SituacionBolsaCobertura{}, false
	}
	s.mu.Lock()
	if s.memoria == nil {
		s.memoria = make(map[string]situacionBolsaCoberturaMemorizada)
	}
	s.memoria[categoriaRef] = situacionBolsaCoberturaMemorizada{
		situacion: situacion, hasta: s.instante().Add(s.validez()),
	}
	s.mu.Unlock()
	return situacion, true
}

// SituacionBolsaCobertura sirve la misma lectura memorizada a los avisos de
// la vía; sin enlace o si Bolsa no responde, no está disponible.
func (s *situacionBolsaCoberturaFijable) SituacionBolsaCobertura(
	ctx context.Context,
	categoriaRef string,
) (ports.SituacionBolsaCobertura, error) {
	situacion, ok := s.situacion(ctx, categoriaRef)
	if !ok {
		return ports.SituacionBolsaCobertura{}, ports.ErrSituacionBolsaCoberturaNoDisponible
	}
	return situacion, nil
}

var _ ports.ConsultaSituacionBolsaCobertura = (*situacionBolsaCoberturaFijable)(nil)

func (s *situacionBolsaCoberturaFijable) memorizada(categoriaRef string) (ports.SituacionBolsaCobertura, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	memorizada, ok := s.memoria[categoriaRef]
	if !ok || !s.instante().Before(memorizada.hasta) {
		return ports.SituacionBolsaCobertura{}, false
	}
	return memorizada.situacion, true
}

func (s *situacionBolsaCoberturaFijable) instante() time.Time {
	if s.ahora != nil {
		return s.ahora()
	}
	return time.Now()
}

func (s *situacionBolsaCoberturaFijable) validez() time.Duration {
	if s.validezDe > 0 {
		return s.validezDe
	}
	return validezSituacionBolsaCoberturaCT
}

// resultadoBolsaCobertura traduce la situación de Bolsa a las dos
// comprobaciones de la vía de bolsa. Cualquier otra comprobación no se
// responde aquí.
func resultadoBolsaCobertura(
	situacion ports.SituacionBolsaCobertura,
	comprobacion domain.ClaveCatalogo,
) (domain.ResultadoComprobacion, bool) {
	switch comprobacion {
	case comprobacionExisteBolsaVigenteCT:
		if situacion.Existe {
			return domain.ComprobacionAfirmativa, true
		}
		return domain.ComprobacionNegativa, true
	case comprobacionCandidaturasDisponibleCT:
		if situacion.Existe && situacion.Disponibles > 0 {
			return domain.ComprobacionAfirmativa, true
		}
		return domain.ComprobacionNegativa, true
	}
	return "", false
}
