package domain

import (
	"errors"
	"testing"
	"time"
)

func borradorPrueba() BorradorComision {
	return BorradorComision{PersonaRef: "per_001", Objeto: "Visita", FechaInicio: "2026-01-02", FechaFin: "2026-01-02", HoraInicio: "09:00", HoraFin: "11:00", VehiculoPropio: true,
		Gastos: GastosComision{"12.50", "20.00", "1.25"}, Ruta: RutaVersionada{Fuente: "osrm_interno", Version: "grafo-v1", Referencia: "RUTA-001", CatalogoVersion: "catalogo-v1", AlternativaRef: "ALT-001", Recomendada: true, Kilometros: "10.50", Paradas: []ParadaRuta{{"A1", "Origen", 37, -3}, {"B1", "Destino", 38, -3}}, Tramos: []TramoRuta{{"A1", "B1", "10.00", 15, "0.50", "Desvío declarado"}}, Trazado: [][2]float64{{37, -3}, {38, -3}}}}
}
func politicaPrueba() PoliticaKilometraje {
	return PoliticaKilometraje{"pol_sintetica", "v1", "0.5000"}
}
func TestPrepararBorradorConservaGastosYRecalculaExacto(t *testing.T) {
	b := borradorPrueba()
	b.Desglose.TotalEUR = "999.00"
	x, e := PrepararBorrador(b, politicaPrueba(), time.UTC)
	if e != nil || x.Validar() != nil || x.Desglose.KilometrajeEUR != "5.25" || x.Desglose.TotalEUR != "39.00" || x.Liquidable || x.Estado != "borrador" || x.ContextoPersonal != "contexto_personal_pendiente" {
		t.Fatalf("borrador incompleto o cálculo inexacto: %v", e)
	}
	b.Ruta.Paradas[0].Nombre = "Mutada"
	b.Ruta.Tramos[0].MotivoAjuste = "Mutado"
	b.Ruta.Trazado[0][0] = 0
	if x.Ruta.Paradas[0].Nombre == "Mutada" || x.Ruta.Tramos[0].MotivoAjuste == "Mutado" || x.Ruta.Trazado[0][0] == 0 {
		t.Fatal("alias mutable")
	}
}
func TestPrepararBorradorRedondeoDecimalYNoVehiculo(t *testing.T) {
	b := borradorPrueba()
	b.Ruta.Kilometros = "0.01"
	b.Ruta.Tramos[0].Kilometros = "0.01"
	b.Ruta.Tramos[0].AjusteKilometros = "0"
	b.Ruta.Tramos[0].MotivoAjuste = ""
	x, e := PrepararBorrador(b, politicaPrueba(), time.UTC)
	if e != nil || x.Desglose.KilometrajeEUR != "0.01" {
		t.Fatal("mitad exacta sin céntimo", e)
	}
	b.VehiculoPropio = false
	x, e = PrepararBorrador(b, politicaPrueba(), time.UTC)
	if e != nil || x.Desglose.KilometrajeEUR != "0.00" || x.Desglose.TotalEUR != "33.75" {
		t.Fatal("kilometraje sin vehículo propio", e)
	}
}
func TestBorradorRechazaManipulaciones(t *testing.T) {
	for nombre, mutar := range map[string]func(*BorradorComision){
		"importe exponente":      func(b *BorradorComision) { b.Gastos.OtrosEUR = "1e2" },
		"importe precisión":      func(b *BorradorComision) { b.Gastos.OtrosEUR = "0.001" },
		"total manipulado":       func(b *BorradorComision) { b.Desglose.TotalEUR = "0.00" },
		"km discordante":         func(b *BorradorComision) { b.Ruta.Kilometros = "100.00" },
		"tramo ajeno":            func(b *BorradorComision) { b.Ruta.Tramos[0].DestinoCodigo = "C1" },
		"ajuste sin motivo":      func(b *BorradorComision) { b.Ruta.Tramos[0].MotivoAjuste = "" },
		"alternativa sin motivo": func(b *BorradorComision) { b.Ruta.Recomendada = false },
		"ruta liquidable":        func(b *BorradorComision) { b.Ruta.Liquidable = true },
		"estado aprobado":        func(b *BorradorComision) { b.Estado = "aprobada" },
		"contexto inventado":     func(b *BorradorComision) { b.ContextoPersonal = "verificado" },
		"instante discordante":   func(b *BorradorComision) { b.Inicio = b.Inicio.Add(time.Hour) },
		"tarifa excesiva":        func(b *BorradorComision) { b.PoliticaKilometraje.TarifaEURPorKM = "1001.0000" },
	} {
		t.Run(nombre, func(t *testing.T) {
			b, e := PrepararBorrador(borradorPrueba(), politicaPrueba(), time.UTC)
			if e != nil {
				t.Fatal(e)
			}
			mutar(&b)
			if !errors.Is(b.Validar(), ErrBorradorComisionInvalido) {
				t.Fatal("aceptado")
			}
		})
	}
}
func TestCivilZonaGobernadaYAmbiguedadDST(t *testing.T) {
	zona, e := time.LoadLocation("Europe/Madrid")
	if e != nil {
		t.Fatal(e)
	}
	for _, civil := range [][2]string{{"2026-03-29", "02:30"}, {"2026-10-25", "02:30"}, {"2026-02-30", "08:00"}, {"2026-01-01", "8:00"}} {
		if _, e := ResolverInstanteCivil(civil[0], civil[1], zona); e == nil {
			t.Fatalf("civil inválido aceptado: %v", civil)
		}
	}
	v, e := ResolverInstanteCivil("2026-07-01", "08:00", zona)
	if e != nil || v.Format(time.RFC3339) != "2026-07-01T06:00:00Z" {
		t.Fatal("zona no respetada", e)
	}
	if _, e := ResolverInstanteCivil("2026-07-01", "08:00", nil); e == nil {
		t.Fatal("zona inventada")
	}
}

func TestBorradorRutaDeclaradaYResumenSinGeometria(t *testing.T) {
	b := borradorPrueba()
	b.ProcedenciaRuta = "verificada"
	b.RevalidacionRutaRequerida = false
	x, e := PrepararBorrador(b, politicaPrueba(), time.UTC)
	if e != nil || x.ProcedenciaRuta != ProcedenciaRutaDeclarada || !x.RevalidacionRutaRequerida || x.Validar() != nil {
		t.Fatal("se acreditó la declaración del cliente")
	}
	resumen := x.Resumen()
	if resumen.ValidarResumen() != nil || resumen.Validar() == nil || len(resumen.Ruta.Trazado) != 0 || len(resumen.Ruta.Paradas) != 0 || len(resumen.Ruta.Tramos) != 0 || resumen.RutaEtiquetas[0] != "Origen" {
		t.Fatal("proyección de listado inválida")
	}
	resumen.RutaEtiquetas[0] = "Otra"
	if x.RutaEtiquetas[0] != "Origen" {
		t.Fatal("alias de etiquetas")
	}
	for _, mutar := range []func(*BorradorComision){
		func(b *BorradorComision) { b.ProcedenciaRuta = "osrm_verificada" },
		func(b *BorradorComision) { b.RevalidacionRutaRequerida = false },
		func(b *BorradorComision) { b.RutaEtiquetas[0] = "Origen ajeno" },
	} {
		copia := x.Clonar()
		mutar(&copia)
		if copia.Validar() == nil {
			t.Fatal("procedencia o etiquetas manipuladas aceptadas")
		}
	}
}
