package firmaemisorv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestEmisionConservaIdentidadCorrelacionYContratoNominal(t *testing.T) {
	for _, via := range []string{ports.ViaFirmaCertificadoVEC, ports.ViaFirmaExternaPortafirmas} {
		t.Run(via, func(t *testing.T) {
			a, f, e, m, r, ctx := escenario(t, via)
			perfil, err := a.ObtenerPerfilActivoOperadorFirmaV2(ctx)
			if err != nil || perfil != m.PerfilActivoOperadorRef {
				t.Fatal("perfil no procede de la fuente")
			}
			material, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
			if err != nil || material.ValidarEstructura() != nil || e.llamadas != 1 || f.llamadas != 2 {
				t.Fatalf("emisión nominal: %v", err)
			}
			d, err := e.solicitud.Datos()
			if err != nil {
				t.Fatal(err)
			}
			actor, _ := d.VinculoAutenticacionActor.Datos()
			correlacion, _ := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			actual, _ := d.Correlacion.ValorCanonico()
			esperado, _ := correlacion.ValorCanonico()
			accion, audiencia := ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
			if via == ports.ViaFirmaExternaPortafirmas {
				accion, audiencia = ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
			}
			if actual != esperado || d.Accion != accion || d.Finalidad != ports.FinalidadFirmaDocumento || d.ReferenciaMotivo != a.motivo ||
				actor.PrincipalID != f.base.Resultado.Contexto.Principal.ID || actor.PerfilActivoRef != m.PerfilActivoOperadorRef || material.ResumenCapacidad().AudienciaConsumo() != audiencia {
				t.Fatal("identidad, correlación o contrato sustituidos")
			}
			r.Atributos["material_sha256"] = strings.Repeat("0", 64)
			r.Ambitos["organizacion_ref"] = "organizacion:alterada"
			despues, _ := e.solicitud.Datos()
			if despues.Recurso.Atributos["material_sha256"] != d.Recurso.Atributos["material_sha256"] || despues.Recurso.Ambitos["organizacion_ref"] != m.OrganizacionRef {
				t.Fatal("solicitud mutable desde llamador")
			}
			canon := material.ContextoActorCanonico()
			canon[0] ^= 1
			if !bytes.Equal(material.ContextoActorCanonico(), f.base.Resultado.RepresentacionCanonica) {
				t.Fatal("exportación comparte bytes")
			}
		})
	}
}

func TestRechazosPreviosNoAlcanzanEmisor(t *testing.T) {
	casos := map[string]func(*fuentePrueba, *emisorPrueba, *ports.MaterialFirmaVerificadaV2, *vd.RecursoAutorizable){
		"perfil": func(_ *fuentePrueba, _ *emisorPrueba, m *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			m.PerfilActivoOperadorRef = "prf_ajeno"
			*r = recursoPrueba(t, *m)
		},
		"principal VEC": func(_ *fuentePrueba, _ *emisorPrueba, m *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			m.FirmantePrincipalRef = "per_ajeno"
			*r = recursoPrueba(t, *m)
		},
		"perfil firmante VEC": func(_ *fuentePrueba, _ *emisorPrueba, m *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			m.PerfilActivoFirmanteRef = "prf_ajeno"
			*r = recursoPrueba(t, *m)
		},
		"cuenta VEC": func(_ *fuentePrueba, _ *emisorPrueba, m *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			m.CuentaFirmanteRef = "cuenta:ajena"
			*r = recursoPrueba(t, *m)
		},
		"certificado canal": func(f *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, _ *vd.RecursoAutorizable) {
			f.base.CertificadoCanalSHA256 = strings.Repeat("f", 64)
		},
		"contexto inválido": func(f *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, _ *vd.RecursoAutorizable) {
			f.base.Resultado.RepresentacionCanonica[0] ^= 1
		},
		"vínculo vacío": func(f *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, _ *vd.RecursoAutorizable) {
			f.base.Vinculo = vd.VinculoAutenticacionActorV2{}
		},
		"sesión vencida": func(_ *fuentePrueba, e *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, _ *vd.RecursoAutorizable) {
			e.reloj.ahora = e.reloj.ahora.Add(11 * time.Minute)
		},
		"descriptor ausente": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			delete(r.Atributos, "descriptor_firma_sha256")
		},
		"descriptor inválido": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.Atributos["descriptor_firma_sha256"] = "secreto"
		},
		"material ajeno": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.Atributos["material_sha256"] = strings.Repeat("f", 64)
		},
		"recurso ajeno": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.Referencia = "operacion:ajena"
		},
		"módulo ajeno": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.ModuloID = "bolsa"
		},
		"tipo ajeno": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.Tipo = ports.TipoRecursoFirmaExterna
		},
		"ámbito añadido": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.Ambitos["unidad_ref"] = "unidad:ajena"
		},
		"atributo añadido": func(_ *fuentePrueba, _ *emisorPrueba, _ *ports.MaterialFirmaVerificadaV2, r *vd.RecursoAutorizable) {
			r.Atributos["otro"] = "valor"
		},
		"material inválido": func(_ *fuentePrueba, _ *emisorPrueba, m *ports.MaterialFirmaVerificadaV2, _ *vd.RecursoAutorizable) {
			m.Via = "vía desconocida"
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			a, f, e, m, r, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
			alterar(f, e, &m, &r)
			material, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || material.ValidarEstructura() == nil || e.llamadas != 0 {
				t.Fatalf("guardas previas: %v llamadas=%d", err, e.llamadas)
			}
		})
	}
	for _, caso := range []string{"correlación ausente", "operador externo es firmante"} {
		t.Run(caso, func(t *testing.T) {
			a, f, e, m, r, ctx := escenario(t, ports.ViaFirmaExternaPortafirmas)
			if caso == "correlación ausente" {
				ctx = context.Background()
			} else {
				m.FirmantePrincipalRef = f.base.Resultado.Contexto.Principal.ID
				r = recursoPrueba(t, m)
			}
			_, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
				t.Fatalf("identidad/canal externa: %v", err)
			}
		})
	}
}

func TestPerfilSeRevalidaEnCadaOperacion(t *testing.T) {
	a, f, e, m, r, ctx := escenario(t, ports.ViaFirmaExternaPortafirmas)
	if _, err := a.ObtenerPerfilActivoOperadorFirmaV2(ctx); err != nil {
		t.Fatal(err)
	}
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(e.reloj.ahora, f.base.Resultado.Contexto.PersonaRef, "prf_abcdefghijkl0123456789", vd.AuthMethodCertificate, vd.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	f.base.Resultado, f.base.Vinculo = resultado, vinculo
	if _, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
		t.Fatal("perfil anterior autorizado")
	}
	perfil, err := a.ObtenerPerfilActivoOperadorFirmaV2(ctx)
	if err != nil || perfil != resultado.Contexto.PerfilActivoRef || f.llamadas != 3 {
		t.Fatal("perfil capturado por constructor")
	}
}

func TestCancelacionYErroresSonOpacos(t *testing.T) {
	for _, etapa := range []string{"antes", "fuente", "emisor", "exportador", "error fuente", "error emisor", "error exportador"} {
		t.Run(etapa, func(t *testing.T) {
			a, f, e, m, r, base := escenario(t, ports.ViaFirmaCertificadoVEC)
			ctx, cancelar := context.WithCancel(base)
			defer cancelar()
			fallo := fmt.Errorf("fixture-privada: %w", context.DeadlineExceeded)
			causa := error(context.Canceled)
			switch etapa {
			case "antes":
				cancelar()
			case "fuente":
				f.antes = cancelar
			case "emisor":
				e.antes = cancelar
			case "exportador":
				e.exportador = &exportadorPrueba{antes: cancelar}
			case "error fuente":
				f.fallo = fallo
				causa = context.DeadlineExceeded
			case "error emisor":
				e.fallo = fallo
				causa = context.DeadlineExceeded
			case "error exportador":
				e.exportador = &exportadorPrueba{fallo: fallo}
				causa = context.DeadlineExceeded
			}
			material, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
			if !errors.Is(err, causa) || material.ValidarEstructura() == nil || strings.Contains(err.Error(), "fixture-privada") {
				t.Fatalf("fallo/cancelación no cerrado: %v", err)
			}
			if (etapa == "antes" || etapa == "fuente" || etapa == "error fuente") && e.llamadas != 0 {
				t.Fatal("cancelación previa alcanzó emisor")
			}
		})
	}
}

func TestEmisionRechazaRestriccionesYExportacionCruzada(t *testing.T) {
	for _, caso := range []string{"campos", "obligaciones", "exportador nulo tipado", "contexto cruzado", "motivo cruzado", "decisión cruzada", "capacidad vencida"} {
		t.Run(caso, func(t *testing.T) {
			a, _, e, m, r, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
			switch caso {
			case "campos":
				e.campos = []string{"Documento"}
			case "obligaciones":
				e.obligaciones = []string{"doble_control"}
			case "exportador nulo tipado":
				var nulo *exportadorPrueba
				e.exportador = nulo
			case "contexto cruzado", "motivo cruzado", "decisión cruzada":
				e.alterar = func(x vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
					decision, motivo, actor := x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico()
					switch caso {
					case "contexto cruzado":
						actor = []byte(`{"contexto":"ajeno"}`)
					case "motivo cruzado":
						motivo = []byte(`{"motivo":"ajeno"}`)
					case "decisión cruzada":
						decision = []byte(`{"decision":"ajena"}`)
					}
					cruzado, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(x.CapacidadCanonica(), x.ResumenCapacidad(), decision, motivo, actor,
						x.PersonaVersion(), x.PerfilVersion(), x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
					if err != nil || cruzado.ValidarEstructura() != nil {
						t.Fatal("transporte cruzado debe ser estructuralmente válido")
					}
					return cruzado
				}
			case "capacidad vencida":
				e.alterar = func(x vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
					e.reloj.ahora = e.reloj.ahora.Add(5 * time.Second)
					return x
				}
			}
			x, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || x.ValidarEstructura() == nil {
				t.Fatalf("material no acreditado aceptado: %v", err)
			}
		})
	}
}

func TestNulosNoPanican(t *testing.T) {
	a, f, e, m, r, _ := escenario(t, ports.ViaFirmaCertificadoVEC)
	var fuenteNula *fuentePrueba
	var emisorNulo *emisorPrueba
	var relojNulo *relojPrueba
	for _, c := range []struct {
		f FuenteContextoActorFirmaV2
		e EmisorComunV3
		r vd.RelojVinculoAutenticacionActorV2
	}{{fuenteNula, e, e.reloj}, {f, emisorNulo, e.reloj}, {f, e, relojNulo}} {
		if x, err := NuevoEmisor(c.f, c.e, a.motivo, c.r); x != nil || !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) {
			t.Fatal("constructor aceptó nulo tipado")
		}
	}
	if x, err := NuevoEmisor(f, e, vd.ReferenciaEntradaCatalogo{}, e.reloj); x != nil || err == nil {
		t.Fatal("motivo no gobernado aceptado")
	}
	var emisor *Emisor
	for _, x := range []*Emisor{emisor, {}, a} {
		if _, err := x.ObtenerPerfilActivoOperadorFirmaV2(nil); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
			t.Fatal("perfil desde contexto nil")
		}
		if _, err := x.AutorizarMaterialFirmaVerificadaV2(nil, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
			t.Fatal("material desde contexto nil")
		}
	}
}
