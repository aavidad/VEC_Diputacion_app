package contactopropio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaOperacionContactoPreparar  = "/api/vec/usuarios/contacto-propio/operaciones/preparar"
	RutaOperacionContactoConfirmar = "/api/vec/usuarios/contacto-propio/operaciones/confirmar"
	RutaOperacionContactoCancelar  = "/api/vec/usuarios/contacto-propio/operaciones/cancelar"
	RutaOperacionContactoConsultas = "/api/vec/usuarios/contacto-propio/operaciones/consultas"
	RutaOperacionContactoDetalle   = "/api/vec/usuarios/contacto-propio/operaciones/detalle"
)

// El transporte no acepta persona, actor, perfil, permisos ni referencias de
// material V3. El ejecutor resuelve identidad fresca y persiste/consulta por
// puertos nominales. Ninguna ruta se monta sin ese ejecutor completo.
type EjecutorOperacionesContactoPropio interface {
	PrepararOperacion(context.Context, string, uint64) (ports.OperacionContactoUsuario, error)
	ConfirmarOperacion(context.Context, string, string, uint64) (ports.OperacionContactoUsuario, error)
	CancelarOperacion(context.Context, string) (ports.OperacionContactoUsuario, error)
	ListarOperaciones(context.Context, uint32, string) (ports.ResultadoListaOperacionesContacto, error)
	DetalleOperacion(context.Context, string) (ports.ResultadoDetalleOperacionContacto, error)
}

type manejadorOperacionesContacto struct {
	ejecutor EjecutorOperacionesContactoPropio
	ruta     string
}

type entradaConfirmarOperacionContacto struct {
	OperacionRef    string `json:"operacion_ref"`
	Correo          string `json:"correo"`
	VersionEsperada uint64 `json:"version_esperada"`
}

type entradaReferenciaOperacionContacto struct {
	OperacionRef string `json:"operacion_ref"`
}

type entradaListaOperacionesContacto struct {
	Limite    uint32 `json:"limite"`
	DespuesDe string `json:"despues_de"`
}

type respuestaOperacionContacto struct {
	OperacionRef    string `json:"operacion_ref"`
	Estado          string `json:"estado"`
	VersionEsperada uint64 `json:"version_esperada"`
	Version         uint64 `json:"version,omitempty"`
	ReciboRef       string `json:"recibo_ref,omitempty"`
}

func respuestaOperacion(op ports.OperacionContactoUsuario) respuestaOperacionContacto {
	return respuestaOperacionContacto{OperacionRef: op.OperacionRef, Estado: string(op.Estado), VersionEsperada: op.VersionEsperada, Version: op.Version, ReciboRef: op.ReciboRef}
}

// NuevasRutasOperaciones sólo entrega declaraciones exactas. La composición
// registra estas cinco rutas juntas al pasar a AD3-54/T13-8/Contacto3. Esa
// transición también retira el POST directo anterior: Contacto3 revoca su
// EXECUTE y el runtime debe consumir ServicioOperaciones con V3 nominal.
func NuevasRutasOperaciones(e EjecutorOperacionesContactoPropio, catalogo *i18n.Catalog) ([]httpapi.RutaExacta, error) {
	if dependenciaContactoPropioNula(e) || catalogo == nil {
		return nil, ErrManejadorContactoPropioInvalido
	}
	rutas := []string{RutaOperacionContactoPreparar, RutaOperacionContactoConfirmar, RutaOperacionContactoCancelar, RutaOperacionContactoConsultas, RutaOperacionContactoDetalle}
	salida := make([]httpapi.RutaExacta, 0, len(rutas))
	for _, ruta := range rutas {
		salida = append(salida, httpapi.RutaExacta{Ruta: ruta, Manejador: &manejadorOperacionesContacto{ejecutor: e, ruta: ruta}})
	}
	return salida, nil
}

func (h *manejadorOperacionesContacto) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h == nil || dependenciaContactoPropioNula(h.ejecutor) || r == nil || r.URL == nil {
		responderCodigoOperacion(w, http.StatusForbidden, "acceso_denegado", "")
		return
	}
	if r.URL.Path != h.ruta || r.URL.RawQuery != "" || r.URL.EscapedPath() != h.ruta {
		responderCodigoOperacion(w, http.StatusNotFound, "no_encontrada", "")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderCodigoOperacion(w, http.StatusMethodNotAllowed, "metodo_no_admitido", "")
		return
	}
	if r.Header.Get("Cookie") != "" || len(r.Cookies()) != 0 || !esJSONContactoPropio(r.Header.Get("Content-Type")) {
		responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
		return
	}
	switch h.ruta {
	case RutaOperacionContactoPreparar:
		var p entradaContactoPropio
		if leerJSONOperacionContacto(r.Body, []string{"correo", "version_esperada"}, &p) != nil || !correoOperacionContactoValido(p.Correo) || p.VersionEsperada >= 1<<53-1 {
			responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
			return
		}
		op, err := h.ejecutor.PrepararOperacion(r.Context(), p.Correo, p.VersionEsperada)
		h.responderOperacion(w, op, err, http.StatusCreated, ports.OperacionContactoPreparada)
	case RutaOperacionContactoConfirmar:
		var p entradaConfirmarOperacionContacto
		if leerJSONOperacionContacto(r.Body, []string{"operacion_ref", "correo", "version_esperada"}, &p) != nil || !application.ReferenciaOperacionContactoValida(p.OperacionRef) || !correoOperacionContactoValido(p.Correo) || p.VersionEsperada >= 1<<53-1 {
			responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
			return
		}
		op, err := h.ejecutor.ConfirmarOperacion(r.Context(), p.OperacionRef, p.Correo, p.VersionEsperada)
		if errors.Is(err, ErrContactoPropioCommitIncierto) {
			responderCodigoOperacion(w, http.StatusServiceUnavailable, "confirmacion_incierta", p.OperacionRef)
			return
		}
		h.responderOperacion(w, op, err, http.StatusCreated, ports.OperacionContactoConfirmada)
	case RutaOperacionContactoCancelar:
		var p entradaReferenciaOperacionContacto
		if leerJSONOperacionContacto(r.Body, []string{"operacion_ref"}, &p) != nil || !application.ReferenciaOperacionContactoValida(p.OperacionRef) {
			responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
			return
		}
		op, err := h.ejecutor.CancelarOperacion(r.Context(), p.OperacionRef)
		if errors.Is(err, ErrContactoPropioCommitIncierto) {
			responderCodigoOperacion(w, http.StatusServiceUnavailable, "confirmacion_incierta", p.OperacionRef)
			return
		}
		h.responderOperacion(w, op, err, http.StatusOK, ports.OperacionContactoCancelada)
	case RutaOperacionContactoConsultas:
		var p entradaListaOperacionesContacto
		if leerJSONOperacionContacto(r.Body, []string{"limite", "despues_de?"}, &p) != nil || p.Limite < 1 || p.Limite > 50 || p.DespuesDe != "" && !application.ReferenciaOperacionContactoValida(p.DespuesDe) {
			responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
			return
		}
		lista, err := h.ejecutor.ListarOperaciones(r.Context(), p.Limite, p.DespuesDe)
		if err != nil {
			responderErrorOperacion(w, err, "")
			return
		}
		operaciones := make([]respuestaOperacionContacto, 0, len(lista.Operaciones))
		for _, op := range lista.Operaciones {
			if application.ValidarOperacionContacto(op) != nil {
				responderCodigoOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", "")
				return
			}
			operaciones = append(operaciones, respuestaOperacion(op))
		}
		if lista.SiguienteDesde != "" && (!application.ReferenciaOperacionContactoValida(lista.SiguienteDesde) || len(operaciones) == 0 || lista.SiguienteDesde != operaciones[len(operaciones)-1].OperacionRef) || len(operaciones) > int(p.Limite) {
			responderCodigoOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", "")
			return
		}
		responderJSONContactoPropio(w, http.StatusOK, struct {
			Operaciones    []respuestaOperacionContacto `json:"operaciones"`
			SiguienteDesde string                       `json:"siguiente_desde,omitempty"`
		}{operaciones, lista.SiguienteDesde})
	case RutaOperacionContactoDetalle:
		var p entradaReferenciaOperacionContacto
		if leerJSONOperacionContacto(r.Body, []string{"operacion_ref"}, &p) != nil || !application.ReferenciaOperacionContactoValida(p.OperacionRef) {
			responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
			return
		}
		detalle, err := h.ejecutor.DetalleOperacion(r.Context(), p.OperacionRef)
		if err != nil {
			responderErrorOperacion(w, err, "")
			return
		}
		if !detalle.Encontrada {
			responderCodigoOperacion(w, http.StatusNotFound, "no_encontrada", "")
			return
		}
		if detalle.Operacion.OperacionRef != p.OperacionRef || application.ValidarOperacionContacto(detalle.Operacion) != nil {
			responderCodigoOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", "")
			return
		}
		responderJSONContactoPropio(w, http.StatusOK, respuestaOperacion(detalle.Operacion))
	default:
		responderCodigoOperacion(w, http.StatusNotFound, "no_encontrada", "")
	}
}

func (h *manejadorOperacionesContacto) responderOperacion(w http.ResponseWriter, op ports.OperacionContactoUsuario, err error, nuevo int, esperado ports.EstadoOperacionContactoUsuario) {
	if err != nil {
		ref := ""
		if errors.Is(err, application.ErrOperacionContactoPreparada) && application.ReferenciaOperacionContactoValida(op.OperacionRef) {
			ref = op.OperacionRef
		}
		responderErrorOperacion(w, err, ref)
		return
	}
	preparacionYaConfirmada := esperado == ports.OperacionContactoPreparada && op.Estado == ports.OperacionContactoConfirmada && op.ReplayConfirmado
	if application.ValidarOperacionContacto(op) != nil || (op.Estado != esperado && !preparacionYaConfirmada) {
		responderCodigoOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", "")
		return
	}
	if op.ReplayConfirmado {
		nuevo = http.StatusOK
	}
	responderJSONContactoPropio(w, nuevo, respuestaOperacion(op))
}

func responderErrorOperacion(w http.ResponseWriter, err error, ref string) {
	switch {
	case errors.Is(err, ErrContactoPropioInvalido):
		responderCodigoOperacion(w, http.StatusBadRequest, "peticion_invalida", "")
	case errors.Is(err, ErrContactoPropioConflicto):
		responderCodigoOperacion(w, http.StatusConflict, "conflicto", "")
	case errors.Is(err, application.ErrOperacionContactoPreparada):
		responderCodigoOperacion(w, http.StatusConflict, "operacion_preparada", ref)
	case errors.Is(err, application.ErrOperacionContactoNoEncontrada):
		responderCodigoOperacion(w, http.StatusNotFound, "no_encontrada", "")
	case errors.Is(err, application.ErrOperacionContactoAccesoDenegado):
		responderCodigoOperacion(w, http.StatusForbidden, "acceso_denegado", "")
	case errors.Is(err, ErrContactoPropioCommitIncierto):
		responderCodigoOperacion(w, http.StatusServiceUnavailable, "confirmacion_incierta", ref)
	default:
		responderCodigoOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", "")
	}
}

func responderCodigoOperacion(w http.ResponseWriter, estado int, codigo, ref string) {
	responderJSONContactoPropio(w, estado, struct {
		Codigo       string `json:"codigo"`
		OperacionRef string `json:"operacion_ref,omitempty"`
	}{codigo, ref})
}

func leerJSONOperacionContacto(cuerpo io.Reader, claves []string, destino any) error {
	if cuerpo == nil {
		return ErrContactoPropioInvalido
	}
	contenido, err := io.ReadAll(io.LimitReader(cuerpo, maximoCuerpoContacto+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoContacto || validarClavesJSONContactoPropio(contenido) != nil {
		return ErrContactoPropioInvalido
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(contenido, &objeto) != nil || len(objeto) > len(claves) {
		return ErrContactoPropioInvalido
	}
	obligatorias := 0
	for _, clave := range claves {
		opcional := strings.HasSuffix(clave, "?")
		clave = strings.TrimSuffix(clave, "?")
		if !opcional {
			obligatorias++
		}
		valor, ok := objeto[clave]
		if !ok && opcional {
			continue
		}
		if !ok || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
			return ErrContactoPropioInvalido
		}
	}
	if len(objeto) < obligatorias {
		return ErrContactoPropioInvalido
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(new(any)) != io.EOF {
		return ErrContactoPropioInvalido
	}
	return nil
}

func correoOperacionContactoValido(correo string) bool {
	return correo != "" && len(correo) <= maximoCorreoContacto && strings.TrimSpace(correo) == correo && !strings.ContainsAny(correo, "\r\n")
}
