package httpinterno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const RutaSolicitudesRectificacionDietas = personalports.RutaSolicitudesRectificacionDietas

type CasoUsoRectificacionDietas interface {
	Ejecutar(context.Context, personaldomain.SolicitudRectificacionDietas) (personalports.ResultadoRectificacionDietas, error)
}

type CasoUsoRectificacionesCompetentesDietas interface {
	Consultar(context.Context, personaldomain.SolicitudRectificacionesCompetentesDietas) (personalports.ResultadoConsultaRectificacionesCompetentesDietas, error)
}

type ManejadorRectificacionDietas struct {
	identidades ResolutorIdentidadAsignacionDietas
	casoUso     CasoUsoRectificacionDietas
	competentes CasoUsoRectificacionesCompetentesDietas
	auditoria   personalports.RegistradorAuditoriaFronteraRectificacionDietas
}

func NuevoManejadorRectificacionDietas(i ResolutorIdentidadAsignacionDietas, c CasoUsoRectificacionDietas, competentes CasoUsoRectificacionesCompetentesDietas, a personalports.RegistradorAuditoriaFronteraRectificacionDietas) (*ManejadorRectificacionDietas, error) {
	if nuloAsignacionHTTP(i) || nuloAsignacionHTTP(c) || nuloAsignacionHTTP(competentes) || nuloAsignacionHTTP(a) {
		return nil, personalports.ErrRectificacionDietasNoDisponible
	}
	return &ManejadorRectificacionDietas{identidades: i, casoUso: c, competentes: competentes, auditoria: a}, nil
}

func (m *ManejadorRectificacionDietas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if r == nil || m == nil || nuloAsignacionHTTP(m.auditoria) {
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	respuesta := &respuestaRetenidaAsignacion{cabeceras: make(http.Header)}
	accion, actorRef := m.servir(respuesta, r)
	if respuesta.estado >= 400 && m.auditar(r, respuesta.estado, accion, actorRef) != nil {
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	for k, vs := range respuesta.cabeceras {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(respuesta.estado)
	_, _ = w.Write(respuesta.cuerpo.Bytes())
}

func (m *ManejadorRectificacionDietas) servir(w http.ResponseWriter, r *http.Request) (accion, actorRef string) {
	accion = "metodo_no_admitido"
	if nuloAsignacionHTTP(m.identidades) || nuloAsignacionHTTP(m.casoUso) || nuloAsignacionHTTP(m.competentes) || r.URL == nil {
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("X-Vec-Actor") != "" || r.Header.Get("X-Vec-Persona") != "" || r.Header.Get("X-Vec-Perfil") != "" || r.URL.EscapedPath() != r.URL.Path {
		responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	if r.Header.Get("Accept") != "application/json" {
		responderErrorAsignacion(w, http.StatusNotAcceptable, "representacion_no_admitida")
		return
	}
	var s personaldomain.SolicitudRectificacionDietas
	lecturaCompetentes := false
	switch {
	case r.URL.Path == RutaSolicitudesRectificacionDietas+"/competentes" && r.Method == http.MethodGet:
		accion = "consultar_competentes"
		if r.URL.RawQuery != "" || r.URL.ForceQuery || !sinCuerpoAsignacion(r) {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		lecturaCompetentes = true
	case r.URL.Path == RutaSolicitudesRectificacionDietas && r.Method == http.MethodPost:
		accion = "solicitar"
		if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		var p peticionRectificacionJSON
		if decodificarRectificacion(w, r, &p) != nil {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		s = personaldomain.SolicitudRectificacionDietas{Operacion: personaldomain.SolicitarRectificacionDietas,
			RelacionRef: p.RelacionRef, UnidadRef: p.UnidadRef, AsignacionRef: p.AsignacionRef,
			VersionEsperada: p.VersionEsperada, FechaReferencia: personaldomain.FechaCivil(p.FechaReferencia),
			ClaveIdempotencia: p.ClaveIdempotencia, CamposARevisar: p.CamposARevisar,
			MotivoRevision: p.MotivoRevision, DetalleSolicitado: p.DetalleSolicitado}
	case r.URL.Path == RutaSolicitudesRectificacionDietas && r.Method == http.MethodGet:
		accion = "consultar"
		if !sinCuerpoAsignacion(r) {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil || r.URL.ForceQuery || len(q) != 3 || len(q["relacion_ref"]) != 1 || len(q["unidad_ref"]) != 1 || len(q["fecha_referencia"]) != 1 {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		s = personaldomain.SolicitudRectificacionDietas{Operacion: personaldomain.ConsultarRectificacionDietas,
			RelacionRef: q.Get("relacion_ref"), UnidadRef: q.Get("unidad_ref"), FechaReferencia: personaldomain.FechaCivil(q.Get("fecha_referencia"))}
	case strings.HasPrefix(r.URL.Path, RutaSolicitudesRectificacionDietas+"/") && r.Method == http.MethodPut:
		if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		ref := strings.TrimPrefix(r.URL.Path, RutaSolicitudesRectificacionDietas+"/")
		var p resolucionRectificacionJSON
		if strings.Contains(ref, "/") || decodificarRectificacion(w, r, &p) != nil {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		accion = p.Decision
		op := personaldomain.OperacionRectificacionDietas(p.Decision)
		s = personaldomain.SolicitudRectificacionDietas{Operacion: op, SolicitudRef: ref,
			PersonaRef: p.PersonaRef, EmpleadoRef: p.EmpleadoRef,
			RelacionRef: p.RelacionRef, UnidadRef: p.UnidadRef,
			AsignacionRef: p.AsignacionRef, VersionEsperada: p.VersionEsperada,
			FechaReferencia:   personaldomain.FechaCivil(p.FechaReferencia),
			ClaveIdempotencia: p.ClaveIdempotencia, MotivoRevision: p.MotivoRevision}
		if op == personaldomain.ConfirmarRectificacionDietas && p.Correccion != nil {
			s.Correccion = &personaldomain.SolicitudAsignacionDietas{
				Operacion:  personaldomain.CorregirAsignacionDietas,
				PersonaRef: p.PersonaRef, EmpleadoRef: p.EmpleadoRef,
				RelacionRef: p.RelacionRef, UnidadRef: p.UnidadRef,
				FechaReferencia:   personaldomain.FechaCivil(p.FechaReferencia),
				ClaveIdempotencia: p.ClaveIdempotencia, VersionEsperada: p.VersionEsperada,
				CentroRef:                p.Correccion.CentroRef,
				AdministrativoPersonaRef: p.Correccion.AdministrativoPersonaRef,
				ResponsablePersonaRef:    p.Correccion.ResponsablePersonaRef,
				GrupoDieta:               p.Correccion.GrupoDieta,
				VigenteDesde:             personaldomain.FechaCivil(p.Correccion.VigenteDesde),
				MotivoRevision:           p.MotivoRevision, ProcedenciaActoRef: ref,
			}
		}
	default:
		responderErrorAsignacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	actor, err := m.identidades.ResolverIdentidadAsignacionDietas(r.Context())
	if err != nil || actor.Validar() != nil {
		if errors.Is(err, personalports.ErrAutenticacionAsignacionDietasRequerida) {
			responderErrorAsignacion(w, http.StatusUnauthorized, "autenticacion_requerida")
		} else if errors.Is(err, personalports.ErrAsignacionDietasDenegada) {
			responderErrorAsignacion(w, http.StatusForbidden, "acceso_denegado")
		} else {
			responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		}
		return
	}
	actorRef = actor.Principal.ID
	if lecturaCompetentes {
		fecha, err := personaldomain.NuevaFechaCivil(time.Now().UTC().Format("2006-01-02"))
		if err != nil {
			responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
			return
		}
		resultado, err := m.competentes.Consultar(r.Context(), personaldomain.SolicitudRectificacionesCompetentesDietas{Actor: actor, FechaReferencia: fecha})
		if err != nil {
			if errors.Is(err, personalports.ErrRectificacionDietasDenegada) {
				responderErrorAsignacion(w, http.StatusForbidden, "acceso_denegado")
			} else {
				responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
			}
			return
		}
		responderAsignacion(w, http.StatusOK, resultado)
		return
	}
	s.Actor = actor
	if s.Correccion != nil {
		s.Correccion.Actor = actor
	}
	resultado, err := m.casoUso.Ejecutar(r.Context(), s)
	if err != nil {
		switch {
		case errors.Is(err, personalports.ErrRectificacionDietasInvalida):
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
		case errors.Is(err, personalports.ErrRectificacionDietasDenegada):
			responderErrorAsignacion(w, http.StatusForbidden, "acceso_denegado")
		case errors.Is(err, personalports.ErrRectificacionDietasConflicto):
			responderErrorAsignacion(w, http.StatusConflict, "conflicto")
		case errors.Is(err, personalports.ErrRectificacionDietasNoEncontrada):
			responderErrorAsignacion(w, http.StatusNotFound, "no_encontrada")
		default:
			responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		}
		return
	}
	estado := http.StatusOK
	if s.Operacion == personaldomain.SolicitarRectificacionDietas && resultado.Estado == "pendiente" {
		estado = http.StatusCreated
	}
	responderAsignacion(w, estado, resultado)
	return
}

type peticionRectificacionJSON struct {
	RelacionRef       string   `json:"relacion_ref"`
	UnidadRef         string   `json:"unidad_ref"`
	AsignacionRef     string   `json:"asignacion_ref"`
	VersionEsperada   int64    `json:"version_esperada"`
	FechaReferencia   string   `json:"fecha_referencia"`
	ClaveIdempotencia string   `json:"clave_idempotencia"`
	CamposARevisar    []string `json:"campos_a_revisar"`
	MotivoRevision    string   `json:"motivo_revision"`
	DetalleSolicitado string   `json:"detalle_solicitado"`
}

type resolucionRectificacionJSON struct {
	Decision          string                       `json:"decision"`
	PersonaRef        string                       `json:"persona_ref"`
	EmpleadoRef       string                       `json:"empleado_ref"`
	RelacionRef       string                       `json:"relacion_ref"`
	UnidadRef         string                       `json:"unidad_ref"`
	AsignacionRef     string                       `json:"asignacion_ref"`
	VersionEsperada   int64                        `json:"version_esperada"`
	FechaReferencia   string                       `json:"fecha_referencia"`
	ClaveIdempotencia string                       `json:"clave_idempotencia"`
	MotivoRevision    string                       `json:"motivo_revision"`
	Correccion        *correccionRectificacionJSON `json:"correccion"`
}

type correccionRectificacionJSON struct {
	CentroRef                string `json:"centro_ref"`
	AdministrativoPersonaRef string `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string `json:"responsable_persona_ref"`
	GrupoDieta               int16  `json:"grupo_dieta"`
	VigenteDesde             string `json:"vigente_desde"`
}

func decodificarRectificacion(w http.ResponseWriter, r *http.Request, destino any) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength < 1 || r.ContentLength > maximoCuerpoAsignacion || len(r.TransferEncoding) != 0 {
		return personalports.ErrRectificacionDietasInvalida
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximoCuerpoAsignacion))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return personalports.ErrRectificacionDietasInvalida
	}
	return nil
}

func (m *ManejadorRectificacionDietas) auditar(r *http.Request, estado int, accion, actorRef string) error {
	motivo := personalports.MotivoFronteraPersonalDenegado
	if estado == http.StatusServiceUnavailable {
		motivo = personalports.MotivoFronteraPersonalDependencia
	} else if estado == http.StatusUnauthorized {
		motivo = personalports.MotivoFronteraPersonalAutenticacion
	}
	if accion != "solicitar" && accion != "consultar" && accion != "consultar_competentes" && accion != "confirmar" && accion != "rechazar" {
		accion = "metodo_no_admitido"
	}
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	recurso := ""
	if r.URL != nil {
		if strings.HasPrefix(r.URL.Path, RutaSolicitudesRectificacionDietas+"/") {
			candidato := strings.TrimPrefix(r.URL.Path, RutaSolicitudesRectificacionDietas+"/")
			if personalports.ReferenciaFronteraRectificacionDietasValida(candidato) {
				recurso = candidato
			}
		} else if r.Method == http.MethodGet && r.URL.Path == RutaSolicitudesRectificacionDietas {
			candidato := r.URL.Query().Get("relacion_ref")
			if personalports.ReferenciaFronteraRectificacionDietasValida(candidato) {
				recurso = candidato
			}
		}
	}
	o := personalports.OrdenAuditoriaFronteraRectificacionDietas{CorrelacionRef: correlacion, Motivo: motivo,
		Ruta: RutaSolicitudesRectificacionDietas, Accion: accion, ActorRef: actorRef, RecursoRef: recurso, EstadoHTTP: estado}
	if o.Validar() != nil {
		return personalports.ErrRectificacionDietasNoDisponible
	}
	c, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	return m.auditoria.RegistrarAuditoriaFronteraRectificacionDietas(c, o)
}

var _ http.Handler = (*ManejadorRectificacionDietas)(nil)
