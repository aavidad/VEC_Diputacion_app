package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"log/slog"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

var instante = time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)

const persona = "per_0123456789abcdefghijkl"
const perfil = "prf_0123456789abcdefghijkl"
const decisionRef = "decision:rum:prueba"

type relojPrueba struct{}

func (relojPrueba) Ahora() time.Time { return instante.Add(2 * time.Second) }

func escenario(t *testing.T) (*Servicio, Solicitud, *autorizadorPrueba, *registroPrueba, *auditoriaPrueba) {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, persona, perfil, vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	concesion, err := pruebas.NuevaConcesionV3Prueba(vecDatos(vec.RecursoAutorizable{Referencia: "hecho:prueba", ModuloID: "meritos", Tipo: "hecho", Ambitos: map[string]string{"persona_ref": persona}}, accionDeclarar, "declaracion_hecho_propio"))
	if err != nil {
		t.Fatal(err)
	}
	datos, err := concesion.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	auth, registro, audit := &autorizadorPrueba{t: t}, &registroPrueba{}, &auditoriaPrueba{}
	s, err := NuevoServicio(auth, registro, audit, relojPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	solicitud := Solicitud{Vinculo: vinculo, Contexto: resultado, Correlacion: datos.Correlacion, Motivo: datos.ReferenciaMotivo,
		ClaveIdempotencia: "operacion:prueba", FechaCorte: "2026-10-01",
		Hecho: domain.Hecho{Referencia: "hecho:prueba", PersonaRef: persona, Version: 1, Tipo: "curso_asistencia", ConceptoRef: "curso:prueba", Denominacion: "Curso de prueba", Procedencia: domain.Procedencia{FuenteRef: "fuente:prueba", Version: "1", HechoOrigenRef: "origen:prueba", CapturadaEn: instante.Format(time.RFC3339)}, Vigencia: domain.Vigencia{Desde: "2026-09-01"}, Estado: domain.Declarado, Evidencias: []vec.ReferenciaDocumento{{ID: "documento:prueba", Version: 1}}}}
	return s, solicitud, auth, registro, audit
}

func vecDatos(recurso vec.RecursoAutorizable, accion, finalidad string) pruebas.DatosConcesionV3Prueba {
	return pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: persona, PerfilRef: perfil, Accion: accion, Recurso: recurso, Finalidad: finalidad,
		Campos: []string{"hecho", "declarante_ref", "version", "recibo"}, Obligaciones: []string{"auditar"}, DecisionRef: decisionRef}
}

type autorizadorPrueba struct {
	t        *testing.T
	llamadas int
	err      error
	defecto  string
}

func (a *autorizadorPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, solicitud vec.SolicitudAutorizacionLigadaV3, resultado vec.ResultadoContextoActorRegistradoV2) (vec.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	if a.err != nil {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, a.err
	}
	datos, err := solicitud.Datos()
	if err != nil {
		a.t.Fatal(err)
	}
	d := vecDatos(datos.Recurso, datos.Accion, datos.Finalidad)
	if a.defecto == "campos" {
		d.Campos = []string{"hecho"}
	}
	c, err := pruebas.NuevaConcesionV3Prueba(d)
	if err != nil {
		a.t.Fatal(err)
	}
	if c.Decision.ValidarPara(solicitud) != nil {
		a.t.Fatal("concesión de prueba no ligada a solicitud")
	}
	dh, _ := vec.HuellaSHA256DecisionAutorizacionV3(c.Decision)
	mh, _ := vec.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	_, audiencia := finalidadAudiencia(datos.Accion)
	if a.defecto == "audiencia" {
		audiencia = "otra.audiencia.v1"
	}
	expira := instante.Add(5 * time.Second)
	if a.defecto == "caducada" {
		expira = instante.Add(1500 * time.Millisecond)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, dh, mh, resultado.RegistroContextoRef, resultado.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh, audiencia, instante.Add(time.Second), expira)
	if err != nil {
		a.t.Fatal(err)
	}
	dc, _ := vec.RepresentacionCanonicaDecisionAutorizacionV3(c.Decision)
	mc, _ := vec.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	// Preimágenes de material sintético solo para cotejo unitario. No prueban
	// firma COSE, consumo V3 ni persistencia, y nunca se conectan en runtime.
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		a.t.Fatal(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'b'}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, dc, mc, resultado.RepresentacionCanonica, resultado.Contexto.Instantanea.PersonaVersion, resultado.Contexto.Instantanea.PerfilVersion, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		a.t.Fatal(err)
	}
	return c.Decision, c.Confirmacion, exportadorPrueba{m}, nil
}

type exportadorPrueba struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (e exportadorPrueba) ExportarMaterialParaConsumidor() (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.material, nil
}
func (exportadorPrueba) String() string         { return "[EXPORTADOR-PRUEBA]" }
func (e exportadorPrueba) LogValue() slog.Value { return slog.StringValue(e.String()) }

// Dobles exclusivos del test: sus recibos no acreditan una transacción real.
type registroPrueba struct {
	lecturas, confirmaciones, recuperaciones int
	recibo                                   *ports.Recibo
	actual                                   *ports.RegistroActual
	resultado                                *ports.ResultadoOperacion
	err                                      error
	mutar                                    bool
}

func (r *registroPrueba) EjecutarOperacion(_ context.Context, orden ports.OrdenOperacion) (ports.ResultadoOperacion, error) {
	r.lecturas++
	if r.resultado != nil {
		return *r.resultado, nil
	}
	if r.recibo != nil {
		r.recuperaciones++
		return ports.ResultadoOperacion{Codigo: "confirmada", AuditoriaRef: "auditoria:acceso", Anterior: r.actual, Recibo: r.recibo}, nil
	}
	r.confirmaciones++
	if r.err != nil {
		return ports.ResultadoOperacion{}, r.err
	}
	cambio, err := prepararCambio(orden, r.actual, instante)
	if err != nil {
		return ports.ResultadoOperacion{}, err
	}
	out := ports.Recibo{Referencia: "recibo:prueba", Accion: orden.Accion, ActorRef: orden.ActorRef, ClaveIdempotencia: orden.ClaveIdempotencia, HuellaComando: orden.HuellaComando, VersionEsperada: orden.VersionEsperada, Registro: cambio.Nuevo, RegistradoEn: instante, AuditoriaRef: "auditoria:prueba", EventoRef: "evento:prueba"}
	if r.mutar {
		out.Registro.Hecho.PersonaRef = "persona:ajena"
	}
	r.recibo = &out
	return ports.ResultadoOperacion{Codigo: "confirmada", AuditoriaRef: "auditoria:prueba", Anterior: r.actual, Recibo: r.recibo}, nil
}

type auditoriaPrueba struct {
	llamadas int
	err      error
	ultima   vec.AuditEntry
}

func (a *auditoriaPrueba) AppendAudit(_ context.Context, e vec.AuditEntry) (vec.AuditEntry, error) {
	a.llamadas++
	a.ultima = e
	return e, a.err
}
