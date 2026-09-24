package interna

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCargarMaterialPoolsSeguimientoContratoPrivado(t *testing.T) {
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(directorio, "pools.json")
	const dsn = "postgresql://identidad:secreto@db.interno/vec?sslmode=verify-full"
	claves := []string{"alta_personal", "registro_ct", "inicial_ct", "localizador_ct", "localizador_personal", "lectura_personal",
		"historia_registro_ct", "historia_autenticacion", "historia_contexto", "historia_evaluacion", "historia_concesion"}
	var entradas []string
	for i, clave := range claves {
		entradas = append(entradas, `"`+clave+`":{"login":"login_`+string(rune('a'+i))+`","dsn":"`+dsn+`"}`)
	}
	valido := `{"version":1,"pools":{` + strings.Join(entradas, ",") + `}}`
	escribir := func(contenido string, modo os.FileMode) {
		t.Helper()
		if err := os.WriteFile(ruta, []byte(contenido), modo); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ruta, modo); err != nil {
			t.Fatal(err)
		}
	}
	escribir(valido, 0600)
	material, err := cargarMaterialPoolsSeguimiento(directorio)
	if err != nil || material.AltaPersonal.Login != "login_a" || material.HistoriaConcesion.Login != "login_k" {
		t.Fatalf("material valido rechazado: %v", err)
	}
	for nombre, contenido := range map[string]string{
		"duplicado":   `{"version":1,"version":1,"pools":{` + strings.Join(entradas, ",") + `}}`,
		"desconocido": `{"version":1,"pools":{` + strings.Join(entradas, ",") + `},"otro":1}`,
		"faltante":    `{"version":1,"pools":{` + strings.Join(entradas[:10], ",") + `}}`,
		"version":     strings.Replace(valido, `"version":1`, `"version":2`, 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			escribir(contenido, 0600)
			_, err := cargarMaterialPoolsSeguimiento(directorio)
			if !errors.Is(err, ErrMaterialSeguimientoNoDisponible) || strings.Contains(err.Error(), dsn) {
				t.Fatalf("material rechazado sin detalles privados: %v", err)
			}
		})
	}
	escribir(valido, 0644)
	if _, err := cargarMaterialPoolsSeguimiento(directorio); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
		t.Fatalf("permisos permisivos admitidos: %v", err)
	}
	escribir(valido, 0600)
	enlace := filepath.Join(directorio, "pools_enlace.json")
	if err := os.Rename(ruta, enlace); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(enlace, ruta); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarMaterialPoolsSeguimiento(directorio); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
		t.Fatalf("enlace simbolico admitido: %v", err)
	}
	if _, err := cargarMaterialPoolsSeguimiento("directorio-relativo"); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
		t.Fatalf("directorio relativo admitido: %v", err)
	}
}

func TestLectoresIdentidadPrivadaRechazanAmbiguedad(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	identidad := filepath.Join(base, "identidad")
	if err := os.Mkdir(identidad, 0700); err != nil {
		t.Fatal(err)
	}
	contextos := filepath.Join(identidad, "contextos.json")
	valido := `{"version":1,"contextos":[{"cuenta_ref":"cta_12345678901234567890","perfil_ref":"prf_12345678901234567890","organizacion_ref":"org_12345678901234567890","unidad_ref":"uni_12345678901234567890"}]}`
	if err := os.WriteFile(contextos, []byte(valido), 0600); err != nil {
		t.Fatal(err)
	}
	mapa, err := cargarContextosNominalesIdentidad(base)
	if err != nil || len(mapa) != 1 {
		t.Fatalf("contexto privado válido rechazado: %v", err)
	}
	duplicado := strings.Replace(valido, `}]}`, `},{"cuenta_ref":"cta_12345678901234567890","perfil_ref":"otro","organizacion_ref":"org","unidad_ref":"uni"}]}`, 1)
	if err := os.WriteFile(contextos, []byte(duplicado), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarContextosNominalesIdentidad(base); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
		t.Fatalf("cuenta duplicada admitida: %v", err)
	}
	if err := os.WriteFile(contextos, []byte(valido), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(identidad, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarContextosNominalesIdentidad(base); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
		t.Fatalf("directorio privado permisivo admitido: %v", err)
	}
}

func TestCargarConfiguracionHMACIdentidadSoloMetadatos(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	identidad := filepath.Join(base, "identidad")
	if err := os.Mkdir(identidad, 0700); err != nil {
		t.Fatal(err)
	}
	pin := filepath.Join(identidad, "hmac.pin")
	if err := os.WriteFile(pin, []byte("pinprivado"), 0600); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(identidad, "hmac.json")
	contenido := `{"version":1,"modulo":"/usr/lib/softhsm/libsofthsm2.so","token_label":"vec_desarrollo","token_serial":"serial123",` +
		`"objeto_id_hex":"aabbcc","clave_id":"key_123","clave_version":1,"dominio_ref":"idh_123456789012345678901234",` +
		`"espacio_identidad":"https://vec.cidonia.cloud/identidad","pin_fichero":"` + pin + `"}`
	if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
		t.Fatal(err)
	}
	config, err := cargarConfiguracionHMACIdentidad(base)
	if err != nil || config.PINFichero != pin || len(config.ObjetoID) != 3 {
		t.Fatalf("metadatos HMAC validos rechazados: %v", err)
	}
	if err := os.Chmod(pin, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarConfiguracionHMACIdentidad(base); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
		t.Fatalf("PIN con permisos abiertos admitido: %v", err)
	}
}

func TestConfigurarPoolIdentidadInternaExigeTLSVerificadoYLogin(t *testing.T) {
	for nombre, entrada := range map[string]entradaPoolSeguimiento{
		"sin TLS":        {Login: "vec_registro", DSN: "postgresql://vec_registro:clave@db.interno/vec?sslmode=disable"},
		"login distinto": {Login: "vec_otro", DSN: "postgresql://vec_registro:clave@db.interno/vec?sslmode=verify-full"},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := configurarPoolIdentidadInterna(entrada, "vec_identidad_sesiones_v1_registrador", "vec_identidad_sesiones_v1.registrar_sesion_v1(text)", false); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
				t.Fatalf("pool inseguro admitido: %v", err)
			}
		})
	}
}
