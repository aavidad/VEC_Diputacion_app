package rptpublica

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
)

func TestVerificadorAltaCompruebaPuestoYPublicacionExacta(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "..", "..", "data", "catalogos", "rpt", "v1.rpt-2026.json")
	fuente, err := personalrpt.NuevaFuente(ruta)
	if err != nil {
		t.Fatal(err)
	}
	c, err := fuente.ObtenerRPTPublica(context.Background())
	if err != nil || len(c.Puestos) == 0 {
		t.Fatal("RPT pública no disponible", err)
	}
	v, err := NuevoVerificadorAlta(fuente)
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudVerificarPuestoRPTAlta{
		CatalogoRef:          c.Fuente.Importacion,
		CatalogoHuellaSHA256: c.Fuente.HuellaSHA256, PuestoCodigo: c.Puestos[0].Codigo,
	}
	resultado, err := v.VerificarPuestoRPTAlta(context.Background(), s)
	if err != nil || resultado.ValidarPara(s) != nil {
		t.Fatal("puesto real rechazado", err)
	}
	s.PuestoCodigo = "99999999-NO-EXISTE"
	resultado, err = v.VerificarPuestoRPTAlta(context.Background(), s)
	if err != nil || resultado.ExisteEnPublicacion {
		t.Fatal("puesto ajeno admitido", err)
	}
	s.PuestoCodigo = c.Puestos[0].Codigo
	s.CatalogoHuellaSHA256 = strings.Repeat("a", 64)
	if _, err := v.VerificarPuestoRPTAlta(context.Background(), s); err == nil {
		t.Fatal("publicación alterada admitida")
	}
}
