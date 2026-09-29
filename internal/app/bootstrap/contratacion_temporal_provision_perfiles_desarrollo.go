package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgrescontexto "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	EnvCTPerfilesProvisionadorDatabaseURL     = "VEC_CT_PERFILES_PROVISIONADOR_DATABASE_URL"
	EnvCTPerfilesContextoHistoricoDatabaseURL = "VEC_CT_PERFILES_CONTEXTO_HISTORICO_DATABASE_URL"
	esquemaManifiestoProvisionPerfilesCT      = "vec.ct.perfiles-demo.provision.v1"
	tamanoMaximoManifiestoPerfilesCT          = 32 << 10
)

var (
	errProvisionPerfilesCTEntrada       = errors.New("ct_perfiles_entrada_invalida")
	errProvisionPerfilesCTNoDisponible  = errors.New("ct_perfiles_no_disponible")
	errProvisionPerfilesCTObsoleta      = errors.New("ct_perfiles_preimagen_obsoleta")
	huellaProvisionPerfilesCTValida     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	aprobacionProvisionPerfilesCTValida = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)
)

// SolicitudProvisionPerfilesCT sólo acepta datos de gobierno, nunca identidad,
// ámbitos ni concesiones del cliente. Estos últimos se derivan del material
// privado de desarrollo y de los constructores CT revisados.
type SolicitudProvisionPerfilesCT struct {
	ManifiestoRuta   string
	ManifiestoSHA256 string
	AprobacionRef    string
	Aplicar          bool
	Preparar         bool
}

type ReciboPerfilProvisionCT struct {
	Clave         string `json:"clave"`
	PerfilRef     string `json:"perfil_ref"`
	AsignacionRef string `json:"asignacion_ref"`
	Version       int    `json:"version"`
	HuellaSHA256  string `json:"huella_sha256"`
	Reutilizada   bool   `json:"reutilizada"`
}

type ResultadoProvisionPerfilesCT struct {
	Estado           string                    `json:"estado"`
	ManifiestoSHA256 string                    `json:"manifiesto_sha256"`
	AprobacionRef    string                    `json:"aprobacion_ref,omitempty"`
	Perfiles         []ReciboPerfilProvisionCT `json:"perfiles,omitempty"`
	Manifiesto       json.RawMessage           `json:"manifiesto,omitempty"`
}

type manifiestoProvisionPerfilesCT struct {
	Esquema       string                      `json:"esquema"`
	AprobacionRef string                      `json:"aprobacion_ref"`
	Perfiles      [2]entradaProvisionPerfilCT `json:"perfiles"`
}

type entradaProvisionPerfilCT struct {
	Clave                  string `json:"clave"`
	PerfilRef              string `json:"perfil_ref"`
	ContextoRef            string `json:"contexto_ref"`
	ContextoHuellaSHA256   string `json:"contexto_huella_sha256"`
	PreimagenAsignacionRef string `json:"preimagen_asignacion_ref"`
	PreimagenSHA256        string `json:"preimagen_sha256"`
	ObjetivoAsignacionRef  string `json:"objetivo_asignacion_ref"`
	ObjetivoSHA256         string `json:"objetivo_sha256"`
}

func (m manifiestoProvisionPerfilesCT) validar() error {
	if m.Esquema != esquemaManifiestoProvisionPerfilesCT ||
		(m.AprobacionRef != "" && !aprobacionProvisionPerfilesCTValida.MatchString(m.AprobacionRef)) ||
		m.Perfiles[0].Clave != "alta" || m.Perfiles[1].Clave != "cobertura" ||
		m.Perfiles[0].PerfilRef == m.Perfiles[1].PerfilRef {
		return errProvisionPerfilesCTEntrada
	}
	for _, p := range m.Perfiles {
		if p.PerfilRef == "" || p.ContextoRef == "" ||
			!huellaProvisionPerfilesCTValida.MatchString(p.ContextoHuellaSHA256) ||
			p.ObjetivoAsignacionRef == "" ||
			!huellaProvisionPerfilesCTValida.MatchString(p.ObjetivoSHA256) ||
			(p.PreimagenAsignacionRef == "") != (p.PreimagenSHA256 == "") ||
			(p.PreimagenSHA256 != "" && !huellaProvisionPerfilesCTValida.MatchString(p.PreimagenSHA256)) {
			return errProvisionPerfilesCTEntrada
		}
	}
	return nil
}

func leerManifiestoProvisionPerfilesCT(ruta, huellaEsperada, aprobacion string) (manifiestoProvisionPerfilesCT, string, error) {
	vacio := manifiestoProvisionPerfilesCT{}
	if ruta == "" || !huellaProvisionPerfilesCTValida.MatchString(huellaEsperada) ||
		!aprobacionProvisionPerfilesCTValida.MatchString(aprobacion) {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 ||
		info.Size() <= 0 || info.Size() > tamanoMaximoManifiestoPerfilesCT {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	f, err := os.Open(ruta)
	if err != nil {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	defer f.Close()
	abierta, err := f.Stat()
	if err != nil || !os.SameFile(info, abierta) || !abierta.Mode().IsRegular() {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	contenido, err := io.ReadAll(io.LimitReader(f, tamanoMaximoManifiestoPerfilesCT+1))
	if err != nil || len(contenido) == 0 || len(contenido) > tamanoMaximoManifiestoPerfilesCT {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	huella := sha256.Sum256(contenido)
	huellaTexto := hex.EncodeToString(huella[:])
	if huellaTexto != huellaEsperada {
		return vacio, "", errProvisionPerfilesCTObsoleta
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var m manifiestoProvisionPerfilesCT
	if dec.Decode(&m) != nil || dec.Decode(new(any)) != io.EOF || m.validar() != nil ||
		m.AprobacionRef != aprobacion {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	canon, err := json.Marshal(m)
	if err != nil || !bytes.Equal(canon, contenido) {
		return vacio, "", errProvisionPerfilesCTEntrada
	}
	return m, huellaTexto, nil
}

func huellaJSONProvisionPerfilesCT(valor any) (string, error) {
	// El adaptador PostgreSQL decodifica el catálogo vacío como [] y las
	// semillas locales lo construyen como nil. Ambos representan la misma
	// instantánea sin políticas; la huella del manifiesto fija una forma única.
	if i, ok := valor.(dominiovec.InstantaneaAutorizacion); ok && len(i.Politicas) == 0 {
		i.Politicas = nil
		valor = i
	}
	b, err := json.Marshal(valor)
	if err != nil {
		return "", errProvisionPerfilesCTEntrada
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func manifiestoCanonicoProvisionPerfilesCT(m manifiestoProvisionPerfilesCT) (json.RawMessage, string, error) {
	if m.validar() != nil {
		return nil, "", errProvisionPerfilesCTEntrada
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, "", errProvisionPerfilesCTEntrada
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func EjecutarProvisionPerfilesCT(ctx context.Context, cfg config.Config, s SolicitudProvisionPerfilesCT) (ResultadoProvisionPerfilesCT, error) {
	if ctx == nil || ctx.Err() != nil || !cfg.DevelopmentEnabledByDoubleKey() ||
		s.Preparar == s.Aplicar ||
		(s.Preparar && (s.ManifiestoRuta != "" || s.ManifiestoSHA256 != "")) ||
		(s.Aplicar && (s.ManifiestoRuta == "" || s.ManifiestoSHA256 == "" || s.AprobacionRef == "")) {
		return ResultadoProvisionPerfilesCT{}, errProvisionPerfilesCTEntrada
	}
	return ejecutarProvisionPerfilesCTPostgreSQL(ctx, cfg, s)
}

func ejecutarProvisionPerfilesCTPostgreSQL(ctx context.Context, cfg config.Config,
	s SolicitudProvisionPerfilesCT) (ResultadoProvisionPerfilesCT, error) {
	vacio := ResultadoProvisionPerfilesCT{}
	ctx, cancelar := plazoProvisionPerfilesCT(ctx)
	defer cancelar()
	var aprobado manifiestoProvisionPerfilesCT
	var huellaAprobada string
	if s.Aplicar {
		var err error
		aprobado, huellaAprobada, err = leerManifiestoProvisionPerfilesCT(
			s.ManifiestoRuta, s.ManifiestoSHA256, s.AprobacionRef)
		if err != nil {
			return vacio, err
		}
	}
	conexiones, err := abrirConexionesProvisionPerfilesCT(ctx, cfg, s.Preparar)
	if err != nil {
		return vacio, errProvisionPerfilesCTNoDisponible
	}
	defer conexiones.cerrar()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	perfiles, soporte, err := geometriaProvisionPerfilesCT(cfg, ahora)
	if err != nil {
		return vacio, errProvisionPerfilesCTNoDisponible
	}
	if err = completarContextosProvisionPerfilesCT(ctx, conexiones, soporte, &perfiles); err != nil {
		return vacio, errProvisionPerfilesCTNoDisponible
	}
	for i := range perfiles {
		if err = prepararPerfilProvisionCT(ctx, conexiones, &perfiles[i], ahora, s.Preparar); err != nil {
			return vacio, err
		}
	}
	if s.Preparar {
		m := manifiestoProvisionPerfilesCT{Esquema: esquemaManifiestoProvisionPerfilesCT}
		for i := range perfiles {
			m.Perfiles[i], err = entradaManifiestoProvisionCT(perfiles[i])
			if err != nil {
				return vacio, errProvisionPerfilesCTNoDisponible
			}
		}
		canon, huella, err := manifiestoCanonicoProvisionPerfilesCT(m)
		if err != nil {
			return vacio, err
		}
		return ResultadoProvisionPerfilesCT{
			Estado: "propuesta", ManifiestoSHA256: huella, Manifiesto: canon,
		}, nil
	}
	if err := revalidarContextosActualesProvisionCT(ctx, conexiones.contextoRuntime, soporte, &perfiles); err != nil {
		return vacio, errProvisionPerfilesCTObsoleta
	}
	resultado := ResultadoProvisionPerfilesCT{
		Estado: "confirmada", ManifiestoSHA256: huellaAprobada,
		AprobacionRef: aprobado.AprobacionRef,
		Perfiles:      make([]ReciboPerfilProvisionCT, 0, len(perfiles)),
	}
	for i := range perfiles {
		recibo, err := aplicarPerfilProvisionCT(ctx, conexiones, perfiles[i],
			aprobado.Perfiles[i], huellaAprobada)
		if err != nil {
			return vacio, err
		}
		resultado.Perfiles = append(resultado.Perfiles, recibo)
	}
	// Las dos publicaciones son transacciones separadas. El resultado solo se
	// entrega si ambas siguen siendo las postimágenes exactas tras completar el
	// par; un fallo intermedio se recupera con el mismo manifiesto aprobado.
	for i := range perfiles {
		post, err := conexiones.almacen.ObtenerInstantaneaAutorizacion(ctx,
			perfiles[i].semilla.AsignacionPerfil.PrincipalID,
			perfiles[i].semilla.AsignacionPerfil.PerfilActivoRef)
		huella, errHuella := huellaJSONProvisionPerfilesCT(post)
		if err != nil || errHuella != nil || huella != aprobado.Perfiles[i].ObjetivoSHA256 ||
			post.AsignacionPerfil.Referencia() != aprobado.Perfiles[i].ObjetivoAsignacionRef ||
			validarPreimagenProvisionPerfilCT(post, perfiles[i].semilla, time.Now().UTC().Truncate(time.Microsecond)) != nil {
			return vacio, errProvisionPerfilesCTObsoleta
		}
	}
	return resultado, nil
}

// Aplicar puede añadir recibos de contexto V2. Preparar nunca invoca esta
// autoridad con escritura; su lectura histórica se mantiene separada.
func revalidarContextosActualesProvisionCT(ctx context.Context, pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo, perfiles *[2]perfilPreparadoProvisionCT) error {
	if ctx == nil || pool == nil || soporte == nil || perfiles == nil {
		return errProvisionPerfilesCTNoDisponible
	}
	resolutor, err := postgrescontexto.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pool)
	if err != nil {
		return errProvisionPerfilesCTNoDisponible
	}
	servicio, err := aplicacionvec.NuevoServicioContextoActorProductivoV2(
		resolutor, postgrescontexto.NuevoGeneradorOperacionContextoActorV2Criptografico(),
		relojContratacionTemporalDesarrollo{},
	)
	if err != nil {
		return errProvisionPerfilesCTNoDisponible
	}
	autoridad, err := aplicacionvec.NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		return errProvisionPerfilesCTNoDisponible
	}
	return cotejarContextosActualesProvisionCT(ctx, autoridad, soporte, perfiles)
}

func cotejarContextosActualesProvisionCT(ctx context.Context,
	resolutor dominiovec.ResolutorContextoActorRegistradoV2,
	soporte *soporteAltaContratacionTemporalDesarrollo, perfiles *[2]perfilPreparadoProvisionCT) error {
	if ctx == nil || resolutor == nil || soporte == nil || perfiles == nil {
		return errProvisionPerfilesCTNoDisponible
	}
	for i := range perfiles {
		actual, err := contextoEsperadoRegistradoParaSemillaDesarrollo(ctx, resolutor,
			soporte, perfiles[i].contexto.Resultado)
		if err != nil || !mismoContextoEsperadoRegistradoDesarrollo(perfiles[i].registrado, actual) {
			return errProvisionPerfilesCTObsoleta
		}
	}
	return nil
}

// La composición concreta vive debajo: no arranca HTTP ni reutiliza los
// LOGIN de servidor. Cada intento tiene plazo propio, incluso en preparación.
func plazoProvisionPerfilesCT(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 45*time.Second)
}

type perfilPreparadoProvisionCT struct {
	clave      string
	contexto   ctports.ContextoAutorizacionAltaV3
	registrado dominiovec.ResultadoContextoActorRegistradoV2
	semilla    dominiovec.InstantaneaAutorizacion
	preimagen  dominiovec.InstantaneaAutorizacion
	existe     bool
	objetivo   dominiovec.InstantaneaAutorizacion
	replay     bool
}

func geometriaProvisionPerfilesCT(cfg config.Config, ahora time.Time) ([2]perfilPreparadoProvisionCT, *soporteAltaContratacionTemporalDesarrollo, error) {
	vacios := [2]perfilPreparadoProvisionCT{}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	defer func() {
		material.idempotencia.borrar()
		borrarBytes(material.claveKMS[:])
		borrarBytes(material.claveTSA[:])
		borrarBytes(material.firmaAtestacionKMS)
		borrarBytes(material.firmaRevalidacionKMS)
	}()
	if material.identidad == nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	principal, ok := material.identidad.principalConRolUnico(rolTecnicoRRHHContratacionTemporalDesarrollo)
	if !ok || !principalContratacionTemporalDesarrolloValido(principal) {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	alta, err := nuevoContextoAltaFijoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	cobertura, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	vAlta, errAlta := alta.Vinculo.Datos()
	vCobertura, errCobertura := cobertura.Vinculo.Datos()
	if errAlta != nil || errCobertura != nil || vAlta.PrincipalID != vCobertura.PrincipalID ||
		vAlta.PerfilActivoRef == vCobertura.PerfilActivoRef ||
		alta.Resultado.Contexto.Instantanea.CuentaRef != cobertura.Resultado.Contexto.Instantanea.CuentaRef ||
		alta.Resultado.Contexto.PersonaRef != cobertura.Resultado.Contexto.PersonaRef {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	catalogo, err := nuevoCatalogoDesarrollo(cfg.PersonalOrganizacionSourcePath, cfg.RPTCatalogoPath)
	if err != nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	origen := nuevoOrigenConsultasConCatalogoDesarrollo(catalogo)
	semillaAlta, err := nuevaInstantaneaAutorizacionAltaFijaContratacionTemporalDesarrollo(
		vAlta.PrincipalID, vAlta.PerfilActivoRef, ahora, origen)
	if err != nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	semillaCobertura, err := nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
		vCobertura.PrincipalID, vCobertura.PerfilActivoRef, ahora)
	if err != nil {
		return vacios, nil, errProvisionPerfilesCTNoDisponible
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"],
		contextoAltaFijo: alta, contextoCobertura: cobertura,
	}
	return [2]perfilPreparadoProvisionCT{
		{clave: "alta", contexto: alta, semilla: semillaAlta},
		{clave: "cobertura", contexto: cobertura, semilla: semillaCobertura},
	}, soporte, nil
}

func completarContextosProvisionPerfilesCT(ctx context.Context, conexiones conexionesProvisionPerfilesCT,
	soporte *soporteAltaContratacionTemporalDesarrollo, perfiles *[2]perfilPreparadoProvisionCT) error {
	if ctx == nil || conexiones.lector == nil || soporte == nil || perfiles == nil {
		return errProvisionPerfilesCTNoDisponible
	}
	for i := range perfiles {
		semilla := perfiles[i].contexto.Resultado
		registrado, err := conexiones.lector.LeerContextoOriginalV2(ctx,
			puertosvec.SolicitudLecturaContextoOriginalV2{
				RegistroContextoRef:               semilla.RegistroContextoRef,
				HuellaSHA256:                      semilla.HuellaSHA256,
				ManifiestoProcedenciaHuellaSHA256: semilla.ManifiestoProcedenciaHuellaSHA256,
			})
		if err != nil || registrado.Validar() != nil ||
			registrado.Contexto.PerfilActivoRef != perfiles[i].semilla.AsignacionPerfil.PerfilActivoRef ||
			registrado.Contexto.Principal.ID != soporte.principalID ||
			registrado.Contexto.Instantanea.CuentaRef != semilla.Contexto.Instantanea.CuentaRef ||
			registrado.Contexto.PersonaRef != semilla.Contexto.PersonaRef ||
			registrado.Contexto.Instantanea.VinculoRef != semilla.Contexto.Instantanea.VinculoRef {
			return errProvisionPerfilesCTNoDisponible
		}
		perfiles[i].registrado = registrado
	}
	return nil
}

func autoridadProvisionPerfilCT(pool *pgxpool.Pool, p perfilPreparadoProvisionCT, actoAsignacion string) autoridadPostgreSQLDesarrollo {
	return autoridadPostgreSQLDesarrollo{
		pool: pool, vinculo: p.contexto.Vinculo,
		prefijoBloqueo: "vec:ct:desarrollo:autorizacion:",
		actoControlRol: "acto:ct:desarrollo:control-rol:v1",
		actoAsignacion: actoAsignacion,
		actoSesion:     "acto:ct:desarrollo:sesion:v1",
	}
}

func validarPreimagenProvisionPerfilCT(actual, semilla dominiovec.InstantaneaAutorizacion, ahora time.Time) error {
	if actual.Validar() != nil || semilla.Validar() != nil ||
		actual.AsignacionPerfil.Estado != dominiovec.EstadoAsignacionPerfilActiva ||
		!actual.AsignacionPerfil.VigenteEn(ahora) ||
		actual.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		actual.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		actual.AsignacionPerfil.PrincipalID != semilla.AsignacionPerfil.PrincipalID ||
		actual.AsignacionPerfil.PerfilActivoRef != semilla.AsignacionPerfil.PerfilActivoRef ||
		actual.AsignacionPerfil.AsignacionID != semilla.AsignacionPerfil.AsignacionID ||
		actual.VersionRol.RolID != semilla.VersionRol.RolID ||
		!reflect.DeepEqual(actual.AsignacionPerfil.Ambitos, semilla.AsignacionPerfil.Ambitos) ||
		len(actual.Politicas) != 0 ||
		!concesionesContenidasProvisionCT(actual.VersionRol.Concesiones, semilla.VersionRol.Concesiones) {
		return errProvisionPerfilesCTObsoleta
	}
	return nil
}

func concesionesContenidasProvisionCT(actuales, objetivo []dominiovec.ConcesionRol) bool {
	if len(actuales) > len(objetivo) {
		return false
	}
	for _, anterior := range actuales {
		encontrada := false
		for _, propuesta := range objetivo {
			if reflect.DeepEqual(anterior, propuesta) {
				encontrada = true
				break
			}
		}
		if !encontrada {
			return false
		}
	}
	return true
}

func prepararPerfilProvisionCT(ctx context.Context, conexiones conexionesProvisionPerfilesCT,
	p *perfilPreparadoProvisionCT, ahora time.Time, soloLectura bool) error {
	if p == nil || p.semilla.Validar() != nil || p.registrado.Validar() != nil {
		return errProvisionPerfilesCTNoDisponible
	}
	actual, err := conexiones.almacen.ObtenerInstantaneaAutorizacion(ctx,
		p.semilla.AsignacionPerfil.PrincipalID, p.semilla.AsignacionPerfil.PerfilActivoRef)
	switch {
	case err == nil:
		if validarPreimagenProvisionPerfilCT(actual, p.semilla, ahora) != nil {
			return errProvisionPerfilesCTObsoleta
		}
		p.preimagen, p.existe = actual, true
		if reflect.DeepEqual(actual.VersionRol.Concesiones, p.semilla.VersionRol.Concesiones) {
			p.objetivo, p.replay = actual, true
			return nil
		}
		// El corte v1 sólo provisiona las dos semillas fijas y recupera su
		// postimagen exacta. Una ampliación de concesiones tendrá un manifiesto
		// y planificador propio; aquí nunca se infiere desde un binario nuevo.
		return errProvisionPerfilesCTObsoleta
	case errors.Is(err, puertosvec.ErrAsignacionPerfilNoEncontrada):
		p.existe = false
	default:
		return errProvisionPerfilesCTNoDisponible
	}
	if soloLectura {
		// La ausencia de asignación se acredita con la fuente nominal. El
		// objetivo puro es v1; aplicar volverá a comprobar bajo lock/CAS.
		p.objetivo = p.semilla
		return nil
	}
	a := autoridadProvisionPerfilCT(conexiones.provisionador, *p, "acto:ct:perfiles-demo:preparacion")
	objetivo, err := a.prepararInstantanea(ctx, p.semilla, !p.existe)
	if err != nil || objetivo.Validar() != nil ||
		objetivo.AsignacionPerfil.AsignacionID != p.semilla.AsignacionPerfil.AsignacionID ||
		objetivo.AsignacionPerfil.PerfilActivoRef != p.semilla.AsignacionPerfil.PerfilActivoRef ||
		objetivo.VersionRol.RolID != p.semilla.VersionRol.RolID ||
		!reflect.DeepEqual(objetivo.AsignacionPerfil.Ambitos, p.semilla.AsignacionPerfil.Ambitos) ||
		!reflect.DeepEqual(objetivo.VersionRol.Concesiones, p.semilla.VersionRol.Concesiones) ||
		(!p.existe && (objetivo.AsignacionPerfil.Version != 1 ||
			objetivo.VersionRol.Version != p.semilla.VersionRol.Version)) {
		return errProvisionPerfilesCTObsoleta
	}
	p.objetivo = objetivo
	return nil
}

func entradaManifiestoProvisionCT(p perfilPreparadoProvisionCT) (entradaProvisionPerfilCT, error) {
	if p.registrado.Validar() != nil || p.objetivo.Validar() != nil ||
		(p.existe && p.preimagen.Validar() != nil) {
		return entradaProvisionPerfilCT{}, errProvisionPerfilesCTNoDisponible
	}
	hObjetivo, err := huellaJSONProvisionPerfilesCT(p.objetivo)
	if err != nil {
		return entradaProvisionPerfilCT{}, err
	}
	e := entradaProvisionPerfilCT{
		Clave: p.clave, PerfilRef: p.semilla.AsignacionPerfil.PerfilActivoRef,
		ContextoRef:           p.registrado.RegistroContextoRef,
		ContextoHuellaSHA256:  p.registrado.HuellaSHA256,
		ObjetivoAsignacionRef: p.objetivo.AsignacionPerfil.Referencia(),
		ObjetivoSHA256:        hObjetivo,
	}
	if p.existe {
		e.PreimagenAsignacionRef = p.preimagen.AsignacionPerfil.Referencia()
		e.PreimagenSHA256, err = huellaJSONProvisionPerfilesCT(p.preimagen)
		if err != nil {
			return entradaProvisionPerfilCT{}, err
		}
	}
	return e, nil
}

func entradaObjetivoCompatibleProvisionCT(p perfilPreparadoProvisionCT, e entradaProvisionPerfilCT) bool {
	h, err := huellaJSONProvisionPerfilesCT(p.objetivo)
	return err == nil && p.clave == e.Clave &&
		p.semilla.AsignacionPerfil.PerfilActivoRef == e.PerfilRef &&
		p.registrado.RegistroContextoRef == e.ContextoRef &&
		p.registrado.HuellaSHA256 == e.ContextoHuellaSHA256 &&
		p.objetivo.AsignacionPerfil.Referencia() == e.ObjetivoAsignacionRef && h == e.ObjetivoSHA256
}

func preimagenCompatibleProvisionCT(p perfilPreparadoProvisionCT, e entradaProvisionPerfilCT) bool {
	if !p.existe {
		return e.PreimagenAsignacionRef == "" && e.PreimagenSHA256 == ""
	}
	h, err := huellaJSONProvisionPerfilesCT(p.preimagen)
	return err == nil && e.PreimagenAsignacionRef == p.preimagen.AsignacionPerfil.Referencia() &&
		e.PreimagenSHA256 == h
}

func actoAsignacionProvisionCT(huellaManifest, clave string) string {
	h := sha256.Sum256([]byte(huellaManifest + "\x00" + clave))
	return "acto:ct:perfiles-demo:" + hex.EncodeToString(h[:])
}

func aplicarPerfilProvisionCT(ctx context.Context, conexiones conexionesProvisionPerfilesCT,
	p perfilPreparadoProvisionCT, e entradaProvisionPerfilCT, huellaManifest string) (ReciboPerfilProvisionCT, error) {
	vacio := ReciboPerfilProvisionCT{}
	if !entradaObjetivoCompatibleProvisionCT(p, e) ||
		(!p.replay && !preimagenCompatibleProvisionCT(p, e)) {
		return vacio, errProvisionPerfilesCTObsoleta
	}
	acto := actoAsignacionProvisionCT(huellaManifest, p.clave)
	if !p.replay {
		a := autoridadProvisionPerfilCT(conexiones.provisionador, p, acto)
		if !p.existe {
			a.soloInicial = true
			if err := a.publicarInstantanea(ctx, p.objetivo); err != nil {
				return vacio, errProvisionPerfilesCTObsoleta
			}
		} else if err := a.publicarInstantaneaDesdePreimagen(ctx, p.objetivo, p.preimagen); err != nil {
			return vacio, errProvisionPerfilesCTObsoleta
		}
	}
	post, err := conexiones.almacen.ObtenerInstantaneaAutorizacion(ctx,
		p.semilla.AsignacionPerfil.PrincipalID, p.semilla.AsignacionPerfil.PerfilActivoRef)
	hPost, errHuella := huellaJSONProvisionPerfilesCT(post)
	if err != nil || errHuella != nil || hPost != e.ObjetivoSHA256 ||
		post.AsignacionPerfil.Referencia() != e.ObjetivoAsignacionRef ||
		validarPreimagenProvisionPerfilCT(post, p.semilla, time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return vacio, errProvisionPerfilesCTObsoleta
	}
	if !p.replay {
		actoActual, err := leerActoAsignacionActualProvisionCT(ctx, conexiones.provisionador,
			post.AsignacionPerfil.PerfilActivoRef, post.AsignacionPerfil.Referencia())
		if err != nil || actoActual != acto {
			return vacio, errProvisionPerfilesCTObsoleta
		}
	}
	huellaAsignacion, err := post.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return vacio, errProvisionPerfilesCTObsoleta
	}
	return ReciboPerfilProvisionCT{
		Clave: p.clave, PerfilRef: post.AsignacionPerfil.PerfilActivoRef,
		AsignacionRef: post.AsignacionPerfil.Referencia(), Version: post.AsignacionPerfil.Version,
		HuellaSHA256: huellaAsignacion, Reutilizada: p.replay,
	}, nil
}

func leerActoAsignacionActualProvisionCT(ctx context.Context, pool *pgxpool.Pool,
	perfilRef, asignacionRef string) (string, error) {
	if ctx == nil || pool == nil || perfilRef == "" || asignacionRef == "" {
		return "", errProvisionPerfilesCTNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", errProvisionPerfilesCTNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_propietario`); err != nil {
		return "", errProvisionPerfilesCTNoDisponible
	}
	var acto string
	if err = tx.QueryRow(ctx, `SELECT acto_ref FROM vec_autorizacion.asignacion_perfil_actual
		WHERE perfil_activo_ref=$1 AND asignacion_ref=$2`, perfilRef, asignacionRef).Scan(&acto); err != nil {
		return "", errProvisionPerfilesCTNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return "", errProvisionPerfilesCTNoDisponible
	}
	return acto, nil
}
