package domain

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func borradorCatalogoCoberturaConPreparacion() BorradorCatalogoViasCobertura {
	borrador := borradorCatalogoCoberturaValido()
	borrador.EsEjemplo = true
	borrador.Vias[1].Documentos = []ElementoPreparacionViaCobertura{
		{Clave: "documento_b", Orden: 20, ClaveI18n: "ct.cobertura.documento_b"},
		{Clave: "documento_a", Orden: 10, ClaveI18n: "ct.cobertura.documento_a"},
	}
	borrador.Vias[1].Datos = []ElementoPreparacionViaCobertura{
		{Clave: "dato_b", Orden: 20, ClaveI18n: "ct.cobertura.dato_b"},
		{Clave: "dato_a", Orden: 10, ClaveI18n: "ct.cobertura.dato_a"},
	}
	borrador.Vias[0].Datos = []ElementoPreparacionViaCobertura{{
		Clave: "dato_sae", Orden: 1, ClaveI18n: "ct.cobertura.dato_sae",
	}}
	return borrador
}

func TestCatalogoCoberturaV1ConservaJSONYRechazaPreparacion(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.Canon() != CanonHuellaCatalogoCoberturaV1() {
		t.Fatal("la publicación histórica dejó de usar V1")
	}
	datos, err := json.Marshal(catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), `"documentos"`) || strings.Contains(string(datos), `"datos"`) {
		t.Fatal("los nuevos campos alteraron el JSON histórico")
	}
	mutada := catalogo.Publicacion()
	mutada.Vias[0].Documentos = []ElementoPreparacionViaCobertura{{
		Clave: "documento_nuevo", Orden: 1, ClaveI18n: "ct.cobertura.documento_nuevo",
	}}
	if _, err := RestaurarCatalogoViasCobertura(mutada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("V1 aceptó contenido no sellado: %v", err)
	}
	mutada = catalogo.Publicacion()
	mutada.Vias[0].Documentos = []ElementoPreparacionViaCobertura{}
	if _, err := RestaurarCatalogoViasCobertura(mutada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("V1 aceptó un array explícitamente vacío: %v", err)
	}
	var explicita PublicacionCatalogoViasCobertura
	conMarca := strings.Replace(string(datos), `"vias":`, `"es_ejemplo":false,"vias":`, 1)
	if err := json.Unmarshal([]byte(conMarca), &explicita); err == nil {
		if _, err := RestaurarCatalogoViasCobertura(explicita); !errors.Is(err, ErrDatoInvalido) {
			t.Fatalf("V1 aceptó es_ejemplo presente: %v", err)
		}
	}
	conDocumentoNulo := strings.Replace(string(datos), `"comprobaciones":`, `"documentos":null,"comprobaciones":`, 1)
	if err := json.Unmarshal([]byte(conDocumentoNulo), &explicita); err == nil {
		if _, err := RestaurarCatalogoViasCobertura(explicita); !errors.Is(err, ErrDatoInvalido) {
			t.Fatalf("V1 aceptó documentos:null: %v", err)
		}
	}
}

func TestCatalogoCoberturaV2SellaYRestauraPreparacionOrdenada(t *testing.T) {
	borrador := borradorCatalogoCoberturaConPreparacion()
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.Canon() != CanonHuellaCatalogoCoberturaV2() {
		t.Fatal("los metadatos no activaron V2")
	}
	if !catalogo.Publicacion().EsEjemplo {
		t.Fatal("se perdió la procedencia de ejemplo")
	}
	publicacion := catalogo.Publicacion()
	if publicacion.Vias[0].Documentos[0].Clave != "documento_a" ||
		publicacion.Vias[0].Datos[0].Clave != "dato_a" {
		t.Fatal("listas no ordenadas por su orden gobernado")
	}
	borrador.Vias[1].Documentos[0].Clave = "alterado"
	publicacion.Vias[0].Documentos[0].Clave = "alterado"
	publicacion.Vias[0].Datos[0].Clave = "alterado"
	via, ok := catalogo.Via("bolsa_vigente")
	if !ok || via.Documentos[0].Clave != "documento_a" || via.Datos[0].Clave != "dato_a" {
		t.Fatal("el catálogo compartió metadatos mutables")
	}
	via.Documentos[0].Clave = "alterado"
	if nueva, _ := catalogo.Via("bolsa_vigente"); nueva.Documentos[0].Clave != "documento_a" {
		t.Fatal("Via devolvió una lista mutable compartida")
	}
	codificado, err := json.Marshal(catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	var recuperada PublicacionCatalogoViasCobertura
	if err := json.Unmarshal(codificado, &recuperada); err != nil {
		t.Fatal(err)
	}
	restaurado, err := RestaurarCatalogoViasCobertura(recuperada)
	if err != nil || !restaurado.Identidad().CoincideExactamente(catalogo.Identidad()) {
		t.Fatalf("V2 no sobrevivió JSON/rehidratación: %v", err)
	}
	if strings.Contains(string(codificado), `"documentos":[]`) ||
		strings.Contains(string(codificado), `"datos":[]`) {
		t.Fatal("se emitieron arrays vacíos no canónicos")
	}
	if !strings.Contains(string(codificado), `"es_ejemplo":true`) {
		t.Fatal("el JSON V2 perdió la marca de ejemplo")
	}
	var sinClaveI18n map[string]any
	if err := json.Unmarshal(codificado, &sinClaveI18n); err != nil {
		t.Fatal(err)
	}
	viasJSON := sinClaveI18n["vias"].([]any)
	documentosJSON := viasJSON[0].(map[string]any)["documentos"].([]any)
	delete(documentosJSON[0].(map[string]any), "clave_i18n")
	alteradoJSON, err := json.Marshal(sinClaveI18n)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(alteradoJSON, &recuperada); err == nil {
		if _, err := RestaurarCatalogoViasCobertura(recuperada); !errors.Is(err, ErrDatoInvalido) {
			t.Fatalf("V2 aceptó un documento sin clave i18n: %v", err)
		}
	}
}

func TestCatalogoCoberturaV2DetectaMutacionYConflictoDeVersion(t *testing.T) {
	borrador := borradorCatalogoCoberturaConPreparacion()
	primero, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	segundoBorrador := borradorCatalogoCoberturaConPreparacion()
	segundoBorrador.Vias[1].Datos[0].Clave = "dato_distinto"
	segundo, err := PublicarCatalogoViasCobertura(segundoBorrador)
	if err != nil {
		t.Fatal(err)
	}
	if primero.HuellaSHA256() == segundo.HuellaSHA256() ||
		!errors.Is(ValidarReintentoPublicacionCatalogoCobertura(
			primero.Identidad(), segundo.Identidad(),
		), ErrPublicacionCatalogoEnConflicto) {
		t.Fatal("otra preparación reutilizó la misma identidad durable")
	}
	mutada := primero.Publicacion()
	mutada.Vias[0].Datos[0].Clave = "dato_distinto"
	if _, err := RestaurarCatalogoViasCobertura(mutada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("V2 aceptó preparación alterada: %v", err)
	}
	mutada = primero.Publicacion()
	mutada.Canon = CanonHuellaCatalogoCoberturaV1()
	if _, err := RestaurarCatalogoViasCobertura(mutada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se rebajó V2 a V1: %v", err)
	}
	mutada = primero.Publicacion()
	mutada.EsEjemplo = false
	if _, err := RestaurarCatalogoViasCobertura(mutada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("V2 aceptó alteración de marca de ejemplo: %v", err)
	}
	mutada = primero.Publicacion()
	mutada.Vias[0].Documentos = nil
	mutada.Vias[0].Datos = nil
	mutada.Vias[1].Datos = nil
	if _, err := RestaurarCatalogoViasCobertura(mutada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("V2 aceptó una preparación ausente: %v", err)
	}
}

func TestCatalogoCoberturaV2LimitesYDuplicados(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*BorradorCatalogoViasCobertura)
	}{
		{"documento sin clave", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Documentos[0].Clave = "" }},
		{"orden cero", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Datos[0].Orden = 0 }},
		{"sin clave i18n", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Datos[0].ClaveI18n = "" }},
		{"clave i18n sin espacio de nombres", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Datos[0].ClaveI18n = "dato_solo" }},
		{"clave duplicada", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Documentos[1].Clave = b.Vias[1].Documentos[0].Clave }},
		{"orden duplicado", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Datos[1].Orden = b.Vias[1].Datos[0].Orden }},
		{"array vacío", func(b *BorradorCatalogoViasCobertura) { b.Vias[1].Documentos = []ElementoPreparacionViaCobertura{} }},
		{"demasiados documentos", func(b *BorradorCatalogoViasCobertura) {
			b.Vias[1].Documentos = elementosPreparacionPrueba(maximoElementosPreparacionPorVia + 1)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := borradorCatalogoCoberturaConPreparacion()
			caso.mutar(&borrador)
			if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
				t.Fatalf("se aceptó preparación inválida: %v", err)
			}
		})
	}
}

func TestCatalogoCoberturaV2MarcaEjemploNoCreaEsquemaVacio(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.EsEjemplo = true
	if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("una marca de ejemplo sin relaciones creó V2: %v", err)
	}
	borrador = borradorCatalogoCoberturaConPreparacion()
	borrador.EsEjemplo = false
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil || catalogo.Canon() != CanonHuellaCatalogoCoberturaV2() ||
		catalogo.Publicacion().EsEjemplo {
		t.Fatalf("V2 no admitió una futura fuente aprobada: %v", err)
	}
}

func TestCatalogoCoberturaV2AcotaElementosTotales(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vias = viasNumeradas(16)
	for indice := range borrador.Vias {
		borrador.Vias[indice].Documentos = elementosPreparacionPrueba(16)
		borrador.Vias[indice].Datos = elementosPreparacionPrueba(16)
	}
	if _, err := PublicarCatalogoViasCobertura(borrador); err != nil {
		t.Fatalf("límite positivo de 512 rechazado: %v", err)
	}
	borrador.Vias = append(borrador.Vias, DefinicionViaCobertura{
		Clave: "via_extra", Orden: 17,
		Comprobaciones: []ComprobacionExigibleCobertura{{
			Clave: "comprobacion_extra", Orden: 1, Obligatoria: true,
			Procedencia: procedencia("fuente_extra", "definicion_fuente_extra"),
		}},
		Datos: elementosPreparacionPrueba(1),
	})
	if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se aceptaron más de 512 elementos: %v", err)
	}
}

func TestCatalogoCoberturaRechazaNombresJSONInexactosYDuplicados(t *testing.T) {
	v1, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaValido())
	if err != nil {
		t.Fatal(err)
	}
	v2, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaConPreparacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonV1, err := json.Marshal(v1.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonV2, err := json.Marshal(v2.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, original, anterior, nuevo string
	}{
		{"V1 DOCUMENTOS null", string(jsonV1), `"comprobaciones":`, `"DOCUMENTOS":null,"comprobaciones":`},
		{"V1 DATOS vacío", string(jsonV1), `"comprobaciones":`, `"DATOS":[],"comprobaciones":`},
		{"V1 ES_EJEMPLO", string(jsonV1), `"vias":`, `"ES_EJEMPLO":true,"vias":`},
		{"V2 CLAVE_I18N", string(jsonV2), `"clave_i18n":`, `"CLAVE_I18N":`},
		{"V2 Documentos mixto", string(jsonV2), `"documentos":`, `"Documentos":`},
		{"V2 es_ejemplo mixto", string(jsonV2), `"es_ejemplo":`, `"Es_Ejemplo":`},
		{"V2 procedencia anidada", string(jsonV2), `"procedencia":`, `"Procedencia":`},
		{"duplicado exacto raíz", string(jsonV1), `"referencia":`, `"referencia":"otra","referencia":`},
		{"duplicado exacto vía", string(jsonV2), `"documentos":`, `"documentos":null,"documentos":`},
		{"duplicado exacto elemento", string(jsonV2), `"clave_i18n":`, `"clave_i18n":"otra.clave","clave_i18n":`},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := strings.Replace(caso.original, caso.anterior, caso.nuevo, 1)
			if alterado == caso.original {
				t.Fatal("el caso no alteró el JSON")
			}
			var publicacion PublicacionCatalogoViasCobertura
			if err := json.Unmarshal([]byte(alterado), &publicacion); err == nil {
				t.Fatal("se aceptaron nombres inexactos o duplicados")
			}
		})
	}
}

func TestCatalogoCoberturaRechazaHastaNuloEnJSONV1(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vigencia.Hasta = time.Time{}
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	alterado := strings.Replace(string(contenido), `"hasta":"0001-01-01T00:00:00Z"`, `"hasta":null`, 1)
	if alterado == string(contenido) {
		t.Fatal("el caso no alteró la fecha abierta")
	}
	var publicacion PublicacionCatalogoViasCobertura
	if err := json.Unmarshal([]byte(alterado), &publicacion); err == nil {
		if _, err := RestaurarCatalogoViasCobertura(publicacion); err == nil {
			t.Fatal("se aceptó hasta:null como la ausencia temporal sellada")
		}
	}
}

func TestCatalogoCoberturaRechazaObligatoriedadNulaEnJSONV1(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vias[0].Comprobaciones[0].Obligatoria = false
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	alterado := strings.Replace(string(contenido), `"obligatoria":false`, `"obligatoria":null`, 1)
	if alterado == string(contenido) {
		t.Fatal("el caso no alteró la obligatoriedad")
	}
	var publicacion PublicacionCatalogoViasCobertura
	if err := json.Unmarshal([]byte(alterado), &publicacion); err == nil {
		if _, err := RestaurarCatalogoViasCobertura(publicacion); err == nil {
			t.Fatal("se aceptó obligatoria:null como false sellado")
		}
	}
}

func TestCatalogoCoberturaExigeClavesObligatoriasPorObjetoJSON(t *testing.T) {
	borradorV1 := borradorCatalogoCoberturaValido()
	borradorV1.Vigencia.Hasta = time.Time{}
	borradorV1.Vias[0].Comprobaciones[0].Obligatoria = false
	v1, err := PublicarCatalogoViasCobertura(borradorV1)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaConPreparacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonV1, err := json.Marshal(v1.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonV2, err := json.Marshal(v2.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		base   []byte
		ruta   []string
	}{
		{"raíz", jsonV1, []string{"version"}},
		{"canon", jsonV1, []string{"canon", "version_esquema"}},
		{"vigencia hasta cero", jsonV1, []string{"vigencia", "hasta"}},
		{"vía", jsonV1, []string{"vias", "0", "orden"}},
		{"comprobación false", jsonV1, []string{"vias", "0", "comprobaciones", "0", "obligatoria"}},
		{"procedencia", jsonV1, []string{"vias", "0", "comprobaciones", "0", "procedencia", "clave"}},
		{"elemento V2", jsonV2, []string{"vias", "0", "documentos", "0", "clave_i18n"}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := omitirClaveCatalogoJSONPrueba(t, caso.base, caso.ruta...)
			var publicacion PublicacionCatalogoViasCobertura
			if err := json.Unmarshal(alterado, &publicacion); err == nil {
				t.Fatal("la decodificación aceptó una clave obligatoria omitida")
			}
		})
	}
}

func TestCatalogoCoberturaDistingueCamposOpcionalesV1V2EnJSON(t *testing.T) {
	v1, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaValido())
	if err != nil {
		t.Fatal(err)
	}
	v2, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaConPreparacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonV1, err := json.Marshal(v1.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonV2, err := json.Marshal(v2.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, original, anterior, nuevo string
	}{
		{"V1 es_ejemplo", string(jsonV1), `"vias":`, `"es_ejemplo":true,"vias":`},
		{"V1 documentos", string(jsonV1), `"comprobaciones":`, `"documentos":[{"clave":"documento_x","orden":1,"clave_i18n":"ct.documento_x"}],"comprobaciones":`},
		{"V1 datos", string(jsonV1), `"comprobaciones":`, `"datos":[{"clave":"dato_x","orden":1,"clave_i18n":"ct.dato_x"}],"comprobaciones":`},
		{"V2 false explícito", string(jsonV2), `"es_ejemplo":true`, `"es_ejemplo":false`},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := strings.Replace(caso.original, caso.anterior, caso.nuevo, 1)
			if alterado == caso.original {
				t.Fatal("el caso no alteró el JSON")
			}
			var publicacion PublicacionCatalogoViasCobertura
			if err := json.Unmarshal([]byte(alterado), &publicacion); err == nil {
				t.Fatal("se aceptó un campo opcional fuera de su canon")
			}
		})
	}
}

func omitirClaveCatalogoJSONPrueba(t *testing.T, contenido []byte, ruta ...string) []byte {
	t.Helper()
	var objeto any
	if err := json.Unmarshal(contenido, &objeto); err != nil {
		t.Fatal(err)
	}
	actual := objeto
	for _, paso := range ruta[:len(ruta)-1] {
		switch valor := actual.(type) {
		case map[string]any:
			actual = valor[paso]
		case []any:
			indice, err := strconv.Atoi(paso)
			if err != nil || indice < 0 || indice >= len(valor) {
				t.Fatalf("índice inválido en ruta JSON: %q", paso)
			}
			actual = valor[indice]
		default:
			t.Fatalf("ruta JSON inválida antes de %q", paso)
		}
	}
	campo := ruta[len(ruta)-1]
	destino, ok := actual.(map[string]any)
	if !ok {
		t.Fatalf("ruta JSON no acaba en objeto: %v", ruta)
	}
	if _, existe := destino[campo]; !existe {
		t.Fatalf("campo a omitir ausente: %q", campo)
	}
	delete(destino, campo)
	alterado, err := json.Marshal(objeto)
	if err != nil {
		t.Fatal(err)
	}
	return alterado
}

func elementosPreparacionPrueba(cuantos int) []ElementoPreparacionViaCobertura {
	elementos := make([]ElementoPreparacionViaCobertura, cuantos)
	for indice := range elementos {
		elementos[indice] = ElementoPreparacionViaCobertura{
			Clave:     ClaveCatalogo("dato_" + strings.Repeat("x", indice+1)),
			Orden:     uint16(indice + 1),
			ClaveI18n: ClaveCatalogo("ct.cobertura.dato_" + strings.Repeat("x", indice+1)),
		}
	}
	return elementos
}

func TestCatalogoCoberturaV2VectorSQL(t *testing.T) {
	instante := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	borrador := BorradorCatalogoViasCobertura{
		Referencia: "catalogo_vector_v2", Version: 1,
		PublicadoEn: instante, Vigencia: VigenciaCatalogoCobertura{Desde: instante},
		ProcedenciaRef: "procedencia_vector_v2", EsEjemplo: true,
		Vias: []DefinicionViaCobertura{{
			Clave: "bolsa_vigente", Orden: 1,
			Comprobaciones: []ComprobacionExigibleCobertura{{
				Clave: "existe_bolsa_vigente", Orden: 1, Obligatoria: true,
				Procedencia: ProcedenciaComprobacionCobertura{
					Clave: "bolsa", DefinicionFuenteRef: "fuente_vector_v2",
				},
			}},
			Documentos: []ElementoPreparacionViaCobertura{{
				Clave: "informe_necesidad", Orden: 1,
				ClaveI18n: "ct.cobertura.doc.informe_necesidad",
			}},
			Datos: []ElementoPreparacionViaCobertura{{
				Clave: "categoria", Orden: 1,
				ClaveI18n: "ct.cobertura.dato.categoria",
			}},
		}},
	}
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	const huellaV2Esperada = "ed4cb260730e1f95bf235e564f070d3bc99203d228c2ae8c01922a561ade675c"
	if catalogo.HuellaSHA256() != huellaV2Esperada {
		t.Fatalf("vector V2 alterado: %s", catalogo.HuellaSHA256())
	}
	material, err := materialCanonicoCatalogoCoberturaV2(catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	jsonVector, err := json.Marshal(catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("v2-json=%s", jsonVector)
	t.Logf("v2-material=%s", hex.EncodeToString(material))
	t.Logf("v2-huella=%s", catalogo.HuellaSHA256())
}
