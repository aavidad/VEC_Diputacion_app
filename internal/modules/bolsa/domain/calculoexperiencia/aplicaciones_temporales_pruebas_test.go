package calculoexperiencia

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

type reglaTemporalPrueba struct {
	clave     string
	grupo     string
	extremo   reglasbaremo.TratamientoExtremoFinal
	prioridad uint32
}

type asignacionTemporalPrueba struct {
	tramo TramoExperiencia
	regla string
}

func planTemporalPrueba(
	t *testing.T,
	corte baremacion.FechaCivil,
	reglas []reglaTemporalPrueba,
) PlanExperiencia {
	t.Helper()
	cero := debeTemporal(baremacion.PuntosDesdeMicropuntos(0))
	maximo := debeTemporal(baremacion.PuntosDesdeMicropuntos(100_000_000))
	seccion := debeTemporal(reglasbaremo.NuevaSeccionBaremo(
		"experiencia",
		referenciaTemporalPrueba(t, "definicion:temporal:seccion", 1),
		1,
		cero,
		maximo,
	))

	clavesGrupo := make([]string, 0)
	gruposVistos := make(map[string]struct{})
	for _, regla := range reglas {
		if _, existe := gruposVistos[regla.grupo]; existe {
			continue
		}
		gruposVistos[regla.grupo] = struct{}{}
		clavesGrupo = append(clavesGrupo, regla.grupo)
	}
	grupos := make([]reglasbaremo.GrupoConcurrenciaExperiencia, len(clavesGrupo))
	for indice, clave := range clavesGrupo {
		grupos[indice] = debeTemporal(reglasbaremo.NuevoGrupoConcurrenciaExperiencia(
			clave,
			referenciaTemporalPrueba(t, "definicion:temporal:grupo:"+clave, 1),
			uint32(indice+1),
			debeTemporal(reglasbaremo.NuevaPoliticaCoincidenciaReglas(
				reglasbaremo.CoincidenciaReglasAcumular,
			)),
			debeTemporal(reglasbaremo.NuevaPoliticaSolape(
				reglasbaremo.SolapeRechazar,
			)),
			nil,
		))
	}

	catalogo := referenciaTemporalPrueba(t, "catalogo:temporal:ambito", 1)
	reglasDominio := make([]reglasbaremo.ReglaExperiencia, len(reglas))
	for indice, regla := range reglas {
		criterio := debeTemporal(reglasbaremo.NuevoCriterioExperiencia(
			"ambito",
			catalogo,
			[]string{"admitida"},
		))
		unidad := debeTemporal(reglasbaremo.NuevaPoliticaUnidadTemporal(
			reglasbaremo.UnidadTemporalDia,
			reglasbaremo.UnidadTemporalMes,
			debeTemporal(baremacion.NuevoRacional(30, 1)),
			regla.extremo,
		))
		reglasDominio[indice] = debeTemporal(reglasbaremo.NuevaReglaExperiencia(
			regla.clave,
			referenciaTemporalPrueba(t, "definicion:temporal:regla:"+regla.clave, 1),
			"experiencia",
			uint32(indice+1),
			[]reglasbaremo.CriterioExperiencia{criterio},
			regla.grupo,
			regla.prioridad,
			unidad,
			debeTemporal(reglasbaremo.NuevaPoliticaJornada(
				reglasbaremo.JornadaProporcional,
			)),
			debeTemporal(reglasbaremo.NuevaPoliticaRestos(
				reglasbaremo.RestosConservarExactos,
			)),
			debeTemporal(reglasbaremo.NuevaPoliticaRedondeo(
				reglasbaremo.RedondearPorRegla,
				baremacion.RedondeoMitadAlPar,
			)),
			debeTemporal(baremacion.PuntosDesdeMicropuntos(500_000)),
			reglasbaremo.SinLimiteUnidades(),
			reglasbaremo.SinLimitePuntos(),
		))
	}

	identidad := debeTemporal(reglasbaremo.NuevaIdentidadConjuntoReglasBaremo(
		"rgl_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		1,
		"con_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"exp_cccccccccccccccccccccccccccccccc",
	))
	conjunto := debeTemporal(reglasbaremo.NuevoConjuntoReglasBaremo(
		identidad,
		referenciaTemporalPrueba(t, "bases:temporal:v1", 1),
		corte,
		[]reglasbaremo.SeccionBaremo{seccion},
		grupos,
		reglasDominio,
	))
	return debeTemporal(Compilar(conjunto))
}

func tramoTemporalPrueba(
	t *testing.T,
	etiqueta string,
	servicio string,
	periodo PeriodoServicio,
) TramoExperiencia {
	t.Helper()
	return debeTemporal(NuevoTramoExperiencia(
		referenciaTokenTemporalPrueba(t, prefijoTramoEntrada, "tramo-"+etiqueta, 1),
		tokenTemporalPrueba(prefijoServicioEntrada, "servicio-"+servicio),
		periodo,
		debeTemporal(baremacion.NuevaFraccionJornada(1, 1)),
		SinComputoIntegroAtestado(),
		nil,
	))
}

func entradaTemporalPrueba(t *testing.T, tramos ...TramoExperiencia) EntradaExperiencia {
	t.Helper()
	return debeTemporal(NuevaEntradaExperiencia(
		referenciaTokenTemporalPrueba(t, prefijoInstantaneaEntrada, "instantanea-temporal", 1),
		tramos,
	))
}

func seleccionTemporalPrueba(
	t *testing.T,
	plan PlanExperiencia,
	asignaciones ...asignacionTemporalPrueba,
) seleccionExperiencia {
	t.Helper()
	porClave := make(map[string]reglasbaremo.ReglaExperiencia)
	for _, regla := range plan.Reglas() {
		porClave[regla.Clave()] = regla
	}
	seleccion := seleccionExperiencia{}
	for _, asignacion := range asignaciones {
		regla, existe := porClave[asignacion.regla]
		if !existe {
			t.Fatalf("regla temporal de prueba ausente: %s", asignacion.regla)
		}
		seleccion.aplicaciones = append(seleccion.aplicaciones, aplicacionSeleccion{
			tramo:        asignacion.tramo.Referencia(),
			reglaClave:   regla.Clave(),
			grupoClave:   regla.GrupoConcurrenciaClave(),
			seccionClave: regla.SeccionClave(),
			prioridad:    regla.PrioridadConcurrencia(),
			razon:        motivoAplicacionUnica,
		})
	}
	return seleccion
}

func periodoCerradoTemporalPrueba(
	t *testing.T,
	desde baremacion.FechaCivil,
	hasta baremacion.FechaCivil,
) PeriodoServicio {
	t.Helper()
	return debeTemporal(NuevoPeriodoServicioCerrado(desde, hasta))
}

func fechaTemporalPrueba(t *testing.T, anio, mes, dia int) baremacion.FechaCivil {
	t.Helper()
	return debeTemporal(baremacion.NuevaFechaCivil(anio, mes, dia))
}

func referenciaTemporalPrueba(
	t *testing.T,
	referencia string,
	version uint64,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	suma := sha256.Sum256([]byte(referencia + "#" + strconv.FormatUint(version, 10)))
	return debeTemporal(reglasbaremo.NuevaReferenciaVersionada(
		referencia,
		version,
		hex.EncodeToString(suma[:]),
	))
}

func referenciaTokenTemporalPrueba(
	t *testing.T,
	prefijo string,
	etiqueta string,
	version uint64,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	return referenciaTemporalPrueba(t, tokenTemporalPrueba(prefijo, etiqueta), version)
}

func tokenTemporalPrueba(prefijo, etiqueta string) string {
	suma := sha256.Sum256([]byte(etiqueta))
	return prefijo + hex.EncodeToString(suma[:])
}

func debeTemporal[T any](valor T, err error) T {
	if err != nil {
		panic(err)
	}
	return valor
}
