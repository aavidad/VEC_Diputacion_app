package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
	administracion "vec-diputacion-granada/internal/modules/administracion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteGobiernoModulosPrueba struct {
	cfg ports.ConfiguracionGobiernoModulosAprobada
}

func (f *fuenteGobiernoModulosPrueba) ConfiguracionAprobadaGobiernoModulos(context.Context) (ports.ConfiguracionGobiernoModulosAprobada, error) {
	return f.cfg, nil
}
func (f *fuenteGobiernoModulosPrueba) IdentidadRegistradaGobiernoModulos(context.Context, ports.EvidenciaUsoDecisionAutorizacion) (ports.IdentidadGobiernoModulosV3, error) {
	return ports.IdentidadGobiernoModulosV3{}, errors.New("no identidad")
}
func configGobiernoModulosPrueba() ports.ConfiguracionGobiernoModulosAprobada {
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos.admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "gobierno"}
	return ports.ConfiguracionGobiernoModulosAprobada{Version: 1, HuellaSHA256: strings.Repeat("a", 64), RegistroSHA256: strings.Repeat("b", 64), RegistroVersionRef: "registro:modulos:1", CatalogoID: "administracion.modulos", PerfilFijoRef: "prf_admin_modulos", FinalidadRef: finalidadGobiernoModulos, Registrados: []string{administracion.ModuleID, "vec.module.usuarios", "vec.module.cronos"}, Gobernados: []string{"vec.module.cronos"}, MotivoCrear: motivo, MotivoPublicar: motivo}
}
func TestGobiernoModulosConfiguracionNoAmpliaGobernados(t *testing.T) {
	cambios := map[string]func(*ports.ConfiguracionGobiernoModulosAprobada){
		"admin":       func(c *ports.ConfiguracionGobiernoModulosAprobada) { c.Gobernados = []string{administracion.ModuleID} },
		"identidad":   func(c *ports.ConfiguracionGobiernoModulosAprobada) { c.Gobernados = []string{"vec.module.usuarios"} },
		"desconocido": func(c *ports.ConfiguracionGobiernoModulosAprobada) { c.Gobernados = []string{"vec.module.ajeno"} },
		"duplicado": func(c *ports.ConfiguracionGobiernoModulosAprobada) {
			c.Registrados = append(c.Registrados, "vec.module.cronos")
		},
		"perfil ausente":       func(c *ports.ConfiguracionGobiernoModulosAprobada) { c.PerfilFijoRef = "" },
		"registro sin version": func(c *ports.ConfiguracionGobiernoModulosAprobada) { c.RegistroVersionRef = "" },
		"catalogo ajeno":       func(c *ports.ConfiguracionGobiernoModulosAprobada) { c.CatalogoID = "otro.modulos" },
	}
	for nombre, cambio := range cambios {
		t.Run(nombre, func(t *testing.T) {
			cfg := configGobiernoModulosPrueba()
			cambio(&cfg)
			r := &RepositorioGobiernoModulosPostgreSQL{fuente: &fuenteGobiernoModulosPrueba{cfg}}
			if _, err := r.configuracion(context.Background()); err == nil {
				t.Fatal("configuracion ampliada aceptada")
			}
		})
	}
	cfg := configGobiernoModulosPrueba()
	r := &RepositorioGobiernoModulosPostgreSQL{fuente: &fuenteGobiernoModulosPrueba{cfg}}
	c, err := r.configuracion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Gobernados[0] = "ajeno"
	if c.Gobernados[0] != "vec.module.cronos" {
		t.Fatal("alias al slice del provider")
	}
}
func catalogoGobiernoModulosPrueba(t *testing.T) ([]byte, []byte, string, expectativaReciboGobiernoModulos) {
	t.Helper()
	fecha := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	c := domain.CatalogoConfigurable{ID: "administracion.modulos", Version: 2, Revision: 1, VersionAnteriorRef: "administracion.modulos:1", ModuloID: administracion.ModuleID, Nombre: "Modulos", FuenteRef: "fuente:sintetica", MotivoCreacion: "Prueba", Estado: domain.EstadoCatalogoBorrador, CreadoPor: "actor:sintetico", CreadoEn: fecha, Entradas: []domain.EntradaCatalogoConfigurable{{Clave: "vec.module.cronos", Etiqueta: "Cronos", VigenteDesde: fecha, Atributos: map[string]string{"habilitado": "true"}}}}
	c, err := c.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := json.Marshal(c)
	h, _ := c.HuellaSHA256()
	w := reciboGobiernoModulosWire{Referencia: "recibo:sintetico", Clave: "clave:sintetica", Material: strings.Repeat("a", 64), Accion: ports.AccionCrearCatalogoConfigurable, CatalogoID: c.ID, Version: 2, Huella: h, Estado: c.Estado, Actor: c.CreadoPor, Auditoria: "audit:sintetica", Outbox: "outbox:sintetica", Fecha: fecha}
	recibo, _ := json.Marshal(w)
	sum := sha256.Sum256(recibo)
	return canon, recibo, hex.EncodeToString(sum[:]), expectativaReciboGobiernoModulos{clave: w.Clave, material: w.Material, accion: w.Accion, id: w.CatalogoID, version: 2, actor: w.Actor, catalogo: canon}
}
func TestGobiernoModulosReciboOriginalYManipulaciones(t *testing.T) {
	canon, recibo, h, x := catalogoGobiernoModulosPrueba(t)
	original, err := comprobarReciboGobiernoModulos(canon, recibo, h, true, x)
	if err != nil || !original.Recuperada {
		t.Fatalf("original rechazado: %v", err)
	}
	x.catalogo = append(bytes.Clone(canon), ' ')
	if _, err := comprobarReciboGobiernoModulos(canon, recibo, h, true, x); err != nil {
		t.Fatal("replay exige reloj/canon del nuevo intento")
	}
	if _, err := comprobarReciboGobiernoModulos(canon, recibo, h, false, x); !errors.Is(err, ports.ErrReciboCatalogoOperativoInvalido) {
		t.Fatal("postimagen inesperada aceptada")
	}
	x.catalogo = canon
	x.actor = "actor:ajeno"
	if _, err := comprobarReciboGobiernoModulos(canon, recibo, h, true, x); !errors.Is(err, ports.ErrReciboCatalogoOperativoInvalido) {
		t.Fatal("recibo de otro actor aceptado")
	}
	x.actor = "actor:sintetico"
	if _, err := comprobarReciboGobiernoModulos(append(bytes.Clone(canon), ' '), recibo, h, true, x); err == nil {
		t.Fatal("bytes catalogo alterados aceptados")
	}
	if _, err := comprobarReciboGobiernoModulos(canon, append(bytes.Clone(recibo), ' '), h, true, x); err == nil {
		t.Fatal("recibo alterado aceptado")
	}
	x.recuperacion = true
	if _, err := comprobarReciboGobiernoModulos(canon, recibo, h, false, x); err == nil {
		t.Fatal("recuperacion crea recibo nuevo")
	}
}

func TestGobiernoModulosLecturaHistoricaConConfiguracionPosterior(t *testing.T) {
	canon, _, _, _ := catalogoGobiernoModulosPrueba(t)
	var borrador domain.CatalogoConfigurable
	if err := json.Unmarshal(canon, &borrador); err != nil {
		t.Fatal(err)
	}
	publicado, err := borrador.Publicar("actor:publicador", "aprobacion:sintetica", "Publicación sintética", borrador.CreadoEn.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	cfg := configGobiernoModulosPrueba()
	cfg.Registrados = append(cfg.Registrados, "vec.module.personal")
	cfg.Gobernados = []string{"vec.module.personal"}
	if !catalogoLeidoGobiernoModulosValido(publicado, cfg, publicado.ID, publicado.Version, false) {
		t.Fatal("la configuración actual invalidó una publicación histórica")
	}
	if catalogoLeidoGobiernoModulosValido(publicado, cfg, publicado.ID, 0, true) {
		t.Fatal("la cabeza actual aceptó entradas ajenas al registro aprobado")
	}
}

func TestGobiernoModulosRecuperacionLigaRevisionYEstadoCAT6(t *testing.T) {
	cfg := configGobiernoModulosPrueba()
	orden := ports.RecuperacionCatalogoOperativo{ClaveIdempotencia: "clave:sintetica", CatalogoID: cfg.CatalogoID, Version: 2, HuellaMaterialSHA256: strings.Repeat("a", 64)}
	for _, tc := range []struct {
		operacion string
		estado    domain.EstadoCatalogoConfigurable
	}{
		{"crear", domain.EstadoCatalogoBorrador},
		{"publicar", domain.EstadoCatalogoPublicado},
	} {
		bruto, err := materialRecuperacionGobiernoModulos(orden, cfg, tc.estado, tc.operacion, []byte(`{"Esquema":"vec.catalogos.operacion.v1"}`))
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(bruto, &wire); err != nil || wire["revision"] != float64(1) || wire["estado"] != string(tc.estado) {
			t.Fatalf("contrato CAT6 de recuperación incompleto: %s, %v", bruto, err)
		}
	}
}

type txGobiernoModulosPrueba struct {
	pgx.Tx
	canon, recibo               []byte
	huella                      string
	commitErr, scanErr, execErr error
	onScan                      func()
	committed, rolled           bool
}

func (t *txGobiernoModulosPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), t.execErr
}
func (t *txGobiernoModulosPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaGobiernoModulosPrueba{t}
}
func (t *txGobiernoModulosPrueba) Commit(context.Context) error {
	t.committed = true
	return t.commitErr
}
func (t *txGobiernoModulosPrueba) Rollback(context.Context) error { t.rolled = true; return nil }

type filaGobiernoModulosPrueba struct{ tx *txGobiernoModulosPrueba }

func (f filaGobiernoModulosPrueba) Scan(dest ...any) error {
	if f.tx.scanErr != nil {
		return f.tx.scanErr
	}
	*dest[0].(*[]byte) = bytes.Clone(f.tx.canon)
	*dest[1].(*[]byte) = bytes.Clone(f.tx.recibo)
	*dest[2].(*string) = f.tx.huella
	*dest[3].(*bool) = false
	if f.tx.onScan != nil {
		f.tx.onScan()
	}
	return nil
}

type poolGobiernoModulosPrueba struct {
	tx      *txGobiernoModulosPrueba
	options pgx.TxOptions
}

func (p *poolGobiernoModulosPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.options = o
	return p.tx, nil
}
func TestGobiernoModulosReciboSoloTrasCommitYRollbackEnFallos(t *testing.T) {
	for _, nombre := range []string{"ok", "commit ambiguo", "recibo alterado", "SQL deniega", "contexto cancelado"} {
		t.Run(nombre, func(t *testing.T) {
			canon, recibo, h, x := catalogoGobiernoModulosPrueba(t)
			tx := &txGobiernoModulosPrueba{canon: canon, recibo: recibo, huella: h}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch nombre {
			case "commit ambiguo":
				tx.commitErr = errors.New("secreto privado")
			case "recibo alterado":
				tx.huella = strings.Repeat("0", 64)
			case "SQL deniega":
				tx.scanErr = &pgconn.PgError{Code: "42501", Message: "secreto privado"}
			case "contexto cancelado":
				tx.onScan = cancel
			}
			pool := &poolGobiernoModulosPrueba{tx: tx}
			r := &RepositorioGobiernoModulosPostgreSQL{pool: pool}
			// La exportación estructural del fixture no acredita permisos SQL. Esta
			// prueba ejercita únicamente la frontera transaccional del adaptador.
			result, err := r.ejecutar(ctx, consultaConfirmarGobiernoModulos, []byte("{}"), autorizacionRPTPrueba(t), x)
			if !tx.rolled || pool.options.IsoLevel != pgx.Serializable || pool.options.AccessMode != pgx.ReadWrite {
				t.Fatal("frontera transaccional incorrecta")
			}
			if nombre == "ok" {
				if err != nil || !tx.committed || result.Recibo.Referencia == "" {
					t.Fatalf("commit confirmado: %v", err)
				}
			} else {
				if err == nil || result.Recibo.Referencia != "" || strings.Contains(err.Error(), "secreto") {
					t.Fatalf("recibo antes de confirmacion/error privado: %v", err)
				}
				if nombre != "commit ambiguo" && tx.committed {
					t.Fatal("commit pese al fallo")
				}
			}
		})
	}
}
func TestGobiernoModulosCapacidadVaciaNoIniciaSQL(t *testing.T) {
	pool := &poolGobiernoModulosPrueba{tx: &txGobiernoModulosPrueba{}}
	r := &RepositorioGobiernoModulosPostgreSQL{pool: pool}
	result, err := r.ejecutar(context.Background(), consultaConfirmarGobiernoModulos, []byte("{}"), ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, expectativaReciboGobiernoModulos{})
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || result.Recibo.Referencia != "" || pool.options.IsoLevel != "" {
		t.Fatal("capacidad cero alcanza SQL")
	}
}

func TestGobiernoModulosDecisionesRestringidasYCambioBytesSemanticos(t *testing.T) {
	for _, bruto := range []string{`{}`, `{"campos_permitidos":null,"obligaciones":[]}`, `{"campos_permitidos":["estado"],"obligaciones":[]}`, `{"campos_permitidos":[],"obligaciones":["firma"]}`} {
		if sinRestriccionesGobiernoModulos([]byte(bruto)) {
			t.Fatal("restricción desconocida omitida")
		}
	}
	if !sinRestriccionesGobiernoModulos([]byte(`{"campos_permitidos":[],"obligaciones":[]}`)) {
		t.Fatal("decisión sin restricciones rechazada")
	}
	s := semanticoGobiernoModulos{Esquema: "vec.catalogos.operacion.v1", Accion: ports.AccionCrearCatalogoConfigurable, Contenido: json.RawMessage(`{"CatalogoID":"administracion.modulos"}`)}
	bruto, _ := json.Marshal(s)
	sum := sha256.Sum256(bruto)
	huella := hex.EncodeToString(sum[:])
	if _, err := leerSemanticoGobiernoModulos(bruto, huella); err != nil {
		t.Fatal(err)
	}
	if _, err := leerSemanticoGobiernoModulos(append(bytes.Clone(bruto), ' '), huella); err == nil {
		t.Fatal("bytes semánticos alterados aceptados")
	}
	alternativo := append(bytes.Clone(bruto), ' ')
	sum = sha256.Sum256(alternativo)
	if _, err := leerSemanticoGobiernoModulos(alternativo, hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("semántica reconstruida de JSONB aceptada")
	}
	canon, recibo, _, x := catalogoGobiernoModulosPrueba(t)
	duplicado := append([]byte(`{"referencia":"recibo:sintetico",`), recibo[1:]...)
	sum = sha256.Sum256(duplicado)
	if _, err := comprobarReciboGobiernoModulos(canon, duplicado, hex.EncodeToString(sum[:]), true, x); err == nil {
		t.Fatal("recibo con claves duplicadas aceptado")
	}
}
