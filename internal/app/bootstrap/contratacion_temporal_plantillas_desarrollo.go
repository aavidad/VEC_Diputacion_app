package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	plantillaspg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres/plantillascatalogo"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	docxvec "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	pdfvec "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// rutaPlantillasCTEjemplo es el catálogo de plantillas que trae el
// repositorio. VEC_CT_PLANTILLAS_SOURCE_PATH lo sustituye por otro con el
// mismo formato; en ambos casos se valida entero al arrancar.
const rutaPlantillasCTEjemplo = "data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json"

var errPlantillasCTNoDisponibles = errors.New("bootstrap: catalogo de plantillas de contratacion temporal no disponible")

// cargarPlantillasBorradorCTDesarrollo lee y valida el catálogo. Un catálogo
// ilegible, con campos desconocidos o fuera de límites impide arrancar en
// lugar de servir documentos con texto incompleto.
func cargarPlantillasBorradorCTDesarrollo(cfg config.Config, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	reglas, _, err := cfg.ReglasEjemploDesarrollo()
	if err != nil {
		return nil, err
	}
	candidatas := []string{reglas.CTPlantillasSourcePath}
	if reglas.CTPlantillasSourcePath == "" {
		candidatas = []string{rutaPlantillasCTEjemplo, "../../../" + rutaPlantillasCTEjemplo}
	}
	for _, ruta := range candidatas {
		if _, err := os.Stat(ruta); os.IsNotExist(err) && reglas.CTPlantillasSourcePath == "" {
			continue
		}
		return plantillasBorradorCTDesdeFichero(ruta, instante)
	}
	return nil, errPlantillasCTNoDisponibles
}

func plantillasBorradorCTDesdeFichero(ruta string, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	catalogo, err := CargarCatalogoPlantillasCT(ruta)
	if err != nil {
		return nil, err
	}
	plantillas, err := informejuridico.NuevasPlantillasBorrador(catalogo, instante)
	if err != nil {
		return nil, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	return plantillas, nil
}

// CargarCatalogoPlantillasCT devuelve exactamente la versión publicada que
// consumen el arranque y la CLI de provisión. La huella se calcula después
// con CatalogoConfigurable.HuellaSHA256, sobre el contenido canónico validado.
func CargarCatalogoPlantillasCT(ruta string) (vecdomain.CatalogoConfigurable, error) {
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	versiones, err := consulta.ListarVersionesCatalogo(ctx, informejuridico.CatalogoPlantillasBorradorID)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	var elegido *vecdomain.CatalogoConfigurable
	for i := range versiones {
		if versiones[i].Estado == vecdomain.EstadoCatalogoPublicado && (elegido == nil || versiones[i].Version > elegido.Version) {
			elegido = &versiones[i]
		}
	}
	if elegido == nil {
		return vecdomain.CatalogoConfigurable{}, errPlantillasCTNoDisponibles
	}
	if err := informejuridico.ValidarCatalogoPlantillasBorrador(*elegido); err != nil {
		return vecdomain.CatalogoConfigurable{}, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	copia, err := elegido.ClonarCanonico()
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	return copia, nil
}

const envCTPlantillasGobiernoEnabled = "VEC_CT_PLANTILLAS_GOBIERNO_ENABLED"
const envCTPlantillasDocumentalEnabled = "VEC_CT_PLANTILLAS_DOCUMENTAL_ENABLED"

func plantillasCatalogoCTDesarrolloSolicitado(cfg config.Config) (bool, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envCTPlantillasGobiernoEnabled)
	if err != nil || activo && cfg.Normalize().ReglasEjemplo.CTPlantillasSourcePath == "" {
		return false, ErrActivacionDesarrolloInvalida
	}
	return activo, nil
}

func plantillasDocumentalCTDesarrolloSolicitado(cfg config.Config) (bool, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envCTPlantillasDocumentalEnabled)
	if err != nil || activo && cfg.Normalize().ReglasEjemplo.CTPlantillasSourcePath == "" {
		return false, ErrActivacionDesarrolloInvalida
	}
	return activo, nil
}

func rutaPlantillasCatalogoCTDesarrollo(ruta string) bool {
	return ruta == plantillashttp.RutaCatalogo || ruta == plantillashttp.RutaEntradas || ruta == plantillashttp.RutaPublicar
}

func rutaPlantillasDocumentalCTDesarrollo(ruta string) bool {
	return ruta == plantillashttp.RutaBorradoresDisponibles || ruta == plantillashttp.RutaBorradores
}

func discriminadorContextoPlantillasCatalogoCTDesarrollo() discriminadorContextoSinteticoDesarrollo {
	const etiqueta = "plantillas-catalogo-ct-v1"
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-" + etiqueta, vinculo: "vinculo-" + etiqueta,
		procedencia: "procedencia", registro: "registro-contexto-" + etiqueta,
		autenticacion: "autenticacion-" + etiqueta, asercion: "asercion-" + etiqueta,
		sesion: "sesion-" + etiqueta, controlSesion: "control-sesion-" + etiqueta,
		politicaGarantia: "politica-garantia-" + etiqueta,
	}
}

func discriminadorContextoPlantillasDocumentalCTDesarrollo() discriminadorContextoSinteticoDesarrollo {
	const etiqueta = "plantillas-documental-ct-v1"
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-" + etiqueta, vinculo: "vinculo-" + etiqueta,
		procedencia: "procedencia", registro: "registro-contexto-" + etiqueta,
		autenticacion: "autenticacion-" + etiqueta, asercion: "asercion-" + etiqueta,
		sesion: "sesion-" + etiqueta, controlSesion: "control-sesion-" + etiqueta,
		politicaGarantia: "politica-garantia-" + etiqueta,
	}
}

// El principal procede de la hoja mTLS ya aceptada por CT. El perfil y
// vínculo son distintos de la tramitación y de la auditoría RRHH.
func nuevoSoportePlantillasCatalogoCTDesdeBaseDesarrollo(base *soporteAltaContratacionTemporalDesarrollo, ahora time.Time) (*soporteAltaContratacionTemporalDesarrollo, string, error) {
	return nuevoSoportePlantillasCTDesdeBaseDesarrollo(base, ahora, discriminadorContextoPlantillasCatalogoCTDesarrollo())
}

func nuevoSoportePlantillasDocumentalCTDesdeBaseDesarrollo(base *soporteAltaContratacionTemporalDesarrollo, ahora time.Time) (*soporteAltaContratacionTemporalDesarrollo, string, error) {
	return nuevoSoportePlantillasCTDesdeBaseDesarrollo(base, ahora, discriminadorContextoPlantillasDocumentalCTDesarrollo())
}

func nuevoSoportePlantillasCTDesdeBaseDesarrollo(base *soporteAltaContratacionTemporalDesarrollo, ahora time.Time,
	discriminador discriminadorContextoSinteticoDesarrollo) (*soporteAltaContratacionTemporalDesarrollo, string, error) {
	if base == nil || !ctdomain.InstanteUTCCanonico(ahora) {
		return nil, "", plantillasapp.ErrNoDisponible
	}
	base.mu.Lock()
	principalID, certificado := base.principalID, base.certificadoSHA256
	contextoBase, sello, reloj := base.contexto, base.sello, base.reloj
	base.mu.Unlock()
	if sello == nil || !identificadorSesionDesarrolloValido(principalID) ||
		!contextoSinteticoCTConsistenteParaBorradorBolsa(principalID, certificado, contextoBase) {
		return nil, "", plantillasapp.ErrNoDisponible
	}
	principal := vecdomain.Principal{
		ID: principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": certificado},
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminador)
	if err != nil || !contextoSinteticoBolsaSeparadoDeCT(contextoBase, contexto) {
		return nil, "", plantillasapp.ErrNoDisponible
	}
	perfil := contexto.Resultado.Contexto.PerfilActivoRef
	if !perfilActivoSeguridadComunValido(perfil) {
		return nil, "", plantillasapp.ErrNoDisponible
	}
	return &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: principalID, certificadoSHA256: certificado,
		contexto: contexto, reloj: reloj,
	}, perfil, nil
}

func descriptoresFronterasPlantillasCTDesarrollo(perfil string) []descriptorFronteraComunDesarrollo {
	return []descriptorFronteraComunDesarrollo{
		{Clave: "ct-plantillas-consultar", Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: plantillashttp.RutaCatalogo, PerfilesActivosRef: []string{perfil},
			ClavePolitica:  clavePoliticaContratacionTemporalDesarrollo,
			ClaveCapacidad: "contratacion_temporal.plantillas_documentos.consultar"},
		fronteraContratacionTemporalDesarrollo("ct-plantillas-editar", "contratacion_temporal.plantillas_documentos.editar", plantillashttp.RutaEntradas, []string{perfil}),
		fronteraContratacionTemporalDesarrollo("ct-plantillas-publicar", "contratacion_temporal.plantillas_documentos.publicar", plantillashttp.RutaPublicar, []string{perfil}),
	}
}

func descriptoresFronterasPlantillasDocumentalCTDesarrollo(perfil string) []descriptorFronteraComunDesarrollo {
	const politica = "ct-plantillas-documental-v3"
	return []descriptorFronteraComunDesarrollo{
		{Clave: "ct-plantillas-documental-listar", Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: plantillashttp.RutaBorradoresDisponibles, PerfilesActivosRef: []string{perfil},
			ClavePolitica: politica, ClaveCapacidad: "contratacion_temporal.plantillas_documentos.documental_listar"},
		{Clave: "ct-plantillas-documental-descargar", Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: plantillashttp.RutaBorradores, PerfilesActivosRef: []string{perfil},
			ClavePolitica: politica, ClaveCapacidad: "contratacion_temporal.plantillas_documentos.documental_descargar"},
	}
}

func (c catalogoFronterasComunDesarrollo) contienePerfilesRutasPlantillasCT(perfil string) bool {
	for _, par := range []struct{ metodo, ruta, accion string }{
		{http.MethodGet, plantillashttp.RutaCatalogo, "contratacion_temporal.plantillas_documentos.consultar"},
		{http.MethodPost, plantillashttp.RutaEntradas, "contratacion_temporal.plantillas_documentos.editar"},
		{http.MethodPost, plantillashttp.RutaPublicar, "contratacion_temporal.plantillas_documentos.publicar"},
	} {
		d, ok := c.resolver(par.metodo, par.ruta)
		if !ok || d.ClaveCapacidad != par.accion || !d.admitePerfil(perfil) {
			return false
		}
	}
	return true
}

func (c catalogoFronterasComunDesarrollo) contienePerfilRutasPlantillasDocumentalCT(perfil string) bool {
	for _, par := range []struct{ ruta, accion string }{
		{plantillashttp.RutaBorradoresDisponibles, "contratacion_temporal.plantillas_documentos.documental_listar"},
		{plantillashttp.RutaBorradores, "contratacion_temporal.plantillas_documentos.documental_descargar"},
	} {
		d, ok := c.resolver(http.MethodPost, par.ruta)
		if !ok || d.ClaveCapacidad != par.accion || !d.admitePerfil(perfil) {
			return false
		}
	}
	return true
}

func descriptoresMaterialPlantillasCTDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: audienciaCatalogoPlantillasCT, Dominio: "vec.ct.plantillas-catalogo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-plantillas:", ProveedorNominal: proveedorMaterialContratacionTemporal},
	}
}

func descriptorMaterialPlantillasDocumentalCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        audienciaDocumentalPlantillasCT,
		Dominio:          "vec.ct.plantillas-documental.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:ct-plantillas-doc:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

func motivoDocumentalPlantillasCTDesarrollo() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_plantillas_documental_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("motivos-autorizacion-plantillas-documental-ct-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "consultar-plantillas-documental-ct"),
	}
}

func instantaneaInicialPlantillasDocumentalCTDesarrollo(principal, perfil string, ahora time.Time) (vecdomain.InstantaneaAutorizacion, error) {
	campos := []string{"catalogo", "catalogo_huella_sha256", "contenido_json_sha256", "procedencia_ref", "revision", "version"}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
		"ct_plantillas_documental_desarrollo", "Lectura documental de plantillas CT de desarrollo", "ct-plantillas-documental-desarrollo",
		[]vecdomain.ConcesionRol{
			{Accion: ctports.AccionConsultarDetalleRRHH, ModuloID: plantillasapp.ModuloID,
				TipoRecurso: ctports.TipoRecursoExpediente, Finalidades: []string{ctports.FinalidadConsultarDetalleRRHH}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: "contratacion_temporal.plantillas_documentos.documental_listar", ModuloID: plantillasapp.ModuloID,
				TipoRecurso: tipoDocumentalPlantillasCT, Finalidades: []string{finalidadDocumentalPlantillasCT},
				CamposPermitidos: append([]string(nil), campos...), GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: "contratacion_temporal.plantillas_documentos.documental_descargar", ModuloID: plantillasapp.ModuloID,
				TipoRecurso: tipoDocumentalPlantillasCT, Finalidades: []string{finalidadDocumentalPlantillasCT},
				CamposPermitidos: append([]string(nil), campos...), GarantiaMinima: vecdomain.AuthAssuranceHigh},
		}, []vecdomain.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "clase_ambito", Valores: []string{string(ctports.AmbitoOrganizacionRRHH)}},
			{Clave: "ambito_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
		})
}

func motivoCatalogoPlantillasCTDesarrollo() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion_plantillas_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("motivos-autorizacion-plantillas-ct-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "gobernar-plantillas-ct")}
}

// El abridor compartido existente verifica TLS autenticado, LOGIN único,
// membresía directa sin SET/ADMIN y ausencia de BYPASSRLS. El preflight
// posterior coteja las dos ACL de función y la falta de acceso a tablas.
func abrirPoolAutorizacionRRHHDesarrollo(ctx context.Context, dsn, rol, aplicacion string) (*pgxpool.Pool, error) {
	if rol != config.RolAutorizacionFuenteRRHH && rol != config.RolAutorizacionMotivosEvaluadorRRHH {
		return nil, plantillasapp.ErrNoDisponible
	}
	pool, err := abrirPoolAutoridadAuditoriaDesarrollo(ctx, dsn, rol, aplicacion)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	return pool, nil
}

func preflightAutoridadesPlantillasCT(ctx context.Context, fuente, motivos *pgxpool.Pool) error {
	if ctx == nil || ctx.Err() != nil || fuente == nil || motivos == nil || fuente == motivos {
		return plantillasapp.ErrNoDisponible
	}
	return comprobarPreflightAutoridadesPlantillasCT(ctx, fuente, motivos)
}

func comprobarPreflightAutoridadesPlantillasCT(ctx context.Context, fuente, motivos interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || ctx.Err() != nil || dependenciaEsNulaContratacionTemporalDesarrollo(fuente) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(motivos) {
		return plantillasapp.ErrNoDisponible
	}
	const sonda = `SELECT
     current_user=session_user
     AND pg_catalog.pg_has_role(session_user,$3::regrole,'USAGE')
     AND pg_catalog.to_regprocedure($1) IS NOT NULL AND pg_catalog.to_regprocedure($2) IS NOT NULL
     AND pg_catalog.has_schema_privilege(session_user,'vec_autorizacion','USAGE')
     AND pg_catalog.has_schema_privilege($3::text,'vec_autorizacion','USAGE')
     AND NOT pg_catalog.has_schema_privilege(session_user,'vec_autorizacion','CREATE')
     AND NOT pg_catalog.has_schema_privilege($3::text,'vec_autorizacion','CREATE')
     AND NOT coalesce(pg_catalog.pg_has_role(session_user,pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor'),'MEMBER'),false)
     AND NOT coalesce(pg_catalog.pg_has_role(session_user,pg_catalog.to_regrole('vec_contratacion_temporal_propietario'),'MEMBER'),false)
     AND NOT coalesce(pg_catalog.pg_has_role(session_user,pg_catalog.to_regrole('vec_bolsa_llamamientos_ejecutor'),'MEMBER'),false)
     AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_contratacion_temporal'),'USAGE'),false)
     AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_bolsa_llamamientos'),'USAGE'),false)
     AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3'),'USAGE'),false)
     AND coalesce(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($1)::oid,'EXECUTE'),false)
     AND NOT coalesce(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($2)::oid,'EXECUTE'),false)
     AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_proc p
       JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_contratacion_temporal'
         AND p.proname IN ('operar_catalogo_plantillas_v1',
           'consultar_auditoria_ct_atestada_v1',
           'registrar_auditoria_frontera_ruta_exacta_v1',
           'registrar_auditoria_frontera_auditoria_v1')
         AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))
     AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_class c
       JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
       WHERE n.nspname IN ('vec_autorizacion','vec_contratacion_temporal','vec_bolsa_llamamientos','vec_autorizacion_atestada_v3')
         AND c.relkind IN ('r','p','v','m')
         AND (pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
           OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))`
	const funcionFuente = "vec_autorizacion.obtener_instantanea(text,text)"
	const funcionMotivos = "vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)"
	for _, caso := range []struct {
		pool interface {
			QueryRow(context.Context, string, ...any) pgx.Row
		}
		propia, ajena, rol string
	}{{fuente, funcionFuente, funcionMotivos, config.RolAutorizacionFuenteRRHH},
		{motivos, funcionMotivos, funcionFuente, config.RolAutorizacionMotivosEvaluadorRRHH}} {
		var valida bool
		if err := caso.pool.QueryRow(ctx, sonda, caso.propia, caso.ajena, caso.rol).Scan(&valida); err != nil || !valida {
			return plantillasapp.ErrNoDisponible
		}
	}
	return nil
}

// solicitudAutorizacionPlantillasCTDesarrolloValida se usa también en la
// fuente V3: el proveedor de material no basta para publicar una instantánea.
func solicitudAutorizacionPlantillasCTDesarrolloValida(ruta string, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	if !rutaPlantillasCatalogoCTDesarrollo(ruta) || d.ReferenciaMotivo != motivoCatalogoPlantillasCTDesarrollo() ||
		d.Finalidad != finalidadCatalogoPlantillasCT {
		return false
	}
	accion := "contratacion_temporal.plantillas_documentos.consultar"
	if ruta == plantillashttp.RutaEntradas {
		accion = "contratacion_temporal.plantillas_documentos.editar"
	}
	if ruta == plantillashttp.RutaPublicar {
		accion = "contratacion_temporal.plantillas_documentos.publicar"
	}
	if d.Accion != accion {
		return false
	}
	return recursoCatalogoPlantillasCTValido(accion, d.Recurso, false) ||
		(accion != "contratacion_temporal.plantillas_documentos.consultar" && recursoCatalogoPlantillasCTValido(accion, d.Recurso, true))
}

// La semilla positiva sólo se publica al crear el perfil dedicado. Una
// asignación posterior (también revocada o restringida) gobierna el arranque
// y se lee en vivo por el PDP; nunca se reconstituye desde esta semilla.
func instantaneaInicialPlantillasCatalogoCTDesarrollo(principal, perfil string, ahora time.Time) (vecdomain.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
		"ct_plantillas_catalogo_desarrollo", "Gobierno de plantillas CT de desarrollo", "ct-plantillas-catalogo-desarrollo",
		[]vecdomain.ConcesionRol{
			{Accion: "contratacion_temporal.plantillas_documentos.consultar", ModuloID: plantillasapp.ModuloID,
				TipoRecurso: tipoCatalogoPlantillasCT, Finalidades: []string{finalidadCatalogoPlantillasCT},
				CamposPermitidos: []string{"borrador", "editor_de_esta_version", "publicado"}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: "contratacion_temporal.plantillas_documentos.editar", ModuloID: plantillasapp.ModuloID,
				TipoRecurso: tipoCatalogoPlantillasCT, Finalidades: []string{finalidadCatalogoPlantillasCT},
				CamposPermitidos: []string{"catalogo", "recibo"}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: "contratacion_temporal.plantillas_documentos.publicar", ModuloID: plantillasapp.ModuloID,
				TipoRecurso: tipoCatalogoPlantillasCT, Finalidades: []string{finalidadCatalogoPlantillasCT},
				CamposPermitidos: []string{"catalogo", "recibo"}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
		}, []vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

type publicadorInicialPlantillasCT interface {
	PublicarInicial(context.Context, vecdomain.InstantaneaAutorizacion) error
}

type publicadorPostgreSQLInicialPlantillasCT struct {
	autoridad autoridadPostgreSQLDesarrollo
}

func (p publicadorPostgreSQLInicialPlantillasCT) PublicarInicial(ctx context.Context, semilla vecdomain.InstantaneaAutorizacion) error {
	if !p.autoridad.soloInicial || !p.autoridad.validaConfiguracion() || semilla.Validar() != nil {
		return plantillasapp.ErrNoDisponible
	}
	preparada, err := p.autoridad.prepararInstantanea(ctx, semilla, true)
	if err != nil || preparada.Validar() != nil || preparada.AsignacionPerfil.Version != 1 ||
		preparada.AsignacionPerfil.PerfilActivoRef != semilla.AsignacionPerfil.PerfilActivoRef ||
		preparada.AsignacionPerfil.PrincipalID != semilla.AsignacionPerfil.PrincipalID ||
		preparada.VersionRol.RolID != semilla.VersionRol.RolID ||
		!reflect.DeepEqual(preparada.VersionRol.Concesiones, semilla.VersionRol.Concesiones) ||
		!reflect.DeepEqual(preparada.AsignacionPerfil.Ambitos, semilla.AsignacionPerfil.Ambitos) {
		return plantillasapp.ErrNoDisponible
	}
	// El CAS soloInicial impide que una carrera con revocación o restricción
	// convierta una semilla antigua en autoridad nueva.
	if p.autoridad.publicarInstantanea(ctx, preparada) != nil {
		return plantillasapp.ErrNoDisponible
	}
	return nil
}

func asegurarPerfilPlantillasCatalogoCTSoloInicial(ctx context.Context, fuente vecports.FuenteAutorizacion,
	publicador publicadorInicialPlantillasCT, semilla vecdomain.InstantaneaAutorizacion) error {
	if ctx == nil || ctx.Err() != nil || dependenciaEsNulaContratacionTemporalDesarrollo(fuente) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(publicador) || semilla.Validar() != nil {
		return plantillasapp.ErrNoDisponible
	}
	actual, err := fuente.ObtenerInstantaneaAutorizacion(ctx,
		semilla.AsignacionPerfil.PrincipalID, semilla.AsignacionPerfil.PerfilActivoRef)
	if err == nil {
		// Se conserva la concesión actual aunque esté restringida o revocada.
		// El PDP central decidirá en cada operación con esa versión vigente.
		if actual.Validar() != nil || actual.VersionRol.RolID != semilla.VersionRol.RolID ||
			actual.AsignacionPerfil.PrincipalID != semilla.AsignacionPerfil.PrincipalID ||
			actual.AsignacionPerfil.PerfilActivoRef != semilla.AsignacionPerfil.PerfilActivoRef {
			return plantillasapp.ErrNoDisponible
		}
		return nil
	}
	if !errors.Is(err, vecports.ErrAsignacionPerfilNoEncontrada) {
		return plantillasapp.ErrNoDisponible
	}
	// Solo la ausencia demostrada permite intentar crear. El publicador
	// PostgreSQL vuelve a comparar bajo lock y CAS dentro de la transacción.
	if publicador.PublicarInicial(ctx, semilla) != nil {
		return plantillasapp.ErrNoDisponible
	}
	return nil
}

// nuevasRutasPlantillasCTDesarrollo recibe pools de identidades técnicas
// nominales ya abiertas por la raíz. Nunca elige DSN ni instala SQL.
func nuevasRutasPlantillasCTDesarrollo(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo,
	identidadBase *proveedorSesionConsultaRRHHDesarrollo, fronteras catalogoFronterasComunDesarrollo,
	fuenteAutorizacion, motivosEvaluador *pgxpool.Pool, reloj relojContratacionTemporalDesarrollo,
	materialCatalogo *proveedorMaterialAltaContratacionTemporalDesarrollo,
) ([]vechttp.RutaExacta, error) {
	activo, err := plantillasCatalogoCTDesarrolloSolicitado(cfg)
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	if ctx == nil || ctx.Err() != nil || alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil ||
		soporte == nil || identidadBase == nil || fronteras.identidad == nil ||
		fuenteAutorizacion == nil || motivosEvaluador == nil || materialCatalogo == nil ||
		fuenteAutorizacion == motivosEvaluador || fuenteAutorizacion == alta.postgresql.registroAutorizacion ||
		motivosEvaluador == alta.postgresql.registroAutorizacion {
		return nil, plantillasapp.ErrNoDisponible
	}
	if err := preflightCatalogoPlantillasGobiernoCT(ctx, alta.postgresql.ejecucion); err != nil {
		return nil, err
	}
	catalogo, err := CargarCatalogoPlantillasCT(cfg.Normalize().ReglasEjemplo.CTPlantillasSourcePath)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	if err := comprobarPreimagenCatalogoPlantillasCT(ctx, alta.postgresql.ejecucion, catalogo); err != nil {
		return nil, err
	}
	if soporte.contexto.Resultado.Contexto.PerfilActivoRef == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef ||
		soporte.principalID != alta.soporte.principalID || soporte.certificadoSHA256 != alta.soporte.certificadoSHA256 ||
		!fronteras.contienePerfilesRutasPlantillasCT(soporte.contexto.Resultado.Contexto.PerfilActivoRef) {
		return nil, plantillasapp.ErrNoDisponible
	}
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuenteAutorizacion)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	registro, err := postgresvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	motivo := motivoCatalogoPlantillasCTDesarrollo()
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosEvaluador, motivo.CatalogoID)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	// El catálogo de motivos tiene replay exacto; nunca se expande el rol.
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]vecdomain.ReferenciaEntradaCatalogo{motivo}, desde) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	operacion := referenciaAltaContratacionTemporalDesarrollo("oca_", soporte.principalID+"\x00"+soporte.certificadoSHA256+"\x00registro-contexto-plantillas-ct-v1")
	if publicarResultadoContextoPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, soporte.contexto.Resultado, operacion) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, identidadBase.resolutor, soporte)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	soporte.mu.Lock()
	soporte.contextoEsperadoRegistrado = esperado
	soporte.mu.Unlock()
	semilla, err := instantaneaInicialPlantillasCatalogoCTDesarrollo(
		soporte.contexto.Resultado.Contexto.Principal.ID,
		soporte.contexto.Resultado.Contexto.PerfilActivoRef, reloj.Ahora())
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	publicador := publicadorPostgreSQLInicialPlantillasCT{autoridad: autoridadPostgreSQLDesarrollo{
		pool: alta.postgresql.gobierno, vinculo: soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:ct:plantillas:desarrollo:autorizacion:",
		actoControlRol: "acto:ct:plantillas:desarrollo:control-rol:v1",
		actoAsignacion: "acto:ct:plantillas:desarrollo:asignacion:v1",
		actoSesion:     "acto:ct:plantillas:desarrollo:sesion:v1", soloInicial: true}}
	if asegurarPerfilPlantillasCatalogoCTSoloInicial(ctx, fuente, publicador, semilla) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	identidad, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte,
		identidadBase.registro, identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	soporte.mu.Lock()
	soporte.sesionOperativa = identidad
	soporte.mu.Unlock()
	pdp, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, registro, registro, validador, reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	proveedor, err := nuevoProveedorCatalogoPlantillasCT(soporte, pdp, materialCatalogo, motivo, reloj)
	if err != nil {
		return nil, err
	}
	repo, err := plantillaspg.NuevoRepositorio(alta.postgresql.ejecucion, proveedor, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		return nil, err
	}
	servicio, err := plantillasapp.NuevoServicio(repo, reloj, informejuridico.ValidarCatalogoPlantillasBorrador, &catalogo)
	if err != nil {
		return nil, err
	}
	h, err := plantillashttp.NuevoManejador(proveedor, servicio)
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{{Ruta: plantillashttp.RutaCatalogo, Manejador: h},
		{Ruta: plantillashttp.RutaEntradas, Manejador: h}, {Ruta: plantillashttp.RutaPublicar, Manejador: h}}, nil
}

// CT133 usa un único perfil mTLS para la lectura de detalle y la del catálogo.
// El detalle se resuelve primero y entrega el recibo V3 al proveedor documental.
func nuevasRutasPlantillasDocumentalCTDesarrollo(
	ctx context.Context, cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	soporte *soporteAltaContratacionTemporalDesarrollo, identidadBase *proveedorSesionConsultaRRHHDesarrollo,
	fronteras catalogoFronterasComunDesarrollo, fuentePool, motivosPool *pgxpool.Pool,
	reloj relojContratacionTemporalDesarrollo, consultas dependenciasConsultasRRHHDesarrollo,
	etiquetas informejuridico.EtiquetadorReferencias,
) ([]vechttp.RutaExacta, error) {
	activo, err := plantillasDocumentalCTDesarrolloSolicitado(cfg)
	if err != nil || !activo || ctx == nil || ctx.Err() != nil || alta == nil || alta.soporte == nil ||
		alta.postgresql.gobierno == nil || alta.postgresql.ejecucion == nil || alta.postgresql.registroAutorizacion == nil ||
		alta.postgresql.proveedorMaterialPlantillasDocumental == nil || soporte == nil || identidadBase == nil ||
		fuentePool == nil || motivosPool == nil || fuentePool == motivosPool ||
		dependenciaEsNulaContratacionTemporalDesarrollo(consultas.motivos) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(consultas.sesion) ||
		consultas.materialDetalle == nil || consultas.emisorCuadro == nil || fronteras.identidad == nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	if err := preflightCatalogoPlantillasCT(ctx, alta.postgresql.ejecucion); err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	perfil := soporte.contexto.Resultado.Contexto.PerfilActivoRef
	if perfil == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef ||
		soporte.principalID != alta.soporte.principalID ||
		soporte.certificadoSHA256 != alta.soporte.certificadoSHA256 ||
		!fronteras.contienePerfilRutasPlantillasDocumentalCT(perfil) {
		return nil, plantillasapp.ErrNoDisponible
	}
	catalogo, err := CargarCatalogoPlantillasCT(cfg.Normalize().ReglasEjemplo.CTPlantillasSourcePath)
	if err != nil || comprobarPreimagenCatalogoPlantillasCT(ctx, alta.postgresql.ejecucion, catalogo) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	registro, err := postgresvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	motivoDocumental := motivoDocumentalPlantillasCTDesarrollo()
	validadorDocumental, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, motivoDocumental.CatalogoID)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	motivoDetalle, err := consultas.motivos.ResolverMotivoDetalleRRHH(ctx, reloj.Ahora())
	if err != nil || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivoDetalle) {
		return nil, plantillasapp.ErrNoDisponible
	}
	validadorDetalle, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, motivoDetalle.CatalogoID)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]vecdomain.ReferenciaEntradaCatalogo{motivoDocumental}, desde) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	operacion := referenciaAltaContratacionTemporalDesarrollo("oca_", soporte.principalID+"\x00"+soporte.certificadoSHA256+"\x00registro-contexto-plantillas-documental-ct-v1")
	if publicarResultadoContextoPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, soporte.contexto.Resultado, operacion) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, identidadBase.resolutor, soporte)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	soporte.mu.Lock()
	soporte.contextoEsperadoRegistrado = esperado
	soporte.mu.Unlock()
	semilla, err := instantaneaInicialPlantillasDocumentalCTDesarrollo(
		soporte.contexto.Resultado.Contexto.Principal.ID, perfil, reloj.Ahora())
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	publicador := publicadorPostgreSQLInicialPlantillasCT{autoridad: autoridadPostgreSQLDesarrollo{
		pool: alta.postgresql.gobierno, vinculo: soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:ct:plantillas-documental:desarrollo:autorizacion:",
		actoControlRol: "acto:ct:plantillas-documental:desarrollo:control-rol:v1",
		actoAsignacion: "acto:ct:plantillas-documental:desarrollo:asignacion:v1",
		actoSesion:     "acto:ct:plantillas-documental:desarrollo:sesion:v1", soloInicial: true}}
	if asegurarPerfilPlantillasCatalogoCTSoloInicial(ctx, fuente, publicador, semilla) != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	identidad, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte,
		identidadBase.registro, identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	soporte.mu.Lock()
	soporte.sesionOperativa = identidad
	soporte.mu.Unlock()
	configuracionPDP := aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second}
	pdpDocumental, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, registro, registro, validadorDocumental, reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, configuracionPDP)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	pdpDetalle, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, registro, registro, validadorDetalle, reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, configuracionPDP)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	autoridadDetalle := &autoridadConsultasRRHHDesarrollo{
		soporte: soporte, delegado: pdpDetalle, reloj: reloj, clase: ctports.AmbitoOrganizacionRRHH,
		ambitoRef:       organizacionAltaContratacionTemporalDesarrollo,
		documentalCT133: true, motivoDetalleDocumental: motivoDetalle,
	}
	err = autoridadDetalle.configurarProveedorContextoConsultaRRHHDesarrollo(
		proveedorContextoConsultaRRHHDesarrolloFunc(func(ctx context.Context) (ctports.ContextoAutorizacionAltaV3, error) {
			return soporte.contextoOperativoDesarrollo(ctx)
		}))
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	materialDetalle := *consultas.materialDetalle
	materialDetalle.soporte, materialDetalle.motivo = soporte, motivoDetalle
	emisorDetalle, err := nuevoEmisorMaterialRenovableCTDesarrollo(autoridadDetalle, &materialDetalle)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	emisor, err := ctports.NuevoEmisorMaterialConsultaRRHH(consultas.motivos,
		seguridadvec.GeneradorReferenciasCriptograficas{}, reloj, consultas.emisorCuadro, emisorDetalle)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	consultorDetalle, err := ctapplication.NuevoServicioConsultaDetalleRRHH(autoridadDetalle, emisor, consultas.sesion, reloj)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	materialDocumental := alta.postgresql.proveedorMaterialPlantillasDocumental
	materialDocumental.soporte, materialDocumental.motivo = soporte, motivoDocumental
	proveedorAutorizacion, err := nuevoProveedorDocumentalPlantillasCT(soporte, pdpDocumental, materialDocumental, motivoDocumental, reloj)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	proveedorDocumental, err := plantillaspg.NuevoProveedorDocumental(alta.postgresql.ejecucion,
		proveedorAutorizacion, proveedorAutorizacion, &catalogo, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	manejador, err := plantillashttp.NuevoManejadorBorradores(consultorDetalle, proveedorDocumental,
		plantillashttp.GeneradorBorradoresCatalogo{PDF: pdfvec.Renderizador{}, DOCX: docxvec.Renderizador{}, Etiquetas: etiquetas},
		reloj.Ahora)
	if err != nil {
		return nil, plantillasapp.ErrNoDisponible
	}
	return []vechttp.RutaExacta{{Ruta: plantillashttp.RutaBorradoresDisponibles, Manejador: manejador},
		{Ruta: plantillashttp.RutaBorradores, Manejador: manejador}}, nil
}
