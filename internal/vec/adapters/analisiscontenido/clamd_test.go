package analisiscontenido

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAnalizarClamdProtocoloYVeredictos(t *testing.T) {
	ahora := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	// La cadena EICAR se mantiene solo en memoria como bytes inertes.
	eicar := eicarInerte()
	for _, caso := range []struct {
		nombre, respuesta string
		contenido         []byte
		versiones         []string
		estado            estadoAV
		error             bool
	}{
		{"limpio", "stream: OK", []byte("documento sintético"), []string{versionFalsa(ahora, "27604"), versionFalsa(ahora, "27604")}, avLimpio, false},
		{"eicar_inerte", "stream: Eicar-Signature FOUND", eicar, []string{versionFalsa(ahora, "27604"), versionFalsa(ahora, "27604")}, avMalicioso, false},
		{"firmas_caducadas", "stream: OK", []byte("documento sintético"), []string{versionFalsa(ahora.Add(-25*time.Hour), "27604")}, avNoConcluyente, true},
		{"firmas_cambiantes", "stream: OK", []byte("documento sintético"), []string{versionFalsa(ahora, "27604"), versionFalsa(ahora, "27605")}, avNoConcluyente, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			socket, cerrar := clamdFalso(t, caso.contenido, caso.respuesta, caso.versiones)
			defer cerrar()
			info, estado, err := analizarClamd(context.Background(), socket, caso.contenido, 24*time.Hour, ahora)
			if (err != nil) != caso.error || estado != caso.estado {
				t.Fatalf("estado=%q error=%v; esperados %q error=%t", estado, err, caso.estado, caso.error)
			}
			if caso.error {
				if !errors.Is(err, errClamd) || info != (infoClamd{}) {
					t.Fatalf("el fallo debe cerrar sin metadatos: %+v, %v", info, err)
				}
				return
			}
			if info.motorVersion != "1.4.2" || info.firmasRef != "clamdb:27604" {
				t.Fatalf("metadatos de motor inesperados: %+v", info)
			}
			if caso.estado == avMalicioso && !strings.HasPrefix(info.firmaDeteccion, "firma:") {
				t.Fatalf("falta huella de firma: %+v", info)
			}
			if caso.estado == avLimpio && info.firmaDeteccion != "" {
				t.Fatalf("el resultado limpio contiene firma: %+v", info)
			}
		})
	}
}

// TestClamdRealLocal solo se ejecuta contra el socket Unix privado que el
// operador indique expresamente; nunca arranca ni descubre servicios.
func TestClamdRealLocal(t *testing.T) {
	socket := os.Getenv("VEC_TEST_CLAMD_SOCKET_REAL")
	if socket == "" {
		t.Skip("VEC_TEST_CLAMD_SOCKET_REAL no configurado")
	}
	pasivo, err := os.ReadFile(filepath.Join("testdata", "pasivo.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre    string
		contenido []byte
		estado    estadoAV
	}{
		{"pdf_pasivo", pasivo, avLimpio},
		{"eicar_inerte", eicarInerte(), avMalicioso},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancelar()
			info, estado, err := analizarClamd(ctx, socket, caso.contenido, 48*time.Hour, time.Now().UTC())
			if err != nil || estado != caso.estado {
				t.Fatalf("estado=%q error=%v; esperado=%q", estado, err, caso.estado)
			}
			if info.motorVersion == "" || info.firmasRef == "" {
				t.Fatalf("faltan metadatos del motor: %+v", info)
			}
			if caso.estado == avMalicioso && !strings.HasPrefix(info.firmaDeteccion, "firma:") {
				t.Fatalf("falta referencia opaca de firma: %+v", info)
			}
		})
	}
}

func eicarInerte() []byte {
	return []byte(strings.Join([]string{
		"X5O!P%@AP[4\\PZX54(P^)7CC)7}",
		"$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*",
	}, ""))
}

func TestAnalizarClamdRechazaLimitesAntesDeConectar(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		contenido []byte
		edad      time.Duration
	}{
		{"vacio", nil, time.Hour},
		{"tamano", make([]byte, tamanoMaximoPDF+1), time.Hour},
		{"edad_cero", []byte("a"), 0},
		{"edad_excesiva", []byte("a"), 73 * time.Hour},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			_, estado, err := analizarClamd(context.Background(), "/socket/inexistente", caso.contenido, caso.edad, time.Now())
			if estado != avNoConcluyente || !errors.Is(err, errClamd) {
				t.Fatalf("estado=%q error=%v", estado, err)
			}
		})
	}
}

func versionFalsa(fecha time.Time, base string) string {
	return "ClamAV 1.4.2/" + base + "/" + fecha.UTC().Format(time.ANSIC)
}

// clamdFalso verifica comandos, framing big-endian y bytes sin ejecutar el PDF.
func clamdFalso(t *testing.T, esperado []byte, respuesta string, versiones []string) (string, func()) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "clamd.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	hecho := make(chan error, 1)
	go func() {
		indiceVersion := 0
		for {
			conn, err := listener.Accept()
			if err != nil {
				hecho <- nil
				return
			}
			func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				lector := bufio.NewReader(conn)
				comando, err := lector.ReadBytes(0)
				if err != nil {
					t.Errorf("leer comando falso: %v", err)
					return
				}
				switch string(comando) {
				case "zVERSION\x00":
					if indiceVersion >= len(versiones) {
						t.Error("VERSION adicional")
						return
					}
					_, _ = io.WriteString(conn, versiones[indiceVersion]+"\x00")
					indiceVersion++
				case "zINSTREAM\x00":
					var recibido bytes.Buffer
					for {
						var longitud [4]byte
						if _, err := io.ReadFull(lector, longitud[:]); err != nil {
							t.Errorf("leer longitud: %v", err)
							return
						}
						n := binary.BigEndian.Uint32(longitud[:])
						if n == 0 {
							break
						}
						if n > 32<<10 {
							t.Errorf("chunk demasiado grande: %d", n)
							return
						}
						if _, err := io.CopyN(&recibido, lector, int64(n)); err != nil {
							t.Errorf("leer chunk: %v", err)
							return
						}
					}
					if !bytes.Equal(recibido.Bytes(), esperado) {
						t.Errorf("contenido recibido no coincide: %d bytes", recibido.Len())
					}
					_, _ = io.WriteString(conn, respuesta+"\x00")
				default:
					t.Error(fmt.Sprintf("comando desconocido: %q", comando))
				}
			}()
		}
	}()
	return socket, func() {
		_ = listener.Close()
		<-hecho
	}
}
