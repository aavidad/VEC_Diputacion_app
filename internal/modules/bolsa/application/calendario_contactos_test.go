package application

import (
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func fuenteCalendarioPrueba(t *testing.T) puertosbolsa.FuenteCalendarioContactos {
	t.Helper()
	f := puertosbolsa.FuenteCalendarioContactos{
		Tipo: dominiobolsa.TipoCalendarioHabilSede, SedeRef: "municipio:18087", Anio: 2026,
		Fuentes: []dominiobolsa.VersionFuenteCalendario{
			{AmbitoTipo: "nacional", AmbitoRef: "es", VersionID: "calendario:nacional:2026:v1", Numero: 1},
			{AmbitoTipo: "autonomico", AmbitoRef: "andalucia", VersionID: "calendario:andalucia:2026:v1", Numero: 1},
			{AmbitoTipo: "local", AmbitoRef: "municipio:18087", VersionID: "calendario:granada:2026:v1", Numero: 1},
		},
	}
	for d := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); d.Year() == 2026; d = d.AddDate(0, 0, 1) {
		f.Dias = append(f.Dias, dominiobolsa.DiaCalendarioContactos{Fecha: d.Format("2006-01-02"), Habil: d.Weekday() != time.Saturday && d.Weekday() != time.Sunday})
	}
	c := dominiobolsa.CalendarioContactos{Esquema: dominiobolsa.EsquemaCalendarioContactos, Tipo: f.Tipo,
		SedeRef: f.SedeRef, Anio: f.Anio, Version: 1, Fuentes: f.Fuentes, Dias: f.Dias}
	var err error
	f.HuellaFuenteSHA256, err = c.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestPrepararFuenteCalendarioRechazaDatosSinPublicacionIntegra(t *testing.T) {
	f := fuenteCalendarioPrueba(t)
	s := SolicitudImportarCalendarioContactos{Tipo: f.Tipo, SedeRef: f.SedeRef, Anio: f.Anio}
	c, err := prepararFuenteCalendarioContactos(f, s)
	if err != nil || c.Validar() != nil {
		t.Fatalf("fuente integra = %v, %v", c.HuellaSHA256, err)
	}
	f.Dias[100].Habil = !f.Dias[100].Habil
	if _, err := prepararFuenteCalendarioContactos(f, s); !errors.Is(err, ErrCalendarioContactosNoDisponible) {
		t.Fatalf("fuente alterada = %v", err)
	}
	f = fuenteCalendarioPrueba(t)
	f.Dias = f.Dias[:len(f.Dias)-1]
	if _, err := prepararFuenteCalendarioContactos(f, s); !errors.Is(err, ErrCalendarioContactosNoDisponible) {
		t.Fatalf("año incompleto = %v", err)
	}
	f = fuenteCalendarioPrueba(t)
	f.SedeRef = "municipio:otro"
	if _, err := prepararFuenteCalendarioContactos(f, s); !errors.Is(err, ErrCalendarioContactosNoDisponible) {
		t.Fatalf("sede ajena = %v", err)
	}
}
