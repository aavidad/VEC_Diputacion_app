package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func esperadoCustodiaPrueba() *esperadoCustodiaFirmadoCTDesarrollo {
	return &esperadoCustodiaFirmadoCTDesarrollo{documentoRef: "ref:" + strings.Repeat("d", 64),
		expedienteRef: "ref:" + strings.Repeat("e", 64), huellaSHA256: strings.Repeat("1", 64),
		motivo: motivoFirmaDocumentoCTDesarrollo()}
}

func preimagenCustodiaPrueba(e *esperadoCustodiaFirmadoCTDesarrollo, cambios map[string]any) []byte {
	p := map[string]any{"accion": docports.AccionCustodiarFirmado, "id": e.documentoRef, "clave_idempotencia": "ref:" + strings.Repeat("c", 64),
		"modulo_id": moduloProductorCustodiaCT, "expediente_ref": e.expedienteRef, "huella_sha256": e.huellaSHA256}
	for k, v := range cambios {
		p[k] = v
	}
	b, _ := json.Marshal(p)
	return b
}

// La V3 de custodia solo se pide para el documento, el expediente, el módulo
// y el PDF que CT está custodiando en esta petición.
func TestPreimagenCustodiaEsperada(t *testing.T) {
	e := esperadoCustodiaPrueba()
	if !preimagenCustodiaEsperada(preimagenCustodiaPrueba(e, nil), e) {
		t.Fatal("la preimagen esperada no se admite")
	}
	for nombre, cambio := range map[string]map[string]any{
		"otra acción": {"accion": docports.AccionAlta}, "otro documento": {"id": "ref:" + strings.Repeat("a", 64)},
		"otro módulo": {"modulo_id": "dietas"}, "otro expediente": {"expediente_ref": "ref:" + strings.Repeat("f", 64)},
		"otro PDF": {"huella_sha256": strings.Repeat("2", 64)},
	} {
		if preimagenCustodiaEsperada(preimagenCustodiaPrueba(e, cambio), e) {
			t.Errorf("%s: admitida", nombre)
		}
	}
	if preimagenCustodiaEsperada([]byte("no es json"), e) {
		t.Error("preimagen ilegible admitida")
	}
}

func TestPredicadoCustodiaFirmadoCTLigadoAlDocumentoYLaDecision(t *testing.T) {
	e := esperadoCustodiaPrueba()
	preimagen := preimagenCustodiaPrueba(e, nil)
	recurso, err := docports.RecursoV3(docports.AccionCustodiarFirmado, e.documentoRef, preimagen)
	if err != nil {
		t.Fatal(err)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: docports.AccionCustodiarFirmado, Finalidad: docports.FinalidadCustodiarFirmado,
		ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Recurso: recurso}
	ctx := context.WithValue(context.Background(), claveCustodiaFirmadoCTDesarrollo{}, e)
	if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		t.Fatal("sin preimagen fijada por el autorizador se admite")
	}
	suma := sha256.Sum256(preimagen)
	e.fijar(hex.EncodeToString(suma[:]), "")
	if !solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		t.Fatal("la V3 de la preimagen exacta no se admite")
	}
	if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(context.Background(), datos) {
		t.Fatal("sin custodia en curso se admite")
	}
	for nombre, alterar := range map[string]func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		"otra finalidad": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Finalidad = "alta_documento_generado" },
		"otro motivo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = dominiovec.ReferenciaEntradaCatalogo{}
		},
		"otro documento": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Referencia = "ref:" + strings.Repeat("a", 64)
		},
		"otro módulo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.ModuloID = "contratacion_temporal" },
		"otra acción": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Accion = docports.AccionAlta },
		"otra organización": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = map[string]string{"organizacion_ref": "organizacion:ajena"}
		},
		"otra preimagen": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"preimagen_sha256": strings.Repeat("9", 64)}
		},
	} {
		x := datos
		x.Recurso.Ambitos = map[string]string{"organizacion_ref": docports.OrganizacionRefV3}
		x.Recurso.Atributos = map[string]string{"preimagen_sha256": hex.EncodeToString(suma[:])}
		alterar(&x)
		if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, x) {
			t.Errorf("%s: admitida", nombre)
		}
	}
	// Concesión del almacén: exige la decisión que consumirá la SQL y el PDF.
	almacen := datos
	almacen.Recurso.Atributos = map[string]string{
		docautorizacion.AtributoDecisionCustodia: "decision:sql", "documento_huella_sha256": e.huellaSHA256,
		"documento_expediente_ref": e.expedienteRef, "documento_modulo_productor": moduloProductorCustodiaCT,
		"documento_version": "1",
	}
	if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, almacen) {
		t.Fatal("concesión de almacén antes de la decisión SQL")
	}
	e.fijar("", "decision:sql")
	if !solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, almacen) {
		t.Fatal("la concesión de almacén exacta no se admite")
	}
	for clave, valor := range map[string]string{docautorizacion.AtributoDecisionCustodia: "decision:otra",
		"documento_huella_sha256": strings.Repeat("2", 64), "documento_expediente_ref": "ref:" + strings.Repeat("f", 64),
		"documento_modulo_productor": "dietas"} {
		x := almacen
		x.Recurso.Atributos = map[string]string{}
		for k, v := range almacen.Recurso.Atributos {
			x.Recurso.Atributos[k] = v
		}
		x.Recurso.Atributos[clave] = valor
		if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, x) {
			t.Errorf("concesión de almacén con %s ajeno admitida", clave)
		}
	}
}

func TestPredicadoCustodiaExternaV2ConservaMotivoYPDFExacto(t *testing.T) {
	e := esperadoCustodiaPrueba()
	e.motivo = dominiovec.ReferenciaEntradaCatalogo{}
	preimagen := preimagenCustodiaPrueba(e, nil)
	recurso, err := docports.RecursoV3(docports.AccionCustodiarFirmado, e.documentoRef, preimagen)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(preimagen)
	e.fijar(hex.EncodeToString(suma[:]), "")
	if !e.fijarMotivo(motivoFirmaV2CTDesarrollo()) || e.fijarMotivo(motivoFirmaDocumentoCTDesarrollo()) {
		t.Fatal("el motivo de custodia se pudo sustituir")
	}
	ctx := context.WithValue(context.Background(), claveCustodiaFirmadoCTDesarrollo{}, e)
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: docports.AccionCustodiarFirmado,
		Finalidad: docports.FinalidadCustodiarFirmado, ReferenciaMotivo: motivoFirmaV2CTDesarrollo(), Recurso: recurso}
	if !solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		t.Fatal("custodia V2 exacta rechazada")
	}
	datos.ReferenciaMotivo = motivoFirmaDocumentoCTDesarrollo()
	if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		t.Fatal("motivo V1 aceptado en custodia V2")
	}
	datos.ReferenciaMotivo = motivoFirmaV2CTDesarrollo()
	datos.Recurso.Atributos["preimagen_sha256"] = strings.Repeat("2", 64)
	if solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		t.Fatal("PDF ajeno aceptado en custodia V2")
	}
}

// El rol de firma solo incluye la custodia cuando se compone.
func TestInstantaneaFirmaIncluyeCustodiaSoloAlComponer(t *testing.T) {
	v := dominiovec.DatosVinculoAutenticacionActorV2{PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "perfil:ct:firma"}
	ahora := time.Now().UTC()
	sin, err := instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora, false)
	if err != nil {
		t.Fatal(err)
	}
	con, err := instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora, true)
	if err != nil {
		t.Fatal(err)
	}
	cuenta := func(i dominiovec.InstantaneaAutorizacion) (firma, custodia int) {
		for _, c := range i.VersionRol.Concesiones {
			switch c.Accion {
			case ctports.AccionFirmarDocumento:
				firma++
			case docports.AccionCustodiarFirmado:
				custodia++
			}
		}
		return
	}
	if f, c := cuenta(sin); f != 1 || c != 0 {
		t.Fatalf("sin custodia: %d firma, %d custodia", f, c)
	}
	if f, c := cuenta(con); f != 1 || c != 1 {
		t.Fatalf("con custodia: %d firma, %d custodia", f, c)
	}
}

// La sección de custodia falla cerrado si un tipo no está reservado, falta el
// seudonimizador o el documento no es una clave del circuito.
func TestConfiguracionCustodiaDocumentosFallaCerrado(t *testing.T) {
	catalogo, err := conservacion.NuevoCatalogoProvisional(relojRutasDietas{})
	if err != nil {
		t.Fatal(err)
	}
	seud := nuevoSeudonimizadorAlmacenDesarrollo([32]byte{1})
	repo := repositorioCustodiaNoUsado{}
	almacen := almacenNoUsado{}
	if c, err := nuevaCustodiaDocumentosDesarrollo(nil, repo, almacen, catalogo, relojRutasDietas{}, seud); c != nil || err != nil {
		t.Fatal("sin sección no hay custodia ni error")
	}
	valida := &custodiaFirmadoConfigDesarrollo{Documentos: map[string]string{"resolucion": "contratacion_temporal.resolucion_firmada.v1"}}
	if c, err := nuevaCustodiaDocumentosDesarrollo(valida, repo, almacen, catalogo, relojRutasDietas{}, seud); c == nil || err != nil {
		t.Fatalf("configuración válida rechazada: %v", err)
	}
	for nombre, c := range map[string]*custodiaFirmadoConfigDesarrollo{
		"vacía":          {},
		"no reservado":   {Documentos: map[string]string{"resolucion": "contratacion_temporal.borrador.v1"}},
		"sin política":   {Documentos: map[string]string{"resolucion": "contratacion_temporal.inexistente.v1"}},
		"clave inválida": {Documentos: map[string]string{"Resolución": "contratacion_temporal.resolucion_firmada.v1"}},
	} {
		if _, err := nuevaCustodiaDocumentosDesarrollo(c, repo, almacen, catalogo, relojRutasDietas{}, seud); err == nil {
			t.Errorf("%s: admitida", nombre)
		}
	}
	if _, err := nuevaCustodiaDocumentosDesarrollo(valida, repo, almacen, catalogo, relojRutasDietas{}, nil); err == nil {
		t.Error("sin seudonimizador admitida")
	}
}

func TestComponerCustodiaSinSeccionNoCambiaNada(t *testing.T) {
	var f *firmaDocumentoCTDesarrollo
	if f.componerCustodia(&autoridadDocumentosDesarrollo{}) != nil {
		t.Fatal("sin firma compuesta no hay error")
	}
	if (&firmaDocumentoCTDesarrollo{}).componerCustodia(&autoridadDocumentosDesarrollo{}) != nil {
		t.Fatal("sin sección de custodia no hay error")
	}
	if (&firmaDocumentoCTDesarrollo{}).componerCustodia(&autoridadDocumentosDesarrollo{custodia: &custodiaDocumentosDesarrollo{}}) == nil {
		t.Fatal("con sección y sin rutas de firma debe fallar cerrado")
	}
}

type repositorioCustodiaNoUsado struct {
	docports.RepositorioCustodiaFirmado
}

type almacenNoUsado struct{ vecports.AlmacenObjetos }
