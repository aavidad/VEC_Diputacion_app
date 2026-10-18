package application

import (
	"context"
	"slices"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ServicioGobiernoTransicionesSituacion compone la lectura versionada de B32
// con las restricciones que aplica el servicio operativo de situación.
type ServicioGobiernoTransicionesSituacion struct {
	repositorio     ports.RepositorioGobiernoTransicionesSituacion
	situacion       *ServicioSituacionParticipacion
	catalogoVersion int
	catalogoSHA256  string
}

func NuevoServicioGobiernoTransicionesSituacion(r ports.RepositorioGobiernoTransicionesSituacion,
	s *ServicioSituacionParticipacion, version int, sha string) (*ServicioGobiernoTransicionesSituacion, error) {
	if r == nil || s == nil || version < 1 || len(sha) != 64 {
		return nil, ports.ErrGobiernoTransicionesNoDisponible
	}
	return &ServicioGobiernoTransicionesSituacion{repositorio: r, situacion: s,
		catalogoVersion: version, catalogoSHA256: sha}, nil
}

func (s *ServicioGobiernoTransicionesSituacion) Consultar(ctx context.Context) (ports.GobiernoTransicionesSituacion, error) {
	if s == nil || ctx == nil {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	v, err := s.repositorio.Leer(ctx, s.catalogoVersion, s.catalogoSHA256)
	if err != nil {
		return v, err
	}
	v, err = s.completar(ctx, v)
	if err != nil {
		return v, err
	}
	// GET describe la política vigente, no acusa un POST de esta sesión.
	v.ReciboRef = ""
	v.Reutilizada = false
	return v, nil
}

func (s *ServicioGobiernoTransicionesSituacion) Publicar(ctx context.Context,
	c ports.ComandoGobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error) {
	if s == nil || ctx == nil || c.VersionEsperada < 1 || c.Motivo == "" || c.ClaveIdempotencia == "" {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesInvalido
	}
	politica, err := domain.PoliticaTransicionesDesdePares(c.Transiciones)
	if err != nil || !slices.Equal(politica.Pares(), c.Transiciones) {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesInvalido
	}
	c.CatalogoVersion = s.catalogoVersion
	c.CatalogoSHA256 = s.catalogoSHA256
	v, err := s.repositorio.Publicar(ctx, c)
	if err != nil {
		return v, err
	}
	return s.completar(ctx, v)
}

func (s *ServicioGobiernoTransicionesSituacion) completar(ctx context.Context,
	v ports.GobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error) {
	if v.Version < 1 || v.Limites.CatalogoVersion < 1 || len(v.Limites.CatalogoSHA256) != 64 {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	limites, err := domain.PoliticaTransicionesDesdePares(v.Limites.Transiciones)
	if err != nil || !slices.Equal(limites.Pares(), v.Limites.Transiciones) {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	guardadas, err := domain.PoliticaTransicionesDesdePares(v.TransicionesGuardadas)
	if err != nil || !slices.Equal(guardadas.Pares(), v.TransicionesGuardadas) {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	v.Esquema = ports.EsquemaGobiernoTransicionesSituacion
	v.TransicionesEfectivas = make([]string, 0, len(v.TransicionesGuardadas))
	if v.Reutilizada {
		// El recibo histórico se proyecta desde su política guardada. La
		// restricción operativa usa la misma fuente que el servicio de situación.
		if s.situacion.reglas != nil {
			for _, origen := range domain.SituacionesParticipacion() {
				destinos, configurada, err := s.situacion.reglas.DestinosSituacion(ctx, origen)
				if err != nil {
					return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
				}
				if configurada {
					guardadas = guardadas.Restringir(origen, destinos)
				}
			}
		}
		v.TransicionesEfectivas = guardadas.Pares()
	} else {
		efectivas, err := s.situacion.TransicionesAdmitidas(ctx)
		if err != nil {
			return ports.GobiernoTransicionesSituacion{}, err
		}
		for _, origen := range domain.SituacionesParticipacion() {
			for _, destino := range efectivas[origen] {
				v.TransicionesEfectivas = append(v.TransicionesEfectivas, origen+">"+destino)
			}
		}
	}
	v.TransicionesObligatorias = v.TransicionesObligatorias[:0]
	for _, par := range v.Limites.Transiciones {
		if strings.HasSuffix(par, ">"+domain.SituacionExcluido) {
			v.TransicionesObligatorias = append(v.TransicionesObligatorias, par)
		}
	}
	return v, nil
}
