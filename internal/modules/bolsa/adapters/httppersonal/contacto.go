package httppersonal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// RutaMiBolsaContacto recibe la confirmación del contacto propio de una
// bolsa. El cuerpo trae la bolsa, la versión que vio la persona y la clave.
const RutaMiBolsaContacto = "/api/vec/bolsa/mi-bolsa/contacto"

type EjecutorContacto interface {
	ConfirmarContacto(context.Context, mibolsa.Orden, string, int64, string) (puertosbolsa.ReciboConfirmacionContacto, error)
}

type HandlerContacto struct {
	preparador Preparador
	ejecutor   EjecutorContacto
}

func NuevoContacto(preparador Preparador, ejecutor EjecutorContacto) (http.Handler, error) {
	if nula(preparador) || nula(ejecutor) {
		return nil, ErrDependenciaNoDisponible
	}
	return &HandlerContacto{preparador: preparador, ejecutor: ejecutor}, nil
}

type entradaContacto struct {
	Bolsa   string `json:"bolsa"`
	Version int64  `json:"version"`
	Clave   string `json:"clave"`
}

func (h *HandlerContacto) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || nula(h.preparador) || nula(h.ejecutor) {
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
		return
	}
	if r.URL == nil || r.URL.Path != RutaMiBolsaContacto || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responder(w, 404, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responder(w, 405, errorRespuesta{"metodo_no_permitido"})
		return
	}
	if cabeceraProhibida(r.Header) || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" ||
		(r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin") ||
		r.ContentLength < 2 || r.ContentLength > maximoCuerpoPortal || len(r.TransferEncoding) != 0 || r.Body == nil {
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoPortal))
	if err != nil || int64(len(contenido)) != r.ContentLength {
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
		return
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var entrada entradaContacto
	if decodificador.Decode(&entrada) != nil || decodificador.More() {
		responder(w, 400, errorRespuesta{"datos_no_validos"})
		return
	}
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	recibo, err := h.ejecutor.ConfirmarContacto(r.Context(), orden, entrada.Bolsa, entrada.Version, entrada.Clave)
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	var salida reciboPortal
	salida.Data.Esquema = "vec.bolsa.mi-bolsa.contacto.v1"
	salida.Data.Referencia, salida.Data.Recibo, salida.Data.Repetida = entrada.Bolsa, recibo.ReciboRef, recibo.Reutilizada
	salida.Data.RegistradaEn = recibo.ConfirmadaEn.UTC().Format(formatoInstantePortal)
	salida.Data.Estado = "confirmado"
	responder(w, estadoRecibo(recibo.Reutilizada), salida)
}

func responderErrorContacto(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, puertosbolsa.ErrPortalContactoCambiado):
		responder(w, 409, errorRespuesta{"contacto_cambiado"})
	case errors.Is(e, puertosbolsa.ErrPortalContactoYaConfirmado):
		responder(w, 409, errorRespuesta{"contacto_ya_confirmado"})
	case errors.Is(e, puertosbolsa.ErrPortalSinContacto):
		responder(w, 409, errorRespuesta{"sin_contacto"})
	default:
		responderErrorPortal(w, e)
	}
}

type contactoPortal struct {
	Bolsa   string `json:"bolsa"`
	Version int64  `json:"version"`
	Origen  *struct {
		Estado    string `json:"estado"`
		UltimoDia string `json:"ultimo_dia"`
	} `json:"origen"`
	ConfirmadaEn *string `json:"confirmada_en"`
}

// respuestaContactos traduce el estado del contacto; el origen dice si la
// marca CONVOCA sigue vigente en el instante de la consulta.
func respuestaContactos(i puertosbolsa.InstantaneaMiBolsa) []contactoPortal {
	if i.Contactos == nil {
		return nil
	}
	salida := make([]contactoPortal, 0, len(i.Contactos))
	for _, c := range i.Contactos {
		x := contactoPortal{Bolsa: c.Bolsa, Version: c.Version, ConfirmadaEn: instantePortal(c.ConfirmadaEn)}
		if o := c.Origen; o != nil {
			estado := "vigente"
			if !i.ConsultadaEn.Before(o.VigenteHasta) {
				estado = "vencido"
			}
			x.Origen = &struct {
				Estado    string `json:"estado"`
				UltimoDia string `json:"ultimo_dia"`
			}{Estado: estado, UltimoDia: o.UltimoDia}
		}
		salida = append(salida, x)
	}
	return salida
}
