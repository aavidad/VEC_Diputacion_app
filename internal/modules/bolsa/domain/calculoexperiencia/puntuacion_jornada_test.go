package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPuntuacionAplicaLasCuatroPoliticasDeJornada(t *testing.T) {
	completa := baremacion.JornadaCompleta()
	media := debePuntuacion(baremacion.NuevaFraccionJornada(1, 2))
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
	casos := []struct {
		nombre          string
		jornada         baremacion.FraccionJornada
		politica        reglasbaremo.PoliticaJornada
		atestada        bool
		factor          string
		razon           CodigoRazonResultadoExperienciaV1
		puntos          string
		atestacionUsada bool
	}{
		{
			nombre: "completa_proporcional", jornada: completa,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProporcional)),
			factor:   "1/1", razon: RazonJornadaProporcional, puntos: "2/1",
		},
		{
			nombre: "media_proporcional", jornada: media,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProporcional)),
			factor:   "1/2", razon: RazonJornadaProporcional, puntos: "1/1",
		},
		{
			nombre: "media_integra", jornada: media,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaIntegra)),
			factor:   "1/1", razon: RazonJornadaIntegra, puntos: "2/1",
		},
		{
			nombre: "umbral_alcanzado", jornada: media,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornadaDesdeUmbral(media)),
			factor:   "1/1", razon: RazonUmbralAlcanzado, puntos: "2/1",
		},
		{
			nombre: "umbral_no_alcanzado", jornada: media,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornadaDesdeUmbral(
				debePuntuacion(baremacion.NuevaFraccionJornada(3, 4)),
			)),
			factor: "1/2", razon: RazonUmbralNoAlcanzado, puntos: "1/1",
		},
		{
			nombre: "proteccion_no_atestada", jornada: media,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProtegidaIntegra)),
			factor:   "1/2", razon: RazonProteccionNoAtestada, puntos: "1/1",
		},
		{
			nombre: "proteccion_atestada", jornada: media,
			politica: debePuntuacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProtegidaIntegra)),
			atestada: true, factor: "1/1", razon: RazonProteccionAtestada,
			puntos: "2/1", atestacionUsada: true,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			configuracion := configuracionPuntuacionPrueba(t)
			configuracion.jornada = caso.politica
			escenario := construirEscenarioPuntuacionPrueba(
				t, configuracion, [][2]baremacion.FechaCivil{periodo}, caso.jornada, caso.atestada,
			)
			_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
			aplicacion := resultado.Aplicaciones()[0]
			if aplicacion.Jornada().FactorExacto() != caso.factor ||
				aplicacion.Jornada().Razon() != caso.razon ||
				aplicacion.Jornada().AtestacionUsada() != caso.atestacionUsada ||
				resultado.Reglas()[0].PuntosFinalesExactos() != caso.puntos {
				t.Fatalf("jornada o puntos distintos: %#v / %s",
					aplicacion.Jornada(), resultado.Reglas()[0].PuntosFinalesExactos())
			}
		})
	}
}
