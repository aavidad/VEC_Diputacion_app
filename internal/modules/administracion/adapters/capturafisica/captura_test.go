package capturafisica

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type controlPrueba struct {
	pasos []string
	fallo string
}

func (c *controlPrueba) paso(s string) error {
	c.pasos = append(c.pasos, s)
	if c.fallo == s {
		return ErrControl
	}
	return nil
}
func (c *controlPrueba) ComprobarExclusion(context.Context) error { return c.paso("exclusion") }
func (c *controlPrueba) DetenerPostgreSQL(context.Context) error  { return c.paso("stop") }
func (c *controlPrueba) ComprobarFrio(context.Context) error      { return c.paso("frio") }
func (c *controlPrueba) ReanudarPostgreSQL(context.Context) error { return c.paso("start") }
func preparar(t *testing.T) (*Capturador, copias.Inventario) {
	t.Helper()
	base := t.TempDir()
	pg := filepath.Join(base, "pg")
	dest := filepath.Join(base, "dest")
	if e := os.MkdirAll(filepath.Join(pg, "pg_tblspc"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(dest, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(pg, "PG_VERSION"), []byte("18\n"), 0600); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("../../../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if e != nil {
		t.Fatal(e)
	}
	var ejemplo struct {
		Manifiesto copias.Manifiesto `json:"manifiesto"`
	}
	if e = json.Unmarshal(b, &ejemplo); e != nil {
		t.Fatal(e)
	}
	inv := ejemplo.Manifiesto.Inventario
	c := &Capturador{Config: Configuracion{PGDATA: pg, Destino: dest, MaxBytes: 1 << 20, MaxEntradas: 100, TiempoRecuperacionSegundos: 2}, Control: &controlPrueba{}}
	for _, lista := range [][]copias.Artefacto{inv.Release.Binarios, inv.Release.Componentes} {
		for i := range lista {
			a := &lista[i]
			datos := []byte(a.ID)
			h := sha256.Sum256(datos)
			a.SHA256 = hex.EncodeToString(h[:])
			a.TamanoBytes = int64(len(datos))
			p := filepath.Join(base, a.ID)
			if e = os.WriteFile(p, datos, 0640); e != nil {
				t.Fatal(e)
			}
			c.Config.Fuentes = append(c.Config.Fuentes, Fuente{a.ID, a.Tipo, p})
		}
	}
	inv.PostgreSQL.Almacenes = []copias.Almacen{{ID: "almacen:docs", Tipo: "ficheros", SHA256: inv.Release.Componentes[0].SHA256}}
	almacen := filepath.Join(base, "almacen")
	if e = os.MkdirAll(filepath.Join(almacen, "vacio"), 0750); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(almacen, "doc"), []byte("synthetic-document"), 0640); e != nil {
		t.Fatal(e)
	}
	c.Config.Fuentes = append(c.Config.Fuentes, Fuente{"almacen:docs", "ficheros", almacen})
	return c, inv
}
func TestCapturaCompletaPreservaBytesYMetadatos(t *testing.T) {
	c, inv := preparar(t)
	antes, e := os.Stat(filepath.Join(c.Config.PGDATA, "PG_VERSION"))
	if e != nil {
		t.Fatal(e)
	}
	artefactos, e := c.Capturar(context.Background(), inv)
	if e != nil {
		t.Fatal(e)
	}
	if len(artefactos) != len(c.Config.Fuentes)+1+len(inv.Release.Binarios)+len(inv.Release.Componentes) {
		t.Fatal("missing component")
	}
	ctrl := c.Control.(*controlPrueba)
	if !reflect.DeepEqual(ctrl.pasos, []string{"exclusion", "stop", "frio", "exclusion", "frio", "exclusion", "start"}) {
		t.Fatal(ctrl.pasos)
	}
	var indice Indice
	idx, e := os.ReadFile(filepath.Join(c.Directorio, "indice.json"))
	if e != nil || json.Unmarshal(idx, &indice) != nil {
		t.Fatal(e)
	}
	for _, entrada := range indice.Entradas {
		a := entrada.Artefacto
		b, e := os.ReadFile(filepath.Join(c.Directorio, entrada.Archivo))
		if e != nil {
			t.Fatal(e)
		}
		h := sha256.Sum256(b)
		if a.SHA256 != hex.EncodeToString(h[:]) || a.TamanoBytes != int64(len(b)) {
			t.Fatal("bad artifact")
		}
	}
	f, e := os.Open(filepath.Join(c.Directorio, "componente-0000.tar"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	tr := tar.NewReader(f)
	encontrado := false
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		if h.Name == "contenido/PG_VERSION" {
			encontrado = true
			b, e := io.ReadAll(tr)
			if e != nil || string(b) != "18\n" || h.Mode != 0600 || !h.ModTime.Equal(antes.ModTime()) {
				t.Fatalf("bad header/content %#v %v", h, e)
			}
		}
	}
	if !encontrado {
		t.Fatal("missing pg file")
	}
	despues, _ := os.Stat(filepath.Join(c.Config.PGDATA, "PG_VERSION"))
	if !antes.ModTime().Equal(despues.ModTime()) || antes.Mode() != despues.Mode() {
		t.Fatal("source modified")
	}
}
func TestRechazaYRecuperaSinSalida(t *testing.T) {
	for _, caso := range []string{"huella", "symlink", "fifo", "tablespace", "bytes", "entradas", "stop", "frio", "start"} {
		t.Run(caso, func(t *testing.T) {
			c, inv := preparar(t)
			switch caso {
			case "huella":
				if e := os.WriteFile(c.Config.Fuentes[0].Ruta, []byte("changed"), 0600); e != nil {
					t.Fatal(e)
				}
			case "symlink":
				if e := os.Symlink(c.Config.Fuentes[0].Ruta, filepath.Join(c.Config.PGDATA, "link")); e != nil {
					t.Fatal(e)
				}
			case "fifo":
				if e := syscall.Mkfifo(filepath.Join(c.Config.PGDATA, "pipe"), 0600); e != nil {
					t.Fatal(e)
				}
			case "tablespace":
				if e := os.Symlink(c.Config.Fuentes[0].Ruta, filepath.Join(c.Config.PGDATA, "pg_tblspc", "123")); e != nil {
					t.Fatal(e)
				}
			case "bytes":
				c.Config.MaxBytes = 1
			case "entradas":
				c.Config.MaxEntradas = 1
			default:
				c.Control.(*controlPrueba).fallo = caso
			}
			arts, e := c.Capturar(context.Background(), inv)
			if e == nil || len(arts) != 0 || c.Directorio != "" {
				t.Fatalf("false success %v", e)
			}
			dirs, e := os.ReadDir(c.Config.Destino)
			if e != nil || len(dirs) != 0 {
				t.Fatal("left output", e)
			}
			pasos := c.Control.(*controlPrueba).pasos
			if pasos[len(pasos)-1] != "start" {
				t.Fatal("not recovered", pasos)
			}
		})
	}
}
func TestPrecondicionNoTocaPlataforma(t *testing.T) {
	c, inv := preparar(t)
	c.Config.Fuentes = c.Config.Fuentes[:1]
	_, e := c.Capturar(context.Background(), inv)
	if !errors.Is(e, ErrConfiguracion) || len(c.Control.(*controlPrueba).pasos) != 0 {
		t.Fatal(e)
	}
}

func TestSolapamientoIncluyeRaizSinConfundirPrefijos(t *testing.T) {
	for _, caso := range []struct {
		a, b     string
		esperado bool
	}{
		{"/", "/synthetic", true}, {"/synthetic", "/", true},
		{"/synthetic/pg", "/synthetic/pg/subdir", true},
		{"/synthetic/pg", "/synthetic/pg-other", false},
	} {
		if solapan(caso.a, caso.b) != caso.esperado {
			t.Fatalf("solapamiento %q %q", caso.a, caso.b)
		}
	}
	c, inv := preparar(t)
	c.Config.PGDATA = "/"
	if _, e := c.Capturar(context.Background(), inv); !errors.Is(e, ErrConfiguracion) || len(c.Control.(*controlPrueba).pasos) != 0 {
		t.Fatal("root source reached platform", e)
	}
}
