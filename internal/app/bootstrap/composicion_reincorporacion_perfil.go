package bootstrap

import (
	"context"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// El perfil y el contexto nominales se conservan: forman parte del material
// idempotente histórico. Solo la asignación pasa a consumo fijo por CAS.
func perfilFijoReincorporacionTitular(s *soporteAltaContratacionTemporalDesarrollo) (*perfilFijoCTDesarrollo, error) {
	if s == nil {
		return nil, errSeguimientoCeseDesarrolloNoDisponible
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.reincorporacionTitular
	if r == nil || r.contexto.Resultado.Validar() != nil || r.instantanea.Validar() != nil ||
		r.instantanea.AsignacionPerfil.PerfilActivoRef != r.contexto.Resultado.Contexto.PerfilActivoRef ||
		r.contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef ||
		r.contexto.Resultado.Contexto.Instantanea.CuentaRef != s.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		len(r.instantanea.AsignacionPerfil.Ambitos) != 1 || r.instantanea.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		!reflect.DeepEqual(r.instantanea.AsignacionPerfil.Ambitos[0].Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		return nil, errSeguimientoCeseDesarrolloNoDisponible
	}
	return &perfilFijoCTDesarrollo{clave: "reincorporacion_titular", contexto: r.contexto,
		plantilla:  clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(r.instantanea),
		rutas:      map[string]struct{}{httpinterno.RutaReincorporacionesTitular: {}, httpinterno.RutaCapacidadReincorporacionTitular: {}},
		actoSesion: actoSesionReincorporacionTitularDesarrollo, actoControlRol: actoControlRolReincorporacionTitularDesarrollo,
		contextoEsperadoRegistrado: r.contextoEsperadoRegistrado, sesionOperativa: r.sesionOperativa}, nil
}

func preimagenPropiaReincorporacionTitular(p *perfilFijoCTDesarrollo) func(instantaneaPublicadaDesarrollo, time.Time) bool {
	return func(publicada instantaneaPublicadaDesarrollo, instante time.Time) bool {
		if !preimagenPropiaPerfilFijoCTDesarrollo(p, actoAsignacionPerfilFijoCTDesarrollo, actoAsignacionReincorporacionTitularDesarrollo)(publicada, instante) ||
			publicada.actoControl != actoControlRolReincorporacionTitularDesarrollo ||
			publicada.instantanea.VersionRol.RolID != p.plantilla.VersionRol.RolID ||
			!reflect.DeepEqual(publicada.instantanea.VersionRol.Concesiones, p.plantilla.VersionRol.Concesiones) {
			return false
		}
		// La forma histórica solo contenía la organización, el expediente y,
		// en escritura, el par nombramiento/en_curso. No se sustituye una
		// restricción distinta aunque su acto tenga el mismo nombre.
		ambitos := publicada.instantanea.AsignacionPerfil.Ambitos
		if len(ambitos) != 1 && len(ambitos) != 2 && len(ambitos) != 4 {
			return false
		}
		claves := make(map[string]bool, len(ambitos))
		for _, a := range ambitos {
			claves[a.Clave] = true
			if len(a.Valores) != 1 {
				return false
			}
			switch a.Clave {
			case "organizacion_ref":
				if a.Valores[0] != organizacionAltaContratacionTemporalDesarrollo {
					return false
				}
			case "expediente_ref":
				if !domain.ReferenciaOpacaValida(a.Valores[0]) {
					return false
				}
			case "fase_previa":
				if len(ambitos) != 4 || a.Valores[0] != string(domain.FaseNombramiento) {
					return false
				}
			case "estado_previo":
				if len(ambitos) != 4 || a.Valores[0] != string(domain.EstadoEnCurso) {
					return false
				}
			default:
				return false
			}
		}
		return claves["organizacion_ref"] && (len(ambitos) == 1 || claves["expediente_ref"]) &&
			(len(ambitos) != 4 || (claves["fase_previa"] && claves["estado_previo"]))
	}
}

func componerPerfilFijoReincorporacionTitular(ctx context.Context, pool *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo,
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
) error {
	p, err := perfilFijoReincorporacionTitular(s)
	if err != nil {
		return err
	}
	if _, err := asegurarPerfilFijoCTDesarrollo(ctx, pool, s, p, aprobacion, preimagenPropiaReincorporacionTitular(p)); err != nil {
		return err
	}
	return s.registrarPerfilFijoCTDesarrollo(p)
}
