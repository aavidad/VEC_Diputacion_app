package bootstrap

import (
	"context"
	"log/slog"
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
	// maximoCategoriasCoberturaMemorizadas acota la memoria: pasado el tope
	// se descartan las caducadas y, si sigue llena, no se memoriza.
	maximoCategoriasCoberturaMemorizadas = 256
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
	enCurso   map[string]chan struct{}
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
		s.enCurso = make(map[string]chan struct{})
	}
	turno := s.enCurso[categoriaRef]
	if turno == nil {
		turno = make(chan struct{}, 1)
		s.enCurso[categoriaRef] = turno
	}
	s.mu.Unlock()

	// Quien espera a otra lectura de la misma categoría deja de esperar si su
	// petición se cancela.
	select {
	case turno <- struct{}{}:
	case <-ctx.Done():
		return ports.SituacionBolsaCobertura{}, false
	}
	defer func() { <-turno }()
	if memorizada, ok := s.memorizada(categoriaRef); ok {
		return memorizada, true
	}
	situacion, err := consulta.SituacionBolsaCobertura(ctx, categoriaRef)
	if err != nil || situacion.Validar() != nil {
		// Texto fijo, sin causa ni categoría: la comprobación queda «no consta».
		slog.WarnContext(ctx, "contratacion_temporal.cobertura.situacion_bolsa_no_disponible")
		return ports.SituacionBolsaCobertura{}, false
	}
	s.mu.Lock()
	s.memorizar(categoriaRef, situacion)
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

// memorizar guarda la situación con el cerrojo tomado, sin pasar del tope.
func (s *situacionBolsaCoberturaFijable) memorizar(categoriaRef string, situacion ports.SituacionBolsaCobertura) {
	if s.memoria == nil {
		s.memoria = make(map[string]situacionBolsaCoberturaMemorizada)
	}
	ahora := s.instante()
	if _, existe := s.memoria[categoriaRef]; !existe && len(s.memoria) >= maximoCategoriasCoberturaMemorizadas {
		for categoria, memorizada := range s.memoria {
			if !ahora.Before(memorizada.hasta) {
				delete(s.memoria, categoria)
			}
		}
		if len(s.memoria) >= maximoCategoriasCoberturaMemorizadas {
			return
		}
	}
	s.memoria[categoriaRef] = situacionBolsaCoberturaMemorizada{situacion: situacion, hasta: ahora.Add(s.validez())}
}

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
