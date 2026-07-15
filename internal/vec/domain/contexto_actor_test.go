package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNuevoContextoActorProducePrincipalCanonicoYReferenciasDefensivas(t *testing.T) {
	instante := instanteContextoActorPrueba()
	solicitud := solicitudContextoActorPrueba()
	instantanea := instantaneaContextoActorPrueba(instante)

	contexto, err := NuevoContextoActor(solicitud.Cuenta, instantanea, instante)
	if err != nil {
		t.Fatalf("crear contexto: %v", err)
	}
	if contexto.Principal.ID != instantanea.PersonaRef ||
		contexto.Principal.AuthMethod != solicitud.Cuenta.Metodo ||
		contexto.Principal.AuthAssurance != solicitud.Cuenta.Garantia ||
		contexto.PerfilActivoRef != solicitud.PerfilActivoRef {
		t.Fatalf("proyeccion distinta: %#v", contexto)
	}
	if len(contexto.Principal.Roles) != 0 || len(contexto.Principal.Permissions) != 0 ||
		len(contexto.Principal.Attributes) != 0 || contexto.Principal.DisplayName != "" ||
		contexto.Principal.Email != "" {
		t.Fatalf("el contexto fabrico autoridad o datos personales: %#v", contexto.Principal)
	}

	candidatos, err := contexto.Referencias(TipoReferenciaContextoActorCandidato)
	if err != nil || len(candidatos) != 1 || candidatos[0] != referenciaContextoActorPrueba("can_", "c") {
		t.Fatalf("referencias candidato: %#v, %v", candidatos, err)
	}
	empleados, err := contexto.Referencias(TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != referenciaContextoActorPrueba("emp_", "e") {
		t.Fatalf("referencias empleado: %#v, %v", empleados, err)
	}

	// Ni la entrada, ni la lista devuelta, ni una copia del contexto comparten la
	// memoria mutable de la instantanea canonica conservada por el resultado.
	instantanea.Vinculos[0].Referencia = referenciaContextoActorPrueba("emp_", "x")
	candidatos[0] = referenciaContextoActorPrueba("can_", "x")
	deNuevo, err := contexto.Referencias(TipoReferenciaContextoActorCandidato)
	if err != nil || deNuevo[0] != referenciaContextoActorPrueba("can_", "c") {
		t.Fatalf("una copia externa altero el contexto: %#v, %v", deNuevo, err)
	}
	copia, err := contexto.Clonar()
	if err != nil {
		t.Fatalf("clonar contexto: %v", err)
	}
	copia.Instantanea.Vinculos[0].Referencia = referenciaContextoActorPrueba("can_", "z")
	originales, _ := contexto.Referencias(TipoReferenciaContextoActorCandidato)
	if originales[0] != referenciaContextoActorPrueba("can_", "c") {
		t.Fatal("Clonar compartio los vinculos")
	}
}

func TestSolicitudContextoActorRechazaPerfilImplicitoDNIComodinesYNoCanonicos(t *testing.T) {
	base := solicitudContextoActorPrueba()
	casos := []struct {
		nombre string
		mutar  func(*SolicitudContextoActor)
	}{
		{"perfil ausente", func(s *SolicitudContextoActor) { s.PerfilActivoRef = "" }},
		{"perfil con comodin", func(s *SolicitudContextoActor) { s.PerfilActivoRef = "prf_" + strings.Repeat("p", 21) + "*" }},
		{"perfil con espacio", func(s *SolicitudContextoActor) { s.PerfilActivoRef = " " + s.PerfilActivoRef }},
		{"perfil con prefijo ajeno", func(s *SolicitudContextoActor) { s.PerfilActivoRef = referenciaContextoActorPrueba("per_", "p") }},
		{"cuenta como DNI", func(s *SolicitudContextoActor) { s.Cuenta.CuentaRef = "12345678Z" }},
		{"cuenta con control", func(s *SolicitudContextoActor) { s.Cuenta.CuentaRef += "\n" }},
		{"metodo desconocido", func(s *SolicitudContextoActor) { s.Cuenta.Metodo = "inventado" }},
		{"garantia desconocida", func(s *SolicitudContextoActor) { s.Cuenta.Garantia = "inventada" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := base
			caso.mutar(&solicitud)
			if err := solicitud.Validar(); !errors.Is(err, ErrSolicitudContextoActorInvalida) {
				t.Fatalf("se admitio solicitud no canonica: %v", err)
			}
		})
	}
}

func TestInstantaneaContextoActorRechazaIdentidadAmbiguaONoOpaca(t *testing.T) {
	instante := instanteContextoActorPrueba()
	base := instantaneaContextoActorPrueba(instante)
	casos := []struct {
		nombre string
		mutar  func(*InstantaneaContextoActor)
	}{
		{"version de vinculo cero", func(i *InstantaneaContextoActor) { i.VinculoVersion = 0 }},
		{"persona como DNI", func(i *InstantaneaContextoActor) { i.PersonaRef = "12345678Z" }},
		{"perfil con comodin", func(i *InstantaneaContextoActor) { i.PerfilActivoRef += "*" }},
		{"estado desconocido", func(i *InstantaneaContextoActor) { i.Estado = "desconocido" }},
		{"instante no UTC", func(i *InstantaneaContextoActor) { i.VigenteDesde = i.VigenteDesde.In(time.FixedZone("X", 3600)) }},
		{"precision submicrosegundo", func(i *InstantaneaContextoActor) { i.VigenteHasta = i.VigenteHasta.Add(time.Nanosecond) }},
		{"vinculo repetido", func(i *InstantaneaContextoActor) { i.Vinculos[1].VinculoRef = i.Vinculos[0].VinculoRef }},
		{"referencia repetida", func(i *InstantaneaContextoActor) { i.Vinculos[1] = i.Vinculos[0] }},
		{"dos candidatos canonicos", func(i *InstantaneaContextoActor) {
			i.Vinculos[1].Tipo = TipoReferenciaContextoActorCandidato
			i.Vinculos[1].Referencia = referenciaContextoActorPrueba("can_", "z")
		}},
		{"tipo y prefijo incompatibles", func(i *InstantaneaContextoActor) { i.Vinculos[0].Tipo = TipoReferenciaContextoActorEmpleado }},
		{"referencia de modulo como DNI", func(i *InstantaneaContextoActor) { i.Vinculos[0].Referencia = "12345678Z" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			instantanea, err := base.ClonarCanonica()
			if err != nil {
				t.Fatalf("fixture: %v", err)
			}
			caso.mutar(&instantanea)
			if err = instantanea.Validar(); !errors.Is(err, ErrInstantaneaContextoActorInvalida) {
				t.Fatalf("se admitio instantanea invalida: %v", err)
			}
		})
	}
}

func TestContextoActorDeniegaPerfilOVinculoNoVigente(t *testing.T) {
	instante := instanteContextoActorPrueba()
	solicitud := solicitudContextoActorPrueba()
	casos := []struct {
		nombre string
		mutar  func(*InstantaneaContextoActor)
	}{
		{"perfil revocado", func(i *InstantaneaContextoActor) { i.Estado = EstadoVinculoContextoActorRevocado }},
		{"perfil futuro", func(i *InstantaneaContextoActor) {
			i.VigenteDesde = instante.Add(time.Hour)
			i.VigenteHasta = instante.Add(2 * time.Hour)
		}},
		{"perfil caducado", func(i *InstantaneaContextoActor) {
			i.VigenteDesde = instante.Add(-2 * time.Hour)
			i.VigenteHasta = instante
		}},
		{"referencia revocada", func(i *InstantaneaContextoActor) { i.Vinculos[0].Estado = EstadoVinculoContextoActorRevocado }},
		{"referencia futura", func(i *InstantaneaContextoActor) {
			i.Vinculos[0].VigenteDesde = instante.Add(time.Hour)
			i.Vinculos[0].VigenteHasta = instante.Add(2 * time.Hour)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			instantanea := instantaneaContextoActorPrueba(instante)
			caso.mutar(&instantanea)
			if instantanea.Validar() != nil {
				t.Fatal("el estado historico debe poder validarse antes de comprobar vigencia")
			}
			if _, err := NuevoContextoActor(solicitud.Cuenta, instantanea, instante); !errors.Is(err, ErrContextoActorInvalido) {
				t.Fatalf("se resolvio vinculo no vigente: %v", err)
			}
		})
	}
}

func TestVigenciaContextoActorRechazaInstanteNoCanonico(t *testing.T) {
	instante := instanteContextoActorPrueba()
	instantanea := instantaneaContextoActorPrueba(instante)
	instanteLocal := instante.In(time.FixedZone("local", 60*60))
	instanteSubmicrosegundo := instante.Add(time.Nanosecond)
	instanteFueraDeIntervalo := time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC)

	if instantanea.VigenteEn(instanteLocal) || instantanea.VigenteEn(instanteSubmicrosegundo) ||
		instantanea.VigenteEn(instanteFueraDeIntervalo) {
		t.Fatal("una instantanea no debe aceptar instantes no canonicos")
	}
	if instantanea.Vinculos[0].VigenteEn(instanteLocal) ||
		instantanea.Vinculos[0].VigenteEn(instanteSubmicrosegundo) ||
		instantanea.Vinculos[0].VigenteEn(instanteFueraDeIntervalo) {
		t.Fatal("un vinculo no debe aceptar instantes no canonicos")
	}
}

func TestReferenciasContextoActorNoInterpretaTipoDesconocidoComoTodas(t *testing.T) {
	instante := instanteContextoActorPrueba()
	contexto, err := NuevoContextoActor(
		solicitudContextoActorPrueba().Cuenta,
		instantaneaContextoActorPrueba(instante),
		instante,
	)
	if err != nil {
		t.Fatalf("crear contexto: %v", err)
	}
	if referencias, err := contexto.Referencias(""); len(referencias) != 0 || !errors.Is(err, ErrContextoActorInvalido) {
		t.Fatalf("tipo vacio amplio la consulta: %#v, %v", referencias, err)
	}
}

func solicitudContextoActorPrueba() SolicitudContextoActor {
	return SolicitudContextoActor{
		Cuenta: CuentaAutenticadaContextoActor{
			CuentaRef: referenciaContextoActorPrueba("cta_", "a"),
			Metodo:    AuthMethodCertificate,
			Garantia:  AuthAssuranceHigh,
		},
		PerfilActivoRef: referenciaContextoActorPrueba("prf_", "p"),
	}
}

func instantaneaContextoActorPrueba(instante time.Time) InstantaneaContextoActor {
	solicitud := solicitudContextoActorPrueba()
	return InstantaneaContextoActor{
		VinculoRef:      referenciaContextoActorPrueba("vca_", "v"),
		VinculoVersion:  3,
		CuentaRef:       solicitud.Cuenta.CuentaRef,
		PersonaRef:      referenciaContextoActorPrueba("per_", "r"),
		PersonaVersion:  4,
		PerfilActivoRef: solicitud.PerfilActivoRef,
		PerfilVersion:   5,
		Estado:          EstadoVinculoContextoActorActivo,
		VigenteDesde:    instante.Add(-time.Hour),
		VigenteHasta:    instante.Add(time.Hour),
		Vinculos: []VinculoReferenciaContextoActor{
			{
				VinculoRef:   referenciaContextoActorPrueba("vin_", "c"),
				Version:      7,
				Tipo:         TipoReferenciaContextoActorCandidato,
				Referencia:   referenciaContextoActorPrueba("can_", "c"),
				Estado:       EstadoVinculoContextoActorActivo,
				VigenteDesde: instante.Add(-time.Hour),
				VigenteHasta: instante.Add(time.Hour),
			},
			{
				VinculoRef:   referenciaContextoActorPrueba("vin_", "e"),
				Version:      9,
				Tipo:         TipoReferenciaContextoActorEmpleado,
				Referencia:   referenciaContextoActorPrueba("emp_", "e"),
				Estado:       EstadoVinculoContextoActorActivo,
				VigenteDesde: instante.Add(-time.Hour),
				VigenteHasta: instante.Add(time.Hour),
			},
		},
	}
}

func instanteContextoActorPrueba() time.Time {
	return time.Date(2026, time.July, 15, 10, 30, 0, 123000000, time.UTC)
}

func referenciaContextoActorPrueba(prefijo, caracter string) string {
	return prefijo + strings.Repeat(caracter, longitudMinimaTokenContextoActor)
}
