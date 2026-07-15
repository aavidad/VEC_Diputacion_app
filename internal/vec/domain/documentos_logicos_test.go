package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	huellaPlantillaPrueba = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	huellaDOCXPrueba      = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	huellaPDFPrueba       = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	huellaFirmaPrueba     = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	huellaDatosPrueba     = "hmac-sha256:documentos-1:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	huellaFuentePrueba    = "hmac-sha256:documentos-1:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
)

func documentoLogicoValidoPrueba() DocumentoLogico {
	fecha := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	return DocumentoLogico{
		ID:       "documento-001",
		Version:  1,
		Revision: 1,
		Plantilla: ReferenciaPlantillaDocumento{
			ID:           "contrato_bolsa",
			Version:      7,
			HuellaSHA256: huellaPlantillaPrueba,
		},
		ModuloID:       "bolsa",
		TipoDocumental: "contrato",
		Clasificacion:  "datos_personales_alta",
		Relaciones: []RelacionDocumento{
			{Tipo: TipoRelacionPersona, Referencia: "persona-001", Rol: "interesada"},
			{Tipo: TipoRelacionExpediente, Referencia: "expediente-042", Rol: "principal"},
		},
		Estado:           EstadoDocumentoLogicoBorrador,
		HuellaDatosHMAC:  huellaDatosPrueba,
		HuellaFuenteHMAC: huellaFuentePrueba,
		CreadoPor:        "tecnico-rrhh-1",
		CreadoEn:         fecha,
		CorrelacionRef:   "correlacion-001",
		Motivo:           "Generacion tras un llamamiento aceptado",
		ENI: MetadatosENI{
			Identificador:     "documento-001",
			Organo:            "ORGANO-PRUEBA",
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    "contrato",
			FechaCaptura:      fecha,
		},
	}
}

func representacionValidaPrueba(id string, tipo TipoRepresentacionDocumento, formato FormatoDocumento, huella string) RepresentacionDocumento {
	fecha := time.Date(2026, time.July, 14, 12, 1, 0, 0, time.UTC)
	return RepresentacionDocumento{
		ID:                    id,
		Documento:             ReferenciaDocumento{ID: "documento-001", Version: 1},
		Tipo:                  tipo,
		Formato:               formato,
		MIME:                  formato.MIME(),
		NombreFichero:         id + formato.Extension(),
		Tamano:                1_024,
		HuellaContenidoSHA256: huella,
		HuellaFuenteHMAC:      huellaFuentePrueba,
		ReferenciaContenido:   "objeto:sha256:" + huella,
		EstadoTecnico:         EstadoRepresentacionDisponible,
		EstadoAntivirus:       EstadoAntivirusNoAplica,
		GeneradaPor:           "tecnico-rrhh-1",
		GeneradaEn:            fecha,
	}
}

func TestCanonizarRelacionesDocumentoAdmiteTiposExtensiblesYRechazaDuplicados(t *testing.T) {
	relaciones := []RelacionDocumento{
		{Tipo: TipoRelacionDocumento("dietas.comision"), Referencia: "comision-9", Rol: "origen"},
		{Tipo: TipoRelacionPersona, Referencia: "persona-1", Rol: "interesada"},
		{Tipo: TipoRelacionExpediente, Referencia: "expediente-1", Rol: "principal"},
	}
	canonicas, err := CanonizarRelacionesDocumento(relaciones)
	if err != nil {
		t.Fatalf("CanonizarRelacionesDocumento() error = %v", err)
	}
	if len(canonicas) != 3 || canonicas[0].Tipo != TipoRelacionDocumento("dietas.comision") ||
		canonicas[1].Tipo != TipoRelacionExpediente || canonicas[2].Tipo != TipoRelacionPersona {
		t.Fatalf("orden canonico inesperado: %+v", canonicas)
	}
	canonicas[0].Referencia = "alterada"
	if relaciones[0].Referencia != "comision-9" {
		t.Fatal("la canonizacion comparte memoria con la entrada")
	}

	duplicadas := append(relaciones, relaciones[1])
	_, err = CanonizarRelacionesDocumento(duplicadas)
	if !errors.Is(err, ErrRelacionDocumentoDuplicada) {
		t.Fatalf("relacion duplicada: error = %v", err)
	}
	if strings.Contains(err.Error(), "persona-1") {
		t.Fatalf("el error revela una referencia de entidad: %v", err)
	}
}

func TestValidarRequisitosRelacionesDocumentoCompruebaCardinalidades(t *testing.T) {
	relaciones := []RelacionDocumento{
		{Tipo: TipoRelacionExpediente, Referencia: "expediente-1", Rol: "principal"},
		{Tipo: TipoRelacionPersona, Referencia: "persona-1", Rol: "interesada"},
		{Tipo: TipoRelacionPersona, Referencia: "persona-2", Rol: "representante"},
	}
	requisitos := []RequisitoRelacionDocumento{
		{Tipo: TipoRelacionExpediente, Rol: "principal", Minimo: 1, Maximo: 1},
		{Tipo: TipoRelacionPersona, Minimo: 1, Maximo: 2},
	}
	if err := ValidarRequisitosRelacionesDocumento(relaciones, requisitos); err != nil {
		t.Fatalf("ValidarRequisitosRelacionesDocumento() error = %v", err)
	}

	requisitos[1].Minimo = 3
	requisitos[1].Maximo = 3
	if err := ValidarRequisitosRelacionesDocumento(relaciones, requisitos); !errors.Is(err, ErrRequisitoRelacionDocumentoIncumplido) {
		t.Fatalf("cardinalidad incumplida: error = %v", err)
	}

	repetidos := []RequisitoRelacionDocumento{
		{Tipo: TipoRelacionPersona, Minimo: 1},
		{Tipo: TipoRelacionPersona, Minimo: 1},
	}
	if err := ValidarRequisitosRelacionesDocumento(relaciones, repetidos); !errors.Is(err, ErrRequisitoRelacionDocumentoInvalido) {
		t.Fatalf("requisito duplicado: error = %v", err)
	}
}

func TestDocumentoLogicoVersionaContenidoYClonaRelaciones(t *testing.T) {
	documento := documentoLogicoValidoPrueba()
	if err := documento.Validar(); err != nil {
		t.Fatalf("DocumentoLogico.Validar() error = %v", err)
	}
	canonico, err := documento.ClonarCanonico()
	if err != nil {
		t.Fatalf("ClonarCanonico() error = %v", err)
	}
	if canonico.Relaciones[0].Tipo != TipoRelacionExpediente {
		t.Fatalf("relaciones no canonicas: %+v", canonico.Relaciones)
	}
	canonico.Relaciones[0].Referencia = "alterada"
	if documento.Relaciones[1].Referencia != "expediente-042" {
		t.Fatal("el clon comparte relaciones con el documento original")
	}

	versionDos := documento
	versionDos.Version = 2
	versionDos.VersionAnterior = &ReferenciaDocumento{ID: documento.ID, Version: 1}
	if err := versionDos.Validar(); err != nil {
		t.Fatalf("version dos valida: error = %v", err)
	}
	versionDos.VersionAnterior = &ReferenciaDocumento{ID: documento.ID, Version: 2}
	if err := versionDos.Validar(); !errors.Is(err, ErrDocumentoLogicoInvalido) {
		t.Fatalf("salto de version: error = %v", err)
	}
}

func TestDocumentoLogicoExigeClasificacionCanonica(t *testing.T) {
	documento := documentoLogicoValidoPrueba()
	documento.Clasificacion = ""
	if err := documento.Validar(); !errors.Is(err, ErrDocumentoLogicoInvalido) {
		t.Fatalf("clasificacion vacia: error = %v", err)
	}
	documento.Clasificacion = "Datos personales alta"
	if err := documento.Validar(); !errors.Is(err, ErrDocumentoLogicoInvalido) {
		t.Fatalf("clasificacion no canonica: error = %v", err)
	}
}

func TestResultadoMantieneElEstadoLogicoIndependienteDelFormato(t *testing.T) {
	documento := documentoLogicoValidoPrueba()
	documento.Estado = EstadoDocumentoLogicoEnRevision
	docx := representacionValidaPrueba("representacion-docx", TipoRepresentacionTrabajo, FormatoDocumentoDOCX, huellaDOCXPrueba)
	pdf := representacionValidaPrueba("representacion-pdf", TipoRepresentacionVisualizacion, FormatoDocumentoPDF, huellaPDFPrueba)

	casos := []struct {
		nombre           string
		representaciones []RepresentacionDocumento
	}{
		{nombre: "solo DOCX", representaciones: []RepresentacionDocumento{docx}},
		{nombre: "solo PDF", representaciones: []RepresentacionDocumento{pdf}},
		{nombre: "DOCX y PDF", representaciones: []RepresentacionDocumento{docx, pdf}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado := ResultadoGeneracionDocumento{Documento: documento, Representaciones: caso.representaciones}
			if err := resultado.Validar(); err != nil {
				t.Fatalf("ResultadoGeneracionDocumento.Validar() error = %v", err)
			}
			if resultado.Documento.Estado != EstadoDocumentoLogicoEnRevision {
				t.Fatalf("el formato altero el estado: %s", resultado.Documento.Estado)
			}
		})
	}
	if docx.HuellaFuenteHMAC != pdf.HuellaFuenteHMAC || docx.HuellaContenidoSHA256 == pdf.HuellaContenidoSHA256 {
		t.Fatalf("huellas comunes o por representacion incorrectas: docx=%+v pdf=%+v", docx, pdf)
	}
}

func TestRepresentacionExigeHuellaValidaYPertenencia(t *testing.T) {
	documento := documentoLogicoValidoPrueba()
	representacion := representacionValidaPrueba("representacion-pdf", TipoRepresentacionVisualizacion, FormatoDocumentoPDF, huellaPDFPrueba)
	if err := representacion.ValidarPertenencia(documento); err != nil {
		t.Fatalf("ValidarPertenencia() error = %v", err)
	}

	invalida := representacion
	invalida.HuellaContenidoSHA256 = strings.ToUpper(huellaPDFPrueba)
	if err := invalida.Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("SHA-256 no canonica: error = %v", err)
	}
	invalida = representacion
	invalida.HuellaFuenteHMAC = "hmac-sha256:documentos-1:corta"
	if err := invalida.Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("HMAC invalida: error = %v", err)
	}
	invalida = representacion
	invalida.HuellaFuenteHMAC = "hmac-sha256:documentos-1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := invalida.ValidarPertenencia(documento); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("fuente distinta: error = %v", err)
	}
}

func TestRepresentacionFirmadaDebeDerivarDeBytesExactos(t *testing.T) {
	firmada := representacionValidaPrueba("representacion-firmada", TipoRepresentacionFirma, FormatoDocumentoPDF, huellaFirmaPrueba)
	if err := firmada.Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("firma sin origen: error = %v", err)
	}
	firmada.DerivadaDeRef = "representacion-pdf"
	if err := firmada.Validar(); err != nil {
		t.Fatalf("firma derivada valida: error = %v", err)
	}
	firmada.DerivadaDeRef = firmada.ID
	if err := firmada.Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("autoderivacion: error = %v", err)
	}
}

func TestSolicitudesRepresentacionSeCanonizanYSinDuplicados(t *testing.T) {
	solicitudes := []SolicitudRepresentacionDocumento{
		{Tipo: TipoRepresentacionVisualizacion, Formato: FormatoDocumentoPDF},
		{Tipo: TipoRepresentacionTrabajo, Formato: FormatoDocumentoDOCX},
	}
	canonicas, err := CanonizarSolicitudesRepresentacionDocumento(solicitudes)
	if err != nil {
		t.Fatalf("CanonizarSolicitudesRepresentacionDocumento() error = %v", err)
	}
	if canonicas[0].Tipo != TipoRepresentacionTrabajo || canonicas[1].Tipo != TipoRepresentacionVisualizacion {
		t.Fatalf("orden inesperado: %+v", canonicas)
	}
	_, err = CanonizarSolicitudesRepresentacionDocumento(append(solicitudes, solicitudes[0]))
	if !errors.Is(err, ErrSolicitudRepresentacionDuplicada) {
		t.Fatalf("solicitud duplicada: error = %v", err)
	}
	if err := (SolicitudRepresentacionDocumento{Tipo: TipoRepresentacionFirma, Formato: FormatoDocumentoPDF}).Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("una firma debe derivar de otra representacion: error = %v", err)
	}
}

func TestResultadoRechazaRepresentacionDuplicada(t *testing.T) {
	documento := documentoLogicoValidoPrueba()
	pdf := representacionValidaPrueba("representacion-pdf", TipoRepresentacionVisualizacion, FormatoDocumentoPDF, huellaPDFPrueba)
	resultado := ResultadoGeneracionDocumento{Documento: documento, Representaciones: []RepresentacionDocumento{pdf, pdf}}
	if err := resultado.Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("representacion duplicada: error = %v", err)
	}

	segundoPDF := pdf
	segundoPDF.ID = "representacion-pdf-2"
	segundoPDF.NombreFichero = "representacion-pdf-2.pdf"
	resultado.Representaciones = []RepresentacionDocumento{pdf, segundoPDF}
	if err := resultado.Validar(); !errors.Is(err, ErrRepresentacionDocumentoInvalida) {
		t.Fatalf("tipo y formato duplicados: error = %v", err)
	}
}
