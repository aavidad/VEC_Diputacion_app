package application

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

type vinculosPoliticaConservacionDocumentalAplicacionPrueba struct {
	procedimientoRef   string
	serieDocumentalRef string
	tipoDocumentalRef  string
	expedienteRef      string
	politicaRef        string
	versionPolitica    uint64
	huellaPolitica     []byte
	baseJuridicaRef    string
	vigenteDesde       time.Time
	vigenteHasta       time.Time
}

func nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba() vinculosPoliticaConservacionDocumentalAplicacionPrueba {
	return vinculosPoliticaConservacionDocumentalAplicacionPrueba{
		procedimientoRef:   referenciaPoliticaConservacionDocumentalAplicacionPrueba('a'),
		serieDocumentalRef: referenciaPoliticaConservacionDocumentalAplicacionPrueba('b'),
		tipoDocumentalRef:  referenciaPoliticaConservacionDocumentalAplicacionPrueba('c'),
		expedienteRef:      referenciaPoliticaConservacionDocumentalAplicacionPrueba('d'),
		politicaRef:        referenciaPoliticaConservacionDocumentalAplicacionPrueba('e'),
		versionPolitica:    7,
		huellaPolitica:     bytes.Repeat([]byte{0x5a}, 32),
		baseJuridicaRef:    referenciaPoliticaConservacionDocumentalAplicacionPrueba('f'),
		vigenteDesde:       time.Date(2026, 9, 1, 0, 0, 0, 123_456_000, time.UTC),
		vigenteHasta:       time.Date(2030, 9, 1, 0, 0, 0, 123_456_000, time.UTC),
	}
}

func (v vinculosPoliticaConservacionDocumentalAplicacionPrueba) construir(
	t *testing.T,
) ports.SolicitudPoliticaConservacionDocumental {
	t.Helper()
	solicitud, err := ports.NuevaSolicitudPoliticaConservacionDocumental(
		v.procedimientoRef, v.serieDocumentalRef, v.tipoDocumentalRef, v.expedienteRef,
		v.politicaRef, v.versionPolitica, v.huellaPolitica, v.baseJuridicaRef,
		v.vigenteDesde, v.vigenteHasta,
	)
	if err != nil {
		t.Fatalf("crear solicitud sintetica: %v", err)
	}
	return solicitud
}

func nuevaPoliticaConservacionDocumentalAplicacionPrueba(
	t *testing.T,
	solicitud ports.SolicitudPoliticaConservacionDocumental,
	proteccion ports.ProteccionPoliticaConservacionDocumental,
	estado ports.EstadoPoliticaConservacionDocumental,
) ports.PoliticaConservacionDocumental {
	t.Helper()
	bloqueoRef := ""
	if proteccion == ports.ProteccionPoliticaConservacionDocumentalBloqueada {
		bloqueoRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
	}
	retiradaEn := time.Time{}
	if estado == ports.EstadoPoliticaConservacionDocumentalRetirada {
		retiradaEn = time.Date(2028, 3, 1, 0, 0, 0, 0, time.UTC)
	}
	politica, err := ports.NuevaPoliticaConservacionDocumental(
		solicitud,
		time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
		proteccion,
		bloqueoRef,
		estado,
		retiradaEn,
	)
	if err != nil {
		t.Fatalf("crear politica sintetica: %v", err)
	}
	return politica
}

func referenciaPoliticaConservacionDocumentalAplicacionPrueba(caracter byte) string {
	return "ref:" + strings.Repeat(string(caracter), 64)
}

type resolutorPoliticaConservacionDocumentalAplicacionPrueba struct {
	politicas []ports.PoliticaConservacionDocumental
	err       error
	llamadas  int
	alBuscar  func(context.Context, ports.SolicitudPoliticaConservacionDocumental)
}

func (r *resolutorPoliticaConservacionDocumentalAplicacionPrueba) BuscarPoliticasConservacionDocumental(
	ctx context.Context,
	solicitud ports.SolicitudPoliticaConservacionDocumental,
) ([]ports.PoliticaConservacionDocumental, error) {
	r.llamadas++
	if r.alBuscar != nil {
		r.alBuscar(ctx, solicitud)
	}
	return append([]ports.PoliticaConservacionDocumental(nil), r.politicas...), r.err
}

type relojPoliticaConservacionDocumentalAplicacionPrueba struct {
	ahora    time.Time
	llamadas int
}

func (r *relojPoliticaConservacionDocumentalAplicacionPrueba) Ahora() time.Time {
	r.llamadas++
	return r.ahora
}

type relojPoliticaConservacionDocumentalAplicacionNuloPrueba struct{}

func (*relojPoliticaConservacionDocumentalAplicacionNuloPrueba) Ahora() time.Time {
	panic("un reloj nulo tipado no debe invocarse")
}

func TestPoliticaConservacionDocumentalResolucionValidaYBloqueoPrevalece(t *testing.T) {
	vinculos := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba()
	solicitud := vinculos.construir(t)
	politica := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalBloqueada,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	resolutor := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
		politicas: []ports.PoliticaConservacionDocumental{politica},
	}
	ahora := time.Date(2028, 2, 1, 12, 0, 0, 0, time.UTC)
	reloj := &relojPoliticaConservacionDocumentalAplicacionPrueba{ahora: ahora}

	resultado, err := ResolverPoliticaConservacionDocumental(
		context.Background(), resolutor, reloj, solicitud,
	)
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("politica exacta denegada: resultado=%v error=%v", resultado, err)
	}
	obtenida := resultado.Politica()
	if resolutor.llamadas != 1 || reloj.llamadas != 1 || !resultado.ResueltaEn().Equal(ahora) ||
		obtenida.Proteccion() != ports.ProteccionPoliticaConservacionDocumentalBloqueada ||
		obtenida.BloqueoRef() != referenciaPoliticaConservacionDocumentalAplicacionPrueba('1') ||
		obtenida.Estado() != ports.EstadoPoliticaConservacionDocumentalAprobada ||
		!obtenida.ConservacionHasta().Equal(time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("resultado no conservo el bloqueo prevalente: %#v", resultado)
	}
	if obtenida.Solicitud().PoliticaRef() != vinculos.politicaRef ||
		obtenida.Solicitud().VersionPolitica() != vinculos.versionPolitica ||
		obtenida.Solicitud().BaseJuridicaRef() != vinculos.baseJuridicaRef {
		t.Fatal("resultado desligado de la publicacion aprobada")
	}
}

func TestPoliticaConservacionDocumentalDeniegaVariantesDeTodasLasLigaduras(t *testing.T) {
	base := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba()
	solicitudBase := base.construir(t)
	politicaBase := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitudBase, ports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	casos := []struct {
		nombre  string
		alterar func(*vinculosPoliticaConservacionDocumentalAplicacionPrueba)
	}{
		{"procedimiento", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.procedimientoRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
		}},
		{"serie", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.serieDocumentalRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
		}},
		{"tipo", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.tipoDocumentalRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
		}},
		{"expediente", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.expedienteRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
		}},
		{"politica", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.politicaRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
		}},
		{"version", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) { v.versionPolitica++ }},
		{"huella", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.huellaPolitica = bytes.Repeat([]byte{0x6b}, 32)
		}},
		{"base juridica", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.baseJuridicaRef = referenciaPoliticaConservacionDocumentalAplicacionPrueba('1')
		}},
		{"vigente desde", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.vigenteDesde = v.vigenteDesde.Add(-time.Hour)
		}},
		{"vigente hasta", func(v *vinculosPoliticaConservacionDocumentalAplicacionPrueba) {
			v.vigenteHasta = v.vigenteHasta.Add(time.Hour)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			vinculos := base
			vinculos.huellaPolitica = append([]byte(nil), base.huellaPolitica...)
			caso.alterar(&vinculos)
			solicitud := vinculos.construir(t)
			resolutor := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
				politicas: []ports.PoliticaConservacionDocumental{politicaBase},
			}
			resultado, err := ResolverPoliticaConservacionDocumental(
				context.Background(), resolutor,
				&relojPoliticaConservacionDocumentalAplicacionPrueba{
					ahora: time.Date(2028, 2, 1, 12, 0, 0, 0, time.UTC),
				},
				solicitud,
			)
			if resultado != (ports.ResultadoPoliticaConservacionDocumental{}) ||
				!errors.Is(err, ports.ErrPoliticaConservacionDocumentalNoResuelta) {
				t.Fatalf("ligadura distinta aceptada: resultado=%v error=%v", resultado, err)
			}
		})
	}
}

func TestPoliticaConservacionDocumentalDeniegaAusenciaAmbiguedadRetiradaYVencimiento(
	t *testing.T,
) {
	solicitud := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba().construir(t)
	aprobada := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	retirada := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		ports.EstadoPoliticaConservacionDocumentalRetirada,
	)
	casos := []struct {
		nombre    string
		politicas []ports.PoliticaConservacionDocumental
		ahora     time.Time
	}{
		{"no encontrada", nil, time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)},
		{"ambigua", []ports.PoliticaConservacionDocumental{aprobada, aprobada}, time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)},
		{"retirada", []ports.PoliticaConservacionDocumental{retirada}, time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)},
		{"aun no vigente", []ports.PoliticaConservacionDocumental{aprobada}, solicitud.VigenteDesde().Add(-time.Microsecond)},
		{"vencida en borde exclusivo", []ports.PoliticaConservacionDocumental{aprobada}, solicitud.VigenteHasta()},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := ResolverPoliticaConservacionDocumental(
				context.Background(),
				&resolutorPoliticaConservacionDocumentalAplicacionPrueba{politicas: caso.politicas},
				&relojPoliticaConservacionDocumentalAplicacionPrueba{ahora: caso.ahora},
				solicitud,
			)
			if resultado != (ports.ResultadoPoliticaConservacionDocumental{}) ||
				!errors.Is(err, ports.ErrPoliticaConservacionDocumentalNoResuelta) {
				t.Fatalf("estado no aplicable aceptado: resultado=%v error=%v", resultado, err)
			}
		})
	}
}

func TestPoliticaConservacionDocumentalDeniegaContextoCancelado(t *testing.T) {
	solicitud := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba().construir(t)
	politica := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	ahora := time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)

	t.Run("antes de resolver", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		resolutor := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
			politicas: []ports.PoliticaConservacionDocumental{politica},
		}
		resultado, err := ResolverPoliticaConservacionDocumental(
			ctx, resolutor, &relojPoliticaConservacionDocumentalAplicacionPrueba{ahora: ahora},
			solicitud,
		)
		if resultado != (ports.ResultadoPoliticaConservacionDocumental{}) ||
			!errors.Is(err, ports.ErrPoliticaConservacionDocumentalNoResuelta) ||
			resolutor.llamadas != 0 {
			t.Fatalf("cancelacion previa alcanzo el resolutor: resultado=%v error=%v", resultado, err)
		}
	})

	t.Run("durante la resolucion", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		resolutor := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
			politicas: []ports.PoliticaConservacionDocumental{politica},
			alBuscar: func(context.Context, ports.SolicitudPoliticaConservacionDocumental) {
				cancelar()
			},
		}
		reloj := &relojPoliticaConservacionDocumentalAplicacionPrueba{ahora: ahora}
		resultado, err := ResolverPoliticaConservacionDocumental(ctx, resolutor, reloj, solicitud)
		if resultado != (ports.ResultadoPoliticaConservacionDocumental{}) ||
			!errors.Is(err, ports.ErrPoliticaConservacionDocumentalNoResuelta) || reloj.llamadas != 0 {
			t.Fatalf("cancelacion tardia produjo politica: resultado=%v error=%v", resultado, err)
		}
	})
}

func TestPoliticaConservacionDocumentalDeniegaDependenciasNulasYTipadas(t *testing.T) {
	solicitud := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba().construir(t)
	politica := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	ahora := time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)
	resolutorValido := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
		politicas: []ports.PoliticaConservacionDocumental{politica},
	}
	var resolutorTipado *resolutorPoliticaConservacionDocumentalAplicacionPrueba
	var relojTipado *relojPoliticaConservacionDocumentalAplicacionNuloPrueba
	casos := []struct {
		nombre    string
		resolutor ports.ResolutorPoliticaConservacionDocumental
		reloj     ports.Reloj
	}{
		{"resolutor nulo", nil, &relojPoliticaConservacionDocumentalAplicacionPrueba{ahora: ahora}},
		{"resolutor nulo tipado", resolutorTipado, &relojPoliticaConservacionDocumentalAplicacionPrueba{ahora: ahora}},
		{"reloj nulo", resolutorValido, nil},
		{"reloj nulo tipado", resolutorValido, relojTipado},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := ResolverPoliticaConservacionDocumental(
				context.Background(), caso.resolutor, caso.reloj, solicitud,
			)
			if resultado != (ports.ResultadoPoliticaConservacionDocumental{}) ||
				!errors.Is(err, ports.ErrPoliticaConservacionDocumentalNoResuelta) {
				t.Fatalf("dependencia nula aceptada: resultado=%v error=%v", resultado, err)
			}
		})
	}
}

func TestPoliticaConservacionDocumentalDeniegaResultadoConErrorSinFiltrarlo(t *testing.T) {
	solicitud := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba().construir(t)
	politica := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalOrdinaria,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	resolutor := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
		politicas: []ports.PoliticaConservacionDocumental{politica},
		err:       errors.New("detalle privado del proveedor"),
	}
	resultado, err := ResolverPoliticaConservacionDocumental(
		context.Background(), resolutor,
		&relojPoliticaConservacionDocumentalAplicacionPrueba{
			ahora: time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		solicitud,
	)
	if resultado != (ports.ResultadoPoliticaConservacionDocumental{}) ||
		!errors.Is(err, ports.ErrPoliticaConservacionDocumentalNoResuelta) ||
		strings.Contains(err.Error(), "proveedor") {
		t.Fatalf("resultado+error no fallo cerrado: resultado=%v error=%v", resultado, err)
	}
}

func TestPoliticaConservacionDocumentalConservaCopiasDefensivasEnCoordinacion(t *testing.T) {
	vinculos := nuevosVinculosPoliticaConservacionDocumentalAplicacionPrueba()
	huellaOriginal := append([]byte(nil), vinculos.huellaPolitica...)
	solicitud := vinculos.construir(t)
	politica := nuevaPoliticaConservacionDocumentalAplicacionPrueba(
		t, solicitud, ports.ProteccionPoliticaConservacionDocumentalBloqueada,
		ports.EstadoPoliticaConservacionDocumentalAprobada,
	)
	resolutor := &resolutorPoliticaConservacionDocumentalAplicacionPrueba{
		politicas: []ports.PoliticaConservacionDocumental{politica},
		alBuscar: func(_ context.Context, recibida ports.SolicitudPoliticaConservacionDocumental) {
			copia := recibida.HuellaPoliticaSHA256()
			copia[0] ^= 0xff
		},
	}
	resultado, err := ResolverPoliticaConservacionDocumental(
		context.Background(), resolutor,
		&relojPoliticaConservacionDocumentalAplicacionPrueba{
			ahora: time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		solicitud,
	)
	if err != nil {
		t.Fatalf("resolver tras copia defensiva: %v", err)
	}
	primera := resultado.Politica().Solicitud().HuellaPoliticaSHA256()
	primera[1] ^= 0xff
	segunda := resultado.Politica().Solicitud().HuellaPoliticaSHA256()
	if !bytes.Equal(segunda, huellaOriginal) {
		t.Fatal("el resultado compartio la huella mutable")
	}
}
