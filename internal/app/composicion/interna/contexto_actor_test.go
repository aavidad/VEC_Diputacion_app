package interna

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
)

type selectorPerfilActivoEspia struct{ llamadas int }

func (s *selectorPerfilActivoEspia) SeleccionarPerfilActivo(context.Context, vec.CuentaAutenticadaContextoActor, httpseguridad.ContextoAuditoriaAutenticada) (string, error) {
	s.llamadas++
	return "prf_no_debe_usarse", nil
}

func TestAmbitoRRHHRegistradoRechazaCruceActorPerfilYContexto(t *testing.T) {
	instante := time.Date(2026, 9, 23, 18, 0, 0, 0, time.UTC)
	resultado := vec.ResultadoContextoActorRegistradoV2{}
	resultado.Contexto.Principal.ID = "per_123456789012345678901234"
	resultado.Contexto.PerfilActivoRef = "prf_123456789012345678901234"
	resultado.Contexto.Instantanea.PerfilVersion = 3
	resultado.RegistroContextoRef = "rct_123456789012345678901234"
	resultado.HuellaSHA256 = strings.Repeat("a", 64)
	ambito := ambitoConsultaRRHHRegistrado{
		RegistroRef: "amb_123456789012345678901234", ActorRef: resultado.Contexto.Principal.ID,
		PerfilRef: resultado.Contexto.PerfilActivoRef, PerfilVersion: 3,
		ContextoRef: resultado.RegistroContextoRef, ContextoHuella: resultado.HuellaSHA256,
		OrganizacionRef: "org_123456789012345678901234", Clase: ports.AmbitoOrganizacionRRHH,
		AmbitoRef:    "org_123456789012345678901234",
		VigenteDesde: instante.Add(-time.Minute), VigenteHasta: instante.Add(time.Minute),
	}
	if !ambito.coincideCon(resultado, instante) {
		t.Fatal("el ámbito registrado exacto se rechazó")
	}
	for nombre, alterar := range map[string]func(*ambitoConsultaRRHHRegistrado){
		"otro actor":    func(a *ambitoConsultaRRHHRegistrado) { a.ActorRef = "per_otra_persona_1234567890123" },
		"otro perfil":   func(a *ambitoConsultaRRHHRegistrado) { a.PerfilRef = "prf_otro_perfil_1234567890123" },
		"otra versión":  func(a *ambitoConsultaRRHHRegistrado) { a.PerfilVersion++ },
		"otro contexto": func(a *ambitoConsultaRRHHRegistrado) { a.ContextoRef = "rct_otro_contexto_123456789012" },
		"otra huella":   func(a *ambitoConsultaRRHHRegistrado) { a.ContextoHuella = strings.Repeat("b", 64) },
		"caducado":      func(a *ambitoConsultaRRHHRegistrado) { a.VigenteHasta = instante },
	} {
		t.Run(nombre, func(t *testing.T) {
			cruzado := ambito
			alterar(&cruzado)
			if cruzado.coincideCon(resultado, instante) {
				t.Fatal("ámbito ajeno o caducado aceptado")
			}
		})
	}
}

func TestContextoActorLecturaNoConsultaPerfilSinCapsulaAutenticada(t *testing.T) {
	selector := &selectorPerfilActivoEspia{}
	a := contextoActorLecturaCT{selector: selector}
	resultado, err := a.resolver(context.Background())
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
		resultado.Resultado.Validar() == nil || selector.llamadas != 0 {
		t.Fatalf("sin identidad = (%v, %v), selector llamado %d veces", resultado, err, selector.llamadas)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = a.resolver(ctx)
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || selector.llamadas != 0 {
		t.Fatalf("cancelada = %v, selector llamado %d veces", err, selector.llamadas)
	}
}
