package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Sólo rutas a archivos privados: ni DSN ni claves en variables de entorno,
// JSON de respuesta o errores. Las coordenadas no son concesiones; el PDP,
// consumidores y lectores PostgreSQL revalidan el gobierno efectivo.
type archivoIncorporacionV2 struct {
	Continuidad   *archivoContinuidadNominal           `json:"continuidad_nominal,omitempty"`
	Esquema       string                               `json:"esquema"`
	Referencias   ReferenciasCTIncorporacionDesarrollo `json:"referencias"`
	Planes        string                               `json:"planes_file"`
	TernaPlanes   ct.ReferenciaVersionadaPersonalRPT   `json:"terna_planes"`
	Personal      string                               `json:"personal_file"`
	TernaPersonal fuenteejercicio.TernaEsperada        `json:"terna_personal"`
	MotivoAlta    core.ReferenciaEntradaCatalogo       `json:"motivo_alta"`
	MotivoLectura core.ReferenciaEntradaCatalogo       `json:"motivo_lectura"`
	Pools         map[string]string                    `json:"dsn_files"`
	Material      archivoMaterialIncorporacionV2       `json:"material"`
}

// Nombres cerrados; Personal permite su lector nominal con ejecutor, pero las
// dos operaciones conservan LOGIN/pool distintos. SQL impone las ACL propias.
var rolesPoolsIncorporacionV2 = map[string]string{
	"raices_ct":              "vec_contratacion_temporal_lector_raices_historicas",
	"localizador_ct":         "vec_contratacion_temporal_localizador_incorporacion",
	"localizador_personal":   "vec_personal_localizador_solicitud_alta",
	"lector_personal":        "vec_personal_ejecutor",
	"historia_ct":            "vec_contratacion_temporal_lector_historia_incorporacion",
	"historia_autenticacion": "vec_identidad_sesiones_v1_lector_historico",
	"historia_contexto":      "vec_contexto_actor_v1_lector_historico",
	"historia_evaluacion":    "vec_autorizacion_evaluacion_historica_lector",
	"historia_concesion":     "vec_autorizacion_registro",
	"alta_personal":          "vec_personal_ejecutor",
	"registro_ct":            "vec_contratacion_temporal_ejecutor",
	"fuente_autorizacion":    "vec_autorizacion_fuente",
	"motivos_autorizacion":   "vec_autorizacion_motivos_evaluador",
}

func rolPoolIncorporacionV2(rol string) bool {
	for _, r := range rolesPoolsIncorporacionV2 {
		if r == rol {
			return true
		}
	}
	return false
}

func leerConfiguracionIncorporacionV2(ruta string) (archivoIncorporacionV2, *os.Root, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	var c archivoIncorporacionV2
	if !filepath.IsAbs(ruta) || dentroDeRepositorioGit(ruta) {
		return c, nil, f
	}
	// Misma política privada existente; no se instala ni modifica material.
	if validarArbolMaterialDesarrollo(filepath.Dir(ruta)) != nil {
		return c, nil, f
	}
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return c, nil, f
	}
	ok := false
	defer func() {
		if !ok {
			raiz.Close()
		}
	}()
	b, err := leerArchivoIncorporacionV2(raiz, filepath.Base(ruta), 64<<10)
	if err != nil {
		return c, nil, f
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.Esquema != "vec.contratacion-temporal.incorporacion-servidor.v2" || !c.Referencias.valida() || len(c.Pools) != len(rolesPoolsIncorporacionV2) {
		return c, nil, f
	}
	for k := range rolesPoolsIncorporacionV2 {
		if c.Pools[k] == "" {
			return c, nil, f
		}
	}
	if !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoAlta) || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoLectura) || c.MotivoAlta.CatalogoID != c.MotivoLectura.CatalogoID {
		return c, nil, f
	}
	ok = true
	return c, raiz, nil
}

func leerArchivoIncorporacionV2(raiz *os.Root, nombre string, max int64) ([]byte, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	if raiz == nil || !filepath.IsLocal(nombre) {
		return nil, f
	}
	info, err := raiz.Lstat(nombre)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > max {
		return nil, f
	}
	a, err := raiz.Open(nombre)
	if err != nil {
		return nil, f
	}
	defer a.Close()
	i, err := a.Stat()
	if err != nil || !os.SameFile(info, i) {
		return nil, f
	}
	b, err := io.ReadAll(io.LimitReader(a, max+1))
	if err != nil || int64(len(b)) != i.Size() || int64(len(b)) > max {
		borrarBytes(b)
		return nil, f
	}
	return b, nil
}

func cargarIncorporacionV2Desarrollo(cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo, consultas dependenciasConsultasRRHHDesarrollo, reloj relojContratacionTemporalDesarrollo) (ConfiguracionIncorporacionDesarrollo, func(), error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	vacia := ConfiguracionIncorporacionDesarrollo{}
	if !cfg.DevelopmentEnabledByDoubleKey() || alta == nil || alta.postgresql.registroAutorizacion == nil || consultas.identidad == nil || consultas.sesion == nil || consultas.motivos == nil || consultas.emisorCuadro == nil || consultas.materialDetalle == nil {
		return vacia, nil, f
	}
	c, raiz, err := leerConfiguracionIncorporacionV2(cfg.IncorporacionV2File)
	if err != nil {
		return vacia, nil, f
	}
	defer raiz.Close()
	planes, err := leerArchivoIncorporacionV2(raiz, c.Planes, 256<<10)
	if err != nil {
		return vacia, nil, f
	}
	fuentePlanes, err := inc.NuevaFuentePlanesPreparacionV2(planes, c.TernaPlanes)
	if err != nil {
		return vacia, nil, f
	}
	// La fuente actual no se interpreta aquí: recuperar originales no depende
	// de que siga vigente. Su terna se valida por el consumidor cuando procede.
	personal, err := leerArchivoIncorporacionV2(raiz, c.Personal, 2<<20)
	if err != nil {
		return vacia, nil, f
	}
	material := consultas.materialDetalle
	emisiones, err := cargarMaterialIncorporacionV2(raiz, c.Material, material.raiz, reloj)
	if err != nil {
		return vacia, nil, f
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	pools := map[string]*pgxpool.Pool{}
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			for _, p := range pools {
				p.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	usuarios := map[string]bool{}
	for _, p := range []*pgxpool.Pool{alta.postgresql.ejecucion, alta.postgresql.bolsa, alta.postgresql.gobierno, alta.postgresql.registroAutorizacion, alta.postgresql.confirmador} {
		if p != nil {
			usuarios[p.Config().ConnConfig.User] = true
		}
	}
	for nombre, rol := range rolesPoolsIncorporacionV2 {
		b, e := leerArchivoIncorporacionV2(raiz, c.Pools[nombre], 16<<10)
		if e != nil {
			return vacia, nil, f
		}
		dsn := strings.TrimSpace(string(b))
		borrarBytes(b)
		pc, e := pgxpool.ParseConfig(dsn)
		if e != nil || pc.ConnConfig.User == "" || usuarios[pc.ConnConfig.User] {
			return vacia, nil, f
		}
		usuarios[pc.ConnConfig.User] = true
		p, _, e := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-incorporacion-v2-"+nombre, rol)
		if e != nil {
			return vacia, nil, f
		}
		pools[nombre] = p
	}
	fuente, err := pgvec.NuevoAlmacenAutorizacion(pools["fuente_autorizacion"])
	if err != nil {
		return vacia, nil, f
	}
	registro, err := pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return vacia, nil, f
	}
	motivoDetalle, err := consultas.motivos.ResolverMotivoDetalleRRHH(ctx, reloj.Ahora())
	if err != nil {
		return vacia, nil, f
	}
	motivos, motivosDetalle, err := validadoresMotivosIncorporacionV2(pools["motivos_autorizacion"], c.MotivoAlta, motivoDetalle)
	if err != nil {
		return vacia, nil, f
	}
	pdp, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, motivos, reloj, seg.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return vacia, nil, f
	}
	autoridadOperacion, err := nuevaAutoridadOperacionesIncorporacionV2(alta.soporte, consultas.autoridad, pdp, c, fuentePlanes, personal, motivoDetalle, reloj)
	if err != nil {
		return vacia, nil, f
	}
	cadena, err := inc.NuevaCadenaAutorizacionAplicacion(autoridadOperacion, material.atestador, material.confianza, emisiones)
	if err != nil {
		return vacia, nil, f
	}
	pdpDetalle, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, motivosDetalle, reloj, seg.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return vacia, nil, f
	}
	autoridadDetalle, err := nuevaAutoridadOperacionesIncorporacionV2(alta.soporte, consultas.autoridad, pdpDetalle, c, fuentePlanes, personal, motivoDetalle, reloj)
	if err != nil {
		return vacia, nil, f
	}
	emisorDetalle, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(autoridadDetalle, material.atestador, material.confianza, material.emisor)
	if err != nil {
		return vacia, nil, f
	}
	emisor, err := ct.NuevoEmisorMaterialConsultaRRHH(consultas.motivos, seg.GeneradorReferenciasCriptograficas{}, reloj, consultas.emisorCuadro, emisorDetalle)
	if err != nil {
		return vacia, nil, f
	}
	// El servicio y sesión existentes consumen autorización vigente del PDP;
	// el ámbito de este plan no altera las otras consultas del servidor.
	autoridad := &contextoDetalleNominalIncorporacionV2{consultas.autoridad, c.Referencias, reloj}
	detalle, err := appct.NuevoServicioConsultaDetalleRRHH(autoridad, emisor, consultas.sesion, reloj)
	if err != nil {
		return vacia, nil, f
	}
	continuidad, err := cargarContinuidadNominal(raiz, c.Continuidad, c.Referencias, pools, alta, consultas, fuentePlanes, detalle, reloj)
	if err != nil {
		return vacia, nil, f
	}
	completa = true
	return ConfiguracionIncorporacionDesarrollo{continuidad: continuidad, detalleNominal: detalle, Referencias: c.Referencias, Cadena: cadena,
		MotivoAlta: c.MotivoAlta, MotivoLectura: c.MotivoLectura, AltaPersonal: pools["alta_personal"], RegistroCT: pools["registro_ct"],
		Preparacion: inc.ConfiguracionPreparacionDurableV2PostgreSQL{Planes: planes, TernaPlanes: c.TernaPlanes, FuentePersonal: personal, TernaPersonal: c.TernaPersonal,
			Pools: inc.PoolsPreparacionDurableV2{InicialCT: pools["raices_ct"], LocalizadorCT: pools["localizador_ct"], LocalizadorPersonal: pools["localizador_personal"], LecturaPersonal: pools["lector_personal"], Historia: pgct.PoolsHistoriaIncorporacionV2{RegistroCT: pools["historia_ct"], Autenticacion: pools["historia_autenticacion"], Contexto: pools["historia_contexto"], Evaluacion: pools["historia_evaluacion"], Concesion: pools["historia_concesion"]}}}}, cerrar, nil
}

// Cada PDP conserva su catálogo cerrado y comprueba la terna publicada en el
// mismo lector PostgreSQL. El motivo de detalle procede del resolutor RRHH,
// no de la configuración de alta ni de una lista de catálogos intercambiables.
func validadoresMotivosIncorporacionV2(pool *pgxpool.Pool, alta, detalle core.ReferenciaEntradaCatalogo) (*pgvec.ValidadorReferenciaMotivoPostgreSQLV2, *pgvec.ValidadorReferenciaMotivoPostgreSQLV2, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	if !core.ReferenciaMotivoAutorizacionV2Valida(alta) || !core.ReferenciaMotivoAutorizacionV2Valida(detalle) {
		return nil, nil, f
	}
	a, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pool, alta.CatalogoID)
	if err != nil {
		return nil, nil, f
	}
	d, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pool, detalle.CatalogoID)
	if err != nil {
		return nil, nil, f
	}
	return a, d, nil
}

type contextoDetalleNominalIncorporacionV2 struct {
	consulta    *autoridadConsultasRRHHDesarrollo
	referencias ReferenciasCTIncorporacionDesarrollo
	reloj       ct.Reloj
}

func (a *contextoDetalleNominalIncorporacionV2) ResolverContextoConsultaRRHH(ctx context.Context) (ct.ContextoConsultaRRHH, error) {
	if a == nil || a.consulta == nil || a.consulta.soporte == nil || a.reloj == nil || !a.referencias.valida() || ctx == nil || ctx.Value(claveIncorporacionV2Desarrollo{}) != a.consulta.soporte.sello {
		return ct.ContextoConsultaRRHH{}, ct.ErrAutorizacionDenegada
	}
	c, err := a.consulta.contextoConsultaRRHHDesarrollo(ctx)
	if err != nil {
		return ct.ContextoConsultaRRHH{}, err
	}
	v, err := c.Vinculo.Datos()
	if err != nil || v.PrincipalID != a.referencias.PrincipalV3Ref || v.PerfilActivoRef != a.referencias.PerfilV3Ref {
		return ct.ContextoConsultaRRHH{}, ct.ErrAutorizacionDenegada
	}
	return ct.NuevoContextoConsultaRRHH(c, a.referencias.OrganizacionRef, a.reloj.Ahora())
}
