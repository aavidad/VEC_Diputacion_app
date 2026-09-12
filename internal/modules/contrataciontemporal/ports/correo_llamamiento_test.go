package ports

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestSolicitudCorreoLlamamientoExigeSucesorYLigaduras(t *testing.T) {
	s := SolicitudDespacharCorreoLlamamiento{OrganizacionRef: "org-001", ExpedienteRef: "exp-001", LlamamientoRef: "llamamiento-001", ComunicacionRef: "comunicacion-ct54", IntencionEnvioRef: "intencion-ct54"}
	if s.Validar() != nil {
		t.Fatal("solicitud valida rechazada")
	}
	s.LlamamientoRef = ""
	if s.Validar() == nil {
		t.Fatal("falta llamamiento")
	}
}

func TestCorreoEfimeroSeRedactaEnFmtYJSON(t *testing.T) {
	m, err := NuevoMensajeCorreoLlamamiento("persona@ejemplo.invalid", "secreto", "secreto")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%v", m); got != "MensajeCorreoLlamamiento{redactado}" {
		t.Fatal(got)
	}
	b, e := json.Marshal(m)
	if e != nil || string(b) != "{\"redactado\":true}" {
		t.Fatalf("%s %v", b, e)
	}
}

func TestDestinoCorreoSeRedactaEnFmtYJSON(t *testing.T) {
	d, err := NuevoDestinoCorreoLlamamiento("persona@ejemplo.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%v %#v", d, d); got != "DestinoCorreoLlamamiento{redactado} DestinoCorreoLlamamiento{redactado}" {
		t.Fatal(got)
	}
	b, err := json.Marshal(&d)
	if err != nil || string(b) != "{\"redactado\":true}" {
		t.Fatalf("%s %v", b, err)
	}
}

func TestCapacidadFinalizacionLigaReservaYCopiaSecreto(t *testing.T) {
	r := ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<i@v.invalid>", FechaOrigen: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), SolicitudHuella: strings.Repeat("a", 64), Estado: CorreoLlamamientoIniciado}
	secret := make([]byte, 32)
	secret[0] = 1
	c, e := ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(r, secret)
	if e != nil || c.ValidarPara(r) != nil {
		t.Fatal(e)
	}
	secret[0] = 2
	if c.ExportarSecretoParaConsumidor()[0] != 1 {
		t.Fatal("sin copia")
	}
	exportado := c.ExportarSecretoParaConsumidor()
	exportado[0] = 3
	if c.ExportarSecretoParaConsumidor()[0] != 1 {
		t.Fatal("la exportación permite alterar el secreto original")
	}
	for _, formato := range []string{"%v", "%+v", "%#v"} {
		if got := fmt.Sprintf(formato, c); got != "CapacidadFinalizacionIntentoCorreoLlamamiento{redactada}" {
			t.Fatalf("la capacidad no se redacta con %s", formato)
		}
	}
	for _, valor := range []any{c, &c} {
		b, err := json.Marshal(valor)
		if err != nil || string(b) != `{"redactado":true}` {
			t.Fatal("la capacidad no se redacta en JSON")
		}
	}
	r.IntentoRef = "otro-001"
	if c.ValidarPara(r) == nil {
		t.Fatal("cruce")
	}
	if _, e = ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(ReservaIntentoCorreoLlamamiento{}, make([]byte, 32)); e == nil {
		t.Fatal("cero")
	}
}

func TestCapacidadFinalizacionRechazaReservaOCredencialInvalida(t *testing.T) {
	base := ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<i@v.invalid>", FechaOrigen: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), SolicitudHuella: strings.Repeat("a", 64), Estado: CorreoLlamamientoIniciado}
	secreto := []byte(strings.Repeat("s", 32))
	capacidad, err := ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(base, secreto)
	if err != nil || capacidad.ValidarPara(base) != nil {
		t.Fatal("el control válido no construye una capacidad")
	}
	for _, caso := range []struct {
		nombre  string
		cambiar func(*ReservaIntentoCorreoLlamamiento)
	}{
		{"replay", func(r *ReservaIntentoCorreoLlamamiento) { r.YaReservado = true }},
		{"terminal", func(r *ReservaIntentoCorreoLlamamiento) { r.Estado = CorreoLlamamientoAceptadoPorRelay }},
		{"huella_no_hex", func(r *ReservaIntentoCorreoLlamamiento) { r.SolicitudHuella = strings.Repeat("z", 64) }},
		{"huella_mayuscula", func(r *ReservaIntentoCorreoLlamamiento) { r.SolicitudHuella = strings.Repeat("A", 64) }},
		{"huella_cero", func(r *ReservaIntentoCorreoLlamamiento) { r.SolicitudHuella = strings.Repeat("0", 64) }},
		{"referencia_vacia", func(r *ReservaIntentoCorreoLlamamiento) { r.IntentoRef = "" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := base
			caso.cambiar(&r)
			c, err := ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(r, secreto)
			if err == nil || !c.EsCero() || capacidad.ValidarPara(r) == nil {
				t.Fatal("reserva inválida admitida en la frontera de finalización")
			}
		})
	}
	for _, caso := range []struct {
		nombre  string
		secreto []byte
	}{
		{"ausente", nil}, {"cero", make([]byte, 32)},
		{"corta", secreto[:31]}, {"larga", []byte(strings.Repeat("s", 33))},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c, err := ctdomain.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(base, caso.secreto)
			if err == nil || !c.EsCero() {
				t.Fatal("secreto inválido admitido con reserva sana")
			}
		})
	}
	if (CapacidadFinalizacionIntentoCorreoLlamamiento{}).ValidarPara(base) == nil {
		t.Fatal("capacidad cero admitida con reserva sana")
	}
	otra := base
	otra.SolicitudHuella = strings.Repeat("b", 64)
	if capacidad.ValidarPara(otra) == nil {
		t.Fatal("capacidad admitida con huella ajena válida")
	}
}
