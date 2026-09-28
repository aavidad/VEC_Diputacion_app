package domain

import "testing"

func TestPoliticaOfertasSoloAdmiteReglaEjecutable(t *testing.T) {
	p := PoliticaOfertas{
		Plazo:        PlazoPoliticaOfertas{Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087"},
		Adjudicacion: AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo"},
		NoCubierta:   NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"},
	}
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	for nombre, cambio := range map[string]func(*PoliticaOfertas){
		"plazo sin calendario": func(v *PoliticaOfertas) { v.Plazo.MunicipioSede = "" },
		"plazo excesivo":       func(v *PoliticaOfertas) { v.Plazo.Cantidad = 31 },
		"criterio desconocido": func(v *PoliticaOfertas) { v.Adjudicacion.Criterio = "aleatorio" },
		"resultado inventado":  func(v *PoliticaOfertas) { v.NoCubierta.Accion = "renuncia_automatica" },
	} {
		t.Run(nombre, func(t *testing.T) {
			q := p
			cambio(&q)
			if q.Validar() == nil {
				t.Fatal("política no ejecutable aceptada")
			}
		})
	}
}

func TestPoliticaOfertasHorasNaturalesConfigurables(t *testing.T) {
	p := PoliticaOfertas{
		Plazo:        PlazoPoliticaOfertas{Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc", MunicipioSede: "18087"},
		Adjudicacion: AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo"},
		NoCubierta:   NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"},
	}
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	p.Plazo.Cantidad = 720
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	p.Plazo.Cantidad = 721
	if p.Validar() == nil {
		t.Fatal("duración excesiva aceptada")
	}
	p.Plazo.Cantidad, p.Plazo.Computo = 48, "administrativo"
	if p.Validar() == nil {
		t.Fatal("horas con calendario administrativo aceptadas")
	}
}
