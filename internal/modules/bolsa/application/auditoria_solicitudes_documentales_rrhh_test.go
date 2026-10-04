package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type consultaDocumentalesAuditadaPrueba struct {
	items   []ports.SolicitudDocumentalPendienteRRHH
	err     error
	cerrada bool
	despues func()
}

func (c *consultaDocumentalesAuditadaPrueba) ListarSolicitudesDocumentalesRRHH(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	c.cerrada = true
	if c.despues != nil {
		c.despues()
	}
	return c.items, c.err
}

type registradorDocumentalesAuditadaPrueba struct {
	t             *testing.T
	consulta      *consultaDocumentalesAuditadaPrueba
	ordenes       []vecports.DatosOrdenIntentoAuditoria
	err           error
	acuseInvalido bool
}

func (r *registradorDocumentalesAuditadaPrueba) AppendIntentoAuditoria(ctx context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.t.Helper()
	if !r.consulta.cerrada || ctx.Err() != nil {
		r.t.Fatal("registro antes del cierre o cancelado por HTTP")
	}
	d, err := o.Datos()
	if err != nil {
		r.t.Fatal(err)
	}
	r.ordenes = append(r.ordenes, d)
	if r.err != nil || r.acuseInvalido {
		return vecports.AcuseIntentoAuditoria{}, r.err
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "aud_documental_sintetica", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC)}, nil
}

func itemDocumentalAuditadoPrueba() ports.SolicitudDocumentalPendienteRRHH {
	return ports.SolicitudDocumentalPendienteRRHH{SolicitudRef: "solicitud:documental:prueba", Version: 1,
		ContenidoSHA256: strings.Repeat("b", 64), DocumentoRef: "documento:prueba", DocumentoSHA256: strings.Repeat("c", 64),
		FechaFinCausa: "2026-10-04", Estado: "pendiente_rrhh", ReciboRef: "recibo:prueba", RegistradaEn: time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC)}
}

func TestConsultaDocumentalesAuditadaConservaPositivoYAusencia(t *testing.T) {
	q := solicitudSituacionPrueba(t, time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC))
	for _, items := range [][]ports.SolicitudDocumentalPendienteRRHH{nil, {}, {itemDocumentalAuditadoPrueba()}} {
		c := &consultaDocumentalesAuditadaPrueba{items: items}
		r := &registradorDocumentalesAuditadaPrueba{t: t, consulta: c}
		s, err := NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
		if err != nil {
			t.Fatal(err)
		}
		resultado, err := s.ListarSolicitudesDocumentalesRRHH(context.Background(), q)
		if err != nil || !reflect.DeepEqual(resultado, items) || len(r.ordenes) != 0 {
			t.Fatal("positivo o ausencia alterados")
		}
	}
}

func TestConsultaDocumentalesAuditadaRegistraFalloTrasRetorno(t *testing.T) {
	q := solicitudSituacionPrueba(t, time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC))
	for _, caso := range []struct {
		causa     error
		resultado core.ResultadoIntentoAuditoria
	}{
		{core.ErrAutorizacionDenegada, core.ResultadoIntentoAuditoriaDenegado},
		{ports.ErrSituacionParticipacionNoDisponible, core.ResultadoIntentoAuditoriaError},
		{errors.Join(core.ErrAutorizacionDenegada, ports.ErrSituacionParticipacionNoDisponible), core.ResultadoIntentoAuditoriaError},
	} {
		c := &consultaDocumentalesAuditadaPrueba{err: caso.causa}
		r := &registradorDocumentalesAuditadaPrueba{t: t, consulta: c}
		s, err := NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
		if err != nil {
			t.Fatal(err)
		}
		items, err := s.ListarSolicitudesDocumentalesRRHH(context.Background(), q)
		if items != nil || !errors.Is(err, caso.causa) || len(r.ordenes) != 1 {
			t.Fatal("fallo sin auditoría o con datos")
		}
		d := r.ordenes[0]
		correlacion, _ := q.Correlacion.ValorCanonico()
		if d.Datos.Accion != ports.AccionConsultarSolicitudesDocumentalesRRHH || d.Datos.RecursoRef != q.ParticipacionRef ||
			d.Datos.FinalidadRef != ports.FinalidadCambiarSituacionParticipacion || d.Datos.Motivo != q.MotivoAutorizacion ||
			d.Datos.Resultado != caso.resultado || d.Datos.CorrelacionRef != correlacion || d.Datos.Canal != "interna_corporativa" ||
			!reflect.DeepEqual(d.ResultadoContexto, q.ResultadoContexto) {
			t.Fatal("sobre nominal sustituido")
		}
		if _, ok := AcuseConsultaDocumentalesFallida(err); !ok {
			t.Fatal("acuse confirmado perdido")
		}
	}
}

func TestConsultaDocumentalesAuditadaNoDevuelveProyeccionParcialOInvalida(t *testing.T) {
	q := solicitudSituacionPrueba(t, time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC))
	for _, causa := range []error{nil, core.ErrAutorizacionDenegada} {
		item := itemDocumentalAuditadoPrueba()
		item.FechaFinCausa = "2026-02-30"
		c := &consultaDocumentalesAuditadaPrueba{items: []ports.SolicitudDocumentalPendienteRRHH{item}, err: causa}
		r := &registradorDocumentalesAuditadaPrueba{t: t, consulta: c}
		s, _ := NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
		items, err := s.ListarSolicitudesDocumentalesRRHH(context.Background(), q)
		if items != nil || !errors.Is(err, ports.ErrSituacionParticipacionNoDisponible) || len(r.ordenes) != 1 || r.ordenes[0].Datos.Resultado != core.ResultadoIntentoAuditoriaError {
			t.Fatal("proyección no confiable publicada")
		}
		if causa == nil {
			var fechaInvalida *time.ParseError
			if !errors.As(err, &fechaInvalida) {
				t.Fatal("causa de fecha inválida perdida")
			}
		}
	}
}

func TestConsultaDocumentalesAuditadaCierraAnteRegistroFallidoOAcuseInvalido(t *testing.T) {
	q := solicitudSituacionPrueba(t, time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC))
	for _, invalido := range []bool{false, true} {
		c := &consultaDocumentalesAuditadaPrueba{err: core.ErrAutorizacionDenegada}
		r := &registradorDocumentalesAuditadaPrueba{t: t, consulta: c, acuseInvalido: invalido}
		if !invalido {
			r.err = errors.New("material privado del controlador")
		}
		s, _ := NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
		items, err := s.ListarSolicitudesDocumentalesRRHH(context.Background(), q)
		if items != nil || !errors.Is(err, ports.ErrSituacionParticipacionNoDisponible) || errors.Is(err, core.ErrAutorizacionDenegada) || strings.Contains(err.Error(), "privado") {
			t.Fatal("fallo del registrador con datos o causa cruda")
		}
		if _, ok := AcuseConsultaDocumentalesFallida(err); ok {
			t.Fatal("acuse falso")
		}
	}
}

func TestConsultaDocumentalesAuditadaConservaActorCapturadoTrasCancelacion(t *testing.T) {
	q := solicitudSituacionPrueba(t, time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC))
	historico, err := q.ResultadoContexto.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	c := &consultaDocumentalesAuditadaPrueba{despues: func() { cancelar(); q.ResultadoContexto.RepresentacionCanonica[0] = '!' }}
	r := &registradorDocumentalesAuditadaPrueba{t: t, consulta: c}
	s, _ := NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
	items, err := s.ListarSolicitudesDocumentalesRRHH(ctx, q)
	if items != nil || !errors.Is(err, context.Canceled) || len(r.ordenes) != 1 || !reflect.DeepEqual(r.ordenes[0].ResultadoContexto, historico) {
		t.Fatal("identidad histórica sustituida tras retorno")
	}
}

func TestConsultaDocumentalesAuditadaNoInventaContexto(t *testing.T) {
	c := &consultaDocumentalesAuditadaPrueba{err: core.ErrAutorizacionDenegada}
	r := &registradorDocumentalesAuditadaPrueba{t: t, consulta: c}
	s, _ := NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
	items, err := s.ListarSolicitudesDocumentalesRRHH(context.Background(), ports.SolicitudCambiarSituacionParticipacion{})
	if items != nil || err == nil || c.cerrada || len(r.ordenes) != 0 {
		t.Fatal("consulta o auditoría con actor inventado")
	}
	if _, err := NuevaConsultaSolicitudesDocumentalesAuditada(c, (*registradorDocumentalesAuditadaPrueba)(nil), "vec-bolsa-prueba"); err == nil {
		t.Fatal("registrador nil aceptado")
	}
}
