package firmaemisorv2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Reutiliza exclusivamente el emisor sintético de #524: la exportación
// estructural prueba el cotejo, no COSE, la auditoría SQL ni un recorrido E2E.
type emisorConsultaPrueba struct {
	base             *emisorPrueba
	audiencia        string
	confirmacionNula bool
	exportador       vp.ExportadorMaterialConsumoAutorizacionAtestadaV3
	despues          func()
}

func (e *emisorConsultaPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s vd.SolicitudAutorizacionLigadaV3, b vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, c, x, err := e.base.EmitirMaterialAutorizacionAtestadaV3(ctx, s, b)
	if err != nil {
		return d, c, x, err
	}
	if e.confirmacionNula {
		c = vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	}
	if e.exportador != nil {
		return d, c, e.exportador, nil
	}
	m, err := x.ExportarMaterialParaConsumidor()
	if err != nil {
		return d, c, nil, err
	}
	r := m.ResumenCapacidad()
	audiencia := ports.AudienciaConsultaFirmasR5V2
	if e.audiencia != "" {
		audiencia = e.audiencia
	}
	r, err = vp.NuevoResumenCapacidadAtestacionAutorizacionV3(r.DecisionRef(), r.DecisionHuellaSHA256(), r.MotivoHuellaSHA256(), r.ContextoRef(), r.ContextoHuellaSHA256(), r.Operacion(), r.EfectoRef(), r.EfectoHuellaSHA256(), audiencia, r.EmitidaEn(), r.ExpiraEn())
	if err != nil {
		e.base.t.Fatal(err)
	}
	m, err = vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), r, m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		e.base.t.Fatal(err)
	}
	return d, c, &exportadorPrueba{material: m, antes: e.despues}, nil
}

func escenarioConsulta(t *testing.T, via string) (*Emisor, *fuentePrueba, *emisorConsultaPrueba, ports.MaterialConsultaFirmasR5V2, context.Context) {
	t.Helper()
	a, f, e, base, _, ctx := escenario(t, via)
	e.campos = ports.CamposConsultaFirmasR5V2()
	w := &emisorConsultaPrueba{base: e}
	a.emisor = w
	m := ports.MaterialConsultaFirmasR5V2{Via: via, MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: base.OrganizacionRef, ExpedienteRef: base.ExpedienteRef, VersionExpediente: base.VersionExpediente,
		Documento: base.Documento, FirmantePrincipalCandidatoRef: "per_candidato_ajeno", ClaveIdempotencia: base.ClaveIdempotencia, PasoOrden: 1, CatalogoHuella: base.CatalogoHuella}}
	return a, f, w, m, ctx
}

func TestConsultaEmiteContratoV2SinSuplantarCandidato(t *testing.T) {
	for _, via := range []string{ports.ViaFirmaCertificadoVEC, ports.ViaFirmaExternaPortafirmas} {
		t.Run(via, func(t *testing.T) {
			a, f, e, m, ctx := escenarioConsulta(t, via)
			c, err := a.AutorizarConsultaFirmasR5V2(ctx, m)
			if err != nil || firma.ValidarCapacidadConsultaFirmasR5V2(c, m) != nil || f.llamadas != 1 || e.base.llamadas != 1 {
				t.Fatalf("consulta V2: %v", err)
			}
			d, err := e.base.solicitud.Datos()
			if err != nil {
				t.Fatal(err)
			}
			actor, _ := d.VinculoAutenticacionActor.Datos()
			r, _ := firma.RecursoConsultaFirmasR5V2(m)
			cor, _ := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			esperada, _ := cor.ValorCanonico()
			real, _ := d.Correlacion.ValorCanonico()
			if actor.PrincipalID == m.FirmantePrincipalCandidatoRef || actor.PrincipalID != f.base.Resultado.Contexto.Principal.ID ||
				actor.PerfilActivoRef != f.base.Resultado.Contexto.PerfilActivoRef || d.Accion != ports.AccionConsultarFirmasR5V2 || d.Finalidad != ports.FinalidadFirmaDocumento ||
				d.ReferenciaMotivo != a.motivo || real != esperada || d.Recurso.Referencia != r.Referencia || d.Recurso.Tipo != r.Tipo || d.Recurso.ModuloID != r.ModuloID ||
				!maps.Equal(d.Recurso.Ambitos, r.Ambitos) || !maps.Equal(d.Recurso.Atributos, r.Atributos) {
				t.Fatal("contrato, contexto o correlación sustituidos")
			}
			m.FirmantePrincipalCandidatoRef = "per_otro_candidato"
			if firma.ValidarCapacidadConsultaFirmasR5V2(c, m) == nil {
				t.Fatal("una capacidad autoriza otro candidato")
			}
			// Se admite también que operador y candidato coincidan en una lectura.
			m.FirmantePrincipalCandidatoRef = actor.PrincipalID
			if _, err = a.AutorizarConsultaFirmasR5V2(ctx, m); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConsultaMaterialInvalidoNoAlcanzaAutoridades(t *testing.T) {
	for nombre, alterar := range map[string]func(*ports.MaterialConsultaFirmasR5V2){
		"unidad_mal_formada": func(m *ports.MaterialConsultaFirmasR5V2) { m.UnidadRef = "unidad con espacios" },
		"unidad_via_externa": func(m *ports.MaterialConsultaFirmasR5V2) {
			m.Via, m.UnidadRef = ports.ViaFirmaExternaPortafirmas, "unidad:ajena"
		},
		"vía":       func(m *ports.MaterialConsultaFirmasR5V2) { m.Via = "otra" },
		"paso":      func(m *ports.MaterialConsultaFirmasR5V2) { m.PasoOrden = 3 },
		"candidato": func(m *ports.MaterialConsultaFirmasR5V2) { m.FirmantePrincipalCandidatoRef = "" },
		"versión":   func(m *ports.MaterialConsultaFirmasR5V2) { m.VersionExpediente = 0 },
	} {
		t.Run(nombre, func(t *testing.T) {
			a, f, e, m, ctx := escenarioConsulta(t, ports.ViaFirmaCertificadoVEC)
			alterar(&m)
			c, err := a.AutorizarConsultaFirmasR5V2(ctx, m)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || c.ExportarMaterialParaConsumidor().ValidarEstructura() == nil || f.llamadas != 0 || e.base.llamadas != 0 {
				t.Fatal("material inválido alcanzó fuente o emisor")
			}
		})
	}
}

func TestConsultaProyeccionExactaYExportacionLigada(t *testing.T) {
	for _, caso := range []string{"campos vacíos", "campo ausente", "campo futuro", "obligación", "audiencia antigua", "confirmación nula", "contexto cruzado", "sesión vencida", "capacidad vencida", "exportador nulo tipado"} {
		t.Run(caso, func(t *testing.T) {
			a, _, e, m, ctx := escenarioConsulta(t, ports.ViaFirmaCertificadoVEC)
			switch caso {
			case "campos vacíos":
				e.base.campos = nil
			case "campo ausente":
				e.base.campos = e.base.campos[1:]
			case "campo futuro":
				e.base.campos = append(e.base.campos, "campo_futuro")
			case "obligación":
				e.base.obligaciones = []string{"doble_control"}
			case "audiencia antigua":
				e.audiencia = ports.AudienciaConsultaFirmasR5V3
			case "confirmación nula":
				e.confirmacionNula = true
			case "contexto cruzado":
				e.base.alterar = func(x vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
					cruzado, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(x.CapacidadCanonica(), x.ResumenCapacidad(), x.DecisionCanonica(), x.MotivoCanonico(), []byte(`{"contexto":"ajeno"}`), x.PersonaVersion(), x.PerfilVersion(), x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
					if err != nil {
						t.Fatal(err)
					}
					return cruzado
				}
			case "sesión vencida":
				e.despues = func() { e.base.reloj.ahora = e.base.reloj.ahora.Add(11 * time.Minute) }
			case "capacidad vencida":
				e.despues = func() { e.base.reloj.ahora = e.base.reloj.ahora.Add(5 * time.Second) }
			case "exportador nulo tipado":
				var nulo *exportadorPrueba
				e.exportador = nulo
			}
			c, err := a.AutorizarConsultaFirmasR5V2(ctx, m)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || c.ExportarMaterialParaConsumidor().ValidarEstructura() == nil {
				t.Fatalf("proyección/material divergente autorizado: %v", err)
			}
		})
	}
}

func TestConsultaCierraCanalCancelacionYErrores(t *testing.T) {
	for _, etapa := range []string{"sin correlación", "contexto nil", "emisor nil", "emisor cero", "fuente error", "fuente cancelada", "emisor error", "exportador error", "exportador cancelado"} {
		t.Run(etapa, func(t *testing.T) {
			a, f, e, m, base := escenarioConsulta(t, ports.ViaFirmaExternaPortafirmas)
			ctx, cancelar := context.WithCancel(base)
			defer cancelar()
			fallo := fmt.Errorf("detalle-privado: %w", context.DeadlineExceeded)
			causa := error(ports.ErrFirmaDocumentoDenegada)
			switch etapa {
			case "sin correlación":
				ctx = context.Background()
			case "contexto nil":
				ctx = nil
			case "emisor nil":
				a = nil
			case "emisor cero":
				a = &Emisor{}
			case "fuente error":
				f.fallo = fallo
				causa = context.DeadlineExceeded
			case "fuente cancelada":
				f.antes = cancelar
				causa = context.Canceled
			case "emisor error":
				e.base.fallo = fallo
				causa = context.DeadlineExceeded
			case "exportador error":
				e.exportador = &exportadorPrueba{fallo: fallo}
				causa = context.DeadlineExceeded
			case "exportador cancelado":
				e.despues = cancelar
				causa = context.Canceled
			}
			c, err := a.AutorizarConsultaFirmasR5V2(ctx, m)
			if !errors.Is(err, causa) || strings.Contains(err.Error(), "detalle-privado") || c.ExportarMaterialParaConsumidor().ValidarEstructura() == nil {
				t.Fatalf("error/cancelación no opacos: %v", err)
			}
		})
	}
}

// La consulta lleva los ámbitos de la asignación de quien consulta: con
// unidad, sólo un material con esa misma unidad llega al PDP, que recibe
// organización y unidad; otra unidad o ninguna se deniegan antes. La
// recuperación sigue la misma regla.
func TestConsultaLlevaLosAmbitosDeLaAsignacion(t *testing.T) {
	a, f, e, m, ctx := escenarioConsulta(t, ports.ViaFirmaCertificadoVEC)
	unidad := "unidad:del:paso"
	a.autorizacion.(*autorizacionPrueba).ambitos = []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef}},
		{Clave: "unidad_ref", Valores: []string{unidad}}}
	for caso, u := range map[string]string{"sin_unidad": "", "otra_unidad": "unidad:otra"} {
		m.UnidadRef = u
		if _, err := a.AutorizarConsultaFirmasR5V2(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.base.llamadas != 0 {
			t.Fatalf("%s llegó al PDP: %v", caso, err)
		}
		if _, err := a.AutorizarRecuperacionFirmasV2(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.base.llamadas != 0 {
			t.Fatalf("recuperación %s llegó al PDP: %v", caso, err)
		}
	}
	m.UnidadRef = unidad
	c, err := a.AutorizarConsultaFirmasR5V2(ctx, m)
	if err != nil || firma.ValidarCapacidadConsultaFirmasR5V2(c, m) != nil || f.llamadas == 0 || e.base.llamadas != 1 {
		t.Fatalf("consulta con la unidad de la asignación denegada: %v", err)
	}
	d, _ := e.base.solicitud.Datos()
	if !maps.Equal(d.Recurso.Ambitos, map[string]string{"organizacion_ref": m.OrganizacionRef, "unidad_ref": unidad}) {
		t.Fatalf("el PDP no recibió los ámbitos de la asignación: %v", d.Recurso.Ambitos)
	}
}

// La huella de contexto que calcula el PDP en Go para una consulta con unidad
// es la misma que calcula AD210 en SQL: el vector ct186_ad210 fija este mismo
// valor con las mismas entradas.
func TestFirmaV2HuellaConsultaConUnidadIgualQueSQL(t *testing.T) {
	sol := `{"Via":"certificado_vec","OrganizacionRef":"org_fija","UnidadRef":"unidad:fija"}`
	material := sha256.Sum256([]byte(sol))
	r := vd.RecursoAutorizable{Referencia: "exp:fijo", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5,
		Ambitos:   map[string]string{"organizacion_ref": "org_fija", "unidad_ref": "unidad:fija"},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(material[:])}}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || h != "c051b4e351005621ea4eeaca10696497a6d59c545eac14de9c590a7d939cefcf" {
		t.Fatalf("huella distinta de la de AD210: %s %v", h, err)
	}
}

// Sin fuente de ámbitos (la composición R5 actual usa NuevoEmisor) el emisor
// no deniega por su cuenta: la consulta y la recuperación llegan al PDP, que
// exige los mismos ámbitos, y AD210 los relee al consumir.
func TestConsultaSinFuenteDeAmbitosDecideElPDP(t *testing.T) {
	a, _, e, m, ctx := escenarioConsulta(t, ports.ViaFirmaCertificadoVEC)
	sin, err := NuevoEmisor(a.fuente, a.emisor, a.motivo, a.reloj)
	if err != nil {
		t.Fatal(err)
	}
	m.UnidadRef = "unidad:del:paso"
	if _, err := sin.AutorizarConsultaFirmasR5V2(ctx, m); err != nil || e.base.llamadas != 1 {
		t.Fatalf("consulta sin fuente de ámbitos denegada antes del PDP: %v", err)
	}
	if _, err := sin.AutorizarRecuperacionFirmasV2(ctx, m); e.base.llamadas != 2 {
		t.Fatalf("recuperación sin fuente de ámbitos no llegó al PDP: %v", err)
	}
}
