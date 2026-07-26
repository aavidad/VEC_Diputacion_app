package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	esquemaLoteConsumoC1O404D = "vec.contratacion-temporal." +
		"consumo-c1.o4-04d.v1"
	dominioCanonLoteConsumoC1O404D = "VEC-CT-CONSUMO-C1-LOTE-O4-04D-V1"
	dominioCanonEvidenciaC1O404D   = "VEC-CT-CONSUMO-C1-EVIDENCIA-O4-04D-V1"
	dominioRecursoCoberturaO404D   = "VEC-CT-RECURSO-COBERTURA-O4-04D-V1"
)

type canonConsumoC1O404D struct {
	bytes.Buffer
}

func (c *canonConsumoC1O404D) texto(valor string) {
	_ = binary.Write(&c.Buffer, binary.BigEndian, uint32(len(valor)))
	_, _ = c.WriteString(valor)
}

func (c *canonConsumoC1O404D) entero64(valor uint64) {
	_ = binary.Write(&c.Buffer, binary.BigEndian, valor)
}

func (c *canonConsumoC1O404D) entero32(valor uint32) {
	_ = binary.Write(&c.Buffer, binary.BigEndian, valor)
}

func (c *canonConsumoC1O404D) booleano(valor bool) {
	if valor {
		_ = c.WriteByte(1)
		return
	}
	_ = c.WriteByte(0)
}

func (c *canonConsumoC1O404D) instante(valor time.Time) {
	c.entero64(uint64(valor.UnixMicro()))
}

func textoFixtureConsumoC1O404D(
	datos map[string]any,
	clave string,
) string {
	valor, _ := datos[clave].(string)
	return valor
}

func enteroFixtureConsumoC1O404D(
	datos map[string]any,
	clave string,
) uint64 {
	switch valor := datos[clave].(type) {
	case int:
		return uint64(valor)
	case uint64:
		return valor
	case float64:
		return uint64(valor)
	default:
		return 0
	}
}

func instanteFixtureConsumoC1O404D(
	datos map[string]any,
	clave string,
) time.Time {
	valor, _ := time.Parse(
		time.RFC3339Nano,
		textoFixtureConsumoC1O404D(datos, clave),
	)
	return valor
}

func pruebaFixtureConsumoC1O404D(
	datos map[string]any,
	clave string,
) []byte {
	contenido, _ := hex.DecodeString(
		textoFixtureConsumoC1O404D(datos, clave),
	)
	return contenido
}

func canonEvidenciaFixtureConsumoC1O404D(
	evidencia map[string]any,
) []byte {
	canon := &canonConsumoC1O404D{}
	canon.texto(dominioCanonEvidenciaC1O404D)
	canon.entero64(enteroFixtureConsumoC1O404D(evidencia, "posicion"))
	canon.entero64(enteroFixtureConsumoC1O404D(evidencia, "total"))
	for _, clave := range []string{"organizacion_ref", "expediente_ref"} {
		canon.texto(textoFixtureConsumoC1O404D(evidencia, clave))
	}
	canon.entero64(
		enteroFixtureConsumoC1O404D(evidencia, "version_expediente"),
	)
	for _, clave := range []string{
		"peticion_ref", "huella_peticion_sha256",
		"huella_resultado_sha256", "autoridad_ref",
	} {
		canon.texto(textoFixtureConsumoC1O404D(evidencia, clave))
	}
	canon.entero64(enteroFixtureConsumoC1O404D(evidencia, "generacion"))
	for _, clave := range []string{
		"recibo_respuesta_ref", "huella_respuesta_sha256", "catalogo_ref",
	} {
		canon.texto(textoFixtureConsumoC1O404D(evidencia, clave))
	}
	canon.entero64(
		enteroFixtureConsumoC1O404D(evidencia, "catalogo_version"),
	)
	for _, clave := range []string{
		"catalogo_huella_sha256", "via_clave", "comprobacion_clave",
		"comprobacion_resultado",
	} {
		canon.texto(textoFixtureConsumoC1O404D(evidencia, clave))
	}
	canon.entero64(
		enteroFixtureConsumoC1O404D(evidencia, "orden_comprobacion"),
	)
	obligatoria, _ := evidencia["comprobacion_obligatoria"].(bool)
	canon.booleano(obligatoria)
	for _, clave := range []string{
		"procedencia_clave", "definicion_fuente_ref", "categoria_ref",
	} {
		canon.texto(textoFixtureConsumoC1O404D(evidencia, clave))
	}
	for _, clave := range []string{
		"periodo_inicio", "periodo_fin", "solicitada_en", "emitida_en",
		"valida_hasta",
	} {
		canon.instante(instanteFixtureConsumoC1O404D(evidencia, clave))
	}
	for _, clave := range []string{
		"verificador_ref", "publicador_catalogo_ref",
	} {
		canon.texto(textoFixtureConsumoC1O404D(evidencia, clave))
	}
	for _, clave := range []string{
		"peticion_canon_hex", "resultado_canon_hex",
		"atestacion_canon_hex", "confirmacion_tcb_canon_hex",
		"catalogo_canon_hex", "verificador_canon_hex", "resumen_canon_hex",
	} {
		huella := sha256.Sum256(pruebaFixtureConsumoC1O404D(evidencia, clave))
		_, _ = canon.Write(huella[:])
	}
	return canon.Bytes()
}

func canonLoteFixtureConsumoC1O404D(lote map[string]any) []byte {
	canon := &canonConsumoC1O404D{}
	canon.texto(dominioCanonLoteConsumoC1O404D)
	for _, clave := range []string{
		"lote_ref", "organizacion_ref", "expediente_ref",
	} {
		canon.texto(textoFixtureConsumoC1O404D(lote, clave))
	}
	canon.entero64(enteroFixtureConsumoC1O404D(lote, "version_expediente"))
	for _, clave := range []string{
		"reserva_ref", "preparacion_c1_ref",
		"preparacion_c1_huella_sha256",
		"huella_orden_sha256", "huella_ordenes_c1_sha256",
		"decision_vec_ref", "correlacion_vec_ref", "catalogo_ref",
	} {
		canon.texto(textoFixtureConsumoC1O404D(lote, clave))
	}
	canon.entero64(enteroFixtureConsumoC1O404D(lote, "catalogo_version"))
	canon.texto(textoFixtureConsumoC1O404D(lote, "catalogo_huella_sha256"))
	canon.instante(instanteFixtureConsumoC1O404D(lote, "efecto_en"))
	evidencias, _ := lote["evidencias"].([]any)
	canon.entero32(uint32(len(evidencias)))
	for _, cruda := range evidencias {
		evidencia, _ := cruda.(map[string]any)
		huella, _ := hex.DecodeString(
			textoFixtureConsumoC1O404D(
				evidencia,
				"evidencia_huella_sha256",
			),
		)
		_, _ = canon.Write(huella)
	}
	return canon.Bytes()
}

func nuevaEvidenciaFixtureConsumoC1O404D(
	posicion int,
	total int,
	base time.Time,
) map[string]any {
	sufijo := fmt.Sprintf("%03d", posicion)
	prueba := func(nombre string) string {
		return hex.EncodeToString([]byte(nombre + ":" + sufijo))
	}
	evidencia := map[string]any{
		"posicion":                   posicion,
		"total":                      total,
		"organizacion_ref":           "organizacion:o404d",
		"expediente_ref":             "expediente:o404d:001",
		"version_expediente":         2,
		"peticion_ref":               "peticion:o404d:" + sufijo,
		"huella_peticion_sha256":     strings.Repeat("1", 64),
		"huella_resultado_sha256":    strings.Repeat("2", 64),
		"autoridad_ref":              "autoridad:o404d:" + sufijo,
		"generacion":                 1,
		"recibo_respuesta_ref":       "respuesta:o404d:" + sufijo,
		"huella_respuesta_sha256":    strings.Repeat("3", 64),
		"catalogo_ref":               "catalogo:o404d",
		"catalogo_version":           1,
		"catalogo_huella_sha256":     strings.Repeat("4", 64),
		"via_clave":                  "bolsa_vigente",
		"comprobacion_clave":         "disponibilidad",
		"comprobacion_resultado":     "cumplida",
		"orden_comprobacion":         posicion,
		"comprobacion_obligatoria":   true,
		"procedencia_clave":          "bolsa",
		"definicion_fuente_ref":      "fuente:o404d",
		"categoria_ref":              "categoria:o404d",
		"periodo_inicio":             base.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"periodo_fin":                base.Add(48 * time.Hour).Format(time.RFC3339Nano),
		"solicitada_en":              base.Add(-time.Second).Format(time.RFC3339Nano),
		"emitida_en":                 base.Format(time.RFC3339Nano),
		"valida_hasta":               base.Add(5 * time.Second).Format(time.RFC3339Nano),
		"verificador_ref":            "verificador:o404d:" + sufijo,
		"publicador_catalogo_ref":    "publicador:o404d",
		"peticion_canon_hex":         prueba("peticion"),
		"resultado_canon_hex":        prueba("resultado"),
		"atestacion_canon_hex":       prueba("atestacion"),
		"confirmacion_tcb_canon_hex": prueba("confirmacion"),
		"catalogo_canon_hex":         prueba("catalogo"),
		"verificador_canon_hex":      prueba("verificador"),
		"resumen_canon_hex":          prueba("resumen"),
		"evidencia_huella_sha256":    strings.Repeat("f", 64),
	}
	canon := canonEvidenciaFixtureConsumoC1O404D(evidencia)
	huella := sha256.Sum256(canon)
	evidencia["evidencia_huella_sha256"] = hex.EncodeToString(huella[:])
	return evidencia
}

func nuevoLoteFixtureConsumoC1O404D(
	cantidad int,
	base time.Time,
) (map[string]any, error) {
	if cantidad < 1 || cantidad > 512 {
		return nil, errors.New("cantidad C1 fuera de contrato")
	}
	base = base.UTC().Truncate(time.Microsecond)
	evidencias := make([]any, cantidad)
	for indice := range cantidad {
		evidencias[indice] = nuevaEvidenciaFixtureConsumoC1O404D(
			indice+1,
			cantidad,
			base,
		)
	}
	lote := map[string]any{
		"esquema":                      esquemaLoteConsumoC1O404D,
		"lote_ref":                     "lote:o404d:" + strconv.Itoa(cantidad),
		"organizacion_ref":             "organizacion:o404d",
		"expediente_ref":               "expediente:o404d:001",
		"version_expediente":           2,
		"reserva_ref":                  "reserva:o404d:" + strconv.Itoa(cantidad),
		"preparacion_c1_ref":           "preparacion:o404d:" + strconv.Itoa(cantidad),
		"preparacion_c1_huella_sha256": strings.Repeat("5", 64),
		"huella_orden_sha256":          strings.Repeat("6", 64),
		"huella_ordenes_c1_sha256":     strings.Repeat("7", 64),
		"decision_vec_ref":             "decision:o404d:" + strconv.Itoa(cantidad),
		"correlacion_vec_ref":          "correlacion:o404d:" + strconv.Itoa(cantidad),
		"catalogo_ref":                 "catalogo:o404d",
		"catalogo_version":             1,
		"catalogo_huella_sha256":       strings.Repeat("4", 64),
		"efecto_en":                    base.Add(time.Millisecond).Format(time.RFC3339Nano),
		"evidencias":                   evidencias,
		"lote_huella_sha256":           strings.Repeat("e", 64),
	}
	canon := canonLoteFixtureConsumoC1O404D(lote)
	huella := sha256.Sum256(canon)
	lote["lote_huella_sha256"] = hex.EncodeToString(huella[:])
	return lote, nil
}

func huellaContextoRecursoO404D(
	rama string,
	organizacion string,
	expediente string,
	version uint64,
	reserva string,
	orden string,
	lote *string,
) string {
	canon := &canonConsumoC1O404D{}
	for _, valor := range []string{
		dominioRecursoCoberturaO404D,
		rama,
		organizacion,
		expediente,
		reserva,
	} {
		canon.texto(valor)
	}
	canon.entero64(version)
	ordenBytes, _ := hex.DecodeString(orden)
	_, _ = canon.Write(ordenBytes)
	canon.booleano(lote != nil)
	if lote != nil {
		loteBytes, _ := hex.DecodeString(*lote)
		_, _ = canon.Write(loteBytes)
	}
	huella := sha256.Sum256(canon.Bytes())
	return hex.EncodeToString(huella[:])
}

func resultadoVECFixtureO404D(
	lote map[string]any,
	mutacion string,
) map[string]any {
	decision := sha256.Sum256([]byte(
		"decision:" + textoFixtureConsumoC1O404D(lote, "lote_ref"),
	))
	if mutacion != "" {
		decision = sha256.Sum256([]byte("decision:" + mutacion))
	}
	registrada := instanteFixtureConsumoC1O404D(lote, "efecto_en")
	revalidada := registrada.Add(time.Microsecond)
	loteHuella := textoFixtureConsumoC1O404D(lote, "lote_huella_sha256")
	contexto := huellaContextoRecursoO404D(
		"concedida",
		textoFixtureConsumoC1O404D(lote, "organizacion_ref"),
		textoFixtureConsumoC1O404D(lote, "expediente_ref"),
		enteroFixtureConsumoC1O404D(lote, "version_expediente"),
		textoFixtureConsumoC1O404D(lote, "reserva_ref"),
		textoFixtureConsumoC1O404D(lote, "huella_orden_sha256"),
		&loteHuella,
	)
	resultado := map[string]any{
		"rama": "concedida", "concedida": true,
		"codigo":                         "concedida",
		"decision_ref":                   lote["decision_vec_ref"],
		"correlacion_ref":                lote["correlacion_vec_ref"],
		"organizacion_ref":               lote["organizacion_ref"],
		"expediente_ref":                 lote["expediente_ref"],
		"version_expediente":             lote["version_expediente"],
		"reserva_ref":                    lote["reserva_ref"],
		"contexto_recurso_huella_sha256": contexto,
		"decision_huella_sha256":         hex.EncodeToString(decision[:]),
		"huella_orden_sha256":            lote["huella_orden_sha256"],
		"lote_huella_sha256":             lote["lote_huella_sha256"],
		"registrada_en":                  registrada.Format(time.RFC3339Nano),
		"revalidada_en":                  revalidada.Format(time.RFC3339Nano),
	}
	canon := &canonConsumoC1O404D{}
	_, _ = canon.Write(decision[:])
	contextoBytes, _ := hex.DecodeString(contexto)
	_, _ = canon.Write(contextoBytes)
	for _, clave := range []string{
		"huella_orden_sha256", "lote_huella_sha256",
	} {
		contenido, _ := hex.DecodeString(resultado[clave].(string))
		_, _ = canon.Write(contenido)
	}
	for _, clave := range []string{"decision_ref", "correlacion_ref"} {
		canon.texto(resultado[clave].(string))
	}
	canon.instante(registrada)
	canon.instante(revalidada)
	huella := sha256.Sum256(canon.Bytes())
	resultado["prueba_vinculo_sha256"] = hex.EncodeToString(huella[:])
	return resultado
}

func clonarLoteFixtureConsumoC1O404D(
	t *testing.T,
	lote map[string]any,
) map[string]any {
	t.Helper()
	contenido, err := json.Marshal(lote)
	if err != nil {
		t.Fatal(err)
	}
	var clon map[string]any
	if err := json.Unmarshal(contenido, &clon); err != nil {
		t.Fatal(err)
	}
	return clon
}
