package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func materialNotificacionPrueba() MaterialRegistroNotificacion {
	return MaterialRegistroNotificacion{ActorRef: "per_AAAAAAAAAAAAAAAAAAAAAA", PerfilRef: "prf_PPPPPPPPPPPPPPPPPPPPPP", EmpleadoRef: "emp_AAAAAAAAAAAAAAAAAAAAAA",
		ClaveOperacion: "not-a-00001", TipoVersionRef: "notificacion:cronos:tipo:incidencia-marcaje:sintetico-1", FechaReferida: "2026-09-24",
		Texto: "No pude fichar la salida.\nLo comunico.", ZonaHoraria: ZonaSaldoPeninsula}
}

func TestMaterialNotificacionValidaYCanonicoConLasClavesDeSQL(t *testing.T) {
	m := materialNotificacionPrueba()
	b, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	var claves map[string]string
	if json.Unmarshal(b, &claves) != nil || len(claves) != 10 || claves["adjunto_ref"] != "" || claves["texto"] != m.Texto {
		t.Fatal(string(b))
	}
	m.AdjuntoRef, m.AdjuntoSHA256 = "registro:sintetico:0001", strings.Repeat("a", 64)
	m.Texto = strings.Repeat("ñ", MaximoTextoNotificacion)
	if m.Validar() != nil {
		t.Fatal("rechaza adjunto completo y 512 caracteres")
	}
	for nombre, alterar := range map[string]func(*MaterialRegistroNotificacion){
		"texto vacío":           func(m *MaterialRegistroNotificacion) { m.Texto = "" },
		"texto en blanco":       func(m *MaterialRegistroNotificacion) { m.Texto = " \n\t" },
		"texto largo":           func(m *MaterialRegistroNotificacion) { m.Texto = strings.Repeat("x", MaximoTextoNotificacion+1) },
		"control":               func(m *MaterialRegistroNotificacion) { m.Texto = "a\x07b" },
		"retorno de carro":      func(m *MaterialRegistroNotificacion) { m.Texto = "a\rb" },
		"UTF-8 inválido":        func(m *MaterialRegistroNotificacion) { m.Texto = string([]byte{0xff}) },
		"fecha imposible":       func(m *MaterialRegistroNotificacion) { m.FechaReferida = "2026-02-30" },
		"tipo sin versión":      func(m *MaterialRegistroNotificacion) { m.TipoVersionRef = "notificacion:cronos:tipo:incidencia" },
		"referencia sin huella": func(m *MaterialRegistroNotificacion) { m.AdjuntoRef = "registro:sintetico:0001" },
		"huella sin referencia": func(m *MaterialRegistroNotificacion) { m.AdjuntoSHA256 = strings.Repeat("a", 64) },
		"huella nula": func(m *MaterialRegistroNotificacion) {
			m.AdjuntoRef, m.AdjuntoSHA256 = "registro:sintetico:0001", strings.Repeat("0", 64)
		},
		"huella en mayúsculas": func(m *MaterialRegistroNotificacion) {
			m.AdjuntoRef, m.AdjuntoSHA256 = "registro:sintetico:0001", strings.Repeat("A", 64)
		},
		"referencia como URL": func(m *MaterialRegistroNotificacion) {
			m.AdjuntoRef, m.AdjuntoSHA256 = "https://custodia/doc", strings.Repeat("a", 64)
		},
		"clave corta":      func(m *MaterialRegistroNotificacion) { m.ClaveOperacion = "x" },
		"zona desconocida": func(m *MaterialRegistroNotificacion) { m.ZonaHoraria = "UTC" },
	} {
		c := materialNotificacionPrueba()
		alterar(&c)
		if c.Validar() == nil {
			t.Fatalf("acepta %s", nombre)
		}
		if _, err := c.Canonico(); err == nil {
			t.Fatalf("canoniza %s", nombre)
		}
	}
}

func TestMaterialesDeConsultaYAtencionDeNotificaciones(t *testing.T) {
	c := MaterialConsultaNotificaciones{ActorRef: "per_RRRRRRRRRRRRRRRRRRRRRR", PerfilRef: "prf_QQQQQQQQQQQQQQQQQQQQQQ", EmpleadoRef: "emp_RRRRRRRRRRRRRRRRRRRRRR", ZonaHoraria: ZonaSaldoPeninsula}
	if b, err := c.Canonico(); err != nil || strings.Count(string(b), `":"`) != 4 {
		t.Fatal(string(b), err)
	}
	a := MaterialAtencionNotificacion{ActorRef: c.ActorRef, PerfilRef: c.PerfilRef, EmpleadoRef: c.EmpleadoRef, ClaveOperacion: "ate-r-00001",
		NotificacionRef: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001", ZonaHoraria: ZonaSaldoPeninsula}
	if a.Validar() != nil {
		t.Fatal("rechaza atención válida")
	}
	for _, ref := range []string{"notificacion:cronos:x", "notificacion:cronos:atencion:0f0e0d0c-0b0a-4000-8000-000000000001", ""} {
		a.NotificacionRef = ref
		if a.Validar() == nil {
			t.Fatalf("acepta referencia %q", ref)
		}
	}
	if !AtencionRefValida("notificacion:cronos:atencion:0f0e0d0c-0b0a-4000-8000-000000000001") || AtencionRefValida("notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001") {
		t.Fatal("referencia de atención")
	}
	if !TipoNotificacionValido("notificacion:cronos:tipo:otra-comunicacion") || TipoNotificacionValido("notificacion:cronos:tipo:otra-comunicacion:v1") {
		t.Fatal("referencia de tipo")
	}
}
