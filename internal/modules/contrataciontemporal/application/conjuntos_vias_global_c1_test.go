package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	analisisGlobalRefC1 = "analisis_durable_cobertura_global_01"
	huellaAnalisisC1    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

type escenarioConjuntosViasC1 struct {
	entorno     *entornoCoberturaAplicacionPrueba
	catalogo    domain.CatalogoViasCobertura
	politica    domain.PoliticaDecisionCobertura
	solicitudes []ports.SolicitudConsultarCobertura
	conjuntos   []cobertura.ConjuntoEvidenciasCobertura
	instante    time.Time
}

func TestPreparacionConjuntosViasC1DeterministaYListaParaPropuesta(
	t *testing.T,
) {
	escenario := nuevoEscenarioConjuntosViasC1(t, "", "")
	directa := prepararGlobalC1(t, escenario, escenario.conjuntos)
	inversa := prepararGlobalC1(
		t,
		escenario,
		[]cobertura.ConjuntoEvidenciasCobertura{
			escenario.conjuntos[1], escenario.conjuntos[0],
		},
	)
	refDirecta, _ := directa.Referencia()
	refInversa, _ := inversa.Referencia()
	huellaDirecta, _ := directa.HuellaSHA256()
	huellaInversa, _ := inversa.HuellaSHA256()
	if refDirecta != refInversa || huellaDirecta != huellaInversa {
		t.Fatal("la preparacion global depende de la permutacion de entrada")
	}
	validaHasta, err := directa.ValidaHasta()
	if err != nil ||
		!validaHasta.Equal(escenario.entorno.inicio.Add(5*time.Second)) {
		t.Fatalf("vigencia minima inesperada: %v, %v", validaHasta, err)
	}
	datos, err := directa.DatosCrearPropuestaEn(escenario.instante)
	if err != nil {
		t.Fatal(err)
	}
	if datos.AnalisisRef != analisisGlobalRefC1 ||
		datos.AnalisisHuellaSHA256 != huellaAnalisisC1 ||
		datos.PreparacionEvidenciasRef != refDirecta ||
		datos.PreparacionEvidenciasHuellaSHA256 != huellaDirecta ||
		len(datos.Resultados) != 2 ||
		datos.Resultados[0].Clave != "comprobacion_secundaria" ||
		datos.Resultados[1].Clave != "comprobacion_primaria" {
		t.Fatalf("proyeccion autoritativa inesperada: %#v", datos)
	}
	if _, err := domain.CrearPropuestaDecisionCobertura(datos); err != nil {
		t.Fatalf("la proyeccion no crea propuesta: %v", err)
	}
	ordenes, err := directa.OrdenesPendientesEn(escenario.instante)
	if err != nil || len(ordenes) != 2 {
		t.Fatalf("ordenes pendientes inesperadas: %d, %v", len(ordenes), err)
	}
	for indice, via := range []domain.ClaveCatalogo{
		"via_secundaria", "via_primaria",
	} {
		resumen, err := ordenes[indice].ResumenPendienteEn(escenario.instante)
		if err != nil || resumen.ViaClave != via {
			t.Fatalf("orden %d fuera de prioridad: %#v, %v", indice, resumen, err)
		}
	}
	comprobarCeroConsumoGlobalC1(t, escenario.entorno)
}

func TestPreparacionConjuntosViasC1RechazaFaltaExtraYDuplicada(
	t *testing.T,
) {
	escenario := nuevoEscenarioConjuntosViasC1(t, "", "")
	casos := []struct {
		nombre    string
		conjuntos []cobertura.ConjuntoEvidenciasCobertura
	}{
		{"falta", escenario.conjuntos[:1]},
		{"duplicada", []cobertura.ConjuntoEvidenciasCobertura{
			escenario.conjuntos[0], escenario.conjuntos[0],
		}},
		{"extra", []cobertura.ConjuntoEvidenciasCobertura{
			escenario.conjuntos[0],
			escenario.conjuntos[1],
			escenario.conjuntos[0],
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := cobertura.PrepararConjuntosViasCobertura(
				datosGlobalesC1(escenario, caso.conjuntos),
			)
			if !errors.Is(
				err,
				ports.ErrResultadoFuenteCoberturaNoConfiable,
			) {
				t.Fatalf("coleccion incompleta aceptada: %v", err)
			}
		})
	}
	comprobarCeroConsumoGlobalC1(t, escenario.entorno)
}

func TestPreparacionConjuntosViasC1RechazaCoordenadasDistintas(
	t *testing.T,
) {
	escenario := nuevoEscenarioConjuntosViasC1(t, "", "")
	solicitud := escenario.solicitudes[1]
	solicitud.VersionExpediente++
	solicitud.PeticionRef = "peticion_via_secundaria_version_distinta"
	evidencia := prepararEvidenciaCoberturaPrueba(
		t,
		escenario.entorno,
		solicitud,
		"recibo_via_secundaria_version_distinta",
		domain.ComprobacionAfirmativa,
	)
	coordenadas := coordenadasCoberturaPrueba(solicitud, escenario.politica)
	distinto := nuevoConjuntoCoberturaPrueba(
		t,
		coordenadas,
		escenario.catalogo,
		escenario.politica,
		[]cobertura.EvidenciaConsultaCobertura{evidencia},
		escenario.instante,
	)
	_, err := cobertura.PrepararConjuntosViasCobertura(
		datosGlobalesC1(
			escenario,
			[]cobertura.ConjuntoEvidenciasCobertura{
				escenario.conjuntos[0], distinto,
			},
		),
	)
	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("version distinta aceptada: %v", err)
	}
}

func TestPreparacionConjuntosViasC1LigaAnalisisCatalogoYPolitica(
	t *testing.T,
) {
	escenario := nuevoEscenarioConjuntosViasC1(t, "", "")
	original := prepararGlobalC1(t, escenario, escenario.conjuntos)
	otraHuella := strings.Repeat("b", 64)
	datosOtroAnalisis := datosGlobalesC1(escenario, escenario.conjuntos)
	datosOtroAnalisis.AnalisisHuellaSHA256 = otraHuella
	otroAnalisis, err := cobertura.PrepararConjuntosViasCobertura(
		datosOtroAnalisis,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaOriginal, _ := original.HuellaSHA256()
	huellaOtra, _ := otroAnalisis.HuellaSHA256()
	if huellaOriginal == huellaOtra {
		t.Fatal("adulterar el analisis no cambia la huella global")
	}
	invalido := datosGlobalesC1(escenario, escenario.conjuntos)
	invalido.AnalisisHuellaSHA256 = "no-es-una-huella"
	exigirPreparacionGlobalC1Rechazada(t, invalido)

	otroCatalogo := catalogoGlobalC1(
		t,
		escenario.entorno.inicio,
		"catalogo_global_alterado",
		2,
	)
	conCatalogoAlterado := datosGlobalesC1(escenario, escenario.conjuntos)
	conCatalogoAlterado.Catalogo = otroCatalogo
	exigirPreparacionGlobalC1Rechazada(t, conCatalogoAlterado)

	otraPolitica := politicaCoberturaPrueba(
		t,
		escenario.catalogo,
		escenario.entorno.inicio,
	)
	publicacion := otraPolitica.Publicacion()
	publicacion.Referencia = "politica_global_alterada"
	otraPolitica, err = domain.RestaurarPoliticaDecisionCobertura(
		publicacion,
		escenario.catalogo,
	)
	if err == nil {
		t.Fatal("el dominio acepto una politica adulterada")
	}
	conPoliticaAjena := datosGlobalesC1(escenario, escenario.conjuntos)
	conPoliticaAjena.Politica = politicaGlobalC1(
		t,
		escenario.catalogo,
		escenario.entorno.inicio,
		"politica_global_otra",
	)
	exigirPreparacionGlobalC1Rechazada(t, conPoliticaAjena)
}

func TestPreparacionConjuntosViasC1RechazaReusoCruzado(
	t *testing.T,
) {
	t.Run("peticion", func(t *testing.T) {
		escenario := nuevoEscenarioConjuntosViasC1(
			t,
			"peticion_reutilizada_entre_vias",
			"",
		)
		exigirPreparacionGlobalC1Rechazada(
			t,
			datosGlobalesC1(escenario, escenario.conjuntos),
		)
	})
	t.Run("prueba", func(t *testing.T) {
		escenario := nuevoEscenarioConjuntosViasC1(
			t,
			"",
			"recibo_reutilizado_entre_vias",
		)
		exigirPreparacionGlobalC1Rechazada(
			t,
			datosGlobalesC1(escenario, escenario.conjuntos),
		)
	})
}

func TestPreparacionConjuntosViasC1CaducidadRollbackYCopias(
	t *testing.T,
) {
	escenario := nuevoEscenarioConjuntosViasC1(t, "", "")
	entrada := datosGlobalesC1(escenario, escenario.conjuntos)
	preparacion, err := cobertura.PrepararConjuntosViasCobertura(entrada)
	if err != nil {
		t.Fatal(err)
	}
	entrada.Conjuntos[0] = cobertura.ConjuntoEvidenciasCobertura{}
	datos, err := preparacion.DatosCrearPropuestaEn(escenario.instante)
	if err != nil {
		t.Fatalf("la mutacion de entrada afecto la capacidad: %v", err)
	}
	datos.Resultados[0] = domain.ComprobacionCobertura{}
	publicacionCatalogo := datos.Catalogo.Publicacion()
	publicacionCatalogo.Vias[0].Clave = "via_mutada"
	relectura, err := preparacion.DatosCrearPropuestaEn(escenario.instante)
	if err != nil || relectura.Resultados[0].Clave == "" ||
		relectura.Catalogo.Identidad() != escenario.catalogo.Identidad() {
		t.Fatalf("salida sin copia defensiva: %#v, %v", relectura, err)
	}
	if _, err := preparacion.DatosCrearPropuestaEn(
		escenario.instante.Add(-time.Microsecond),
	); !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("rollback aceptado: %v", err)
	}
	if _, err := preparacion.OrdenesPendientesEn(
		escenario.entorno.inicio.Add(5 * time.Second),
	); !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("caducidad aceptada: %v", err)
	}
	comprobarCeroConsumoGlobalC1(t, escenario.entorno)
}

func TestPreparacionConjuntosViasC1RedactaFormatos(t *testing.T) {
	escenario := nuevoEscenarioConjuntosViasC1(t, "", "")
	datos := datosGlobalesC1(escenario, escenario.conjuntos)
	preparacion := prepararGlobalC1(t, escenario, escenario.conjuntos)
	secretos := []string{
		analisisGlobalRefC1,
		huellaAnalisisC1,
		escenario.solicitudes[0].ExpedienteRef,
		escenario.solicitudes[0].PeticionRef,
		escenario.catalogo.Identidad().Referencia,
		escenario.politica.Identidad().Referencia,
	}
	comprobarRedaccionCobertura(t, datos, secretos)
	comprobarRedaccionCobertura(t, preparacion, secretos)
	jsonPreparacion, err := json.Marshal(preparacion)
	if err != nil || strings.Contains(
		string(jsonPreparacion),
		escenario.solicitudes[0].ExpedienteRef,
	) {
		t.Fatalf("JSON no redactado: %s, %v", jsonPreparacion, err)
	}
}
