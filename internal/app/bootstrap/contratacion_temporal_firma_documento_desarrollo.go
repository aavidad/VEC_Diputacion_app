package bootstrap

import (
	"context"
	"errors"
	"log"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	consultafirmas "vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Registro de firmas de prueba de los borradores de CT en el perfil de
// desarrollo. Se compone solo con VEC_CT_FIRMA_REGISTRO_ENABLED=true (AD3-85
// y CT118 instaladas) y el circuito de firma de ejemplo. La autorización exige
// un perfil nominal publicado para la persona y el paso exacto del catálogo.
// Sin esa publicación se deniega, incluido el perfil genérico antiguo.
// La verificación la hace GrxFirma como servicio separado; apagado, toda firma se
// rechaza. Ninguna firma registrada tiene eficacia administrativa.

var errFirmaDocumentoCTDesarrolloNoDisponible = errors.New("contratacion temporal: registro de firmas de desarrollo no disponible")

func rutaFirmaDocumentoCTDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaFirmaDocumento || ruta == httpinterno.RutaConsultaFirmaDocumento
}

func firmaDocumentoPerfilFijoCompuesto(s *soporteAltaContratacionTemporalDesarrollo) bool {
	return s != nil && s.perfilFijoParaRuta(httpinterno.RutaFirmaDocumento) != nil
}

// descriptorMaterialFirmaDocumentoCTDesarrollo es el consumidor V3 de AD3-85.
func descriptorMaterialFirmaDocumentoCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: ports.AudienciaFirmaDocumentoV3, Dominio: "vec.ct.firma-documento.desarrollo.capacidad-v3",
		Prefijo: "clave:capacidad:ct-firma-documento:", ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

func descriptorMaterialConsultaFirmasDocumentoCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: ports.AudienciaConsultaFirmasDocumentoV3, Dominio: "vec.ct.firmas-documento.consulta.capacidad-v3",
		Prefijo: "clave:capacidad:ct-firmas-documento-consulta:", ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

// El mismo catálogo motivado de firma cubre la gestión y su consulta; la
// acción y la proyección autorizada se distinguen en concesiones separadas.
func motivoConsultaFirmasDocumentoCTDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return motivoFirmaDocumentoCTDesarrollo()
}

// custodia solo se fija desde la configuración documental validada ANTES de
// publicar la plantilla inicial. Una ampliación posterior exige huella y CAS.
func instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(principalID, perfilRef string, ahora time.Time, custodia ...bool) (dominiovec.InstantaneaAutorizacion, error) {
	concesiones := []dominiovec.ConcesionRol{
		{Accion: ports.AccionFirmarDocumento, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaDocumento,
			Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh},
		{Accion: ports.AccionConsultarFirmasDocumento, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoConsultaFirmasDocumento,
			Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
			CamposPermitidos: consultafirmas.CamposConsultaFirmasDocumento()},
	}
	if len(custodia) > 1 {
		return dominiovec.InstantaneaAutorizacion{}, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	if len(custodia) == 1 && custodia[0] {
		concesiones = append(concesiones, concesionCustodiaFirmadoCTDesarrollo())
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		"firma_documento_ct_desarrollo", "Firma de prueba de borradores CT de desarrollo", "asignacion-firma-documento-ct-desarrollo-no-autoritativa",
		concesiones, []dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

type claveConsultaFirmasDocumentoCTDesarrollo struct{}

func solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil {
		return false
	}
	m, ok := ctx.Value(claveConsultaFirmasDocumentoCTDesarrollo{}).(ports.MaterialConsultaFirmasDocumento)
	esperado, err := consultafirmas.RecursoConsultaFirmasDocumento(m)
	r := datos.Recurso
	return ok && err == nil && m.OrganizacionRef == organizacionAltaContratacionTemporalDesarrollo &&
		datos.Accion == ports.AccionConsultarFirmasDocumento && datos.Finalidad == ports.FinalidadFirmaDocumento &&
		datos.ReferenciaMotivo == motivoConsultaFirmasDocumentoCTDesarrollo() && r.Referencia == esperado.Referencia &&
		r.ModuloID == esperado.ModuloID && r.Tipo == esperado.Tipo && maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
}

func motivoFirmaDocumentoCTDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_firma_documento_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("firma-documento-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "firma-documento-ct"),
	}
}

// claveMaterialFirmaDocumentoCTDesarrollo liga el material ya construido por
// la aplicación con el predicado del PDP; no contiene identidad ni concesión.
type claveMaterialFirmaDocumentoCTDesarrollo struct{}

// solicitudAutorizacionFirmaDocumentoCTDesarrolloValida exige que la
// solicitud V3 describa exactamente el material que se va a registrar.
func solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil {
		return false
	}
	m, ok := ctx.Value(claveMaterialFirmaDocumentoCTDesarrollo{}).(ports.MaterialFirmaDocumento)
	if !ok || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return false
	}
	esperado, err := ctapplication.RecursoFirmaDocumento(m)
	r := datos.Recurso
	return err == nil && datos.Accion == ports.AccionFirmarDocumento && datos.Finalidad == ports.FinalidadFirmaDocumento &&
		datos.ReferenciaMotivo == motivoFirmaDocumentoCTDesarrollo() &&
		r.Referencia == esperado.Referencia && r.ModuloID == esperado.ModuloID && r.Tipo == esperado.Tipo &&
		maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
}

// firmaDocumentoCTDesarrollo reúne autoridad de canal, autorizador V3 y
// registro PostgreSQL; la ruta se compone cuando llega el circuito.
type firmaDocumentoCTDesarrollo struct {
	alta          *dependenciasAltaContratacionTemporalDesarrollo
	circuito      *reglas.Resolutor
	registro      *postgrescontratacion.RegistroFirmasDocumentoPostgreSQL
	lector        ports.LectorFirmasDocumentoAutorizadas
	lectorInterno *lectorFirmasIntervencionCTDesarrollo
	reloj         interface{ Ahora() time.Time }
	// fiscalizacion recibe al componer las rutas la comprobación de la firma
	// que habilita la remisión a Intervención (duda 4).
	fiscalizacion *ctapplication.ServicioFiscalizaciones
	// informeTrasSubsanacion es nil salvo que el catálogo exija informe
	// nuevo tras subsanar: entonces su documento se firma en otra ronda.
	informeTrasSubsanacion ports.FuenteInformeTrasSubsanacion
	// servicio queda al componer las rutas: la custodia en Documentos se le
	// añade después, cuando Documentos ya está compuesto.
	servicio *ctapplication.ServicioFirmaDocumento
	// Se fija después de componer Documentos y antes de servir HTTP. El cliente
	// ya construido consulta esta fuente sólo al observar una respuesta real.
	resultadosFirma func() puertosvec.EmisorResultadosTecnicosConContexto
	// Las vías R5 se preparan juntas sobre el mismo servicio, circuito y
	// custodia. Ninguna de ellas se entrega a una ruta mientras falte su
	// autoridad nominal y transaccional.
	firmaExterna *ctapplication.ServicioFirmaExterna
	firmaVec     *ctapplication.ServicioFirmaVec
	// Registro que reciben ambas vías: siempre el decorador con plan (CT176).
	registroR5 ports.RegistroFirmasVerificadasV2
	// verificadorR5 es el mismo cliente GrxFirma de la vía V1, visto como
	// verificador acumulado de firmas múltiples. Nil sin verificación: entonces
	// R5 no se compone (componerFirmasR5 exige d.verificador).
	verificadorR5 docports.VerificadorFirmasDocumento
	// Se fija únicamente después de que Documentos acepte la custodia. Los
	// constructores R5 la exigen; no consumimos el original antes de tiempo.
	custodiaR5Compuesta bool
}

func (f *firmaDocumentoCTDesarrollo) emisorResultadosFirma() puertosvec.EmisorResultadosTecnicosConContexto {
	if f == nil || f.resultadosFirma == nil {
		return nil
	}
	return f.resultadosFirma()
}

func vincularResultadosFirmaCT(f *firmaDocumentoCTDesarrollo, d *autoridadDocumentosDesarrollo) {
	if f == nil || d == nil {
		return
	}
	f.resultadosFirma = func() puertosvec.EmisorResultadosTecnicosConContexto {
		emisor, ok := d.incidencias.(puertosvec.EmisorResultadosTecnicosConContexto)
		if !ok {
			return nil
		}
		return emisor
	}
}

// registroFirmaV2DurableDesarrollo es lo único que el montaje usa del
// adaptador durable de firmas V2: la consulta y el registro con plan fijado
// (CT176). No incluye el registro directo de CT172, así que esta dependencia
// no puede entregarse a las vías R5 por error: no compilaría.
type registroFirmaV2DurableDesarrollo interface {
	ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2,
		ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error)
	ports.RegistradorFirmaConPlanV2
}

// consultaFirmasV2SinRegistroDirecto adapta la consulta durable al puerto que
// pide el decorador. Su registro directo rechaza siempre: RegistroConPlanV2
// sólo escribe por CT176.
type consultaFirmasV2SinRegistroDirecto struct {
	durable registroFirmaV2DurableDesarrollo
}

func (c consultaFirmasV2SinRegistroDirecto) RegistrarFirmaVerificadaV2(context.Context,
	ports.MaterialFirmaVerificadaV2, ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	return ports.ReciboFirmaDocumento{}, ports.ErrRegistroFirmaDocumentoNoDisponible
}

func (c consultaFirmasV2SinRegistroDirecto) ConsultarFirmasAutorizadasV2(ctx context.Context,
	m ports.MaterialConsultaFirmasR5V2, cap ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	return c.durable.ConsultarFirmasAutorizadasV2(ctx, m, cap)
}

// Ambas vías consumen una sola historia nominal V2, el mismo verificador
// acumulado y la custodia de las revisiones del PDF original. Desde CT176 una
// firma V2 sin plan fijado no se confirma: las vías nunca reciben el registro
// directo de CT172, sólo el decorador que liga el plan publicado.
type dependenciasFirmaR5Desarrollo struct {
	original          ports.FuenteOriginalFirmaAutorizado
	registro          registroFirmaV2DurableDesarrollo
	descriptorPlan    ports.FuenteDescriptorPlanFijadoFirmaV2
	emisorPlan        ports.EmisorMaterialPlanFirmaV2
	consulta          ports.AutorizadorConsultaFirmasR5V2
	autorizar         ports.AutorizadorFirmaVerificadaV2
	verificador       docports.VerificadorFirmasDocumento
	pdfAnterior       ports.FuentePDFFirmaAnterior
	competencia       ports.FuenteCompetenciaFirmante
	politicaFirmantes ports.FuentePoliticaMismaPersonaEnPasos
}

// prepararFirmasR5 construye la base y el registro CT176 sin publicar estado.
// Así un fallo no ocupa el original ni modifica el servicio anterior.
func (f *firmaDocumentoCTDesarrollo) prepararFirmasR5(d dependenciasFirmaR5Desarrollo) (*ctapplication.ServicioFirmaDocumento, ports.RegistroFirmasVerificadasV2, error) {
	if f == nil || f.servicio == nil || !f.custodiaR5Compuesta ||
		f.firmaExterna != nil || f.firmaVec != nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.original) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.descriptorPlan) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.emisorPlan) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.consulta) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.autorizar) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.verificador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.pdfAnterior) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.competencia) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.politicaFirmantes) {
		return nil, nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	registro, err := firmaautorizacionv2.NuevoRegistroConPlanV2(d.descriptorPlan, d.emisorPlan, d.registro,
		consultaFirmasV2SinRegistroDirecto{d.registro})
	if err != nil {
		return nil, nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	base := *f.servicio
	if err := base.ComponerOriginalAutorizado(d.original); err != nil {
		return nil, nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	return &base, registro, nil
}

// componerFirmasR5Externa publica únicamente el registro externo de RRHH.
// La vía Vec requiere su propia autoridad de firmante y se compone aparte.
func (f *firmaDocumentoCTDesarrollo) componerFirmasR5Externa(d dependenciasFirmaR5Desarrollo) error {
	base, registro, err := f.prepararFirmasR5(d)
	if err != nil {
		return err
	}
	externa, err := ctapplication.NuevoServicioFirmaExternaV2(base, d.verificador, registro,
		d.autorizar, d.consulta, d.pdfAnterior, d.competencia, d.politicaFirmantes)
	if err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	f.firmaExterna, f.registroR5 = externa, registro
	return nil
}

// componerFirmasR5 conserva el montaje conjunto para quien ya dispone de
// ambas autoridades. Publica las dos vías sólo después de construirlas.
func (f *firmaDocumentoCTDesarrollo) componerFirmasR5(d dependenciasFirmaR5Desarrollo) error {
	base, registro, err := f.prepararFirmasR5(d)
	if err != nil {
		return err
	}
	externa, err := ctapplication.NuevoServicioFirmaExternaV2(base, d.verificador, registro,
		d.autorizar, d.consulta, d.pdfAnterior, d.competencia, d.politicaFirmantes)
	if err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	vec, err := ctapplication.NuevoServicioFirmaVecV2(base, d.verificador, registro,
		d.autorizar, d.consulta, d.pdfAnterior, d.competencia, d.politicaFirmantes)
	if err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	f.firmaExterna, f.firmaVec, f.registroR5 = externa, vec, registro
	return nil
}

func (f *firmaDocumentoCTDesarrollo) certificadoVigenteFirmaDocumentoCTDesarrollo(
	capacidad capacidadConsultaContratacionTemporalDesarrollo,
) (time.Time, bool) {
	if f == nil || f.reloj == nil {
		return time.Time{}, false
	}
	ahora := f.reloj.Ahora()
	return ahora, ctdomain.InstanteUTCCanonico(ahora) &&
		ctdomain.InstanteUTCCanonico(capacidad.certificadoVerificadoEn) &&
		ctdomain.InstanteUTCCanonico(capacidad.certificadoValidoHasta) &&
		!capacidad.certificadoVerificadoEn.After(ahora) && ahora.Before(capacidad.certificadoValidoHasta)
}

func capacidadFirmaDocumentoCTDesarrolloVigenteEn(emitida, expira, ahora time.Time) bool {
	return !ahora.Before(emitida) && ahora.Before(expira)
}

// El catálogo solo identifica el perfil requerido; la asignación vigente y
// la persona las resuelve la autoridad central. Ningún cargo, prefijo o rol
// genérico se convierte aquí en competencia. Se lee el catálogo de nuevo
// para que un material preparado bajo otra versión no llegue al PDP.
func perfilPasoFirmaDocumentoCTDesarrolloCoincide(m ports.MaterialFirmaDocumento, circuito reglas.CircuitoFirma, perfilRef string) bool {
	if perfilRef == "" || circuito.CatalogoID+":"+strconv.Itoa(circuito.Version) != m.CatalogoRef ||
		circuito.HuellaCatalogo != m.CatalogoHuella {
		return false
	}
	for _, documento := range circuito.Documentos {
		if documento.Documento != m.Documento {
			continue
		}
		if m.PasoOrden < 1 || m.PasoOrden > len(documento.Pasos) {
			return false
		}
		paso := documento.Pasos[m.PasoOrden-1]
		if paso.Orden != m.PasoOrden || paso.Referencia != m.PasoRef {
			return false
		}
		if paso.PerfilRef == perfilRef {
			return true
		}
		for _, alternativo := range paso.PerfilesAlternativos {
			if alternativo == perfilRef {
				return true
			}
		}
		return false
	}
	return false
}

var (
	_ ports.AutorizadorFirmaDocumento          = (*firmaDocumentoCTDesarrollo)(nil)
	_ ports.AutorizadorConsultaFirmasDocumento = (*firmaDocumentoCTDesarrollo)(nil)
	_ httpinterno.AutoridadCanalFirmaDocumento = (*firmaDocumentoCTDesarrollo)(nil)
)

// nuevaFirmaDocumentoCTDesarrollo publica el catálogo de motivos y el rol
// nominal de firma. Devuelve nil sin error cuando el selector está apagado.
func nuevaFirmaDocumentoCTDesarrollo(cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, fiscalizacion *ctapplication.ServicioFiscalizaciones) (*firmaDocumentoCTDesarrollo, error) {
	activo, err := cfg.CTFirmaRegistroDesarrolloActivo()
	if err != nil || !activo {
		return nil, err
	}
	if alta == nil || alta.soporte == nil || alta.autorizador == nil || alta.postgresql.gobierno == nil || fiscalizacion == nil ||
		alta.postgresql.ejecucion == nil || alta.postgresql.proveedorMaterialFirmaDocumento == nil ||
		alta.postgresql.proveedorMaterialConsultaFirmasDocumento == nil || alta.soporte.perfilFijoParaRuta(httpinterno.RutaConsultaFirmaDocumento) == nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	registro, err := postgrescontratacion.NuevoRegistroFirmasDocumentoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	lector, err := postgrescontratacion.NuevoLectorFirmasDocumentoAutorizadasPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{motivoFirmaDocumentoCTDesarrollo()}, desde) != nil {
		log.Print("contratacion temporal: registro de firmas no disponible; etapa=catalogo_motivos")
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	alta.soporte.mu.Lock()
	alta.soporte.motivoFirmaDocumento = motivoFirmaDocumentoCTDesarrollo()
	alta.soporte.mu.Unlock()
	return &firmaDocumentoCTDesarrollo{alta: alta, registro: registro, lector: lector, reloj: reloj, fiscalizacion: fiscalizacion}, nil
}

// ResolverOrganizacionFirmaDocumento valida el canal, sin conceder acceso al
// expediente. El lector nominal exige después su propia decisión V3.
func (f *firmaDocumentoCTDesarrollo) ResolverOrganizacionFirmaDocumento(ctx context.Context) (string, error) {
	if f == nil || f.alta == nil || f.alta.soporte == nil || ctx == nil {
		return "", ports.ErrAutorizacionDenegada
	}
	capacidad, valida := f.alta.soporte.capacidadValida(ctx)
	if !valida || !rutaFirmaDocumentoCTDesarrollo(capacidad.ruta) ||
		capacidad.ruta == httpinterno.RutaConsultaFirmaDocumento && dependenciaEsNulaContratacionTemporalDesarrollo(f.lector) {
		return "", ports.ErrAutorizacionDenegada
	}
	return organizacionAltaContratacionTemporalDesarrollo, nil
}

// AutorizarFirmaDocumento obtiene del PDP común la concesión ligada al
// material exacto y el material V3 de la audiencia de firma.
func (f *firmaDocumentoCTDesarrollo) AutorizarFirmaDocumento(ctx context.Context, m ports.MaterialFirmaDocumento) (ports.CapacidadFirmaDocumento, error) {
	vacia := ports.CapacidadFirmaDocumento{}
	if ctx == nil || f == nil || f.alta == nil || f.alta.soporte == nil || f.alta.autorizador == nil ||
		f.alta.postgresql.proveedorMaterialFirmaDocumento == nil || m.Validar() != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	s := f.alta.soporte
	capacidad, valida := s.capacidadValida(ctx)
	perfil := s.perfilFijoParaContexto(ctx, capacidad.ruta)
	_, certificadoVigente := f.certificadoVigenteFirmaDocumentoCTDesarrollo(capacidad)
	if !valida || capacidad.ruta != httpinterno.RutaFirmaDocumento || perfil == nil ||
		f.circuito == nil || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		!certificadoVigente {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	// La huella del documento firmado debe ser la del certificado verificado
	// en este canal, nunca la enviada por el navegador.
	if m.Resultado == ctdomain.ResultadoFirmaFirmado &&
		(m.CertificadoHuella == "" || m.CertificadoHuella != capacidad.principal.Attributes["certificate_sha256"]) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	// El perfil compartido de la prueba anterior nunca acredita un cargo,
	// aunque el catálogo llegase a contener su referencia opaca.
	if perfil.clave == clavePerfilFijoFirmaCTDesarrollo ||
		perfil.plantilla.VersionRol.RolID == "firma_documento_ct_desarrollo" {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	// La asignación fija se consume sin prepararla ni publicarla. Una
	// revocación, cambio de plantilla o caída de la fuente cierra el paso.
	if _, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil); estado != perfilFijoConsumoVigente {
		if estado == perfilFijoConsumoFuenteNoDisponible {
			return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	circuito, err := f.circuito.CircuitoFirma(ctx)
	if err != nil || !perfilPasoFirmaDocumentoCTDesarrolloCoincide(m, circuito, perfil.perfilRef()) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	recurso, err := ctapplication.RecursoFirmaDocumento(m)
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	// La sesión revalidada debe ser la registrada del perfil de este paso. El
	// identificador del certificado mTLS no es la persona registrada: ya lo
	// cotejan capacidadValida y la huella del certificado.
	if !contextoRegistradoPerfilFijoCTDesarrollo(operativo.Resultado.Contexto, perfil) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	motivo := motivoFirmaDocumentoCTDesarrollo()
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivo, Accion: ports.AccionFirmarDocumento,
		Recurso: recurso, Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion,
	}
	ctx = context.WithValue(ctx, claveMaterialFirmaDocumentoCTDesarrollo{}, m)
	if !solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, datos) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := f.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
			return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	material, err := f.alta.postgresql.proveedorMaterialFirmaDocumento.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, motivo, operativo.Resultado)
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	c := ports.TransportarMaterialFirmaDocumento(material)
	if ctapplication.ValidarCapacidadFirmaDocumento(c, m) != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	r := material.ResumenCapacidad()
	ahoraCapacidad, certificadoVigente := f.certificadoVigenteFirmaDocumentoCTDesarrollo(capacidad)
	if !certificadoVigente || !capacidadFirmaDocumentoCTDesarrolloVigenteEn(r.EmitidaEn(), r.ExpiraEn(), ahoraCapacidad) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

func (f *firmaDocumentoCTDesarrollo) AutorizarConsultaFirmasDocumento(ctx context.Context, m ports.MaterialConsultaFirmasDocumento) (ports.CapacidadConsultaFirmasDocumento, error) {
	vacia := ports.CapacidadConsultaFirmasDocumento{}
	if ctx == nil || f == nil || f.alta == nil || f.alta.soporte == nil || f.alta.autorizador == nil ||
		f.reloj == nil || f.alta.postgresql.proveedorMaterialConsultaFirmasDocumento == nil || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	s := f.alta.soporte
	capacidad, valida := s.capacidadValida(ctx)
	perfil := s.perfilFijoParaContexto(ctx, capacidad.ruta)
	if !valida || !rutaFirmaDocumentoCTDesarrollo(capacidad.ruta) || perfil == nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	_, estadoPerfil := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil)
	if estadoPerfil == perfilFijoConsumoFuenteNoDisponible {
		return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if estadoPerfil != perfilFijoConsumoVigente {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	recurso, err := consultafirmas.RecursoConsultaFirmasDocumento(m)
	if err != nil {
		return vacia, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
			return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	motivo := motivoConsultaFirmasDocumentoCTDesarrollo()
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivo, Accion: ports.AccionConsultarFirmasDocumento,
		Recurso: recurso, Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion,
	}
	ctx = context.WithValue(ctx, claveConsultaFirmasDocumentoCTDesarrollo{}, m)
	if !solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx, datos) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := f.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
			_, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil)
			if estado == perfilFijoConsumoDenegado {
				return vacia, ports.ErrFirmaDocumentoDenegada
			}
			return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		if errors.Is(err, errAutorizacionComunDesarrolloNoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
			return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
		}
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	material, err := f.alta.postgresql.proveedorMaterialConsultaFirmasDocumento.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, motivo, operativo.Resultado)
	if err != nil {
		return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	c := ports.TransportarMaterialConsultaFirmasDocumento(material)
	r := material.ResumenCapacidad()
	ahora := f.reloj.Ahora()
	if consultafirmas.ValidarCapacidadConsultaFirmasDocumento(c, m) != nil || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

// Toda lectura de la historia, incluida la prelectura de una firma, pasa por
// el mismo consumidor nominal. El almacén de escritura conserva su contrato.
type registroFirmasDocumentoNominal struct{ firma *firmaDocumentoCTDesarrollo }

func (r registroFirmasDocumentoNominal) RegistrarFirma(ctx context.Context, m ports.MaterialFirmaDocumento, c ports.CapacidadFirmaDocumento) (ports.ReciboFirmaDocumento, error) {
	if r.firma == nil || r.firma.registro == nil {
		return ports.ReciboFirmaDocumento{}, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return r.firma.registro.RegistrarFirma(ctx, m, c)
}

func (r registroFirmasDocumentoNominal) ConsultarFirmas(ctx context.Context, organizacion, expediente string) ([]ports.FirmaRegistrada, error) {
	if r.firma == nil || dependenciaEsNulaContratacionTemporalDesarrollo(r.firma.lector) {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	if ctx != nil {
		canal, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		if ok && canal.ruta == httpinterno.RutaResultadosFiscalizacion && canal.metodo == http.MethodPost {
			if r.firma.lectorInterno == nil {
				return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
			}
			if !r.firma.lectorInterno.capacidadValida(ctx) {
				return nil, ports.ErrAutorizacionDenegada
			}
			return r.firma.lectorInterno.ConsultarFirmas(ctx, organizacion, expediente)
		}
	}
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacion, ExpedienteRef: expediente}
	c, err := r.firma.AutorizarConsultaFirmasDocumento(ctx, m)
	if err != nil {
		return nil, err
	}
	return r.firma.lector.ConsultarFirmasAutorizadas(ctx, m, c)
}

// La composición llama este hook cuando la identidad nominal y las
// autoridades de sesión están disponibles, antes de servir peticiones.
func (f *firmaDocumentoCTDesarrollo) configurarLecturaIntervencion(ctx context.Context, canal *soporteFiscalizacionContratacionTemporalDesarrollo,
	base *proveedorSesionConsultaRRHHDesarrollo, aprobacion aprobacionProvisionPerfilesRRHHDesarrollo) error {
	if f == nil || f.lectorInterno != nil {
		return ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	lector, err := nuevoLectorFirmasIntervencionCTDesarrollo(ctx, f, canal, base, aprobacion)
	if err != nil {
		return err
	}
	f.lectorInterno = lector
	return nil
}

// fuenteCircuitoFirmaReglasDesarrollo traduce el circuito del catálogo de
// ejemplo al vocabulario del dominio CT.
type fuenteCircuitoFirmaReglasDesarrollo struct{ resolutor *reglas.Resolutor }

func (f fuenteCircuitoFirmaReglasDesarrollo) CircuitoFirma(ctx context.Context) (ctdomain.CircuitoFirma, error) {
	if f.resolutor == nil {
		return ctdomain.CircuitoFirma{}, ctapplication.ErrCircuitoFirmaNoDisponible
	}
	c, err := f.resolutor.CircuitoFirma(ctx)
	if err != nil || c.Version <= 0 {
		return ctdomain.CircuitoFirma{}, ctapplication.ErrCircuitoFirmaNoDisponible
	}
	salida := ctdomain.CircuitoFirma{CatalogoVersion: uint64(c.Version), CatalogoRef: c.CatalogoID + ":" + strconv.Itoa(c.Version),
		HuellaCatalogo: strings.ToLower(c.HuellaCatalogo), Ejemplo: c.PaqueteEjemplo,
		PermiteMismaPersonaEnPasos: c.PermiteMismaPersonaEnPasos}
	for _, d := range c.Documentos {
		doc := ctdomain.CircuitoFirmaDocumento{Documento: d.Documento, Etiqueta: d.Etiqueta}
		for _, p := range d.Pasos {
			doc.Pasos = append(doc.Pasos, ctdomain.PasoCircuitoFirma{Orden: p.Orden, Cargo: p.Cargo, PerfilRef: p.PerfilRef,
				PerfilesAlternativos: append([]string(nil), p.PerfilesAlternativos...),
				Accion:               string(p.Accion), Devolucion: ctdomain.DevolucionPasoFirma(p.Devolucion), Habilita: string(p.Habilita), Referencia: p.Referencia})
		}
		salida.Documentos = append(salida.Documentos, doc)
	}
	return salida, nil
}

// verificadorFirmasR5 conserva el cliente sólo si también verifica firmas
// múltiples. Un verificador motivado que no lo haga deja R5 sin componer.
func verificadorFirmasR5(v docports.VerificadorFirmaMotivado) docports.VerificadorFirmasDocumento {
	if dependenciaEsNulaContratacionTemporalDesarrollo(v) {
		return nil
	}
	multiple, ok := v.(docports.VerificadorFirmasDocumento)
	if !ok || dependenciaEsNulaContratacionTemporalDesarrollo(multiple) {
		return nil
	}
	return multiple
}

// PoliticaMismaPersonaEnPasos lee la bandera del mismo circuito vigente que
// fija el paso. Sólo responde para la versión y huella exactas que pide la
// firma: otra versión, una huella distinta o el catálogo caído nunca permiten
// que la misma persona firme dos pasos.
func (f fuenteCircuitoFirmaReglasDesarrollo) PoliticaMismaPersonaEnPasos(ctx context.Context, ref, huella string) (ports.PoliticaMismaPersonaEnPasos, error) {
	if ctx == nil {
		return ports.PoliticaMismaPersonaEnPasos{}, ctapplication.ErrCircuitoFirmaNoDisponible
	}
	c, err := f.CircuitoFirma(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ports.PoliticaMismaPersonaEnPasos{}, ctx.Err()
		}
		return ports.PoliticaMismaPersonaEnPasos{}, ctapplication.ErrCircuitoFirmaNoDisponible
	}
	p := ports.PoliticaMismaPersonaEnPasos{CatalogoRef: c.CatalogoRef, CatalogoHuella: c.HuellaCatalogo,
		Permite: c.PermiteMismaPersonaEnPasos}
	if err := p.ValidarContra(ref, huella); err != nil {
		return ports.PoliticaMismaPersonaEnPasos{}, err
	}
	return p, nil
}

var _ ports.FuentePoliticaMismaPersonaEnPasos = fuenteCircuitoFirmaReglasDesarrollo{}

// rutas compone las dos rutas exactas. Sin circuito no hay firma.
func (f *firmaDocumentoCTDesarrollo) rutas(cfg config.Config, circuito *reglas.Resolutor) ([]vechttp.RutaExacta, error) {
	if f == nil {
		return nil, nil
	}
	if circuito == nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	// Una composición fallida no conserva el verificador de un intento anterior.
	f.verificadorR5 = nil
	verificador, err := nuevoVerificadorFirmaDocumentos(cfg, f.emisorResultadosFirma)
	if err != nil {
		return nil, err
	}
	if verificador == nil {
		log.Print("contratacion temporal: registro de firmas sin verificacion; toda firma se rechazara")
	}
	servicio, err := ctapplication.NuevoServicioFirmaDocumento(fuenteCircuitoFirmaReglasDesarrollo{resolutor: circuito}, registroFirmasDocumentoNominal{f}, f, verificador)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	if f.informeTrasSubsanacion != nil && servicio.AbrirRondaInformeNuevo(f.informeTrasSubsanacion, f.registro) != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	f.servicio = servicio
	f.verificadorR5 = verificadorFirmasR5(verificador)
	f.custodiaR5Compuesta = false
	h, err := httpinterno.NuevoManejadorFirmaDocumento(f, servicio)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	// Con el registro de firmas compuesto, la remisión a Intervención exige
	// la firma del paso que la habilita según el circuito del catálogo.
	if f.fiscalizacion.ExigirFirmaRemision(servicio) != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	f.circuito = circuito
	return []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaFirmaDocumento, Manejador: h},
		{Ruta: httpinterno.RutaConsultaFirmaDocumento, Manejador: h},
	}, nil
}
