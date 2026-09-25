package domain

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func politicaIntentosPrueba(t *testing.T, controlSeparacion, controlFranja string) PoliticaIntentosTelefonicos {
	t.Helper()
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	return PoliticaIntentosTelefonicos{
		IntentosPorProceso: 2, Procesos: 2, SeparacionMinima: 2 * time.Hour, ControlSeparacion: controlSeparacion,
		ResultadosSinContacto: []string{ResultadoContactoNoContesta, ResultadoContactoBuzon, ResultadoContactoNumeroErroneo},
		Franja:                FranjaLlamadas{DesdeMinuto: 9 * 60, HastaMinuto: 14 * 60, Zona: zona, SoloDiasHabiles: true, Control: controlFranja},
	}
}

// 28/09/2026 es lunes; las 08:00 UTC son las 10:00 en Madrid.
var lunesDiez = time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)

func TestPoliticaIntentosRechazaValoresFueraDeContrato(t *testing.T) {
	base := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaAdvertir)
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*PoliticaIntentosTelefonicos){
		"sin intentos":        func(p *PoliticaIntentosTelefonicos) { p.IntentosPorProceso = 0 },
		"sin procesos":        func(p *PoliticaIntentosTelefonicos) { p.Procesos = 0 },
		"control desconocido": func(p *PoliticaIntentosTelefonicos) { p.ControlSeparacion = "avisar" },
		"sin resultados":      func(p *PoliticaIntentosTelefonicos) { p.ResultadosSinContacto = nil },
		"resultado inventado": func(p *PoliticaIntentosTelefonicos) { p.ResultadosSinContacto = []string{"colgado"} },
		"resultado repetido":  func(p *PoliticaIntentosTelefonicos) { p.ResultadosSinContacto = []string{"buzon", "buzon"} },
		"franja invertida":    func(p *PoliticaIntentosTelefonicos) { p.Franja.DesdeMinuto = 15 * 60 },
		"franja sin control":  func(p *PoliticaIntentosTelefonicos) { p.Franja.Control = "" },
		"separacion negativa": func(p *PoliticaIntentosTelefonicos) { p.SeparacionMinima = -time.Hour },
		"separacion fraccion": func(p *PoliticaIntentosTelefonicos) { p.SeparacionMinima = time.Millisecond },
	}
	for nombre, cambiar := range casos {
		p := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaAdvertir)
		cambiar(&p)
		if !errors.Is(p.Validar(), ErrPoliticaIntentosInvalida) {
			t.Errorf("%s: política aceptada", nombre)
		}
	}
}

func TestEvaluarIntentoAplicaSeparacionYAgotamiento(t *testing.T) {
	p := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaAdvertir)
	primero := ResumenIntentosTelefonicos{SinContacto: 1, UltimoIntento: lunesDiez}
	if _, err := EvaluarIntentoTelefonico(p, primero, lunesDiez.Add(119*time.Minute), true); !errors.Is(err, ErrIntentoAntesDeSeparacion) {
		t.Fatalf("segundo intento a 1 h 59 min: %v", err)
	}
	if avisos, err := EvaluarIntentoTelefonico(p, primero, lunesDiez.Add(2*time.Hour), true); err != nil || len(avisos) != 0 {
		t.Fatalf("segundo intento a 2 h: %v %v", avisos, err)
	}
	// Un intento registrado con hora anterior al último también respeta la separación.
	if _, err := EvaluarIntentoTelefonico(p, primero, lunesDiez.Add(-time.Hour), true); !errors.Is(err, ErrIntentoAntesDeSeparacion) {
		t.Fatalf("intento anterior cercano: %v", err)
	}
	agotado := ResumenIntentosTelefonicos{SinContacto: 4, UltimoIntento: lunesDiez}
	if _, err := EvaluarIntentoTelefonico(p, agotado, lunesDiez.Add(24*time.Hour), true); !errors.Is(err, ErrIntentosContactoAgotados) {
		t.Fatalf("quinto intento: %v", err)
	}
	contactado := ResumenIntentosTelefonicos{SinContacto: 4, Contactado: true, UltimoIntento: lunesDiez}
	if avisos, err := EvaluarIntentoTelefonico(p, contactado, lunesDiez.Add(time.Minute), true); err != nil || avisos != nil {
		t.Fatalf("tras contacto no se controla: %v %v", avisos, err)
	}
	advertir := politicaIntentosPrueba(t, ControlReglaAdvertir, ControlReglaAdvertir)
	avisos, err := EvaluarIntentoTelefonico(advertir, primero, lunesDiez.Add(time.Hour), true)
	if err != nil || !slices.Equal(avisos, []string{AvisoIntentoAntesDeSeparacion}) {
		t.Fatalf("modo advertir: %v %v", avisos, err)
	}
}

func TestEvaluarIntentoAplicaFranjaYDiaHabil(t *testing.T) {
	p := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaAdvertir)
	tarde := lunesDiez.Add(5 * time.Hour) // 15:00 en Madrid
	avisos, err := EvaluarIntentoTelefonico(p, ResumenIntentosTelefonicos{}, tarde, false)
	if err != nil || !slices.Equal(avisos, []string{AvisoIntentoFueraDeFranja, AvisoIntentoDiaNoHabil}) {
		t.Fatalf("avisos de franja: %v %v", avisos, err)
	}
	limite := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) // 14:00: fuera
	if avisos, _ := EvaluarIntentoTelefonico(p, ResumenIntentosTelefonicos{}, limite, true); !slices.Equal(avisos, []string{AvisoIntentoFueraDeFranja}) {
		t.Fatalf("14:00 debe quedar fuera: %v", avisos)
	}
	impedir := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaImpedir)
	if _, err := EvaluarIntentoTelefonico(impedir, ResumenIntentosTelefonicos{}, lunesDiez, false); !errors.Is(err, ErrIntentoFueraDeFranja) {
		t.Fatalf("día no hábil con impedir: %v", err)
	}
	sinFranja := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaImpedir)
	sinFranja.Franja = FranjaLlamadas{}
	if avisos, err := EvaluarIntentoTelefonico(sinFranja, ResumenIntentosTelefonicos{}, tarde, false); err != nil || avisos != nil {
		t.Fatalf("sin franja no se avisa: %v %v", avisos, err)
	}
}

func TestEstadoIntentosNumeraProcesosYProponeBaja(t *testing.T) {
	p := politicaIntentosPrueba(t, ControlReglaImpedir, ControlReglaAdvertir)
	contactos := []ContactoParticipacion{
		{Canal: CanalContactoTelefono, LlamamientoRef: "llam:1", Resultado: ResultadoContactoNoContesta, Instante: lunesDiez},
		{Canal: CanalContactoTelefono, LlamamientoRef: "llam:1", Resultado: ResultadoContactoNumeroErroneo, Instante: lunesDiez.Add(2 * time.Hour)},
		{Canal: CanalContactoTelefono, LlamamientoRef: "llam:2", Resultado: ResultadoContactoContactado, Instante: lunesDiez},
		{Canal: CanalContactoCorreo, LlamamientoRef: "llam:1", Resultado: ResultadoContactoNoEntregado, Instante: lunesDiez.Add(3 * time.Hour)},
	}
	resumen := ResumirIntentosTelefonicos(p, "llam:1", contactos)
	if resumen.SinContacto != 2 || resumen.Contactado || !resumen.UltimoIntento.Equal(lunesDiez.Add(2*time.Hour)) {
		t.Fatalf("resumen: %+v", resumen)
	}
	e := EstadoIntentos(p, resumen)
	if e.Proceso != 2 || e.Intento != 1 || e.Maximo != 4 || e.BajaPropuesta || !e.SiguientePermitidoDesde.Equal(lunesDiez.Add(4*time.Hour)) {
		t.Fatalf("estado tras el primer proceso: %+v", e)
	}
	final := resumen.Sumar(p, ResultadoContactoNoContesta, lunesDiez.Add(24*time.Hour)).Sumar(p, ResultadoContactoBuzon, lunesDiez.Add(26*time.Hour))
	if e := EstadoIntentos(p, final); !e.BajaPropuesta || e.Proceso != 0 {
		t.Fatalf("dos procesos sin contacto deben proponer la baja: %+v", e)
	}
	if e := EstadoIntentos(p, final.Sumar(p, ResultadoContactoContactado, lunesDiez.Add(27*time.Hour))); e.BajaPropuesta || !e.Contactado {
		t.Fatalf("con contacto no hay propuesta: %+v", e)
	}
}

func TestContactoAdmiteResultadosNuevos(t *testing.T) {
	for _, resultado := range []string{ResultadoContactoNumeroErroneo, ResultadoContactoNoEntregado} {
		if !slices.Contains(ResultadosContactoParticipacion(), resultado) {
			t.Fatalf("%s no está en el catálogo de resultados", resultado)
		}
	}
}
