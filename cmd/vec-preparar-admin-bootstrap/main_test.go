package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) { return 0, errors.New("salida no disponible") }

var _ io.Writer = salidaFallida{}

func materialSintetico() material {
	h := func(c string) string { return strings.Repeat(c, 64) }
	ref := func(prefijo, c string) string { return prefijo + strings.Repeat(c, 24) }
	e := func(nombre, c string) evidencia { return evidencia{Referencia: nombre, Version: 1, HuellaSHA256: h(c)} }
	primera := persona{
		CuentaRef: ref("cta_", "a"), CuentaVersion: 2, PersonaRef: ref("per_", "a"), PersonaVersion: 3,
		PerfilRef: ref("prf_", "a"), VinculoRef: ref("vca_", "a"), PreimagenHuellaSHA256: h("a"),
		Procedencia: e("fuente:identidad:uno", "b"), VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		Certificado: certificado{PersonaRef: ref("per_", "a"), CuentaRef: ref("cta_", "a"), HuellaSHA256: h("c"), CAHuellaSHA256: h("d"), Acreditacion: e("atestacion:admin:uno", "e")},
	}
	segunda := primera
	segunda.CuentaRef = ref("cta_", "z")
	segunda.PersonaRef = ref("per_", "z")
	segunda.PerfilRef = ref("prf_", "z")
	segunda.VinculoRef = ref("vca_", "z")
	segunda.Certificado.CuentaRef = segunda.CuentaRef
	segunda.Certificado.PersonaRef = segunda.PersonaRef
	segunda.Certificado.HuellaSHA256 = h("f")
	segunda.Certificado.Acreditacion = e("atestacion:admin:dos", "f")
	plan := material{
		Version: 2, PreparadoEn: time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC),
		CaducaEn:                           time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC),
		ControlContinuidadRevisionEsperada: 1, BootstrapEstadoEsperado: "pendiente",
		Rol:             rol{VersionRef: "rol:administracion_perfiles:v3", HuellaSHA256: h("1"), ControlRevision: 1, ControlHuellaSHA256: h("2")},
		FuenteIdentidad: e("autoridad:identidad:declarada", "3"), FuenteCA: e("autoridad:ca-admin:declarada", "d"),
		Personas: [2]persona{primera, segunda},
		Gobierno: domain.GobiernoBootstrapAdministracion{
			AudienciaAdministrativa: "admin:ensayo:v1", PoliticaCertificadoRef: "politica:admin:ejemplo",
			PoliticaCertificadoHuellaSHA256: h("4"),
			Roles: []domain.RolGobernadoBootstrapAdministracion{{VersionRef: "rol:administracion_perfiles:v3", HuellaSHA256: h("1"),
				Clase: domain.ClaseControlPerfilAdministrador, UnidadRequerida: false,
				AmbitosFijos: []domain.AmbitoFijoBootstrapAdministracion{{Clave: "unidad", Valores: []string{"unidad:ejemplo"}}},
				VigenteDesde: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
				DuracionPropuestaSegundos: 300}},
		},
	}
	plan.Personas[0].Sistemas = []domain.AsignacionSistemasBootstrapAdministracion{{Rol: domain.RolBootstrapAdministracion{VersionRef: "rol:operador_plataforma:v1", HuellaSHA256: h("5"), ControlRevision: 1, ControlHuellaSHA256: h("6")}, PerfilRef: ref("prf_", "s"), VinculoRef: ref("vca_", "s"), VigenteHasta: plan.Personas[0].VigenteHasta}}
	plan.Personas[1].Sistemas = []domain.AsignacionSistemasBootstrapAdministracion{}
	plan.Gobierno.AudienciaSelectorADMIN = "admin:selector:ensayo:v1"
	plan.Gobierno.Motivos = []domain.MotivoGobernadoBootstrapAdministracion{}
	plan.Gobierno.Roles[0].CategoriaAdmin = "aplicacion"
	plan.Gobierno.Roles[0].AmbitosFijos = []domain.AmbitoFijoBootstrapAdministracion{{Clave: "administracion", Valores: []string{"perfiles"}}}
	rolSys := plan.Gobierno.Roles[0]
	rolSys.VersionRef = "rol:operador_plataforma:v1"
	rolSys.CategoriaAdmin = "sistemas"
	rolSys.HuellaSHA256 = h("5")
	rolSys.AmbitosFijos = []domain.AmbitoFijoBootstrapAdministracion{{Clave: "administracion", Valores: []string{"sistemas"}}}
	plan.Gobierno.Roles = append(plan.Gobierno.Roles, rolSys)
	return plan
}

func prepararFuente(t *testing.T, m material) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "privado")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	fuente, plan := filepath.Join(dir, "fuente.json"), filepath.Join(dir, "plan.json")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fuente, b, 0600); err != nil {
		t.Fatal(err)
	}
	return fuente, plan
}

func TestPlanCanonicoCotejoYDivergencia(t *testing.T) {
	m := materialSintetico()
	fuente, plan := prepararFuente(t, m)
	var salida, errores bytes.Buffer
	args := []string{"-fuente", fuente, "-plan", plan}
	if ejecutar(args, &salida, &errores) != 0 {
		t.Fatalf("rechazo: %s", errores.String())
	}
	huella := strings.TrimSpace(salida.String())
	if len(huella) != 64 || strings.Contains(salida.String(), m.Personas[0].PersonaRef) {
		t.Fatal("salida no minimizada")
	}
	i, err := os.Stat(plan)
	if err != nil || i.Mode().Perm() != 0600 {
		t.Fatal("plan sin modo 0600")
	}
	primero, err := os.ReadFile(plan)
	if err != nil {
		t.Fatal(err)
	}
	salida.Reset()
	errores.Reset()
	if ejecutar(append(args, "-cotejar"), &salida, &errores) != 0 || strings.TrimSpace(salida.String()) != huella {
		t.Fatalf("cotejo fallido: %s", errores.String())
	}
	if ejecutar(args, &salida, &errores) != 0 {
		t.Fatalf("repetición fallida: %s", errores.String())
	}
	segundo, _ := os.ReadFile(plan)
	if !bytes.Equal(primero, segundo) {
		t.Fatal("plan reescrito")
	}
	m.Personas[1].CuentaVersion++
	b, _ := json.Marshal(m)
	if err := os.WriteFile(fuente, b, 0600); err != nil {
		t.Fatal(err)
	}
	salida.Reset()
	errores.Reset()
	if ejecutar(args, &salida, &errores) == 0 || salida.Len() != 0 {
		t.Fatal("divergencia aceptada")
	}
	segundo, _ = os.ReadFile(plan)
	if !bytes.Equal(primero, segundo) {
		t.Fatal("plan divergente sobrescrito")
	}
}

func TestRechazaColisionesYFuenteNoCanonica(t *testing.T) {
	m := materialSintetico()
	m.Personas[1].PerfilRef = m.Personas[0].PerfilRef
	if validar(m) == nil {
		t.Fatal("perfil compartido")
	}
	m = materialSintetico()
	m.Personas[1].Certificado.PersonaRef = m.Personas[0].PersonaRef
	if validar(m) == nil {
		t.Fatal("certificado ajeno")
	}
	m = materialSintetico()
	m.Rol.VersionRef = "rol:administracion_perfiles:v1"
	if validar(m) == nil {
		t.Fatal("rol anterior")
	}
	m = materialSintetico()
	m.ControlContinuidadRevisionEsperada = 0
	if validar(m) == nil {
		t.Fatal("control ausente")
	}
	if decodificarEstricto([]byte(`{"version":1,"version":1}`), &material{}) == nil {
		t.Fatal("duplicado")
	}
	if decodificarEstricto([]byte(`{"Version":1}`), &material{}) == nil {
		t.Fatal("clave alternativa")
	}
}

func TestRechazaFicheroInseguro(t *testing.T) {
	fuente, plan := prepararFuente(t, materialSintetico())
	if err := os.Chmod(fuente, 0644); err != nil {
		t.Fatal(err)
	}
	var salida, errores bytes.Buffer
	if ejecutar([]string{"-fuente", fuente, "-plan", plan}, &salida, &errores) == 0 || salida.Len() != 0 {
		t.Fatal("fuente pública")
	}
}

func TestFalloAlEntregarHuellaTieneCodigoNominal(t *testing.T) {
	fuente, plan := prepararFuente(t, materialSintetico())
	var errores bytes.Buffer
	if ejecutar([]string{"-fuente", fuente, "-plan", plan}, salidaFallida{}, &errores) == 0 {
		t.Fatal("salida fallida aceptada")
	}
	if errores.String() != "plan_salida_fallida\n" {
		t.Fatalf("error no minimizado: %q", errores.String())
	}
	if _, err := os.Stat(plan); err != nil {
		t.Fatal("el fallo de salida no debe borrar el plan preparado")
	}
}

func TestGobiernoObligatorioYBooleanoExplicito(t *testing.T) {
	b, err := json.Marshal(materialSintetico())
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.Replace(b, []byte(`"unidad_requerida":false,`), nil, 1)
	if decodificarEstricto(b, &material{}) == nil {
		t.Fatal("booleano omitido se convierte en alcance sin unidad")
	}
	m := materialSintetico()
	m.Gobierno.Roles[0].AmbitosFijos[0].Valores = []string{"*"}
	if validar(m) == nil {
		t.Fatal("comodín administrativo aceptado")
	}
}

func TestAplicarExigeCotejoYAprobacionPrivada(t *testing.T) {
	fuente, plan := prepararFuente(t, materialSintetico())
	var salida, errores bytes.Buffer
	if ejecutar([]string{"-fuente", fuente, "-plan", plan, "-aplicar"}, &salida, &errores) == 0 || salida.Len() != 0 {
		t.Fatal("aplicación sin aprobación aceptada")
	}
	if _, err := os.Stat(plan); !os.IsNotExist(err) {
		t.Fatal("el uso inválido modifica el plan")
	}
}

func TestAplicarSinProveedorNoProduceRecibo(t *testing.T) {
	m := materialSintetico()
	fuente, plan := prepararFuente(t, m)
	var salida, errores bytes.Buffer
	if ejecutar([]string{"-fuente", fuente, "-plan", plan}, &salida, &errores) != 0 {
		t.Fatal(errores.String())
	}
	huella := strings.TrimSpace(salida.String())
	dir := filepath.Dir(fuente)
	conexion, aprobacion := filepath.Join(dir, "conexion.json"), filepath.Join(dir, "aprobacion.json")
	for ruta, valor := range map[string]any{
		conexion:   conexionPrivada{DSN: "host=/var/run/postgresql user=operador dbname=ensayo sslmode=disable", TimeoutSegundos: 1},
		aprobacion: aprobacionPrivada{HuellaPlanSHA256: huella},
	} {
		b, err := json.Marshal(valor)
		if err != nil || os.WriteFile(ruta, b, 0600) != nil {
			t.Fatal("configuración sintética no disponible")
		}
	}
	if _, err := aplicarPlan(m, conexion, aprobacion); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
		t.Fatalf("autoridad ausente: %v", err)
	}
	salida.Reset()
	errores.Reset()
	recibo := filepath.Join(dir, "recibo.json")
	args := []string{"-fuente", fuente, "-plan", plan, "-cotejar", "-aplicar", "-conexion", conexion, "-aprobacion", aprobacion, "-recibo", recibo}
	if ejecutar(args, &salida, &errores) == 0 || salida.Len() != 0 || errores.String() != "provision_no_confirmada\n" {
		t.Fatalf("resultado de provisión ausente: salida=%q error=%q", salida.String(), errores.String())
	}
	if _, err := os.Stat(recibo); !os.IsNotExist(err) {
		t.Fatal("provisión sin autoridad creó recibo")
	}
}

func TestFallbackRemotoNoHeredaExencionDelSocket(t *testing.T) {
	for _, caso := range []struct {
		hosts, ssl string
		valido     bool
	}{
		{"/var/run/postgresql,servidor.example", "disable", false},
		{"/var/run/postgresql,servidor.example", "verify-full", true},
		{"servidor.example,/var/run/postgresql", "verify-full", true},
		{"/var/run/postgresql", "disable", true},
	} {
		t.Run(caso.hosts+caso.ssl, func(t *testing.T) {
			cfg, err := pgxpool.ParseConfig("host=" + caso.hosts + " user=operador dbname=ensayo sslmode=" + caso.ssl)
			if err != nil {
				t.Fatal(err)
			}
			if conexionBootstrapValida(&cfg.ConnConfig.Config) != caso.valido {
				t.Fatal("validación ignora un destino alternativo")
			}
		})
	}
}
