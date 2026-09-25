package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

// historiaEnMemoria reproduce el contrato del repositorio: por ámbito y año,
// la versión de mayor número conocida hasta el instante pedido.
type historiaEnMemoria struct {
	versiones []domain.VersionConDias
	fallo     error
}

func (h *historiaEnMemoria) VersionesVigentes(_ context.Context, c ports.ConsultaVersiones) ([]domain.VersionConDias, error) {
	if h.fallo != nil {
		return nil, h.fallo
	}
	var r []domain.VersionConDias
	for _, a := range c.Ambitos {
		var elegida *domain.VersionConDias
		for i, v := range h.versiones {
			if v.Version.Ambito == a && v.Version.Anio == c.Anio && !v.Version.ConocidoDesde.After(c.ConocidoEn) &&
				(elegida == nil || v.Version.Numero > elegida.Version.Numero) {
				elegida = &h.versiones[i]
			}
		}
		if elegida != nil {
			r = append(r, *elegida)
		}
	}
	return r, nil
}

func (h *historiaEnMemoria) CentrosConCalendario(_ context.Context, anio int, conocido time.Time) ([]domain.VersionCalendario, error) {
	var r []domain.VersionCalendario
	for _, v := range h.versiones {
		if v.Version.Ambito.Tipo == domain.AmbitoCentro && v.Version.Anio == anio && !v.Version.ConocidoDesde.After(conocido) {
			r = append(r, v.Version)
		}
	}
	return r, nil
}

var (
	publicada = time.Date(2025, 10, 1, 8, 0, 0, 0, time.UTC)
	corregida = time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC)
	ahora     = time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
)

func f(t *testing.T, s string) domain.FechaCivil {
	t.Helper()
	v, err := domain.ParsearFechaCivil(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func v(t *testing.T, id string, a domain.Ambito, numero int, conocido time.Time, efecto domain.Efecto, fechas ...string) domain.VersionConDias {
	t.Helper()
	r := domain.VersionConDias{Version: domain.VersionCalendario{
		ID: id, Ambito: a, Anio: 2026, Numero: numero, Denominacion: "Calendario " + id,
		Procedencia:   domain.Procedencia{Norma: "Datos sintéticos", Referencia: "prueba", PublicadaEn: f(t, "2025-10-01"), Sintetica: true},
		ConocidoDesde: conocido,
	}}
	if numero > 1 {
		r.Version.SustituyeID = id + "-previa"
	}
	if a.Tipo == domain.AmbitoLocal || a.Tipo == domain.AmbitoCentro {
		r.Version.ComunidadRef = "es-an"
	}
	if a.Tipo == domain.AmbitoCentro {
		r.Version.MunicipioRef = "municipio:ine:18087"
	}
	for _, d := range fechas {
		r.Dias = append(r.Dias, domain.DiaSenalado{Fecha: f(t, d), Efecto: efecto, Denominacion: "Día " + d})
	}
	return r
}

func historia(t *testing.T) *historiaEnMemoria {
	nac := domain.Ambito{Tipo: domain.AmbitoNacional, Ref: "es"}
	and := domain.Ambito{Tipo: domain.AmbitoAutonomico, Ref: "es-an"}
	gra := domain.Ambito{Tipo: domain.AmbitoLocal, Ref: "municipio:ine:18087"}
	cen := domain.Ambito{Tipo: domain.AmbitoCentro, Ref: "centro-530"}
	return &historiaEnMemoria{versiones: []domain.VersionConDias{
		v(t, "nacional-2026-v1", nac, 1, publicada, domain.EfectoFestivo, "2026-01-01", "2026-01-06", "2026-04-03", "2026-05-01", "2026-08-15", "2026-10-12", "2026-12-08", "2026-12-25"),
		v(t, "andalucia-2026-v1", and, 1, publicada, domain.EfectoFestivo, "2026-02-28", "2026-04-02", "2026-11-02", "2026-12-07"),
		v(t, "granada-2026-v1", gra, 1, publicada, domain.EfectoFestivo, "2026-05-11", "2026-09-14"),
		// Corrección conocida en abril: la segunda fiesta local pasa al 15.
		v(t, "granada-2026-v2", gra, 2, corregida, domain.EfectoFestivo, "2026-05-11", "2026-09-15"),
		v(t, "centro-530-2026-v1", cen, 1, publicada, domain.EfectoNoLaborable, "2026-12-24", "2026-12-31"),
	}}
}

func servicio(t *testing.T, h *historiaEnMemoria) *Servicio {
	t.Helper()
	s, err := NuevoServicio(h, relojFijo(ahora))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCalendarioCentroCompletoYResumen(t *testing.T) {
	c, err := servicio(t, historia(t)).CalendarioCentro(context.Background(), ports.SolicitudCalendarioCentro{CentroRef: "centro-530", Anio: 2026})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Dias) != 365 || len(c.Versiones) != 4 || c.MunicipioRef != "municipio:ine:18087" || c.Zona != "Europe/Madrid" {
		t.Fatalf("calendario: %d días, %d versiones", len(c.Dias), len(c.Versiones))
	}
	// 104 días de fin de semana; festivos en sábado: 28 feb y 15 ago.
	if c.Resumen.FestivosOficiales != 14 || c.Resumen.HabilesAdministrativos != 365-104-12 || c.Resumen.Laborables != 365-104-12-2 || c.Resumen.NoLaborablesCentro != 2 {
		t.Fatalf("resumen: %+v", c.Resumen)
	}
}

func TestReconstruyeLoConocidoAntesDeLaCorreccion(t *testing.T) {
	s := servicio(t, historia(t))
	antes, err := s.CalendarioCentro(context.Background(), ports.SolicitudCalendarioCentro{CentroRef: "centro-530", Anio: 2026, ConocidoEn: corregida.Add(-time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	despues, err := s.CalendarioCentro(context.Background(), ports.SolicitudCalendarioCentro{CentroRef: "centro-530", Anio: 2026})
	if err != nil {
		t.Fatal(err)
	}
	dia := func(c ports.CalendarioCentro, fecha string) domain.Clasificacion {
		for _, d := range c.Dias {
			if d.Fecha.String() == fecha {
				return d
			}
		}
		t.Fatalf("falta %s", fecha)
		return domain.Clasificacion{}
	}
	if !dia(antes, "2026-09-14").FestivoOficial || dia(antes, "2026-09-15").FestivoOficial ||
		dia(despues, "2026-09-14").FestivoOficial || !dia(despues, "2026-09-15").FestivoOficial {
		t.Fatal("la corrección no respeta el instante de conocimiento")
	}
	if dia(despues, "2026-09-15").Motivos[0].VersionID != "granada-2026-v2" {
		t.Fatal("el motivo debe citar la versión que lo publica")
	}
}

func TestPlazoConsultaCalendariosYFallaCerrado(t *testing.T) {
	s := servicio(t, historia(t))
	r, err := s.CalcularPlazo(context.Background(), ports.SolicitudCalculoPlazo{
		NotificadoEn: time.Date(2026, 9, 3, 22, 30, 0, 0, time.UTC), // 4 sep en Madrid
		Unidad:       domain.UnidadDiasHabiles, Cantidad: 7, MunicipioSede: "municipio:ine:18087",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 7 hábiles desde el 7 sep saltando el 12-13 y la fiesta local del 15.
	if r.Inicio.String() != "2026-09-04" || r.Vencimiento.String() != "2026-09-16" || len(r.VersionesUtilizadas) != 3 {
		t.Fatalf("plazo: %+v", r)
	}
	_, err = s.CalcularPlazo(context.Background(), ports.SolicitudCalculoPlazo{
		Inicio: f(t, "2026-12-20"), Unidad: domain.UnidadDiasHabiles, Cantidad: 10, MunicipioSede: "municipio:ine:18087",
	})
	var cobertura *domain.ErrorCobertura
	if !errors.As(err, &cobertura) || cobertura.Anio != 2027 || len(cobertura.Faltan) != 1 {
		t.Fatalf("sin 2027 debe fallar cerrado: %v", err)
	}
	_, err = s.CalcularPlazo(context.Background(), ports.SolicitudCalculoPlazo{
		Inicio: f(t, "2026-03-02"), Unidad: domain.UnidadDiasHabiles, Cantidad: 2, MunicipioSede: "municipio:ine:18087", MunicipioResidencia: "municipio:ine:18098",
	})
	if !errors.As(err, &cobertura) || cobertura.Faltan[0].Ref != "municipio:ine:18098" {
		t.Fatalf("una residencia sin calendario no se supone hábil: %v", err)
	}
}

func TestSolicitudesInvalidasYErroresOpacos(t *testing.T) {
	s := servicio(t, historia(t))
	ctx := context.Background()
	for _, sol := range []ports.SolicitudCalculoPlazo{
		{Unidad: domain.UnidadMeses, Cantidad: 1, MunicipioSede: "municipio:ine:18087"},
		{Inicio: f(t, "2026-01-02"), NotificadoEn: ahora, Unidad: domain.UnidadMeses, Cantidad: 1, MunicipioSede: "municipio:ine:18087"},
		{Inicio: f(t, "2026-01-02"), Unidad: domain.UnidadMeses, Cantidad: 1, MunicipioSede: "Granada"},
		{Inicio: f(t, "2026-01-02"), Unidad: domain.UnidadMeses, Cantidad: 99, MunicipioSede: "municipio:ine:18087"},
		{Inicio: f(t, "2026-01-02"), Unidad: domain.UnidadMeses, Cantidad: 1, MunicipioSede: "municipio:ine:18087", ConocidoEn: ahora.Add(time.Hour)},
	} {
		if _, err := s.CalcularPlazo(ctx, sol); !errors.Is(err, ErrSolicitudInvalida) {
			t.Fatalf("%+v: %v", sol, err)
		}
	}
	if _, err := s.CalendarioCentro(ctx, ports.SolicitudCalendarioCentro{CentroRef: "centro-999", Anio: 2026}); !errors.Is(err, domain.ErrCalendarioNoCubre) {
		t.Fatalf("centro sin calendario: %v", err)
	}
	fallo := servicio(t, &historiaEnMemoria{fallo: errors.New("detalle interno de conexión")})
	if _, err := fallo.CalendarioCentro(ctx, ports.SolicitudCalendarioCentro{CentroRef: "centro-530", Anio: 2026}); !errors.Is(err, ErrNoDisponible) || err.Error() != ErrNoDisponible.Error() {
		t.Fatalf("el error no es opaco: %v", err)
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := s.Centros(cancelado, 2026, time.Time{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación: %v", err)
	}
	centros, err := s.Centros(ctx, 2026, time.Time{})
	if err != nil || len(centros) != 1 || centros[0].CentroRef != "centro-530" {
		t.Fatalf("centros: %+v %v", centros, err)
	}
	if _, err := NuevoServicio(nil, relojFijo(ahora)); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("sin repositorio no se construye")
	}
}
