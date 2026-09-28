package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

var errAutoridadesAuditoriaConsultaDesarrollo = errors.New("bootstrap: autoridades de consulta de auditoria no disponibles")

const catalogoMotivosAuditoriaConsultaDesarrollo = "motivos_autorizacion_auditoria"

// AD3-91 define una sola audiencia de consumo para ambas fuentes. La clave
// material se publica una vez en el catálogo común; el perfil V3 y el
// consumidor SQL nominal separan CT de Bolsa.
func descriptorMaterialAuditoriaConsultaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        auditoria.AudienciaConsumo,
		Dominio:          "vec.auditoria.consulta-rrhh.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:auditoria-consulta:",
		ProveedorNominal: "proveedor-material-auditoria-consulta",
	}
}

// Ambos discriminadores comparten la cuenta y persona ya acreditadas en la
// raíz CT, pero crean perfiles, vínculos y sesiones distintos entre sí y de
// CT/B-BACK. Ningún identificador procede de parámetros HTTP.
func discriminadorContextoAuditoriaConsultaDesarrollo(fuente string) discriminadorContextoSinteticoDesarrollo {
	etiqueta := "auditoria-consulta-" + fuente + "-v1"
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-" + etiqueta, vinculo: "vinculo-" + etiqueta,
		procedencia: "procedencia", registro: "registro-contexto-" + etiqueta,
		autenticacion: "autenticacion-" + etiqueta, asercion: "asercion-" + etiqueta,
		sesion: "sesion-" + etiqueta, controlSesion: "control-sesion-" + etiqueta,
		politicaGarantia: "politica-garantia-" + etiqueta,
	}
}

type soportesAuditoriaConsultaDesarrollo struct {
	CT, Bolsa              *soporteAltaContratacionTemporalDesarrollo
	PerfilCT, PerfilBolsa  string
	OperacionContextoCT    string
	OperacionContextoBolsa string
}

// Deriva dos soportes para el proveedor de sesión mTLS existente. La raíz
// publica cada Resultado con su OperacionContexto nominal y crea las sesiones
// con nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo; nunca sustituye
// la asignación del perfil CT o del perfil B-BACK.
func nuevosSoportesAuditoriaConsultaDesarrollo(
	cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	bolsa *soporteSesionBorradorBolsaDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (soportesAuditoriaConsultaDesarrollo, error) {
	vacio := soportesAuditoriaConsultaDesarrollo{}
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || cfg.DevelopmentMaterialDir == "" ||
		alta == nil || alta.soporte == nil || bolsa == nil || bolsa.soporteCanal == nil {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	soportes, err := nuevosSoportesAuditoriaConsultaDesdeBaseDesarrollo(alta.soporte, reloj.Ahora())
	if err != nil || !contextoSinteticoBolsaSeparadoDeCT(alta.soporte.contexto, bolsa.soporteCanal.contexto) ||
		!contextoSinteticoBolsaSeparadoDeCT(bolsa.soporteCanal.contexto, soportes.CT.contexto) ||
		!contextoSinteticoBolsaSeparadoDeCT(bolsa.soporteCanal.contexto, soportes.Bolsa.contexto) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return soportes, nil
}

func nuevosSoportesAuditoriaConsultaDesdeBaseDesarrollo(
	base *soporteAltaContratacionTemporalDesarrollo, ahora time.Time,
) (soportesAuditoriaConsultaDesarrollo, error) {
	vacio := soportesAuditoriaConsultaDesarrollo{}
	if base == nil || !ctdomain.InstanteUTCCanonico(ahora) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	base.mu.Lock()
	principalID, certificado := base.principalID, base.certificadoSHA256
	contextoBase, sello, reloj := base.contexto, base.sello, base.reloj
	base.mu.Unlock()
	if sello == nil || !identificadorSesionDesarrolloValido(principalID) ||
		!contextoSinteticoCTConsistenteParaBorradorBolsa(principalID, certificado, contextoBase) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	principal := vecdomain.Principal{
		ID: principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": certificado,
		},
	}
	contextoCT, errCT := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoAuditoriaConsultaDesarrollo("ct"))
	contextoBolsa, errBolsa := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoAuditoriaConsultaDesarrollo("bolsa"))
	if errCT != nil || errBolsa != nil ||
		!contextoSinteticoBolsaSeparadoDeCT(contextoBase, contextoCT) ||
		!contextoSinteticoBolsaSeparadoDeCT(contextoBase, contextoBolsa) ||
		!contextoSinteticoBolsaSeparadoDeCT(contextoCT, contextoBolsa) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	perfilCT := contextoCT.Resultado.Contexto.PerfilActivoRef
	perfilBolsa := contextoBolsa.Resultado.Contexto.PerfilActivoRef
	if !perfilActivoSeguridadComunValido(perfilCT) || !perfilActivoSeguridadComunValido(perfilBolsa) || perfilCT == perfilBolsa {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	nuevoCanal := func(contexto ctports.ContextoAutorizacionAltaV3) *soporteAltaContratacionTemporalDesarrollo {
		return &soporteAltaContratacionTemporalDesarrollo{
			sello: sello, principalID: principalID, certificadoSHA256: certificado,
			contexto: contexto, reloj: reloj,
		}
	}
	baseOperacion := principalID + "\x00" + certificado + "\x00registro-contexto-auditoria-consulta-"
	return soportesAuditoriaConsultaDesarrollo{
		CT: nuevoCanal(contextoCT), Bolsa: nuevoCanal(contextoBolsa),
		PerfilCT: perfilCT, PerfilBolsa: perfilBolsa,
		OperacionContextoCT:    referenciaAltaContratacionTemporalDesarrollo("oca_", baseOperacion+"ct-v1"),
		OperacionContextoBolsa: referenciaAltaContratacionTemporalDesarrollo("oca_", baseOperacion+"bolsa-v1"),
	}, nil
}

// La raíz llama a esta función solo tras el selector explícito y la doble
// llave. El paquete sigue siendo DEMO; la referencia se valida además contra
// la publicación vigente de la autoridad PostgreSQL, no contra el fichero.
func nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(
	ctx context.Context, cfg config.Config, reloj relojContratacionTemporalDesarrollo,
	validador vecports.ValidadorReferenciaMotivoAutorizacionV2,
) (*auditoria.OpcionesCatalogo, auditoria.Opciones, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(validador) {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	ruta, err := cfg.RutaCatalogoAuditoriaConsultaDesarrollo()
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	lector, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: catalogoMotivosAuditoriaConsultaDesarrollo,
		ModuloID: auditoria.ModuloAutorizacion, Reloj: reloj,
	})
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	proveedor, err := auditoria.NuevoProveedorOpcionesCatalogo(lector, validador)
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	opciones, err := proveedor.Actuales(ctx)
	if err != nil || !opciones.EsEjemplo || opciones.PermisoRequerido != auditoria.AccionConsultar {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return proveedor, opciones, nil
}

// Un perfil V3 tiene una sola fuente y una lista explícita de expedientes.
// El filtro de fechas, actor y página queda ligado adicionalmente por la
// huella de RecursoFiltro; la decisión solo puede exponer CamposPermitidos.
func instantaneaAuditoriaConsultaNominalDesarrollo(
	principalID, perfilRef, fuente, expedienteRef, finalidad string, ahora time.Time,
) (vecdomain.InstantaneaAutorizacion, error) {
	if (fuente != "ct" && fuente != "bolsa") || expedienteRef == "" || finalidad == "" ||
		!perfilActivoSeguridadComunValido(perfilRef) {
		return vecdomain.InstantaneaAutorizacion{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID, perfilRef, ahora, "consulta_auditoria_"+fuente+"_desarrollo",
		"Consulta de auditoria "+fuente+" en desarrollo", "consulta-auditoria-"+fuente,
		[]vecdomain.ConcesionRol{{
			Accion: auditoria.AccionConsultar, ModuloID: auditoria.ModuloAutorizacion,
			TipoRecurso: auditoria.TipoRecurso, Finalidades: []string{finalidad},
			GarantiaMinima: vecdomain.AuthAssuranceHigh, CamposPermitidos: auditoria.CamposPermitidos(),
		}},
		[]vecdomain.AmbitoPerfil{
			{Clave: "expediente_ref", Valores: []string{expedienteRef}},
			{Clave: "fuente", Valores: []string{fuente}},
		},
	)
	if err != nil || instantanea.Validar() != nil {
		return vecdomain.InstantaneaAutorizacion{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return instantanea, nil
}

// El publicador de la raíz debe instalar ambas instantáneas en la autoridad
// VEC antes de exponer rutas. Esta fábrica usa exclusivamente los lectores y
// registros PostgreSQL centrales y dos proveedores materiales nominales.
type dependenciasAutoridadesAuditoriaConsultaDesarrollo struct {
	FuenteCT, FuenteBolsa         *pgxpool.Pool
	RegistroCT, RegistroBolsa     *pgxpool.Pool
	MotivosCT, MotivosBolsa       *pgxpool.Pool
	MaterialCT, MaterialBolsa     *proveedorMaterialAltaContratacionTemporalDesarrollo
	Reloj                         relojContratacionTemporalDesarrollo
	Opciones                      auditoria.Opciones
	PerfilCT, PerfilBolsa         string
	ExpedienteCT, ExpedienteBolsa string
}

func nuevosEmisoresAuditoriaConsultaDesarrollo(d dependenciasAutoridadesAuditoriaConsultaDesarrollo) (auditoria.EmisorMaterialV3, auditoria.EmisorMaterialV3, error) {
	if d.FuenteCT == nil || d.FuenteBolsa == nil || d.RegistroCT == nil || d.RegistroBolsa == nil ||
		d.MotivosCT == nil || d.MotivosBolsa == nil || d.MaterialCT == nil || d.MaterialBolsa == nil ||
		d.FuenteCT == d.RegistroCT || d.FuenteBolsa == d.RegistroBolsa ||
		!perfilActivoSeguridadComunValido(d.PerfilCT) || !perfilActivoSeguridadComunValido(d.PerfilBolsa) || d.PerfilCT == d.PerfilBolsa ||
		d.Opciones.FinalidadRef == "" || d.Opciones.Motivo.Validar() != nil ||
		d.Opciones.MotivoRef != d.Opciones.Motivo.Referencia() ||
		!d.Opciones.EsEjemplo || d.Opciones.PermisoRequerido != auditoria.AccionConsultar {
		return nil, nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	ct, err := nuevoEmisorAuditoriaConsultaNominalDesarrollo("ct", d.PerfilCT, d.ExpedienteCT,
		d.FuenteCT, d.RegistroCT, d.MotivosCT, d.MaterialCT, d.Opciones, d.Reloj)
	if err != nil {
		return nil, nil, err
	}
	bolsa, err := nuevoEmisorAuditoriaConsultaNominalDesarrollo("bolsa", d.PerfilBolsa, d.ExpedienteBolsa,
		d.FuenteBolsa, d.RegistroBolsa, d.MotivosBolsa, d.MaterialBolsa, d.Opciones, d.Reloj)
	if err != nil {
		return nil, nil, err
	}
	return ct, bolsa, nil
}

func nuevoEmisorAuditoriaConsultaNominalDesarrollo(
	fuente, perfil, expediente string, poolFuente, poolRegistro, poolMotivos *pgxpool.Pool,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, opciones auditoria.Opciones,
	reloj relojContratacionTemporalDesarrollo,
) (auditoria.EmisorMaterialV3, error) {
	if expediente == "" || material == nil || poolFuente == nil || poolRegistro == nil || poolMotivos == nil ||
		material.emisor == nil || material.atestador == nil || material.confianza == nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	fuenteCentral, err := postgresvec.NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	registroCentral, err := postgresvec.NuevoAlmacenAutorizacion(poolRegistro)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, catalogoMotivosAuditoriaConsultaDesarrollo)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuenteCentral, registroCentral, registroCentral, validador, reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	guardia := autorizadorAuditoriaConsultaNominalDesarrollo{
		delegado: servicio, fuente: fuente, perfil: perfil, expediente: expediente,
		finalidad: opciones.FinalidadRef, motivo: opciones.Motivo,
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(guardia, material)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return emisor, nil
}

// La comprobación nominal precede al PDP; la instantánea publicada y el CAS
// centrales vuelven a restringir exactamente esas mismas dimensiones.
type autorizadorAuditoriaConsultaNominalDesarrollo struct {
	delegado                              vecports.AutorizadorSolicitudLigadaV3
	fuente, perfil, expediente, finalidad string
	motivo                                vecdomain.ReferenciaEntradaCatalogo
}

func (a autorizadorAuditoriaConsultaNominalDesarrollo) ExigirSolicitudLigadaV3(
	ctx context.Context, solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	vacia := vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	datos, err := solicitud.Datos()
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(a.delegado) || err != nil ||
		resultado.Validar() != nil || resultado.Contexto.PerfilActivoRef != a.perfil ||
		datos.Accion != auditoria.AccionConsultar || datos.Recurso.ModuloID != auditoria.ModuloAutorizacion ||
		datos.Recurso.Tipo != auditoria.TipoRecurso || datos.Recurso.Referencia != a.expediente ||
		len(datos.Recurso.Ambitos) != 2 || datos.Recurso.Ambitos["fuente"] != a.fuente ||
		datos.Recurso.Ambitos["expediente_ref"] != a.expediente ||
		len(datos.Recurso.Atributos) != 1 || !huellaAuditoriaConsultaValida(datos.Recurso.Atributos["filtro_sha256"]) ||
		datos.Finalidad != a.finalidad || datos.ReferenciaMotivo != a.motivo {
		return vecdomain.DecisionAutorizacionLigadaV3{}, vacia, auditoria.ErrDenegada
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}
