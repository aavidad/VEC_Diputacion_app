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
		"plazo sin calendario":     func(v *PoliticaOfertas) { v.Plazo.MunicipioSede = "" },
		"plazo excesivo":           func(v *PoliticaOfertas) { v.Plazo.Cantidad = 31 },
		"criterio desconocido":     func(v *PoliticaOfertas) { v.Adjudicacion.Criterio = "aleatorio" },
		"confirmación desconocida": func(v *PoliticaOfertas) { v.Adjudicacion.Confirmacion = "respuesta_automatica" },
		"resultado inventado":      func(v *PoliticaOfertas) { v.NoCubierta.Accion = "renuncia_automatica" },
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

func TestPoliticaOfertasApartadoDePlazas(t *testing.T) {
	base := PoliticaOfertas{Plazo: PlazoPoliticaOfertas{Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc", MunicipioSede: "18087"},
		Adjudicacion: AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo"},
		NoCubierta:   NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"}}
	if base.Validar() != nil {
		t.Fatal("una política sin apartado de plazas sigue siendo válida")
	}
	validas := []PlazasPoliticaOfertas{
		{Llamada: LlamadaPlazasSimultanea, RespuestaHoras: 24, TrasRenuncia: TrasRenunciaSiguienteEnOrden},
		{Llamada: LlamadaPlazasSucesiva, RespuestaHoras: 1, TrasRenuncia: TrasRenunciaLlamamientoDirecto},
		{Llamada: LlamadaPlazasSimultanea, RespuestaHoras: MaximoHorasRespuestaPlazaOferta, TrasRenuncia: TrasRenunciaSiguienteEnOrden},
	}
	for _, plazas := range validas {
		p := base
		p.Plazas = &plazas
		if err := p.Validar(); err != nil {
			t.Fatalf("%+v rechazada: %v", plazas, err)
		}
	}
	invalidas := []PlazasPoliticaOfertas{
		{Llamada: "al_azar", RespuestaHoras: 24, TrasRenuncia: TrasRenunciaSiguienteEnOrden},
		{Llamada: LlamadaPlazasSimultanea, RespuestaHoras: 0, TrasRenuncia: TrasRenunciaSiguienteEnOrden},
		{Llamada: LlamadaPlazasSimultanea, RespuestaHoras: MaximoHorasRespuestaPlazaOferta + 1, TrasRenuncia: TrasRenunciaSiguienteEnOrden},
		{Llamada: LlamadaPlazasSimultanea, RespuestaHoras: 24, TrasRenuncia: "baja"},
		{},
	}
	for _, plazas := range invalidas {
		p := base
		p.Plazas = &plazas
		if p.Validar() == nil {
			t.Fatalf("%+v aceptada", plazas)
		}
	}
}

func TestPoliticaInicioExplicitoSoloObligatorioParaOfertasNuevas(t *testing.T) {
	p := PoliticaOfertas{Plazo: PlazoPoliticaOfertas{Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087"}, Adjudicacion: AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo"}, NoCubierta: NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"}}
	if p.Validar() != nil {
		t.Fatal("política histórica rechazada en lectura")
	}
	if p.ValidarParaOfertasNuevas() == nil {
		t.Fatal("se interpretó una política sin origen como notificación")
	}
	p.Plazo.Inicio = "notificacion"
	if p.ValidarParaOfertasNuevas() != nil {
		t.Fatal("política de notificación explícita rechazada")
	}
	p.Plazo.Inicio = "publicacion"
	if p.Validar() == nil || p.ValidarParaOfertasNuevas() == nil {
		t.Fatal("origen no ejecutable aceptado")
	}
}
