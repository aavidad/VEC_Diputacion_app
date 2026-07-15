package domain

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMensajeAtestacionAutorizacionV1VectorDeterminista(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV1Prueba()
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	original := clonarDecisionAtestacionAutorizacionV1Prueba(decision)

	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("serializar mensaje VEC-AD-1: %v", err)
	}
	if !reflect.DeepEqual(decision, original) {
		t.Fatal("la serializacion modifico la decision recibida")
	}
	prefijo := append([]byte(EsquemaMensajeAtestacionAutorizacionV1), 0)
	if !bytes.HasPrefix(mensaje, prefijo) {
		t.Fatalf("dominio criptografico ausente: %x", mensaje[:min(len(mensaje), len(prefijo))])
	}
	posicionVersion := len(prefijo)
	if version := binary.BigEndian.Uint16(mensaje[posicionVersion : posicionVersion+2]); version != VersionFormatoAtestacionAutorizacionV1 {
		t.Fatalf("version binaria = %d", version)
	}
	if longitud := binary.BigEndian.Uint64(mensaje[len(mensaje)-8:]); longitud != uint64(len(mensaje)) {
		t.Fatalf("longitud final = %d; bytes completos = %d", longitud, len(mensaje))
	}

	huella, err := HuellaSHA256MensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("calcular huella: %v", err)
	}
	const longitudEsperada = 1884
	const huellaEsperada = "1b0f9ffda063000914d21cb7d4e98e9eb069af442b2d9bd6b9516546fddb6308"
	if len(mensaje) != longitudEsperada || huella != huellaEsperada {
		t.Fatalf("cambio incompatible del vector: longitud=%d huella=%s; revisar esquema y publicar nueva version", len(mensaje), huella)
	}

	// El orden de insercion de un mapa Go no cambia su representacion.
	otra := clonarDecisionAtestacionAutorizacionV1Prueba(decision)
	otra.PoliticasEvaluadasHuellasSHA256 = map[string]string{
		otra.PoliticasEvaluadasRefs[1]: decision.PoliticasEvaluadasHuellasSHA256[otra.PoliticasEvaluadasRefs[1]],
		otra.PoliticasEvaluadasRefs[0]: decision.PoliticasEvaluadasHuellasSHA256[otra.PoliticasEvaluadasRefs[0]],
	}
	otra.PoliticasHuellasSHA256 = map[string]string{
		otra.PoliticasRefs[0]: decision.PoliticasHuellasSHA256[otra.PoliticasRefs[0]],
	}
	otroMensaje, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, otra)
	if err != nil || !bytes.Equal(mensaje, otroMensaje) {
		t.Fatalf("el orden fisico del mapa altero VEC-AD-1: err=%v", err)
	}
}

func TestMensajeAtestacionAutorizacionV1LigaCabeceraPreseleccionada(t *testing.T) {
	base := cabeceraAtestacionAutorizacionV1Prueba()
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	huellaBase, err := HuellaSHA256MensajeAtestacionAutorizacionV1(base, decision)
	if err != nil {
		t.Fatalf("huella base: %v", err)
	}
	cambios := []struct {
		nombre string
		mutar  func(*CabeceraAtestacionAutorizacionV1)
	}{
		{"suite", func(c *CabeceraAtestacionAutorizacionV1) { c.Suite = "VEC-AD-OTRA-SUITE-1" }},
		{"clave", func(c *CabeceraAtestacionAutorizacionV1) { c.ClaveID = "clave:prueba:2026-02" }},
		{"audiencia", func(c *CabeceraAtestacionAutorizacionV1) { c.Audiencia = "vec-diputacion/otro/vec/autorizacion" }},
	}
	for _, cambio := range cambios {
		t.Run(cambio.nombre, func(t *testing.T) {
			candidata := base
			cambio.mutar(&candidata)
			huella, err := HuellaSHA256MensajeAtestacionAutorizacionV1(candidata, decision)
			if err != nil || huella == huellaBase {
				t.Fatalf("cabecera no ligada: huella=%q err=%v", huella, err)
			}
		})
	}
	versionDistinta := base
	versionDistinta.FormatoVersion++
	if _, err := SerializarMensajeAtestacionAutorizacionV1(versionDistinta, decision); !errors.Is(err, ErrConfiguracionAccesoInvalida) ||
		!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("version desconocida aceptada: %v", err)
	}
}

func TestEsquemaAtestacionAutorizacionV1EnumeraDecisionYBloqueObligatorio(t *testing.T) {
	esperados := []string{
		"decision_ref", "concedida", "codigo", "principal_id", "perfil_activo_ref",
		"accion", "recurso_ref", "modulo_id", "tipo_recurso", "contexto_recurso_huella_sha256",
		"finalidad", "correlacion_ref", "vinculo_autenticacion_actor", "asignacion_ref", "asignacion_huella_sha256",
		"version_rol_ref", "version_rol_huella_sha256", "control_vigencia_version_rol_ref",
		"control_vigencia_version_rol_revision", "control_vigencia_version_rol_huella_sha256",
		"revision_catalogo_politicas", "catalogo_politicas_huella_sha256", "politicas_evaluadas_refs",
		"politicas_evaluadas_huellas_sha256", "politicas_refs", "politicas_huellas_sha256",
		"garantia_minima", "campos_permitidos", "obligaciones", "emitida_en", "valida_hasta",
	}
	tipo := reflect.TypeOf(DecisionAutorizacion{})
	if tipo.NumField() != len(esperados) {
		t.Fatalf("DecisionAutorizacion tiene %d campos; VEC-AD-1 enumera %d: un cambio exige version nueva", tipo.NumField(), len(esperados))
	}
	for indice, esperado := range esperados {
		campo := tipo.Field(indice)
		etiqueta := strings.Split(campo.Tag.Get("json"), ",")[0]
		if campo.PkgPath != "" || etiqueta != esperado {
			t.Fatalf("campo %d = %s/%q; esperado exportado con json %q", indice, campo.Name, etiqueta, esperado)
		}
	}
}

func TestMensajeAtestacionAutorizacionV1SeDecodificaConEsquemaIndependiente(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV1Prueba()
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("mensaje base: %v", err)
	}

	lector := nuevoLectorAtestacionAutorizacionV1Prueba(t, mensaje)
	lector.exigirBytes(append([]byte(EsquemaMensajeAtestacionAutorizacionV1), 0))
	lector.exigirUint16(cabecera.FormatoVersion)
	lector.exigirTexto(cabecera.Suite)
	lector.exigirTexto(cabecera.ClaveID)
	lector.exigirTexto(cabecera.Audiencia)

	lector.exigirTexto(decision.DecisionRef)
	lector.exigirBooleano(decision.Concedida)
	lector.exigirTexto(decision.Codigo)
	lector.exigirTexto(decision.PrincipalID)
	lector.exigirTexto(decision.PerfilActivoRef)
	lector.exigirTexto(decision.Accion)
	lector.exigirTexto(decision.RecursoRef)
	lector.exigirTexto(decision.ModuloID)
	lector.exigirTexto(decision.TipoRecurso)
	lector.exigirTexto(decision.ContextoRecursoHuellaSHA256)
	lector.exigirTexto(decision.Finalidad)
	lector.exigirTexto(decision.CorrelacionRef)
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatalf("obtener vinculo: %v", err)
	}
	lector.exigirUint16(vinculo.BloqueVersion)
	lector.exigirTexto(vinculo.AutenticacionRef)
	lector.exigirTexto(vinculo.AutenticacionHuellaSHA256)
	lector.exigirTexto(vinculo.AsercionRef)
	lector.exigirTexto(vinculo.SesionRef)
	lector.exigirTexto(vinculo.ControlSesionRef)
	lector.exigirUint64(vinculo.ControlSesionRevision)
	lector.exigirTexto(vinculo.ControlSesionHuellaSHA256)
	lector.exigirTexto(vinculo.CuentaRef)
	lector.exigirTexto(vinculo.CuentaOrdinariaRef)
	lector.exigirTexto(vinculo.PrincipalID)
	lector.exigirTexto(vinculo.PerfilActivoRef)
	lector.exigirBooleano(vinculo.CuentaPrivilegiada)
	lector.exigirTexto(string(vinculo.Superficie))
	lector.exigirTexto(string(vinculo.MetodoObservado))
	lector.exigirTexto(string(vinculo.GarantiaObservada))
	lector.exigirTexto(vinculo.PoliticaGarantiaRef)
	lector.exigirTexto(vinculo.PoliticaGarantiaHuellaSHA256)
	lector.exigirInstante(vinculo.AutenticacionVerificadaEn)
	lector.exigirInstante(vinculo.SesionEmitidaEn)
	lector.exigirInstante(vinculo.SesionValidaHasta)
	lector.exigirInstante(vinculo.SesionRevalidadaEn)
	lector.exigirTexto(vinculo.ContextoActorRef)
	lector.exigirUint64(vinculo.ContextoActorVersion)
	lector.exigirTexto(vinculo.ContextoActorHuellaSHA256)
	lector.exigirTexto(decision.AsignacionRef)
	lector.exigirTexto(decision.AsignacionHuellaSHA256)
	lector.exigirTexto(decision.VersionRolRef)
	lector.exigirTexto(decision.VersionRolHuellaSHA256)
	lector.exigirTexto(decision.ControlVigenciaVersionRolRef)
	lector.exigirUint64(decision.ControlVigenciaVersionRolRevision)
	lector.exigirTexto(decision.ControlVigenciaVersionRolHuellaSHA256)
	lector.exigirUint64(decision.RevisionCatalogoPoliticas)
	lector.exigirTexto(decision.CatalogoPoliticasHuellaSHA256)
	lector.exigirLista(decision.PoliticasEvaluadasRefs)
	lector.exigirMapa(decision.PoliticasEvaluadasHuellasSHA256)
	lector.exigirLista(decision.PoliticasRefs)
	lector.exigirMapa(decision.PoliticasHuellasSHA256)
	lector.exigirTexto(string(decision.GarantiaMinima))
	lector.exigirLista(decision.CamposPermitidos)
	lector.exigirLista(decision.Obligaciones)
	lector.exigirInstante(decision.EmitidaEn)
	lector.exigirInstante(decision.ValidaHasta)
	lector.exigirLongitudFinal()
}

func TestMensajeAtestacionAutorizacionV1MutaORechazaCadaCampo(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV1Prueba()
	base := decisionAtestacionAutorizacionV1Prueba(t)
	mensajeBase, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, base)
	if err != nil {
		t.Fatalf("mensaje base: %v", err)
	}
	mutaciones := []struct {
		campo         string
		debeSerValida bool
		mutar         func(*DecisionAutorizacion)
	}{
		{"decision_ref", true, func(d *DecisionAutorizacion) { d.DecisionRef = "decision:atestacion:otra" }},
		{"concedida", false, func(d *DecisionAutorizacion) { d.Concedida = false }},
		{"codigo", false, func(d *DecisionAutorizacion) { d.Codigo = "otro" }},
		{"principal_id", false, func(d *DecisionAutorizacion) { d.PrincipalID = "per_otra234567890abcdefghijkl" }},
		{"perfil_activo_ref", false, func(d *DecisionAutorizacion) { d.PerfilActivoRef = "prf_otra234567890abcdefghijkl" }},
		{"accion", true, func(d *DecisionAutorizacion) { d.Accion = "bolsa.merito.consultar" }},
		{"recurso_ref", true, func(d *DecisionAutorizacion) { d.RecursoRef = "merito:otro" }},
		{"modulo_id", true, func(d *DecisionAutorizacion) { d.ModuloID = "seleccion" }},
		{"tipo_recurso", true, func(d *DecisionAutorizacion) { d.TipoRecurso = "expediente" }},
		{"contexto_recurso_huella_sha256", true, func(d *DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = huellaAtestacionPrueba('9') }},
		{"finalidad", true, func(d *DecisionAutorizacion) { d.Finalidad = "auditar_bolsa" }},
		{"correlacion_ref", true, func(d *DecisionAutorizacion) { d.CorrelacionRef = "correlacion:otra" }},
		{"vinculo_autenticacion_actor", true, func(d *DecisionAutorizacion) {
			d.VinculoAutenticacionActor = vinculoAtestacionAutorizacionAlternativoPrueba(t, d.EmitidaEn)
		}},
		{"asignacion_ref", true, func(d *DecisionAutorizacion) { d.AsignacionRef = "asignacion:otra:v1" }},
		{"asignacion_huella_sha256", true, func(d *DecisionAutorizacion) { d.AsignacionHuellaSHA256 = huellaAtestacionPrueba('8') }},
		{"version_rol_ref", false, func(d *DecisionAutorizacion) { d.VersionRolRef = "rol:otro:v1" }},
		{"version_rol_huella_sha256", true, func(d *DecisionAutorizacion) { d.VersionRolHuellaSHA256 = huellaAtestacionPrueba('7') }},
		{"control_vigencia_version_rol_ref", false, func(d *DecisionAutorizacion) { d.ControlVigenciaVersionRolRef = "rol:otro:v1" }},
		{"control_vigencia_version_rol_revision", true, func(d *DecisionAutorizacion) { d.ControlVigenciaVersionRolRevision++ }},
		{"control_vigencia_version_rol_huella_sha256", true, func(d *DecisionAutorizacion) { d.ControlVigenciaVersionRolHuellaSHA256 = huellaAtestacionPrueba('6') }},
		{"revision_catalogo_politicas", true, func(d *DecisionAutorizacion) { d.RevisionCatalogoPoliticas++ }},
		{"catalogo_politicas_huella_sha256", false, func(d *DecisionAutorizacion) { d.CatalogoPoliticasHuellaSHA256 = huellaAtestacionPrueba('5') }},
		{"politicas_evaluadas_refs", true, func(d *DecisionAutorizacion) {
			anterior := d.PoliticasEvaluadasRefs[0]
			nueva := "politica:ambito2:v1"
			d.PoliticasEvaluadasRefs[0] = nueva
			d.PoliticasEvaluadasHuellasSHA256[nueva] = d.PoliticasEvaluadasHuellasSHA256[anterior]
			delete(d.PoliticasEvaluadasHuellasSHA256, anterior)
			d.CatalogoPoliticasHuellaSHA256, _ = HuellaEvidenciasCatalogoPoliticasAutorizacion(
				d.PoliticasEvaluadasRefs, d.PoliticasEvaluadasHuellasSHA256,
			)
		}},
		{"politicas_evaluadas_huellas_sha256", true, func(d *DecisionAutorizacion) {
			d.PoliticasEvaluadasHuellasSHA256[d.PoliticasEvaluadasRefs[0]] = huellaAtestacionPrueba('4')
			d.CatalogoPoliticasHuellaSHA256, _ = HuellaEvidenciasCatalogoPoliticasAutorizacion(
				d.PoliticasEvaluadasRefs, d.PoliticasEvaluadasHuellasSHA256,
			)
		}},
		{"politicas_refs", true, func(d *DecisionAutorizacion) {
			d.PoliticasRefs = []string{}
			d.PoliticasHuellasSHA256 = map[string]string{}
		}},
		{"politicas_huellas_sha256", false, func(d *DecisionAutorizacion) {
			d.PoliticasHuellasSHA256[d.PoliticasRefs[0]] = huellaAtestacionPrueba('3')
		}},
		{"garantia_minima", true, func(d *DecisionAutorizacion) { d.GarantiaMinima = AuthAssuranceSubstantial }},
		{"campos_permitidos", true, func(d *DecisionAutorizacion) { d.CamposPermitidos = append(d.CamposPermitidos, "fecha") }},
		{"obligaciones", true, func(d *DecisionAutorizacion) { d.Obligaciones = append(d.Obligaciones, "validar") }},
		{"emitida_en", true, func(d *DecisionAutorizacion) { d.EmitidaEn = d.EmitidaEn.Add(time.Microsecond) }},
		{"valida_hasta", true, func(d *DecisionAutorizacion) { d.ValidaHasta = d.ValidaHasta.Add(-time.Microsecond) }},
	}
	if len(mutaciones) != 31 {
		t.Fatalf("matriz contractual incompleta: %d campos", len(mutaciones))
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.campo, func(t *testing.T) {
			candidata := clonarDecisionAtestacionAutorizacionV1Prueba(base)
			mutacion.mutar(&candidata)
			mensaje, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, candidata)
			if mutacion.debeSerValida && err != nil {
				t.Fatalf("la mutacion debia conservar una concesion valida: %v", err)
			}
			if !mutacion.debeSerValida && err == nil {
				t.Fatal("la mutacion incoherente debia denegarse")
			}
			if err == nil && bytes.Equal(mensaje, mensajeBase) {
				t.Fatal("la mutacion conservo el mensaje firmado")
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV1NoAceptaSeparadorAnterior(t *testing.T) {
	if EsquemaMensajeAtestacionAutorizacionV1 == "VEC-AUTORIZACION-ATESTACION" {
		t.Fatal("el separador anterior sigue activo")
	}
	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(), decisionAtestacionAutorizacionV1Prueba(t),
	)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	if bytes.HasPrefix(mensaje, append([]byte("VEC-AUTORIZACION-ATESTACION"), 0)) {
		t.Fatal("el mensaje nuevo conserva el dominio binario antiguo")
	}
}

func TestDecisionAutorizacionRechazaVinculoValidoDeOtraPersonaYPerfil(t *testing.T) {
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	actorBase := contextoActorVinculoPrueba(t, decision.EmitidaEn)
	instantaneaAjena := actorBase.Instantanea
	instantaneaAjena.VinculoRef = "vca_otra234567890abcdefghijkl"
	instantaneaAjena.PersonaRef = "per_otra234567890abcdefghijkl"
	instantaneaAjena.PerfilActivoRef = "prf_otra234567890abcdefghijkl"
	cuenta := CuentaAutenticadaContextoActor{
		CuentaRef: actorBase.Instantanea.CuentaRef,
		Metodo:    actorBase.Principal.AuthMethod,
		Garantia:  actorBase.Principal.AuthAssurance,
	}
	actorAjeno, err := NuevoContextoActor(cuenta, instantaneaAjena, actorBase.ResueltoEn)
	if err != nil {
		t.Fatalf("crear actor ajeno valido: %v", err)
	}
	autenticacion := autenticacionRevalidadaVinculoPrueba(decision.EmitidaEn)
	vinculoAjeno, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion), actorAjeno, decision.EmitidaEn,
	)
	if err != nil {
		t.Fatalf("crear vinculo ajeno valido: %v", err)
	}
	if vinculoAjeno.Validar() != nil {
		t.Fatal("la precondicion exige un vinculo ajeno estructuralmente valido")
	}

	decision.VinculoAutenticacionActor = vinculoAjeno
	if err := decision.Validar(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
		t.Fatalf("Validar acepto vinculo A con decision B: %v", err)
	}
	if err := decision.ValidarEvidenciaInstantanea(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
		t.Fatalf("ValidarEvidenciaInstantanea acepto vinculo A con decision B: %v", err)
	}
	if _, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(), decision,
	); !errors.Is(err, ErrDecisionAutorizacionInvalida) ||
		!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("VEC-AD-1 serializo vinculo A con decision B: %v", err)
	}
}

func TestDecisionAutorizacionVigenteEnExigeEvidenciaInstantaneaValida(t *testing.T) {
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	if !decision.VigenteEn(decision.EmitidaEn) {
		t.Fatal("la precondicion exige una decision reforzada vigente")
	}
	decision.RevisionCatalogoPoliticas = 0
	if decision.VigenteEn(decision.EmitidaEn) {
		t.Fatal("VigenteEn acepto una decision con evidencia instantanea invalida")
	}
}

func TestCabeceraAtestacionAutorizacionV1RechazaConfiguracionAmbigua(t *testing.T) {
	base := cabeceraAtestacionAutorizacionV1Prueba()
	casos := []struct {
		nombre string
		mutar  func(*CabeceraAtestacionAutorizacionV1)
	}{
		{"version cero", func(c *CabeceraAtestacionAutorizacionV1) { c.FormatoVersion = 0 }},
		{"suite ausente", func(c *CabeceraAtestacionAutorizacionV1) { c.Suite = "" }},
		{"suite recortable", func(c *CabeceraAtestacionAutorizacionV1) { c.Suite = " VEC-AD-PRUEBA-1" }},
		{"suite comodin", func(c *CabeceraAtestacionAutorizacionV1) { c.Suite = "VEC-AD-*" }},
		{"clave ausente", func(c *CabeceraAtestacionAutorizacionV1) { c.ClaveID = "" }},
		{"clave unicode", func(c *CabeceraAtestacionAutorizacionV1) { c.ClaveID = "clave:ñ" }},
		{"audiencia con espacio", func(c *CabeceraAtestacionAutorizacionV1) { c.Audiencia = "vec diputacion" }},
		{"audiencia comodin", func(c *CabeceraAtestacionAutorizacionV1) { c.Audiencia = "vec/*" }},
		{"audiencia de control", func(c *CabeceraAtestacionAutorizacionV1) { c.Audiencia = "vec\nproduccion" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			cabecera := base
			caso.mutar(&cabecera)
			if err := cabecera.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) ||
				!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("configuracion aceptada o error no cerrado: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV1RechazaDecisionNoCanonica(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV1Prueba()
	base := decisionAtestacionAutorizacionV1Prueba(t)
	casos := []struct {
		nombre string
		mutar  func(*DecisionAutorizacion)
	}{
		{"denegacion", func(d *DecisionAutorizacion) { d.Concedida = false; d.Codigo = "denegada" }},
		{"comodin", func(d *DecisionAutorizacion) { d.Accion = "bolsa.*" }},
		{"politicas evaluadas desordenadas", func(d *DecisionAutorizacion) {
			d.PoliticasEvaluadasRefs[0], d.PoliticasEvaluadasRefs[1] = d.PoliticasEvaluadasRefs[1], d.PoliticasEvaluadasRefs[0]
		}},
		{"campos desordenados", func(d *DecisionAutorizacion) {
			d.CamposPermitidos[0], d.CamposPermitidos[1] = d.CamposPermitidos[1], d.CamposPermitidos[0]
		}},
		{"obligaciones desordenadas", func(d *DecisionAutorizacion) {
			d.Obligaciones[0], d.Obligaciones[1] = d.Obligaciones[1], d.Obligaciones[0]
		}},
		{"instante no UTC", func(d *DecisionAutorizacion) {
			d.EmitidaEn = d.EmitidaEn.In(time.FixedZone("CEST", 2*60*60))
		}},
		{"instante submicrosegundo", func(d *DecisionAutorizacion) { d.EmitidaEn = d.EmitidaEn.Add(time.Nanosecond) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			decision := clonarDecisionAtestacionAutorizacionV1Prueba(base)
			caso.mutar(&decision)
			if _, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, decision); !errors.Is(err, ErrDecisionAutorizacionInvalida) ||
				!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("decision no canonica aceptada: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV1RechazaExcesoDeTamano(t *testing.T) {
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	decision.CamposPermitidos = listaGrandeAtestacionAutorizacionV1Prueba("campo")
	decision.Obligaciones = listaGrandeAtestacionAutorizacionV1Prueba("obligacion")
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("la fixture debe ser valida antes del limite binario: %v", err)
	}
	if _, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(), decision,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("mensaje sobredimensionado aceptado: %v", err)
	}
}

func TestMensajeAtestacionAutorizacionV1AplicaLimiteExactoDuranteLaEscritura(t *testing.T) {
	for _, objetivo := range []int{
		TamanoMaximoMensajeAtestacionAutorizacionV1 - 1,
		TamanoMaximoMensajeAtestacionAutorizacionV1,
	} {
		t.Run(fmt.Sprintf("%d_bytes", objetivo), func(t *testing.T) {
			decision := decisionAtestacionAutorizacionV1ConTamanoObjetivo(t, objetivo)
			mensaje, err := SerializarMensajeAtestacionAutorizacionV1(
				cabeceraAtestacionAutorizacionV1Prueba(), decision,
			)
			if err != nil || len(mensaje) != objetivo {
				t.Fatalf("tamano limite %d: longitud=%d err=%v", objetivo, len(mensaje), err)
			}
		})
	}

	decision := decisionAtestacionAutorizacionV1ConTamanoObjetivo(
		t, TamanoMaximoMensajeAtestacionAutorizacionV1+1,
	)
	if _, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(), decision,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("limite+1 aceptado: %v", err)
	}
}

func TestMensajeAtestacionAutorizacionV1RechazaAnoFueraDelIntervaloInteroperable(t *testing.T) {
	for _, ano := range []int{0, 10_000} {
		t.Run(fmt.Sprintf("ano_%d", ano), func(t *testing.T) {
			decision := decisionAtestacionAutorizacionV1Prueba(t)
			decision.EmitidaEn = time.Date(ano, 1, 1, 0, 0, 0, 0, time.UTC)
			decision.ValidaHasta = decision.EmitidaEn.Add(time.Minute)
			if _, err := SerializarMensajeAtestacionAutorizacionV1(
				cabeceraAtestacionAutorizacionV1Prueba(), decision,
			); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
				t.Fatalf("ano no interoperable aceptado: %v", err)
			}
		})
	}
}

func cabeceraAtestacionAutorizacionV1Prueba() CabeceraAtestacionAutorizacionV1 {
	return CabeceraAtestacionAutorizacionV1{
		FormatoVersion: VersionFormatoAtestacionAutorizacionV1,
		Suite:          "VEC-AD-PRUEBA-1",
		ClaveID:        "clave:prueba:2026-01",
		Audiencia:      "vec-diputacion/pruebas/vec/autorizacion",
	}
}

func decisionAtestacionAutorizacionV1Prueba(t *testing.T) DecisionAutorizacion {
	t.Helper()
	evaluadas := []string{"politica:ambito:v1", "politica:horario:v2"}
	huellasEvaluadas := map[string]string{
		evaluadas[0]: huellaAtestacionPrueba('a'),
		evaluadas[1]: huellaAtestacionPrueba('b'),
	}
	huellaCatalogo, err := HuellaEvidenciasCatalogoPoliticasAutorizacion(evaluadas, huellasEvaluadas)
	if err != nil {
		t.Fatalf("crear catalogo de prueba: %v", err)
	}
	emitida := time.Date(2026, 7, 15, 10, 11, 12, 123_456_000, time.UTC)
	decision := DecisionAutorizacion{
		DecisionRef:                           "decision:atestacion:1",
		Concedida:                             true,
		Codigo:                                "concedida",
		PrincipalID:                           "per_0123456789abcdefghijkl",
		PerfilActivoRef:                       "prf_0123456789abcdefghijkl",
		Accion:                                "bolsa.merito.revisar",
		RecursoRef:                            "merito:456",
		ModuloID:                              "bolsa",
		TipoRecurso:                           "merito",
		ContextoRecursoHuellaSHA256:           huellaAtestacionPrueba('c'),
		Finalidad:                             "gestion_bolsa",
		CorrelacionRef:                        "correlacion:789",
		VinculoAutenticacionActor:             vinculoAutenticacionActorV1Prueba(t, emitida),
		AsignacionRef:                         "asignacion:rrhh:v3",
		AsignacionHuellaSHA256:                huellaAtestacionPrueba('d'),
		VersionRolRef:                         "rol:tecnico_rrhh:v4",
		VersionRolHuellaSHA256:                huellaAtestacionPrueba('e'),
		ControlVigenciaVersionRolRef:          "rol:tecnico_rrhh:v4",
		ControlVigenciaVersionRolRevision:     7,
		ControlVigenciaVersionRolHuellaSHA256: huellaAtestacionPrueba('f'),
		RevisionCatalogoPoliticas:             11,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogo,
		PoliticasEvaluadasRefs:                evaluadas,
		PoliticasEvaluadasHuellasSHA256:       huellasEvaluadas,
		PoliticasRefs:                         []string{evaluadas[1]},
		PoliticasHuellasSHA256: map[string]string{
			evaluadas[1]: huellasEvaluadas[evaluadas[1]],
		},
		GarantiaMinima:   AuthAssuranceHigh,
		CamposPermitidos: []string{"descripcion", "estado"},
		Obligaciones:     []string{"registrar_acceso", "trazar_revision"},
		EmitidaEn:        emitida,
		ValidaHasta:      emitida.Add(90 * time.Second),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de prueba invalida: %v", err)
	}
	return decision
}

func vinculoAtestacionAutorizacionAlternativoPrueba(
	t *testing.T,
	instante time.Time,
) VinculoAutenticacionActorV1 {
	t.Helper()
	actor := contextoActorVinculoPrueba(t, instante)
	autenticacion := autenticacionRevalidadaVinculoPrueba(instante)
	autenticacion.AutenticacionRef = "aut_otra234567890abcdefghijkl"
	autenticacion.AutenticacionHuellaSHA256 = huellaAtestacionPrueba('8')
	autenticacion.AsercionRef = "ase_otra234567890abcdefghijkl"
	autenticacion.SesionRef = "ses_otra234567890abcdefghijkl"
	autenticacion.ControlSesionRef = "cse_otra234567890abcdefghijkl"
	autenticacion.ControlSesionRevision++
	autenticacion.ControlSesionHuellaSHA256 = huellaAtestacionPrueba('7')
	vinculo, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion), actor, instante,
	)
	if err != nil {
		t.Fatalf("crear vinculo alternativo: %v", err)
	}
	return vinculo
}

func clonarDecisionAtestacionAutorizacionV1Prueba(decision DecisionAutorizacion) DecisionAutorizacion {
	copia := decision
	copia.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
	copia.PoliticasRefs = append([]string(nil), decision.PoliticasRefs...)
	copia.CamposPermitidos = append([]string(nil), decision.CamposPermitidos...)
	copia.Obligaciones = append([]string(nil), decision.Obligaciones...)
	copia.PoliticasEvaluadasHuellasSHA256 = make(map[string]string, len(decision.PoliticasEvaluadasHuellasSHA256))
	for clave, valor := range decision.PoliticasEvaluadasHuellasSHA256 {
		copia.PoliticasEvaluadasHuellasSHA256[clave] = valor
	}
	copia.PoliticasHuellasSHA256 = make(map[string]string, len(decision.PoliticasHuellasSHA256))
	for clave, valor := range decision.PoliticasHuellasSHA256 {
		copia.PoliticasHuellasSHA256[clave] = valor
	}
	return copia
}

func huellaAtestacionPrueba(caracter byte) string {
	return strings.Repeat(string(caracter), 64)
}

func listaGrandeAtestacionAutorizacionV1Prueba(prefijo string) []string {
	valores := make([]string, 512)
	for indice := range valores {
		inicio := fmt.Sprintf("%s_%03d_", prefijo, indice)
		valores[indice] = inicio + strings.Repeat("x", 512-len(inicio))
	}
	return valores
}

func decisionAtestacionAutorizacionV1ConTamanoObjetivo(
	t *testing.T,
	objetivo int,
) DecisionAutorizacion {
	t.Helper()
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	decision.CamposPermitidos = listaAjustableAtestacionAutorizacionV1Prueba("c")
	decision.Obligaciones = listaAjustableAtestacionAutorizacionV1Prueba("o")
	base, err := SerializarMensajeAtestacionAutorizacionV1(
		cabeceraAtestacionAutorizacionV1Prueba(), decision,
	)
	if err != nil || len(base) > objetivo {
		t.Fatalf("fixture base de %d para objetivo %d: %v", len(base), objetivo, err)
	}
	restante := objetivo - len(base)
	for _, lista := range []*[]string{&decision.CamposPermitidos, &decision.Obligaciones} {
		for indice := range *lista {
			capacidad := 512 - len((*lista)[indice])
			aumento := min(restante, capacidad)
			(*lista)[indice] += strings.Repeat("x", aumento)
			restante -= aumento
			if restante == 0 {
				return decision
			}
		}
	}
	t.Fatalf("fixture sin capacidad para objetivo %d: faltan %d bytes", objetivo, restante)
	return DecisionAutorizacion{}
}

func listaAjustableAtestacionAutorizacionV1Prueba(prefijo string) []string {
	valores := make([]string, 512)
	for indice := range valores {
		valores[indice] = fmt.Sprintf("%s%04d_", prefijo, indice)
	}
	return valores
}

// lectorAtestacionAutorizacionV1Prueba es deliberadamente independiente del
// escritor de produccion: fija tipos, orden y fronteras del vector binario.
// Si el escritor omite, repite o desplaza un campo, la lectura deja de cuadrar.
type lectorAtestacionAutorizacionV1Prueba struct {
	t         *testing.T
	contenido []byte
	posicion  int
}

func nuevoLectorAtestacionAutorizacionV1Prueba(
	t *testing.T,
	contenido []byte,
) *lectorAtestacionAutorizacionV1Prueba {
	t.Helper()
	return &lectorAtestacionAutorizacionV1Prueba{t: t, contenido: append([]byte(nil), contenido...)}
}

func (l *lectorAtestacionAutorizacionV1Prueba) tomar(cantidad int) []byte {
	l.t.Helper()
	if cantidad < 0 || l.posicion > len(l.contenido)-cantidad {
		l.t.Fatalf("mensaje truncado en byte %d al pedir %d", l.posicion, cantidad)
	}
	inicio := l.posicion
	l.posicion += cantidad
	return l.contenido[inicio:l.posicion]
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirBytes(esperado []byte) {
	l.t.Helper()
	if recibido := l.tomar(len(esperado)); !bytes.Equal(recibido, esperado) {
		l.t.Fatalf("bytes en posicion %d = %x; esperados %x", l.posicion-len(esperado), recibido, esperado)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) leerUint16() uint16 {
	l.t.Helper()
	return binary.BigEndian.Uint16(l.tomar(2))
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirUint16(esperado uint16) {
	l.t.Helper()
	if recibido := l.leerUint16(); recibido != esperado {
		l.t.Fatalf("uint16 = %d; esperado %d", recibido, esperado)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) leerUint64() uint64 {
	l.t.Helper()
	return binary.BigEndian.Uint64(l.tomar(8))
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirUint64(esperado uint64) {
	l.t.Helper()
	if recibido := l.leerUint64(); recibido != esperado {
		l.t.Fatalf("uint64 = %d; esperado %d", recibido, esperado)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) leerTexto() string {
	l.t.Helper()
	longitud := uint64(binary.BigEndian.Uint32(l.tomar(4)))
	if longitud > uint64(len(l.contenido)-l.posicion) {
		l.t.Fatalf("texto declara %d bytes y solo quedan %d", longitud, len(l.contenido)-l.posicion)
	}
	return string(l.tomar(int(longitud)))
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirTexto(esperado string) {
	l.t.Helper()
	if recibido := l.leerTexto(); recibido != esperado {
		l.t.Fatalf("texto = %q; esperado %q", recibido, esperado)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirBooleano(esperado bool) {
	l.t.Helper()
	recibido := l.tomar(1)[0]
	if recibido > 1 || (recibido == 1) != esperado {
		l.t.Fatalf("booleano = 0x%02x; esperado %t", recibido, esperado)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirLista(esperada []string) {
	l.t.Helper()
	cantidad := uint64(binary.BigEndian.Uint32(l.tomar(4)))
	if cantidad != uint64(len(esperada)) {
		l.t.Fatalf("lista declara %d elementos; esperados %d", cantidad, len(esperada))
	}
	recibida := make([]string, 0, int(cantidad))
	for indice := uint64(0); indice < cantidad; indice++ {
		recibida = append(recibida, l.leerTexto())
	}
	if !reflect.DeepEqual(recibida, esperada) {
		l.t.Fatalf("lista = %#v; esperada %#v", recibida, esperada)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirMapa(esperado map[string]string) {
	l.t.Helper()
	cantidad := uint64(binary.BigEndian.Uint32(l.tomar(4)))
	if cantidad != uint64(len(esperado)) {
		l.t.Fatalf("mapa declara %d pares; esperados %d", cantidad, len(esperado))
	}
	recibido := make(map[string]string, int(cantidad))
	var anterior string
	for indice := uint64(0); indice < cantidad; indice++ {
		clave := l.leerTexto()
		if indice > 0 && bytes.Compare([]byte(anterior), []byte(clave)) >= 0 {
			l.t.Fatalf("claves de mapa no estan en orden UTF-8 estricto: %q, %q", anterior, clave)
		}
		if _, repetida := recibido[clave]; repetida {
			l.t.Fatalf("clave de mapa repetida: %q", clave)
		}
		recibido[clave] = l.leerTexto()
		anterior = clave
	}
	if !reflect.DeepEqual(recibido, esperado) {
		l.t.Fatalf("mapa = %#v; esperado %#v", recibido, esperado)
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirInstante(esperado time.Time) {
	l.t.Helper()
	recibido := time.UnixMicro(int64(l.leerUint64())).UTC()
	if !recibido.Equal(esperado) || recibido.Nanosecond() != esperado.Nanosecond() {
		l.t.Fatalf("instante = %s; esperado %s", recibido.Format(time.RFC3339Nano), esperado.Format(time.RFC3339Nano))
	}
}

func (l *lectorAtestacionAutorizacionV1Prueba) exigirLongitudFinal() {
	l.t.Helper()
	if recibido := l.leerUint64(); recibido != uint64(len(l.contenido)) {
		l.t.Fatalf("longitud final = %d; esperada %d", recibido, len(l.contenido))
	}
	if l.posicion != len(l.contenido) {
		l.t.Fatalf("sobran %d bytes tras el esquema cerrado", len(l.contenido)-l.posicion)
	}
}
