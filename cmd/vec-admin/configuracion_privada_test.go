package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func archivoPrivadoPrueba(t *testing.T, contenido string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "material.json")
	if err := os.WriteFile(p, []byte(contenido), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func configuracionPrivadaPrueba(t *testing.T) configuracionPerfilesPrivada {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return configuracionPerfilesPrivada{Pools: poolsPerfilesPrivados{FuenteAutorizacion: filepath.Join(dir, "fuente.json"), RegistroAutorizacion: filepath.Join(dir, "registro.json"), Motivos: filepath.Join(dir, "motivos.json"), RegistroSesiones: filepath.Join(dir, "sesiones.json"), RevalidacionSesiones: filepath.Join(dir, "revalidacion.json"), CuentasADMIN: filepath.Join(dir, "cuentas.json"), ActosADMIN: filepath.Join(dir, "actos.json"), AuditoriaFrontera: filepath.Join(dir, "auditoria.json")}, ConfianzaJSON: confianzaPrivadaPrueba(t, dir), Firmante: firmantePerfilesPrivado{ClaveID: "raiz:privada", Audiencia: "audiencia:admin", PrefijoEvidencia: "evidencia:admin", ClavePrivadaArchivo: filepath.Join(dir, "semilla.bin"), PublicaEsperadaBase64: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}, Identidad: identidadPerfilesPrivada{DirectorioMaterial: dir, RutaConfiguracionHMAC: filepath.Join(dir, "seudonimos.json"), EspacioIdentidad: "espacio:admin", DominioRef: "dominio:ref", EspacioClave: "espacio:clave", DominioHMAC: "dominio:hmac", IncluirCuentaOrdinaria: false}, MotivosLectura: map[string]domain.ReferenciaEntradaCatalogo{"administracion.perfiles.consultar": {CatalogoID: "admin.motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "lectura"}}, CatalogoMotivosID: "admin.motivos", VigenciaDecisionSegundos: 5, TimeoutArranqueSegundos: 10, ActivosDirectorio: dir}
}

func TestCargarConfiguracionPrivadaSoloMetadataYPaths(t *testing.T) {
	cfg := configuracionPrivadaPrueba(t)
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := archivoPrivadoPrueba(t, string(b))
	leida, err := cargarConfiguracionPerfilesPrivada(p)
	if err != nil {
		t.Fatal(err)
	}
	if leida.Pools != cfg.Pools || leida.Firmante != cfg.Firmante || leida.Identidad != cfg.Identidad || leida.VigenciaDecisionSegundos != 5 {
		t.Fatal("configuracion sustituida")
	}
}

func TestJSONPrivadoRechazaDuplicadosDesconocidosNullYOmisiones(t *testing.T) {
	for _, b := range []string{`{"dsn":"a","dsn":"b"}`, `{"dsn":"a","\u0064sn":"b"}`, `{"dsn":"a","extra":1}`, `{"dsn":null}`, `{}`, `{"dsn":"a"} {}`, `[]`, `{"dsn":{"nested":{"x":1,"x":2}}}`} {
		var x struct {
			DSN string `json:"dsn"`
		}
		if decodificarConfiguracionPrivada([]byte(b), &x) == nil {
			t.Fatal("JSON ambiguo aceptado")
		}
	}
	cfg := configuracionPrivadaPrueba(t)
	cfg.ConfianzaJSON = json.RawMessage(`{"raiz":{"clave_id":"a","clave_id":"b"}}`)
	b, _ := json.Marshal(cfg)
	if _, err := cargarConfiguracionPerfilesPrivada(archivoPrivadoPrueba(t, string(b))); err == nil {
		t.Fatal("trust ambiguo aceptado")
	}
}

func TestConfiguracionPrivadaNoAceptaSecretsInlineNiPathsAliased(t *testing.T) {
	for _, caso := range []string{"clave_inline", "alias", "timeout", "ttl", "catalogo", "path", "dir"} {
		t.Run(caso, func(t *testing.T) {
			cfg := configuracionPrivadaPrueba(t)
			switch caso {
			case "clave_inline":
				cfg.ConfianzaJSON = json.RawMessage(`{"entradas_capacidad":[{"material":"privado-no-imprimir"}]}`)
			case "alias":
				cfg.Pools.ActosADMIN = cfg.Pools.FuenteAutorizacion
			case "timeout":
				cfg.TimeoutArranqueSegundos = 61
			case "ttl":
				cfg.VigenciaDecisionSegundos = 0
			case "catalogo":
				cfg.CatalogoMotivosID = "otro.catalogo"
			case "path":
				cfg.Firmante.ClavePrivadaArchivo = "relativo.bin"
			case "dir":
				if err := os.Chmod(cfg.ActivosDirectorio, 0755); err != nil {
					t.Fatal(err)
				}
			}
			b, _ := json.Marshal(cfg)
			_, err := cargarConfiguracionPerfilesPrivada(archivoPrivadoPrueba(t, string(b)))
			if !errors.Is(err, errConfiguracionPrivadaPerfiles) || strings.Contains(err.Error(), "privado-no-imprimir") {
				t.Fatal("config insegura aceptada o filtrada")
			}
		})
	}
}

func TestArchivoPrivadoRechazaPermisosEnlacesGitYTipo(t *testing.T) {
	for _, caso := range []string{"modo_archivo", "modo_padre", "symlink", "symlink_padre", "hardlink", "git", "fifo", "grande"} {
		t.Run(caso, func(t *testing.T) {
			p := archivoPrivadoPrueba(t, `{"dsn":"secreto-no-imprimir"}`)
			dir := filepath.Dir(p)
			switch caso {
			case "modo_archivo":
				if err := os.Chmod(p, 0644); err != nil {
					t.Fatal(err)
				}
			case "modo_padre":
				if err := os.Chmod(dir, 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				link := filepath.Join(dir, "link.json")
				if err := os.Symlink(p, link); err != nil {
					t.Fatal(err)
				}
				p = link
			case "symlink_padre":
				otro := t.TempDir()
				link := filepath.Join(otro, "link")
				if err := os.Symlink(dir, link); err != nil {
					t.Fatal(err)
				}
				p = filepath.Join(link, filepath.Base(p))
			case "hardlink":
				if err := os.Link(p, filepath.Join(dir, "alias.json")); err != nil {
					t.Fatal(err)
				}
			case "git":
				if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: privado"), 0600); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(p, 0600); err != nil {
					t.Fatal(err)
				}
			case "grande":
				if err := os.WriteFile(p, []byte(strings.Repeat("a", limiteConfiguracionPrivadaPerfiles+1)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			b, err := leerArchivoPrivadoPerfiles(p)
			if err == nil || b != nil {
				t.Fatal("fichero inseguro leido")
			}
			if strings.Contains(err.Error(), "secreto-no-imprimir") {
				t.Fatal("secret filtrado")
			}
		})
	}
}

func TestDSNPrivadoNoLeeAmbienteYExigeTLSCadaDestino(t *testing.T) {
	for _, caso := range []struct {
		nombre, dsn string
		valido      bool
	}{
		{"tls_uri", "postgresql://runtime:secreto@db.internal/vec?sslmode=verify-full", true},
		{"multihost_uri", "postgresql://runtime:secreto@db1.internal,db2.internal/vec?sslmode=verify-full", true},
		{"password_missing", "host=db.internal user=runtime dbname=vec sslmode=verify-full", false},
		{"passfile_external", "host=db.internal user=runtime dbname=vec password=secreto sslmode=verify-full passfile=/secreto", false},
		{"tls_keyword", "host=db1.internal,db2.internal port=5432,5433 user=runtime dbname=vec password='secreto con espacios' sslmode=verify-full", true},
		{"unix", "host=/run/postgresql user=runtime dbname=vec password='' sslmode=disable", true},
		{"tls_localhost", "host=localhost user=runtime dbname=vec password=secreto sslmode=verify-full", true},
		{"localhost_weak", "host=localhost user=runtime dbname=vec sslmode=require", false},
		{"fallback_weak", "host=db1.internal,db2.internal user=runtime dbname=vec sslmode=prefer", false},
		{"mixed_weak", "host=/run/postgresql,db.internal user=runtime dbname=vec sslmode=disable", false},
		{"host_missing", "user=runtime dbname=vec sslmode=verify-full", false},
		{"user_missing", "host=db.internal dbname=vec sslmode=verify-full", false},
		{"database_missing", "host=db.internal user=runtime sslmode=verify-full", false},
		{"tls_missing", "host=db.internal user=runtime dbname=vec", false},
		{"duplicada", "host=db.internal host=evil.internal user=runtime dbname=vec sslmode=verify-full", false},
		{"uri_query_override", "postgresql://runtime:secreto@db.internal/vec?sslmode=verify-full&user=otro", false},
		{"uri_tls_duplicate", "postgresql://runtime:secreto@db.internal/vec?sslmode=verify-full&sslmode=disable", false},
		{"service", "service=secreto host=db.internal user=runtime dbname=vec password=secreto sslmode=verify-full", false},
		{"options", "host=db.internal user=runtime dbname=vec password=secreto sslmode=verify-full options='-c role=superuser'", false},
		{"ports", "host=db1.internal,db2.internal,db3.internal port=5432,5433 user=runtime dbname=vec sslmode=verify-full", false},
		{"unterminated", "host=db.internal user=runtime dbname=vec password='secreto sslmode=verify-full", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			b, _ := json.Marshal(struct {
				DSN string `json:"dsn"`
			}{caso.dsn})
			dsn, err := cargarDSNPrivado(archivoPrivadoPrueba(t, string(b)))
			if (err == nil) != caso.valido {
				t.Fatal("resultado inesperado")
			}
			if caso.valido && dsn != desactivarPassfileDSNPerfiles(caso.dsn) {
				t.Fatal("DSN alterado")
			}
			if err != nil && strings.Contains(err.Error(), "secreto") {
				t.Fatal("credencial filtrada")
			}
		})
	}
	t.Setenv("PGPASSWORD", "privado-no-imprimir")
	if validarDSNPerfilesPrivado("host=db.internal user=runtime dbname=vec password=secreto sslmode=verify-full") == nil {
		t.Fatal("password del entorno aceptado")
	}
}

func TestDSNPrivadoPermiteAmbienteNoIdentitario(t *testing.T) {
	t.Setenv("PGAPPNAME", "observabilidad")
	t.Setenv("PGTZ", "UTC")
	if validarDSNPerfilesPrivado("host=db.internal user=runtime dbname=vec password=secreto sslmode=verify-full") != nil {
		t.Fatal("ambiente ajeno a identidad rechazado")
	}
}

func confianzaPrivadaPrueba(t *testing.T, dir string) json.RawMessage {
	t.Helper()
	desde := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	hasta := desde.Add(time.Hour)
	c := metadatosConfianzaPerfilesPrivados{Cabecera: cabeceraConfianzaPerfilesPrivada{FormatoVersion: 3, Suite: "COSE-EdDSA", ClaveID: "raiz:privada", Audiencia: "audiencia:admin"}, Raiz: raizConfianzaPerfilesPrivada{ClaveID: "raiz:privada", Audiencia: "audiencia:admin", Version: 1, PublicaBase64: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", Estado: "activa", ValidaDesde: desde, ValidaHasta: hasta}, Gobierno: gobiernoConfianzaPerfilesPrivada{Revision: "revision:1", HuellaSHA256: strings.Repeat("a", 64), Secuencia: 1, PublicadaEn: desde, ExpiraEn: hasta}, EntradasCapacidad: []capacidadConfianzaPerfilesPrivada{{Audiencia: "audiencia:nominal", ClaveID: "hmac:privada", EmisorID: "emisor:admin", HuellaGobierno: strings.Repeat("a", 64), Version: 1, RevisionGobierno: 1, MaterialArchivo: filepath.Join(dir, "hmac.bin"), Estado: "emision", ValidaDesde: desde, ValidaHasta: hasta}}}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestModoLecturaNoExigeDSNDeActos(t *testing.T) {
	cfg := configuracionPrivadaPrueba(t)
	cfg.Pools.ActosADMIN = ""
	if err := validarConfiguracionPerfilesPrivada(cfg); err != nil {
		t.Fatal("modo lectura exige pool de escritura", err)
	}
}
