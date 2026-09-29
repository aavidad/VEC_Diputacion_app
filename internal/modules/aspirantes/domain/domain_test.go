package domain

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Datos sintéticos: números con letra válida que no se combinan con datos
// reales ni salen del entorno de desarrollo.
const (
	dniSintetico = "12345678Z"
	nieSintetico = "X1234567L"
)

func TestReferenciasAleatoriasYFormato(t *testing.T) {
	a, err := GenerarReferenciaAspirante(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerarReferenciaAspirante(rand.Reader)
	if a == b || !ReferenciaAspiranteValida(a) || !ReferenciaAspiranteValida(b) || !strings.HasPrefix(a, "asp_") || len(a) != 26 {
		t.Fatalf("referencias %q %q", a, b)
	}
	d, err := GenerarReferenciaDocumento(bytes.NewReader(make([]byte, 16)))
	if err != nil || !ReferenciaDocumentoValida(d) || ReferenciaAspiranteValida(d) {
		t.Fatalf("documento %q %v", d, err)
	}
	if _, err := GenerarReferenciaAspirante(nil); err == nil {
		t.Fatal("sin azar debe fallar")
	}
	if _, err := GenerarReferenciaAspirante(bytes.NewReader(make([]byte, 10))); err == nil {
		t.Fatal("azar corto debe fallar")
	}
	for _, mala := range []string{"", "asp_", "asp_corta", "per_" + a[4:], a + "x", "asp_" + strings.Repeat("+", 22), "asp_AAAAAAAAAAAAAAAAAAAAAB"} {
		if ReferenciaAspiranteValida(mala) {
			t.Fatalf("aceptada %q", mala)
		}
	}
}

func TestDocumentoNormalizaYValidaLetra(t *testing.T) {
	d, err := NuevoDocumentoIdentidad(DocumentoDNI, " es ", "12.345.678-z")
	if err != nil || d.Numero != dniSintetico || d.Pais != "ES" {
		t.Fatalf("dni %+v %v", d, err)
	}
	if _, err := NuevoDocumentoIdentidad(DocumentoNIE, "ES", nieSintetico); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"Y1234567X", "Z1234567R"} {
		if _, err := NuevoDocumentoIdentidad(DocumentoNIE, "ES", n); err != nil {
			t.Fatalf("nie %s: %v", n, err)
		}
	}
	casos := []struct {
		tipo         TipoDocumento
		pais, numero string
	}{
		{DocumentoDNI, "ES", "12345678A"},  // letra mal
		{DocumentoDNI, "PT", dniSintetico}, // DNI de otro país
		{DocumentoDNI, "ES", "1234567Z"},   // corto
		{DocumentoNIE, "ES", "W1234567L"},  // prefijo
		{DocumentoNIE, "ES", "X1234567A"},  // letra
		{DocumentoPasaporte, "FR", "AB"},   // corto
		{DocumentoPasaporte, "FR", "AB12€34"},
		{DocumentoOtro, "ES", "ABC123"}, // «otro» nunca es español
		{"cedula", "ES", dniSintetico},
		{DocumentoPasaporte, "F", "AB1234567"},
	}
	for _, c := range casos {
		if _, err := NuevoDocumentoIdentidad(c.tipo, c.pais, c.numero); !errors.Is(err, ErrDocumentoInvalido) {
			t.Fatalf("aceptado %+v", c)
		}
	}
}

func TestEnmascaradoSegunTipo(t *testing.T) {
	casos := map[string]struct {
		tipo         TipoDocumento
		pais, numero string
	}{
		"***4567**":  {DocumentoDNI, "ES", dniSintetico},
		"****4567*":  {DocumentoNIE, "ES", nieSintetico},
		"***2345**":  {DocumentoPasaporte, "FR", "AB1234567"},
		"*****":      {DocumentoPasaporte, "FR", "AB123"},
		"***DEF4***": {DocumentoOtro, "MA", "ABCDEF4567"},
	}
	for esperado, c := range casos {
		d, err := NuevoDocumentoIdentidad(c.tipo, c.pais, c.numero)
		if err != nil {
			t.Fatal(err)
		}
		if got := d.Enmascarado(); got != esperado {
			t.Fatalf("%s: %q, esperado %q", c.numero, got, esperado)
		}
	}
	if (DocumentoIdentidad{Tipo: DocumentoDNI, Pais: "ES", Numero: "roto"}).Enmascarado() != "" {
		t.Fatal("un documento inválido no se enmascara")
	}
}

func TestNormalizarValoresDeContacto(t *testing.T) {
	bien := []struct {
		campo         CampoFicha
		entrada, sale string
	}{
		{CampoTelefono, "958 12 34 56", "958123456"},
		{CampoTelefono, "+34 612 345 678", "612345678"},
		{CampoTelefono, "0044 20 7946 0958", "+442079460958"},
		{CampoMovil, "612-345-678", "612345678"},
		{CampoMovil, "(+351) 912 345 678", ""},
		{CampoCodigoPostal, "18001", "18001"},
		{CampoCodigoPostal, " 01000 ", "01000"},
		{CampoDomicilio, "  Calle   Recogidas, 12,  3.º B ", "Calle Recogidas, 12, 3.º B"},
		{CampoNombre, "María  José", "María José"},
		{CampoApellidos, "O'Connor-Ruiz  de la Torre", "O'Connor-Ruiz de la Torre"},
	}
	for _, c := range bien {
		got, err := NormalizarValor(c.campo, c.entrada)
		if c.sale == "" {
			if err == nil {
				t.Fatalf("%s %q aceptado como %q", c.campo, c.entrada, got)
			}
			continue
		}
		if err != nil || got != c.sale {
			t.Fatalf("%s %q -> %q %v", c.campo, c.entrada, got, err)
		}
	}
	mal := []struct {
		campo CampoFicha
		valor string
	}{
		{CampoMovil, "958123456"}, {CampoTelefono, "12345"}, {CampoTelefono, "+34 12345678"}, {CampoTelefono, "+34 6123456789"}, {CampoTelefono, "958a23456"}, {CampoTelefono, "+0123456789"},
		{CampoCodigoPostal, "53000"}, {CampoCodigoPostal, "00100"}, {CampoCodigoPostal, "1800"}, {CampoCodigoPostal, "18001-2"},
		{CampoDomicilio, "C/ 1"}, {CampoDomicilio, "Calle\x00 Mayor 1"}, {CampoDomicilio, strings.Repeat("a", 201)},
		{CampoNombre, "R2D2"}, {CampoNombre, "   "}, {CampoNombre, "--"}, {"discapacidad", "33 %"},
	}
	for _, c := range mal {
		if v, err := NormalizarValor(c.campo, c.valor); err == nil {
			t.Fatalf("%s %q aceptado como %q", c.campo, c.valor, v)
		}
	}
}

func TestVocabularioSinCategoriasEspeciales(t *testing.T) {
	for _, c := range []CampoFicha{"discapacidad", "adaptacion", "victima_violencia", "salud", "correo"} {
		if c.Valido() {
			t.Fatalf("campo %s no debe existir en este corte", c)
		}
	}
	for _, c := range append(append([]CampoFicha{}, CamposIdentidad...), CamposContacto...) {
		if !c.Valido() || c.EsIdentidad() == c.EsContacto() {
			t.Fatalf("campo %s mal clasificado", c)
		}
	}
}

func TestContactoPedidoSoloLoQuePideElCatalogo(t *testing.T) {
	exig := []ExigenciaCampo{{Campo: CampoTelefono, Obligatorio: true}, {Campo: CampoDomicilio}, {Campo: CampoCodigoPostal}}
	got, err := ContactoPedido(map[CampoFicha]string{CampoTelefono: "612 345 678", CampoDomicilio: ""}, exig, true)
	if err != nil || len(got) != 1 || got[CampoTelefono] != "612345678" {
		t.Fatalf("alta %v %v", got, err)
	}
	if _, err := ContactoPedido(map[CampoFicha]string{CampoDomicilio: "Calle Mayor, 3"}, exig, true); err == nil {
		t.Fatal("alta sin el obligatorio")
	}
	if _, err := ContactoPedido(map[CampoFicha]string{CampoTelefono: "612345678", CampoMovil: "612345678"}, exig, true); err == nil {
		t.Fatal("campo no pedido")
	}
	if _, err := ContactoPedido(map[CampoFicha]string{CampoNombre: "Antonio"}, exig, false); err == nil {
		t.Fatal("la identidad no se rectifica aquí")
	}
	if _, err := ContactoPedido(map[CampoFicha]string{CampoTelefono: ""}, exig, false); err == nil {
		t.Fatal("un obligatorio no se vacía")
	}
	got, err = ContactoPedido(map[CampoFicha]string{CampoMovil: "", CampoDomicilio: ""}, exig, false)
	if err != nil || len(got) != 2 || got[CampoMovil] != "" {
		t.Fatalf("retirar opcionales o no pedidos: %v %v", got, err)
	}
	if _, err := ContactoPedido(map[CampoFicha]string{}, exig, false); err == nil {
		t.Fatal("rectificar sin cambios")
	}
	if _, err := ContactoPedido(map[CampoFicha]string{CampoTelefono: "612345678"}, []ExigenciaCampo{{Campo: CampoTelefono}, {Campo: CampoTelefono}}, true); err == nil {
		t.Fatal("exigencia repetida")
	}
	if _, err := ContactoPedido(map[CampoFicha]string{CampoTelefono: "612345678"}, []ExigenciaCampo{{Campo: CampoNombre}}, true); err == nil {
		t.Fatal("exigencia de identidad")
	}
}

func TestMotivoSegunLoQueHabia(t *testing.T) {
	presentes := []CampoFicha{CampoTelefono}
	casos := []struct {
		motivo  MotivoCambio
		cambios map[CampoFicha]string
		ok      bool
	}{
		{MotivoDatoNuevo, map[CampoFicha]string{CampoDomicilio: "Calle Mayor, 3"}, true},
		{MotivoCambioDeDato, map[CampoFicha]string{CampoDomicilio: "Calle Mayor, 3"}, false},
		{MotivoCambioDeDato, map[CampoFicha]string{CampoTelefono: "958123456"}, true},
		{MotivoCorreccionDeError, map[CampoFicha]string{CampoTelefono: "958123456", CampoDomicilio: "Calle Mayor, 3"}, true},
		{MotivoDatoNuevo, map[CampoFicha]string{CampoTelefono: "958123456"}, false},
		{MotivoCambioDeDato, map[CampoFicha]string{CampoTelefono: ""}, true},
		{MotivoCambioDeDato, map[CampoFicha]string{CampoMovil: ""}, false}, // retirar lo que no hay
		{MotivoAltaTitular, map[CampoFicha]string{CampoDomicilio: "Calle Mayor, 3"}, false},
		{"otro", map[CampoFicha]string{CampoDomicilio: "Calle Mayor, 3"}, false},
	}
	for i, c := range casos {
		if err := ValidarMotivo(c.motivo, c.cambios, presentes); (err == nil) != c.ok {
			t.Fatalf("caso %d: %v", i, err)
		}
	}
}

func TestIdentidadAcreditadaRedactadaYValidada(t *testing.T) {
	doc, _ := NuevoDocumentoIdentidad(DocumentoNIE, "ES", nieSintetico)
	id, err := NuevaIdentidadAcreditada(" Karim ", "Benali", doc)
	if err != nil {
		t.Fatal(err)
	}
	v := id.Valores()
	if len(v) != 2 || v[CampoNombre] != "Karim" || id.Documento() != doc || id.Validar() != nil {
		t.Fatalf("valores %v", v)
	}
	for _, s := range []string{fmt.Sprint(id), fmt.Sprintf("%#v", id), fmt.Sprintf("%+v", id)} {
		if strings.Contains(s, "Karim") || strings.Contains(s, nieSintetico) {
			t.Fatalf("fuga en %q", s)
		}
	}
	if _, err := NuevaIdentidadAcreditada("Antonio", "Reyes Álvarez", DocumentoIdentidad{}); err == nil {
		t.Fatal("sin documento")
	}
	if _, err := NuevaIdentidadAcreditada("", "Reyes Álvarez", doc); err == nil {
		t.Fatal("sin nombre")
	}
	if (IdentidadAcreditada{}).Validar() == nil {
		t.Fatal("identidad vacía")
	}
}

func TestDocumentoNoSeImprimeEnClaro(t *testing.T) {
	d, _ := NuevoDocumentoIdentidad(DocumentoDNI, "ES", dniSintetico)
	for _, texto := range []string{fmt.Sprint(d), fmt.Sprintf("%v", d), fmt.Sprintf("%+v", d), fmt.Sprintf("%#v", d), fmt.Sprint(struct{ D DocumentoIdentidad }{d}), func() string { b, _ := json.Marshal(struct{ D DocumentoIdentidad }{d}); return string(b) }()} {
		if strings.Contains(texto, dniSintetico) || !strings.Contains(texto, "***4567**") {
			t.Fatalf("fuga en %q", texto)
		}
	}
}

func TestCondicionDeExigencia(t *testing.T) {
	if ValidarExigencias([]ExigenciaCampo{{Campo: CampoDomicilio, Condicion: "si_elige_notificacion_papel"}}) != nil {
		t.Fatal("condición válida rechazada")
	}
	if ValidarExigencias([]ExigenciaCampo{{Campo: CampoDomicilio, Condicion: "<script>"}}) == nil {
		t.Fatal("condición con caracteres fuera del código admitida")
	}
}
