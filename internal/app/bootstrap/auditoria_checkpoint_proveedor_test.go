package bootstrap

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

func politicaCheckpointPrueba() domain.PoliticaCheckpoint {
	return domain.PoliticaCheckpoint{Version: 1, PoliticaRef: "politica:sintetica", PoliticaVersion: 2, ClaveRef: "clave:auditoria:sintetica", ClaveVersion: 3, ProveedorKMS: "kms:sintetico", ProveedorKMSVersion: 4, ProveedorTSA: "tsa:sintetica", ProveedorTSAVersion: 5, OperacionTSA: "checkpoint.sintetico", Modo: "DESARROLLO"}
}
func checkpointPrueba() domain.CheckpointDesarrollo {
	z := strings.Repeat("0", 64)
	return domain.CheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo, Politica: politicaCheckpointPrueba(), Cobertura: domain.CoberturaCheckpoint{CadenaID: "cadena:sintetica", AnteriorSHA256: z, CabezaSHA256: z}}
}
func TestCheckpointDesarrolloAutenticidadSeparada(t *testing.T) {
	var maestra, tsa [32]byte
	maestra[0] = 11
	tsa[0] = 22
	p, e := NuevoProveedorCheckpointDesarrollo(maestra, tsa, politicaCheckpointPrueba())
	if e != nil {
		t.Fatal(e)
	}
	defer p.CerrarCheckpoint()
	ctx := context.Background()
	r, e := application.EmitirCheckpointDesarrollo(ctx, checkpointPrueba(), p, p, 10)
	if e != nil {
		t.Fatal(e)
	}
	der, e := p.PublicaCheckpointDER()
	if e != nil {
		t.Fatal(e)
	}
	v, e := NuevoVerificadorCheckpointDesarrollo(der, p.PinCheckpoint(), politicaCheckpointPrueba())
	if e != nil {
		t.Fatal(e)
	}
	o := application.VerificarCheckpointDesarrollo(ctx, r, v, 10)
	if o.Firma != "verificada_con_pin_externo" || o.IntegridadCadena != "no_evaluada" || o.OrigenExtraccion != "no_acreditado" || o.TSA != "no_verificada_offline" || o.TiempoIndependiente || o.FirmaLegal || o.Modo != "DESARROLLO" {
		t.Fatalf("resultado: %+v", o)
	}
	for nombre, mutar := range map[string]func(*domain.ReciboCheckpointDesarrollo){
		"hash": func(r *domain.ReciboCheckpointDesarrollo) {
			r.Checkpoint.Cobertura.CabezaSHA256 = strings.Repeat("1", 64)
		},
		"cadena":    func(r *domain.ReciboCheckpointDesarrollo) { r.Checkpoint.Cobertura.CadenaID = "cadena:otra" },
		"politica":  func(r *domain.ReciboCheckpointDesarrollo) { r.Checkpoint.Politica.PoliticaVersion++ },
		"proveedor": func(r *domain.ReciboCheckpointDesarrollo) { r.Checkpoint.Politica.ProveedorKMS = "kms:otro" },
		"modo":      func(r *domain.ReciboCheckpointDesarrollo) { r.Checkpoint.Politica.Modo = "PRODUCCION" },
		"tsa": func(r *domain.ReciboCheckpointDesarrollo) {
			r.TSA.Referencia = "tsa-desarrollo:hmac-sha256:" + strings.Repeat("0", 64)
		},
		"firma": func(r *domain.ReciboCheckpointDesarrollo) {
			r.FirmaBase64 = base64.StdEncoding.EncodeToString(make([]byte, 64))
		},
		"pin": func(r *domain.ReciboCheckpointDesarrollo) { r.PinSPKISHA256 = strings.Repeat("0", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := r
			mutar(&c)
			if v.VerificarCheckpoint(ctx, c) == nil {
				t.Fatal("alteración admitida")
			}
			rechazo := application.VerificarCheckpointDesarrollo(ctx, c, v, 10)
			if rechazo.Firma != "rechazada" || rechazo.FirmaLegal || rechazo.TiempoIndependiente {
				t.Fatalf("informe de rechazo: %+v", rechazo)
			}
		})
	}
	// Una clave de otro propósito del mismo KMS no autentica el checkpoint.
	envoltura := derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1")
	defer clear(envoltura[:])
	otra := derivarClaveDesarrollo(envoltura, dominioOrdenesCopiasDesarrollo)
	defer clear(otra[:])
	privada := ed25519.NewKeyFromSeed(otra[:])
	defer clear(privada)
	b, _ := r.CanonicoParaFirma()
	r.FirmaBase64 = base64.StdEncoding.EncodeToString(ed25519.Sign(privada, b))
	if v.VerificarCheckpoint(ctx, r) == nil {
		t.Fatal("firma de copias aceptada")
	}
}
func TestCheckpointFirmadorRestringeCargaYProveedorTSA(t *testing.T) {
	var m, ts [32]byte
	m[0] = 1
	ts[0] = 2
	p, _ := NuevoProveedorCheckpointDesarrollo(m, ts, politicaCheckpointPrueba())
	defer p.CerrarCheckpoint()
	ctx := context.Background()
	c := checkpointPrueba()
	s, _ := p.SellarCheckpoint(ctx, c)
	r := domain.ReciboCheckpointDesarrollo{Checkpoint: c, TSA: s, PinSPKISHA256: p.PinCheckpoint()}
	r.TSA.Referencia = "tsa-desarrollo:hmac-sha256:" + strings.Repeat("0", 64)
	if _, e := p.FirmarCheckpoint(ctx, r); e == nil {
		t.Fatal("TSA ajena firmada")
	}
	c.Cobertura.PrimeraSecuencia = 1
	c.Cobertura.UltimaSecuencia = 2
	c.Cobertura.Registros = 1
	if _, e := application.EmitirCheckpointDesarrollo(ctx, c, p, p, 10); e == nil {
		t.Fatal("rango inconsistente firmado")
	}
	c = checkpointPrueba()
	c.Politica.ClaveVersion++
	if _, e := application.EmitirCheckpointDesarrollo(ctx, c, p, p, 10); e == nil {
		t.Fatal("otra política firmada")
	}
	if _, e := NuevoProveedorCheckpointDesarrollo(m, m, politicaCheckpointPrueba()); e == nil {
		t.Fatal("secreto TSA compartido")
	}
	p.CerrarCheckpoint()
	if _, e := p.FirmarCheckpoint(ctx, r); e == nil {
		t.Fatal("firma tras cierre")
	}
}
