package ginpixfichero

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPrepararExportacionConservaBytesHuellaMetadatosYContenidoParaFirma(t *testing.T) {
	carga := cargaGINPIXFicheroPrueba(t, false)
	esperado, err := Codificar(carga)
	if err != nil {
		t.Fatalf("codificar carga sintética: %v", err)
	}
	preparacion, err := PrepararExportacion(carga)
	if err != nil || preparacion.Validar() != nil {
		t.Fatalf("preparar contenido sintético: %v", err)
	}

	contenido, err := preparacion.Contenido()
	if err != nil {
		t.Fatalf("obtener contenido preparado: %v", err)
	}
	contenidoFirma, err := preparacion.ContenidoParaFirma()
	if err != nil {
		t.Fatalf("obtener contenido para firma: %v", err)
	}
	if !bytes.Equal(contenido, esperado) || !bytes.Equal(contenidoFirma, esperado) {
		t.Fatal("la preparación cambió los bytes producidos por Codificar")
	}

	huella, err := preparacion.HuellaSHA256()
	if err != nil {
		t.Fatalf("obtener huella preparada: %v", err)
	}
	suma := sha256.Sum256(esperado)
	if huella != hex.EncodeToString(suma[:]) || huella != strings.ToLower(huella) {
		t.Fatalf("huella de fichero divergente o no minúscula: %q", huella)
	}

	metadatos, err := preparacion.Metadatos()
	if err != nil {
		t.Fatalf("obtener metadatos opacos: %v", err)
	}
	datosCarga := carga.Datos()
	esperados := MetadatosPreparacionExportacion{
		VersionExpediente:    datosCarga.VersionExpediente,
		ExpedienteRef:        datosCarga.ExpedienteRef,
		IncorporacionRef:     datosCarga.IncorporacionRef,
		ProcedenciaModeloRef: datosCarga.ProcedenciaModeloRef,
		CorrelacionRef:       datosCarga.CorrelacionRef,
		IdempotenciaRef:      datosCarga.IdempotenciaRef,
		MapeoRef:             datosCarga.MapeoRef,
		MapeoVersion:         datosCarga.MapeoVersion,
		ProcedenciaMapeoRef:  datosCarga.ProcedenciaMapeoRef,
	}
	if metadatos != esperados {
		t.Fatalf("metadatos opacos divergentes: got %+v want %+v", metadatos, esperados)
	}
}

func TestPreparacionExportacionNoExponeAliasMutables(t *testing.T) {
	preparacion, err := PrepararExportacion(cargaGINPIXFicheroPrueba(t, false))
	if err != nil {
		t.Fatalf("preparar contenido sintético: %v", err)
	}
	referencia, _ := preparacion.Contenido()
	contenido, _ := preparacion.Contenido()
	contenidoFirma, _ := preparacion.ContenidoParaFirma()
	contenido[0] ^= 0xff
	contenidoFirma[len(contenidoFirma)-1] ^= 0xff

	posterior, err := preparacion.Contenido()
	if err != nil || !bytes.Equal(posterior, referencia) ||
		bytes.Equal(contenido, posterior) || bytes.Equal(contenidoFirma, posterior) {
		t.Fatal("una vista de contenido alteró la preparación inmutable")
	}
	posterior[0] ^= 0xff
	paraFirmaPosterior, err := preparacion.ContenidoParaFirma()
	if err != nil || !bytes.Equal(paraFirmaPosterior, referencia) {
		t.Fatal("contenido y contenido para firma comparten memoria mutable")
	}
}

func TestPreparacionExportacionDeniegaCeroConErrorOpaco(t *testing.T) {
	preparacion, err := PrepararExportacion(domain.CargaMapeadaGINPIX{})
	if preparacion != (PreparacionExportacion{}) ||
		!errors.Is(err, ErrPreparacionExportacionInvalida) ||
		errors.Is(err, ErrCargaGINPIXInvalida) {
		t.Fatalf("carga cero no falló de forma opaca: %+v / %v", preparacion, err)
	}

	var cero PreparacionExportacion
	if !errors.Is(cero.Validar(), ErrPreparacionExportacionInvalida) {
		t.Fatal("preparación cero aceptada")
	}
	if contenido, err := cero.Contenido(); contenido != nil ||
		!errors.Is(err, ErrPreparacionExportacionInvalida) {
		t.Fatalf("preparación cero expuso contenido: %q / %v", contenido, err)
	}
	if contenido, err := cero.ContenidoParaFirma(); contenido != nil ||
		!errors.Is(err, ErrPreparacionExportacionInvalida) {
		t.Fatalf("preparación cero expuso contenido para firma: %q / %v", contenido, err)
	}
	if huella, err := cero.HuellaSHA256(); huella != "" ||
		!errors.Is(err, ErrPreparacionExportacionInvalida) {
		t.Fatalf("preparación cero expuso huella: %q / %v", huella, err)
	}
	if metadatos, err := cero.Metadatos(); metadatos != (MetadatosPreparacionExportacion{}) ||
		!errors.Is(err, ErrPreparacionExportacionInvalida) {
		t.Fatalf("preparación cero expuso metadatos: %+v / %v", metadatos, err)
	}
}

func TestPreparacionExportacionAplicaLimiteAntesDeCopiar(t *testing.T) {
	datos := cargaGINPIXFicheroPrueba(t, false).Datos()
	metadatos := MetadatosPreparacionExportacion{
		VersionExpediente:    datos.VersionExpediente,
		ExpedienteRef:        datos.ExpedienteRef,
		IncorporacionRef:     datos.IncorporacionRef,
		ProcedenciaModeloRef: datos.ProcedenciaModeloRef,
		CorrelacionRef:       datos.CorrelacionRef,
		IdempotenciaRef:      datos.IdempotenciaRef,
		MapeoRef:             datos.MapeoRef,
		MapeoVersion:         datos.MapeoVersion,
		ProcedenciaMapeoRef:  datos.ProcedenciaMapeoRef,
	}
	contenidoExcesivo := make([]byte, MaximoBytesFicheroGINPIX+1)
	asignaciones := testing.AllocsPerRun(100, func() {
		if _, err := prepararContenidoExportacion(contenidoExcesivo, metadatos); !errors.Is(
			err,
			ErrPreparacionExportacionInvalida,
		) {
			panic("contenido fuera de límite aceptado")
		}
	})
	if asignaciones != 0 {
		t.Fatalf("el límite reservó memoria adicional: %.2f", asignaciones)
	}
}
