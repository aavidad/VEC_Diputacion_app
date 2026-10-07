package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestConsultaCuadroSesionV2ConservaFamiliaYLigaCadaPagina(t *testing.T) {
	crear := func(cursor string, estados []domain.EstadoOperativo) ConsultaCuadroRRHHSesionV2 {
		t.Helper()
		c, err := NuevaConsultaCuadroRRHHSesionV2("2026/CT", "centro:desarrollo:001",
			"categoria:desarrollo:c2", estados,
			[]domain.ClaveFase{"solicitud", "fiscalizacion"}, 25, cursor)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	primera := crear("", []domain.EstadoOperativo{domain.EstadoEnCurso, domain.EstadoCompletado})
	const esperado = `{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v2","version":2,"texto":"2026/CT","centro_ref":"centro:desarrollo:001","categoria_ref":"categoria:desarrollo:c2","estados_clave":["completado","en_curso"],"fases_clave":["fiscalizacion","solicitud"],"limite":25,"cursor":""}`
	canon, err := primera.CanonConsulta()
	if err != nil || string(canon) != esperado {
		t.Fatalf("canon v2 distinto: %s %v", canon, err)
	}
	familia, err := primera.CanonFamilia()
	if err != nil || bytes.Contains(familia, []byte(`"cursor"`)) {
		t.Fatalf("familia incluye cursor: %s %v", familia, err)
	}
	cursor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, sha256.Size))
	segunda := crear(cursor, []domain.EstadoOperativo{domain.EstadoCompletado, domain.EstadoEnCurso})
	f1, _ := primera.HuellaFamilia()
	f2, _ := segunda.HuellaFamilia()
	h1, _ := primera.HuellaConsulta()
	h2, _ := segunda.HuellaConsulta()
	if f1 != f2 || h1 == h2 {
		t.Fatal("la familia/cursor no conserva su ligadura")
	}
	otra := crear(cursor, []domain.EstadoOperativo{domain.EstadoIncidencia})
	f3, _ := otra.HuellaFamilia()
	if f3 == f1 {
		t.Fatal("otro filtro reutilizó familia")
	}
	if _, err := json.Marshal(primera); !errors.Is(err, ErrMaterialConsultaRRHHSensible) {
		t.Fatal("consulta serializable")
	}
	if strings.Contains(fmt.Sprintf("%+v", primera), "centro:desarrollo:001") ||
		strings.Contains(primera.LogValue().String(), "centro:desarrollo:001") ||
		primera.LogValue().Kind() != slog.KindString {
		t.Fatal("una representación de bitácora expuso el filtro")
	}
}

func TestConsultaCuadroSesionV2RechazaFiltrosAmbiguos(t *testing.T) {
	casos := []struct {
		centro  string
		estados []domain.EstadoOperativo
		fases   []domain.ClaveFase
	}{
		{centro: "centro con espacio"},
		{estados: []domain.EstadoOperativo{domain.EstadoEnCurso, domain.EstadoEnCurso}},
		{estados: []domain.EstadoOperativo{"otro"}},
		{fases: []domain.ClaveFase{"solicitud", "solicitud"}},
	}
	for _, caso := range casos {
		if _, err := NuevaConsultaCuadroRRHHSesionV2("", caso.centro, "", caso.estados, caso.fases, 25, ""); err == nil {
			t.Fatal("se aceptó un filtro ambiguo")
		}
	}
}
