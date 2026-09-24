package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func notificacionValida() Notificacion {
	return Notificacion{EmpleadoRef: "emp_0123456789abcdefghijkl", TipoRef: "incidencia:v1", TipoVersion: 1, FechaReferida: "2026-09-24", Texto: "Olvidé el fichaje", AdjuntoRef: "doc_12345678", ClaveOperacion: "op_12345678", InstanteUTC: time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)}
}

func TestNotificacionValidaFechaCivilUTF8YReferenciaExterna(t *testing.T) {
	n := notificacionValida()
	n.Texto = strings.Repeat("á", 512)
	if err := n.Validar(); err != nil {
		t.Fatal(err)
	}
	casos := []func(*Notificacion){
		func(n *Notificacion) { n.TipoVersion = 0 },
		func(n *Notificacion) { n.Texto = strings.Repeat("á", 513) },
		func(n *Notificacion) { n.Texto = string([]byte{0xff}) },
		func(n *Notificacion) { n.Texto = "secreto\x00" },
		func(n *Notificacion) { n.FechaReferida = "2026-02-30" },
		func(n *Notificacion) { n.AdjuntoRef = "https://custodia/archivo" },
		func(n *Notificacion) { n.AdjuntoRef = "" },
	}
	for i, cambiar := range casos {
		copia := n
		cambiar(&copia)
		if !errors.Is(copia.Validar(), ErrNotificacionInvalida) {
			t.Fatalf("caso %d aceptado", i)
		}
	}
}

func TestHuellaSemanticaNotificacionConservaReintentoYDetectaCambio(t *testing.T) {
	n := notificacionValida()
	m := MaterialAutorizacionNotificacion{ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl", Notificacion: n}
	primera, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	m.Notificacion.InstanteUTC = m.Notificacion.InstanteUTC.Add(time.Hour)
	segunda, err := m.Canonico()
	if err != nil || !bytes.Equal(primera, segunda) {
		t.Fatal("reintento cambió el material")
	}
	m.Notificacion.Texto += "."
	tercera, err := m.Canonico()
	if err != nil || bytes.Equal(primera, tercera) {
		t.Fatal("contenido diferente comparte material")
	}
}
