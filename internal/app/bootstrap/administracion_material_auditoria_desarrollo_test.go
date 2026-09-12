package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func configuracionMaterialAuditoriaAdministracionPrueba(t *testing.T) (config.Config, string, []byte) {
	t.Helper()
	directorio := t.TempDir()
	administracion := filepath.Join(directorio, "administracion")
	if err := os.Mkdir(administracion, 0700); err != nil {
		t.Fatal(err)
	}
	clave := bytes.Repeat([]byte{0x7a}, sha256.Size)
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DevelopmentMaterialDir: directorio}
	return cfg, filepath.Join(administracion, "auditoria-hmac.json"), clave
}

func escribirMaterialAuditoriaAdministracionPrueba(t *testing.T, ruta string, clave []byte) []byte {
	t.Helper()
	contenido, err := json.Marshal(archivoMaterialAuditoriaAdministracionDesarrollo{
		Esquema: esquemaMaterialAuditoriaAdministracionDesarrolloV1, ClaveRef: "administracion_auditoria_t13_v1", ClaveBase64: base64.StdEncoding.EncodeToString(clave),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	return contenido
}

func TestSeudonimizadorAuditoriaAdministracionCargaClaveT13Exclusiva(t *testing.T) {
	cfg, ruta, clave := configuracionMaterialAuditoriaAdministracionPrueba(t)
	escribirMaterialAuditoriaAdministracionPrueba(t, ruta, clave)
	seudonimizador, err := nuevoSeudonimizadorAuditoriaAdministracionDesarrollo(cfg)
	if err != nil || seudonimizador == nil {
		t.Fatalf("carga de material T13: %v", err)
	}
	solicitud, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen("principal_sintetico", "bolsa_registro_accesos_t13")
	if err != nil {
		t.Fatal(err)
	}
	seudonimo, err := seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), solicitud)
	if err != nil || !strings.HasPrefix(seudonimo, "hmac-sha256:administracion_auditoria_t13_v1:") || len(seudonimo) != len("hmac-sha256:administracion_auditoria_t13_v1:")+sha256.Size*2 {
		t.Fatal("el sellador no conserva el dominio exclusivo T13")
	}
}

func TestSeudonimizadorAuditoriaAdministracionFallaCerrado(t *testing.T) {
	for _, nombre := range []string{"ausente", "json_parcial", "campo_ajeno", "clave_corta", "clave_cero", "ref_ajena", "permisos", "enlace"} {
		t.Run(nombre, func(t *testing.T) {
			cfg, ruta, clave := configuracionMaterialAuditoriaAdministracionPrueba(t)
			contenido := escribirMaterialAuditoriaAdministracionPrueba(t, ruta, clave)
			switch nombre {
			case "ausente":
				if err := os.Remove(ruta); err != nil {
					t.Fatal(err)
				}
			case "json_parcial":
				if err := os.WriteFile(ruta, []byte(`{"esquema":"`+esquemaMaterialAuditoriaAdministracionDesarrolloV1+`"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "campo_ajeno":
				if err := os.WriteFile(ruta, append([]byte(`{"ajeno":true,`), contenido[1:]...), 0600); err != nil {
					t.Fatal(err)
				}
			case "clave_corta":
				escribirMaterialAuditoriaAdministracionPrueba(t, ruta, clave[:sha256.Size-1])
			case "clave_cero":
				escribirMaterialAuditoriaAdministracionPrueba(t, ruta, make([]byte, sha256.Size))
			case "ref_ajena":
				if err := os.WriteFile(ruta, []byte(`{"esquema":"`+esquemaMaterialAuditoriaAdministracionDesarrolloV1+`","clave_ref":"idempotencia_v1","clave_base64":"`+base64.StdEncoding.EncodeToString(clave)+`"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "permisos":
				if err := os.Chmod(ruta, 0644); err != nil {
					t.Fatal(err)
				}
			case "enlace":
				objetivo := filepath.Join(filepath.Dir(ruta), "material-real.json")
				if err := os.WriteFile(objetivo, contenido, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(ruta); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(objetivo, ruta); err != nil {
					t.Fatal(err)
				}
			}
			if valor, err := nuevoSeudonimizadorAuditoriaAdministracionDesarrollo(cfg); err == nil || valor != nil {
				t.Fatal("material parcial o inseguro compone un seudonimizador")
			}
		})
	}
}

func TestSeudonimizadorAuditoriaAdministracionRequierePerfilDesarrolloCompleto(t *testing.T) {
	cfg, ruta, clave := configuracionMaterialAuditoriaAdministracionPrueba(t)
	escribirMaterialAuditoriaAdministracionPrueba(t, ruta, clave)
	cfg.DevelopmentGuard = ""
	if valor, err := nuevoSeudonimizadorAuditoriaAdministracionDesarrollo(cfg); err == nil || valor != nil {
		t.Fatal("un perfil parcial compone una clave de auditoría")
	}
}
