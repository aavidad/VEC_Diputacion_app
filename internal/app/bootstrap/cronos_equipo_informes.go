package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionCronosEquipoInformesNoDisponible = errors.New("bootstrap: equipo e informes de Cronos no disponibles")

var (
	referenciaEquipoRutaC10 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_-]{0,127}$`)
	referenciaFotoC10       = regexp.MustCompile(`^[-A-Za-z0-9_.:/#]{1,128}$`)
	referenciaRolC10        = regexp.MustCompile(`^rol:[-A-Za-z0-9_.:/#]{1,128}:v[1-9][0-9]*$`)
)

type equipoPublicadoCronosC10 struct {
	EquipoRef       string `json:"equipo_ref"`
	Nombre          string `json:"nombre"`
	VersionRef      string `json:"version_ref"`
	FuenteRef       string `json:"fuente_ref"`
	ReciboFuenteRef string `json:"recibo_fuente_ref"`
}

// El archivo privado enumera rutas y cotejos de procedencia. No concede
// perfiles ni exportaciones: esas decisiones salen de autorización y Cronos.
type configuracionEquipoInformesCronos struct {
	Version   int    `json:"version"`
	Autoridad string `json:"autoridad"`
	Roles     struct {
		JefaturaVersionRolRef string `json:"jefatura_version_rol_ref"`
		RRHHVersionRolRef     string `json:"rrhh_version_rol_ref"`
	} `json:"roles"`
	Equipos []equipoPublicadoCronosC10 `json:"equipos"`
	Motivos struct {
		PresenciaJefatura   core.ReferenciaEntradaCatalogo `json:"presencia_jefatura"`
		PresenciaRRHH       core.ReferenciaEntradaCatalogo `json:"presencia_rrhh"`
		IncidenciasJefatura core.ReferenciaEntradaCatalogo `json:"incidencias_jefatura"`
		IncidenciasRRHH     core.ReferenciaEntradaCatalogo `json:"incidencias_rrhh"`
		InformePropio       core.ReferenciaEntradaCatalogo `json:"informe_propio"`
		InformeJefatura     core.ReferenciaEntradaCatalogo `json:"informe_jefatura"`
		InformeRRHH         core.ReferenciaEntradaCatalogo `json:"informe_rrhh"`
	} `json:"motivos"`
}

func cargarConfiguracionEquipoInformesCronos(cfg config.Config) (configuracionEquipoInformesCronos, error) {
	var vacia configuracionEquipoInformesCronos
	activo, err := cfg.CronosEquipoInformesDesarrolloActivo()
	if err != nil || !activo {
		return vacia, ErrComposicionCronosEquipoInformesNoDisponible
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, cfg.CronosEquipoInformesConfigFile)
	contenido, err := leerFicheroMaterialSeguro(ruta, 128<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, ErrComposicionCronosEquipoInformesNoDisponible
	}
	defer borrarBytes(contenido)
	var c configuracionEquipoInformesCronos
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) ||
		validarConfiguracionEquipoInformesCronos(c) != nil {
		return vacia, ErrComposicionCronosEquipoInformesNoDisponible
	}
	return c, nil
}

func validarConfiguracionEquipoInformesCronos(c configuracionEquipoInformesCronos) error {
	if c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		!referenciaRolC10.MatchString(c.Roles.JefaturaVersionRolRef) ||
		!referenciaRolC10.MatchString(c.Roles.RRHHVersionRolRef) ||
		c.Roles.JefaturaVersionRolRef == c.Roles.RRHHVersionRolRef ||
		len(c.Equipos) == 0 || len(c.Equipos) > 32 {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	vistos := make(map[string]bool, len(c.Equipos))
	for _, e := range c.Equipos {
		if !referenciaEquipoRutaC10.MatchString(e.EquipoRef) || vistos[e.EquipoRef] ||
			!referenciaFotoC10.MatchString(e.VersionRef) ||
			!referenciaFotoC10.MatchString(e.FuenteRef) ||
			!referenciaFotoC10.MatchString(e.ReciboFuenteRef) ||
			len(e.Nombre) < 1 || len(e.Nombre) > 120 || strings.TrimSpace(e.Nombre) != e.Nombre ||
			strings.ContainsAny(e.Nombre, "\r\n\t") {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
		vistos[e.EquipoRef] = true
	}
	for _, r := range []core.ReferenciaEntradaCatalogo{
		c.Motivos.PresenciaJefatura, c.Motivos.PresenciaRRHH,
		c.Motivos.IncidenciasJefatura, c.Motivos.IncidenciasRRHH,
		c.Motivos.InformePropio, c.Motivos.InformeJefatura, c.Motivos.InformeRRHH,
	} {
		if !core.ReferenciaMotivoAutorizacionV2Valida(r) {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	return nil
}

// Las cuatro audiencias adicionales usan exclusivamente la raíz y el
// publicador de CT. Jefatura y RRHH comparten audiencia y se separan por
// perfil fijo AD273, contexto del actor y recurso.
func descriptoresMaterialEquipoInformesCronosDesarrollo() [4]descriptorMaterialConsumidorV3Desarrollo {
	return [4]descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: cronosapp.AudienciaPresenciaEquipo, Dominio: "vec.cronos.presencia-equipo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-presencia-equipo:", ProveedorNominal: "proveedor-material-cronos-presencia-equipo"},
		{Audiencia: cronosapp.AudienciaIncidenciasEquipo, Dominio: "vec.cronos.incidencias-equipo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-incidencias-equipo:", ProveedorNominal: "proveedor-material-cronos-incidencias-equipo"},
		{Audiencia: cronosapp.AudienciaExportarInformePropioC12, Dominio: "vec.cronos.informe-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-informe-propio:", ProveedorNominal: "proveedor-material-cronos-informe-propio"},
		{Audiencia: cronosapp.AudienciaExportarInformeEquipoC12, Dominio: "vec.cronos.informe-equipo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-informe-equipo:", ProveedorNominal: "proveedor-material-cronos-informe-equipo"},
	}
}

func publicarMaterialEquipoInformesCronosEnLote(
	ctx context.Context, cfg config.Config, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) ([4]*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	var vacio [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
	activo, err := cfg.CronosEquipoInformesDesarrolloActivo()
	if err != nil {
		return vacio, ErrComposicionCronosEquipoInformesNoDisponible
	}
	if !activo {
		return vacio, nil
	}
	if ctx == nil || ctx.Err() != nil || gobierno == nil {
		return vacio, ErrComposicionCronosEquipoInformesNoDisponible
	}
	descriptores := descriptoresMaterialEquipoInformesCronosDesarrollo()
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores[:])
	if err != nil {
		return vacio, ErrComposicionCronosEquipoInformesNoDisponible
	}
	var derivados [4]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range derivados {
			borrarBytes(derivados[i].claveHMAC)
		}
	}()
	for i, descriptor := range descriptores {
		if _, ok := catalogo.descriptorPara(descriptor.Audiencia); !ok {
			return vacio, ErrComposicionCronosEquipoInformesNoDisponible
		}
		derivados[i], err = derivarMaterialConsumidorV3Desarrollo(material, descriptor)
		if err != nil {
			return vacio, ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	var proveedores [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
	err = ejecutarTransaccionGobiernoCTDesarrolloUnaVez(ctx, gobierno, func(tx pgx.Tx) error {
		for i := range derivados {
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &derivados[i]); err != nil {
				return err
			}
			var err error
			proveedores[i], err = nuevoProveedorMaterialAutorizacionBaseDesarrollo(derivados[i], reloj)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return vacio, ErrComposicionCronosEquipoInformesNoDisponible
	}
	return proveedores, nil
}

type resolutorAlcanceEquipoInformesCronos struct {
	fuente      vecports.FuenteAutorizacion
	reloj       vecports.Reloj
	rolJefatura string
	rolRRHH     string
}

func (r resolutorAlcanceEquipoInformesCronos) ResolverAlcanceEquipoC10(
	ctx context.Context, id cronoscomp.IdentidadRegistradaCronos,
) (cronosports.AlcanceConsultaEquipo, error) {
	if ctx == nil || ctx.Err() != nil || r.fuente == nil || r.reloj == nil ||
		id.Contexto.Validar() != nil || id.Vinculo.ValidarPara(id.Contexto) != nil {
		return "", ErrComposicionCronosEquipoInformesNoDisponible
	}
	actor := id.Contexto.Contexto
	ahora := r.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) ||
		!id.Vinculo.VigenteEn(ahora, id.Contexto) {
		return "", ErrComposicionCronosEquipoInformesNoDisponible
	}
	i, err := r.fuente.ObtenerInstantaneaAutorizacion(ctx, actor.PersonaRef, actor.PerfilActivoRef)
	if err != nil || i.Validar() != nil || i.AsignacionPerfil.PrincipalID != actor.PersonaRef ||
		i.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef ||
		!i.AsignacionPerfil.VigenteEn(ahora) ||
		i.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		i.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		i.VersionRol.PublicadaEn.After(ahora) || i.ControlVigenciaVersionRol.ActualizadoEn.After(ahora) ||
		!rolEquipoInformesTieneConsultas(i.VersionRol.Concesiones) {
		return "", ErrComposicionCronosEquipoInformesNoDisponible
	}
	switch i.VersionRol.Referencia() {
	case r.rolJefatura:
		return cronosports.AlcanceEquipoJefatura, nil
	case r.rolRRHH:
		return cronosports.AlcanceEquipoRRHH, nil
	default:
		return "", ErrComposicionCronosEquipoInformesNoDisponible
	}
}

func rolEquipoInformesTieneConsultas(concesiones []core.ConcesionRol) bool {
	necesarias := map[string]string{
		cronosapp.AccionPresenciaEquipo:   "presencia_equipo",
		cronosapp.AccionIncidenciasEquipo: "incidencias_equipo",
	}
	for _, c := range concesiones {
		tipo, esperada := necesarias[c.Accion]
		if !esperada {
			continue
		}
		if c.ModuloID != "cronos" || c.TipoRecurso != tipo ||
			!c.AdmiteFinalidad("consultar_"+strings.TrimSuffix(tipo, "_equipo")+"_equipo") ||
			!contieneCampoEquipoInformes(c.CamposPermitidos, "empleado_etiqueta") {
			return false
		}
		delete(necesarias, c.Accion)
	}
	return len(necesarias) == 0
}

func contieneCampoEquipoInformes(campos []string, buscado string) bool {
	for _, campo := range campos {
		if campo == buscado {
			return true
		}
	}
	return false
}

type idiomaDocumentoEquipoInformesContexto struct{}

type renderizadorIdiomaEquipoInformesCronos struct {
	porIdioma map[string]*cronoscomp.RenderizadorInformeC12
}

func (r renderizadorIdiomaEquipoInformesCronos) RenderizarInforme(
	ctx context.Context, formato cronosdomain.FormatoInformeCronos,
	preparacion cronosports.PreparacionInformeC12,
) ([]byte, error) {
	idioma, ok := ctx.Value(idiomaDocumentoEquipoInformesContexto{}).(string)
	if !ok || idioma != preparacion.Idioma || r.porIdioma[idioma] == nil {
		return nil, ErrComposicionCronosEquipoInformesNoDisponible
	}
	return r.porIdioma[idioma].RenderizarInforme(ctx, formato, preparacion)
}

func renderizadoresEquipoInformesCronos(catalogos map[string][]byte) (
	renderizadorIdiomaEquipoInformesCronos, error,
) {
	vacio := renderizadorIdiomaEquipoInformesCronos{}
	if len(catalogos) < 2 || len(catalogos) > 16 {
		return vacio, ErrComposicionCronosEquipoInformesNoDisponible
	}
	resultado := renderizadorIdiomaEquipoInformesCronos{
		porIdioma: make(map[string]*cronoscomp.RenderizadorInformeC12, len(catalogos)),
	}
	for idioma, contenido := range catalogos {
		if !regexp.MustCompile(`^[a-z]{2,8}(?:-[A-Za-z0-9]{2,8})?$`).MatchString(idioma) ||
			len(contenido) == 0 || len(contenido) > 64<<10 ||
			validarClavesJSONUnicas(contenido) != nil {
			return vacio, ErrComposicionCronosEquipoInformesNoDisponible
		}
		var datos struct {
			Equipo    json.RawMessage `json:"equipo"`
			Documento struct {
				Titulo      string            `json:"titulo"`
				Desconocido string            `json:"desconocido"`
				Etiquetas   map[string]string `json:"etiquetas"`
			} `json:"documento"`
		}
		dec := json.NewDecoder(bytes.NewReader(contenido))
		dec.DisallowUnknownFields()
		var extra any
		if dec.Decode(&datos) != nil || !errors.Is(dec.Decode(&extra), io.EOF) ||
			len(datos.Equipo) == 0 || len(datos.Documento.Etiquetas) < 10 {
			return vacio, ErrComposicionCronosEquipoInformesNoDisponible
		}
		renderer, err := cronoscomp.NuevoRenderizadorInformeC12(cronoscomp.CatalogoInformeC12{
			Idioma: idioma, Titulo: datos.Documento.Titulo,
			Desconocido: datos.Documento.Desconocido,
			Etiquetas:   datos.Documento.Etiquetas,
		})
		if err != nil {
			return vacio, ErrComposicionCronosEquipoInformesNoDisponible
		}
		resultado.porIdioma[idioma] = renderer
	}
	return resultado, nil
}

func conIdiomaDocumentoEquipoInformesCronos(
	h http.Handler, catalogos renderizadorIdiomaEquipoInformesCronos,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || h == nil {
			responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
			return
		}
		idioma := r.Header.Get("Accept-Language")
		if catalogos.porIdioma[idioma] == nil {
			responderDenegacionCronosEmpleado(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		ctx := context.WithValue(r.Context(), idiomaDocumentoEquipoInformesContexto{}, idioma)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

var _ cronosports.RenderizadorInformeC12 = renderizadorIdiomaEquipoInformesCronos{}

var fachadasEquipoInformesCronos = [7]string{
	"consumir_cronos_presencia_equipo_jefatura_v3_atestada",
	"consumir_cronos_presencia_equipo_rrhh_v3_atestada",
	"consumir_cronos_incidencias_equipo_jefatura_v3_atestada",
	"consumir_cronos_incidencias_equipo_rrhh_v3_atestada",
	"consumir_cronos_informe_propio_exportar_v3_atestada",
	"consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada",
	"consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada",
}

func preflightEquipoInformesCronos(ctx context.Context, ejecutor *pgxpool.Pool) error {
	if ctx == nil || ctx.Err() != nil || ejecutor == nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	for _, firma := range []string{
		"vec_cronos_v1.foto_equipo_c10_v1(text)",
		"vec_cronos_v1.consultar_equipo_periodo_c10_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_cronos_v1.politica_informe_c12_v1(text,text,text)",
		"vec_cronos_v1.preparar_informe_c12_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_cronos_v1.confirmar_informe_c12_v1(text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	} {
		var disponible bool
		if ejecutor.QueryRow(ctx, `SELECT coalesce(pg_catalog.has_function_privilege(
			pg_catalog.to_regprocedure($1),'EXECUTE'),false)`, firma).Scan(&disponible) != nil || !disponible {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	for _, nombre := range fachadasEquipoInformesCronos {
		var disponible bool
		if ejecutor.QueryRow(ctx, consultaPermisoConsumoReciboCorreccionCronos,
			nombre, "vec_cronos_v1_propietario").Scan(&disponible) != nil || !disponible {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	return nil
}

func politicaInformeDisponibleCronos(
	ctx context.Context, ejecutor *pgxpool.Pool, alcance string,
	tipo cronosdomain.TipoInformeCronos, formato cronosdomain.FormatoInformeCronos,
) (bool, error) {
	if ctx == nil || ctx.Err() != nil || ejecutor == nil {
		return false, ErrComposicionCronosEquipoInformesNoDisponible
	}
	var bruto []byte
	err := ejecutor.QueryRow(ctx,
		`SELECT vec_cronos_v1.politica_informe_c12_v1($1,$2,$3)`,
		alcance, string(tipo), string(formato)).Scan(&bruto)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "42501" {
			return false, nil
		}
		return false, ErrComposicionCronosEquipoInformesNoDisponible
	}
	defer borrarBytes(bruto)
	if len(bruto) == 0 || len(bruto) > 4096 {
		return false, ErrComposicionCronosEquipoInformesNoDisponible
	}
	return true, nil
}

type dependenciasEquipoInformesCronos struct {
	cfg               config.Config
	autoridad         *autoridadCronosEmpleadoDesarrollo
	rutas             map[string]http.Handler
	ejecutor          *pgxpool.Pool
	fuente            vecports.FuenteAutorizacion
	validadorMotivos  vecports.ValidadorReferenciaMotivoAutorizacionV2
	catalogoMotivosID string
	autorizador       *cronoscomp.AutorizadorCronos
	zona              *time.Location
	catalogos         map[string][]byte
	proveedores       [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
}

// La llamada se hace tras componer Cronos base. Construye en memoria todo el
// corte y añade únicamente rutas exactas después de validar cada dependencia.
// Un error deja intacto el mapa de rutas previo; Dirección lo registra y sigue.
func montarEquipoInformesCronos(
	ctx context.Context, d dependenciasEquipoInformesCronos,
) error {
	activo, err := d.cfg.CronosEquipoInformesDesarrolloActivo()
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	if !activo {
		return nil
	}
	if ctx == nil || ctx.Err() != nil || d.autoridad == nil || d.rutas == nil ||
		d.ejecutor == nil || d.fuente == nil || d.validadorMotivos == nil ||
		d.autorizador == nil || d.zona == nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	for _, p := range d.proveedores {
		if p == nil {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	c, err := cargarConfiguracionEquipoInformesCronos(d.cfg)
	if err != nil || preflightEquipoInformesCronos(ctx, d.ejecutor) != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	refs := []core.ReferenciaEntradaCatalogo{
		c.Motivos.PresenciaJefatura, c.Motivos.PresenciaRRHH,
		c.Motivos.IncidenciasJefatura, c.Motivos.IncidenciasRRHH,
		c.Motivos.InformePropio, c.Motivos.InformeJefatura, c.Motivos.InformeRRHH,
	}
	for _, ref := range refs {
		if ref.CatalogoID != d.catalogoMotivosID ||
			d.validadorMotivos.ValidarReferenciaMotivoAutorizacionV2(ctx, ref, time.Now().UTC()) != nil {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	renderer, err := renderizadoresEquipoInformesCronos(d.catalogos)
	if err != nil {
		return err
	}
	perfiles := resolutorAlcanceEquipoInformesCronos{
		fuente: d.fuente, reloj: relojRutasDietas{},
		rolJefatura: c.Roles.JefaturaVersionRolRef, rolRRHH: c.Roles.RRHHVersionRolRef,
	}
	identidad := seguridadCronosEmpleadoDesarrollo{autoridad: d.autoridad}
	repoEquipo, err := cronospg.NuevoRepositorioEquipoInformesC10(d.ejecutor)
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	servicioEquipo, err := cronosapp.NuevoServicioConsultaEquipo(repoEquipo, relojCronosEmpleado{}, d.zona)
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	resolutorEquipo, err := cronoscomp.NuevoResolutorEquipoC10(identidad, perfiles, d.autorizador,
		cronoscomp.MotivosEquipoC10{
			PresenciaJefatura: c.Motivos.PresenciaJefatura, PresenciaRRHH: c.Motivos.PresenciaRRHH,
			IncidenciasJefatura: c.Motivos.IncidenciasJefatura, IncidenciasRRHH: c.Motivos.IncidenciasRRHH,
		})
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	manejadorEquipo, err := cronoshttp.NuevoManejadorEquipoC10(servicioEquipo, resolutorEquipo)
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	repoInforme, err := cronospg.NuevoRepositorioInformeC12(d.ejecutor)
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	servicioInforme, err := cronosapp.NuevoServicioInformeC12(repoInforme, renderer, relojCronosEmpleado{}, d.zona)
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	resolutorInforme, err := cronoscomp.NuevoResolutorInformeC12(identidad, perfiles, d.autorizador,
		cronoscomp.MotivosInformeC12{
			Propio: c.Motivos.InformePropio, Jefatura: c.Motivos.InformeJefatura,
			RRHH: c.Motivos.InformeRRHH,
		})
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	manejadorInforme, err := cronoshttp.NuevoManejadorInformeC12(servicioInforme, resolutorInforme)
	if err != nil {
		return ErrComposicionCronosEquipoInformesNoDisponible
	}
	informesConIdioma := conIdiomaDocumentoEquipoInformesCronos(manejadorInforme, renderer)
	adiciones := make(map[string]http.Handler)
	var equipos = make(map[string]cronosdomain.FotoEquipoAutoritativa, len(c.Equipos))
	for _, e := range c.Equipos {
		foto, err := repoEquipo.FotoEquipo(ctx, e.EquipoRef)
		if err != nil || foto.EquipoRef != e.EquipoRef || foto.VersionRef != e.VersionRef ||
			foto.FuenteRef != e.FuenteRef || foto.ReciboFuenteRef != e.ReciboFuenteRef {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
		equipos[e.EquipoRef] = foto
		adiciones[cronoshttp.RutaEquipoCronosC10+e.EquipoRef+"/presencia"] = manejadorEquipo
		adiciones[cronoshttp.RutaEquipoCronosC10+e.EquipoRef+"/incidencias"] = manejadorEquipo
	}
	tipos := []cronosdomain.TipoInformeCronos{
		cronosdomain.InformeSaldo, cronosdomain.InformeMovimientos, cronosdomain.InformePermisos,
	}
	formatos := []cronosdomain.FormatoInformeCronos{cronosdomain.InformePDF, cronosdomain.InformeCSV}
	for _, tipo := range tipos {
		for _, formato := range formatos {
			nombre := string(tipo) + "." + string(formato)
			propio, err := politicaInformeDisponibleCronos(ctx, d.ejecutor, "empleado", tipo, formato)
			if err != nil {
				return err
			}
			jefatura, err := politicaInformeDisponibleCronos(ctx, d.ejecutor, "jefatura", tipo, formato)
			if err != nil {
				return err
			}
			rrhh, err := politicaInformeDisponibleCronos(ctx, d.ejecutor, "rrhh", tipo, formato)
			if err != nil {
				return err
			}
			if propio {
				adiciones[cronoshttp.RutaInformesCronosC12+nombre] = informesConIdioma
			}
			if jefatura || rrhh {
				for equipo := range equipos {
					adiciones[cronoshttp.RutaEquipoCronosC10+equipo+"/informes/"+nombre] = informesConIdioma
				}
			}
		}
	}
	adiciones[cronoshttp.RutaInformesCronosC12+"oferta"] =
		manejadorOfertaEquipoInformesCronos(d, identidad, perfiles, "", equipos)
	for equipo := range equipos {
		adiciones[cronoshttp.RutaEquipoCronosC10+equipo+"/informes/oferta"] =
			manejadorOfertaEquipoInformesCronos(d, identidad, perfiles, equipo, equipos)
	}
	for ruta := range adiciones {
		if _, existe := d.rutas[ruta]; existe {
			return ErrComposicionCronosEquipoInformesNoDisponible
		}
	}
	for ruta, h := range adiciones {
		d.rutas[ruta] = h
	}
	return nil
}

func manejadorOfertaEquipoInformesCronos(
	d dependenciasEquipoInformesCronos,
	identidad cronoscomp.ResolutorIdentidadRegistradaCronos,
	perfiles resolutorAlcanceEquipoInformesCronos,
	equipo string,
	equipos map[string]cronosdomain.FotoEquipoAutoritativa,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r == nil || r.URL == nil || r.Method != http.MethodGet || r.URL.RawQuery != "" ||
			r.URL.RawPath != "" {
			responderDenegacionCronosEmpleado(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		id, err := identidad.ResolverIdentidadRegistradaCronos(r.Context())
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		if err != nil || id.Contexto.Validar() != nil || id.Vinculo.ValidarPara(id.Contexto) != nil ||
			!id.Contexto.Contexto.Instantanea.VigenteEn(ahora) ||
			!id.Vinculo.VigenteEn(ahora, id.Contexto) {
			responderDenegacionCronosEmpleado(w, http.StatusForbidden, "acceso_denegado")
			return
		}
		alcance := "empleado"
		if equipo != "" {
			foto, existe := equipos[equipo]
			if !existe {
				responderDenegacionCronosEmpleado(w, http.StatusNotFound, "no_disponible")
				return
			}
			a, err := perfiles.ResolverAlcanceEquipoC10(r.Context(), id)
			if err != nil || (a != cronosports.AlcanceEquipoJefatura && a != cronosports.AlcanceEquipoRRHH) ||
				!perfilCubreEquipoInformesCronos(r.Context(), d.fuente, id, foto) {
				responderDenegacionCronosEmpleado(w, http.StatusForbidden, "acceso_denegado")
				return
			}
			alcance = string(a)
		} else if !perfilTieneInformePropioCronos(r.Context(), d.fuente, id) {
			responderDenegacionCronosEmpleado(w, http.StatusForbidden, "acceso_denegado")
			return
		}
		opciones := make([]string, 0, 6)
		for _, tipo := range []cronosdomain.TipoInformeCronos{
			cronosdomain.InformeSaldo, cronosdomain.InformeMovimientos, cronosdomain.InformePermisos,
		} {
			for _, formato := range []cronosdomain.FormatoInformeCronos{
				cronosdomain.InformePDF, cronosdomain.InformeCSV,
			} {
				disponible, err := politicaInformeDisponibleCronos(r.Context(), d.ejecutor, alcance, tipo, formato)
				if err != nil {
					responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
					return
				}
				if disponible {
					opciones = append(opciones, string(tipo)+"."+string(formato))
				}
			}
		}
		_ = json.NewEncoder(w).Encode(struct {
			EquipoRef     string   `json:"equipo_ref,omitempty"`
			Alcance       string   `json:"alcance"`
			Exportaciones []string `json:"exportaciones"`
		}{equipo, alcance, opciones})
	})
}

func perfilCubreEquipoInformesCronos(
	ctx context.Context, fuente vecports.FuenteAutorizacion,
	id cronoscomp.IdentidadRegistradaCronos, foto cronosdomain.FotoEquipoAutoritativa,
) bool {
	if ctx == nil || ctx.Err() != nil || fuente == nil {
		return false
	}
	actor := id.Contexto.Contexto
	i, err := fuente.ObtenerInstantaneaAutorizacion(ctx, actor.PersonaRef, actor.PerfilActivoRef)
	if err != nil || i.Validar() != nil || !i.AsignacionPerfil.VigenteEn(time.Now().UTC()) ||
		i.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		i.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		!rolTieneExportacionEquipoInformes(i.VersionRol.Concesiones, false) {
		return false
	}
	recurso := core.RecursoAutorizable{
		Referencia: foto.EquipoRef, ModuloID: "cronos", Tipo: "informe_equipo",
		Ambitos: map[string]string{
			"equipo_ref": foto.EquipoRef, "unidad_ref": foto.UnidadRef,
			"organismo_ref": foto.OrganismoRef,
		},
	}
	return i.AsignacionPerfil.Cubre(recurso)
}

func perfilTieneInformePropioCronos(
	ctx context.Context, fuente vecports.FuenteAutorizacion,
	id cronoscomp.IdentidadRegistradaCronos,
) bool {
	if ctx == nil || ctx.Err() != nil || fuente == nil {
		return false
	}
	actor := id.Contexto.Contexto
	i, err := fuente.ObtenerInstantaneaAutorizacion(ctx, actor.PersonaRef, actor.PerfilActivoRef)
	return err == nil && i.Validar() == nil &&
		i.AsignacionPerfil.VigenteEn(time.Now().UTC()) &&
		i.VersionRol.Estado == core.EstadoVersionRolPublicada &&
		i.ControlVigenciaVersionRol.Estado == core.EstadoControlVigenciaVersionRolHabilitada &&
		rolTieneExportacionEquipoInformes(i.VersionRol.Concesiones, true)
}

func rolTieneExportacionEquipoInformes(concesiones []core.ConcesionRol, propia bool) bool {
	accion, tipo, finalidad := cronosapp.AccionExportarInformeEquipoC12,
		"informe_equipo", cronosapp.FinalidadExportarInformeEquipoC12
	if propia {
		accion, tipo, finalidad = cronosapp.AccionExportarInformePropioC12,
			"informe_propio", cronosapp.FinalidadExportarInformePropioC12
	}
	for _, c := range concesiones {
		if c.Accion == accion && c.ModuloID == "cronos" && c.TipoRecurso == tipo &&
			c.AdmiteFinalidad(finalidad) &&
			contieneCampoEquipoInformes(c.CamposPermitidos, "archivo") &&
			contieneCampoEquipoInformes(c.CamposPermitidos, "recibo") {
			return true
		}
	}
	return false
}
