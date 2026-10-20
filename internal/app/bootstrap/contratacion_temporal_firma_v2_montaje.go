package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal/firmavec"
	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecbase "vec-diputacion-granada/internal/vec/application"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

type montajeFirmaVecR5CT struct {
	cfg                config.Config
	configuracion      configuracionFirmasCT
	alta               *dependenciasAltaContratacionTemporalDesarrollo
	consultas          dependenciasConsultasRRHHDesarrollo
	nominal            *fuenteNominalSesionFirmaCT
	fuenteAutorizacion *pgvec.AlmacenAutorizacion
	registro           *pgvec.AlmacenAutorizacion
	motivosFirma       *pgvec.ValidadorReferenciaMotivoPostgreSQLV2
	motivosOriginal    *pgvec.ValidadorReferenciaMotivoPostgreSQLV2
	motivosCustodia    *pgvec.ValidadorReferenciaMotivoPostgreSQLV2
	motivosDetalle     *pgvec.ValidadorReferenciaMotivoPostgreSQLV2
	plan               *plannominal.Fuente
	descriptor         *plannominal.FuenteDescriptorFirmaV2
	seleccion          ports.FuenteSeleccionFirmanteV2
	competencia        *fuenteCompetenciaFirmaVecV2
	consulta           *firmaemisorv2.Emisor
	recuperacion       *firmaemisorv2.Emisor
	firmas             *emisoresFirmaVecPorPerfilCT
	emisorConsulta     *emisorMaterialRenovableCTDesarrollo
	emisorRecuperacion *emisorMaterialRenovableCTDesarrollo
	emisorRegistro     *emisorMaterialRenovableCTDesarrollo
	emisorCuadro       *emisorMaterialRenovableCTDesarrollo
	emisorDetalle      *emisorMaterialRenovableCTDesarrollo
	reloj              relojContratacionTemporalDesarrollo
	cerrarAuditoria    func()
	cierreAuditoria    sync.Once
}

// Un error al montar rutas y el cierre del proceso pueden alcanzar la misma
// limpieza. El cierre ausente no tapa el error original ni provoca un pánico.
func (m *montajeFirmaVecR5CT) cerrarAuditoriaSiExiste() {
	if m != nil && m.cerrarAuditoria != nil {
		m.cierreAuditoria.Do(m.cerrarAuditoria)
	}
}

// Sólo consume cuentas, perfiles y plan ya publicados. Los pools, el PDP y
// la raíz V3 pertenecen al arranque existente; no crea una identidad C4.
func prepararMontajeFirmaVecR5CT(ctx context.Context, cfg config.Config, c configuracionFirmasCT, alta *dependenciasAltaContratacionTemporalDesarrollo,
	consultas dependenciasConsultasRRHHDesarrollo, resolvedor *resolvedorIdentidadDesarrollo, fuente, motivos *pgxpool.Pool, derivador *derivadorIdentidadOperacionDesarrollo) (*montajeFirmaVecR5CT, error) {
	if ctx == nil || alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || fuente == nil || motivos == nil || consultas.identidad == nil || consultas.motivos == nil || derivador == nil {
		return nil, marcarFalloComponenteArranque("ct_firma_vec_dependencias", &falloFuenteConfiguracionArranque{causa: errMontajeFirmaVecR5})
	}
	m := &montajeFirmaVecR5CT{cfg: cfg, configuracion: c, alta: alta, consultas: consultas, reloj: relojContratacionTemporalDesarrollo{}, cerrarAuditoria: func() {}}
	var err error
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(m.reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]core.ReferenciaEntradaCatalogo{motivoFirmaV2CTDesarrollo()}, desde) != nil {
		return nil, marcarFalloComponenteArranque("ct_firma_vec_motivo", errMontajeFirmaVecR5)
	}
	m.fuenteAutorizacion, err = pgvec.NuevoAlmacenAutorizacion(fuente)
	if err != nil {
		return nil, err
	}
	m.registro, err = pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return nil, err
	}
	m.motivosFirma, err = pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, motivoFirmaV2CTDesarrollo().CatalogoID)
	if err != nil {
		return nil, err
	}
	m.motivosOriginal, err = pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, motivoOriginalFirmableCTDesarrollo().CatalogoID)
	if err != nil {
		return nil, err
	}
	m.motivosCustodia, err = pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, motivoFirmaDocumentoCTDesarrollo().CatalogoID)
	if err != nil {
		return nil, err
	}
	motivoDetalle, err := consultas.motivos.ResolverMotivoDetalleRRHH(ctx, m.reloj.Ahora())
	if err != nil {
		return nil, err
	}
	m.motivosDetalle, err = pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, motivoDetalle.CatalogoID)
	if err != nil {
		return nil, err
	}
	retirada, err := time.Parse(time.RFC3339, c.RetiradaEn)
	if err != nil {
		return nil, errMontajeFirmaVecR5
	}
	// La sesión nominal necesita la proyección empleada de Personal. Usa el
	// mismo origen y LOGIN de ContextoActor; las consultas CT conservan su
	// resolvedor de alcance vacío y sus recibos anteriores.
	dsnContexto, err := cfg.ContratacionTemporalPostgreSQL.DSNContextoActorConsultasSeparado()
	if err != nil {
		return nil, errMontajeFirmaVecR5
	}
	poolContextoFirma, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsnContexto,
		"vec-ct-firma-vec-contexto-empleado", rolContextoActorConsultasDesarrollo)
	if err != nil {
		return nil, errMontajeFirmaVecR5
	}
	contextoPreparado := false
	defer func() {
		if !contextoPreparado {
			poolContextoFirma.Close()
		}
	}()
	resolutorFirma, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, poolContextoFirma)
	if err != nil {
		return nil, errMontajeFirmaVecR5
	}
	contextosFirma, err := nuevaAutoridadContextoEmpleadoFirmaVecR5(resolutorFirma, m.reloj)
	if err != nil {
		return nil, errMontajeFirmaVecR5
	}
	autoridad, err := nuevaAutoridadSesionFirmanteV2ConPuertos(resolvedor, consultas.identidad.registro,
		consultas.identidad.revalidador, contextosFirma, m.reloj, retirada, huellaAltaContratacionTemporalDesarrollo("firma-vec-sesion-software-v1"),
		c.PoliticaAutenticacionRef, c.PoliticaAutenticacionSHA256, c.CRLPath)
	if err != nil {
		return nil, err
	}
	m.nominal, err = nuevaFuenteNominalSesionFirmaCT(autoridad, m.fuenteAutorizacion, c.Firmantes, m.reloj, m.alta.soporte.referenciaCT)
	if err != nil {
		return nil, err
	}
	consulta, err := fichero.NuevaConsultaCatalogoProvisionable(c.Plan.Archivo)
	if err != nil {
		return nil, err
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, Metadatos: consulta, CatalogoID: c.Plan.Referencia, ModuloID: ports.ModuloContratacion, Reloj: m.reloj, MunicipioSede: reglas.MunicipioSedeDiputacion})
	if err != nil {
		return nil, err
	}
	publicada, err := pgct.NuevaPublicacionPlanFirmaPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	m.plan, err = plannominal.NuevaFuente(resolutor, publicada, ctdomain.VersionPlanFirmaV2{Referencia: c.Plan.Referencia, Version: c.Plan.Version, HuellaSHA256: c.Plan.HuellaSHA256})
	if err != nil {
		return nil, err
	}
	if _, err = m.plan.Plan(ctx); err != nil {
		return nil, marcarFalloComponenteArranque("ct_firma_vec_plan", &falloFuenteConfiguracionArranque{causa: err})
	}
	seleccion, err := pgct.NuevaSeleccionFirmantePostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	m.seleccion = seleccion
	// AUT56 consulta CA25 y Personal dentro de su fachada autorizada, en una
	// transacción serializable revertida. El selector JSON no liga certificado
	// con persona: esa relación se comprueba antes de abrir cada sesión F1.
	m.nominal.fuente.acreditarCA25 = func(ctx context.Context, huella string, s seleccionNominalFirmaCT) error {
		return acreditarCertificadoUnaVezCT(ctx, huella, s, func() error {
			plan, err := m.plan.Plan(ctx)
			if err != nil {
				return errSesionFirmanteV2Denegada
			}
			rol := ""
			for _, a := range m.configuracion.Admisiones {
				if a.PerfilRef == s.PerfilRef {
					partes := strings.Split(a.RolVersionRef, ":")
					if len(partes) == 3 && partes[0] == "rol" {
						rol = partes[1]
					}
				}
			}
			if rol == "" {
				return errSesionFirmanteV2Denegada
			}
			for _, paso := range plan.Pasos {
				// El perfil del circuito es lógico; ADMIN asigna un prf_
				// físico, que AUT56 coteja por certificado/cargo/rol.
				if paso.RolID != rol {
					continue
				}
				q := ports.SolicitudSeleccionFirmanteV2{CertificadoHuella: huella, CargoRef: paso.CargoRef,
					RolID: paso.RolID, Accion: paso.Accion, TipoRecurso: paso.TipoRecurso,
					Finalidad: paso.Finalidad, OrganizacionRef: paso.OrganizacionRef, UnidadRef: paso.UnidadRef}
				acreditada, e := seleccion.SeleccionarFirmanteV2(ctx, q)
				if e == nil && acreditada.PersonaRef == s.PersonaRef && acreditada.CuentaRef == s.CuentaRef &&
					acreditada.PerfilActivoRef == s.PerfilRef && acreditada.RolID == paso.RolID &&
					acreditada.CargoRef == paso.CargoRef && acreditada.VinculoCertificado.Version > 0 {
					return nil
				}
			}
			return errSesionFirmanteV2Denegada
		})
	}
	selector, err := plannominal.NuevoSelectorCentralFirmanteV2(seleccion, motivoFirmaV2CTDesarrollo())
	if err != nil {
		return nil, err
	}
	m.descriptor, err = plannominal.NuevaFuenteDescriptorFirmaV2(m.plan, selector)
	if err != nil {
		return nil, err
	}
	competencia, err := plannominal.NuevaFuenteCompetenciaFirmantePlanV2(m.plan, seleccion, m.reloj.Ahora)
	if err != nil {
		return nil, err
	}
	m.competencia, err = nuevaFuenteCompetenciaFirmaVecV2(competencia)
	if err != nil {
		return nil, err
	}
	desConsulta, desRecuperacion := descriptoresMaterialFirmasR5V2CTDesarrollo()
	desVec, _ := descriptoresMaterialFirmaV2EscrituraCTDesarrollo()
	m.emisorConsulta, err = emisorMaterialFirmaR5Actual(ctx, alta, derivador, m.reloj, desConsulta)
	if err != nil {
		return nil, err
	}
	m.emisorRecuperacion, err = emisorMaterialFirmaR5Actual(ctx, alta, derivador, m.reloj, desRecuperacion)
	if err != nil {
		return nil, err
	}
	m.emisorRegistro, err = emisorMaterialFirmaR5Actual(ctx, alta, derivador, m.reloj, desVec)
	if err != nil {
		return nil, err
	}
	for _, d := range descriptoresMaterialAutorizacionContratacionTemporalDesarrollo() {
		switch d.Audiencia {
		case ports.AudienciaConsumoConsultaCuadroRRHHV3:
			m.emisorCuadro, err = emisorMaterialFirmaR5Actual(ctx, alta, derivador, m.reloj, d)
		case ports.AudienciaConsumoConsultaDetalleRRHHV3:
			m.emisorDetalle, err = emisorMaterialFirmaR5Actual(ctx, alta, derivador, m.reloj, d)
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	if m.emisorCuadro == nil || m.emisorDetalle == nil || consultas.sesionOriginal == nil || consultas.motivos == nil {
		return nil, errMontajeFirmaVecR5
	}
	m.consulta, err = firmaemisorv2.NuevoEmisorConsultaFirmaVecDesarrollo(m.nominal, m.emisorConsulta, motivoFirmaV2CTDesarrollo(), m.reloj, m.fuenteAutorizacion)
	if err != nil {
		return nil, err
	}
	m.recuperacion, err = firmaemisorv2.NuevoEmisorConsultaFirmaVecDesarrollo(m.nominal, m.emisorRecuperacion, motivoFirmaV2CTDesarrollo(), m.reloj, m.fuenteAutorizacion)
	if err != nil {
		return nil, err
	}
	fuenteVec, err := firmavec.NuevaFuenteNominalFirmaVecV2(autoridad)
	if err != nil {
		return nil, err
	}
	m.firmas = &emisoresFirmaVecPorPerfilCT{fuente: fuenteVec, porPerfil: map[string]*firmaemisorv2.Emisor{}}
	for _, aprobada := range c.Admisiones {
		admision, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{EntornoConfiable: vecapp.EntornoAdmisionFirmaDesarrollo,
			Politica: politicaAdmisionFirmaCT(c, aprobada, retirada), Reloj: m.reloj})
		if err != nil {
			return nil, err
		}
		emisor, err := firmaemisorv2.NuevoEmisorFirmaVecDesarrollo(fuenteVec, m.emisorRegistro, motivoFirmaV2CTDesarrollo(), m.reloj, m.fuenteAutorizacion, admision)
		if err != nil {
			return nil, err
		}
		m.firmas.porPerfil[aprobada.PerfilRef] = emisor
	}
	m.cerrarAuditoria = poolContextoFirma.Close
	contextoPreparado = true
	return m, nil
}

func (m *montajeFirmaVecR5CT) servicioOriginalNominal(render ports.RenderizadorBorradorRRHH,
	original *montajeOriginalFirmaR5Desarrollo) (*vecapp.ServicioOriginalFirmableCT, error) {
	if m == nil || original == nil || original.custodia == nil || m.consultas.sesion == nil || m.consultas.sesionOriginal == nil || render == nil {
		return nil, errMontajeFirmaVecR5
	}
	emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(m.consultas.motivos,
		seguridadvec.GeneradorReferenciasCriptograficas{}, m.reloj, m.emisorCuadro, m.emisorDetalle)
	if err != nil {
		return nil, err
	}
	actual, err := ctapp.NuevoServicioConsultaDetalleRRHH(m.nominal, emisor, m.consultas.sesion, m.reloj)
	if err != nil {
		return nil, err
	}
	historico, err := ctapp.NuevoServicioConsultaDetalleRRHH(m.nominal, emisor, m.consultas.sesionOriginal, m.reloj)
	if err != nil {
		return nil, err
	}
	fuente, err := ctapp.NuevaFuenteOriginalFirmableRRHH(actual, historico, render, original.tipos)
	if err != nil {
		return nil, err
	}
	return vecapp.NuevoServicioOriginalFirmableCT(fuente, original.custodia, original.tipos)
}

func emisorMaterialFirmaR5Actual(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo, d descriptorMaterialConsumidorV3Desarrollo) (*emisorMaterialRenovableCTDesarrollo, error) {
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, err
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	p, err := nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, d)
	if err != nil {
		return nil, err
	}
	return nuevoEmisorMaterialRenovableCTDesarrollo(alta.autorizador, p)
}

func fronterasFirmaVecR5CT(c configuracionFirmasCT) []descriptorFronteraComunDesarrollo {
	var perfiles []string
	var perfilesFirma []string
	for _, s := range c.Firmantes {
		perfiles = append(perfiles, s.PerfilRef, s.PerfilConsultaCTRef, s.PerfilDocumentosRef)
		perfilesFirma = append(perfilesFirma, s.PerfilRef)
	}
	var salida []descriptorFronteraComunDesarrollo
	for _, ruta := range []struct{ ruta, clave, capacidad string }{
		{httpinterno.RutaRegistroFirmaVec, "ct-firma-vec-registrar", ports.AccionRegistrarFirmaVec},
		{httpinterno.RutaPreflightFirmaR5, "ct-firma-vec-preflight", ports.AccionConsultarFirmasR5V2},
		{httpinterno.RutaConsultaFirmasR5V2, "ct-firma-vec-consultar", ports.AccionConsultarFirmasR5V2},
		{httpinterno.RutaRecuperacionFirmasR5V2, "ct-firma-vec-recuperar", ports.AccionRecuperarFirmasR5V2},
		{httpinterno.RutaDescargaFirmaR5V2, "ct-firma-vec-descargar", ports.AccionConsultarFirmasR5V2},
	} {
		perfilesRuta := perfiles
		if ruta.ruta == httpinterno.RutaRecuperacionFirmasR5V2 {
			perfilesRuta = perfilesFirma
		}
		salida = append(salida, descriptorFronteraComunDesarrollo{Clave: ruta.clave, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: ruta.ruta,
			PerfilesActivosRef: perfilesRuta, ClavePolitica: "politica-ct-firma-v2-nominal", ClaveCapacidad: ruta.capacidad})
	}
	return salida
}

func rutaFirmaVecR5CT(ruta string) bool {
	return ruta == httpinterno.RutaRegistroFirmaVec || ruta == httpinterno.RutaPreflightFirmaR5 ||
		ruta == httpinterno.RutaConsultaFirmasR5V2 || ruta == httpinterno.RutaRecuperacionFirmasR5V2 ||
		ruta == httpinterno.RutaDescargaFirmaR5V2
}

func accionesFronteraFirmaVecR5CT(ruta string) []string {
	if ruta == httpinterno.RutaRecuperacionFirmasR5V2 {
		return []string{ports.AccionRecuperarFirmasR5V2}
	}
	if !rutaFirmaVecR5CT(ruta) {
		return nil
	}
	acciones := []string{ports.AccionConsultarFirmasR5V2}
	if ruta == httpinterno.RutaRegistroFirmaVec {
		acciones = append(acciones, ports.AccionRegistrarFirmaVec)
	}
	acciones = append(acciones, docports.AccionDescargar)
	if ruta == httpinterno.RutaRegistroFirmaVec {
		acciones = append(acciones, docports.AccionCustodiarFirmado)
	}
	return acciones
}

func (m *montajeFirmaVecR5CT) autorizaciones() ([]descriptorAutorizacionComunDesarrollo, error) {
	if m == nil || m.fuenteAutorizacion == nil || m.registro == nil || m.motivosFirma == nil || m.motivosOriginal == nil || m.motivosCustodia == nil {
		return nil, errMontajeFirmaVecR5
	}
	politicaFirma, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(m.fuenteAutorizacion, m.registro, m.registro, m.motivosFirma)
	if err != nil {
		return nil, err
	}
	politicaOriginal, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(m.fuenteAutorizacion, m.registro, m.registro, m.motivosOriginal)
	if err != nil {
		return nil, err
	}
	politicaCustodia, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(m.fuenteAutorizacion, m.registro, m.registro, m.motivosCustodia)
	if err != nil {
		return nil, err
	}
	var salida []descriptorAutorizacionComunDesarrollo
	for _, frontera := range fronterasFirmaVecR5CT(m.configuracion) {
		for _, accion := range accionesFronteraFirmaVecR5CT(frontera.Ruta) {
			politica := politicaOriginal
			switch accion {
			case ports.AccionConsultarFirmasR5V2, ports.AccionRegistrarFirmaVec, ports.AccionRecuperarFirmasR5V2:
				politica = politicaFirma
			case docports.AccionCustodiarFirmado:
				politica = politicaCustodia
			}
			salida = append(salida, descriptorAutorizacionComunDesarrollo{Accion: accion,
				ClavePolitica: frontera.ClavePolitica, ClaveCapacidad: frontera.ClaveCapacidad,
				Fronteras: []string{frontera.Clave}, Politica: politica})
		}
	}
	return salida, nil
}

func (m *montajeFirmaVecR5CT) rutas(f *firmaDocumentoCTDesarrollo, original *montajeOriginalFirmaR5Desarrollo) ([]vechttp.RutaExacta, error) {
	if m == nil || m.recuperacion == nil || f == nil || f.servicio == nil || f.circuito == nil || original == nil || original.servicioOriginal == nil || f.verificadorR5 == nil {
		return nil, errMontajeFirmaVecR5
	}
	montado := false
	defer func() {
		if !montado {
			m.cerrarAuditoriaSiExiste()
		}
	}()
	durable, err := pgct.NuevoRegistroFirmasVerificadasPostgreSQL(m.alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	autorizador, err := ctapp.NuevoAutorizadorNominalFirmaV2(m.descriptor, m.firmas)
	if err != nil {
		return nil, err
	}
	baseOriginal, err := ctadapters.NuevaFuenteOriginalFirmaDocumentos(original.servicioOriginal, original.tipos)
	if err != nil {
		return nil, err
	}
	pdfAnterior, err := nuevaFuentePDFFirmaAnteriorCTDesarrollo(original.original, original.servicioDocumentos)
	if err != nil {
		return nil, err
	}
	if err := f.componerFirmasR5(dependenciasFirmaR5Desarrollo{original: baseOriginal, registro: durable,
		descriptorPlan: m.descriptor, emisorPlan: m.firmas, consulta: m.consulta, autorizar: autorizador,
		verificador: f.verificadorR5, pdfAnterior: pdfAnterior, competencia: m.competencia,
		politicaFirmantes: fuenteCircuitoFirmaReglasDesarrollo{resolutor: f.circuito}}); err != nil {
		return nil, err
	}
	preflight, err := ctapp.NuevoServicioPreflightFirmaR5(f.firmaExterna, m.nominal, preparadorViaFirmaVecCT{montaje: m})
	if err != nil {
		return nil, err
	}
	hPre, err := httpinterno.NuevoManejadorPreflightFirmaR5V2(m.nominal, preflight)
	if err != nil {
		return nil, err
	}
	hVec, err := httpinterno.NuevoManejadorRegistroFirmaVec(m.nominal, f.firmaVec)
	if err != nil {
		return nil, err
	}
	interior := hVec
	hVec = m.nominal.envolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		preparada, err := firmavec.PrepararPeticionFirmaVecV2(r)
		if err != nil {
			responderNominalFirmaCT(w, http.StatusForbidden)
			return
		}
		interior.ServeHTTP(w, preparada)
	}))
	ctxAudit, cancelarAudit := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	defer cancelarAudit()
	reservados := []string{m.alta.postgresql.gobierno.Config().ConnConfig.User,
		m.alta.postgresql.registroAutorizacion.Config().ConnConfig.User,
		m.alta.postgresql.ejecucion.Config().ConnConfig.User}
	intentos, proceso, cerrar, err := AbrirRegistradorIntentosAuditoriaDesarrollo(ctxAudit, m.cfg, m.alta.postgresql.ejecucion, reservados)
	if err != nil {
		return nil, err
	}
	m.nominal.proceso = proceso
	hConsulta, err := httpinterno.NuevoManejadorConsultaFirmasR5V2(m.nominal, m.consulta, f.registroR5, intentos, m.nominal)
	if err != nil {
		cerrar()
		return nil, err
	}
	hRecuperacion, err := httpinterno.NuevoManejadorRecuperacionFirmasR5V2(m.nominal, m.recuperacion, durable, intentos, m.nominal)
	if err != nil {
		cerrar()
		return nil, err
	}
	rutaDescarga, err := nuevaRutaDescargaFirmaR5V2CT(m, pdfAnterior, durable, intentos)
	if err != nil {
		cerrar()
		return nil, err
	}
	cerrarContexto := m.cerrarAuditoria
	m.cerrarAuditoria = func() { cerrar(); cerrarContexto() }
	montado = true
	return []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaPreflightFirmaR5, Manejador: m.nominal.envolver(hPre)},
		{Ruta: httpinterno.RutaRegistroFirmaVec, Manejador: hVec},
		{Ruta: httpinterno.RutaConsultaFirmasR5V2, Manejador: m.nominal.envolver(hConsulta)},
		{Ruta: httpinterno.RutaRecuperacionFirmasR5V2, Manejador: m.nominal.envolver(hRecuperacion)},
		rutaDescarga,
	}, nil
}

// El mismo servicio original y Documentos alimentan ambas vías.
type montajeOriginalFirmaR5Desarrollo struct {
	original           *originalFirmableCTDesarrollo
	servicioOriginal   *vecbase.ServicioOriginalFirmableCT
	servicioDocumentos *docapp.Servicio
	tipos              *ctadapters.TiposOriginalFirmableRRHH
	custodia           vp.CustodiaOriginalFirmableCT
}

type emisoresFirmaVecPorPerfilCT struct {
	fuente    *firmavec.FuenteNominalFirmaVecV2
	porPerfil map[string]*firmaemisorv2.Emisor
}

func (e *emisoresFirmaVecPorPerfilCT) actual(ctx context.Context) (*firmaemisorv2.Emisor, error) {
	if e == nil || e.fuente == nil {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	actual, err := e.fuente.RevalidarContextoActorFirmaV2(ctx)
	if err != nil {
		return nil, err
	}
	seleccionado := e.porPerfil[actual.Resultado.Contexto.PerfilActivoRef]
	if seleccionado == nil {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	return seleccionado, nil
}
func (e *emisoresFirmaVecPorPerfilCT) ObtenerPerfilActivoOperadorFirmaV2(ctx context.Context) (string, error) {
	x, err := e.actual(ctx)
	if err != nil {
		return "", err
	}
	return x.ObtenerPerfilActivoOperadorFirmaV2(ctx)
}
func (e *emisoresFirmaVecPorPerfilCT) ObtenerAmbitosOperadorFirmaV2(ctx context.Context) (ports.AmbitosOperadorFirmaV2, error) {
	x, err := e.actual(ctx)
	if err != nil {
		return ports.AmbitosOperadorFirmaV2{}, err
	}
	return x.ObtenerAmbitosOperadorFirmaV2(ctx)
}
func (e *emisoresFirmaVecPorPerfilCT) AutorizarMaterialFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, r core.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	x, err := e.actual(ctx)
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return x.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
}
func (e *emisoresFirmaVecPorPerfilCT) AutorizarMaterialPlanFirmaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, r core.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	x, err := e.actual(ctx)
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return x.AutorizarMaterialPlanFirmaV2(ctx, m, r)
}
