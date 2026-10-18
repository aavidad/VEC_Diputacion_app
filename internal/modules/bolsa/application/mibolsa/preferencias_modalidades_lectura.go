package mibolsa

import (
	"context"
	"slices"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type LecturaPreferenciasModalidades struct {
	lector   bolsa.LecturaPreferenciasModalidades
	catalogo bolsa.CatalogoPreferenciasModalidades
	autoridadLecturaPreferencias
}

func NuevaLecturaPreferenciasModalidades(lector bolsa.LecturaPreferenciasModalidades, auditor bolsa.AuditoriaIntentosPreferenciasBolsas, fuente vecports.FuenteAutorizacion, catalogo bolsa.CatalogoPreferenciasModalidades, reloj vecports.Reloj) (*LecturaPreferenciasModalidades, error) {
	return nuevaLecturaModalidadesSuperficie(lector, auditor, fuente, catalogo, reloj, core.SuperficieAutenticacionExternaPersonalV1)
}

func NuevaLecturaPreferenciasModalidadesInterna(lector bolsa.LecturaPreferenciasModalidades, auditor bolsa.AuditoriaIntentosPreferenciasBolsas, fuente vecports.FuenteAutorizacion, catalogo bolsa.CatalogoPreferenciasModalidades, reloj vecports.Reloj) (*LecturaPreferenciasModalidades, error) {
	return nuevaLecturaModalidadesSuperficie(lector, auditor, fuente, catalogo, reloj, core.SuperficieAutenticacionInternaCorporativaV1)
}

func nuevaLecturaModalidadesSuperficie(lector bolsa.LecturaPreferenciasModalidades, auditor bolsa.AuditoriaIntentosPreferenciasBolsas, fuente vecports.FuenteAutorizacion, catalogo bolsa.CatalogoPreferenciasModalidades, reloj vecports.Reloj, superficie core.SuperficieAutenticacionActorV1) (*LecturaPreferenciasModalidades, error) {
	if nula(lector) || nula(auditor) || nula(fuente) || nula(catalogo) || nula(reloj) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &LecturaPreferenciasModalidades{lector: lector, catalogo: catalogo,
		autoridadLecturaPreferencias: autoridadLecturaPreferencias{auditor, fuente, reloj, superficie}}, nil
}

func (s *LecturaPreferenciasModalidades) ConsultarPreferenciasModalidades(ctx context.Context, orden Orden) (bolsa.EstadoPreferenciasModalidades, error) {
	vacio := bolsa.EstadoPreferenciasModalidades{}
	if s == nil || nula(s.lector) || nula(s.catalogo) {
		return vacio, bolsa.ErrPreferenciasNoDisponibles
	}
	captura, err := s.prepararLectura(ctx, orden, bolsa.CamposLecturaPreferenciasModalidades())
	if err != nil {
		return vacio, err
	}
	disponibles, err := s.catalogo.ConsultarModalidadesDisponibles(ctx)
	if err != nil || len(disponibles) == 0 {
		return vacio, s.registrarErrorLectura(ctx, orden, captura, err)
	}
	estado, err := s.lector.LeerPreferenciasModalidades(ctx, captura)
	if err != nil {
		return vacio, s.registrarErrorLectura(ctx, orden, captura, err)
	}
	// La proyección SQL audita el campo; su contenido procede de la lectura
	// vigente del catálogo común, no de una lista suministrada por la persona.
	estado.ModalidadesDisponibles = slices.Clone(disponibles)
	return estado, nil
}
