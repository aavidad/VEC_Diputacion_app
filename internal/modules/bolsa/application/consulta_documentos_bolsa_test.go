package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type lectorDocumentosBolsaPrueba struct {
	filas    []ports.ConstitucionVigente
	bolsa    string
	lecturas int
}

func (l *lectorDocumentosBolsaPrueba) ListarVigentesBolsa(_ context.Context, bolsa string) ([]ports.ConstitucionVigente, error) {
	l.lecturas++
	l.bolsa = bolsa
	return l.filas, nil
}

func TestConsultaDocumentosBolsaConservaRegistroYAcotaFuente(t *testing.T) {
	fecha := time.Date(2026, 10, 10, 12, 34, 56, 0, time.UTC)
	l := &lectorDocumentosBolsaPrueba{filas: []ports.ConstitucionVigente{{
		ActaRef: "acta:importacion-convoca:uno", ConfirmadaEn: fecha, TotalParticipaciones: 8,
	}}}
	l.filas[0].Bolsa.BolsaRef = "bolsa:uno"
	l.filas[0].Bolsa.Version = 2
	l.filas[0].Instantanea.InstantaneaRef = "instantanea:uno"
	l.filas[0].Instantanea.Version = 3
	l.filas[0].Instantanea.ListadoDefinitivoRef = "listado:convoca:uno"
	l.filas[0].Instantanea.VersionListado = 4
	l.filas[0].Instantanea.HuellaListadoSHA256 = strings.Repeat("a", 64)
	s, err := NuevoServicioDocumentosBolsa(l)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Consultar(context.Background(), ConsultaDocumentosBolsa{BolsaRef: "bolsa:uno", Tipo: "constitucion", Texto: "convoca", Limite: 10})
	if err != nil || l.lecturas != 1 || l.bolsa != "bolsa:uno" || p.Total != 1 || len(p.Documentos) != 1 || p.CursorSiguiente != nil {
		t.Fatalf("consulta focal: pagina=%+v lecturas=%d bolsa=%q error=%v", p, l.lecturas, l.bolsa, err)
	}
	d := p.Documentos[0]
	if d.DocumentoRef != l.filas[0].ActaRef || d.BolsaRef != "bolsa:uno" || d.VersionBolsa != 2 || d.InstantaneaRef != "instantanea:uno" || d.VersionInstantanea != 3 || d.ListadoDefinitivoRef != "listado:convoca:uno" || d.VersionListado != 4 || d.HuellaListadoSHA256 != strings.Repeat("a", 64) || !d.ConfirmadaEn.Equal(fecha) || d.TotalParticipaciones != 8 {
		t.Fatalf("registro alterado: %+v", d)
	}
	p, err = s.Consultar(context.Background(), ConsultaDocumentosBolsa{BolsaRef: "bolsa:uno", Cursor: d.DocumentoRef, Limite: 1})
	if err != nil || p.Total != 1 || len(p.Documentos) != 0 || p.CursorSiguiente != nil {
		t.Fatalf("cursor de misma bolsa: pagina=%+v error=%v", p, err)
	}
	if _, err = s.Consultar(context.Background(), ConsultaDocumentosBolsa{BolsaRef: "bolsa:uno", Cursor: "acta:ajena", Limite: 1}); !errors.Is(err, ErrConsultaDocumentosBolsaInvalida) {
		t.Fatalf("cursor ajeno: %v", err)
	}
}
