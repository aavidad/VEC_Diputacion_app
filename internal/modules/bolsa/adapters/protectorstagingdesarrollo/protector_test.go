package protectorstagingdesarrollo

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"
	postgres "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func TestProtectorCifraRecuperaYRechazaMutacion(t *testing.T) {
	clave := sha256.Sum256([]byte("material-sintetico-prueba"))
	p, e := Nuevo(clave)
	if e != nil {
		t.Fatal(e)
	}
	h := strings.Repeat("a", 64)
	s := postgres.SolicitudProteccionStaging{ImportacionRef: "importacion:convoca:" + h, HuellaFicheroSHA256: h, Esquema: dominio.EsquemaResumenPersona, Filas: []dominio.FilaAceptada{{Numero: 2, Esquema: dominio.EsquemaResumenPersona, Identidad: dominio.IdentidadEnmascarada{Documento: "***0001**", PrimerApellido: "Prueba", Nombre: "Ana"}, Turno: "Libre", Resumen: &dominio.ResumenPersona{Experiencia: "1", Formacion: "1", Total: "2"}}}}
	r, e := p.ProtegerStaging(context.Background(), s)
	if e != nil {
		t.Fatal(e)
	}
	f := r.Filas[0]
	if f.ClaveRef == f.ClaveDerivacionRef || f.ClaveRef == f.ClaveAtestacionRef || f.ClaveDerivacionRef == f.ClaveAtestacionRef {
		t.Fatal("referencias no separadas")
	}
	got, e := p.RecuperarStaging(context.Background(), postgres.SolicitudRecuperacionStaging{ImportacionRef: s.ImportacionRef, HuellaFicheroSHA256: h, Esquema: s.Esquema, Filas: r.Filas})
	if e != nil || got[0].Identidad.Documento != "***0001**" {
		t.Fatalf("recuperacion=%#v err=%v", got, e)
	}
	r.Filas[0].ContenidoCifrado[0] ^= 1
	if _, e = p.RecuperarStaging(context.Background(), postgres.SolicitudRecuperacionStaging{ImportacionRef: s.ImportacionRef, HuellaFicheroSHA256: h, Esquema: s.Esquema, Filas: r.Filas}); e == nil {
		t.Fatal("acepto contenido alterado")
	}
}
