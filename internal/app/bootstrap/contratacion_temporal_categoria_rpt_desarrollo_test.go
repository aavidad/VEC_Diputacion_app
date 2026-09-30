package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const categoriaRPTLlamamientoPrueba = "categoria:rpt:administrativo"

func instalarCatalogoRPTLlamamientoPrueba(t *testing.T, s *soporteAltaContratacionTemporalDesarrollo) {
	t.Helper()
	catalogo, err := nuevoCatalogoDesarrollo(
		"../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json",
		"../../../data/catalogos/rpt/v1.rpt-2026.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	s.origen = nuevoOrigenConsultasConCatalogoDesarrollo(catalogo)
}

func prepararCategoriaRPTLlamamientoPrueba(t *testing.T, ctx context.Context, conCategoriaInicial bool) context.Context {
	t.Helper()
	p, ok := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !ok {
		t.Fatal("preparación ausente")
	}
	p.expediente.Fiscalizado = p.expediente.Fiscalizado.Clonar()
	p.expediente.Fiscalizado.Analisis.CategoriaRef = categoriaRPTLlamamientoPrueba
	p.expediente.Fiscalizado.Analisis.GrupoSubgrupo = "C1"
	if conCategoriaInicial {
		p.categoria = categoriaRPTLlamamientoPrueba
	}
	if p.expediente.Fiscalizado.Validar() != nil {
		t.Fatal("expediente RPT de prueba inválido")
	}
	return context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, p)
}

func TestResolucionManualCategoriaRPTPersistidaAutorizaBolsa(t *testing.T) {
	for _, renuncia := range []bool{false, true} {
		t.Run(map[bool]string{false: "aceptacion", true: "renuncia"}[renuncia], func(t *testing.T) {
			ctx, proveedor, l := escenarioRevisionManualPrueba(t)
			instalarCatalogoRPTLlamamientoPrueba(t, proveedor.soporte)
			ctx = prepararCategoriaRPTLlamamientoPrueba(t, ctx, true)
			accion, motivo := puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, motivoResolucionManualDesarrollo(true)
			if renuncia {
				l.solicitud.Respuesta = ports.RespuestaLlamamientoRenunciada
				l.justificante.Respuesta.Solicitud.Respuesta = l.solicitud.Respuesta
				l.local.Solicitud = l.solicitud
				l.local.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{
					Solicitud: l.solicitud, ResolucionRef: l.local.ResolucionRef, LlamamientoRef: l.solicitud.LlamamientoRef,
					ClaveIdempotencia: l.solicitud.ClaveIdempotencia, VersionEsperada: 2, VersionResultante: 3,
					IntencionRef: "outbox:renuncia-rpt-prueba", ComandoOpacoRef: "comando:renuncia-rpt-prueba",
					Estado: ports.OutboxSiguienteCandidatoPendiente, ActualizadaEn: l.local.ResueltaEn,
				}
				accion, motivo = puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo, motivoRenunciaBolsaDesarrollo()
			}
			ctx = context.WithValue(ctx, claveAceptacionRevisadaDesarrollo{}, l)
			r := dominiovec.RecursoAutorizable{Referencia: operacionAceptacionManualDesarrollo(l), ModuloID: "bolsa", Tipo: "integracion_llamamientos_bolsa",
				Ambitos:   map[string]string{"categoria_ref": categoriaRPTLlamamientoPrueba, "unidad_ref": unidadCoberturaContratacionTemporalDesarrollo},
				Atributos: map[string]string{"necesidad_ref": l.justificante.Seleccion.Necesidad.Referencia, "contenido_sha256": strings.Repeat("a", 64)}}
			d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal", Accion: accion, ReferenciaMotivo: motivo, Recurso: r}
			correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
			if err != nil {
				t.Fatal(err)
			}
			d.Correlacion, d.VinculoAutenticacionActor = correlacion, proveedor.soporte.contexto.Vinculo
			ruta := httpinterno.RutaResolucionComunicacionLlamamiento
			if !proveedor.soporte.categoriaBolsaPersistidaEnCatalogo(ctx, ruta, accion) ||
				!solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
				t.Fatal("categoría RPT persistida rechazada")
			}
			vinculo, err := proveedor.soporte.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
				vinculo.PrincipalID, vinculo.PerfilActivoRef, time.Now().UTC(),
				"categoria_rpt_bolsa_prueba", "Categoría RPT Bolsa", "categoria-rpt-bolsa-prueba",
				[]dominiovec.ConcesionRol{{Accion: accion, ModuloID: "bolsa", TipoRecurso: "integracion_llamamientos_bolsa",
					Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}},
				[]dominiovec.AmbitoPerfil{{Clave: "categoria_ref", Valores: proveedor.soporte.origen.referenciasCategorias()},
					{Clave: "unidad_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}}},
			)
			if err != nil {
				t.Fatal(err)
			}
			proveedor.soporte.instantaneaConsultaJustificante = instantanea
			if renuncia {
				proveedor.soporte.instantaneaRenunciaBolsa = instantanea
			} else {
				proveedor.soporte.instantaneaAceptacionBolsa = instantanea
			}
			ctxAutorizacion := context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
			if _, ok := proveedor.soporte.instantaneaParaContexto(ctxAutorizacion, ruta); !ok {
				base, baseValida := proveedor.soporte.instantaneaParaRuta(ruta)
				t.Fatalf("instantánea no emitida para categoría RPT existente: base=%v base_err=%v instantanea_err=%v perfil_fijo=%v", baseValida, base.Validar(), instantanea.Validar(), proveedor.soporte.perfilFijoParaRuta(ruta) != nil)
			}
			if _, ok := proveedor.soporte.motivoAutorizacionParaContexto(ctxAutorizacion, ruta); !ok {
				t.Fatal("motivo de autorización no ligado a la categoría RPT")
			}
			d.Recurso.Ambitos["categoria_ref"] = "categoria:desarrollo:c2"
			if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
				t.Fatal("recurso C2 autorizó expediente RPT")
			}
			d.Recurso.Ambitos["categoria_ref"] = categoriaRPTLlamamientoPrueba
			p := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
			p.categoria = "categoria:desarrollo:c2"
			ctxDesligado := context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, p)
			if proveedor.soporte.categoriaBolsaPersistidaEnCatalogo(ctxDesligado, ruta, accion) ||
				solicitudAutorizacionLlamamientoDesarrolloValida(ctxDesligado, ruta, d) {
				t.Fatal("preparación distinta del análisis persistido autorizada")
			}
			p.categoria, p.expediente.Fiscalizado.Analisis.CategoriaRef = "categoria:rpt:ajena", "categoria:rpt:ajena"
			ctxAjeno := context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, p)
			if proveedor.soporte.categoriaBolsaPersistidaEnCatalogo(ctxAjeno, ruta, accion) {
				t.Fatal("categoría ajena al catálogo RPT autorizada")
			}
			ctxAjeno = context.WithValue(ctxAjeno, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
			if _, ok := proveedor.soporte.instantaneaParaContexto(ctxAjeno, ruta); ok {
				t.Fatal("instantánea emitida para categoría ajena a la RPT")
			}
		})
	}
}

func TestContinuacionCategoriaRPTPersistidaAutorizaSiguiente(t *testing.T) {
	f := escenarioContinuacionCoordinadorPrueba(t)
	if _, err := f.ejecutor.Continuar(f.ctx, f.solicitud); err != nil {
		t.Fatal(err)
	}
	instalarCatalogoRPTLlamamientoPrueba(t, f.ejecutor.soporte)
	var ctx context.Context
	for i, etapa := range f.autoridad.etapas {
		if etapa == "bolsa" {
			ctx = f.autoridad.contextos[i]
			break
		}
	}
	if ctx == nil {
		t.Fatal("permiso de siguiente ausente")
	}
	ctx = prepararCategoriaRPTLlamamientoPrueba(t, ctx, false)
	l := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
	accion := puertosbolsa.AccionAbrirSiguienteLlamamientoDesarrollo
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal", Accion: accion,
		ReferenciaMotivo: motivoContinuacionDesarrollo(true), Recurso: dominiovec.RecursoAutorizable{
			Referencia: operacionSiguienteDesarrollo(l.solicitud), ModuloID: "bolsa", Tipo: "integracion_llamamientos_bolsa",
			Ambitos:   map[string]string{"categoria_ref": categoriaRPTLlamamientoPrueba, "unidad_ref": unidadCoberturaContratacionTemporalDesarrollo},
			Atributos: map[string]string{"necesidad_ref": l.seleccion.Necesidad.Referencia, "contenido_sha256": strings.Repeat("a", 64)},
		}}
	ruta := httpinterno.RutaContinuacionLlamamiento
	if !f.ejecutor.soporte.categoriaBolsaPersistidaEnCatalogo(ctx, ruta, accion) ||
		!solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
		t.Fatal("siguiente llamamiento RPT rechazado")
	}
	d.Recurso.Ambitos["categoria_ref"] = "categoria:desarrollo:c2"
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
		t.Fatal("siguiente llamamiento admitió C2 ajena al análisis")
	}
}
