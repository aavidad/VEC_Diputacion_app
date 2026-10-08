package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// AutoridadInscripcionBolsa se compone con la sesión nominal, la concesión
// exacta y el emisor V3 existentes. Ninguna de sus respuestas procede de HTTP.
// CapturarLectura debe comprobar sesión revocable y permiso central vigente;
// la captura sola no concede permiso. AutorizarEscritura debe decidir, registrar
// y atestar V3 para la operación y el recurso recibidos.
type AutoridadInscripcionBolsa interface {
	CapturarLectura(context.Context, contextoSeguridadComunDesarrollo, string, string, inscripcion.Filtro) (inscripcion.CapturaLectura, error)
	AutorizarEscritura(context.Context, contextoSeguridadComunDesarrollo, string, string, []byte, []byte) (AutorizacionEscrituraInscripcionBolsa, error)
}

type AutorizacionEscrituraInscripcionBolsa struct {
	Accion         string
	Recurso        string
	MaterialSHA256 string
	Material       vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// La implementación común revalida certificado y sesión revocable por petición
// y devuelve la acreditación ligada al mismo contexto_actor.
type SesionInscripcionBolsa interface {
	ResolverInscripcion(*http.Request) (contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, error)
}

type AcreditacionSesionInscripcionBolsa struct {
	CertificadoHuellaSHA256 string
	Canal                   string
	PersonaRef              string
	PerfilRef               string
	CuentaRef               string
	SesionRef               string
	AutenticacionRef        string
	VerificadaEn            time.Time
	ValidaHasta             time.Time
}

// ConfiguracionPreparadorInscripcionBolsa sólo admite identidades instaladas
// de forma explícita. La raíz conecta la autoridad común de sesión y PDP.
type ConfiguracionPreparadorInscripcionBolsa struct {
	Aspirante       *identidadCandidatoBolsaDesarrollo
	RRHH            []identidadConsultaRRHHDesarrollo
	SesionAspirante SesionInscripcionBolsa
	SesionRRHH      SesionInscripcionBolsa
	Autoridad       AutoridadInscripcionBolsa
	Reloj           relojContratacionTemporalDesarrollo
}

type preparadorInscripcionBolsa struct {
	configuracion ConfiguracionPreparadorInscripcionBolsa
}

var _ httpinscripcion.Preparador = (*preparadorInscripcionBolsa)(nil)

func NuevoPreparadorInscripcionBolsa(c ConfiguracionPreparadorInscripcionBolsa) (httpinscripcion.Preparador, error) {
	if nuloInscripcionBolsa(c.SesionAspirante) || nuloInscripcionBolsa(c.SesionRRHH) || nuloInscripcionBolsa(c.Autoridad) ||
		nuloInscripcionBolsa(c.Reloj) || c.Aspirante == nil || c.Aspirante.personaRef == "" ||
		c.Aspirante.perfilRef == "" || c.Aspirante.cuentaRef == "" ||
		len(c.RRHH) == 0 {
		return nil, inscripcion.ErrNoDisponible
	}
	return &preparadorInscripcionBolsa{configuracion: c}, nil
}

func nuloInscripcionBolsa(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

type lecturaInscripcion struct {
	accion, ref, idioma string
	filtro              inscripcion.Filtro
}

func (p *preparadorInscripcionBolsa) PrepararLecturaAspirante(r *http.Request, accion, ref string, filtro inscripcion.Filtro, idioma string) (inscripcion.Actor, error) {
	return p.preparar(r, false, &lecturaInscripcion{accion, ref, idioma, filtro}, nil)
}

func (p *preparadorInscripcionBolsa) PrepararLecturaRRHH(r *http.Request, accion, ref string, filtro inscripcion.Filtro, idioma string) (inscripcion.Actor, error) {
	return p.preparar(r, true, &lecturaInscripcion{accion, ref, idioma, filtro}, nil)
}

type operacionEscrituraInscripcion struct {
	ref          string
	material     []byte
	recurso      []byte
	presentacion *inscripcion.Presentacion
}

func (p *preparadorInscripcionBolsa) PrepararPresentacion(r *http.Request, comando inscripcion.Presentacion) (inscripcion.Actor, error) {
	material, huella, err := inscripcion.MaterialPresentacion(comando)
	if err != nil {
		return inscripcion.Actor{}, err
	}
	recurso, err := inscripcion.RecursoPresentacion(comando, huella)
	if err != nil {
		return inscripcion.Actor{}, err
	}
	// La solicitud propia queda ligada a la persona después de resolver sesión.
	return p.preparar(r, false, nil, &operacionEscrituraInscripcion{material: material, recurso: recurso, presentacion: &comando})
}

func (p *preparadorInscripcionBolsa) PrepararDecision(r *http.Request, d inscripcion.Decision) (inscripcion.Actor, error) {
	material, huella, err := inscripcion.MaterialDecision(d)
	if err != nil {
		return inscripcion.Actor{}, err
	}
	recurso, err := inscripcion.RecursoDecision(d, huella)
	if err != nil {
		return inscripcion.Actor{}, err
	}
	return p.preparar(r, true, nil, &operacionEscrituraInscripcion{ref: d.SolicitudRef, material: material, recurso: recurso})
}

func (p *preparadorInscripcionBolsa) PrepararIncorporacion(r *http.Request, i inscripcion.Incorporacion) (inscripcion.Actor, error) {
	material, huella, err := inscripcion.MaterialIncorporacion(i)
	if err != nil {
		return inscripcion.Actor{}, err
	}
	recurso, err := inscripcion.RecursoIncorporacion(i, huella)
	if err != nil {
		return inscripcion.Actor{}, err
	}
	return p.preparar(r, true, nil, &operacionEscrituraInscripcion{ref: i.SolicitudRef, material: material, recurso: recurso})
}

func (p *preparadorInscripcionBolsa) preparar(r *http.Request, rrhh bool, lectura *lecturaInscripcion, escritura *operacionEscrituraInscripcion) (inscripcion.Actor, error) {
	var vacio inscripcion.Actor
	if p == nil || r == nil || r.URL == nil || r.Context().Err() != nil ||
		nuloInscripcionBolsa(p.configuracion.SesionAspirante) || nuloInscripcionBolsa(p.configuracion.SesionRRHH) ||
		nuloInscripcionBolsa(p.configuracion.Autoridad) || nuloInscripcionBolsa(p.configuracion.Reloj) {
		return vacio, inscripcion.ErrNoDisponible
	}
	c := p.configuracion
	accion, recurso, filtro, valido := operacionInscripcionBolsa(r, rrhh)
	if (r.Method == http.MethodGet && (lectura == nil || escritura != nil)) ||
		(r.Method == http.MethodPost && (escritura == nil || lectura != nil)) {
		return vacio, inscripcion.ErrSolicitudInvalida
	}
	if !valido {
		return vacio, inscripcion.ErrSolicitudInvalida
	}
	if lectura != nil {
		refEsperada := recurso
		switch accion {
		case inscripcion.AccionListarAbiertas, inscripcion.AccionListarPropias, inscripcion.AccionListarRRHH:
			refEsperada = ""
		case inscripcion.AccionMotivosRRHH:
			refEsperada = strings.TrimPrefix(recurso, "motivos:")
		}
		idioma := r.URL.Query().Get("idioma")
		if idioma == "" {
			idioma = "es"
		}
		if lectura.accion != accion || lectura.ref != refEsperada || lectura.filtro != filtro || lectura.idioma != idioma {
			return vacio, inscripcion.ErrSolicitudInvalida
		}
	}
	ahora := c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	sesion := c.SesionAspirante
	canalEsperado := "externa_personal"
	if rrhh {
		sesion = c.SesionRRHH
		canalEsperado = "interna_corporativa"
	}
	ctx, acreditacion, err := sesion.ResolverInscripcion(r)
	if err != nil || !huellaCertificadoInscripcionValida(acreditacion.CertificadoHuellaSHA256) ||
		acreditacion.Canal != canalEsperado || acreditacion.VerificadaEn.IsZero() ||
		acreditacion.VerificadaEn.After(ahora) || !acreditacion.ValidaHasta.After(ahora) {
		return vacio, inscripcion.ErrSesionAusente
	}
	var personaEsperada, perfilEsperado, cuentaEsperada string
	if rrhh {
		coincidencias := 0
		for _, identidad := range c.RRHH {
			if acreditacion.CertificadoHuellaSHA256 == identidad.identidad.principal.Attributes["certificate_sha256"] {
				coincidencias++
				perfilEsperado = identidad.perfilRef
			}
		}
		if coincidencias != 1 || perfilEsperado == "" {
			return vacio, inscripcion.ErrAccesoDenegado
		}
	} else {
		id := c.Aspirante
		if id == nil || acreditacion.CertificadoHuellaSHA256 != id.identidad.principal.Attributes["certificate_sha256"] ||
			id.verificadoEn.After(ahora) || !ahora.Before(id.validoHasta) {
			return vacio, inscripcion.ErrSesionAusente
		}
		personaEsperada, perfilEsperado, cuentaEsperada = id.personaRef, id.perfilRef, id.cuentaRef
	}
	ahora = c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !ahora.Before(acreditacion.ValidaHasta) || (!rrhh && !ahora.Before(c.Aspirante.validoHasta)) ||
		!contextoInscripcionBolsaValido(ctx, personaEsperada, perfilEsperado, cuentaEsperada, rrhh, ahora) {
		return vacio, inscripcion.ErrSesionAusente
	}
	vinculo, err := ctx.Vinculo.Datos()
	if err != nil || acreditacion.PersonaRef != ctx.Resultado.Contexto.PersonaRef ||
		acreditacion.PerfilRef != perfilEsperado || acreditacion.CuentaRef != vinculo.CuentaRef ||
		acreditacion.SesionRef != vinculo.SesionRef || acreditacion.AutenticacionRef != vinculo.AutenticacionRef ||
		acreditacion.ValidaHasta.After(vinculo.SesionValidaHasta) {
		return vacio, inscripcion.ErrSesionAusente
	}
	actor := inscripcion.Actor{PersonaRef: ctx.Resultado.Contexto.PersonaRef, PerfilRef: perfilEsperado,
		SesionRef: vinculo.SesionRef, Canal: canalEsperado, ResultadoContexto: ctx.Resultado, Vinculo: ctx.Vinculo}
	if lectura != nil {
		actor.Idioma = lectura.idioma
	}
	if !actor.Valido() {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	if lectura != nil {
		recurso, err = inscripcion.RecursoLectura(accion, actor.PersonaRef, lectura.idioma, filtro, lectura.ref)
		if err != nil {
			return vacio, inscripcion.ErrSolicitudInvalida
		}
	}
	if accion == inscripcion.AccionPresentar {
		if escritura == nil || escritura.presentacion == nil {
			return vacio, inscripcion.ErrSolicitudInvalida
		}
		recurso, err = inscripcion.ReferenciaSolicitud(actor.PersonaRef, *escritura.presentacion)
		if err != nil {
			return vacio, inscripcion.ErrSolicitudInvalida
		}
	}
	if r.Method == http.MethodGet {
		captura, err := c.Autoridad.CapturarLectura(r.Context(), ctx, accion, recurso, filtro)
		ahoraLectura := c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
		if err != nil || !ahoraLectura.Before(acreditacion.ValidaHasta) ||
			!ctx.Vinculo.VigenteEn(ahoraLectura, ctx.Resultado) ||
			captura.PersonaRef != actor.PersonaRef || captura.PerfilRef != actor.PerfilRef ||
			captura.CuentaRef != vinculo.CuentaRef || captura.SesionRef != actor.SesionRef ||
			captura.AutenticacionRef != vinculo.AutenticacionRef ||
			captura.CertificadoHuellaSHA256 != acreditacion.CertificadoHuellaSHA256 ||
			captura.Accion != accion || captura.RecursoRef != recurso || captura.Filtro != filtro ||
			captura.RevisionPermisos == 0 || captura.Canal != canalEsperado || captura.Finalidad == "" ||
			captura.CorrelacionRef == "" || captura.EmitidaEn.After(ahoraLectura) ||
			!captura.ValidaHasta.After(ahoraLectura) || captura.ValidaHasta.After(vinculo.SesionValidaHasta) ||
			!actorConLecturaValidaInscripcion(actor, &captura, accion, recurso, filtro) {
			return vacio, inscripcion.ErrAccesoDenegado
		}
		actor.Lectura = &captura
		return actor, nil
	}
	if escritura == nil || len(escritura.material) == 0 || len(escritura.recurso) == 0 {
		return vacio, inscripcion.ErrNoDisponible
	}
	if escritura.ref != "" && escritura.ref != recurso {
		return vacio, inscripcion.ErrSolicitudInvalida
	}
	hash := sha256.Sum256(escritura.material)
	concesion, err := c.Autoridad.AutorizarEscritura(r.Context(), ctx, accion, recurso, escritura.material, escritura.recurso)
	ahoraEscritura := c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	resumen := concesion.Material.ResumenCapacidad()
	if err != nil || !ahoraEscritura.Before(acreditacion.ValidaHasta) ||
		!ctx.Vinculo.VigenteEn(ahoraEscritura, ctx.Resultado) ||
		concesion.Accion != accion || concesion.Recurso != recurso || concesion.MaterialSHA256 != hex.EncodeToString(hash[:]) ||
		concesion.Material.ValidarEstructura() != nil || resumen.ContextoRef() != ctx.Resultado.RegistroContextoRef ||
		resumen.ContextoHuellaSHA256() != ctx.Resultado.HuellaSHA256 || resumen.Operacion() != accion ||
		resumen.EfectoRef() != recurso || ahoraEscritura.Before(resumen.EmitidaEn()) || !ahoraEscritura.Before(resumen.ExpiraEn()) ||
		!bytes.Equal(concesion.Material.ContextoActorCanonico(), ctx.Resultado.RepresentacionCanonica) ||
		concesion.Material.PersonaVersion() != ctx.Resultado.Contexto.Instantanea.PersonaVersion ||
		concesion.Material.PerfilVersion() != ctx.Resultado.Contexto.Instantanea.PerfilVersion {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	actor.MaterialEscritura = &concesion.Material
	return actor, nil
}

func contextoInscripcionBolsaValido(ctx contextoSeguridadComunDesarrollo, persona, perfil, cuenta string, rrhh bool, ahora time.Time) bool {
	if ctx.Resultado.Validar() != nil || ctx.Vinculo.ValidarPara(ctx.Resultado) != nil ||
		!ctx.Vinculo.VigenteEn(ahora, ctx.Resultado) ||
		ctx.Resultado.Contexto.PerfilActivoRef != perfil ||
		(persona != "" && ctx.Resultado.Contexto.PersonaRef != persona) ||
		(cuenta != "" && ctx.Resultado.Contexto.Instantanea.CuentaRef != cuenta) {
		return false
	}
	v, err := ctx.Vinculo.Datos()
	return err == nil && v.PrincipalID == ctx.Resultado.Contexto.PersonaRef && v.PerfilActivoRef == perfil &&
		v.MetodoObservado == vecdomain.AuthMethodCertificate && v.GarantiaObservada == vecdomain.AuthAssuranceHigh &&
		((rrhh && v.Superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1) ||
			(!rrhh && v.Superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1))
}

func actorConLecturaValidaInscripcion(a inscripcion.Actor, c *inscripcion.CapturaLectura, accion, recurso string, filtro inscripcion.Filtro) bool {
	a.Lectura = c
	return a.LecturaValida(accion, recurso, filtro)
}

// La captura se pide con el mismo filtro normalizado que entregará el handler.
// Una ruta, método o query distintos no obtienen autoridad.
func operacionInscripcionBolsa(r *http.Request, rrhh bool) (string, string, inscripcion.Filtro, bool) {
	var cero inscripcion.Filtro
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path || r.URL.ForceQuery {
		return "", "", cero, false
	}
	q := r.URL.Query()
	for _, valores := range q {
		if len(valores) != 1 {
			return "", "", cero, false
		}
	}
	if idioma := q.Get("idioma"); idioma != "" && idioma != "es" && idioma != "en" {
		return "", "", cero, false
	}
	path := r.URL.Path
	if rrhh {
		if path == httpinscripcion.RutaRRHH && r.Method == http.MethodGet && soloQueryInscripcion(q, "estado", "convocatoria_ref", "limite", "cursor", "idioma") {
			f, ok := filtroListaInscripcion(q)
			if f.Estado = q.Get("estado"); f.Estado == "" {
				f.Estado = inscripcion.EstadoPendiente
			}
			f.ConvocatoriaRef = q.Get("convocatoria_ref")
			return inscripcion.AccionListarRRHH, "inscripciones:rrhh", f, ok && f.Validar() == nil
		}
		if path == httpinscripcion.RutaRRHH+"/motivos" && r.Method == http.MethodGet && soloQueryInscripcion(q, "decision", "idioma") && len(q["decision"]) == 1 && (q.Get("decision") == "admitir" || q.Get("decision") == "rechazar") {
			return inscripcion.AccionMotivosRRHH, "motivos:" + q.Get("decision"), cero, true
		}
		if ref, resto, ok := segmentoInscripcion(path, httpinscripcion.RutaRRHH); ok {
			if resto == "" && r.Method == http.MethodGet && soloQueryInscripcion(q, "idioma") {
				return inscripcion.AccionDetalleRRHH, ref, cero, true
			}
			if r.Method == http.MethodPost && len(q) == 0 {
				if resto == "decisiones" {
					return inscripcion.AccionDecidir, ref, cero, true
				}
				if resto == "incorporaciones" {
					return inscripcion.AccionIncorporar, ref, cero, true
				}
			}
		}
		return "", "", cero, false
	}
	if path == httpinscripcion.RutaAbiertas && r.Method == http.MethodGet && soloQueryInscripcion(q, "limite", "cursor", "idioma") {
		f, ok := filtroListaInscripcion(q)
		return inscripcion.AccionListarAbiertas, "convocatorias-abiertas", f, ok
	}
	if ref, resto, ok := segmentoInscripcion(path, httpinscripcion.RutaAbiertas); ok && resto == "" && r.Method == http.MethodGet && soloQueryInscripcion(q, "idioma") {
		return inscripcion.AccionDetalleAbierta, ref, cero, true
	}
	if path == httpinscripcion.RutaPropias {
		if r.Method == http.MethodPost && len(q) == 0 {
			return inscripcion.AccionPresentar, "inscripciones:propias", cero, true
		}
		if r.Method == http.MethodGet && soloQueryInscripcion(q, "limite", "cursor", "idioma") {
			f, ok := filtroListaInscripcion(q)
			return inscripcion.AccionListarPropias, "inscripciones:propias:", f, ok
		}
	}
	if ref, resto, ok := segmentoInscripcion(path, httpinscripcion.RutaPropias); ok && resto == "" && r.Method == http.MethodGet && soloQueryInscripcion(q, "idioma") {
		return inscripcion.AccionDetallePropia, ref, cero, true
	}
	return "", "", cero, false
}

func soloQueryInscripcion(q url.Values, claves ...string) bool {
	for clave := range q {
		admitida := false
		for _, conocida := range claves {
			if clave == conocida {
				admitida = true
				break
			}
		}
		if !admitida {
			return false
		}
	}
	return true
}

func filtroListaInscripcion(q url.Values) (inscripcion.Filtro, bool) {
	limite := 20
	if valor := q.Get("limite"); valor != "" {
		var err error
		limite, err = strconv.Atoi(valor)
		if err != nil || limite < 1 || limite > 100 {
			return inscripcion.Filtro{}, false
		}
	}
	f := inscripcion.Filtro{Limite: limite, Cursor: q.Get("cursor")}
	return f, len(f.Cursor) <= 512
}

func segmentoInscripcion(path, base string) (string, string, bool) {
	if !strings.HasPrefix(path, base+"/") {
		return "", "", false
	}
	partes := strings.Split(strings.TrimPrefix(path, base+"/"), "/")
	if len(partes) < 1 || len(partes) > 2 || !segmentoInscripcionValido(partes[0]) {
		return "", "", false
	}
	if len(partes) == 2 {
		return partes[0], partes[1], true
	}
	return partes[0], "", true
}

func segmentoInscripcionValido(v string) bool {
	if len(v) < 3 || len(v) > 256 {
		return false
	}
	for _, b := range []byte(v) {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.ContainsRune(":._-", rune(b)) {
			continue
		}
		return false
	}
	return true
}

// Evita normalizar una huella del navegador: la huella viene sólo de la
// capacidad mTLS del servidor, y su codificación debe ser canónica.
func huellaCertificadoInscripcionValida(v string) bool {
	b, err := hex.DecodeString(v)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == v
}
