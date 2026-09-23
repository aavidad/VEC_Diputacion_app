package domain

import (
	"bytes"
	"testing"
	"time"
)

func TestRegistroPropioExigeFuenteInstitucionalYNoDuplicaIdentidadPorAlias(t *testing.T) {
	ahora := time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC)
	a := AcreditacionInstitucionalRegistroPropioV1{
		CredencialRef: "cre_abcdefghijklmnopqrstuv", SujetoRef: "suj_abcdefghijklmnopqrstuv",
		DominioHMACRef: "idh_abcdefghijklmnopqrstuv", ClaveHMACID: "clave:institucional:prueba", ClaveHMACVersion: 1,
		CuentaHuellaHMAC: bytes.Repeat([]byte{0x31}, 32), SujetoHuellaHMAC: bytes.Repeat([]byte{0x32}, 32),
		EquivalenciaRef: "equ_abcdefghijklmnopqrstuv", ProcedenciaRef: "prc_abcdefghijklmnopqrstuv",
		ProcedenciaVersion: 1, ProcedenciaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ProcedenciaAutoridad: "autoridad_maestra_acreditada", VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	if err := a.ValidarEn(ahora); err != nil {
		t.Fatal(err)
	}
	if !a.MismaAcreditacion(a) {
		t.Fatal("la misma preimagen acreditada debe conservar identidad")
	}
	mutada := a
	mutada.CuentaHuellaHMAC = bytes.Clone(a.SujetoHuellaHMAC)
	if mutada.ValidarEn(ahora) == nil || a.MismaAcreditacion(mutada) {
		t.Fatal("alias de cuenta y sujeto confundidos")
	}
	mutada = a
	mutada.ProcedenciaAutoridad = "no_autoritativa"
	if mutada.ValidarEn(ahora) == nil {
		t.Fatal("procedencia no autoritativa aceptada")
	}
	mutada = a
	mutada.EquivalenciaRef = "equ_bbbbbbbbbbbbbbbbbbbbbb"
	if mutada.ValidarEn(ahora) != nil || a.MismaAcreditacion(mutada) {
		t.Fatal("la prueba de equivalencia debe quedar fija")
	}
	if a.ValidarEn(ahora.Add(time.Hour)) == nil {
		t.Fatal("credencial caducada aceptada")
	}
}
