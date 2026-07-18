package almacen

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func resultadoOperacionPrueba() DatosResultadoOperacionObjeto {
	instante := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	referencia := ReferenciaObjetoAlmacen{Referencia: "objeto:1", Version: "version:1"}
	objeto := ObjetoAlmacenado{
		Objeto: referencia, ConectorID: "almacen_s3_corporativo", Zona: ZonaCuarentena,
		MIME: "application/pdf", Tamano: 1024, HuellaSHA256: strings.Repeat("a", 64),
		EvidenciaCreacionRef: "evidencia:creacion:1", AlmacenadoEn: instante,
	}
	evidencia := DatosEvidenciaOperacionAlmacen{
		Referencia: objeto.EvidenciaCreacionRef, ConectorID: objeto.ConectorID,
		EsquemaContexto: "vec.almacen.contexto.v1", EsquemaEsperado: "vec.almacen.contexto.v1",
		AccionNegocio: "bolsa.documento.custodiar", Accion: AccionEscribir,
		EfectoRef: "efecto:1", HuellaPlanEfectoSHA256: strings.Repeat("b", 64),
		PasoRef: "01_escribir", HuellaDecisionSHA256: strings.Repeat("c", 64), Objeto: referencia,
		OperacionRef: "operacion:1", CorrelacionRef: "correlacion:1", AutorizacionRef: "autorizacion:1",
		Finalidad: "tramitar", Clasificacion: "confidencial", RealizadaEn: instante,
		CargaRef: "carga:1", SujetoSeudonimoHMAC: hmacReciboPrueba("d"),
		RecursoRef: "recurso:1", ModuloID: "bolsa", HuellaSolicitudHMAC: hmacReciboPrueba("e"),
	}
	return DatosResultadoOperacionObjeto{Objeto: objeto, Evidencia: evidencia}
}

func TestEvidenciaYResultadoExigenCreacionExacta(t *testing.T) {
	t.Parallel()

	resultado := resultadoOperacionPrueba()
	if err := ValidarResultadoOperacion(resultado); err != nil {
		t.Fatalf("resultado valido rechazado: %v", err)
	}
	alterado := resultado
	alterado.Evidencia.Referencia = "evidencia:otra"
	if !errors.Is(ValidarResultadoOperacion(alterado), ErrSolicitudAlmacenInvalida) {
		t.Fatal("una creacion acepto otra evidencia")
	}
	alterado = resultado
	alterado.Evidencia.RealizadaEn = resultado.Objeto.AlmacenadoEn.Add(time.Nanosecond)
	if !errors.Is(ValidarResultadoOperacion(alterado), ErrSolicitudAlmacenInvalida) {
		t.Fatal("una creacion acepto otra fecha")
	}
	alterado = resultado
	alterado.Evidencia.Accion = AccionLeer
	if !errors.Is(ValidarResultadoOperacion(alterado), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto una accion sin resultado material")
	}
}

func TestEvidenciaDocumentalExigeParejaDeHuellas(t *testing.T) {
	t.Parallel()

	evidencia := resultadoOperacionPrueba().Evidencia
	evidencia.HuellaManifiestoSHA256 = strings.Repeat("f", 64)
	if !errors.Is(ValidarEvidenciaOperacion(evidencia), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto solo una huella documental")
	}
	evidencia.HuellaPasoSHA256 = strings.Repeat("0", 64)
	if err := ValidarEvidenciaOperacion(evidencia); err != nil {
		t.Fatalf("se rechazo la pareja documental: %v", err)
	}
	evidencia.AccionNegocio = "bolsa.*"
	if !errors.Is(ValidarEvidenciaOperacion(evidencia), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto un comodin en la accion")
	}
}

func TestResultadoMutacionConservaObjetoYExigeLigadura(t *testing.T) {
	t.Parallel()

	creacion := resultadoOperacionPrueba()
	resultado := creacion
	resultado.Evidencia.Referencia = "evidencia:retencion:1"
	resultado.Evidencia.Accion = AccionAplicarRetencion
	resultado.Evidencia.FundamentoRef = "politica:retencion:1"
	resultado.Evidencia.RealizadaEn = creacion.Objeto.AlmacenadoEn.Add(time.Minute)
	resultado.Objeto.RetenidoHasta = creacion.Objeto.AlmacenadoEn.Add(24 * time.Hour)
	if err := ValidarResultadoMutacion(
		resultado, creacion.Objeto, AccionAplicarRetencion, resultado.Evidencia.FundamentoRef, true,
	); err != nil {
		t.Fatalf("mutacion valida rechazada: %v", err)
	}
	if !errors.Is(
		ValidarResultadoMutacion(
			resultado, creacion.Objeto, AccionAplicarRetencion, resultado.Evidencia.FundamentoRef, false,
		),
		ErrSolicitudAlmacenInvalida,
	) {
		t.Fatal("se acepto una evidencia no ligada")
	}
	resultado.Objeto.MIME = "text/plain"
	if !errors.Is(
		ValidarResultadoMutacion(
			resultado, creacion.Objeto, AccionAplicarRetencion, resultado.Evidencia.FundamentoRef, true,
		),
		ErrSolicitudAlmacenInvalida,
	) {
		t.Fatal("se acepto sustituir el objeto durante una mutacion")
	}
}

func TestObjetoRechazaEstadosMaterialesIncoherentes(t *testing.T) {
	t.Parallel()

	objeto := resultadoOperacionPrueba().Objeto
	if err := objeto.Validar(); err != nil {
		t.Fatal(err)
	}
	objeto.Eliminado, objeto.Inmovilizado = true, true
	if !errors.Is(objeto.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto un objeto eliminado e inmovilizado")
	}
	objeto = resultadoOperacionPrueba().Objeto
	objeto.RetenidoHasta = objeto.AlmacenadoEn
	if !errors.Is(objeto.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto retencion sin intervalo positivo")
	}
}
