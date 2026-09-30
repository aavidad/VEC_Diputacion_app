package canonico

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

func eventoExternoCanonPrueba() ports.EventoAvisoExterno {
	return ports.EventoAvisoExterno{EventoRef: "evento:1", ProductorRef: "productor:bolsa", TipoVersionado: ports.TipoAvisoLlamamientoExternoV1, OcurridoEn: "2026-09-30T12:13:14.123456Z", CorrelacionRef: "corr:1", DestinatarioExternoRef: "can_" + strings.Repeat("a", 24), ComunicacionRef: "llamamiento:" + strings.Repeat("b", 64), PlantillaRef: "plantilla:aviso", PlantillaVersion: "1"}
}
func TestAvisoExternoPreimagenCerrada(t *testing.T) {
	e := eventoExternoCanonPrueba()
	b, err := MaterialAvisoExterno(e)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"evento_ref":"evento:1","productor_ref":"productor:bolsa","tipo_versionado":"vec.bolsa.aviso-llamamiento.v1","ocurrido_en":"2026-09-30T12:13:14.123456Z","correlacion_ref":"corr:1","destinatario_externo_ref":"can_aaaaaaaaaaaaaaaaaaaaaaaa","comunicacion_ref":"llamamiento:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","plantilla_ref":"plantilla:aviso","plantilla_version":"1","recurso_publico_ref":""}`
	if string(b) != esperado {
		t.Fatalf("preimagen divergente: %s", b)
	}
	if got, err := LeerAvisoExterno(b); err != nil || got != e {
		t.Fatal(got, err)
	}
	for _, v := range []string{strings.Replace(esperado, `"evento_ref":"evento:1"`, `"evento_ref":"evento:1","evento_ref":"evento:1"`, 1), strings.Replace(esperado, `"recurso_publico_ref":""`, `"direccion":"a@example.test","recurso_publico_ref":""`, 1), strings.Replace(esperado, `"plantilla_version":"1"`, `"plantilla_version":1`, 1), esperado + "\n", strings.Replace(esperado, `,"recurso_publico_ref":""`, "", 1)} {
		if _, err := LeerAvisoExterno([]byte(v)); err == nil {
			t.Fatalf("aceptó preimagen abierta %s", v)
		}
	}
}
func TestAvisoExternoNoAdmiteIdentidadInternaNiTextoLibre(t *testing.T) {
	original := eventoExternoCanonPrueba()
	cambios := []func(*ports.EventoAvisoExterno){
		func(e *ports.EventoAvisoExterno) { e.DestinatarioExternoRef = "per_" + strings.Repeat("a", 24) },
		func(e *ports.EventoAvisoExterno) { e.DestinatarioExternoRef = "a@example.test" },
		func(e *ports.EventoAvisoExterno) { e.PlantillaRef = "Hola candidata" },
		func(e *ports.EventoAvisoExterno) { e.ProductorRef = "<script>" },
		func(e *ports.EventoAvisoExterno) { e.OcurridoEn = "2026-09-30T14:13:14.123456+02:00" },
		func(e *ports.EventoAvisoExterno) { e.OcurridoEn = "2026-09-30T12:13:14Z" },
		func(e *ports.EventoAvisoExterno) { e.OcurridoEn = "2026-09-31T12:13:14.123456Z" },
		func(e *ports.EventoAvisoExterno) { e.RecursoPublicoRef = "https://example.test/privado?token=1" },
	}
	for i, f := range cambios {
		e := original
		f(&e)
		if _, err := MaterialAvisoExterno(e); err == nil {
			t.Fatalf("caso %d admitido", i)
		}
	}
	h, _ := HuellaAvisoExterno(original)
	original.PlantillaVersion = "2"
	h2, _ := HuellaAvisoExterno(original)
	if h == h2 {
		t.Fatal("huella ignora versión")
	}
}
