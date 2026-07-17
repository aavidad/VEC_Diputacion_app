package calculoexperiencia

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

type criterioSeleccionPrueba struct {
	clave    string
	catalogo reglasbaremo.ReferenciaVersionada
	valores  []string
}

type reglaSeleccionPrueba struct {
	clave     string
	orden     uint32
	grupo     string
	prioridad uint32
	criterios []criterioSeleccionPrueba
}

type grupoSeleccionPrueba struct {
	clave string
	orden uint32
	modo  reglasbaremo.ModoCoincidenciaReglas
}

func planSeleccionPrueba(
	t *testing.T,
	grupos []grupoSeleccionPrueba,
	reglas []reglaSeleccionPrueba,
) PlanExperiencia {
	t.Helper()
	cero := debeSeleccion(baremacion.PuntosDesdeMicropuntos(0))
	maximo := debeSeleccion(baremacion.PuntosDesdeMicropuntos(100_000_000))
	seccion := debeSeleccion(reglasbaremo.NuevaSeccionBaremo(
		"experiencia",
		referenciaSeleccionPrueba(t, "definicion:seleccion:seccion", 1),
		1,
		cero,
		maximo,
	))

	politicaSolape := debeSeleccion(reglasbaremo.NuevaPoliticaSolape(
		reglasbaremo.SolapeRechazar,
	))
	gruposDominio := make([]reglasbaremo.GrupoConcurrenciaExperiencia, len(grupos))
	for indice, grupo := range grupos {
		coincidencia := debeSeleccion(reglasbaremo.NuevaPoliticaCoincidenciaReglas(grupo.modo))
		gruposDominio[indice] = debeSeleccion(reglasbaremo.NuevoGrupoConcurrenciaExperiencia(
			grupo.clave,
			referenciaSeleccionPrueba(t, "definicion:seleccion:grupo:"+grupo.clave, 1),
			grupo.orden,
			coincidencia,
			politicaSolape,
			nil,
		))
	}

	unidad := debeSeleccion(reglasbaremo.NuevaPoliticaUnidadTemporal(
		reglasbaremo.UnidadTemporalDia,
		reglasbaremo.UnidadTemporalMes,
		debeSeleccion(baremacion.NuevoRacional(30, 1)),
		reglasbaremo.ExtremoFinalInclusivo,
	))
	jornada := debeSeleccion(reglasbaremo.NuevaPoliticaJornada(
		reglasbaremo.JornadaProporcional,
	))
	restos := debeSeleccion(reglasbaremo.NuevaPoliticaRestos(
		reglasbaremo.RestosConservarExactos,
	))
	redondeo := debeSeleccion(reglasbaremo.NuevaPoliticaRedondeo(
		reglasbaremo.RedondearPorRegla,
		baremacion.RedondeoMitadAlPar,
	))
	coeficiente := debeSeleccion(baremacion.PuntosDesdeMicropuntos(500_000))

	reglasDominio := make([]reglasbaremo.ReglaExperiencia, len(reglas))
	for indice, regla := range reglas {
		criterios := make([]reglasbaremo.CriterioExperiencia, len(regla.criterios))
		for posicion, criterio := range regla.criterios {
			criterios[posicion] = debeSeleccion(reglasbaremo.NuevoCriterioExperiencia(
				criterio.clave,
				criterio.catalogo,
				criterio.valores,
			))
		}
		reglasDominio[indice] = debeSeleccion(reglasbaremo.NuevaReglaExperiencia(
			regla.clave,
			referenciaSeleccionPrueba(t, "definicion:seleccion:regla:"+regla.clave, 1),
			"experiencia",
			regla.orden,
			criterios,
			regla.grupo,
			regla.prioridad,
			unidad,
			jornada,
			restos,
			redondeo,
			coeficiente,
			reglasbaremo.SinLimiteUnidades(),
			reglasbaremo.SinLimitePuntos(),
		))
	}

	identidad := debeSeleccion(reglasbaremo.NuevaIdentidadConjuntoReglasBaremo(
		"reglas:seleccion:v1",
		1,
		"convocatoria:seleccion:v1",
		"expediente:seleccion:v1",
	))
	fecha := debeSeleccion(baremacion.NuevaFechaCivil(2026, 7, 17))
	conjunto := debeSeleccion(reglasbaremo.NuevoConjuntoReglasBaremo(
		identidad,
		referenciaSeleccionPrueba(t, "bases:seleccion:v1", 1),
		fecha,
		[]reglasbaremo.SeccionBaremo{seccion},
		gruposDominio,
		reglasDominio,
	))
	return debeSeleccion(Compilar(conjunto))
}

func entradaSeleccionPrueba(t *testing.T, tramos []TramoExperiencia) EntradaExperiencia {
	t.Helper()
	return debeSeleccion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "instantanea-seleccion", 1),
		tramos,
	))
}

func tramoSeleccionPrueba(
	t *testing.T,
	etiqueta string,
	atributos []AtributoCatalogado,
) TramoExperiencia {
	t.Helper()
	periodo := debeSeleccion(NuevoPeriodoServicioCerrado(
		debeSeleccion(baremacion.NuevaFechaCivil(2025, 1, 1)),
		debeSeleccion(baremacion.NuevaFechaCivil(2025, 12, 31)),
	))
	jornada := debeSeleccion(baremacion.NuevaFraccionJornada(1, 1))
	return debeSeleccion(NuevoTramoExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoTramoEntrada, "tramo-"+etiqueta, 1),
		tokenSeleccionPrueba(prefijoServicioEntrada, "servicio-"+etiqueta),
		periodo,
		jornada,
		SinComputoIntegroAtestado(),
		atributos,
	))
}

func atributoSeleccionPrueba(
	t *testing.T,
	clave string,
	catalogo reglasbaremo.ReferenciaVersionada,
	valor string,
) AtributoCatalogado {
	t.Helper()
	return debeSeleccion(NuevoAtributoCatalogado(clave, catalogo, valor))
}

func referenciaSeleccionPrueba(
	t *testing.T,
	referencia string,
	version uint64,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	suma := sha256.Sum256([]byte(referencia + "#" + strconv.FormatUint(version, 10)))
	return debeSeleccion(reglasbaremo.NuevaReferenciaVersionada(
		referencia,
		version,
		hex.EncodeToString(suma[:]),
	))
}

func referenciaSeleccionConHuellaPrueba(
	t *testing.T,
	referencia string,
	version uint64,
	digito byte,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	return debeSeleccion(reglasbaremo.NuevaReferenciaVersionada(
		referencia,
		version,
		strings.Repeat(string(digito), 64),
	))
}

func referenciaTokenSeleccionPrueba(
	t *testing.T,
	prefijo string,
	etiqueta string,
	version uint64,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	return referenciaSeleccionPrueba(t, tokenSeleccionPrueba(prefijo, etiqueta), version)
}

func tokenSeleccionPrueba(prefijo string, etiqueta string) string {
	suma := sha256.Sum256([]byte(etiqueta))
	return prefijo + hex.EncodeToString(suma[:])
}

func debeSeleccion[T any](valor T, err error) T {
	if err != nil {
		panic(err)
	}
	return valor
}
