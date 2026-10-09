package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	gobiernoperfiles "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/auditoria"
	"vec-diputacion-granada/internal/vec/domain"
)

type escenario struct {
	dir, salida, config, textos string
	dsnLectura, dsnOperador     string
	llamadas                    int
	ops                         operaciones
}

func relojFijo() time.Time { return time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC) }

func nuevoEscenario(t *testing.T) *escenario {
	t.Helper()
	base := t.TempDir()
	e := &escenario{dir: filepath.Join(base, "privado"), salida: filepath.Join(base, "salida")}
	socket := filepath.Join(base, "socket")
	for _, d := range []string{e.dir, e.salida, socket} {
		if os.Mkdir(d, 0700) != nil {
			t.Fatal("directorio")
		}
	}
	e.dsnLectura = "host=" + socket + " port=5432 user=lector_sintetico dbname=vec_sintetica sslmode=disable"
	e.dsnOperador = "host=" + socket + " port=5432 user=operador_sintetico password=secreto-sintetico dbname=vec_sintetica sslmode=disable"
	textos, err := filepath.Abs("../../web/static/textos/es/admin-gobierno-usuarios.json")
	if err != nil {
		t.Fatal(err)
	}
	e.textos = textos
	e.config = e.escribirConfig(t, configuracionPrivada{DirectorioMaterial: "/srv/privado/material", RutaConfiguracionHMAC: "/srv/privado/hmac.json", ArchivoSemillaRaiz: "/srv/privado/semilla", DSNLectura: e.dsnLectura, DSNOperador: e.dsnOperador, Salida: e.salida, HorasValidezClaves: 2})
	e.ops = operaciones{
		preparar: func(context.Context, string, time.Duration, bootstrap.MaterialOrigenGobiernoUsuariosAdmin, *os.Root) (bootstrap.PreparacionGobiernoUsuariosAdmin, error) {
			e.llamadas++
			return bootstrap.PreparacionGobiernoUsuariosAdmin{PlanSHA256: strings.Repeat("a", 64), MaterialSHA256: strings.Repeat("b", 64), PreimagenSHA256: strings.Repeat("c", 64), CaducaEn: relojFijo().Add(2 * time.Hour)}, nil
		},
		aplicar: func(context.Context, string, time.Duration, *os.Root, string) (bootstrap.ConfirmacionGobiernoUsuariosAdmin, error) {
			e.llamadas++
			return bootstrap.ConfirmacionGobiernoUsuariosAdmin{Estado: "permitido", Codigo: "gobierno_usuarios_registrado"}, nil
		},
		verificar: func(context.Context, string, time.Duration, *os.Root, string) (auditoria.InformeVerificacion, error) {
			e.llamadas++
			return auditoria.InformeVerificacion{Estado: "verificada"}, nil
		},
		versionBolsa: func(*os.Root, bootstrap.DestinoOverlayVersionBolsaAdmin, string) error {
			e.llamadas++
			return nil
		},
	}
	return e
}

func (e *escenario) escribirConfig(t *testing.T, c any) string {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(e.dir, "config.json")
	_ = os.Remove(ruta)
	if os.WriteFile(ruta, b, 0600) != nil {
		t.Fatal("config")
	}
	return ruta
}

// socketAbierto crea un directorio de socket escribible por cualquiera.
func (e *escenario) socketAbierto(t *testing.T) string {
	t.Helper()
	d := filepath.Join(filepath.Dir(e.dir), "socket-abierto")
	if os.Mkdir(d, 0700) != nil || os.Chmod(d, 0777) != nil {
		t.Fatal("socket abierto")
	}
	return d
}

func (e *escenario) args(fase string, extra ...string) []string {
	return append([]string{"-fase", fase, "-config", e.config, "-textos", e.textos, "-timeout", "5s"}, extra...)
}

func (e *escenario) ejecutar(args []string) (int, diagnostico, string, string) {
	var salida, errores bytes.Buffer
	code := ejecutar(args, &salida, &errores, e.ops, relojFijo)
	var d diagnostico
	texto := salida.Bytes()
	if len(texto) == 0 {
		texto = errores.Bytes()
	}
	_ = json.Unmarshal(texto, &d)
	return code, d, salida.String(), errores.String()
}

func TestUsoInvalidoNoConecta(t *testing.T) {
	e := nuevoEscenario(t)
	casos := map[string][]string{
		"sin_fase":          {"-config", e.config, "-textos", e.textos, "-timeout", "5s"},
		"fase_desconocida":  e.args("publicar"),
		"timeout_cero":      {"-fase", "preparar", "-config", e.config, "-textos", e.textos, "-timeout", "0s"},
		"timeout_excesivo":  {"-fase", "preparar", "-config", e.config, "-textos", e.textos, "-timeout", "1h"},
		"aplicar_sin_acuse": e.args("aplicar"),
		"acuse_en_preparar": e.args("preparar", "-acuse", "acuse.json"),
		"argumento_libre":   e.args("verificar", "sobrante"),
		"bandera_ajena":     e.args("verificar", "-dsn", "x"),
	}
	for nombre, args := range casos {
		t.Run(nombre, func(t *testing.T) {
			code, d, salida, _ := e.ejecutar(args)
			if code != 1 || d.Codigo != "uso_invalido" || salida != "" || e.llamadas != 0 || d.Mensaje == "" || d.Limite == "" {
				t.Fatal("uso aceptado", code, d.Codigo)
			}
		})
	}
}

func TestConfiguracionInsegura(t *testing.T) {
	t.Run("permisos_amplios", func(t *testing.T) {
		e := nuevoEscenario(t)
		if os.Chmod(e.config, 0640) != nil {
			t.Fatal("chmod")
		}
		if code, d, _, _ := e.ejecutar(e.args("preparar")); code != 1 || d.Codigo != "configuracion_insegura" || e.llamadas != 0 {
			t.Fatal("permisos aceptados", d.Codigo)
		}
	})
	t.Run("enlace", func(t *testing.T) {
		e := nuevoEscenario(t)
		alias := filepath.Join(e.dir, "alias.json")
		if os.Symlink(e.config, alias) != nil {
			t.Fatal("enlace")
		}
		e.config = alias
		if code, d, _, _ := e.ejecutar(e.args("preparar")); code != 1 || d.Codigo != "configuracion_insegura" || e.llamadas != 0 {
			t.Fatal("enlace seguido", d.Codigo)
		}
	})
	t.Run("carpeta_publica", func(t *testing.T) {
		e := nuevoEscenario(t)
		if os.Chmod(e.dir, 0755) != nil {
			t.Fatal("chmod")
		}
		if code, d, _, _ := e.ejecutar(e.args("preparar")); code != 1 || d.Codigo != "configuracion_insegura" {
			t.Fatal("carpeta pública aceptada", d.Codigo)
		}
	})
}

func TestConfiguracionInvalida(t *testing.T) {
	base := func(e *escenario) map[string]any {
		return map[string]any{"directorio_material": "/srv/privado/material", "ruta_configuracion_hmac": "/srv/privado/hmac.json", "archivo_semilla_raiz": "/srv/privado/semilla", "dsn_lectura": e.dsnLectura, "dsn_operador": e.dsnOperador, "salida": e.salida, "horas_validez_claves": 2}
	}
	casos := map[string]struct {
		fase   string
		cambio func(*escenario, map[string]any)
	}{
		"campo_extra":          {"preparar", func(e *escenario, m map[string]any) { m["dsn_propietario"] = e.dsnLectura }},
		"horas_cero":           {"preparar", func(e *escenario, m map[string]any) { m["horas_validez_claves"] = 0 }},
		"horas_25":             {"preparar", func(e *escenario, m map[string]any) { m["horas_validez_claves"] = 25 }},
		"horas_negativas":      {"verificar", func(e *escenario, m map[string]any) { m["horas_validez_claves"] = -1 }},
		"conjunto_inexistente": {"preparar", func(e *escenario, m map[string]any) { m["conjunto_capacidades"] = 9 }},
		"ruta_relativa":        {"preparar", func(e *escenario, m map[string]any) { m["directorio_material"] = "material" }},
		"salida_relativa":      {"verificar", func(e *escenario, m map[string]any) { m["salida"] = "salida" }},
		"sin_dsn_operador":     {"aplicar", func(e *escenario, m map[string]any) { m["dsn_operador"] = "" }},
		"operador_igual":       {"aplicar", func(e *escenario, m map[string]any) { m["dsn_operador"] = e.dsnLectura }},
		"dsn_con_role":         {"verificar", func(e *escenario, m map[string]any) { m["dsn_lectura"] = e.dsnLectura + " role=vec_propietario" }},
		"dsn_con_ROLE":         {"verificar", func(e *escenario, m map[string]any) { m["dsn_lectura"] = e.dsnLectura + " ROLE=x" }},
		"url_con_Role":         {"verificar", func(e *escenario, m map[string]any) { m["dsn_lectura"] = "postgres://l@" + "localhost/v?Role=x" }},
		"dsn_search_path":      {"verificar", func(e *escenario, m map[string]any) { m["dsn_lectura"] = e.dsnLectura + " search_path=x" }},
		"dsn_transaccion": {"verificar", func(e *escenario, m map[string]any) {
			m["dsn_lectura"] = e.dsnLectura + " default_transaction_read_only=off"
		}},
		"socket_escribible": {"verificar", func(e *escenario, m map[string]any) {
			m["dsn_lectura"] = "host=" + e.socketAbierto(t) + " user=l dbname=v"
		}},
		"socket_tmp": {"verificar", func(e *escenario, m map[string]any) { m["dsn_lectura"] = "host=/tmp user=l dbname=v" }},
		"socket_ausente": {"verificar", func(e *escenario, m map[string]any) {
			m["dsn_lectura"] = "host=/nonexistent/vec-socket user=l dbname=v"
		}},
		"dsn_remoto_sin_tls": {"verificar", func(e *escenario, m map[string]any) {
			m["dsn_lectura"] = "host=db.ejemplo.invalid user=l dbname=v sslmode=disable"
		}},
		"dsn_remoto_sin_verif": {"verificar", func(e *escenario, m map[string]any) {
			m["dsn_lectura"] = "host=db.ejemplo.invalid user=l dbname=v sslmode=require"
		}},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenario(t)
			m := base(e)
			c.cambio(e, m)
			e.config = e.escribirConfig(t, m)
			args := e.args(c.fase)
			if c.fase == "aplicar" {
				args = e.args(c.fase, "-acuse", "acuse.json")
			}
			code, d, _, errores := e.ejecutar(args)
			if code != 1 || d.Codigo != "configuracion_invalida" || e.llamadas != 0 {
				t.Fatal("configuración aceptada", code, d.Codigo)
			}
			if strings.Contains(errores, "secreto") || strings.Contains(errores, "/srv/") || strings.Contains(errores, "user=") {
				t.Fatal("expone configuración")
			}
		})
	}
	t.Run("clave_repetida", func(t *testing.T) {
		e := nuevoEscenario(t)
		ruta := filepath.Join(e.dir, "config.json")
		_ = os.Remove(ruta)
		if os.WriteFile(ruta, []byte(`{"salida":"`+e.salida+`","salida":"/otra","dsn_lectura":"`+e.dsnLectura+`"}`), 0600) != nil {
			t.Fatal("config")
		}
		if code, d, _, _ := e.ejecutar(e.args("verificar")); code != 1 || d.Codigo != "configuracion_invalida" {
			t.Fatal("clave repetida aceptada", d.Codigo)
		}
	})
}

func TestDSNAdmiteSoloApplicationName(t *testing.T) {
	e := nuevoEscenario(t)
	if dsnValido(e.dsnLectura+" application_name=vec-gobierno-usuarios") != nil {
		t.Fatal("application_name rechazado")
	}
}

func TestCatalogoIncompletoSinConexion(t *testing.T) {
	e := nuevoEscenario(t)
	b, err := os.ReadFile(e.textos)
	if err != nil {
		t.Fatal(err)
	}
	var datos datosTextos
	if json.Unmarshal(b, &datos) != nil {
		t.Fatal("catálogo")
	}
	delete(datos.Mensajes, "cadena_rechazada")
	b, _ = json.Marshal(datos)
	incompleto := filepath.Join(e.dir, "textos.json")
	if os.WriteFile(incompleto, b, 0600) != nil {
		t.Fatal("catálogo")
	}
	for _, ruta := range []string{incompleto, filepath.Join(e.dir, "ausente.json")} {
		e.textos = ruta
		code, _, salida, errores := e.ejecutar(e.args("verificar"))
		if code != 2 || salida != "" || errores != "{\"codigo\":\"catalogo_no_disponible\"}\n" || e.llamadas != 0 {
			t.Fatal("catálogo incompleto aceptado", code, errores)
		}
	}
}

func TestCatalogosMismasClaves(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		e := nuevoEscenario(t)
		e.textos = strings.Replace(e.textos, "/es/", "/"+idioma+"/", 1)
		if code, d, _, _ := e.ejecutar(e.args("verificar")); code != 0 || d.Codigo != "cadena_verificada" || d.Mensaje == "" {
			t.Fatal("catálogo", idioma, code, d.Codigo)
		}
	}
}

func TestSalidaDentroDeGitRechazada(t *testing.T) {
	e := nuevoEscenario(t)
	repo := filepath.Join(filepath.Dir(e.salida), "repo")
	dentro := filepath.Join(repo, "salida")
	if os.MkdirAll(filepath.Join(repo, ".git"), 0700) != nil || os.WriteFile(filepath.Join(repo, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0600) != nil || os.Mkdir(dentro, 0700) != nil {
		t.Fatal("repo")
	}
	e.config = e.escribirConfig(t, configuracionPrivada{DSNLectura: e.dsnLectura, Salida: dentro})
	if code, d, _, _ := e.ejecutar(e.args("verificar")); code != 1 || d.Codigo != "salida_insegura" || e.llamadas != 0 {
		t.Fatal("salida en Git aceptada", d.Codigo)
	}
	if os.Chmod(e.salida, 0755) != nil {
		t.Fatal("chmod")
	}
	e.config = e.escribirConfig(t, configuracionPrivada{DSNLectura: e.dsnLectura, Salida: e.salida})
	if code, d, _, _ := e.ejecutar(e.args("verificar")); code != 1 || d.Codigo != "salida_insegura" {
		t.Fatal("salida pública aceptada", d.Codigo)
	}
}

func TestAcuseConRutaOExistenteRechazado(t *testing.T) {
	e := nuevoEscenario(t)
	for _, acuse := range []string{"../acuse.json", "/tmp/acuse.json", "sub/acuse.json", "..", "."} {
		if code, d, _, _ := e.ejecutar(e.args("aplicar", "-acuse", acuse)); code != 1 || d.Codigo != "acuse_inseguro" || e.llamadas != 0 {
			t.Fatal("acuse con ruta aceptado", acuse, d.Codigo)
		}
	}
	if os.WriteFile(filepath.Join(e.salida, "acuse.json"), []byte("{}"), 0600) != nil {
		t.Fatal("acuse")
	}
	if code, d, _, _ := e.ejecutar(e.args("aplicar", "-acuse", "acuse.json")); code != 1 || d.Codigo != "acuse_inseguro" || e.llamadas != 0 {
		t.Fatal("acuse existente aceptado", d.Codigo)
	}
}

func TestAplicarCodigosDeSalida(t *testing.T) {
	casos := []struct {
		nombre  string
		acuse   bootstrap.ConfirmacionGobiernoUsuariosAdmin
		err     error
		exit    int
		codigo  string
		estdout bool
	}{
		{"permitido", bootstrap.ConfirmacionGobiernoUsuariosAdmin{Estado: "permitido", Codigo: "gobierno_usuarios_registrado"}, nil, 0, "gobierno_confirmado", true},
		{"replay", bootstrap.ConfirmacionGobiernoUsuariosAdmin{Estado: "permitido", Codigo: "gobierno_usuarios_replay"}, nil, 0, "gobierno_confirmado", true},
		{"denegado", bootstrap.ConfirmacionGobiernoUsuariosAdmin{Estado: "denegado", Codigo: "gobierno_usuarios_denegado"}, nil, 1, "gobierno_rechazado", false},
		{"error", bootstrap.ConfirmacionGobiernoUsuariosAdmin{Estado: "error", Codigo: "gobierno_usuarios_error"}, nil, 1, "gobierno_no_disponible", false},
		{"previo_commit", bootstrap.ConfirmacionGobiernoUsuariosAdmin{}, bootstrap.ErrGobiernoUsuariosAdmin, 1, "operacion_no_confirmada", false},
		{"commit_incierto", bootstrap.ConfirmacionGobiernoUsuariosAdmin{}, bootstrap.ErrCommitGobiernoUsuariosIndeterminado, 2, "commit_no_confirmado", false},
		{"acuse_no_guardado", bootstrap.ConfirmacionGobiernoUsuariosAdmin{Estado: "permitido", Codigo: "gobierno_usuarios_registrado"}, bootstrap.ErrAcuseGobiernoUsuariosNoGuardado, 2, "acuse_no_guardado", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e := nuevoEscenario(t)
			var dsnUsado, acuseUsado string
			e.ops.aplicar = func(_ context.Context, dsn string, _ time.Duration, _ *os.Root, acuse string) (bootstrap.ConfirmacionGobiernoUsuariosAdmin, error) {
				e.llamadas++
				dsnUsado, acuseUsado = dsn, acuse
				return c.acuse, c.err
			}
			code, d, salida, errores := e.ejecutar(e.args("aplicar", "-acuse", "acuse-1.json"))
			if code != c.exit || d.Codigo != c.codigo || (salida != "") != c.estdout || e.llamadas != 1 {
				t.Fatal("salida", code, d.Codigo)
			}
			if dsnUsado != e.dsnOperador || acuseUsado != "acuse-1.json" {
				t.Fatal("usa conexión o acuse ajenos")
			}
			if strings.Contains(salida+errores, "secreto") || strings.Contains(salida+errores, e.salida) {
				t.Fatal("expone secretos o rutas")
			}
			if c.codigo == "commit_no_confirmado" && (d.Estado != "indeterminado" || d.Confirmado) {
				t.Fatal("no indica incertidumbre")
			}
			if c.codigo == "acuse_no_guardado" && (!d.Confirmado || d.AcuseGuardado) {
				t.Fatal("no distingue COMMIT confirmado")
			}
			if c.nombre == "replay" && !d.Replay {
				t.Fatal("replay no indicado")
			}
		})
	}
}

func TestPrepararEmiteSoloHuellas(t *testing.T) {
	e := nuevoEscenario(t)
	var validez time.Duration
	var dsnUsado string
	preparar := e.ops.preparar
	e.ops.preparar = func(ctx context.Context, dsn string, l time.Duration, o bootstrap.MaterialOrigenGobiernoUsuariosAdmin, r *os.Root) (bootstrap.PreparacionGobiernoUsuariosAdmin, error) {
		validez, dsnUsado = o.ValidezClaves, dsn
		return preparar(ctx, dsn, l, o, r)
	}
	code, d, salida, _ := e.ejecutar(e.args("preparar"))
	if code != 0 || d.Codigo != "preparacion_lista" || d.Preparacion == nil || d.Preparacion.PlanSHA256 != strings.Repeat("a", 64) || validez != 2*time.Hour || dsnUsado != e.dsnLectura {
		t.Fatal("preparación", code, d.Codigo)
	}
	if strings.Contains(salida, "/srv/") || strings.Contains(salida, "user=") || strings.Contains(salida, e.salida) {
		t.Fatal("expone rutas o conexión")
	}
	e.ops.preparar = func(context.Context, string, time.Duration, bootstrap.MaterialOrigenGobiernoUsuariosAdmin, *os.Root) (bootstrap.PreparacionGobiernoUsuariosAdmin, error) {
		return bootstrap.PreparacionGobiernoUsuariosAdmin{}, bootstrap.ErrGobiernoUsuariosAdmin
	}
	if code, d, _, _ := e.ejecutar(e.args("preparar")); code != 1 || d.Codigo != "preparacion_fallida" || d.Preparacion != nil {
		t.Fatal("fallo de preparación", d.Codigo)
	}
}

func TestVerificarRechazoYNombreNuevo(t *testing.T) {
	e := nuevoEscenario(t)
	var nombre string
	e.ops.verificar = func(_ context.Context, _ string, _ time.Duration, _ *os.Root, n string) (auditoria.InformeVerificacion, error) {
		nombre = n
		return auditoria.InformeVerificacion{Estado: "rechazada"}, nil
	}
	if code, d, _, _ := e.ejecutar(e.args("verificar")); code != 1 || d.Codigo != "cadena_rechazada" || filepath.Base(nombre) != nombre || !strings.HasPrefix(nombre, "verificacion-cadena-20261005T090000Z") {
		t.Fatal("verificación", d.Codigo, nombre)
	}
}

// La fase version-bolsa no conecta: exige el conjunto 5, su bloque privado y
// el nombre del acuse de aplicar, y nunca muestra rutas ni motivos.
func TestVersionBolsaExigeConjuntoCincoYAcuse(t *testing.T) {
	e := nuevoEscenario(t)
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)}
	vb := &configuracionVersionBolsa{PoolGobierno: "/srv/privado/pools/version-bolsa.json", PoolCatalogo: "/srv/privado/pools/catalogo.json",
		Motivos: map[string]domain.ReferenciaEntradaCatalogo{gobiernoperfiles.AudienciaVersionarRolBolsaProponer: motivo, gobiernoperfiles.AudienciaVersionarRolBolsaAprobar: motivo}}
	var destino bootstrap.DestinoOverlayVersionBolsaAdmin
	var acuseUsado string
	e.ops.versionBolsa = func(_ *os.Root, d bootstrap.DestinoOverlayVersionBolsaAdmin, acuse string) error {
		e.llamadas++
		destino, acuseUsado = d, acuse
		return nil
	}
	e.config = e.escribirConfig(t, configuracionPrivada{Salida: e.salida, ConjuntoCapacidades: 5, VersionBolsa: vb})
	if code, d, _, _ := e.ejecutar(e.args("version-bolsa")); code != 1 || d.Codigo != "uso_invalido" || e.llamadas != 0 {
		t.Fatal("version-bolsa sin acuse aceptada", d.Codigo)
	}
	if code, d, _, _ := e.ejecutar(e.args("version-bolsa", "-acuse", "../acuse.json")); code != 1 || d.Codigo != "acuse_inseguro" || e.llamadas != 0 {
		t.Fatal("acuse con ruta aceptado", d.Codigo)
	}
	code, d, salida, errores := e.ejecutar(e.args("version-bolsa", "-acuse", "acuse-aplicar.json"))
	if code != 0 || d.Codigo != "version_bolsa_escrita" || e.llamadas != 1 || acuseUsado != "acuse-aplicar.json" ||
		destino.DirectorioSalida != e.salida || destino.PoolGobierno != vb.PoolGobierno || destino.PoolCatalogo != vb.PoolCatalogo || len(destino.Motivos) != 2 {
		t.Fatal("version-bolsa", code, d.Codigo)
	}
	if strings.Contains(salida+errores, "/srv/") || strings.Contains(salida+errores, e.salida) || strings.Contains(salida+errores, "motivo_") {
		t.Fatal("expone rutas o motivos")
	}
	e.ops.versionBolsa = func(*os.Root, bootstrap.DestinoOverlayVersionBolsaAdmin, string) error {
		return bootstrap.ErrGobiernoUsuariosAdmin
	}
	if code, d, _, _ := e.ejecutar(e.args("version-bolsa", "-acuse", "acuse-aplicar.json")); code != 1 || d.Codigo != "version_bolsa_fallida" {
		t.Fatal("fallo de version-bolsa", d.Codigo)
	}
	for nombre, c := range map[string]configuracionPrivada{
		"conjunto_4":    {Salida: e.salida, ConjuntoCapacidades: 4, VersionBolsa: vb},
		"sin_bloque":    {Salida: e.salida, ConjuntoCapacidades: 5},
		"pool_relativo": {Salida: e.salida, ConjuntoCapacidades: 5, VersionBolsa: &configuracionVersionBolsa{PoolGobierno: "pools/x.json", PoolCatalogo: vb.PoolCatalogo, Motivos: vb.Motivos}},
		"un_motivo":     {Salida: e.salida, ConjuntoCapacidades: 5, VersionBolsa: &configuracionVersionBolsa{PoolGobierno: vb.PoolGobierno, PoolCatalogo: vb.PoolCatalogo, Motivos: map[string]domain.ReferenciaEntradaCatalogo{gobiernoperfiles.AudienciaVersionarRolBolsaProponer: motivo}}},
	} {
		e.llamadas = 0
		e.config = e.escribirConfig(t, c)
		if code, d, _, _ := e.ejecutar(e.args("version-bolsa", "-acuse", "acuse-aplicar.json")); code != 1 || d.Codigo != "configuracion_invalida" || e.llamadas != 0 {
			t.Fatal("configuración aceptada", nombre, d.Codigo)
		}
	}
}
