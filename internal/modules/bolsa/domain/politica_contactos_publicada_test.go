package domain

import (
	"errors"
	"strings"
	"testing"
)

func politicaContactosPublicadaPrueba(t *testing.T) PoliticaContactosPublicada {
	t.Helper()
	p := PoliticaContactosPublicada{
		Esquema: EsquemaPoliticaContactos, BolsaRef: "bolsa:administrativo:2026", Version: 1,
		CatalogoRef: "bolsa-reglas:2:b04.franja_llamadas", CatalogoHuellaSHA256: strings.Repeat("a", 64),
		TipoDia: TipoCalendarioHabilSede, SedeRef: "municipio:18087", Zona: "Europe/Madrid",
		DesdeMinuto: 540, HastaMinuto: 840, ControlFranja: ControlReglaImpedir,
		IntentosPorCiclo: 2, Ciclos: 2, SeparacionSegundos: 7200,
		ControlSeparacion:     ControlReglaImpedir,
		ResultadosSinContacto: []string{ResultadoContactoNoContesta, ResultadoContactoBuzon},
	}
	var err error
	p.HuellaSHA256, err = p.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPoliticaContactosImpideFranjaFueraYConservaVersion(t *testing.T) {
	p := politicaContactosPublicadaPrueba(t)
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(*PoliticaContactosPublicada){
		"control solo aviso":                  func(p *PoliticaContactosPublicada) { p.ControlFranja = ControlReglaAdvertir },
		"zona ajena":                          func(p *PoliticaContactosPublicada) { p.Zona = "UTC" },
		"catalogo sin huella":                 func(p *PoliticaContactosPublicada) { p.CatalogoHuellaSHA256 = "" },
		"minutos alterados sin nueva version": func(p *PoliticaContactosPublicada) { p.HastaMinuto = 900 },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := p
			cambiar(&alterada)
			if !errors.Is(alterada.Validar(), ErrPoliticaContactosPublicadaInvalida) {
				t.Fatal("politica sin gobierno admitida")
			}
		})
	}
	nueva := p
	nueva.Version = 2
	nueva.VersionAnterior = p.HuellaSHA256
	nueva.DesdeMinuto = 480
	nueva.HastaMinuto = 780
	var err error
	nueva.HuellaSHA256, err = nueva.HuellaCanonica()
	if err != nil || nueva.Validar() != nil || nueva.HuellaSHA256 == p.HuellaSHA256 {
		t.Fatalf("version configurable = %v, %v", nueva.HuellaSHA256, err)
	}
}
