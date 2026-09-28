package application

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type fuenteCalendarioContactosFalsa struct {
	f         puertosbolsa.FuenteCalendarioContactos
	err       error
	consultas int
}

func (f *fuenteCalendarioContactosFalsa) ObtenerPublicado(context.Context, string, string, int) (puertosbolsa.FuenteCalendarioContactos, error) {
	f.consultas++
	return f.f, f.err
}

type repoCalendarioContactosFalso struct {
	actual        dominiobolsa.CalendarioContactos
	existe        bool
	publicaciones int
	recibo        puertosbolsa.ReciboCalendarioContactos
}

func (r *repoCalendarioContactosFalso) VersionActual(context.Context, string, string, int) (dominiobolsa.CalendarioContactos, bool, error) {
	return r.actual, r.existe, nil
}

func (r *repoCalendarioContactosFalso) PublicarSiVersion(_ context.Context, orden puertosbolsa.OrdenPublicarCalendarioContactos) (puertosbolsa.ReciboCalendarioContactos, error) {
	r.publicaciones++
	if r.existe {
		reutilizada := r.recibo
		reutilizada.Reutilizado = true
		return reutilizada, nil
	}
	r.actual, r.existe = orden.Calendario, true
	r.recibo = puertosbolsa.ReciboCalendarioContactos{
		ReciboRef: orden.ReciboRef, Tipo: orden.Calendario.Tipo,
		SedeRef: orden.Calendario.SedeRef, Anio: orden.Calendario.Anio,
		Version: orden.Calendario.Version, HuellaSHA256: orden.Calendario.HuellaSHA256,
	}
	return r.recibo, nil
}

func materialEstructuralCalendarioPrueba(t *testing.T, accion, audiencia, efecto string) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	huella := strings.Repeat("a", 64)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:calendario", huella, huella, "contexto:calendario", huella,
		accion, efecto, huella, audiencia, ahora, ahora.Add(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	material, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	return material
}

func fuenteCalendarioPrueba(t *testing.T) puertosbolsa.FuenteCalendarioContactos {
	t.Helper()
	f := puertosbolsa.FuenteCalendarioContactos{
		Tipo: dominiobolsa.TipoCalendarioHabilSede, SedeRef: "municipio:18087", Anio: 2026,
		Fuentes: []dominiobolsa.VersionFuenteCalendario{
			{AmbitoTipo: "nacional", AmbitoRef: "es", VersionID: "calendario:nacional:2026:v1", Numero: 1},
			{AmbitoTipo: "autonomico", AmbitoRef: "andalucia", VersionID: "calendario:andalucia:2026:v1", Numero: 1},
			{AmbitoTipo: "local", AmbitoRef: "municipio:18087", VersionID: "calendario:granada:2026:v1", Numero: 1},
		},
	}
	for d := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); d.Year() == 2026; d = d.AddDate(0, 0, 1) {
		f.Dias = append(f.Dias, dominiobolsa.DiaCalendarioContactos{Fecha: d.Format("2006-01-02"), Habil: d.Weekday() != time.Saturday && d.Weekday() != time.Sunday})
	}
	c := dominiobolsa.CalendarioContactos{Esquema: dominiobolsa.EsquemaCalendarioContactos, Tipo: f.Tipo,
		SedeRef: f.SedeRef, Anio: f.Anio, Version: 1, Fuentes: f.Fuentes, Dias: f.Dias}
	var err error
	f.HuellaFuenteSHA256, err = c.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestPrepararFuenteCalendarioRechazaDatosSinPublicacionIntegra(t *testing.T) {
	f := fuenteCalendarioPrueba(t)
	s := SolicitudImportarCalendarioContactos{Tipo: f.Tipo, SedeRef: f.SedeRef, Anio: f.Anio}
	c, err := prepararFuenteCalendarioContactos(f, s)
	if err != nil || c.Validar() != nil {
		t.Fatalf("fuente integra = %v, %v", c.HuellaSHA256, err)
	}
	f.Dias[100].Habil = !f.Dias[100].Habil
	if _, err := prepararFuenteCalendarioContactos(f, s); !errors.Is(err, ErrCalendarioContactosNoDisponible) {
		t.Fatalf("fuente alterada = %v", err)
	}
	f = fuenteCalendarioPrueba(t)
	f.Dias = f.Dias[:len(f.Dias)-1]
	if _, err := prepararFuenteCalendarioContactos(f, s); !errors.Is(err, ErrCalendarioContactosNoDisponible) {
		t.Fatalf("año incompleto = %v", err)
	}
	f = fuenteCalendarioPrueba(t)
	f.SedeRef = "municipio:otro"
	if _, err := prepararFuenteCalendarioContactos(f, s); !errors.Is(err, ErrCalendarioContactosNoDisponible) {
		t.Fatalf("sede ajena = %v", err)
	}
}

func TestImportarCalendarioReutilizaReciboYFallaSinFuente(t *testing.T) {
	f := fuenteCalendarioPrueba(t)
	falso := &fuenteCalendarioContactosFalsa{f: f}
	repo := &repoCalendarioContactosFalso{}
	servicio, err := NuevoServicioCalendarioContactos(falso, repo)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudImportarCalendarioContactos{
		Tipo: f.Tipo, SedeRef: f.SedeRef, Anio: f.Anio,
		ActorRef: "per_0123456789abcdefghijkl", ClaveIdempotencia: "calendario-2026-001",
		ReciboRef: "recibo:calendario-contactos:" + strings.Repeat("b", 64),
		Material: materialEstructuralCalendarioPrueba(t,
			puertosbolsa.AccionEntregarCalendarioContactos,
			puertosbolsa.AudienciaEntregarCalendarioContactos,
			puertosbolsa.RecursoCalendarioContactos(f.SedeRef, f.Anio)),
	}
	primero, err := servicio.Importar(t.Context(), solicitud)
	if err != nil || primero.Reutilizado || repo.publicaciones != 1 {
		t.Fatalf("primera importacion = %+v, %v", primero, err)
	}
	segundo, err := servicio.Importar(t.Context(), solicitud)
	if err != nil || !segundo.Reutilizado || segundo.ReciboRef != primero.ReciboRef ||
		segundo.HuellaSHA256 != primero.HuellaSHA256 || segundo.Version != primero.Version || repo.publicaciones != 2 {
		t.Fatalf("replay = %+v, %v", segundo, err)
	}
	falso.err = errors.New("fuente ausente")
	if _, err := servicio.Importar(t.Context(), solicitud); !errors.Is(err, ErrCalendarioContactosNoDisponible) || repo.publicaciones != 2 {
		t.Fatalf("fuente ausente = %v", err)
	}
	falso.err = nil
	solicitud.Material = materialEstructuralCalendarioPrueba(t,
		puertosbolsa.AccionEntregarCalendarioContactos, "audiencia:ajena",
		puertosbolsa.RecursoCalendarioContactos(f.SedeRef, f.Anio))
	antes := falso.consultas
	if _, err := servicio.Importar(t.Context(), solicitud); !errors.Is(err, ErrCalendarioContactosNoDisponible) ||
		falso.consultas != antes || repo.publicaciones != 2 {
		t.Fatalf("audiencia ajena alcanzo fuente o repositorio: %v", err)
	}
}
