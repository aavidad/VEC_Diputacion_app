package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	gobiernoperfiles "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

// El conjunto 5 de Go es exactamente la fila que inserta AD234: mismas
// audiencias y tramos, en el mismo orden, y conserva el conjunto 4.
func TestConjuntoCincoCoincideConAD234(t *testing.T) {
	sql, err := os.ReadFile("../../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000234_conjunto_capacidades_version_bolsa.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	fila := regexp.MustCompile(`(?s)INSERT INTO vec_autorizacion_atestada_v3\.conjunto_audiencias_capacidad_admin_v1\(version,audiencias,segmentos\) VALUES\s*\(5,ARRAY\[(.*?)\],\s*ARRAY\[(.*?)\]\);`).FindSubmatch(sql)
	if fila == nil {
		t.Fatal("AD234 sin la fila del conjunto 5")
	}
	lista := func(b []byte) []string {
		var r []string
		for _, m := range regexp.MustCompile(`'([^']*)'`).FindAllSubmatch(b, -1) {
			r = append(r, string(m[1]))
		}
		return r
	}
	audiencias, segmentos := lista(fila[1]), lista(fila[2])
	cuatro, _ := AudienciasConjuntoCapacidadesAdmin(4)
	cinco, ok := AudienciasConjuntoCapacidadesAdmin(5)
	if !ok || len(cinco) != 8 || len(audiencias) != 8 || len(segmentos) != 8 {
		t.Fatalf("conjunto 5 incompleto: go=%d sql=%d/%d", len(cinco), len(audiencias), len(segmentos))
	}
	emisor := regexp.MustCompile(`^emisor:admin:[a-z0-9:._-]{1,120}$`)
	for i, a := range cinco {
		if a.Audiencia != audiencias[i] || a.Segmento != segmentos[i] || !emisor.MatchString(a.EmisorID) {
			t.Fatalf("audiencia %d distinta de AD234: %+v", i, a)
		}
		if i < len(cuatro) && a != cuatro[i] {
			t.Fatalf("el conjunto 5 cambia la audiencia %d del conjunto 4", i)
		}
	}
	if cinco[6].Audiencia != gobiernoperfiles.AudienciaVersionarRolBolsaProponer || cinco[7].Audiencia != gobiernoperfiles.AudienciaVersionarRolBolsaAprobar {
		t.Fatal("las dos últimas audiencias no son las de B1")
	}
}

type salidaVersionBolsaPrueba struct {
	dir    string
	raiz   *os.Root
	reloj  relojGobiernoUsuariosPrueba
	plan   []byte
	claves map[string][]byte
}

// prepararSalidaVersionBolsaPrueba deja en una carpeta 0700 lo que dejan
// preparar y aplicar del conjunto indicado: configuración, material, plan y
// un acuse permitido ligado a ambos.
func prepararSalidaVersionBolsaPrueba(t *testing.T, conjunto uint64) salidaVersionBolsaPrueba {
	t.Helper()
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	audiencias, ok := AudienciasConjuntoCapacidadesAdmin(conjunto)
	if !ok {
		t.Fatal("conjunto")
	}
	cfg.ConjuntoVersion = conjunto
	cfg.Entradas = descriptoresClavesUsuariosAdmin(audiencias, 20261009, 4, 9, reloj.Ahora(), time.Hour)
	m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	var material bytes.Buffer
	if m.EscribirMaterialPrivado(&material) != nil {
		t.Fatal("material")
	}
	conf, _, _ := m.Configuracion()
	claves := map[string][]byte{}
	for _, e := range conf.EntradasCapacidad {
		claves[e.Audiencia] = append([]byte(nil), e.Material...)
	}
	datos, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := planGobiernoUsuariosAdmin{Version: 2, OperacionRef: "gca_" + strings.Repeat("c", 32), PreparadoEn: reloj.Ahora(), CaducaEn: reloj.Ahora().Add(time.Hour), PreimagenSHA256: strings.Repeat("c", 64), ConjuntoVersion: conjunto}
	p.Configuracion.Revision, p.Configuracion.Secuencia, p.Configuracion.Huella = cfg.Gobierno.Revision, cfg.Gobierno.Secuencia, cfg.Gobierno.HuellaSHA256
	p.Configuracion.PublicadaEn, p.Configuracion.ExpiraEn = cfg.Gobierno.PublicadaEn, cfg.Gobierno.ExpiraEn
	for i := range audiencias {
		p.Ordenes = append(p.Ordenes, uint64(100+i))
	}
	plan, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "salida")
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("salida")
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raiz.Close() })
	for nombre, b := range map[string][]byte{ArchivoConfiguracionGobiernoUsuarios: datos, ArchivoMaterialGobiernoUsuarios: material.Bytes(), ArchivoPlanGobiernoUsuarios: plan} {
		if escribirPrivadoGobiernoUsuarios(raiz, nombre, b) != nil {
			t.Fatal(nombre)
		}
	}
	s := salidaVersionBolsaPrueba{dir: dir, raiz: raiz, reloj: reloj, plan: plan, claves: claves}
	s.escribirAcuse(t, "acuse-aplicar.json", "permitido", plan, material.Bytes())
	return s
}

func (s salidaVersionBolsaPrueba) escribirAcuse(t *testing.T, nombre, estado string, plan, material []byte) {
	t.Helper()
	ph, mh := sha256.Sum256(plan), sha256.Sum256(material)
	sha := hex.EncodeToString(ph[:])
	var p planGobiernoUsuariosAdmin
	if json.Unmarshal(plan, &p) != nil {
		t.Fatal("plan")
	}
	en := s.reloj.Ahora().UTC()
	intento := map[string]any{"auditoria_ref": "aud_v3_gui_" + strings.Repeat("e", 32), "secuencia": 8, "huella_sha256": strings.Repeat("f", 64),
		"correlacion_ref": "correlacion_" + strings.Repeat("1", 32), "registrada_en": en, "solicitud_sha256": sha}
	var recibo any
	codigo := "gobierno_usuarios_" + estado
	if estado == "permitido" {
		codigo = "gobierno_usuarios_registrado"
		recibo = map[string]any{"auditoria_ref": "aud_v3_gu_" + sha[:32], "secuencia": 7, "huella_sha256": strings.Repeat("a", 64),
			"correlacion_ref": "correlacion_" + strings.Repeat("2", 32), "registrada_en": en, "plan_sha256": sha, "preimagen_sha256": p.PreimagenSHA256,
			"claves_sha256": strings.Repeat("b", 64), "material_sha256": hex.EncodeToString(mh[:]), "configuracion_ref": p.Configuracion.Revision, "replay": false}
	}
	b, err := json.Marshal(map[string]any{"estado": estado, "codigo": codigo, "recibo": recibo, "auditoria_intento": intento})
	if err != nil || escribirPrivadoGobiernoUsuarios(s.raiz, nombre, b) != nil {
		t.Fatal("acuse")
	}
}

func destinoVersionBolsaPrueba(dir string) DestinoOverlayVersionBolsaAdmin {
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)}
	return DestinoOverlayVersionBolsaAdmin{DirectorioSalida: dir, PoolGobierno: "/srv/privado/pools/version-bolsa.json", PoolCatalogo: "/srv/privado/pools/catalogo.json",
		Motivos: map[string]domain.ReferenciaEntradaCatalogo{gobiernoperfiles.AudienciaVersionarRolBolsaProponer: motivo, gobiernoperfiles.AudienciaVersionarRolBolsaAprobar: motivo}}
}

// Con el acuse permitido de un conjunto 5, escribe las dos claves B1 en
// ficheros 0600 propios y un overlay con exactamente esas dos entradas.
func TestOverlayVersionBolsaConDosClavesPropias(t *testing.T) {
	s := prepararSalidaVersionBolsaPrueba(t, 5)
	d := destinoVersionBolsaPrueba(s.dir)
	if err := EscribirOverlayVersionBolsaAdmin(s.raiz, d, "acuse-aplicar.json", s.reloj); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.dir, ArchivoOverlayVersionBolsa))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte("secreto")) || bytes.Contains(b, []byte("material\"")) {
		t.Fatal("el overlay lleva material en línea")
	}
	var o overlayVersionBolsa
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&o) != nil || o.PoolGobierno != d.PoolGobierno || o.PoolCatalogo != d.PoolCatalogo || len(o.Motivos) != 2 || len(o.Confianza.EntradasCapacidad) != 2 {
		t.Fatal("overlay incompleto")
	}
	archivos := map[string]string{gobiernoperfiles.AudienciaVersionarRolBolsaProponer: ArchivoMaterialVersionBolsaPropuesta, gobiernoperfiles.AudienciaVersionarRolBolsaAprobar: ArchivoMaterialVersionBolsaCierre}
	for _, e := range o.Confianza.EntradasCapacidad {
		if e.MaterialArchivo != filepath.Join(s.dir, archivos[e.Audiencia]) || e.EmisorID != EmisorVersionBolsaAdmin || e.Estado != "emision" || !e.RevocadaEn.IsZero() {
			t.Fatalf("entrada B1 incorrecta: %+v", e)
		}
		i, err := os.Lstat(e.MaterialArchivo)
		secreto, _ := os.ReadFile(e.MaterialArchivo)
		if err != nil || i.Mode().Perm() != 0600 || !bytes.Equal(secreto, s.claves[e.Audiencia]) || len(secreto) != 32 {
			t.Fatal("clave B1 sin fichero 0600 propio o distinta de la publicada")
		}
	}
	for _, f := range []string{ArchivoOverlayVersionBolsa} {
		if i, err := os.Lstat(filepath.Join(s.dir, f)); err != nil || i.Mode().Perm() != 0600 {
			t.Fatal("overlay sin permisos 0600")
		}
	}
	// Nada se sobrescribe: una segunda generación en la misma carpeta falla.
	if EscribirOverlayVersionBolsaAdmin(s.raiz, d, "acuse-aplicar.json", s.reloj) == nil {
		t.Fatal("segunda generación aceptada")
	}
}

func TestOverlayVersionBolsaRechazaSinPublicacionConfirmada(t *testing.T) {
	casos := map[string]func(*testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string){
		"conjunto_4": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 4)
			return s, destinoVersionBolsaPrueba(s.dir), "acuse-aplicar.json"
		},
		"acuse_denegado": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			material, _ := os.ReadFile(filepath.Join(s.dir, ArchivoMaterialGobiernoUsuarios))
			s.escribirAcuse(t, "acuse-denegado.json", "denegado", s.plan, material)
			return s, destinoVersionBolsaPrueba(s.dir), "acuse-denegado.json"
		},
		"acuse_de_otro_material": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			s.escribirAcuse(t, "acuse-otro.json", "permitido", s.plan, []byte(`{"claves":[]}`))
			return s, destinoVersionBolsaPrueba(s.dir), "acuse-otro.json"
		},
		"acuse_inexistente": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			return s, destinoVersionBolsaPrueba(s.dir), "acuse-replay.json"
		},
		"acuse_con_nombre_del_overlay": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			return s, destinoVersionBolsaPrueba(s.dir), ArchivoOverlayVersionBolsa
		},
		"gobierno_caducado": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			s.reloj.ahora = s.reloj.ahora.Add(48 * time.Hour)
			return s, destinoVersionBolsaPrueba(s.dir), "acuse-aplicar.json"
		},
		"pools_iguales": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			d := destinoVersionBolsaPrueba(s.dir)
			d.PoolCatalogo = d.PoolGobierno
			return s, d, "acuse-aplicar.json"
		},
		"pool_en_la_carpeta_del_dia": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			d := destinoVersionBolsaPrueba(s.dir)
			d.PoolGobierno = filepath.Join(s.dir, "pool.json")
			return s, d, "acuse-aplicar.json"
		},
		"motivo_ajeno": func(t *testing.T) (salidaVersionBolsaPrueba, DestinoOverlayVersionBolsaAdmin, string) {
			s := prepararSalidaVersionBolsaPrueba(t, 5)
			d := destinoVersionBolsaPrueba(s.dir)
			m := d.Motivos[gobiernoperfiles.AudienciaVersionarRolBolsaAprobar]
			d.Motivos = map[string]domain.ReferenciaEntradaCatalogo{gobiernoperfiles.AudienciaVersionarRolBolsaProponer: m, "vec.admin.usuarios.listar.v1": m}
			return s, d, "acuse-aplicar.json"
		},
	}
	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, d, acuse := preparar(t)
			if EscribirOverlayVersionBolsaAdmin(s.raiz, d, acuse, s.reloj) == nil {
				t.Fatal("overlay escrito sin publicación B1 confirmada")
			}
			for _, f := range []string{ArchivoOverlayVersionBolsa, ArchivoMaterialVersionBolsaPropuesta, ArchivoMaterialVersionBolsaCierre} {
				if _, err := os.Lstat(filepath.Join(s.dir, f)); err == nil {
					t.Fatal("deja ficheros tras el rechazo:", f)
				}
			}
		})
	}
}
