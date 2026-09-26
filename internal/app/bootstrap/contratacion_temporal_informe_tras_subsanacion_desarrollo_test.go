package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/vec/reglas"
)

// El catálogo de ejemplo recoge la duda 5: tras subsanar, informe nuevo y
// nueva firma del informe definitivo. Exigirlo sin CT123 impide arrancar.
func TestInformeTrasSubsanacionDesdeCatalogoEjemplo(t *testing.T) {
	resolutor, err := nuevoResolutorReglasEjemplo(rutaReglasCTEjemploPrueba, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := fuenteInformeTrasSubsanacionReglas{resolutor: resolutor}.InformeTrasSubsanacion(t.Context())
	if err != nil || !p.ExigeInformeNuevo || p.DocumentoFirma != "informe_definitivo" {
		t.Fatalf("regla c19: %+v %v", p, err)
	}
	_, err = gobernarInformeTrasSubsanacionDesarrollo(resolutor, nil, nil, &ctapplication.ServicioFiscalizaciones{}, &ctapplication.ServicioInformesJuridicos{})
	if !errors.Is(err, errInformeTrasSubsanacionSinCT123) {
		t.Fatalf("sin CT123 no se arranca: %v", err)
	}
	if fuente, err := gobernarInformeTrasSubsanacionDesarrollo(nil, nil, nil, nil, nil); fuente != nil || err != nil {
		t.Fatal("sin catálogo rige la conducta de siempre")
	}
}

func TestInformeTrasSubsanacionAtributosDeLaRegla(t *testing.T) {
	casos := []struct {
		atributos map[string]string
		exige     bool
		valida    bool
	}{
		{nil, false, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "no"}, false, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "si"}, true, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "si", atributoDocumentoFirmaInformeNuevo: "informe_definitivo"}, true, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "quizas"}, false, false},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "no", atributoDocumentoFirmaInformeNuevo: "informe_definitivo"}, false, false},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "si", atributoDocumentoFirmaInformeNuevo: "Informe Definitivo"}, false, false},
	}
	for _, c := range casos {
		p, err := politicaInformeTrasSubsanacionDesdeRegla(reglas.Regla{Atributos: c.atributos})
		if (err == nil) != c.valida || (c.valida && p.ExigeInformeNuevo != c.exige) {
			t.Errorf("%v: %+v %v", c.atributos, p, err)
		}
	}
}

// consultaPoliticaInformeNuevoPrueba responde a la detección de CT123 y a la
// publicación, en ese orden, y guarda los argumentos publicados.
type consultaPoliticaInformeNuevoPrueba struct {
	instalada  bool
	resultado  string
	fallo      error
	publicados *[]any
}

func (c consultaPoliticaInformeNuevoPrueba) QueryRow(_ context.Context, sql string, argumentos ...any) pgx.Row {
	publicar := strings.Contains(sql, "publicar_politica_informe_tras_subsanacion_v1($1")
	if publicar {
		*c.publicados = append(*c.publicados, argumentos...)
	}
	return filaPoliticaInformeNuevoPrueba{consulta: c, publicar: publicar}
}

type filaPoliticaInformeNuevoPrueba struct {
	consulta consultaPoliticaInformeNuevoPrueba
	publicar bool
}

func (f filaPoliticaInformeNuevoPrueba) Scan(destinos ...any) error {
	if !f.publicar {
		*(destinos[0].(*bool)) = f.consulta.instalada
		return nil
	}
	if f.consulta.fallo != nil {
		return f.consulta.fallo
	}
	*(destinos[0].(*string)) = f.consulta.resultado
	*(destinos[1].(*int64)) = 1
	return nil
}

// La política se publica en la base para que la nueva fiscalización la
// compruebe también en SQL; sin CT123 no exigir no publica nada.
func TestPublicacionPoliticaInformeTrasSubsanacion(t *testing.T) {
	for _, resultado := range []string{"publicada", "vigente"} {
		var publicados []any
		consulta := consultaPoliticaInformeNuevoPrueba{instalada: true, resultado: resultado, publicados: &publicados}
		if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), consulta, true, "fuente:c19"); err != nil ||
			len(publicados) != 2 || publicados[0] != true || publicados[1] != "fuente:c19" {
			t.Fatalf("%s: %v %v", resultado, err, publicados)
		}
	}
	var publicados []any
	sinCT123 := consultaPoliticaInformeNuevoPrueba{publicados: &publicados}
	if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), sinCT123, false, fuentePoliticaInformeNuevoPredeterminada); err != nil || len(publicados) != 0 {
		t.Fatalf("sin CT123 y sin exigir debía arrancar sin publicar: %v", err)
	}
	if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), sinCT123, true, "fuente:c19"); !errors.Is(err, errInformeTrasSubsanacionSinCT123) {
		t.Fatalf("exigir sin CT123 debía impedir arrancar: %v", err)
	}
	if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), nil, true, "fuente:c19"); !errors.Is(err, errInformeTrasSubsanacionSinCT123) {
		t.Fatalf("exigir sin base debía impedir arrancar: %v", err)
	}
	if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), nil, false, "fuente:c19"); err != nil {
		t.Fatalf("sin base ni exigencia rige la conducta de siempre: %v", err)
	}
	rechazada := consultaPoliticaInformeNuevoPrueba{instalada: true, fallo: errors.New("42501"), publicados: &publicados}
	if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), rechazada, false, "fuente:c19"); !errors.Is(err, errInformeTrasSubsanacionNoPublicada) {
		t.Fatalf("un rechazo de la publicación no detuvo el arranque: %v", err)
	}
	extraña := consultaPoliticaInformeNuevoPrueba{instalada: true, resultado: "otra", publicados: &publicados}
	if err := publicarPoliticaInformeTrasSubsanacionCT(t.Context(), extraña, true, "fuente:c19"); !errors.Is(err, errInformeTrasSubsanacionNoPublicada) {
		t.Fatalf("un resultado desconocido se dio por bueno: %v", err)
	}
	// Sin catálogo se publica «no exigir» con la fuente predeterminada.
	publicados = nil
	vigente := consultaPoliticaInformeNuevoPrueba{instalada: true, resultado: "vigente", publicados: &publicados}
	if fuente, err := gobernarInformeTrasSubsanacionDesarrollo(nil, nil, vigente, nil, nil); fuente != nil || err != nil ||
		len(publicados) != 2 || publicados[0] != false || publicados[1] != fuentePoliticaInformeNuevoPredeterminada {
		t.Fatalf("sin catálogo: %v %v", err, publicados)
	}
}
