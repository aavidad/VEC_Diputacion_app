package auditoria

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registroConsultaV3Prueba struct{ registradaEn time.Time }

func (r registroConsultaV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context, ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return r.registradaEn, nil
}

// La fixture usa constructores V3 reales y datos sintéticos. El registrador
// doble sólo permite probar la ligadura local; no acredita I/O durable ni firma.
func consultaAutorizadaAlcancePrueba(t *testing.T, fuente, claveAlcance, valorAlcance string) (ConsultaAutorizada, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	peticion := peticionAuditoriaIdentidadPrueba(t, ahora)
	filtro := peticion.Filtro
	filtro.Fuente = fuente
	if fuente == "bolsa" {
		filtro.ExpedienteRef = "participacion:bolsa:prueba"
	}
	recurso, err := RecursoFiltro(filtro)
	if err != nil {
		t.Fatal(err)
	}
	if claveAlcance != "expediente_ref" {
		recurso.Ambitos = map[string]string{"fuente": fuente, claveAlcance: valorAlcance}
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: peticion.Contexto.Vinculo,
		ReferenciaMotivo:          peticion.Contexto.Motivo,
		Accion:                    AccionConsultar,
		Recurso:                   recurso,
		Finalidad:                 filtro.FinalidadRef,
		Correlacion:               peticion.Contexto.Correlacion,
	})
	if err != nil {
		t.Fatalf("solicitud ligada V3: %v", err)
	}
	actor := peticion.Contexto.Resultado.Contexto
	version := domain.VersionRol{RolID: "auditor_consulta_sintetica", Version: 1, Nombre: "Auditor sintético",
		Estado: domain.EstadoVersionRolPublicada, PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
		Concesiones: []domain.ConcesionRol{{Accion: AccionConsultar, ModuloID: ModuloAutorizacion, TipoRecurso: TipoRecurso,
			Finalidades: []string{filtro.FinalidadRef}, GarantiaMinima: domain.AuthAssuranceSubstantial,
			CamposPermitidos: CamposPermitidos()}},
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{AsignacionID: "asig-auditoria-sintetica", Version: 1,
			PerfilActivoRef: actor.Instantanea.PerfilActivoRef, PrincipalID: actor.Instantanea.PersonaRef,
			VersionRolRef: version.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos: []domain.AmbitoPerfil{{Clave: "fuente", Valores: []string{fuente}},
				{Clave: claveAlcance, Valores: []string{recurso.Ambitos[claveAlcance]}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour)},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea,
		"dec_0123456789abcdef0123456789abcdef", ahora, ahora.Add(90*time.Second))
	if err != nil {
		t.Fatalf("evidencia V3: %v", err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatalf("decisión V3: %v", err)
	}
	if concedida, _, err := decision.Resultado(); err != nil || !concedida {
		t.Fatalf("fixture V3 sin concesión: %v", err)
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud, decision, peticion.Contexto.Motivo, peticion.Contexto.Resultado)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), registroConsultaV3Prueba{registradaEn: ahora.Add(time.Second)}, orden)
	if err != nil {
		t.Fatal(err)
	}
	datosConfirmacion, err := confirmacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	decisionCanonica, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	motivoCanonico, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(peticion.Contexto.Motivo)
	if err != nil {
		t.Fatal(err)
	}
	huellaDecision, err := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(peticion.Contexto.Motivo)
	if err != nil {
		t.Fatal(err)
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(datosConfirmacion.DecisionRef,
		huellaDecision, huellaMotivo, peticion.Contexto.Resultado.RegistroContextoRef,
		peticion.Contexto.Resultado.HuellaSHA256, AccionConsultar, recurso.Referencia, huellaRecurso,
		AudienciaConsumo, ahora.Add(2*time.Second), ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		t.Fatal(err)
	}
	material, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(make([]byte, 512), resumen,
		decisionCanonica, motivoCanonico, peticion.Contexto.Resultado.RepresentacionCanonica,
		actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion,
		[]byte("payload-sintetico"), []byte("sobre-sintetico"), []byte("evidencia-sintetica"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ConsultaAutorizada{Filtro: filtro, Material: material, Solicitud: solicitud, Decision: decision,
		Confirmacion: confirmacion, ResultadoContexto: peticion.Contexto.Resultado}, ahora.Add(3 * time.Second)
}

func TestConsultaAutorizadaV3AdmiteAmbitosHistoricoYNominalesExactos(t *testing.T) {
	for _, caso := range []struct{ nombre, fuente, clave, valor string }{
		{"historico_ct", "ct", "expediente_ref", ""},
		{"nominal_ct", "ct", "organizacion_ref", "organizacion:dipgra"},
		{"nominal_bolsa", "bolsa", "bolsa_ref", "bolsa:prueba"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q, ahora := consultaAutorizadaAlcancePrueba(t, caso.fuente, caso.clave, caso.valor)
			if err := ValidarConsultaAutorizadaEn(q, ahora); err != nil {
				t.Fatalf("V3 nominal exacto rechazado: %v", err)
			}
		})
	}
}

func TestRecursoConsultaRechazaCrucesExtrasYComodines(t *testing.T) {
	q, _ := consultaAutorizadaAlcancePrueba(t, "ct", "organizacion_ref", "organizacion:dipgra")
	datos, err := q.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	base := datos.Recurso
	if validarRecursoConsultaLigadoAlFiltro(base, q.Filtro) != nil {
		t.Fatal("fixture nominal CT inválida")
	}
	casos := []struct {
		nombre string
		mutar  func(*domain.RecursoAutorizable)
	}{
		{"referencia", func(r *domain.RecursoAutorizable) { r.Referencia = "otro:expediente" }},
		{"modulo", func(r *domain.RecursoAutorizable) { r.ModuloID = "contratacion_temporal" }},
		{"tipo", func(r *domain.RecursoAutorizable) { r.Tipo = "otro" }},
		{"atributo_extra", func(r *domain.RecursoAutorizable) { r.Atributos["extra"] = "valor" }},
		{"atributo_vacio", func(r *domain.RecursoAutorizable) { r.Atributos = nil }},
		{"huella", func(r *domain.RecursoAutorizable) { r.Atributos["filtro_sha256"] = strings.Repeat("a", 64) }},
		{"ambito_extra", func(r *domain.RecursoAutorizable) { r.Ambitos["unidad_ref"] = "otra" }},
		{"ambito_vacio", func(r *domain.RecursoAutorizable) { r.Ambitos = nil }},
		{"cruce_bolsa", func(r *domain.RecursoAutorizable) {
			r.Ambitos = map[string]string{"fuente": "ct", "bolsa_ref": "bolsa:otra"}
		}},
		{"fuente_ajena", func(r *domain.RecursoAutorizable) { r.Ambitos["fuente"] = "bolsa" }},
		{"organizacion_vacia", func(r *domain.RecursoAutorizable) { r.Ambitos["organizacion_ref"] = "" }},
		{"organizacion_comodin", func(r *domain.RecursoAutorizable) { r.Ambitos["organizacion_ref"] = "organizacion:*" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := base
			r.Ambitos = map[string]string{}
			for k, v := range base.Ambitos {
				r.Ambitos[k] = v
			}
			r.Atributos = map[string]string{}
			for k, v := range base.Atributos {
				r.Atributos[k] = v
			}
			caso.mutar(&r)
			if validarRecursoConsultaLigadoAlFiltro(r, q.Filtro) == nil {
				t.Fatalf("recurso adulterado admitido: %+v", r)
			}
		})
	}
}

func TestRecursoConsultaBolsaRechazaAmbitoCTYBolsaNoSingular(t *testing.T) {
	q, _ := consultaAutorizadaAlcancePrueba(t, "bolsa", "bolsa_ref", "bolsa:prueba")
	datos, err := q.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for _, ambitos := range []map[string]string{
		{"fuente": "bolsa", "organizacion_ref": "organizacion:otra"},
		{"fuente": "bolsa", "bolsa_ref": "bolsa:*"},
		{"fuente": "bolsa", "bolsa_ref": ""},
		{"fuente": "bolsa", "bolsa_ref": "bolsa:prueba", "expediente_ref": q.Filtro.ExpedienteRef},
	} {
		recurso := datos.Recurso
		recurso.Ambitos = ambitos
		if validarRecursoConsultaLigadoAlFiltro(recurso, q.Filtro) == nil {
			t.Fatalf("alcance Bolsa adulterado admitido: %v", ambitos)
		}
	}
}

func TestConsultaAutorizadaV3ConservaLigaduraFiltroActorYVentana(t *testing.T) {
	q, ahora := consultaAutorizadaAlcancePrueba(t, "ct", "organizacion_ref", "organizacion:dipgra")
	for _, caso := range []struct {
		nombre string
		mutar  func(*ConsultaAutorizada)
	}{
		{"expediente", func(q *ConsultaAutorizada) { q.Filtro.ExpedienteRef = "expediente:ct:otro" }},
		{"filtro", func(q *ConsultaAutorizada) { q.Filtro.Limite++ }},
		{"cursor", func(q *ConsultaAutorizada) {
			q.Filtro.Antes = Posicion{OcurridoEn: q.Filtro.Desde.Add(time.Minute), Fuente: "ct", ID: "evento:anterior"}
		}},
		{"motivo", func(q *ConsultaAutorizada) { q.Filtro.MotivoRef = "motivo:otro" }},
		{"actor", func(q *ConsultaAutorizada) {
			q.ResultadoContexto.Contexto.Instantanea.PersonaRef = "per_otra_persona_sintetica"
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := q
			caso.mutar(&alterada)
			if err := ValidarConsultaAutorizadaEn(alterada, ahora); err == nil {
				t.Fatal("consulta V3 adulterada admitida")
			}
		})
	}
	if err := ValidarConsultaAutorizadaEn(q, ahora.Add(6*time.Second)); err == nil {
		t.Fatal("material V3 caducado admitido")
	}
}
