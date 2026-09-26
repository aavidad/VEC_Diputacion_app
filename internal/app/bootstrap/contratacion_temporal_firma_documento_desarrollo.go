package bootstrap

import (
	"context"
	"errors"
	"log"
	"maps"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Registro de firmas de prueba de los borradores de CT en el perfil de
// desarrollo. Se compone solo con VEC_CT_FIRMA_REGISTRO_ENABLED=true (AD3-85
// y CT118 instaladas) y el circuito de firma de ejemplo. La autorización es
// un rol nominal propio publicado para la identidad del canal CT: no se
// infiere del cargo del catálogo, cuyo perfil_ref sigue siendo informativo.
// La verificación la hace el validador de AutoFirma; apagado, toda firma se
// rechaza. Ninguna firma registrada tiene eficacia administrativa.

var errFirmaDocumentoCTDesarrolloNoDisponible = errors.New("contratacion temporal: registro de firmas de desarrollo no disponible")

func rutaFirmaDocumentoCTDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaFirmaDocumento || ruta == httpinterno.RutaConsultaFirmaDocumento
}

// descriptorMaterialFirmaDocumentoCTDesarrollo es el consumidor V3 de AD3-85.
func descriptorMaterialFirmaDocumentoCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: ports.AudienciaFirmaDocumentoV3, Dominio: "vec.ct.firma-documento.desarrollo.capacidad-v3",
		Prefijo: "clave:capacidad:ct-firma-documento:", ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
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
	alta     *dependenciasAltaContratacionTemporalDesarrollo
	registro *postgrescontratacion.RegistroFirmasDocumentoPostgreSQL
	reloj    relojContratacionTemporalDesarrollo
	// fiscalizacion recibe al componer las rutas la comprobación de la firma
	// que habilita la remisión a Intervención (duda 4).
	fiscalizacion *ctapplication.ServicioFiscalizaciones
}

var (
	_ ports.AutorizadorFirmaDocumento          = (*firmaDocumentoCTDesarrollo)(nil)
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
		alta.postgresql.ejecucion == nil || alta.postgresql.proveedorMaterialFirmaDocumento == nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	registro, err := postgrescontratacion.NuevoRegistroFirmasDocumentoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"firma_documento_ct_desarrollo", "Firma de prueba de borradores CT de desarrollo", "asignacion-firma-documento-ct-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{{Accion: ports.AccionFirmarDocumento, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaDocumento,
			Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh}},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
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
	alta.soporte.instantaneaFirmaDocumento = instantanea
	alta.soporte.motivoFirmaDocumento = motivoFirmaDocumentoCTDesarrollo()
	alta.soporte.mu.Unlock()
	return &firmaDocumentoCTDesarrollo{alta: alta, registro: registro, reloj: reloj, fiscalizacion: fiscalizacion}, nil
}

// ResolverOrganizacionFirmaDocumento solo responde dentro de la frontera mTLS
// de CT y para las dos rutas de firma.
func (f *firmaDocumentoCTDesarrollo) ResolverOrganizacionFirmaDocumento(ctx context.Context) (string, error) {
	if f == nil || f.alta == nil || f.alta.soporte == nil || ctx == nil {
		return "", ports.ErrAutorizacionDenegada
	}
	capacidad, valida := f.alta.soporte.capacidadValida(ctx)
	if !valida || !rutaFirmaDocumentoCTDesarrollo(capacidad.ruta) {
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
	if !valida || capacidad.ruta != httpinterno.RutaFirmaDocumento || m.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
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
	servicio, err := ctapplication.NuevoServicioFirmaDocumento(fuenteCircuitoFirmaReglasDesarrollo{resolutor: circuito}, f.registro, f, verificador)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
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
