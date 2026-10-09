package bootstrap

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	calendariosapp "vec-diputacion-granada/internal/modules/calendarios/application"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// calendariosContadosPrueba sirve el calendario sintético de 2026 (nacional,
// Andalucía y Granada) y cuenta las lecturas: en PostgreSQL cada lectura es
// una consulta.
type calendariosContadosPrueba struct {
	versiones []calendariosdomain.VersionConDias
	lecturas  atomic.Int64
}

func (c *calendariosContadosPrueba) VersionesVigentes(_ context.Context, consulta calendariosports.ConsultaVersiones) ([]calendariosdomain.VersionConDias, error) {
	c.lecturas.Add(1)
	var r []calendariosdomain.VersionConDias
	for _, ambito := range consulta.Ambitos {
		for _, v := range c.versiones {
			if v.Version.Ambito == ambito && v.Version.Anio == consulta.Anio {
				r = append(r, v)
			}
		}
	}
	return r, nil
}

func (c *calendariosContadosPrueba) CentrosConCalendario(context.Context, int, time.Time) ([]calendariosdomain.VersionCalendario, error) {
	return nil, nil
}

func nuevosCalendariosContadosPrueba(t *testing.T) (calendariosports.ConsultaCalendarios, *calendariosContadosPrueba) {
	t.Helper()
	publicada, err := calendariosdomain.ParsearFechaCivil("2025-10-01")
	if err != nil {
		t.Fatal(err)
	}
	version := func(id string, ambito calendariosdomain.Ambito, fechas ...string) calendariosdomain.VersionConDias {
		v := calendariosdomain.VersionConDias{Version: calendariosdomain.VersionCalendario{
			ID: id, Ambito: ambito, Anio: 2026, Numero: 1, Denominacion: "Calendario " + id,
			Procedencia:   calendariosdomain.Procedencia{Norma: "Datos sintéticos", Referencia: "prueba", PublicadaEn: publicada, Sintetica: true},
			ConocidoDesde: time.Date(2025, 10, 1, 8, 0, 0, 0, time.UTC),
		}}
		if ambito.Tipo == calendariosdomain.AmbitoLocal {
			v.Version.ComunidadRef = "es-an"
		}
		for _, fecha := range fechas {
			dia, err := calendariosdomain.ParsearFechaCivil(fecha)
			if err != nil {
				t.Fatal(err)
			}
			v.Dias = append(v.Dias, calendariosdomain.DiaSenalado{Fecha: dia, Efecto: calendariosdomain.EfectoFestivo, Denominacion: "Día " + fecha})
		}
		return v
	}
	repositorio := &calendariosContadosPrueba{versiones: []calendariosdomain.VersionConDias{
		version("nacional-2026-v1", calendariosdomain.Ambito{Tipo: calendariosdomain.AmbitoNacional, Ref: calendariosdomain.ReferenciaNacional}, "2026-01-01", "2026-01-06", "2026-04-03", "2026-05-01", "2026-08-15", "2026-10-12", "2026-12-08", "2026-12-25"),
		version("andalucia-2026-v1", calendariosdomain.Ambito{Tipo: calendariosdomain.AmbitoAutonomico, Ref: "es-an"}, "2026-02-28", "2026-04-02", "2026-11-02", "2026-12-07"),
		version("granada-2026-v1", calendariosdomain.Ambito{Tipo: calendariosdomain.AmbitoLocal, Ref: reglas.MunicipioSedeDiputacion}, "2026-05-11", "2026-09-14"),
	}}
	servicio, err := calendariosapp.NuevoServicio(repositorio, relojCalendariosDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	return servicio, repositorio
}

// El cuadro calcula el plazo de cada grupo (fase y fecha de entrada). Antes
// cada grupo leía el calendario (dos consultas por grupo); ahora la consulta
// del cuadro lo lee una vez para todos los grupos, con los mismos plazos.
func TestPlazosCuadroCTLeenElCalendarioUnaVezPorConsulta(t *testing.T) {
	const grupos = 50
	nuevaCalculadora := func() (ports.CalculadoraPlazoFaseRRHH, *calendariosContadosPrueba) {
		calendarios, repositorio := nuevosCalendariosContadosPrueba(t)
		resolutor, err := nuevoResolutorReglasEjemplo(rutaReglasCTEjemploPrueba, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal,
			calculadoraPlazosCalendarios{consulta: calendarios}, relojPresentacionReglasEjemplo)
		if err != nil {
			t.Fatal(err)
		}
		return nuevaCalculadoraPlazoFaseCT(resolutor), repositorio
	}
	original, antes := nuevaCalculadora()
	calculadora, despues := nuevaCalculadora()
	cuadro := calculadora.(ports.PreparadorPlazosFaseConsultaRRHH).PrepararPlazosFaseConsulta(t.Context(), true)
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	fases := []domain.ClaveFase{"asignacion_unidad", "informe_juridico", "fiscalizacion", "subsanacion_unidad"}
	calculados := 0
	for i := range grupos {
		solicitud := ports.SolicitudPlazoFaseRRHH{
			Fase: fases[i%len(fases)], Desde: time.Date(2026, 3, 2, 9, 30, 0, 0, time.UTC).AddDate(0, 0, 3*i),
			Ahora: ahora, Urgente: i%5 == 0,
		}
		p1, a1, e1 := original.CalcularPlazoFase(t.Context(), solicitud)
		p2, a2, e2 := cuadro.CalcularPlazoFase(t.Context(), solicitud)
		if e1 != nil || e2 != nil || !a1 || !reflect.DeepEqual(p1, p2) || a1 != a2 {
			t.Fatalf("grupo %d: antes %+v %v %v; ahora %+v %v %v", i, p1, a1, e1, p2, a2, e2)
		}
		calculados++
	}
	if calculados != grupos || antes.lecturas.Load() != 2*grupos {
		t.Fatalf("antes: %d grupos y %d lecturas del calendario", calculados, antes.lecturas.Load())
	}
	// Calendarios locales y oficiales de 2026: dos consultas en total.
	if despues.lecturas.Load() > 2 {
		t.Fatalf("la consulta del cuadro leyó el calendario %d veces", despues.lecturas.Load())
	}
	// Otra consulta del cuadro vuelve a leer: no hay memoria entre peticiones.
	otra := calculadora.(ports.PreparadorPlazosFaseConsultaRRHH).PrepararPlazosFaseConsulta(t.Context(), true)
	if _, _, err := otra.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{Fase: "fiscalizacion", Desde: ahora.AddDate(0, 0, -3), Ahora: ahora}); err != nil {
		t.Fatal(err)
	}
	if despues.lecturas.Load() != 4 {
		t.Fatalf("otra consulta no volvió a leer el calendario: %d", despues.lecturas.Load())
	}
}
