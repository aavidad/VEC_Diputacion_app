package bootstrap

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// R5 se monta únicamente después de componer rutas, perfiles publicados,
// descriptores de audiencia y el PDP común. Esta pieza no provisiona perfiles.
var errAutorizadoresFirmaR5DesarrolloNoDisponibles = errors.New("contratacion temporal: PARO R5; faltan ruta, perfil, descriptor o autoridad V3")

type exportadorMaterialFirmaR5Desarrollo interface {
	proveerMaterialConfirmacion(context.Context, core.SolicitudAutorizacionLigadaV3, core.DecisionAutorizacionLigadaV3,
		vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, core.ReferenciaEntradaCatalogo,
		core.ResultadoContextoActorRegistradoV2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type operacionAutorizacionFirmaR5Desarrollo struct {
	rutas      []string
	perfil     *perfilFijoCTDesarrollo
	motivo     core.ReferenciaEntradaCatalogo
	exportador exportadorMaterialFirmaR5Desarrollo
}

type configuracionAutorizadoresFirmaR5Desarrollo struct {
	soporte                            *soporteAltaContratacionTemporalDesarrollo
	pdp                                *autorizadorComunDesarrollo
	fronteras                          catalogoFronterasComunDesarrollo
	reloj                              relojContratacionTemporalDesarrollo
	catalogo                           catalogoMaterialAutorizacionComunDesarrollo
	externa, vec, consulta, consultaV2 operacionAutorizacionFirmaR5Desarrollo
}

// El constructor comprueba las dependencias de composición. Si la ruta R5 no
// ha sido incorporada a la sesión y a la instantánea del perfil fijo, detiene
// el montaje antes de atender tráfico. Las concesiones proceden de la
// asignación publicada, nunca del material recibido.
func nuevosAutorizadoresFirmaR5Desarrollo(c configuracionAutorizadoresFirmaR5Desarrollo) (*autorizadoresFirmaR5Desarrollo, error) {
	if c.soporte == nil || c.soporte.sello == nil || c.pdp == nil || c.pdp.servicio == nil ||
		!c.pdp.selector.catalogo.aceptaCatalogoFronteras(c.fronteras) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.reloj) {
		return nil, errAutorizadoresFirmaR5DesarrolloNoDisponibles
	}
	for _, o := range []struct {
		operacion               operacionAutorizacionFirmaR5Desarrollo
		accion, tipo, audiencia string
		campos                  []string
	}{
		{c.externa, ports.AccionRegistrarFirmaExterna, ports.TipoRecursoFirmaExterna, ports.AudienciaFirmaExternaV2, nil},
		{c.vec, ports.AccionRegistrarFirmaVec, ports.TipoRecursoFirmaVec, ports.AudienciaFirmaVecV2, nil},
		{c.consultaV2, ports.AccionConsultarFirmasR5V2, ports.TipoRecursoConsultaFirmasR5, ports.AudienciaConsultaFirmasR5V2, ports.CamposConsultaFirmasR5V2()},
	} {
		d, existe := c.catalogo.descriptorPara(o.audiencia)
		p := o.operacion.perfil
		if !existe || d.Audiencia != o.audiencia || d.ProveedorNominal != proveedorMaterialContratacionTemporal ||
			d.Dominio == "" || d.Prefijo == "" || p == nil || p.plantilla.Validar() != nil ||
			p.contexto.Resultado.Validar() != nil || p.contexto.Vinculo.ValidarPara(p.contexto.Resultado) != nil ||
			p.contextoEsperadoRegistrado.Validar() != nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.sesionOperativa) ||
			p.perfilRef() != p.contexto.Resultado.Contexto.PerfilActivoRef ||
			p.contexto.Resultado.Contexto.Principal.ID != c.soporte.contexto.Resultado.Contexto.Principal.ID ||
			p.contexto.Resultado.Contexto.PersonaRef != c.soporte.contexto.Resultado.Contexto.PersonaRef ||
			!core.ReferenciaMotivoAutorizacionV2Valida(o.operacion.motivo) ||
			dependenciaEsNulaContratacionTemporalDesarrollo(o.operacion.exportador) ||
			!perfilFirmaR5Concede(p, o.accion, o.tipo, o.campos) || len(o.operacion.rutas) == 0 {
			return nil, errAutorizadoresFirmaR5DesarrolloNoDisponibles
		}
		for _, ruta := range o.operacion.rutas {
			if ruta == "" || ruta == httpinterno.RutaFirmaDocumento || ruta == httpinterno.RutaConsultaFirmaDocumento ||
				!rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta) ||
				!rutaSesionOperativaCTDesarrollo(ruta) || !p.atiendeMetodo(ruta, http.MethodPost) ||
				c.soporte.perfilFijoParaRutaYMetodo(ruta, http.MethodPost) != p ||
				!fronteraFirmaR5Compuesta(c.pdp, c.fronteras, ruta, o.accion, p.perfilRef()) ||
				!fuenteFirmaR5Compuesta(c.soporte, p, ruta, o.accion, o.tipo, o.operacion.motivo) {
				return nil, errAutorizadoresFirmaR5DesarrolloNoDisponibles
			}
		}
	}
	return &autorizadoresFirmaR5Desarrollo{soporte: c.soporte, pdp: c.pdp, reloj: c.reloj,
		externa: c.externa, vec: c.vec, consulta: c.consulta, consultaV2: c.consultaV2}, nil
}

func fronteraFirmaR5Compuesta(pdp *autorizadorComunDesarrollo, fronteras catalogoFronterasComunDesarrollo, ruta, accion, perfil string) bool {
	if pdp == nil || !pdp.selector.catalogo.aceptaCatalogoFronteras(fronteras) {
		return false
	}
	f, ok := fronteras.resolver(http.MethodPost, ruta)
	if !ok || f.Superficie != superficieInternaSeguridadComunDesarrollo ||
		f.ClavePolitica != clavePoliticaContratacionTemporalDesarrollo || !f.admitePerfil(perfil) {
		return false
	}
	politica, ok := pdp.selector.catalogo.politicaPara(accion, f.Clave, f.ClavePolitica, f.ClaveCapacidad)
	return ok && politica.valida()
}

// Comprueba en el montaje que la fuente efectiva del PDP reconoce la ruta
// R5 y rechaza tanto otra acción como otra huella. Solo consulta la
// asignación publicada: nunca prepara, publica ni registra una concesión.
func fuenteFirmaR5Compuesta(s *soporteAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo, ruta, accion, tipo string, motivo core.ReferenciaEntradaCatalogo) bool {
	if s == nil || p == nil {
		return false
	}
	recurso := core.RecursoAutorizable{Referencia: "operacion-firma-r5-probe", ModuloID: ports.ModuloContratacion, Tipo: tipo,
		Ambitos:   map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"material_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	d := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: p.contexto.Vinculo, ReferenciaMotivo: motivo,
		Accion: accion, Recurso: recurso, Finalidad: ports.FinalidadFirmaDocumento}
	ctx := context.WithValue(context.Background(), claveSolicitudFirmaR5Desarrollo{},
		solicitudFirmaR5Desarrollo{accion: accion, motivo: motivo, recurso: recurso})
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	i, ok := s.instantaneaPerfilFijoParaContexto(ctx, ruta, p)
	if !ok || i.Validar() != nil || i.AsignacionPerfil.PerfilActivoRef != p.perfilRef() {
		return false
	}
	d.Accion = "contratacion_temporal.documento.firma_r5.otra"
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctx, ruta, p); ok {
		return false
	}
	d.Accion = accion
	d.Recurso.Atributos = map[string]string{"material_sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	_, ok = s.instantaneaPerfilFijoParaContexto(ctx, ruta, p)
	return !ok
}

func perfilFirmaR5Concede(p *perfilFijoCTDesarrollo, accion, tipo string, campos []string) bool {
	if p == nil || len(p.plantilla.AsignacionPerfil.Ambitos) != 1 {
		return false
	}
	ambito := p.plantilla.AsignacionPerfil.Ambitos[0]
	if ambito.Clave != "organizacion_ref" || !slices.Equal(ambito.Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		return false
	}
	for _, concesion := range p.plantilla.VersionRol.Concesiones {
		if concesion.Accion == accion && concesion.ModuloID == ports.ModuloContratacion && concesion.TipoRecurso == tipo &&
			slices.Equal(concesion.Finalidades, []string{ports.FinalidadFirmaDocumento}) &&
			concesion.GarantiaMinima == core.AuthAssuranceHigh && slices.Equal(concesion.CamposPermitidos, campos) &&
			len(concesion.Obligaciones) == 0 {
			return true
		}
	}
	return false
}

type autorizadoresFirmaR5Desarrollo struct {
	soporte                            *soporteAltaContratacionTemporalDesarrollo
	pdp                                vp.AutorizadorSolicitudLigadaV3
	reloj                              relojContratacionTemporalDesarrollo
	externa, vec, consulta, consultaV2 operacionAutorizacionFirmaR5Desarrollo
}

var (
	_ ports.AutorizadorRegistroFirmaExterna   = (*autorizadoresFirmaR5Desarrollo)(nil)
	_ ports.AutorizadorFirmaVec               = (*autorizadoresFirmaR5Desarrollo)(nil)
	_ ports.AutorizadorConsultaFirmasR5       = (*autorizadoresFirmaR5Desarrollo)(nil)
	_ ports.AutorizadorFirmaVerificadaV2      = (*autorizadoresFirmaR5Desarrollo)(nil)
	_ ports.AutorizadorConsultaFirmasR5V2     = (*autorizadoresFirmaR5Desarrollo)(nil)
	_ ports.FuentePerfilActivoOperadorFirmaV2 = (*autorizadoresFirmaR5Desarrollo)(nil)
)

func (a *autorizadoresFirmaR5Desarrollo) AutorizarRegistroFirmaExterna(ctx context.Context, m ports.MaterialFirmaExterna) (ports.CapacidadFirmaExterna, error) {
	if m.Validar() != nil {
		return ports.CapacidadFirmaExterna{}, ports.ErrFirmaDocumentoDenegada
	}
	r, err := ctapp.RecursoFirmaExterna(m)
	if err != nil {
		return ports.CapacidadFirmaExterna{}, ports.ErrFirmaDocumentoDenegada
	}
	material, err := a.autorizar(ctx, a.externa, ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV3, r, "", "")
	if err != nil {
		return ports.CapacidadFirmaExterna{}, err
	}
	c := ports.TransportarMaterialFirmaExterna(material)
	if ctapp.ValidarCapacidadFirmaExterna(c, m) != nil {
		return ports.CapacidadFirmaExterna{}, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

func (a *autorizadoresFirmaR5Desarrollo) AutorizarFirmaVec(ctx context.Context, m ports.MaterialFirmaVec) (ports.CapacidadFirmaVec, error) {
	if m.Validar() != nil {
		return ports.CapacidadFirmaVec{}, ports.ErrFirmaDocumentoDenegada
	}
	r, err := ctapp.RecursoFirmaVec(m)
	if err != nil {
		return ports.CapacidadFirmaVec{}, ports.ErrFirmaDocumentoDenegada
	}
	material, err := a.autorizar(ctx, a.vec, ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV3,
		r, m.FirmantePrincipalRef, m.CertificadoHuella)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	c := ports.TransportarMaterialFirmaVec(material)
	if ctapp.ValidarCapacidadFirmaVec(c, m) != nil {
		return ports.CapacidadFirmaVec{}, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

func (a *autorizadoresFirmaR5Desarrollo) AutorizarConsultaFirmasR5(ctx context.Context, m ports.MaterialConsultaFirmasR5) (ports.CapacidadConsultaFirmasR5, error) {
	r, err := ctapp.RecursoConsultaFirmasR5(m)
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, ports.ErrFirmaDocumentoDenegada
	}
	material, err := a.autorizar(ctx, a.consulta, ports.AccionConsultarFirmasR5, ports.AudienciaConsultaFirmasR5V3, r, "", "")
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, err
	}
	c := ports.TransportarMaterialConsultaFirmasR5(material)
	if ctapp.ValidarCapacidadConsultaFirmasR5(c, m) != nil {
		return ports.CapacidadConsultaFirmasR5{}, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

// La vía externa acredita al firmante del PDF; la vía VEC exige además que
// coincida con la persona y el certificado de la petición sellada.
func (a *autorizadoresFirmaR5Desarrollo) AutorizarFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2) (ports.CapacidadFirmaVerificadaV2, error) {
	var cero ports.CapacidadFirmaVerificadaV2
	if a == nil || m.Validar() != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	r, err := ctapp.RecursoFirmaVerificadaV2(m)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	op, accion, audiencia := a.externa, ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	principal, certificado := "", ""
	if m.Via == ports.ViaFirmaCertificadoVEC {
		op, accion, audiencia = a.vec, ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
		principal, certificado = m.FirmantePrincipalRef, m.CertificadoHuella
	}
	if op.perfil == nil || m.PerfilActivoOperadorRef != op.perfil.perfilRef() {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	material, err := a.autorizar(ctx, op, accion, audiencia, r, principal, certificado)
	if err != nil {
		return cero, err
	}
	c := ports.TransportarMaterialFirmaVerificadaV2(material)
	if err := ctapp.ValidarCapacidadFirmaVerificadaV2(c, m); err != nil {
		return cero, err
	}
	return c, nil
}

func (a *autorizadoresFirmaR5Desarrollo) AutorizarConsultaFirmasR5V2(ctx context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	var cero ports.CapacidadConsultaFirmasR5V2
	if a == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	r, err := ctapp.RecursoConsultaFirmasR5V2(m)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	material, err := a.autorizar(ctx, a.consultaV2, ports.AccionConsultarFirmasR5V2, ports.AudienciaConsultaFirmasR5V2, r, "", "")
	if err != nil {
		return cero, err
	}
	c := ports.TransportarMaterialConsultaFirmasR5V2(material)
	if err := ctapp.ValidarCapacidadConsultaFirmasR5V2(c, m); err != nil {
		return cero, err
	}
	return c, nil
}

func (a *autorizadoresFirmaR5Desarrollo) ObtenerPerfilActivoOperadorFirmaV2(ctx context.Context) (string, error) {
	if a == nil || a.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) || ctx == nil || ctx.Err() != nil {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	if !valida || capacidad.metodo != http.MethodPost {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	ahora := a.reloj.Ahora()
	if capacidad.certificadoVerificadoEn.IsZero() || capacidad.certificadoValidoHasta.IsZero() ||
		capacidad.certificadoVerificadoEn.After(ahora) || !ahora.Before(capacidad.certificadoValidoHasta) {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	p := a.soporte.perfilFijoParaRutaYMetodo(capacidad.ruta, capacidad.metodo)
	if p == nil {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	if _, estado := a.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, p); estado != perfilFijoConsumoVigente {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	operativo, err := a.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || operativo.Resultado.Contexto.PerfilActivoRef != p.perfilRef() ||
		operativo.Resultado.Contexto.Principal.ID != p.contexto.Resultado.Contexto.Principal.ID ||
		operativo.Resultado.Contexto.PersonaRef != p.contexto.Resultado.Contexto.PersonaRef ||
		!operativo.Vinculo.VigenteEn(ahora, operativo.Resultado) {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	return p.perfilRef(), nil
}

func (a *autorizadoresFirmaR5Desarrollo) autorizar(ctx context.Context, o operacionAutorizacionFirmaR5Desarrollo,
	accion, audiencia string, recurso core.RecursoAutorizable, firmantePrincipal, certificado string,
) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	fallo := ports.ErrFirmaDocumentoDenegada
	if a == nil || a.soporte == nil || a.pdp == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) || o.perfil == nil || o.exportador == nil ||
		ctx == nil || ctx.Err() != nil || recurso.Validar() != nil || recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo {
		return vacia, fallo
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	ahoraCanal := a.reloj.Ahora()
	if !valida || capacidad.metodo != http.MethodPost || !slices.Contains(o.rutas, capacidad.ruta) ||
		capacidad.certificadoVerificadoEn.IsZero() || capacidad.certificadoValidoHasta.IsZero() ||
		capacidad.certificadoVerificadoEn.After(ahoraCanal) || !ahoraCanal.Before(capacidad.certificadoValidoHasta) ||
		!o.perfil.atiendeMetodo(capacidad.ruta, capacidad.metodo) ||
		a.soporte.perfilFijoParaRutaYMetodo(capacidad.ruta, capacidad.metodo) != o.perfil {
		return vacia, fallo
	}
	if firmantePrincipal != "" && (firmantePrincipal != o.perfil.contexto.Resultado.Contexto.PersonaRef ||
		certificado == "" || certificado != capacidad.principal.Attributes["certificate_sha256"]) {
		return vacia, fallo
	}
	if _, estado := a.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, o.perfil); estado != perfilFijoConsumoVigente {
		if estado == perfilFijoConsumoFuenteNoDisponible {
			return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return vacia, fallo
	}
	operativo, err := a.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || operativo.Resultado.Contexto.PerfilActivoRef != o.perfil.perfilRef() ||
		operativo.Resultado.Contexto.Principal.ID != o.perfil.contexto.Resultado.Contexto.Principal.ID ||
		operativo.Resultado.Contexto.PersonaRef != o.perfil.contexto.Resultado.Contexto.PersonaRef {
		return vacia, fallo
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, fallo
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo: o.motivo, Accion: accion, Recurso: recurso,
		Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion}
	if !solicitudFirmaR5Exacta(datos, accion, o.motivo, recurso) {
		return vacia, fallo
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, fallo
	}
	ctx = context.WithValue(ctx, claveSolicitudFirmaR5Desarrollo{}, solicitudFirmaR5Desarrollo{accion: accion, motivo: o.motivo, recurso: recurso})
	if !solicitudAutorizacionFirmaR5DesarrolloValida(ctx, datos) {
		return vacia, fallo
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := a.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		return vacia, fallo
	}
	material, err := o.exportador.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, o.motivo, operativo.Resultado)
	if err != nil || material.ValidarEstructura() != nil {
		return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	resumen := material.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if resumen.Operacion() != accion || resumen.EfectoRef() != recurso.Referencia ||
		resumen.AudienciaConsumo() != audiencia || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return vacia, fallo
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.EfectoHuellaSHA256() != huella {
		return vacia, fallo
	}
	return material, nil
}

type claveSolicitudFirmaR5Desarrollo struct{}

type solicitudFirmaR5Desarrollo struct {
	accion  string
	motivo  core.ReferenciaEntradaCatalogo
	recurso core.RecursoAutorizable
}

// La fuente de instantáneas del PDP debe llamar a este predicado para la
// ruta R5. El marcador solo nace en el proveedor tras calcular el canon del
// material de aplicación; una ruta sin él queda denegada.
func solicitudAutorizacionFirmaR5DesarrolloValida(ctx context.Context, d core.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil {
		return false
	}
	esperada, ok := ctx.Value(claveSolicitudFirmaR5Desarrollo{}).(solicitudFirmaR5Desarrollo)
	return ok && esperada.recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		solicitudFirmaR5Exacta(d, esperada.accion, esperada.motivo, esperada.recurso)
}

func solicitudFirmaR5Exacta(d core.DatosSolicitudAutorizacionLigadaV3, accion string,
	motivo core.ReferenciaEntradaCatalogo, recurso core.RecursoAutorizable) bool {
	return d.Accion == accion && d.Finalidad == ports.FinalidadFirmaDocumento && d.ReferenciaMotivo == motivo &&
		d.Recurso.Referencia == recurso.Referencia && d.Recurso.ModuloID == recurso.ModuloID &&
		d.Recurso.Tipo == recurso.Tipo && maps.Equal(d.Recurso.Ambitos, recurso.Ambitos) &&
		maps.Equal(d.Recurso.Atributos, recurso.Atributos)
}
