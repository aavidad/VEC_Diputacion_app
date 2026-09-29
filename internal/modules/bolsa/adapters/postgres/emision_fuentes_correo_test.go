package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const correoRefFuentePrueba = "correo:0123456789abcdef0123456789abcdef"

func TestSalidaV2AplicaFuentesPorRecibo(t *testing.T) {
	salida := []byte(`{"llamamiento_ref":"llamamiento:1","recibo_ref":"recibo:llamamiento:1","bolsa_ref":"bolsa:1","estado":"emitido_pendiente_respuesta",
 "participaciones":["participacion:1","participacion:2","participacion:3"],"configuracion":{},"emitido_en":"2026-09-29T10:00:00Z",
 "contactos":[{"participacion_ref":"participacion:1","resultado":"enviado","recibo_ref":"recibo:contacto:1"},
              {"participacion_ref":"participacion:2","resultado":"enviado","recibo_ref":"recibo:contacto:2"},
              {"participacion_ref":"participacion:3","resultado":"no_enviado","recibo_ref":"recibo:contacto:3"}],
 "fuentes_correo":{"recibo:contacto:1":{"fuente":"mis_correos","motivo":"correo_activo","correo_ref":"` + correoRefFuentePrueba + `"},
                   "recibo:contacto:2":{"fuente":"alta_bolsa","motivo":"sin_correo_activo"},
                   "recibo:contacto:3":{"fuente":"mis_correos","motivo":"correo_activo"}}}`)
	var out emisionConFuentesSQL
	if err := json.Unmarshal(salida, &out); err != nil {
		t.Fatal(err)
	}
	e := aplicarFuentesCorreo(out.EmisionLlamamiento, out.FuentesCorreo)
	if f := e.Contactos[0].FuenteCorreo; f == nil || f.Fuente != ports.FuenteCorreoMisCorreos || f.CorreoRef != correoRefFuentePrueba {
		t.Fatalf("contacto 1: %+v", f)
	}
	if f := e.Contactos[1].FuenteCorreo; f == nil || f.Fuente != ports.FuenteCorreoAltaBolsa || f.Motivo != ports.MotivoFuenteSinCorreoActivo {
		t.Fatalf("contacto 2: %+v", f)
	}
	if e.Contactos[2].FuenteCorreo != nil {
		t.Fatal("una fuente que no cumple el contrato no se muestra")
	}
	if out.Contactos[0].FuenteCorreo != nil {
		t.Fatal("aplicar no debe modificar la emisión original")
	}
	b, _ := json.Marshal(e.Contactos[1])
	if string(b) != `{"participacion_ref":"participacion:2","resultado":"enviado","recibo_ref":"recibo:contacto:2","fuente_correo":{"fuente":"alta_bolsa","motivo":"sin_correo_activo"}}` {
		t.Fatalf("respuesta: %s", b)
	}
}

func TestContactoSinFuenteConservaLaFormaV1(t *testing.T) {
	b, _ := json.Marshal(ports.ResultadoContactoEmision{ParticipacionRef: "participacion:1", Resultado: "enviado", ReciboRef: "recibo:contacto:1"})
	if string(b) != `{"participacion_ref":"participacion:1","resultado":"enviado","recibo_ref":"recibo:contacto:1"}` {
		t.Fatalf("la v1 exige exactamente tres claves: %s", b)
	}
}

func TestRegistrarContactosConFuenteExigeB59Activo(t *testing.T) {
	r := &RepositorioEmisionLlamamientoPostgreSQL{}
	if _, err := r.CandidatoParticipacionAvisos(context.Background(), "bolsa:1", "llamamiento:1", "participacion:1"); err == nil {
		t.Fatal("sin B59 activo no se consultan candidatos")
	}
	if err := r.ActivarFuentesCorreo(context.Background()); err == nil {
		t.Fatal("sin pool no se activa B59")
	}
}
