package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Fichero de cargos sintético; su plan esperado es testdata/plan_postgresql.txt,
// impreso por PostgreSQL (jsonb::text) con las huellas calculadas en SQL por
// documento_rol_cargo_firma_v1 y canon_version_rol_admin_v1 de AUT53.
const cargosPrueba = `{"esquema":"vec.admin.cargos-firma.cargos.v1","cargos":[
{"rol_id":"ct_direccion_rrhh","version":1,"nombre":"Jefatura de Servicio de RRHH \"firma\" — área","version_anterior_sha256":"",
 "operaciones_v2":["consultar_r5","firmar_vec"],"competencias":[{"accion":"contratacion_temporal.documento.firma_vec.registrar","tipo_recurso":"documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"}],
 "regla_asignacion":"lote_ordinario","organizacion_ref":"organizacion:desarrollo:dipgra","vigente_desde":"2026-10-05T10:00:00Z","vigente_hasta":"2027-10-05T10:00:00Z","duracion_propuesta_segundos":86400},
{"rol_id":"ct_secretaria_general","version":2,"nombre":"Secretaría General","version_anterior_sha256":"abababababababababababababababababababababababababababababababab",
 "operaciones_v2":[],"competencias":[{"accion":"contratacion_temporal.documento.firma_vec.registrar","tipo_recurso":"documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"},{"accion":"contratacion_temporal.documento.resolucion.firmar","tipo_recurso":"documento_resolucion_contratacion_temporal","finalidad":"resolver_contratacion_temporal"}],
 "regla_asignacion":"lote_ordinario","organizacion_ref":"organizacion:desarrollo:dipgra","vigente_desde":"2026-10-05T10:00:00Z","vigente_hasta":"2027-10-05T10:00:00Z","duracion_propuesta_segundos":3600}]}`

const operacionPrueba = "rpa_cf_00000000000000000000000000000001"

var preparadoPrueba = time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)

func ficheroPrueba(t *testing.T) ficheroCargos {
	t.Helper()
	var f ficheroCargos
	if err := decodificarEstricto([]byte(cargosPrueba), &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func planPrueba(t *testing.T) (documentoPlan, []byte) {
	t.Helper()
	p, err := prepararPlan(ficheroPrueba(t), operacionPrueba, preparadoPrueba, 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return p, textoPlan(p)
}

// La huella de cada rol calculada en Go y el texto canónico del plan deben
// ser byte a byte los que produce PostgreSQL; si no, AUT53 deniega.
func TestPlanIgualQueElDePostgreSQL(t *testing.T) {
	esperado, err := os.ReadFile("testdata/plan_postgresql.txt")
	if err != nil {
		t.Fatal(err)
	}
	p, b := planPrueba(t)
	if !bytes.Equal(b, esperado) {
		t.Fatalf("plan distinto del de PostgreSQL:\n%s\n%s", b, esperado)
	}
	var leido documentoPlan
	if decodificarEstricto(b, &leido) != nil || comprobarPlan(b, leido) != nil || leido.Cargos[0].VersionRolSHA256 != p.Cargos[0].VersionRolSHA256 {
		t.Fatal("el plan preparado no se relee como coherente")
	}
}

func TestPrepararRechazaCargosNoAdmitidos(t *testing.T) {
	casos := map[string]func(*cargoEntrada){
		"tipo_de_fachada_v3":     func(c *cargoEntrada) { c.Competencias[0].TipoRecurso = "firma_externa_documento_contratacion_temporal" },
		"accion_administracion":  func(c *cargoEntrada) { c.Competencias[0].Accion = "administracion.perfiles.otorgar" },
		"operaciones_al_reves":   func(c *cargoEntrada) { c.OperacionesV2 = []string{"firmar_vec", "consultar_r5"} },
		"operacion_desconocida":  func(c *cargoEntrada) { c.OperacionesV2 = []string{"firmar_externa"} },
		"nombre_de_intervencion": func(c *cargoEntrada) { c.Nombre = "Intervención General" },
		"version_sin_anterior":   func(c *cargoEntrada) { c.Version = 2 },
		"sin_competencias":       func(c *cargoEntrada) { c.Competencias = nil },
		"duracion_corta":         func(c *cargoEntrada) { c.DuracionPropuestaSegundos = 59 },
		"vigencia_al_reves":      func(c *cargoEntrada) { c.VigenteHasta = c.VigenteDesde },
		"rol_con_mayusculas":     func(c *cargoEntrada) { c.RolID = "CT_Direccion" },
		"rol_de_intervencion":    func(c *cargoEntrada) { c.RolID = "intervencion_firmas" },
		"rol_externo":            func(c *cargoEntrada) { c.RolID = "ct_firma_externa" },
		"rol_de_candidato":       func(c *cargoEntrada) { c.RolID = "candidato_firma" },
		"rol_de_administracion":  func(c *cargoEntrada) { c.RolID = "administracion_perfiles" },
		"nombre_sin_recortar":    func(c *cargoEntrada) { c.Nombre = " Dirección de RRHH" },
		"nombre_con_control":     func(c *cargoEntrada) { c.Nombre = "Dirección\tde RRHH" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f := ficheroPrueba(t)
			cambiar(&f.Cargos[0])
			if _, err := prepararPlan(f, operacionPrueba, preparadoPrueba, time.Hour); err == nil {
				t.Fatal("cargo no admitido aceptado")
			}
		})
	}
	f := ficheroPrueba(t)
	f.Cargos[1].RolID = f.Cargos[0].RolID
	if _, err := prepararPlan(f, operacionPrueba, preparadoPrueba, time.Hour); err == nil {
		t.Fatal("rol repetido aceptado")
	}
	if _, err := prepararPlan(ficheroPrueba(t), operacionPrueba, preparadoPrueba, 25*time.Hour); err == nil {
		t.Fatal("caducidad de más de un día aceptada")
	}
	if _, err := prepararPlan(ficheroPrueba(t), operacionPrueba, preparadoPrueba.Add(time.Millisecond), time.Hour); err == nil {
		t.Fatal("instante con fracción aceptado")
	}
}

// Un plan editado a mano (huella o concesión cambiada) se para antes de
// conectar, aunque su aprobación coincida con los bytes.
func TestAplicarRechazaPlanEditadoSinConectar(t *testing.T) {
	_, b := planPrueba(t)
	editado := bytes.Replace(b, []byte(`"resolver_contratacion_temporal"`), []byte(`"ampliar_contratacion_temporal"`), 1)
	huella := []byte(`"33f40d4af8293301b7ac2f93a721919d851d94d5e47647db41611ff7df18932a"`)
	editadoHuella := bytes.Replace(b, huella, []byte(`"`+strings.Repeat("a", 64)+`"`), 1)
	if bytes.Equal(editadoHuella, b) {
		t.Fatal("la huella de prueba no está en el plan")
	}
	for _, plan := range [][]byte{editado, editadoHuella, append(append([]byte{}, b...), '\n')} {
		dir := dirPrivado(t)
		h := sha256.Sum256(plan)
		escribir(t, dir, "plan.json", plan)
		escribir(t, dir, "conexion.json", []byte(`{"dsn":"postgres://x@localhost/x","permitir_socket_desarrollo":false}`))
		escribir(t, dir, "aprobacion.json", []byte(`{"huella_plan_sha256":"`+hex.EncodeToString(h[:])+`"}`))
		abiertas := 0
		abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
			abiertas++
			return nil, errors.New("no")
		}
		var out, errout bytes.Buffer
		codigo := ejecutar([]string{"aplicar", "--plan", filepath.Join(dir, "plan.json"), "--conexion", filepath.Join(dir, "conexion.json"),
			"--aprobacion", filepath.Join(dir, "aprobacion.json"), "--acuse", filepath.Join(dir, "acuse.json"), "--timeout", "5s",
			"--textos", textos(t, "es"), "--idioma", "es"}, &out, &errout, abrir, time.Now)
		if codigo != 1 || abiertas != 0 || !strings.Contains(errout.String(), `"entrada_invalida"`) {
			t.Fatalf("plan editado admitido: %d %s", codigo, errout.String())
		}
	}
}

type txPrueba struct {
	resultado   []byte
	commitError bool
	confirmada  bool
	cerrada     bool
	plan, sha   string
}

func (t *txPrueba) PrepararUTC(context.Context) error { return nil }
func (t *txPrueba) Publicar(_ context.Context, plan, sha string) ([]byte, error) {
	t.plan, t.sha = plan, sha
	return append([]byte(nil), t.resultado...), nil
}
func (t *txPrueba) Confirmar(context.Context) error {
	if t.commitError {
		return errors.New("secreto_no_exponer")
	}
	t.confirmada = true
	return nil
}
func (t *txPrueba) Cerrar(context.Context) { t.cerrada = true }

func envolturaPrueba(t *testing.T, p documentoPlan, sha, estado string) []byte {
	t.Helper()
	h := strings.Repeat("a", 64)
	e := envoltura{Estado: estado, AuditoriaIntento: auditoriaIntento{AuditoriaRef: "aud_v3_pai_" + strings.Repeat("b", 32), Secuencia: 9, HuellaSHA256: h,
		CorrelacionRef: "correlacion_" + strings.Repeat("c", 32), RegistradaEn: "2026-10-05T10:00:01.123456+00:00"}}
	if estado == "permitido" {
		r := &reciboCargos{Esquema: "vec.admin.cargos-firma.recibo.v1", OperacionRef: p.OperacionRef, PlanSHA256: sha, AprobacionRef: "aprobacion:prueba",
			AprobacionSHA256: h, PerfilesSHA256: h, AuditoriaRef: "aud_v3_pa_" + strings.Repeat("d", 32), AuditoriaSecuencia: 8, AuditoriaHuella: h,
			ConfirmadoEn: "2026-10-05T10:00:01.123456+00:00"}
		for _, c := range p.Cargos {
			r.Cargos = append(r.Cargos, cargoRecibo{RolID: c.RolID, VersionRolRef: "rol:" + c.RolID + ":v" + map[uint64]string{1: "1", 2: "2"}[c.Version],
				VersionRolSHA256: c.VersionRolSHA256, ControlRevision: 1, ControlSHA256: h, ReglaAsignacion: c.ReglaAsignacion})
		}
		e.Recibo = r
	} else {
		c := map[string]string{"denegado": "cargos_firma_rechazado", "error": "cargos_firma_no_disponible"}[estado]
		e.Codigo = &c
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCommitDeCadaEstadoYErrorIncierto(t *testing.T) {
	p, b := planPrueba(t)
	h := sha256.Sum256(b)
	sha := hex.EncodeToString(h[:])
	for _, estado := range []string{"permitido", "denegado", "error"} {
		tx := &txPrueba{resultado: envolturaPrueba(t, p, sha, estado)}
		abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
		r, e, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, b, sha, p, abrir)
		if err != nil || len(r) == 0 || e.Estado != estado || !tx.confirmada || !tx.cerrada || tx.plan != string(b) || tx.sha != sha {
			t.Fatalf("%s no confirmado", estado)
		}
	}
	tx := &txPrueba{resultado: envolturaPrueba(t, p, sha, "permitido"), commitError: true}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	if r, _, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, b, sha, p, abrir); err != errCommit || len(r) != 0 || !tx.cerrada {
		t.Fatal("COMMIT dudoso presentado como confirmado")
	}
}

func TestEnvolturaNoAceptaCargoAjenoNiHuellaDistinta(t *testing.T) {
	p, b := planPrueba(t)
	h := sha256.Sum256(b)
	sha := hex.EncodeToString(h[:])
	if _, err := validarEnvoltura(envolturaPrueba(t, p, sha, "permitido"), p, sha); err != nil {
		t.Fatal("envoltura correcta rechazada")
	}
	for nombre, cambiar := range map[string]func(*envoltura){
		"cargo_ajeno":        func(e *envoltura) { e.Recibo.Cargos[0].RolID = "otro_cargo" },
		"huella_distinta":    func(e *envoltura) { e.Recibo.Cargos[1].VersionRolSHA256 = strings.Repeat("f", 64) },
		"cargo_de_menos":     func(e *envoltura) { e.Recibo.Cargos = e.Recibo.Cargos[:1] },
		"otro_plan":          func(e *envoltura) { e.Recibo.PlanSHA256 = strings.Repeat("e", 64) },
		"regla_distinta":     func(e *envoltura) { e.Recibo.Cargos[0].ReglaAsignacion = "doble_control" },
		"auditoria_de_aut49": func(e *envoltura) { e.AuditoriaIntento.AuditoriaRef = "aud_v3_mfi_" + strings.Repeat("b", 32) },
	} {
		var e envoltura
		if json.Unmarshal(envolturaPrueba(t, p, sha, "permitido"), &e) != nil {
			t.Fatal("fixture")
		}
		cambiar(&e)
		x, _ := json.Marshal(e)
		if _, err := validarEnvoltura(x, p, sha); err == nil {
			t.Fatalf("%s admitido", nombre)
		}
	}
	var d envoltura
	if json.Unmarshal(envolturaPrueba(t, p, sha, "denegado"), &d) != nil {
		t.Fatal("fixture")
	}
	d.Replay = true
	x, _ := json.Marshal(d)
	if _, err := validarEnvoltura(x, p, sha); err == nil {
		t.Fatal("denegación con replay admitida")
	}
}

// Recorrido de la CLI: preparar escribe un plan privado nuevo y su huella;
// aplicar con esa aprobación guarda el acuse tras COMMIT.
func TestPrepararYAplicarConTransaccionDePrueba(t *testing.T) {
	dir := dirPrivado(t)
	escribir(t, dir, "cargos.json", []byte(cargosPrueba))
	var out, errout bytes.Buffer
	ahora := func() time.Time { return preparadoPrueba.Add(1500 * time.Millisecond) }
	if c := ejecutar([]string{"preparar", "--cargos", filepath.Join(dir, "cargos.json"), "--plan", filepath.Join(dir, "plan.json"),
		"--caduca", "2h", "--textos", textos(t, "es"), "--idioma", "es"}, &out, &errout, nil, ahora); c != 0 {
		t.Fatalf("preparar: %d %s", c, errout.String())
	}
	var d diagnostico
	if json.Unmarshal(out.Bytes(), &d) != nil || d.Codigo != "plan_preparado" || d.Mensaje == "" || !hashValido(d.PlanSHA256) {
		t.Fatalf("diagnóstico de preparar: %s", out.String())
	}
	plan, err := leerPrivado(filepath.Join(dir, "plan.json"))
	if err != nil {
		t.Fatal("plan no privado")
	}
	if h := sha256.Sum256(plan); hex.EncodeToString(h[:]) != d.PlanSHA256 || bytes.HasSuffix(plan, []byte("\n")) {
		t.Fatal("huella del plan distinta de la anunciada")
	}
	var p documentoPlan
	if decodificarEstricto(plan, &p) != nil || !p.PreparadoEn.Equal(preparadoPrueba.Add(time.Second)) || p.OperacionRef == operacionPrueba {
		t.Fatal("plan preparado inesperado")
	}
	escribir(t, dir, "conexion.json", []byte(`{"dsn":"postgres://x@localhost/x","permitir_socket_desarrollo":false}`))
	escribir(t, dir, "aprobacion.json", []byte(`{"huella_plan_sha256":"`+d.PlanSHA256+`"}`))
	tx := &txPrueba{resultado: envolturaPrueba(t, p, d.PlanSHA256, "permitido")}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	out.Reset()
	errout.Reset()
	args := []string{"aplicar", "--plan", filepath.Join(dir, "plan.json"), "--conexion", filepath.Join(dir, "conexion.json"),
		"--aprobacion", filepath.Join(dir, "aprobacion.json"), "--acuse", filepath.Join(dir, "acuse.json"), "--timeout", "5s",
		"--textos", textos(t, "en"), "--idioma", "en"}
	if c := ejecutar(args, &out, &errout, abrir, time.Now); c != 0 || !tx.confirmada || !strings.Contains(out.String(), `"cargos_confirmados"`) {
		t.Fatalf("aplicar: %d %s %s", c, out.String(), errout.String())
	}
	if acuse, err := leerPrivado(filepath.Join(dir, "acuse.json")); err != nil || !bytes.Contains(acuse, []byte(p.OperacionRef)) {
		t.Fatal("acuse no guardado")
	}
	// El acuse no se sobrescribe: un segundo intento con el mismo acuse se para.
	if c := ejecutar(args, &out, &errout, abrir, time.Now); c != 1 || !strings.Contains(errout.String(), `"acuse_inseguro"`) {
		t.Fatal("acuse existente sobrescrito")
	}
}

// Los dos catálogos tienen las mismas claves y cubren todos los códigos.
func TestCatalogosCompletos(t *testing.T) {
	codigos := []string{"catalogo_no_disponible", "uso_invalido", "entrada_insegura", "entrada_invalida", "cargos_invalidos", "plan_no_guardado",
		"plan_preparado", "aprobacion_divergente", "acuse_inseguro", "commit_no_confirmado", "operacion_no_confirmada", "acuse_no_guardado",
		"cargos_confirmados", "cargos_rechazados", "cargos_no_disponibles"}
	for _, idioma := range []string{"es", "en"} {
		b, err := os.ReadFile(textos(t, idioma))
		if err != nil {
			t.Fatal(err)
		}
		var d struct {
			Mensajes map[string]string `json:"mensajes"`
		}
		if json.Unmarshal(b, &d) != nil || len(d.Mensajes) != len(codigos) {
			t.Fatalf("%s: catálogo incompleto", idioma)
		}
		for _, c := range codigos {
			if strings.TrimSpace(d.Mensajes[c]) == "" {
				t.Fatalf("%s: falta %s", idioma, c)
			}
		}
	}
}

func textos(t *testing.T, idioma string) string {
	t.Helper()
	_, fichero, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(fichero), "..", "..", "web", "static", "textos", idioma, "admin-cargos-firma.json")
}

func dirPrivado(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "privado")
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("directorio")
	}
	return dir
}

func escribir(t *testing.T, dir, nombre string, b []byte) {
	t.Helper()
	if os.WriteFile(filepath.Join(dir, nombre), b, 0600) != nil {
		t.Fatal("fixture")
	}
}
