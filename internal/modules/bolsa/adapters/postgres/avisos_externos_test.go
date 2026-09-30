package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func eventoAvisoPrueba() ports.EventoAvisoExterno {
	return ports.EventoAvisoExterno{
		EventoRef: "evento_aviso:" + strings.Repeat("a", 64), ProductorRef: "productor:bolsa:prueba", TipoVersionado: "vec.bolsa.aviso-llamamiento.v1", OcurridoEn: "2026-09-30T00:01:02.123456Z", CorrelacionRef: "corr_" + strings.Repeat("a", 32), DestinatarioExternoRef: "can_" + strings.Repeat("A", 22), ComunicacionRef: "llamamiento:" + strings.Repeat("b", 64), PlantillaRef: "bolsa-llamamiento-v1", PlantillaVersion: "bolsa-llamamiento-v1",
	}
}

func huellaAviso(e ports.EventoAvisoExterno) string {
	b, _ := json.Marshal(e)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestAvisoExternoCanonIndependienteOrdenJSON(t *testing.T) {
	e := eventoAvisoPrueba()
	// PostgreSQL jsonb devuelve otro orden y espacios. El hash corresponde al
	// DTO del protocolo, incluida la cadena vacía del recurso público.
	raw := `{"recurso_publico_ref":"", "plantilla_version":"bolsa-llamamiento-v1", "plantilla_ref":"bolsa-llamamiento-v1", "comunicacion_ref":"llamamiento:` + strings.Repeat("b", 64) + `", "destinatario_externo_ref":"can_` + strings.Repeat("A", 22) + `", "correlacion_ref":"corr_` + strings.Repeat("a", 32) + `", "ocurrido_en":"2026-09-30T00:01:02.123456Z", "tipo_versionado":"vec.bolsa.aviso-llamamiento.v1", "productor_ref":"productor:bolsa:prueba", "evento_ref":"evento_aviso:` + strings.Repeat("a", 64) + `"}`
	got, err := decodificarEventoAvisoExterno([]byte(raw), huellaAviso(e))
	if err != nil || got != e {
		t.Fatalf("canon divergente: %v", err)
	}
}

func TestAvisoExternoRechazaMaterialAjenoYCanonDivergente(t *testing.T) {
	original := eventoAvisoPrueba()
	cases := []struct {
		name    string
		alterar func(*ports.EventoAvisoExterno)
	}{
		{"correo", func(e *ports.EventoAvisoExterno) { e.DestinatarioExternoRef = "prueba@example.invalid" }},
		{"tipo", func(e *ports.EventoAvisoExterno) { e.TipoVersionado = "vec.otro.v1" }},
		{"escape", func(e *ports.EventoAvisoExterno) { e.PlantillaRef = "plantilla\ntexto" }},
		{"fecha zona", func(e *ports.EventoAvisoExterno) { e.OcurridoEn = "2026-09-30T00:01:02.123456+00:00" }},
		{"fecha nanosegundos", func(e *ports.EventoAvisoExterno) { e.OcurridoEn = "2026-09-30T00:01:02.123456789Z" }},
		{"fecha imposible", func(e *ports.EventoAvisoExterno) { e.OcurridoEn = "2026-99-30T00:01:02.123456Z" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := original
			tc.alterar(&e)
			raw, _ := json.Marshal(e)
			if _, err := decodificarEventoAvisoExterno(raw, huellaAviso(e)); err == nil {
				t.Fatal("contrato ajeno aceptado")
			}
		})
	}
	raw, _ := json.Marshal(original)
	if _, err := decodificarEventoAvisoExterno(raw, strings.Repeat("0", 64)); !errors.Is(err, ports.ErrEmisionLlamamientoConflicto) {
		t.Fatalf("hash divergente: %v", err)
	}
	for _, bad := range [][]byte{append(append([]byte{}, raw...), raw...), []byte(strings.Replace(string(raw), `"recurso_publico_ref":""`, `"recurso_publico_ref":null`, 1)), []byte(strings.Replace(string(raw), `"recurso_publico_ref":""`, `"recurso_publico_ref":"","direccion":"prueba@example.invalid"`, 1)), []byte(strings.Replace(string(raw), `,"recurso_publico_ref":""`, "", 1))} {
		if _, err := decodificarEventoAvisoExterno(bad, huellaAviso(original)); err == nil {
			t.Fatal("propiedad ajena, ausente o null aceptada")
		}
	}
}

func TestAvisoExternoSinPoolFallaCerrado(t *testing.T) {
	if _, err := NuevoRepositorioAvisosExternosPostgreSQL(nil); !errors.Is(err, ports.ErrEmisionLlamamientoNoDisponible) {
		t.Fatalf("pool nulo: %v", err)
	}
	var r *RepositorioAvisosExternosPostgreSQL
	if _, err := r.Extraer(context.Background(), 1); err == nil {
		t.Fatal("extraccion sin infraestructura")
	}
	if err := r.ConfirmarAceptacion(context.Background(), "", "", "", ""); err == nil {
		t.Fatal("ACK sin infraestructura")
	}
	if err := r.RegistrarResultadoDespacho(context.Background(), "", "", "", "", "aceptado"); err == nil {
		t.Fatal("resultado sin infraestructura")
	}
}

func TestProyeccionAvisosConTerminalParcialConservaPendienteYRecibosReales(t *testing.T) {
	e := ports.EmisionLlamamiento{LlamamientoRef: "llamamiento:" + strings.Repeat("b", 64), Participaciones: []string{"participacion:prueba:uno", "participacion:prueba:dos"}, Contactos: []ports.ResultadoContactoEmision{{ParticipacionRef: "participacion:prueba:uno", Resultado: "enviado"}}}
	pendientes := make([]ports.AvisoExternoPendiente, 0, 2)
	for _, participacion := range e.Participaciones {
		evento := eventoAvisoPrueba()
		hEvento := sha256.Sum256([]byte(e.LlamamientoRef + "\x1f" + participacion))
		evento.EventoRef = "evento_aviso:" + hex.EncodeToString(hEvento[:])
		huella := huellaAviso(evento)
		hRecibo := sha256.Sum256([]byte(evento.ProductorRef + "\x1f" + evento.EventoRef + "\x1f" + huella))
		pendientes = append(pendientes, ports.AvisoExternoPendiente{Evento: evento, Huella: huella, ReciboOutboxRef: "recibo_outbox:" + hex.EncodeToString(hRecibo[:]), EstadoDespacho: "reservado_incierto"})
	}
	pendientes[0].EstadoDespacho = "aceptado"
	got, err := proyectarContactosAvisosExternos(e, pendientes, "bolsa:prueba", "clave:prueba")
	if err != nil || len(got) != 2 || got[0].ParticipacionRef != e.Participaciones[0] || got[0].Resultado != "enviado" || got[0].ReciboRef != "recibo:contacto:c56f6f61b1bdfb8c554eb9d352adee3255db7a145873b1d5c5a980cef7b14bf7" || got[1].ParticipacionRef != e.Participaciones[1] || got[1].Resultado != "aviso_pendiente" || got[1].ReciboRef != pendientes[1].ReciboOutboxRef {
		t.Fatalf("lote parcial o recibos no conservados: %#v %v", got, err)
	}
	for _, terminal := range []string{"no_aceptado", "sin_destino"} {
		pendientes[1].EstadoDespacho = terminal
		got, err = proyectarContactosAvisosExternos(e, pendientes, "bolsa:prueba", "clave:prueba")
		if err != nil || got[1].Resultado != "no_enviado" || got[1].ReciboRef != "recibo:contacto:7703851ee37fb48ba1216e555f4daeccad69a9fc566a20af63d783d6e23f04d4" {
			t.Fatal("terminal sin recibo de contacto real", got, err)
		}
	}
	pendientes[1].EstadoDespacho = "resultado_ajeno"
	if _, err = proyectarContactosAvisosExternos(e, pendientes, "bolsa:prueba", "clave:prueba"); err == nil {
		t.Fatal("resultado ajeno aceptado")
	}
}
