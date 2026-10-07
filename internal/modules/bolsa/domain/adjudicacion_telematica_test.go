package domain

import "testing"

func TestConfirmacionAdjudicacionNoAdmiteModosSinImplementar(t *testing.T) {
	if !ConfirmacionAdjudicacionValida("") || !ConfirmacionAdjudicacionValida(ConfirmacionOfertaAceptacionPrevia) {
		t.Fatal("políticas históricas y aceptación previa deben admitirse")
	}
	for _, valor := range []string{"automatica", "aceptacion_previa ", "otra"} {
		if ConfirmacionAdjudicacionValida(valor) {
			t.Fatalf("modo no implementado admitido: %q", valor)
		}
	}
}

func TestPoliticaTelematicaVersionaLaAceptacionPrevia(t *testing.T) {
	p := PoliticaOfertas{
		Plazo:        PlazoPoliticaOfertas{Unidad: "dias_naturales", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087"},
		Adjudicacion: AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo", Confirmacion: ConfirmacionOfertaAceptacionPrevia},
		NoCubierta:   NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"},
		Plazas:       &PlazasPoliticaOfertas{Llamada: LlamadaPlazasSimultanea, RespuestaHoras: 24, TrasRenuncia: TrasRenunciaSiguienteEnOrden},
	}
	if err := p.Validar(); err != nil {
		t.Fatalf("política telemática rechazada: %v", err)
	}
	p.Adjudicacion.Confirmacion = ""
	if err := p.Validar(); err != nil {
		t.Fatalf("política histórica rechazada: %v", err)
	}
}
