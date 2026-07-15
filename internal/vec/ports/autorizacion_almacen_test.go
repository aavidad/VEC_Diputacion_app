package ports

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestContextoOperacionAlmacenPreparacionEsOpacoTemporalYDePlanCerrado(t *testing.T) {
	decision, recurso, vinculos, instante := autorizacionAlmacenPrueba(
		t, AccionNegocioPrepararCargaDocumental,
		[]string{"tamano", "contenido", "mime", "clasificacion", "huella_sha256"}, false,
	)
	contexto, err := NuevoContextoPrepararCargaDirectaAlmacen(decision, recurso, vinculos, instante)
	if err != nil {
		t.Fatalf("crear capacidad: %v", err)
	}
	proyeccion, err := contexto.Proyeccion()
	if err != nil || proyeccion.AccionNegocio != AccionNegocioPrepararCargaDocumental ||
		proyeccion.AccionTecnica != AccionAlmacenPrepararCargaDirecta ||
		proyeccion.PasoRef != PasoAlmacenPrepararCargaDirecta ||
		proyeccion.AutorizacionRef != decision.DecisionRef || proyeccion.EfectoRef != vinculos.EfectoRef ||
		!esSHA256Hexadecimal(proyeccion.HuellaPlanEfectoSHA256) ||
		!esSHA256Hexadecimal(proyeccion.HuellaDecisionSHA256) {
		t.Fatalf("proyeccion inesperada: %#v, %v", proyeccion, err)
	}
	if err := contexto.ValidarParaEn(AccionAlmacenPrepararCargaDirecta, instante); err != nil {
		t.Fatalf("capacidad vigente denegada: %v", err)
	}
	abandono, err := contexto.DerivarPaso(PasoAlmacenAbandonarCargaDirecta)
	if err != nil {
		t.Fatalf("derivar compensacion declarada: %v", err)
	}
	proyeccionAbandono, err := abandono.Proyeccion()
	if err != nil || proyeccionAbandono.AccionTecnica != AccionAlmacenAbandonarCargaDirecta ||
		proyeccionAbandono.HuellaPlanEfectoSHA256 != proyeccion.HuellaPlanEfectoSHA256 ||
		proyeccionAbandono.EfectoRef != proyeccion.EfectoRef {
		t.Fatalf("compensacion fuera del plan: %#v, %v", proyeccionAbandono, err)
	}
	if _, err := contexto.DerivarPaso(PasoAlmacenAnalizarContenido); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("un paso no declarado debe denegarse: %v", err)
	}
	if err := contexto.ValidarParaEn(AccionAlmacenLeer, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("otra accion tecnica debe denegarse: %v", err)
	}
	if err := contexto.ValidarEn(decision.ValidaHasta); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("el extremo de caducidad debe denegarse: %v", err)
	}
	if _, err := json.Marshal(contexto); !errors.Is(err, ErrSerializacionContextoAlmacenProhibida) {
		t.Fatalf("serializacion de capacidad permitida: %v", err)
	}
	if _, err := json.Marshal(proyeccion); !errors.Is(err, ErrSerializacionContextoAlmacenProhibida) {
		t.Fatalf("serializacion de proyeccion permitida: %v", err)
	}
	texto := strings.Join([]string{contexto.String(), contexto.GoString(), proyeccion.String()}, " ")
	if strings.Contains(texto, decision.DecisionRef) || strings.Contains(texto, vinculos.EfectoRef) {
		t.Fatalf("formato revela referencias: %s", texto)
	}
}

func TestContextoOperacionAlmacenDeniegaAusenciaCamposObligacionesComodinYAccionDesconocida(t *testing.T) {
	decision, recurso, vinculos, instante := autorizacionAlmacenPrueba(
		t, AccionNegocioPrepararCargaDocumental,
		[]string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}, false,
	)
	casos := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion, *domain.RecursoAutorizable, *VinculosOperacionAlmacen)
	}{
		{"accion desconocida", func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *VinculosOperacionAlmacen) {
			d.Accion = "vec.documentos.carga.desconocida"
		}},
		{"campo omitido", func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *VinculosOperacionAlmacen) {
			d.CamposPermitidos = d.CamposPermitidos[:4]
		}},
		{"campo adicional", func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *VinculosOperacionAlmacen) {
			d.CamposPermitidos = append(d.CamposPermitidos, "todo")
		}},
		{"obligacion no implementada", func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *VinculosOperacionAlmacen) {
			d.Obligaciones = []string{"doble_control"}
		}},
		{"comodin", func(_ *domain.DecisionAutorizacion, r *domain.RecursoAutorizable, _ *VinculosOperacionAlmacen) {
			r.Atributos[AtributoAlmacenEfectoRef] = "efecto:*"
		}},
		{"efecto no vinculado", func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *VinculosOperacionAlmacen) {
			v.EfectoRef = "efecto:otro"
		}},
		{"objeto inesperado", func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *VinculosOperacionAlmacen) {
			v.ObjetoVinculado = ReferenciaObjetoAlmacen{Referencia: "objeto:1", Version: "v1"}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d := clonarDecisionAutorizacionCanonica(decision)
			r := clonarRecursoAlmacenPrueba(recurso)
			v := vinculos
			caso.mutar(&d, &r, &v)
			if _, err := NuevoContextoPrepararCargaDirectaAlmacen(d, r, v, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
				t.Fatalf("caso debe denegarse: %v", err)
			}
		})
	}
	var cero ContextoOperacionAlmacen
	if _, err := cero.Proyeccion(); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("valor cero debe denegarse: %v", err)
	}
}

func TestContextoOperacionAlmacenAnalisisYRetencionVinculanObjetoYVersionExactos(t *testing.T) {
	decision, recurso, vinculos, instante := autorizacionAlmacenPrueba(
		t, AccionNegocioAnalizarCargaDocumental,
		[]string{"estado", "analisis_seguridad"}, true,
	)
	contexto, err := NuevoContextoAnalizarCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
	if err != nil {
		t.Fatalf("crear plan de analisis: %v", err)
	}
	if !contexto.coincideObjeto(vinculos.ObjetoVinculado) ||
		contexto.coincideObjeto(ReferenciaObjetoAlmacen{Referencia: vinculos.ObjetoVinculado.Referencia, Version: "v2"}) {
		t.Fatal("el objeto no quedo ligado por referencia y version")
	}
	analisis, err := contexto.DerivarPaso(PasoAlmacenAnalizarContenido)
	if err != nil || analisis.ValidarParaEn(AccionAlmacenAnalizarContenido, instante) != nil {
		t.Fatalf("paso de analisis declarado invalido: %v", err)
	}

	decisionRetencion, recursoRetencion, vinculosRetencion, instanteRetencion := autorizacionAlmacenPrueba(
		t, AccionNegocioRetenerDocumentoFirmado,
		[]string{"documento_firmado.retencion", "evidencia_retencion"}, true,
	)
	retencion, err := NuevoContextoRetenerDocumentoFirmadoAlmacen(
		decisionRetencion, recursoRetencion, vinculosRetencion, instanteRetencion,
	)
	if err != nil || !retencion.coincideObjeto(vinculosRetencion.ObjetoVinculado) {
		t.Fatalf("retencion exacta invalida: %v", err)
	}
	conCampos := clonarDecisionAutorizacionCanonica(decisionRetencion)
	conCampos.CamposPermitidos = []string{"objeto"}
	if _, err := NuevoContextoRetenerDocumentoFirmadoAlmacen(
		conCampos, recursoRetencion, vinculosRetencion, instanteRetencion,
	); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("retencion de baremacion con campos debe denegarse: %v", err)
	}
}

func autorizacionAlmacenPrueba(
	t *testing.T,
	accion string,
	campos []string,
	requiereObjeto bool,
) (domain.DecisionAutorizacion, domain.RecursoAutorizable, VinculosOperacionAlmacen, time.Time) {
	t.Helper()
	decision, instante := decisionAutorizacionReforzadaPrueba(t)
	vinculos := VinculosOperacionAlmacen{
		OperacionRef: "operacion:almacen:001", CargaRef: "carga:documental:001",
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:almacen:001",
	}
	atributos := map[string]string{
		AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		AtributoAlmacenCargaRef:            vinculos.CargaRef,
		AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if requiereObjeto {
		vinculos.ObjetoVinculado = ReferenciaObjetoAlmacen{Referencia: "objeto:almacen:001", Version: "version:1"}
		atributos[AtributoAlmacenObjetoRef] = vinculos.ObjetoVinculado.Referencia
		atributos[AtributoAlmacenObjetoVersion] = vinculos.ObjetoVinculado.Version
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:almacen:001", ModuloID: "bolsa", Tipo: "documento_bolsa",
		Ambitos: map[string]string{"organizacion": "diputacion_granada"}, Atributos: atributos,
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella recurso: %v", err)
	}
	decision.Accion = accion
	decision.RecursoRef = recurso.Referencia
	decision.ModuloID = recurso.ModuloID
	decision.TipoRecurso = recurso.Tipo
	decision.ContextoRecursoHuellaSHA256 = huellaRecurso
	decision.Finalidad = "custodia_documental"
	decision.CorrelacionRef = "correlacion:almacen:001"
	decision.CamposPermitidos = append([]string(nil), campos...)
	decision.Obligaciones = nil
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de almacen: %v", err)
	}
	return decision, recurso, vinculos, instante
}

func clonarRecursoAlmacenPrueba(recurso domain.RecursoAutorizable) domain.RecursoAutorizable {
	resultado := recurso
	resultado.Ambitos = make(map[string]string, len(recurso.Ambitos))
	for clave, valor := range recurso.Ambitos {
		resultado.Ambitos[clave] = valor
	}
	resultado.Atributos = make(map[string]string, len(recurso.Atributos))
	for clave, valor := range recurso.Atributos {
		resultado.Atributos[clave] = valor
	}
	return resultado
}
