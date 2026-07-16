package ports

import (
	"crypto/sha256"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func tokenReservaOrdenCobroPrueba(t *testing.T) TokenReservaOrdenCobro {
	t.Helper()
	token, err := NuevoTokenReservaOrdenCobro()
	if err != nil {
		t.Fatalf("generar token de reserva de alta: %v", err)
	}
	return token
}

func tokenReservaDevolucionCobroPrueba(t *testing.T) TokenReservaDevolucionCobro {
	t.Helper()
	token, err := NuevoTokenReservaDevolucionCobro()
	if err != nil {
		t.Fatalf("generar token de reserva de devolucion: %v", err)
	}
	return token
}

func TestTokensReservaCobroSonCapacidadesOpacasNoPersistibles(t *testing.T) {
	const entropiaComun = "0123456789abcdef0123456789abcdef"
	operacionAlta, err := nuevaOperacionCapacidadReservaConFuenteEntropia(
		dominioHuellaTokenAltaCobro, strings.NewReader(entropiaComun),
	)
	if err != nil {
		t.Fatalf("crear operacion de alta determinista: %v", err)
	}
	operacionDevolucion, err := nuevaOperacionCapacidadReservaConFuenteEntropia(
		dominioHuellaTokenDevolucionCobro, strings.NewReader(entropiaComun),
	)
	if err != nil {
		t.Fatalf("crear operacion de devolucion determinista: %v", err)
	}
	tokenAlta := TokenReservaOrdenCobro{operar: operacionAlta}
	otroTokenAlta := tokenReservaOrdenCobroPrueba(t)
	tokenDevolucion := TokenReservaDevolucionCobro{operar: operacionDevolucion}
	huellaAlta, err := tokenAlta.HuellaSHA256()
	if err != nil || len(huellaAlta) != sha256.Size*2 {
		t.Fatalf("huella persistible de alta invalida: %q, %v", huellaAlta, err)
	}
	huellaDevolucion, err := tokenDevolucion.HuellaSHA256()
	if err != nil || len(huellaDevolucion) != sha256.Size*2 {
		t.Fatalf("huella persistible de devolucion invalida: %q, %v", huellaDevolucion, err)
	}
	if !tokenAlta.CoincideConHuellaSHA256(huellaAlta) ||
		otroTokenAlta.CoincideConHuellaSHA256(huellaAlta) ||
		tokenDevolucion.CoincideConHuellaSHA256(huellaAlta) ||
		huellaAlta == huellaDevolucion ||
		tokenAlta.CoincideConHuellaSHA256(strings.ToUpper(huellaAlta)) ||
		tokenAlta.CoincideConHuellaSHA256("no-es-una-huella") {
		t.Fatal("la huella no separo dominios, no rechazo replay o normalizo su forma")
	}
	if (TokenReservaOrdenCobro{}).Valido() || (TokenReservaDevolucionCobro{}).Valido() {
		t.Fatal("el valor cero adquirio autoridad")
	}

	probarOpacidad := func(
		nombre, redaccion string,
		token any,
		destinoJSON json.Unmarshaler,
		destinoTexto encoding.TextUnmarshaler,
		destinoBinario encoding.BinaryUnmarshaler,
	) {
		t.Helper()
		if _, err := json.Marshal(token); !errors.Is(err, ErrSerializacionTokenReservaCobroProhibida) {
			t.Fatalf("%s json.Marshal() error = %v", nombre, err)
		}
		if err := destinoJSON.UnmarshalJSON([]byte(`"filtracion"`)); !errors.Is(err, ErrSerializacionTokenReservaCobroProhibida) {
			t.Fatalf("%s UnmarshalJSON() error = %v", nombre, err)
		}
		serializadorTexto, okTexto := token.(encoding.TextMarshaler)
		serializadorBinario, okBinario := token.(encoding.BinaryMarshaler)
		if !okTexto || !okBinario {
			t.Fatalf("%s no bloquea las codificaciones estándar", nombre)
		}
		if _, err := serializadorTexto.MarshalText(); !errors.Is(err, ErrSerializacionTokenReservaCobroProhibida) {
			t.Fatalf("%s MarshalText() error = %v", nombre, err)
		}
		if _, err := serializadorBinario.MarshalBinary(); !errors.Is(err, ErrSerializacionTokenReservaCobroProhibida) {
			t.Fatalf("%s MarshalBinary() error = %v", nombre, err)
		}
		if err := destinoTexto.UnmarshalText([]byte("filtracion")); !errors.Is(err, ErrSerializacionTokenReservaCobroProhibida) {
			t.Fatalf("%s UnmarshalText() error = %v", nombre, err)
		}
		if err := destinoBinario.UnmarshalBinary([]byte("filtracion")); !errors.Is(err, ErrSerializacionTokenReservaCobroProhibida) {
			t.Fatalf("%s UnmarshalBinary() error = %v", nombre, err)
		}
		for _, formato := range []string{"%s", "%v", "%+v", "%#v", "%q", "%x"} {
			if obtenido := fmt.Sprintf(formato, token); obtenido != redaccion {
				t.Fatalf("%s con %s = %q", nombre, formato, obtenido)
			}
		}
		logValuer, ok := token.(interface{ LogValue() slog.Value })
		if !ok || logValuer.LogValue().String() != redaccion {
			t.Fatalf("%s no redacta slog", nombre)
		}
		metodos := reflect.TypeOf(token)
		for indice := 0; indice < metodos.NumMethod(); indice++ {
			if strings.Contains(strings.ToLower(metodos.Method(indice).Name), "revelar") {
				t.Fatalf("%s expone el metodo %s", nombre, metodos.Method(indice).Name)
			}
		}
	}
	var destinoAlta TokenReservaOrdenCobro
	probarOpacidad(
		"alta", tokenAlta.String(), tokenAlta,
		&destinoAlta, &destinoAlta, &destinoAlta,
	)
	var destinoDevolucion TokenReservaDevolucionCobro
	probarOpacidad(
		"devolucion", tokenDevolucion.String(), tokenDevolucion,
		&destinoDevolucion, &destinoDevolucion, &destinoDevolucion,
	)
}
