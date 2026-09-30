package domain

import (
	"strings"
	"testing"
)

func TestAuditoriaFronteraUsuariosPreferenciasSoloRutaYActorVerificado(t *testing.T) {
	const ruta = "/api/vec/usuarios/mis-preferencias"
	const rutaExterior = "/api/vec/usuarios/area-personal/mis-preferencias"
	base := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Superficie:     SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias,
		Ruta:           ruta,
		Motivo:         MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("401 sin actor: %v", err)
	}
	conActor := base
	conActor.ActorRef = "per_" + strings.Repeat("a", 22)
	if conActor.Validar() == nil {
		t.Fatal("401 con actor aceptado")
	}
	conActor.Motivo = MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	if err := conActor.Validar(); err != nil {
		t.Fatalf("403 con persona opaca: %v", err)
	}
	for _, exacta := range []string{ruta, rutaExterior} {
		for _, orden := range []OrdenAuditoriaFronteraRutaExacta{base, conActor} {
			orden.Ruta = exacta
			if err := orden.Validar(); err != nil {
				t.Fatalf("ruta nominal %q rechazada: %v", exacta, err)
			}
		}
	}
	for _, actor := range []string{"", "cta_" + strings.Repeat("a", 22), "per_" + strings.Repeat("a", 21), "per_" + strings.Repeat("a", 22) + ":detalle"} {
		alterada := conActor
		alterada.ActorRef = actor
		if alterada.Validar() == nil {
			t.Fatalf("403 con actor no canónico aceptado: %q", actor)
		}
	}
	for _, variante := range []string{ruta + "/", ruta + "/otra", ruta + "?x=1", rutaExterior + "/", rutaExterior + "-extra", rutaExterior + "?x=1", "/api/vec/usuarios/otra", "/api/vec/usuarios/mis-preferencias%2Fotra"} {
		alterada := base
		alterada.Ruta = variante
		if alterada.Validar() == nil {
			t.Fatalf("ruta ampliada aceptada: %q", variante)
		}
	}
	for _, exacta := range []string{ruta, rutaExterior} {
		for _, superficie := range []string{SuperficieAuditoriaFronteraRutaExactaContratacionTemporal, SuperficieAuditoriaFronteraRutaExactaPersonal, SuperficieAuditoriaFronteraRutaExactaAuditoria} {
			alterada := base
			alterada.Ruta = exacta
			alterada.Superficie = superficie
			if alterada.Validar() == nil {
				t.Fatalf("ruta de Usuarios %q atribuida a %q", exacta, superficie)
			}
		}
	}
}

func TestAuditoriaFronteraAuditoriaAceptaSoloSuperficieYRutasNominales(t *testing.T) {
	base := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     SuperficieAuditoriaFronteraRutaExactaAuditoria,
	}
	for _, ruta := range []string{
		"/api/vec/auditoria/opciones",
		"/api/vec/auditoria/consultas",
	} {
		orden := base
		orden.Ruta = ruta
		if err := orden.Validar(); err != nil {
			t.Fatalf("ruta nominal %q rechazada: %v", ruta, err)
		}
		orden.Motivo = MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		if err := orden.Validar(); err != nil {
			t.Fatalf("denegacion 401 de %q rechazada: %v", ruta, err)
		}
		orden.Superficie = SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
		if orden.Validar() == nil {
			t.Fatalf("ruta %q aceptada sobre superficie CT", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/auditoria", "/api/vec/auditoria/",
		"/api/vec/auditoria/consultas/otra",
		"/api/vec/auditoria/consultas?dato=privado",
		"/api/vec/auditoria/opciones%2Fotra",
		"/api/vec/contratacion-temporal/solicitudes",
		"/api/vec/personal/empleados",
	} {
		orden := base
		orden.Ruta = ruta
		if orden.Validar() == nil {
			t.Fatalf("ruta ajena o ampliada %q aceptada", ruta)
		}
	}
}

func TestAuditoriaFronteraAspirantesSinPersona(t *testing.T) {
	base := OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_no_disponible", Motivo: MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie: SuperficieAuditoriaFronteraRutaExactaAspirantes, Ruta: "/api/vec/aspirantes/area-personal/mi-ficha"}
	if base.Validar() != nil {
		t.Fatal("denegación sin persona rechazada")
	}
	conPersona := base
	conPersona.ActorRef = "per_AAAAAAAAAAAAAAAAAAAAAA"
	otraRuta := base
	otraRuta.Ruta = "/api/vec/usuarios/area-personal/mis-preferencias"
	if conPersona.Validar() == nil || otraRuta.Validar() == nil {
		t.Fatal("Aspirantes nunca anota persona ni otra ruta")
	}
}
