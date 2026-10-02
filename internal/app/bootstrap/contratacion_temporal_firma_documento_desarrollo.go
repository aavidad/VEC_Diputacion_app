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
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Registro de firmas de prueba de los borradores de CT en el perfil de
// desarrollo. Se compone solo con VEC_CT_FIRMA_REGISTRO_ENABLED=true (AD3-85
// y CT118 instaladas) y el circuito de firma de ejemplo. La autorización es
// un rol nominal propio publicado para la identidad del canal CT: no se
// infiere del cargo del catálogo, cuyo perfil_ref sigue siendo informativo.
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
	registro      *postgrescontratacion.RegistroFirmasDocumentoPostgreSQL
	lector        ports.LectorFirmasDocumentoAutorizadas
	lectorInterno *lectorFirmasIntervencionCTDesarrollo
	reloj         relojContratacionTemporalDesarrollo
	// fiscalizacion recibe al componer las rutas la comprobación de la firma
	// que habilita la remisión a Intervención (duda 4).
	fiscalizacion *ctapplication.ServicioFiscalizaciones
	// informeTrasSubsanacion es nil salvo que el catálogo exija informe
	// nuevo tras subsanar: entonces su documento se firma en otra ronda.
	informeTrasSubsanacion ports.FuenteInformeTrasSubsanacion
	// servicio queda al componer las rutas: la custodia en Documentos se le
	// añade después, cuando Documentos ya está compuesto.
	servicio *ctapplication.ServicioFirmaDocumento
	// Las vías R5 se preparan juntas sobre el mismo servicio, circuito y
	// custodia. Ninguna de ellas se entrega a una ruta mientras falte su
	// autoridad nominal y transaccional.
	firmaExterna *ctapplication.ServicioFirmaExterna
	firmaVec     *ctapplication.ServicioFirmaVec
	// Se fija únicamente después de que Documentos acepte la custodia. Los
	// constructores R5 la exigen; no consumimos el original antes de tiempo.
	custodiaR5Compuesta bool
}

// registroFirmasR5Desarrollo es una sola autoridad de historia: CT170
// registra ambas vías y ofrece la lectura nominal AD159. No se mezclan dos
// registros independientes que pudieran discrepar sobre la cabeza CT.
type registroFirmasR5Desarrollo interface {
	ports.RegistroFirmasExternas
	ports.RegistroFirmasVec
}

type dependenciasFirmaR5Desarrollo struct {
	original          ports.FuenteOriginalFirmaAutorizado
	registro          registroFirmasR5Desarrollo
	consulta          ports.AutorizadorConsultaFirmasR5
	autorizarExterna  ports.AutorizadorRegistroFirmaExterna
	autorizarVec      ports.AutorizadorFirmaVec
	competencia       ports.FuenteCompetenciaFirmante
	politicaFirmantes ports.FuentePoliticaMismaPersonaEnPasos
}

// componerFirmasR5 prepara las dos vías como una unidad. El llamador debe
// aportar el lector autorizado de Documentos, CT170, AD159, AD156, AD157 y
// AUT30 ya compuestos. Hasta entonces no se invoca ni se monta una ruta R5.
func (f *firmaDocumentoCTDesarrollo) componerFirmasR5(d dependenciasFirmaR5Desarrollo) error {
	if f == nil || f.servicio == nil || !f.custodiaR5Compuesta || !f.servicio.VerificacionDisponible() ||
		f.firmaExterna != nil || f.firmaVec != nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.original) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.consulta) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.autorizarExterna) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.autorizarVec) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.competencia) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.politicaFirmantes) {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	if err := f.servicio.ComponerOriginalAutorizado(d.original); err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	externa, err := ctapplication.NuevoServicioFirmaExterna(f.servicio, d.registro, d.consulta, d.autorizarExterna, d.competencia)
	if err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	vec, err := ctapplication.NuevoServicioFirmaVec(f.servicio, d.registro, d.consulta, d.autorizarVec, d.competencia)
	if err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	if externa.ComponerPoliticaMismaPersonaEnPasos(d.politicaFirmantes) != nil ||
		vec.ComponerPoliticaMismaPersonaEnPasos(d.politicaFirmantes) != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	f.firmaExterna, f.firmaVec = externa, vec
	return nil
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
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
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
	if !valida || capacidad.ruta != httpinterno.RutaFirmaDocumento || s.perfilFijoParaContexto(ctx, capacidad.ruta) == nil || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	// Esta rama de desarrollo solo admite una firma de prueba cuyo
	// certificado sea exactamente el ya verificado para el canal actual.
	// La huella se toma del contexto sellado, nunca del cuerpo HTTP.
	if m.Resultado == ctdomain.ResultadoFirmaFirmado &&
		(m.CertificadoHuella == "" || m.CertificadoHuella != capacidad.principal.Attributes["certificate_sha256"]) {
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
	ahora := f.reloj.Ahora()
	r := material.ResumenCapacidad()
	if ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}

func (f *firmaDocumentoCTDesarrollo) AutorizarConsultaFirmasDocumento(ctx context.Context, m ports.MaterialConsultaFirmasDocumento) (ports.CapacidadConsultaFirmasDocumento, error) {
	vacia := ports.CapacidadConsultaFirmasDocumento{}
	if ctx == nil || f == nil || f.alta == nil || f.alta.soporte == nil || f.alta.autorizador == nil ||
		f.alta.postgresql.proveedorMaterialConsultaFirmasDocumento == nil || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
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
	if err != nil {
		return ctdomain.CircuitoFirma{}, ctapplication.ErrCircuitoFirmaNoDisponible
	}
	salida := ctdomain.CircuitoFirma{CatalogoRef: c.CatalogoID + ":" + strconv.Itoa(c.Version),
		HuellaCatalogo: strings.ToLower(c.HuellaCatalogo), Ejemplo: c.PaqueteEjemplo}
	for _, d := range c.Documentos {
		doc := ctdomain.CircuitoFirmaDocumento{Documento: d.Documento, Etiqueta: d.Etiqueta}
		for _, p := range d.Pasos {
			doc.Pasos = append(doc.Pasos, ctdomain.PasoCircuitoFirma{Orden: p.Orden, Cargo: p.Cargo, PerfilRef: p.PerfilRef,
				Accion: string(p.Accion), Devolucion: ctdomain.DevolucionPasoFirma(p.Devolucion), Habilita: string(p.Habilita), Referencia: p.Referencia})
		}
		salida.Documentos = append(salida.Documentos, doc)
	}
	return salida, nil
}

// rutas compone las dos rutas exactas. Sin circuito no hay firma.
func (f *firmaDocumentoCTDesarrollo) rutas(cfg config.Config, circuito *reglas.Resolutor) ([]vechttp.RutaExacta, error) {
	if f == nil {
		return nil, nil
	}
	if circuito == nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	verificador, err := nuevoVerificadorFirmaDocumentos(cfg)
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
	return []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaFirmaDocumento, Manejador: h},
		{Ruta: httpinterno.RutaConsultaFirmaDocumento, Manejador: h},
	}, nil
}
