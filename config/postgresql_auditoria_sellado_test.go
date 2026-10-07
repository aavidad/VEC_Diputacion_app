package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDSNAuditoriaSelladoSeparadoYRedactado(t *testing.T) {
	secreto := "host=/tmp user=vec_sellado_lab password=no-mostrar dbname=vec"
	t.Setenv(EnvAuditoriaSelladoDatabaseURL, " "+secreto+" ")
	c := Load()
	dsn, err := c.DSNAuditoriaSelladoSeparado()
	if err != nil || dsn != secreto {
		t.Fatalf("dsn=%q err=%v", dsn, err)
	}
	for _, texto := range []string{fmt.Sprint(c.AuditoriaSelladoPostgreSQL), fmt.Sprintf("%#v", c.AuditoriaSelladoPostgreSQL)} {
		if strings.Contains(texto, "no-mostrar") {
			t.Fatal("el DSN aparece en el texto")
		}
	}
	c.BolsaRelevoCesePostgreSQL = NuevaConfiguracionPostgreSQLBolsaRelevoCese("host=/tmp user=vec_sellado_lab dbname=vec")
	if _, err := c.DSNAuditoriaSelladoSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLAuditoriaSelladoNoSeparada) {
		t.Fatalf("LOGIN compartido admitido: %v", err)
	}
	t.Setenv(EnvAuditoriaSelladoDatabaseURL, "")
	if dsn, err := Load().DSNAuditoriaSelladoSeparado(); dsn != "" || err != nil {
		t.Fatalf("sin configurar: dsn=%q err=%v", dsn, err)
	}
}
