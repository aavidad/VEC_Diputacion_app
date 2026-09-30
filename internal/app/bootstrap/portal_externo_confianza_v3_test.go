package bootstrap

import (
	"context"
	"errors"
	"testing"
)

func TestPreflightV3PortalExternoExigeVerifyFullInclusoEnLoopback(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "::1", "/tmp"} {
		for _, modo := range []string{"disable", "allow", "prefer", "require", "verify-ca"} {
			t.Run(host+"/"+modo, func(t *testing.T) {
				dsn := "host=" + host + " user=" + loginPreflightV3PortalExterno + " dbname=vec sslmode=" + modo
				if c, err := prepararConfiguracionPreflightV3PortalExterno(dsn); c != nil || !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
					t.Fatalf("preflight no verificado admitido: %v", err)
				}
				if pool, err := abrirPoolPreflightV3PortalExterno(context.Background(), dsn); pool != nil || !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
					t.Fatalf("la apertura no rechazo antes de conectar: %v", err)
				}
			})
		}
	}
}

func TestPreflightV3PortalExternoPreparaLoginPropioConTLSVerificado(t *testing.T) {
	dsn := "host=localhost user=" + loginPreflightV3PortalExterno + " dbname=vec sslmode=verify-full"
	c, err := prepararConfiguracionPreflightV3PortalExterno(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if c.ConnConfig.User != loginPreflightV3PortalExterno || c.ConnConfig.TLSConfig == nil ||
		c.ConnConfig.TLSConfig.InsecureSkipVerify || c.ConnConfig.TLSConfig.ServerName != "localhost" ||
		c.ConnConfig.RuntimeParams["default_transaction_read_only"] != "on" {
		t.Fatal("el preflight no conserva login, lectura y TLS verificado")
	}
	if c, err := prepararConfiguracionPreflightV3PortalExterno("host=localhost user=otro_sintetico dbname=vec sslmode=verify-full"); c != nil || !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("login ajeno admitido: %v", err)
	}
}
