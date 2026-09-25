package bootstrap

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

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

// Cese, cierre y modificación tras el nombramiento (CT115/CT116, AD3-82/83)
// en el perfil de desarrollo. Se componen solo si están declarados los tres
// catálogos de ejemplo (reglas, causas de cese y motivos de rectificación):
// las decisiones inventadas viven en esos catálogos, no en el código.

const accionConsultarSeguimientoCeseDesarrollo = "contratacion_temporal.seguimiento.consultar"

var errSeguimientoCeseDesarrolloNoDisponible = errors.New("contratacion temporal: cese, cierre y modificacion de desarrollo no disponibles")

type operacionSeguimientoCeseDesarrollo struct {
	ruta, frontera, accion, tipo, finalidad, audiencia string
	escritura                                          bool
}

func operacionesSeguimientoCeseDesarrollo() []operacionSeguimientoCeseDesarrollo {
	return []operacionSeguimientoCeseDesarrollo{
		{httpinterno.RutaCesesNombramiento, "ct-cese-registrar", string(domain.AccionCesarNombramiento), ports.TipoRecursoCese, ports.FinalidadRegistrarCese, ports.AudienciaConsumoCeseV1, true},
		{httpinterno.RutaCierresExpediente, "ct-expediente-cerrar", string(domain.AccionCerrarExpediente), ports.TipoRecursoCierreExpediente, ports.FinalidadCerrarExpediente, ports.AudienciaConsumoCierreExpedienteV1, true},
		{httpinterno.RutaModificacionesNombramiento, "ct-nombramiento-modificar", string(domain.AccionModificarTrasNombramiento), ports.TipoRecursoModificacionNombramiento, ports.FinalidadModificarTrasNombramiento, ports.AudienciaConsumoModificacionNombramientoV1, true},
		{httpinterno.RutaSeguimientoCese, "ct-seguimiento-cese-consultar", accionConsultarSeguimientoCeseDesarrollo, "seguimiento_contratacion_temporal", "gestionar_contratacion_temporal", "", false},
	}
}

func operacionSeguimientoCesePorRuta(ruta string) (operacionSeguimientoCeseDesarrollo, bool) {
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		if o.ruta == ruta {
			return o, true
		}
	}
	return operacionSeguimientoCeseDesarrollo{}, false
}

func rutaSeguimientoCeseDesarrollo(ruta string) bool {
	_, ok := operacionSeguimientoCesePorRuta(ruta)
	return ok
}

// descriptoresFronterasSeguimientoCeseDesarrollo declara las cuatro rutas
// con el perfil CT base y su acción nominal como capacidad.
func descriptoresFronterasSeguimientoCeseDesarrollo(perfilCT string) []descriptorFronteraComunDesarrollo {
	var d []descriptorFronteraComunDesarrollo
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		d = append(d, fronteraContratacionTemporalDesarrollo(o.frontera, o.accion, o.ruta, []string{perfilCT}))
	}
	return d
}

func descriptoresMaterialSeguimientoCeseDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: ports.AudienciaConsumoCeseV1, Dominio: "vec.ct.cese.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-cese:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		{Audiencia: ports.AudienciaConsumoCierreExpedienteV1, Dominio: "vec.ct.cierre-expediente.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-cierre-expediente:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		{Audiencia: ports.AudienciaConsumoModificacionNombramientoV1, Dominio: "vec.ct.modificacion-nombramiento.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-modificacion-nombramiento:", ProveedorNominal: proveedorMaterialContratacionTemporal},
	}
}

// seguimientoCeseSolicitado exige pedirlo expresamente
// (VEC_CT_SEGUIMIENTO_CESE_ENABLED, que requiere AD3-82, AD3-83, CT115 y CT116
// instaladas) y los tres catálogos de ejemplo; la doble llave de desarrollo
// la comprueba el propio selector.
func seguimientoCeseSolicitado(cfg config.Config) bool {
	activo, err := cfg.CTSeguimientoCeseDesarrolloActivo()
	if err != nil {
		slog.Error("seguimiento de cese de CT no compuesto: selector inválido", "causa", err)
		return false
	}
	if !activo {
		return false
	}
	r := cfg.ReglasEjemplo
	return strings.TrimSpace(r.CTSourcePath) != "" && strings.TrimSpace(r.CausasCeseSourcePath) != "" &&
		strings.TrimSpace(cfg.CTAnalisisMotivosSourcePath) != ""
}

func motivoSeguimientoCeseDesarrollo(ruta string) vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_seguimiento_cese_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("seguimiento-cese-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "seguimiento-cese-ct:"+ruta)}
}

// soporteSeguimientoCeseDesarrollo es la instantánea de desarrollo, no
// autoritativa, con las cuatro concesiones nominales.
type soporteSeguimientoCeseDesarrollo struct {
	instantanea vecdomain.InstantaneaAutorizacion
}

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaSeguimientoCese() (vecdomain.InstantaneaAutorizacion, bool) {
	if s == nil || s.seguimientoCese == nil {
		return vecdomain.InstantaneaAutorizacion{}, false
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.seguimientoCese.instantanea), s.seguimientoCese.instantanea.Validar() == nil
}

// ambitosSeguimientoCese valida la solicitud ligada de una ruta y devuelve
// los ámbitos exactos de la instantánea para ese recurso.
func (s *soporteAltaContratacionTemporalDesarrollo) ambitosSeguimientoCese(ruta string, d vecdomain.DatosSolicitudAutorizacionLigadaV3) ([]vecdomain.AmbitoPerfil, bool) {
	o, ok := operacionSeguimientoCesePorRuta(ruta)
	r := d.Recurso
	if s == nil || !ok || d.Accion != o.accion || d.Finalidad != o.finalidad || d.ReferenciaMotivo != motivoSeguimientoCeseDesarrollo(ruta) ||
		r.ModuloID != ports.ModuloContratacion || r.Tipo != o.tipo || r.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		r.Ambitos["expediente_ref"] != r.Referencia || !domain.ReferenciaOpacaValida(r.Referencia) {
		return nil, false
	}
	claves := []string{"organizacion_ref", "expediente_ref"}
	if o.escritura {
		if r.Ambitos["fase_previa"] != string(domain.FaseNombramiento) || r.Ambitos["estado_previo"] != string(domain.EstadoEnCurso) {
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

func configurarSoporteSeguimientoCeseDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	var concesiones []vecdomain.ConcesionRol
	motivos := make([]vecdomain.ReferenciaEntradaCatalogo, 0, 4)
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: o.accion, ModuloID: ports.ModuloContratacion, TipoRecurso: o.tipo,
			Finalidades: []string{o.finalidad}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
		motivos = append(motivos, motivoSeguimientoCeseDesarrollo(o.ruta))
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"seguimiento_cese_ct_desarrollo", "Cese, cierre y modificación de desarrollo", "seguimiento-cese-ct-desarrollo", concesiones,
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente {
		return errSeguimientoCeseDesarrolloNoDisponible
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, motivos, desde); err != nil {
		return err
	}
	alta.soporte.mu.Lock()
	alta.soporte.seguimientoCese = &soporteSeguimientoCeseDesarrollo{instantanea: instantanea}
	alta.soporte.mu.Unlock()
	return nil
}

// autoridadSeguimientoCeseDesarrollo liga cada petición a la capacidad mTLS de
// su ruta, exige la decisión V3 con la instantánea de desarrollo y entrega el
// material atestado para la audiencia de la operación.
type autoridadSeguimientoCeseDesarrollo struct {
	alta        *dependenciasAltaContratacionTemporalDesarrollo
	proveedores map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (a *autoridadSeguimientoCeseDesarrollo) ResolverContextoCanalSeguimiento(ctx context.Context) (application.ContextoCanalSeguimiento, error) {
	if a == nil || a.alta == nil || a.alta.soporte == nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	if !valida || !rutaSeguimientoCeseDesarrollo(capacidad.ruta) {
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

func (a *autoridadSeguimientoCeseDesarrollo) exigir(ctx context.Context, accion, finalidad string, recurso vecdomain.RecursoAutorizable) (vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2, error) {
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
	o, ok := operacionSeguimientoCesePorRuta(capacidad.ruta)
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
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivoSeguimientoCeseDesarrollo(o.ruta),
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion}
	if _, ok := a.alta.soporte.ambitosSeguimientoCese(o.ruta, datos); !ok {
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

func (a *autoridadSeguimientoCeseDesarrollo) AutorizarOperacionSeguimiento(ctx context.Context, sol ports.SolicitudAutorizarOperacionSeguimiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	proveedor := a.proveedores[sol.Audiencia]
	if proveedor == nil || sol.Motivo.Validar() != nil {
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
	return proveedor.proveerMaterialConfirmacion(ctx, s, d, c, sol.Motivo, resultado)
}

// AutorizarLecturaSeguimiento exige una decisión de lectura del expediente
// exacto; no emite capacidad de consumo ni escribe ningún efecto.
func (a *autoridadSeguimientoCeseDesarrollo) AutorizarLecturaSeguimiento(ctx context.Context, organizacionRef, expedienteRef string) error {
	if organizacionRef != organizacionAltaContratacionTemporalDesarrollo || !domain.ReferenciaOpacaValida(expedienteRef) {
		return ports.ErrAutorizacionDenegada
	}
	_, _, _, _, err := a.exigir(ctx, accionConsultarSeguimientoCeseDesarrollo, "gestionar_contratacion_temporal", vecdomain.RecursoAutorizable{
		Referencia: expedienteRef, ModuloID: ports.ModuloContratacion, Tipo: "seguimiento_contratacion_temporal",
		Ambitos:   map[string]string{"organizacion_ref": organizacionRef, "expediente_ref": expedienteRef},
		Atributos: map[string]string{"lectura": "cese_cierre_opciones"}})
	return err
}

// fuenteReglasSeguimientoDesarrollo lee los catálogos de ejemplo: causas de
// cese (catálogo propio), reglas c09/c10/c11 y motivos de rectificación.
type fuenteReglasSeguimientoDesarrollo struct {
	reglas  *reglas.Resolutor
	causas  puertosvec.ConsultaCatalogosConfigurables
	motivos puertosvec.ConsultaCatalogosConfigurables
}

func politicaSeguimientoDesarrollo(regla reglas.Regla, ruta string, instante time.Time) ports.PoliticaOperacionSeguimiento {
	return ports.PoliticaOperacionSeguimiento{DefinicionRef: regla.ReferenciaEntrada.CatalogoID, DefinicionVersion: uint64(regla.ReferenciaEntrada.CatalogoVersion),
		DefinicionHuellaSHA256: regla.ReferenciaEntrada.CatalogoHuellaSHA256, MotivoAutorizacion: motivoSeguimientoCeseDesarrollo(ruta),
		EvaluadaEn: instante, ValidaHasta: instante.Add(2 * time.Minute)}
}

// catalogoVigente devuelve la última versión publicada con sus entradas
// vigentes en el instante; nunca una versión fijada en el código.
func catalogoVigenteSeguimiento(ctx context.Context, consulta puertosvec.ConsultaCatalogosConfigurables, id string, instante time.Time) (vecdomain.CatalogoConfigurable, []vecdomain.EntradaCatalogoConfigurable, error) {
	versiones, err := consulta.ListarVersionesCatalogo(ctx, id)
	if err != nil || len(versiones) == 0 {
		return vecdomain.CatalogoConfigurable{}, nil, errSeguimientoCeseDesarrolloNoDisponible
	}
	sort.Slice(versiones, func(i, j int) bool { return versiones[i].Version > versiones[j].Version })
	for _, c := range versiones {
		if c.Estado != vecdomain.EstadoCatalogoPublicado || c.ModuloID != ports.ModuloContratacion {
			continue
		}
		var vigentes []vecdomain.EntradaCatalogoConfigurable
		for _, e := range c.Entradas {
			if e.VigenteEn(instante) {
				vigentes = append(vigentes, e)
			}
		}
		sort.SliceStable(vigentes, func(i, j int) bool { return vigentes[i].Orden < vigentes[j].Orden })
		return c, vigentes, nil
	}
	return vecdomain.CatalogoConfigurable{}, nil, errSeguimientoCeseDesarrolloNoDisponible
}

func (f fuenteReglasSeguimientoDesarrollo) CausasCese(ctx context.Context, instante time.Time) ([]ports.CausaCese, ports.PoliticaOperacionSeguimiento, error) {
	regla, err := f.reglas.Regla(ctx, reglas.CTCausasCese)
	if err != nil || regla.Atributos["catalogo"] == "" {
		return nil, ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	catalogo, entradas, err := catalogoVigenteSeguimiento(ctx, f.causas, regla.Atributos["catalogo"], instante)
	if err != nil {
		return nil, ports.PoliticaOperacionSeguimiento{}, err
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return nil, ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	admitidas := regla.Elementos()
	var causas []ports.CausaCese
	for _, e := range entradas {
		if !slices.Contains(admitidas, e.Clave) {
			continue
		}
		causas = append(causas, ports.CausaCese{Clave: domain.ClaveCatalogo(e.Clave), Etiqueta: e.Etiqueta, ClaveI18n: e.Atributos["clave_i18n"],
			JustificanteTipo: domain.ClaveCatalogo(e.Atributos["justificante"])})
	}
	p := ports.PoliticaOperacionSeguimiento{DefinicionRef: catalogo.ID, DefinicionVersion: uint64(catalogo.Version), DefinicionHuellaSHA256: huella,
		MotivoAutorizacion: motivoSeguimientoCeseDesarrollo(httpinterno.RutaCesesNombramiento), EvaluadaEn: instante, ValidaHasta: instante.Add(2 * time.Minute)}
	return causas, p, nil
}

func (f fuenteReglasSeguimientoDesarrollo) ReglaCierre(ctx context.Context, instante time.Time) (ports.ReglaCierreExpediente, ports.PoliticaOperacionSeguimiento, error) {
	regla, err := f.reglas.Regla(ctx, reglas.CTCierreExpediente)
	if err != nil {
		return ports.ReglaCierreExpediente{}, ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	condiciones := append([]string(nil), regla.Elementos()...)
	sort.Strings(condiciones)
	return ports.ReglaCierreExpediente{Condiciones: condiciones}, politicaSeguimientoDesarrollo(regla, httpinterno.RutaCierresExpediente, instante), nil
}

func (f fuenteReglasSeguimientoDesarrollo) ReglaModificacion(ctx context.Context, instante time.Time) (ports.ReglaModificacionNombramiento, ports.PoliticaOperacionSeguimiento, error) {
	regla, err := f.reglas.Regla(ctx, reglas.CTModificacionFaseRetorno)
	elementos := regla.Elementos()
	if err != nil || len(elementos) != 1 || regla.Atributos["catalogo"] == "" {
		return ports.ReglaModificacionNombramiento{}, ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	_, entradas, err := catalogoVigenteSeguimiento(ctx, f.motivos, regla.Atributos["catalogo"], instante)
	if err != nil {
		return ports.ReglaModificacionNombramiento{}, ports.PoliticaOperacionSeguimiento{}, err
	}
	admitidos := strings.Split(regla.Atributos["motivos"], ",")
	var motivos []ports.OpcionMotivoSeguimiento
	for _, e := range entradas {
		if slices.Contains(admitidos, e.Clave) {
			motivos = append(motivos, ports.OpcionMotivoSeguimiento{Clave: domain.ClaveCatalogo(e.Clave), Etiqueta: e.Etiqueta, ClaveI18n: e.Atributos["clave_i18n"]})
		}
	}
	return ports.ReglaModificacionNombramiento{FaseRetorno: domain.ClaveFase(elementos[0]), Motivos: motivos},
		politicaSeguimientoDesarrollo(regla, httpinterno.RutaModificacionesNombramiento, instante), nil
}

// calculadorCosteModificacionDesarrollo usa el mismo catálogo ct.retribuciones
// que la fuente del análisis (categoría o grupo, periodo y jornada). Sin
// catálogo, o sin fila para el expediente, el coste no está disponible.
type calculadorCosteModificacionDesarrollo struct {
	retribuciones *fuenteRetribucionesDesarrollo
}

func (c calculadorCosteModificacionDesarrollo) CalcularCosteModificacion(ctx context.Context, e domain.Expediente, periodo domain.PeriodoPrevisto, jornada domain.JornadaDiezmilesimas) (domain.Importe, string, error) {
	if e.Analisis == nil {
		return domain.Importe{}, "", errSeguimientoCeseDesarrolloNoDisponible
	}
	fila, ok, err := c.retribuciones.retribucion(ctx, e.Analisis.CategoriaRef, e.Analisis.GrupoSubgrupo)
	if err != nil || !ok {
		return domain.Importe{}, "", errSeguimientoCeseDesarrolloNoDisponible
	}
	importe, ok := costeEstimadoAnalisisDesarrollo(fila, periodo, jornada)
	if !ok {
		return domain.Importe{}, "", errSeguimientoCeseDesarrolloNoDisponible
	}
	return importe, autoridadCosteAnalisisDesarrollo, nil
}

// nuevasRutasSeguimientoCeseDesarrollo compone las cuatro rutas cuando se han
// declarado los catálogos. Sin ellos, no hay rutas: la conducta es la de hoy.
func nuevasRutasSeguimientoCeseDesarrollo(dependencias *DependenciasCT, alta *dependenciasAltaContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	if dependencias == nil || !seguimientoCeseSolicitado(dependencias.cfg) {
		return nil, nil
	}
	cfg, derivador, reloj := dependencias.cfg, dependencias.derivador, dependencias.reloj
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.proveedorMaterial == nil || derivador == nil || !derivador.valido() {
		log.Print("contratacion temporal: cese y modificacion no disponibles; etapa=dependencias")
		return nil, errSeguimientoCeseDesarrolloNoDisponible
	}
	fallar := func(etapa string, err error) ([]vechttp.RutaExacta, error) {
		log.Printf("contratacion temporal: cese y modificacion no disponibles; etapa=%s", etapa)
		return nil, errors.Join(errSeguimientoCeseDesarrolloNoDisponible, err)
	}
	causas, err := fichero.NuevaConsultaCatalogos(cfg.ReglasEjemplo.CausasCeseSourcePath)
	if err != nil {
		return fallar("catalogo_causas", err)
	}
	motivos, err := fichero.NuevaConsultaCatalogos(cfg.CTAnalisisMotivosSourcePath)
	if err != nil {
		return fallar("catalogo_motivos", err)
	}
	consultaReglas, err := fichero.NuevaConsultaCatalogos(cfg.ReglasEjemplo.CTSourcePath)
	if err != nil {
		return fallar("catalogo_reglas", err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consultaReglas, Metadatos: consultaReglas,
		CatalogoID: reglas.CatalogoContratacionTemporal, ModuloID: reglas.ModuloContratacionTemporal, Reloj: reloj, MunicipioSede: reglas.MunicipioSedeDiputacion})
	if err != nil {
		return fallar("resolutor", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	if err := configurarSoporteSeguimientoCeseDesarrollo(ctx, alta, reloj); err != nil {
		return fallar("instantanea", err)
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return fallar("material", err)
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	autoridad := &autoridadSeguimientoCeseDesarrollo{alta: alta, proveedores: map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{}}
	for _, d := range descriptoresMaterialSeguimientoCeseDesarrollo() {
		p, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, alta.postgresql.catalogoMaterial, d.Audiencia)
		if err != nil {
			return fallar("proveedor_material", err)
		}
		autoridad.proveedores[d.Audiencia] = p
	}
	var llaveros []seguridadct.ConfiguracionLlaverosSeguimiento
	for _, operacion := range []string{ports.OperacionRegistrarCese, ports.OperacionCerrarExpediente, ports.OperacionModificarTrasNombramiento} {
		dominioAmbito, dominioHuella, _ := ports.DominiosHMACOperacionSeguimiento(operacion)
		aa, ra, err := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, dominioAmbito, true)
		if err != nil {
			return fallar("sellos", err)
		}
		ah, rh, err := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, dominioHuella, false)
		if err != nil {
			return fallar("sellos", err)
		}
		llaveros = append(llaveros, seguridadct.ConfiguracionLlaverosSeguimiento{Operacion: operacion, ActivaAmbito: aa, RetenidasAmbito: ra, ActivaHuella: ah, RetenidasHuella: rh})
	}
	sellos, err := seguridadct.NuevaAutoridadSellosSeguimientoHMAC(llaveros...)
	if err != nil {
		return fallar("sellos", err)
	}
	repositorio, err := postgresct.NuevoRepositorioOperacionSeguimientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return fallar("repositorio", err)
	}
	servicio, err := application.NuevoServicioOperacionesSeguimiento(application.DependenciasOperacionesSeguimiento{
		Contextos: alta.soporte, Sellos: sellos, Repositorio: repositorio,
		Reglas:      fuenteReglasSeguimientoDesarrollo{reglas: resolutor, causas: causas, motivos: motivos},
		Autorizador: autoridad, Referencias: seguridadct.NuevoGeneradorReferenciasAltaCriptografico(),
		Coste: calculadorCosteModificacionDesarrollo{retribuciones: dependencias.retribucionesCT}, Lector: repositorio, Reloj: reloj})
	if err != nil {
		return fallar("servicio", err)
	}
	manejadores, err := httpinterno.NuevosManejadoresSeguimiento(autoridad, autoridad, servicio)
	if err != nil {
		return fallar("http", err)
	}
	rutas := make([]vechttp.RutaExacta, 0, len(manejadores))
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: o.ruta, Manejador: manejadores[o.ruta]})
	}
	return rutas, nil
}
