package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repositorioOriginalPrueba struct {
	ports.Repositorio
	reservas  []ports.ReservaOriginalFirmable
	respuesta ports.IntentoOriginalFirmable
	err       error
}

func (r *repositorioOriginalPrueba) ReservarOriginalFirmable(_ context.Context, reserva ports.ReservaOriginalFirmable) (ports.IntentoOriginalFirmable, error) {
	r.reservas = append(r.reservas, reserva)
	if r.err != nil {
		return ports.IntentoOriginalFirmable{}, r.err
	}
	return r.respuesta, nil
}
func (r *repositorioOriginalPrueba) ConfirmarOriginalFirmable(context.Context, ports.ConfirmacionOriginalFirmable) (domain.Documento, error) {
	panic("confirmacion inesperada")
}

type almacenOriginalNoLlamado struct{ vecports.AlmacenObjetos }

func (almacenOriginalNoLlamado) Capacidades(context.Context) (vecports.CapacidadesAlmacenObjetos, error) {
	panic("almacen consultado")
}
func (almacenOriginalNoLlamado) Escribir(context.Context, vecports.SolicitudEscribirObjeto) (vecports.ResultadoOperacionObjeto, error) {
	panic("objeto escrito")
}

type autoridadOriginalPrueba struct {
	t        *testing.T
	ahora    time.Time
	llamadas int
}

func (a *autoridadOriginalPrueba) AutorizarReservaOriginal(_ context.Context, preimagen []byte, id, expediente string) (ports.AutorizacionV3, error) {
	a.llamadas++
	return ports.AutorizacionV3{Material: materialExternaPrueba(a.t, ports.AccionReservarOriginalFirmable, id, preimagen, a.ahora),
		Accion: ports.AccionReservarOriginalFirmable, Finalidad: ports.FinalidadOriginalFirmable,
		RecursoRef: id, AmbitoRef: expediente, PrincipalID: "per:0001", PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001"}, nil
}
func (*autoridadOriginalPrueba) AutorizarConfirmacionOriginal(context.Context, []byte, string, string) (ports.AutorizacionV3, error) {
	panic("confirmacion autorizada antes del objeto")
}
func (*autoridadOriginalPrueba) ContextoEscrituraOriginal(context.Context, ports.ReservaOriginalFirmable, ports.IntentoOriginalFirmable) (vecports.ContextoOperacionAlmacen, error) {
	panic("concesion de almacen solicitada")
}

func escenarioOriginalPrueba(t *testing.T) (*Servicio, *repositorioOriginalPrueba, *autoridadOriginalPrueba, ports.OrdenCustodiarOriginalFirmable) {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	politica := politicaConservacionPrueba(t, time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC), vecports.ProteccionPoliticaConservacionDocumentalOrdinaria)
	s := politica.Solicitud()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	in := ports.OrdenCustodiarOriginalFirmable{ID: ref("8"), ClaveIdempotencia: ref("9"), ModuloID: "contrataciontemporal",
		ExpedienteRef: s.ExpedienteRef(), TipoRef: s.TipoDocumentalRef(), Version: 1,
		MIME: "application/pdf", Contenido: []byte("%PDF-1.7\nensayo"), SolicitudPolitica: s}
	suma := sha256.Sum256(in.Contenido)
	repo := &repositorioOriginalPrueba{respuesta: ports.IntentoOriginalFirmable{ReservaRef: ref("a"), Estado: "confirmado",
		DocumentoID: in.ID, HuellaSHA256: hex.EncodeToString(suma[:]), Numero: 1, ClaveAlmacenRef: ref("b")}}
	autoridad := &autoridadOriginalPrueba{t: t, ahora: ahora}
	servicio := &Servicio{Repositorio: repo, Almacen: almacenOriginalNoLlamado{}, Politicas: politicasExternaPrueba{politica}, Reloj: relojExternaPrueba{ahora}}
	return servicio, repo, autoridad, in
}

func TestOriginalFirmableReservaAntesDeTocarAlmacenYReplayConfirmado(t *testing.T) {
	s, repo, autoridad, in := escenarioOriginalPrueba(t)
	got, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad)
	if err != nil || got != repo.respuesta || len(repo.reservas) != 1 || autoridad.llamadas != 1 {
		t.Fatalf("replay confirmado: intento=%+v err=%v reservas=%d autorizaciones=%d", got, err, len(repo.reservas), autoridad.llamadas)
	}
	if repo.reservas[0].HuellaSHA256 != got.HuellaSHA256 || repo.reservas[0].Tamano != int64(len(in.Contenido)) {
		t.Fatal("la reserva no fijo bytes exactos")
	}
	fallo := errors.New("reserva no disponible")
	repo.err = fallo
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, fallo) {
		t.Fatalf("fallo de reserva: %v", err)
	}
	// Las llamadas al almacén lanzan panic: el fallo SQL no debe producir objeto.
}

func TestOriginalFirmableNoAdmitePDFAlteradoNiTipoDistintoConReplay(t *testing.T) {
	s, repo, autoridad, in := escenarioOriginalPrueba(t)
	in.Contenido = []byte("%PDF-1.7\notros bytes")
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, ports.ErrCapacidadNoDisponible) {
		t.Fatalf("replay de bytes distintos: %v", err)
	}
	in.Contenido = []byte("%PDF-1.7\nensayo")
	in.MIME = "text/plain"
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("tipo no PDF: %v", err)
	}
	if len(repo.reservas) != 1 || autoridad.llamadas != 1 {
		t.Fatalf("la segunda solicitud llego a SQL: reservas=%d autorizaciones=%d", len(repo.reservas), autoridad.llamadas)
	}
}
