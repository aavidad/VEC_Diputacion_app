package calculoexperienciaoficial

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestClaveEfectoCanonicaDeterministaYRestaurable(t *testing.T) {
	primera := clavePrueba(t)
	segunda, err := NuevaClaveEfectoV1(datosClavePrueba())
	if err != nil {
		t.Fatal(err)
	}
	bytesPrimera, err := primera.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	bytesSegunda, err := segunda.RepresentacionCanonica()
	if err != nil || !bytes.Equal(bytesPrimera, bytesSegunda) {
		t.Fatalf("representación no determinista: %v", err)
	}
	huella, _ := primera.HuellaSHA256()
	restaurada, err := RestaurarClaveEfectoV1ConHuellaSHA256(bytesPrimera, huella)
	if err != nil {
		t.Fatalf("restaurar clave canónica: %v", err)
	}
	huellaRestaurada, _ := restaurada.HuellaSHA256()
	if huellaRestaurada != huella {
		t.Fatal("la restauración cambió la identidad")
	}
}

func TestClaveEfectoRechazaJSONNoCanonicoYManipulacion(t *testing.T) {
	clave := clavePrueba(t)
	canonico, _ := clave.RepresentacionCanonica()
	huella, _ := clave.HuellaSHA256()
	casos := [][]byte{
		append([]byte(" "), canonico...),
		append(append([]byte(nil), canonico[:len(canonico)-1]...),
			[]byte(`,"campo_ajeno":1}`)...),
		append(append([]byte(nil), canonico[:len(canonico)-1]...),
			[]byte(`,"tipo":"calculo_inicial"}`)...),
		bytes.Repeat([]byte("x"), maximoBytesRepresentacionV1+1),
	}
	for indice, caso := range casos {
		if _, err := RestaurarClaveEfectoV1(caso); err == nil {
			t.Fatalf("caso no canónico %d aceptado", indice)
		}
	}
	manipulada := bytes.Replace(
		canonico, []byte(huellaPrueba("8")), []byte(huellaPrueba("f")), 1,
	)
	if _, err := RestaurarClaveEfectoV1(manipulada); err != nil {
		t.Fatalf("la variante canónica válida debía poder restaurarse: %v", err)
	}
	if _, err := RestaurarClaveEfectoV1ConHuellaSHA256(manipulada, huella); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("manipulación no detectada por la huella: %v", err)
	}
}

func TestIndiceHMACDeterministaSecretoYSemantica(t *testing.T) {
	clave := clavePrueba(t)
	secreto := []byte(strings.Repeat("s", 32))
	primero, err := CalcularIndiceHMACSHA256(clave, secreto)
	segundo, errSegundo := CalcularIndiceHMACSHA256(clave, append([]byte(nil), secreto...))
	if err != nil || errSegundo != nil || primero != segundo || !huellaSHA256Valida(primero) {
		t.Fatalf("índice HMAC no determinista: %v / %v", err, errSegundo)
	}
	otroSecreto, _ := CalcularIndiceHMACSHA256(clave, []byte(strings.Repeat("t", 32)))
	if otroSecreto == primero {
		t.Fatal("dos secretos produjeron el mismo índice")
	}
	if _, err := CalcularIndiceHMACSHA256(clave, []byte("corta")); !errors.Is(err, ErrSecretoInvalido) {
		t.Fatalf("secreto corto aceptado: %v", err)
	}
}

func TestCadaEntradaSemanticaCambiaClaveEIndice(t *testing.T) {
	base := datosClavePrueba()
	mutaciones := []struct {
		nombre  string
		aplicar func(*DatosClaveEfectoV1)
	}{
		{"sujeto", func(d *DatosClaveEfectoV1) { d.SujetoPseudonimizado.Version++ }},
		{"convocatoria", func(d *DatosClaveEfectoV1) { d.Convocatoria.Version++ }},
		{"reglas", func(d *DatosClaveEfectoV1) { d.Reglas.Revision++ }},
		{"entrada", func(d *DatosClaveEfectoV1) { d.Entrada.HuellaContenidoSHA256 = huellaPrueba("a") }},
		{"motor", func(d *DatosClaveEfectoV1) { d.Motor.HuellaContratoSHA256 = huellaPrueba("b") }},
		{"plan", func(d *DatosClaveEfectoV1) { d.HuellaPlanSHA256 = huellaPrueba("c") }},
		{"causa", func(d *DatosClaveEfectoV1) { d.Causa.Clave = "recalculo_por_resolucion" }},
		{"rectificacion", func(d *DatosClaveEfectoV1) {
			d.Tipo = EfectoRectificacion
			d.Predecesor = &VinculoPredecesorV1{
				ReferenciaRecibo: "ReciboCalculo/Previo#1", HuellaReciboSHA256: huellaPrueba("d"),
			}
		}},
	}
	original, _ := NuevaClaveEfectoV1(base)
	huellaOriginal, _ := original.HuellaSHA256()
	indiceOriginal, _ := CalcularIndiceHMACSHA256(original, []byte(strings.Repeat("k", 32)))
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			modificada := base
			caso.aplicar(&modificada)
			clave, err := NuevaClaveEfectoV1(modificada)
			if err != nil {
				t.Fatalf("mutación válida rechazada: %v", err)
			}
			huella, _ := clave.HuellaSHA256()
			indice, _ := CalcularIndiceHMACSHA256(clave, []byte(strings.Repeat("k", 32)))
			if huella == huellaOriginal || indice == indiceOriginal {
				t.Fatal("la entrada semántica no alteró identidad e índice")
			}
		})
	}
}

func TestPredecesorSoloEnRectificacionYRolesDistintos(t *testing.T) {
	datos := datosClavePrueba()
	datos.Predecesor = &VinculoPredecesorV1{
		ReferenciaRecibo: "ReciboCalculo/Previo#1", HuellaReciboSHA256: huellaPrueba("a"),
	}
	if _, err := NuevaClaveEfectoV1(datos); !errors.Is(err, ErrEstadoIncompatible) {
		t.Fatalf("cálculo inicial con predecesor aceptado: %v", err)
	}
	datos.Tipo, datos.Predecesor = EfectoRectificacion, nil
	if _, err := NuevaClaveEfectoV1(datos); !errors.Is(err, ErrEstadoIncompatible) {
		t.Fatalf("rectificación sin predecesor aceptada: %v", err)
	}
	datos = datosClavePrueba()
	datos.Convocatoria = datos.SujetoPseudonimizado
	if _, err := NuevaClaveEfectoV1(datos); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("confusión sujeto/convocatoria aceptada: %v", err)
	}
}
