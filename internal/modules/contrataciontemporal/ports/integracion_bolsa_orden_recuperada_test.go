package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestReciboOrdenRecuperadaCanonAnteriorIntacto(t *testing.T) {
	base := instanteBolsaPrueba()
	comando, recibo := comandoOrdenPrueba(t, base), reciboOrdenPrueba(t, base)
	anterior := materialReciboOrdenBolsa(comando, recibo)
	// Capturada ejecutando el canon ANTES de añadir OrdenRecuperada.
	const huellaAnterior = "f0a39f903beb3c2dedc8c239f637ac833f902fafe3041143d5751b765fdc467a"
	if len(anterior) != 3630 || fmt.Sprintf("%x", sha256.Sum256(anterior)) != huellaAnterior {
		t.Fatal("cambiaron los bytes de un recibo anterior sin marca")
	}
	b, err := json.Marshal(recibo)
	if err != nil || bytes.Contains(b, []byte("orden_recuperada")) {
		t.Fatal("false debe permanecer ausente del JSON anterior")
	}
	recibo.OrdenRecuperada = true
	extension := constructorCanonicoBolsa{}
	extension.booleano("recibo_orden_recuperada", true)
	if !bytes.Equal(materialReciboOrdenBolsa(comando, recibo), append(anterior, extension.bytes()...)) {
		t.Fatal("true debe añadir exclusivamente el campo cubierto por HMAC")
	}
	b, err = json.Marshal(recibo)
	if err != nil || !bytes.Contains(b, []byte(`"orden_recuperada":true`)) {
		t.Fatal("la recuperación debe quedar explícita en JSON")
	}
}

func TestReciboOrdenRecuperadaAtestacionFresca(t *testing.T) {
	ctx, base := context.Background(), instanteBolsaPrueba()
	comando, recibo := comandoOrdenPrueba(t, base), reciboOrdenPrueba(t, base)
	ahora := base.Add(3 * time.Minute)
	recibo.ConfirmadaEn = base.Add(-time.Minute)
	original := recibo.ConfirmadaEn
	if recibo.ValidarParaEn(comando, ahora) == nil {
		t.Fatal("sin marca no debe admitirse una confirmación anterior")
	}
	recibo.OrdenRecuperada = true
	emisor, err := NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaBolsaPrueba, claveRespuestaBolsaV1Prueba, selladorRespuestaBolsaPrueba())
	if err != nil {
		t.Fatal(err)
	}
	firmado, err := emisor.FirmarOrden(ctx, comando, recibo, ahora)
	if err != nil || firmado.ConfirmadaEn != original || !firmado.OrdenRecuperada {
		t.Fatal("la atestación fresca debe conservar la confirmación original:", err)
	}
	if _, _, err := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba).VerificarReciboOrden(ctx, comando, firmado, ahora); err != nil {
		t.Fatal(err)
	}
	for nombre, alterar := range map[string]func(*ReciboOrdenBolsa){
		"UTC":                func(r *ReciboOrdenBolsa) { r.ConfirmadaEn = r.ConfirmadaEn.In(time.FixedZone("no-canonica", 0)) },
		"futuro":             func(r *ReciboOrdenBolsa) { r.ConfirmadaEn = ahora.Add(time.Second) },
		"evidencia anterior": func(r *ReciboOrdenBolsa) { r.Procedencia.Evidencia.EmitidaEn = base.Add(-time.Second) },
		"operacion":          func(r *ReciboOrdenBolsa) { r.OperacionRef = "operacion:distinta" },
		"necesidad":          func(r *ReciboOrdenBolsa) { r.Necesidad = referenciaBolsaPrueba("necesidad:distinta", "f") },
		"sin orden":          func(r *ReciboOrdenBolsa) { r.OrdenGenerada = false },
	} {
		t.Run(nombre, func(t *testing.T) {
			r := firmado
			alterar(&r)
			if r.ValidarParaEn(comando, ahora) == nil {
				t.Fatal("la marca eludió una guarda vigente")
			}
		})
	}
	if firmado.ValidarParaEn(comando, base.Add(24*time.Hour)) == nil {
		t.Fatal("la recuperación no permite usar una petición caducada")
	}
}

func TestReciboOrdenRecuperadaMarcaCubiertaPorHMAC(t *testing.T) {
	ctx, base := context.Background(), instanteBolsaPrueba()
	comando := comandoOrdenPrueba(t, base)
	ahora := base.Add(3 * time.Minute)
	emisor, err := NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaBolsaPrueba, claveRespuestaBolsaV1Prueba, selladorRespuestaBolsaPrueba())
	if err != nil {
		t.Fatal(err)
	}
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	for _, marca := range []bool{false, true} {
		r := reciboOrdenPrueba(t, base)
		r.OrdenRecuperada = marca
		firmado, err := emisor.FirmarOrden(ctx, comando, r, ahora)
		if err != nil {
			t.Fatal(err)
		}
		firmado.OrdenRecuperada = !marca
		// Ambos estados son nominalmente válidos con esta fecha: el rechazo
		// debe provenir de la autenticación, no de la guarda temporal.
		if firmado.ValidarParaEn(comando, ahora) != nil {
			t.Fatal("fixture de alteración no aísla la autenticación")
		}
		if _, _, err := verificador.VerificarReciboOrden(ctx, comando, firmado, ahora); err == nil {
			t.Fatal("cambiar la marca conservó indebidamente una firma válida")
		}
	}
}
