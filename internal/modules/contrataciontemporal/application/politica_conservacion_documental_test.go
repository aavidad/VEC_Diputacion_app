package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type resolutorPoliticaConservacionCTPrueba struct {
	politicas []puertosvec.PoliticaConservacionDocumental
	err       error
	llamadas  int
	recibida  puertosvec.SolicitudPoliticaConservacionDocumental
	alBuscar  func()
}

func (r *resolutorPoliticaConservacionCTPrueba) BuscarPoliticasConservacionDocumental(
	_ context.Context,
	solicitud puertosvec.SolicitudPoliticaConservacionDocumental,
) ([]puertosvec.PoliticaConservacionDocumental, error) {
	r.llamadas++
	r.recibida = solicitud
	if r.alBuscar != nil {
		r.alBuscar()
	}
	return append([]puertosvec.PoliticaConservacionDocumental(nil), r.politicas...), r.err
}

type relojPoliticaConservacionCTPrueba struct {
	ahora    time.Time
	llamadas int
}

func (r *relojPoliticaConservacionCTPrueba) Ahora() time.Time {
	r.llamadas++
	return r.ahora
}

type relojPoliticaConservacionCTNuloPrueba struct{}

func (*relojPoliticaConservacionCTNuloPrueba) Ahora() time.Time {
	panic("el reloj nulo tipado no debe invocarse")
}

func referenciaPoliticaConservacionCTPrueba(caracter byte) string {
	return "ref:" + strings.Repeat(string(caracter), 64)
}

func solicitudPoliticaConservacionCTPrueba(
	t *testing.T,
) puertosvec.SolicitudPoliticaConservacionDocumental {
	t.Helper()
	huella := sha256.Sum256([]byte("politica sintetica de conservacion CT-LITE-O8-03B"))
	solicitud, err := puertosvec.NuevaSolicitudPoliticaConservacionDocumental(
		referenciaPoliticaConservacionCTPrueba('a'),
		referenciaPoliticaConservacionCTPrueba('b'),
		referenciaPoliticaConservacionCTPrueba('c'),
		referenciaPoliticaConservacionCTPrueba('d'),
		referenciaPoliticaConservacionCTPrueba('e'),
		3,
		huella[:],
		referenciaPoliticaConservacionCTPrueba('f'),
		time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2031, 8, 31, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("crear solicitud sintetica: %v", err)
	}
	return solicitud
}

func politicaConservacionCTPrueba(
	t *testing.T,
	solicitud puertosvec.SolicitudPoliticaConservacionDocumental,
	proteccion puertosvec.ProteccionPoliticaConservacionDocumental,
	estado puertosvec.EstadoPoliticaConservacionDocumental,
) puertosvec.PoliticaConservacionDocumental {
	t.Helper()
	bloqueoRef := ""
	if proteccion == puertosvec.ProteccionPoliticaConservacionDocumentalBloqueada {
		bloqueoRef = referenciaPoliticaConservacionCTPrueba('1')
	}
	retiradaEn := time.Time{}
	if estado == puertosvec.EstadoPoliticaConservacionDocumentalRetirada {
		retiradaEn = time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	politica, err := puertosvec.NuevaPoliticaConservacionDocumental(
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

func TestConsumoPoliticaConservacionDocumentalDelegaEnVECYConservaBloqueo(
	t *testing.T,
) {
	solicitud := solicitudPoliticaConservacionCTPrueba(t)
	politica := politicaConservacionCTPrueba(
		t,
		solicitud,
		puertosvec.ProteccionPoliticaConservacionDocumentalBloqueada,
		puertosvec.EstadoPoliticaConservacionDocumentalAprobada,
	)
	resolutor := &resolutorPoliticaConservacionCTPrueba{
		politicas: []puertosvec.PoliticaConservacionDocumental{politica},
	}
	ahora := time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC)
	reloj := &relojPoliticaConservacionCTPrueba{ahora: ahora}
	servicio, err := NuevoServicioConsultaPoliticaConservacionDocumental(resolutor, reloj)
	if err != nil {
		t.Fatalf("crear consumidor CT: %v", err)
	}

	resultado, err := servicio.Obtener(context.Background(), solicitud)
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("consumir politica exacta: resultado=%v error=%v", resultado, err)
	}
	obtenida := resultado.Politica()
	if resolutor.llamadas != 1 || reloj.llamadas != 1 ||
		!solicitudesPoliticaConservacionCTCoinciden(resolutor.recibida, solicitud) ||
		!resultado.ResueltaEn().Equal(ahora) ||
		obtenida.Proteccion() != puertosvec.ProteccionPoliticaConservacionDocumentalBloqueada ||
		obtenida.BloqueoRef() != referenciaPoliticaConservacionCTPrueba('1') ||
		!obtenida.ConservacionHasta().Equal(time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("CT altero el resultado gobernado por VEC: %#v", resultado)
	}
}

func solicitudesPoliticaConservacionCTCoinciden(
	primera puertosvec.SolicitudPoliticaConservacionDocumental,
	segunda puertosvec.SolicitudPoliticaConservacionDocumental,
) bool {
	return primera.ProcedimientoRef() == segunda.ProcedimientoRef() &&
		primera.SerieDocumentalRef() == segunda.SerieDocumentalRef() &&
		primera.TipoDocumentalRef() == segunda.TipoDocumentalRef() &&
		primera.ExpedienteRef() == segunda.ExpedienteRef() &&
		primera.PoliticaRef() == segunda.PoliticaRef() &&
		primera.VersionPolitica() == segunda.VersionPolitica() &&
		bytes.Equal(primera.HuellaPoliticaSHA256(), segunda.HuellaPoliticaSHA256()) &&
		primera.BaseJuridicaRef() == segunda.BaseJuridicaRef() &&
		primera.VigenteDesde().Equal(segunda.VigenteDesde()) &&
		primera.VigenteHasta().Equal(segunda.VigenteHasta())
}

func TestConsumoPoliticaConservacionDocumentalFallaCerradoSinSeleccionLocal(
	t *testing.T,
) {
	solicitud := solicitudPoliticaConservacionCTPrueba(t)
	aprobada := politicaConservacionCTPrueba(
		t,
		solicitud,
		puertosvec.ProteccionPoliticaConservacionDocumentalOrdinaria,
		puertosvec.EstadoPoliticaConservacionDocumentalAprobada,
	)
	retirada := politicaConservacionCTPrueba(
		t,
		solicitud,
		puertosvec.ProteccionPoliticaConservacionDocumentalOrdinaria,
		puertosvec.EstadoPoliticaConservacionDocumentalRetirada,
	)
	casos := []struct {
		nombre    string
		politicas []puertosvec.PoliticaConservacionDocumental
		err       error
		ahora     time.Time
	}{
		{
			nombre: "ausente",
			ahora:  time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			nombre:    "ambigua",
			politicas: []puertosvec.PoliticaConservacionDocumental{aprobada, aprobada},
			ahora:     time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			nombre:    "retirada",
			politicas: []puertosvec.PoliticaConservacionDocumental{retirada},
			ahora:     time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			nombre:    "fuera de vigencia",
			politicas: []puertosvec.PoliticaConservacionDocumental{aprobada},
			ahora:     solicitud.VigenteHasta(),
		},
		{
			nombre:    "resultado parcial con error privado",
			politicas: []puertosvec.PoliticaConservacionDocumental{aprobada},
			err:       errors.New("ruta privada del catalogo documental"),
			ahora:     time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resolutor := &resolutorPoliticaConservacionCTPrueba{
				politicas: caso.politicas,
				err:       caso.err,
			}
			servicio, err := NuevoServicioConsultaPoliticaConservacionDocumental(
				resolutor,
				&relojPoliticaConservacionCTPrueba{ahora: caso.ahora},
			)
			if err != nil {
				t.Fatalf("crear consumidor CT: %v", err)
			}
			resultado, err := servicio.Obtener(context.Background(), solicitud)
			if resultado != (puertosvec.ResultadoPoliticaConservacionDocumental{}) ||
				!errors.Is(err, puertosvec.ErrPoliticaConservacionDocumentalNoResuelta) ||
				strings.Contains(err.Error(), "privada") {
				t.Fatalf("resolucion no aplicable expuesta: resultado=%v error=%v", resultado, err)
			}
		})
	}
}

func TestConsumoPoliticaConservacionDocumentalFallaCerradoAnteContextoYSolicitud(
	t *testing.T,
) {
	solicitud := solicitudPoliticaConservacionCTPrueba(t)
	politica := politicaConservacionCTPrueba(
		t,
		solicitud,
		puertosvec.ProteccionPoliticaConservacionDocumentalOrdinaria,
		puertosvec.EstadoPoliticaConservacionDocumentalAprobada,
	)
	nuevoServicio := func(t *testing.T) (
		*ServicioConsultaPoliticaConservacionDocumental,
		*resolutorPoliticaConservacionCTPrueba,
	) {
		t.Helper()
		resolutor := &resolutorPoliticaConservacionCTPrueba{
			politicas: []puertosvec.PoliticaConservacionDocumental{politica},
		}
		servicio, err := NuevoServicioConsultaPoliticaConservacionDocumental(
			resolutor,
			&relojPoliticaConservacionCTPrueba{
				ahora: time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC),
			},
		)
		if err != nil {
			t.Fatalf("crear consumidor CT: %v", err)
		}
		return servicio, resolutor
	}

	servicio, resolutor := nuevoServicio(t)
	assertPoliticaConservacionCTNoResuelta(t, servicio, nil, solicitud)
	if resolutor.llamadas != 0 {
		t.Fatal("un contexto nulo alcanzo la dependencia")
	}

	servicio, resolutor = nuevoServicio(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	assertPoliticaConservacionCTNoResuelta(t, servicio, ctx, solicitud)
	if resolutor.llamadas != 0 {
		t.Fatal("un contexto cancelado alcanzo la dependencia")
	}

	servicio, resolutor = nuevoServicio(t)
	assertPoliticaConservacionCTNoResuelta(
		t,
		servicio,
		context.Background(),
		puertosvec.SolicitudPoliticaConservacionDocumental{},
	)
	if resolutor.llamadas != 0 {
		t.Fatal("una solicitud cero alcanzo la dependencia")
	}

	var servicioNulo *ServicioConsultaPoliticaConservacionDocumental
	assertPoliticaConservacionCTNoResuelta(
		t,
		servicioNulo,
		context.Background(),
		solicitud,
	)
}

func TestConsumoPoliticaConservacionDocumentalFallaCerradoAnteDependenciasNulas(
	t *testing.T,
) {
	var resolutorNulo *resolutorPoliticaConservacionCTPrueba
	var relojNulo *relojPoliticaConservacionCTNuloPrueba
	resolutorValido := &resolutorPoliticaConservacionCTPrueba{}
	relojValido := &relojPoliticaConservacionCTPrueba{}
	casos := []struct {
		nombre    string
		resolutor puertosvec.ResolutorPoliticaConservacionDocumental
		reloj     puertosvec.Reloj
	}{
		{"resolutor nulo", nil, relojValido},
		{"resolutor nulo tipado", resolutorNulo, relojValido},
		{"reloj nulo", resolutorValido, nil},
		{"reloj nulo tipado", resolutorValido, relojNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := NuevoServicioConsultaPoliticaConservacionDocumental(
				caso.resolutor,
				caso.reloj,
			)
			if servicio != nil ||
				!errors.Is(err, puertosvec.ErrPoliticaConservacionDocumentalNoResuelta) {
				t.Fatalf("dependencia nula aceptada: servicio=%v error=%v", servicio, err)
			}
		})
	}
}

func TestConsumoPoliticaConservacionDocumentalCancelaDuranteResolucion(
	t *testing.T,
) {
	solicitud := solicitudPoliticaConservacionCTPrueba(t)
	politica := politicaConservacionCTPrueba(
		t,
		solicitud,
		puertosvec.ProteccionPoliticaConservacionDocumentalOrdinaria,
		puertosvec.EstadoPoliticaConservacionDocumentalAprobada,
	)
	ctx, cancelar := context.WithCancel(context.Background())
	resolutor := &resolutorPoliticaConservacionCTPrueba{
		politicas: []puertosvec.PoliticaConservacionDocumental{politica},
		alBuscar:  cancelar,
	}
	reloj := &relojPoliticaConservacionCTPrueba{
		ahora: time.Date(2028, 4, 15, 10, 30, 0, 0, time.UTC),
	}
	servicio, err := NuevoServicioConsultaPoliticaConservacionDocumental(resolutor, reloj)
	if err != nil {
		t.Fatalf("crear consumidor CT: %v", err)
	}
	assertPoliticaConservacionCTNoResuelta(t, servicio, ctx, solicitud)
	if resolutor.llamadas != 1 || reloj.llamadas != 0 {
		t.Fatalf(
			"la cancelacion durante la frontera continuo: resolutor=%d reloj=%d",
			resolutor.llamadas,
			reloj.llamadas,
		)
	}
}

func assertPoliticaConservacionCTNoResuelta(
	t *testing.T,
	servicio *ServicioConsultaPoliticaConservacionDocumental,
	ctx context.Context,
	solicitud puertosvec.SolicitudPoliticaConservacionDocumental,
) {
	t.Helper()
	resultado, err := servicio.Obtener(ctx, solicitud)
	if resultado != (puertosvec.ResultadoPoliticaConservacionDocumental{}) ||
		!errors.Is(err, puertosvec.ErrPoliticaConservacionDocumentalNoResuelta) {
		t.Fatalf("fallo abierto: resultado=%v error=%v", resultado, err)
	}
}
