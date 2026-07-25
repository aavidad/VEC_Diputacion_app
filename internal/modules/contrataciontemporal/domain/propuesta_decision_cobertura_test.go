package domain

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPropuestaDecisionCoberturaClasificaResultadosSinConfundirAusencia(
	t *testing.T,
) {
	casos := []struct {
		nombre     string
		resultados func() []ComprobacionCobertura
		estado     EstadoPropuestaDecisionCobertura
		via        ClaveCatalogo
	}{
		{
			nombre: "viable_con_ausencia_admitida",
			resultados: func() []ComprobacionCobertura {
				return []ComprobacionCobertura{
					resultadoDecisionCoberturaPrueba(
						"hecho_compartido",
						ComprobacionAfirmativa,
						"01",
					),
					resultadoDecisionCoberturaPrueba(
						"hecho_alternativo",
						ComprobacionAfirmativa,
						"02",
					),
					resultadoDecisionCoberturaPrueba(
						"hecho_futuro",
						ComprobacionNoConsta,
						"02b",
					),
				}
			},
			estado: PropuestaCoberturaViable,
			via:    "via_futura_configurable",
		},
		{
			nombre: "incompleta_por_ausencia_bloqueante",
			resultados: func() []ComprobacionCobertura {
				return []ComprobacionCobertura{
					resultadoDecisionCoberturaPrueba(
						"hecho_alternativo",
						ComprobacionAfirmativa,
						"03",
					),
				}
			},
			estado: PropuestaCoberturaIncompleta,
		},
		{
			nombre: "conflictiva",
			resultados: func() []ComprobacionCobertura {
				return []ComprobacionCobertura{
					resultadoDecisionCoberturaPrueba(
						"hecho_compartido",
						ComprobacionAfirmativa,
						"04",
					),
					resultadoDecisionCoberturaPrueba(
						"hecho_compartido",
						ComprobacionNegativa,
						"05",
					),
				}
			},
			estado: PropuestaCoberturaConflictiva,
		},
		{
			nombre: "sin_via",
			resultados: func() []ComprobacionCobertura {
				return []ComprobacionCobertura{
					resultadoDecisionCoberturaPrueba(
						"hecho_compartido",
						ComprobacionNegativa,
						"06",
					),
					resultadoDecisionCoberturaPrueba(
						"hecho_alternativo",
						ComprobacionAfirmativa,
						"07",
					),
					resultadoDecisionCoberturaPrueba(
						"hecho_futuro",
						ComprobacionNoConsta,
						"07b",
					),
				}
			},
			estado: PropuestaCoberturaSinVia,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := datosPropuestaDecisionCoberturaPrueba(t)
			datos.Resultados = caso.resultados()
			propuesta, err := CrearPropuestaDecisionCobertura(datos)
			if err != nil {
				t.Fatalf("crear propuesta: %v", err)
			}
			if propuesta.Estado() != caso.estado ||
				propuesta.ViaPropuesta() != caso.via {
				t.Fatalf(
					"clasificación inesperada: estado=%s vía=%s",
					propuesta.Estado(),
					propuesta.ViaPropuesta(),
				)
			}
			if propuesta.ValidarPara(datos.Catalogo, datos.Politica) != nil {
				t.Fatal("la propuesta creada no se puede restaurar")
			}
		})
	}
}

func TestPropuestaDecisionCoberturaFallaCerradaAnteResultadosDesconocidos(
	t *testing.T,
) {
	casos := []struct {
		nombre  string
		alterar func(*DatosCrearPropuestaDecisionCobertura)
	}{
		{
			nombre: "resultado_desconocido",
			alterar: func(d *DatosCrearPropuestaDecisionCobertura) {
				d.Resultados[0].Resultado = "indeterminado"
			},
		},
		{
			nombre: "clave_no_publicada",
			alterar: func(d *DatosCrearPropuestaDecisionCobertura) {
				d.Resultados[0].Clave = "comprobacion_no_publicada"
			},
		},
		{
			nombre: "detalle_libre",
			alterar: func(d *DatosCrearPropuestaDecisionCobertura) {
				d.Resultados[0].Detalle =
					"Nombre y documento que no deben propagarse"
			},
		},
		{
			nombre: "evaluacion_futura",
			alterar: func(d *DatosCrearPropuestaDecisionCobertura) {
				d.Resultados[0].EvaluadaEn =
					d.GeneradaEn.Add(time.Microsecond)
			},
		},
		{
			nombre: "version_expediente_fuera_de_rango",
			alterar: func(d *DatosCrearPropuestaDecisionCobertura) {
				d.VersionExpediente = 1 << 53
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := datosPropuestaDecisionCoberturaPrueba(t)
			caso.alterar(&datos)
			if _, err := CrearPropuestaDecisionCobertura(datos); err == nil {
				t.Fatal("una entrada no confiable produjo propuesta")
			}
		})
	}
}

func TestPropuestaDecisionCoberturaRechazaCatalogoMutado(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	publicacion := datos.Catalogo.Publicacion()
	mutado, err := PublicarCatalogoViasCobertura(
		BorradorCatalogoViasCobertura{
			Referencia:     publicacion.Referencia,
			Version:        publicacion.Version + 1,
			PublicadoEn:    publicacion.PublicadoEn,
			Vigencia:       publicacion.Vigencia,
			ProcedenciaRef: publicacion.ProcedenciaRef,
			Vias:           publicacion.Vias,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	datos.Catalogo = mutado
	if _, err := CrearPropuestaDecisionCobertura(datos); err == nil {
		t.Fatal("la política se aplicó a otra identidad de catálogo")
	}
}

func TestPropuestaDecisionCoberturaEsDeterministaYDireccionadaPorContenido(
	t *testing.T,
) {
	primeraEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	segundaEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	segundaEntrada.Resultados = []ComprobacionCobertura{
		primeraEntrada.Resultados[2],
		primeraEntrada.Resultados[0],
		primeraEntrada.Resultados[1],
		primeraEntrada.Resultados[0],
	}
	primera, errPrimera := CrearPropuestaDecisionCobertura(primeraEntrada)
	segunda, errSegunda := CrearPropuestaDecisionCobertura(segundaEntrada)
	if errPrimera != nil || errSegunda != nil ||
		!reflect.DeepEqual(primera.Publicacion(), segunda.Publicacion()) {
		t.Fatalf(
			"propuestas no deterministas: %v, %v",
			errPrimera,
			errSegunda,
		)
	}
	if primera.Referencia() !=
		"propuesta-cobertura:sha256:"+primera.HuellaSHA256() {
		t.Fatal("la referencia no está ligada al contenido")
	}
}

func TestPropuestaDecisionCoberturaEntregaCopiasYVinculoMinimizado(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := propuesta.Publicacion()
	contenido, err := json.Marshal(publicacion)
	if err != nil || strings.Contains(string(contenido), "detalle") {
		t.Fatalf("la propuesta expone detalle libre: %s, %v", contenido, err)
	}
	publicacion.Resultados[0].Evidencias[0].ReciboRef =
		"recibo_alterado_fuera_del_agregado"
	publicacion.Evaluaciones[0].AusenciasAdmitidas =
		append(publicacion.Evaluaciones[0].AusenciasAdmitidas, "otra_clave")
	if propuesta.ValidarPara(datos.Catalogo, datos.Politica) != nil {
		t.Fatal("una copia de salida alteró la propuesta")
	}
	vinculo, err := propuesta.VinculoParaDecision(
		"via_futura_configurable",
		datos.GeneradaEn,
	)
	if err != nil ||
		vinculo.PropuestaRef != propuesta.Referencia() ||
		vinculo.PropuestaHuella != propuesta.HuellaSHA256() ||
		len(vinculo.Resultados) != 3 {
		t.Fatalf("vínculo posterior incompleto: %#v, %v", vinculo, err)
	}
	vinculo.Resultados[0].Evidencias[0].ReciboRef =
		"recibo_alterado_en_vinculo"
	segundo, err := propuesta.VinculoParaDecision(
		"via_futura_configurable",
		datos.GeneradaEn,
	)
	if err != nil ||
		segundo.Resultados[0].Evidencias[0].ReciboRef ==
			"recibo_alterado_en_vinculo" {
		t.Fatal("el vínculo comparte evidencia mutable")
	}
	if _, err := propuesta.VinculoParaDecision(
		"via_que_no_esta_evaluada",
		datos.GeneradaEn,
	); err == nil {
		t.Fatal("se ligó una decisión a una vía no evaluada")
	}
}

func TestPropuestaDecisionCoberturaRechazaPublicacionAdulterada(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := propuesta.Publicacion()
	publicacion.Evaluaciones[0].Estado = EvaluacionViaCoberturaNoViable
	if _, err := RestaurarPropuestaDecisionCobertura(
		publicacion,
		datos.Catalogo,
		datos.Politica,
	); err == nil {
		t.Fatal("se restauró una evaluación alterada con la huella anterior")
	}
}

func TestPropuestaDecisionCoberturaDistingueOmitidoDeNoConstaAcreditado(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	datos.Resultados = datos.Resultados[:2]
	omitida, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil || omitida.Estado() != PropuestaCoberturaIncompleta {
		t.Fatalf("omisión no bloqueada: %s, %v", omitida.Estado(), err)
	}
	encontrada := false
	for _, evaluacion := range omitida.Evaluaciones() {
		if evaluacion.ViaClave == "via_futura_configurable" &&
			len(evaluacion.ResultadosOmitidos) == 1 &&
			evaluacion.ResultadosOmitidos[0] == "hecho_futuro" {
			encontrada = true
		}
	}
	if !encontrada {
		t.Fatal("la omisión se confundió con una ausencia acreditada")
	}

	datos = datosPropuestaDecisionCoberturaPrueba(t)
	datos.Resultados[0].Resultado = ComprobacionNoConsta
	bloqueada, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil || bloqueada.Estado() != PropuestaCoberturaIncompleta {
		t.Fatalf("no_consta bloqueante no se aplicó: %s, %v", bloqueada.Estado(), err)
	}
}

func TestPropuestaDecisionCoberturaVariasEvidenciasMismoResultadoNoConflicto(
	t *testing.T,
) {
	primeraEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	segundaEvidencia := resultadoDecisionCoberturaPrueba(
		"hecho_compartido",
		ComprobacionAfirmativa,
		"99",
	)
	primeraEntrada.Resultados = append(
		primeraEntrada.Resultados,
		segundaEvidencia,
	)
	segundaEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	segundaEntrada.Resultados = append(
		[]ComprobacionCobertura{segundaEvidencia},
		segundaEntrada.Resultados...,
	)
	primera, errPrimera := CrearPropuestaDecisionCobertura(primeraEntrada)
	segunda, errSegunda := CrearPropuestaDecisionCobertura(segundaEntrada)
	if errPrimera != nil || errSegunda != nil ||
		primera.Estado() != PropuestaCoberturaViable ||
		!reflect.DeepEqual(primera.Publicacion(), segunda.Publicacion()) {
		t.Fatalf(
			"evidencia coincidente se trató como conflicto: %v, %v",
			errPrimera,
			errSegunda,
		)
	}
}

func TestPropuestaDecisionCoberturaLigaAnalisisYCaducaEnLimiteExclusivo(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := propuesta.VinculoParaDecision(
		propuesta.ViaPropuesta(),
		datos.ValidaHasta.Add(-time.Microsecond),
	); err != nil {
		t.Fatalf("vínculo vigente rechazado: %v", err)
	}
	if _, err := propuesta.VinculoParaDecision(
		propuesta.ViaPropuesta(),
		datos.ValidaHasta,
	); err == nil {
		t.Fatal("el límite exclusivo de propuesta se reabrió")
	}
	publicacion := propuesta.Publicacion()
	publicacion.AnalisisHuellaSHA256 = strings.Repeat("b", 64)
	if _, err := RestaurarPropuestaDecisionCobertura(
		publicacion,
		datos.Catalogo,
		datos.Politica,
	); err == nil {
		t.Fatal("se desligó la propuesta del análisis")
	}

	datos = datosPropuestaDecisionCoberturaPrueba(t)
	datos.ValidaHasta = datos.Politica.Vigencia().Hasta.Add(time.Microsecond)
	if _, err := CrearPropuestaDecisionCobertura(datos); err == nil {
		t.Fatal("una propuesta sobrevivió a la retirada de su política")
	}
}

func TestPropuestaDecisionCoberturaAcotaEvidenciasContradictorias(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	for indice := 0; indice < maximoEvidenciasPorComprobacionCobertura; indice++ {
		datos.Resultados = append(
			datos.Resultados,
			resultadoDecisionCoberturaPrueba(
				"hecho_compartido",
				ComprobacionAfirmativa,
				"limite"+string(rune('a'+indice)),
			),
		)
	}
	if _, err := CrearPropuestaDecisionCobertura(datos); err == nil {
		t.Fatal("se superó el límite de evidencias por comprobación")
	}
}

func datosPropuestaDecisionCoberturaPrueba(
	t *testing.T,
) DatosCrearPropuestaDecisionCobertura {
	t.Helper()
	catalogo := catalogoDecisionCoberturaPrueba(t)
	politica, err := PublicarPoliticaDecisionCobertura(
		borradorPoliticaDecisionCoberturaPrueba(catalogo),
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return DatosCrearPropuestaDecisionCobertura{
		OrganizacionRef:      "organizacion_diputacion_granada",
		ExpedienteRef:        "expediente_temporal_configurable_01",
		VersionExpediente:    3,
		AnalisisRef:          "analisis_rrhh_configurable_01",
		AnalisisHuellaSHA256: strings.Repeat("a", 64),
		Catalogo:             catalogo,
		Politica:             politica,
		FinalidadClave:       "gestionar_cobertura_temporal",
		FinalidadRef:         "finalidad:contratacion-temporal:cobertura",
		CategoriaRef:         "categoria_configurable_01",
		Periodo: PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		GeneradaEn:  instantePoliticaCoberturaPrueba.Add(time.Minute),
		ValidaHasta: instantePoliticaCoberturaPrueba.Add(10 * time.Minute),
		Resultados: []ComprobacionCobertura{
			resultadoDecisionCoberturaPrueba(
				"hecho_compartido",
				ComprobacionAfirmativa,
				"10",
			),
			resultadoDecisionCoberturaPrueba(
				"hecho_alternativo",
				ComprobacionAfirmativa,
				"11",
			),
			resultadoDecisionCoberturaPrueba(
				"hecho_futuro",
				ComprobacionNoConsta,
				"12",
			),
		},
	}
}

func resultadoDecisionCoberturaPrueba(
	clave ClaveCatalogo,
	resultado ResultadoComprobacion,
	sufijo string,
) ComprobacionCobertura {
	return ComprobacionCobertura{
		Clave:     clave,
		Resultado: resultado,
		FuenteRef: "fuente_comprobacion_configurable_" + sufijo,
		ReciboRef: "recibo_comprobacion_configurable_" + sufijo,
		EvaluadaEn: instantePoliticaCoberturaPrueba.Add(
			30 * time.Second,
		),
	}
}
