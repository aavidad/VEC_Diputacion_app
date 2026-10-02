package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNotificacionOfertaExigeHechoYaRealizadoYReferenciaSinDatos(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	base := NotificacionOferta{NotificadaEn: ahora.Add(-time.Hour), ReferenciaCorreo: "correo:extracto:oferta-1", HuellaCorreoSHA256: strings.Repeat("a", 64), Fuente: "correo_externo_declarado_rrhh"}
	if err := base.ValidarPara(ahora); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(*NotificacionOferta){
		"sin instante":          func(n *NotificacionOferta) { n.NotificadaEn = time.Time{} },
		"futura":                func(n *NotificacionOferta) { n.NotificadaEn = ahora.Add(time.Microsecond) },
		"precision excesiva":    func(n *NotificacionOferta) { n.NotificadaEn = ahora.Add(-time.Nanosecond) },
		"correo sin referencia": func(n *NotificacionOferta) { n.ReferenciaCorreo = "" },
		"direccion personal":    func(n *NotificacionOferta) { n.ReferenciaCorreo = "antonio@example.invalid" },
		"dni en referencia":     func(n *NotificacionOferta) { n.ReferenciaCorreo = "correo:dni-12345678Z" },
		"nie en referencia":     func(n *NotificacionOferta) { n.ReferenciaCorreo = "correo:X1234567L" },
		"etiqueta identidad":    func(n *NotificacionOferta) { n.ReferenciaCorreo = "correo:pasaporte:extracto-1" },
		"sin huella":            func(n *NotificacionOferta) { n.HuellaCorreoSHA256 = "" },
		"entrega no acreditada": func(n *NotificacionOferta) { n.Fuente = "entregado_smtp" },
	} {
		t.Run(nombre, func(t *testing.T) {
			n := base
			cambiar(&n)
			if n.ValidarPara(ahora) == nil {
				t.Fatal("notificación inválida aceptada")
			}
		})
	}
}
