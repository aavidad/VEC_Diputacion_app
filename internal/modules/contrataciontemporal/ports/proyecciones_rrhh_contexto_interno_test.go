package ports

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestContextoConsultaRRHHValidaEvidenciaNuevaDeSesionYPerfil(t *testing.T) {
	t.Parallel()
	ahora := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	base := ContextoConsultaRRHH{
		autenticacionRef:          "aut_0123456789abcdefghijkl",
		autenticacionHuella:       strings.Repeat("1", 64),
		sesionRef:                 "ses_0123456789abcdefghijkl",
		controlSesionRef:          "cse_0123456789abcdefghijkl",
		controlSesionRevision:     2,
		controlSesionHuellaSHA256: strings.Repeat("2", 64),
		actorRef:                  "per_0123456789abcdefghijkl",
		perfilRef:                 "prf_0123456789abcdefghijkl",
		perfilVersion:             5,
		registroContextoRef:       "rca_0123456789abcdefghijkl",
		contextoActorHuella:       strings.Repeat("3", 64),
		organizacionRef:           "organizacion:diputacion-granada",
		resueltoEn:                ahora.Add(-time.Minute),
		validoHasta:               ahora.Add(time.Minute),
	}
	if err := base.validarEn(ahora); err != nil {
		t.Fatalf("contexto base inválido: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*ContextoConsultaRRHH)
	}{
		{"control sin referencia", func(c *ContextoConsultaRRHH) {
			c.controlSesionRef = ""
		}},
		{"control sin revisión", func(c *ContextoConsultaRRHH) {
			c.controlSesionRevision = 0
		}},
		{"control con huella nula", func(c *ContextoConsultaRRHH) {
			c.controlSesionHuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"control con huella no canónica", func(c *ContextoConsultaRRHH) {
			c.controlSesionHuellaSHA256 = strings.Repeat("A", 64)
		}},
		{"perfil sin versión", func(c *ContextoConsultaRRHH) {
			c.perfilVersion = 0
		}},
		{"perfil con versión fuera del rango interoperable", func(c *ContextoConsultaRRHH) {
			c.perfilVersion = versionMaximaJSONSegura + 1
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			mutada := base
			caso.mutar(&mutada)
			if err := mutada.validarEn(ahora); !errors.Is(
				err, ErrContextoConsultaRRHHInvalido,
			) {
				t.Fatalf("mutación aceptada: %v", err)
			}
		})
	}
}
