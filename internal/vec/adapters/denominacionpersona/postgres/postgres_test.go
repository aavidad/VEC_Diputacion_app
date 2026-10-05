package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/vec/ports"
)

func preparacion(t *testing.T) ports.PreparacionDenominacionPersona {
	t.Helper()
	// Transporte sintético: no se presenta como cifrado acreditado por KMS.
	s := ports.SobreDenominacionPersona{Esquema: "vec.persona.denominacion.aead.v1", PersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ClaveRef: "kms:denominacion:cifrado:1", Version: 1, Nonce: make([]byte, 12), Cifrado: make([]byte, 17), Indice: ports.IndiceDenominacionPersona{AmbitoRef: "ambito:denominacion:unidad:1", NormaRef: "norma:denominacion:1", NormaSHA256: strings.Repeat("a", 64), ClaveRef: "kms:denominacion:indice:1", Tokens: [][]byte{make([]byte, 32)}}}
	b, e := sobreCanonico(s)
	if e != nil {
		t.Fatal(e)
	}
	return ports.PreparacionDenominacionPersona{PersonaRef: s.PersonaRef, ProcedenciaRef: "prc_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SobreSHA256: sha(b), Sobre: s}
}
func TestMaterialLigaSobreFuenteYAmbitosSinNombre(t *testing.T) {
	p := preparacion(t)
	proc := Procedencia{Ref: p.ProcedenciaRef, Version: 1, HuellaSHA256: strings.Repeat("b", 64), Autoridad: "no_autoritativa"}
	b, r, e := MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica")
	if e != nil {
		t.Fatal(e)
	}
	again, againR, e := MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica")
	if e != nil || string(b) != string(again) || r.Atributos["material_sha256"] != againR.Atributos["material_sha256"] {
		t.Fatal("canon_inestable")
	}
	if strings.Contains(string(b), "nombre") || r.Atributos["procedencia_version"] != "1" {
		t.Fatal("material_no_opaco")
	}
	proc.Version = 2
	changed, _, e := MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica")
	if e != nil || sha(changed) == sha(b) {
		t.Fatal("fuente_no_ligada")
	}
	changed, _, e = MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_distinta")
	if e != nil || sha(changed) == sha(b) {
		t.Fatal("ambito_no_ligado")
	}
	p.Sobre.Nonce[0] = 1
	if _, _, e = MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica"); e == nil {
		t.Fatal("nonce_sustituido")
	}
}
func TestFronteraMaterialRechazaVersionTokensFuenteYCampoExtra(t *testing.T) {
	p := preparacion(t)
	proc := Procedencia{Ref: p.ProcedenciaRef, Version: 1, HuellaSHA256: strings.Repeat("b", 64), Autoridad: "no_autoritativa"}
	p.Sobre.Indice.Tokens = append(p.Sobre.Indice.Tokens, p.Sobre.Indice.Tokens[0])
	if _, _, e := MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica"); e == nil {
		t.Fatal("token_duplicado")
	}
	p = preparacion(t)
	proc.Version = 0
	if _, _, e := MaterialPublicacion(p, proc, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica"); e == nil {
		t.Fatal("fuente_cero")
	}
	if _, _, e := MaterialLectura(p.PersonaRef, 1<<53, "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unidad_admin_sintetica"); e == nil {
		t.Fatal("version_inexacta")
	}
	var x lectura
	if decodificar([]byte(`{"esquema":"x","persona_ref":"p","version":1,"ambitos":{"organizacion_ref":"o","unidad_ref":"u"},"nombre":"Ana Valdivia"}`), &x) == nil {
		t.Fatal("campo_extra")
	}
	var s ports.SobreDenominacionPersona
	raw, _ := json.Marshal(p.Sobre)
	if decodificar(raw, &s) != nil || s.Version != 1 {
		t.Fatal("transporte_sobre")
	}
}

type poolPrueba struct {
	tx    *txPrueba
	opts  pgx.TxOptions
	begin int
}

func (p *poolPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.opts = o
	p.begin++
	return p.tx, nil
}

type txPrueba struct {
	pgx.Tx
	commitError        error
	commits, rollbacks int
	exec               string
}

func (t *txPrueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	t.exec = q
	return pgconn.CommandTag{}, nil
}
func (t *txPrueba) Commit(context.Context) error   { t.commits++; return t.commitError }
func (t *txPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }
func TestTransaccionNoConfirmaResultadoConCommitIncierto(t *testing.T) {
	tx := &txPrueba{commitError: errors.New("commit_indeterminado")}
	pool := &poolPrueba{tx: tx}
	a := &Adaptador{pool: pool}
	e := a.transaccion(context.Background(), func(pgx.Tx) error { return nil })
	if e == nil || tx.commits != 1 || tx.rollbacks != 1 || pool.opts.IsoLevel != pgx.Serializable || pool.opts.AccessMode != pgx.ReadWrite || !strings.Contains(tx.exec, "timezone='UTC'") {
		t.Fatal("commit_incierto_con_resultado")
	}
	tx = &txPrueba{}
	pool.tx = tx
	e = a.transaccion(context.Background(), func(pgx.Tx) error { return ErrNoDisponible })
	if e == nil || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("efecto_fallido_confirmado")
	}
}
func TestEvidenciaAusenteCierraAntesDeBaseYBusqueda(t *testing.T) {
	pool := &poolPrueba{tx: &txPrueba{}}
	a := &Adaptador{pool: pool}
	if r, e := a.LeerDenominacionPersonaAutorizada(context.Background(), ports.AccesoDenominacionPersona{}); e == nil || r.Sobre.PersonaRef != "" || pool.begin != 0 {
		t.Fatal("lectura_sin_evidencia")
	}
	if refs, e := a.BuscarDenominacionPersonaAutorizada(context.Background(), ports.AccesoDenominacionPersona{}, ports.IndiceDenominacionPersona{}, 10); e == nil || refs != nil || pool.begin != 0 {
		t.Fatal("busqueda_no_cerrada")
	}
	if _, e := nuevo(context.Background(), pool, nil); e == nil || pool.begin != 0 {
		t.Fatal("constructor_fingido")
	}
}
