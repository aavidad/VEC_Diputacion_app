package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Cancelación del expediente antes de la fiscalización por RRHH (CT122,
// AD3-87) en el perfil de desarrollo. Se compone solo con
// VEC_CT_CANCELACION_ENABLED y los catálogos de reglas de CT (regla c12) y de
// motivos de cancelación: fases, motivos y canales viven en esos catálogos.

const (
	accionConsultarCancelacionCTDesarrollo    = "contratacion_temporal.cancelacion.consultar"
	finalidadConsultarCancelacionCTDesarrollo = "gestionar_contratacion_temporal"
)

var (
	errCancelacionCTDesarrolloNoDisponible = errors.New("contratacion temporal: cancelacion de desarrollo no disponible")

	// ErrCancelacionCTMigracionesNoDisponibles detiene el arranque cuando la
	// cancelación está encendida y falta alguna de sus migraciones.
	ErrCancelacionCTMigracionesNoDisponibles = errors.New("bootstrap: cancelacion de expedientes de CT sin sus migraciones")
	ErrCancelacionCTFaltaAD387               = fmt.Errorf("%w: falta AD3-87 (consumidor de la cancelación)", ErrCancelacionCTMigracionesNoDisponibles)
	ErrCancelacionCTFaltaCT122               = fmt.Errorf("%w: falta CT 000122 (cancelación del expediente)", ErrCancelacionCTMigracionesNoDisponibles)
	errCancelacionCTComprobacionRota         = fmt.Errorf("%w: no se pudo comprobar el catálogo", ErrCancelacionCTMigracionesNoDisponibles)
)

type operacionCancelacionCTDesarrollo struct {
	ruta, frontera, accion, tipo, finalidad, audiencia string
	escritura                                          bool
}

func operacionesCancelacionCTDesarrollo() []operacionCancelacionCTDesarrollo {
	return []operacionCancelacionCTDesarrollo{
		{httpinterno.RutaCancelacionesExpediente, "ct-expediente-cancelar", string(domain.AccionCancelarExpediente), ports.TipoRecursoCancelacion,
			ports.FinalidadCancelarExpediente, ports.AudienciaConsumoCancelacionV1, true},
		{httpinterno.RutaCancelacionExpediente, "ct-cancelacion-consultar", accionConsultarCancelacionCTDesarrollo, ports.TipoRecursoCancelacion,
			finalidadConsultarCancelacionCTDesarrollo, "", false},
	}
}

func operacionCancelacionCTPorRuta(ruta string) (operacionCancelacionCTDesarrollo, bool) {
	for _, o := range operacionesCancelacionCTDesarrollo() {
		if o.ruta == ruta {
			return o, true
		}
	}
	return operacionCancelacionCTDesarrollo{}, false
}

func rutaCancelacionCTDesarrollo(ruta string) bool {
	_, ok := operacionCancelacionCTPorRuta(ruta)
	return ok
}

// descriptoresFronterasCancelacionCTDesarrollo declara las dos rutas con el
// perfil CT base y su acción nominal como capacidad.
func descriptoresFronterasCancelacionCTDesarrollo(perfilCT string) []descriptorFronteraComunDesarrollo {
	var d []descriptorFronteraComunDesarrollo
	for _, o := range operacionesCancelacionCTDesarrollo() {
		d = append(d, fronteraContratacionTemporalDesarrollo(o.frontera, o.accion, o.ruta, []string{perfilCT}))
	}
	return d
}

func descriptoresMaterialCancelacionCTDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: ports.AudienciaConsumoCancelacionV1, Dominio: "vec.ct.cancelacion-expediente.desarrollo.capacidad-v3",
			Prefijo: "clave:capacidad:ct-cancelacion-expediente:", ProveedorNominal: proveedorMaterialContratacionTemporal},
	}
}

// cancelacionCTSolicitada exige pedirla expresamente; el selector comprueba
// la doble llave y los dos catálogos, de modo que pedirla sin alguno detiene
// el arranque en lugar de dejar las rutas sin montar en silencio.
func cancelacionCTSolicitada(cfg config.Config) bool {
	activo, err := cfg.CTCancelacionDesarrolloActivo()
	if err != nil {
		slog.Error("cancelación de expedientes de CT no compuesta: selector inválido", "causa", err)
		return false
	}
	return activo
}

func motivoCancelacionCTDesarrollo(ruta string) vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cancelacion_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("cancelacion-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "cancelacion-ct:"+ruta)}
}

// soporteCancelacionCTDesarrollo es la instantánea de desarrollo, no
// autoritativa, con las dos concesiones nominales.
type soporteCancelacionCTDesarrollo struct {
	instantanea vecdomain.InstantaneaAutorizacion
}

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaCancelacionCT() (vecdomain.InstantaneaAutorizacion, bool) {
	if s == nil || s.cancelacion == nil {
		return vecdomain.InstantaneaAutorizacion{}, false
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.cancelacion.instantanea), s.cancelacion.instantanea.Validar() == nil
}

// ambitosCancelacionCT valida la solicitud ligada de una ruta y devuelve los
// ámbitos exactos de la instantánea para ese recurso. RRHH cancela en
// cualquier fase que admita la regla: la fase previa solo ha de ser una clave
// válida y el estado previo, en curso; SQL coteja ambos con el expediente.
func (s *soporteAltaContratacionTemporalDesarrollo) ambitosCancelacionCT(ruta string, d vecdomain.DatosSolicitudAutorizacionLigadaV3) ([]vecdomain.AmbitoPerfil, bool) {
	o, ok := operacionCancelacionCTPorRuta(ruta)
	r := d.Recurso
	if s == nil || !ok || d.Accion != o.accion || d.Finalidad != o.finalidad || d.ReferenciaMotivo != motivoCancelacionCTDesarrollo(ruta) ||
		r.ModuloID != ports.ModuloContratacion || r.Tipo != o.tipo || r.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		r.Ambitos["expediente_ref"] != r.Referencia || !domain.ReferenciaOpacaValida(r.Referencia) {
		return nil, false
	}
	claves := []string{"organizacion_ref", "expediente_ref"}
	if o.escritura {
		if !domain.ClaveFase(r.Ambitos["fase_previa"]).Valida() || r.Ambitos["estado_previo"] != string(domain.EstadoEnCurso) ||
			r.Atributos["canal"] != string(domain.CanalCancelacionRRHH) {
			return nil, false
		}
		claves = append(claves, "fase_previa", "estado_previo")
	}
	if len(r.Ambitos) != len(claves) {
		return nil, false
	}
	ambitos := make([]vecdomain.AmbitoPerfil, 0, len(claves))
	for _, c := range claves {
		ambitos = append(ambitos, vecdomain.AmbitoPerfil{Clave: c, Valores: []string{r.Ambitos[c]}})
	}
	return ambitos, true
}

func configurarSoporteCancelacionCTDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	var concesiones []vecdomain.ConcesionRol
	var motivos []vecdomain.ReferenciaEntradaCatalogo
	for _, o := range operacionesCancelacionCTDesarrollo() {
		concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: o.accion, ModuloID: ports.ModuloContratacion, TipoRecurso: o.tipo,
			Finalidades: []string{o.finalidad}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
		motivos = append(motivos, motivoCancelacionCTDesarrollo(o.ruta))
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"cancelacion_ct_desarrollo", "Cancelación de expedientes de desarrollo", "cancelacion-ct-desarrollo", concesiones,
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente {
		return errCancelacionCTDesarrolloNoDisponible
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, motivos, desde); err != nil {
		return err
	}
	alta.soporte.mu.Lock()
	alta.soporte.cancelacion = &soporteCancelacionCTDesarrollo{instantanea: instantanea}
	alta.soporte.mu.Unlock()
	return nil
}

// autoridadCancelacionCTDesarrollo liga cada petición a la capacidad mTLS de
// su ruta, exige la decisión V3 con la instantánea de desarrollo y entrega el
// material atestado para la audiencia de la cancelación.
type autoridadCancelacionCTDesarrollo struct {
	alta      *dependenciasAltaContratacionTemporalDesarrollo
	proveedor *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (a *autoridadCancelacionCTDesarrollo) ResolverContextoCanalSeguimiento(ctx context.Context) (application.ContextoCanalSeguimiento, error) {
	if a == nil || a.alta == nil || a.alta.soporte == nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	if !valida || !rutaCancelacionCTDesarrollo(capacidad.ruta) {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	operativo, err := a.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	v, err := operativo.Vinculo.Datos()
	if err != nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	return application.ContextoCanalSeguimiento{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, nil
}

func (a *autoridadCancelacionCTDesarrollo) exigir(ctx context.Context, accion, finalidad string, recurso vecdomain.RecursoAutorizable) (vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2, error) {
	var (
		s vecdomain.SolicitudAutorizacionLigadaV3
		d vecdomain.DecisionAutorizacionLigadaV3
		c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
		r vecdomain.ResultadoContextoActorRegistradoV2
	)
	if ctx == nil || a == nil || a.alta == nil || a.alta.soporte == nil || a.alta.autorizador == nil {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	o, ok := operacionCancelacionCTPorRuta(capacidad.ruta)
	if !valida || !ok || o.accion != accion || o.finalidad != finalidad {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return s, d, c, r, err
	}
	operativo, err := a.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivoCancelacionCTDesarrollo(o.ruta),
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion}
	if _, ok := a.alta.soporte.ambitosCancelacionCT(o.ruta, datos); !ok {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	s, err = vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return s, d, c, r, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	d, c, err = a.alta.autorizador.ExigirSolicitudLigadaV3(ctx, s, operativo.Resultado)
	return s, d, c, operativo.Resultado, err
}

func (a *autoridadCancelacionCTDesarrollo) AutorizarOperacionSeguimiento(ctx context.Context, sol ports.SolicitudAutorizarOperacionSeguimiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if a == nil || a.proveedor == nil || sol.Audiencia != ports.AudienciaConsumoCancelacionV1 || sol.Motivo.Validar() != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	s, d, c, resultado, err := a.exigir(ctx, string(sol.Accion), sol.Finalidad, sol.Recurso)
	if err != nil {
		return vacio, err
	}
	datos, err := s.Datos()
	if err != nil || datos.ReferenciaMotivo != sol.Motivo {
		return vacio, ports.ErrAutorizacionDenegada
	}
	return a.proveedor.proveerMaterialConfirmacion(ctx, s, d, c, sol.Motivo, resultado)
}

// AutorizarLecturaSeguimiento exige una decisión de lectura del expediente
// exacto; no emite capacidad de consumo ni escribe ningún efecto.
func (a *autoridadCancelacionCTDesarrollo) AutorizarLecturaSeguimiento(ctx context.Context, organizacionRef, expedienteRef string) error {
	if organizacionRef != organizacionAltaContratacionTemporalDesarrollo || !domain.ReferenciaOpacaValida(expedienteRef) {
		return ports.ErrAutorizacionDenegada
	}
	_, _, _, _, err := a.exigir(ctx, accionConsultarCancelacionCTDesarrollo, finalidadConsultarCancelacionCTDesarrollo, vecdomain.RecursoAutorizable{
		Referencia: expedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
		Ambitos:   map[string]string{"organizacion_ref": organizacionRef, "expediente_ref": expedienteRef},
		Atributos: map[string]string{"lectura": "cancelacion_opciones"}})
	return err
}

// fuenteReglasCancelacionDesarrollo lee la regla c12 (fases y motivos
// admitidos) y el catálogo de motivos (etiquetas, i18n y canales). Nada se
// fija en el código: sin los catálogos, la capacidad no se compone.
type fuenteReglasCancelacionDesarrollo struct {
	reglas  *reglas.Resolutor
	motivos puertosvec.ConsultaCatalogosConfigurables
}

func (f fuenteReglasCancelacionDesarrollo) ReglaCancelacion(ctx context.Context, instante time.Time) (ports.ReglaCancelacion, ports.PoliticaOperacionSeguimiento, error) {
	var vacia ports.ReglaCancelacion
	regla, err := f.reglas.Regla(ctx, reglas.CTCancelacionExpediente)
	if err != nil || regla.Atributos["catalogo"] == "" || regla.Atributos["motivos"] == "" {
		return vacia, ports.PoliticaOperacionSeguimiento{}, errCancelacionCTDesarrolloNoDisponible
	}
	catalogo, entradas, err := catalogoVigenteSeguimiento(ctx, f.motivos, regla.Atributos["catalogo"], instante)
	if err != nil {
		return vacia, ports.PoliticaOperacionSeguimiento{}, err
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return vacia, ports.PoliticaOperacionSeguimiento{}, errCancelacionCTDesarrolloNoDisponible
	}
	var r ports.ReglaCancelacion
	for _, fase := range regla.Elementos() {
		r.Fases = append(r.Fases, domain.ClaveFase(fase))
	}
	admitidos := strings.Split(regla.Atributos["motivos"], ",")
	for _, e := range entradas {
		if !slices.Contains(admitidos, e.Clave) {
			continue
		}
		var canales []domain.CanalCancelacion
		for _, c := range strings.Split(e.Atributos["canales"], ",") {
			canales = append(canales, domain.CanalCancelacion(c))
		}
		r.Motivos = append(r.Motivos, ports.MotivoCancelacion{Clave: domain.ClaveCatalogo(e.Clave), Etiqueta: e.Etiqueta,
			ClaveI18n: e.Atributos["clave_i18n"], Canales: canales})
	}
	if !r.Valida() {
		return vacia, ports.PoliticaOperacionSeguimiento{}, errCancelacionCTDesarrolloNoDisponible
	}
	p := ports.PoliticaOperacionSeguimiento{DefinicionRef: catalogo.ID, DefinicionVersion: uint64(catalogo.Version), DefinicionHuellaSHA256: huella,
		MotivoAutorizacion: motivoCancelacionCTDesarrollo(httpinterno.RutaCancelacionesExpediente), EvaluadaEn: instante, ValidaHasta: instante.Add(2 * time.Minute)}
	return r, p, nil
}

// consultaMigracionesCancelacionCT se ejecuta con el LOGIN ejecutor de CT:
// exige las fachadas de CT122 ejecutables y, de AD3-87, su fachada de consumo
// en el catálogo (legible por todos), que ese LOGIN no ejecuta directamente.
const consultaMigracionesCancelacionCT = `SELECT
 (SELECT count(*)=1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_cancelacion_expediente_ct_v3_atestada'),
 (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL AND pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE')),false) FROM pg_catalog.unnest(ARRAY[
  'vec_contratacion_temporal.preparar_cancelacion_expediente_v1(jsonb)',
  'vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.consultar_cancelacion_expediente_v1(text,text)']) f)`

// comprobarMigracionesCancelacionCTDesarrollo se llama al componer, antes de
// publicar la clave de la audiencia, y nombra la primera migración ausente.
func comprobarMigracionesCancelacionCTDesarrollo(ctx context.Context, ejecucion *pgxpool.Pool) error {
	if ejecucion == nil {
		return errCancelacionCTComprobacionRota
	}
	var ad387, ct122 bool
	if err := ejecucion.QueryRow(ctx, consultaMigracionesCancelacionCT).Scan(&ad387, &ct122); err != nil {
		return errCancelacionCTComprobacionRota
	}
	switch {
	case !ad387:
		return ErrCancelacionCTFaltaAD387
	case !ct122:
		return ErrCancelacionCTFaltaCT122
	}
	return nil
}

// nuevasRutasCancelacionCTDesarrollo compone las dos rutas de RRHH cuando se
// ha pedido la capacidad. Sin el selector no hay rutas: la conducta es la de
// hoy. Pedida y sin sus migraciones o catálogos, el arranque se detiene.
func nuevasRutasCancelacionCTDesarrollo(dependencias *DependenciasCT, alta *dependenciasAltaContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	if dependencias == nil || !cancelacionCTSolicitada(dependencias.cfg) {
		return nil, nil
	}
	cfg, derivador, reloj := dependencias.cfg, dependencias.derivador, dependencias.reloj
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.proveedorMaterial == nil || derivador == nil || !derivador.valido() {
		log.Print("contratacion temporal: cancelacion no disponible; etapa=dependencias")
		return nil, errCancelacionCTDesarrolloNoDisponible
	}
	fallar := func(etapa string, err error) ([]vechttp.RutaExacta, error) {
		log.Printf("contratacion temporal: cancelacion no disponible; etapa=%s", etapa)
		return nil, errors.Join(errCancelacionCTDesarrolloNoDisponible, err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	if err := comprobarMigracionesCancelacionCTDesarrollo(ctx, alta.postgresql.ejecucion); err != nil {
		slog.Error("cancelación de expedientes de CT encendida sin sus migraciones", "causa", err)
		return nil, err
	}
	reglasCfg := cfg.ReglasEjemplo
	motivos, err := fichero.NuevaConsultaCatalogos(strings.TrimSpace(reglasCfg.MotivosCancelacionSourcePath))
	if err != nil {
		return fallar("catalogo_motivos", err)
	}
	consultaReglas, err := fichero.NuevaConsultaCatalogos(strings.TrimSpace(reglasCfg.CTSourcePath))
	if err != nil {
		return fallar("catalogo_reglas", err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consultaReglas, Metadatos: consultaReglas,
		CatalogoID: reglas.CatalogoContratacionTemporal, ModuloID: reglas.ModuloContratacionTemporal, Reloj: reloj, MunicipioSede: reglas.MunicipioSedeDiputacion})
	if err != nil {
		return fallar("resolutor", err)
	}
	fuente := fuenteReglasCancelacionDesarrollo{reglas: resolutor, motivos: motivos}
	// La regla y el catálogo se comprueban al componer: un catálogo que no
	// casa con la regla detiene el arranque en lugar de fallar al usarlo.
	if _, _, err := fuente.ReglaCancelacion(ctx, reloj.Ahora().UTC()); err != nil {
		return fallar("regla_c12", err)
	}
	if err := configurarSoporteCancelacionCTDesarrollo(ctx, alta, reloj); err != nil {
		return fallar("instantanea", err)
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return fallar("material", err)
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	proveedor, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj,
		alta.postgresql.catalogoMaterial, ports.AudienciaConsumoCancelacionV1)
	if err != nil {
		return fallar("proveedor_material", err)
	}
	autoridad := &autoridadCancelacionCTDesarrollo{alta: alta, proveedor: proveedor}
	dominioAmbito, dominioHuella, _ := ports.DominiosHMACOperacionSeguimiento(ports.OperacionCancelarExpediente)
	aa, ra, err := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, dominioAmbito, true)
	if err != nil {
		return fallar("sellos", err)
	}
	ah, rh, err := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, dominioHuella, false)
	if err != nil {
		return fallar("sellos", err)
	}
	sellos, err := seguridadct.NuevaAutoridadSellosSeguimientoHMAC(seguridadct.ConfiguracionLlaverosSeguimiento{Operacion: ports.OperacionCancelarExpediente,
		ActivaAmbito: aa, RetenidasAmbito: ra, ActivaHuella: ah, RetenidasHuella: rh})
	if err != nil {
		return fallar("sellos", err)
	}
	repositorio, err := postgresct.NuevoRepositorioOperacionSeguimientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return fallar("repositorio", err)
	}
	servicio, err := application.NuevoServicioCancelacionExpediente(application.DependenciasCancelacionExpediente{Canal: domain.CanalCancelacionRRHH,
		Contextos: alta.soporte, Sellos: sellos, Repositorio: repositorio, Reglas: fuente, Autorizador: autoridad,
		Referencias: seguridadct.NuevoGeneradorReferenciasAltaCriptografico(), Lector: repositorio, Reloj: reloj})
	if err != nil {
		return fallar("servicio", err)
	}
	manejadores, err := httpinterno.NuevosManejadoresCancelacion(autoridad, autoridad, servicio)
	if err != nil {
		return fallar("http", err)
	}
	rutas := make([]vechttp.RutaExacta, 0, len(manejadores))
	for _, o := range operacionesCancelacionCTDesarrollo() {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: o.ruta, Manejador: manejadores[o.ruta]})
	}
	return rutas, nil
}
