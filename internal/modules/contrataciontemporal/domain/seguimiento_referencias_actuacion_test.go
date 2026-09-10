package domain

import (
	"bytes"
	"strings"
	"testing"
)

func TestSeguimientoReferenciasActuacionOriginales(t *testing.T) {
	def := definicionSeguimientoValida(t, false)
	alta := altaReferenciasSeguimientoPrueba(t, def)
	alta.OrganizacionRef = organizacionSeguimientoExistente
	alta.ExpedienteRef = expedienteSeguimientoExistente
	s, err := NuevoSeguimiento(def, alta)
	if err != nil {
		t.Fatal(err)
	}
	d := seguimientoIncorporado(t, def).Actuaciones()[0].datos()
	d.ActorRef = "per_956e44e7abde00cefad2999ecff7e0fe"
	d.UnidadRef = "unidad:desarrollo:rrhh"
	d.Documentos[0].Referencia = "documento-resolucion:c927ff73aa152b782b3131e35699f5a5d73e4ee7081dfedac7431b411a1e2e14"
	aplicado, err := s.Aplicar(def, 0, d)
	if err != nil {
		t.Fatal(err)
	}
	material := comprobarRestauracionReferenciasSeguimiento(t, def, aplicado)
	for _, ref := range []string{d.ActorRef, d.UnidadRef, d.Documentos[0].Referencia} {
		if !bytes.Contains(material, []byte(ref)) {
			t.Fatal("se sustituyó una referencia original")
		}
	}
	act := aplicado.Actuaciones()[0]
	if act.ActorRef != d.ActorRef || act.UnidadRef != d.UnidadRef || act.Documentos[0] != d.Documentos[0] || aplicado.Version() != 1 {
		t.Fatal("actuación no conserva las coordenadas originales")
	}
}

func TestSeguimientoReferenciasActuacionRechazaCruces(t *testing.T) {
	def := definicionSeguimientoValida(t, false)
	base := seguimientoIncorporado(t, def).Actuaciones()[0].datos()
	for _, caso := range []struct {
		nombre    string
		asignar   func(*DatosTransicionSeguimiento, string)
		invalidos []string
	}{
		{"unidad", func(d *DatosTransicionSeguimiento, v string) { d.UnidadRef = v }, []string{"unidad:", "Unidad:rrhh", "unidad:con espacio", "unidad:rrhh\n", "unidad:" + strings.Repeat("a", 154), "per_956e44e7abde00cefad2999ecff7e0fe", "ref:" + strings.Repeat("0", 64)}},
		{"actor", func(d *DatosTransicionSeguimiento, v string) { d.ActorRef = v }, []string{"per_", "per_" + strings.Repeat("0", 32), "per_" + strings.Repeat("a", 31), "per_" + strings.Repeat("A", 32), "per_" + strings.Repeat("g", 32), "unidad:rrhh", "actor:rrhh", "12345678Z"}},
		{"documento", func(d *DatosTransicionSeguimiento, v string) { d.Documentos[0].Referencia = v }, []string{"documento-resolucion:", "documento-resolucion:" + strings.Repeat("0", 64), "documento-resolucion:" + strings.Repeat("A", 64), "documento-resolucion:" + strings.Repeat("a", 63), "documento:" + strings.Repeat("a", 64), "unidad:rrhh"}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			for _, v := range caso.invalidos {
				d := base.clonar()
				caso.asignar(&d, v)
				if _, err := normalizarDatosTransicionSeguimiento(d); err == nil {
					t.Fatal("se admitió una referencia ajena")
				}
			}
		})
	}
}
