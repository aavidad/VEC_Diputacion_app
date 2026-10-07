package postgres

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Esta fábrica se usa solo para pruebas de traducción del adaptador. No
// simula un PostgreSQL positivo ni acredita firma/consumo de las preimágenes.
func ordenPrueba(t *testing.T) (ports.OrdenOperacion, ports.ResultadoOperacion) {
	t.Helper()
	instante := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	persona, perfil := "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl"
	accion, finalidad := "meritos.hecho.declarar", "declaracion_hecho_propio"
	datos := pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: persona, PerfilRef: perfil, Accion: accion, Finalidad: finalidad,
		Campos: []string{"hecho", "declarante_ref", "version", "recibo"}, Obligaciones: []string{"auditar"}, DecisionRef: "decision:rum:adapter",
		Recurso: vec.RecursoAutorizable{Referencia: "hecho:prueba", ModuloID: "meritos", Tipo: "hecho", Ambitos: map[string]string{"persona_ref": persona}}}
	seed, err := pruebas.NuevaConcesionV3Prueba(datos)
	if err != nil {
		t.Fatal(err)
	}
	solicitudSeed, err := seed.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	h := domain.Hecho{Referencia: "hecho:prueba", PersonaRef: persona, Version: 1, Tipo: "titulacion", ConceptoRef: "titulo:prueba", Denominacion: "Título de prueba",
		Procedencia: domain.Procedencia{FuenteRef: "fuente:prueba", Version: "1", HechoOrigenRef: "origen:prueba", CapturadaEn: instante.Format(time.RFC3339)}, Vigencia: domain.Vigencia{Desde: "2026-06-01"}, Estado: domain.Declarado, Evidencias: []vec.ReferenciaDocumento{}}
	comando := domain.ComandoHecho{Esquema: domain.EsquemaComandoHecho, Accion: accion, ActorRef: persona, ClaveIdempotencia: "clave:prueba", Hecho: h, Motivo: solicitudSeed.ReferenciaMotivo, FechaCorte: "2026-10-01"}
	raw, err := comando.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(raw)
	huella := hex.EncodeToString(suma[:])
	datos.Recurso.Ambitos = map[string]string{"persona_ref": persona, "version_esperada": "0", "huella_comando_sha256": huella}
	c, err := pruebas.NuevaConcesionV3Prueba(datos)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, persona, perfil, vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	dc, _ := vec.RepresentacionCanonicaDecisionAutorizacionV3(c.Decision)
	mc, _ := vec.RepresentacionCanonicaMotivoAutorizacionV2(comando.Motivo)
	dh, _ := vec.HuellaSHA256DecisionAutorizacionV3(c.Decision)
	mh, _ := vec.HuellaSHA256MotivoAutorizacionV2(comando.Motivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(datos.DecisionRef, dh, mh, ctx.RegistroContextoRef, ctx.HuellaSHA256, accion, h.Referencia, rh, "vec_meritos.hecho.declarar.v1", instante.Add(time.Second), instante.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		t.Fatal(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'b'}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, dc, mc, ctx.RepresentacionCanonica, ctx.Contexto.Instantanea.PersonaVersion, ctx.Contexto.Instantanea.PerfilVersion, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	orden := ports.OrdenOperacion{Accion: accion, ActorRef: persona, ClaveIdempotencia: comando.ClaveIdempotencia, HuellaComando: huella, Hecho: h, Motivo: comando.Motivo, FechaCorte: comando.FechaCorte,
		Autorizacion: ports.AutorizacionOperacion{Contexto: ctx, Solicitud: c.Solicitud, Decision: c.Decision, Confirmacion: c.Confirmacion, Material: m}}
	recibo := ports.Recibo{Referencia: "recibo:prueba", Accion: accion, ActorRef: persona, ClaveIdempotencia: comando.ClaveIdempotencia, HuellaComando: huella,
		Registro: ports.RegistroActual{Hecho: h, DeclaranteRef: persona}, RegistradoEn: instante, AuditoriaRef: "auditoria:prueba", EventoRef: "evento:prueba"}
	return orden, ports.ResultadoOperacion{Codigo: "confirmada", AuditoriaRef: "auditoria:prueba", Recibo: &recibo}
}
