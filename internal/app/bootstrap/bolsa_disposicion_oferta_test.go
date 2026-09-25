package bootstrap

import (
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// La disposición actúa sobre la oferta; el resto de acciones propias, solo
// sobre 'mi-bolsa:<candidato>'. Ninguna admite el recurso de la otra.
func TestRecursoPortalPropioSeparaOfertaYMiBolsa(t *testing.T) {
	candidato := "can_candidato_sintetico_1234567890123456"
	oferta := dominiovec.RecursoAutorizable{Referencia: "oferta:" + strings.Repeat("a", 64), Tipo: puertosbolsa.TipoRecursoOfertaBolsa}
	propia := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato, Tipo: puertosbolsa.TipoRecursoMiBolsa}
	if !recursoPortalPropioDesarrollo(puertosbolsa.AccionManifestarDisposicionPropia, oferta, candidato) ||
		!recursoPortalPropioDesarrollo(puertosbolsa.AccionResponderLlamamientoPropio, propia, candidato) {
		t.Fatal("recursos legítimos rechazados")
	}
	for nombre, caso := range map[string]struct {
		accion  string
		recurso dominiovec.RecursoAutorizable
	}{
		"disposición sobre mi bolsa": {puertosbolsa.AccionManifestarDisposicionPropia, propia},
		"respuesta sobre la oferta":  {puertosbolsa.AccionResponderLlamamientoPropio, oferta},
		"oferta con tipo ajeno":      {puertosbolsa.AccionManifestarDisposicionPropia, dominiovec.RecursoAutorizable{Referencia: oferta.Referencia, Tipo: puertosbolsa.TipoRecursoMiBolsa}},
		"oferta mal formada":         {puertosbolsa.AccionManifestarDisposicionPropia, dominiovec.RecursoAutorizable{Referencia: "oferta:x", Tipo: puertosbolsa.TipoRecursoOfertaBolsa}},
		"mi bolsa de otro candidato": {puertosbolsa.AccionSolicitarPausaPropia, dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:can_otro", Tipo: puertosbolsa.TipoRecursoMiBolsa}},
	} {
		if recursoPortalPropioDesarrollo(caso.accion, caso.recurso, candidato) {
			t.Fatalf("%s admitido", nombre)
		}
	}
}
