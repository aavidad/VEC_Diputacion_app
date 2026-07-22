package interna

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestAbrirFicheroTLSFinalRechazaPropietarioRuntime(t *testing.T) {
	directorio := t.TempDir()
	ruta := filepath.Join(directorio, "clave.pem")
	if err := os.WriteFile(ruta, []byte("material"), 0o400); err != nil {
		t.Fatal(err)
	}
	fd := abrirDirectorioPrueba(t, directorio)
	defer syscall.Close(fd)
	if _, err := abrirFicheroTLSFinal(fd, filepath.Base(ruta), true); !errors.Is(err, ErrTLSMutuoNoVerificado) {
		t.Fatalf("fichero del runtime aceptado: %v", err)
	}
}

func TestAbrirFicheroTLSFinalNoBloqueaEnFIFO(t *testing.T) {
	directorio := t.TempDir()
	nombre := "material.fifo"
	if err := syscall.Mkfifo(filepath.Join(directorio, nombre), 0o400); err != nil {
		t.Fatal(err)
	}
	fd := abrirDirectorioPrueba(t, directorio)
	defer syscall.Close(fd)
	terminado := make(chan error, 1)
	go func() {
		_, err := abrirFicheroTLSFinal(fd, nombre, true)
		terminado <- err
	}()
	select {
	case err := <-terminado:
		if !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("FIFO = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("apertura FIFO quedo bloqueada")
	}
}

func TestAbrirFicheroTLSFinalRechazaDispositivoSinBloquear(t *testing.T) {
	fd := abrirDirectorioPrueba(t, "/dev")
	defer syscall.Close(fd)
	inicio := time.Now()
	if _, err := abrirFicheroTLSFinal(fd, "null", false); !errors.Is(err, ErrTLSMutuoNoVerificado) {
		t.Fatalf("dispositivo = %v", err)
	}
	if duracion := time.Since(inicio); duracion > 500*time.Millisecond {
		t.Fatalf("rechazo dispositivo tardo %s", duracion)
	}
}

func abrirDirectorioPrueba(t *testing.T, ruta string) int {
	t.Helper()
	fd, err := syscall.Open(ruta, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	return fd
}
