package almacen

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func hmacReciboPrueba(letra string) string {
	return "hmac-sha256:v1:" + strings.Repeat(letra, 64)
}

func resultadoConsumoReciboPrueba() ResultadoConsumoReciboCargaDirecta {
	registrado := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	return ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: hmacReciboPrueba("a"), GrupoHMAC: hmacReciboPrueba("b"),
		VinculoHMAC: hmacReciboPrueba("c"), EvidenciaConsumoRef: "evidencia:consumo:1",
		IntencionConfirmacionRef: "intencion:confirmacion:1", HuellaIntencionHMAC: hmacReciboPrueba("d"),
		RegistradoEn: registrado, ConsumidoEn: registrado.Add(time.Minute), ExpiraEn: registrado.Add(5 * time.Minute),
	}
}

func TestReciboEsOpacoYNoSerializable(t *testing.T) {
	t.Parallel()

	secreto := "recibo:mac:v1:0123456789abcdefghijkl"
	recibo, err := NuevoReciboCargaDirecta(secreto)
	if err != nil || !recibo.Valido() {
		t.Fatalf("crear recibo: %v", err)
	}
	revelado, err := recibo.RevelarParaEntregaOConsumo()
	if err != nil || revelado != secreto {
		t.Fatalf("revelar recibo: %q, %v", revelado, err)
	}
	if recibo.String() == secreto || recibo.GoString() == secreto {
		t.Fatal("la representacion textual revelo el secreto")
	}
	if _, err := json.Marshal(recibo); !errors.Is(err, ErrSerializacionReciboCargaProhibida) {
		t.Fatalf("serializacion no denegada: %v", err)
	}
	if _, err := NuevoReciboCargaDirecta("recibo con espacios"); !errors.Is(err, ErrReciboCargaDirectaNoValido) {
		t.Fatalf("se acepto un recibo no opaco: %v", err)
	}
}

func TestRegistroYPredecesorExigenCronologiaDurableExacta(t *testing.T) {
	t.Parallel()

	registrado := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	registro := RegistroReciboCargaDirecta{
		IndiceHMAC: hmacReciboPrueba("a"), GrupoHMAC: hmacReciboPrueba("b"),
		VinculoHMAC: hmacReciboPrueba("c"), EvidenciaAltaRef: "evidencia:alta:1",
		AutorizacionEmisionRef: "autorizacion:emision:1", ExpiraEn: registrado.Add(5 * time.Minute),
	}
	resultado := ResultadoRegistroReciboCargaDirecta{
		IndiceHMAC: registro.IndiceHMAC, GrupoHMAC: registro.GrupoHMAC,
		AutorizacionEmisionRef: registro.AutorizacionEmisionRef, RegistradoEn: registrado,
	}
	if err := resultado.ValidarContra(registro); err != nil {
		t.Fatalf("alta durable valida rechazada: %v", err)
	}
	predecesor := PredecesorReciboCargaDirecta{
		IndiceHMAC: hmacReciboPrueba("e"), GrupoHMAC: registro.GrupoHMAC,
		AutorizacionEmisionRef: "autorizacion:anterior:1", SustituidoEn: registrado,
	}
	resultado.Predecesor = &predecesor
	if err := resultado.ValidarContra(registro); err != nil {
		t.Fatalf("predecesor ligado rechazado: %v", err)
	}
	resultado.Predecesor.GrupoHMAC = hmacReciboPrueba("f")
	if !errors.Is(resultado.ValidarContra(registro), ErrReciboCargaDirectaNoValido) {
		t.Fatal("se acepto un predecesor de otro grupo")
	}
	resultado.Predecesor = nil
	resultado.RegistradoEn = registro.ExpiraEn
	if !errors.Is(resultado.ValidarContra(registro), ErrReciboCargaDirectaNoValido) {
		t.Fatal("se acepto un alta en el limite de expiracion")
	}
}

func TestResultadoConsumoSeLigaExactamenteALaOrden(t *testing.T) {
	t.Parallel()

	resultado := resultadoConsumoReciboPrueba()
	orden := OrdenConsumoReciboCargaDirecta{
		IndiceHMAC: resultado.IndiceHMAC, GrupoHMAC: resultado.GrupoHMAC, VinculoHMAC: resultado.VinculoHMAC,
		EvidenciaConsumoRef:      resultado.EvidenciaConsumoRef,
		IntencionConfirmacionRef: resultado.IntencionConfirmacionRef,
		HuellaIntencionHMAC:      resultado.HuellaIntencionHMAC, RegistradoEn: resultado.RegistradoEn,
		ValidaHasta: resultado.ConsumidoEn.Add(time.Minute),
	}
	if err := resultado.ValidarContra(orden); err != nil {
		t.Fatalf("consumo exacto rechazado: %v", err)
	}
	alterada := orden
	alterada.IntencionConfirmacionRef = "intencion:otra"
	if !errors.Is(resultado.ValidarContra(alterada), ErrReciboCargaDirectaNoValido) {
		t.Fatal("se acepto una intencion distinta")
	}
	alterada = orden
	alterada.ValidaHasta = resultado.ConsumidoEn
	if !errors.Is(resultado.ValidarContra(alterada), ErrReciboCargaDirectaNoValido) {
		t.Fatal("se acepto consumo en el limite superior")
	}
}

func TestComprobanteConservaProyeccionYAtestacionOpacas(t *testing.T) {
	t.Parallel()

	resultado := resultadoConsumoReciboPrueba()
	validaHasta := resultado.ConsumidoEn.Add(time.Minute)
	atestacion := hmacReciboPrueba("e")
	comprobante, err := NuevoComprobanteConsumoReciboCargaDirecta(resultado, validaHasta, atestacion)
	if err != nil {
		t.Fatalf("crear comprobante: %v", err)
	}
	datos, err := comprobante.DatosVerificados()
	if err != nil || datos.IntencionRef != resultado.IntencionConfirmacionRef || datos.AtestacionHMAC != atestacion {
		t.Fatalf("proyeccion distinta: %#v, %v", datos, err)
	}
	if _, err := json.Marshal(comprobante); !errors.Is(err, ErrSerializacionReciboCargaProhibida) {
		t.Fatalf("serializacion no denegada: %v", err)
	}
	if comprobante.String() == atestacion {
		t.Fatal("la representacion textual revelo la atestacion")
	}

	datos.ValidaHasta = resultado.ConsumidoEn
	if !errors.Is(
		ValidarDatosComprobanteConsumoReciboCargaDirecta(datos),
		ErrReciboCargaDirectaNoValido,
	) {
		t.Fatal("se acepto una atestacion fuera de su ventana")
	}
}

func TestFormatoHMACEsCerradoYVersionado(t *testing.T) {
	t.Parallel()

	if !HMACSHA256Valido(hmacReciboPrueba("a")) {
		t.Fatal("se rechazo un HMAC canonico")
	}
	for _, valor := range []string{
		strings.Repeat("a", 64),
		"hmac-sha256::" + strings.Repeat("a", 64),
		"sha256:v1:" + strings.Repeat("a", 64),
		"hmac-sha256:v1:" + strings.Repeat("A", 64),
	} {
		if HMACSHA256Valido(valor) {
			t.Errorf("se acepto el HMAC no canonico %q", valor)
		}
	}
}
