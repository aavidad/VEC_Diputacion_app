package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func datosInformeJuridicoPerfilFijoPrueba(expediente, version, fase, estado string) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
	sello := func(dominio, digito string) string {
		return "hmac-sha256:" + dominio + "/v1:" + strings.Repeat(digito, 64)
	}
	return dominiovec.DatosSolicitudAutorizacionLigadaV3{
		ReferenciaMotivo: referenciaMotivoAutorizacionInformeJuridicoDesarrollo(),
		Accion:           ports.AccionEmitirInformeJuridico,
		Finalidad:        finalidadInformeJuridicoContratacionTemporalDesarrollo,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: expediente, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoInformeJuridico,
			Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"fase_previa": fase, "estado_previo": estado},
			Atributos: map[string]string{
				"version_expediente":          version,
				"configuracion_ref":           definicionInformeJuridicoDesarrollo,
				"configuracion_version":       "1",
				"configuracion_huella_sha256": huellaAltaContratacionTemporalDesarrollo("configuracion-informe-juridico"),
				"plantilla_ref":               plantillaInformeJuridicoDesarrollo,
				"plantilla_version":           "1",
				"plantilla_huella_sha256":     huellaAltaContratacionTemporalDesarrollo("plantilla-informe-juridico"),
				"borrador_huella_sha256":      strings.Repeat("c", 64),
				"ambito_idempotencia_hmac":    sello(ports.DominioAmbitoIdempotenciaInformeJuridico, "a"),
				"huella_peticion_hmac":        sello(ports.DominioHuellaPeticionInformeJuridico, "b"),
			},
		},
	}
}

// El informe se admite solo en los pares fase/estado del catálogo: el
// inicial tras la asignación y el nuevo tras subsanar. Las combinaciones
// cruzadas, que el ámbito V3 por sí solo no distingue, se rechazan antes del
// PDP, igual que el expediente en los ámbitos.
func TestInformeJuridicoPerfilFijoAdmiteSoloParesDelCatalogo(t *testing.T) {
	fase := fasesOperacionPredeterminadasCT()[operacionFaseInformeJuridicoCT]
	ruta := httpinterno.RutaPreparacionesInformeJuridico
	subsanacion := string(domain.FaseSubsanacionUnidad)
	for _, caso := range []struct {
		version, fase, estado string
		valida                bool
	}{
		{"4", "asignacion_unidad", string(domain.EstadoEnCurso), true},
		{"7", subsanacion, string(domain.EstadoIncidencia), true},
		{"4", "asignacion_unidad", string(domain.EstadoIncidencia), false},
		{"7", subsanacion, string(domain.EstadoEnCurso), false},
		{"3", "asignacion_unidad", string(domain.EstadoEnCurso), false},
		{"4", "solicitud", string(domain.EstadoEnCurso), false},
	} {
		datos := datosInformeJuridicoPerfilFijoPrueba("expediente:informe:uno", caso.version, caso.fase, caso.estado)
		if solicitudAutorizacionInformeJuridicoContratacionTemporalDesarrolloValida(ruta, datos, fase) != caso.valida {
			t.Fatalf("v%s %s/%s: validez distinta de %v", caso.version, caso.fase, caso.estado, caso.valida)
		}
	}
	datos := datosInformeJuridicoPerfilFijoPrueba("expediente:informe:uno", "4", "asignacion_unidad", string(domain.EstadoEnCurso))
	datos.Recurso.Ambitos["expediente_ref"] = "expediente:informe:uno"
	if solicitudAutorizacionInformeJuridicoContratacionTemporalDesarrolloValida(ruta, datos, fase) {
		t.Fatal("aceptado con el expediente en los ámbitos")
	}
	// La plantilla del perfil cubre las fases y estados declarados, sin
	// expediente.
	plantilla, err := nuevaInstantaneaAutorizacionInformeJuridicoContratacionTemporalDesarrollo(
		"persona:tecnica:desarrollo:001", "perfil:tecnico:desarrollo:001", time.Now().UTC(), fase)
	if err != nil {
		t.Fatal(err)
	}
	ambitos := map[string][]string{}
	for _, a := range plantilla.AsignacionPerfil.Ambitos {
		ambitos[a.Clave] = a.Valores
	}
	if len(ambitos) != 3 || len(ambitos["fase_previa"]) != 2 || len(ambitos["estado_previo"]) != 2 || ambitos["expediente_ref"] != nil {
		t.Fatalf("ámbitos del perfil del informe inesperados: %+v", ambitos)
	}
	if _, err := nuevaInstantaneaAutorizacionInformeJuridicoContratacionTemporalDesarrollo(
		"persona:tecnica:desarrollo:001", "perfil:tecnico:desarrollo:001", time.Now().UTC(), faseOperacionCT{}); err == nil {
		t.Fatal("plantilla creada sin fases")
	}
}

// La asignación y el informe consumen la asignación publicada de su perfil
// fijo, sin preparar ni publicar, y la deniegan si no es consumible.
func TestAsignacionEInformePerfilFijoConsumenSinPublicar(t *testing.T) {
	s, _, autoridad, _, principal := escenarioAnalisisPerfilFijoPrueba(t)
	informe := datosInformeJuridicoPerfilFijoPrueba("expediente:informe:dos", "4", "asignacion_unidad", string(domain.EstadoEnCurso))
	fijo := s.perfilFijoParaRuta(httpinterno.RutaPreparacionesInformeJuridico)
	ctx := context.WithValue(contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaPreparacionesInformeJuridico),
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, informe)
	consumida, ok := s.instantaneaParaContexto(ctx, httpinterno.RutaPreparacionesInformeJuridico)
	if !ok || consumida.AsignacionPerfil.PerfilActivoRef != fijo.perfilRef() || !consumida.AsignacionPerfil.Cubre(informe.Recurso) {
		t.Fatalf("el informe no consumió su perfil fijo: %v", ok)
	}
	// Revocada, se deniega.
	publicada := autoridad.asignaciones[fijo.perfilRef()]
	publicada.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	autoridad.asignaciones[fijo.perfilRef()] = publicada
	if _, ok := s.instantaneaParaContexto(ctx, httpinterno.RutaPreparacionesInformeJuridico); ok {
		t.Fatal("el informe consumió una asignación revocada")
	}
	// La asignación sin solicitud válida se deniega (nunca cae al dinámico).
	ctxAsignacion := context.WithValue(contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaAsignaciones),
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, informe)
	if _, ok := s.instantaneaParaContexto(ctxAsignacion, httpinterno.RutaAsignaciones); ok {
		t.Fatal("la asignación aceptó una solicitud de informe")
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatalf("se preparó %d y publicó %d", autoridad.preparadas, autoridad.publicadas)
	}
}
