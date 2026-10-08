package bootstrap

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/config"
	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const envCTFirmaR5ExternaEnabled = "VEC_CT_FIRMA_R5_EXTERNA_ENABLED"

var errFirmaR5ExternaMontajeNoDisponible = errors.New("bootstrap: firma externa R5 no disponible")

// La raíz entrega las fuentes que poseen la identidad de ruta, el PDF original,
// la publicación del plan y las dos audiencias de material. Este montaje no
// crea concesiones ni sustituye el comprobante de ninguna de esas autoridades.
type insumosFirmaR5ExternaCT struct {
	alta             *dependenciasAltaContratacionTemporalDesarrollo
	firma            *firmaDocumentoCTDesarrollo
	documentos       *autoridadDocumentosDesarrollo
	original         *montajeOriginalFirmaR5Desarrollo
	circuito         *reglas.Resolutor
	planResolutor    *reglas.Resolutor
	planVersion      ctdomain.VersionPlanFirmaV2
	publicacionPlan  plannominal.PublicacionAutorizada
	perfil           *perfilFijoCTDesarrollo
	materialConsulta *emisorMaterialRenovableCTDesarrollo
	materialRegistro *emisorMaterialRenovableCTDesarrollo
	verificador      docports.VerificadorFirmasDocumento
	autoridadRutas   autoridadRutasFirmaR5ExternaCT
	reloj            relojContratacionTemporalDesarrollo
	procesoAuditoria string
}

// La autoridad conserva tres decisiones de canal independientes: preparar un
// original, consultar el paso y registrar un PDF ya firmado externamente.
type autoridadRutasFirmaR5ExternaCT interface {
	httpinterno.AutoridadCanalRegistroFirmaVec
	httpinterno.AutoridadContextoCanalCircuitoRRHH
	httpinterno.AutoridadCanalRegistroFirmaExterna
	ports.ResolutorContextoAutorizacionAltaV3
}

type rutasFirmaR5ExternaCT struct {
	rutas []vechttp.RutaExacta
}

// nuevasRutasFirmaR5ExternaCT publica sólo la vía del portafirmas registrada
// por RRHH. El selector apagado no requiere fuentes ni registra rutas. Con el
// selector activo, las tres comprobaciones de preparación vienen de sus
// propietarios y el registro durable escribe con plan en CT185.
func nuevasRutasFirmaR5ExternaCT(cfg config.Config, d insumosFirmaR5ExternaCT,
	custodia comprobadorCustodiaPreparadaR5CTDesarrollo,
	registro comprobadorRegistroConPlanR5CTDesarrollo,
	verificador comprobadorConfiguracionVerificadorR5CTDesarrollo,
) (*rutasFirmaR5ExternaCT, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envCTFirmaR5ExternaEnabled)
	if err != nil || !activo {
		return nil, err
	}
	if d.alta == nil || d.alta.soporte == nil || d.alta.postgresql.ejecucion == nil ||
		d.firma == nil || d.firma.alta != d.alta || d.firma.servicio == nil ||
		!d.firma.custodiaR5Compuesta || d.firma.firmaExterna != nil || d.firma.firmaVec != nil || d.firma.registroR5 != nil ||
		d.documentos == nil || d.documentos.custodia == nil || d.original == nil ||
		d.original.original == nil || d.original.servicioOriginal == nil ||
		d.original.servicioDocumentos == nil || d.original.tipos == nil ||
		d.original.original.politicas != d.documentos.politicas ||
		d.original.original.seudonimizador != d.documentos.seudonimizador ||
		d.circuito == nil || d.firma.circuito != d.circuito ||
		d.planResolutor == nil || !d.planVersion.Valida() ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.publicacionPlan) ||
		d.perfil == nil || d.materialConsulta == nil || d.materialRegistro == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.verificador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.autoridadRutas) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(custodia) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(verificador) || d.procesoAuditoria == "" {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}

	// Los dos puentes leen el mismo original y el mismo almacén documental.
	original, err := ctadapters.NuevaFuenteOriginalFirmaDocumentos(d.original.servicioOriginal, d.original.tipos)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	pdfAnterior, err := nuevaFuentePDFFirmaAnteriorCTDesarrollo(d.original.original, d.original.servicioDocumentos)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	plan, err := plannominal.NuevaFuente(d.planResolutor, d.publicacionPlan, d.planVersion)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	ctxPlan, cancelarPlan := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	_, err = plan.Plan(ctxPlan)
	cancelarPlan()
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	seleccion, err := postgresct.NuevaSeleccionFirmantePostgreSQL(d.alta.postgresql.ejecucion)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	selector, err := plannominal.NuevoSelectorCentralFirmanteV2(seleccion, motivoFirmaV2CTDesarrollo())
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	descriptor, err := plannominal.NuevaFuenteDescriptorFirmaV2(plan, selector)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	competencia, err := plannominal.NuevaFuenteCompetenciaFirmantePlanV2(plan, seleccion, d.reloj.Ahora)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	emisor, registrador, err := nuevoEmisorFirmaExternaV2CTDesarrollo(d.alta.soporte, d.perfil,
		d.materialConsulta, d.materialRegistro, d.reloj, d.procesoAuditoria)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	autorizador, err := ctapplication.NuevoAutorizadorNominalFirmaV2(descriptor, emisor)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	durable, err := postgresct.NuevoRegistroFirmasVerificadasPostgreSQL(d.alta.postgresql.ejecucion)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}

	// El servicio anterior se copia. Ningún fallo posterior puede publicar una
	// vía R5 ni ocupar su original hasta tener los tres manejadores válidos.
	firma := *d.firma
	if err := firma.componerFirmasR5Externa(dependenciasFirmaR5Desarrollo{
		original: original, registro: durable, descriptorPlan: descriptor, emisorPlan: emisor,
		consulta: emisor, autorizar: autorizador, verificador: d.verificador,
		pdfAnterior: pdfAnterior, competencia: competencia,
		politicaFirmantes: fuenteCircuitoFirmaReglasDesarrollo{resolutor: d.circuito},
	}); err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	preparacion, err := nuevaFuentePreparacionExternaR5CTDesarrollo(original, plan, registrador,
		custodia, registro, verificador, d.reloj)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	preflight, err := ctapplication.NuevoServicioPreflightFirmaR5V2Externa(firma.firmaExterna, d.autoridadRutas, preparacion)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	hOriginal, err := httpinterno.NuevoManejadorOriginalFirmableCT(d.autoridadRutas, d.original.servicioOriginal)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	hPreflight, err := httpinterno.NuevoManejadorPreflightFirmaR5V2(d.autoridadRutas, preflight)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	hRegistro, err := httpinterno.NuevoManejadorRegistroFirmaExternaV2(d.autoridadRutas, firma.firmaExterna)
	if err != nil {
		return nil, errFirmaR5ExternaMontajeNoDisponible
	}
	d.firma.firmaExterna, d.firma.registroR5 = firma.firmaExterna, firma.registroR5
	return &rutasFirmaR5ExternaCT{rutas: []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaOriginalFirmableCT, Manejador: hOriginal},
		{Ruta: httpinterno.RutaPreflightFirmaR5, Manejador: hPreflight},
		{Ruta: httpinterno.RutaRegistroFirmaExterna, Manejador: hRegistro},
	}}, nil
}

var (
	_ ports.FuenteOriginalFirmaAutorizado = (*ctadapters.FuenteOriginalFirmaDocumentos)(nil)
	_ ports.FuentePDFFirmaAnterior        = (*fuentePDFFirmaAnteriorCTDesarrollo)(nil)
)
