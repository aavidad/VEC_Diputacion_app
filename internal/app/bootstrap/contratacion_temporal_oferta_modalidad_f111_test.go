package bootstrap

import (
	"encoding/json"
	"testing"
	"time"

	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestF111ComandoCongelaModalidadYConservaLegado(t *testing.T) {
	i, antiguo := datosResolucionOfertaCTPrueba(t)
	antes, err := json.Marshal(antiguo)
	if err != nil {
		t.Fatal(err)
	}
	i.ModalidadClave, i.CausaClave = "interinidad", "vacante"
	if err := antiguo.ValidarPara(i); err != nil {
		t.Fatalf("la captura nueva impide recuperar el comando histórico: %v", err)
	}
	despues, err := json.Marshal(antiguo)
	if err != nil || string(antes) != string(despues) {
		t.Fatal("se alteró el comando histórico al recuperar")
	}
	nuevo := antiguo
	nuevo.Datos.ModalidadClave, nuevo.Datos.CausaClave = i.ModalidadClave, i.CausaClave
	if err := nuevo.ValidarPara(i); err != nil {
		t.Fatal(err)
	}
	nuevo.Datos.CausaClave = "programa"
	if err := nuevo.ValidarPara(i); err == nil {
		t.Fatal("se aceptó una causa distinta de la captura del análisis")
	}
}

func TestF111ReciboBolsaConservaModalidadDelComando(t *testing.T) {
	i, c := datosResolucionOfertaCTPrueba(t)
	i.ModalidadClave, i.CausaClave = "interinidad", "vacante"
	c.Datos.ModalidadClave, c.Datos.CausaClave = i.ModalidadClave, i.CausaClave
	instante := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	o := bolsa.OfertaPublicada{
		OfertaRef: "oferta:bolsa:1", ReciboRef: "recibo:oferta:1", BolsaRef: c.BolsaRef,
		NumeroPlazas: c.NumeroPlazas, PublicadaEn: instante, VenceAntesDe: instante.Add(48 * time.Hour),
		Datos: bolsadomain.DatosOferta{Categoria: c.Datos.Categoria, Centro: c.Datos.Centro,
			FechaInicio: c.Datos.FechaInicio, FechaFin: c.Datos.FechaFin, Descripcion: c.Datos.Descripcion,
			ModalidadClave: c.Datos.ModalidadClave, CausaClave: c.Datos.CausaClave},
	}
	r, err := proyectarOfertaBolsaSeleccionCT(c, o)
	if err != nil || r.Datos != c.Datos {
		t.Fatalf("la proyección perdió los datos del análisis: %v", err)
	}
	o.Datos.CausaClave = "programa"
	if _, err := proyectarOfertaBolsaSeleccionCT(c, o); err == nil {
		t.Fatal("se aceptó un recibo con causa distinta del comando")
	}
}
