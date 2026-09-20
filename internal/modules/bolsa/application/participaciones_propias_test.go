package application

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type relojParticipacionesPrueba struct{ ahora time.Time }

func (r relojParticipacionesPrueba) Ahora() time.Time { return r.ahora }

type autorizadorParticipacionesPrueba struct {
	accion   string
	recurso  dominiovec.RecursoAutorizable
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a *autorizadorParticipacionesPrueba) AutorizarOperacion(_ context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.accion, a.recurso = accion, recurso
	return a.material, nil
}

type persistenciaParticipacionesPrueba struct {
	consulta  puertosbolsa.ConsultaParticipacionesPropias
	material  puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	resultado puertosbolsa.ResultadoParticipacionesPropias
}

func (p *persistenciaParticipacionesPrueba) ConsultarParticipacionesPropias(_ context.Context, c puertosbolsa.ConsultaParticipacionesPropias, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ResultadoParticipacionesPropias, error) {
	p.consulta, p.material = c, m
	return p.resultado, nil
}

func TestServicioParticipacionesPropiasDerivaCandidatoAutorizaYEntregaMaterialCompleto(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	actor := actorParticipacionesPrueba(t, ahora)
	consulta, err := puertosbolsa.NuevaConsultaParticipacionesPropias("can_"+strings.Repeat("c", 22), ahora)
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := puertosbolsa.RecursoAutorizableParticipacionesPropias(consulta)
	if err != nil {
		t.Fatal(err)
	}
	material := materialParticipacionesPrueba(t, recurso, ahora)
	persistencia := &persistenciaParticipacionesPrueba{resultado: puertosbolsa.ResultadoParticipacionesPropias{Esquema: puertosbolsa.EsquemaParticipacionesPropiasV1, ConsultadaEn: ahora, Participaciones: []puertosbolsa.ParticipacionPropia{{BolsaRef: "bol_" + strings.Repeat("b", 22), CategoriaRef: "cat_" + strings.Repeat("a", 22), VersionBolsa: 2, Orden: 3, TotalParticipaciones: 12, EstadoBolsa: "activa", VigenteDesde: ahora.Add(-time.Hour)}}}}
	autorizador := &autorizadorParticipacionesPrueba{material: material}
	servicio, err := NuevoServicioParticipacionesPropias(autorizador, persistencia, relojParticipacionesPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := servicio.Consultar(context.Background(), OrdenConsultaParticipacionesPropias{ContextoActor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Esquema != puertosbolsa.EsquemaParticipacionesPropiasV1 || autorizador.accion != puertosbolsa.AccionConsultarParticipacionesPropias || autorizador.recurso.Atributos["finalidad"] != puertosbolsa.FinalidadConsultarParticipacionesPropias {
		t.Fatalf("contrato de autorizacion incorrecto: %#v", autorizador)
	}
	candidato, _ := persistencia.consulta.CandidatoRef()
	if candidato != "can_"+strings.Repeat("c", 22) || persistencia.material.ValidarEstructura() != nil {
		t.Fatalf("persistencia no recibio contexto/material completo")
	}
}

func TestServicioParticipacionesPropiasDeniegaSinUnCandidatoVigente(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	actor := actorParticipacionesPrueba(t, ahora)
	actor.Instantanea.Vinculos[0].VigenteHasta = ahora
	servicio, err := NuevoServicioParticipacionesPropias(&autorizadorParticipacionesPrueba{}, &persistenciaParticipacionesPrueba{}, relojParticipacionesPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.Consultar(context.Background(), OrdenConsultaParticipacionesPropias{ContextoActor: actor}); err == nil {
		t.Fatal("se admitio candidato no vigente")
	}
}

func actorParticipacionesPrueba(t *testing.T, ahora time.Time) dominiovec.ContextoActor {
	t.Helper()
	ref := func(p, c string) string { return p + strings.Repeat(c, 22) }
	i := dominiovec.InstantaneaContextoActor{VinculoRef: ref("vca_", "v"), VinculoVersion: 1, CuentaRef: ref("cta_", "a"), PersonaRef: ref("per_", "p"), PersonaVersion: 1, PerfilActivoRef: ref("prf_", "r"), PerfilVersion: 1, Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []dominiovec.VinculoReferenciaContextoActor{{VinculoRef: ref("vin_", "i"), Version: 1, Tipo: dominiovec.TipoReferenciaContextoActorCandidato, Referencia: ref("can_", "c"), Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	a, err := dominiovec.NuevoContextoActor(dominiovec.CuentaAutenticadaContextoActor{CuentaRef: i.CuentaRef, Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh}, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func materialParticipacionesPrueba(t *testing.T, r dominiovec.RecursoAutorizable, ahora time.Time) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h := strings.Repeat("a", 64)
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", h, h, "contexto:prueba", h, puertosbolsa.AccionConsultarParticipacionesPropias, r.Referencia, huella, puertosbolsa.AudienciaParticipacionesPropias, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	m, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("a"), []byte("a"), []byte("a"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
