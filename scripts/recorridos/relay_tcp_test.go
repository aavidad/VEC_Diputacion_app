package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPuertoSoloDecimalNoPrivilegiado(t *testing.T) {
	for _, valor := range []string{"1024", "8443", "65535"} {
		if _, err := validarPuerto(valor); err != nil {
			t.Errorf("puerto %q: %v", valor, err)
		}
	}
	for _, valor := range []string{"", "0", "443", "1023", "65536", "-1", "+8443", "08443", " 8443", "8443\n", "127.0.0.1:8443", "example.invalid", "--sha256"} {
		if _, err := validarPuerto(valor); err == nil {
			t.Errorf("puerto aceptado: %q", valor)
		}
	}
	for _, args := range [][]string{nil, {"8443", "extra"}, {"--sha256", "8443"}, {"--help"}, {"443"}} {
		if codigo := ejecutar(args); codigo != 2 {
			t.Errorf("argumentos %q: código %d", args, codigo)
		}
	}
}

func parejaTCP(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	escucha, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = escucha.Close() })
	cliente, err := net.DialTCP("tcp4", nil, escucha.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cliente.Close() })
	servidor, err := escucha.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = servidor.Close() })
	for _, tcp := range []*net.TCPConn{cliente, servidor} {
		if err := tcp.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	return cliente, servidor
}

func esperarRelay(t *testing.T, resultado <-chan error) error {
	t.Helper()
	select {
	case err := <-resultado:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("el relay no terminó al cerrar o cancelar")
		return nil
	}
}

func iniciarRelay(t *testing.T, ctx context.Context, cliente *net.TCPConn, entrada io.ReadCloser, salida io.WriteCloser, limite int64) <-chan error {
	t.Helper()
	resultado := make(chan error, 1)
	go func() { resultado <- transportar(ctx, cliente, entrada, salida, limite) }()
	return resultado
}

type salidaBuffer struct{ bytes.Buffer }

func (*salidaBuffer) Close() error { return nil }

func TestBytesOpacosYRespuestaTrasEOFEntrada(t *testing.T) {
	cliente, servidor := parejaTCP(t)
	// Incluye cabecera semejante a TLS, NUL, UTF-8 inválido y saltos de línea.
	peticion := bytes.Repeat([]byte{0x16, 0x03, 0x03, 0x00, 0xff, '\n'}, 1024)
	respuesta := bytes.Repeat([]byte{0x17, 0x03, 0x03, 0xfe, 0x00}, 2048)
	salida := &salidaBuffer{}
	resultado := iniciarRelay(t, context.Background(), cliente, io.NopCloser(bytes.NewReader(peticion)), salida, maxBytesSentido)
	recibido, err := io.ReadAll(servidor)
	if err != nil || !bytes.Equal(recibido, peticion) {
		t.Fatalf("petición alterada: len=%d, err=%v", len(recibido), err)
	}
	if _, err := servidor.Write(respuesta); err != nil {
		t.Fatal(err)
	}
	if err := servidor.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	if err := esperarRelay(t, resultado); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(salida.Bytes(), respuesta) {
		t.Fatal("respuesta alterada o truncada después de EOF de stdin")
	}
}

func TestCopiaBidireccionalSinEsperarEOFEntrada(t *testing.T) {
	cliente, servidor := parejaTCP(t)
	entrada, escribirEntrada := io.Pipe()
	leerSalida, salida := io.Pipe()
	t.Cleanup(func() { _ = escribirEntrada.Close(); _ = leerSalida.Close() })
	ctx, cancelar := context.WithTimeout(context.Background(), time.Second)
	defer cancelar()
	resultado := iniciarRelay(t, ctx, cliente, entrada, salida, 1024)
	if _, err := servidor.Write([]byte{0x00, 0xff}); err != nil {
		t.Fatal(err)
	}
	var recibido [2]byte
	if _, err := io.ReadFull(leerSalida, recibido[:]); err != nil || recibido != [2]byte{0x00, 0xff} {
		t.Fatalf("respuesta mientras stdin sigue abierto: %v", err)
	}
	if _, err := escribirEntrada.Write([]byte{0x01, 0xfe}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(servidor, recibido[:]); err != nil || recibido != [2]byte{0x01, 0xfe} {
		t.Fatalf("petición: %v", err)
	}
	_ = servidor.Close()
	if err := esperarRelay(t, resultado); err != nil {
		t.Fatal(err)
	}
	if _, err := escribirEntrada.Write([]byte{0x01}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("stdin sigue abierto tras desconexión TCP: %v", err)
	}
}

func TestCancelacionCierraEntradaBloqueada(t *testing.T) {
	cliente, _ := parejaTCP(t)
	entrada, escribirEntrada := io.Pipe()
	t.Cleanup(func() { _ = escribirEntrada.Close() })
	ctx, cancelar := context.WithCancel(context.Background())
	resultado := iniciarRelay(t, ctx, cliente, entrada, &salidaBuffer{}, 1024)
	cancelar()
	if err := esperarRelay(t, resultado); err == nil {
		t.Fatal("cancelación devolvió éxito")
	}
	if _, err := escribirEntrada.Write([]byte{0x00}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("entrada no cerrada: %v", err)
	}
}

func TestPlazoCierraSalidaBloqueada(t *testing.T) {
	cliente, servidor := parejaTCP(t)
	entrada, escribirEntrada := io.Pipe()
	leerSalida, salida := io.Pipe()
	t.Cleanup(func() { _ = escribirEntrada.Close(); _ = leerSalida.Close() })
	ctx, cancelar := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelar()
	resultado := iniciarRelay(t, ctx, cliente, entrada, salida, 1024)
	if _, err := servidor.Write([]byte{0x00}); err != nil {
		t.Fatal(err)
	}
	if err := esperarRelay(t, resultado); err == nil {
		t.Fatal("deadline devolvió éxito con salida bloqueada")
	}
	if _, err := leerSalida.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("salida no cerrada: %v", err)
	}
}

func TestLimiteNoCopiaByteAdicional(t *testing.T) {
	for _, entrada := range []string{"1234", "12345", strings.Repeat("x", 4096)} {
		var salida bytes.Buffer
		if err := copiarAcotado(&salida, strings.NewReader(entrada), 4); !errors.Is(err, errLimite) {
			t.Fatalf("límite no rechazado: %v", err)
		}
		if salida.Len() != 4 {
			t.Fatalf("límite sobrepasado: %d bytes", salida.Len())
		}
	}
	var salida bytes.Buffer
	if err := copiarAcotado(&salida, strings.NewReader("123"), 4); err != nil || salida.String() != "123" {
		t.Fatalf("copia inferior al límite: %v", err)
	}
}

func TestLimiteTCPEnAmbosSentidos(t *testing.T) {
	for _, sentido := range []string{"entrada", "salida"} {
		t.Run(sentido, func(t *testing.T) {
			cliente, servidor := parejaTCP(t)
			entrada, escribirEntrada := io.Pipe()
			t.Cleanup(func() { _ = escribirEntrada.Close() })
			salida := &salidaBuffer{}
			resultado := iniciarRelay(t, context.Background(), cliente, entrada, salida, 4)
			if sentido == "entrada" {
				go func() { _, _ = escribirEntrada.Write([]byte("123456")) }()
				recibido, err := io.ReadAll(servidor)
				if err != nil || string(recibido) != "1234" {
					t.Fatalf("límite entrada TCP: %q, %v", recibido, err)
				}
			} else if _, err := servidor.Write([]byte("123456")); err != nil {
				t.Fatal(err)
			}
			if err := esperarRelay(t, resultado); !errors.Is(err, errLimite) {
				t.Fatalf("límite no termina canal: %v", err)
			}
			if sentido == "salida" && salida.String() != "1234" {
				t.Fatalf("límite salida TCP: %q", salida.String())
			}
		})
	}
}

func TestHuellaRegularSinEnlaces(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "relay")
	contenido := []byte("binario sintético sin secretos")
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if err := huellaEjecutable(ruta, &salida); err != nil {
		t.Fatal(err)
	}
	if esperado := fmt.Sprintf("%x\n", sha256.Sum256(contenido)); salida.String() != esperado || salida.Len() != 65 {
		t.Fatalf("formato huella incorrecto: %q", salida.String())
	}
	enlace := filepath.Join(dir, "enlace")
	if err := os.Symlink(ruta, enlace); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, invalida := range []string{enlace, fifo, dir, filepath.Join(dir, "ausente")} {
		salida.Reset()
		if err := huellaEjecutable(invalida, &salida); err == nil || salida.Len() != 0 {
			t.Fatalf("huella de ruta inválida aceptada: %v", err)
		}
	}
}
