package documental

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSobrePruebaDespachoV3CopiaYDetectaAlteraciones(t *testing.T) {
	t.Parallel()

	mensaje := []byte("mensaje canonico de despacho")
	firma := append([]byte{1}, make([]byte, TamanoFirmaHMACSHA256V3-1)...)
	sobre, err := NuevoSobrePruebaAtestacionDespachoV3(
		AlgoritmoHMACSHA256V3, AudienciaInicioEfectoV3, ContextoInicioEfectoV3,
		"clave:atestacion", 3, "evidencia:inicio", mensaje, firma,
	)
	if err != nil {
		t.Fatal(err)
	}
	mensaje[0] ^= 0xff
	firma[0] ^= 0xff
	primera, _ := sobre.MensajeCanonico()
	primera[0] ^= 0xff
	segunda, _ := sobre.MensajeCanonico()
	if string(segunda) != "mensaje canonico de despacho" {
		t.Fatal("el sobre no defendio las copias del mensaje")
	}
	datos, err := sobre.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(datos); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON datos: %v", err)
	}
	if texto := fmt.Sprintf("%v|%+v|%#v", datos, datos, datos); strings.Contains(texto, "mensaje canonico") {
		t.Fatal("el formateo del DTO filtro el mensaje")
	}
	datos.MensajeCanonico[0] ^= 0xff
	if RestaurarSobrePruebaAtestacionDespachoV3(datos).Validar() == nil {
		t.Fatal("se acepto un mensaje distinto de su huella")
	}
	if _, err := json.Marshal(sobre); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if texto := fmt.Sprintf("%v|%+v|%#v", sobre, sobre, sobre); strings.Contains(texto, "mensaje canonico") {
		t.Fatal("el formateo filtro el mensaje")
	}
}

func TestSelloEvidenciaV3MantienePerfilYFirmaOpacos(t *testing.T) {
	t.Parallel()

	perfil, err := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:sello")
	if err != nil {
		t.Fatal(err)
	}
	huella := strings.Repeat("a", 64)
	firma := append([]byte{1}, make([]byte, TamanoFirmaHMACSHA256V3-1)...)
	firmadoEn := time.Date(2026, 7, 18, 12, 0, 0, 123_456_000, time.UTC)
	sello, err := NuevoSelloEvidenciaV3(perfil, huella, firma, "evidencia:sello", firmadoEn)
	if err != nil {
		t.Fatal(err)
	}
	firma[0] ^= 0xff
	datos, err := sello.Datos()
	if err != nil || datos.Firma[0] != 1 {
		t.Fatalf("no se copio defensivamente la firma: %v", err)
	}
	huellaSolicitud := HuellaSolicitudVerificacionEvidenciaV3(huella, datos)
	huellaEsperada := HuellaCamposSHA256V3([]string{
		"vec.documentos.solicitud-verificacion-evidencia.v3", huella,
		datos.Algoritmo, datos.ClaveID, datos.Audiencia,
		HuellaBytesSHA256(datos.Firma), datos.EvidenciaOperacionRef,
		datos.FirmadoEn.Format(time.RFC3339Nano),
	})
	if huellaSolicitud != huellaEsperada {
		t.Fatal("cambio la preimagen de verificacion del sello")
	}
	datos.Firma[0] ^= 0xff
	segunda, _ := sello.Datos()
	if segunda.Firma[0] != 1 {
		t.Fatal("Datos expuso la firma interna")
	}
	alterado := segunda
	alterado.Firma = make([]byte, TamanoFirmaHMACSHA256V3)
	if RestaurarSelloEvidenciaV3(alterado).ValidarPara(perfil, huella) == nil {
		t.Fatal("se acepto una firma enteramente nula")
	}
	if (SelloEvidenciaV3{}).EsCero() != true || sello.EsCero() {
		t.Fatal("la deteccion del valor cero cambio")
	}
	if _, err := json.Marshal(perfil); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON perfil: %v", err)
	}
	if _, err := json.Marshal(sello); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON sello: %v", err)
	}
}

func resultadoReconciliacionV3Prueba(
	t *testing.T,
) (DatosResultadoReconciliacionV3, ExpectativasResultadoReconciliacionV3) {
	t.Helper()

	sobre, err := NuevoSobreAtestacionReconciliacionV3(
		[]byte("cose-sign1-reconciliacion-v3"),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaSobre, _ := sobre.HuellaSHA256()
	datos := DatosResultadoReconciliacionV3{
		ReservaRef: "reserva:uno", EfectoRef: "efecto:uno", SecuenciaCercado: 7,
		HuellaVinculoSHA256: strings.Repeat("a", 64), HuellaPlanSHA256: strings.Repeat("b", 64),
		Estado: ResultadoReconciliacionV3AplicadoExacto,
		Resultado: DatosResultadoRenderizadoV3{
			BorradorRef: "borrador:uno", EfectoRef: "efecto:uno",
			ContenidoRef: "contenido:uno", ContenidoVersion: "version:uno",
			ConectorRef: "conector:uno", MIME: "application/pdf",
			HuellaSalidaSHA256: strings.Repeat("c", 64), TamanoSalida: 1024,
			EvidenciaOperacionRef: "evidencia:salida",
		},
		AtestacionRef: "atestacion:uno", HuellaAtestacionSHA256: huellaSobre,
		SobreAtestacion: sobre,
		ConsultadaEn:    time.Date(2026, 7, 18, 12, 1, 0, 123_456_000, time.UTC),
	}
	esperado := ExpectativasResultadoReconciliacionV3{
		ReservaRef: datos.ReservaRef, EfectoRef: datos.EfectoRef,
		SecuenciaCercado:    datos.SecuenciaCercado,
		HuellaVinculoSHA256: datos.HuellaVinculoSHA256,
		HuellaPlanSHA256:    datos.HuellaPlanSHA256, ResultadoAplicadoValido: true,
	}
	return datos, esperado
}

func TestReconciliacionV3ConservaEstadoPreimagenYLigadura(t *testing.T) {
	t.Parallel()

	datos, esperado := resultadoReconciliacionV3Prueba(t)
	if err := datos.ValidarContra(esperado); err != nil {
		t.Fatalf("se rechazo resultado valido: %v", err)
	}
	preimagenEsperada := SerializarCamposV3([]string{
		"vec.documentos.resultado-reconciliacion.v3", datos.ReservaRef, datos.EfectoRef,
		"7", datos.HuellaVinculoSHA256, datos.HuellaPlanSHA256, "aplicado_exacto",
		datos.Resultado.BorradorRef, datos.Resultado.EfectoRef, datos.Resultado.ContenidoRef,
		datos.Resultado.ContenidoVersion, datos.Resultado.ConectorRef, datos.Resultado.MIME,
		datos.Resultado.HuellaSalidaSHA256, "1024", datos.Resultado.EvidenciaOperacionRef,
		datos.AtestacionRef, datos.ConsultadaEn.Format(time.RFC3339Nano),
	})
	if !BytesIguales(datos.Bytes(), preimagenEsperada) {
		t.Fatal("cambio el orden historico de la preimagen de reconciliacion")
	}
	huellaSolicitud := HuellaSolicitudVerificacionReconciliacionV3(
		datos.Bytes(), datos.HuellaAtestacionSHA256,
	)
	huellaEsperada := HuellaCamposSHA256V3([]string{
		"vec.documentos.solicitud-verificacion-reconciliacion.v3",
		HuellaBytesSHA256(datos.Bytes()), datos.HuellaAtestacionSHA256,
	})
	if huellaSolicitud != huellaEsperada {
		t.Fatal("cambio la ligadura de la solicitud de verificacion")
	}

	noAplicado := datos
	noAplicado.Estado = ResultadoReconciliacionV3NoAplicado
	if noAplicado.ValidarContra(esperado) == nil {
		t.Fatal("un resultado no aplicado con salida se acepto")
	}
	noAplicado.Resultado = DatosResultadoRenderizadoV3{}
	if err := noAplicado.ValidarContra(esperado); err != nil {
		t.Fatalf("se rechazo no aplicado atestado sin salida: %v", err)
	}
	for _, estado := range []EstadoResultadoReconciliacionV3{
		ResultadoReconciliacionV3AplicadoExacto, ResultadoReconciliacionV3NoAplicado,
		ResultadoReconciliacionV3Desconocido, ResultadoReconciliacionV3Conflictivo,
	} {
		if !estado.Valido() {
			t.Fatalf("se rechazo estado cerrado %q", estado)
		}
	}
	if EstadoResultadoReconciliacionV3("otro").Valido() {
		t.Fatal("se acepto un estado abierto")
	}
}

func TestSobreReconciliacionV3EsOpacoYCopiaDefensivamente(t *testing.T) {
	t.Parallel()

	original := []byte("cose-sign1-reconciliacion-v3")
	sobre, err := NuevoSobreAtestacionReconciliacionV3(original)
	if err != nil {
		t.Fatal(err)
	}
	original[0] ^= 0xff
	primera, _ := sobre.COSESign1()
	primera[0] ^= 0xff
	segunda, _ := sobre.COSESign1()
	if string(segunda) != "cose-sign1-reconciliacion-v3" {
		t.Fatal("el sobre COSE no realizo copias defensivas")
	}
	huella, _ := sobre.HuellaSHA256()
	alterado := RestaurarSobreAtestacionReconciliacionV3([]byte("otro-sobre-reconciliacion-v3"), huella)
	if alterado.Validar() == nil {
		t.Fatal("se acepto un COSE distinto de su huella")
	}
	if _, err := json.Marshal(sobre); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if texto := fmt.Sprintf("%v|%+v|%#v", sobre, sobre, sobre); strings.Contains(texto, "cose-sign1") {
		t.Fatal("el formateo filtro el COSE")
	}
}
