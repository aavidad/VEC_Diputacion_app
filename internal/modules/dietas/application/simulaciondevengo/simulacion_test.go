package simulaciondevengo

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func entradaPrueba(t *testing.T) Entrada {
	t.Helper()
	b, err := os.ReadFile("../../../../../cmd/vec-dietas/testdata/ensayo_nacional.json")
	if err != nil {
		t.Fatal(err)
	}
	var e Entrada
	if err = json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	return e
}
func TestSimulacionVersionadaReproducible(t *testing.T) {
	e := entradaPrueba(t)
	a, err := Simular(e)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Simular(e)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("non reproducible: %v", err)
	}
	if a.Liquidable || a.Procedencia != "propuesta_sin_publicar" || a.Resultado.TotalMaximoOrientativoCentimos != 1871 || a.Huellas.ReglaImportadaDeclaradaSHA256 != e.Regla.HuellaSHA256 {
		t.Fatalf("invalid snapshot: %+v", a)
	}
	original := a.Huellas.EntradaSHA256
	e.Catalogo[1].ManutencionCentimos = 4001
	c, err := Simular(e)
	if err != nil {
		t.Fatal(err)
	}
	if c.Resultado.TotalMaximoOrientativoCentimos != 2001 || c.Huellas.EntradaSHA256 == original || a.Entrada.Catalogo[1].ManutencionCentimos != 3741 || a.Huellas.ResultadoSHA256 == c.Huellas.ResultadoSHA256 {
		t.Fatal("rate change or input mutation lost")
	}
	e.Regla.Configuracion.PorcentajeMismoDia = 75
	d, err := Simular(e)
	if err != nil {
		t.Fatal(err)
	}
	if d.Resultado.TotalMaximoOrientativoCentimos != 3001 || c.Huellas.ConfiguracionSHA256 == d.Huellas.ConfiguracionSHA256 {
		t.Fatal("configuration change lost")
	}
}
func TestLimitesHorariosYRedondeo(t *testing.T) {
	for _, tc := range []struct {
		inicio, fin string
		total       int64
	}{{"11:00", "17:00", 1871}, {"14:00", "20:00", 0}, {"11:00", "16:00", 0}, {"11:00", "16:01", 1871}, {"13:00", "17:01", 0}} {
		e := entradaPrueba(t)
		e.Inicio.Hora = tc.inicio
		e.Fin.Hora = tc.fin
		r, err := Simular(e)
		if err != nil || r.Resultado.TotalMaximoOrientativoCentimos != tc.total {
			t.Fatalf("%+v: %+v %v", tc, r, err)
		}
	}
}
func TestCambioHoraCivil(t *testing.T) {
	e := entradaPrueba(t)
	e.Inicio = Civil{"2026-10-24", "20:00"}
	e.Fin = Civil{"2026-10-25", "09:00"}
	r, err := Simular(e)
	if err != nil || len(r.Resultado.Tramos) != 2 || r.Resultado.AlojamientoTopeCentimos != 6597 || r.Resultado.ManutencionCentimos != 1871 {
		t.Fatalf("DST: %+v %v", r, err)
	}
	for _, fecha := range []string{"2026-10-25", "2026-03-29"} {
		e.Inicio = Civil{fecha, "02:30"}
		e.Fin = Civil{fecha, "17:00"}
		r, err := Simular(e)
		if err == nil || r != nil {
			t.Fatalf("ambiguous/nonexistent: %+v %v", r, err)
		}
	}
}
func TestInvalidaDevuelveNil(t *testing.T) {
	cases := []func(*Entrada){
		func(e *Entrada) { e.Esquema = "otro" },
		func(e *Entrada) { e.Inicio.Fecha = "2026-02-30" },
		func(e *Entrada) { e.Fin = e.Inicio },
		func(e *Entrada) { e.Regla.VigenteDesde = "" },
		func(e *Entrada) { e.Regla.VigenteHasta = e.Fin.Fecha },
		func(e *Entrada) { e.Regla.VigenteHasta = "2026-99-99" },
		func(e *Entrada) { e.Regla.VigenteDesde = "2026-11-01" },
		func(e *Entrada) { e.Catalogo[1].VigenteHasta = e.Fin.Fecha },
		func(e *Entrada) { e.Seleccion.ReglaHuellaDeclaradaSHA256 = "b" },
		func(e *Entrada) { e.Seleccion.ReglaRef += "otro" },
		func(e *Entrada) { e.Seleccion.VersionTarifaRef += "otro" },
		func(e *Entrada) { e.Catalogo[1].VersionRef += "otro" },
		func(e *Entrada) { e.Catalogo[2].Grupo = 2 },
		func(e *Entrada) { e.Catalogo = e.Catalogo[:1] },
		func(e *Entrada) { e.Catalogo[1].ManutencionCentimos = -1 },
		func(e *Entrada) { e.Regla.Configuracion.Liquidable = true },
		func(e *Entrada) { e.Regla.Configuracion.PorcentajeMismoDia = 101 },
		func(e *Entrada) { e.Regla.Configuracion.DiasMaximos = 1; e.Fin.Fecha = "2026-10-02" },
		func(e *Entrada) { e.Regla.PaisISO2 = "FR" },
	}
	for n, change := range cases {
		e := entradaPrueba(t)
		change(&e)
		r, err := Simular(e)
		if err == nil || r != nil {
			t.Errorf("case %d: %+v %v", n, r, err)
		}
	}
}

func TestUmbralesSalidaYRegreso(t *testing.T) {
	for _, tc := range []struct {
		inicio, fin string
		manutencion int64
	}{{"13:59", "14:00", 3741}, {"14:00", "14:00", 0}, {"14:01", "14:00", 1871}, {"21:59", "14:01", 3742}, {"22:00", "14:01", 1871}} {
		e := entradaPrueba(t)
		e.Inicio.Hora = tc.inicio
		e.Fin = Civil{"2026-10-02", tc.fin}
		r, err := Simular(e)
		if err != nil || r.Resultado.ManutencionCentimos != tc.manutencion || r.Resultado.AlojamientoTopeCentimos != 6597 {
			t.Fatalf("%+v: %+v %v", tc, r, err)
		}
	}
}

func TestHuellaFallaSinValorParcial(t *testing.T) {
	h, err := huella(make(chan int))
	if err != ErrSerializacion || h != "" {
		t.Fatalf("%s %v", h, err)
	}
}
