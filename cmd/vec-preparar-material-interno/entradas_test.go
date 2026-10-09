package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLeerDSNRechazaFicherosBajoGitAntesDeLeer(t *testing.T) {
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	privado := filepath.Join(raiz, "privado")
	if err := os.Mkdir(privado, 0700); err != nil {
		t.Fatal(err)
	}
	for _, nombre := range []string{"gobierno.dsn", "motivos_rrhh.dsn"} {
		ruta := filepath.Join(privado, nombre)
		if err := os.WriteFile(ruta, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if dsn, err := leerDSN(ruta, ""); dsn != "" || !errors.Is(err, errDSNFichero) {
			t.Fatalf("%s: DSN bajo Git admitido", nombre)
		}
	}
}

func TestResolutorRRHHRechazaLoginDistintoSinConectar(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "motivos_rrhh.dsn")
	if err := os.WriteFile(ruta, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := resolverMotivoDetalleCTServidorConLogin(context.Background(), ruta, "otro_login", time.Now())
	if !errors.Is(err, errMotivosRRHHDSN) {
		t.Fatal("LOGIN distinto no rechazado antes de conectar")
	}
}
