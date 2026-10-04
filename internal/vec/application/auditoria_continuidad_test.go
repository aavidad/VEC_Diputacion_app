package application_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestContinuidadCheckpointFirmasRealesYTramos(t *testing.T) {
	var m, ts [32]byte
	if _, err := rand.Read(m[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(ts[:]); err != nil {
		t.Fatal(err)
	}
	defer clear(m[:])
	defer clear(ts[:])
	p := domain.PoliticaCheckpoint{Version: 1, PoliticaRef: "politica:sintetica", PoliticaVersion: 1, ClaveRef: "clave:sintetica", ClaveVersion: 1,
		ProveedorKMS: "kms:sintetico", ProveedorKMSVersion: 1, ProveedorTSA: "tsa:sintetica", ProveedorTSAVersion: 1, OperacionTSA: "checkpoint.sintetico", Modo: "DESARROLLO"}
	f, err := bootstrap.NuevoProveedorCheckpointDesarrollo(m, ts, p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.CerrarCheckpoint()
	der, err := f.PublicaCheckpointDER()
	if err != nil {
		t.Fatal(err)
	}
	v, err := bootstrap.NuevoVerificadorCheckpointDesarrollo(der, f.PinCheckpoint(), p)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	firmar := func(primera, ultima uint64, anterior, cabeza, cadena string) domain.ReciboCheckpointDesarrollo {
		c := domain.CheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo, Politica: p, Cobertura: domain.CoberturaCheckpoint{
			CadenaID: cadena, PrimeraSecuencia: primera, UltimaSecuencia: ultima, AnteriorSHA256: anterior, CabezaSHA256: cabeza, Registros: ultima - primera + 1}}
		r, e := application.EmitirCheckpointDesarrollo(ctx, c, f, f, 100)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	z, h := strings.Repeat("0", 64), strings.Repeat("1", 64)
	ancla := firmar(1, 2, z, h, "cadena:sintetica")
	siguiente := firmar(3, 4, h, strings.Repeat("2", 64), "cadena:sintetica")
	r := application.VerificarContinuidadCheckpoint(ctx, ancla, []domain.ReciboCheckpointDesarrollo{siguiente}, v, 4, 1)
	if r.Firma != "verificada_con_pin_externo" || r.Continuidad != "verificada" || r.IntegridadCadena != "no_evaluada" || r.OrigenExtraccion != "no_acreditado" || r.TSA != "no_verificada_offline" || r.TiempoIndependiente || r.FirmaLegal {
		t.Fatal(r)
	}
	for _, c := range []domain.ReciboCheckpointDesarrollo{firmar(4, 5, h, h, "cadena:sintetica"), firmar(3, 4, h, h, "cadena:otra"), ancla} {
		r := application.VerificarContinuidadCheckpoint(ctx, ancla, []domain.ReciboCheckpointDesarrollo{c}, v, 10, 1)
		if r.Firma != "verificada_con_pin_externo" || r.Continuidad != "rechazada" {
			t.Fatal("firma correcta confundida con continuidad", r)
		}
	}
	for _, cambiar := range []func(*domain.ReciboCheckpointDesarrollo){
		func(r *domain.ReciboCheckpointDesarrollo) {
			r.FirmaBase64 = base64.StdEncoding.EncodeToString(make([]byte, 64))
		},
		func(r *domain.ReciboCheckpointDesarrollo) {
			r.Checkpoint.Cobertura.CabezaSHA256 = strings.Repeat("7", 64)
		},
		func(r *domain.ReciboCheckpointDesarrollo) { r.PinSPKISHA256 = strings.Repeat("7", 64) },
		func(r *domain.ReciboCheckpointDesarrollo) { r.Checkpoint.Politica.PoliticaVersion++ },
	} {
		c := siguiente
		cambiar(&c)
		if r := application.VerificarContinuidadCheckpoint(ctx, ancla, []domain.ReciboCheckpointDesarrollo{c}, v, 10, 1); r.Firma != "rechazada" || r.Continuidad != "rechazada" {
			t.Fatal("alteración admitida")
		}
		if r := application.VerificarContinuidadCheckpoint(ctx, c, []domain.ReciboCheckpointDesarrollo{siguiente}, v, 10, 1); r.Firma != "rechazada" {
			t.Fatal("ancla no verificada")
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if application.VerificarContinuidadCheckpoint(ctx, ancla, []domain.ReciboCheckpointDesarrollo{siguiente}, v, 10, 1).Continuidad != "rechazada" {
		t.Fatal("cancelación admitida")
	}
}
