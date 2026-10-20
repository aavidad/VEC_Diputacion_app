package bootstrap

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La lectura nominal conserva los casos de uso y sesiones PostgreSQL de CT.
// Sólo cambia la fuente F1 del actor, con el perfil consulta_ct de su cuenta.
type montajeLecturasFirmaV2 struct {
	nominal        *fuenteNominalSesionFirmaCT
	cuadro         httpinterno.ConsultorCuadroRRHH
	detalle        httpinterno.ConsultorDetalleRRHH
	historico      httpinterno.ConsultorDetalleRRHH
	descargas      ports.RegistradorDescargaBorradorRRHH
	circuito       http.Handler
	motivoCircuito *pgvec.ValidadorReferenciaMotivoPostgreSQLV2
}

type filtroUnidadSelectoresFirmaCT struct {
	nominal *fuenteNominalSesionFirmaCT
}

func (f filtroUnidadSelectoresFirmaCT) ResolverUnidadSelectoresRecuperacionRRHH(
	ctx context.Context, lector ports.ContextoConsultaRRHH,
) (ports.FiltroUnidadSelectoresRecuperacionRRHH, error) {
	var vacio ports.FiltroUnidadSelectoresRecuperacionRRHH
	if f.nominal == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	firma, err := f.nominal.contexto(ctx)
	if err != nil {
		return vacio, err
	}
	p, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	if !ok || p == nil || p.fuente != f.nominal ||
		!ctdomain.HuellaSHA256FirmaValida(firma.CertificadoCanalSHA256) {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	seleccion, presente := f.nominal.porCertificado[firma.CertificadoCanalSHA256]
	p.mu.Lock()
	mismaPeticion := p.solicitud.CertificadoCanalSHA256 == firma.CertificadoCanalSHA256 &&
		p.solicitud.PersonaEsperadaRef == lector.ActorRef() &&
		presente && seleccion.PerfilConsultaCTRef == lector.PerfilRef() &&
		p.extras[lector.PerfilRef()] != nil
	p.mu.Unlock()
	if !mismaPeticion || firma.Resultado.Contexto.PersonaRef != lector.ActorRef() {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	instantanea, err := f.nominal.asignaciones.ObtenerInstantaneaAutorizacion(
		ctx, lector.ActorRef(), firma.Resultado.Contexto.PerfilActivoRef)
	if err != nil {
		return vacio, err
	}
	if instantanea.Validar() != nil ||
		instantanea.AsignacionPerfil.PrincipalID != lector.ActorRef() ||
		instantanea.AsignacionPerfil.PerfilActivoRef != firma.Resultado.Contexto.PerfilActivoRef ||
		instantanea.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		instantanea.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		!instantanea.AsignacionPerfil.VigenteEn(f.nominal.reloj.Ahora()) ||
		len(instantanea.AsignacionPerfil.Ambitos) != 2 {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	var organizacion, unidad string
	for _, ambito := range instantanea.AsignacionPerfil.Ambitos {
		if len(ambito.Valores) != 1 {
			return vacio, ports.ErrFirmaDocumentoDenegada
		}
		switch ambito.Clave {
		case "organizacion_ref":
			organizacion = ambito.Valores[0]
		case "unidad_ref":
			unidad = ambito.Valores[0]
		default:
			return vacio, ports.ErrFirmaDocumentoDenegada
		}
	}
	if organizacion != lector.OrganizacionRef() || !ctdomain.ReferenciaOpacaValida(unidad) {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	huella, err := instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	filtro := ports.FiltroUnidadSelectoresRecuperacionRRHH{
		UnidadRef: unidad, PerfilAuxRef: instantanea.AsignacionPerfil.PerfilActivoRef,
		VersionRolAuxRef: instantanea.VersionRol.Referencia(),
		AsignacionRef:    instantanea.AsignacionPerfil.Referencia(), AsignacionHuellaSHA256: huella,
	}
	if !filtro.Valido() {
		return vacio, ports.ErrFirmaDocumentoDenegada
	}
	return filtro, nil
}

func rutaLecturaCTFirmaV2(ruta string) bool {
	return ruta == httpinterno.RutaConsultaCuadroRRHH || ruta == httpinterno.RutaConsultaDetalleRRHH ||
		ruta == httpinterno.RutaConsultaCircuitoRRHH
}

func rutaCatalogoCTFirmaV2(r *http.Request) bool {
	return r != nil && r.URL != nil &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		(r.URL.Path == rutaCatalogosAltaContratacionTemporalDesarrollo ||
			r.URL.Path == rutaCircuitoFirmaContratacionTemporalDesarrollo)
}

func nuevasLecturasFirmaV2(ctx context.Context, m *montajeFirmaVecR5CT,
	derivador *derivadorIdentidadOperacionDesarrollo, motivos *pgxpool.Pool) (*montajeLecturasFirmaV2, error) {
	if m == nil || m.nominal == nil || m.consultas.motivos == nil || m.consultas.sesion == nil || m.consultas.sesionOriginal == nil ||
		m.emisorCuadro == nil || m.emisorDetalle == nil || m.consultas.repoDescarga == nil ||
		m.alta.auditoriaLecturasCT == nil || motivos == nil || derivador == nil || ctx == nil {
		return nil, marcarFalloMontajeCT("lecturas_requisitos", errMontajeFirmaVecR5)
	}
	emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(m.consultas.motivos,
		seguridadvec.GeneradorReferenciasCriptograficas{}, m.reloj, m.emisorCuadro, m.emisorDetalle)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_emisor", err)
	}
	cuadro, err := ctapp.NuevoServicioConsultaCuadroRRHH(m.nominal, emisor, m.consultas.sesion, m.reloj)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_cuadro", err)
	}
	cuadro.ConfigurarPlazosFase(m.consultas.plazosFase)
	cuadroNominal := &continuadorCuadroFirmaV2{delegado: cuadro, nominal: m.nominal,
		entradas: make(map[[sha256.Size]byte]continuidadCuadroFirmaV2)}
	// La asignación propia de firma solo reduce el complemento de selectores;
	// la decisión de detalle sigue usando consulta_ct del mismo certificado.
	detalle, err := ctapp.NuevoServicioConsultaDetalleRRHHConSelectores(
		m.nominal, emisor, m.consultas.sesion, m.reloj,
		filtroUnidadSelectoresFirmaCT{nominal: m.nominal},
	)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_detalle", err)
	}
	historico, err := ctapp.NuevoServicioConsultaOriginalPropuestaRRHH(m.nominal, emisor, m.consultas.sesionOriginal, m.reloj)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_historico", err)
	}
	var descriptorDescarga descriptorMaterialConsumidorV3Desarrollo
	for _, descriptor := range descriptoresMaterialAutorizacionContratacionTemporalDesarrollo() {
		if descriptor.Audiencia == ports.AudienciaConsumoDescargaBorradorRRHHV3 {
			descriptorDescarga = descriptor
		}
	}
	if descriptorDescarga.Audiencia == "" {
		return nil, marcarFalloMontajeCT("lecturas_descriptor_descarga", errMontajeFirmaVecR5)
	}
	materialDescarga, err := emisorMaterialFirmaR5Actual(ctx, m.alta, derivador, m.reloj, descriptorDescarga)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_material_descarga", err)
	}
	motivoDetalle, err := m.consultas.motivos.ResolverMotivoDetalleRRHH(ctx, m.reloj.Ahora())
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_motivo_detalle", err)
	}
	denegaciones := auditoriaLecturasCT{resolver: func(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
		return contextoNominalLecturaCT(ctx, m.nominal)
	}, registrador: m.alta.auditoriaLecturasCT,
		configuracion: configuracionAuditoriaLecturasCT{Proceso: m.alta.procesoAuditoriaLecturasCT,
			Canal: string(core.SuperficieAutenticacionInternaCorporativaV1)},
		accion: ports.AccionDescargarBorradorRRHH, motivo: motivoDetalle,
		finalidad: ports.FinalidadDescargarBorradorRRHH}
	if !procesoAuditoriaIntentosConfigurado(denegaciones.configuracion.Proceso) {
		return nil, marcarFalloMontajeCT("lecturas_proceso_auditoria", errMontajeFirmaVecR5)
	}
	descargas := &registradorDescargaFirmaV2{nominal: m.nominal, alta: m.alta,
		material: materialDescarga, repo: m.consultas.repoDescarga, auditoria: denegaciones,
		motivo: motivoDetalle}
	motivoCircuito, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, motivoConsultaCircuitoRRHHDesarrollo().CatalogoID)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_motivo_circuito", err)
	}
	material, err := emisorMaterialFirmaR5Actual(ctx, m.alta, derivador, m.reloj, descriptorMaterialConsultaCircuitoRRHHDesarrollo())
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_material_circuito", err)
	}
	preparador := &preparadorCircuitoFirmaV2{nominal: m.nominal, alta: m.alta, material: material, reloj: m.reloj}
	lector, err := postgresct.NuevoLectorCircuitoRRHHPostgreSQL(m.alta.postgresql.ejecucion)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_lector_circuito", err)
	}
	servicio, err := ctapp.NuevoServicioConsultaCircuitoRRHH(preparador, lector)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_servicio_circuito", err)
	}
	hCircuito, err := httpinterno.NuevoManejadorConsultaCircuitoRRHH(preparador, servicio)
	if err != nil {
		return nil, marcarFalloMontajeCT("lecturas_manejador_circuito", err)
	}
	return &montajeLecturasFirmaV2{nominal: m.nominal, cuadro: cuadroNominal, detalle: detalle,
		descargas: descargas,
		historico: historico, circuito: hCircuito, motivoCircuito: motivoCircuito}, nil
}

type continuidadCuadroFirmaV2 struct {
	certificado, persona, cuenta, perfil string
	canal                                [sha256.Size]byte
	validaHasta                          time.Time
}

type continuadorCuadroFirmaV2 struct {
	delegado httpinterno.ConsultorCuadroRRHH
	nominal  *fuenteNominalSesionFirmaCT
	mu       sync.Mutex
	entradas map[[sha256.Size]byte]continuidadCuadroFirmaV2
}

func (c *continuadorCuadroFirmaV2) Consultar(ctx context.Context, q ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	var vacia ports.PaginaCuadroRRHH
	if c == nil || c.nominal == nil || c.delegado == nil || ctx == nil {
		return vacia, ports.ErrConsultaRRHHNoDisponible
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	e, err := c.nominal.contexto(ctx)
	if !ok || err != nil || capacidad.ruta != httpinterno.RutaConsultaCuadroRRHH ||
		capacidad.vinculoCanalTLS == ([sha256.Size]byte{}) ||
		capacidad.principal.ID != e.Resultado.Contexto.PersonaRef ||
		capacidad.principal.Attributes["certificate_sha256"] != e.CertificadoCanalSHA256 {
		return vacia, ports.ErrAutorizacionDenegada
	}
	s := c.nominal.porCertificado[e.CertificadoCanalSHA256]
	actual := continuidadCuadroFirmaV2{certificado: e.CertificadoCanalSHA256,
		persona: e.Resultado.Contexto.PersonaRef, cuenta: e.Resultado.Contexto.Instantanea.CuentaRef,
		perfil: s.PerfilConsultaCTRef, canal: capacidad.vinculoCanalTLS,
		validaHasta: c.nominal.reloj.Ahora().Add(2 * time.Minute)}
	for _, limite := range []time.Time{e.CertificadoValidoHasta, capacidad.certificadoValidoHasta} {
		if limite.Before(actual.validaHasta) {
			actual.validaHasta = limite
		}
	}
	if !actual.validaHasta.After(c.nominal.reloj.Ahora()) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	if cursor := q.Cursor(); cursor != "" {
		clave, valida := huellaCursorSesionRRHHDesarrollo(cursor)
		if !valida {
			return vacia, ports.ErrAutorizacionDenegada
		}
		c.mu.Lock()
		anterior, existe := c.entradas[clave]
		delete(c.entradas, clave)
		c.mu.Unlock()
		if !existe || anterior.certificado != actual.certificado || anterior.persona != actual.persona ||
			anterior.cuenta != actual.cuenta || anterior.perfil != actual.perfil || anterior.canal != actual.canal ||
			!c.nominal.reloj.Ahora().Before(anterior.validaHasta) {
			return vacia, ports.ErrAutorizacionDenegada
		}
	}
	pagina, err := c.delegado.Consultar(ctx, q)
	if err != nil {
		return vacia, err
	}
	if pagina.HayMas && pagina.CursorSiguiente != "" {
		clave, valida := huellaCursorSesionRRHHDesarrollo(pagina.CursorSiguiente)
		if !valida {
			return vacia, ports.ErrConsultaRRHHNoDisponible
		}
		c.mu.Lock()
		for k, entrada := range c.entradas {
			if !c.nominal.reloj.Ahora().Before(entrada.validaHasta) {
				delete(c.entradas, k)
			}
		}
		if len(c.entradas) >= 256 {
			c.mu.Unlock()
			return vacia, ports.ErrConsultaRRHHNoDisponible
		}
		c.entradas[clave] = actual
		c.mu.Unlock()
	}
	return pagina, nil
}

type consultorCuadroFirmaV2 struct {
	legacy, nominal httpinterno.ConsultorCuadroRRHH
}

func (c consultorCuadroFirmaV2) Consultar(ctx context.Context, q ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	if peticionNominalFirmaV2DesdeContexto(ctx) {
		return c.nominal.Consultar(ctx, q)
	}
	return c.legacy.Consultar(ctx, q)
}

type consultorDetalleFirmaV2 struct {
	legacy, nominal httpinterno.ConsultorDetalleRRHH
}

func (c consultorDetalleFirmaV2) Consultar(ctx context.Context, q ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	if peticionNominalFirmaV2DesdeContexto(ctx) {
		return c.nominal.Consultar(ctx, q)
	}
	return c.legacy.Consultar(ctx, q)
}

func peticionNominalFirmaV2DesdeContexto(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	p, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	return ok && p != nil && p.fuente != nil
}

func contextoNominalLecturaCT(ctx context.Context, f *fuenteNominalSesionFirmaCT) (ports.ContextoAutorizacionAltaV3, error) {
	var vacio ports.ContextoAutorizacionAltaV3
	if f == nil || !peticionNominalFirmaV2DesdeContexto(ctx) || rutaNominalFirmaCT(ctx) != httpinterno.RutaConsultaDetalleRRHH {
		return vacio, ports.ErrAutorizacionDenegada
	}
	if _, err := f.ResolverContextoConsultaRRHH(ctx); err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	return f.contextoPerfil(ctx, "consulta_ct")
}

type registradorDescargaFirmaV2 struct {
	nominal   *fuenteNominalSesionFirmaCT
	alta      *dependenciasAltaContratacionTemporalDesarrollo
	material  *emisorMaterialRenovableCTDesarrollo
	repo      ports.RepositorioDescargaBorradorRRHH
	auditoria auditoriaLecturasCT
	motivo    core.ReferenciaEntradaCatalogo
}

func (r *registradorDescargaFirmaV2) RegistrarFalloDescarga(ctx context.Context, expediente string, causa error) error {
	if r == nil || causa == nil || ctx == nil {
		return ports.ErrDescargaBorradorRRHHSinAnotar
	}
	z, correlacion, err := r.auditoria.capturar(context.WithoutCancel(ctx))
	if err != nil {
		return ports.ErrDescargaBorradorRRHHSinAnotar
	}
	resultado := core.ResultadoIntentoAuditoriaError
	if ctx.Err() == nil && errorDenegacionDescargaBorrador(causa) {
		resultado = core.ResultadoIntentoAuditoriaDenegado
	}
	return r.auditoria.registrar(ctx, z, correlacion, expediente, resultado, causa)
}

func (r *registradorDescargaFirmaV2) RegistrarDescarga(ctx context.Context,
	s ports.SolicitudDescargaBorradorRRHH) (ports.ReciboDescargaBorradorRRHH, error) {
	var vacio ports.ReciboDescargaBorradorRRHH
	if r == nil || r.nominal == nil || r.alta == nil || r.alta.autorizador == nil ||
		r.material == nil || r.material.proveedor == nil || r.repo == nil || ctx == nil {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	z, correlacionRef, err := r.auditoria.capturar(ctx)
	if err != nil {
		return vacio, ports.ErrDescargaBorradorRRHHNoDisponible
	}
	fallo := func(causa error) (ports.ReciboDescargaBorradorRRHH, error) {
		resultado := core.ResultadoIntentoAuditoriaError
		if ctx.Err() == nil && errorDenegacionDescargaBorrador(causa) {
			resultado = core.ResultadoIntentoAuditoriaDenegado
		}
		return vacio, r.auditoria.registrar(ctx, z, correlacionRef, s.ExpedienteRef, resultado, causa)
	}
	if s.Validar() != nil {
		return fallo(ports.ErrDescargaBorradorRRHHInvalida)
	}
	org, err := r.nominal.ResolverOrganizacionFirmaVec(ctx)
	if err != nil {
		return fallo(ports.ErrAutorizacionDenegada)
	}
	alcance := ports.AlcanceDescargaBorradorRRHH{OrganizacionRef: org,
		ClaseAmbito: ports.AmbitoOrganizacionRRHH, AmbitoRef: org}
	recurso, err := ports.RecursoDescargaBorradorRRHH(s, alcance)
	if err != nil {
		return fallo(ports.ErrDescargaBorradorRRHHInvalida)
	}
	correlacion, err := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return fallo(ports.ErrDescargaBorradorRRHHNoDisponible)
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: z.Vinculo,
		ReferenciaMotivo: r.motivo, Accion: ports.AccionDescargarBorradorRRHH,
		Recurso: recurso, Finalidad: ports.FinalidadDescargarBorradorRRHH, Correlacion: correlacion}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return fallo(ports.ErrDescargaBorradorRRHHNoDisponible)
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := r.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, z.Resultado)
	if err != nil {
		if errorDenegacionDescargaBorrador(err) {
			return fallo(ports.ErrAutorizacionDenegada)
		}
		return fallo(ports.ErrDescargaBorradorRRHHNoDisponible)
	}
	material, err := r.material.proveedor.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, r.motivo, z.Resultado)
	if err != nil || material.ValidarEstructura() != nil {
		return fallo(ports.ErrDescargaBorradorRRHHNoDisponible)
	}
	resumen := material.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.Operacion() != ports.AccionDescargarBorradorRRHH ||
		resumen.AudienciaConsumo() != ports.AudienciaConsumoDescargaBorradorRRHHV3 ||
		resumen.EfectoRef() != s.ExpedienteRef || resumen.EfectoHuellaSHA256() != huella {
		return fallo(ports.ErrDescargaBorradorRRHHNoDisponible)
	}
	recibo, err := r.repo.RegistrarDescargaBorrador(ctx, alcance, s, material)
	if err != nil {
		return fallo(err)
	}
	return recibo, nil
}

type selectorDescargaFirmaV2 struct {
	legacy, nominal ports.RegistradorDescargaBorradorRRHH
}

func (s selectorDescargaFirmaV2) RegistrarDescarga(ctx context.Context, q ports.SolicitudDescargaBorradorRRHH) (ports.ReciboDescargaBorradorRRHH, error) {
	if peticionNominalFirmaV2DesdeContexto(ctx) {
		return s.nominal.RegistrarDescarga(ctx, q)
	}
	return s.legacy.RegistrarDescarga(ctx, q)
}
func (s selectorDescargaFirmaV2) RegistrarFalloDescarga(ctx context.Context, expediente string, causa error) error {
	if peticionNominalFirmaV2DesdeContexto(ctx) {
		return s.nominal.RegistrarFalloDescarga(ctx, expediente, causa)
	}
	return s.legacy.RegistrarFalloDescarga(ctx, expediente, causa)
}

type preparadorCircuitoFirmaV2 struct {
	nominal  *fuenteNominalSesionFirmaCT
	alta     *dependenciasAltaContratacionTemporalDesarrollo
	material *emisorMaterialRenovableCTDesarrollo
	reloj    relojContratacionTemporalDesarrollo
}

func (p *preparadorCircuitoFirmaV2) contexto(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	var vacio ports.ContextoAutorizacionAltaV3
	if p == nil || p.nominal == nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	v, err := p.nominal.contextoPerfil(ctx, "circuito")
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	org, err := p.nominal.ResolverOrganizacionFirmaVec(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	snapshot, err := p.nominal.asignaciones.ObtenerInstantaneaAutorizacion(ctx,
		v.Resultado.Contexto.PersonaRef, v.Resultado.Contexto.PerfilActivoRef)
	if err != nil || snapshot.Validar() != nil || len(snapshot.AsignacionPerfil.Ambitos) != 1 ||
		snapshot.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		len(snapshot.AsignacionPerfil.Ambitos[0].Valores) != 1 ||
		snapshot.AsignacionPerfil.Ambitos[0].Valores[0] != org {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	return v, nil
}

func (p *preparadorCircuitoFirmaV2) ResolverContextoCanalCircuitoRRHH(ctx context.Context) (httpinterno.ContextoCanalCircuitoRRHH, error) {
	var vacio httpinterno.ContextoCanalCircuitoRRHH
	if p == nil || p.nominal == nil || rutaNominalFirmaCT(ctx) != httpinterno.RutaConsultaCircuitoRRHH {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	v, err := p.contexto(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	d, err := v.Vinculo.Datos()
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	org, err := p.nominal.ResolverOrganizacionFirmaVec(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	return httpinterno.ContextoCanalCircuitoRRHH{AutenticacionRef: d.AutenticacionRef,
		SesionRef: d.SesionRef, PerfilRef: d.PerfilActivoRef, OrganizacionRef: org}, nil
}

func (p *preparadorCircuitoFirmaV2) PrepararConsultaCircuitoRRHH(ctx context.Context,
	s ports.SolicitudConsultaCircuitoRRHH) (ports.MaterialConsultaCircuitoRRHH, error) {
	var vacio ports.MaterialConsultaCircuitoRRHH
	if p == nil || p.alta == nil || p.alta.autorizador == nil || p.material == nil ||
		p.material.proveedor == nil || s.Validar() != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	canal, err := p.ResolverContextoCanalCircuitoRRHH(ctx)
	if err != nil || canal.AutenticacionRef != s.AutenticacionRef || canal.SesionRef != s.SesionRef ||
		canal.PerfilRef != s.PerfilRef || canal.OrganizacionRef != s.OrganizacionRef {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	operativo, err := p.contexto(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	recurso, err := postgresct.RecursoConsultaCircuitoRRHH(s)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHInvalida
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo: motivoConsultaCircuitoRRHHDesarrollo(), Accion: postgresct.AccionConsultaCircuitoRRHH,
		Recurso: recurso, Finalidad: "gestionar_contratacion_temporal", Correlacion: correlacion}
	ctx = context.WithValue(ctx, claveConsultaCircuitoRRHHDesarrollo{}, s)
	if !solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	d, confirmacion, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		if errors.Is(err, core.ErrAutorizacionDenegada) {
			return vacio, ports.ErrConsultaCircuitoRRHHDenegada
		}
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	autorizacion, err := p.material.proveedor.proveerMaterialConfirmacion(ctx, solicitud, d,
		confirmacion, motivoConsultaCircuitoRRHHDesarrollo(), operativo.Resultado)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	resultado := ports.MaterialConsultaCircuitoRRHH{Solicitud: s, ActorRef: vinculo.PrincipalID,
		PerfilRef: vinculo.PerfilActivoRef, Autorizacion: autorizacion}
	if resultado.ValidarPara(s) != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	return resultado, nil
}

func (m *montajeLecturasFirmaV2) envolverCircuito(legacy http.Handler) (http.Handler, error) {
	if m == nil || m.nominal == nil || m.circuito == nil || legacy == nil {
		return nil, errMontajeFirmaVecR5
	}
	nominal := m.nominal.envolver(m.circuito)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, huella, ok := certificadoSesionFirmanteV2(r, m.nominal.reloj.Ahora()); ok {
			if _, elegido := m.nominal.porCertificado[huella]; elegido {
				nominal.ServeHTTP(w, r)
				return
			}
		}
		legacy.ServeHTTP(w, r)
	}), nil
}

// Sólo estas rutas de lectura reciben la sesión nominal. El handler HTTP,
// formato, consulta y auditoría siguen siendo los ya registrados por CT.
func (m *montajeLecturasFirmaV2) envolverRutas(rutas []vechttp.RutaExacta) error {
	if m == nil || m.nominal == nil {
		return errMontajeFirmaVecR5
	}
	encontradas := map[string]bool{}
	for i := range rutas {
		ruta := rutas[i].Ruta
		if ruta != httpinterno.RutaConsultaCuadroRRHH && ruta != httpinterno.RutaConsultaDetalleRRHH {
			continue
		}
		if rutas[i].Manejador == nil || encontradas[ruta] {
			return errMontajeFirmaVecR5
		}
		encontradas[ruta] = true
		legacy := rutas[i].Manejador
		nominal := m.nominal.envolver(legacy)
		rutas[i].Manejador = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, huella, ok := certificadoSesionFirmanteV2(r, m.nominal.reloj.Ahora()); ok {
				if _, elegido := m.nominal.porCertificado[huella]; elegido {
					nominal.ServeHTTP(w, r)
					return
				}
			}
			legacy.ServeHTTP(w, r)
		})
	}
	if !encontradas[httpinterno.RutaConsultaCuadroRRHH] || !encontradas[httpinterno.RutaConsultaDetalleRRHH] {
		return errMontajeFirmaVecR5
	}
	return nil
}
