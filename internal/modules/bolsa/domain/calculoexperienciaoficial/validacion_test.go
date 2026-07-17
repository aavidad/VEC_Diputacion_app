package calculoexperienciaoficial

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestClaveRechazaCadaComponenteInvalido(t *testing.T) {
	casos := []struct {
		nombre  string
		aplicar func(*DatosClaveEfectoV1)
	}{
		{"sujeto", func(d *DatosClaveEfectoV1) { d.SujetoPseudonimizado.Referencia = "" }},
		{"convocatoria_version", func(d *DatosClaveEfectoV1) { d.Convocatoria.Version = 0 }},
		{"reglas_contenido", func(d *DatosClaveEfectoV1) { d.Reglas.Contenido.HuellaSHA256 = "X" }},
		{"reglas_revision", func(d *DatosClaveEfectoV1) { d.Reglas.Revision = 0 }},
		{"reglas_estado", func(d *DatosClaveEfectoV1) { d.Reglas.HuellaEstadoSHA256 = "X" }},
		{"entrada_instantanea", func(d *DatosClaveEfectoV1) { d.Entrada.Instantanea.Version = 0 }},
		{"entrada_huella", func(d *DatosClaveEfectoV1) { d.Entrada.HuellaContenidoSHA256 = "X" }},
		{"motor_contrato", func(d *DatosClaveEfectoV1) { d.Motor.Contrato = "NO CANONICO" }},
		{"motor_version", func(d *DatosClaveEfectoV1) { d.Motor.Version = 0 }},
		{"motor_huella", func(d *DatosClaveEfectoV1) { d.Motor.HuellaContratoSHA256 = "X" }},
		{"plan", func(d *DatosClaveEfectoV1) { d.HuellaPlanSHA256 = "X" }},
		{"causa_catalogo", func(d *DatosClaveEfectoV1) { d.Causa.Catalogo.Referencia = "?" }},
		{"causa_clave", func(d *DatosClaveEfectoV1) { d.Causa.Clave = "causa:no-admitida" }},
		{"tipo", func(d *DatosClaveEfectoV1) { d.Tipo = "otro" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := datosClavePrueba()
			caso.aplicar(&datos)
			if _, err := NuevaClaveEfectoV1(datos); err == nil {
				t.Fatal("componente inválido aceptado")
			}
		})
	}
}

func TestLimitesReferenciaCausaVersionYSecreto(t *testing.T) {
	datos := datosClavePrueba()
	datos.SujetoPseudonimizado.Referencia = "hmac-sha256:a" +
		strings.Repeat("z", 127) + ":" + huellaPrueba("0")
	datos.SujetoPseudonimizado.Version = maximoVersionV1
	datos.Causa.Clave = "a" + strings.Repeat("z", 127)
	clave, err := NuevaClaveEfectoV1(datos)
	if err != nil {
		t.Fatalf("bordes válidos rechazados: %v", err)
	}
	if _, err := CalcularIndiceHMACSHA256(clave, bytes.Repeat([]byte{1}, minimoBytesSecretoHMACV1)); err != nil {
		t.Fatalf("secreto mínimo rechazado: %v", err)
	}
	if _, err := CalcularIndiceHMACSHA256(clave, bytes.Repeat([]byte{1}, maximoBytesSecretoHMACV1)); err != nil {
		t.Fatalf("secreto máximo rechazado: %v", err)
	}
	datos.SujetoPseudonimizado.Referencia += "z"
	if _, err := NuevaClaveEfectoV1(datos); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("referencia excesiva aceptada: %v", err)
	}
	datos = datosClavePrueba()
	datos.Causa.Clave = "a" + strings.Repeat("z", 128)
	if _, err := NuevaClaveEfectoV1(datos); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("causa excesiva aceptada: %v", err)
	}
	if _, err := CalcularIndiceHMACSHA256(clave, bytes.Repeat([]byte{1}, maximoBytesSecretoHMACV1+1)); !errors.Is(err, ErrSecretoInvalido) {
		t.Fatalf("secreto excesivo aceptado: %v", err)
	}
}

func TestSujetoDirectoSeRechazaEnConstructorYRestauracion(t *testing.T) {
	base := datosClavePrueba()
	clave, err := NuevaClaveEfectoV1(base)
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := clave.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	original := []byte(base.SujetoPseudonimizado.Referencia)
	for _, hostil := range []string{
		"12345678Z", "persona@example.test", "../../expedientes/usuario",
	} {
		t.Run(hostil, func(t *testing.T) {
			datos := base
			datos.SujetoPseudonimizado.Referencia = hostil
			if _, err := NuevaClaveEfectoV1(datos); !errors.Is(err, ErrValorNoCanonico) {
				t.Fatalf("constructor acepto sujeto directo: %v", err)
			}
			alterado := bytes.Replace(canonico, original, []byte(hostil), 1)
			if bytes.Equal(alterado, canonico) {
				t.Fatal("la prueba no altero el sujeto")
			}
			if _, err := RestaurarClaveEfectoV1(alterado); !errors.Is(
				err, ErrValorNoCanonico,
			) {
				t.Fatalf("restauracion acepto sujeto directo: %v", err)
			}
		})
	}
}

func TestIndiceHMACUsaDominioSeparado(t *testing.T) {
	clave := clavePrueba(t)
	secreto := []byte(strings.Repeat("d", 32))
	canonico, _ := clave.RepresentacionCanonica()
	indice, _ := CalcularIndiceHMACSHA256(clave, secreto)
	conDominio := hmac.New(sha256.New, secreto)
	_, _ = conDominio.Write([]byte(dominioIndiceHMACV1))
	_, _ = conDominio.Write([]byte{0})
	_, _ = conDominio.Write(canonico)
	sinDominio := hmac.New(sha256.New, secreto)
	_, _ = sinDominio.Write(canonico)
	if indice != hex.EncodeToString(conDominio.Sum(nil)) ||
		indice == hex.EncodeToString(sinDominio.Sum(nil)) {
		t.Fatal("el índice no aplica separación de dominio")
	}
}

func TestRectificacionConPredecesorExactoSeRestauraSinCompartir(t *testing.T) {
	datos := datosClavePrueba()
	predecesor := &VinculoPredecesorV1{
		ReferenciaRecibo: "ReciboCalculo/Previo#1", HuellaReciboSHA256: huellaPrueba("d"),
	}
	datos.Tipo, datos.Predecesor = EfectoRectificacion, predecesor
	clave, err := NuevaClaveEfectoV1(datos)
	if err != nil {
		t.Fatal(err)
	}
	predecesor.ReferenciaRecibo = "Manipulada"
	recuperado, existe := clave.Predecesor()
	if !existe || recuperado.ReferenciaRecibo != "ReciboCalculo/Previo#1" {
		t.Fatal("la clave compartió el puntero del predecesor")
	}
	canonico, _ := clave.RepresentacionCanonica()
	restaurada, err := RestaurarClaveEfectoV1(canonico)
	if err != nil {
		t.Fatal(err)
	}
	if _, existe := restaurada.Predecesor(); !existe {
		t.Fatal("la restauración perdió el predecesor")
	}
	datos = datosClavePrueba()
	datos.Tipo = EfectoRectificacion
	datos.Predecesor = &VinculoPredecesorV1{ReferenciaRecibo: "?", HuellaReciboSHA256: "X"}
	if _, err := NuevaClaveEfectoV1(datos); err == nil {
		t.Fatal("predecesor inválido aceptado")
	}
}

func TestRestauracionesRechazanEsquemaYHuellaEsperadaInvalidos(t *testing.T) {
	clave, intencion, recibo := clavePrueba(t), intencionPrueba(t), reciboPrueba(t)
	casos := []struct {
		canonico  func() ([]byte, error)
		restaurar func([]byte) error
		esquema   string
	}{
		{clave.RepresentacionCanonica, func(b []byte) error { _, e := RestaurarClaveEfectoV1(b); return e }, esquemaClaveEfectoV1},
		{intencion.RepresentacionCanonica, func(b []byte) error { _, e := RestaurarIntencionResultadoV1(b); return e }, esquemaIntencionV1},
		{recibo.RepresentacionCanonica, func(b []byte) error { _, e := RestaurarReciboV1(b); return e }, esquemaReciboV1},
	}
	for _, caso := range casos {
		canonico, _ := caso.canonico()
		alterado := bytes.Replace(canonico, []byte(caso.esquema), []byte(caso.esquema+".otro"), 1)
		if err := caso.restaurar(alterado); !errors.Is(err, ErrEsquemaIncompatible) {
			t.Fatalf("esquema incompatible no clasificado: %v", err)
		}
	}
	if _, err := RestaurarClaveEfectoV1ConHuellaSHA256([]byte(`{}`), "x"); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("huella esperada inválida aceptada: %v", err)
	}
	if _, err := RestaurarIntencionResultadoV1ConHuellaSHA256([]byte(`{}`), "x"); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("huella esperada inválida aceptada: %v", err)
	}
	if _, err := RestaurarReciboV1ConHuellaSHA256([]byte(`{}`), "x"); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("huella esperada inválida aceptada: %v", err)
	}
}
