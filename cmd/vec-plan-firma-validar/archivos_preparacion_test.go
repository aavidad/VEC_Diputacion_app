package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPrepararNoSobrescribeFicheroDirectorioNiEnlace(t *testing.T) {
	for _, tipo := range []string{"fichero", "directorio", "enlace"} {
		t.Run(tipo, func(t *testing.T) {
			args, _ := entradasPreparacionPrueba(t)
			original := []byte("contenido conservado")
			destino := args[5]
			switch tipo {
			case "fichero":
				if err := os.WriteFile(destino, original, 0600); err != nil {
					t.Fatal(err)
				}
			case "directorio":
				if err := os.Mkdir(destino, 0700); err != nil {
					t.Fatal(err)
				}
			case "enlace":
				destino += ".original"
				if err := os.WriteFile(destino, original, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(destino, args[5]); err != nil {
					t.Fatal(err)
				}
			}
			var salida bytes.Buffer
			if codigo := ejecutar(args, &salida); codigo != 1 || salida.Len() != 0 {
				t.Fatal("destino existente aceptado")
			}
			if tipo != "directorio" {
				b, err := os.ReadFile(destino)
				if err != nil || !bytes.Equal(original, b) {
					t.Fatal("contenido previo alterado")
				}
			}
			if _, err := os.Lstat(args[5]); err != nil {
				t.Fatal("destino previo eliminado")
			}
		})
	}
}

func TestPrepararExigeEntradaRegularYCarpetaPrivada(t *testing.T) {
	for _, tipo := range []string{"enlace", "directorio", "fifo", "carpeta_publica"} {
		t.Run(tipo, func(t *testing.T) {
			args, _ := entradasPreparacionPrueba(t)
			if tipo == "carpeta_publica" {
				if err := os.Chmod(filepath.Dir(args[5]), 0755); err != nil {
					t.Fatal(err)
				}
			} else {
				ruta := args[2] + ".no_regular"
				switch tipo {
				case "enlace":
					if err := os.Symlink(args[2], ruta); err != nil {
						t.Fatal(err)
					}
				case "directorio":
					if err := os.Mkdir(ruta, 0700); err != nil {
						t.Fatal(err)
					}
				case "fifo":
					if err := syscall.Mkfifo(ruta, 0600); err != nil {
						t.Fatal(err)
					}
				}
				args[2] = ruta
			}
			var salida bytes.Buffer
			if codigo := ejecutar(args, &salida); codigo != 1 || salida.Len() != 0 {
				t.Fatal("entrada o carpeta rechazada produjo salida")
			}
			if _, err := os.Lstat(args[5]); !os.IsNotExist(err) {
				t.Fatal("se conservó una salida inválida")
			}
		})
	}
}

type salidaPreparacionFallida struct{}

func (salidaPreparacionFallida) Write([]byte) (int, error) {
	return 0, errors.New("salida_prueba_rechazada")
}

func TestPrepararLimpiaSalidaSiNoPuedeInformarDelResultado(t *testing.T) {
	args, _ := entradasPreparacionPrueba(t)
	if codigo := ejecutar(args, salidaPreparacionFallida{}); codigo != 1 {
		t.Fatal("error de salida ignorado")
	}
	if _, err := os.Lstat(args[5]); !os.IsNotExist(err) {
		t.Fatal("salida parcial conservada")
	}
	var salida bytes.Buffer
	if codigo := ejecutar(args, &salida); codigo != 0 {
		t.Fatal("el mismo destino no se recuperó tras limpiar el fallo")
	}
}
