package firmaemisorv2

import (
	"errors"
	"maps"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestFirmaV2RecursoLlevaLosAmbitosDeLaAsignacion(t *testing.T) {
	a, _, e, m, r, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	fuente := a.autorizacion.(*autorizacionPrueba)
	fuente.ambitos = []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef}}, {Clave: "unidad_ref", Valores: []string{m.UnidadFirmanteRef}}}
	ambitos, err := a.ObtenerAmbitosOperadorFirmaV2(ctx)
	if err != nil || ambitos != (ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef}) {
		t.Fatalf("ámbitos de la asignación no leídos: %v %+v", err, ambitos)
	}
	// Un recurso sólo de organización no cubre una asignación con unidad.
	if _, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
		t.Fatal("recurso sin la unidad de la asignación llegó al PDP")
	}
	r.Ambitos["unidad_ref"] = m.UnidadFirmanteRef
	material, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
	if err != nil || material.ValidarEstructura() != nil || e.llamadas != 1 {
		t.Fatalf("recurso con la unidad de la asignación denegado: %v", err)
	}
	d, _ := e.solicitud.Datos()
	if !maps.Equal(d.Recurso.Ambitos, r.Ambitos) {
		t.Fatal("el PDP no recibió los ámbitos de la asignación")
	}
}

func TestFirmaV2RechazaAsignacionesQueNoSonDeFirma(t *testing.T) {
	org := func(m ports.MaterialFirmaVerificadaV2) vd.AmbitoPerfil {
		return vd.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef}}
	}
	for nombre, preparar := range map[string]func(*autorizacionPrueba, ports.MaterialFirmaVerificadaV2){
		"otra_unidad": func(f *autorizacionPrueba, m ports.MaterialFirmaVerificadaV2) {
			f.ambitos = []vd.AmbitoPerfil{org(m), {Clave: "unidad_ref", Valores: []string{"unidad:otra"}}}
		},
		"dimension_ajena": func(f *autorizacionPrueba, m ports.MaterialFirmaVerificadaV2) {
			f.ambitos = []vd.AmbitoPerfil{org(m), {Clave: "expediente_ref", Valores: []string{"expediente:prueba"}}}
		},
		"dos_valores": func(f *autorizacionPrueba, m ports.MaterialFirmaVerificadaV2) {
			f.ambitos = []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef, "organizacion:otra"}}}
		},
		"sin_organizacion": func(f *autorizacionPrueba, m ports.MaterialFirmaVerificadaV2) {
			f.ambitos = []vd.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{m.UnidadFirmanteRef}}}
		},
		"perfil_ajeno": func(f *autorizacionPrueba, _ ports.MaterialFirmaVerificadaV2) { f.ajena = true },
		"fuente_caida": func(f *autorizacionPrueba, _ ports.MaterialFirmaVerificadaV2) { f.fallo = errors.New("caída") },
		"caducada": func(f *autorizacionPrueba, _ ports.MaterialFirmaVerificadaV2) {
			f.ahora = f.ahora.Add(-3 * time.Hour)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			a, _, e, m, r, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
			preparar(a.autorizacion.(*autorizacionPrueba), m)
			r.Ambitos["unidad_ref"] = m.UnidadFirmanteRef
			if _, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
				t.Fatal("asignación no admitida llegó al PDP")
			}
			delete(r.Ambitos, "unidad_ref")
			if _, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
				t.Fatal("asignación no admitida llegó al PDP sin unidad")
			}
		})
	}
}

func TestFirmaV2SinFuenteDeAutorizacionNoAutoriza(t *testing.T) {
	a, f, e, m, r, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	sin, err := NuevoEmisor(f, e, a.motivo, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sin.AutorizarMaterialFirmaVerificadaV2(ctx, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
		t.Fatal("firma autorizada sin leer la asignación")
	}
	if _, err := sin.ObtenerAmbitosOperadorFirmaV2(ctx); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("ámbitos sin fuente")
	}
	var nula *autorizacionPrueba
	if x, err := NuevoEmisorConAmbitos(f, e, a.motivo, e.reloj, nula); x != nil || !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) {
		t.Fatal("constructor aceptó una fuente nula")
	}
}
