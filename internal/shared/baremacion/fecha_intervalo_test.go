package baremacion

import (
	"encoding/json"
	"errors"
	"testing"
)

func fechaPrueba(t *testing.T, anio, mes, dia int) FechaCivil {
	t.Helper()
	fecha, err := NuevaFechaCivil(anio, mes, dia)
	if err != nil {
		t.Fatalf("NuevaFechaCivil(%d,%d,%d): %v", anio, mes, dia, err)
	}
	return fecha
}

func TestFechaCivilRespetaCalendarioGregoriano(t *testing.T) {
	t.Parallel()

	bisiesta := fechaPrueba(t, 2000, 2, 29)
	if bisiesta.Anio() != 2000 || bisiesta.Mes() != 2 || bisiesta.Dia() != 29 {
		t.Fatalf("componentes = %d-%d-%d", bisiesta.Anio(), bisiesta.Mes(), bisiesta.Dia())
	}
	siguiente, err := bisiesta.Siguiente()
	if err != nil || siguiente.String() != "2000-03-01" {
		t.Fatalf("siguiente = %s, %v", siguiente, err)
	}
	if _, err := NuevaFechaCivil(1900, 2, 29); !errors.Is(err, ErrFechaInvalida) {
		t.Fatalf("1900-02-29: error = %v", err)
	}
	if _, err := NuevaFechaCivil(2024, 2, 29); err != nil {
		t.Fatalf("2024-02-29: %v", err)
	}
	finAnio := fechaPrueba(t, 2023, 12, 31)
	primero, err := finAnio.Siguiente()
	if err != nil || primero.String() != "2024-01-01" {
		t.Fatalf("cambio de anio = %s, %v", primero, err)
	}
	maxima := fechaPrueba(t, AnioCivilMaximo, 12, 31)
	if _, err := maxima.Siguiente(); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("dia tras fecha maxima: error = %v", err)
	}
}

func TestFechaCivilRoundtripJSON(t *testing.T) {
	t.Parallel()

	original := fechaPrueba(t, 2026, 7, 16)
	datos, err := json.Marshal(original)
	if err != nil || string(datos) != `"2026-07-16"` {
		t.Fatalf("Marshal = %s, %v", datos, err)
	}
	var recuperada FechaCivil
	if err := json.Unmarshal(datos, &recuperada); err != nil || recuperada != original {
		t.Fatalf("roundtrip = %s, %v", recuperada, err)
	}
	for _, noCanonica := range []string{`"2026-7-16"`, `"0000-01-01"`, `"2023-02-29"`, `20260716`} {
		var destino FechaCivil
		if err := json.Unmarshal([]byte(noCanonica), &destino); err == nil {
			t.Errorf("Unmarshal(%s) no rechazo", noCanonica)
		}
	}
}

func TestIntervaloCivilSemibiertoDiaUnico(t *testing.T) {
	t.Parallel()

	dia := fechaPrueba(t, 2024, 2, 29)
	intervalo, err := IntervaloDeUnDia(dia)
	if err != nil {
		t.Fatalf("IntervaloDeUnDia: %v", err)
	}
	dias, err := intervalo.NumeroDias()
	if err != nil || dias != 1 {
		t.Fatalf("NumeroDias = %d, %v", dias, err)
	}
	if !intervalo.Contiene(dia) {
		t.Fatal("el intervalo no contiene su inicio")
	}
	if intervalo.Desde() != dia {
		t.Fatalf("Desde = %s", intervalo.Desde())
	}
	if intervalo.Contiene(intervalo.Hasta()) {
		t.Fatal("el intervalo contiene su extremo exclusivo")
	}
	if _, err := NuevoIntervaloCivil(dia, dia); !errors.Is(err, ErrIntervaloVacio) {
		t.Fatalf("intervalo vacio: error = %v", err)
	}
	ultimoDia := fechaPrueba(t, AnioCivilMaximo, 12, 31)
	if _, err := IntervaloDeUnDia(ultimoDia); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("intervalo del ultimo dia representable: error = %v", err)
	}
}

func TestIntervalosDistinguenAdyacenciaYSolape(t *testing.T) {
	t.Parallel()

	diaUno := fechaPrueba(t, 2026, 1, 1)
	diaDos := fechaPrueba(t, 2026, 1, 2)
	diaTres := fechaPrueba(t, 2026, 1, 3)
	diaCuatro := fechaPrueba(t, 2026, 1, 4)
	primero, _ := NuevoIntervaloCivil(diaUno, diaTres)
	adyacente, _ := NuevoIntervaloCivil(diaTres, diaCuatro)
	solapado, _ := NuevoIntervaloCivil(diaDos, diaCuatro)
	if !primero.EsAdyacente(adyacente) || primero.Solapa(adyacente) {
		t.Fatal("los intervalos adyacentes no deben solapar")
	}
	if primero.EsAdyacente(solapado) || !primero.Solapa(solapado) {
		t.Fatal("no se detecto el solape")
	}
}

func TestIntervaloCivilJSONCanonicoYEstricto(t *testing.T) {
	t.Parallel()

	original, _ := NuevoIntervaloCivil(
		fechaPrueba(t, 2026, 1, 1),
		fechaPrueba(t, 2026, 2, 1),
	)
	datos, err := json.Marshal(original)
	esperado := `{"desde":"2026-01-01","hasta":"2026-02-01"}`
	if err != nil || string(datos) != esperado {
		t.Fatalf("Marshal = %s, %v", datos, err)
	}
	var recuperado IntervaloCivil
	if err := json.Unmarshal(datos, &recuperado); err != nil || recuperado != original {
		t.Fatalf("roundtrip = %v, %v", recuperado, err)
	}
	conDesconocido := `{"desde":"2026-01-01","hasta":"2026-02-01","zona":"UTC"}`
	if err := json.Unmarshal([]byte(conDesconocido), &recuperado); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("campo desconocido: error = %v", err)
	}
	noCanonicos := []string{
		`{"Desde":"2026-01-01","hasta":"2026-02-01"}`,
		`{"DESDE":"2026-01-01","hasta":"2026-02-01"}`,
		`{"desde":"2026-01-01","Hasta":"2026-02-01"}`,
		`{"desde":"2026-01-01","desde":"2026-01-02","hasta":"2026-02-01"}`,
		`{"desde":"2026-01-01","hasta":"2026-02-01","hasta":"2026-03-01"}`,
	}
	for _, noCanonico := range noCanonicos {
		if err := json.Unmarshal([]byte(noCanonico), &recuperado); !errors.Is(err, ErrValorNoCanonico) {
			t.Errorf("Unmarshal(%s): error = %v", noCanonico, err)
		}
	}
}

func FuzzFechaCivilRoundtripJSON(f *testing.F) {
	semillas := [][3]int{{2000, 2, 29}, {1900, 2, 29}, {2026, 7, 16}, {1, 1, 1}, {9999, 12, 31}}
	for _, semilla := range semillas {
		f.Add(semilla[0], semilla[1], semilla[2])
	}
	f.Fuzz(func(t *testing.T, anio, mes, dia int) {
		original, err := NuevaFechaCivil(anio, mes, dia)
		if err != nil {
			return
		}
		datos, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%s): %v", original, err)
		}
		var recuperada FechaCivil
		if err := json.Unmarshal(datos, &recuperada); err != nil {
			t.Fatalf("Unmarshal(%s): %v", datos, err)
		}
		if recuperada != original {
			t.Fatalf("roundtrip: %v != %v", recuperada, original)
		}
	})
}

func FuzzIntervaloCivilNoConfundeAdyacencia(f *testing.F) {
	f.Add(2024, 2, 28)
	f.Add(2026, 7, 16)
	f.Fuzz(func(t *testing.T, anio, mes, dia int) {
		inicio, err := NuevaFechaCivil(anio, mes, dia)
		if err != nil {
			return
		}
		medio, err := inicio.Siguiente()
		if err != nil {
			return
		}
		fin, err := medio.Siguiente()
		if err != nil {
			return
		}
		primero, _ := NuevoIntervaloCivil(inicio, medio)
		segundo, _ := NuevoIntervaloCivil(medio, fin)
		if !primero.EsAdyacente(segundo) || primero.Solapa(segundo) {
			t.Fatalf("adyacencia incorrecta: %s, %s, %s", inicio, medio, fin)
		}
		if primero.EsAdyacente(segundo) != segundo.EsAdyacente(primero) ||
			primero.Solapa(segundo) != segundo.Solapa(primero) {
			t.Fatalf("la relacion no es simetrica: %s, %s, %s", inicio, medio, fin)
		}
		dias, err := primero.NumeroDias()
		if err != nil || dias != 1 {
			t.Fatalf("duracion de un dia = %d, %v", dias, err)
		}
	})
}

func FuzzIntervaloCivilJSONHostil(f *testing.F) {
	semillas := [][]byte{
		[]byte(`{"desde":"2026-01-01","hasta":"2026-02-01"}`),
		[]byte(`{"Desde":"2026-01-01","hasta":"2026-02-01"}`),
		[]byte(`{"desde":"2026-01-01","desde":"2026-01-02","hasta":"2026-02-01"}`),
		[]byte(`null`),
		[]byte(`{`),
		{0xff, 0x00, 0x7b},
	}
	for _, semilla := range semillas {
		f.Add(semilla)
	}
	original, _ := NuevoIntervaloCivil(
		FechaCivil{anio: 2025, mes: 1, dia: 1},
		FechaCivil{anio: 2025, mes: 2, dia: 1},
	)
	f.Fuzz(func(t *testing.T, datos []byte) {
		destino := original
		err := json.Unmarshal(datos, &destino)
		if err != nil {
			if destino != original {
				t.Fatalf("Unmarshal fallido muto el receptor: %q", datos)
			}
			return
		}
		if !destino.EsValido() {
			t.Fatalf("Unmarshal produjo intervalo invalido: %q", datos)
		}
		canonico, err := json.Marshal(destino)
		if err != nil {
			t.Fatalf("Marshal tras Unmarshal(%q): %v", datos, err)
		}
		var repetido IntervaloCivil
		if err := json.Unmarshal(canonico, &repetido); err != nil || repetido != destino {
			t.Fatalf("roundtrip canonico(%q): %v", canonico, err)
		}
	})
}
