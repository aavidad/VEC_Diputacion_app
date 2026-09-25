package application

import (
	"bytes"
	"strings"
	"testing"
	"time"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

func politicaConservacionPrueba(t *testing.T, hasta time.Time, proteccion vecports.ProteccionPoliticaConservacionDocumental) vecports.PoliticaConservacionDocumental {
	t.Helper()
	return politicaConservacionEstadoPrueba(t, hasta, proteccion, vecports.EstadoPoliticaConservacionDocumentalAprobada)
}

func politicaConservacionEstadoPrueba(t *testing.T, hasta time.Time, proteccion vecports.ProteccionPoliticaConservacionDocumental,
	estado vecports.EstadoPoliticaConservacionDocumental) vecports.PoliticaConservacionDocumental {
	t.Helper()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	s, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(
		ref("1"), ref("2"), ref("3"), ref("4"), ref("5"), 1, bytes.Repeat([]byte{0x6a}, 32), ref("6"),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("solicitud politica: %v", err)
	}
	bloqueo := ""
	if proteccion == vecports.ProteccionPoliticaConservacionDocumentalBloqueada {
		bloqueo = ref("7")
	}
	p, err := vecports.NuevaPoliticaConservacionDocumental(s, hasta, proteccion, bloqueo, estado, time.Time{})
	if err != nil {
		t.Fatalf("politica: %v", err)
	}
	return p
}

func TestConservacionDocumentalExigeMicrosegundosExactos(t *testing.T) {
	hasta := time.Date(2035, 1, 1, 0, 0, 0, 123456000, time.UTC)
	p := politicaConservacionPrueba(t, hasta, vecports.ProteccionPoliticaConservacionDocumentalOrdinaria)
	almacenada, err := time.Parse(time.RFC3339Nano, p.ConservacionHasta().Format(time.RFC3339Nano))
	if err != nil || !almacenada.Equal(p.ConservacionHasta()) {
		t.Fatalf("desfase microsegundos: %v", err)
	}
	// El contrato VEC-DOC-CONS-01 rechaza, sin redondear, precisión que
	// PostgreSQL timestamptz(6) no puede representar.
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	s, _ := vecports.NuevaSolicitudPoliticaConservacionDocumental(ref("1"), ref("2"), ref("3"), ref("4"), ref("5"), 1,
		bytes.Repeat([]byte{0x6a}, 32), ref("6"), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if _, err := vecports.NuevaPoliticaConservacionDocumental(s, hasta.Add(time.Nanosecond),
		vecports.ProteccionPoliticaConservacionDocumentalOrdinaria, "",
		vecports.EstadoPoliticaConservacionDocumentalAprobada, time.Time{}); err == nil {
		t.Fatal("admitio nanosegundo fuera de precision PostgreSQL")
	}
}

func TestAltaDeniegaCustodiaSinRetencionSuficienteOBloqueo(t *testing.T) {
	hasta := time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	p := politicaConservacionPrueba(t, hasta, vecports.ProteccionPoliticaConservacionDocumentalOrdinaria)
	cap := vecports.CapacidadesAlmacenObjetos{Retencion: true, BloqueoLegal: true}
	r := vecports.ResultadoOperacionObjeto{}
	r.Objeto.RetenidoHasta = hasta.Add(-time.Microsecond)
	if custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("admitio retencion insuficiente")
	}
	r.Objeto.RetenidoHasta = hasta
	if !custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("denego retencion exacta")
	}
	cap.Retencion = false
	if custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("admitio conector sin capacidad de retencion")
	}
	cap.Retencion = true
	bloqueada := politicaConservacionPrueba(t, hasta, vecports.ProteccionPoliticaConservacionDocumentalBloqueada)
	if custodiaSatisfacePolitica(r, cap, bloqueada) {
		t.Fatal("admitio bloqueo sin inmovilizacion")
	}
	r.Objeto.Inmovilizado = true
	if !custodiaSatisfacePolitica(r, cap, bloqueada) {
		t.Fatal("denego bloqueo inmovilizado")
	}
	cap.BloqueoLegal = false
	if custodiaSatisfacePolitica(r, cap, bloqueada) {
		t.Fatal("admitio conector sin capacidad de bloqueo")
	}
}

// Con política provisional el conector no debe haber fijado retención ni
// inmovilizado el objeto: ese efecto sería irreversible con plazos sin aprobar.
func TestAltaProvisionalExigeObjetoSinRetencionNiInmovilizacion(t *testing.T) {
	hasta := time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	p := politicaConservacionEstadoPrueba(t, hasta, vecports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		vecports.EstadoPoliticaConservacionDocumentalProvisional)
	cap := vecports.CapacidadesAlmacenObjetos{Retencion: true}
	r := vecports.ResultadoOperacionObjeto{}
	if !custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("denego objeto sin retencion con politica provisional")
	}
	r.Objeto.RetenidoHasta = hasta
	if custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("admitio retencion fijada con politica provisional")
	}
	r.Objeto.RetenidoHasta = time.Time{}
	r.Objeto.Inmovilizado = true
	if custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("admitio inmovilizacion con politica provisional")
	}
	r.Objeto.Inmovilizado = false
	cap.Retencion = false
	if custodiaSatisfacePolitica(r, cap, p) {
		t.Fatal("admitio conector que no podra fijar despues la retencion")
	}
}
