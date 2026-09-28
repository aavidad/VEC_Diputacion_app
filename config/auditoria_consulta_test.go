package config

import (
	"errors"
	"strings"
	"testing"
)

func TestRutaCatalogoAuditoriaConsultaDesarrollo(t *testing.T) {
	activo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	if ruta, err := activo.RutaCatalogoAuditoriaConsultaDesarrollo(); err != nil || ruta != RutaCatalogoAuditoriaConsultaEjemplo {
		t.Fatalf("catalogo predeterminado: %q, %v", ruta, err)
	}
	t.Setenv(EnvAuditoriaConsultaCatalogoPath, " /tmp/catalogo-auditoria.json ")
	if ruta, err := activo.RutaCatalogoAuditoriaConsultaDesarrollo(); err != nil || ruta != "/tmp/catalogo-auditoria.json" {
		t.Fatalf("catalogo editable: %q, %v", ruta, err)
	}
	if _, err := (Config{ExecutionProfile: ExecutionProfileProduction}).RutaCatalogoAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrCatalogoAuditoriaConsultaFueraDesarrollo) {
		t.Fatalf("catalogo de ejemplo aceptado fuera de desarrollo: %v", err)
	}
	t.Setenv(EnvAuditoriaConsultaCatalogoPath, "  ")
	if _, err := activo.RutaCatalogoAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrCatalogoAuditoriaConsultaFueraDesarrollo) {
		t.Fatalf("ruta vacia aceptada: %v", err)
	}
}

func TestExpedientesAuditoriaConsultaDesarrolloExigeDosReferenciasExactas(t *testing.T) {
	activo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	if _, _, err := activo.ExpedientesAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) {
		t.Fatalf("se admitieron referencias ausentes: %v", err)
	}
	t.Setenv(EnvAuditoriaConsultaExpedienteCT, "expediente:ct:sintetico:001")
	t.Setenv(EnvAuditoriaConsultaExpedienteBolsa, "participacion:bolsa:sintetica:001")
	ct, bolsa, err := activo.ExpedientesAuditoriaConsultaDesarrollo()
	if err != nil || ct != "expediente:ct:sintetico:001" || bolsa != "participacion:bolsa:sintetica:001" {
		t.Fatalf("referencias explícitas rechazadas: %q %q %v", ct, bolsa, err)
	}
	for _, invalida := range []string{"*", "participacion:bolsa:*", " participacion:bolsa:sintetica:001", "expediente:ct:sintetico:001"} {
		t.Setenv(EnvAuditoriaConsultaExpedienteBolsa, invalida)
		if _, _, err := activo.ExpedientesAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) || err.Error() != ErrExpedientesAuditoriaConsultaInvalidos.Error() {
			t.Fatalf("referencia ajena o abierta admitida o expuesta: %q, %v", invalida, err)
		}
	}
	if _, _, err := (Config{ExecutionProfile: ExecutionProfileProduction}).ExpedientesAuditoriaConsultaDesarrollo(); !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) {
		t.Fatalf("referencias DEMO disponibles en producción: %v", err)
	}
}

func TestReferenciaOpacaConfiguracionValidaSintaxisTecnica(t *testing.T) {
	for _, caso := range []struct {
		ref    string
		valida bool
	}{
		{"expediente:ct:sintetico:001", true},
		{"participacion:bolsa:sintetica:001", true},
		{"A._:/#-", true},
		{"a12", true},
		{strings.Repeat("a", 160), true},
		{"", false},
		{"a1", false},
		{strings.Repeat("a", 161), false},
		{"-a1", false},
		{"a*1", false},
		{"a 1", false},
		{"a\n1", false},
		{"á12", false},
	} {
		if obtenida := referenciaOpacaConfiguracionValida(caso.ref); obtenida != caso.valida {
			t.Errorf("referencia sintética %q: valida=%t, esperada=%t", caso.ref, obtenida, caso.valida)
		}
	}
}

func TestExpedientesAuditoriaConsultaDesarrolloRechazaAmbasReferenciasInvalidasSinExponerlas(t *testing.T) {
	activo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	const ct = "expediente:ct:sintetico:001"
	const bolsa = "participacion:bolsa:sintetica:001"
	for _, caso := range []struct{ ct, bolsa string }{
		{" " + ct, bolsa},
		{ct + "*", bolsa},
		{ct, " " + bolsa},
		{ct, bolsa + "*"},
		{ct, ct},
	} {
		t.Setenv(EnvAuditoriaConsultaExpedienteCT, caso.ct)
		t.Setenv(EnvAuditoriaConsultaExpedienteBolsa, caso.bolsa)
		ctObtenida, bolsaObtenida, err := activo.ExpedientesAuditoriaConsultaDesarrollo()
		if ctObtenida != "" || bolsaObtenida != "" || !errors.Is(err, ErrExpedientesAuditoriaConsultaInvalidos) ||
			strings.Contains(err.Error(), caso.ct) || strings.Contains(err.Error(), caso.bolsa) {
			t.Fatalf("referencias no rechazadas o expuestas: %q, %q, %v", ctObtenida, bolsaObtenida, err)
		}
	}
}

func TestDSNFuenteAutorizacionAuditoriaDesarrolloExigeLoginNominalSeparado(t *testing.T) {
	dsn := func(login string) string {
		return "postgres://" + login + ":secreto-privado@localhost/vec?sslmode=require"
	}
	c := Config{
		ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
		ContratacionTemporalPostgreSQL: ConfiguracionPostgreSQLContratacionTemporal{
			dsnEjecucion: dsn("ct_ejecutor"), dsnGobierno: dsn("gobierno"),
			dsnRegistroAutorizacion: dsn("registro_v3"), dsnConfirmador: dsn("confirmador"),
			dsnLectorResultado: dsn("lector"), dsnBolsaLlamamientos: dsn("bolsa_puente"),
			dsnConsultasRRHH: dsn("ct_consultor"), dsnMotivosRRHH: dsn("motivos"),
		},
		BolsaBorradoresPostgreSQL: ConfiguracionPostgreSQLBorradores{
			dsnEjecutorConsulta:  dsn("bolsa_ejecutor"),
			dsnProyectorGobierno: dsn("bolsa_gobierno"),
			dsnVerificadorRecibo: dsn("bolsa_verificador"),
		},
	}
	if _, err := c.DSNFuenteAutorizacionAuditoriaDesarrollo(); !errors.Is(err, ErrFuenteAutorizacionAuditoriaIncompleta) {
		t.Fatalf("fuente ausente admitida: %v", err)
	}
	const fuente = "postgres://fuente_auditoria:secreto-privado@localhost/vec?sslmode=require"
	t.Setenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, " "+fuente+" ")
	if obtenida, err := c.DSNFuenteAutorizacionAuditoriaDesarrollo(); err != nil || obtenida != fuente {
		t.Fatalf("fuente nominal: %t, %v", obtenida == fuente, err)
	}
	for _, login := range []string{"ct_consultor", "bolsa_ejecutor", "registro_v3", "motivos", "gobierno"} {
		t.Setenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL,
			"postgres://"+login+":otra-clave@localhost/otra_base?sslmode=require")
		if _, err := c.DSNFuenteAutorizacionAuditoriaDesarrollo(); !errors.Is(err, ErrFuenteAutorizacionAuditoriaNoSeparada) || strings.Contains(err.Error(), "otra-clave") {
			t.Fatalf("LOGIN %s compartido o secreto expuesto: %v", login, err)
		}
	}
	t.Setenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, "postgres://mal%zz")
	if _, err := c.DSNFuenteAutorizacionAuditoriaDesarrollo(); !errors.Is(err, ErrFuenteAutorizacionAuditoriaIncompleta) || strings.Contains(err.Error(), "mal%zz") {
		t.Fatalf("DSN inválido admitido o expuesto: %v", err)
	}
	t.Setenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, fuente)
	if _, err := c.DSNMotivosAuditoriaDesarrollo(); !errors.Is(err, ErrMotivosAuditoriaIncompletos) {
		t.Fatalf("resolutor de motivos ausente admitido: %v", err)
	}
	const motivos = "postgres://motivos_auditoria:secreto-privado@localhost/vec?sslmode=require"
	t.Setenv(EnvRRHHAuditoriaMotivosDatabaseURL, motivos)
	if _, err := c.DSNFronteraAuditoriaDesarrollo(); !errors.Is(err, ErrFronteraAuditoriaIncompleta) {
		t.Fatalf("registrador de frontera ausente admitido: %v", err)
	}
	const frontera = "postgres://frontera_auditoria:secreto-privado@localhost/vec?sslmode=require"
	t.Setenv(EnvRRHHAuditoriaFronteraDatabaseURL, frontera)
	if obtenido, err := c.DSNFronteraAuditoriaDesarrollo(); err != nil || obtenido != frontera {
		t.Fatalf("registrador nominal: %t, %v", obtenido == frontera, err)
	}
	for _, login := range []string{"fuente_auditoria", "motivos_auditoria", "ct_consultor", "registro_v3"} {
		t.Setenv(EnvRRHHAuditoriaFronteraDatabaseURL,
			"postgres://"+login+":otra-clave@localhost/otra_base?sslmode=require")
		if _, err := c.DSNFronteraAuditoriaDesarrollo(); !errors.Is(err, ErrFronteraAuditoriaNoSeparada) || strings.Contains(err.Error(), "otra-clave") {
			t.Fatalf("LOGIN de frontera %s compartido o secreto expuesto: %v", login, err)
		}
	}
	t.Setenv(EnvRRHHAuditoriaFronteraDatabaseURL, frontera)
	if obtenido, err := c.DSNMotivosAuditoriaDesarrollo(); err != nil || obtenido != motivos {
		t.Fatalf("resolutor nominal: %t, %v", obtenido == motivos, err)
	}
	for _, login := range []string{"fuente_auditoria", "motivos", "ct_consultor", "registro_v3"} {
		t.Setenv(EnvRRHHAuditoriaMotivosDatabaseURL,
			"postgres://"+login+":otra-clave@localhost/otra_base?sslmode=require")
		if _, err := c.DSNMotivosAuditoriaDesarrollo(); !errors.Is(err, ErrMotivosAuditoriaNoSeparados) || strings.Contains(err.Error(), "otra-clave") {
			t.Fatalf("LOGIN de motivos %s compartido o secreto expuesto: %v", login, err)
		}
	}
	t.Setenv(EnvRRHHAuditoriaMotivosDatabaseURL, motivos)
	if _, err := (Config{}).DSNFuenteAutorizacionAuditoriaDesarrollo(); !errors.Is(err, ErrFuenteAutorizacionAuditoriaIncompleta) {
		t.Fatalf("fuente DEMO disponible fuera de desarrollo: %v", err)
	}
	if _, err := (Config{}).DSNMotivosAuditoriaDesarrollo(); !errors.Is(err, ErrMotivosAuditoriaIncompletos) {
		t.Fatalf("motivos DEMO disponibles fuera de desarrollo: %v", err)
	}
	if _, err := (Config{}).DSNFronteraAuditoriaDesarrollo(); !errors.Is(err, ErrFronteraAuditoriaIncompleta) {
		t.Fatalf("frontera DEMO disponible fuera de desarrollo: %v", err)
	}
}
