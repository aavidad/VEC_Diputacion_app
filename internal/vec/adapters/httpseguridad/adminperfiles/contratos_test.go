package adminperfiles

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestCuentaADMINExigeCuentaPrivilegiadaSeparadaYRolNominal(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	base := CuentaADMIN{
		SujetoID: "admin-persona-v1:per_ABCDEFGHIJKLMNOPQRSTUV", CuentaID: "admin-cuenta-v1:cta_abcdefghijklmnopqrstuv",
		CuentaOrdinariaID: "admin-cuenta-v1:cta_zyxwvutsrqponmlkjihgfe",
		PersonaRef:        "per_ABCDEFGHIJKLMNOPQRSTUV", CuentaRef: "cta_abcdefghijklmnopqrstuv",
		CuentaOrdinariaRef: "cta_zyxwvutsrqponmlkjihgfe", PerfilActivoRef: "prf_ABCDEFGHIJKLMNOPQRSTUV",
		RolID: "administracion_perfiles", VinculoRef: "vca_ABCDEFGHIJKLMNOPQRSTUV", VinculoVersion: 1, SeleccionRevision: 1,
		PoliticaGarantiaRef: "pga_ABCDEFGHIJKLMNOPQRSTUV", PoliticaGarantiaHuellaSHA256: strings.Repeat("a", 64),
		GarantiaObservada: domain.AuthAssuranceHigh, VigenteHasta: ahora.Add(time.Minute),
	}
	if !base.Valida(ahora) {
		t.Fatal("la cuenta nominal central válida fue rechazada")
	}
	casos := map[string]func(*CuentaADMIN){
		"rol de copias":     func(c *CuentaADMIN) { c.RolID = "operador_plataforma" },
		"cuenta compartida": func(c *CuentaADMIN) { c.CuentaOrdinariaRef = c.CuentaRef },
		"perfil ausente":    func(c *CuentaADMIN) { c.PerfilActivoRef = "" },
		"política ausente":  func(c *CuentaADMIN) { c.PoliticaGarantiaHuellaSHA256 = "" },
		"caducada":          func(c *CuentaADMIN) { c.VigenteHasta = ahora },
		"sin selección":     func(c *CuentaADMIN) { c.SeleccionRevision = 0 },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			c := base
			alterar(&c)
			if c.Valida(ahora) {
				t.Fatal("una cuenta no autorizable se aceptó")
			}
		})
	}
}

func TestErroresDeFuenteADMINNoConviertenCaidaEnDenegacion(t *testing.T) {
	for _, caso := range []struct {
		origen, esperado error
	}{
		{api.ErrAutenticacionRequerida, api.ErrAutenticacionRequerida},
		{fmt.Errorf("CAS: %w", api.ErrConflictoEstado), api.ErrConflictoEstado},
		{api.ErrAccesoDenegado, api.ErrAccesoDenegado},
		{errors.New("postgresql sintético caído"), api.ErrConfiguracionIncompleta},
	} {
		if got := errorAutoridad(caso.origen); !errors.Is(got, caso.esperado) {
			t.Fatalf("origen=%v normalizado=%v esperado=%v", caso.origen, got, caso.esperado)
		}
	}
}

func TestObservacionADMINNoSobreviveALaCRL(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	o := ObservacionADMIN{
		Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.perfiles.v1",
		CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64),
		AutenticacionVerificadaEn: ahora.Add(-time.Minute), RevocacionVerificadaEn: ahora,
		CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Hour),
	}
	if !o.Valida(ahora) {
		t.Fatal("la observación válida fue rechazada")
	}
	o.CRLVigenteHasta = ahora
	if o.Valida(ahora) {
		t.Fatal("CRL caducada aceptada")
	}
}
