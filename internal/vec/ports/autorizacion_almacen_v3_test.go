package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const finalidadAlmacenV3Prueba = "gestion_documental"

var camposCustodiaV3Prueba = []string{"documento_firmado.custodia", "evidencia_custodia"}

type opcionesAlmacenV3Prueba struct {
	accion          string // acción de la solicitud
	accionConcedida string // acción que concede el rol (vacía: la misma)
	finalidad       string
	campos          []string
	obligaciones    []string
	requiereObjeto  bool
	decisionRef     string
}

type almacenV3Prueba struct {
	ahora        time.Time
	solicitud    domain.SolicitudAutorizacionLigadaV3
	decision     domain.DecisionAutorizacionLigadaV3
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3
	vinculos     VinculosOperacionAlmacen
	validaHasta  time.Time
}

// nuevoAlmacenV3Prueba evalúa y registra una decisión V3 real (con la
// instantánea de autorización del escenario común) sobre un recurso que
// vincula la operación de almacén. Si la decisión no es una concesión, la
// confirmación queda a cero: no hay registro de denegaciones como concesión.
func nuevoAlmacenV3Prueba(t *testing.T, o opcionesAlmacenV3Prueba) almacenV3Prueba {
	t.Helper()
	base := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosBase, err := base.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if o.finalidad == "" {
		o.finalidad = finalidadAlmacenV3Prueba
	}
	if o.accionConcedida == "" {
		o.accionConcedida = o.accion
	}
	if o.decisionRef == "" {
		o.decisionRef = "dec_almacen56789abcdef0123456789abcdef"
	}
	vinculos := VinculosOperacionAlmacen{
		OperacionRef: "operacion:almacen:v3", CargaRef: "carga:documental:v3",
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:almacen:v3",
	}
	atributos := map[string]string{
		AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		AtributoAlmacenCargaRef:            vinculos.CargaRef,
		AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if o.requiereObjeto {
		vinculos.ObjetoVinculado = ReferenciaObjetoAlmacen{Referencia: "objeto:almacen:v3", Version: "version:1"}
		atributos[AtributoAlmacenObjetoRef] = vinculos.ObjetoVinculado.Referencia
		atributos[AtributoAlmacenObjetoVersion] = vinculos.ObjetoVinculado.Version
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "documento:v3:001", ModuloID: "documentos", Tipo: "documento_firmado",
		Ambitos: map[string]string{"unidad": "seleccion"}, Atributos: atributos,
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: datosBase.VinculoAutenticacionActor, ReferenciaMotivo: datosBase.ReferenciaMotivo,
		Accion: o.accion, Recurso: recurso, Finalidad: o.finalidad, Correlacion: datosBase.Correlacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	instantanea := base.instantnea
	instantanea.VersionRol.Concesiones = []domain.ConcesionRol{{
		Accion: o.accionConcedida, ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		Finalidades: []string{o.finalidad}, GarantiaMinima: domain.AuthAssuranceSubstantial,
		CamposPermitidos: append([]string(nil), o.campos...), Obligaciones: append([]string(nil), o.obligaciones...),
	}}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, o.decisionRef, base.ahora, base.ahora.Add(90*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	escenario := almacenV3Prueba{ahora: base.ahora, solicitud: solicitud, decision: decision,
		vinculos: vinculos, validaHasta: base.ahora.Add(90 * time.Second)}
	if concedida, _, _ := decision.Resultado(); concedida {
		orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
			solicitud, decision, datosBase.ReferenciaMotivo, base.resultado)
		if err != nil {
			t.Fatal(err)
		}
		escenario.confirmacion, err = RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(), &registroConcesionLigadaV3Prueba{registradaEn: base.ahora.Add(time.Second)}, orden)
		if err != nil {
			t.Fatal(err)
		}
	}
	return escenario
}

func (e almacenV3Prueba) custodiar(instante time.Time) (ContextoOperacionAlmacen, error) {
	return NuevoContextoCustodiarDocumentoFirmadoExpedienteAlmacenV3(e.solicitud, e.decision, e.confirmacion, e.vinculos, instante)
}

func custodiaV3Prueba(t *testing.T) almacenV3Prueba {
	return nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
		accion: AccionNegocioCustodiarDocumentoFirmadoExpediente, campos: camposCustodiaV3Prueba})
}

func exigirDenegacionAlmacenV3(t *testing.T, caso string, _ ContextoOperacionAlmacen, err error) {
	t.Helper()
	if !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Errorf("%s: el almacén se abrió: %v", caso, err)
	}
}

func TestAlmacenV3CustodiaRegistradaSoloEscribe(t *testing.T) {
	e := custodiaV3Prueba(t)
	instante := e.ahora.Add(2 * time.Second)
	contexto, err := e.custodiar(instante)
	if err != nil || contexto.ValidarParaEn(AccionAlmacenEscribir, instante) != nil {
		t.Fatalf("custodia V3 registrada: %v", err)
	}
	proyeccion, err := contexto.Proyeccion()
	if err != nil || proyeccion.AccionNegocio != AccionNegocioCustodiarDocumentoFirmadoExpediente ||
		proyeccion.RecursoRef != "documento:v3:001" || proyeccion.Finalidad != finalidadAlmacenV3Prueba ||
		!proyeccion.ValidaHasta.Equal(e.validaHasta) || proyeccion.AutorizacionRef != "dec_almacen56789abcdef0123456789abcdef" {
		t.Fatalf("proyección: %+v %v", proyeccion, err)
	}
	if _, err := contexto.EvidenciaAutorizacion(); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("un contexto V3 no entrega evidencia V1: %v", err)
	}
	for _, accion := range []string{AccionAlmacenLeer, AccionAlmacenPrepararCargaDirecta, AccionAlmacenConfirmarCargaDirecta,
		AccionAlmacenAbandonarCargaDirecta, AccionAlmacenPromover, AccionAlmacenAplicarRetencion, AccionAlmacenInmovilizar,
		AccionAlmacenLevantarInmovilizacion, AccionAlmacenEliminar, AccionAlmacenAnalizarContenido} {
		if contexto.ValidarParaEn(accion, instante) == nil {
			t.Fatalf("la custodia V3 habilita %s", accion)
		}
	}
	if _, err := contexto.DerivarPaso(PasoAlmacenLeerOriginalDocumento); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("la custodia V3 deriva lectura: %v", err)
	}
	derivado, err := contexto.DerivarPaso(PasoAlmacenCustodiarFirmado)
	if err != nil || derivado.ValidarParaEn(AccionAlmacenEscribir, instante) != nil {
		t.Fatalf("el paso propio se deriva: %v", err)
	}
	// Fuera de la ventana registrada (antes del registro o al caducar) no vale.
	if contexto.ValidarEn(e.ahora.Add(500*time.Millisecond)) == nil || contexto.ValidarEn(e.validaHasta) == nil {
		t.Fatal("la custodia V3 vale fuera de la ventana registrada")
	}
}

func TestAlmacenV3OriginalFirmableExigeConcesionPropiaYVinculoDeIntento(t *testing.T) {
	e := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
		accion: AccionNegocioEscribirOriginalFirmable,
		campos: []string{"original_firmable.contenido", "evidencia_almacen"},
	})
	instante := e.ahora.Add(2 * time.Second)
	contexto, err := NuevoContextoEscribirOriginalFirmableAlmacenV3(
		e.solicitud, e.decision, e.confirmacion, e.vinculos, instante)
	if err != nil || contexto.ValidarParaEn(AccionAlmacenEscribir, instante) != nil ||
		contexto.ValidarParaEn(AccionAlmacenLeer, instante) == nil {
		t.Fatalf("plan de escritura original: %v", err)
	}
	ajeno := e.vinculos
	ajeno.CargaRef = "carga:otro-intento"
	if _, err := NuevoContextoEscribirOriginalFirmableAlmacenV3(
		e.solicitud, e.decision, e.confirmacion, ajeno, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("se acepto otra clave de intento: %v", err)
	}
	firmado := custodiaV3Prueba(t)
	if _, err := NuevoContextoEscribirOriginalFirmableAlmacenV3(
		firmado.solicitud, firmado.decision, firmado.confirmacion, firmado.vinculos, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("se presto la concesion de documento firmado: %v", err)
	}
}

func TestAlmacenV3LecturaRegistradaSoloLeeElObjetoExacto(t *testing.T) {
	e := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
		accion: AccionNegocioLeerOriginalDocumentoGenerado, campos: []string{"contenido", "documento"}, requiereObjeto: true})
	instante := e.ahora.Add(2 * time.Second)
	contexto, err := NuevoContextoLeerDocumentoGeneradoAlmacenV3(e.solicitud, e.decision, e.confirmacion, e.vinculos, instante)
	if err != nil || contexto.ValidarParaEn(AccionAlmacenLeer, instante) != nil || !contexto.coincideObjeto(e.vinculos.ObjetoVinculado) {
		t.Fatalf("lectura V3 registrada: %v", err)
	}
	if contexto.ValidarParaEn(AccionAlmacenEscribir, instante) == nil {
		t.Fatal("la lectura V3 permite escribir")
	}
	ajeno := e.vinculos
	ajeno.ObjetoVinculado.Version = "version:2"
	_, err = NuevoContextoLeerDocumentoGeneradoAlmacenV3(e.solicitud, e.decision, e.confirmacion, ajeno, instante)
	exigirDenegacionAlmacenV3(t, "otra versión del objeto", ContextoOperacionAlmacen{}, err)
	// La concesión de lectura no sirve para custodiar, ni la de custodia para leer.
	c, err := NuevoContextoCustodiarDocumentoFirmadoExpedienteAlmacenV3(e.solicitud, e.decision, e.confirmacion, e.vinculos, instante)
	exigirDenegacionAlmacenV3(t, "lectura usada para custodiar", c, err)
	custodia := custodiaV3Prueba(t)
	c, err = NuevoContextoLeerDocumentoGeneradoAlmacenV3(custodia.solicitud, custodia.decision, custodia.confirmacion, custodia.vinculos, instante)
	exigirDenegacionAlmacenV3(t, "custodia usada para leer", c, err)
}

func TestAlmacenV3NoSeAbreSinRegistroNiConOtraDecision(t *testing.T) {
	e := custodiaV3Prueba(t)
	instante := e.ahora.Add(2 * time.Second)

	sinRegistro := e
	sinRegistro.confirmacion = ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	c, err := sinRegistro.custodiar(instante)
	exigirDenegacionAlmacenV3(t, "decisión no registrada", c, err)

	otra := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{accion: AccionNegocioCustodiarDocumentoFirmadoExpediente,
		campos: camposCustodiaV3Prueba, decisionRef: "dec_otraregistrada0123456789abcdef01"})
	cruzada := e
	cruzada.confirmacion = otra.confirmacion
	c, err = cruzada.custodiar(instante)
	exigirDenegacionAlmacenV3(t, "registro de otra decisión", c, err)
	cruzada = e
	cruzada.decision = otra.decision
	c, err = cruzada.custodiar(instante)
	exigirDenegacionAlmacenV3(t, "decisión con registro ajeno", c, err)

	// Mismo DecisionRef y misma ventana, pero otra decisión (otra finalidad):
	// solo la huella del registro las distingue.
	mismaReferencia := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{accion: AccionNegocioCustodiarDocumentoFirmadoExpediente,
		campos: camposCustodiaV3Prueba, finalidad: "otra_finalidad"})
	cruzada = e
	cruzada.confirmacion = mismaReferencia.confirmacion
	c, err = cruzada.custodiar(instante)
	exigirDenegacionAlmacenV3(t, "registro con la misma referencia de otra decisión", c, err)

	c, err = e.custodiar(e.validaHasta)
	exigirDenegacionAlmacenV3(t, "decisión caducada", c, err)
	c, err = e.custodiar(e.ahora.Add(500 * time.Millisecond))
	exigirDenegacionAlmacenV3(t, "antes del registro", c, err)
	c, err = e.custodiar(time.Time{})
	exigirDenegacionAlmacenV3(t, "sin instante", c, err)
}

func TestAlmacenV3NoSeAbreConOtraAccionRecursoOFinalidad(t *testing.T) {
	instante := time.Date(2026, 7, 15, 8, 0, 2, 0, time.UTC)
	casos := map[string]opcionesAlmacenV3Prueba{
		"otra acción de Documentos": {accion: AccionNegocioLeerOriginalDocumentoGenerado, campos: camposCustodiaV3Prueba},
		"custodia de Bolsa":         {accion: AccionNegocioCustodiarDocumentoFirmado, campos: camposCustodiaV3Prueba},
		"denegada":                  {accion: AccionNegocioCustodiarDocumentoFirmadoExpediente, accionConcedida: AccionNegocioLeerOriginalDocumentoGenerado, campos: camposCustodiaV3Prueba},
		"campos de más":             {accion: AccionNegocioCustodiarDocumentoFirmadoExpediente, campos: append([]string{"contenido"}, camposCustodiaV3Prueba...)},
		"campos de menos":           {accion: AccionNegocioCustodiarDocumentoFirmadoExpediente, campos: camposCustodiaV3Prueba[:1]},
		"con obligación":            {accion: AccionNegocioCustodiarDocumentoFirmadoExpediente, campos: camposCustodiaV3Prueba, obligaciones: []string{"revisar_manual"}},
	}
	for nombre, o := range casos {
		e := nuevoAlmacenV3Prueba(t, o)
		c, err := e.custodiar(instante)
		exigirDenegacionAlmacenV3(t, nombre, c, err)
	}

	e := custodiaV3Prueba(t)
	// Otra finalidad u otro recurso: la decisión no está ligada a esa solicitud.
	otraFinalidad := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
		accion: AccionNegocioCustodiarDocumentoFirmadoExpediente, campos: camposCustodiaV3Prueba, finalidad: "otra_finalidad"})
	cruzada := e
	cruzada.solicitud = otraFinalidad.solicitud
	c, err := cruzada.custodiar(instante)
	exigirDenegacionAlmacenV3(t, "otra finalidad", c, err)
	lectura := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
		accion: AccionNegocioLeerOriginalDocumentoGenerado, campos: []string{"contenido", "documento"}, requiereObjeto: true})
	cruzada = e
	cruzada.solicitud = lectura.solicitud
	c, err = cruzada.custodiar(instante)
	exigirDenegacionAlmacenV3(t, "otro recurso", c, err)

	// Vínculos que no son los del recurso evaluado.
	for nombre, alterar := range map[string]func(*VinculosOperacionAlmacen){
		"otro efecto": func(v *VinculosOperacionAlmacen) { v.EfectoRef = "efecto:almacen:otro" },
		"otro sujeto": func(v *VinculosOperacionAlmacen) {
			v.SujetoSeudonimoHMAC = "hmac-sha256:sujeto_v1:" + strings.Repeat("c", 64)
		},
		"otra carga": func(v *VinculosOperacionAlmacen) { v.CargaRef = "carga:documental:otra" },
		"con objeto": func(v *VinculosOperacionAlmacen) {
			v.ObjetoVinculado = ReferenciaObjetoAlmacen{Referencia: "objeto:x", Version: "v1"}
		},
		"otra operación": func(v *VinculosOperacionAlmacen) { v.OperacionRef = "operacion:almacen:otra" },
	} {
		alterada := e
		alterar(&alterada.vinculos)
		c, err := alterada.custodiar(instante)
		exigirDenegacionAlmacenV3(t, nombre, c, err)
	}
}

// Una copia alterada del contexto V3 no pasa la revalidación estructural.
func TestAlmacenV3ContextoAlteradoNoValida(t *testing.T) {
	e := custodiaV3Prueba(t)
	instante := e.ahora.Add(2 * time.Second)
	contexto, err := e.custodiar(instante)
	if err != nil {
		t.Fatal(err)
	}
	alteraciones := map[string]func(*datosContextoOperacionAlmacen){
		"recurso":        func(d *datosContextoOperacionAlmacen) { d.recursoRef = "documento:v3:otro" },
		"acción":         func(d *datosContextoOperacionAlmacen) { d.accionNegocio = AccionNegocioCustodiarDocumentoFirmado },
		"finalidad":      func(d *datosContextoOperacionAlmacen) { d.finalidad = "otra_finalidad" },
		"vigencia":       func(d *datosContextoOperacionAlmacen) { d.validaHasta = d.validaHasta.Add(time.Hour) },
		"efecto":         func(d *datosContextoOperacionAlmacen) { d.efectoRef = "efecto:almacen:otro" },
		"huella de plan": func(d *datosContextoOperacionAlmacen) { d.huellaPlanEfectoSHA256 = strings.Repeat("e", 64) },
		"evidencia V1 añadida": func(d *datosContextoOperacionAlmacen) {
			decision, recurso, vinculos, momento := autorizacionAlmacenPrueba(t, AccionNegocioCustodiarDocumentoFirmadoExpediente, camposCustodiaV3Prueba, false)
			v1, err := NuevoContextoCustodiarDocumentoFirmadoExpedienteAlmacen(decision, recurso, vinculos, momento)
			if err != nil {
				t.Fatal(err)
			}
			d.evidencia = v1.datos.evidencia
		},
		"sin registro": func(d *datosContextoOperacionAlmacen) {
			copia := *d.decisionV3
			copia.confirmacion = ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
			d.decisionV3 = &copia
		},
	}
	for nombre, alterar := range alteraciones {
		copia := *contexto.datos
		alterar(&copia)
		alterado := ContextoOperacionAlmacen{datos: &copia}
		if alterado.ValidarParaEn(AccionAlmacenEscribir, instante) == nil {
			t.Errorf("%s: el contexto alterado sigue siendo válido", nombre)
		}
	}
	// La huella del plan V3 no coincide con la V1 de los mismos datos.
	ligada, err := contexto.datos.decisionV3.validarPara(especificacionCustodiarDocumentoFirmadoExpediente(), e.vinculos)
	if err != nil {
		t.Fatal(err)
	}
	sinMarca := huellaPlanOperacionAlmacenCampos("", ligada.decisionRef, ligada.huellaDecision, ligada.accion,
		ligada.recurso.Referencia, ligada.huellaRecurso, ligada.finalidad, ligada.correlacionRef, e.vinculos,
		especificacionCustodiarDocumentoFirmadoExpediente())
	if sinMarca == contexto.datos.huellaPlanEfectoSHA256 {
		t.Fatal("la huella del plan V3 no está separada de la V1")
	}
}

// El contexto V3 es opaco: no se serializa ni se imprime su contenido.
func TestAlmacenV3ContextoEsOpaco(t *testing.T) {
	e := custodiaV3Prueba(t)
	contexto, err := e.custodiar(e.ahora.Add(2 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(contexto); err == nil {
		t.Fatal("el contexto V3 se serializa en JSON")
	}
	for _, formato := range []string{"%v", "%+v", "%#v", "%s"} {
		if texto := fmt.Sprintf(formato, contexto); texto != "[CONTEXTO-OPERACION-ALMACEN-OPACO]" ||
			strings.Contains(texto, "documento:v3") || strings.Contains(texto, "dec_") {
			t.Fatalf("%s revela el contexto: %q", formato, texto)
		}
	}
}

// Fija la huella de plan V1 de unas entradas conocidas: la refactorización
// que admite la marca V3 no puede cambiar la identidad de los planes V1.
func TestHuellaPlanV1NoCambia(t *testing.T) {
	vinculos := VinculosOperacionAlmacen{
		OperacionRef: "operacion:1", CargaRef: "carga:1", Clasificacion: "clase",
		SujetoSeudonimoHMAC: "hmac-sha256:s_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:h_v1:" + strings.Repeat("b", 64), EfectoRef: "efecto:1",
	}
	e := especificacionCustodiarDocumentoFirmado()
	v1 := huellaPlanOperacionAlmacenCampos("", "decision:1", strings.Repeat("c", 64), e.accionNegocio,
		"recurso:1", strings.Repeat("d", 64), "finalidad", "correlacion:1", vinculos, e)
	if v1 != huellaPlanV1Fijada {
		t.Fatalf("la huella de plan V1 cambió: %s", v1)
	}
	if v3 := huellaPlanOperacionAlmacenCampos(marcaPlanDecisionV3, "decision:1", strings.Repeat("c", 64), e.accionNegocio,
		"recurso:1", strings.Repeat("d", 64), "finalidad", "correlacion:1", vinculos, e); v3 == v1 {
		t.Fatal("la marca V3 no separa la huella")
	}
}

// huellaPlanV1Fijada se calculó con la función anterior a la refactorización.
const huellaPlanV1Fijada = "a086ceebcbd20684feb201728933dd4f3ff32851509374073c03990307c3c509"
