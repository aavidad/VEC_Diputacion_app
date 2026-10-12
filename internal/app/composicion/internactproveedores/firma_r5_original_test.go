package internactproveedores

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/documentosbase"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestFirmaR5ExigeInventarioDocumentosCanonico(t *testing.T) {
	contenido := []byte(`{"version":1,"autoridad":"no_autoritativo","dsn_documentos":"host=db.local user=vec_doc_login","almacen":{"tipo":"ficheros","directorio":"/var/lib/vec/documentos","tamano_maximo":1048576,"retencion_minima_dias":0}}`)
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	d, err := ExtraerDocumentoFirmaR5(contenido, huella)
	if err != nil || d.Pool.DSN == "" || d.Almacen.Directorio != "/var/lib/vec/documentos" {
		t.Fatalf("configuración canónica: %v", err)
	}
	alterado := append([]byte(nil), contenido...)
	alterado[len(alterado)-2] = '1'
	if _, err := ExtraerDocumentoFirmaR5(alterado, huella); !errors.Is(err, ErrMaterialFirmaR5NoDisponible) {
		t.Fatalf("inventario documental cambiado: %v", err)
	}
}

func TestMaterialFirmaR5NoExponeDSNEnFormatoNiLog(t *testing.T) {
	const secreto = "clave_privada_solo_prueba"
	d := DocumentoFirmaR5{Pool: PoolMaterial{DSN: "password=" + secreto}, Almacen: documentosbase.ConfiguracionAlmacen{S3: map[string]string{"secret": secreto}}}
	m := MaterialFirmaR5{PlantillasArchivo: secreto}
	for _, v := range []any{d, m} {
		for _, formato := range []string{"%v", "%+v", "%#v"} {
			if strings.Contains(fmt.Sprintf(formato, v), secreto) {
				t.Fatal("se filtró material en fmt")
			}
		}
		codificado, err := json.Marshal(v)
		if err != nil || strings.Contains(string(codificado), secreto) {
			t.Fatal("se filtró material en JSON")
		}
		var salida bytes.Buffer
		rec := slog.NewRecord(time.Now(), slog.LevelInfo, "prueba", 0)
		rec.AddAttrs(slog.Any("material", v))
		if err := slog.NewJSONHandler(&salida, nil).Handle(context.Background(), rec); err != nil || strings.Contains(salida.String(), secreto) {
			t.Fatal("se filtró material en slog")
		}
	}
}

func TestFirmaR5InventarioSeparadoDeniegaClavesDuplicadas(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raiz.Close(); err != nil {
			t.Error(err)
		}
	})
	base := Material{raiz: raiz}
	ruta := filepath.Join(dir, "ct_firma_r5.json")
	huella := strings.Repeat("a", 64)
	contenido := []byte(`{"version":1,"catalogo_motivos":"motivos.ct","motivo_original":{"catalogo_id":"motivos.ct","catalogo_version":1,"catalogo_huella_sha256":"` + huella + `","entrada_clave":"motivo_0123456789abcdef0123456789abcdef"},"capacidad_documentos":{"archivo":"doc.key"},"documentos_archivo":"identidad/documentos.json","documentos_sha256":"` + huella + `","plantillas_archivo":"data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json","plantillas_sha256":"` + huella + `","plantillas_id":"contratacion_temporal.plantillas_borrador","plantillas_version":1,"plantillas_huella_sha256":"` + huella + `","organizacion_archivo":"data/catalogos/estructura-organizativa/v1.rpt-publica.json","organizacion_sha256":"` + huella + `","rpt_archivo":"data/catalogos/rpt/v1.rpt-2026.json","rpt_sha256":"` + huella + `","seudonimos_archivo":"hmac.key","seudonimos_sha256":"` + huella + `"}`)
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialFirmaR5(base); err != nil {
		t.Fatalf("inventario privado válido: %v", err)
	}
	duplicado := append([]byte(`{"version":2,`), contenido[1:]...)
	if err := os.WriteFile(ruta, duplicado, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialFirmaR5(base); !errors.Is(err, ErrMaterialFirmaR5NoDisponible) {
		t.Fatalf("clave duplicada aceptada: %v", err)
	}
}

func TestFirmaR5PreimagenNoAutorizaOtroOriginal(t *testing.T) {
	e := &originalR5Esperado{id: "doc_0123456789abcdef0123456789abcdef", expediente: "exp_0123456789abcdef0123456789abcdef",
		tipo: "tipo_0123456789abcdef0123456789abcdef", clave: "clave_0123456789abcdef0123456789abcdef",
		huella: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", version: 7, tamano: 123}
	pre := []byte(`{"accion":"documentos.original_firmable.reservar","id":"doc_0123456789abcdef0123456789abcdef","clave_idempotencia":"clave_0123456789abcdef0123456789abcdef","modulo_id":"contratacion_temporal","expediente_ref":"exp_0123456789abcdef0123456789abcdef","tipo_ref":"tipo_0123456789abcdef0123456789abcdef","version":7,"mime":"application/pdf","tamano":123,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if !preimagenOriginalR5Valida(docports.AccionReservarOriginalFirmable, pre, e) {
		t.Fatal("la reserva propia fue rechazada")
	}
	mutada := bytes.Replace(pre, []byte(`"version":7`), []byte(`"version":8`), 1)
	if preimagenOriginalR5Valida(docports.AccionReservarOriginalFirmable, mutada, e) {
		t.Fatal("una versión ajena quedó autorizada")
	}
}
