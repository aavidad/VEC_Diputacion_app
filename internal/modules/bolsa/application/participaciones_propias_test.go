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
	llamadas  int
	vinculo   dominiovec.VinculoAutenticacionActorV2
	resultado dominiovec.ResultadoContextoActorRegistradoV2
	accion    string
	recurso   dominiovec.RecursoAutorizable
	material  puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a *autorizadorParticipacionesPrueba) AutorizarOperacion(_ context.Context, vinculo dominiovec.VinculoAutenticacionActorV2, resultado dominiovec.ResultadoContextoActorRegistradoV2, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.vinculo, a.resultado, a.accion, a.recurso = vinculo, resultado, accion, recurso
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
	par := vinculoResultadoParticipacionesPrueba(t, actor, ahora)
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
	resultado, err := servicio.Consultar(context.Background(), OrdenConsultaParticipacionesPropias{Vinculo: par.vinculo, Resultado: par.resultado})
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Esquema != puertosbolsa.EsquemaParticipacionesPropiasV1 || autorizador.llamadas != 1 || autorizador.accion != puertosbolsa.AccionConsultarParticipacionesPropias || autorizador.recurso.Atributos["finalidad"] != puertosbolsa.FinalidadConsultarParticipacionesPropias || !autorizador.vinculo.CoincideExactamenteCon(par.vinculo) || autorizador.resultado.Validar() != nil || autorizador.vinculo.ValidarPara(autorizador.resultado) != nil || autorizador.resultado.RegistroContextoRef != par.resultado.RegistroContextoRef {
		t.Fatalf("contrato de autorizacion incorrecto: %#v", autorizador)
	}
	candidato, _ := persistencia.consulta.CandidatoRef()
	if candidato != "can_"+strings.Repeat("c", 22) || persistencia.material.ValidarEstructura() != nil {
		t.Fatalf("persistencia no recibio contexto/material completo")
	}
}

func TestServicioParticipacionesPropiasDeniegaVinculoResultadoCruzadosYCaducados(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	actor := actorParticipacionesPrueba(t, ahora)
	par := vinculoResultadoParticipacionesPrueba(t, actor, ahora)
	otroActor := actorParticipacionesPrueba(t, ahora)
	otroActor.Instantanea.CuentaRef = "cta_" + strings.Repeat("z", 22)
	otroActor.Principal.ID = "per_" + strings.Repeat("z", 22)
	otroActor.PersonaRef = otroActor.Principal.ID
	otroActor.Instantanea.PersonaRef = otroActor.PersonaRef
	otro := vinculoResultadoParticipacionesPrueba(t, otroActor, ahora)
	for nombre, caso := range map[string]struct {
		orden OrdenConsultaParticipacionesPropias
		reloj time.Time
	}{
		"vinculo_resultado_cruzados": {OrdenConsultaParticipacionesPropias{Vinculo: par.vinculo, Resultado: otro.resultado}, ahora},
		"vinculo_caducado":           {OrdenConsultaParticipacionesPropias{Vinculo: par.vinculo, Resultado: par.resultado}, ahora.Add(21 * time.Minute)},
	} {
		t.Run(nombre, func(t *testing.T) {
			autorizador := &autorizadorParticipacionesPrueba{}
			servicio, err := NuevoServicioParticipacionesPropias(autorizador, &persistenciaParticipacionesPrueba{}, relojParticipacionesPrueba{caso.reloj})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = servicio.Consultar(context.Background(), caso.orden); err == nil || autorizador.llamadas != 0 {
				t.Fatalf("se autorizo par invalido: error=%v llamadas=%d", err, autorizador.llamadas)
			}
		})
	}
}

type parVinculoResultadoParticipacionesPrueba struct {
	vinculo   dominiovec.VinculoAutenticacionActorV2
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

type revalidadorParticipacionesPrueba struct {
	autenticacion dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorParticipacionesPrueba) RevalidarAutenticacionActorV1(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.autenticacion, nil
}

type resolutorParticipacionesPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorParticipacionesPrueba) ResolverContextoActorRegistradoV2(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

func vinculoResultadoParticipacionesPrueba(t *testing.T, actor dominiovec.ContextoActor, ahora time.Time) parVinculoResultadoParticipacionesPrueba {
	t.Helper()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + strings.Repeat("x", 22), ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("4", 64), ProcedenciaAutoridad: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	i := actor.Instantanea
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{Esquema: dominiovec.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{CuentaRef: i.CuentaRef, Version: i.CuentaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Persona: dominiovec.ProcedenciaPersonaContextoActorV1{PersonaRef: i.PersonaRef, Version: i.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{PerfilRef: i.PerfilActivoRef, Version: i.PerfilVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{VinculoRef: i.VinculoRef, Version: i.VinculoVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Vinculos: make([]dominiovec.ProcedenciaVinculoReferenciaContextoActorV1, len(i.Vinculos))}
	for n, v := range i.Vinculos {
		manifiesto.Vinculos[n] = dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{VinculoRef: v.VinculoRef, Version: v.Version, Tipo: v.Tipo, Referencia: v.Referencia, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanon)
	if err != nil {
		t.Fatal(err)
	}
	if err := actor.Validar(); err != nil {
		t.Fatalf("actor de prueba inválido: %v", err)
	}
	if err := manifiesto.ValidarParaContexto(actor); err != nil {
		t.Fatalf("manifiesto de prueba inválido: %v", err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + strings.Repeat("r", 24), Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: manifiestoCanon, ManifiestoProcedenciaHuellaSHA256: manifiestoHuella, AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: actor.ResueltoEn}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + strings.Repeat("a", 22), AutenticacionHuellaSHA256: strings.Repeat("1", 64), AsercionRef: "ase_" + strings.Repeat("s", 22), SesionRef: "ses_" + strings.Repeat("s", 22), ControlSesionRef: "cse_" + strings.Repeat("c", 22), ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: i.CuentaRef, CuentaOrdinariaRef: i.CuentaRef, Superficie: dominiovec.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: actor.Principal.AuthMethod, GarantiaObservada: actor.Principal.AuthAssurance, PoliticaGarantiaRef: "pga_" + strings.Repeat("p", 22), PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64), AutenticacionVerificadaEn: ahora.Add(-2 * time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Minute), SesionValidaHasta: ahora.Add(20 * time.Minute)}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado de prueba inválido: %v", err)
	}
	if err := autenticacion.Validar(); err != nil {
		t.Fatalf("autenticación de prueba inválida: %v", err)
	}
	vinculo, resultadoLigado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorParticipacionesPrueba{autenticacion}, dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef}, resolutorParticipacionesPrueba{resultado}, dominiovec.SolicitudContextoActor{Cuenta: dominiovec.CuentaAutenticadaContextoActor{CuentaRef: i.CuentaRef, Metodo: actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance}, PerfilActivoRef: i.PerfilActivoRef}, relojParticipacionesPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return parVinculoResultadoParticipacionesPrueba{vinculo: vinculo, resultado: resultadoLigado}
}

func actorParticipacionesPrueba(t *testing.T, ahora time.Time) dominiovec.ContextoActor {
	t.Helper()
	ref := func(p, c string) string { return p + strings.Repeat(c, 22) }
	i := dominiovec.InstantaneaContextoActor{VinculoRef: ref("vca_", "v"), VinculoVersion: 1, CuentaRef: ref("cta_", "a"), CuentaVersion: 1, PersonaRef: ref("per_", "p"), PersonaVersion: 1, PerfilActivoRef: ref("prf_", "r"), PerfilVersion: 1, Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []dominiovec.VinculoReferenciaContextoActor{{VinculoRef: ref("vin_", "i"), Version: 1, Tipo: dominiovec.TipoReferenciaContextoActorCandidato, Referencia: ref("can_", "c"), Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
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
