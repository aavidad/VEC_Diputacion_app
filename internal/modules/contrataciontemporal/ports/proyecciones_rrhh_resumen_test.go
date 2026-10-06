package ports

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Los agregados solo se aceptan si cuadran con los totales y los filtros de
// la misma consulta y no traen grupos vacíos, repetidos o futuros.
func TestAgregadosCuadroRRHHCuadranConTotalesYFiltros(t *testing.T) {
	generada := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	desde := generada.Add(-48 * time.Hour)
	solicitud, err := NuevaSolicitudCuadroRRHH("", "", "", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	solicitud = solicitud.ConResumen()
	totales := &TotalesCuadroRRHH{Total: 6, EnTramitacion: 3, ConIncidencia: 1}
	valido := AgregadosCuadroRRHH{
		Recuentos: []RecuentoCuadroRRHH{
			{EstadoClave: domain.EstadoEnCurso, FaseClave: "solicitud", Numero: 3},
			{EstadoClave: domain.EstadoIncidencia, FaseClave: "fiscalizacion", Numero: 1},
			{EstadoClave: domain.EstadoCompletado, FaseClave: "nombramiento", Numero: 2},
		},
		GruposPlazo: []GrupoPlazoCuadroRRHH{
			{FaseClave: "solicitud", Desde: desde, Numero: 3},
			{FaseClave: "fiscalizacion", Desde: desde, Urgente: true, Numero: 1},
		},
	}
	if !valido.validarPara(solicitud, totales, generada) {
		t.Fatal("agregados coherentes rechazados")
	}
	mal := func(cambiar func(*AgregadosCuadroRRHH)) AgregadosCuadroRRHH {
		copia := AgregadosCuadroRRHH{
			Recuentos:   append([]RecuentoCuadroRRHH(nil), valido.Recuentos...),
			GruposPlazo: append([]GrupoPlazoCuadroRRHH(nil), valido.GruposPlazo...),
		}
		cambiar(&copia)
		return copia
	}
	casos := map[string]AgregadosCuadroRRHH{
		"total distinto":  mal(func(a *AgregadosCuadroRRHH) { a.Recuentos[2].Numero = 3 }),
		"grupos no suman": mal(func(a *AgregadosCuadroRRHH) { a.GruposPlazo[0].Numero = 2 }),
		"recuento repetido": mal(func(a *AgregadosCuadroRRHH) {
			a.Recuentos[1] = a.Recuentos[0]
			a.Recuentos[0].Numero = 1
			a.Recuentos[1].Numero = 3
		}),
		"numero cero": mal(func(a *AgregadosCuadroRRHH) {
			a.GruposPlazo = append(a.GruposPlazo, GrupoPlazoCuadroRRHH{FaseClave: "solicitud", Desde: desde})
		}),
		"desde futuro":   mal(func(a *AgregadosCuadroRRHH) { a.GruposPlazo[0].Desde = generada.Add(time.Hour) }),
		"fase no valida": mal(func(a *AgregadosCuadroRRHH) { a.GruposPlazo[0].FaseClave = "Fase Mala" }),
	}
	for nombre, agregados := range casos {
		if agregados.validarPara(solicitud, totales, generada) {
			t.Fatalf("%s: aceptado", nombre)
		}
	}
	if valido.validarPara(solicitud, nil, generada) {
		t.Fatal("aceptado sin totales")
	}
	// Con filtro de fase, un recuento de otra fase delata un alcance mayor.
	filtrada, err := NuevaSolicitudCuadroRRHH("", "", "solicitud", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if valido.validarPara(filtrada.ConResumen(), totales, generada) {
		t.Fatal("recuento fuera del filtro aceptado")
	}
	if !solicitud.Resumen() || filtrada.Resumen() {
		t.Fatal("ConResumen no marca solo la copia")
	}
	if h1, _ := filtrada.HuellaCanonicaSHA256(); func() string { h2, _ := filtrada.ConResumen().HuellaCanonicaSHA256(); return h2 }() != h1 {
		t.Fatal("el resumen no debe cambiar la huella de la consulta")
	}
}
