package conservacion

import (
	"context"
	"strings"
	"testing"
	"time"

	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojFijo struct{ t time.Time }

func (r relojFijo) Ahora() time.Time { return r.t }

func TestCatalogoProvisionalResuelveUnaPoliticaExactaPorTipo(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	c, err := NuevoCatalogoProvisional(relojFijo{ahora})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Provisional() {
		t.Fatal("el catálogo v1 debe declararse provisional")
	}
	expediente := "ref:" + strings.Repeat("ab", 32)
	s, err := c.SolicitudPara("dietas.comision.borrador.v1", expediente)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := vecapp.ResolverPoliticaConservacionDocumental(context.Background(), c, relojFijo{ahora}, s)
	if err != nil {
		t.Fatal(err)
	}
	p := resultado.Politica()
	if !p.ConservacionHasta().Equal(ahora.AddDate(6, 0, 0)) || p.Proteccion() != ports.ProteccionPoliticaConservacionDocumentalOrdinaria ||
		p.Solicitud().ExpedienteRef() != expediente {
		t.Fatalf("política inesperada: %v %v", p.ConservacionHasta(), p.Proteccion())
	}
	// Tipos distintos: referencias y huellas distintas.
	otra, _ := c.SolicitudPara("dietas.justificante.v1", expediente)
	if otra.TipoDocumentalRef() == s.TipoDocumentalRef() || otra.PoliticaRef() == s.PoliticaRef() ||
		string(otra.HuellaPoliticaSHA256()) == string(s.HuellaPoliticaSHA256()) {
		t.Fatal("tipos distintos comparten referencias")
	}
	if _, err := c.SolicitudPara("tipo.desconocido.v1", expediente); err == nil {
		t.Fatal("tipo no catalogado aceptado")
	}
	// Una solicitud con otra versión o huella no coincide: sin política.
	manipulada, err := ports.NuevaSolicitudPoliticaConservacionDocumental(s.ProcedimientoRef(), s.SerieDocumentalRef(),
		s.TipoDocumentalRef(), expediente, s.PoliticaRef(), 2, s.HuellaPoliticaSHA256(), s.BaseJuridicaRef(),
		s.VigenteDesde(), s.VigenteHasta())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vecapp.ResolverPoliticaConservacionDocumental(context.Background(), c, relojFijo{ahora}, manipulada); err == nil {
		t.Fatal("versión distinta resuelta")
	}
	// Fuera de vigencia del catálogo: no resuelve.
	tarde := relojFijo{time.Date(2031, 1, 2, 0, 0, 0, 0, time.UTC)}
	if _, err := vecapp.ResolverPoliticaConservacionDocumental(context.Background(), c, tarde, s); err == nil {
		t.Fatal("política fuera de vigencia resuelta")
	}
}

func TestCatalogoRechazaEntradasNoCerradas(t *testing.T) {
	reloj := relojFijo{time.Now().UTC()}
	base := `{"catalogo":"vec.documentos.conservacion","version":1,"provisional":true,"rotulo":"r","vigente_desde":"2026-01-01T00:00:00Z","vigente_hasta":"2027-01-01T00:00:00Z","politicas":[%s]}`
	for nombre, politicas := range map[string]string{
		"bloqueo":        `{"tipo":"a.b.v1","procedimiento":"p.x","serie":"s.x","base_juridica":"b.x","plazo_anios":1,"proteccion":"bloqueo"}`,
		"plazo cero":     `{"tipo":"a.b.v1","procedimiento":"p.x","serie":"s.x","base_juridica":"b.x","plazo_anios":0,"proteccion":"conservacion"}`,
		"campo libre":    `{"tipo":"a.b.v1","procedimiento":"p.x","serie":"s.x","base_juridica":"b.x","plazo_anios":1,"proteccion":"conservacion","nota":"x"}`,
		"tipo repetido":  `{"tipo":"a.b.v1","procedimiento":"p.x","serie":"s.x","base_juridica":"b.x","plazo_anios":1,"proteccion":"conservacion"},{"tipo":"a.b.v1","procedimiento":"p.x","serie":"s.x","base_juridica":"b.x","plazo_anios":2,"proteccion":"conservacion"}`,
		"clave con ruta": `{"tipo":"../a","procedimiento":"p.x","serie":"s.x","base_juridica":"b.x","plazo_anios":1,"proteccion":"conservacion"}`,
	} {
		raw := strings.Replace(base, "%s", politicas, 1)
		if _, err := NuevoCatalogo([]byte(raw), reloj); err == nil {
			t.Fatalf("%s: catálogo aceptado", nombre)
		}
	}
	if _, err := NuevoCatalogo(catalogoV1, nil); err == nil {
		t.Fatal("sin reloj aceptado")
	}
}
