package domain

import (
	"bytes"
	"encoding/json"
	"testing"
)

const (
	huellaSemanticaPropuestaSQLVector      = "a9f78ff73feb40d6a0ef0de9506ea23668c67b50fe2f1a685631bc63586a1806"
	huellaSemanticaPropuestaBordeSQLVector = "8f0cc8725238cbc355f4a1bb651e163938fe4fe36d7554554092602de0fd223b"
	huellaExactaPropuestaSQLVector         = "d17048b801f5bbacf56589d314b22a168a9427f058c79e27e610d1abf69e19af"
	huellaExactaPropuestaBordeSQLVector    = "8d50612104114047b20cfdf76e674dc4dc4996537e114ff8330c6b8836250171"
)

func TestVectorSQLHuellaSemanticaPropuestaCobertura(t *testing.T) {
	propuesta, err := CrearPropuestaDecisionCobertura(
		datosPropuestaDecisionCoberturaPrueba(t),
	)
	if err != nil {
		t.Fatalf("crear propuesta del vector: %v", err)
	}
	publicacion := propuesta.Publicacion()
	if publicacion.HuellaSHA256 != huellaExactaPropuestaSQLVector {
		t.Fatalf("huella exacta compartida pendiente/divergente: %s", publicacion.HuellaSHA256)
	}
	huella, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
		publicacion,
	)
	if err != nil || huella != huellaSemanticaPropuestaSQLVector {
		t.Fatalf("huella semántica compartida pendiente/divergente: %s, %v", huella, err)
	}
	contenido, err := json.Marshal(publicacion)
	if err != nil {
		t.Fatalf("serializar propuesta del vector: %v", err)
	}
	if bytes.Contains(contenido, []byte(`"conflictos":[]`)) ||
		bytes.Contains(contenido, []byte(`"no_habilitantes":[]`)) {
		t.Fatal("omitempty dejó listas vacías en el vector")
	}
	borde := propuesta.Publicacion()
	if len(borde.Evaluaciones) != 2 {
		t.Fatalf("el fixture dejó de tener dos vías: %d", len(borde.Evaluaciones))
	}
	borde.Evaluaciones[0].Prioridad = 32768
	borde.Evaluaciones[1].Prioridad = 65535
	actualizarIdentidadExactaPropuestaSQLVector(t, &borde)
	if borde.HuellaSHA256 != huellaExactaPropuestaBordeSQLVector {
		t.Fatalf("huella exacta de borde pendiente/divergente: %s", borde.HuellaSHA256)
	}
	huellaBorde, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
		borde,
	)
	if err != nil || huellaBorde != huellaSemanticaPropuestaBordeSQLVector {
		t.Fatalf("huella semántica de borde pendiente/divergente: %s, %v", huellaBorde, err)
	}
	reordenada := propuesta.Publicacion()
	invertirResultadosAgrupadosSemanticos(reordenada.Resultados)
	invertirEvaluacionesSemanticas(reordenada.Evaluaciones)
	for indice := range reordenada.Evaluaciones {
		invertirClavesSemanticasEvaluacion(&reordenada.Evaluaciones[indice])
	}
	huellaReordenada, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
		reordenada,
	)
	if err != nil || huellaReordenada != huella {
		t.Fatalf("la normalización semántica divergió: %s, %v", huellaReordenada, err)
	}
}

func actualizarIdentidadExactaPropuestaSQLVector(
	t *testing.T,
	publicacion *PublicacionPropuestaDecisionCobertura,
) {
	t.Helper()
	huella, err := calcularHuellaPropuestaDecisionCobertura(*publicacion)
	if err != nil {
		t.Fatalf("calcular identidad exacta del vector: %v", err)
	}
	publicacion.HuellaSHA256 = huella
	publicacion.Referencia = referenciaPropuestaDecisionCobertura(huella)
}

func invertirClavesSemanticasEvaluacion(evaluacion *EvaluacionViaPropuestaCobertura) {
	if evaluacion == nil {
		return
	}
	for _, lista := range []*[]ClaveCatalogo{
		&evaluacion.ResultadosOmitidos,
		&evaluacion.AusenciasBloqueantes,
		&evaluacion.AusenciasAdmitidas,
		&evaluacion.NoHabilitantes,
		&evaluacion.Conflictos,
	} {
		for izquierda, derecha := 0, len(*lista)-1; izquierda < derecha; izquierda, derecha =
			izquierda+1, derecha-1 {
			(*lista)[izquierda], (*lista)[derecha] =
				(*lista)[derecha], (*lista)[izquierda]
		}
	}
}
