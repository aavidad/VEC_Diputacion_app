package referencias

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestGeneradorCriptograficoLlamamientosSeparaNamespacesYConserva256Bits(t *testing.T) {
	material := append(bytes.Repeat([]byte{0x11}, bytesAleatoriosReferencia),
		bytes.Repeat([]byte{0x11}, bytesAleatoriosReferencia)...)
	generador := GeneradorCriptograficoLlamamientos{lector: bytes.NewReader(material)}

	instantanea, err := generador.NuevaReferenciaInstantaneaOrdenBolsa()
	if err != nil {
		t.Fatalf("generar referencia de instantanea: %v", err)
	}
	propuesta, err := generador.NuevaReferenciaPropuestaLlamamiento()
	if err != nil {
		t.Fatalf("generar referencia de propuesta: %v", err)
	}

	comprobarReferenciaLlamamiento(t, instantanea, prefijoInstantaneaOrdenBolsa)
	comprobarReferenciaLlamamiento(t, propuesta, prefijoPropuestaLlamamiento)
	if instantanea == propuesta {
		t.Fatal("dos namespaces distintos produjeron la misma referencia")
	}
}

func TestGeneradorCriptograficoLlamamientosReintentaCoincidenciaPersonalAccidental(t *testing.T) {
	invalido := bytesReferenciaBase64URL(t,
		"12345678A"+strings.Repeat("a", 33)+"A",
	)
	valido := bytes.Repeat([]byte{0xff}, bytesAleatoriosReferencia)
	generador := GeneradorCriptograficoLlamamientos{
		lector: bytes.NewReader(append(invalido, valido...)),
	}

	referencia, err := generador.NuevaReferenciaInstantaneaOrdenBolsa()
	if err != nil {
		t.Fatalf("el reintento valido fallo: %v", err)
	}
	comprobarReferenciaLlamamiento(t, referencia, prefijoInstantaneaOrdenBolsa)
	if strings.Contains(referencia, "12345678A") {
		t.Fatalf("se devolvio la coincidencia personal descartada: %q", referencia)
	}
}

func TestGeneradorCriptograficoLlamamientosFallaCerrado(t *testing.T) {
	errorEntropia := errors.New("entropia no disponible")
	invalido := bytesReferenciaBase64URL(t,
		"12345678A"+strings.Repeat("a", 33)+"A",
	)

	casos := []struct {
		nombre      string
		nuevoLector func() io.Reader
		esperar     error
	}{
		{nombre: "valor cero", esperar: puertosbolsa.ErrGeneracionReferenciaLlamamiento},
		{
			nombre: "error de CSPRNG",
			nuevoLector: func() io.Reader {
				return lectorErrorLlamamiento{err: errorEntropia}
			},
			esperar: errorEntropia,
		},
		{
			nombre: "entropia incompleta",
			nuevoLector: func() io.Reader {
				return bytes.NewReader(make([]byte, bytesAleatoriosReferencia-1))
			},
			esperar: io.ErrUnexpectedEOF,
		},
		{
			nombre: "todos los intentos invalidos",
			nuevoLector: func() io.Reader {
				return bytes.NewReader(bytes.Repeat(invalido, maximoIntentosReferenciaValida))
			},
			esperar: puertosbolsa.ErrGeneracionReferenciaLlamamiento,
		},
	}
	metodos := []struct {
		nombre  string
		generar func(GeneradorCriptograficoLlamamientos) (string, error)
	}{
		{nombre: "instantanea", generar: func(g GeneradorCriptograficoLlamamientos) (string, error) {
			return g.NuevaReferenciaInstantaneaOrdenBolsa()
		}},
		{nombre: "propuesta", generar: func(g GeneradorCriptograficoLlamamientos) (string, error) {
			return g.NuevaReferenciaPropuestaLlamamiento()
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			for _, metodo := range metodos {
				t.Run(metodo.nombre, func(t *testing.T) {
					var lector io.Reader
					if caso.nuevoLector != nil {
						lector = caso.nuevoLector()
					}
					referencia, err := metodo.generar(GeneradorCriptograficoLlamamientos{lector: lector})
					if referencia != "" || !errors.Is(err, puertosbolsa.ErrGeneracionReferenciaLlamamiento) ||
						!errors.Is(err, caso.esperar) {
						t.Fatalf("resultado = %q, %v", referencia, err)
					}
				})
			}
		})
	}
}

func TestGeneradorCriptograficoLlamamientosEsSeguroEnConcurrencia(t *testing.T) {
	generador := NuevoGeneradorCriptograficoLlamamientos()
	const trabajadores = 32
	const referenciasPorTrabajador = 64

	resultados := make(chan string, trabajadores*referenciasPorTrabajador)
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	for trabajador := 0; trabajador < trabajadores; trabajador++ {
		grupo.Add(1)
		go func(indice int) {
			defer grupo.Done()
			for numero := 0; numero < referenciasPorTrabajador; numero++ {
				var referencia string
				var err error
				if indice%2 == 0 {
					referencia, err = generador.NuevaReferenciaInstantaneaOrdenBolsa()
				} else {
					referencia, err = generador.NuevaReferenciaPropuestaLlamamiento()
				}
				if err != nil {
					errores <- err
					return
				}
				resultados <- referencia
			}
		}(trabajador)
	}
	grupo.Wait()
	close(resultados)
	close(errores)

	for err := range errores {
		t.Fatalf("generacion concurrente: %v", err)
	}
	vistas := make(map[string]struct{}, trabajadores*referenciasPorTrabajador)
	for referencia := range resultados {
		if !puertosbolsa.ReferenciaOpacaLlamamientoValida(referencia) {
			t.Fatalf("referencia concurrente invalida: %q", referencia)
		}
		if _, existe := vistas[referencia]; existe {
			t.Fatalf("referencia concurrente repetida: %q", referencia)
		}
		vistas[referencia] = struct{}{}
	}
	if len(vistas) != trabajadores*referenciasPorTrabajador {
		t.Fatalf("referencias obtenidas = %d", len(vistas))
	}
}

type lectorErrorLlamamiento struct{ err error }

func (l lectorErrorLlamamiento) Read([]byte) (int, error) { return 0, l.err }

func comprobarReferenciaLlamamiento(t *testing.T, referencia, prefijo string) {
	t.Helper()
	if !strings.HasPrefix(referencia, prefijo) ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(referencia) {
		t.Fatalf("referencia fuera del perfil: %q", referencia)
	}
	sufijo := strings.TrimPrefix(referencia, prefijo)
	contenido, err := base64.RawURLEncoding.DecodeString(sufijo)
	if err != nil || len(contenido) != bytesAleatoriosReferencia ||
		base64.RawURLEncoding.EncodeToString(contenido) != sufijo {
		t.Fatalf("sufijo sin 256 bits base64url canonicos: %q, %v", sufijo, err)
	}
}

func bytesReferenciaBase64URL(t *testing.T, valor string) []byte {
	t.Helper()
	contenido, err := base64.RawURLEncoding.DecodeString(valor)
	if err != nil || len(contenido) != bytesAleatoriosReferencia ||
		base64.RawURLEncoding.EncodeToString(contenido) != valor {
		t.Fatalf("fixture base64url no canonico: %q, %v", valor, err)
	}
	return contenido
}
