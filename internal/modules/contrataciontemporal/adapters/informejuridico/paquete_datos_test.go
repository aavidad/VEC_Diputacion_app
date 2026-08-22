package informejuridico

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestGenerarPaqueteDatosFijaJSONOrdenYHuellasExactas(t *testing.T) {
	borradorOrdenado := nuevoBorradorPrueba(t, false)
	borradorInvertido := nuevoBorradorPrueba(t, true)

	paquete, err := GenerarPaqueteDatos(borradorOrdenado)
	if err != nil {
		t.Fatalf("generar paquete: %v", err)
	}
	paqueteInvertido, err := GenerarPaqueteDatos(borradorInvertido)
	if err != nil {
		t.Fatalf("generar paquete reordenado: %v", err)
	}
	contenido := contenidoPaquetePrueba(t, paquete)
	contenidoInvertido := contenidoPaquetePrueba(t, paqueteInvertido)
	if !bytes.Equal(contenido, contenidoInvertido) {
		t.Fatal("el orden de entrada altero los bytes del paquete")
	}

	esperado := `{"esquema":"vec.dipgra.contratacion-temporal.informe-juridico.paquete-datos",` +
		`"version_esquema":1,` +
		`"expediente_ref":"expediente:contratacion-temporal:01",` +
		`"version_expediente":7,` +
		`"plantilla":{"plantilla_ref":"plantilla:informe-juridico:rrhh","version":3,` +
		`"huella_sha256":"` + strings.Repeat("a", 64) + `"},` +
		`"referencias_normativas":[` +
		`{"norma_ref":"norma:empleo-publico:trebep","version":5,"huella_sha256":"` +
		strings.Repeat("b", 64) + `"},` +
		`{"norma_ref":"norma:procedimiento:ley-39-2015","version":2,"huella_sha256":"` +
		strings.Repeat("c", 64) + `"}],` +
		`"anexos":[` +
		`{"documento_ref":"documento:analisis-rrhh:01","version_documento":11,` +
		`"huella_sha256":"` + strings.Repeat("d", 64) + `"},` +
		`{"documento_ref":"documento:cobertura:01","version_documento":7,` +
		`"huella_sha256":"` + strings.Repeat("e", 64) + `"}],` +
		`"huella_borrador_sha256":"4cb476f1a8b82c0622e41413d1675e4324e4f120d7815cc1eb1ab1346e70d7c6"}`
	if string(contenido) != esperado {
		t.Fatalf("JSON canonico divergente:\n got: %s\nwant: %s", contenido, esperado)
	}

	suma := sha256.Sum256(contenido)
	huellaCalculada := hex.EncodeToString(suma[:])
	huellaPaquete, err := paquete.HuellaSHA256()
	if err != nil {
		t.Fatalf("leer huella del paquete: %v", err)
	}
	const huellaEsperada = "b4863d0e1ae9524bd3f68987395603d6d55a015ed799a0bf83eebe81d34dc586"
	if huellaCalculada != huellaEsperada || huellaPaquete != huellaEsperada {
		t.Fatalf(
			"SHA-256 del paquete divergente: calculada=%s paquete=%s",
			huellaCalculada,
			huellaPaquete,
		)
	}
	if !utf8.Valid(contenido) || !json.Valid(contenido) || bytes.HasSuffix(contenido, []byte("\n")) {
		t.Fatal("el contenido no es JSON UTF-8 cerrado sin sufijo")
	}
}

func TestGenerarPaqueteDatosCierraSuperficieYNoAfirmaInforme(t *testing.T) {
	paquete, err := GenerarPaqueteDatos(nuevoBorradorPrueba(t, false))
	if err != nil {
		t.Fatalf("generar paquete: %v", err)
	}
	contenido := contenidoPaquetePrueba(t, paquete)

	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &raiz); err != nil {
		t.Fatalf("decodificar superficie: %v", err)
	}
	exigirClavesExactas(t, raiz,
		"esquema", "version_esquema", "expediente_ref", "version_expediente",
		"plantilla", "referencias_normativas", "anexos", "huella_borrador_sha256",
	)
	var plantilla map[string]json.RawMessage
	if err := json.Unmarshal(raiz["plantilla"], &plantilla); err != nil {
		t.Fatalf("decodificar plantilla: %v", err)
	}
	exigirClavesExactas(t, plantilla, "plantilla_ref", "version", "huella_sha256")

	var normativas []map[string]json.RawMessage
	if err := json.Unmarshal(raiz["referencias_normativas"], &normativas); err != nil {
		t.Fatalf("decodificar normativas: %v", err)
	}
	for _, normativa := range normativas {
		exigirClavesExactas(t, normativa, "norma_ref", "version", "huella_sha256")
	}
	var anexos []map[string]json.RawMessage
	if err := json.Unmarshal(raiz["anexos"], &anexos); err != nil {
		t.Fatalf("decodificar anexos: %v", err)
	}
	for _, anexo := range anexos {
		exigirClavesExactas(t, anexo, "documento_ref", "version_documento", "huella_sha256")
	}

	for _, campoProhibido := range []string{
		"texto_juridico", "conclusiones", "firma", "aprobacion", "publicacion",
		"identidad", "pii", "url", "contenido_anexo", "informe_existe", "estado_informe",
	} {
		if _, existe := raiz[campoProhibido]; existe ||
			bytes.Contains(contenido, []byte(`"`+campoProhibido+`"`)) {
			t.Fatalf("el paquete expuso el campo prohibido %q", campoProhibido)
		}
	}
}

func TestCodificacionJSONFijaEscapeUnicodeYCamposVacios(t *testing.T) {
	contenido, err := codificarJSONCanonico(paqueteDatosJSON{
		Esquema: "esquema:jurídico\u2028cerrado", VersionEsquema: 1,
		ExpedienteRef: "expediente:unicode", VersionExpediente: 1,
		Plantilla: plantillaJSON{
			PlantillaRef: "plantilla:unicode", Version: 1,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		ReferenciasNormativas: []referenciaNormativaJSON{},
		Anexos:                []anexoJSON{}, HuellaBorradorSHA256: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatalf("codificar muestra Unicode: %v", err)
	}
	if !utf8.Valid(contenido) || !bytes.Contains(contenido, []byte("jurídico\\u2028cerrado")) ||
		!bytes.Contains(contenido, []byte(`"referencias_normativas":[]`)) ||
		!bytes.Contains(contenido, []byte(`"anexos":[]`)) {
		t.Fatalf("politica Unicode o campos cerrados divergentes: %s", contenido)
	}
}

func TestGenerarPaqueteDatosRechazaInvalidoAdulteradoYCopiaBuffers(t *testing.T) {
	if paquete, err := GenerarPaqueteDatos(domain.BorradorInformeJuridico{}); paquete.datos != nil ||
		!errors.Is(err, ErrPaqueteDatosInformeJuridicoInvalido) {
		t.Fatalf("borrador cero aceptado: %+v / %v", paquete, err)
	}
	if contenido, err := (PaqueteDatosInformeJuridico{}).ContenidoJSON(); contenido != nil ||
		!errors.Is(err, ErrPaqueteDatosInformeJuridicoInvalido) {
		t.Fatalf("paquete cero aceptado: %q / %v", contenido, err)
	}

	borrador := nuevoBorradorPrueba(t, false)
	estadoAdulterado := borrador.Estado()
	estadoAdulterado.Anexos[0].VersionDocumento++
	if _, err := domain.RestaurarBorradorInformeJuridico(estadoAdulterado); !errors.Is(
		err,
		domain.ErrBorradorInformeJuridicoInvalido,
	) {
		t.Fatalf("se restauro un borrador adulterado: %v", err)
	}

	paquete, err := GenerarPaqueteDatos(borrador)
	if err != nil {
		t.Fatalf("generar paquete: %v", err)
	}
	primera := contenidoPaquetePrueba(t, paquete)
	referencia := append([]byte(nil), primera...)
	primera[0] ^= 0xff
	segunda := contenidoPaquetePrueba(t, paquete)
	if !bytes.Equal(segunda, referencia) {
		t.Fatal("la salida comparte el buffer privado del paquete")
	}

	estadoExterno := borrador.Estado()
	estadoExterno.ReferenciasNormativas[0].NormaRef = "norma:alterada"
	tercera := contenidoPaquetePrueba(t, paquete)
	if !bytes.Equal(tercera, referencia) {
		t.Fatal("el paquete retuvo alias del borrador o de su snapshot")
	}
}

func TestPresupuestoPaqueteDatosLimitaAntesDeReservarOCodificar(t *testing.T) {
	estadoCardinalidad := domain.EstadoBorradorInformeJuridico{
		DatosBorradorInformeJuridico: domain.DatosBorradorInformeJuridico{
			ReferenciasNormativas: make(
				[]domain.ReferenciaNormativaInformeJuridico,
				maximoReferenciasNormativasPaquete+1,
			),
		},
	}
	asignaciones := testing.AllocsPerRun(1_000, func() {
		if _, valido := presupuestoPaqueteDatos(estadoCardinalidad); valido {
			panic("cardinalidad fuera de limite aceptada")
		}
	})
	if asignaciones != 0 {
		t.Fatalf("la guardia de cardinalidad reservo memoria: %.0f", asignaciones)
	}

	estadoTamano := domain.EstadoBorradorInformeJuridico{
		DatosBorradorInformeJuridico: domain.DatosBorradorInformeJuridico{
			ExpedienteRef:         strings.Repeat("x", maximoBytesReferenciasPaquete+1),
			ReferenciasNormativas: []domain.ReferenciaNormativaInformeJuridico{{}},
		},
	}
	asignaciones = testing.AllocsPerRun(1_000, func() {
		if _, valido := presupuestoPaqueteDatos(estadoTamano); valido {
			panic("tamano fuera de limite aceptado")
		}
	})
	if asignaciones != 0 {
		t.Fatalf("la guardia de tamano reservo memoria: %.0f", asignaciones)
	}

	borrador := nuevoBorradorCardinalidadMaxima(t)
	paquete, err := GenerarPaqueteDatos(borrador)
	if err != nil {
		t.Fatalf("generar cardinalidad maxima valida: %v", err)
	}
	contenido := contenidoPaquetePrueba(t, paquete)
	if len(contenido) > MaximoBytesPaqueteDatosInformeJuridico {
		t.Fatalf("el paquete excedio el limite: %d", len(contenido))
	}
}

func nuevoBorradorPrueba(t *testing.T, invertir bool) domain.BorradorInformeJuridico {
	t.Helper()
	datos := domain.DatosBorradorInformeJuridico{
		Canon:                     domain.CanonBorradorInformeJuridicoV1(),
		ExpedienteRef:             "expediente:contratacion-temporal:01",
		VersionEsperadaExpediente: 7,
		Plantilla: domain.ReferenciaPlantillaInformeJuridico{
			PlantillaRef: "plantilla:informe-juridico:rrhh", Version: 3,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		ReferenciasNormativas: []domain.ReferenciaNormativaInformeJuridico{
			{NormaRef: "norma:empleo-publico:trebep", Version: 5, HuellaSHA256: strings.Repeat("b", 64)},
			{NormaRef: "norma:procedimiento:ley-39-2015", Version: 2, HuellaSHA256: strings.Repeat("c", 64)},
		},
		Anexos: []domain.AnexoDocumentalInformeJuridico{
			{DocumentoRef: "documento:analisis-rrhh:01", VersionDocumento: 11, HuellaSHA256: strings.Repeat("d", 64)},
			{DocumentoRef: "documento:cobertura:01", VersionDocumento: 7, HuellaSHA256: strings.Repeat("e", 64)},
		},
	}
	if invertir {
		datos.ReferenciasNormativas[0], datos.ReferenciasNormativas[1] =
			datos.ReferenciasNormativas[1], datos.ReferenciasNormativas[0]
		datos.Anexos[0], datos.Anexos[1] = datos.Anexos[1], datos.Anexos[0]
	}
	borrador, err := domain.NuevoBorradorInformeJuridico(datos)
	if err != nil {
		t.Fatalf("construir borrador: %v", err)
	}
	return borrador
}

func nuevoBorradorCardinalidadMaxima(t *testing.T) domain.BorradorInformeJuridico {
	t.Helper()
	datos := nuevoBorradorPrueba(t, false).Estado().DatosBorradorInformeJuridico
	datos.ReferenciasNormativas = make(
		[]domain.ReferenciaNormativaInformeJuridico,
		maximoReferenciasNormativasPaquete,
	)
	datos.Anexos = make([]domain.AnexoDocumentalInformeJuridico, maximoAnexosPaquete)
	for indice := range datos.ReferenciasNormativas {
		datos.ReferenciasNormativas[indice] = domain.ReferenciaNormativaInformeJuridico{
			NormaRef: fmt.Sprintf("norma:paquete:%02d", indice), Version: uint64(indice + 1),
			HuellaSHA256: fmt.Sprintf("%064x", indice+1),
		}
	}
	for indice := range datos.Anexos {
		datos.Anexos[indice] = domain.AnexoDocumentalInformeJuridico{
			DocumentoRef:     fmt.Sprintf("documento:paquete:%02d", indice),
			VersionDocumento: uint64(indice + 1), HuellaSHA256: fmt.Sprintf("%064x", indice+101),
		}
	}
	borrador, err := domain.NuevoBorradorInformeJuridico(datos)
	if err != nil {
		t.Fatalf("construir borrador de cardinalidad maxima: %v", err)
	}
	return borrador
}

func contenidoPaquetePrueba(t *testing.T, paquete PaqueteDatosInformeJuridico) []byte {
	t.Helper()
	contenido, err := paquete.ContenidoJSON()
	if err != nil {
		t.Fatalf("leer contenido JSON: %v", err)
	}
	return contenido
}

func exigirClavesExactas(t *testing.T, valores map[string]json.RawMessage, claves ...string) {
	t.Helper()
	if len(valores) != len(claves) {
		t.Fatalf("numero de campos inesperado: got=%d want=%d (%v)", len(valores), len(claves), valores)
	}
	for _, clave := range claves {
		if _, existe := valores[clave]; !existe {
			t.Fatalf("falta el campo cerrado %q", clave)
		}
	}
}
