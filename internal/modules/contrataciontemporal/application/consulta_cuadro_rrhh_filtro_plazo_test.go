package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFiltroPlazoCuadroComparteClasificacionConResumenCompleto(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	desde := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	plazos := map[domain.ClaveFase]*ports.PlazoFaseRRHH{
		"fase_vencida": plazoResumenPrueba(ports.PlazoFaseVencido, "2026-10-07"),
		"fase_hoy":     plazoResumenPrueba(ports.PlazoFaseVenceHoy, "2026-10-08"),
		"fase_semana":  plazoResumenPrueba(ports.PlazoFaseEnPlazo, "2026-10-14"),
		"fase_fuera":   plazoResumenPrueba(ports.PlazoFaseEnPlazo, "2026-10-15"),
		"fase_sin":     plazoResumenPrueba(ports.PlazoFaseNoCalculado, ""),
	}
	contextos := []ports.ContextoPlazoCuadroRRHH{
		{ExpedienteRef: "expediente:ct:001", VersionExpediente: 1, FaseClave: "fase_vencida", FaseDesde: desde},
		{ExpedienteRef: "expediente:ct:002", VersionExpediente: 1, FaseClave: "fase_hoy", FaseDesde: desde},
		{ExpedienteRef: "expediente:ct:003", VersionExpediente: 1, FaseClave: "fase_semana", FaseDesde: desde},
		{ExpedienteRef: "expediente:ct:004", VersionExpediente: 1, FaseClave: "fase_fuera", FaseDesde: desde},
		{ExpedienteRef: "expediente:ct:005", VersionExpediente: 1, FaseClave: "fase_sin", FaseDesde: desde},
		{ExpedienteRef: "expediente:ct:006", VersionExpediente: 1, FaseClave: "fase_vencida", FaseDesde: desde},
	}
	calculadora := &calculadoraResumenPrueba{plazos: plazos}
	seleccion, err := SeleccionarPlazosCuadroRRHH(context.Background(), calculadora, ahora,
		ports.FiltroPlazoVencido, contextos)
	if err != nil || seleccion.Vencidos != 2 || seleccion.VencenHoy != 1 ||
		seleccion.VencenSemana != 2 || seleccion.SinCalcular != 1 ||
		!reflect.DeepEqual(seleccion.Referencias, []string{"expediente:ct:001", "expediente:ct:006"}) ||
		calculadora.llamadas != 5 {
		t.Fatalf("selección de todo el corte incorrecta: %+v, llamadas=%d, err=%v", seleccion, calculadora.llamadas, err)
	}
	agregados := ports.AgregadosCuadroRRHH{GruposPlazo: []ports.GrupoPlazoCuadroRRHH{
		{FaseClave: "fase_vencida", Desde: desde, Numero: 2},
		{FaseClave: "fase_hoy", Desde: desde, Numero: 1},
		{FaseClave: "fase_semana", Desde: desde, Numero: 1},
		{FaseClave: "fase_fuera", Desde: desde, Numero: 1},
		{FaseClave: "fase_sin", Desde: desde, Numero: 1},
	}}
	resumen, err := resumirCuadroRRHH(context.Background(), &calculadoraResumenPrueba{plazos: plazos}, agregados, ahora)
	if err != nil || resumen.Vencidos != seleccion.Vencidos ||
		resumen.VencenHoy != seleccion.VencenHoy ||
		resumen.VencenSemana != seleccion.VencenSemana ||
		resumen.SinCalcular != seleccion.SinCalcular {
		t.Fatalf("Inicio y lista discrepan: resumen=%+v, seleccion=%+v, err=%v", resumen, seleccion, err)
	}
	semana, err := SeleccionarPlazosCuadroRRHH(context.Background(),
		&calculadoraResumenPrueba{plazos: plazos}, ahora, ports.FiltroPlazoVenceSemana, contextos)
	if err != nil || !reflect.DeepEqual(semana.Referencias, []string{"expediente:ct:002", "expediente:ct:003"}) {
		t.Fatalf("semana no coincide con sus filas: %+v, %v", semana, err)
	}
}

func TestFiltroPlazoCuadroUsaDiaCivilDeMadrid(t *testing.T) {
	antes := time.Date(2026, 10, 7, 21, 59, 0, 0, time.UTC)
	despues := time.Date(2026, 10, 7, 22, 0, 0, 0, time.UTC)
	plazo := plazoResumenPrueba(ports.PlazoFaseEnPlazo, "2026-10-14")
	hoyAntes, err := diaCivilMadrid(antes)
	if err != nil {
		t.Fatal(err)
	}
	hoyDespues, err := diaCivilMadrid(despues)
	if err != nil {
		t.Fatal(err)
	}
	a, err := clasificarPlazoCuadroRRHH(plazo, hoyAntes)
	if err != nil {
		t.Fatal(err)
	}
	b, err := clasificarPlazoCuadroRRHH(plazo, hoyDespues)
	if err != nil || a.venceSemana || !b.venceSemana {
		t.Fatalf("límite Madrid incorrecto: antes=%+v después=%+v err=%v", a, b, err)
	}
}

func TestFiltroPlazoCuadroRechazaContextoIncompleto(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	_, err := SeleccionarPlazosCuadroRRHH(context.Background(),
		&calculadoraResumenPrueba{plazos: nil}, ahora,
		ports.FiltroPlazoVencido, []ports.ContextoPlazoCuadroRRHH{{
			ExpedienteRef: "expediente:ct:001", VersionExpediente: 1,
			FaseClave: "solicitud", FaseDesde: time.Time{},
		}})
	if err == nil {
		t.Fatal("una entrada de fase ausente se interpretó como no vencida")
	}
}
