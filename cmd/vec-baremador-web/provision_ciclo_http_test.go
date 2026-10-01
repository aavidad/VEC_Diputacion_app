package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func TestCicloLocalConservaHistoriaYBorrador(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	c, err := leerCatalogoEnsayos()
	if err != nil {
		t.Fatal(err)
	}
	var inicial string
	for _, caso := range c.Casos {
		t.Run(caso.CasoRef, func(t *testing.T) {
			b, err := json.Marshal(struct {
				EjemploRef string `json:"ejemplo_ref"`
				CasoRef    string `json:"caso_ref"`
			}{c.CicloRef, caso.CasoRef})
			if err != nil {
				t.Fatal(err)
			}
			w := request(h, http.MethodPost, rutaSimularCiclo, string(b), nil)
			var r domain.CicloEnsayado
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &r) != nil {
				t.Fatal(w.Code, w.Body.String())
			}
			if r.Resolucion.Firmada || r.Resolucion.Publicada || r.Resolucion.EfectoOficial || r.Resolucion.Estado != "borrador" {
				t.Fatal("ensayo afirma efecto oficial")
			}
			if len(r.Valoraciones) == 0 {
				t.Fatal("sin provisional")
			}
			if inicial == "" {
				inicial = r.Valoraciones[0].HuellaRevision
			}
			if r.Valoraciones[0].HuellaRevision != inicial {
				t.Fatal("decisión reescribe provisional")
			}
			if caso.Decision == "pendiente" {
				if len(r.Valoraciones) != 1 || len(r.Resolucion.ReclamacionesPendientes) != 1 {
					t.Fatal("reclamación pendiente cerrada")
				}
			} else if len(r.Valoraciones) != 2 || r.Valoraciones[1].HuellaAnterior != inicial {
				t.Fatal("revisión sin enlace causal")
			}
			repetida := request(h, http.MethodPost, rutaSimularCiclo, string(b), nil)
			if !bytes.Equal(w.Body.Bytes(), repetida.Body.Bytes()) {
				t.Fatal("ensayo no reproducible")
			}
		})
	}
	w := request(h, http.MethodPost, rutaSimularCiclo, `{"ejemplo_ref":"`+c.CicloRef+`","caso_ref":"mantener","entrada":{}}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatal("entrada del cliente admitida", w.Code)
	}
	w = request(h, http.MethodPost, rutaSimularCiclo, `{"ejemplo_ref":"`+c.CicloRef+`","caso_ref":"ajeno"}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatal("caso ajeno admitido", w.Code)
	}
}

func TestCicloLocalFrontera(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	if w := request(h, http.MethodGet, rutaCiclosLocales, "", nil); w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(h, http.MethodGet, rutaCiclosLocales+"?persona=ajena", "", nil); w.Code != http.StatusBadRequest {
		t.Fatal(w.Code)
	}
	if w := request(h, http.MethodPost, rutaSimularCiclo, "{}", func(r *http.Request) { r.Header.Del("Origin") }); w.Code != http.StatusForbidden {
		t.Fatal(w.Code)
	}
	if w := request(h, http.MethodGet, rutaSimularCiclo, "", nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatal(w.Code)
	}
}
