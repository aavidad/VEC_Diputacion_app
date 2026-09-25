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

// RutaMiBolsaDisposiciones recibe la disposición de la persona a una oferta
// publicada de su bolsa. El cuerpo solo trae la oferta y la clave.
const RutaMiBolsaDisposiciones = "/api/vec/bolsa/mi-bolsa/disposiciones"

type EjecutorDisposicion interface {
	ManifestarDisposicion(context.Context, mibolsa.Orden, string, string) (puertosbolsa.ReciboDisposicionPortal, error)
}

type HandlerDisposicion struct {
	preparador Preparador
	ejecutor   EjecutorDisposicion
}

func NuevoDisposicion(preparador Preparador, ejecutor EjecutorDisposicion) (http.Handler, error) {
	if nula(preparador) || nula(ejecutor) {
		return nil, ErrDependenciaNoDisponible
	}
	return &HandlerDisposicion{preparador: preparador, ejecutor: ejecutor}, nil
}

type entradaDisposicion struct {
	Oferta string `json:"oferta"`
	Clave  string `json:"clave"`
}

func (h *HandlerDisposicion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || nula(h.preparador) || nula(h.ejecutor) {
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
		return
	}
	if r.URL == nil || r.URL.Path != RutaMiBolsaDisposiciones || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
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
	var entrada entradaDisposicion
	if decodificador.Decode(&entrada) != nil || decodificador.More() {
		responder(w, 400, errorRespuesta{"datos_no_validos"})
		return
	}
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderErrorDisposicion(w, err)
		return
	}
	recibo, err := h.ejecutor.ManifestarDisposicion(r.Context(), orden, entrada.Oferta, entrada.Clave)
	if err != nil {
		responderErrorDisposicion(w, err)
		return
	}
	var salida reciboPortal
	salida.Data.Esquema = "vec.bolsa.mi-bolsa.disposicion.v1"
	salida.Data.Referencia, salida.Data.Recibo, salida.Data.Repetida = recibo.OfertaRef, recibo.ReciboRef, recibo.Reutilizada
	salida.Data.RegistradaEn = recibo.ManifestadaEn.UTC().Format(formatoInstantePortal)
	salida.Data.Estado = "manifestada"
	responder(w, estadoRecibo(recibo.Reutilizada), salida)
}

func responderErrorDisposicion(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, puertosbolsa.ErrPortalOfertaNoAbierta):
		responder(w, 409, errorRespuesta{"oferta_no_abierta"})
	case errors.Is(e, puertosbolsa.ErrPortalDisposicionYaManifestada):
		responder(w, 409, errorRespuesta{"disposicion_ya_manifestada"})
	default:
		responderErrorPortal(w, e)
	}
}

type ofertaPortal struct {
	Oferta       string  `json:"oferta"`
	Bolsa        string  `json:"bolsa"`
	Categoria    string  `json:"categoria"`
	Centro       string  `json:"centro"`
	FechaInicio  string  `json:"fecha_inicio"`
	FechaFin     *string `json:"fecha_fin"`
	Descripcion  string  `json:"descripcion"`
	PublicadaEn  string  `json:"publicada_en"`
	VenceAntesDe string  `json:"vence_antes_de"`
	Estado       string  `json:"estado"`
	Disposicion  *struct {
		Recibo        string `json:"recibo"`
		ManifestadaEn string `json:"manifestada_en"`
	} `json:"disposicion"`
}

func respuestaOfertas(i puertosbolsa.InstantaneaMiBolsa) []ofertaPortal {
	if i.Ofertas == nil {
		return nil
	}
	salida := make([]ofertaPortal, 0, len(i.Ofertas))
	for _, o := range i.Ofertas {
		x := ofertaPortal{
			Oferta: o.OfertaRef, Bolsa: o.Bolsa, Categoria: o.Datos.Categoria, Centro: o.Datos.Centro,
			FechaInicio: o.Datos.FechaInicio, Descripcion: o.Datos.Descripcion,
			PublicadaEn: o.PublicadaEn.UTC().Format(formatoInstantePortal), VenceAntesDe: o.VenceAntesDe.UTC().Format(formatoInstantePortal),
			Estado: o.Estado,
		}
		if o.Datos.FechaFin != "" {
			fin := o.Datos.FechaFin
			x.FechaFin = &fin
		}
		if d := o.Disposicion; d != nil {
			x.Disposicion = &struct {
				Recibo        string `json:"recibo"`
				ManifestadaEn string `json:"manifestada_en"`
			}{Recibo: d.Recibo, ManifestadaEn: d.ManifestadaEn.UTC().Format(formatoInstantePortal)}
		}
		salida = append(salida, x)
	}
	return salida
}
