package application

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"testing"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	documentos "vec-diputacion-granada/internal/vec/documentos/domain"
)

func TestGeneradorReferenciaHechoCumpleFormatoDeRUMYDocumentos(t *testing.T) {
	g := NuevoGeneradorReferenciaHecho()
	patron := regexp.MustCompile(`^ref:[0-9a-f]{64}$`)
	vistas := map[string]bool{}
	for i := 0; i < 64; i++ {
		ref, err := g.NuevoHechoRef()
		if err != nil || !patron.MatchString(ref) || !domain.ReferenciaValida(ref) ||
			!documentos.ReferenciaExpedienteModuloValida("meritos", ref) || vistas[ref] {
			t.Fatalf("referencia fuera del contrato: %q, %v", ref, err)
		}
		vistas[ref] = true
	}
}

type lectorReferenciaFallida struct{}

func (lectorReferenciaFallida) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestGeneradorReferenciaHechoFallaCerradoSinEntropia(t *testing.T) {
	for _, g := range []*GeneradorReferenciaHecho{
		nil, {}, {entropia: lectorReferenciaFallida{}}, {entropia: bytes.NewReader(make([]byte, 32))},
	} {
		if ref, err := g.NuevoHechoRef(); ref != "" || !errors.Is(err, ErrReferenciaHechoNoDisponible) {
			t.Fatalf("fallo de entropia emitio ID: %q, %v", ref, err)
		}
	}
}
