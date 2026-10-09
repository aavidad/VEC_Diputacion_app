package application

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

// maximoVersionesMemorizadas acota la memoria de una consulta: un cuadro usa
// uno o dos años y dos lecturas por año. Pasado el tope se lee sin memorizar.
const maximoVersionesMemorizadas = 32

// ParaConsulta devuelve un servicio para una sola consulta (por ejemplo, los
// plazos de todas las filas de un cuadro). Fija «lo conocido ahora» al crearse
// y lee cada año y ámbito una sola vez: los cálculos son los mismos que con el
// servicio original en ese instante. No se guarda entre consultas; solo
// memoriza lecturas correctas y cada lectura usa el contexto de quien la pide.
func (s *Servicio) ParaConsulta() ports.ConsultaCalendarios {
	if s == nil || nulo(s.repositorio) || nulo(s.reloj) {
		return s
	}
	return &Servicio{
		repositorio: &repositorioMemorizado{base: s.repositorio, memoria: map[string][]domain.VersionConDias{}},
		reloj:       relojConsultaFijo(s.reloj.Ahora()),
	}
}

type relojConsultaFijo time.Time

func (r relojConsultaFijo) Ahora() time.Time { return time.Time(r) }

type repositorioMemorizado struct {
	base    ports.RepositorioCalendarios
	mu      sync.Mutex
	memoria map[string][]domain.VersionConDias
}

func (r *repositorioMemorizado) VersionesVigentes(ctx context.Context, c ports.ConsultaVersiones) ([]domain.VersionConDias, error) {
	clave := claveVersiones(c)
	r.mu.Lock()
	guardadas, existe := r.memoria[clave]
	r.mu.Unlock()
	if existe {
		return copiarVersiones(guardadas), nil
	}
	leidas, err := r.base.VersionesVigentes(ctx, c)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	if len(r.memoria) < maximoVersionesMemorizadas {
		r.memoria[clave] = copiarVersiones(leidas)
	}
	r.mu.Unlock()
	return leidas, nil
}

func (r *repositorioMemorizado) CentrosConCalendario(ctx context.Context, anio int, conocidoEn time.Time) ([]domain.VersionCalendario, error) {
	return r.base.CentrosConCalendario(ctx, anio, conocidoEn)
}

func claveVersiones(c ports.ConsultaVersiones) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(c.Anio))
	b.WriteByte('#')
	b.WriteString(c.ConocidoEn.UTC().Format(time.RFC3339Nano))
	for _, a := range c.Ambitos {
		b.WriteByte('#')
		b.WriteString(a.Clave())
	}
	return b.String()
}

func copiarVersiones(origen []domain.VersionConDias) []domain.VersionConDias {
	if origen == nil {
		return nil
	}
	copia := make([]domain.VersionConDias, len(origen))
	for i, v := range origen {
		copia[i] = v
		copia[i].Dias = append([]domain.DiaSenalado(nil), v.Dias...)
	}
	return copia
}
