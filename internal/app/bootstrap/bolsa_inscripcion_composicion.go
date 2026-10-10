package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Composición de la inscripción en Bolsa en cada raíz. Con
// VEC_BOLSA_INSCRIPCIONES_ENABLED apagado ninguna de las dos funciones abre
// conexiones ni devuelve rutas. Encendido, cualquier dependencia ausente
// (SQL, material V3, permisos o cuentas) impide arrancar: no hay modo parcial.

// motivoPresentarInscripcionBolsaDesarrollo es el motivo V3 con el que la
// persona aspirante presenta su solicitud. El validador central lo coteja en
// cada acto contra el catálogo publicado; sin ese catálogo se deniega.
func motivoPresentarInscripcionBolsaDesarrollo() core.ReferenciaEntradaCatalogo {
	return core.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_inscripcion_bolsa_desarrollo", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-inscripcion-bolsa-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-inscripcion-presentar"),
	}
}

// Las audiencias V3 de RRHH. Su material se publica con el gobierno único de
// CT sólo cuando el selector está encendido en vec-server.
func descriptoresMaterialInscripcionRRHHDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{{
		Audiencia:        "vec_bolsa_llamamientos.inscripcion.revisar.v1",
		Dominio:          "vec.bolsa.inscripcion.revisar.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-inscripcion-revisar:",
		ProveedorNominal: "proveedor-material-bolsa-inscripcion-revisar",
	}, {
		Audiencia:        "vec_bolsa_llamamientos.inscripcion.incorporar.v1",
		Dominio:          "vec.bolsa.inscripcion.incorporar.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-inscripcion-incorporar:",
		ProveedorNominal: "proveedor-material-bolsa-inscripcion-incorporar",
	}}
}

// materialInscripcionRRHHDesarrollo lleva los dos proveedores V3 de RRHH:
// decidir (audiencia revisar) e incorporar.
type materialInscripcionRRHHDesarrollo struct {
	revisar, incorporar *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func descriptoresLecturaInscripcionBolsa(superficie string) map[ClaveOperacionInscripcionBolsa]DescriptorLecturaInscripcionBolsa {
	resultado := map[ClaveOperacionInscripcionBolsa]DescriptorLecturaInscripcionBolsa{}
	for clave, d := range descriptoresLecturaActualInscripcionBolsa(superficie) {
		resultado[clave] = DescriptorLecturaInscripcionBolsa{Accion: d.Accion, Finalidad: d.Finalidad, Campos: d.Campos}
	}
	return resultado
}

func descriptoresEscrituraInscripcionBolsa(superficie string, motivos map[string]core.ReferenciaEntradaCatalogo) map[ClaveOperacionInscripcionBolsa]DescriptorInscripcionBolsa {
	resultado := map[ClaveOperacionInscripcionBolsa]DescriptorInscripcionBolsa{}
	for _, clave := range clavesEscrituraInscripcionBolsa(superficie) {
		tipo, finalidad, ok := tipoFinalidadInscripcionBolsa(clave)
		if !ok {
			return nil
		}
		resultado[clave] = DescriptorInscripcionBolsa{Accion: clave.Accion, ModuloID: "bolsa", TipoRecurso: tipo,
			Finalidad: finalidad, Motivo: motivos[clave.Accion]}
	}
	return resultado
}

// cierrePoolsInscripcion cierra una sola vez los pools abiertos al componer.
type cierrePoolsInscripcion struct {
	pools []*pgxpool.Pool
	extra []func()
	una   sync.Once
}

func (c *cierrePoolsInscripcion) cerrar() {
	c.una.Do(func() {
		for i := len(c.extra) - 1; i >= 0; i-- {
			c.extra[i]()
		}
		for _, p := range c.pools {
			p.Close()
		}
	})
}

// nuevaInscripcionPortalExterno compone las rutas de la persona aspirante en
// el proceso del portal externo. Reutiliza la sesión del Área personal y las
// fachadas externas de autorización; nunca recibe credenciales ni rutas de RRHH.
func nuevaInscripcionPortalExterno(ctx context.Context, cfg config.Config,
	preferencias *autoridadPreferenciasUsuariosDesarrollo, preflight *pgxpool.Pool,
) (http.Handler, func(), error) {
	nada := func() {}
	activo, err := cfg.BolsaInscripcionesExternoActivo()
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	if !activo {
		return nil, nada, nil
	}
	if ctx == nil || preferencias == nil || preflight == nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, preflight)
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	cierre := &cierrePoolsInscripcion{}
	completa := false
	defer func() {
		if !completa {
			cierre.cerrar()
		}
	}()
	for _, entrada := range []struct {
		configuracion config.ConfiguracionPostgreSQLExterna
		rol           string
	}{
		{cfg.ExternoAutorizacionFuentePostgreSQL, "vec_autorizacion_fuente_externa"},
		{cfg.ExternoAutorizacionRegistroPostgreSQL, "vec_autorizacion_registro_externo"},
		{cfg.ExternoAutorizacionMotivosPostgreSQL, "vec_autorizacion_motivos_externos"},
	} {
		dsn, err := entrada.configuracion.DSN()
		if err != nil {
			return nil, nada, errMontajeInscripcionBolsa
		}
		pool, _, err := abrirPoolMiBolsaPortalExterno(ctx, dsn, entrada.rol)
		if err != nil {
			return nil, nada, errMontajeInscripcionBolsa
		}
		cierre.pools = append(cierre.pools, pool)
		if cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
			return nil, nada, errMontajeInscripcionBolsa
		}
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacionExterna(cierre.pools[0])
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	registro, err := vecpg.NuevoAlmacenAutorizacionExterna(cierre.pools[1])
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	motivo := motivoPresentarInscripcionBolsaDesarrollo()
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2Externo(cierre.pools[2], motivo.CatalogoID)
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	pdp, err := nuevaAutorizacionMiBolsaPortalExterno(fuente, registro, registro, motivos)
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	reloj := relojContratacionTemporalDesarrollo{}
	proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight, consumidorInscripcionPortalExternoV3, reloj)
	if err != nil || proveedores[audienciaPresentarInscripcionExternaV3] == nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	superficie := superficieExternaInscripcionBolsa
	decisor, err := NuevoDecisorLecturaActualInscripcionPostgreSQL(cierre.pools[0], ConfiguracionDecisorLecturaActualInscripcion{
		Superficie: superficie, Reloj: reloj, Descriptores: descriptoresLecturaActualInscripcionBolsa(superficie)})
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	autoridad, err := NuevaAutoridadInscripcionBolsa(ConfiguracionAutoridadInscripcionBolsa{
		Superficie: superficie, Lectura: decisor, PDP: pdp, Motivos: motivos, Reloj: reloj, FuenteActual: decisor,
		Material: map[ClaveOperacionInscripcionBolsa]*proveedorMaterialAltaContratacionTemporalDesarrollo{
			{inscripcion.AccionPresentar, superficie}: proveedores[audienciaPresentarInscripcionExternaV3]},
		Descriptores: descriptoresEscrituraInscripcionBolsa(superficie, map[string]core.ReferenciaEntradaCatalogo{inscripcion.AccionPresentar: motivo}),
		Lecturas:     descriptoresLecturaInscripcionBolsa(superficie),
	})
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	sesion, err := NuevaSesionExternaInscripcion(preferencias)
	if err != nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	manejador, cerrarMontaje, err := nuevoMontajeInscripcionBolsaExterno(ctx, cfg, sesion, autoridad)
	if err != nil || manejador == nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	cierre.extra = append(cierre.extra, cerrarMontaje)
	completa = true
	return manejador, cierre.cerrar, nil
}

// configuracionInscripcionRRHHDesarrollo es el archivo privado
// identidad/inscripcion-bolsa-rrhh.json: cuentas RRHH nominales, conexiones de
// identidad, contexto y autorización, y los motivos de decidir e incorporar.
// No concede permisos: la concesión sigue en el gobierno nominal.
type configuracionInscripcionRRHHDesarrollo struct {
	redaccionMaterialRutasDietas
	Version                  int                           `json:"version"`
	Autoridad                string                        `json:"autoridad"`
	Cuentas                  []cuentaRutasDietasDesarrollo `json:"cuentas"`
	DSNRegistroIdentidad     string                        `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string                        `json:"dsn_revalidacion_identidad"`
	DSNContexto              string                        `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string                        `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string                        `json:"dsn_registro_autorizacion"`
	DSNMotivos               string                        `json:"dsn_motivos"`
	Motivos                  struct {
		Decidir    core.ReferenciaEntradaCatalogo `json:"decidir"`
		Incorporar core.ReferenciaEntradaCatalogo `json:"incorporar"`
	} `json:"motivos"`
}

func cargarConfiguracionInscripcionRRHHDesarrollo(cfg config.Config) (configuracionInscripcionRRHHDesarrollo, error) {
	var c configuracionInscripcionRRHHDesarrollo
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "inscripcion-bolsa-rrhh.json"), 64<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return c, errMontajeInscripcionBolsa
	}
	defer borrarBytes(contenido)
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || !core.ReferenciaMotivoAutorizacionV2Valida(c.Motivos.Decidir) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.Motivos.Incorporar) || c.Motivos.Decidir.CatalogoID != c.Motivos.Incorporar.CatalogoID ||
		c.Motivos.Decidir.Referencia() == c.Motivos.Incorporar.Referencia() {
		return configuracionInscripcionRRHHDesarrollo{}, errMontajeInscripcionBolsa
	}
	return c, nil
}

// cuentasRRHHInscripcionDesarrollo cruza cada cuenta del archivo con una
// identidad RRHH ya registrada en el resolvedor (mismo certificado y perfil).
func cuentasRRHHInscripcionDesarrollo(identidad *resolvedorIdentidadDesarrollo, cuentas []cuentaRutasDietasDesarrollo) ([]identidadConsultaRRHHDesarrollo, error) {
	lectores := map[string]identidadConsultaRRHHDesarrollo{}
	for _, lector := range identidad.lectoresConsultaRRHH() {
		lectores[lector.identidad.principal.Attributes["certificate_sha256"]] = lector
	}
	vistas := map[string]bool{}
	rrhh := make([]identidadConsultaRRHHDesarrollo, 0, len(cuentas))
	for _, cuenta := range cuentas {
		lector, existe := lectores[cuenta.CertificadoSHA256]
		if !existe || vistas[cuenta.CertificadoSHA256] || !huellaCertificadoInscripcionValida(cuenta.CertificadoSHA256) ||
			lector.identidad.principal.ID != cuenta.Sujeto || lector.perfilRef != cuenta.PerfilRef || cuenta.CuentaRef == "" {
			return nil, errMontajeInscripcionBolsa
		}
		vistas[cuenta.CertificadoSHA256] = true
		rrhh = append(rrhh, lector)
	}
	return rrhh, nil
}

// inscripcionRRHHDesarrollo es la bandeja de RRHH compuesta en vec-server.
type inscripcionRRHHDesarrollo struct {
	manejador http.Handler
	cerrar    func()
}

// nuevaInscripcionRRHHDesarrollo compone /api/vec/bolsa/rrhh/inscripciones en
// vec-server con la sesión interna nominal, el PDP interno, el ámbito de
// gestión de RRHH y el material V3 de decidir e incorporar.
func nuevaInscripcionRRHHDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo,
	material materialInscripcionRRHHDesarrollo,
) (*inscripcionRRHHDesarrollo, error) {
	activo, err := cfg.BolsaInscripcionesInternoActivo()
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	if !activo {
		return nil, nil
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || material.revisar == nil || material.incorporar == nil {
		return nil, errMontajeInscripcionBolsa
	}
	c, err := cargarConfiguracionInscripcionRRHHDesarrollo(cfg)
	if err != nil {
		return nil, err
	}
	cuentas := c.Cuentas
	rrhh, err := cuentasRRHHInscripcionDesarrollo(identidad, cuentas)
	if err != nil {
		return nil, err
	}
	dsnLector, err := cfg.DSNBolsaInscripcionesLectorRRHHSeparado()
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancelar()
	cierre := &cierrePoolsInscripcion{}
	completa := false
	defer func() {
		if !completa {
			cierre.cerrar()
		}
	}()
	usuarios := map[string]bool{}
	for _, entrada := range []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
	} {
		pool, usuario, err := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if err != nil {
			return nil, errMontajeInscripcionBolsa
		}
		cierre.pools = append(cierre.pools, pool)
		if usuarios[usuario] {
			return nil, errMontajeInscripcionBolsa
		}
		usuarios[usuario] = true
	}
	ejecutor, err := abrirEjecutorInternoInscripcionBolsa(ctx, cfg.ContratacionTemporalPostgreSQL)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	cierre.pools = append(cierre.pools, ejecutor)
	lector, err := abrirLectorInscripcionBolsa(ctx, dsnLector, lectorInscripcionRRHH)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	cierre.pools = append(cierre.pools, lector)
	p := cierre.pools
	registroSesiones, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, p[0], p[1], &seudonimizadorSesionDesarrollo{derivador: derivador},
		espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, p[1])
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, p[2])
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	relojBase := relojRutasDietas{}
	servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojBase)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(p[3])
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(p[4])
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(p[5], c.Motivos.Decidir.CatalogoID)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	pdp, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, relojBase,
		seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	porHuella := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range cuentas {
		porHuella[cuenta.CertificadoSHA256] = cuenta
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: porHuella, registro: registroSesiones,
		revalidador: revalidador, contextos: contextos, reloj: relojBase, instancia: nonce}
	sesion, err := nuevaSesionInternaInscripcionBolsa(base, cuentas)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	reloj := relojContratacionTemporalDesarrollo{}
	ambito, err := NuevaFuenteAmbitoRRHHInscripcionPostgreSQL(lector, reloj, rrhh)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	superficie := superficieInternaInscripcionBolsa
	decisor, err := NuevoDecisorLecturaActualInscripcionPostgreSQL(p[3], ConfiguracionDecisorLecturaActualInscripcion{
		Superficie: superficie, Reloj: reloj, Descriptores: descriptoresLecturaActualInscripcionBolsa(superficie),
		AmbitoRRHH: ambito, RRHHNominal: rrhh})
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	autoridad, err := NuevaAutoridadInscripcionBolsa(ConfiguracionAutoridadInscripcionBolsa{
		Superficie: superficie, Lectura: decisor, PDP: pdp, Motivos: motivos, Reloj: reloj, FuenteActual: decisor,
		AmbitoRRHH: ambito, RRHHNominal: rrhh,
		Material: map[ClaveOperacionInscripcionBolsa]*proveedorMaterialAltaContratacionTemporalDesarrollo{
			{inscripcion.AccionDecidir, superficie}: material.revisar, {inscripcion.AccionIncorporar, superficie}: material.incorporar},
		Descriptores: descriptoresEscrituraInscripcionBolsa(superficie, map[string]core.ReferenciaEntradaCatalogo{
			inscripcion.AccionDecidir: c.Motivos.Decidir, inscripcion.AccionIncorporar: c.Motivos.Incorporar}),
		Lecturas: descriptoresLecturaInscripcionBolsa(superficie),
	})
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	repositorio, err := postgresbolsa.NuevoRepositorioInscripcionesInternoPostgreSQL(ejecutor, lector)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	servicio, err := inscripcion.NuevoServicio(repositorio)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	preparador, err := NuevoPreparadorInscripcionBolsaInterno(ConfiguracionPreparadorInscripcionBolsa{
		RRHH: rrhh, SesionRRHH: sesion, Autoridad: autoridad, Reloj: reloj})
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	manejador, err := httpinscripcion.NuevoInterno(preparador, servicio)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	completa = true
	return &inscripcionRRHHDesarrollo{manejador: manejador, cerrar: cierre.cerrar}, nil
}

// componerRaizConInscripcionRRHH sólo añade el prefijo de RRHH cuando la
// bandeja está compuesta; apagada, la raíz queda exactamente igual.
func componerRaizConInscripcionRRHH(raiz http.Handler, i *inscripcionRRHHDesarrollo) http.Handler {
	if i == nil || i.manejador == nil {
		return raiz
	}
	mux := http.NewServeMux()
	mux.Handle(httpinscripcion.RutaRRHH, i.manejador)
	mux.Handle(httpinscripcion.RutaRRHH+"/", i.manejador)
	mux.Handle("/", raiz)
	return mux
}
