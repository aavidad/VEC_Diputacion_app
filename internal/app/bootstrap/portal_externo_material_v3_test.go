package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

// materialesPublicadosPrueba deriva, como haría la preparación, el material
// de los consumidores pedidos a partir del material de desarrollo generado.
// Las coordenadas de gobierno son las de una primera publicación.
func materialesPublicadosPrueba(t *testing.T, consumidores ...string) map[string][]materialAtestacionContratacionTemporalDesarrollo {
	t.Helper()
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	raiz := cfg.DevelopmentMaterialDir
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	defer idempotencia.borrar()
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	if err != nil {
		t.Fatal(err)
	}
	defer derivador.borrar()
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPortalExternoV3())
	if err != nil {
		t.Fatal(err)
	}
	publicados := map[string][]materialAtestacionContratacionTemporalDesarrollo{}
	for _, consumidor := range consumidores {
		for _, audiencia := range audienciasConsumidorPortalExternoV3(consumidor) {
			d, ok := catalogo.descriptorPara(audiencia)
			if !ok {
				t.Fatalf("sin descriptor para %s", audiencia)
			}
			m, err := derivarMaterialConsumidorV3Desarrollo(base, d)
			if err != nil {
				t.Fatal(err)
			}
			m.privada = append(m.privada[:0:0], base.privada...)
			m.spki = append([]byte(nil), base.spki...)
			publicados[consumidor] = append(publicados[consumidor], m)
		}
	}
	return publicados
}

func directorioExternoPrueba(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "material-externo")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestMaterialV3PortalExternoSeEscribeYSeReconstruye(t *testing.T) {
	publicados := materialesPublicadosPrueba(t, "usuarios_preferencias", "portal_candidato")
	dir := directorioExternoPrueba(t)
	if err := escribirMaterialV3PortalExterno(dir, publicados); err != nil {
		t.Fatalf("escribir: %v", err)
	}
	inv, err := leerInventarioV3PortalExterno(dir)
	if err != nil {
		t.Fatalf("leer: %v", err)
	}
	if len(inv.Consumidores) != 2 || len(inv.Consumidores["portal_candidato"]) != len(accionesPropiasPortalDesarrollo()) {
		t.Fatalf("inventario incompleto: %+v", inv.Consumidores)
	}
	for _, consumidor := range []string{"usuarios_preferencias", "portal_candidato"} {
		materiales, err := materialesConsumidorV3PortalExterno(dir, inv, consumidor, inv.Configuracion)
		if err != nil {
			t.Fatalf("%s: %v", consumidor, err)
		}
		for i, m := range materiales {
			original := publicados[consumidor][i]
			if !bytes.Equal(m.claveHMAC, original.claveHMAC) || m.audienciaConsumo != original.audienciaConsumo ||
				m.claveHMACID != original.claveHMACID || m.claveHMACHuella != original.claveHMACHuella ||
				m.configuracionHuella != original.configuracionHuella || !bytes.Equal(m.spki, original.spki) {
				t.Fatalf("%s[%d]: material reconstruido distinto del publicado", consumidor, i)
			}
		}
		borrarMaterialesV3PortalExterno(materiales)
	}
	// El proceso externo no recibe la clave base ni la de otras audiencias.
	entradas, err := os.ReadDir(filepath.Join(dir, "externo", "v3"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 1+1+2+len(accionesPropiasPortalDesarrollo()) {
		t.Fatalf("ficheros inesperados en el material externo: %d", len(entradas))
	}
	for _, e := range entradas {
		info, err := e.Info()
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s no es 0600", e.Name())
		}
	}
	// Argumento de AD3-112: claves en el orden cerrado, configuración y raíz.
	contenido, err := materialJSONV3PortalExterno(inv, "portal_candidato", inv.Configuracion)
	if err != nil {
		t.Fatal(err)
	}
	var arg struct {
		Claves []struct {
			Audiencia string `json:"audiencia_consumo"`
		} `json:"claves"`
		Raiz map[string]any `json:"raiz"`
	}
	if json.Unmarshal(contenido, &arg) != nil || len(arg.Claves) != len(accionesPropiasPortalDesarrollo()) ||
		arg.Claves[len(arg.Claves)-1].Audiencia != "vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1" ||
		arg.Raiz["audiencia_despliegue"] != audienciaAtestacionContratacionTemporalDesarrollo {
		t.Fatalf("argumento AD3-112 inesperado: %s", contenido)
	}
}

func TestMaterialV3PortalExternoManipuladoSeRechaza(t *testing.T) {
	publicados := materialesPublicadosPrueba(t, "usuarios_preferencias")
	preparar := func(t *testing.T) (string, inventarioV3PortalExterno) {
		dir := directorioExternoPrueba(t)
		if err := escribirMaterialV3PortalExterno(dir, publicados); err != nil {
			t.Fatal(err)
		}
		inv, err := leerInventarioV3PortalExterno(dir)
		if err != nil {
			t.Fatal(err)
		}
		return dir, inv
	}
	dir, inv := preparar(t)
	clave := filepath.Join(dir, filepath.FromSlash(inv.Consumidores["usuarios_preferencias"][0].Archivo))
	if err := os.WriteFile(clave, bytes.Repeat([]byte{7}, 32), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := materialesConsumidorV3PortalExterno(dir, inv, "usuarios_preferencias", inv.Configuracion); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("clave cambiada aceptada: %v", err)
	}
	dir, inv = preparar(t)
	if err := os.WriteFile(filepath.Join(dir, "externo", "v3", "raiz.bin"), bytes.Repeat([]byte{9}, 32), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := materialesConsumidorV3PortalExterno(dir, inv, "usuarios_preferencias", inv.Configuracion); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("raiz cambiada aceptada: %v", err)
	}
	dir, inv = preparar(t)
	if err := os.Chmod(filepath.Join(dir, filepath.FromSlash(inv.Consumidores["usuarios_preferencias"][1].Archivo)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := materialesConsumidorV3PortalExterno(dir, inv, "usuarios_preferencias", inv.Configuracion); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("clave legible por otros aceptada: %v", err)
	}
	if _, err := materialesConsumidorV3PortalExterno(dir, inv, "mi_bolsa", inv.Configuracion); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("consumidor sin claves aceptado: %v", err)
	}
	for nombre, cambiar := range map[string]func(string) string{
		"orden de audiencias": func(s string) string {
			return strings.Replace(s, "consultar.externa_personal", "XX", 1)
		},
		"consumidor ajeno": func(s string) string {
			return strings.Replace(s, `"usuarios_preferencias"`, `"ct"`, 1)
		},
		"fichero fuera de externo/v3": func(s string) string {
			return strings.Replace(s, "externo/v3/usuarios_preferencias-1.bin", "kms/clave-maestra.bin", 1)
		},
	} {
		dir, _ := preparar(t)
		ruta := filepath.Join(dir, "externo", "v3", "inventario.json")
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, []byte(cambiar(string(contenido))), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := leerInventarioV3PortalExterno(dir); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
}

func TestPreparacionPortalExternoSoloDesdeElLadoInterno(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	destino := directorioExternoPrueba(t)
	if err := os.WriteFile(filepath.Join(destino, "portal-proceso.json"), []byte(`{"version":1,"portal":"externo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgExterno := cfg
	cfgExterno.PortalProceso = "externo"
	for nombre, caso := range map[string]struct {
		cfg          config.Config
		destino      string
		consumidores []string
	}{
		"desde el proceso externo": {cfgExterno, destino, []string{"usuarios_preferencias"}},
		"consumidor interno":       {cfg, destino, []string{"ct"}},
		"consumidor repetido":      {cfg, destino, []string{"mi_bolsa", "mi_bolsa"}},
		"sin consumidores":         {cfg, destino, nil},
		"destino = interno":        {cfg, cfg.DevelopmentMaterialDir, []string{"mi_bolsa"}},
	} {
		if _, err := PrepararMaterialPortalExterno(t.Context(), caso.cfg, OpcionesPreparacionPortalExterno{Destino: caso.destino, Consumidores: caso.consumidores}); err == nil {
			t.Fatalf("%s: preparacion aceptada", nombre)
		}
	}
	// Destino que no es un material externo separado.
	sinMarca := directorioExternoPrueba(t)
	if _, err := PrepararMaterialPortalExterno(t.Context(), cfg, OpcionesPreparacionPortalExterno{Destino: sinMarca, Consumidores: []string{"mi_bolsa"}}); err == nil {
		t.Fatal("destino sin marca externa aceptado")
	}
}

func TestInventarioExternoConservaConsumidoresYBorraClavesSinUso(t *testing.T) {
	publicados := materialesPublicadosPrueba(t, "usuarios_preferencias", "portal_candidato")
	dir := directorioExternoPrueba(t)
	if err := escribirMaterialV3PortalExterno(dir, map[string][]materialAtestacionContratacionTemporalDesarrollo{
		"usuarios_preferencias": publicados["usuarios_preferencias"]}); err != nil {
		t.Fatal(err)
	}
	huerfana := filepath.Join(dir, "externo", "v3", "mi_bolsa-1.bin")
	if err := os.WriteFile(huerfana, bytes.Repeat([]byte{1}, 32), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := escribirMaterialV3PortalExterno(dir, map[string][]materialAtestacionContratacionTemporalDesarrollo{
		"portal_candidato": publicados["portal_candidato"]}); err != nil {
		t.Fatal(err)
	}
	inv, err := leerInventarioV3PortalExterno(dir)
	if err != nil || len(inv.Consumidores) != 2 {
		t.Fatalf("la segunda preparacion debe conservar la primera: %v %+v", err, inv.Consumidores)
	}
	if _, err := os.Stat(huerfana); !os.IsNotExist(err) {
		t.Fatal("una clave sin referencia en el inventario debe borrarse")
	}
	// Con otra raíz no se mezclan consumidores de preparaciones distintas.
	otra := materialesPublicadosPrueba(t, "mi_bolsa")
	if err := escribirMaterialV3PortalExterno(dir, otra); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("mezcla de raices aceptada: %v", err)
	}
}

func TestPreparacionExigeIdempotenciaPropiaDelExterno(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	externo := directorioExternoPrueba(t)
	copiar := func(origen string) {
		if err := os.MkdirAll(filepath.Join(externo, "idempotencia"), 0o700); err != nil {
			t.Fatal(err)
		}
		entradas, _ := os.ReadDir(filepath.Join(origen, "idempotencia"))
		for _, e := range entradas {
			contenido, err := os.ReadFile(filepath.Join(origen, "idempotencia", e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(externo, "idempotencia", e.Name()), contenido, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	copiar(cfg.DevelopmentMaterialDir)
	if err := exigirIdempotenciaPropiaPortalExterno(cfg.DevelopmentMaterialDir, externo); err == nil {
		t.Fatal("un externo con la idempotencia del interno debe rechazarse")
	}
	propio, _ := generarMaterialDesarrolloPrueba(t)
	if err := os.RemoveAll(filepath.Join(externo, "idempotencia")); err != nil {
		t.Fatal(err)
	}
	copiar(propio.DevelopmentMaterialDir)
	if err := exigirIdempotenciaPropiaPortalExterno(cfg.DevelopmentMaterialDir, externo); err != nil {
		t.Fatalf("idempotencia propia rechazada: %v", err)
	}
}
