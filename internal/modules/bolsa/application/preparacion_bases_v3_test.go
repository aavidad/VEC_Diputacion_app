package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

type correlacionBasesV3Prueba struct{}

func (correlacionBasesV3Prueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_" + strings.Repeat("a", 32), nil
}

func solicitudBasesV3Prueba(t *testing.T) ports.SolicitudGuardarPreparacionBasesV3 {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	cuenta := vec.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vec.AuthMethodCertificate, Garantia: vec.AuthAssuranceHigh}
	i := vec.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	a, err := vec.NuevoContextoActor(cuenta, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	c, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), correlacionBasesV3Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	ambito, err := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "uni_seleccionexterna")
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudGuardarPreparacionBasesV3{Actor: a, Correlacion: c, Ambito: ambito, Esperada: prep.Esperada{PreparacionRef: "prep:sintetica"}, ClaveOperacion: "operacion:sintetica"}
}

// Fixture de forma nominal, con bytes deliberadamente sinteticos. No es una
// concesion registrada ni una atestacion valida; ningun SQL real la aceptaria.
func exportacionEstructuralBasesPrueba(t *testing.T, p PreparacionOperacionBasesV3, a vec.ContextoActor, accion, audiencia string) core.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, _ := p.Recurso.HuellaContextoAutorizacionSHA256()
	canon, _ := a.RepresentacionCanonicaVinculadaV2()
	x, err := core.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prep:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "contexto:prep:prueba", strings.Repeat("c", 64), accion, p.Recurso.Referencia, h, audiencia, a.ResueltoEn, a.ResueltoEn.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := json.Marshal(map[string]any{"accion": accion, "modulo_id": "bolsa", "tipo_recurso": ports.TipoRecursoPreparacionBases, "finalidad": ports.FinalidadPreparacionBases, "recurso_ref": p.Recurso.Referencia, "campos_permitidos": CamposPreparacionBasesV3(accion), "obligaciones": []string{}, "vinculo_autenticacion_actor": map[string]string{"superficie": "interna_corporativa"}})
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	e, err := core.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, decision, []byte("motivo-sintetico"), canon, 1, 1, []byte("payload-sintetico"), []byte("sobre-sintetico"), []byte("evidencia-sintetica"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestPreparacionBasesV3LigaClaveAmbitoCASYCuerpo(t *testing.T) {
	q := solicitudBasesV3Prueba(t)
	p, err := PrepararGuardadoPreparacionBasesV3(q)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if json.Unmarshal(p.Canonico, &m) != nil || len(m) != 16 || m["actor_ref"] != q.Actor.PersonaRef || m["clave_operacion"] != q.ClaveOperacion {
		t.Fatal("ABI no ligada")
	}
	intencion, _ := prep.HuellaIntencion(q.Esperada, q.Material, q.Ambito)
	q.ClaveOperacion = "operacion:otra"
	otro, _ := PrepararGuardadoPreparacionBasesV3(q)
	intencion2, _ := prep.HuellaIntencion(q.Esperada, q.Material, q.Ambito)
	if intencion != intencion2 || bytes.Equal(p.Canonico, otro.Canonico) {
		t.Fatal("clave no ligada a V3 o altera intencion estable")
	}
	e := exportacionEstructuralBasesPrueba(t, p, q.Actor, ports.AccionGuardarPreparacionBases, AudienciaGuardarPreparacionBasesV3)
	q.ClaveOperacion = "operacion:sintetica"
	o := ports.OrdenGuardarPreparacionBasesV3{Solicitud: q, Autorizacion: e}
	if ValidarMaterialGuardarPreparacionBasesV3(o) != nil {
		t.Fatal("fixture estructural rechazada")
	}
	o.Solicitud.Material.Contenido.Titulo = "Cambio sin otra concesión"
	if ValidarMaterialGuardarPreparacionBasesV3(o) == nil {
		t.Fatal("contenido distinto reutiliza capacidad")
	}
	o.Solicitud = q
	o.Solicitud.Ambito, _ = bolsa.NuevoAmbitoOrganizativoConvocatoria("org_otradiputacionab", "uni_seleccionexterna")
	if ValidarMaterialGuardarPreparacionBasesV3(o) == nil {
		t.Fatal("ambito cruzado autorizado")
	}
}

func TestPreparacionBasesV3ModoExplicitoYLigado(t *testing.T) {
	s := solicitudBasesV3Prueba(t)
	q := ports.SolicitudConsultarPreparacionBasesV3{Actor: s.Actor, Correlacion: s.Correlacion, Ambito: s.Ambito, Selector: ports.SelectorConsultaPreparacionBases{Modo: "actual", Exacta: s.Esperada}}
	p, err := PrepararConsultaPreparacionBasesV3(q)
	if err != nil {
		t.Fatal(err)
	}
	e := exportacionEstructuralBasesPrueba(t, p, q.Actor, ports.AccionConsultarPreparacionBases, AudienciaConsultarPreparacionBasesV3)
	o := ports.OrdenConsultarPreparacionBasesV3{Solicitud: q, Autorizacion: e}
	if ValidarMaterialConsultarPreparacionBasesV3(o) != nil {
		t.Fatal("consulta actual estructural rechazada")
	}
	q.Selector.Modo = "exacta"
	q.Selector.Exacta.Revision = 1
	q.Selector.Exacta.HuellaMaterialSHA256 = strings.Repeat("a", 64)
	o.Solicitud = q
	if ValidarMaterialConsultarPreparacionBasesV3(o) == nil {
		t.Fatal("modo cambiado reutiliza capacidad")
	}
	for _, selector := range []ports.SelectorConsultaPreparacionBases{{Exacta: s.Esperada}, {Modo: "actual", Exacta: q.Selector.Exacta}, {Modo: "exacta", Exacta: s.Esperada}} {
		q.Selector = selector
		if _, err := PrepararConsultaPreparacionBasesV3(q); err == nil {
			t.Fatal("modo ambiguo admitido")
		}
	}
}

func TestPreparacionBasesV3ExigeCampoExplicitoRecibo(t *testing.T) {
	q := solicitudBasesV3Prueba(t)
	p, _ := PrepararGuardadoPreparacionBasesV3(q)
	d := map[string]any{"accion": ports.AccionGuardarPreparacionBases, "modulo_id": "bolsa", "tipo_recurso": ports.TipoRecursoPreparacionBases, "finalidad": ports.FinalidadPreparacionBases, "recurso_ref": p.Recurso.Referencia, "campos_permitidos": []string{"auditoria", "evento_outbox", "historia", "material_preparacion"}, "obligaciones": []string{}, "vinculo_autenticacion_actor": map[string]string{"superficie": "interna_corporativa"}}
	b, _ := json.Marshal(d)
	if decisionPreparacionV3Valida(b, p.Recurso, ports.AccionGuardarPreparacionBases) {
		t.Fatal("recibo autorizado implicitamente")
	}
}

type autorizacionBasesV3Prueba struct {
	fallo    error
	llamadas int
}

func (a *autorizacionBasesV3Prueba) AutorizarGuardadoPreparacionBases(context.Context, ports.SolicitudGuardarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return core.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.fallo
}
func (a *autorizacionBasesV3Prueba) AutorizarConsultaPreparacionBases(context.Context, ports.SolicitudConsultarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return core.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.fallo
}

type repositorioBasesV3Prueba struct{ llamadas int }

func (r *repositorioBasesV3Prueba) GuardarPreparacionBasesV3(context.Context, ports.OrdenGuardarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	r.llamadas++
	return ports.ResultadoPreparacionBasesV3{}, nil
}
func (r *repositorioBasesV3Prueba) ConsultarPreparacionBasesV3(context.Context, ports.OrdenConsultarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	r.llamadas++
	return ports.ResultadoPreparacionBasesV3{}, nil
}

func TestPreparacionBasesV3NoLlegaAPGSinMaterialNominal(t *testing.T) {
	for _, fallo := range []error{nil, ports.ErrPreparacionBasesDenegada, ports.ErrPreparacionBasesNoDisponible} {
		a, r := &autorizacionBasesV3Prueba{fallo: fallo}, &repositorioBasesV3Prueba{}
		s, err := NuevoServicioPreparacionBasesV3(a, r)
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Guardar(context.Background(), solicitudBasesV3Prueba(t))
		esperado := fallo
		if esperado == nil {
			esperado = ports.ErrPreparacionBasesNoDisponible
		}
		if !errors.Is(err, esperado) || r.llamadas != 0 || a.llamadas != 1 {
			t.Fatal("PG alcanzo capacidad vacia")
		}
	}
}

func TestPreparacionBasesV3ValidaTiempoSQLYReciboHistoricoSeparado(t *testing.T) {
	q := solicitudBasesV3Prueba(t)
	p, _ := PrepararGuardadoPreparacionBasesV3(q)
	e := exportacionEstructuralBasesPrueba(t, p, q.Actor, ports.AccionGuardarPreparacionBases, AudienciaGuardarPreparacionBasesV3)
	h, _ := q.Material.HuellaSHA256()
	i, _ := prep.HuellaIntencion(q.Esperada, q.Material, q.Ambito)
	c, _ := q.Correlacion.ValorCanonico()
	r := ports.ResultadoPreparacionBasesV3{Estado: "recuperada", Version: prep.Version{Ambito: q.Ambito, Estado: prep.Esperada{PreparacionRef: q.Esperada.PreparacionRef, Revision: 1, HuellaMaterialSHA256: h}, Material: q.Material}, Recibo: ports.ReciboPreparacionBases{ReciboRef: "recibo:historico", HistoriaRef: "historia:original", AuditoriaRef: "auditoria:efecto", EventoRef: "evento:original", HuellaIntencionSHA256: i, ConfirmadaEn: q.Actor.ResueltoEn}, Acceso: ports.EvidenciaAccesoPreparacionBasesV3{DecisionRef: e.ResumenCapacidad().DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("b", 64), AuditoriaRef: "auditoria:acceso", ReciboRef: "recibo:acceso", CorrelacionRef: c, AccedidaEn: q.Actor.ResueltoEn.Add(time.Second)}}
	o := ports.OrdenGuardarPreparacionBasesV3{Solicitud: q, Autorizacion: e}
	if ValidarResultadoGuardarPreparacionBasesV3(o, r) != nil {
		t.Fatal("instante SQL posterior rechazado")
	}
	r.Estado = "guardada"
	r.Acceso.AuditoriaRef = r.Recibo.AuditoriaRef
	if ValidarResultadoGuardarPreparacionBasesV3(o, r) != nil {
		t.Fatal("una auditoria real inicial de efecto y acceso rechazada")
	}
	r.Estado = "recuperada"
	if ValidarResultadoGuardarPreparacionBasesV3(o, r) == nil {
		t.Fatal("replay acepta auditoria historica como acceso nuevo")
	}
	r.Acceso.AuditoriaRef = "auditoria:acceso"
	r.Recibo.ConfirmadaEn = r.Acceso.AccedidaEn.Add(time.Second)
	if ValidarResultadoGuardarPreparacionBasesV3(o, r) == nil {
		t.Fatal("recibo futuro aceptado")
	}
}
