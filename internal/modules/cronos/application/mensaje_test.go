package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorComunicacionesVacio struct{ llamadas int }

func (p *proveedorComunicacionesVacio) ProveerArchivoMensaje(context.Context, ports.MaterialArchivoMensaje) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}
func (p *proveedorComunicacionesVacio) ProveerMensajeResolucion(context.Context, ports.MaterialMensajeResolucion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

type repoComunicacionesPrueba struct{ escrituras, lecturas int }

func (r *repoComunicacionesPrueba) RegistrarDesdeResolucionAutorizada(context.Context, ports.MaterialMensajeResolucion, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMensaje, error) {
	r.escrituras++
	return ports.ReciboMensaje{}, nil
}
func (r *repoComunicacionesPrueba) ArchivarAutorizado(context.Context, ports.MaterialArchivoMensaje, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMensaje, error) {
	r.escrituras++
	return ports.ReciboMensaje{}, nil
}
func (r *repoComunicacionesPrueba) ListarPropios(context.Context, vecdomain.ContextoActor, string, bool) ([]domain.MensajeResolucion, error) {
	r.lecturas++
	return nil, nil
}
func (r *repoComunicacionesPrueba) ConsultarPropio(context.Context, vecdomain.ContextoActor, string, string) (domain.MensajeResolucion, error) {
	r.lecturas++
	return domain.MensajeResolucion{}, nil
}
func (r *repoComunicacionesPrueba) HistoriaPropia(context.Context, vecdomain.ContextoActor, string, string) ([]ports.EventoMensaje, error) {
	r.lecturas++
	return nil, nil
}

func ordenComunicacionesPrueba(t *testing.T, proveedor *proveedorComunicacionesVacio) ports.OrdenComunicaciones {
	t.Helper()
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenComunicaciones(actor, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	return orden
}

func TestComunicacionesFallaCerradoSinV3NiContexto(t *testing.T) {
	repo := &repoComunicacionesPrueba{}
	proveedor := &proveedorComunicacionesVacio{}
	reloj := relojMarcajePrueba{time.Now().UTC()}
	mensajes, _ := NuevoServicioMensajes(repo, reloj)
	orden := ordenComunicacionesPrueba(t, proveedor)
	if _, err := mensajes.RegistrarMensajeResolucion(context.Background(), orden, ports.SolicitudMensajeResolucion{ResolucionRef: "res_12345678", ResolucionVersion: 1, ClaveOperacion: "op_12345678"}); !errors.Is(err, ports.ErrComunicacionNoAcreditada) {
		t.Fatal(err)
	}
	if _, err := mensajes.ArchivarMensaje(context.Background(), orden, domain.ArchivoMensaje{MensajeRef: "msg_12345678", VersionEsperada: 1, ClaveOperacion: "op_12345678"}); !errors.Is(err, ports.ErrComunicacionNoAcreditada) {
		t.Fatal(err)
	}
	if repo.escrituras != 0 || proveedor.llamadas != 2 {
		t.Fatal(repo.escrituras, proveedor.llamadas)
	}
	if _, err := mensajes.ConsultarMensaje(context.Background(), ports.OrdenComunicaciones{}, "msg_12345678"); !errors.Is(err, ports.ErrComunicacionNoAcreditada) || repo.lecturas != 0 {
		t.Fatal(err, repo.lecturas)
	}
}
