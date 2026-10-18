package analisiscontenido

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var errClamd = errors.New("analisiscontenido: antivirus no concluyente")

type estadoAV string

const (
	avLimpio        estadoAV = "limpio"
	avMalicioso     estadoAV = "malicioso"
	avNoConcluyente estadoAV = "no_concluyente"
)

type infoClamd struct {
	motorVersion   string
	firmasRef      string
	firmaDeteccion string
}

func analizarClamd(ctx context.Context, socket string, contenido []byte, maxEdad time.Duration, ahora time.Time) (infoClamd, estadoAV, error) {
	if len(contenido) == 0 || len(contenido) > tamanoMaximoPDF || maxEdad <= 0 || maxEdad > 72*time.Hour {
		return infoClamd{}, avNoConcluyente, errClamd
	}
	previa, err := versionClamd(ctx, socket, ahora, maxEdad)
	if err != nil {
		return infoClamd{}, avNoConcluyente, errClamd
	}
	respuesta, err := enviarINSTREAM(ctx, socket, contenido)
	if err != nil {
		return infoClamd{}, avNoConcluyente, errClamd
	}
	posterior, err := versionClamd(ctx, socket, ahora, maxEdad)
	if err != nil || previa.motorVersion != posterior.motorVersion || previa.firmasRef != posterior.firmasRef {
		return infoClamd{}, avNoConcluyente, errClamd
	}
	if respuesta == "stream: OK" {
		return posterior, avLimpio, nil
	}
	if strings.HasPrefix(respuesta, "stream: ") && strings.HasSuffix(respuesta, " FOUND") {
		firma := strings.TrimSuffix(strings.TrimPrefix(respuesta, "stream: "), " FOUND")
		if len(firma) < 1 || len(firma) > 256 || strings.ContainsAny(firma, "\r\n\x00") {
			return infoClamd{}, avNoConcluyente, errClamd
		}
		suma := sha256.Sum256([]byte(firma))
		posterior.firmaDeteccion = "firma:" + hex.EncodeToString(suma[:])
		return posterior, avMalicioso, nil
	}
	return infoClamd{}, avNoConcluyente, errClamd
}

func versionClamd(ctx context.Context, socket string, ahora time.Time, maxEdad time.Duration) (infoClamd, error) {
	conn, err := conectarClamd(ctx, socket)
	if err != nil {
		return infoClamd{}, errClamd
	}
	defer conn.Close()
	if err = escribirTodo(conn, []byte("zVERSION\x00")); err != nil {
		return infoClamd{}, errClamd
	}
	linea, err := leerRegistroClamd(conn, 512)
	if err != nil || !strings.HasPrefix(linea, "ClamAV ") {
		return infoClamd{}, errClamd
	}
	partes := strings.SplitN(strings.TrimPrefix(linea, "ClamAV "), "/", 3)
	if len(partes) != 3 || !referenciaSegura(partes[0], 128) {
		return infoClamd{}, errClamd
	}
	base, err := strconv.ParseUint(partes[1], 10, 64)
	if err != nil || base == 0 {
		return infoClamd{}, errClamd
	}
	fecha, err := time.Parse(time.ANSIC, strings.TrimSpace(partes[2]))
	if err != nil || fecha.After(ahora.UTC().Add(5*time.Minute)) || ahora.UTC().Sub(fecha) > maxEdad {
		return infoClamd{}, errClamd
	}
	return infoClamd{motorVersion: partes[0], firmasRef: "clamdb:" + partes[1]}, nil
}

func enviarINSTREAM(ctx context.Context, socket string, contenido []byte) (string, error) {
	conn, err := conectarClamd(ctx, socket)
	if err != nil {
		return "", errClamd
	}
	defer conn.Close()
	if err = escribirTodo(conn, []byte("zINSTREAM\x00")); err != nil {
		return "", errClamd
	}
	var longitud [4]byte
	for len(contenido) > 0 {
		n := len(contenido)
		if n > 32<<10 {
			n = 32 << 10
		}
		binary.BigEndian.PutUint32(longitud[:], uint32(n))
		if escribirTodo(conn, longitud[:]) != nil || escribirTodo(conn, contenido[:n]) != nil {
			return "", errClamd
		}
		contenido = contenido[n:]
	}
	if escribirTodo(conn, []byte{0, 0, 0, 0}) != nil {
		return "", errClamd
	}
	return leerRegistroClamd(conn, 512)
}

func conectarClamd(ctx context.Context, socket string) (net.Conn, error) {
	if !filepath.IsAbs(socket) {
		return nil, errClamd
	}
	info, err := os.Lstat(socket)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return nil, errClamd
	}
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		return nil, errClamd
	}
	limite := time.Now().Add(12 * time.Second)
	if fecha, existe := ctx.Deadline(); existe && fecha.Before(limite) {
		limite = fecha
	}
	if conn.SetDeadline(limite) != nil {
		_ = conn.Close()
		return nil, errClamd
	}
	return conn, nil
}

func leerRegistroClamd(r io.Reader, max int64) (string, error) {
	lector := bufio.NewReader(io.LimitReader(r, max+1))
	registro, err := lector.ReadBytes(0)
	if err != nil || len(registro) < 2 || int64(len(registro)) > max {
		return "", errClamd
	}
	return string(registro[:len(registro)-1]), nil
}

func escribirTodo(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
