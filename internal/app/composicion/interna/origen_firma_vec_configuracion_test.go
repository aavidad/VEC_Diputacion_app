package interna

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func directorioPrivadoFirmaVecPrueba(t *testing.T) string {
	t.Helper()
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	return directorio
}

func TestCargarOrigenFirmaVecPrivadoAceptaSoloOrigenTLSGobernado(t *testing.T) {
	for _, caso := range []struct{ nombre, origen, nombreTLS, final string }{
		{"sin_puerto", "https://firma.example.invalid", "firma.example.invalid", "\n"},
		{"puerto_no_estandar", "https://firma.example.invalid:8443", "firma.example.invalid", ""},
		{"ip_tls_configurada", "https://127.0.0.1:8443", "127.0.0.1", "\n"},
		{"ipv6_tls_configurada", "https://[::1]", "::1", "\n"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if !origenFirmaVecCanonico(caso.origen, caso.nombreTLS) {
				t.Fatal("el parser rechazo un origen TLS valido")
			}
			directorio := directorioPrivadoFirmaVecPrueba(t)
			ruta := filepath.Join(directorio, nombreArchivoOrigenFirmaVec)
			if err := os.WriteFile(ruta, []byte(prefijoOrigenFirmaVec+caso.origen+caso.final), 0o600); err != nil {
				t.Fatal(err)
			}
			obtenido, err := cargarOrigenFirmaVecPrivado(directorio, caso.nombreTLS)
			if err != nil || obtenido != caso.origen {
				t.Fatalf("origen privado obtenido=%q, err=%v", obtenido, err)
			}
		})
	}
}

func TestCargarOrigenFirmaVecPrivadoRechazaValorAjenoYNoLoExpone(t *testing.T) {
	for _, caso := range []struct{ nombre, origen string }{
		{"http", "http://firma.example.invalid"},
		{"credenciales", "https://usuario:secreto@firma.example.invalid"},
		{"ruta", "https://firma.example.invalid/registro"},
		{"consulta", "https://firma.example.invalid?token=secreto"},
		{"fragmento", "https://firma.example.invalid#parte"},
		{"host_ajeno", "https://otro.example.invalid"},
		{"ip", "https://127.0.0.1"},
		{"ipv6", "https://[::1]"},
		{"mayusculas", "https://Firma.example.invalid"},
		{"puerto_cero", "https://firma.example.invalid:0"},
		{"puerto_excesivo", "https://firma.example.invalid:65536"},
		{"puerto_no_canonico", "https://firma.example.invalid:08443"},
		{"puerto_por_defecto", "https://firma.example.invalid:443"},
		{"puerto_vacio", "https://firma.example.invalid:"},
		{"dns_punto_final", "https://firma.example.invalid."},
		{"lineas_duplicadas", "https://firma.example.invalid\norigen_firma_vec=https://otro.example.invalid"},
		{"clave_ajena", "https://firma.example.invalid\npolitica=libre"},
		{"espacio", " https://firma.example.invalid"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			directorio := directorioPrivadoFirmaVecPrueba(t)
			ruta := filepath.Join(directorio, nombreArchivoOrigenFirmaVec)
			if err := os.WriteFile(ruta, []byte(prefijoOrigenFirmaVec+caso.origen+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			obtenido, err := cargarOrigenFirmaVecPrivado(directorio, "firma.example.invalid")
			if obtenido != "" || !errors.Is(err, ErrOrigenFirmaVecNoDisponible) ||
				strings.Contains(err.Error(), directorio) || strings.Contains(err.Error(), caso.origen) {
				t.Fatalf("valor ajeno aceptado o expuesto: obtenido=%q, err=%v", obtenido, err)
			}
		})
	}
}

func TestCargarOrigenFirmaVecPrivadoRechazaMaterialNoPrivado(t *testing.T) {
	contenido := []byte(prefijoOrigenFirmaVec + "https://firma.example.invalid\n")
	for _, caso := range []struct {
		nombre   string
		preparar func(*testing.T, string)
	}{
		{"ausente", func(*testing.T, string) {}},
		{"archivo_0644", func(t *testing.T, d string) {
			ruta := filepath.Join(d, nombreArchivoOrigenFirmaVec)
			if err := os.WriteFile(ruta, contenido, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(ruta, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"enlace", func(t *testing.T, d string) {
			objetivo := filepath.Join(d, "objetivo")
			if err := os.WriteFile(objetivo, contenido, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(objetivo, filepath.Join(d, nombreArchivoOrigenFirmaVec)); err != nil {
				t.Fatal(err)
			}
		}},
		{"directorio_0755", func(t *testing.T, d string) {
			if err := os.WriteFile(filepath.Join(d, nombreArchivoOrigenFirmaVec), contenido, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"archivo_excesivo", func(t *testing.T, d string) {
			if err := os.WriteFile(filepath.Join(d, nombreArchivoOrigenFirmaVec),
				[]byte(prefijoOrigenFirmaVec+strings.Repeat("a", limiteArchivoOrigenFirmaVec)), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			directorio := directorioPrivadoFirmaVecPrueba(t)
			caso.preparar(t, directorio)
			if origen, err := cargarOrigenFirmaVecPrivado(directorio, "firma.example.invalid"); origen != "" || !errors.Is(err, ErrOrigenFirmaVecNoDisponible) {
				t.Fatalf("material no privado aceptado: origen=%q, err=%v", origen, err)
			}
		})
	}
}
