package bootstrap

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strconv"
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
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Cese, cierre y modificación tras el nombramiento (CT115/CT116, AD3-82/83)
// en el perfil de desarrollo. Se componen solo si están declarados los tres
// catálogos de ejemplo (reglas, causas de cese y motivos de rectificación):
// las decisiones inventadas viven en esos catálogos, no en el código.

const accionConsultarSeguimientoCeseDesarrollo = "contratacion_temporal.seguimiento.consultar"
const envCTReincorporacionTitularEnabled = "VEC_CT_REINCORPORACION_TITULAR_ENABLED"

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
		// Confirmación de GINPIX (CT124): solo se monta con la incorporación
		// acreditada encendida; sin ella la ruta no existe.
		{httpinterno.RutaConfirmacionesGINPIX, "ct-ginpix-confirmar", string(domain.AccionConfirmarGINPIX), ports.TipoRecursoConfirmacionGINPIX, ports.FinalidadConfirmarGINPIX, ports.AudienciaConsumoConfirmacionGINPIXV1, true},
		// No incorporación (CT124): mismo selector que la anterior.
		{httpinterno.RutaNoIncorporaciones, "ct-no-incorporacion-registrar", string(domain.AccionRegistrarNoIncorporacion), ports.TipoRecursoNoIncorporacion, ports.FinalidadRegistrarNoIncorporacion, ports.AudienciaConsumoNoIncorporacionV1, true},
	}
}

func operacionSeguimientoCesePorRuta(ruta string) (operacionSeguimientoCeseDesarrollo, bool) {
	if ruta == httpinterno.RutaReincorporacionesTitular {
		return operacionSeguimientoCeseDesarrollo{ruta, "ct-reincorporacion-titular-registrar", string(domain.AccionRegistrarReincorporacionTitular),
			ports.TipoRecursoReincorporacionTitular, ports.FinalidadRegistrarReincorporacionTitular,
			ports.AudienciaConsumoReincorporacionTitularV1, true}, true
	}
	if ruta == httpinterno.RutaCapacidadReincorporacionTitular {
		return operacionSeguimientoCeseDesarrollo{ruta, "ct-reincorporacion-titular-capacidad", accionConsultarSeguimientoCeseDesarrollo,
			"seguimiento_contratacion_temporal", "gestionar_contratacion_temporal", "", false}, true
	}
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

// descriptoresFronterasSeguimientoCeseDesarrollo declara sólo las rutas base.
func descriptoresFronterasSeguimientoCeseDesarrollo(perfilCT string) []descriptorFronteraComunDesarrollo {
	var d []descriptorFronteraComunDesarrollo
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		d = append(d, fronteraContratacionTemporalDesarrollo(o.frontera, o.accion, o.ruta, []string{perfilCT}))
	}
	return d
}

func descriptoresFronterasReincorporacionTitularDesarrollo(perfil string) []descriptorFronteraComunDesarrollo {
	var descriptores []descriptorFronteraComunDesarrollo
	for _, ruta := range []string{httpinterno.RutaReincorporacionesTitular, httpinterno.RutaCapacidadReincorporacionTitular} {
		o, _ := operacionSeguimientoCesePorRuta(ruta)
		f := fronteraContratacionTemporalDesarrollo(o.frontera, o.accion, o.ruta, []string{perfil})
		if ruta == httpinterno.RutaCapacidadReincorporacionTitular {
			f.Metodo = http.MethodPost
		}
		descriptores = append(descriptores, f)
	}
	return descriptores
}

func descriptorMaterialReincorporacionTitularDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: ports.AudienciaConsumoReincorporacionTitularV1,
		Dominio: "vec.ct.reincorporacion-titular.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-reincorporacion-titular:",
		ProveedorNominal: proveedorMaterialContratacionTemporal}
}

func descriptorMaterialLecturaReincorporacionTitularDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: ports.AudienciaLecturaReincorporacionTitularV1,
		Dominio: "vec.ct.lectura-reincorporacion-titular.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-lectura-reincorporacion-titular:",
		ProveedorNominal: proveedorMaterialContratacionTemporal}
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
// instaladas). El propio selector comprueba la doble llave de desarrollo y los
// tres catálogos de ejemplo; la raíz valida el selector antes de componer, de
// modo que pedirlo sin alguno de ellos detiene el arranque en lugar de dejar
// las rutas sin montar en silencio.
func seguimientoCeseSolicitado(cfg config.Config) bool {
	activo, err := cfg.CTSeguimientoCeseDesarrolloActivo()
	if err != nil {
		slog.Error("seguimiento de cese de CT no compuesto: selector inválido", "causa", err)
		return false
	}
	return activo
}

// operacionIncorporacionAcreditadaDesarrollo distingue las dos rutas de CT124
// (confirmación de GINPIX y no incorporación), que solo existen con la
// incorporación acreditada encendida.
func operacionIncorporacionAcreditadaDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaConfirmacionesGINPIX || ruta == httpinterno.RutaNoIncorporaciones
}

// motivoSeguimientoCeseDesarrollo devuelve el motivo de cada ruta. Las cuatro
// rutas originales siguen en «motivos_seguimiento_cese_ct» v1, tal como ya
// está publicado en las bases existentes: una versión publicada es inmutable y
// añadirle entradas hace que su replay idempotente se rechace al arrancar. Las
// rutas de CT124 van en su propio catálogo.
func motivoSeguimientoCeseDesarrollo(ruta string) vecdomain.ReferenciaEntradaCatalogo {
	if ruta == httpinterno.RutaCapacidadReincorporacionTitular {
		return motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular)
	}
	if ruta == httpinterno.RutaReincorporacionesTitular {
		return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_reincorporacion_titular_ct", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("reincorporacion-titular-ct-desarrollo-v1"),
			EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "reincorporacion-titular-ct")}
	}
	if operacionIncorporacionAcreditadaDesarrollo(ruta) {
		return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_incorporacion_acreditada_ct", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("incorporacion-acreditada-ct-desarrollo-v1"),
			EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "seguimiento-cese-ct:"+ruta)}
	}
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_seguimiento_cese_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("seguimiento-cese-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "seguimiento-cese-ct:"+ruta)}
}

// motivosSeguimientoCesePorCatalogo agrupa los motivos de las operaciones
// compuestas por catálogo, en orden estable, para publicar cada uno aparte.
func motivosSeguimientoCesePorCatalogo(acreditada bool) [][]vecdomain.ReferenciaEntradaCatalogo {
	var orden []string
	grupos := map[string][]vecdomain.ReferenciaEntradaCatalogo{}
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		if operacionIncorporacionAcreditadaDesarrollo(o.ruta) && !acreditada {
			continue
		}
		m := motivoSeguimientoCeseDesarrollo(o.ruta)
		if _, ok := grupos[m.CatalogoID]; !ok {
			orden = append(orden, m.CatalogoID)
		}
		grupos[m.CatalogoID] = append(grupos[m.CatalogoID], m)
	}
	resultado := make([][]vecdomain.ReferenciaEntradaCatalogo, 0, len(orden))
	for _, id := range orden {
		resultado = append(resultado, grupos[id])
	}
	return resultado
}

// soporteSeguimientoCeseDesarrollo es la instantánea de desarrollo, no
// autoritativa, con las cuatro concesiones nominales.
type soporteSeguimientoCeseDesarrollo struct {
	instantanea                vecdomain.InstantaneaAutorizacion
	contexto                   ports.ContextoAutorizacionAltaV3
	contextoEsperadoRegistrado vecdomain.ResultadoContextoActorRegistradoV2
	sesionOperativa            proveedorSesionOperativaCTDesarrollo
}

func discriminadorContextoReincorporacionTitularDesarrollo() discriminadorContextoSinteticoDesarrollo {
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "reincorporacion-titular-perfil", vinculo: "reincorporacion-titular-vinculo",
		// La cuenta y la persona son las mismas autoridades ya preparadas.
		procedencia: "procedencia", registro: "reincorporacion-titular-registro-contexto",
		autenticacion: "reincorporacion-titular-autenticacion", asercion: "reincorporacion-titular-asercion",
		sesion: "reincorporacion-titular-sesion", controlSesion: "reincorporacion-titular-control-sesion",
		politicaGarantia: "reincorporacion-titular-politica-garantia",
	}
}

func nuevoContextoReincorporacionTitularDesarrollo(soporte *soporteAltaContratacionTemporalDesarrollo, ahora time.Time) (ports.ContextoAutorizacionAltaV3, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	if soporte == nil || soporte.principalID == "" || !huellaSHA256ValidaContratacionTemporalDesarrollo(soporte.certificadoSHA256) {
		return vacio, errSeguimientoCeseDesarrolloNoDisponible
	}
	base := soporte.contexto
	if base.Resultado.Validar() != nil || base.Vinculo.ValidarPara(base.Resultado) != nil {
		return vacio, errSeguimientoCeseDesarrolloNoDisponible
	}
	// El contexto V3 no transporta los atributos de la identidad mTLS que
	// iniciaron la semilla. Se reconstruye sólo desde el soporte ya sellado.
	principal := vecdomain.Principal{ID: soporte.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": soporte.certificadoSHA256}}
	if !principalContratacionTemporalDesarrolloValido(principal) {
		return vacio, errSeguimientoCeseDesarrolloNoDisponible
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoReincorporacionTitularDesarrollo())
	if err != nil || contexto.Resultado.Contexto.PerfilActivoRef == base.Resultado.Contexto.PerfilActivoRef ||
		contexto.Resultado.Contexto.Instantanea.CuentaRef != base.Resultado.Contexto.Instantanea.CuentaRef ||
		contexto.Resultado.Contexto.PersonaRef != base.Resultado.Contexto.PersonaRef {
		return vacio, errSeguimientoCeseDesarrolloNoDisponible
	}
	return contexto, nil
}

// El contexto nominal sólo se usa para preparar las autoridades al arrancar.
// Las peticiones obtienen un vínculo fresco mediante contextoOperativoDesarrollo.
func (s *soporteAltaContratacionTemporalDesarrollo) contextoReincorporacionTitular(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	if s == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	s.mu.Lock()
	reincorporacion := s.reincorporacionTitular
	s.mu.Unlock()
	if reincorporacion == nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	datos, err := reincorporacion.contexto.Vinculo.Datos()
	if err != nil || reincorporacion.contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef, PerfilRef: datos.PerfilActivoRef,
	}, s.reloj.Ahora()) != nil || datos.PerfilActivoRef == s.contexto.Resultado.Contexto.PerfilActivoRef {
		return vacio, ports.ErrAutorizacionDenegada
	}
	resultado, err := reincorporacion.contexto.Resultado.Clonar()
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: reincorporacion.contexto.Vinculo, Resultado: resultado}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaSeguimientoCese(ruta ...string) (vecdomain.InstantaneaAutorizacion, bool) {
	if len(ruta) != 0 && (ruta[0] == httpinterno.RutaReincorporacionesTitular || ruta[0] == httpinterno.RutaCapacidadReincorporacionTitular) {
		if s == nil || s.reincorporacionTitular == nil {
			return vecdomain.InstantaneaAutorizacion{}, false
		}
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.reincorporacionTitular.instantanea),
			s.reincorporacionTitular.instantanea.Validar() == nil
	}
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
	if s != nil && rutaReincorporacionTitularDesarrollo(ruta) {
		if s.perfilFijoParaRuta(ruta) == nil || !solicitudAutorizacionReincorporacionTitularValida(ruta, d) {
			return nil, false
		}
		return []vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}}, true
	}
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

func configurarSoporteSeguimientoCeseDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, acreditada bool, reincorporacion ...bool) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	var concesiones []vecdomain.ConcesionRol
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		if operacionIncorporacionAcreditadaDesarrollo(o.ruta) && !acreditada {
			continue
		}
		concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: o.accion, ModuloID: ports.ModuloContratacion, TipoRecurso: o.tipo,
			Finalidades: []string{o.finalidad}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
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
	for _, motivos := range motivosSeguimientoCesePorCatalogo(acreditada) {
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, motivos, desde); err != nil {
			return err
		}
	}
	alta.soporte.mu.Lock()
	alta.soporte.seguimientoCese = &soporteSeguimientoCeseDesarrollo{instantanea: instantanea}
	alta.soporte.mu.Unlock()
	if len(reincorporacion) != 0 && reincorporacion[0] {
		alta.soporte.mu.Lock()
		soporteReincorporacion := alta.soporte.reincorporacionTitular
		proveedorBase, proveedorValido := alta.soporte.sesionOperativa.(*proveedorSesionConsultaRRHHDesarrollo)
		alta.soporte.mu.Unlock()
		if soporteReincorporacion == nil || !proveedorValido || proveedorBase == nil {
			return errSeguimientoCeseDesarrolloNoDisponible
		}
		contextoNominal, err := alta.soporte.contextoReincorporacionTitular(ctx)
		if err != nil {
			return err
		}
		vinculoReincorporacion, err := contextoNominal.Vinculo.Datos()
		if err != nil || vinculoReincorporacion.PerfilActivoRef == v.PerfilActivoRef {
			return errSeguimientoCeseDesarrolloNoDisponible
		}
		acciones := []vecdomain.ConcesionRol{
			{Accion: accionConsultarSeguimientoCeseDesarrollo, ModuloID: ports.ModuloContratacion,
				TipoRecurso: "seguimiento_contratacion_temporal", Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: string(domain.AccionRegistrarReincorporacionTitular), ModuloID: ports.ModuloContratacion,
				TipoRecurso: ports.TipoRecursoReincorporacionTitular, Finalidades: []string{ports.FinalidadRegistrarReincorporacionTitular}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: string(ports.AccionConsultarAntecedenteReincorporacionTitular), ModuloID: ports.ModuloContratacion,
				TipoRecurso: ports.TipoRecursoLecturaReincorporacionTitular, Finalidades: []string{ports.FinalidadLecturaReincorporacionTitular},
				CamposPermitidos: []string{"cese_evento_ref", "cese_recibo_ref", "documento_ref", "documento_sha256", "existe_cese", "fecha_efectiva", "relacion_ref"},
				GarantiaMinima:   vecdomain.AuthAssuranceHigh},
		}
		otra, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculoReincorporacion.PrincipalID, vinculoReincorporacion.PerfilActivoRef, reloj.Ahora(),
			"reincorporacion_titular_ct_desarrollo", "Reincorporación titular de desarrollo", "reincorporacion-titular-ct-desarrollo", acciones,
			[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		if err != nil {
			return err
		}
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
			[]vecdomain.ReferenciaEntradaCatalogo{motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular)}, desde); err != nil {
			return err
		}
		soporteReincorporacion.instantanea = otra
		if err := publicarContextoPostgreSQLReincorporacionTitularDesarrollo(ctx, alta.postgresql.gobierno, alta.soporte); err != nil {
			return err
		}
		esperado, err := contextoEsperadoRegistradoParaSemillaDesarrollo(ctx, proveedorBase.resolutor, alta.soporte, contextoNominal.Resultado)
		if err != nil {
			return err
		}
		sesion, err := nuevaSesionReincorporacionTitularDesarrollo(proveedorBase, esperado)
		if err != nil {
			return err
		}
		alta.soporte.mu.Lock()
		soporteReincorporacion.contextoEsperadoRegistrado = esperado
		soporteReincorporacion.sesionOperativa = sesion
		alta.soporte.mu.Unlock()
	}
	return nil
}

// autoridadSeguimientoCeseDesarrollo liga cada petición a la capacidad mTLS de
// su ruta, exige la decisión V3 con la instantánea de desarrollo y entrega el
// material atestado para la audiencia de la operación.
type autoridadSeguimientoCeseDesarrollo struct {
	alta                   *dependenciasAltaContratacionTemporalDesarrollo
	proveedores            map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
	lecturaReincorporacion *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
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
	if sol.Accion == ports.AccionConsultarAntecedenteReincorporacionTitular {
		s, d, c, resultado, err := a.exigirLecturaAntecedenteReincorporacion(ctx, sol)
		if err != nil {
			return vacio, err
		}
		return proveedor.proveerMaterialConfirmacion(ctx, s, d, c, sol.Motivo, resultado)
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

func (a *autoridadSeguimientoCeseDesarrollo) exigirLecturaAntecedenteReincorporacion(ctx context.Context, sol ports.SolicitudAutorizarOperacionSeguimiento) (vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2, error) {
	var s vecdomain.SolicitudAutorizacionLigadaV3
	var d vecdomain.DecisionAutorizacionLigadaV3
	var c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	var r vecdomain.ResultadoContextoActorRegistradoV2
	if a == nil || a.alta == nil || a.alta.soporte == nil || a.lecturaReincorporacion == nil || ctx == nil ||
		sol.Finalidad != ports.FinalidadLecturaReincorporacionTitular || sol.Audiencia != ports.AudienciaLecturaReincorporacionTitularV1 ||
		sol.Motivo != motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular) ||
		a.alta.soporte.perfilFijoParaRuta(httpinterno.RutaReincorporacionesTitular) == nil {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	res := sol.Recurso
	if !valida || capacidad.ruta != httpinterno.RutaReincorporacionesTitular || res.ModuloID != ports.ModuloContratacion ||
		res.Tipo != ports.TipoRecursoLecturaReincorporacionTitular || !domain.ReferenciaOpacaValida(res.Referencia) ||
		len(res.Ambitos) != 1 || res.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo || len(res.Atributos) != 5 ||
		res.Atributos["version_expediente"] == "" || res.Atributos["relacion_ref"] == "" ||
		res.Atributos["fecha_efectiva"] == "" || res.Atributos["documento_ref"] == "" || res.Atributos["documento_sha256"] == "" {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	operativo, err := a.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return s, d, c, r, err
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo: sol.Motivo, Accion: string(sol.Accion), Recurso: res, Finalidad: sol.Finalidad, Correlacion: correlacion}
	s, err = vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return s, d, c, r, ports.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	d, c, err = a.lecturaReincorporacion.ExigirSolicitudLigadaV3(ctx, s, operativo.Resultado)
	return s, d, c, operativo.Resultado, err
}

// AutorizarLecturaSeguimiento exige una decisión de lectura del expediente
// exacto; no emite capacidad de consumo ni escribe ningún efecto.
func (a *autoridadSeguimientoCeseDesarrollo) AutorizarLecturaSeguimiento(ctx context.Context, organizacionRef, expedienteRef string) error {
	if organizacionRef != organizacionAltaContratacionTemporalDesarrollo || !domain.ReferenciaOpacaValida(expedienteRef) {
		return ports.ErrAutorizacionDenegada
	}
	if a == nil || a.alta == nil || a.alta.soporte == nil {
		return ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	if !valida {
		return ports.ErrAutorizacionDenegada
	}
	if rutaReincorporacionTitularDesarrollo(capacidad.ruta) {
		return a.autorizarLecturaReincorporacion(ctx, organizacionRef, expedienteRef)
	}
	_, _, _, _, err := a.exigir(ctx, accionConsultarSeguimientoCeseDesarrollo, "gestionar_contratacion_temporal", vecdomain.RecursoAutorizable{
		Referencia: expedienteRef, ModuloID: ports.ModuloContratacion, Tipo: "seguimiento_contratacion_temporal",
		Ambitos:   map[string]string{"organizacion_ref": organizacionRef, "expediente_ref": expedienteRef},
		Atributos: map[string]string{"lectura": "cese_cierre_opciones"}})
	return err
}

func (a *autoridadSeguimientoCeseDesarrollo) autorizarLecturaReincorporacion(ctx context.Context, organizacionRef, expedienteRef string) error {
	if a.lecturaReincorporacion == nil || a.alta.soporte.perfilFijoParaRuta(httpinterno.RutaReincorporacionesTitular) == nil {
		return ports.ErrAutorizacionDenegada
	}
	operativo, err := a.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return ports.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return err
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo:          motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular),
		Accion:                    accionConsultarSeguimientoCeseDesarrollo,
		Recurso: vecdomain.RecursoAutorizable{Referencia: expedienteRef, ModuloID: ports.ModuloContratacion,
			Tipo: "seguimiento_contratacion_temporal", Ambitos: map[string]string{
				"organizacion_ref": organizacionRef},
			Atributos: map[string]string{"lectura": "reincorporacion_titular"}},
		Finalidad: "gestionar_contratacion_temporal", Correlacion: correlacion}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return ports.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	_, _, err = a.lecturaReincorporacion.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	return err
}

// comprobadorCapacidadReincorporacionTitularDesarrollo evalúa una concesión
// preliminar sin registrar material consumible. El POST repite la autorización
// con el documento, la relación y la fecha exactos antes del efecto.
type comprobadorCapacidadReincorporacionTitularDesarrollo struct {
	autoridad  *autoridadSeguimientoCeseDesarrollo
	lector     ports.LectorEstadoSeguimiento
	preparador *aplicacionvec.ServicioPreparacionSolicitudLigadaV3
}

func (c *comprobadorCapacidadReincorporacionTitularDesarrollo) ComprobarCapacidadReincorporacionTitular(
	ctx context.Context, canal application.ContextoCanalSeguimiento, expedienteRef string, version uint64,
) (bool, error) {
	if c == nil || c.autoridad == nil || c.autoridad.alta == nil || c.autoridad.alta.soporte == nil ||
		c.lector == nil || c.preparador == nil || ctx == nil || ctx.Err() != nil || !canal.Valido() ||
		!domain.ReferenciaOpacaValida(expedienteRef) || version == 0 {
		return false, ports.ErrAutorizacionDenegada
	}
	if err := c.autoridad.AutorizarLecturaSeguimiento(ctx, canal.OrganizacionRef, expedienteRef); err != nil {
		return false, err
	}
	estado, err := c.lector.ConsultarEstadoSeguimiento(ctx, canal.OrganizacionRef, expedienteRef)
	if err != nil {
		return false, err
	}
	if estado.ExpedienteRef != expedienteRef || estado.Cese == nil || estado.Cese.CausaClave != "fin_sustitucion" {
		return false, nil
	}
	operativo, err := c.autoridad.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return false, ports.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return false, err
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: expedienteRef, ModuloID: ports.ModuloContratacion,
		Tipo:    ports.TipoRecursoReincorporacionTitular,
		Ambitos: map[string]string{"organizacion_ref": canal.OrganizacionRef},
		Atributos: map[string]string{"version_expediente": strconv.FormatUint(version, 10),
			"fase_previa": string(domain.FaseNombramiento), "estado_previo": string(domain.EstadoEnCurso)}}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular),
		Accion: string(domain.AccionRegistrarReincorporacionTitular), Recurso: recurso,
		Finalidad: ports.FinalidadRegistrarReincorporacionTitular, Correlacion: correlacion}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return false, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	_, _, err = c.preparador.PrepararSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return false, nil
	}
	return err == nil, err
}

// fuenteReglasSeguimientoDesarrollo lee los catálogos de ejemplo: causas de
// cese (catálogo propio), reglas c09/c10/c11 y motivos de rectificación.
type fuenteReglasSeguimientoDesarrollo struct {
	reglas  *reglas.Resolutor
	causas  puertosvec.ConsultaCatalogosConfigurables
	motivos puertosvec.ConsultaCatalogosConfigurables
}

// CT130 comparte el catálogo de causas con cese, pero su decisión V3 exige
// el motivo de la ruta de reincorporación. La definición y huella permanecen
// ligadas a la versión publicada del catálogo.
type fuenteReglasReincorporacionTitularDesarrollo struct {
	ports.FuenteReglasSeguimiento
}

func (f fuenteReglasReincorporacionTitularDesarrollo) CausasCese(ctx context.Context, instante time.Time) ([]ports.CausaCese, ports.PoliticaOperacionSeguimiento, error) {
	causas, politica, err := f.FuenteReglasSeguimiento.CausasCese(ctx, instante)
	if err != nil {
		return nil, ports.PoliticaOperacionSeguimiento{}, err
	}
	politica.MotivoAutorizacion = motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular)
	return causas, politica, nil
}

func politicaSeguimientoDesarrollo(regla reglas.Regla, ruta string, instante time.Time) ports.PoliticaOperacionSeguimiento {
	return ports.PoliticaOperacionSeguimiento{DefinicionRef: regla.ReferenciaEntrada.CatalogoID, DefinicionVersion: uint64(regla.ReferenciaEntrada.CatalogoVersion),
		DefinicionHuellaSHA256: regla.ReferenciaEntrada.CatalogoHuellaSHA256, MotivoAutorizacion: motivoSeguimientoCeseDesarrollo(ruta),
		EvaluadaEn: instante, ValidaHasta: instante.Add(2 * time.Minute)}
}

func (f fuenteReglasSeguimientoDesarrollo) PoliticaLecturaReincorporacionTitular(ctx context.Context, instante time.Time) (ports.PoliticaOperacionSeguimiento, error) {
	_, politica, err := f.CausasCese(ctx, instante)
	if err != nil {
		return ports.PoliticaOperacionSeguimiento{}, err
	}
	politica.MotivoAutorizacion = motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular)
	return politica, nil
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
// declarado los catálogos, y la confirmación de GINPIX si la incorporación
// acreditada está encendida. Sin ellos, no hay rutas: la conducta es la de hoy.
// finPersonal, opcional, termina en Personal B2 la relación del expediente
// después de cada cese confirmado; nil conserva el cese solo de CT.
func nuevasRutasSeguimientoCeseDesarrollo(dependencias *DependenciasCT, alta *dependenciasAltaContratacionTemporalDesarrollo, finPersonal *finCesePersonalB2Desarrollo) ([]vechttp.RutaExacta, error) {
	if dependencias == nil || !seguimientoCeseSolicitado(dependencias.cfg) {
		return nil, nil
	}
	cfg, derivador, reloj := dependencias.cfg, dependencias.derivador, dependencias.reloj
	reincorporacion, err := selectorCapacidadRRHHDesarrollo(cfg, envCTReincorporacionTitularEnabled)
	if err != nil {
		return nil, err
	}
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
	var preparador *aplicacionvec.ServicioPreparacionSolicitudLigadaV3
	if reincorporacion {
		preparador, err = aplicacionvec.NuevoServicioPreparacionSolicitudLigadaV3(alta.soporte, alta.soporte, alta.soporte,
			reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
		if err != nil {
			return fallar("preflight_reincorporacion", err)
		}
		var instalada bool
		if err := alta.postgresql.ejecucion.QueryRow(ctx, `SELECT
			to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()') IS NOT NULL AND
			to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NOT NULL AND
			to_regprocedure('vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)') IS NOT NULL AND
			to_regprocedure('vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND
			to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND
			to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(&instalada); err != nil || !instalada {
			return fallar("migracion_reincorporacion", errSeguimientoCeseDesarrolloNoDisponible)
		}
		if err := alta.postgresql.ejecucion.QueryRow(ctx,
			`SELECT vec_contratacion_temporal.perfil_reincorporacion_ct153_v1() = 'organizacion_ref'`).Scan(&instalada); err != nil || !instalada {
			return fallar("perfil_reincorporacion_ct153", errSeguimientoCeseDesarrolloNoDisponible)
		}
	}
	acreditada := incorporacionAcreditadaSolicitada(cfg)
	if err := configurarSoporteSeguimientoCeseDesarrollo(ctx, alta, reloj, acreditada, reincorporacion); err != nil {
		return fallar("instantanea", err)
	}
	if reincorporacion {
		if err := componerPerfilFijoReincorporacionTitular(ctx, alta.postgresql.gobierno, alta.soporte,
			aprobacionProvisionPerfilesRRHHDesdeConfig(cfg)); err != nil {
			return fallar("perfil_fijo_reincorporacion", err)
		}
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return fallar("material", err)
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	autoridad := &autoridadSeguimientoCeseDesarrollo{alta: alta, proveedores: map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{}}
	if reincorporacion {
		autoridad.lecturaReincorporacion, err = aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
			alta.soporte, alta.soporte, alta.soporte, alta.soporte, reloj,
			seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
		if err != nil {
			return fallar("lectura_reincorporacion", err)
		}
	}
	descriptores := descriptoresMaterialSeguimientoCeseDesarrollo()
	if acreditada {
		if err := comprobarMigracionesIncorporacionAcreditadaDesarrollo(ctx, alta.postgresql.ejecucion); err != nil {
			return fallar("migraciones_incorporacion_acreditada", err)
		}
		descriptores = append(descriptores, descriptorMaterialConfirmacionGINPIXDesarrollo(), descriptorMaterialNoIncorporacionDesarrollo())
	}
	if reincorporacion {
		descriptores = append(descriptores, descriptorMaterialReincorporacionTitularDesarrollo(), descriptorMaterialLecturaReincorporacionTitularDesarrollo())
	}
	for _, d := range descriptores {
		p, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, alta.postgresql.catalogoMaterial, d.Audiencia)
		if err != nil {
			return fallar("proveedor_material", err)
		}
		autoridad.proveedores[d.Audiencia] = p
	}
	var llaveros []seguridadct.ConfiguracionLlaverosSeguimiento
	for _, operacion := range []string{ports.OperacionRegistrarCese, ports.OperacionCerrarExpediente, ports.OperacionModificarTrasNombramiento, ports.OperacionConfirmarGINPIX,
		ports.OperacionRegistrarNoIncorporacion} {
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
	fuente := fuenteReglasSeguimientoDesarrollo{reglas: resolutor, causas: causas, motivos: motivos}
	dependenciasServicio := application.DependenciasOperacionesSeguimiento{
		Contextos: alta.soporte, Sellos: sellos, Repositorio: repositorio, Reglas: fuente,
		Autorizador: autoridad, Referencias: seguridadct.NuevoGeneradorReferenciasAltaCriptografico(),
		Coste: calculadorCosteModificacionDesarrollo{retribuciones: dependencias.retribucionesCT}, Lector: repositorio, Reloj: reloj}
	if acreditada {
		dependenciasServicio.GINPIX, dependenciasServicio.Acreditada, dependenciasServicio.NoIncorporacion = fuente, repositorio, fuente
	}
	servicio, err := application.NuevoServicioOperacionesSeguimiento(dependenciasServicio)
	if err != nil {
		return fallar("servicio", err)
	}
	manejadores, err := httpinterno.NuevosManejadoresSeguimiento(autoridad, autoridad, finPersonal.envolver(servicio, repositorio))
	if err != nil {
		return fallar("http", err)
	}
	rutas := make([]vechttp.RutaExacta, 0, len(manejadores))
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		if operacionIncorporacionAcreditadaDesarrollo(o.ruta) && !acreditada {
			continue
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: o.ruta, Manejador: manejadores[o.ruta]})
	}
	if reincorporacion {
		ambito, retenidasAmbito, err := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, ports.DominioAmbitoReincorporacionTitular, true)
		if err != nil {
			return fallar("sellos_reincorporacion", err)
		}
		huella, retenidasHuella, err := configuracionesHMACAltaContratacionTemporalDesarrollo(derivador, ports.DominioHuellaReincorporacionTitular, false)
		if err != nil {
			return fallar("sellos_reincorporacion", err)
		}
		sellosRetorno, err := seguridadct.NuevaAutoridadSellosReincorporacionTitularHMAC(ambito, huella, retenidasAmbito, retenidasHuella)
		if err != nil {
			return fallar("sellos_reincorporacion", err)
		}
		repositorioRetorno, err := postgresct.NuevoRepositorioReincorporacionTitularPostgreSQL(alta.postgresql.ejecucion)
		if err != nil {
			return fallar("repositorio_reincorporacion", err)
		}
		lectorRetorno, err := postgresct.NuevoLectorAntecedenteReincorporacionTitularPostgreSQL(alta.postgresql.ejecucion)
		if err != nil {
			return fallar("lector_reincorporacion", err)
		}
		rutasRetorno, err := ComponerRutasReincorporacionTitular(autoridad, application.DependenciasReincorporacionTitular{
			Contextos: alta.soporte, Sellos: sellosRetorno, Repositorio: repositorioRetorno,
			Reglas:      fuenteReglasReincorporacionTitularDesarrollo{FuenteReglasSeguimiento: fuente},
			Autorizador: autoridad, Lector: lectorRetorno, PoliticaLectura: fuente,
			Referencias: seguridadct.NuevoGeneradorReferenciasAltaCriptografico(), Reloj: reloj},
			&comprobadorCapacidadReincorporacionTitularDesarrollo{autoridad: autoridad, lector: repositorio, preparador: preparador})
		if err != nil {
			return fallar("composicion_reincorporacion", err)
		}
		rutas = append(rutas, rutasRetorno...)
	}
	return rutas, nil
}
