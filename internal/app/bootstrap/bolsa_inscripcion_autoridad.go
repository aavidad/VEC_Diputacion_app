package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// DecisorLecturaActualInscripcionBolsa es el seam pendiente de la autoridad
// común: debe revalidar la sesión revocable y la concesión central ACTUAL en
// su fuente PostgreSQL, con acción, recurso, finalidad, perfil y campos exactos. No puede
// implementarse con una instantánea histórica ni con los roles del principal.
type DecisorLecturaActualInscripcionBolsa interface {
	DecidirLecturaActual(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, inscripcion.Filtro) (DecisionLecturaActualInscripcionBolsa, error)
}

// La decisión conserva todos los datos de una única lectura. El constructor
// exige un decisor real; ningún valor por defecto concede acceso.
type DecisionLecturaActualInscripcionBolsa struct {
	Concedida                                                     bool
	PersonaRef, PerfilRef, CuentaRef, SesionRef, AutenticacionRef string
	CertificadoHuellaSHA256, Canal, Accion, RecursoRef, Finalidad string
	CorrelacionRef                                                string
	RevisionPermisos                                              uint64
	HuellaInstantaneaSHA256                                       string
	Campos                                                        []string
	Filtro                                                        inscripcion.Filtro
	EmitidaEn, ValidaHasta                                        time.Time
	ConjuntoGestion                                               *ConjuntoGestionRRHHInscripcionBolsa
	AmbitoSolicitud                                               *AmbitoSolicitudRRHHInscripcionBolsa
}

// DescriptorInscripcionBolsa se aporta desde el catálogo publicado. Su motivo
// se vuelve a resolver en el validador central para cada acto V3.
type DescriptorInscripcionBolsa struct {
	Accion, ModuloID, TipoRecurso, Finalidad string
	Motivo                                   vecdomain.ReferenciaEntradaCatalogo
}

type ClaveOperacionInscripcionBolsa struct{ Accion, Canal string }

const accionListarConvocatoriasGestionRRHHInscripcion = inscripcion.AccionConvocatoriasRRHH

func clavesLecturaInscripcionBolsa() []ClaveOperacionInscripcionBolsa {
	accionesAspirante := []string{inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta,
		inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia}
	claves := make([]ClaveOperacionInscripcionBolsa, 0, 12)
	for _, accion := range accionesAspirante {
		claves = append(claves, ClaveOperacionInscripcionBolsa{accion, "externa_personal"}, ClaveOperacionInscripcionBolsa{accion, "interna_corporativa"})
	}
	for _, accion := range []string{accionListarConvocatoriasGestionRRHHInscripcion, inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH} {
		claves = append(claves, ClaveOperacionInscripcionBolsa{accion, "interna_corporativa"})
	}
	return claves
}

func clavesEscrituraInscripcionBolsa() []ClaveOperacionInscripcionBolsa {
	return []ClaveOperacionInscripcionBolsa{{inscripcion.AccionPresentar, "externa_personal"},
		{inscripcion.AccionPresentar, "interna_corporativa"}, {inscripcion.AccionDecidir, "interna_corporativa"},
		{inscripcion.AccionIncorporar, "interna_corporativa"}}
}

func tipoFinalidadInscripcionBolsa(clave ClaveOperacionInscripcionBolsa) (string, string, bool) {
	empleado := clave.Canal == "interna_corporativa" && !accionRRHHInscripcion(clave.Accion)
	if clave.Canal != "externa_personal" && clave.Canal != "interna_corporativa" {
		return "", "", false
	}
	sufijo := ""
	if empleado {
		sufijo = "_empleado"
	}
	switch clave.Accion {
	case inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta:
		return "convocatoria_inscripcion" + sufijo, "consulta_convocatoria_abierta", true
	case inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia:
		return "solicitud_inscripcion" + sufijo, "consulta_inscripcion_propia", true
	case inscripcion.AccionPresentar:
		return "inscripcion_convocatoria" + sufijo, "presentar_inscripcion", true
	case inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH:
		if clave.Canal == "interna_corporativa" {
			return "solicitud_inscripcion", "consulta_inscripcion_rrhh", true
		}
	case accionListarConvocatoriasGestionRRHHInscripcion:
		if clave.Canal == "interna_corporativa" {
			return "conjunto_gestion_inscripcion", "consulta_convocatorias_gestion_rrhh", true
		}
	case inscripcion.AccionMotivosRRHH:
		if clave.Canal == "interna_corporativa" {
			return "motivos_inscripcion", "consulta_motivos_inscripcion_rrhh", true
		}
	case inscripcion.AccionDecidir:
		if clave.Canal == "interna_corporativa" {
			return "solicitud_inscripcion", "revisar_inscripcion", true
		}
	case inscripcion.AccionIncorporar:
		if clave.Canal == "interna_corporativa" {
			return "solicitud_inscripcion", "incorporar_inscripcion", true
		}
	}
	return "", "", false
}

type DescriptorLecturaInscripcionBolsa struct {
	Accion, Finalidad string
	Campos            []string
}

// La fuente de ámbito pertenece a Bolsa. Resuelve únicamente metadatos
// opacos congelados de la solicitud tras consumir una captura RRHH nominal;
// no recibe ni devuelve datos personales de la inscripción.
type FuenteAmbitoRecursoInscripcionBolsa interface {
	ResolverAmbitoRRHH(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, inscripcion.CapturaLectura) (AmbitoRecursoRRHHInscripcionBolsa, error)
}

type AmbitoRecursoRRHHInscripcionBolsa struct {
	SolicitudRef, UnidadRef, AmbitoRef string
	FuenteRef                          string
	FuenteVersion                      uint64
	FuenteHuellaSHA256                 string
	AuditoriaRef                       string
}

type contextoRecursoEscrituraInscripcionBolsa struct {
	Ambitos         map[string]string
	ConjuntoGestion *inscripcion.AmbitoGestionInscripcion
	AmbitoSolicitud *inscripcion.AmbitoGestionInscripcion
}

// Sólo sirve para cotejar un recurso ya derivado de contexto/propietario.
// La decisión ejecutable corresponde al PDP V3 y su registro CAS.
type FuenteActualEscrituraInscripcionBolsa interface {
	ObtenerInstantaneaEscrituraActual(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa) (vecdomain.InstantaneaAutorizacion, error)
}

type ConfiguracionAutoridadInscripcionBolsa struct {
	Lectura      DecisorLecturaActualInscripcionBolsa
	PDP          *vecapp.ServicioAutorizacionSolicitudLigadaV3
	Material     map[ClaveOperacionInscripcionBolsa]*proveedorMaterialAltaContratacionTemporalDesarrollo
	Motivos      vecports.ValidadorReferenciaMotivoAutorizacionV2
	Reloj        interface{ Ahora() time.Time }
	Descriptores map[ClaveOperacionInscripcionBolsa]DescriptorInscripcionBolsa
	Lecturas     map[ClaveOperacionInscripcionBolsa]DescriptorLecturaInscripcionBolsa
	FuenteActual FuenteActualEscrituraInscripcionBolsa
	AmbitoRRHH   FuenteAmbitoRecursoInscripcionBolsa
	RRHHNominal  []identidadConsultaRRHHDesarrollo
}

type autoridadNominalInscripcionBolsa struct {
	c           ConfiguracionAutoridadInscripcionBolsa
	rrhhNominal []identidadRRHHInscripcionBolsa
}

type identidadRRHHInscripcionBolsa struct{ perfilRef, certificadoSHA256 string }

func copiarIdentidadesRRHHInscripcion(lista []identidadConsultaRRHHDesarrollo) []identidadRRHHInscripcionBolsa {
	resultado := make([]identidadRRHHInscripcionBolsa, 0, len(lista))
	for _, identidad := range lista {
		resultado = append(resultado, identidadRRHHInscripcionBolsa{
			perfilRef: identidad.perfilRef, certificadoSHA256: identidad.identidad.principal.Attributes["certificate_sha256"]})
	}
	return resultado
}

var _ AutoridadInscripcionBolsa = (*autoridadNominalInscripcionBolsa)(nil)

func NuevaAutoridadInscripcionBolsa(c ConfiguracionAutoridadInscripcionBolsa) (AutoridadInscripcionBolsa, error) {
	if nuloInscripcionBolsa(c.Lectura) || nuloInscripcionBolsa(c.FuenteActual) || nuloInscripcionBolsa(c.AmbitoRRHH) ||
		c.PDP == nil || len(c.Material) != 4 ||
		nuloInscripcionBolsa(c.Motivos) || nuloInscripcionBolsa(c.Reloj) ||
		len(c.Descriptores) != 4 || len(c.Lecturas) != 12 || !identidadesRRHHInscripcionValidas(c.RRHHNominal) {
		return nil, inscripcion.ErrNoDisponible
	}
	for _, clave := range clavesEscrituraInscripcionBolsa() {
		d, ok := c.Descriptores[clave]
		tipo, finalidad, esperado := tipoFinalidadInscripcionBolsa(clave)
		if !ok || !esperado || d.Accion != clave.Accion || d.ModuloID != "bolsa" || d.TipoRecurso != tipo || d.Finalidad != finalidad || d.Motivo.Validar() != nil ||
			c.Material[clave] == nil {
			return nil, inscripcion.ErrNoDisponible
		}
	}
	lecturas := make(map[ClaveOperacionInscripcionBolsa]DescriptorLecturaInscripcionBolsa, 12)
	for _, clave := range clavesLecturaInscripcionBolsa() {
		d, ok := c.Lecturas[clave]
		_, finalidad, esperado := tipoFinalidadInscripcionBolsa(clave)
		if !ok || !esperado || d.Accion != clave.Accion || d.Finalidad != finalidad || len(d.Campos) == 0 || !slices.IsSorted(d.Campos) {
			return nil, inscripcion.ErrNoDisponible
		}
		vistos := make(map[string]struct{}, len(d.Campos))
		for _, campo := range d.Campos {
			if campo == "" || campo == "*" {
				return nil, inscripcion.ErrNoDisponible
			}
			if _, duplicado := vistos[campo]; duplicado {
				return nil, inscripcion.ErrNoDisponible
			}
			vistos[campo] = struct{}{}
		}
		ordenados := slices.Clone(d.Campos)
		slices.Sort(ordenados)
		if !slices.Equal(ordenados, camposLecturaInscripcionBolsa(clave.Accion, accionRRHHInscripcion(clave.Accion))) {
			return nil, inscripcion.ErrNoDisponible
		}
		d.Campos = slices.Clone(d.Campos)
		lecturas[clave] = d
	}
	clon := make(map[ClaveOperacionInscripcionBolsa]DescriptorInscripcionBolsa, 4)
	for k, v := range c.Descriptores {
		clon[k] = v
	}
	c.Descriptores = clon
	proveedores := make(map[ClaveOperacionInscripcionBolsa]*proveedorMaterialAltaContratacionTemporalDesarrollo, 4)
	for k, v := range c.Material {
		proveedores[k] = v
	}
	c.Material = proveedores
	c.Lecturas = lecturas
	nominal := copiarIdentidadesRRHHInscripcion(c.RRHHNominal)
	c.RRHHNominal = nil
	return &autoridadNominalInscripcionBolsa{c: c, rrhhNominal: nominal}, nil
}

func (a *autoridadNominalInscripcionBolsa) CapturarLectura(ctx context.Context, s contextoSeguridadComunDesarrollo, acreditacion AcreditacionSesionInscripcionBolsa, accion, recurso string, filtro inscripcion.Filtro) (inscripcion.CapturaLectura, error) {
	vacia := inscripcion.CapturaLectura{}
	if a == nil || ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(a.c.Lectura) || nuloInscripcionBolsa(a.c.Reloj) ||
		!accionLecturaInscripcion(accion) || s.Resultado.Validar() != nil || s.Vinculo.ValidarPara(s.Resultado) != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora := a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !acreditacionInscripcionBolsaActual(s, acreditacion, ahora) {
		return vacia, inscripcion.ErrSesionAusente
	}
	v, err := s.Vinculo.Datos()
	if err != nil {
		return vacia, inscripcion.ErrSesionAusente
	}
	decision, err := a.c.Lectura.DecidirLecturaActual(ctx, s, acreditacion, accion, recurso, filtro)
	if err != nil || ctx.Err() != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora = a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	descriptor, publicado := a.c.Lecturas[ClaveOperacionInscripcionBolsa{accion, acreditacion.Canal}]
	if !decision.Concedida || !acreditacionInscripcionBolsaActual(s, acreditacion, ahora) ||
		!publicado || descriptor.Accion != accion || descriptor.Finalidad == "" || len(descriptor.Campos) == 0 ||
		decision.PersonaRef != s.Resultado.Contexto.PersonaRef || decision.PerfilRef != v.PerfilActivoRef ||
		decision.CuentaRef != v.CuentaRef || decision.SesionRef != v.SesionRef || decision.AutenticacionRef != v.AutenticacionRef ||
		decision.CertificadoHuellaSHA256 != acreditacion.CertificadoHuellaSHA256 ||
		decision.Accion != accion || decision.RecursoRef != recurso || decision.Filtro != filtro ||
		decision.RevisionPermisos == 0 || !revisionHuellaLecturaInscripcionCoincide(decision.RevisionPermisos, decision.HuellaInstantaneaSHA256) ||
		decision.Finalidad != descriptor.Finalidad ||
		!vecdomain.ReferenciaCorrelacionAutorizacionV2Valida(decision.CorrelacionRef) ||
		!slices.IsSorted(decision.Campos) || !slices.Equal(decision.Campos, descriptor.Campos) ||
		decision.EmitidaEn.IsZero() || decision.EmitidaEn.After(ahora) ||
		!decision.ValidaHasta.After(ahora) || decision.ValidaHasta.After(ahora.Add(30*time.Second)) ||
		decision.ValidaHasta.After(v.SesionValidaHasta) ||
		!canalLecturaInscripcion(accion, decision.Canal, v.Superficie) ||
		!ambitoDecisionLecturaInscripcionValido(accion, recurso, decision) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	var conjunto *inscripcion.AmbitoGestionInscripcion
	if c := decision.ConjuntoGestion; c != nil {
		conjunto = &inscripcion.AmbitoGestionInscripcion{ConjuntoRef: c.ConjuntoRef, UnidadRef: c.UnidadRef,
			AmbitoRef: c.AmbitoRef, FuenteRef: c.FuenteRef, FuenteVersion: c.FuenteVersion,
			FuenteSHA256: c.FuenteHuellaSHA256}
		if !conjunto.Valido(true) {
			return vacia, inscripcion.ErrAccesoDenegado
		}
	}
	var historico *inscripcion.AmbitoGestionInscripcion
	if h := decision.AmbitoSolicitud; h != nil {
		historico = &inscripcion.AmbitoGestionInscripcion{UnidadRef: h.UnidadRef, AmbitoRef: h.AmbitoRef,
			FuenteRef: h.FuenteRef, FuenteVersion: h.FuenteVersion, FuenteSHA256: h.FuenteHuellaSHA256}
		if !historico.Valido(false) {
			return vacia, inscripcion.ErrAccesoDenegado
		}
	}
	captura := inscripcion.CapturaLectura{
		PersonaRef: decision.PersonaRef, PerfilRef: decision.PerfilRef, CuentaRef: decision.CuentaRef,
		SesionRef: decision.SesionRef, AutenticacionRef: decision.AutenticacionRef,
		CertificadoHuellaSHA256: decision.CertificadoHuellaSHA256, Canal: decision.Canal,
		Accion: accion, RecursoRef: recurso, Finalidad: decision.Finalidad,
		CorrelacionRef: decision.CorrelacionRef, RevisionPermisos: decision.RevisionPermisos,
		HuellaInstantaneaSHA256: decision.HuellaInstantaneaSHA256,
		Campos:                  slices.Clone(decision.Campos), ConjuntoGestion: conjunto, AmbitoSolicitud: historico,
		Filtro: filtro, EmitidaEn: decision.EmitidaEn, ValidaHasta: decision.ValidaHasta,
	}
	actor := inscripcion.Actor{PersonaRef: decision.PersonaRef, PerfilRef: decision.PerfilRef, SesionRef: decision.SesionRef,
		Canal: decision.Canal, ResultadoContexto: s.Resultado, Vinculo: s.Vinculo, Lectura: &captura}
	if !actor.LecturaValida(accion, recurso, filtro) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	return captura, nil
}

func ambitoDecisionLecturaInscripcionValido(accion, recurso string, d DecisionLecturaActualInscripcionBolsa) bool {
	if !accionRRHHInscripcion(accion) {
		return d.ConjuntoGestion == nil && d.AmbitoSolicitud == nil
	}
	c := d.ConjuntoGestion
	if c == nil || c.ConjuntoRef == "" || c.UnidadRef == "" || c.AmbitoRef == "" || c.FuenteRef == "" ||
		c.FuenteVersion == 0 || !huellaCertificadoInscripcionValida(c.FuenteHuellaSHA256) {
		return false
	}
	if accion != inscripcion.AccionDetalleRRHH {
		return d.AmbitoSolicitud == nil
	}
	h := d.AmbitoSolicitud
	return h != nil && h.SolicitudRef == recurso && h.UnidadRef == c.UnidadRef && h.AmbitoRef == c.AmbitoRef &&
		h.FuenteRef != "" && h.FuenteVersion > 0 && huellaCertificadoInscripcionValida(h.FuenteHuellaSHA256)
}

func revisionHuellaLecturaInscripcionCoincide(revision uint64, huella string) bool {
	if revision == 0 || !huellaCertificadoInscripcionValida(huella) {
		return false
	}
	b, err := hex.DecodeString(huella)
	return err == nil && binary.BigEndian.Uint64(b[:8]) == revision
}

func accionLecturaInscripcion(a string) bool {
	switch a {
	case inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta, inscripcion.AccionListarPropias,
		inscripcion.AccionDetallePropia, inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH,
		accionListarConvocatoriasGestionRRHHInscripcion:
		return true
	}
	return false
}

func accionRRHHInscripcion(a string) bool {
	switch a {
	case accionListarConvocatoriasGestionRRHHInscripcion, inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH,
		inscripcion.AccionDecidir, inscripcion.AccionIncorporar:
		return true
	default:
		return false
	}
}

func canalLecturaInscripcion(a, canal string, superficie vecdomain.SuperficieAutenticacionActorV1) bool {
	if a == accionListarConvocatoriasGestionRRHHInscripcion || a == inscripcion.AccionListarRRHH ||
		a == inscripcion.AccionDetalleRRHH || a == inscripcion.AccionMotivosRRHH {
		return canal == "interna_corporativa" && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1
	}
	return canal == "externa_personal" && superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1 ||
		canal == "interna_corporativa" && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1
}

func vinculoUnicoInscripcion(s contextoSeguridadComunDesarrollo, tipo vecdomain.TipoReferenciaContextoActor, ahora time.Time) (string, bool) {
	referencia := ""
	for _, enlace := range s.Resultado.Contexto.Instantanea.Vinculos {
		if enlace.Tipo != tipo || !enlace.VigenteEn(ahora) {
			continue
		}
		if referencia != "" {
			return "", false
		}
		referencia = enlace.Referencia
	}
	return referencia, referencia != ""
}

func (a *autoridadNominalInscripcionBolsa) rrhhNominalInscripcion(s contextoSeguridadComunDesarrollo, acreditacion AcreditacionSesionInscripcionBolsa) bool {
	if a == nil {
		return false
	}
	return rrhhNominalInscripcionEnLista(s, acreditacion, a.rrhhNominal)
}

func rrhhNominalInscripcionEnLista(s contextoSeguridadComunDesarrollo, acreditacion AcreditacionSesionInscripcionBolsa, lista []identidadRRHHInscripcionBolsa) bool {
	if acreditacion.Canal != "interna_corporativa" {
		return false
	}
	coincidencias := 0
	for _, identidad := range lista {
		if identidad.perfilRef == s.Resultado.Contexto.PerfilActivoRef &&
			identidad.certificadoSHA256 == acreditacion.CertificadoHuellaSHA256 {
			coincidencias++
		}
	}
	return coincidencias == 1
}

func identidadesRRHHInscripcionValidas(lista []identidadConsultaRRHHDesarrollo) bool {
	if len(lista) == 0 {
		return false
	}
	vistos := make(map[string]struct{}, len(lista))
	for _, identidad := range lista {
		huella := identidad.identidad.principal.Attributes["certificate_sha256"]
		if identidad.perfilRef == "" || !huellaCertificadoInscripcionValida(huella) {
			return false
		}
		clave := identidad.perfilRef + "\x00" + huella
		if _, duplicada := vistos[clave]; duplicada {
			return false
		}
		vistos[clave] = struct{}{}
	}
	return true
}

func (a *autoridadNominalInscripcionBolsa) ambitosEscrituraInscripcion(ctx context.Context, s contextoSeguridadComunDesarrollo, acreditacion AcreditacionSesionInscripcionBolsa, accion, recurso string, ahora time.Time) (contextoRecursoEscrituraInscripcionBolsa, error) {
	vacio := contextoRecursoEscrituraInscripcionBolsa{}
	if accion == inscripcion.AccionPresentar {
		v, err := s.Vinculo.Datos()
		if err != nil {
			return vacio, inscripcion.ErrAccesoDenegado
		}
		switch v.Superficie {
		case vecdomain.SuperficieAutenticacionExternaPersonalV1:
			ref, ok := vinculoUnicoInscripcion(s, vecdomain.TipoReferenciaContextoActorCandidato, ahora)
			if ok {
				return contextoRecursoEscrituraInscripcionBolsa{Ambitos: map[string]string{"candidato_ref": ref}}, nil
			}
		case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
			ref, ok := vinculoUnicoInscripcion(s, vecdomain.TipoReferenciaContextoActorEmpleado, ahora)
			if ok && !a.rrhhNominalInscripcion(s, acreditacion) {
				return contextoRecursoEscrituraInscripcionBolsa{Ambitos: map[string]string{"empleado_ref": ref}}, nil
			}
		}
		return vacio, inscripcion.ErrAccesoDenegado
	}
	if (accion != inscripcion.AccionDecidir && accion != inscripcion.AccionIncorporar) ||
		!a.rrhhNominalInscripcion(s, acreditacion) || nuloInscripcionBolsa(a.c.AmbitoRRHH) {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	captura, err := a.CapturarLectura(ctx, s, acreditacion, inscripcion.AccionDetalleRRHH, recurso, inscripcion.Filtro{})
	if err != nil {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	if captura.ConjuntoGestion == nil || !captura.ConjuntoGestion.Valido(true) ||
		captura.AmbitoSolicitud == nil || !captura.AmbitoSolicitud.Valido(false) ||
		captura.ConjuntoGestion.UnidadRef != captura.AmbitoSolicitud.UnidadRef ||
		captura.ConjuntoGestion.AmbitoRef != captura.AmbitoSolicitud.AmbitoRef {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	ambito, err := a.c.AmbitoRRHH.ResolverAmbitoRRHH(ctx, s, acreditacion, recurso, captura)
	if err != nil || ambito.SolicitudRef != recurso || ambito.UnidadRef == "" || ambito.AmbitoRef == "" ||
		ambito.FuenteRef == "" || ambito.FuenteVersion == 0 ||
		!huellaCertificadoInscripcionValida(ambito.FuenteHuellaSHA256) || ambito.AuditoriaRef == "" {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	if ambito.UnidadRef != captura.AmbitoSolicitud.UnidadRef || ambito.AmbitoRef != captura.AmbitoSolicitud.AmbitoRef ||
		ambito.FuenteRef != captura.AmbitoSolicitud.FuenteRef || ambito.FuenteVersion != captura.AmbitoSolicitud.FuenteVersion ||
		ambito.FuenteHuellaSHA256 != captura.AmbitoSolicitud.FuenteSHA256 {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	conjunto, historico := *captura.ConjuntoGestion, *captura.AmbitoSolicitud
	return contextoRecursoEscrituraInscripcionBolsa{Ambitos: map[string]string{"unidad_ref": ambito.UnidadRef, "ambito_ref": ambito.AmbitoRef},
		ConjuntoGestion: &conjunto, AmbitoSolicitud: &historico}, nil
}

func recursoCanonicoDesdeMaterialInscripcion(accion string, material []byte, huella string, contexto contextoRecursoEscrituraInscripcionBolsa) ([]byte, error) {
	switch accion {
	case inscripcion.AccionPresentar:
		if contexto.ConjuntoGestion != nil || contexto.AmbitoSolicitud != nil {
			return nil, inscripcion.ErrAccesoDenegado
		}
		var v struct {
			ConvocatoriaRef   string                    `json:"convocatoria_ref"`
			CategoriaRef      string                    `json:"categoria_ref"`
			CatalogoVersion   uint64                    `json:"catalogo_version"`
			ClaveIdempotencia string                    `json:"clave_idempotencia"`
			Declaraciones     []inscripcion.Declaracion `json:"declaraciones"`
		}
		if json.Unmarshal(material, &v) != nil {
			return nil, inscripcion.ErrSolicitudInvalida
		}
		return inscripcion.RecursoPresentacion(inscripcion.Presentacion{ConvocatoriaRef: v.ConvocatoriaRef, CategoriaRef: v.CategoriaRef,
			CatalogoVersion: v.CatalogoVersion, ClaveIdempotencia: v.ClaveIdempotencia, Declaraciones: v.Declaraciones}, huella, contexto.Ambitos)
	case inscripcion.AccionDecidir:
		if contexto.ConjuntoGestion == nil || contexto.AmbitoSolicitud == nil {
			return nil, inscripcion.ErrAccesoDenegado
		}
		var v struct {
			SolicitudRef      string `json:"solicitud_ref"`
			Decision          string `json:"decision"`
			MotivoCodigo      string `json:"motivo_codigo"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if json.Unmarshal(material, &v) != nil {
			return nil, inscripcion.ErrSolicitudInvalida
		}
		return inscripcion.RecursoDecision(inscripcion.Decision{SolicitudRef: v.SolicitudRef, Tipo: v.Decision,
			MotivoCodigo: v.MotivoCodigo, VersionEsperada: v.VersionEsperada, ClaveIdempotencia: v.ClaveIdempotencia}, huella,
			contexto.Ambitos, *contexto.ConjuntoGestion, *contexto.AmbitoSolicitud)
	case inscripcion.AccionIncorporar:
		if contexto.ConjuntoGestion == nil || contexto.AmbitoSolicitud == nil {
			return nil, inscripcion.ErrAccesoDenegado
		}
		var v struct {
			SolicitudRef      string `json:"solicitud_ref"`
			EvidenciaRef      string `json:"evidencia_ref"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if json.Unmarshal(material, &v) != nil {
			return nil, inscripcion.ErrSolicitudInvalida
		}
		return inscripcion.RecursoIncorporacion(inscripcion.Incorporacion{SolicitudRef: v.SolicitudRef,
			EvidenciaRef: v.EvidenciaRef, VersionEsperada: v.VersionEsperada, ClaveIdempotencia: v.ClaveIdempotencia}, huella,
			contexto.Ambitos, *contexto.ConjuntoGestion, *contexto.AmbitoSolicitud)
	default:
		return nil, inscripcion.ErrAccesoDenegado
	}
}

func (a *autoridadNominalInscripcionBolsa) AutorizarEscritura(ctx context.Context, s contextoSeguridadComunDesarrollo, acreditacion AcreditacionSesionInscripcionBolsa, accion, recurso string, material []byte) (AutorizacionEscrituraInscripcionBolsa, error) {
	vacia := AutorizacionEscrituraInscripcionBolsa{}
	if a == nil || ctx == nil || ctx.Err() != nil || a.c.PDP == nil || nuloInscripcionBolsa(a.c.FuenteActual) ||
		nuloInscripcionBolsa(a.c.Motivos) || nuloInscripcionBolsa(a.c.Reloj) ||
		s.Resultado.Validar() != nil || s.Vinculo.ValidarPara(s.Resultado) != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	clave := ClaveOperacionInscripcionBolsa{accion, acreditacion.Canal}
	d, ok := a.c.Descriptores[clave]
	proveedor := a.c.Material[clave]
	if !ok || proveedor == nil || d.Accion != accion || d.Motivo.Validar() != nil || d.ModuloID == "" || d.TipoRecurso == "" || d.Finalidad == "" {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora := a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !acreditacionInscripcionBolsaActual(s, acreditacion, ahora) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	atributos, valido := atributosMaterialEscrituraInscripcion(accion, s.Resultado.Contexto.PersonaRef, recurso, material)
	if !valido {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	v, err := s.Vinculo.Datos()
	if err != nil || (accion != inscripcion.AccionPresentar && v.Superficie != vecdomain.SuperficieAutenticacionInternaCorporativaV1) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	if err := a.c.Motivos.ValidarReferenciaMotivoAutorizacionV2(ctx, d.Motivo, ahora); err != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	contextoRecurso, err := a.ambitosEscrituraInscripcion(ctx, s, acreditacion, accion, recurso, ahora)
	if err != nil || !acreditacionInscripcionBolsaActual(s, acreditacion, a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	canon, err := recursoCanonicoDesdeMaterialInscripcion(accion, material, atributos["material_sha256"], contextoRecurso)
	if accion == inscripcion.AccionDecidir || accion == inscripcion.AccionIncorporar {
		if contextoRecurso.ConjuntoGestion == nil || contextoRecurso.AmbitoSolicitud == nil {
			return vacia, inscripcion.ErrAccesoDenegado
		}
		conjunto, historico := contextoRecurso.ConjuntoGestion, contextoRecurso.AmbitoSolicitud
		atributos["conjunto_ref"] = conjunto.ConjuntoRef
		atributos["conjunto_fuente_ref"] = conjunto.FuenteRef
		atributos["conjunto_fuente_version"] = strconv.FormatUint(conjunto.FuenteVersion, 10)
		atributos["conjunto_fuente_sha256"] = conjunto.FuenteSHA256
		atributos["solicitud_fuente_ref"] = historico.FuenteRef
		atributos["solicitud_fuente_version"] = strconv.FormatUint(historico.FuenteVersion, 10)
		atributos["solicitud_fuente_sha256"] = historico.FuenteSHA256
	}
	var rc struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}
	if err != nil || len(canon) == 0 || len(canon) > 8*1024 || json.Unmarshal(canon, &rc) != nil ||
		!maps.Equal(rc.Ambitos, contextoRecurso.Ambitos) || !maps.Equal(rc.Atributos, atributos) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	recursoV3 := vecdomain.RecursoAutorizable{Referencia: recurso, ModuloID: d.ModuloID, Tipo: d.TipoRecurso,
		Ambitos: rc.Ambitos, Atributos: rc.Atributos}
	if recursoV3.Validar() != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	instantanea, err := a.c.FuenteActual.ObtenerInstantaneaEscrituraActual(ctx, s, acreditacion)
	if err != nil || !instantanea.AsignacionPerfil.Cubre(recursoV3) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, inscripcion.ErrNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.Vinculo, ReferenciaMotivo: d.Motivo, Accion: accion,
		Recurso:   recursoV3,
		Finalidad: d.Finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	decision, confirmacion, err := a.c.PDP.ExigirSolicitudLigadaV3(ctx, solicitud, s.Resultado)
	concedida, _, errResultado := decision.Resultado()
	if err != nil || errResultado != nil || !concedida || decision.ValidarPara(solicitud) != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	materialV3, err := proveedor.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, d.Motivo, s.Resultado)
	if err != nil || materialV3.ValidarEstructura() != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	resumen := materialV3.ResumenCapacidad()
	hr := sha256.Sum256(canon)
	h := sha256.Sum256(material)
	if resumen.Operacion() != accion || resumen.AudienciaConsumo() != audienciaEscrituraInscripcion(clave) ||
		resumen.EfectoRef() != recurso || resumen.EfectoHuellaSHA256() != hex.EncodeToString(hr[:]) ||
		resumen.ContextoRef() != s.Resultado.RegistroContextoRef || resumen.ContextoHuellaSHA256() != s.Resultado.HuellaSHA256 ||
		!bytes.Equal(materialV3.ContextoActorCanonico(), s.Resultado.RepresentacionCanonica) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	return AutorizacionEscrituraInscripcionBolsa{Accion: accion, Recurso: recurso, MaterialSHA256: hex.EncodeToString(h[:]), Material: materialV3,
		RecursoCanonico: bytes.Clone(canon)}, nil
}

func audienciaEscrituraInscripcion(clave ClaveOperacionInscripcionBolsa) string {
	switch clave.Accion {
	case inscripcion.AccionPresentar:
		if clave.Canal == "externa_personal" {
			return "vec_bolsa_llamamientos.inscripcion.presentar.v1"
		}
		if clave.Canal == "interna_corporativa" {
			return "vec_bolsa_llamamientos.inscripcion.presentar_empleado.v1"
		}
	case inscripcion.AccionDecidir:
		if clave.Canal == "interna_corporativa" {
			return "vec_bolsa_llamamientos.inscripcion.revisar.v1"
		}
	case inscripcion.AccionIncorporar:
		if clave.Canal == "interna_corporativa" {
			return "vec_bolsa_llamamientos.inscripcion.incorporar.v1"
		}
	}
	return ""
}

func atributosMaterialEscrituraInscripcion(accion, persona, recurso string, material []byte) (map[string]string, bool) {
	if len(material) == 0 || len(material) > 64*1024 {
		return nil, false
	}
	var esperado []byte
	var huella, ref string
	var atributos map[string]string
	var err error
	switch accion {
	case inscripcion.AccionPresentar:
		var v struct {
			Esquema           string                    `json:"esquema"`
			ConvocatoriaRef   string                    `json:"convocatoria_ref"`
			CategoriaRef      string                    `json:"categoria_ref"`
			CatalogoVersion   uint64                    `json:"catalogo_version"`
			ClaveIdempotencia string                    `json:"clave_idempotencia"`
			Declaraciones     []inscripcion.Declaracion `json:"declaraciones"`
		}
		if json.Unmarshal(material, &v) != nil || v.Esquema != inscripcion.EsquemaMaterialPresentacion {
			return nil, false
		}
		p := inscripcion.Presentacion{ConvocatoriaRef: v.ConvocatoriaRef, CategoriaRef: v.CategoriaRef, CatalogoVersion: v.CatalogoVersion, ClaveIdempotencia: v.ClaveIdempotencia, Declaraciones: v.Declaraciones}
		esperado, huella, err = inscripcion.MaterialPresentacion(p)
		if err == nil {
			ref, err = inscripcion.ReferenciaSolicitud(persona, p)
		}
		atributos = map[string]string{"convocatoria_ref": p.ConvocatoriaRef, "categoria_ref": p.CategoriaRef,
			"catalogo_version": strconv.FormatUint(p.CatalogoVersion, 10)}
	case inscripcion.AccionDecidir:
		var v struct {
			Esquema           string `json:"esquema"`
			SolicitudRef      string `json:"solicitud_ref"`
			Decision          string `json:"decision"`
			MotivoCodigo      string `json:"motivo_codigo"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if json.Unmarshal(material, &v) != nil || v.Esquema != inscripcion.EsquemaMaterialDecision {
			return nil, false
		}
		d := inscripcion.Decision{SolicitudRef: v.SolicitudRef, Tipo: v.Decision, MotivoCodigo: v.MotivoCodigo, VersionEsperada: v.VersionEsperada, ClaveIdempotencia: v.ClaveIdempotencia}
		esperado, huella, err = inscripcion.MaterialDecision(d)
		ref = d.SolicitudRef
		atributos = map[string]string{"solicitud_ref": d.SolicitudRef, "decision": d.Tipo,
			"version_esperada": strconv.FormatUint(d.VersionEsperada, 10)}
		if d.MotivoCodigo != "" {
			atributos["motivo_codigo"] = d.MotivoCodigo
		}
	case inscripcion.AccionIncorporar:
		var v struct {
			Esquema           string `json:"esquema"`
			SolicitudRef      string `json:"solicitud_ref"`
			EvidenciaRef      string `json:"evidencia_ref"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if json.Unmarshal(material, &v) != nil || v.Esquema != inscripcion.EsquemaMaterialIncorporacion {
			return nil, false
		}
		i := inscripcion.Incorporacion{SolicitudRef: v.SolicitudRef, EvidenciaRef: v.EvidenciaRef, VersionEsperada: v.VersionEsperada, ClaveIdempotencia: v.ClaveIdempotencia}
		esperado, huella, err = inscripcion.MaterialIncorporacion(i)
		ref = i.SolicitudRef
		atributos = map[string]string{"solicitud_ref": i.SolicitudRef, "evidencia_ref": i.EvidenciaRef,
			"version_esperada": strconv.FormatUint(i.VersionEsperada, 10)}
	default:
		return nil, false
	}
	if err != nil || ref != recurso || !bytes.Equal(esperado, material) {
		return nil, false
	}
	atributos["material_sha256"] = huella
	return atributos, true
}
