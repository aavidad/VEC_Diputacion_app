package bootstrap

import (
	"context"
	"errors"
	"testing"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type puertaOriginalCTPrueba struct{ err error }

func (p puertaOriginalCTPrueba) Verificar(context.Context) error { return p.err }

type consultorOriginalCTMontajePrueba struct {
	ctports.ConsultorOriginalFirmableRRHH
}
type renderOriginalCTMontajePrueba struct {
	ctports.RenderizadorBorradorRRHH
}
type autorizacionesOriginalCTMontajePrueba struct {
	almacenvec.AutorizacionesDocumentosOriginalCT
}
type almacenOriginalCTMontajePrueba struct{ vecports.AlmacenObjetos }
type lecturaOriginalCTMontajePrueba struct {
	docports.FabricaContextoLectura
}

func preparacionOriginalCTPrueba(t *testing.T, v2 bool, puerta puertaOriginalFirmableCTDesarrollo) *autoridadDocumentosDesarrollo {
	t.Helper()
	reloj := relojRutasDietas{}
	var (
		catalogo *conservacion.Catalogo
		err      error
	)
	if v2 {
		catalogo, err = conservacion.NuevoCatalogoProvisionalV2(reloj)
	} else {
		catalogo, err = conservacion.NuevoCatalogoProvisional(reloj)
	}
	if err != nil {
		t.Fatal(err)
	}
	return &autoridadDocumentosDesarrollo{originalCT: &preparacionOriginalFirmableCTDesarrollo{
		servicio: &docapp.Servicio{Repositorio: &docpg.Repositorio{}, Almacen: almacenOriginalCTMontajePrueba{},
			Politicas: catalogo, Reloj: reloj, ContextosLectura: lecturaOriginalCTMontajePrueba{}},
		catalogo: catalogo, puerta: puerta,
	}}
}

func TestOriginalFirmableCTMontajeDeniegaSinAutoridadOContrato(t *testing.T) {
	ctx := context.Background()
	lector := consultorOriginalCTMontajePrueba{}
	render := renderOriginalCTMontajePrueba{}
	autorizaciones := autorizacionesOriginalCTMontajePrueba{}
	for _, tc := range []struct {
		nombre     string
		documentos *autoridadDocumentosDesarrollo
		permisos   almacenvec.AutorizacionesDocumentosOriginalCT
	}{
		{"sin_documentos", nil, autorizaciones},
		{"sin_autorizaciones_nominales", preparacionOriginalCTPrueba(t, true, puertaOriginalCTPrueba{}), nil},
		{"sin_doc13_o_ad158", preparacionOriginalCTPrueba(t, true, puertaOriginalCTPrueba{errors.New("fachada ausente")}), autorizaciones},
		{"catalogo_v1_sin_reserva", preparacionOriginalCTPrueba(t, false, puertaOriginalCTPrueba{}), autorizaciones},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			servicio, err := nuevoServicioOriginalFirmableCTDesarrollo(ctx, tc.documentos, lector, lector, render, tc.permisos)
			if servicio != nil || !errors.Is(err, ErrOriginalFirmableCTMontajeNoDisponible) {
				t.Fatalf("montaje debe denegarse: servicio=%v error=%v", servicio, err)
			}
		})
	}
}

func TestOriginalFirmableCTMontajeCatalogoV2YDependencias(t *testing.T) {
	documentos := preparacionOriginalCTPrueba(t, true, puertaOriginalCTPrueba{})
	lector := consultorOriginalCTMontajePrueba{}
	servicio, err := nuevoServicioOriginalFirmableCTDesarrollo(context.Background(), documentos,
		lector, lector, renderOriginalCTMontajePrueba{}, autorizacionesOriginalCTMontajePrueba{})
	if err != nil || servicio == nil {
		t.Fatalf("montaje con tipos reservados: servicio=%v error=%v", servicio, err)
	}
}
