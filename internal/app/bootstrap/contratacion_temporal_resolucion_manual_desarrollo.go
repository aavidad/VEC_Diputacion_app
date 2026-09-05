package bootstrap

import (
	"context"
	"maps"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Exclusiva de la composición de doble llave de desarrollo. La persona
// revisora declara un hecho del ejercicio sintético; no se calcula un plazo
// legal ni se acredita una entrega a partir del registro local del aviso.
const criterioRevisionManualDesarrollo = "politica:ct:revision-manual-sintetica:20260906"
const politicaRevisionManualDesarrollo = `{"esquema":"vec.ct.revision-manual.sintetica.v1","entorno":"desarrollo_sintetico","metodo":"declaracion_explicita_rrhh","requiere_revision_respuesta":true,"requiere_revision_plazo_ejercicio":true,"calcula_plazo_legal":false,"acredita_entrega":false}`

type claveResolucionManualDesarrollo struct{}
type claveAceptacionRevisadaDesarrollo struct{}

type aceptacionRevisadaDesarrollo struct {
	solicitud    ports.SolicitudResolverLlamamiento
	justificante ports.JustificanteRespuestaRecibida
	local        ports.ResultadoResolucionLlamamiento
}

type aceptadorRespuestaRRHHDesarrollo interface {
	AceptarRespuestaRRHH(context.Context, ports.SolicitudResolverLlamamiento,
		ports.ReciboSolicitudLlamamientoBolsa, puertosbolsa.ResolucionLlamamientoDesarrollo) (puertosbolsa.ReciboLlamamientoDesarrollo, error)
	RenunciarRespuestaRRHH(context.Context, ports.SolicitudResolverLlamamiento,
		ports.ReciboSolicitudLlamamientoBolsa, puertosbolsa.ResolucionLlamamientoDesarrollo) (puertosbolsa.ReciboLlamamientoDesarrollo, error)
}

func politicaManualDesarrollo() ports.ReferenciaGobernadaComunicacionLlamamiento {
	return referenciaCatalogoComunicacionDesarrollo(criterioRevisionManualDesarrollo, politicaRevisionManualDesarrollo)
}

func motivoResolucionManualDesarrollo(bolsa bool) dominiovec.ReferenciaEntradaCatalogo {
	tipo := "resolucion_manual_rrhh"
	if bolsa {
		tipo = "aceptacion_bolsa_rrhh"
	}
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_" + tipo, CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(tipo + "-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", tipo),
	}
}

func motivoRenunciaBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_renuncia_bolsa_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("renuncia_bolsa_rrhh-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "renuncia_bolsa_rrhh"),
	}
}

func (p *proveedorComunicacionLlamamientoDesarrollo) PrepararResolucionManual(ctx context.Context, s ports.SolicitudResolverLlamamiento) (postgresct.MaterialResolucionManualLlamamiento, error) {
	vacio := postgresct.MaterialResolucionManualLlamamiento{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) || s.Validar() != nil ||
		!s.RevisionManualConfirmada() || s.CriterioValidacionRef != criterioRevisionManualDesarrollo {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	c, valida := p.soporte.capacidadValida(ctx)
	preparacion, preparada := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	ligada, existe := ctx.Value(claveResolucionManualDesarrollo{}).(aceptacionRevisadaDesarrollo)
	if !valida || c.ruta != httpinterno.RutaResolucionComunicacionLlamamiento ||
		!preparada || !existe || ligada.solicitud != s || ligada.justificante.ValidarPara(s) != nil ||
		!consultaJustificanteLigadaAlExpedienteDesarrollo(preparacion.expediente, s) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return postgresct.MaterialResolucionManualLlamamiento{Solicitud: s, Politica: politicaManualDesarrollo()}, nil
}

func (p *proveedorComunicacionLlamamientoDesarrollo) AutorizarResolucionManual(ctx context.Context, m postgresct.MaterialResolucionManualLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	esperado, err := p.PrepararResolucionManual(ctx, m.Solicitud)
	if err != nil {
		return vacio, err
	}
	if esperado != m || dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizadorResolucion) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	recurso, err := postgresct.RecursoResolucionManualLlamamiento(m)
	if err != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	a, err := p.autorizadorResolucion.AutorizarOperacion(ctx, postgresct.AccionResolucionManualLlamamiento, recurso)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || a.ValidarEstructura() != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	r, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) || r.ExpiraEn().Sub(r.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return a, nil
}

func (e *ejecutorComunicacionLlamamientoDesarrollo) resolverConRevisionManual(ctx context.Context, s ports.SolicitudResolverLlamamiento, j ports.JustificanteRespuestaRecibida) (ports.ResultadoResolucionLlamamiento, error) {
	vacio := ports.ResultadoResolucionLlamamiento{}
	if s.CriterioValidacionRef != criterioRevisionManualDesarrollo || !s.RevisionManualConfirmada() {
		return vacio, application.ErrComunicacionLlamamientoDenegada
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) || dependenciaEsNulaContratacionTemporalDesarrollo(e.aceptador) {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	ligada := aceptacionRevisadaDesarrollo{solicitud: s, justificante: j}
	ctx = context.WithValue(ctx, claveResolucionManualDesarrollo{}, ligada)
	local, err := e.servicio.Resolver(ctx, s)
	if err != nil {
		return vacio, err
	}
	if local.ValidarPara(s) != nil || local.Politica != politicaManualDesarrollo() || local.ResueltaEn.Before(j.Respuesta.RegistradaEn) {
		return vacio, application.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	// El commit CT queda durable incluso si Bolsa falla. No hay compensación
	// destructiva ni recibo de éxito parcial: reintento explícito, misma clave,
	// mismos antecedentes y autorización fresca para ambos módulos.
	ligada.local = local
	ctx = context.WithValue(ctx, claveAceptacionRevisadaDesarrollo{}, ligada)
	resolucionBolsa := puertosbolsa.ResolucionLlamamientoDesarrollo{
		AperturaOperacionRef: j.Seleccion.OperacionRef, JustificanteRef: s.PruebaRespuestaRef,
		EvaluacionPlazoRef: local.EvaluacionPlazoRef, PoliticaRef: local.Politica.Referencia,
		PoliticaVersion: local.Politica.Version, PoliticaSHA256: local.Politica.HuellaSHA256, VersionEsperada: 1,
	}
	var b puertosbolsa.ReciboLlamamientoDesarrollo
	tipo := "aceptacion_rrhh"
	if s.Respuesta == ports.RespuestaLlamamientoRenunciada {
		tipo = "renuncia_rrhh"
		b, err = e.aceptador.RenunciarRespuestaRRHH(ctx, s, j.Seleccion, resolucionBolsa)
	} else {
		b, err = e.aceptador.AceptarRespuestaRRHH(ctx, s, j.Seleccion, resolucionBolsa)
	}
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	res := b.Registro.Resolucion
	if res == nil || res.Validar() != nil || b.Registro.Tipo != tipo ||
		b.Registro.OperacionRef != operacionAceptacionManualDesarrollo(ligada) ||
		res.AperturaOperacionRef != j.Seleccion.OperacionRef || res.JustificanteRef != s.PruebaRespuestaRef ||
		res.EvaluacionPlazoRef != local.EvaluacionPlazoRef || res.PoliticaRef != local.Politica.Referencia ||
		res.PoliticaVersion != local.Politica.Version || res.PoliticaSHA256 != local.Politica.HuellaSHA256 ||
		!domain.ReferenciaOpacaValida(b.ReciboRef) || !domain.InstanteUTCCanonico(b.ConfirmadaEn) || b.ConfirmadaEn.Before(local.ResueltaEn) {
		return vacio, application.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return local, nil
}

func operacionAceptacionManualDesarrollo(l aceptacionRevisadaDesarrollo) string {
	prefijo := "operacion-aceptacion-rrhh"
	if l.solicitud.Respuesta == ports.RespuestaLlamamientoRenunciada {
		prefijo = "operacion-renuncia-rrhh"
	}
	return referenciaPuenteLlamamientoDesarrollo(prefijo,
		l.solicitud.OrganizacionRef, l.solicitud.ExpedienteRef, l.justificante.Seleccion.OperacionRef, l.solicitud.ClaveIdempotencia)
}

func solicitudAutorizacionResolucionManualDesarrolloValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3, p preparacionLlamamientoDesarrollo) bool {
	if datos.Accion == postgresct.AccionResolucionManualLlamamiento {
		l, ok := ctx.Value(claveResolucionManualDesarrollo{}).(aceptacionRevisadaDesarrollo)
		if !ok || !l.solicitud.RevisionManualConfirmada() || l.solicitud.CriterioValidacionRef != criterioRevisionManualDesarrollo ||
			l.justificante.ValidarPara(l.solicitud) != nil || !consultaJustificanteLigadaAlExpedienteDesarrollo(p.expediente, l.solicitud) ||
			datos.ReferenciaMotivo != motivoResolucionManualDesarrollo(false) {
			return false
		}
		esperado, err := postgresct.RecursoResolucionManualLlamamiento(postgresct.MaterialResolucionManualLlamamiento{Solicitud: l.solicitud, Politica: politicaManualDesarrollo()})
		r := datos.Recurso
		return err == nil && r.Referencia == esperado.Referencia && r.ModuloID == esperado.ModuloID && r.Tipo == esperado.Tipo &&
			maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
	}
	l, ok := ctx.Value(claveAceptacionRevisadaDesarrollo{}).(aceptacionRevisadaDesarrollo)
	if !ok || !l.solicitud.RevisionManualConfirmada() || l.justificante.ValidarPara(l.solicitud) != nil ||
		!consultaJustificanteLigadaAlExpedienteDesarrollo(p.expediente, l.solicitud) ||
		l.local.ValidarPara(l.solicitud) != nil || l.local.Politica != politicaManualDesarrollo() {
		return false
	}
	accion, motivo := puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, motivoResolucionManualDesarrollo(true)
	if l.solicitud.Respuesta == ports.RespuestaLlamamientoRenunciada {
		accion, motivo = puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo, motivoRenunciaBolsaDesarrollo()
	}
	if datos.Accion != accion || datos.ReferenciaMotivo != motivo {
		return false
	}
	// Solo la composición instala el recibo CT confirmado. El servicio Bolsa
	// construye y comprueba el canon completo antes de pedir este permiso.
	r := datos.Recurso
	return r.ModuloID == "bolsa" && r.Tipo == "integracion_llamamientos_bolsa" &&
		r.Referencia == operacionAceptacionManualDesarrollo(l) &&
		len(r.Ambitos) == 2 && r.Ambitos["categoria_ref"] == "categoria:desarrollo:c2" &&
		r.Ambitos["unidad_ref"] == unidadCoberturaContratacionTemporalDesarrollo &&
		len(r.Atributos) == 2 && r.Atributos["necesidad_ref"] == l.justificante.Seleccion.Necesidad.Referencia &&
		huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["contenido_sha256"])
}

func (s *soporteAltaContratacionTemporalDesarrollo) motivoAutorizacionParaContexto(ctx context.Context, ruta string) (dominiovec.ReferenciaEntradaCatalogo, bool) {
	if ruta == httpinterno.RutaResolucionComunicacionLlamamiento && ctx != nil {
		d, ok := ctx.Value(claveSolicitudAutorizacionContratacionTemporalDesarrollo{}).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
		if !ok || !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
			return dominiovec.ReferenciaEntradaCatalogo{}, false
		}
		return d.ReferenciaMotivo, true
	}
	return s.motivoAutorizacionParaRuta(ruta)
}

func configurarAutoridadResolucionManualDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, desde time.Time) error {
	vinculo, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	concesion := func(accion, modulo, tipo string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{Accion: accion, ModuloID: modulo, TipoRecurso: tipo,
			Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}
	}
	local, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora(),
		"resolucion_manual_rrhh_desarrollo", "Revisión manual RRHH sintética", "resolucion-manual-rrhh-desarrollo",
		[]dominiovec.ConcesionRol{concesion(postgresct.AccionResolucionManualLlamamiento, "contratacion_temporal", postgresct.TipoRecursoResolucionManualLlamamiento)},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	bolsa, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora(),
		"aceptacion_bolsa_rrhh_desarrollo", "Aceptación RRHH en Bolsa sintética", "aceptacion-bolsa-rrhh-desarrollo",
		[]dominiovec.ConcesionRol{concesion(puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, "bolsa", "integracion_llamamientos_bolsa")},
		[]dominiovec.AmbitoPerfil{{Clave: "categoria_ref", Valores: []string{"categoria:desarrollo:c2"}},
			{Clave: "unidad_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	renuncia, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora(),
		"renuncia_bolsa_rrhh_desarrollo", "Renuncia RRHH en Bolsa sintética", "renuncia-bolsa-rrhh-desarrollo",
		[]dominiovec.ConcesionRol{concesion(puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo, "bolsa", "integracion_llamamientos_bolsa")},
		[]dominiovec.AmbitoPerfil{{Clave: "categoria_ref", Valores: []string{"categoria:desarrollo:c2"}},
			{Clave: "unidad_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	// Son catálogos distintos: el publicador exige una sola identidad de
	// catálogo por llamada y no agrupa permisos de CT con los de Bolsa.
	for _, motivo := range []dominiovec.ReferenciaEntradaCatalogo{motivoResolucionManualDesarrollo(false), motivoResolucionManualDesarrollo(true), motivoRenunciaBolsaDesarrollo()} {
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
			[]dominiovec.ReferenciaEntradaCatalogo{motivo}, desde); err != nil {
			return err
		}
	}
	alta.soporte.mu.Lock()
	alta.soporte.instantaneaResolucionManual, alta.soporte.instantaneaAceptacionBolsa = local, bolsa
	alta.soporte.instantaneaRenunciaBolsa = renuncia
	alta.soporte.mu.Unlock()
	return nil
}
