package calculoexperienciaoficial

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func referenciaPrueba(t *testing.T, nombre string, version uint64) reglas.ReferenciaVersionada {
	t.Helper()
	suma := sha256.Sum256([]byte(nombre + "#" + strconv.FormatUint(version, 10)))
	return debePrueba(reglas.NuevaReferenciaVersionada(
		nombre, version, hex.EncodeToString(suma[:]),
	))
}

func conjuntoPrueba(t *testing.T) reglas.ConjuntoReglasBaremo {
	t.Helper()
	cero := debePrueba(baremacion.PuntosDesdeMicropuntos(0))
	maximo := debePrueba(baremacion.PuntosDesdeMicropuntos(100_000_000))
	seccion := debePrueba(reglas.NuevaSeccionBaremo(
		"experiencia", referenciaPrueba(t, "definicion:seccion:oficial", 1),
		1, cero, maximo,
	))
	coincidencia := debePrueba(reglas.NuevaPoliticaCoincidenciaReglas(
		reglas.CoincidenciaReglasElegirPrioridad,
	))
	solape := debePrueba(reglas.NuevaPoliticaSolape(reglas.SolapeRechazar))
	grupo := debePrueba(reglas.NuevoGrupoConcurrenciaExperiencia(
		"grupo_experiencia", referenciaPrueba(t, "definicion:grupo:oficial", 1),
		1, coincidencia, solape, nil,
	))
	criterio := debePrueba(reglas.NuevoCriterioExperiencia(
		"ambito", referenciaPrueba(t, "catalogo:ambito:oficial", 1),
		[]string{"diputacion_granada"},
	))
	unidad := debePrueba(reglas.NuevaPoliticaUnidadTemporal(
		reglas.UnidadTemporalDia, reglas.UnidadTemporalDia,
		debePrueba(baremacion.NuevoRacional(1, 1)), reglas.ExtremoFinalInclusivo,
	))
	jornada := debePrueba(reglas.NuevaPoliticaJornada(reglas.JornadaProporcional))
	restos := debePrueba(reglas.NuevaPoliticaRestos(reglas.RestosConservarExactos))
	redondeo := debePrueba(reglas.NuevaPoliticaRedondeo(
		reglas.RedondearPorRegla, baremacion.RedondeoExacto,
	))
	regla := debePrueba(reglas.NuevaReglaExperiencia(
		"experiencia_local", referenciaPrueba(t, "definicion:regla:oficial", 1),
		"experiencia", 1, []reglas.CriterioExperiencia{criterio}, "grupo_experiencia", 1,
		unidad, jornada, restos, redondeo,
		debePrueba(baremacion.PuntosDesdeMicropuntos(1)),
		reglas.SinLimiteUnidades(), reglas.SinLimitePuntos(),
	))
	identidad := debePrueba(reglas.NuevaIdentidadConjuntoReglasBaremo(
		tokenPrueba("reglas:", "reglas-oficial-v2"), 1,
		tokenPrueba("convocatoria:", "convocatoria-oficial-v2"), "expediente:oficial:v1",
	))
	return debePrueba(reglas.NuevoConjuntoReglasBaremo(
		identidad, referenciaPrueba(t, "bases:oficial:v1", 1),
		debePrueba(baremacion.NuevaFechaCivil(2026, 12, 31)),
		[]reglas.SeccionBaremo{seccion}, []reglas.GrupoConcurrenciaExperiencia{grupo},
		[]reglas.ReglaExperiencia{regla},
	))
}

func versionActivaPrueba(
	t *testing.T,
	ahora time.Time,
) (reglas.VersionGobernadaReglasBaremo, reglas.ReferenciaVersionada) {
	t.Helper()
	conjunto := conjuntoPrueba(t)
	motivo := func(clave string) reglas.MotivoCatalogadoReglasBaremo {
		return debePrueba(reglas.NuevoMotivoCatalogadoReglasBaremo(
			referenciaPrueba(t, "catalogo:motivos:gobierno", 1), clave,
		))
	}
	borrador := debePrueba(reglas.NuevaVersionGobernadaReglasBaremo(
		conjunto, "principal:rrhh", motivo("creacion"), ahora.Add(-10*time.Minute),
	))
	vinculoBorrador := debePrueba(borrador.VinculoEstado())
	aprobacion := debePrueba(reglas.NuevaAtestacionAprobacionFirmadaReglasBaremo(
		reglas.DatosAtestacionAprobacionFirmadaReglasBaremo{
			Atestacion:    referenciaPrueba(t, "atestacion:aprobacion:oficial", 1),
			Vinculo:       vinculoBorrador,
			Firma:         referenciaPrueba(t, "firma:aprobacion:oficial", 1),
			PoliticaFirma: referenciaPrueba(t, "politica:firma:oficial", 1),
			Firmantes:     []string{"principal:firmante:uno"},
			FirmadaEn:     ahora.Add(-9 * time.Minute),
			VerificadaEn:  ahora.Add(-8 * time.Minute), ValidaHasta: ahora.Add(time.Minute),
		},
	))
	publicada := debePrueba(borrador.Publicar(
		borrador.Revision(), "principal:rrhh", motivo("publicacion"),
		aprobacion, ahora.Add(-7*time.Minute),
	))
	vinculoPublicada := debePrueba(publicada.VinculoEstado())
	dependencias := debePrueba(publicada.DependenciasContenido())
	convocatoria := referenciaPrueba(t, tokenPrueba("convocatoria:", "convocatoria-oficial-v2"), 3)
	atestacion := debePrueba(reglas.NuevaAtestacionDependenciasVigentesReglasBaremo(
		reglas.DatosAtestacionDependenciasVigentesReglasBaremo{
			Atestacion: referenciaPrueba(t, "atestacion:dependencias:oficial", 1),
			Vinculo:    vinculoPublicada, Convocatoria: convocatoria, Bases: conjunto.Bases(),
			Dependencias: dependencias, VerificadorRef: "servicio:verificador:oficial",
			VerificadaEn: ahora.Add(-6 * time.Minute), ValidaHasta: ahora.Add(time.Minute),
		},
	))
	activa := debePrueba(publicada.Activar(
		publicada.Revision(), "principal:rrhh", motivo("activacion"),
		atestacion, ahora.Add(-5*time.Minute),
	))
	return activa, convocatoria
}

func entradaPrueba(t *testing.T, bloqueada bool) calculo.EntradaExperiencia {
	t.Helper()
	instantanea := referenciaPrueba(t, tokenPrueba("iex_", "instantanea-oficial"), 1)
	if !bloqueada {
		return debePrueba(calculo.NuevaEntradaExperiencia(instantanea, nil))
	}
	atributo := debePrueba(calculo.NuevoAtributoCatalogado(
		"ambito", referenciaPrueba(t, "catalogo:ambito:incompatible", 1),
		"diputacion_granada",
	))
	periodo := debePrueba(calculo.NuevoPeriodoServicioCerrado(
		debePrueba(baremacion.NuevaFechaCivil(2026, 1, 1)),
		debePrueba(baremacion.NuevaFechaCivil(2026, 1, 2)),
	))
	tramo := debePrueba(calculo.NuevoTramoExperiencia(
		referenciaPrueba(t, tokenPrueba("trm_", "tramo-oficial"), 1),
		tokenPrueba("srv_", "servicio-oficial"), periodo, baremacion.JornadaCompleta(),
		calculo.SinComputoIntegroAtestado(), []calculo.AtributoCatalogado{atributo},
	))
	return debePrueba(calculo.NuevaEntradaExperiencia(
		instantanea, []calculo.TramoExperiencia{tramo},
	))
}

func tokenPrueba(prefijo, etiqueta string) string {
	suma := sha256.Sum256([]byte(etiqueta))
	return prefijo + hex.EncodeToString(suma[:])
}

func debePrueba[T any](valor T, err error) T {
	if err != nil {
		panic(err)
	}
	return valor
}
