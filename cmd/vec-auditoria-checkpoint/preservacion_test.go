package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestPreservacionCLIRechazaPrestamosDeFirmaYDatos(t *testing.T) {
	for _, args := range [][]string{
		{"--operacion", "configurar-preservacion", "--kms-master", "material"},
		{"--operacion", "consultar-preservacion", "--entrada", "datos"},
		{"--operacion", "configurar-preservacion", "--version", "1"},
	} {
		var out bytes.Buffer
		if run(args, &out, &out) != 1 {
			t.Fatal("modo incompatible aceptado")
		}
	}
}

func TestPreservacionConsultaVersionInvalidaDespachaAuditoria(t *testing.T) {
	for _, version := range []uint64{0, domain.MaxVersionPreservacionAuditoria, domain.MaxVersionPreservacionAuditoria + 1} {
		llamadas := 0
		valida := versionConsultaPreservacionValida(context.Background(), version, func(_ context.Context, accion, resultado string) error {
			llamadas++
			if accion != "consultar_preservacion_auditoria_v1" || resultado != "error" {
				t.Fatal("despacho ajeno al perfil consultor")
			}
			return errors.New("registrador_no_disponible")
		})
		if version <= domain.MaxVersionPreservacionAuditoria && (!valida || llamadas != 0) ||
			version > domain.MaxVersionPreservacionAuditoria && (valida || llamadas != 1) {
			t.Fatal("consulta inválida sin registro o habilitada tras fallo de auditoría")
		}
	}
}
