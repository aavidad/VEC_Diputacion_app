package application

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestCustodiaImagenSoloPNG256Completo(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	img.Set(0, 0, color.RGBA{R: 8, A: 255})
	var salida bytes.Buffer
	if err := png.Encode(&salida, img); err != nil {
		t.Fatal(err)
	}
	if !bytesPNG256(salida.Bytes()) {
		t.Fatal("PNG 256 válido rechazado")
	}
	truncado := salida.Bytes()[:salida.Len()-8]
	if bytesPNG256(truncado) {
		t.Fatal("no se debe aceptar un PNG incompleto")
	}
	img2 := image.NewRGBA(image.Rect(0, 0, 257, 256))
	salida.Reset()
	if err := png.Encode(&salida, img2); err != nil {
		t.Fatal(err)
	}
	if bytesPNG256(salida.Bytes()) {
		t.Fatal("dimensiones distintas de 256")
	}
}

type referenciaImagenPrueba struct {
	activa   bool
	err      error
	llamadas int
}

func (r *referenciaImagenPrueba) ReferenciaActiva(_ context.Context, _ ports.OperacionImagen) (bool, error) {
	r.llamadas++
	return r.activa, r.err
}
func TestCustodiaImagenReferenciaRetiradaCierraLectura(t *testing.T) {
	fuente := &referenciaImagenPrueba{activa: false}
	s := ServicioCustodiaImagen{Usuarios: fuente}
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); !errors.Is(err, ports.ErrImagenProhibida) {
		t.Fatalf("retirada: %v", err)
	}
	fuente.activa = true
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); err != nil {
		t.Fatalf("activa: %v", err)
	}
	fuente.err = errors.New("usuarios indisponible")
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); !errors.Is(err, ports.ErrImagenProhibida) {
		t.Fatalf("fallo de Usuarios debe cerrar: %v", err)
	}
	if fuente.llamadas != 3 {
		t.Fatalf("cada apertura debe consultar de nuevo: %d", fuente.llamadas)
	}
}
