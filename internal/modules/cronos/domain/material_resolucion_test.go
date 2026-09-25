package domain

import (
	"strings"
	"testing"
)

func materialResolucionPrueba() MaterialResolucionPermiso {
	return MaterialResolucionPermiso{ActorRef: "per_JJJJJJJJJJJJJJJJJJJJJJ", PerfilRef: "prf_QQQQQQQQQQQQQQQQQQQQQQ", EmpleadoRef: "emp_JJJJJJJJJJJJJJJJJJJJJJ",
		ClaveOperacion: "res-va-00001", SolicitudRef: "permiso:cronos:solicitud:perm-va-00001", Paso: PasoResponsable,
		Decision: DecisionAprobar, VersionEsperada: 1, ZonaHoraria: ZonaSaldoPeninsula}
}

func TestMaterialResolucionValidaYCanonico(t *testing.T) {
	m := materialResolucionPrueba()
	b, err := m.Canonico()
	if err != nil || !strings.Contains(string(b), `"version_esperada":"1"`) || !strings.Contains(string(b), `"motivo":""`) {
		t.Fatal(string(b), err)
	}
	for nombre, alterar := range map[string]func(*MaterialResolucionPermiso){
		"paso desconocido":          func(m *MaterialResolucionPermiso) { m.Paso = "jefatura" },
		"decisión desconocida":      func(m *MaterialResolucionPermiso) { m.Decision = "conceder" },
		"denegar sin motivo":        func(m *MaterialResolucionPermiso) { m.Decision = DecisionDenegar },
		"motivo largo":              func(m *MaterialResolucionPermiso) { m.Motivo = strings.Repeat("x", MaximoMotivoResolucion+1) },
		"motivo con salto":          func(m *MaterialResolucionPermiso) { m.Motivo = "a\nb" },
		"motivo con espacios":       func(m *MaterialResolucionPermiso) { m.Motivo = " a" },
		"versión cero":              func(m *MaterialResolucionPermiso) { m.VersionEsperada = 0 },
		"versión de cuatro cifras":  func(m *MaterialResolucionPermiso) { m.VersionEsperada = 1000 },
		"solicitud ajena al patrón": func(m *MaterialResolucionPermiso) { m.SolicitudRef = "permiso:cronos:solicitud:x" },
		"clave corta":               func(m *MaterialResolucionPermiso) { m.ClaveOperacion = "x" },
		"zona desconocida":          func(m *MaterialResolucionPermiso) { m.ZonaHoraria = "UTC" },
	} {
		c := materialResolucionPrueba()
		alterar(&c)
		if c.Validar() == nil {
			t.Fatalf("acepta %s", nombre)
		}
	}
	m.Decision, m.Motivo = DecisionDenegar, strings.Repeat("ñ", MaximoMotivoResolucion)
	if m.Validar() != nil {
		t.Fatal("rechaza un motivo de 500 caracteres multibyte")
	}
}

func TestEstadoTrasResolucion(t *testing.T) {
	for _, c := range []struct {
		paso     PasoPermiso
		decision DecisionPermiso
		estado   EstadoSolicitudPermiso
	}{
		{PasoResponsable, DecisionAprobar, EstadoPermisoPendienteAdministracion},
		{PasoResponsable, DecisionDenegar, EstadoPermisoDenegado},
		{PasoAdministracion, DecisionAprobar, EstadoPermisoConcedido},
		{PasoAdministracion, DecisionDenegar, EstadoPermisoDenegado},
	} {
		if e, ok := EstadoTrasResolucion(c.paso, c.decision); !ok || e != c.estado {
			t.Fatalf("%s/%s: %s", c.paso, c.decision, e)
		}
	}
	if _, ok := EstadoTrasResolucion("otro", DecisionAprobar); ok {
		t.Fatal("paso desconocido con estado")
	}
}

func TestMaterialesAvisosYBandeja(t *testing.T) {
	b := MaterialBandejaPermisos{ActorRef: "per_JJJJJJJJJJJJJJJJJJJJJJ", PerfilRef: "prf_QQQQQQQQQQQQQQQQQQQQQQ", EmpleadoRef: "emp_JJJJJJJJJJJJJJJJJJJJJJ", Paso: PasoAdministracion, ZonaHoraria: ZonaSaldoPeninsula}
	if _, err := b.Canonico(); err != nil {
		t.Fatal(err)
	}
	b.Paso = ""
	if b.Validar() == nil {
		t.Fatal("bandeja sin paso")
	}
	a := MaterialArchivoAvisoPropio{ActorRef: "per_AAAAAAAAAAAAAAAAAAAAAA", PerfilRef: "prf_PPPPPPPPPPPPPPPPPPPPPP", EmpleadoRef: "emp_AAAAAAAAAAAAAAAAAAAAAA",
		ClaveOperacion: "arch-00000001", AvisoRef: "aviso:cronos:00000000-0000-4000-8000-000000000001", ZonaHoraria: ZonaSaldoPeninsula}
	if _, err := a.Canonico(); err != nil {
		t.Fatal(err)
	}
	a.AvisoRef = "aviso:cronos:x"
	if a.Validar() == nil {
		t.Fatal("aviso mal formado")
	}
}
