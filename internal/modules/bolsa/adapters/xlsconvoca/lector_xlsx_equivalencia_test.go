package xlsconvoca_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	memoria "vec-diputacion-granada/internal/modules/bolsa/adapters/memory"
	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

// El OOXML se construye en memoria desde los XLS sintéticos. No incorpora
// exportaciones reales ni depende de una herramienta ofimática externa.
func TestXLSXEquivaleALosDosXLSDeEnsayo(t *testing.T) {
	lector := xlsconvoca.NuevoLector()
	for _, nombre := range []string{"resumen.xls", "detalle.xls", "convoca_v2_resumen.xls", "convoca_v2_detalle.xls"} {
		t.Run(nombre, func(t *testing.T) {
			xls, err := lector.Decodificar(context.Background(), bytes.NewReader(leerFixture(t, nombre)))
			if err != nil {
				t.Fatalf("XLS de referencia: %v", err)
			}
			libro := construirXLSXPrueba(t, xls, opcionesXLSX{omitirVacias: true})
			xlsx, err := lector.Decodificar(context.Background(), bytes.NewReader(libro))
			if err != nil {
				t.Fatalf("XLSX sintético: %v", err)
			}
			if !reflect.DeepEqual(xlsx, xls) {
				t.Fatalf("Hoja distinta:\nXLS  %#v\nXLSX %#v", xls, xlsx)
			}
			stagingXLS, err := dominio.ValidarHoja(xls)
			if err != nil {
				t.Fatalf("staging XLS: %v", err)
			}
			stagingXLSX, err := dominio.ValidarHoja(xlsx)
			if err != nil || !reflect.DeepEqual(stagingXLSX, stagingXLS) {
				t.Fatalf("staging XLSX distinto: %#v; error=%v", stagingXLSX, err)
			}
			if nombre != "resumen.xls" {
				return
			}
			repositorio := memoria.NuevoRepositorioImportacionesConvoca()
			servicio, err := aplicacion.NuevoServicio(lector, repositorio,
				func() time.Time { return time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC) })
			if err != nil {
				t.Fatal(err)
			}
			solicitud := aplicacion.SolicitudImportacion{
				CategoriaRef: "categoria:rpt:administrativo", BolsaRef: "bolsa:administrativo:2026-09-18",
				NombreFichero: "resumen-sintetico.xlsx", FicheroCustodiadoRef: "almacen:objeto:convoca:xlsx-ensayo",
				ActorRef: "actor:rrhh:xlsx-ensayo", Contenido: libro,
			}
			primero, err := servicio.Importar(context.Background(), solicitud)
			if err != nil {
				t.Fatalf("importar XLSX: %v", err)
			}
			segundo, err := servicio.Importar(context.Background(), solicitud)
			if err != nil {
				t.Fatalf("recuperar XLSX: %v", err)
			}
			suma := sha256.Sum256(libro)
			if primero.Acta.HuellaFicheroSHA256 != hex.EncodeToString(suma[:]) || primero.Reutilizada ||
				!segundo.Reutilizada || !segundo.Acta.CoincideExactamente(primero.Acta) ||
				repositorio.NumeroLotes() != 1 || primero.Acta.FilasAceptadas != 2 || primero.Acta.FilasRechazadas != 1 {
				t.Fatalf("acta o replay inesperados: primero=%#v segundo=%#v lotes=%d", primero, segundo, repositorio.NumeroLotes())
			}
		})
	}
}

func TestXLSXRechazaPartesHostilesYLimites(t *testing.T) {
	referencia, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader(leerFixture(t, "resumen.xls")))
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre   string
		opciones opcionesXLSX
		esperado error
	}{
		{"macro", opcionesXLSX{extra: map[string]string{"xl/vbaProject.bin": "dummy"}}, xlsconvoca.ErrXLSInvalido},
		{"vinculo_externo", opcionesXLSX{extra: map[string]string{"xl/externalLinks/externalLink1.xml": `<externalLink/>`}}, xlsconvoca.ErrXLSInvalido},
		{"formula_externa", opcionesXLSX{celdas: map[string]string{"B2": `<c r="B2"><f>[1]Hoja!A1</f><v>0</v></c>`}}, xlsconvoca.ErrXLSInvalido},
		{"sharedStrings_invalida", opcionesXLSX{celdas: map[string]string{"A1": `<c r="A1" t="s"><v>999</v></c>`}}, xlsconvoca.ErrXLSInvalido},
		{"columna_33", opcionesXLSX{anexoHoja: `<row r="5"><c r="AG5" t="inlineStr"><is><t>x</t></is></c></row>`}, xlsconvoca.ErrLimiteXLSExcedido},
		{"fila_100002", opcionesXLSX{anexoHoja: `<row r="100002"><c r="A100002" t="inlineStr"><is><t>x</t></is></c></row>`}, xlsconvoca.ErrLimiteXLSExcedido},
		{"celda_64K", opcionesXLSX{celdas: map[string]string{"B2": `<c r="B2" t="inlineStr"><is><t>` + strings.Repeat("x", 64*1024+1) + `</t></is></c>`}}, xlsconvoca.ErrLimiteXLSExcedido},
		{"descomprimido_16MiB", opcionesXLSX{extra: map[string]string{"xl/extra.xml": strings.Repeat("x", 16*1024*1024+1)}}, xlsconvoca.ErrLimiteXLSExcedido},
		{"comprimido_16MiB", opcionesXLSX{extra: map[string]string{"xl/extra.xml": strings.Repeat("x", 16*1024*1024+1)}, extraSinCompresion: true}, xlsconvoca.ErrLimiteXLSExcedido},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			libro := construirXLSXPrueba(t, referencia, caso.opciones)
			if _, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader(libro)); !errors.Is(err, caso.esperado) {
				t.Fatalf("error=%v; esperado %v", err, caso.esperado)
			}
		})
	}
	if _, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader([]byte("PK\x03\x04truncado"))); !errors.Is(err, xlsconvoca.ErrXLSInvalido) {
		t.Fatalf("ZIP corrupto: %v", err)
	}
}

func TestXLSXFechaConEstiloNoSeInterpretaComoPuntos(t *testing.T) {
	referencia, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader(leerFixture(t, "resumen.xls")))
	if err != nil {
		t.Fatal(err)
	}
	libro := construirXLSXPrueba(t, referencia, opcionesXLSX{
		celdas:  map[string]string{"F2": `<c r="F2" s="1"><v>12.5</v></c>`},
		estilos: true,
	})
	hoja, err := xlsconvoca.NuevoLector().Decodificar(context.Background(), bytes.NewReader(libro))
	if err != nil {
		t.Fatalf("decodificar fecha: %v", err)
	}
	if hoja.Filas[0].Celdas[5].Tipo != dominio.CeldaFecha {
		t.Fatalf("fecha convertida en puntos: %#v", hoja.Filas[0].Celdas[5])
	}
	staging, err := dominio.ValidarHoja(hoja)
	if err != nil || !hayIncidencia(staging.Incidencias, 2, "Experiencia", "tipo_celda_invalido") {
		t.Fatalf("staging aceptó fecha como puntos: %#v; error=%v", staging, err)
	}
}

type opcionesXLSX struct {
	celdas             map[string]string
	extra              map[string]string
	anexoHoja          string
	estilos            bool
	omitirVacias       bool
	extraSinCompresion bool
}

func construirXLSXPrueba(t *testing.T, hoja dominio.HojaStaging, o opcionesXLSX) []byte {
	t.Helper()
	var datos bytes.Buffer
	archivo := zip.NewWriter(&datos)
	escribir := func(nombre, contenido string) {
		t.Helper()
		entrada, err := archivo.Create(nombre)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entrada.Write([]byte(contenido)); err != nil {
			t.Fatal(err)
		}
	}
	partes := `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>`
	if o.estilos {
		partes += `<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`
	}
	escribir("[Content_Types].xml", partes+`</Types>`)
	escribir("_rels/.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`)
	escribir("xl/workbook.xml", `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="`+escaparXMLPrueba(hoja.NombreHoja)+`" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	relaciones := `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>`
	if o.estilos {
		relaciones += `<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`
		escribir("xl/styles.xml", `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font/></fonts><fills count="1"><fill/></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf numFmtId="0"/></cellStyleXfs><cellXfs count="2"><xf numFmtId="0"/><xf numFmtId="14" applyNumberFormat="1"/></cellXfs></styleSheet>`)
	}
	escribir("xl/_rels/workbook.xml.rels", relaciones+`</Relationships>`)
	var compartidas strings.Builder
	fmt.Fprintf(&compartidas, `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, len(hoja.Cabeceras), len(hoja.Cabeceras))
	for i, cabecera := range hoja.Cabeceras {
		if i == 0 {
			compartidas.WriteString(`<si><r><t>` + escaparXMLPrueba(cabecera[:2]) + `</t></r><r><t>` + escaparXMLPrueba(cabecera[2:]) + `</t></r></si>`)
		} else {
			compartidas.WriteString(`<si><t>` + escaparXMLPrueba(cabecera) + `</t></si>`)
		}
	}
	compartidas.WriteString(`</sst>`)
	escribir("xl/sharedStrings.xml", compartidas.String())
	var filas strings.Builder
	filas.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1">`)
	for i := range hoja.Cabeceras {
		referencia := fmt.Sprintf("%c1", 'A'+i)
		if reemplazo, ok := o.celdas[referencia]; ok {
			filas.WriteString(reemplazo)
		} else {
			fmt.Fprintf(&filas, `<c r="%s" t="s"><v>%d</v></c>`, referencia, i)
		}
	}
	filas.WriteString(`</row>`)
	for _, fila := range hoja.Filas {
		fmt.Fprintf(&filas, `<row r="%d">`, fila.Numero)
		for i, celda := range fila.Celdas {
			referencia := fmt.Sprintf("%c%d", 'A'+i, fila.Numero)
			if reemplazo, ok := o.celdas[referencia]; ok {
				filas.WriteString(reemplazo)
				continue
			}
			switch celda.Tipo {
			case dominio.CeldaVacia:
				if !o.omitirVacias {
					fmt.Fprintf(&filas, `<c r="%s"/>`, referencia)
				}
			case dominio.CeldaNumero:
				fmt.Fprintf(&filas, `<c r="%s"><v>%s</v></c>`, referencia, escaparXMLPrueba(celda.Valor))
			case dominio.CeldaTexto:
				fmt.Fprintf(&filas, `<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, referencia, escaparXMLPrueba(celda.Valor))
			default:
				t.Fatalf("tipo XLS no cubierto por fixture %s: %s", referencia, celda.Tipo)
			}
		}
		filas.WriteString(`</row>`)
	}
	filas.WriteString(o.anexoHoja)
	filas.WriteString(`</sheetData></worksheet>`)
	escribir("xl/worksheets/sheet1.xml", filas.String())
	for nombre, contenido := range o.extra {
		if !o.extraSinCompresion {
			escribir(nombre, contenido)
			continue
		}
		entrada, err := archivo.CreateHeader(&zip.FileHeader{Name: nombre, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entrada.Write([]byte(contenido)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archivo.Close(); err != nil {
		t.Fatal(err)
	}
	return datos.Bytes()
}

func escaparXMLPrueba(valor string) string {
	var salida bytes.Buffer
	_ = xml.EscapeText(&salida, []byte(valor))
	return salida.String()
}
