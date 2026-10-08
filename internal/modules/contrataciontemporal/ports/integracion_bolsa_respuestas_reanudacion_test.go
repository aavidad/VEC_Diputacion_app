package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestReciboLlamamientoRecuperadoCanonAnteriorIntacto(t *testing.T) {
	base := instanteBolsaPrueba()
	c := comandoLlamamientoPrueba(t, base, selladorRespuestaBolsaPrueba())
	r := reciboLlamamientoPrueba(t, c, base)
	anterior := materialReciboLlamamientoBolsa(c, r)
	const huellaAnterior = "eb6df2bc687be0fd6b7b7c795b53aa172a8034bf78f50bb210f71ed9c793ae47"
	if len(anterior) != 4466 || fmt.Sprintf("%x", sha256.Sum256(anterior)) != huellaAnterior {
		t.Fatal("cambiaron bytes del recibo anterior sin marca")
	}
	b, err := json.Marshal(r)
	if err != nil || bytes.Contains(b, []byte("llamamiento_recuperado")) {
		t.Fatal("marca false modificó JSON histórico")
	}
	r.LlamamientoRecuperado = true
	extension := constructorCanonicoBolsa{}
	extension.booleano("recibo_llamamiento_recuperado", true)
	if !bytes.Equal(materialReciboLlamamientoBolsa(c, r), append(anterior, extension.bytes()...)) {
		t.Fatal("extensión canónica incorrecta")
	}
	b, err = json.Marshal(r)
	if err != nil || !bytes.Contains(b, []byte(`"llamamiento_recuperado":true`)) {
		t.Fatal("marca ausente del recibo recuperado")
	}
}

func TestReciboLlamamientoRecuperadoAtestacionFresca(t *testing.T) {
	ctx, base := context.Background(), instanteBolsaPrueba()
	c := comandoLlamamientoPrueba(t, base, selladorRespuestaBolsaPrueba())
	r := reciboLlamamientoPrueba(t, c, base)
	ahora := base.Add(3 * time.Minute)
	r.ConfirmadaEn = base.Add(-time.Minute)
	if r.ValidarParaEn(c, ahora) == nil {
		t.Fatal("fecha anterior sin marca fue admitida")
	}
	r.LlamamientoRecuperado = true
	emisor, err := NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaBolsaPrueba, claveRespuestaBolsaV1Prueba, selladorRespuestaBolsaPrueba())
	if err != nil {
		t.Fatal(err)
	}
	firmado, err := emisor.FirmarLlamamiento(ctx, c, r, ahora)
	if err != nil || firmado.ConfirmadaEn != r.ConfirmadaEn {
		t.Fatal("perdió fecha original:", err)
	}
	v := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	prueba, evidencia, err := v.VerificarReciboLlamamiento(ctx, c, firmado, ahora)
	if err != nil {
		t.Fatal(err)
	}
	artefacto, err := NuevoArtefactoProbatorioLlamamientoBolsa(c, firmado, evidencia, prueba)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(artefacto)
	if err != nil {
		t.Fatal(err)
	}
	var recuperado ArtefactoProbatorioLlamamientoBolsa
	if err = json.Unmarshal(b, &recuperado); err != nil {
		t.Fatal("rehidratación:", err)
	}
	for nombre, alterar := range map[string]func(*ReciboSolicitudLlamamientoBolsa){
		"UTC": func(r *ReciboSolicitudLlamamientoBolsa) {
			r.ConfirmadaEn = r.ConfirmadaEn.In(time.FixedZone("otra", 0))
		},
		"futuro":             func(r *ReciboSolicitudLlamamientoBolsa) { r.ConfirmadaEn = ahora.Add(time.Second) },
		"operacion":          func(r *ReciboSolicitudLlamamientoBolsa) { r.OperacionRef = "operacion:distinta" },
		"fecha no anterior":  func(r *ReciboSolicitudLlamamientoBolsa) { r.ConfirmadaEn = base },
		"sin propuesta":      func(r *ReciboSolicitudLlamamientoBolsa) { r.PropuestaGenerada = false },
		"evidencia anterior": func(r *ReciboSolicitudLlamamientoBolsa) { r.Procedencia.Evidencia.EmitidaEn = base.Add(-time.Second) },
	} {
		t.Run(nombre, func(t *testing.T) {
			mutado := firmado
			alterar(&mutado)
			if mutado.ValidarParaEn(c, ahora) == nil {
				t.Fatal("marca eludió guarda")
			}
		})
	}
	if firmado.ValidarParaEn(c, base.Add(24*time.Hour)) == nil {
		t.Fatal("marca reabrió petición caducada")
	}
}

func TestReciboLlamamientoRecuperadoMarcaCubiertaPorHMAC(t *testing.T) {
	ctx, base := context.Background(), instanteBolsaPrueba()
	c := comandoLlamamientoPrueba(t, base, selladorRespuestaBolsaPrueba())
	ahora := base.Add(3 * time.Minute)
	emisor, err := NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaBolsaPrueba, claveRespuestaBolsaV1Prueba, selladorRespuestaBolsaPrueba())
	if err != nil {
		t.Fatal(err)
	}
	for _, marca := range []bool{false, true} {
		r := reciboLlamamientoPrueba(t, c, base)
		r.LlamamientoRecuperado = marca
		if marca {
			r.ConfirmadaEn = base.Add(-time.Minute)
		}
		firmado, err := emisor.FirmarLlamamiento(ctx, c, r, ahora)
		if err != nil {
			t.Fatal(err)
		}
		firmado.LlamamientoRecuperado = !marca
		// Aísla criptografía de la guarda temporal: alterar la marca también
		// debe invalidar el HMAC aun si el validador nominal ya la rechazaría.
		datos, _ := c.datosCanonicos()
		contexto, _ := datos.Contexto.datosDurables()
		if _, _, err = verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba).verificarFresco(ctx, "recibo_llamamiento", contexto.OperacionRef, materialComandoLlamamientoBolsa(c), materialReciboLlamamientoBolsa(c, firmado), firmado.Procedencia, ahora); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
			t.Fatal("marca alterada conserva firma:", err)
		}

	}
}
