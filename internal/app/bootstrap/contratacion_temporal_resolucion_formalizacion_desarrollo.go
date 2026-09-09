package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveMaterialResolucionFormalizacionDesarrollo struct{}
type ejecutorResolucionFormalizacionDesarrollo struct {
	soporte      *soporteAltaContratacionTemporalDesarrollo
	detalle      httpinterno.ConsultorDetalleRRHH
	preparacion  ports.ConsultorPreparacionResolucionFormalizacion
	renderizador ports.RenderizadorBorradorRRHH
	autorizador  *autorizadorLlamamientoDesarrollo
	servicio     ports.TransaccionResolucionFormalizacion
	reloj        ports.Reloj
}

func nuevasDependenciasResolucionFormalizacionDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo,
	reloj ports.Reloj, detalle httpinterno.ConsultorDetalleRRHH, renderizador ports.RenderizadorBorradorRRHH,
	preparacion ports.ConsultorPreparacionResolucionFormalizacion) (*ejecutorResolucionFormalizacionDesarrollo, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || dependenciaEsNulaContratacionTemporalDesarrollo(detalle) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(renderizador) || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(preparacion) {
		return nil, ports.ErrResolucionFormalizacionNoDisponible
	}
	e := &ejecutorResolucionFormalizacionDesarrollo{soporte: alta.soporte, detalle: detalle, preparacion: preparacion, renderizador: renderizador, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterial, resolucionFormalizacion: true}}
	r, err := postgresct.NuevoRegistroResolucionFormalizacionPostgreSQL(alta.postgresql.ejecucion, e)
	if err != nil {
		return nil, err
	}
	e.servicio, err = application.NuevoServicioResolucionFormalizacion(r)
	if err != nil {
		return nil, err
	}
	return e, nil
}
func (e *ejecutorResolucionFormalizacionDesarrollo) ResolverContextoResolucionFormalizacion(ctx context.Context) error {
	if e == nil || e.soporte == nil || ctx == nil || ctx.Err() != nil {
		return ports.ErrResolucionFormalizacionDenegada
	}
	c, ok := e.soporte.capacidadValida(ctx)
	if !ok || c.ruta != httpinterno.RutaResolucionFormalizacion {
		return ports.ErrResolucionFormalizacionDenegada
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.reloj) {
		return ports.ErrResolucionFormalizacionNoDisponible
	}
	if _, _, ok = ventanaAutoridadSinteticaContratacionTemporalDesarrollo(e.reloj.Ahora()); !ok {
		return ports.ErrResolucionFormalizacionDenegada
	}
	return nil
}
func (e *ejecutorResolucionFormalizacionDesarrollo) RegistrarResolucionFormalizacion(ctx context.Context, s ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
	z := ports.ResultadoResolucionFormalizacion{}
	if err := e.ResolverContextoResolucionFormalizacion(ctx); err != nil {
		return z, err
	}
	if s.Validar() != nil {
		return z, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.detalle) || dependenciaEsNulaContratacionTemporalDesarrollo(e.renderizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	c, _ := e.soporte.capacidadValida(ctx)
	// Subconsulta nominal con el mismo canal/actor, sin inventar autorización
	// de lectura: el servicio existente emite y consume su propia V3.
	c.ruta = httpinterno.RutaConsultaDetalleRRHH
	c.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
	lectura := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(s.ExpedienteRef, 7)
	if err != nil {
		return z, err
	}
	d, err := e.detalle.Consultar(lectura, solicitud)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	if d.ValidarContenidoPublicablePara(solicitud) != nil || len(d.Hitos) != 7 ||
		d.Hitos[6].AccionClave != "registrar_propuesta_formalizacion" {
		return z, ports.ErrResolucionFormalizacionEnConflicto
	}
	pdf, err := e.renderizador.RenderizarBorrador(ctx, ports.BorradorResolucion, d.Clonar())
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil || len(pdf) > httpinterno.MaximoPDFBorradorRRHHBytes || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	h := sha256.Sum256(pdf)
	m := ports.MaterialResolucionFormalizacion{Solicitud: s, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoVersion: 7, DocumentoSHA256: hex.EncodeToString(h[:]),
		PropuestaConfirmadaEn: d.Hitos[6].RealizadaEn}
	if m.Validar() != nil {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	ctx = context.WithValue(ctx, claveMaterialResolucionFormalizacionDesarrollo{}, m)
	return e.servicio.RegistrarResolucionFormalizacion(ctx, s)
}
func (e *ejecutorResolucionFormalizacionDesarrollo) ConsultarPreparacionResolucionFormalizacion(ctx context.Context, expediente string) (ports.PreparacionResolucionFormalizacion, error) {
	z := ports.PreparacionResolucionFormalizacion{}
	if err := e.ResolverContextoResolucionFormalizacion(ctx); err != nil {
		return z, err
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.preparacion) {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	c, _ := e.soporte.capacidadValida(ctx)
	c.ruta = httpinterno.RutaConsultaDetalleRRHH
	c.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
	lectura := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	p, err := e.preparacion.ConsultarPreparacionResolucionFormalizacion(lectura, expediente)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if errors.Is(err, application.ErrConsultaRRHHNoObservable) || errors.Is(err, ports.ErrAutorizacionDenegada) {
		return z, ports.ErrResolucionFormalizacionDenegada
	}
	if err != nil {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	if p.ValidarPara(expediente) != nil {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	return p.Clonar(), nil
}

func (e *ejecutorResolucionFormalizacionDesarrollo) PrepararResolucionFormalizacion(ctx context.Context, s ports.SolicitudResolucionFormalizacion) (ports.MaterialResolucionFormalizacion, error) {
	if err := e.ResolverContextoResolucionFormalizacion(ctx); err != nil {
		return ports.MaterialResolucionFormalizacion{}, err
	}
	m, ok := ctx.Value(claveMaterialResolucionFormalizacionDesarrollo{}).(ports.MaterialResolucionFormalizacion)
	if !ok || m.Validar() != nil || m.Solicitud != s || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return ports.MaterialResolucionFormalizacion{}, ports.ErrResolucionFormalizacionDenegada
	}
	return m, nil
}
func (e *ejecutorResolucionFormalizacionDesarrollo) AutorizarResolucionFormalizacion(ctx context.Context, m ports.MaterialResolucionFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	z := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	p, err := e.PrepararResolucionFormalizacion(ctx, m.Solicitud)
	if err != nil {
		return z, err
	}
	if p != m || e.autorizador == nil {
		return z, ports.ErrResolucionFormalizacionDenegada
	}
	r, err := postgresct.RecursoResolucionFormalizacion(m)
	if err != nil {
		return z, err
	}
	return e.autorizador.AutorizarOperacion(ctx, postgresct.AccionResolucionFormalizacion, r)
}
func motivoResolucionFormalizacionDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_resolucion_formalizacion_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("resolucion-formalizacion-manual-ejercicio-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "resolucion-formalizacion-ct")}
}
func solicitudAutorizacionResolucionFormalizacionValida(ctx context.Context, d dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil {
		return false
	}
	m, ok := ctx.Value(claveMaterialResolucionFormalizacionDesarrollo{}).(ports.MaterialResolucionFormalizacion)
	if !ok || m.Validar() != nil || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		d.Accion != postgresct.AccionResolucionFormalizacion || d.Finalidad != "gestionar_contratacion_temporal" ||
		d.ReferenciaMotivo != motivoResolucionFormalizacionDesarrollo() {
		return false
	}
	e, err := postgresct.RecursoResolucionFormalizacion(m)
	r := d.Recurso
	return err == nil && r.Referencia == e.Referencia && r.ModuloID == e.ModuloID && r.Tipo == e.Tipo && maps.Equal(r.Ambitos, e.Ambitos) && maps.Equal(r.Atributos, e.Atributos)
}
func configurarAutoridadResolucionFormalizacionDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, desde time.Time) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"resolucion_formalizacion_ct_desarrollo", "Validación manual de ejercicio", "resolucion-formalizacion-manual-ejercicio",
		[]dominiovec.ConcesionRol{{Accion: postgresct.AccionResolucionFormalizacion, ModuloID: "contratacion_temporal",
			TipoRecurso: postgresct.TipoRecursoResolucionFormalizacion, Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	if err = publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{motivoResolucionFormalizacionDesarrollo()}, desde); err != nil {
		return err
	}
	alta.soporte.mu.Lock()
	alta.soporte.instantaneaResolucionFormalizacion = i
	alta.soporte.mu.Unlock()
	return nil
}
