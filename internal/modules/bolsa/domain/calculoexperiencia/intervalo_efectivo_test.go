package calculoexperiencia

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestNormalizarPeriodoEfectivoMismoDiaSegunExtremo(t *testing.T) {
	dia := fechaEntrada(t, 2026, 7, 17)
	periodo, _ := NuevoPeriodoServicioCerrado(dia, dia)

	inclusivo, existe, err := normalizarPeriodoEfectivo(
		periodo, dia, reglasbaremo.ExtremoFinalInclusivo,
	)
	comprobarIntervaloEfectivo(t, inclusivo, existe, err, "2026-07-17", "2026-07-18", 1)

	exclusivo, existe, err := normalizarPeriodoEfectivo(
		periodo, dia, reglasbaremo.ExtremoFinalExclusivo,
	)
	if err != nil || existe || exclusivo.EsValido() {
		t.Fatalf("mismo dia exclusivo no vacio: %#v, %t, %v", exclusivo, existe, err)
	}
}

func TestNormalizarPeriodoEfectivoRecortaPorCorteInclusivo(t *testing.T) {
	periodo, _ := NuevoPeriodoServicioCerrado(
		fechaEntrada(t, 2026, 7, 15),
		fechaEntrada(t, 2026, 7, 25),
	)
	corte := fechaEntrada(t, 2026, 7, 17)

	for _, extremo := range []reglasbaremo.TratamientoExtremoFinal{
		reglasbaremo.ExtremoFinalInclusivo,
		reglasbaremo.ExtremoFinalExclusivo,
	} {
		intervalo, existe, err := normalizarPeriodoEfectivo(periodo, corte, extremo)
		comprobarIntervaloEfectivo(
			t, intervalo, existe, err, "2026-07-15", "2026-07-18", 3,
		)
	}
}

func TestNormalizarPeriodoEfectivoPosteriorAlCorteEsVacio(t *testing.T) {
	periodo, _ := NuevoPeriodoServicioCerrado(
		fechaEntrada(t, 2026, 7, 18),
		fechaEntrada(t, 2026, 7, 31),
	)
	intervalo, existe, err := normalizarPeriodoEfectivo(
		periodo, fechaEntrada(t, 2026, 7, 17), reglasbaremo.ExtremoFinalInclusivo,
	)
	if err != nil || existe || intervalo.EsValido() {
		t.Fatalf("periodo posterior no vacio: %#v, %t, %v", intervalo, existe, err)
	}
}

func TestNormalizarPeriodoEfectivoAbiertoIncluyeElCorte(t *testing.T) {
	periodo, _ := NuevoPeriodoServicioEnCurso(fechaEntrada(t, 2026, 7, 15))
	intervalo, existe, err := normalizarPeriodoEfectivo(
		periodo, fechaEntrada(t, 2026, 7, 17), reglasbaremo.ExtremoFinalExclusivo,
	)
	comprobarIntervaloEfectivo(
		t, intervalo, existe, err, "2026-07-15", "2026-07-18", 3,
	)
}

func TestNormalizarPeriodoEfectivoCuentaDiaBisiesto(t *testing.T) {
	periodo, _ := NuevoPeriodoServicioCerrado(
		fechaEntrada(t, 2024, 2, 28),
		fechaEntrada(t, 2024, 3, 1),
	)
	intervalo, existe, err := normalizarPeriodoEfectivo(
		periodo, fechaEntrada(t, 2024, 12, 31), reglasbaremo.ExtremoFinalInclusivo,
	)
	comprobarIntervaloEfectivo(
		t, intervalo, existe, err, "2024-02-28", "2024-03-02", 3,
	)
}

func TestNormalizarPeriodoEfectivoRecortaFinMaximoAntesDeAvanzar(t *testing.T) {
	periodo, _ := NuevoPeriodoServicioCerrado(
		fechaEntrada(t, 2026, 7, 15),
		fechaEntrada(t, 9999, 12, 31),
	)
	intervalo, existe, err := normalizarPeriodoEfectivo(
		periodo, fechaEntrada(t, 2026, 7, 17), reglasbaremo.ExtremoFinalInclusivo,
	)
	comprobarIntervaloEfectivo(
		t, intervalo, existe, err, "2026-07-15", "2026-07-18", 3,
	)
}

func TestNormalizarPeriodoEfectivoRechazaEntradasInvalidas(t *testing.T) {
	periodo, _ := NuevoPeriodoServicioEnCurso(fechaEntrada(t, 2026, 7, 15))
	casos := []struct {
		nombre   string
		periodo  PeriodoServicio
		corte    baremacion.FechaCivil
		extremo  reglasbaremo.TratamientoExtremoFinal
		esperado error
	}{
		{"periodo", PeriodoServicio{}, fechaEntrada(t, 2026, 7, 17), reglasbaremo.ExtremoFinalInclusivo, ErrValorInvalido},
		{"corte", periodo, baremacion.FechaCivil{}, reglasbaremo.ExtremoFinalInclusivo, ErrValorInvalido},
		{"extremo", periodo, fechaEntrada(t, 2026, 7, 17), reglasbaremo.TratamientoExtremoFinal("inventado"), ErrValorInvalido},
		{"corte_maximo", periodo, fechaEntrada(t, 9999, 12, 31), reglasbaremo.ExtremoFinalInclusivo, ErrFueraDeLimites},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, _, err := normalizarPeriodoEfectivo(caso.periodo, caso.corte, caso.extremo)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("error = %v; quiere %v", err, caso.esperado)
			}
		})
	}
}

func comprobarIntervaloEfectivo(
	t *testing.T,
	intervalo baremacion.IntervaloCivil,
	existe bool,
	err error,
	desde string,
	hasta string,
	dias int64,
) {
	t.Helper()
	if err != nil || !existe || !intervalo.EsValido() {
		t.Fatalf("intervalo invalido: %#v, %t, %v", intervalo, existe, err)
	}
	numeroDias, err := intervalo.NumeroDias()
	if err != nil || intervalo.Desde().String() != desde ||
		intervalo.Hasta().String() != hasta || numeroDias != dias {
		t.Fatalf(
			"intervalo = [%s,%s), dias=%d, error=%v; quiere [%s,%s), dias=%d",
			intervalo.Desde(), intervalo.Hasta(), numeroDias, err, desde, hasta, dias,
		)
	}
}
