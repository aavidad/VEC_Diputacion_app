package application

import (
	"errors"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPrepararFuentePoliticaContactosRechazaCambioSinHuella(t *testing.T) {
	f := puertosbolsa.FuentePoliticaContactos{
		BolsaRef: "bolsa:administrativo:2026", CatalogoRef: "bolsa-reglas:2:b04.franja_llamadas",
		CatalogoHuellaSHA256: strings.Repeat("a", 64), TipoDia: dominiobolsa.TipoCalendarioHabilSede,
		SedeRef: "municipio:18087", Zona: "Europe/Madrid", DesdeMinuto: 540, HastaMinuto: 840,
		ControlFranja: dominiobolsa.ControlReglaImpedir, IntentosPorCiclo: 2, Ciclos: 2,
		SeparacionSegundos: 7200, ControlSeparacion: dominiobolsa.ControlReglaImpedir,
		ResultadosSinContacto: []string{dominiobolsa.ResultadoContactoNoContesta},
	}
	p := dominiobolsa.PoliticaContactosPublicada{
		Esquema: dominiobolsa.EsquemaPoliticaContactos, BolsaRef: f.BolsaRef, Version: 1,
		CatalogoRef: f.CatalogoRef, CatalogoHuellaSHA256: f.CatalogoHuellaSHA256,
		TipoDia: f.TipoDia, SedeRef: f.SedeRef, Zona: f.Zona,
		DesdeMinuto: f.DesdeMinuto, HastaMinuto: f.HastaMinuto,
		ControlFranja: f.ControlFranja, IntentosPorCiclo: f.IntentosPorCiclo,
		Ciclos: f.Ciclos, SeparacionSegundos: f.SeparacionSegundos,
		ControlSeparacion: f.ControlSeparacion, ResultadosSinContacto: f.ResultadosSinContacto,
	}
	var err error
	f.HuellaFuenteSHA256, err = p.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := prepararFuentePoliticaContactos(f, f.BolsaRef)
	if err != nil || preparada.Validar() != nil {
		t.Fatalf("fuente integra = %+v, %v", preparada, err)
	}
	f.HastaMinuto = 900
	if _, err := prepararFuentePoliticaContactos(f, f.BolsaRef); !errors.Is(err, ErrPoliticaContactosNoDisponible) {
		t.Fatalf("politica alterada = %v", err)
	}
	f.HastaMinuto = 840
	if _, err := prepararFuentePoliticaContactos(f, "bolsa:ajena"); !errors.Is(err, ErrPoliticaContactosNoDisponible) {
		t.Fatalf("bolsa ajena = %v", err)
	}
}
