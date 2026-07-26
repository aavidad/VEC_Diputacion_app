package cobertura_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSesionTCBProyectaSietePruebasC1DeterministasYDefensivas(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(escenario.reciboConcedido),
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cobertura.ConfirmarOperacionDecisionCobertura(
		context.Background(),
		transaccion,
		escenario.ordenConcedida,
	); err != nil {
		t.Fatal(err)
	}
	_, _, _, consumos := ejecutor.ultimaSesion().instantanea()
	if len(consumos) == 0 {
		t.Fatal("la concesión no proyectó consumos C1")
	}
	nominal := consumos[0].PruebasCanonicas
	primera, err := nominal.Datos()
	if err != nil {
		t.Fatal(err)
	}
	pruebas := pruebasCanonicasC1ComoLista(primera)
	if len(pruebas) != 7 {
		t.Fatalf("número de pruebas canónicas: %d", len(pruebas))
	}
	for indice, prueba := range pruebas {
		if len(prueba) == 0 || len(prueba) > 64*1024 {
			t.Fatalf("prueba %d fuera de límite: %d", indice, len(prueba))
		}
	}
	mutaciones := []struct {
		nombre string
		mutar  func(*puertosct.DatosPruebasCanonicasOrdenConsumoCobertura)
	}{
		{"petición", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.Peticion[0] ^= 0xff
		}},
		{"resultado", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.Resultado[0] ^= 0xff
		}},
		{"atestación", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.Atestacion[0] ^= 0xff
		}},
		{"confirmación TCB", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.ConfirmacionTCB[0] ^= 0xff
		}},
		{"catálogo", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.Catalogo[0] ^= 0xff
		}},
		{"verificador", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.Verificador[0] ^= 0xff
		}},
		{"resumen", func(d *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura) {
			d.Resumen[0] ^= 0xff
		}},
	}
	for _, mutacion := range mutaciones {
		t.Run("copia defensiva "+mutacion.nombre, func(t *testing.T) {
			adulterada, err := nominal.Datos()
			if err != nil {
				t.Fatal(err)
			}
			mutacion.mutar(&adulterada)
			intacta, err := nominal.Datos()
			if err != nil {
				t.Fatal(err)
			}
			if reflectPruebasCanonicasC1Iguales(adulterada, intacta) {
				t.Fatal("la adulteración no cambió la copia entregada")
			}
			if !reflectPruebasCanonicasC1Iguales(primera, intacta) {
				t.Fatal("la adulteración alcanzó el valor nominal")
			}
		})
	}
	segunda, err := nominal.Datos()
	tercera, err := nominal.Datos()
	if err != nil || !reflectPruebasCanonicasC1Iguales(primera, segunda) ||
		!reflectPruebasCanonicasC1Iguales(segunda, tercera) {
		t.Fatal("la proyección canónica no es determinista")
	}
	comprobarRedaccionYCodecsPruebasCanonicasC1(t, nominal, primera)
	if _, err := (puertosct.PruebasCanonicasOrdenConsumoCobertura{}).Datos(); err == nil {
		t.Fatal("el valor cero de pruebas fue aceptado")
	}
}

func TestSesionTCBPruebasC1MantienenVectoresDorados(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(escenario.reciboConcedido),
	}
	transaccion, _ :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if _, err := cobertura.ConfirmarOperacionDecisionCobertura(
		context.Background(),
		transaccion,
		escenario.ordenConcedida,
	); err != nil {
		t.Fatal(err)
	}
	_, _, _, consumos := ejecutor.ultimaSesion().instantanea()
	datos, err := consumos[0].PruebasCanonicas.Datos()
	if err != nil {
		t.Fatal(err)
	}
	pruebas := map[string][]byte{
		"peticion": datos.Peticion, "resultado": datos.Resultado,
		"atestacion": datos.Atestacion, "confirmacion_tcb": datos.ConfirmacionTCB,
		"catalogo": datos.Catalogo, "resumen": datos.Resumen,
	}
	// La prueba del verificador incorpora por diseño un desafío aleatorio
	// antirrepetición. Su estabilidad dentro de la misma orden se comprueba en
	// TestSesionTCBProyectaSietePruebasC1DeterministasYDefensivas, pero no debe
	// congelarse como vector entre ejecuciones.
	esperadas := map[string]string{
		"peticion":         "4acddcdedcaf803f74f2c01af89ca5f25498b5e700f0cb319dac33014840970a",
		"resultado":        "b722b69018760d6d05b3079a847751614932ccf97c27614f34e1a1ea42649987",
		"atestacion":       "05949b36431c504c232613df9178be07e2d4a4635f74930a84e1e05f787a8b2e",
		"confirmacion_tcb": "412c9d52c94b850f444466f7240cc4bc1929be7dd5e36623497be797e1165b7b",
		"catalogo":         "1675bde8e3cd0832e3258e8ec4ab31cb5c28bb21726631076008259453617481",
		"resumen":          "5f95110ebc0550f496c463402ca2dd5213aad110f4a2b7edc0b733ffa09dc501",
	}
	for nombre, prueba := range pruebas {
		huella := fmt.Sprintf("%x", sha256.Sum256(prueba))
		if huella != esperadas[nombre] {
			t.Errorf("%s: %s", nombre, huella)
		}
	}
}

func comprobarRedaccionYCodecsPruebasCanonicasC1(
	t *testing.T,
	nominal puertosct.PruebasCanonicasOrdenConsumoCobertura,
	datos puertosct.DatosPruebasCanonicasOrdenConsumoCobertura,
) {
	t.Helper()
	const redaccion = "[PRUEBAS-CANONICAS-CONSUMO-COBERTURA-REDACTADAS]"
	for _, valor := range []any{nominal, datos} {
		formato := fmt.Sprintf("%v %#v", valor, valor)
		var registro bytes.Buffer
		slog.New(slog.NewJSONHandler(&registro, nil)).Info(
			"prueba",
			"valor",
			valor,
		)
		if !strings.Contains(formato, redaccion) ||
			!strings.Contains(registro.String(), redaccion) {
			t.Fatalf(
				"%T no quedó redactado: %q / %s",
				valor,
				formato,
				registro.String(),
			)
		}
		for _, prueba := range pruebasCanonicasC1ComoLista(datos) {
			if strings.Contains(formato, string(prueba)) ||
				strings.Contains(registro.String(), string(prueba)) {
				t.Fatalf("%T filtró material canónico", valor)
			}
		}
		if _, err := json.Marshal(valor); !errors.Is(
			err,
			puertosct.ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida,
		) {
			t.Fatalf("%T admitió JSON: %v", valor, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(
			err,
			puertosct.ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida,
		) {
			t.Fatalf("%T admitió XML: %v", valor, err)
		}
	}
	comprobarCodecsPruebasCanonicasC1(t, &nominal)
	comprobarCodecsPruebasCanonicasC1(t, &datos)
}

type codecsPruebasCanonicasC1 interface {
	MarshalText() ([]byte, error)
	UnmarshalText([]byte) error
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	GobEncode() ([]byte, error)
	GobDecode([]byte) error
	MarshalCBOR() ([]byte, error)
	UnmarshalCBOR([]byte) error
	MarshalYAML() (any, error)
	UnmarshalYAML(func(any) error) error
}

func comprobarCodecsPruebasCanonicasC1(
	t *testing.T,
	valor codecsPruebasCanonicasC1,
) {
	t.Helper()
	esperado :=
		puertosct.ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
	comprobar := func(nombre string, err error) {
		t.Helper()
		if !errors.Is(err, esperado) {
			t.Fatalf("%s no bloqueado para %T: %v", nombre, valor, err)
		}
	}
	_, err := valor.MarshalText()
	comprobar("texto", err)
	comprobar("texto decode", valor.UnmarshalText([]byte("adulterado")))
	_, err = valor.MarshalBinary()
	comprobar("binario", err)
	comprobar("binario decode", valor.UnmarshalBinary([]byte("adulterado")))
	_, err = valor.GobEncode()
	comprobar("gob", err)
	comprobar("gob decode", valor.GobDecode([]byte("adulterado")))
	_, err = valor.MarshalCBOR()
	comprobar("CBOR", err)
	comprobar("CBOR decode", valor.UnmarshalCBOR([]byte("adulterado")))
	_, err = valor.MarshalYAML()
	comprobar("YAML", err)
	comprobar("YAML decode", valor.UnmarshalYAML(func(any) error {
		return nil
	}))
	var gobDestino bytes.Buffer
	comprobar("gob encoder", gob.NewEncoder(&gobDestino).Encode(valor))
	comprobar("JSON decode", json.Unmarshal([]byte(`{}`), valor))
	comprobar("XML decode", xml.Unmarshal([]byte(`<pruebas/>`), valor))
}

func pruebasCanonicasC1ComoLista(
	datos puertosct.DatosPruebasCanonicasOrdenConsumoCobertura,
) [][]byte {
	return [][]byte{
		datos.Peticion,
		datos.Resultado,
		datos.Atestacion,
		datos.ConfirmacionTCB,
		datos.Catalogo,
		datos.Verificador,
		datos.Resumen,
	}
}

func reflectPruebasCanonicasC1Iguales(
	a puertosct.DatosPruebasCanonicasOrdenConsumoCobertura,
	b puertosct.DatosPruebasCanonicasOrdenConsumoCobertura,
) bool {
	pruebasA := pruebasCanonicasC1ComoLista(a)
	pruebasB := pruebasCanonicasC1ComoLista(b)
	for indice := range pruebasA {
		if !bytes.Equal(pruebasA[indice], pruebasB[indice]) {
			return false
		}
	}
	return true
}

func TestSesionTCBErrorTrasIntentarConfirmarSiempreEsAmbiguo(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	for _, caso := range []struct {
		nombre     string
		configurar func(*ejecutorSesionTCBOperacionDecisionPrueba)
	}{
		{
			nombre: "error de confirmar",
			configurar: func(e *ejecutorSesionTCBOperacionDecisionPrueba) {
				e.errorEnSesion = "confirmar"
			},
		},
		{
			nombre: "recibo inválido",
			configurar: func(e *ejecutorSesionTCBOperacionDecisionPrueba) {
				e.recibo.AuditoriaRef = ""
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			transaccion, ejecutor := nuevaTransaccionTCBConfirmacionOrdenC3(
				t,
				escenario.reciboConcedido,
				caso.configurar,
			)
			intento, err :=
				cobertura.IntentarConfirmacionOperacionDecisionCobertura(
					context.Background(),
					transaccion,
					escenario.ordenConcedida,
				)
			if !errors.Is(
				err,
				cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo,
			) || ejecutor.llamadas.Load() != 1 {
				t.Fatalf("resultado posterior mal clasificado: %v", err)
			}
			if _, reconciliar := intento.ReconciliacionPara(
				escenario.ordenConcedida,
			); !reconciliar {
				t.Fatal("la ambigüedad no produjo solicitud primaria")
			}
			if intento.FalloAntesCommitPara(escenario.ordenConcedida) {
				t.Fatal("la confirmación intentada se marcó como pre-COMMIT")
			}
		})
	}
}
