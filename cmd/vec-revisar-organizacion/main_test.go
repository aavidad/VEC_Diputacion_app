package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/domain"

	"vec-diputacion-granada/web"
)

func ejemploRevision(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("ejemplo.sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func revisarCLI(t *testing.T, args []string, datos []byte) (int, salida) {
	t.Helper()
	var output bytes.Buffer
	codigo := ejecutar(args, bytes.NewReader(datos), &output)
	var s salida
	if err := json.Unmarshal(output.Bytes(), &s); err != nil {
		t.Fatalf("salida inválida: %v", err)
	}
	return codigo, s
}

func TestRevisionCLIFormatoExistenteYCatalogos(t *testing.T) {
	datos := ejemploRevision(t)
	codigo, es := revisarCLI(t, nil, datos)
	codigoEN, en := revisarCLI(t, []string{"--idioma", "en"}, datos)
	if codigo != 0 || codigoEN != 0 || !es.Informe.Valido || es.Informe.PaqueteHuellaSHA256 != en.Informe.PaqueteHuellaSHA256 {
		t.Fatal("revisión válida o huella por idioma")
	}
	if es.Mensajes["alcance"] == en.Mensajes["alcance"] || es.Mensajes["huella"] == "" {
		t.Fatal("traducción ausente")
	}
	if es.Informe.CoberturaConciliacion == nil || !reflect.DeepEqual(es.Informe.CoberturaConciliacion, en.Informe.CoberturaConciliacion) ||
		es.Informe.CoberturaConciliacion.Completa || es.Informe.CoberturaConciliacion.Hechos[0].Resultado != "sin_decision" ||
		es.Informe.PaqueteHuellaSHA256 != "900a156e5ea86103a52ff065a9b56d716553257f2f6c6fae02e2b62b525815b6" {
		t.Fatal("cobertura ausente o huella del ejemplo alterada")
	}
	catalogo, mensajes, err := web.CatalogoRevisionOrganizacion()
	if err != nil {
		t.Fatal(err)
	}
	for idioma, tabla := range mensajes {
		if len(tabla) != len(es.Mensajes) {
			t.Fatal("catálogos distintos")
		}
		for key, value := range tabla {
			if value == "" || catalogo.T(idioma, key) != value || es.Mensajes[key] == "" {
				t.Fatalf("clave %s", key)
			}
		}
	}
	for _, clave := range es.Informe.PendientesPublicacion {
		if es.Mensajes[clave] == "" {
			t.Fatalf("pendiente sin traducción: %s", clave)
		}
	}
}

func TestRevisionCLIRechazaAmbiguedad(t *testing.T) {
	datos := string(ejemploRevision(t))
	casos := map[string]string{
		"clave duplicada":          strings.Replace(datos, `"tipo": "rpt"`, `"tipo": "rpt", "tipo": "plantilla"`, 1),
		"clave escapada duplicada": strings.Replace(datos, `"tipo": "rpt"`, `"tipo": "rpt", "t\u0069po": "plantilla"`, 1),
		"clave desconocida":        strings.Replace(datos, `"tipo": "rpt"`, `"tipo": "rpt", "actor": "inventado"`, 1),
		"mayúsculas":               strings.Replace(datos, `"manifiesto"`, `"MANIFIESTO"`, 1),
		"anidada mayúsculas":       strings.Replace(datos, `"revision": 1`, `"Revision": 1`, 1),
		"nulo":                     strings.Replace(datos, `"vigente_desde": "2026-01-01"`, `"vigente_desde": null`, 1),
		"hechos nulos":             `{"manifiesto":{},"hechos":null}`,
		"tipo incorrecto":          strings.Replace(datos, `"revision": 1`, `"revision": "1"`, 1),
		"múltiples documentos":     datos + datos,
		"raíz array":               `[]`,
		"raíz nula":                `null`,
		"malformado":               `{`,
		"utf8 inválido":            datos + string([]byte{0xff}),
	}
	for nombre, entrada := range casos {
		t.Run(nombre, func(t *testing.T) {
			codigo, s := revisarCLI(t, nil, []byte(entrada))
			if codigo != 1 || s.Informe.Valido || s.Informe.ClaveError != "entrada_invalida" || s.Informe.PaqueteHuellaSHA256 != "" {
				t.Fatalf("informe: %+v", s.Informe)
			}
		})
	}
}

func TestRevisionCLIFilaYDecisiones(t *testing.T) {
	datos := ejemploRevision(t)
	var p map[string]json.RawMessage
	if err := json.Unmarshal(datos, &p); err != nil {
		t.Fatal(err)
	}
	p["decisiones"] = json.RawMessage(`[{"fila_fuente_ref":"fila:1","clase":"unidad","resultado":"pendiente","motivo":"Correspondencia sin acreditar","evidencia_ref":"evidencia:sintetica"}]`)
	conDecisiones, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if codigo, s := revisarCLI(t, nil, conDecisiones); codigo != 0 || s.Informe.Decisiones != 1 {
		t.Fatalf("decisiones: %+v", s.Informe)
	}
	invalido := bytes.Replace(datos, []byte(`"vigente_desde": "2026-01-01"`), []byte(`"vigente_desde": "2026-02-30"`), 1)
	codigo, s := revisarCLI(t, nil, invalido)
	if codigo != 1 || s.Informe.FilaFallida != 1 || s.Informe.ClaveError != "hecho_invalido" || s.Mensajes[s.Informe.ClaveError] == "" {
		t.Fatalf("fila: %+v", s.Informe)
	}
}

func TestComprobarCompletoCLIConservaPreparacionNoAutoritativa(t *testing.T) {
	datos := ejemploRevision(t)
	codigo, incompleto := revisarCLI(t, []string{"--comprobar-completo", "--preparar"}, datos)
	if codigo != 1 || incompleto.Paquete != nil || incompleto.Comprobacion == nil || incompleto.Comprobacion.Completa ||
		incompleto.Informe.Estado != "preparacion_no_autoritativa" || incompleto.Informe.PaqueteHuellaSHA256 == "" {
		t.Fatalf("preparación incompleta: %+v", incompleto)
	}
	for _, falta := range incompleto.Comprobacion.Faltantes {
		if incompleto.Mensajes[falta.Clave] == "" || incompleto.Mensajes[falta.Esperado] == "" || incompleto.Mensajes[falta.Actual] == "" {
			t.Fatalf("código sin texto: %+v", falta)
		}
	}
	var p application.PaquetePreparacionOrganizacion
	if err := json.Unmarshal(datos, &p); err != nil {
		t.Fatal(err)
	}
	p.Manifiesto.DocumentoRef, p.Manifiesto.CustodiaRef = "documento:sintetico", "custodia:sintetica"
	p.Manifiesto.DiccionarioRef, p.Manifiesto.ActoRef = "diccionario:sintetico", "acto:sintetico"
	p.Manifiesto.AprobadaEn, p.Manifiesto.PublicadaEn, p.Manifiesto.EfectosDesde = "2026-01-01", "2026-01-02", "2026-01-03"
	p.Decisiones = []domain.DecisionConciliacionOrganizacion{{FilaFuenteRef: "fila:1", Clase: "unidad", Resultado: "vinculada", DestinoRef: "unidad:sintetica", Motivo: "Correspondencia declarada", EvidenciaRef: "evidencia:sintetica"}}
	completo, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	codigo, preparado := revisarCLI(t, []string{"--preparar", "--comprobar-completo", "--idioma", "en"}, completo)
	if codigo != 0 || preparado.Paquete == nil || preparado.Comprobacion == nil || !preparado.Comprobacion.Completa ||
		len(preparado.Comprobacion.Faltantes) != 0 || preparado.Informe.Estado != "preparacion_no_autoritativa" ||
		!slices.Contains(preparado.Informe.PendientesPublicacion, "acreditacion_fuente") {
		t.Fatalf("preparación completa: %+v", preparado)
	}
	if codigo, sinExportar := revisarCLI(t, []string{"--comprobar-completo"}, completo); codigo != 0 || sinExportar.Paquete != nil {
		t.Fatal("comprobación sin exportación alterada")
	}
	if codigo, duplicada := revisarCLI(t, []string{"--comprobar-completo", "--comprobar-completo"}, completo); codigo != 1 || duplicada.Paquete != nil || duplicada.Informe.ClaveError != "argumentos_invalidos" {
		t.Fatal("opción duplicada admitida")
	}
}

func TestRevisionCLILimitesArgumentosYSinEfectos(t *testing.T) {
	datos := ejemploRevision(t)
	copia := bytes.Clone(datos)
	for _, args := range [][]string{{"--idioma"}, {"--idioma", "xx"}, {"--idioma", "en", "otro"}, {"fichero.json"}} {
		codigo, s := revisarCLI(t, args, datos)
		if codigo != 1 || s.Informe.ClaveError != "argumentos_invalidos" {
			t.Fatal("argumentos admitidos")
		}
	}
	padded := append(bytes.Clone(datos), bytes.Repeat([]byte(" "), limiteEntrada-len(datos))...)
	if codigo, _ := revisarCLI(t, nil, padded); codigo != 0 {
		t.Fatal("límite exacto rechazado")
	}
	if codigo, s := revisarCLI(t, nil, append(padded, ' ')); codigo != 1 || s.Informe.ClaveError != "entrada_invalida" {
		t.Fatal("exceso admitido")
	}
	if !reflect.DeepEqual(datos, copia) {
		t.Fatal("entrada modificada")
	}
	if codigo := ejecutar(nil, lectorFallido{}, io.Discard); codigo != 1 {
		t.Fatal("fallo de lectura")
	}
	if codigo := ejecutar(nil, bytes.NewReader(datos), escritorFallido{}); codigo != 2 {
		t.Fatal("fallo de escritura")
	}
}

type lectorFallido struct{}

func (lectorFallido) Read([]byte) (int, error) { return 0, errors.New("lectura") }

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) { return 0, errors.New("escritura") }

func TestPrepararCLIExportaPaqueteCompatibleYHuella(t *testing.T) {
	var p application.PaquetePreparacionOrganizacion
	if err := json.Unmarshal(ejemploRevision(t), &p); err != nil {
		t.Fatal(err)
	}
	p.Hechos = append(p.Hechos, p.Hechos[0])
	p.Hechos[0].HechoRef = "ffffffff-ffff-4fff-8fff-ffffffffffff"
	p.Hechos[0].FilaFuenteRef = "fila:2"
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	codigo, es := revisarCLI(t, []string{"--preparar"}, data)
	codigoEN, en := revisarCLI(t, []string{"--idioma", "en", "--preparar"}, data)
	if codigo != 0 || codigoEN != 0 || es.Paquete == nil || en.Paquete == nil {
		t.Fatal("paquete no exportado")
	}
	if es.Informe.CoberturaConciliacion == nil || !reflect.DeepEqual(es.Informe.CoberturaConciliacion, en.Informe.CoberturaConciliacion) ||
		es.Informe.CoberturaConciliacion.RecuentosHechos["sin_decision"] != 2 {
		t.Fatal("exportación no conserva cobertura por hecho")
	}
	if es.Paquete.Hechos[0].HechoRef != p.Hechos[1].HechoRef || !reflect.DeepEqual(es.Paquete, en.Paquete) {
		t.Fatal("orden o idioma cambió el paquete")
	}
	canonico, err := json.Marshal(es.Paquete)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(canonico)
	if hex.EncodeToString(huella[:]) != es.Informe.PaqueteHuellaSHA256 {
		t.Fatal("huella no identifica material exportado")
	}
	// La salida se puede extraer y alimentar al lector estricto ya existente.
	var output bytes.Buffer
	if codigo := ejecutar(nil, bytes.NewReader(canonico), &output); codigo != 0 {
		t.Fatal("material exportado no admitido")
	}
	for _, pendiente := range []string{"acreditacion_fuente", "publicacion_autorizada", "aprobacion_separada"} {
		if !slices.Contains(es.Informe.PendientesPublicacion, pendiente) {
			t.Fatal("deuda borrada")
		}
	}
	_, sinFlag := revisarCLI(t, nil, data)
	if sinFlag.Paquete != nil {
		t.Fatal("salida original alterada")
	}
	_, invertido := revisarCLI(t, []string{"--preparar", "--idioma", "en"}, data)
	if !reflect.DeepEqual(en, invertido) {
		t.Fatal("orden de argumentos alteró salida")
	}
}

func TestPrepararCLINoExportaEntradasInvalidas(t *testing.T) {
	datos := ejemploRevision(t)
	invalido := bytes.Replace(datos, []byte(`"vigente_desde": "2026-01-01"`), []byte(`"vigente_desde": "2026-02-30"`), 1)
	for _, caso := range []struct {
		args []string
		data []byte
	}{
		{[]string{"--preparar"}, invalido},
		{[]string{"--preparar"}, []byte(`{`)},
		{[]string{"--preparar", "--preparar"}, datos},
		{[]string{"--preparar", "--idioma", "en", "--idioma", "es"}, datos},
		{[]string{"--preparar", "--idioma"}, datos},
		{[]string{"--preparar", "--idioma", "xx"}, datos},
	} {
		var output bytes.Buffer
		if codigo := ejecutar(caso.args, bytes.NewReader(caso.data), &output); codigo != 1 {
			t.Fatal("fallo aceptado")
		}
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(output.Bytes(), &campos); err != nil {
			t.Fatal(err)
		}
		if _, ok := campos["paquete"]; ok {
			t.Fatal("material inválido exportado")
		}
		var s salida
		if err := json.Unmarshal(output.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		if s.Informe.PaqueteHuellaSHA256 != "" || s.Informe.Valido {
			t.Fatal("huella inválida exportada")
		}
		if s.Informe.CoberturaConciliacion != nil {
			t.Fatal("material inválido tiene cobertura")
		}
	}
}

func TestPrepararCLIRechazaExpansionQueExcedeRelectura(t *testing.T) {
	var p application.PaquetePreparacionOrganizacion
	if err := json.Unmarshal(ejemploRevision(t), &p); err != nil {
		t.Fatal(err)
	}
	h := p.Hechos[0]
	p.Hechos = nil
	for i := 0; i < 1000; i++ {
		h.HechoRef = fmt.Sprintf("%08x-2222-4222-8222-222222222222", i)
		h.FilaFuenteRef = fmt.Sprintf("fila:%d", i)
		p.Hechos = append(p.Hechos, h)
		p.Decisiones = append(p.Decisiones, domain.DecisionConciliacionOrganizacion{
			FilaFuenteRef: h.FilaFuenteRef, Clase: "unidad", Resultado: "pendiente",
			Motivo: strings.Repeat("<", 2048), EvidenciaRef: "evidencia:sintetica",
		})
	}
	var entrada bytes.Buffer
	encoder := json.NewEncoder(&entrada)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(p); err != nil {
		t.Fatal(err)
	}
	canonico, err := json.Marshal(p)
	if err != nil || entrada.Len() > limiteEntrada || len(canonico) <= limiteEntrada {
		t.Fatal("fixture no reproduce expansión del JSON")
	}
	codigo, revision := revisarCLI(t, nil, entrada.Bytes())
	if codigo != 0 || !revision.Informe.Valido || revision.Informe.PaqueteHuellaSHA256 == "" || revision.BytesPaquete != 0 || revision.LimiteBytesPaquete != 0 {
		t.Fatal("revisión existente alterada")
	}
	for _, args := range [][]string{{"--preparar"}, {"--preparar", "--idioma", "en"}} {
		var output bytes.Buffer
		if codigo := ejecutar(args, bytes.NewReader(entrada.Bytes()), &output); codigo != 1 {
			t.Fatal("exportación expansiva aceptada")
		}
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(output.Bytes(), &campos); err != nil {
			t.Fatal(err)
		}
		if _, ok := campos["paquete"]; ok {
			t.Fatal("paquete excesivo exportado")
		}
		var s salida
		if err := json.Unmarshal(output.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		if s.Informe.Valido || s.Informe.ClaveError != "paquete_excede_limite" || s.Informe.Seccion != "paquete" ||
			s.Informe.PaqueteHuellaSHA256 != "" || s.Informe.ManifiestoHuellaSHA256 != "" || s.Mensajes[s.Informe.ClaveError] == "" {
			t.Fatal("fallo de tamaño o traducción no comunicado")
		}
		if s.BytesPaquete != len(canonico) || s.LimiteBytesPaquete != limiteEntrada {
			t.Fatal("tamaño real o límite no comunicados")
		}
		if s.Informe.CoberturaConciliacion != nil || bytes.Contains(output.Bytes(), []byte("fila:999")) {
			t.Fatal("rechazo por tamaño devolvió cobertura de entrada")
		}
		if bytes.Contains(output.Bytes(), []byte("evidencia:sintetica")) || bytes.Contains(output.Bytes(), []byte(strings.Repeat("<", 2048))) {
			t.Fatal("fallo devolvió datos de entrada")
		}
	}
	p.Manifiesto.DocumentoRef, p.Manifiesto.CustodiaRef = "documento:sintetico", "custodia:sintetica"
	p.Manifiesto.DiccionarioRef, p.Manifiesto.ActoRef = "diccionario:sintetico", "acto:sintetico"
	p.Manifiesto.AprobadaEn, p.Manifiesto.PublicadaEn, p.Manifiesto.EfectosDesde = "2026-01-01", "2026-01-02", "2026-01-03"
	for i := range p.Decisiones {
		p.Decisiones[i].Resultado, p.Decisiones[i].DestinoRef = "vinculada", "unidad:sintetica"
	}
	entrada.Reset()
	if err := encoder.Encode(p); err != nil {
		t.Fatal(err)
	}
	if codigo, s := revisarCLI(t, []string{"--comprobar-completo", "--preparar"}, entrada.Bytes()); codigo != 1 || s.Paquete != nil || s.Comprobacion == nil || s.Comprobacion.Completa ||
		s.Informe.Valido || s.Informe.ClaveError != "paquete_excede_limite" ||
		!slices.ContainsFunc(s.Comprobacion.Faltantes, func(f application.FaltanteCompletitudOrganizacion) bool { return f.Clave == "paquete_excede_limite" }) {
		t.Fatalf("tamaño incompatible con preparación completa: %+v", s)
	}
}
