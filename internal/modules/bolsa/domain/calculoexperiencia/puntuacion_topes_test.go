package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPuntuacionAplicaTopesDeUnidadesYReglaEnOrden(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	configuracion.maximoUnidades = debePuntuacion(reglasbaremo.NuevoLimiteUnidades(
		debePuntuacion(baremacion.NuevoRacional(3, 1)),
	))
	configuracion.maximoPuntos = debePuntuacion(reglasbaremo.NuevoLimitePuntos(
		debePuntuacion(baremacion.PuntosDesdeMicropuntos(5)),
	))
	configuracion.maximoSeccion = debePuntuacion(baremacion.PuntosDesdeMicropuntos(5))
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 10})
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, [][2]baremacion.FechaCivil{periodo},
		baremacion.JornadaCompleta(), false,
	)
	_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	regla, seccion := resultado.Reglas()[0], resultado.Secciones()[0]
	total, presente := resultado.Total()
	if regla.UnidadesAgregadas() != "10/1" || regla.TopeUnidades().Despues() != "3/1" ||
		!regla.TopeUnidades().Aplicado() || regla.BrutoExacto() != "6/1" ||
		regla.TopePuntos().Despues() != "5/1" || !regla.TopePuntos().Aplicado() ||
		regla.PuntosFinalesExactos() != "5/1" || seccion.AntesTopeExacto() != "5/1" ||
		seccion.PuntosFinales().Micropuntos() != 5 || !presente || total.Micropuntos() != 5 {
		t.Fatalf("orden de topes incorrecto: regla=%#v seccion=%#v", regla, seccion)
	}
}

func TestPuntuacionMantieneReglaEnormeHastaQueLaRescataLaSeccion(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	configuracion.puntosPorUnidad = debePuntuacion(baremacion.PuntosDesdeMicropuntos(
		baremacion.MaximoMicropuntos,
	))
	configuracion.maximoSeccion = debePuntuacion(baremacion.PuntosDesdeMicropuntos(1_000_000))
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 2})
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, [][2]baremacion.FechaCivil{periodo},
		baremacion.JornadaCompleta(), false,
	)
	_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	total, presente := resultado.Total()
	const enorme = "18000000000000000/1"
	if resultado.Reglas()[0].PuntosFinalesExactos() != enorme ||
		resultado.Secciones()[0].AntesTopeExacto() != enorme ||
		!presente || total.Micropuntos() != 1_000_000 {
		t.Fatalf("se perdio exactitud antes del tope de seccion: %#v", resultado)
	}
}
