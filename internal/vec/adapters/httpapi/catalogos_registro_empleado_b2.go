package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const RutaCatalogosRegistroEmpleadoB2 = "/api/vec/personal/catalogos-registro-empleado"

// OperadorCatalogosRegistroEmpleadoB2 recibe el actor y el organismo que ha
// resuelto el servidor. La solicitud HTTP solo aporta el selector y el acto.
type OperadorCatalogosRegistroEmpleadoB2 interface {
	Consultar(context.Context, personaldomain.SolicitudConsultaCatalogoEmpleadoB2) (personalports.ResultadoConsultaCatalogoEmpleadoB2, error)
	Cambiar(context.Context, personaldomain.SolicitudCambioCatalogoEmpleadoB2) (personalports.ResultadoCambioCatalogoEmpleadoB2, error)
}

type handlerCatalogosRegistroEmpleadoB2 struct {
	autoridad AutoridadContextoRegistroEmpleadoB2
	operador  OperadorCatalogosRegistroEmpleadoB2
	auditoria AuditorDenegacionRegistroEmpleadoB2
}

func NewHandlerCatalogosRegistroEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, o OperadorCatalogosRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2) (http.Handler, error) {
	if dependenciaHTTPNula(a) || dependenciaHTTPNula(o) || dependenciaHTTPNula(auditor) {
		return nil, ErrHandlerRegistroEmpleadoB2Invalido
	}
	return &handlerCatalogosRegistroEmpleadoB2{a, o, auditor}, nil
}

type entradaCambioCatalogoEmpleadoB2 struct {
	Operacion    string `json:"operacion"`
	Tipo         string `json:"tipo"`
	Ref          string `json:"ref"`
	Version      int64  `json:"version"`
	Revision     int64  `json:"revision"`
	Denominacion string `json:"denominacion"`
	HuellaSHA256 string `json:"huella_sha256"`
	VigenteDesde string `json:"vigente_desde"`
	VigenteHasta string `json:"vigente_hasta"`
	ActoRef      string `json:"acto_ref"`
}

func (h *handlerCatalogosRegistroEmpleadoB2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaHTTPNula(h.autoridad) || dependenciaHTTPNula(h.operador) || dependenciaHTTPNula(h.auditoria) {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if !peticionRutaExactaCanonica(r) || r.URL.Path != RutaCatalogosRegistroEmpleadoB2 {
		responderRegistroEmpleadoB2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responderRegistroEmpleadoB2(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	var filtro filtroCatalogoEmpleadoB2
	var clave string
	var entrada entradaCambioCatalogoEmpleadoB2
	if r.Method == http.MethodGet {
		var err error
		filtro, err = leerFiltroCatalogoEmpleadoB2(r)
		if err != nil {
			responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
	} else {
		if r.URL.RawQuery != "" {
			responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
		cuerpo, idempotencia, err := leerCuerpoRegistroEmpleadoB2(w, r)
		if err != nil {
			responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
		clave = idempotencia
		d := json.NewDecoder(bytes.NewReader(cuerpo))
		d.DisallowUnknownFields()
		if d.Decode(&entrada) != nil || d.Decode(new(any)) != io.EOF {
			responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
	}
	actor, organismo, err := h.autoridad.ResolverContextoRegistroEmpleadoB2(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrAutenticacionRutaExactaRequerida), errors.Is(err, vecdomain.ErrContextoActorNoResuelto):
			h.denegar(w, r.Context(), http.StatusUnauthorized, "autenticacion_requerida", "")
		case errors.Is(err, ErrAccesoRutaExactaDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrPermissionDenied):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", "")
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
		h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", actor.Principal.ID)
		return
	}
	if r.Method == http.MethodGet {
		solicitud := personaldomain.SolicitudConsultaCatalogoEmpleadoB2{
			OrganismoRef: organismo, Tipo: filtro.tipo, Estado: filtro.estado,
			CursorRef: filtro.cursorRef, CursorVersion: filtro.cursorVersion,
			Limite: filtro.limite, Actor: actor,
		}
		resultado, err := h.operador.Consultar(r.Context(), solicitud)
		if err != nil {
			h.errorOperacion(w, r.Context(), actor.Principal.ID, err)
			return
		}
		for _, e := range resultado.Entradas {
			if e.Validar() != nil || e.OrganismoRef != organismo || e.Tipo != filtro.tipo || (filtro.estado != "" && e.Estado != filtro.estado) {
				responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
				return
			}
		}
		if resultado.OrganismoRef != organismo || resultado.Evidencia.DecisionRef == "" || resultado.Evidencia.AuditoriaRef == "" ||
			resultado.Evidencia.ConsumoHuellaSHA256 == "" || resultado.Evidencia.EfectoRef != organismo+":"+filtro.tipo ||
			resultado.Evidencia.ConsultadaEn.IsZero() || len(resultado.Entradas) > filtro.limite {
			responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
			return
		}
		responderRegistroEmpleadoB2(w, http.StatusOK, "", map[string]any{"data": resultado})
		return
	}
	desde, err := personaldomain.NuevaFechaCivil(entrada.VigenteDesde)
	if err != nil {
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	var hasta personaldomain.FechaCivil
	if entrada.VigenteHasta != "" {
		hasta, err = personaldomain.NuevaFechaCivil(entrada.VigenteHasta)
		if err != nil {
			responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
	}
	solicitud := personaldomain.SolicitudCambioCatalogoEmpleadoB2{
		Operacion: entrada.Operacion, OrganismoRef: organismo, Tipo: entrada.Tipo, Ref: entrada.Ref,
		Version: entrada.Version, Revision: entrada.Revision, Denominacion: entrada.Denominacion,
		HuellaSHA256: entrada.HuellaSHA256, VigenteDesde: desde, VigenteHasta: hasta,
		ActoRef: entrada.ActoRef, IdempotenciaRef: clave, Actor: actor,
	}
	resultado, err := h.operador.Cambiar(r.Context(), solicitud)
	if err != nil {
		h.errorOperacion(w, r.Context(), actor.Principal.ID, err)
		return
	}
	if resultado.Entrada.Validar() != nil || resultado.Entrada.OrganismoRef != organismo || resultado.Entrada.Tipo != entrada.Tipo ||
		resultado.Entrada.Ref != entrada.Ref || resultado.Entrada.Version != entrada.Version ||
		resultado.Entrada.Revision != entrada.Revision || resultado.Entrada.Denominacion != entrada.Denominacion ||
		resultado.Entrada.HuellaSHA256 != entrada.HuellaSHA256 || resultado.Entrada.VigenteDesde != desde ||
		resultado.Entrada.VigenteHasta != hasta ||
		(entrada.Operacion == "publicar" && resultado.Entrada.Estado != "publicada") ||
		(entrada.Operacion == "retirar" && resultado.Entrada.Estado != "retirada") ||
		resultado.Recibo.DecisionRef == "" ||
		resultado.Recibo.AuditoriaRef == "" || resultado.Recibo.ConsumoHuellaSHA256 == "" ||
		resultado.Recibo.RegistradoEn.IsZero() || resultado.AccesoActual.DecisionRef == "" ||
		resultado.AccesoActual.AuditoriaRef == "" || resultado.AccesoActual.ConsumoHuellaSHA256 == "" ||
		resultado.AccesoActual.RegistradoEn.IsZero() ||
		(resultado.AccesoActual.EstadoReplay != "registrado" && resultado.AccesoActual.EstadoReplay != "replay") {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	estado := http.StatusCreated
	if resultado.AccesoActual.EstadoReplay == "replay" {
		estado = http.StatusOK
	}
	responderRegistroEmpleadoB2(w, estado, "", map[string]any{"data": resultado})
}

type filtroCatalogoEmpleadoB2 struct {
	tipo          string
	estado        string
	cursorRef     string
	cursorVersion int64
	limite        int
}

func leerFiltroCatalogoEmpleadoB2(r *http.Request) (filtroCatalogoEmpleadoB2, error) {
	var f filtroCatalogoEmpleadoB2
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		(r.Body != nil && r.Body != http.NoBody) || cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") || cabeceraOrganizacionHistoricaPresente(r.Header, "Content-Encoding") ||
		len(r.URL.RawQuery) > 600 || strings.Contains(r.URL.RawQuery, ";") {
		return f, errEntradaRegistroEmpleadoB2
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q["tipo"]) != 1 || len(q) > 5 {
		return f, errEntradaRegistroEmpleadoB2
	}
	for clave, valores := range q {
		if (clave != "tipo" && clave != "estado" && clave != "cursor_ref" && clave != "cursor_version" && clave != "limite") ||
			len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
			return f, errEntradaRegistroEmpleadoB2
		}
	}
	f.tipo = q.Get("tipo")
	if f.tipo != "regimen" && f.tipo != "modalidad" && f.tipo != "situacion" && f.tipo != "clase_servicio" {
		return f, errEntradaRegistroEmpleadoB2
	}
	f.estado = q.Get("estado")
	if f.estado != "" && f.estado != "publicada" && f.estado != "retirada" {
		return f, errEntradaRegistroEmpleadoB2
	}
	f.limite = 50
	if valor := q.Get("limite"); valor != "" {
		f.limite, err = strconv.Atoi(valor)
		if err != nil || strconv.Itoa(f.limite) != valor || f.limite < 1 || f.limite > 100 {
			return f, errEntradaRegistroEmpleadoB2
		}
	}
	f.cursorRef = q.Get("cursor_ref")
	if valor := q.Get("cursor_version"); valor != "" {
		f.cursorVersion, err = strconv.ParseInt(valor, 10, 64)
		if err != nil || strconv.FormatInt(f.cursorVersion, 10) != valor || f.cursorVersion < 1 {
			return f, errEntradaRegistroEmpleadoB2
		}
	}
	if (f.cursorRef == "") != (f.cursorVersion == 0) {
		return f, errEntradaRegistroEmpleadoB2
	}
	return f, nil
}

func (h *handlerCatalogosRegistroEmpleadoB2) errorOperacion(w http.ResponseWriter, ctx context.Context, actor string, err error) {
	switch {
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Denegado):
		h.denegar(w, ctx, http.StatusForbidden, "acceso_denegado", actor)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Invalido):
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Conflicto):
		responderRegistroEmpleadoB2(w, http.StatusConflict, "conflicto", nil)
	default:
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}

func (h *handlerCatalogosRegistroEmpleadoB2) denegar(w http.ResponseWriter, ctx context.Context, estado int, codigo, actor string) {
	orden := DenegacionRegistroEmpleadoB2{CorrelacionRef: nuevaCorrelacionRutaExacta(), Motivo: codigo, Ruta: RutaCatalogosRegistroEmpleadoB2, ActorRef: actor}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta)
	defer cancelar()
	if h.auditoria.RegistrarDenegacionRegistroEmpleadoB2(ctxAuditoria, orden) != nil {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderRegistroEmpleadoB2(w, estado, codigo, nil)
}
