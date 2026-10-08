package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestSeudonimosPresentacionCertificadoExigenEpochYPurposesSeparados(t *testing.T) {
	c := CoordenadasPresentacionCertificado{
		EspacioIdentidad: "https://identidad.ejemplo.test", DominioRef: "idh_0123456789abcdefghijkl",
		ClaveID: "clave-presentacion", ClaveVersion: 3,
	}
	d := httpseguridad.DatosPresentacionCertificado{Emisor: c.EspacioIdentidad}
	s := SeudonimosPresentacionCertificado{
		Esquema: EsquemaHMACSHA256V1, EspacioIdentidad: c.EspacioIdentidad,
		DominioRef: c.DominioRef, ClaveID: c.ClaveID, ClaveVersion: c.ClaveVersion,
		AsercionIDHMAC: [32]byte{1}, SesionIDHMAC: [32]byte{2}, SujetoIDHMAC: [32]byte{3},
		CuentaIDHMAC: [32]byte{4}, CertificadoDERHMAC: [32]byte{5}, CAHMAC: [32]byte{6}, NonceHMAC: [32]byte{7},
	}
	if !s.validarPara(c, d) {
		t.Fatal("epoch valido rechazado")
	}
	rotada := s
	rotada.ClaveVersion++
	if rotada.validarPara(c, d) {
		t.Fatal("rotacion no gobernada aceptada como nueva identidad")
	}
	reusada := s
	reusada.NonceHMAC = s.SesionIDHMAC
	if reusada.validarPara(c, d) {
		t.Fatal("nonce reutilizo proposito de sesion")
	}
}

func TestConsultaResultadoPresentacionConservaHistoriaYReciboActual(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	fila := []any{
		"aut_0123456789abcdefghijkl", strings.Repeat("a", 64),
		"ase_0123456789abcdefghijkl", "ses_0123456789abcdefghijkl",
		"cse_0123456789abcdefghijkl", "2", strings.Repeat("b", 64),
		"cta_0123456789abcdefghijkl", "cta_0123456789abcdefghijkl", false,
		"interna_corporativa", "certificado", "sustancial",
		"pga_0123456789abcdefghijkl", strings.Repeat("c", 64),
		ahora.Add(-time.Minute), ahora.Add(-time.Minute), ahora.Add(time.Minute), ahora,
		"prs_0123456789abcdefghijkl", int64(4), strings.Repeat("d", 64),
		ahora, ahora.Add(30 * time.Second), strings.Repeat("e", 64), strings.Repeat("f", 64),
		"opr_0123456789abcdefghijkl", int64(2), "reanudada",
	}
	tx := &transaccionDoble{filas: [][]any{fila}}
	r, err := consultarResultadoPresentacion(context.Background(), tx, consultaReanudarPresentacion, nil)
	if err != nil || r.SesionOriginal.Validar() != nil || r.SesionOriginal.ControlSesionRevision != 2 ||
		r.Recibo.SesionGeneracion != 2 || r.Recibo.Generacion != 4 ||
		r.Recibo.ModoInicio != "reanudada" || len(tx.consultas) != 1 {
		t.Fatalf("resultado original/presentacion mal proyectado: %v", err)
	}
	fila[5] = "0"
	tx = &transaccionDoble{filas: [][]any{fila}}
	if _, err := consultarResultadoPresentacion(context.Background(), tx, consultaReanudarPresentacion, nil); err == nil {
		t.Fatal("revision original invalida aceptada")
	}
}
