package inventariocopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func fixture(t *testing.T) (string, Descriptor, copias.Inventario) {
	t.Helper()
	dir := t.TempDir()
	huella := strings.Repeat("a", 64)
	modulos := []copias.Modulo{{ID: "administracion", EsquemaSHA256: huella, Migraciones: []copias.Migracion{{ID: "000001", SHA256: huella}, {ID: "000002", SHA256: huella}}}}
	inventario := copias.Inventario{
		FormatoVersion: copias.FormatoVersion, Ref: "inventario:sintetico", Completo: true,
		PostgreSQL: copias.PostgreSQL{Version: "18.0", RuntimeSHA256: huella, Plataforma: "linux-amd64", ClusterRef: "cluster:sintetico", Bases: []string{"base:sintetica"}, Extensiones: []copias.Extension{}, Almacenes: []copias.Almacen{}, AmbitoCompleto: true},
		Release:    copias.Release{ID: "release:sintetica", Commit: strings.Repeat("a", 40), Plataforma: "linux-amd64", EsquemaEsperado: modulos}, Modulos: modulos,
	}
	for _, id := range []string{"pg_dump", "pg_restore", "psql", "postgres"} {
		inventario.PostgreSQL.Herramientas = append(inventario.PostgreSQL.Herramientas, copias.Herramienta{ID: id, Version: "18.0", SHA256: huella})
	}
	descriptor := Descriptor{}
	for _, tipo := range []string{"binario", "web", "catalogos", "material", "configuracion", "ficheros"} {
		contenido := []byte("contenido sintetico " + tipo)
		suma := sha256.Sum256(contenido)
		artefacto := copias.Artefacto{ID: tipo, Tipo: tipo, SHA256: hex.EncodeToString(suma[:]), TamanoBytes: int64(len(contenido))}
		if tipo == "binario" {
			inventario.Release.Binarios = append(inventario.Release.Binarios, artefacto)
		} else {
			inventario.Release.Componentes = append(inventario.Release.Componentes, artefacto)
		}
		if err := os.WriteFile(filepath.Join(dir, tipo), contenido, 0600); err != nil {
			t.Fatal(err)
		}
		descriptor.Rutas = append(descriptor.Rutas, Ruta{ID: tipo, RutaRelativa: tipo})
	}
	descriptor.Inventario = inventario
	// El observado se modifica independientemente del descriptor previsto.
	datos, err := json.Marshal(inventario)
	if err != nil {
		t.Fatal(err)
	}
	var observado copias.Inventario
	if err := json.Unmarshal(datos, &observado); err != nil {
		t.Fatal(err)
	}
	if rs := copias.ValidarInventario(inventario); len(rs) != 0 {
		t.Fatalf("fixture no valida: %+v", rs)
	}
	return dir, descriptor, observado
}

func TestInventarioCoincideSinAutorizarEfectos(t *testing.T) {
	dir, descriptor, observado := fixture(t)
	slices.Reverse(observado.Release.Componentes)
	slices.Reverse(observado.Modulos[0].Migraciones)
	slices.Reverse(descriptor.Rutas)
	informe := Inventariar(dir, descriptor, observado)
	if informe.Resultado.Estado != copias.Compatible || informe.AutorizaCopia || informe.AutorizaRestauracion || informe.ProcedenciaPostgreSQL != "declaracion_offline_no_autenticada" {
		t.Fatalf("informe=%+v", informe)
	}
}

func TestFaltantesYBytesDistintos(t *testing.T) {
	for _, tipo := range []string{"binario", "material"} {
		t.Run(tipo, func(t *testing.T) {
			dir, descriptor, observado := fixture(t)
			if err := os.Remove(filepath.Join(dir, tipo)); err != nil {
				t.Fatal(err)
			}
			if informe := Inventariar(dir, descriptor, observado); informe.Resultado.Estado != copias.NoComprobable {
				t.Fatalf("informe=%+v", informe)
			}
		})
	}
	for _, cambiarTamano := range []bool{false, true} {
		dir, descriptor, observado := fixture(t)
		contenido := "contenido sintetico CONFIGURACION"
		if cambiarTamano {
			contenido = "corto"
		}
		if err := os.WriteFile(filepath.Join(dir, "configuracion"), []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
		declaradoAntes := copias.HuellaInventario(observado)
		informe := Inventariar(dir, descriptor, observado)
		if informe.Resultado.Estado != copias.Incompatible {
			t.Fatalf("informe=%+v", informe)
		}
		if copias.HuellaInventario(observado) != declaradoAntes {
			t.Fatal("se ha modificado el inventario declarado")
		}
		if !cambiarTamano {
			encontrada := false
			for _, razon := range informe.Resultado.Razones {
				if razon.Codigo == "huella_archivo_distinta" {
					encontrada = razon.Esperado == copias.HuellaInventario(descriptor.Inventario) && len(razon.Obtenido) == 64 && razon.Esperado != razon.Obtenido
				}
			}
			if !encontrada {
				t.Fatal("falta el contraste de huellas compuestas exactas")
			}
		}
		salida, _ := json.Marshal(informe)
		if strings.Contains(string(salida), "CONFIGURACION") || strings.Contains(string(salida), descriptor.Inventario.Release.Componentes[3].SHA256) || strings.Contains(string(salida), dir) {
			t.Fatal("salida contiene contenido, huella privada o ruta")
		}
	}
}

func TestNoInfiereMigracionesNiCompletitud(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*copias.Inventario)
	}{
		{"huella_sql", func(i *copias.Inventario) { i.Modulos[0].Migraciones[0].SHA256 = strings.Repeat("b", 64) }},
		{"solo_numero_mayor", func(i *copias.Inventario) { i.Modulos[0].Migraciones = i.Modulos[0].Migraciones[1:] }},
		{"modulo_omitido", func(i *copias.Inventario) { i.Modulos = nil }},
		{"esquema_omitido", func(i *copias.Inventario) { i.Modulos[0].EsquemaSHA256 = "" }},
		{"sin_observacion", func(i *copias.Inventario) { *i = copias.Inventario{} }},
		{"incompleto", func(i *copias.Inventario) { i.Completo = false }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			dir, descriptor, observado := fixture(t)
			caso.cambiar(&observado)
			if informe := Inventariar(dir, descriptor, observado); informe.Resultado.Estado == copias.Compatible || len(informe.Resultado.Razones) == 0 {
				t.Fatalf("informe=%+v", informe)
			}
		})
	}
}

func TestDescriptorDebeReconciliarTodasLasRutas(t *testing.T) {
	casos := []func(*Descriptor){
		func(d *Descriptor) { d.Rutas = d.Rutas[1:] },
		func(d *Descriptor) { d.Rutas = append(d.Rutas, d.Rutas[0]) },
		func(d *Descriptor) { d.Rutas[0].ID = "desconocido" },
		func(d *Descriptor) { d.Rutas[0].RutaRelativa = "../privado" },
		func(d *Descriptor) { d.Rutas[1].RutaRelativa = d.Rutas[0].RutaRelativa },
	}
	for _, cambiar := range casos {
		dir, descriptor, observado := fixture(t)
		cambiar(&descriptor)
		if informe := Inventariar(dir, descriptor, observado); informe.Resultado.Estado != copias.NoComprobable {
			t.Fatalf("informe=%+v", informe)
		}
	}
}
