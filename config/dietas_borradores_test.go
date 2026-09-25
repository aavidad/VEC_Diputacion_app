package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestConfiguracionDietasBorradoresFallaCerradaAnteAusenciasYReuso(t *testing.T) {
	validos := []string{"postgres://dietas", "postgres://personal"}
	for i := range validos {
		caso := append([]string(nil), validos...)
		caso[i] = ""
		_, err := NuevaConfiguracionDietasBorradores(caso[0], caso[1])
		if !errors.Is(err, ErrConfiguracionDietasBorradoresIncompleta) {
			t.Fatalf("ausencia %d aceptada: %v", i, err)
		}
	}
	repetido := append([]string(nil), validos...)
	repetido[1] = repetido[0]
	_, err := NuevaConfiguracionDietasBorradores(repetido[0], repetido[1])
	if !errors.Is(err, ErrConfiguracionDietasBorradoresNoSeparada) {
		t.Fatalf("reuso aceptado: %v", err)
	}
	_, err = NuevaConfiguracionDietasBorradores("postgres://vec_operador:uno@localhost/vec", "postgres://vec_operador:dos@otro/vec")
	if !errors.Is(err, ErrConfiguracionDietasBorradoresNoSeparada) {
		t.Fatalf("dos DSN con el mismo login admitidos: %v", err)
	}
}

func TestInventarioPostgreSQLIncluyeLoginsDietas(t *testing.T) {
	c, err := NuevaConfiguracionDietasBorradores("postgres://vec_dietas:uno@localhost/vec", "postgres://vec_personal:dos@localhost/vec")
	if err != nil {
		t.Fatal(err)
	}
	dsns := (Config{DietasBorradoresPostgreSQL: c}).dsnsPostgreSQLConfigurados()
	if len(dsns) != 2 || dsns[0] != "postgres://vec_dietas:uno@localhost/vec" || dsns[1] != "postgres://vec_personal:dos@localhost/vec" {
		t.Fatal("logins de Dietas o Personal ausentes del inventario")
	}
}

func TestConfiguracionDietasBorradoresEntregaSoloLosDosDSNValidadosYLosRedacta(t *testing.T) {
	c, err := NuevaConfiguracionDietasBorradores(" postgres://dietas:secreto1 ", "postgres://personal:secreto2")
	if err != nil {
		t.Fatal(err)
	}
	dietas, _, err := c.DSNSeparados()
	if err != nil || dietas != "postgres://dietas:secreto1" {
		t.Fatalf("DSN no entregado tras validar: %q, %v", dietas, err)
	}
	jsonConfig, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, representacion := range []string{fmt.Sprint(c), fmt.Sprintf("%+v", c), string(jsonConfig), c.LogValue().String()} {
		if strings.Contains(representacion, "secreto") {
			t.Fatalf("secreto expuesto: %q", representacion)
		}
	}
}

func TestLoadCargaSoloLosDosConsumidoresPropiosDeDietas(t *testing.T) {
	variables := []string{
		EnvDietasBorradoresDatabaseURL, EnvDietasPersonalRelacionesDatabaseURL,
	}
	for indice, variable := range variables {
		t.Setenv(variable, fmt.Sprintf(" postgres://rol%d:secreto@bd/vec ", indice))
	}
	c := Load().DietasBorradoresPostgreSQL
	if !c.Configurada() {
		t.Fatal("la configuración explícita de Dietas no quedó activada")
	}
	dietas, personal, err := c.DSNSeparados()
	if err != nil {
		t.Fatal(err)
	}
	obtenidos := []string{dietas, personal}
	for indice, dsn := range obtenidos {
		esperado := fmt.Sprintf("postgres://rol%d:secreto@bd/vec", indice)
		if dsn != esperado {
			t.Fatalf("DSN %d = %q; se esperaba %q", indice, dsn, esperado)
		}
	}
}

func TestAsignacionPersonalDietasRequiereTercerLoginNominal(t *testing.T) {
	base, err := NuevaConfiguracionDietasBorradores("postgres://dietas:uno@bd/vec", "postgres://relaciones:dos@bd/vec")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := base.DSNAsignacionPersonal(); !errors.Is(err, ErrConfiguracionDietasBorradoresIncompleta) {
		t.Fatalf("asignación sin login propio: %v", err)
	}
	if _, err := NuevaConfiguracionDietasBorradores("postgres://dietas:uno@bd/vec", "postgres://relaciones:dos@bd/vec", "postgres://dietas:tres@otra/vec"); !errors.Is(err, ErrConfiguracionDietasBorradoresNoSeparada) {
		t.Fatalf("login de Dietas reutilizado: %v", err)
	}
	completa, err := NuevaConfiguracionDietasBorradores("postgres://dietas:uno@bd/vec", "postgres://relaciones:dos@bd/vec", "postgres://asignacion:tres@bd/vec")
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := completa.DSNAsignacionPersonal()
	if err != nil || dsn != "postgres://asignacion:tres@bd/vec" {
		t.Fatal("la asignación no recibió el login nominal")
	}
	if len((Config{DietasBorradoresPostgreSQL: completa}).dsnsPostgreSQLConfigurados()) != 3 {
		t.Fatal("el login Personal no participa en el inventario")
	}
	conAuditoria, err := NuevaConfiguracionDietasBorradores("postgres://dietas:uno@bd/vec", "postgres://relaciones:dos@bd/vec", "postgres://asignacion:tres@bd/vec", "postgres://auditoria:cuatro@bd/vec")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conAuditoria.DSNAuditoriaPersonal(); err != nil {
		t.Fatalf("auditoría Personal nominal rechazada: %v", err)
	}
	if len((Config{DietasBorradoresPostgreSQL: conAuditoria}).dsnsPostgreSQLConfigurados()) != 4 {
		t.Fatal("el login de auditoría Personal no participa en el inventario")
	}
	if _, err := NuevaConfiguracionDietasBorradores("postgres://dietas:uno@bd/vec", "postgres://relaciones:dos@bd/vec", "postgres://asignacion:tres@bd/vec", "postgres://relaciones:otra@bd/vec"); !errors.Is(err, ErrConfiguracionDietasBorradoresNoSeparada) {
		t.Fatalf("auditoría Personal reutilizó login: %v", err)
	}
}
