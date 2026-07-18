package importacionconvoca

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestDetectarEsquemasConvocaExigeCabecerasLiterales(t *testing.T) {
	for _, esquema := range []EsquemaExportacion{EsquemaResumenPersona, EsquemaDetalleMerito} {
		detectado, err := DetectarEsquema(esquema.Cabeceras())
		if err != nil || detectado != esquema {
			t.Fatalf("detectar %s: detectado=%s error=%v", esquema, detectado, err)
		}
	}
	alteradas := EsquemaResumenPersona.Cabeceras()
	alteradas[6] = "Formación"
	if _, err := DetectarEsquema(alteradas); !errors.Is(err, ErrEsquemaExportacionDesconocido) {
		t.Fatalf("cabecera corregida silenciosamente: %v", err)
	}
	reordenadas := EsquemaDetalleMerito.Cabeceras()
	reordenadas[9], reordenadas[10] = reordenadas[10], reordenadas[9]
	if _, err := DetectarEsquema(reordenadas); !errors.Is(err, ErrEsquemaExportacionDesconocido) {
		t.Fatalf("orden de columnas ignorado: %v", err)
	}
	if _, err := DetectarEsquema(append(EsquemaResumenPersona.Cabeceras(), "Extra")); !errors.Is(err, ErrEsquemaExportacionDesconocido) {
		t.Fatalf("columna adicional ignorada: %v", err)
	}
}

func TestValidarResumenSeparaAceptadasYRechazadasSinFiltrarDatosAlActa(t *testing.T) {
	hoja := HojaStaging{
		Esquema: EsquemaResumenPersona, Cabeceras: EsquemaResumenPersona.Cabeceras(),
		Filas: []FilaStaging{
			filaResumen(2, "***0001**", "Sintetica", "Uno", "Ana", "Libre", "12,50", "2.5", "15"),
			filaResumen(3, "***0002**", "Prueba", "Total", "Bruno", "Libre", "2", "2", "99"),
			filaResumen(4, "12345678Z", "Prueba", "Documento", "Carla", "Libre", "1", "1", "2"),
			{
				Numero: 5,
				Celdas: []CeldaStaging{
					texto("***0003**"), texto("Prueba"), texto("Formula"),
					{Tipo: CeldaFormula, Valor: "Daniel"}, texto("Libre"),
					numero("1"), numero("1"), numero("2"),
				},
			},
		},
	}
	resultado, err := ValidarHoja(hoja)
	if err != nil {
		t.Fatalf("validar resumen: %v", err)
	}
	if resultado.FilasLeidas != 4 || len(resultado.Aceptadas) != 1 || resultado.Rechazadas != 3 {
		t.Fatalf("conteos inesperados: %#v", resultado)
	}
	aceptada := resultado.Aceptadas[0]
	if aceptada.Resumen == nil || aceptada.Detalle != nil ||
		aceptada.Resumen.Experiencia != "12.5" || aceptada.Resumen.Formacion != "2.5" ||
		aceptada.Resumen.Total != "15" || aceptada.Identidad.Documento != "***0001**" {
		t.Fatalf("normalizacion inesperada: %#v", aceptada)
	}
	if !contieneIncidencia(resultado.Incidencias, 3, "Total", codigoTotalIncoherente) ||
		!contieneIncidencia(resultado.Incidencias, 4, "DNI/NIE", codigoDocumentoEnmascarado) ||
		!contieneIncidencia(resultado.Incidencias, 5, "Nombre", codigoFormulaProhibida) {
		t.Fatalf("motivos incompletos: %#v", resultado.Incidencias)
	}
	actaTextual := fmt.Sprint(resultado.Incidencias)
	for _, datoPersonal := range []string{"Sintetica", "Bruno", "12345678Z", "Daniel"} {
		if contiene(actaTextual, datoPersonal) {
			t.Fatalf("incidencia filtra dato de fila %q: %s", datoPersonal, actaTextual)
		}
	}
}

func TestValidarDetalleConservaAutobaremoSoloComoDatoHistorico(t *testing.T) {
	hoja := HojaStaging{
		Esquema: EsquemaDetalleMerito, Cabeceras: EsquemaDetalleMerito.Cabeceras(),
		Filas: []FilaStaging{
			filaDetalle(2, "***0001**", "1", "0,5000", "", ""),
			filaDetalle(3, "***0002**", "0", "1", "1", "orden imposible"),
			filaDetalleConFormula(4),
		},
	}
	resultado, err := ValidarHoja(hoja)
	if err != nil {
		t.Fatalf("validar detalle: %v", err)
	}
	if resultado.FilasLeidas != 3 || len(resultado.Aceptadas) != 1 || resultado.Rechazadas != 2 {
		t.Fatalf("conteos inesperados: %#v", resultado)
	}
	detalle := resultado.Aceptadas[0].Detalle
	if detalle == nil || detalle.OrdenGrupo != 1 ||
		detalle.PuntosAutobaremacionHistoricos != "0.5" || detalle.PuntosTribunal != "" {
		t.Fatalf("detalle inesperado: %#v", detalle)
	}
	if !contieneIncidencia(resultado.Incidencias, 3, "Orden grupo", codigoOrdenGrupoInvalido) ||
		!contieneIncidencia(resultado.Incidencias, 4, "Descripcion del merito", codigoFormulaProhibida) {
		t.Fatalf("rechazos inesperados: %#v", resultado.Incidencias)
	}
}

func TestLoteFijaProcedenciaNoAutoritativaYClonaEnProfundidad(t *testing.T) {
	huella := repetir("a", 64)
	lote := LoteValidado{
		Acta: ActaImportacion{
			ActaRef:             "acta:importacion-convoca:" + huella,
			ImportacionRef:      "importacion:convoca:" + huella,
			HuellaFicheroSHA256: huella, NombreFichero: "sintetico.xls",
			ActorRef: "actor:rrhh:prueba", RegistradaEn: time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC),
			Esquema: EsquemaResumenPersona, FilasLeidas: 1, FilasAceptadas: 1,
			Procedencia: NuevaProcedenciaNoAutoritativa(),
		},
		Aceptadas: []FilaAceptada{{
			Numero: 2, Esquema: EsquemaResumenPersona,
			Identidad: IdentidadEnmascarada{Documento: "***0001**", PrimerApellido: "Sintetica", Nombre: "Ana"},
			Turno:     "Libre", Resumen: &ResumenPersona{Experiencia: "1", Formacion: "1", Total: "2"},
		}},
	}
	if err := lote.Validar(); err != nil {
		t.Fatalf("lote valido rechazado: %v", err)
	}
	copia := ClonarLote(lote)
	copia.Aceptadas[0].Resumen.Total = "999"
	if lote.Aceptadas[0].Resumen.Total != "2" {
		t.Fatal("la copia defensiva comparte el resumen")
	}
	alterado := lote
	alterado.Acta.Procedencia.HabilitaActosConEfectos = true
	if err := alterado.Validar(); !errors.Is(err, ErrLoteImportacionInvalido) {
		t.Fatalf("procedencia autoritativa aceptada: %v", err)
	}
}

func filaResumen(numeroFila int, documento, primero, segundo, nombre, turno, experiencia, formacion, total string) FilaStaging {
	return FilaStaging{Numero: numeroFila, Celdas: []CeldaStaging{
		texto(documento), texto(primero), texto(segundo), texto(nombre), texto(turno),
		numero(experiencia), numero(formacion), numero(total),
	}}
}

func filaDetalle(numeroFila int, documento, orden, auto, tribunal, motivo string) FilaStaging {
	return FilaStaging{Numero: numeroFila, Celdas: []CeldaStaging{
		texto(documento), texto("Sintetica"), texto("Uno"), texto("Ana"), texto("Libre"),
		texto("EXP"), texto("Experiencia profesional"), numero(orden),
		texto("Servicio sintetico"), numero(auto), numero(tribunal), texto(motivo),
	}}
}

func filaDetalleConFormula(numeroFila int) FilaStaging {
	fila := filaDetalle(numeroFila, "***0003**", "1", "1", "1", "")
	fila.Celdas[8] = CeldaStaging{Tipo: CeldaFormula, Valor: "Servicio calculado"}
	return fila
}

func texto(valor string) CeldaStaging {
	if valor == "" {
		return CeldaStaging{Tipo: CeldaVacia}
	}
	return CeldaStaging{Tipo: CeldaTexto, Valor: valor}
}

func numero(valor string) CeldaStaging {
	if valor == "" {
		return CeldaStaging{Tipo: CeldaVacia}
	}
	return CeldaStaging{Tipo: CeldaNumero, Valor: valor}
}

func contieneIncidencia(incidencias []Incidencia, fila int, campo, codigo string) bool {
	for _, actual := range incidencias {
		if actual.Fila == fila && actual.Campo == campo && actual.Codigo == codigo {
			return true
		}
	}
	return false
}

func contiene(texto, fragmento string) bool {
	for i := 0; i+len(fragmento) <= len(texto); i++ {
		if texto[i:i+len(fragmento)] == fragmento {
			return true
		}
	}
	return false
}

func repetir(valor string, veces int) string {
	resultado := ""
	for i := 0; i < veces; i++ {
		resultado += valor
	}
	return resultado
}
