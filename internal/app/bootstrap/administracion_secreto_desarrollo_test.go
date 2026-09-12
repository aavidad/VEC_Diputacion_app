package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestProtectorSecretoCorreoDesarrolloCifraConAADDurableYEntregaEfimera(t *testing.T) {
	clave := [32]byte{1}
	protector, err := nuevoProtectorSecretoCorreoDesarrollo(clave, bytes.NewReader(bytes.Repeat([]byte{7}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	aad := aadSecretoCorreoPrueba(t, 9)
	claro := []byte("secreto-sintetico")
	sobre, err := protector.CifrarSecretoCorreo(context.Background(), claro, aad)
	if err != nil {
		t.Fatal(err)
	}
	if sobre.Version != 9 || sobre.ClaveRef != claveSecretoCorreoDesarrolloRef || bytes.Equal(sobre.Cifrado, claro) || len(sobre.Nonce) != 12 {
		t.Fatal("sobre de secreto invalido")
	}
	var entregado []byte
	if err := protector.ConSecretoCorreoDescifrado(context.Background(), sobre, aad, func(v []byte) error {
		entregado = v
		if !bytes.Equal(v, claro) {
			return errors.New("claro inesperado")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(entregado, make([]byte, len(entregado))) {
		t.Fatal("el buffer claro no se borro tras el callback")
	}
}

func TestProtectorSecretoCorreoDesarrolloRechazaSobreOAADManipulados(t *testing.T) {
	protector, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{2}, bytes.NewReader(bytes.Repeat([]byte{8}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	aad := aadSecretoCorreoPrueba(t, 4)
	sobre, err := protector.CifrarSecretoCorreo(context.Background(), []byte("secreto-sintetico"), aad)
	if err != nil {
		t.Fatal(err)
	}
	// La versión forma parte del AAD canónico y no puede intercambiarse.
	if err := protector.ConSecretoCorreoDescifrado(context.Background(), sobre, aadSecretoCorreoPrueba(t, 5), func([]byte) error { return nil }); err == nil {
		t.Fatal("se admitio AAD de otra version")
	}
	alterado := sobre
	alterado.ClaveRef = "clave:kms:desarrollo:ajena:v1"
	if err := protector.ConSecretoCorreoDescifrado(context.Background(), alterado, aad, func([]byte) error { return nil }); err == nil {
		t.Fatal("se admitio clave ajena")
	}
	alterado = sobre
	alterado.Cifrado = append([]byte(nil), sobre.Cifrado...)
	alterado.Cifrado[0] ^= 1
	if err := protector.ConSecretoCorreoDescifrado(context.Background(), alterado, aad, func([]byte) error { return nil }); err == nil {
		t.Fatal("se admitio cifrado manipulado")
	}
	if _, err := protector.CifrarSecretoCorreo(context.Background(), []byte("secreto"), []byte(`{"esquema":"ajeno","referencia":"configuracion:smtp:diputacion","version":4}`)); err == nil {
		t.Fatal("se admitio AAD ajeno")
	}
}

func TestProtectorSecretoCorreoDesarrolloRechazaDependenciaYNoExponeErrorCallback(t *testing.T) {
	if _, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{}, bytes.NewReader(bytes.Repeat([]byte{3}, 16))); err == nil {
		t.Fatal("se admitio clave maestra cero")
	}
	if _, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{3}, nil); err == nil {
		t.Fatal("se admitio aleatorio nulo")
	}
	var lectorNulo *lectorAleatorioNulo
	if _, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{3}, lectorNulo); err == nil {
		t.Fatal("se admitio aleatorio typed-nil")
	}
	protector, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{3}, bytes.NewReader(bytes.Repeat([]byte{9}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	aad := aadSecretoCorreoPrueba(t, 2)
	sobre, err := protector.CifrarSecretoCorreo(context.Background(), []byte("secreto-sintetico"), aad)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("no debe propagarse")
	if err := protector.ConSecretoCorreoDescifrado(context.Background(), sobre, aad, func([]byte) error { return sentinel }); !errors.Is(err, ErrKMSDesarrolloNoDisponible) || errors.Is(err, sentinel) {
		t.Fatalf("error callback expuesto: %v", err)
	}
}

func TestProtectorSecretoCorreoDesarrolloRepresentacionesRedactadas(t *testing.T) {
	protector, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{10}, bytes.NewReader(bytes.Repeat([]byte{11}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []any{protector, *protector} {
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if texto != "bootstrap.protectorSecretoCorreoDesarrollo{redactado}|bootstrap.protectorSecretoCorreoDesarrollo{redactado}|bootstrap.protectorSecretoCorreoDesarrollo{redactado}" || strings.Contains(texto, "clave:") || strings.Contains(texto, "aleatorio:") {
			t.Fatalf("representacion no redactada: %q", texto)
		}
		serializado, err := json.Marshal(valor)
		if err != nil || string(serializado) != `{"redactado":true}` {
			t.Fatalf("json no redactado: %q, %v", serializado, err)
		}
	}
}

type lectorAleatorioNulo struct{}

func (*lectorAleatorioNulo) Read([]byte) (int, error) { return 0, nil }

func TestProtectorSecretoCorreoDesarrolloReutilizaClaveSinReutilizarNonce(t *testing.T) {
	maestra := [32]byte{4}
	aleatorio := append(bytes.Repeat([]byte{1}, 12), bytes.Repeat([]byte{2}, 12)...)
	protector, err := nuevoProtectorSecretoCorreoDesarrollo(maestra, bytes.NewReader(aleatorio))
	if err != nil {
		t.Fatal(err)
	}
	aad := aadSecretoCorreoPrueba(t, 7)
	claro := []byte("secreto-sintetico")
	sobreUno, err := protector.CifrarSecretoCorreo(context.Background(), claro, aad)
	if err != nil {
		t.Fatal(err)
	}
	sobreDos, err := protector.CifrarSecretoCorreo(context.Background(), claro, aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(sobreUno.Nonce, sobreDos.Nonce) || !bytes.Equal(claro, []byte("secreto-sintetico")) ||
		!bytes.Equal(aad, aadSecretoCorreoPrueba(t, 7)) {
		t.Fatal("cifrado altero entrada o reutilizo nonce")
	}
	reconstruido, err := nuevoProtectorSecretoCorreoDesarrollo(maestra, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := reconstruido.ConSecretoCorreoDescifrado(context.Background(), sobreUno, aad, func(v []byte) error {
		if !bytes.Equal(v, claro) {
			return errors.New("claro inesperado")
		}
		return nil
	}); err != nil {
		t.Fatalf("la misma maestra no abre su sobre: %v", err)
	}
	ajeno, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{5}, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := ajeno.ConSecretoCorreoDescifrado(context.Background(), sobreUno, aad, func([]byte) error { return nil }); err == nil {
		t.Fatal("otra maestra abrio el sobre")
	}
}

func TestProtectorSecretoCorreoDesarrolloRechazaAleatorioCortoYCierreDuranteCallback(t *testing.T) {
	aad := aadSecretoCorreoPrueba(t, 3)
	corto, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{6}, bytes.NewReader([]byte{1, 2, 3}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := corto.CifrarSecretoCorreo(context.Background(), []byte("secreto-sintetico"), aad); err == nil {
		t.Fatal("se admitio aleatorio corto")
	}
	protector, err := nuevoProtectorSecretoCorreoDesarrollo([32]byte{6}, bytes.NewReader(bytes.Repeat([]byte{4}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := protector.CifrarSecretoCorreo(context.Background(), []byte("secreto-sintetico"), aad)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	var entregado []byte
	if err := protector.ConSecretoCorreoDescifrado(ctx, sobre, aad, func(v []byte) error {
		entregado = v
		cancelar()
		return nil
	}); !errors.Is(err, ErrKMSDesarrolloNoDisponible) {
		t.Fatalf("cancelacion=%v", err)
	}
	if !bytes.Equal(entregado, make([]byte, len(entregado))) {
		t.Fatal("el callback cancelado retuvo el claro")
	}
}

func aadSecretoCorreoPrueba(t *testing.T, version uint64) []byte {
	t.Helper()
	aad, err := json.Marshal(struct {
		Esquema, Referencia string
		Version             uint64
	}{esquemaSecretoCorreo, referenciaSecretoCorreo, version})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := versionAADSecretoCorreo(aad); !ok {
		t.Fatal("AAD de prueba invalido")
	}
	return aad
}

func TestVersionAADSecretoCorreoAceptaSoloVectorCanonicoDelAdaptador(t *testing.T) {
	const vector = `{"Esquema":"vec.administracion.configuracion-correo.secreto.v1","Referencia":"configuracion:smtp:diputacion","Version":19}`
	if version, ok := versionAADSecretoCorreo([]byte(vector)); !ok || version != 19 {
		t.Fatalf("AAD canónico del adaptador rechazado: %d,%t", version, ok)
	}
	if _, ok := versionAADSecretoCorreo([]byte(`{"esquema":"vec.administracion.configuracion-correo.secreto.v1","referencia":"configuracion:smtp:diputacion","version":19}`)); ok {
		t.Fatal("se admitio una variante no canónica del AAD")
	}
}
