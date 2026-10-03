package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/simulacion"
)

func entradaPrueba(t *testing.T) []byte {
	t.Helper()
	ejemplos, err := simulacion.Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	puntos := int64(9_000_000)
	material, err := json.Marshal(simulacion.SolicitudRevisionNotas{EjemploRef: ejemplos[0].Referencia, Configuracion: ejemplos[0].Configuracion,
		NotasAntecedente: []simulacion.NotaPrueba{{SolicitudRef: "solicitud_1", FaseRef: "ejercicio_1", PuntosMicropuntos: &puntos}},
		NotasPropuestas:  []simulacion.NotaPrueba{{SolicitudRef: "solicitud_2", FaseRef: "ejercicio_1", PuntosMicropuntos: nil}}})
	if err != nil {
		t.Fatal(err)
	}
	return material
}

func TestCLINotasPreparaJSONRepetibleSinDatosExternos(t *testing.T) {
	entrada := entradaPrueba(t)
	var anterior bytes.Buffer
	for i := 0; i < 2; i++ {
		var salida, diagnostico bytes.Buffer
		if codigo := ejecutar(nil, bytes.NewReader(entrada), &salida, &diagnostico); codigo != 0 || diagnostico.Len() != 0 {
			t.Fatalf("CLI: %d %s", codigo, diagnostico.String())
		}
		var r simulacion.RevisionNotas
		if json.Unmarshal(salida.Bytes(), &r) != nil || r.Esquema != "seleccion.revision_notas.v1" || len(r.Cambios) != 1 || r.Cambios[0].PropuestaMicropuntos != nil || *r.Propuesta.Solicitudes[0].Fases[0].PuntosMicropuntos != 9_000_000 || r.Propuesta.Solicitudes[1].Estado != "pendiente" || len(r.Pendientes) != 4 {
			t.Fatal("salida no conserva el antecedente o el límite sintético")
		}
		if i == 1 && !bytes.Equal(anterior.Bytes(), salida.Bytes()) {
			t.Fatal("el CLI modificó la segunda ejecución")
		}
		anterior.Write(salida.Bytes())
	}
}

type lectorContado struct {
	r      io.Reader
	leidos int
}

func (l *lectorContado) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.leidos += n
	return n, err
}

func TestCLINotasLimitaStdinYNoEmitePreparacionEnErrores(t *testing.T) {
	material := entradaPrueba(t)
	for _, caso := range []struct {
		args    []string
		entrada io.Reader
	}{
		{nil, strings.NewReader("")},
		{nil, strings.NewReader(`{"ejemplo_ref":"ajeno","configuracion":{}}`)},
		{nil, strings.NewReader(`{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{},"persona_ref":"ajena"}`)},
		{nil, bytes.NewReader(bytes.Replace(material, []byte(`"puntos_micropuntos":9000000`), []byte(`"puntos_micropuntos":10000001`), 1))},
		{[]string{"/archivo.json"}, bytes.NewReader(material)},
		{[]string{"--salida=/archivo.json"}, bytes.NewReader(material)},
		{nil, nil},
		{nil, lectorFallido{}},
	} {
		var salida, diagnostico bytes.Buffer
		if codigo := ejecutar(caso.args, caso.entrada, &salida, &diagnostico); codigo != 1 || salida.Len() != 0 || diagnostico.String() != "{\"error\":\"solicitud_invalida\"}\n" {
			t.Fatalf("entrada inválida produjo salida: %d %s %s", codigo, salida.String(), diagnostico.String())
		}
	}
	lector := &lectorContado{r: strings.NewReader(strings.Repeat("x", 2*simulacion.MaximoBytes))}
	var salida, diagnostico bytes.Buffer
	if codigo := ejecutar(nil, lector, &salida, &diagnostico); codigo != 1 || salida.Len() != 0 || lector.leidos != simulacion.MaximoBytes+1 {
		t.Fatal("stdin no quedó acotado antes de decodificar")
	}
}

type lectorFallido struct{}

func (lectorFallido) Read([]byte) (int, error) { return 0, errors.New("detalle privado del lector") }

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) {
	return 0, errors.New("detalle privado del escritor")
}

func TestCLINotasInformaFalloSalidaSinFiltrarDetalles(t *testing.T) {
	var diagnostico bytes.Buffer
	if codigo := ejecutar(nil, bytes.NewReader(entradaPrueba(t)), escritorFallido{}, &diagnostico); codigo != 1 || diagnostico.String() != "{\"error\":\"salida_no_disponible\"}\n" {
		t.Fatalf("fallo stdout: %d %s", codigo, diagnostico.String())
	}
	if codigo := ejecutar(nil, strings.NewReader("invalido"), io.Discard, escritorFallido{}); codigo != 2 {
		t.Fatal("se declaró entregado un diagnóstico que falló")
	}
}
