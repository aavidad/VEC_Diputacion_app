package interna

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	vecapp "vec-diputacion-granada/internal/vec/application"
)

type relojPoliticaFirmaVecPrueba struct{ ahora time.Time }

func (r relojPoliticaFirmaVecPrueba) Ahora() time.Time { return r.ahora }

func documentoPoliticaFirmaVecPrueba(ahora time.Time) documentoPoliticaFirmaVecDesarrollo {
	return documentoPoliticaFirmaVecDesarrollo{
		Esquema: esquemaPoliticaFirmaVecDesarrollo, Entorno: vecapp.EntornoAdmisionFirmaDesarrollo,
		VigenteDesde: ahora.Add(-time.Hour).Format(time.RFC3339),
		Referencia:   "politica:vec:firma:desarrollo-un-factor:v1", Version: 1,
		HuellaSHA256: strings.Repeat("a", 64), RetiradaEn: ahora.Add(time.Hour).Format(time.RFC3339),
		PoliticaAutenticacionRef: "pga_0123456789abcdef", PoliticaAutenticacionSHA256: strings.Repeat("b", 64),
		RolVersionRef: "rol:vec:firma:dev:v1", RolSHA256: strings.Repeat("c", 64),
		ControlRevision: 1, ControlSHA256: strings.Repeat("d", 64),
	}
}

func directorioPoliticaFirmaVecPrueba(t *testing.T, contenido []byte) string {
	t.Helper()
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	if contenido != nil {
		ruta := filepath.Join(directorio, nombrePoliticaFirmaVecDesarrollo)
		if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ruta, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return directorio
}

func pinPoliticaFirmaVecPrueba(contenido []byte) string {
	huella := sha256.Sum256(contenido)
	return "sha256:" + hex.EncodeToString(huella[:])
}

func TestCargarPoliticaFirmaVecDesarrolloPrivadaExigePinYConservaCampos(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	documento := documentoPoliticaFirmaVecPrueba(ahora)
	contenido, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	directorio := directorioPoliticaFirmaVecPrueba(t, contenido)
	politica, err := cargarPoliticaFirmaVecDesarrolloPrivada(directorio,
		vecapp.EntornoAdmisionFirmaDesarrollo, pinPoliticaFirmaVecPrueba(contenido), relojPoliticaFirmaVecPrueba{ahora})
	if err != nil || politica.Referencia != documento.Referencia || politica.Version != documento.Version ||
		politica.HuellaSHA256 != documento.HuellaSHA256 || politica.RolVersionRef != documento.RolVersionRef ||
		politica.RolSHA256 != documento.RolSHA256 || politica.ControlRevision != documento.ControlRevision ||
		politica.ControlSHA256 != documento.ControlSHA256 ||
		politica.PoliticaAutenticacionRef != documento.PoliticaAutenticacionRef ||
		politica.PoliticaAutenticacionSHA256 != documento.PoliticaAutenticacionSHA256 ||
		politica.RetiradaEn.Format(time.RFC3339) != documento.RetiradaEn {
		t.Fatalf("politica privada no conservada: err=%v", err)
	}
	alterado := append(append([]byte(nil), contenido...), ' ')
	if err := os.WriteFile(filepath.Join(directorio, nombrePoliticaFirmaVecDesarrollo), alterado, 0o600); err != nil {
		t.Fatal(err)
	}
	if p, err := cargarPoliticaFirmaVecDesarrolloPrivada(directorio,
		vecapp.EntornoAdmisionFirmaDesarrollo, pinPoliticaFirmaVecPrueba(contenido), relojPoliticaFirmaVecPrueba{ahora}); p.Referencia != "" || !errors.Is(err, ErrPoliticaFirmaVecDesarrolloNoDisponible) ||
		strings.Contains(err.Error(), directorio) || strings.Contains(err.Error(), documento.Referencia) {
		t.Fatalf("archivo alterado aceptado: err=%v", err)
	}
}

func TestCargarPoliticaFirmaVecDesarrolloPrivadaRechazaContratoYVentana(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	base := documentoPoliticaFirmaVecPrueba(ahora)
	for _, caso := range []struct {
		nombre  string
		mutar   func(*documentoPoliticaFirmaVecDesarrollo)
		entorno string
	}{
		{"produccion", nil, "produccion"},
		{"entorno_archivo_ajeno", func(d *documentoPoliticaFirmaVecDesarrollo) { d.Entorno = "presentacion" }, vecapp.EntornoAdmisionFirmaDesarrollo},
		{"inicio_futuro", func(d *documentoPoliticaFirmaVecDesarrollo) {
			d.VigenteDesde = ahora.Add(time.Minute).Format(time.RFC3339)
		}, vecapp.EntornoAdmisionFirmaDesarrollo},
		{"retirada_vencida", func(d *documentoPoliticaFirmaVecDesarrollo) { d.RetiradaEn = ahora.Format(time.RFC3339) }, vecapp.EntornoAdmisionFirmaDesarrollo},
		{"retirada_no_utc", func(d *documentoPoliticaFirmaVecDesarrollo) { d.RetiradaEn = "2026-10-08T14:00:00+01:00" }, vecapp.EntornoAdmisionFirmaDesarrollo},
		{"referencia_otro_version", func(d *documentoPoliticaFirmaVecDesarrollo) { d.Version = 2 }, vecapp.EntornoAdmisionFirmaDesarrollo},
		{"rol_sin_huella", func(d *documentoPoliticaFirmaVecDesarrollo) { d.RolSHA256 = "" }, vecapp.EntornoAdmisionFirmaDesarrollo},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			documento := base
			if caso.mutar != nil {
				caso.mutar(&documento)
			}
			contenido, err := json.Marshal(documento)
			if err != nil {
				t.Fatal(err)
			}
			directorio := directorioPoliticaFirmaVecPrueba(t, contenido)
			if p, err := cargarPoliticaFirmaVecDesarrolloPrivada(directorio,
				caso.entorno, pinPoliticaFirmaVecPrueba(contenido), relojPoliticaFirmaVecPrueba{ahora}); p.Referencia != "" || !errors.Is(err, ErrPoliticaFirmaVecDesarrolloNoDisponible) {
				t.Fatalf("contrato ajeno aceptado: err=%v", err)
			}
		})
	}
}

func TestCargarPoliticaFirmaVecDesarrolloPrivadaRechazaJSONYMaterialInseguro(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	base, err := json.Marshal(documentoPoliticaFirmaVecPrueba(ahora))
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre    string
		contenido []byte
		preparar  func(*testing.T, string)
	}{
		{"clave_duplicada", append(append([]byte(nil), base[:len(base)-1]...), []byte(`,"version":2}`)...), nil},
		{"clave_desconocida", append(append([]byte(nil), base[:len(base)-1]...), []byte(`,"permiso_libre":true}`)...), nil},
		{"dos_documentos", append(append([]byte(nil), base...), base...), nil},
		{"ausente", nil, nil},
		{"archivo_0644", base, func(t *testing.T, d string) {
			if err := os.Chmod(filepath.Join(d, nombrePoliticaFirmaVecDesarrollo), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"enlace", nil, func(t *testing.T, d string) {
			objetivo := filepath.Join(d, "objetivo.json")
			if err := os.WriteFile(objetivo, base, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(objetivo, filepath.Join(d, nombrePoliticaFirmaVecDesarrollo)); err != nil {
				t.Fatal(err)
			}
		}},
		{"directorio_0755", base, func(t *testing.T, d string) {
			if err := os.Chmod(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"demasiado_grande", []byte(strings.Repeat("x", limitePoliticaFirmaVecDesarrollo+1)), nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			directorio := directorioPoliticaFirmaVecPrueba(t, caso.contenido)
			if caso.preparar != nil {
				caso.preparar(t, directorio)
			}
			pin := pinPoliticaFirmaVecPrueba(caso.contenido)
			if caso.contenido == nil {
				pin = pinPoliticaFirmaVecPrueba(base)
			}
			if p, err := cargarPoliticaFirmaVecDesarrolloPrivada(directorio,
				vecapp.EntornoAdmisionFirmaDesarrollo, pin, relojPoliticaFirmaVecPrueba{ahora}); p.Referencia != "" || !errors.Is(err, ErrPoliticaFirmaVecDesarrolloNoDisponible) {
				t.Fatalf("material inseguro aceptado: err=%v", err)
			}
		})
	}
}

func TestCargarPoliticaFirmaVecDesarrolloPrivadaConservaCausasSinExponerMaterial(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	reloj := relojPoliticaFirmaVecPrueba{ahora}
	base, err := json.Marshal(documentoPoliticaFirmaVecPrueba(ahora))
	if err != nil {
		t.Fatal(err)
	}
	comprobarOpaco := func(err error, codigo string, privado string) {
		t.Helper()
		var detalle *ErrorCargaPoliticaFirmaVecDesarrollo
		if !errors.Is(err, ErrPoliticaFirmaVecDesarrolloNoDisponible) || !errors.As(err, &detalle) ||
			detalle.Codigo() != codigo || err.Error() != ErrPoliticaFirmaVecDesarrolloNoDisponible.Error() ||
			strings.Contains(err.Error(), privado) {
			t.Fatalf("error sin causa cerrada/opacidad: codigo=%q, err=%v", codigo, err)
		}
	}

	directorioAusente := directorioPoliticaFirmaVecPrueba(t, nil)
	_, err = cargarPoliticaFirmaVecDesarrolloPrivada(directorioAusente,
		vecapp.EntornoAdmisionFirmaDesarrollo, pinPoliticaFirmaVecPrueba(base), reloj)
	comprobarOpaco(err, string(codigoPoliticaArchivo), directorioAusente)
	var ruta *os.PathError
	if !errors.Is(err, os.ErrNotExist) || !errors.As(err, &ruta) {
		t.Fatalf("Lstat no conserva causa IO: %v", err)
	}

	malformado := []byte(`{"esquema":?}`)
	directorioJSON := directorioPoliticaFirmaVecPrueba(t, malformado)
	_, err = cargarPoliticaFirmaVecDesarrolloPrivada(directorioJSON,
		vecapp.EntornoAdmisionFirmaDesarrollo, pinPoliticaFirmaVecPrueba(malformado), reloj)
	comprobarOpaco(err, string(codigoPoliticaJSON), directorioJSON)
	var sintaxis *json.SyntaxError
	if !errors.As(err, &sintaxis) {
		t.Fatalf("decodificacion no conserva causa JSON: %v", err)
	}

	documento := documentoPoliticaFirmaVecPrueba(ahora)
	documento.RolSHA256 = ""
	contenido, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	directorioConstructor := directorioPoliticaFirmaVecPrueba(t, contenido)
	_, err = cargarPoliticaFirmaVecDesarrolloPrivada(directorioConstructor,
		vecapp.EntornoAdmisionFirmaDesarrollo, pinPoliticaFirmaVecPrueba(contenido), reloj)
	comprobarOpaco(err, string(codigoPoliticaConstructor), directorioConstructor)
	if !errors.Is(err, vecapp.ErrAdmisionGarantiaActoDenegada) {
		t.Fatalf("constructor no conserva causa de admision: %v", err)
	}
}
