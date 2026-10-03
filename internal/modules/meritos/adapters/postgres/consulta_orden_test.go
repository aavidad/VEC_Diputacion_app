package postgres

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Material estructural sintético para probar el adaptador; no sustituye el
// consumo de la concesión ni su verificación criptográfica en PostgreSQL.
func consultaOrdenPrueba(t *testing.T) (ports.OrdenConsultaPropia, ports.ResultadoConsultaPropia) {
	t.Helper()
	instante := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	persona, perfil, hecho := "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", "hecho:consulta:prueba"
	selector, err := json.Marshal(struct {
		Esquema string `json:"esquema"`
		Hecho   string `json:"hecho_ref"`
		Persona string `json:"persona_ref"`
	}{application.EsquemaConsultaPropia, hecho, persona})
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(selector)
	huella := hex.EncodeToString(suma[:])
	datos := pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: persona, PerfilRef: perfil,
		Accion: application.AccionConsultaPropia, Finalidad: application.FinalidadConsultaPropia,
		Campos: []string{"hecho_actual", "recibo_consulta"}, Obligaciones: []string{"auditar"}, DecisionRef: "decision:rum:consulta",
		Recurso: vec.RecursoAutorizable{Referencia: hecho, ModuloID: "meritos", Tipo: "hecho",
			Ambitos: map[string]string{"persona_ref": persona, "huella_consulta_sha256": huella}}}
	c, err := pruebas.NuevaConcesionV3Prueba(datos)
	if err != nil {
		t.Fatal(err)
	}
	d, err := c.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, persona, perfil, vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	dc, _ := vec.RepresentacionCanonicaDecisionAutorizacionV3(c.Decision)
	mc, _ := vec.RepresentacionCanonicaMotivoAutorizacionV2(d.ReferenciaMotivo)
	dh, _ := vec.HuellaSHA256DecisionAutorizacionV3(c.Decision)
	mh, _ := vec.HuellaSHA256MotivoAutorizacionV2(d.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(datos.DecisionRef, dh, mh, ctx.RegistroContextoRef, ctx.HuellaSHA256,
		application.AccionConsultaPropia, hecho, rh, application.AudienciaConsultaPropia, instante.Add(time.Second), instante.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		t.Fatal(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'c'}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, dc, mc,
		ctx.RepresentacionCanonica, ctx.Contexto.Instantanea.PersonaVersion, ctx.Contexto.Instantanea.PerfilVersion,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	orden := ports.OrdenConsultaPropia{SelectorCanonico: selector, HechoRef: hecho, PersonaRef: persona, HuellaConsultaSHA256: huella, Motivo: d.ReferenciaMotivo,
		Autorizacion: ports.AutorizacionOperacion{Contexto: ctx, Solicitud: c.Solicitud, Decision: c.Decision, Confirmacion: c.Confirmacion, Material: m}}
	correlacion, _ := d.Correlacion.ValorCanonico()
	ficha := &ports.FichaHechoPropio{Referencia: hecho, Version: 3, Tipo: "titulacion", ConceptoRef: "titulo:prueba", Denominacion: "Título de prueba",
		Procedencia: domain.Procedencia{FuenteRef: "fuente:prueba", Version: "1", HechoOrigenRef: "origen:prueba", CapturadaEn: instante.Format(time.RFC3339)},
		Vigencia:    domain.Vigencia{Desde: "2026-06-01"}, Estado: domain.Declarado, Evidencias: []vec.ReferenciaDocumento{}}
	consumo := hex.EncodeToString(bytes.Repeat([]byte{3}, 32))
	recibo := &ports.ReciboConsultaPropia{Referencia: "recibo:consulta:prueba", HechoRef: hecho, VersionConsultada: 3,
		DecisionRef: datos.DecisionRef, ConsumoHuellaSHA256: consumo, AuditoriaRef: "aud_v3_consulta_prueba", CorrelacionRef: correlacion,
		ConsultadaEn: instante.Add(2 * time.Second)}
	resultado := ports.ResultadoConsultaPropia{Codigo: "obtenida", HechoActual: ficha, ReciboConsulta: recibo}
	return orden, resultado
}
