package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuenteJustificacionPrueba struct {
	p        ports.PreparacionJustificacion
	err      error
	lecturas int
}

func (f *fuenteJustificacionPrueba) PrepararJustificacion(context.Context, ports.OrdenJustificacion, string) (ports.PreparacionJustificacion, error) {
	f.lecturas++
	return f.p, f.err
}

type documentosJustificacionPrueba struct {
	preflightErr, err   error
	llamadas, preflight int
	cambiada            bool
	sinNumero           bool
}

func (d *documentosJustificacionPrueba) PrepararRegistro(context.Context, ports.OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion) error {
	d.preflight++
	return d.preflightErr
}
func (d *documentosJustificacionPrueba) RegistrarJustificante(_ context.Context, _ ports.OrdenJustificacion, s domain.SolicitudJustificable, p domain.PoliticaJustificacion, v domain.DocumentoJustificacion, _ string) (ports.RegistroDocumentalConfirmado, error) {
	d.llamadas++
	if d.cambiada {
		v.Version++
	}
	numero := "VEC-2026-14"
	if d.sinNumero {
		numero = ""
	}
	return ports.RegistroDocumentalConfirmado{Documento: v, ModuloID: "cronos", ExpedienteRef: s.ExpedienteDocumentalRef, TipoRef: p.TipoDocumentalRef,
		NumeroVEC: numero, CreadoEnUTC: time.Now().UTC(), PoliticaRef: "ref:" + strings.Repeat("1", 64),
		PoliticaVersion: 1, PoliticaSHA256: strings.Repeat("2", 64), ConservacionHastaUTC: time.Now().UTC().AddDate(1, 0, 0),
		Proteccion: "conservacion", EstadoPolitica: "aprobada"}, d.err
}

type repoJustificacionPrueba struct {
	recibo            *ports.ReciboJustificacion
	registro          *ports.RegistroDocumentalConfirmado
	err, errorLectura error
	lecturas, efectos int
	alterarRegistro   bool
	sustituirRegistro bool
}

func (r *repoJustificacionPrueba) RecuperarJustificacion(_ context.Context, _ ports.OrdenJustificacion, m domain.MaterialJustificacion) (ports.ReciboJustificacion, bool, error) {
	r.lecturas++
	if r.errorLectura != nil {
		return ports.ReciboJustificacion{}, false, r.errorLectura
	}
	if r.recibo == nil {
		return ports.ReciboJustificacion{}, false, nil
	}
	h, _ := m.Huella()
	if h != r.recibo.HuellaMaterial {
		return ports.ReciboJustificacion{}, false, domain.ErrJustificacionConflicto
	}
	return *r.recibo, true, nil
}
func (r *repoJustificacionPrueba) ConfirmarJustificacion(_ context.Context, m domain.MaterialJustificacion, j domain.Justificacion, registro *ports.RegistroDocumentalConfirmado, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboJustificacion, error) {
	r.efectos++
	if r.err != nil {
		return ports.ReciboJustificacion{}, r.err
	}
	if m.Accion == domain.AccionAnexarJustificacion {
		if registro == nil || registro.Documento != m.Vinculo.Documento {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		r.registro = registro
		if r.alterarRegistro {
			registro.NumeroVEC = "VEC-2026-15"
		}
		if r.sustituirRegistro {
			copia := *registro
			copia.NumeroVEC = "VEC-2026-15"
			r.registro = &copia
		}
	} else if registro != nil || r.registro == nil {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	h, _ := m.Huella()
	recibo := ports.ReciboJustificacion{Justificacion: j, Registro: r.registro, HuellaMaterial: h, ReciboRef: "recibo:ensayo", FechaUTC: time.Now().UTC().Truncate(time.Microsecond)}
	r.recibo = &recibo
	return recibo, nil
}

// Transporte estructural sólo para probar coordinación. No firma ni capacidad
// real: un consumidor V3 real rechazaría estos bytes, y nunca se monta en CLI.
type proveedorJustificacionPrueba struct {
	err             error
	vacio, cambiado bool
}

func (p *proveedorJustificacionPrueba) ProveerMaterialJustificacion(_ context.Context, m domain.MaterialJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.err != nil || p.vacio {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, p.err
	}
	recurso, e := RecursoJustificacion(m)
	if e != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	h, e := recurso.HuellaContextoAutorizacionSHA256()
	if e != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	if p.cambiado {
		h = strings.Repeat("a", 64)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.Vinculo.SolicitudRef, h, AudienciaJustificacion, ahora.Add(-time.Second), ahora.Add(3*time.Second))
	if e != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}
func escenarioJustificacion(t *testing.T) (*ServicioJustificacion, ports.OrdenJustificacion, *fuenteJustificacionPrueba, *documentosJustificacionPrueba, *repoJustificacionPrueba, *proveedorJustificacionPrueba, ports.PeticionAnexoJustificacion) {
	t.Helper()
	p := domain.PoliticaJustificacion{Referencia: "politica:justificacion:v1", Version: 1, SHA256: strings.Repeat("a", 64), CatalogoVersionRef: "catalogo:permiso:v1", PermisoRef: "permiso:neutral", TipoDocumentalRef: "ref:" + strings.Repeat("b", 64), CustodioID: "custodia.interna", MotivosRef: []string{"motivo:documentacion:conforme", "motivo:documentacion:incompleta"}}
	solicitud := domain.SolicitudJustificable{SolicitudRef: "permiso:cronos:solicitud:ensayo001", EmpleadoRef: "emp_0123456789abcdefghijkl", CatalogoVersionRef: p.CatalogoVersionRef, PermisoRef: p.PermisoRef, ExpedienteDocumentalRef: "ref:" + strings.Repeat("c", 64), Version: 3, Estado: domain.EstadoPermisoConcedido, JustificanteExigido: true}
	f := &fuenteJustificacionPrueba{p: ports.PreparacionJustificacion{Solicitud: solicitud, Politica: p}}
	d := &documentosJustificacionPrueba{}
	r := &repoJustificacionPrueba{}
	prov := &proveedorJustificacionPrueba{}
	a, _ := contexto(t).OrdenConsumo.ContextoActor()
	o, e := ports.NuevaOrdenJustificacion(a, prov)
	if e != nil {
		t.Fatal(e)
	}
	reloj := relojMarcajePrueba{time.Now().UTC()}
	s, e := NuevoServicioJustificacion(f, d, r, reloj)
	if e != nil {
		t.Fatal(e)
	}
	in := ports.PeticionAnexoJustificacion{SolicitudRef: solicitud.SolicitudRef, ClaveOperacion: "ref:" + strings.Repeat("f", 64), Documento: domain.DocumentoJustificacion{ID: "ref:" + strings.Repeat("d", 64), Version: 1, SHA256: strings.Repeat("e", 64), CustodioID: p.CustodioID, CustodiaRef: "original:ensayo001"}}
	return s, o, f, d, r, prov, in
}
func TestJustificacionGateAntesDocumentos(t *testing.T) {
	for _, caso := range []string{"enclave", "personal", "politica", "orden", "proveedor", "material_vacio", "material_cambiado", "documental", "lectura"} {
		t.Run(caso, func(t *testing.T) {
			s, o, f, d, r, p, in := escenarioJustificacion(t)
			switch caso {
			case "enclave", "personal":
				f.err = ports.ErrJustificacionNoDisponible
			case "politica":
				f.p.Politica.SHA256 = ""
			case "orden":
				o = ports.OrdenJustificacion{}
			case "proveedor":
				p.err = ports.ErrJustificacionNoDisponible
			case "material_vacio":
				p.vacio = true
			case "material_cambiado":
				p.cambiado = true
			case "documental":
				d.preflightErr = ports.ErrJustificacionNoDisponible
			case "lectura":
				r.errorLectura = ports.ErrJustificacionNoDisponible
			}
			res, e := s.Anexar(context.Background(), o, in)
			if e == nil || d.llamadas != 0 || r.efectos != 0 || res.Documento != nil {
				t.Fatal("efecto sin preflight", e, d.llamadas, r.efectos)
			}
		})
	}
	var f *fuenteJustificacionPrueba
	var d *documentosJustificacionPrueba
	var r *repoJustificacionPrueba
	for _, s := range []*ServicioJustificacion{nil, {fuente: f}, {documentos: d}, {repo: r}} {
		if _, e := s.Anexar(context.Background(), ports.OrdenJustificacion{}, ports.PeticionAnexoJustificacion{}); !errors.Is(e, ports.ErrJustificacionNoDisponible) {
			t.Fatal(e)
		}
	}
	s, _, f2, d2, r2, p, _ := escenarioJustificacion(t)
	for _, v := range []struct {
		f ports.FuenteJustificacion
		d ports.DocumentosJustificacion
		r ports.RepositorioJustificacion
	}{{f, d2, r2}, {f2, d, r2}, {f2, d2, r}} {
		if _, e := NuevoServicioJustificacion(v.f, v.d, v.r, s.reloj); e == nil {
			t.Fatal("typed nil")
		}
	}
	a, _ := contexto(t).OrdenConsumo.ContextoActor()
	p = nil
	if _, e := ports.NuevaOrdenJustificacion(a, p); e == nil {
		t.Fatal("proveedor typed nil")
	}
}
func TestJustificacionDosEfectosFalloYReplay(t *testing.T) {
	s, o, f, d, r, _, in := escenarioJustificacion(t)
	r.err = ports.ErrJustificacionNoDisponible
	res, e := s.Anexar(context.Background(), o, in)
	if !errors.Is(e, ports.ErrEnlaceJustificacionPendiente) || !res.EnlacePendiente || res.Documento == nil || res.Registro == nil || res.ReciboCronos != nil || d.llamadas != 1 {
		t.Fatal("oculta alta parcial", e, res)
	}
	r.err = nil
	res, e = s.Anexar(context.Background(), o, in)
	if e != nil || res.EnlacePendiente || res.ReciboCronos == nil || res.ReciboCronos.Registro == nil || res.ReciboCronos.Registro.NumeroVEC != "VEC-2026-14" || d.llamadas != 2 {
		t.Fatal(e, res)
	}
	original := *res.ReciboCronos
	f.p.Actual = &original.Justificacion
	replay, e := s.Anexar(context.Background(), o, in)
	if e != nil || !replay.ReciboCronos.Replay || replay.ReciboCronos.ReciboRef != original.ReciboRef || !replay.ReciboCronos.FechaUTC.Equal(original.FechaUTC) || d.llamadas != 2 {
		t.Fatal("replay duplica efecto", e)
	}
	in.Documento.SHA256 = strings.Repeat("a", 64)
	if _, e = s.Anexar(context.Background(), o, in); !errors.Is(e, domain.ErrJustificacionConflicto) || d.llamadas != 2 {
		t.Fatal("clave sustituida", e)
	}
	f.err = ports.ErrJustificacionNoDisponible
	in.Documento = original.Justificacion.Vinculo.Documento
	if _, e = s.Anexar(context.Background(), o, in); e == nil || d.llamadas != 2 {
		t.Fatal("replay sin lectura actual")
	}
}
func TestRevisionJustificacionSeparadaYRefsExactas(t *testing.T) {
	s, o, f, d, r, _, in := escenarioJustificacion(t)
	res, e := s.Anexar(context.Background(), o, in)
	if e != nil {
		t.Fatal(e)
	}
	f.p.Actual = &res.ReciboCronos.Justificacion
	r.recibo = nil
	p := ports.PeticionRevisionJustificacion{SolicitudRef: in.SolicitudRef, ClaveOperacion: "ref:" + strings.Repeat("a", 64), VersionEsperada: 1, Vinculo: f.p.Actual.Vinculo, Decision: domain.JustificacionRechazada, MotivoRef: f.p.Politica.MotivosRef[1]}
	original := f.p.Solicitud
	cambiado := p
	cambiado.Vinculo.Documento.Version++
	if _, e = s.Revisar(context.Background(), o, cambiado); e == nil {
		t.Fatal("revision documento diferente")
	}
	rec, e := s.Revisar(context.Background(), o, p)
	if e != nil || rec.Justificacion.Estado != domain.JustificacionRechazada || f.p.Solicitud != original || d.llamadas != 1 {
		t.Fatal(e)
	}
	f.p.Actual = &rec.Justificacion
	replay, e := s.Revisar(context.Background(), o, p)
	if e != nil || !replay.Replay || replay.ReciboRef != rec.ReciboRef || !replay.FechaUTC.Equal(rec.FechaUTC) {
		t.Fatal("revision replay", e)
	}
}
func TestJustificacionFalloDocumentosNoConfirmaCronos(t *testing.T) {
	s, o, _, d, r, _, in := escenarioJustificacion(t)
	d.err = ports.ErrJustificacionNoDisponible
	if res, e := s.Anexar(context.Background(), o, in); e == nil || r.efectos != 0 || res.ReciboCronos != nil {
		t.Fatal("confirmacion tras error documental")
	}
	d.err = nil
	d.cambiada = true
	if res, e := s.Anexar(context.Background(), o, in); !errors.Is(e, ports.ErrEnlaceJustificacionPendiente) || r.efectos != 0 || res.ReciboCronos != nil {
		t.Fatal("enlace de documento incoherente")
	}
	d.cambiada = false
	d.sinNumero = true
	if res, e := s.Anexar(context.Background(), o, in); !errors.Is(e, ports.ErrEnlaceJustificacionPendiente) || r.efectos != 0 || res.ReciboCronos != nil {
		t.Fatal("enlace sin confirmacion registral")
	}
}
func TestJustificacionNoAceptaReciboConRegistroDistinto(t *testing.T) {
	for _, caso := range []string{"mutar argumento", "sustituir recibo"} {
		t.Run(caso, func(t *testing.T) {
			s, o, _, _, r, _, in := escenarioJustificacion(t)
			r.alterarRegistro = caso == "mutar argumento"
			r.sustituirRegistro = caso == "sustituir recibo"
			res, err := s.Anexar(context.Background(), o, in)
			if !errors.Is(err, ports.ErrEnlaceJustificacionPendiente) || !res.EnlacePendiente || res.Registro == nil || res.Registro.NumeroVEC != "VEC-2026-14" || res.ReciboCronos != nil {
				t.Fatal("recibo aceptado con registro distinto al confirmado", err, res)
			}
		})
	}
}

func TestRecursoJustificacionLigaHuellaDelMaterialAlContextoV3(t *testing.T) {
	_, o, f, _, _, _, in := escenarioJustificacion(t)
	actor, err := o.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	v := domain.VinculoJustificacion{SolicitudRef: f.p.Solicitud.SolicitudRef, EmpleadoRef: f.p.Solicitud.EmpleadoRef,
		CatalogoVersionRef: f.p.Solicitud.CatalogoVersionRef, PermisoRef: f.p.Solicitud.PermisoRef,
		ExpedienteDocumentalRef: f.p.Solicitud.ExpedienteDocumentalRef, Documento: in.Documento}
	m := materialJustificacion(actor, f.p, v, in.ClaveOperacion, domain.AccionAnexarJustificacion, 0)
	recurso, err := RecursoJustificacion(m)
	if err != nil || recurso.Referencia != in.SolicitudRef || recurso.Ambitos["empleado_ref"] != f.p.Solicitud.EmpleadoRef {
		t.Fatal("recurso C8 incoherente", err)
	}
	contexto, err := recurso.HuellaContextoAutorizacionSHA256()
	bruta, _ := m.Huella()
	if err != nil || contexto == bruta || recurso.Atributos["material_sha256"] != bruta {
		t.Fatal("huella V3 confundida con material bruto", err)
	}
	m.Vinculo.Documento.SHA256 = strings.Repeat("9", 64)
	cambiado, err := RecursoJustificacion(m)
	huellaCambiada, e := cambiado.HuellaContextoAutorizacionSHA256()
	if err != nil || e != nil || contexto == huellaCambiada {
		t.Fatal("material alterado conserva recurso V3", err, e)
	}
}
