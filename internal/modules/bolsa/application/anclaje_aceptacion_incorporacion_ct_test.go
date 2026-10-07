package application

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func solicitudAnclajePrueba(t *testing.T) ports.SolicitudConsultaAnclajeAceptacionCT {
	s := solicitudPersonaAceptacionPrueba(t)
	x := s.Selector
	return ports.SolicitudConsultaAnclajeAceptacionCT{Selector: ports.SelectorAnclajeAceptacionCT{UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef, AceptacionOperacionRef: x.AceptacionOperacionRef, AceptacionRegistroSHA256: x.AceptacionRegistroSHA256, AperturaOperacionRef: x.AperturaOperacionRef, LlamamientoRef: x.LlamamientoRef, PropuestaRef: x.PropuestaRef}, ActorConfiable: s.ActorConfiable}
}

type emisorAnclajePrueba struct {
	emisorNominalPersonaAceptacionPrueba
}

// Doble local del emisor: verifica el protocolo nominal, no COSE ni persistencia.
func (e *emisorAnclajePrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, c core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, confirmacion, exportador, err := e.emisorNominalPersonaAceptacionPrueba.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil || exportador == nil {
		return d, confirmacion, exportador, err
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		e.t.Fatal(err)
	}
	datos, _ := s.Datos()
	dh, _ := core.HuellaSHA256DecisionAutorizacionV3(d)
	mh, _ := core.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	anterior := m.ResumenCapacidad()
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(anterior.DecisionRef(), dh, mh, c.RegistroContextoRef, c.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh, ports.AudienciaConsultaAnclajeAceptacionCT, anterior.EmitidaEn(), anterior.ExpiraEn())
	if err != nil {
		e.t.Fatal(err)
	}
	m, err = vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), x, m.DecisionCanonica(), m.MotivoCanonico(), c.RepresentacionCanonica, m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		e.t.Fatal(err)
	}
	return d, confirmacion, exportadorBorradorPrueba{material: m}, nil
}
func proveedorAnclajePrueba(t *testing.T, e *emisorAnclajePrueba) *ProveedorNominalConsultaAnclajeAceptacionCT {
	t.Helper()
	p, err := NuevoProveedorNominalConsultaAnclajeAceptacionCT(e, motivoBorradorPrueba(), func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error) {
		return correlacionBorradorPrueba(t), nil
	}, ahoraAnclajePrueba)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func ahoraAnclajePrueba() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) }
func emisorAnclajeValidoPrueba(t *testing.T) *emisorAnclajePrueba {
	return &emisorAnclajePrueba{emisorNominalPersonaAceptacionPrueba{t: t, campos: []string{"anclaje"}}}
}

type repositorioAnclajePrueba struct {
	llamadas int
	orden    ports.OrdenConsultaAnclajeAceptacionCT
	cambiar  func(*ports.ResultadoConsultaAnclajeAceptacionCT)
	cancelar context.CancelFunc
}

func (r *repositorioAnclajePrueba) ConsultarAnclajeAceptacionCT(_ context.Context, o ports.OrdenConsultaAnclajeAceptacionCT) (ports.ResultadoConsultaAnclajeAceptacionCT, error) {
	r.llamadas++
	r.orden = o
	if r.cancelar != nil {
		r.cancelar()
	}
	v := resultadoAnclajePrueba(o)
	if r.cambiar != nil {
		r.cambiar(&v)
	}
	return v, nil
}
func resultadoAnclajePrueba(o ports.OrdenConsultaAnclajeAceptacionCT) ports.ResultadoConsultaAnclajeAceptacionCT {
	s := o.Solicitud.Selector
	x := o.Material.ResumenCapacidad()
	return ports.ResultadoConsultaAnclajeAceptacionCT{Estado: "acreditado", Anclaje: &ports.AnclajeAceptacionIncorporacionCT{SelectorPersonaAceptacionCT: ports.SelectorPersonaAceptacionCT{UnidadRef: s.UnidadRef, CategoriaRef: s.CategoriaRef, NecesidadRef: s.NecesidadRef, AceptacionOperacionRef: s.AceptacionOperacionRef, AceptacionRegistroSHA256: s.AceptacionRegistroSHA256, AperturaOperacionRef: s.AperturaOperacionRef, AperturaRegistroSHA256: strings.Repeat("b", 64), LlamamientoRef: s.LlamamientoRef, PropuestaRef: s.PropuestaRef}, AceptacionReciboRef: "recibo:aceptacion:original"}, Evidencia: ports.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: x.DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:actual", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
}
func TestAnclajeAceptacionCTCanonOchoSelectoresYAislamiento(t *testing.T) {
	s := solicitudAnclajePrueba(t)
	p, err := PrepararConsultaAnclajeAceptacionCT(s)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.bolsa.anclaje-aceptacion-ct.consulta.v1","unidad_ref":"unidad:rrhh","categoria_ref":"categoria:una","necesidad_ref":"necesidad:una","aceptacion_operacion_ref":"aceptacion:una","aceptacion_registro_sha256":"` + strings.Repeat("a", 64) + `","apertura_operacion_ref":"apertura:una","llamamiento_ref":"llamamiento:uno","propuesta_ref":"propuesta:una"}`
	if string(p.MaterialCanonico) != esperado {
		t.Fatal("canon no conserva los ocho selectores")
	}
	for _, cambiar := range []func(*ports.SelectorAnclajeAceptacionCT){func(x *ports.SelectorAnclajeAceptacionCT) { x.UnidadRef = "unidad:otra" }, func(x *ports.SelectorAnclajeAceptacionCT) { x.CategoriaRef = "categoria:otra" }, func(x *ports.SelectorAnclajeAceptacionCT) { x.NecesidadRef = "necesidad:otra" }, func(x *ports.SelectorAnclajeAceptacionCT) { x.AceptacionOperacionRef = "aceptacion:otra" }, func(x *ports.SelectorAnclajeAceptacionCT) { x.AceptacionRegistroSHA256 = strings.Repeat("c", 64) }, func(x *ports.SelectorAnclajeAceptacionCT) { x.AperturaOperacionRef = "apertura:otra" }, func(x *ports.SelectorAnclajeAceptacionCT) { x.LlamamientoRef = "llamamiento:otro" }, func(x *ports.SelectorAnclajeAceptacionCT) { x.PropuestaRef = "propuesta:otra" }} {
		q := s
		cambiar(&q.Selector)
		p2, err := PrepararConsultaAnclajeAceptacionCT(q)
		if err != nil || bytes.Equal(p.MaterialCanonico, p2.MaterialCanonico) || p.Recurso.Atributos["material_sha256"] == p2.Recurso.Atributos["material_sha256"] {
			t.Fatal("selector ignorado")
		}
	}
	p.Solicitud.ActorConfiable.Resultado.RepresentacionCanonica[0] = 'x'
	if s.ActorConfiable.Resultado.Validar() != nil {
		t.Fatal("actor comparte bytes mutables")
	}
	s.Selector.AceptacionRegistroSHA256 = strings.Repeat("0", 64)
	if _, err := PrepararConsultaAnclajeAceptacionCT(s); err != ports.ErrConsultaAnclajeAceptacionCTInvalida {
		t.Fatal("huella ficticia admitida")
	}
}
func TestAnclajeAceptacionCTProveedorNominalReconstruyeYCotejaProyeccion(t *testing.T) {
	e := emisorAnclajeValidoPrueba(t)
	p := proveedorAnclajePrueba(t, e)
	s := solicitudAnclajePrueba(t)
	entrada, _ := PrepararConsultaAnclajeAceptacionCT(s)
	entrada.Recurso.Referencia = "aceptacion:ajena"
	entrada.MaterialCanonico = []byte("libre")
	m, err := p.AutorizarConsultaAnclajeAceptacionCT(context.Background(), entrada)
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := e.ultima.Datos()
	if datos.Accion != ports.AccionConsultaAnclajeAceptacionCT || datos.Recurso.Referencia != s.Selector.AceptacionOperacionRef || datos.Finalidad != ports.FinalidadConsultaAnclajeAceptacionCT || m.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaConsultaAnclajeAceptacionCT {
		t.Fatal("autoridad ajena")
	}
	for _, campos := range [][]string{nil, {"persona"}, {"anclaje", "persona"}} {
		e.campos = campos
		if _, err := p.AutorizarConsultaAnclajeAceptacionCT(context.Background(), entrada); err != ports.ErrConsultaAnclajeAceptacionCTDenegada {
			t.Fatal("campos ajenos admitidos")
		}
	}
}
func TestAnclajeAceptacionCTServicioMinimizaYCotejaTodaSalida(t *testing.T) {
	cambios := []func(*ports.ResultadoConsultaAnclajeAceptacionCT){func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.UnidadRef = "unidad:otra" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.CategoriaRef = "categoria:otra" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.NecesidadRef = "necesidad:otra" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) {
		v.Anclaje.AceptacionOperacionRef = "aceptacion:otra"
	}, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) {
		v.Anclaje.AceptacionRegistroSHA256 = strings.Repeat("c", 64)
	}, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.AperturaOperacionRef = "apertura:otra" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) {
		v.Anclaje.AperturaRegistroSHA256 = strings.Repeat("0", 64)
	}, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.LlamamientoRef = "llamamiento:otro" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.PropuestaRef = "propuesta:otra" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje.AceptacionReciboRef = "" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Evidencia.DecisionRef = "decision:ajena" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Estado = "pendiente" }, func(v *ports.ResultadoConsultaAnclajeAceptacionCT) { v.Anclaje = nil }}
	for i, cambio := range cambios {
		e := emisorAnclajeValidoPrueba(t)
		r := &repositorioAnclajePrueba{cambiar: cambio}
		s, _ := NuevoServicioConsultaAnclajeAceptacionCT(proveedorAnclajePrueba(t, e), r, ahoraAnclajePrueba)
		v, err := s.ConsultarAnclajeAceptacionCT(context.Background(), solicitudAnclajePrueba(t))
		if err != ports.ErrConsultaAnclajeAceptacionCTNoDisponible || v.Estado != "" || v.Anclaje != nil {
			t.Fatalf("caso=%d salida parcial: %v", i, err)
		}
	}
	for _, estado := range []string{"acreditado", "pendiente", "no_encontrada"} {
		e := emisorAnclajeValidoPrueba(t)
		r := &repositorioAnclajePrueba{cambiar: func(v *ports.ResultadoConsultaAnclajeAceptacionCT) {
			v.Estado = estado
			if estado != "acreditado" {
				v.Anclaje = nil
			}
		}}
		s, _ := NuevoServicioConsultaAnclajeAceptacionCT(proveedorAnclajePrueba(t, e), r, ahoraAnclajePrueba)
		v, err := s.ConsultarAnclajeAceptacionCT(context.Background(), solicitudAnclajePrueba(t))
		if err != nil || v.Estado != estado || e.llamadas != 1 || r.llamadas != 1 {
			t.Fatalf("estado=%s error=%v", estado, err)
		}
	}
}
func TestAnclajeAceptacionCTDenegacionDependenciaCancelacion(t *testing.T) {
	for _, caso := range []struct{ err, esperado error }{{core.ErrAutorizacionDenegada, ports.ErrConsultaAnclajeAceptacionCTDenegada}, {errors.Join(core.ErrAutorizacionDenegada, vecports.ErrFuenteAutorizacionNoDisponible), ports.ErrConsultaAnclajeAceptacionCTNoDisponible}, {errors.New("detalle privado"), ports.ErrConsultaAnclajeAceptacionCTNoDisponible}} {
		e := emisorAnclajeValidoPrueba(t)
		e.err = caso.err
		r := &repositorioAnclajePrueba{}
		s, _ := NuevoServicioConsultaAnclajeAceptacionCT(proveedorAnclajePrueba(t, e), r, ahoraAnclajePrueba)
		v, err := s.ConsultarAnclajeAceptacionCT(context.Background(), solicitudAnclajePrueba(t))
		if err != caso.esperado || v.Estado != "" || r.llamadas != 0 {
			t.Fatal("clasificación o llamada indebida")
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	e := emisorAnclajeValidoPrueba(t)
	r := &repositorioAnclajePrueba{cancelar: cancelar}
	s, _ := NuevoServicioConsultaAnclajeAceptacionCT(proveedorAnclajePrueba(t, e), r, ahoraAnclajePrueba)
	v, err := s.ConsultarAnclajeAceptacionCT(ctx, solicitudAnclajePrueba(t))
	if err != context.Canceled || v.Estado != "" {
		t.Fatal("expuso tras cancelación")
	}
	if _, err := NuevoServicioConsultaAnclajeAceptacionCT(nil, r, ahoraAnclajePrueba); err == nil {
		t.Fatal("proveedor nil")
	}
}
