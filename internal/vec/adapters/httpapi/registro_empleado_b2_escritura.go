package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"unicode/utf8"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	RutaAltaEmpleadoB2             = "/api/vec/personal/empleados"
	RutaHechosEmpleadoB2           = "/api/vec/personal/hechos"
	maximoCuerpoRegistroEmpleadoB2 = 16 << 10
)

var errEntradaRegistroEmpleadoB2 = errors.New("personal http: entrada de registro de empleado invalida")

// leerCuerpoRegistroEmpleadoB2 admite una sola clave y un solo JSON con
// nombres únicos. El DTO de cada operación limita además los campos concretos.
func leerCuerpoRegistroEmpleadoB2(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	if r == nil || r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 ||
		r.ContentLength > maximoCuerpoRegistroEmpleadoB2 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Content-Encoding") ||
		!cabeceraImportacionOrganizacionExacta(r.Header, "Content-Type", "application/json") {
		return nil, "", errEntradaRegistroEmpleadoB2
	}
	clave, ok := cabeceraImportacionOrganizacionUnica(r.Header, "Idempotency-Key")
	if !ok || !patronClaveHTTPImportacionOrganizacion.MatchString(clave) {
		return nil, "", errEntradaRegistroEmpleadoB2
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoRegistroEmpleadoB2+1))
	if err != nil || len(cuerpo) == 0 || len(cuerpo) > maximoCuerpoRegistroEmpleadoB2 || !utf8.Valid(cuerpo) || validarJSONRegistroEmpleadoB2(cuerpo) != nil || !bytes.HasPrefix(bytes.TrimSpace(cuerpo), []byte("{")) {
		return nil, "", errEntradaRegistroEmpleadoB2
	}
	return cuerpo, clave, nil
}

func validarJSONRegistroEmpleadoB2(cuerpo []byte) error {
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	if err := validarValorJSONRegistroEmpleadoB2(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errEntradaRegistroEmpleadoB2
	}
	return nil
}
func validarValorJSONRegistroEmpleadoB2(d *json.Decoder, profundidad int) error {
	if profundidad > 8 {
		return errEntradaRegistroEmpleadoB2
	}
	token, err := d.Token()
	if err != nil {
		return errEntradaRegistroEmpleadoB2
	}
	inicio, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch inicio {
	case '{':
		vistas := map[string]struct{}{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return errEntradaRegistroEmpleadoB2
			}
			clave, ok := token.(string)
			if !ok || !claveCanonicaImportacionOrganizacion(clave) {
				return errEntradaRegistroEmpleadoB2
			}
			if _, repetida := vistas[clave]; repetida {
				return errEntradaRegistroEmpleadoB2
			}
			vistas[clave] = struct{}{}
			if err := validarValorJSONRegistroEmpleadoB2(d, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := validarValorJSONRegistroEmpleadoB2(d, profundidad+1); err != nil {
				return err
			}
		}
	default:
		return errEntradaRegistroEmpleadoB2
	}
	fin, err := d.Token()
	if err != nil || inicio == '{' && fin != json.Delim('}') || inicio == '[' && fin != json.Delim(']') {
		return errEntradaRegistroEmpleadoB2
	}
	return nil
}

// OperadorActosRegistroEmpleadoB2 recibe solicitudes validadas y atribuidas al
// actor de servidor. Obtiene la acreditación B1 y consume V3 en aplicación.
type OperadorActosRegistroEmpleadoB2 interface {
	RegistrarEmpleado(context.Context, personaldomain.SolicitudAltaEmpleadoB2) (personalports.ResultadoAltaEmpleadoB2, error)
	RegistrarHecho(context.Context, personaldomain.SolicitudHechoEmpleadoB2) (personalports.ResultadoHechoEmpleadoB2, error)
}

var _ OperadorActosRegistroEmpleadoB2 = (*personalapp.ServicioActosRegistroEmpleadoB2)(nil)

type handlerActosRegistroEmpleadoB2 struct {
	autoridad AutoridadContextoRegistroEmpleadoB2
	operador  OperadorActosRegistroEmpleadoB2
	auditoria AuditorDenegacionRegistroEmpleadoB2
	hecho     bool
}

func NewHandlerAltaEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, o OperadorActosRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2) (http.Handler, error) {
	return nuevoHandlerActosRegistroEmpleadoB2(a, o, auditor, false)
}
func NewHandlerHechosEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, o OperadorActosRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2) (http.Handler, error) {
	return nuevoHandlerActosRegistroEmpleadoB2(a, o, auditor, true)
}
func nuevoHandlerActosRegistroEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, o OperadorActosRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2, hecho bool) (http.Handler, error) {
	if dependenciaHTTPNula(a) || dependenciaHTTPNula(o) || dependenciaHTTPNula(auditor) {
		return nil, ErrHandlerRegistroEmpleadoB2Invalido
	}
	return &handlerActosRegistroEmpleadoB2{a, o, auditor, hecho}, nil
}

type entradaAltaEmpleadoB2 struct {
	PersonaRef         string                                   `json:"persona_ref"`
	OrganismoRef       string                                   `json:"organismo_ref"`
	UnidadRef          string                                   `json:"unidad_ref"`
	Regimen            personaldomain.EntradaCatalogoEmpleadoB2 `json:"regimen"`
	Modalidad          personaldomain.EntradaCatalogoEmpleadoB2 `json:"modalidad"`
	VigenteDesde       string                                   `json:"vigente_desde"`
	VigenteHasta       string                                   `json:"vigente_hasta"`
	ActoRef            string                                   `json:"acto_ref"`
	FuenteRef          string                                   `json:"fuente_ref"`
	FuenteVersion      int64                                    `json:"fuente_version"`
	FuenteHuellaSHA256 string                                   `json:"fuente_huella_sha256"`
}

type entradaHechoEmpleadoB2 struct {
	Tipo                    string                                   `json:"tipo"`
	EmpleadoRef             string                                   `json:"empleado_ref"`
	RelacionRef             string                                   `json:"relacion_ref"`
	RevisionEsperada        int64                                    `json:"revision_esperada"`
	RelacionVersionEsperada int64                                    `json:"relacion_version_esperada"`
	UnidadRef               string                                   `json:"unidad_ref"`
	Regimen                 personaldomain.EntradaCatalogoEmpleadoB2 `json:"regimen"`
	Modalidad               personaldomain.EntradaCatalogoEmpleadoB2 `json:"modalidad"`
	Situacion               personaldomain.EntradaCatalogoEmpleadoB2 `json:"situacion"`
	ClaseServicio           personaldomain.EntradaCatalogoEmpleadoB2 `json:"clase_servicio"`
	ClaseOcupacion          string                                   `json:"clase_ocupacion"`
	Estado                  string                                   `json:"estado"`
	PlazaRef                string                                   `json:"plaza_ref"`
	PuestoRef               string                                   `json:"puesto_ref"`
	VersionPlazaRef         string                                   `json:"version_plaza_ref"`
	VersionPuestoRef        string                                   `json:"version_puesto_ref"`
	PeriodoDesde            string                                   `json:"periodo_desde"`
	PeriodoHasta            string                                   `json:"periodo_hasta"`
	DiasReconocidos         int64                                    `json:"dias_reconocidos"`
	VigenteDesde            string                                   `json:"vigente_desde"`
	VigenteHasta            string                                   `json:"vigente_hasta"`
	ActoRef                 string                                   `json:"acto_ref"`
	FuenteRef               string                                   `json:"fuente_ref"`
	FuenteVersion           int64                                    `json:"fuente_version"`
	FuenteHuellaSHA256      string                                   `json:"fuente_huella_sha256"`
}

func (h *handlerActosRegistroEmpleadoB2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaHTTPNula(h.autoridad) || dependenciaHTTPNula(h.operador) || dependenciaHTTPNula(h.auditoria) {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	ruta := RutaAltaEmpleadoB2
	if h.hecho {
		ruta = RutaHechosEmpleadoB2
	}
	if !peticionRutaExactaCanonica(r) || r.URL.Path != ruta || r.URL.RawQuery != "" {
		responderRegistroEmpleadoB2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderRegistroEmpleadoB2(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	cuerpo, clave, err := leerCuerpoRegistroEmpleadoB2(w, r)
	if err != nil {
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	var entradaAlta entradaAltaEmpleadoB2
	var entradaHecho entradaHechoEmpleadoB2
	if h.hecho {
		entradaHecho, err = decodificarHechoEmpleadoB2(cuerpo)
	} else {
		entradaAlta, err = decodificarAltaEmpleadoB2(cuerpo)
	}
	if err != nil {
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	actor, organismo, err := h.autoridad.ResolverContextoRegistroEmpleadoB2(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrAutenticacionRutaExactaRequerida), errors.Is(err, vecdomain.ErrContextoActorNoResuelto):
			h.denegar(w, r.Context(), http.StatusUnauthorized, "autenticacion_requerida", ruta, "")
		case errors.Is(err, ErrAccesoRutaExactaDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrPermissionDenied):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", ruta, "")
		default:
			responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	if actor.Validar() != nil {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if organismo == "" {
		h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", ruta, actor.Principal.ID)
		return
	}
	if h.hecho {
		solicitud, err := solicitudHechoEmpleadoB2(entradaHecho, clave, organismo, actor)
		if err != nil {
			responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
		resultado, err := h.operador.RegistrarHecho(r.Context(), solicitud)
		if err != nil {
			h.errorActo(w, r.Context(), ruta, actor.Principal.ID, err)
			return
		}
		if !reciboActoRegistroEmpleadoB2HTTPValido(resultado.Recibo, false) || !accesoActualRegistroEmpleadoB2HTTPValido(resultado.AccesoActual) || resultado.Recibo.EmpleadoRef != solicitud.EmpleadoRef || resultado.Recibo.Tipo != solicitud.Tipo || (solicitud.RelacionRef != "" && resultado.Recibo.RelacionRef != solicitud.RelacionRef) {
			responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
			return
		}
		estado := http.StatusCreated
		if resultado.AccesoActual.EstadoReplay == "replay" {
			estado = http.StatusOK
		}
		responderRegistroEmpleadoB2(w, estado, "", map[string]any{"data": resultado})
		return
	}
	if organismo != "" && entradaAlta.OrganismoRef != organismo {
		h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", ruta, actor.Principal.ID)
		return
	}
	solicitud, err := solicitudAltaEmpleadoB2(entradaAlta, clave, actor)
	if err != nil {
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	resultado, err := h.operador.RegistrarEmpleado(r.Context(), solicitud)
	if err != nil {
		h.errorActo(w, r.Context(), ruta, actor.Principal.ID, err)
		return
	}
	if !reciboActoRegistroEmpleadoB2HTTPValido(resultado.Recibo, true) || !accesoActualRegistroEmpleadoB2HTTPValido(resultado.AccesoActual) {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	estado := http.StatusCreated
	if resultado.AccesoActual.EstadoReplay == "replay" {
		estado = http.StatusOK
	}
	responderRegistroEmpleadoB2(w, estado, "", map[string]any{"data": resultado})
}

func decodificarAltaEmpleadoB2(cuerpo []byte) (entradaAltaEmpleadoB2, error) {
	var entrada entradaAltaEmpleadoB2
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	d.DisallowUnknownFields()
	if err := d.Decode(&entrada); err != nil {
		return entrada, err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return entrada, errEntradaRegistroEmpleadoB2
	}
	return entrada, nil
}
func decodificarHechoEmpleadoB2(cuerpo []byte) (entradaHechoEmpleadoB2, error) {
	var entrada entradaHechoEmpleadoB2
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	d.DisallowUnknownFields()
	if err := d.Decode(&entrada); err != nil {
		return entrada, err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return entrada, errEntradaRegistroEmpleadoB2
	}
	return entrada, nil
}

func solicitudAltaEmpleadoB2(e entradaAltaEmpleadoB2, clave string, actor vecdomain.ContextoActor) (personaldomain.SolicitudAltaEmpleadoB2, error) {
	var s personaldomain.SolicitudAltaEmpleadoB2
	desde, err := personaldomain.NuevaFechaCivil(e.VigenteDesde)
	if err != nil {
		return s, err
	}
	var hasta personaldomain.FechaCivil
	if e.VigenteHasta != "" {
		hasta, err = personaldomain.NuevaFechaCivil(e.VigenteHasta)
		if err != nil {
			return s, err
		}
	}
	s = personaldomain.SolicitudAltaEmpleadoB2{
		PersonaRef: e.PersonaRef, OrganismoRef: e.OrganismoRef, UnidadRef: e.UnidadRef, Regimen: e.Regimen, Modalidad: e.Modalidad,
		VigenteDesde: desde, VigenteHasta: hasta, Procedencia: personaldomain.ProcedenciaActoEmpleadoB2{ActoRef: e.ActoRef, FuenteRef: e.FuenteRef, FuenteVersion: e.FuenteVersion, FuenteHuellaSHA256: e.FuenteHuellaSHA256, IdempotenciaRef: clave}, Actor: actor,
	}
	// Personal enlaza la persona B1 dentro de la transacción SQL. La
	// frontera verifica el material completo sin convertir el selector en permiso.
	if _, err := personaldomain.NuevoMaterialAltaEmpleadoB2(s); err != nil {
		return personaldomain.SolicitudAltaEmpleadoB2{}, err
	}
	return s, nil
}
func solicitudHechoEmpleadoB2(e entradaHechoEmpleadoB2, clave, organismo string, actor vecdomain.ContextoActor) (personaldomain.SolicitudHechoEmpleadoB2, error) {
	var s personaldomain.SolicitudHechoEmpleadoB2
	desde, err := personaldomain.NuevaFechaCivil(e.VigenteDesde)
	if err != nil {
		return s, err
	}
	var hasta, periodoDesde, periodoHasta personaldomain.FechaCivil
	for _, x := range []struct {
		valor   string
		destino *personaldomain.FechaCivil
	}{{e.VigenteHasta, &hasta}, {e.PeriodoDesde, &periodoDesde}, {e.PeriodoHasta, &periodoHasta}} {
		if x.valor != "" {
			*x.destino, err = personaldomain.NuevaFechaCivil(x.valor)
			if err != nil {
				return s, err
			}
		}
	}
	s = personaldomain.SolicitudHechoEmpleadoB2{
		Tipo: e.Tipo, OrganismoRef: organismo, EmpleadoRef: e.EmpleadoRef, RelacionRef: e.RelacionRef, RevisionEsperada: e.RevisionEsperada, RelacionVersionEsperada: e.RelacionVersionEsperada,
		UnidadRef: e.UnidadRef, Regimen: e.Regimen, Modalidad: e.Modalidad, Situacion: e.Situacion, ClaseServicio: e.ClaseServicio, ClaseOcupacion: e.ClaseOcupacion, Estado: e.Estado,
		PlazaRef: e.PlazaRef, PuestoRef: e.PuestoRef, VersionPlazaRef: e.VersionPlazaRef, VersionPuestoRef: e.VersionPuestoRef,
		PeriodoDesde: periodoDesde, PeriodoHasta: periodoHasta, DiasReconocidos: e.DiasReconocidos, VigenteDesde: desde, VigenteHasta: hasta,
		Procedencia: personaldomain.ProcedenciaActoEmpleadoB2{ActoRef: e.ActoRef, FuenteRef: e.FuenteRef, FuenteVersion: e.FuenteVersion, FuenteHuellaSHA256: e.FuenteHuellaSHA256, IdempotenciaRef: clave}, Actor: actor,
	}
	if _, err = personaldomain.NuevoMaterialHechoEmpleadoB2(s); err != nil {
		return personaldomain.SolicitudHechoEmpleadoB2{}, err
	}
	return s, nil
}

func reciboActoRegistroEmpleadoB2HTTPValido(r personalports.ReciboActoRegistroEmpleadoB2, alta bool) bool {
	if r.ReciboRef == "" || !personaldomain.ReferenciaEmpleadoValida(r.EmpleadoRef) || !personaldomain.ReferenciaRelacionValida(r.RelacionRef) ||
		r.Version < 1 || r.RegistradoEn.IsZero() || r.DecisionRef == "" || r.EfectoRef == "" || r.ConsumoHuellaSHA256 == "" || r.AuditoriaRef == "" ||
		r.EficaciaAdministrativa || r.FirmaOficial {
		return false
	}
	if alta {
		return r.Tipo == "alta" && r.ProyeccionRef != "" && r.HechoRef == ""
	}
	return r.Tipo != "" && r.Tipo != "alta" && r.HechoRef != ""
}

func accesoActualRegistroEmpleadoB2HTTPValido(a personalports.AccesoActualRegistroEmpleadoB2) bool {
	return a.DecisionRef != "" && a.EfectoRef != "" && a.ConsumoHuellaSHA256 != "" && a.AuditoriaRef != "" && !a.ConsultadaEn.IsZero() &&
		(a.EstadoReplay == "registrado" || a.EstadoReplay == "replay")
}

func (h *handlerActosRegistroEmpleadoB2) errorActo(w http.ResponseWriter, ctx context.Context, ruta, actor string, err error) {
	switch {
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Denegado):
		h.denegar(w, ctx, http.StatusForbidden, "acceso_denegado", ruta, actor)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2NoEncontrado):
		responderRegistroEmpleadoB2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Invalido):
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Conflicto):
		responderRegistroEmpleadoB2(w, http.StatusConflict, "conflicto", nil)
	default:
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}
func (h *handlerActosRegistroEmpleadoB2) denegar(w http.ResponseWriter, ctx context.Context, estado int, codigo, ruta, actor string) {
	orden := DenegacionRegistroEmpleadoB2{CorrelacionRef: nuevaCorrelacionRutaExacta(), Motivo: codigo, Ruta: ruta, ActorRef: actor}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta)
	defer cancelar()
	if err := h.auditoria.RegistrarDenegacionRegistroEmpleadoB2(ctxAuditoria, orden); err != nil {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderRegistroEmpleadoB2(w, estado, codigo, nil)
}
