package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestConfiguracionPostgreSQLContratacionTemporalSeparaYRedacta(t *testing.T) {
	ejecutor := "postgres://ejecutor:secreto-ejecucion@127.0.0.1/vec"
	gobierno := "postgres://gobierno:secreto-gobierno@127.0.0.1/vec"
	registro := "postgres://registro:secreto-registro@127.0.0.1/vec"
	confirmador := "postgres://confirmador:secreto-confirmador@127.0.0.1/vec"
	lector := "postgres://lector:secreto-lector@127.0.0.1/vec"
	configuracion, err := NuevaConfiguracionPostgreSQLContratacionTemporal(
		ejecutor,
		gobierno,
		registro,
		confirmador,
		lector,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenidoEjecutor, obtenidoGobierno, err := configuracion.DSNSeparados()
	if err != nil || obtenidoEjecutor != ejecutor || obtenidoGobierno != gobierno {
		t.Fatalf("DSN separados = (%q, %q, %v)", obtenidoEjecutor, obtenidoGobierno, err)
	}
	obtenidoRegistro, err := configuracion.DSNRegistroAutorizacionSeparado()
	if err != nil || obtenidoRegistro != registro {
		t.Fatalf("DSN de registro separado = (%q, %v)", obtenidoRegistro, err)
	}
	obtenidoConfirmador, obtenidoLector, err := configuracion.DSNCoberturaSeparados()
	if err != nil || obtenidoConfirmador != confirmador || obtenidoLector != lector {
		t.Fatalf(
			"DSN de cobertura separados = (%q, %q, %v)",
			obtenidoConfirmador,
			obtenidoLector,
			err,
		)
	}
	jsonConfiguracion, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	for _, salida := range []string{
		fmt.Sprint(configuracion),
		fmt.Sprintf("%+v", configuracion),
		fmt.Sprintf("%#v", configuracion),
		string(jsonConfiguracion),
	} {
		if strings.Contains(salida, "secreto-") || !strings.Contains(salida, "redactada") {
			t.Fatalf("representacion no redactada: %q", salida)
		}
	}
}

func TestConfiguracionPostgreSQLContratacionTemporalFallaCerrado(t *testing.T) {
	completas := [5]string{"ejecucion", "gobierno", "registro", "confirmador", "lector"}
	for indice := range completas {
		dsn := completas
		dsn[indice] = "  "
		if _, err := NuevaConfiguracionPostgreSQLContratacionTemporal(
			dsn[0],
			dsn[1],
			dsn[2],
			dsn[3],
			dsn[4],
		); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
			t.Fatalf("configuracion incompleta en posicion %d: %v", indice, err)
		}
	}
	for izquierda := 0; izquierda < len(completas); izquierda++ {
		for derecha := izquierda + 1; derecha < len(completas); derecha++ {
			dsn := completas
			dsn[derecha] = " " + dsn[izquierda] + " "
			if _, err := NuevaConfiguracionPostgreSQLContratacionTemporal(
				dsn[0],
				dsn[1],
				dsn[2],
				dsn[3],
				dsn[4],
			); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada) {
				t.Fatalf("configuracion no separada entre %d y %d: %v", izquierda, derecha, err)
			}
		}
	}
}

func TestLoadCargaCincoConexionesPostgreSQLContratacionTemporal(t *testing.T) {
	dsn := [5]string{
		"postgres://ejecucion@127.0.0.1/vec",
		"postgres://gobierno@127.0.0.1/vec",
		"postgres://registro@127.0.0.1/vec",
		"postgres://confirmador@127.0.0.1/vec",
		"postgres://lector@127.0.0.1/vec",
	}
	t.Setenv(EnvContratacionTemporalDatabaseURL, " "+dsn[0]+" ")
	t.Setenv(EnvContratacionTemporalGobiernoDatabaseURL, " "+dsn[1]+" ")
	t.Setenv(EnvContratacionTemporalRegistroAutorizacionDatabaseURL, " "+dsn[2]+" ")
	t.Setenv(EnvContratacionTemporalConfirmadorDatabaseURL, " "+dsn[3]+" ")
	t.Setenv(EnvContratacionTemporalLectorResultadoDatabaseURL, " "+dsn[4]+" ")

	configuracion := Load().ContratacionTemporalPostgreSQL
	ejecutor, gobierno, err := configuracion.DSNSeparados()
	if err != nil || ejecutor != dsn[0] || gobierno != dsn[1] {
		t.Fatalf("conexiones base = (%q, %q, %v)", ejecutor, gobierno, err)
	}
	registro, err := configuracion.DSNRegistroAutorizacionSeparado()
	if err != nil || registro != dsn[2] {
		t.Fatalf("conexion de registro = (%q, %v)", registro, err)
	}
	confirmador, lector, err := configuracion.DSNCoberturaSeparados()
	if err != nil || confirmador != dsn[3] || lector != dsn[4] {
		t.Fatalf("conexiones de cobertura = (%q, %q, %v)", confirmador, lector, err)
	}
}

func TestConsultasRRHHExigenConexionesCompletasSeparadasYRedactadas(t *testing.T) {
	c, err := NuevaConfiguracionPostgreSQLContratacionTemporal("ejecutor", "gobierno", "registro", "confirmador", "lector")
	if err != nil || c.ConsultasRRHHConfiguradas() {
		t.Fatal("la configuración anterior cambió")
	}
	c.dsnBolsaLlamamientos = "bolsa"
	for _, par := range [][2]string{{"", ""}, {"consultas", ""}, {"", "motivos"}} {
		c.dsnConsultasRRHH, c.dsnMotivosRRHH = par[0], par[1]
		if _, _, err := c.DSNConsultasRRHHSeparados(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
			t.Fatal("aceptó conexiones incompletas")
		}
		if c.ConsultasRRHHConfiguradas() != (par[0] != "" || par[1] != "") {
			t.Fatal("configuración parcial ignorada")
		}
	}
	for _, previa := range []string{"ejecutor", "gobierno", "registro", "confirmador", "lector", "bolsa", "misma"} {
		for _, par := range [][2]string{{" " + previa + " ", "misma"}, {"misma", " " + previa + " "}} {
			c.dsnConsultasRRHH, c.dsnMotivosRRHH = par[0], par[1]
			if _, _, err := c.DSNConsultasRRHHSeparados(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada) {
				t.Fatal("aceptó conexiones compartidas")
			}
		}
	}
	const consultas = "postgres://consultas:secreto-consultas@localhost/vec"
	const motivos = "postgres://motivos:secreto-motivos@localhost/vec"
	t.Setenv(EnvContratacionTemporalConsultasRRHHDatabaseURL, " "+consultas+" ")
	t.Setenv(EnvContratacionTemporalMotivosRRHHDatabaseURL, " "+motivos+" ")
	cargada := Load().ContratacionTemporalPostgreSQL
	c.dsnConsultasRRHH, c.dsnMotivosRRHH = cargada.dsnConsultasRRHH, cargada.dsnMotivosRRHH
	if a, b, err := c.DSNConsultasRRHHSeparados(); err != nil || a != consultas || b != motivos || !c.ConsultasRRHHConfiguradas() {
		t.Fatal("conexiones no cargadas")
	}
	serializado, err := json.Marshal(c)
	if err != nil || strings.Contains(fmt.Sprintf("%+v %#v %s", c, c, serializado), "secreto-") {
		t.Fatal("representación de conexiones no redactada")
	}
}

func TestIdentidadConsultasRRHHExigeDosConexionesNominalesSeparadas(t *testing.T) {
	c, err := NuevaConfiguracionPostgreSQLContratacionTemporal("ejecutor", "gobierno", "registro", "confirmador", "lector")
	if err != nil {
		t.Fatal(err)
	}
	c.dsnConsultasRRHH, c.dsnMotivosRRHH, c.dsnBolsaLlamamientos = "consultas", "motivos", "bolsa"
	for _, par := range [][2]string{{"", ""}, {"identidad", ""}, {"", "revalidacion"}} {
		c.dsnRegistroIdentidad, c.dsnRevalidacionIdentidad = par[0], par[1]
		if _, _, err := c.DSNIdentidadConsultasSeparados(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
			t.Fatal("identidad incompleta aceptada")
		}
	}
	for _, previa := range []string{"ejecutor", "gobierno", "registro", "confirmador", "lector", "bolsa", "consultas", "motivos", "misma"} {
		for _, par := range [][2]string{{previa, "misma"}, {"misma", previa}} {
			c.dsnRegistroIdentidad, c.dsnRevalidacionIdentidad = par[0], par[1]
			if _, _, err := c.DSNIdentidadConsultasSeparados(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada) {
				t.Fatal("identidad compartida aceptada")
			}
		}
	}
	t.Setenv(EnvContratacionTemporalRegistroIdentidadDatabaseURL, " postgres://identidad:secreto-identidad@localhost/vec ")
	t.Setenv(EnvContratacionTemporalRevalidacionIdentidadDatabaseURL, " postgres://revalidacion:secreto-revalidacion@localhost/vec ")
	cargada := Load().ContratacionTemporalPostgreSQL
	c.dsnRegistroIdentidad, c.dsnRevalidacionIdentidad = cargada.dsnRegistroIdentidad, cargada.dsnRevalidacionIdentidad
	if a, b, err := c.DSNIdentidadConsultasSeparados(); err != nil || a != strings.TrimSpace(c.dsnRegistroIdentidad) || b != strings.TrimSpace(c.dsnRevalidacionIdentidad) {
		t.Fatal("identidad no cargada")
	}
	serializado, _ := json.Marshal(c)
	if strings.Contains(fmt.Sprintf("%+v %#v %s", c, c, serializado), "secreto-") {
		t.Fatal("credencial expuesta")
	}
	if !cargada.ConsultasRRHHConfiguradas() {
		t.Fatal("configuración parcial ignorada")
	}
	if _, err := c.DSNContextoActorConsultasSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
		t.Fatal("contexto ausente aceptado")
	}
	for _, previa := range []string{"ejecutor", "gobierno", "registro", "confirmador", "lector", "bolsa", "consultas", "motivos", c.dsnRegistroIdentidad, c.dsnRevalidacionIdentidad} {
		c.dsnContextoActor = previa
		if _, err := c.DSNContextoActorConsultasSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada) {
			t.Fatal("contexto compartido aceptado")
		}
	}
	t.Setenv(EnvContratacionTemporalContextoActorDatabaseURL, " postgres://contexto:secreto-contexto@localhost/vec ")
	c.dsnContextoActor = Load().ContratacionTemporalPostgreSQL.dsnContextoActor
	if dsn, err := c.DSNContextoActorConsultasSeparado(); err != nil || dsn != strings.TrimSpace(c.dsnContextoActor) {
		t.Fatal("contexto no cargado")
	}
}

func TestBolsaLlamamientosSeConfiguraSeparadaSinExponerConexion(t *testing.T) {
	c, err := NuevaConfiguracionPostgreSQLContratacionTemporal("ejecutor", "gobierno", "registro", "confirmador", "lector")
	if err != nil {
		t.Fatal(err)
	}
	if c.BolsaLlamamientosConfigurada() {
		t.Fatal("Bolsa no debe estar habilitada por defecto")
	}
	if _, err := c.DSNBolsaLlamamientosSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
		t.Fatal(err)
	}
	for _, repetida := range []string{"ejecutor", "gobierno", "registro", "confirmador", "lector"} {
		c.dsnBolsaLlamamientos = " " + repetida + " "
		if _, err := c.DSNBolsaLlamamientosSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada) {
			t.Fatal("conexión compartida aceptada")
		}
	}
	const dsn = "postgres://bolsa:secreto-bolsa@localhost/vec"
	t.Setenv(EnvBolsaLlamamientosDatabaseURL, " "+dsn+" ")
	c.dsnBolsaLlamamientos = Load().ContratacionTemporalPostgreSQL.dsnBolsaLlamamientos
	if obtenido, err := c.DSNBolsaLlamamientosSeparado(); err != nil || obtenido != dsn {
		t.Fatal("conexión Bolsa no cargada")
	}
	if !c.BolsaLlamamientosConfigurada() {
		t.Fatal("Bolsa no habilitada")
	}
	serializado, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%+v %#v %s", c, c, serializado), "secreto-bolsa") {
		t.Fatal("conexión expuesta")
	}
}
