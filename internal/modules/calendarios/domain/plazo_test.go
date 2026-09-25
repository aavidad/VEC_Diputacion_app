package domain

import (
	"errors"
	"testing"
	"time"
)

func fecha(t *testing.T, s string) FechaCivil {
	t.Helper()
	f, err := ParsearFechaCivil(s)
	if err != nil {
		t.Fatalf("fecha %q: %v", s, err)
	}
	return f
}

func version(t *testing.T, id string, ambito Ambito, anio int, efecto Efecto, fechas ...string) VersionConDias {
	t.Helper()
	v := VersionConDias{Version: VersionCalendario{
		ID: id, Ambito: ambito, Anio: anio, Numero: 1, Denominacion: "Calendario de prueba",
		Procedencia:   Procedencia{Norma: "Datos sintéticos de prueba", Referencia: "prueba", PublicadaEn: fecha(t, "2025-01-01"), Sintetica: true},
		ConocidoDesde: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}}
	if ambito.Tipo == AmbitoLocal || ambito.Tipo == AmbitoCentro {
		v.Version.ComunidadRef = "es-an"
	}
	if ambito.Tipo == AmbitoCentro {
		v.Version.MunicipioRef = "municipio:ine:18087"
	}
	for _, f := range fechas {
		v.Dias = append(v.Dias, DiaSenalado{Fecha: fecha(t, f), Efecto: efecto, Denominacion: "Día " + f})
	}
	return v
}

var (
	nacional   = Ambito{Tipo: AmbitoNacional, Ref: ReferenciaNacional}
	andalucia  = Ambito{Tipo: AmbitoAutonomico, Ref: "es-an"}
	granada    = Ambito{Tipo: AmbitoLocal, Ref: "municipio:ine:18087"}
	residencia = Ambito{Tipo: AmbitoLocal, Ref: "municipio:ine:18098"}
	centroRRHH = Ambito{Tipo: AmbitoCentro, Ref: "centro-530"}
)

// Fiestas de 2026 según BOE-A-2025-21667 y Decreto 101/2025 (BOJA 93/2025).
func calendario2026(t *testing.T, extra ...VersionConDias) *Calendario {
	t.Helper()
	versiones := append([]VersionConDias{
		version(t, "calendario:nacional:2026:v1", nacional, 2026, EfectoFestivo,
			"2026-01-01", "2026-01-06", "2026-04-03", "2026-05-01", "2026-08-15", "2026-10-12", "2026-12-08", "2026-12-25"),
		version(t, "calendario:andalucia:2026:v1", andalucia, 2026, EfectoFestivo,
			"2026-02-28", "2026-04-02", "2026-11-02", "2026-12-07"),
		version(t, "calendario:granada:2026:v1", granada, 2026, EfectoFestivo, "2026-05-11", "2026-09-14"),
		version(t, "calendario:centro-530:2026:v1", centroRRHH, 2026, EfectoNoLaborable, "2026-12-24", "2026-12-31"),
	}, extra...)
	anios := []int{2026}
	for _, v := range extra {
		if v.Version.Anio != 2026 {
			anios = append(anios, v.Version.Anio)
		}
	}
	c, err := NuevoCalendario(anios, versiones)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func calcular(t *testing.T, c CalendarioComputo, inicio string, u UnidadPlazo, n int) ResultadoPlazo {
	t.Helper()
	r, err := CalcularPlazo(SolicitudPlazo{Inicio: fecha(t, inicio), Unidad: u, Cantidad: n}, c)
	if err != nil {
		t.Fatalf("plazo %s %d %s: %v", inicio, n, u, err)
	}
	return r
}

func TestDiasHabilesSaltanSemanaSantaYFinesDeSemana(t *testing.T) {
	r := calcular(t, calendario2026(t), "2026-04-01", UnidadDiasHabiles, 10)
	if r.PrimerDia.String() != "2026-04-02" || r.Vencimiento.String() != "2026-04-17" || r.Prorrogado {
		t.Fatalf("resultado inesperado: %+v", r)
	}
	if len(r.DiasExcluidos) != 6 { // 2, 3, 4, 5, 11 y 12 de abril
		t.Fatalf("excluidos: %+v", r.DiasExcluidos)
	}
}

func TestNaturalesQueTerminanEnFestivoSeProrrogan(t *testing.T) {
	// 23 oct + 10 naturales = 2 nov (traslado andaluz de Todos los Santos).
	r := calcular(t, calendario2026(t), "2026-10-23", UnidadDiasNaturales, 10)
	if r.FinNominal.String() != "2026-11-02" || r.Vencimiento.String() != "2026-11-03" || !r.Prorrogado {
		t.Fatalf("prorroga incorrecta: %+v", r)
	}
	// Termina en sábado 28 de febrero, que además es Día de Andalucía: el
	// doble motivo no duplica el día y se prorroga al lunes 2 de marzo.
	r = calcular(t, calendario2026(t), "2026-02-18", UnidadDiasNaturales, 10)
	if r.Vencimiento.String() != "2026-03-02" || len(r.DiasExcluidos) != 2 {
		t.Fatalf("fin de semana con festivo: %+v", r)
	}
}

func TestCambioDeAnioFallaCerradoSinCalendarioSiguiente(t *testing.T) {
	_, err := CalcularPlazo(SolicitudPlazo{Inicio: fecha(t, "2026-12-28"), Unidad: UnidadDiasHabiles, Cantidad: 5}, calendario2026(t))
	if !errors.Is(err, ErrCalendarioNoCubre) {
		t.Fatalf("debe fallar cerrado sin 2027: %v", err)
	}
	c := calendario2026(t,
		version(t, "calendario:nacional:2027:v1", nacional, 2027, EfectoFestivo, "2027-01-01", "2027-01-06"),
	)
	r := calcular(t, c, "2026-12-28", UnidadDiasHabiles, 5)
	// 29, 30 y 31 (el cierre interno del 31 no es inhábil), 1 festivo, 2-3
	// fin de semana, 4 y 5 de enero.
	if r.Vencimiento.String() != "2027-01-05" {
		t.Fatalf("cambio de año: %+v", r)
	}
}

func TestMesesAplicanUltimoDiaYVeintinueveDeFebrero(t *testing.T) {
	c := calendario2026(t)
	r := calcular(t, c, "2026-01-31", UnidadMeses, 1)
	if r.FinNominal.String() != "2026-02-28" || r.Vencimiento.String() != "2026-03-02" || !r.Prorrogado {
		t.Fatalf("31 de enero + 1 mes en año no bisiesto: %+v", r)
	}
	vacio := func(anios ...int) *Calendario {
		c, err := NuevoCalendario(anios, nil)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	if r := calcular(t, vacio(2028), "2028-01-31", UnidadMeses, 1); r.Vencimiento.String() != "2028-02-29" {
		t.Fatalf("año bisiesto: %+v", r)
	}
	if r := calcular(t, vacio(2029), "2028-02-29", UnidadAnios, 1); r.FinNominal.String() != "2029-02-28" {
		t.Fatalf("29 de febrero + 1 año: %+v", r)
	}
	if r := calcular(t, vacio(2028), "2028-02-29", UnidadDiasNaturales, 1); r.Vencimiento.String() != "2028-03-01" {
		t.Fatalf("día siguiente al 29 de febrero: %+v", r)
	}
}

func TestCambioDeHoraEnMadrid(t *testing.T) {
	// 23:30 UTC del 28 de marzo son las 00:30 del 29 en hora de invierno.
	f, err := FechaCivilDe(time.Date(2026, 3, 28, 23, 30, 0, 0, time.UTC))
	if err != nil || f.String() != "2026-03-29" {
		t.Fatalf("instante de marzo: %v %v", f, err)
	}
	f, err = FechaCivilDe(time.Date(2026, 10, 24, 22, 30, 0, 0, time.UTC))
	if err != nil || f.String() != "2026-10-25" {
		t.Fatalf("instante de octubre: %v %v", f, err)
	}
	for dia, horas := range map[string]time.Duration{"2026-03-29": 23, "2026-10-25": 25, "2026-10-26": 24} {
		if d, _ := fecha(t, dia).DuracionEnMadrid(); d != horas*time.Hour {
			t.Fatalf("%s dura %v", dia, d)
		}
	}
	// Vence el viernes 23 (horario de verano) y el lunes 26 (de invierno).
	r := calcular(t, calendario2026(t), "2026-10-13", UnidadDiasNaturales, 10)
	if r.VenceAntesDe != time.Date(2026, 10, 23, 22, 0, 0, 0, time.UTC) {
		t.Fatalf("fin en verano: %v", r.VenceAntesDe)
	}
	r = calcular(t, calendario2026(t), "2026-10-15", UnidadDiasNaturales, 10)
	if r.Vencimiento.String() != "2026-10-26" || r.VenceAntesDe != time.Date(2026, 10, 26, 23, 0, 0, 0, time.UTC) {
		t.Fatalf("fin tras el cambio: %+v", r)
	}
}

func TestResidenciaYSedeSumanInhabiles(t *testing.T) {
	// Art. 30.6: inhábil en la residencia o en la sede es inhábil en todo caso.
	c := calendario2026(t, version(t, "calendario:residencia:2026:v1", residencia, 2026, EfectoFestivo, "2026-06-15"))
	r := calcular(t, c, "2026-06-12", UnidadDiasHabiles, 1)
	if r.Vencimiento.String() != "2026-06-16" {
		t.Fatalf("residencia ignorada: %+v", r)
	}
}

func TestClasificacionSeparaLaborableDeHabil(t *testing.T) {
	c := calendario2026(t)
	r, err := c.Clasificar(fecha(t, "2026-12-24"))
	if err != nil || r.InhabilAdministrativo || r.Laborable || r.FestivoOficial {
		t.Fatalf("cierre interno: %+v %v", r, err)
	}
	r, _ = c.Clasificar(fecha(t, "2026-02-28"))
	if !r.FestivoOficial || !r.FinDeSemana || len(r.Motivos) != 1 {
		t.Fatalf("festivo en sábado: %+v", r)
	}
}

func TestSolicitudesYVersionesInvalidas(t *testing.T) {
	c := calendario2026(t)
	for _, s := range []SolicitudPlazo{
		{Inicio: fecha(t, "2026-01-01"), Unidad: UnidadDiasHabiles, Cantidad: 0},
		{Inicio: fecha(t, "2026-01-01"), Unidad: UnidadDiasHabiles, Cantidad: 251},
		{Inicio: fecha(t, "2026-01-01"), Unidad: "horas", Cantidad: 1},
		{Unidad: UnidadMeses, Cantidad: 1},
	} {
		if _, err := CalcularPlazo(s, c); !errors.Is(err, ErrPlazoInvalido) {
			t.Fatalf("%+v aceptada: %v", s, err)
		}
	}
	for _, texto := range []string{"2026-02-29", "2026-13-01", "26-01-01", "2026/01/01", "２026-01-01"} {
		if _, err := ParsearFechaCivil(texto); err == nil {
			t.Fatalf("%q aceptada", texto)
		}
	}
	oficialConCierre := version(t, "calendario:x:2026:v1", nacional, 2026, EfectoNoLaborable, "2026-03-03")
	if _, err := NuevoCalendario([]int{2026}, []VersionConDias{oficialConCierre}); err == nil {
		t.Fatal("un calendario oficial no declara cierres internos")
	}
	duplicada := version(t, "calendario:nacional:2026:v2", nacional, 2026, EfectoFestivo)
	if _, err := NuevoCalendario([]int{2026}, []VersionConDias{version(t, "calendario:nacional:2026:v1", nacional, 2026, EfectoFestivo), duplicada}); err == nil {
		t.Fatal("dos versiones del mismo ámbito y año no se mezclan")
	}
}
