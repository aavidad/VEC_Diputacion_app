package domain

import (
	"errors"
	"testing"
	"time"
)

func calendarioContactosPrueba(t *testing.T) CalendarioContactos {
	t.Helper()
	c := CalendarioContactos{
		Esquema: EsquemaCalendarioContactos, Tipo: TipoCalendarioHabilSede,
		SedeRef: "municipio:18087", Anio: 2026, Version: 1,
		Fuentes: []VersionFuenteCalendario{
			{AmbitoTipo: "nacional", AmbitoRef: "es", VersionID: "calendario:nacional:2026:v1", Numero: 1},
			{AmbitoTipo: "autonomico", AmbitoRef: "andalucia", VersionID: "calendario:andalucia:2026:v1", Numero: 1},
			{AmbitoTipo: "local", AmbitoRef: "municipio:18087", VersionID: "calendario:granada:2026:v1", Numero: 1},
		},
	}
	for dia := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); dia.Year() == 2026; dia = dia.AddDate(0, 0, 1) {
		c.Dias = append(c.Dias, DiaCalendarioContactos{Fecha: dia.Format("2006-01-02"), Habil: dia.Weekday() != time.Saturday && dia.Weekday() != time.Sunday})
	}
	var err error
	c.HuellaSHA256, err = c.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCalendarioContactosExigeAnioCompletoYHuellaExacta(t *testing.T) {
	c := calendarioContactosPrueba(t)
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*CalendarioContactos){
		"dia omitido":                 func(c *CalendarioContactos) { c.Dias = append(c.Dias[:120:120], c.Dias[121:]...) },
		"dia repetido":                func(c *CalendarioContactos) { c.Dias[120] = c.Dias[119] },
		"dia cambiado sin version":    func(c *CalendarioContactos) { c.Dias[120].Habil = !c.Dias[120].Habil },
		"fuente cambiada sin version": func(c *CalendarioContactos) { c.Fuentes[2].VersionID = "calendario:granada:2026:v2" },
		"tipo ajeno":                  func(c *CalendarioContactos) { c.Tipo = "laboral_centro" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := c
			alterado.Fuentes = append([]VersionFuenteCalendario(nil), c.Fuentes...)
			alterado.Dias = append([]DiaCalendarioContactos(nil), c.Dias...)
			mutar(&alterado)
			if !errors.Is(alterado.Validar(), ErrCalendarioContactosInvalido) {
				t.Fatal("se acepto calendario incompleto o cambiado")
			}
		})
	}
}

func TestNuevaVersionEnlazaHuellaAnterior(t *testing.T) {
	anterior := calendarioContactosPrueba(t)
	nueva := anterior
	nueva.Version = 2
	nueva.VersionAnterior = anterior.HuellaSHA256
	nueva.Dias = append([]DiaCalendarioContactos(nil), anterior.Dias...)
	nueva.Dias[120].Habil = false
	var err error
	nueva.HuellaSHA256, err = nueva.HuellaCanonica()
	if err != nil || nueva.HuellaSHA256 == anterior.HuellaSHA256 || nueva.Validar() != nil {
		t.Fatalf("nueva version = %v, %v", nueva.HuellaSHA256, err)
	}
	nueva.VersionAnterior = ""
	if !errors.Is(nueva.Validar(), ErrCalendarioContactosInvalido) {
		t.Fatal("version sucesora sin enlace aceptada")
	}
}
