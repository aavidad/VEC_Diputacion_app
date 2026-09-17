package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfiguracionPostgreSQLImportacionConvocaRedactaDSN(t *testing.T) {
	secreto := "postgres://ejecutor:clave@db/vec"
	c, err := NuevaConfiguracionPostgreSQLImportacionConvoca(" " + secreto + " ")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{fmt.Sprint(c), fmt.Sprintf("%#v", c), c.LogValue().String()} {
		if strings.Contains(v, "clave") || strings.Contains(v, "@db") {
			t.Fatalf("DSN expuesto: %q", v)
		}
	}
	if dsn, err := c.DSN(); err != nil || dsn != secreto {
		t.Fatalf("DSN=%q, %v", dsn, err)
	}
}
func TestConfiguracionPostgreSQLImportacionConvocaFallaSinDSN(t *testing.T) {
	if _, err := NuevaConfiguracionPostgreSQLImportacionConvoca(" "); err == nil {
		t.Fatal("acepto DSN vacio")
	}
}
