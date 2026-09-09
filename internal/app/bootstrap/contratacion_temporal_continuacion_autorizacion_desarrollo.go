package bootstrap

import (
	"context"
	"maps"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func (a *autorizadorLlamamientoDesarrollo) modoResolucionOContinuacionValido(ruta string) bool {
	if a == nil || (a.resolucionFormalizacion && ruta != httpinterno.RutaResolucionFormalizacion) {
		return false
	}
	cuenta := 0
	for _, activo := range []bool{a.consultaJustificante, a.resolucionManual, a.aceptacionBolsa, a.renunciaBolsa, a.continuacionCT, a.siguienteBolsa, a.propuestaFormalizacion, a.resolucionFormalizacion} {
		if activo {
			cuenta++
		}
	}
	switch ruta {
	case httpinterno.RutaResolucionFormalizacion:
		return cuenta == 1 && a.resolucionFormalizacion && !a.comunicacion && !a.respuestaRecibida
	case httpinterno.RutaResolucionComunicacionLlamamiento:
		return cuenta == 1 && !a.comunicacion && !a.respuestaRecibida && !a.continuacionCT && !a.siguienteBolsa && !a.propuestaFormalizacion
	case httpinterno.RutaContinuacionLlamamiento:
		return cuenta == 1 && !a.comunicacion && !a.respuestaRecibida && !a.resolucionManual && !a.aceptacionBolsa && !a.renunciaBolsa && !a.propuestaFormalizacion
	case httpinterno.RutaPropuestaFormalizacion:
		return cuenta == 1 && a.propuestaFormalizacion && !a.comunicacion && !a.respuestaRecibida
	default:
		return cuenta == 0
	}
}

func motivoContinuacionDesarrollo(bolsa bool) dominiovec.ReferenciaEntradaCatalogo {
	tipo := "continuacion_llamamiento_ct"
	if bolsa {
		tipo = "siguiente_llamamiento_bolsa"
	}
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_" + tipo, CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(tipo + "-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", tipo)}
}

func solicitudAutorizacionContinuacionDesarrolloValida(ctx context.Context, d dominiovec.DatosSolicitudAutorizacionLigadaV3, p preparacionLlamamientoDesarrollo) bool {
	m, existe := ctx.Value(claveMaterialContinuacionDesarrollo{}).(ports.MaterialContinuacionLlamamiento)
	if !existe || m.Validar() != nil ||
		!expedienteComunicacionLlamamientoDesarrolloValido(p.expediente, ports.SolicitudRegistrarComunicacionLlamamiento{
			OrganizacionRef: m.Solicitud.OrganizacionRef, ExpedienteRef: m.Solicitud.ExpedienteRef}) {
		return false
	}
	r := d.Recurso
	igual := func(e dominiovec.RecursoAutorizable, err error) bool {
		return err == nil && r.Referencia == e.Referencia && r.ModuloID == e.ModuloID && r.Tipo == e.Tipo &&
			maps.Equal(r.Ambitos, e.Ambitos) && maps.Equal(r.Atributos, e.Atributos)
	}
	if d.Accion == postgresct.AccionContinuacionLlamamiento {
		if d.ReferenciaMotivo != motivoContinuacionDesarrollo(false) {
			return false
		}
		if m.Etapa == "confirmacion" {
			l, ok := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
			if !ok || l.solicitud != m.Solicitud || l.soloRecuperacion != (p.expediente.VersionActual > 6) ||
				!antecedenteContinuacionDesarrolloValido(ctx, l.antecedente.Resolucion.Solicitud) ||
				l.justificante.ValidarPara(l.antecedente.Resolucion.Solicitud) != nil ||
				m.ReciboBolsa.OperacionRef != operacionSiguienteDesarrollo(m.Solicitud) ||
				m.ReciboBolsa.TerminalOperacionRef != terminalContinuacionDesarrollo(l) ||
				m.ReciboBolsa.LlamamientoRef == l.antecedente.Resolucion.Solicitud.LlamamientoRef {
				return false
			}
		}
		return igual(postgresct.RecursoContinuacionLlamamiento(m))
	}
	l, ok := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
	if !ok || l.solicitud != m.Solicitud || l.soloRecuperacion != (p.expediente.VersionActual > 6) ||
		!antecedenteContinuacionDesarrolloValido(ctx, l.antecedente.Resolucion.Solicitud) ||
		!consultaJustificanteLigadaAlExpedienteDesarrollo(p.expediente, l.antecedente.Resolucion.Solicitud) {
		return false
	}
	if d.Accion == postgresct.AccionConsultaJustificanteRespuestaRecibida {
		s, ok := ctx.Value(claveConsultaJustificanteRespuestaDesarrollo{}).(ports.SolicitudResolverLlamamiento)
		return ok && s == l.antecedente.Resolucion.Solicitud && d.ReferenciaMotivo == motivoConsultaJustificanteRespuestaDesarrollo() &&
			igual(postgresct.RecursoConsultaJustificanteRespuestaRecibida(s))
	}
	return d.Accion == puertosbolsa.AccionAbrirSiguienteLlamamientoDesarrollo && d.ReferenciaMotivo == motivoContinuacionDesarrollo(true) &&
		l.justificante.ValidarPara(l.antecedente.Resolucion.Solicitud) == nil &&
		r.ModuloID == "bolsa" && r.Tipo == "integracion_llamamientos_bolsa" && r.Referencia == operacionSiguienteDesarrollo(l.solicitud) &&
		len(r.Ambitos) == 2 && r.Ambitos["categoria_ref"] == "categoria:desarrollo:c2" && r.Ambitos["unidad_ref"] == unidadCoberturaContratacionTemporalDesarrollo &&
		len(r.Atributos) == 2 && r.Atributos["necesidad_ref"] == l.justificante.Seleccion.Necesidad.Referencia &&
		huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["contenido_sha256"])
}

func configurarAutoridadContinuacionDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, desde time.Time) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	concesion := func(accion, modulo, tipo string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{Accion: accion, ModuloID: modulo, TipoRecurso: tipo,
			Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}
	}
	local, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"continuacion_llamamiento_ct_desarrollo", "Continuación de llamamiento sintética", "continuacion-llamamiento-ct-desarrollo",
		[]dominiovec.ConcesionRol{concesion(postgresct.AccionContinuacionLlamamiento, "contratacion_temporal", postgresct.TipoRecursoContinuacionLlamamiento)},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	bolsa, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"siguiente_llamamiento_bolsa_desarrollo", "Siguiente llamamiento en Bolsa sintética", "siguiente-llamamiento-bolsa-desarrollo",
		[]dominiovec.ConcesionRol{concesion(puertosbolsa.AccionAbrirSiguienteLlamamientoDesarrollo, "bolsa", "integracion_llamamientos_bolsa")},
		[]dominiovec.AmbitoPerfil{{Clave: "categoria_ref", Valores: []string{"categoria:desarrollo:c2"}},
			{Clave: "unidad_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	for _, motivo := range []dominiovec.ReferenciaEntradaCatalogo{motivoContinuacionDesarrollo(false), motivoContinuacionDesarrollo(true)} {
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{motivo}, desde); err != nil {
			return err
		}
	}
	alta.soporte.mu.Lock()
	alta.soporte.instantaneaContinuacionCT, alta.soporte.instantaneaSiguienteBolsa = local, bolsa
	alta.soporte.mu.Unlock()
	return nil
}
