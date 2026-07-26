package domain

import (
	"strings"
	"testing"
	"time"
)

func TestIdentidadSemanticaPropuestaCoberturaPermaneceEnRenovacionEquivalente(
	t *testing.T,
) {
	primeraEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	segundaEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	segundaEntrada.PreparacionEvidenciasRef =
		"preparacion_evidencias_renovada_02"
	segundaEntrada.PreparacionEvidenciasHuellaSHA256 = strings.Repeat("d", 64)
	segundaEntrada.GeneradaEn = segundaEntrada.GeneradaEn.Add(time.Microsecond)
	segundaEntrada.ValidaHasta = segundaEntrada.ValidaHasta.Add(-time.Microsecond)
	for indice := range segundaEntrada.Resultados {
		segundaEntrada.Resultados[indice].FuenteRef += "_renovada"
		segundaEntrada.Resultados[indice].ReciboRef += "_renovado"
		segundaEntrada.Resultados[indice].EvaluadaEn =
			segundaEntrada.Resultados[indice].EvaluadaEn.Add(time.Microsecond)
	}

	primera, errPrimera := CrearPropuestaDecisionCobertura(primeraEntrada)
	segunda, errSegunda := CrearPropuestaDecisionCobertura(segundaEntrada)
	if errPrimera != nil || errSegunda != nil {
		t.Fatalf("crear propuestas equivalentes: %v, %v", errPrimera, errSegunda)
	}
	if primera.HuellaSHA256() == segunda.HuellaSHA256() {
		t.Fatal("la identidad exacta ocultó la renovación de evidencias")
	}
	identidadPrimera, errPrimera := primera.IdentidadSemantica()
	identidadSegunda, errSegunda := segunda.IdentidadSemantica()
	if errPrimera != nil || errSegunda != nil {
		t.Fatalf("obtener identidades semánticas: %v, %v", errPrimera, errSegunda)
	}
	if identidadPrimera != identidadSegunda {
		t.Fatalf(
			"la renovación funcionalmente equivalente cambió la identidad: %#v / %#v",
			identidadPrimera,
			identidadSegunda,
		)
	}
	if identidadPrimera.Referencia !=
		referenciaSemanticaPropuestaDecisionCobertura(
			identidadPrimera.HuellaSHA256,
		) {
		t.Fatal("la referencia semántica no está dirigida por contenido")
	}
}

func TestIdentidadSemanticaPropuestaCoberturaValidaYComparaExactamente(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := propuesta.IdentidadSemantica()
	if err != nil || identidad.Validar() != nil {
		t.Fatalf("identidad calculada inválida: %#v, %v", identidad, err)
	}
	if !identidad.CoincideExactamente(identidad) {
		t.Fatal("una identidad válida no coincidió consigo misma")
	}

	ataques := []struct {
		nombre  string
		alterar func(*IdentidadSemanticaPropuestaDecisionCobertura)
	}{
		{"cero", func(i *IdentidadSemanticaPropuestaDecisionCobertura) {
			*i = IdentidadSemanticaPropuestaDecisionCobertura{}
		}},
		{"canon", func(i *IdentidadSemanticaPropuestaDecisionCobertura) {
			i.Canon.VersionEsquema++
		}},
		{"referencia", func(i *IdentidadSemanticaPropuestaDecisionCobertura) {
			i.Referencia = "propuesta-cobertura-semantica:sha256:adulterada"
		}},
		{"huella", func(i *IdentidadSemanticaPropuestaDecisionCobertura) {
			i.HuellaSHA256 = strings.Repeat("b", 64)
		}},
	}
	for _, ataque := range ataques {
		t.Run(ataque.nombre, func(t *testing.T) {
			adulterada := identidad
			ataque.alterar(&adulterada)
			if adulterada.Validar() == nil {
				t.Fatal("la identidad adulterada conservó validez nominal")
			}
			if identidad.CoincideExactamente(adulterada) ||
				adulterada.CoincideExactamente(identidad) {
				t.Fatal("la comparación aceptó una identidad adulterada")
			}
		})
	}

	otrosDatos := datosPropuestaDecisionCoberturaPrueba(t)
	otrosDatos.VersionExpediente++
	otraPropuesta, err := CrearPropuestaDecisionCobertura(otrosDatos)
	if err != nil {
		t.Fatal(err)
	}
	otraIdentidad, err := otraPropuesta.IdentidadSemantica()
	if err != nil || otraIdentidad.Validar() != nil {
		t.Fatalf("segunda identidad inválida: %#v, %v", otraIdentidad, err)
	}
	if identidad.CoincideExactamente(otraIdentidad) {
		t.Fatal("dos significados válidos diferentes coincidieron")
	}
}

func TestHuellaSemanticaPropuestaCoberturaIncluyeCadaDatoDecidible(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	canon := CanonHuellaSemanticaPropuestaDecisionCoberturaV1()
	base := propuesta.Publicacion()
	huellaBase, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		canon,
		base,
	)
	if err != nil {
		t.Fatal(err)
	}
	ataques := []struct {
		nombre  string
		alterar func(*PublicacionPropuestaDecisionCobertura)
	}{
		{"organizacion", func(p *PublicacionPropuestaDecisionCobertura) {
			p.OrganizacionRef = "organizacion_semantica_alternativa"
		}},
		{"expediente", func(p *PublicacionPropuestaDecisionCobertura) {
			p.ExpedienteRef = "expediente_semantico_alternativo"
		}},
		{"version_expediente", func(p *PublicacionPropuestaDecisionCobertura) {
			p.VersionExpediente++
		}},
		{"analisis", func(p *PublicacionPropuestaDecisionCobertura) {
			p.AnalisisRef = "analisis_semantico_alternativo"
		}},
		{"huella_analisis", func(p *PublicacionPropuestaDecisionCobertura) {
			p.AnalisisHuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"catalogo_referencia", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Catalogo.Referencia = "catalogo_semantico_alternativo"
		}},
		{"catalogo_version", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Catalogo.Version++
		}},
		{"catalogo_huella", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Catalogo.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"politica_referencia", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Politica.Referencia = "politica_semantica_alternativa"
		}},
		{"politica_version", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Politica.Version++
		}},
		{"politica_huella", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Politica.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"finalidad_clave", func(p *PublicacionPropuestaDecisionCobertura) {
			p.FinalidadClave = "finalidad_semantica_alternativa"
		}},
		{"finalidad_referencia", func(p *PublicacionPropuestaDecisionCobertura) {
			p.FinalidadRef = "finalidad:semantica:alternativa"
		}},
		{"categoria", func(p *PublicacionPropuestaDecisionCobertura) {
			p.CategoriaRef = "categoria_semantica_alternativa"
		}},
		{"periodo_inicio", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Periodo.Inicio = p.Periodo.Inicio.Add(24 * time.Hour)
		}},
		{"periodo_fin", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Periodo.Fin = p.Periodo.Fin.Add(24 * time.Hour)
		}},
		{"estado", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Estado = PropuestaCoberturaSinVia
			p.ViaPropuesta = ""
		}},
		{"via_recomendada", func(p *PublicacionPropuestaDecisionCobertura) {
			p.ViaPropuesta = "via_semantica_alternativa"
		}},
		{"resultado_clave", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Resultados[0].Clave = "resultado_semantico_alternativo"
		}},
		{"resultado_funcional", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Resultados[0].Evidencias[0].Resultado = ComprobacionNoAplica
		}},
		{"evaluacion_via", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].ViaClave = "via_evaluada_alternativa"
		}},
		{"evaluacion_prioridad", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].Prioridad = 50
		}},
		{"evaluacion_estado", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].Estado = EvaluacionViaCoberturaNoViable
		}},
		{"resultados_omitidos", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].ResultadosOmitidos =
				append(p.Evaluaciones[0].ResultadosOmitidos, "omitido_semantico")
		}},
		{"ausencias_bloqueantes", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].AusenciasBloqueantes =
				append(p.Evaluaciones[0].AusenciasBloqueantes, "bloqueo_semantico")
		}},
		{"ausencias_admitidas", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].AusenciasAdmitidas =
				append(p.Evaluaciones[0].AusenciasAdmitidas, "ausencia_semantica")
		}},
		{"no_habilitantes", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].NoHabilitantes =
				append(p.Evaluaciones[0].NoHabilitantes, "no_habilita_semantico")
		}},
		{"conflictos", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].Conflictos =
				append(p.Evaluaciones[0].Conflictos, "conflicto_semantico")
		}},
	}
	for _, ataque := range ataques {
		t.Run(ataque.nombre, func(t *testing.T) {
			mutada := propuesta.Publicacion()
			ataque.alterar(&mutada)
			huella, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
				canon,
				mutada,
			)
			if err != nil {
				t.Fatalf("el ataque no produjo material válido: %v", err)
			}
			if huella == huellaBase {
				t.Fatal("un dato decidible no cambió la huella semántica")
			}
		})
	}
}

func TestHuellaSemanticaDistingueResultadoAunqueLaEvaluacionSeaIgual(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	afirmativa := propuesta.Publicacion()
	noAplica := propuesta.Publicacion()
	noAplica.Resultados[0].Evidencias[0].Resultado = ComprobacionNoAplica
	canon := CanonHuellaSemanticaPropuestaDecisionCoberturaV1()
	huellaAfirmativa, errAfirmativa :=
		calcularHuellaSemanticaPropuestaDecisionCobertura(canon, afirmativa)
	huellaNoAplica, errNoAplica :=
		calcularHuellaSemanticaPropuestaDecisionCobertura(canon, noAplica)
	if errAfirmativa != nil || errNoAplica != nil {
		t.Fatalf("calcular huellas: %v, %v", errAfirmativa, errNoAplica)
	}
	if huellaAfirmativa == huellaNoAplica {
		t.Fatal("afirmativa y no_aplica compartieron huella con igual evaluación")
	}
}

func TestIdentidadSemanticaPropuestaCoberturaCubreTodosLosEstados(
	t *testing.T,
) {
	casos := []struct {
		nombre     string
		estado     EstadoPropuestaDecisionCobertura
		resultados []ComprobacionCobertura
	}{
		{
			"viable",
			PropuestaCoberturaViable,
			datosPropuestaDecisionCoberturaPrueba(t).Resultados,
		},
		{
			"incompleta",
			PropuestaCoberturaIncompleta,
			[]ComprobacionCobertura{
				resultadoDecisionCoberturaPrueba(
					"hecho_alternativo",
					ComprobacionAfirmativa,
					"semantica_incompleta",
				),
			},
		},
		{
			"conflictiva",
			PropuestaCoberturaConflictiva,
			[]ComprobacionCobertura{
				resultadoDecisionCoberturaPrueba(
					"hecho_compartido",
					ComprobacionAfirmativa,
					"semantica_conflicto_a",
				),
				resultadoDecisionCoberturaPrueba(
					"hecho_compartido",
					ComprobacionNegativa,
					"semantica_conflicto_b",
				),
			},
		},
		{
			"sin_via",
			PropuestaCoberturaSinVia,
			[]ComprobacionCobertura{
				resultadoDecisionCoberturaPrueba(
					"hecho_compartido",
					ComprobacionNegativa,
					"semantica_sin_via_a",
				),
				resultadoDecisionCoberturaPrueba(
					"hecho_alternativo",
					ComprobacionAfirmativa,
					"semantica_sin_via_b",
				),
				resultadoDecisionCoberturaPrueba(
					"hecho_futuro",
					ComprobacionNoConsta,
					"semantica_sin_via_c",
				),
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := datosPropuestaDecisionCoberturaPrueba(t)
			datos.Resultados = caso.resultados
			propuesta, err := CrearPropuestaDecisionCobertura(datos)
			if err != nil {
				t.Fatal(err)
			}
			if propuesta.Estado() != caso.estado {
				t.Fatalf("estado inesperado: %s", propuesta.Estado())
			}
			identidad, err := propuesta.IdentidadSemantica()
			if err != nil || !huellaValida(identidad.HuellaSHA256) {
				t.Fatalf("identidad cerrada inválida: %#v, %v", identidad, err)
			}
		})
	}
}

func TestIdentidadSemanticaPropuestaCoberturaRechazaRestauracionManipulada(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := propuesta.Publicacion()
	publicacion.Evaluaciones[0].Estado = EvaluacionViaCoberturaNoViable
	restaurada, err := RestaurarPropuestaDecisionCobertura(
		publicacion,
		datos.Catalogo,
		datos.Politica,
	)
	if err == nil {
		t.Fatal("se restauró una propuesta exacta manipulada")
	}
	if _, err := restaurada.IdentidadSemantica(); err == nil {
		t.Fatal("una restauración fallida produjo identidad de preview")
	}
}

func TestHuellaSemanticaPropuestaCoberturaEsCanonicaYAcotada(t *testing.T) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	canon := CanonHuellaSemanticaPropuestaDecisionCoberturaV1()
	base := propuesta.Publicacion()
	huellaBase, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		canon,
		base,
	)
	if err != nil {
		t.Fatal(err)
	}
	reordenada := propuesta.Publicacion()
	invertirResultadosAgrupadosSemanticos(reordenada.Resultados)
	invertirEvaluacionesSemanticas(reordenada.Evaluaciones)
	huellaReordenada, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		canon,
		reordenada,
	)
	if err != nil || huellaReordenada != huellaBase {
		t.Fatalf("canon no determinista: %s / %s, %v", huellaBase, huellaReordenada, err)
	}

	casos := []struct {
		nombre  string
		alterar func(*PublicacionPropuestaDecisionCobertura)
	}{
		{"demasiados_resultados", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Resultados = make(
				[]ResultadoAgrupadoPropuestaCobertura,
				maximoComprobacionesCatalogo+1,
			)
		}},
		{"demasiadas_evidencias", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Resultados[0].Evidencias = make(
				[]EvidenciaComprobacionPropuestaCobertura,
				maximoEvidenciasPorComprobacionCobertura+1,
			)
		}},
		{"demasiadas_vias", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones = make(
				[]EvaluacionViaPropuestaCobertura,
				maximoViasCobertura+1,
			)
		}},
		{"demasiadas_comprobaciones", func(p *PublicacionPropuestaDecisionCobertura) {
			p.Evaluaciones[0].ResultadosOmitidos = make(
				[]ClaveCatalogo,
				maximoComprobacionesPorViaCobertura+1,
			)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := propuesta.Publicacion()
			caso.alterar(&mutada)
			if _, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
				canon,
				mutada,
			); err == nil {
				t.Fatal("el canon aceptó cardinalidad fuera de límite")
			}
		})
	}
	canon.VersionEsquema++
	if _, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		canon,
		base,
	); err == nil {
		t.Fatal("se aceptó un canon semántico desconocido")
	}
}

func invertirResultadosAgrupadosSemanticos(
	resultados []ResultadoAgrupadoPropuestaCobertura,
) {
	for izquierda, derecha := 0, len(resultados)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		resultados[izquierda], resultados[derecha] =
			resultados[derecha], resultados[izquierda]
	}
}

func invertirEvaluacionesSemanticas(evaluaciones []EvaluacionViaPropuestaCobertura) {
	for izquierda, derecha := 0, len(evaluaciones)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		evaluaciones[izquierda], evaluaciones[derecha] =
			evaluaciones[derecha], evaluaciones[izquierda]
	}
	for indice := range evaluaciones {
		listas := [][]ClaveCatalogo{
			evaluaciones[indice].ResultadosOmitidos,
			evaluaciones[indice].AusenciasBloqueantes,
			evaluaciones[indice].AusenciasAdmitidas,
			evaluaciones[indice].NoHabilitantes,
			evaluaciones[indice].Conflictos,
		}
		for _, lista := range listas {
			for izquierda, derecha := 0, len(lista)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
				lista[izquierda], lista[derecha] = lista[derecha], lista[izquierda]
			}
		}
	}
}
