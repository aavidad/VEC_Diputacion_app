package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestConsultaCuadroV2LigaFiltrosYCursorAlCanonV3(t *testing.T) {
	const esperado = `{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v2","version":2,"texto":"2026/CT","centro_ref":"centro:desarrollo:001","categoria_ref":"categoria:desarrollo:c2","estados_clave":["completado","en_curso"],"fases_clave":["fiscalizacion","solicitud"],"limite":25,"cursor":""}`
	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 32))
	crear := func(estados []domain.EstadoOperativo, fases []domain.ClaveFase, token string) SolicitudCuadroRRHH {
		t.Helper()
		s, err := NuevaSolicitudCuadroRRHHFiltrada("2026/CT", "centro:desarrollo:001",
			"categoria:desarrollo:c2", estados, fases, 25, token)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	primera := crear([]domain.EstadoOperativo{domain.EstadoEnCurso, domain.EstadoCompletado},
		[]domain.ClaveFase{"solicitud", "fiscalizacion"}, "")
	reordenada := crear([]domain.EstadoOperativo{domain.EstadoCompletado, domain.EstadoEnCurso},
		[]domain.ClaveFase{"fiscalizacion", "solicitud"}, "")
	segunda := crear([]domain.EstadoOperativo{domain.EstadoCompletado, domain.EstadoEnCurso},
		[]domain.ClaveFase{"fiscalizacion", "solicitud"}, cursor)
	canon, err := canonSolicitudCuadroRRHH(primera)
	if err != nil || string(canon.BytesCanonicos()) != esperado {
		t.Fatalf("canon v2 inesperado: %s, %v", canon.BytesCanonicos(), err)
	}
	suma := sha256.Sum256([]byte(esperado))
	if canon.HuellaSHA256() != hex.EncodeToString(suma[:]) {
		t.Fatal("la huella V3 no liga el canon exacto")
	}
	f1, _ := primera.FiltrosHuellaSHA256()
	f2, _ := segunda.FiltrosHuellaSHA256()
	h1, _ := primera.HuellaCanonicaSHA256()
	h2, _ := segunda.HuellaCanonicaSHA256()
	hr, _ := reordenada.HuellaCanonicaSHA256()
	if f1 != f2 || h1 == h2 || h1 != hr {
		t.Fatal("familia/cursor u orden de selecciones incorrectos")
	}
	if primera.dominioConsulta() != DominioHuellaConsultaCuadroRRHHV2 ||
		!primera.admiteEstado(domain.EstadoCompletado) || primera.admiteEstado(domain.EstadoIncidencia) ||
		!primera.admiteFase("solicitud") || primera.admiteFase("nombramiento") {
		t.Fatal("la validación local no corresponde al filtro v2")
	}
}

func TestConsultaCuadroV2RechazaDuplicadosYReferenciasNoOpacas(t *testing.T) {
	casos := []struct {
		centro, categoria string
		estados           []domain.EstadoOperativo
		fases             []domain.ClaveFase
	}{
		{centro: "centro con espacio"},
		{categoria: "categoria con espacio"},
		{estados: []domain.EstadoOperativo{domain.EstadoEnCurso, domain.EstadoEnCurso}},
		{fases: []domain.ClaveFase{"solicitud", "solicitud"}},
		{estados: []domain.EstadoOperativo{"inventado"}},
		{fases: []domain.ClaveFase{"fase inventada"}},
	}
	for _, c := range casos {
		if _, err := NuevaSolicitudCuadroRRHHFiltrada("", c.centro, c.categoria, c.estados, c.fases, 25, ""); err == nil {
			t.Fatalf("se admitió filtro inválido: %+v", c)
		}
	}
}
