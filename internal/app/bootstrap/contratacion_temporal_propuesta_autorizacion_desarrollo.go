package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorPropuestaFormalizacionDesarrollo struct {
	soporte       *soporteAltaContratacionTemporalDesarrollo
	reloj         ports.Reloj
	publicaciones publicacionesPropuestaDesarrollo
	autorizador   *autorizadorLlamamientoDesarrollo
}

func solicitudPropuestaDesarrolloIgual(a, b ports.SolicitudPropuestaFormalizacion) bool {
	if a.Validar() != nil || b.Validar() != nil {
		return false
	}
	primera, err := json.Marshal(a)
	segunda, err2 := json.Marshal(b)
	return err == nil && err2 == nil && bytes.Equal(primera, segunda)
}

func (p *proveedorPropuestaFormalizacionDesarrollo) PrepararPropuestaFormalizacion(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.MaterialPropuestaFormalizacion, error) {
	if contextoInterfazNulo(ctx) || p == nil || p.soporte == nil || !p.publicaciones.admite(s) {
		return ports.MaterialPropuestaFormalizacion{}, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	l, ok := ctx.Value(clavePropuestaFormalizacionDesarrollo{}).(propuestaFormalizacionLigadaDesarrollo)
	if !ok || l.material.Etapa != "confirmacion" || !solicitudPropuestaDesarrolloIgual(l.material.Solicitud, s) ||
		l.antecedente.ValidarPara(s) != nil || l.material.Validar() != nil {
		return ports.MaterialPropuestaFormalizacion{}, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	if l.material.AceptacionBolsa != nil && l.material.AceptacionBolsa.ValidarPara(l.antecedente) != nil ||
		l.material.AceptacionBolsaNativa != nil && l.material.AceptacionBolsaNativa.ValidarPara(l.antecedente) != nil ||
		l.material.AceptacionDerivada != nil && l.material.AceptacionDerivada.ValidarPara(s) != nil {
		return ports.MaterialPropuestaFormalizacion{}, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	if s.DerivacionRef != "" && !contextoPropuestaDerivadaLigado(ctx, s, l.antecedente.Derivacion) {
		return ports.MaterialPropuestaFormalizacion{}, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	return l.material.Clonar(), nil
}

func (p *proveedorPropuestaFormalizacionDesarrollo) AutorizarPropuestaFormalizacion(ctx context.Context, m ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if contextoInterfazNulo(ctx) || p == nil || p.soporte == nil || p.autorizador == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) || !p.publicaciones.admite(m.Solicitud) || m.Validar() != nil {
		return vacio, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	c, valida := p.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaPropuestaFormalizacion {
		return vacio, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	esperado, ok := ctx.Value(claveMaterialPropuestaFormalizacionDesarrollo{}).(ports.MaterialPropuestaFormalizacion)
	if !ok || esperado.Etapa != m.Etapa || !solicitudPropuestaDesarrolloIgual(esperado.Solicitud, m.Solicitud) ||
		!reflect.DeepEqual(esperado.AceptacionBolsa, m.AceptacionBolsa) ||
		!reflect.DeepEqual(esperado.AceptacionBolsaNativa, m.AceptacionBolsaNativa) ||
		!reflect.DeepEqual(esperado.AceptacionDerivada, m.AceptacionDerivada) {
		return vacio, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	if m.Solicitud.DerivacionRef != "" && !contextoPropuestaDerivadaLigado(ctx, m.Solicitud, m.AceptacionDerivada) {
		return vacio, ports.ErrOperacionPropuestaFormalizacionDenegada
	}
	recurso, err := postgresct.RecursoPropuestaFormalizacion(m)
	if err != nil {
		return vacio, err
	}
	return p.autorizador.AutorizarOperacion(ctx, postgresct.AccionPropuestaFormalizacion, recurso)
}

func motivoPropuestaFormalizacionDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_propuesta_formalizacion_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("propuesta-formalizacion-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "propuesta-formalizacion-ct")}
}

func solicitudAutorizacionPropuestaDesarrolloValida(ctx context.Context, d dominiovec.DatosSolicitudAutorizacionLigadaV3, p preparacionLlamamientoDesarrollo) bool {
	m, ok := ctx.Value(claveMaterialPropuestaFormalizacionDesarrollo{}).(ports.MaterialPropuestaFormalizacion)
	if !ok || m.Validar() != nil || m.Solicitud.DerivacionRef != "" ||
		m.Solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		m.Solicitud.ExpedienteRef != p.expediente.Fiscalizado.Referencia ||
		!expedienteFiscalizadoParaPropuestaDesarrollo(p.expediente, m.Solicitud) ||
		d.Accion != postgresct.AccionPropuestaFormalizacion || d.ReferenciaMotivo != motivoPropuestaFormalizacionDesarrollo() {
		return false
	}
	if m.Etapa == "confirmacion" {
		l, existe := ctx.Value(clavePropuestaFormalizacionDesarrollo{}).(propuestaFormalizacionLigadaDesarrollo)
		if !existe || l.antecedente.ValidarPara(m.Solicitud) != nil || l.material.Validar() != nil ||
			!solicitudPropuestaDesarrolloIgual(l.material.Solicitud, m.Solicitud) ||
			!reflect.DeepEqual(l.material.AceptacionBolsa, m.AceptacionBolsa) ||
			!reflect.DeepEqual(l.material.AceptacionBolsaNativa, m.AceptacionBolsaNativa) ||
			!reflect.DeepEqual(l.material.AceptacionDerivada, m.AceptacionDerivada) {
			return false
		}
		if l.antecedente.Resolucion.Politica != politicaManualDesarrollo() ||
			(m.AceptacionBolsa != nil && m.AceptacionBolsa.ValidarPara(l.antecedente) != nil) ||
			(m.AceptacionBolsaNativa != nil && m.AceptacionBolsaNativa.ValidarPara(l.antecedente) != nil) {
			return false
		}
	}
	e, err := postgresct.RecursoPropuestaFormalizacion(m)
	r := d.Recurso
	return err == nil && r.Referencia == e.Referencia && r.ModuloID == e.ModuloID && r.Tipo == e.Tipo &&
		maps.Equal(r.Ambitos, e.Ambitos) && maps.Equal(r.Atributos, e.Atributos)
}

func contextoPropuestaDerivadaLigado(ctx context.Context, s ports.SolicitudPropuestaFormalizacion,
	e *ports.EvidenciaAceptacionDerivada,
) bool {
	l, ok := ctx.Value(claveDetallePropuestaDerivadaDesarrollo{}).(detallePropuestaDerivadaLigado)
	return ok && solicitudPropuestaDesarrolloIgual(l.solicitud, s) &&
		detalleHijoPropuestaDerivadaValido(l.detalle, s) &&
		(e == nil || antecedenteDerivacionLigadoAlDetallePropuesta(l.detalle, *e, s))
}

func solicitudAutorizacionPropuestaDerivadaDesarrolloValida(ctx context.Context,
	d dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	m, ok := ctx.Value(claveMaterialPropuestaFormalizacionDesarrollo{}).(ports.MaterialPropuestaFormalizacion)
	if !ok || m.Validar() != nil || m.Solicitud.DerivacionRef == "" ||
		m.Solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		!contextoPropuestaDerivadaLigado(ctx, m.Solicitud, m.AceptacionDerivada) ||
		d.Accion != postgresct.AccionPropuestaFormalizacion || d.ReferenciaMotivo != motivoPropuestaFormalizacionDesarrollo() ||
		d.Finalidad != "gestionar_contratacion_temporal" {
		return false
	}
	if m.Etapa == "confirmacion" {
		l, existe := ctx.Value(clavePropuestaFormalizacionDesarrollo{}).(propuestaFormalizacionLigadaDesarrollo)
		if !existe || l.antecedente.ValidarPara(m.Solicitud) != nil || l.antecedente.Derivacion == nil ||
			!contextoPropuestaDerivadaLigado(ctx, m.Solicitud, l.antecedente.Derivacion) ||
			l.material.Validar() != nil || !solicitudPropuestaDesarrolloIgual(l.material.Solicitud, m.Solicitud) ||
			!reflect.DeepEqual(l.material.AceptacionDerivada, m.AceptacionDerivada) {
			return false
		}
	}
	esperado, err := postgresct.RecursoPropuestaFormalizacion(m)
	r := d.Recurso
	return err == nil && r.Referencia == esperado.Referencia && r.ModuloID == esperado.ModuloID &&
		r.Tipo == esperado.Tipo && maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
}

func configurarAutoridadPropuestaFormalizacionDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, desde time.Time) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"propuesta_formalizacion_ct_desarrollo", "Propuesta de nombramiento de desarrollo", "propuesta-formalizacion-ct-desarrollo",
		[]dominiovec.ConcesionRol{{Accion: postgresct.AccionPropuestaFormalizacion, ModuloID: "contratacion_temporal",
			TipoRecurso: postgresct.TipoRecursoPropuestaFormalizacion, Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{motivoPropuestaFormalizacionDesarrollo()}, desde); err != nil {
		return err
	}
	alta.soporte.mu.Lock()
	alta.soporte.instantaneaPropuestaFormalizacion = instantanea
	alta.soporte.mu.Unlock()
	return nil
}
