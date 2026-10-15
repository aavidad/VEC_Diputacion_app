package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type operadorInicioOfertaPrueba struct {
	operadorOfertasPrueba
	inicio string
}

func (o operadorInicioOfertaPrueba) InicioPlazoOferta(context.Context, ports.SolicitudConsultarOfertas) (string, error) {
	return o.inicio, nil
}

func TestOfertaPorPublicacionNoInventaAntecedenteDeCorreo(t *testing.T) {
	p := &preparadorOfertasPrueba{}
	h, _ := NuevoHandlerOfertasPublicadas(p, operadorInicioOfertaPrueba{inicio: "publicacion"})
	cuerpo := strings.Replace(cuerpoOfertaPrueba, `"notificacion":{"notificada_en":"2026-10-02T09:00:00Z","referencia_correo":"correo:extracto:oferta-1","huella_correo_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fuente":"correo_externo_declarado_rrhh"},`, "", 1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpo, "clave-publicacion-1"))
	if w.Code != http.StatusCreated || p.publicar == nil || !p.publicar.Notificacion.NotificadaEn.IsZero() || p.publicar.Notificacion.Fuente != "" {
		t.Fatalf("estado=%d; antecedente inventado o publicación rechazada: %s", w.Code, w.Body.String())
	}
}

func TestConsultaOfertaExponeInicioGobernadoYRechazaModoDesconocido(t *testing.T) {
	for _, inicio := range []string{"publicacion", "notificacion", "desconocido"} {
		t.Run(inicio, func(t *testing.T) {
			h, _ := NuevoHandlerOfertasPublicadas(&preparadorOfertasPrueba{}, operadorInicioOfertaPrueba{inicio: inicio})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionOferta(http.MethodGet, RutaOfertasPublicadas+"?bolsa_ref=bolsa:1", "", ""))
			if inicio == "desconocido" {
				if w.Code != http.StatusServiceUnavailable {
					t.Fatalf("modo desconocido aceptado: %s", w.Body.String())
				}
				return
			}
			var sobre struct {
				Data struct {
					Inicio string `json:"inicio_plazo"`
				} `json:"data"`
			}
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &sobre) != nil || sobre.Data.Inicio != inicio {
				t.Fatalf("metadata divergente: %s", w.Body.String())
			}
		})
	}
}
