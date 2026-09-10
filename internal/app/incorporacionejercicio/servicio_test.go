package incorporacionejercicio

// Fixtures nominales públicas copiadas; autoridades y TX son DOBLES, no PG.
import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	gocose "github.com/veraison/go-cose"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	puente "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/personalincorporacion"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	domain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registroV2_registroConcesionV3Doble struct{ registradaEn time.Time }

func (r registroV2_registroConcesionV3Doble) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.registradaEn, nil
}

type registroV2_revalidadorVinculoPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d registroV2_revalidadorVinculoPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type registroV2_resolutorResultadoVinculoPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d registroV2_resolutorResultadoVinculoPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type registroV2_relojVinculoPrueba struct{ instante time.Time }

func (d registroV2_relojVinculoPrueba) Ahora() time.Time { return d.instante }

func registroV2_contextoAutorizacionAltaV3Prueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoAutorizacionAltaV3 {
	return registroV2_contextoAutorizacionAltaV3PruebaConMarcas(t, ahora, "a", "a")
}

func registroV2_contextoAutorizacionAltaV3PruebaConMarcas(
	t *testing.T,
	ahora time.Time,
	marcaActor string,
	marcaPerfil string,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl" + marcaActor + marcaPerfil,
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl" + marcaActor,
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl" + marcaPerfil,
		PerfilVersion:   5,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema:           dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef,
			Version:   instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef,
			Version:    instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef,
			Version:   instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef,
			Version:    instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn" +
			marcaActor + marcaPerfil + strconv.FormatInt(ahora.UnixMicro(), 10),
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  "ase_0123456789abcdefghijkl",
		SesionRef:                    "ses_0123456789abcdefghijkl",
		ControlSesionRef:             "cse_0123456789abcdefghijkl",
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		registroV2_revalidadorVinculoPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		registroV2_resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		registroV2_relojVinculoPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

func registroV2_concesionAutorizacionV3Prueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	ahora time.Time,
	referenciaDecision string,
	conceder bool,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datos.Recurso.Ambitos))
	for clave, valor := range datos.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{Clave: clave, Valores: []string{valor}})
	}
	version := dominiovec.VersionRol{
		RolID:   "tecnico_rrhh",
		Version: 1,
		Nombre:  "Técnico de RRHH",
		Estado:  dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID,
			TipoRecurso:    datos.Recurso.Tipo,
			Finalidades:    []string{datos.Finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
	if !conceder {
		version.Concesiones[0].Accion =
			"contratacion_temporal.solicitud.denegada"
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID:    "asig-contratacion-temporal",
			Version:         1,
			PerfilActivoRef: vinculo.PerfilActivoRef,
			PrincipalID:     vinculo.PrincipalID,
			VersionRolRef:   version.Referencia(),
			Estado:          dominiovec.EstadoAsignacionPerfilActiva,
			Ambitos:         ambitos,
			VigenteDesde:    ahora.Add(-time.Hour),
			VigenteHasta:    ahora.Add(time.Hour),
			EmitidaPor:      "administrador-identidades",
			EmitidaEn:       ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef:  version.Referencia(),
			Revision:       1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		referenciaDecision,
		ahora,
		ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	if !conceder {
		return decision,
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			nil
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud,
		decision,
		motivo,
		resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := puertosvec.
		RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(),
			registroV2_registroConcesionV3Doble{registradaEn: ahora},
			orden,
		)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion, nil
}

func registroV2Exigir(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func registroV2Datos(t *testing.T, ahora time.Time) ct.DatosMaterialConfirmacionIncorporacionV2 {
	t.Helper()
	s := ct.SolicitudAltaPersonalRPT{Esquema: ct.EsquemaAltaPersonalRPT, ContratoVersion: 1,
		SolicitudRef: "solicitud:ejercicio:0001", ExpedienteRef: registroV2Ref("expediente:ejercicio:ct:0001"), VersionExpediente: 7,
		CapacidadRef: "capacidad:ejercicio:0001", CorrelacionRef: "correlacion:ejercicio:0001", IdempotenciaRef: "idempotencia:ejercicio:0001",
		FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{Referencia: "rpt:ejercicio:20260908", Version: 3, HuellaSHA256: "a81229be023f01b99c16b76c9d4f6283dc6efaf5177340b063d36ab30dc654ac"}, PuestoRef: "puesto:ejercicio:0001", PlazaRef: "plaza:ejercicio:0001"}
	contenido, err := os.ReadFile("../../modules/personal/adapters/fuenteejercicio/testdata/fuente-ejercicio.json")
	registroV2Exigir(t, err)
	// Copia local sintética: no toca la fuente congelada ni enlaza otra DB.
	contenido = []byte(strings.ReplaceAll(string(contenido), "expediente:ejercicio:ct:0001", s.ExpedienteRef))
	terna := fuenteejercicio.TernaEsperada{Referencia: "fuente:personal:ejercicio:20260908", Version: 1, HuellaSHA256: registroV2Hash(contenido)}
	fuente, err := fuenteejercicio.NuevaFuenteEjercicio(contenido, terna)
	registroV2Exigir(t, err)
	v, err := fuente.Resolver(context.Background(), s)
	registroV2Exigir(t, err)
	autoridad := registroV2_contextoAutorizacionAltaV3Prueba(t, ahora)
	actor, err := autoridad.Vinculo.Datos()
	registroV2Exigir(t, err)
	materialPersonal := personal.MaterialAlta{Preparacion: personal.PreparacionAlta{Solicitud: s, Fuente: terna, Vinculo: v},
		OrganizacionRef: registroV2Ref("organizacion:ejercicio:personal"), ActorRef: actor.PrincipalID, PerfilRef: actor.PerfilActivoRef}
	ph, err := materialPersonal.HuellaSHA256()
	registroV2Exigir(t, err)
	canon, err := json.Marshal(struct {
		Esquema  string
		Material personal.MaterialAlta
	}{personal.EsquemaMaterialAlta, materialPersonal})
	registroV2Exigir(t, err)
	hs, err := s.HuellaSHA256()
	registroV2Exigir(t, err)
	r := ct.ResultadoAltaPersonalRPT{Esquema: ct.EsquemaAltaPersonalRPT, ContratoVersion: 1, ResultadoRef: "resultado:ejercicio:0001", ReciboRef: "recibo:ejercicio:0001",
		SolicitudRef: s.SolicitudRef, CorrelacionRef: s.CorrelacionRef, IdempotenciaRef: s.IdempotenciaRef, HuellaSolicitudSHA256: hs, Estado: ct.AltaPersonalRPTConfirmada, RelacionRef: registroV2Ref("relacion:ejercicio:0001"), OcupacionRef: "ocupacion:ejercicio:0001"}
	cor, err := core.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), &registroV2GeneradorCorrelacion{referencia: "correlacion_11111111111111111111111111111111"})
	registroV2Exigir(t, err)
	return ct.DatosMaterialConfirmacionIncorporacionV2{
		Confirmacion: ct.DatosConfirmacionIncorporacion{SolicitudPersonal: s, ResultadoPersonal: r, VersionSeguimientoEsperada: 0,
			PeriodoIncorporacion: ctdomain.IntervaloSeguimiento{Desde: ahora.Add(time.Hour), Hasta: ahora.Add(25 * time.Hour)}, MotivoClave: "ejercicio_incorporacion",
			Documentos: []ctdomain.DocumentoSeguimiento{{TipoClave: "resolucion_ejercicio", Referencia: registroV2Ref("documento:ejercicio:resolucion")}, {TipoClave: "anexo_ejercicio", Referencia: registroV2Ref("documento:ejercicio:antecedente")}}},
		VersionActualExpediente: 7,
		Preparacion:             ct.PreparacionSeguimientoConfirmacionIncorporacion{OrganizacionRef: materialPersonal.OrganizacionRef, UnidadRef: registroV2Ref("unidad:ejercicio:rrhh"), ActorRef: registroV2Ref("actor:ejercicio:rrhh"), CorrelacionRef: registroV2Ref("correlacion:ejercicio:ct")},
		SolicitudContexto:       ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: actor.AutenticacionRef, SesionRef: actor.SesionRef, PerfilRef: actor.PerfilActivoRef}, Contexto: autoridad,
		MotivoV3: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_11111111111111111111111111111111"}, CorrelacionV3: cor,
		Personal: ct.RegistroPersonalEjercicio{Solicitud: s, Resultado: r, MaterialCanonico: canon, MaterialSHA256: ph, RegistradoEn: ahora.Add(-24 * time.Hour),
			DecisionOriginalRef: "decision:ejercicio:personal:original", AuditoriaRef: "auditoria:ejercicio:personal", OutboxRef: "outbox:ejercicio:personal", EjercicioSintetico: true}, EjercicioSintetico: true,
	}
}

// Comparte las fixtures de contexto/concesión existentes. COSE, verificación,
// capacidad HMAC y exportación se construyen por APIs reales, claves efímeras.
// El registro de concesión sigue siendo doble: no acredita gobierno PostgreSQL.
func registroV2Autoridad(t *testing.T, m ct.MaterialConfirmacionIncorporacionV2, ahora time.Time, mutar func(*core.DatosSolicitudAutorizacionLigadaV3), audiencia string) ct.AutorizacionConfirmacionIncorporacionV2 {
	t.Helper()
	d, err := m.Datos()
	registroV2Exigir(t, err)
	r, err := ct.RecursoConfirmacionIncorporacionV2(m)
	registroV2Exigir(t, err)
	ds := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: d.Contexto.Vinculo, Accion: ct.AccionConfirmarIncorporacion,
		Finalidad: ct.FinalidadConfirmarIncorporacion, Recurso: r, ReferenciaMotivo: d.MotivoV3, Correlacion: d.CorrelacionV3}
	if mutar != nil {
		mutar(&ds)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(ds)
	registroV2Exigir(t, err)
	decision, registro, err := registroV2_concesionAutorizacionV3Prueba(t, s, d.Contexto.Resultado, ds.ReferenciaMotivo, ahora.Add(-time.Millisecond), fmt.Sprintf("dec_%032x", ahora.UnixMicro()), true)
	registroV2Exigir(t, err)
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	registroV2Exigir(t, err)
	defer clear(privada)
	cabecera := core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:ct", Audiencia: "vec/prueba/ct"}
	sf, err := vecports.NuevaSolicitudFirmaAtestacionAutorizacionV3(cabecera, decision, ds.ReferenciaMotivo, d.Contexto.Resultado)
	registroV2Exigir(t, err)
	mensaje, err := sf.Mensaje()
	registroV2Exigir(t, err)
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	registroV2Exigir(t, err)
	cose := gocose.NewSign1Message()
	cose.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	cose.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(cabecera.ClaveID)
	cose.Payload = mensaje
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, privada)
	registroV2Exigir(t, err)
	registroV2Exigir(t, cose.Sign(rand.Reader, aad, firmante))
	cose.Payload = nil
	cose.Headers.RawProtected = nil
	cose.Headers.RawUnprotected = nil
	sobre, err := cose.MarshalCBOR()
	registroV2Exigir(t, err)
	firma, err := vecports.NuevoResultadoFirmaAtestacionAutorizacionV3(sf, sobre, "evidencia:prueba:ct", ahora)
	registroV2Exigir(t, err)
	atestacion, err := vecports.NuevaAtestacionAutorizacionV3(sf, firma)
	registroV2Exigir(t, err)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(cabecera.ClaveID, 1, publica, cabecera.Audiencia, confianza.EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{})
	registroV2Exigir(t, err)
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:ct", 1, ahora.Add(-time.Minute), ahora.Add(time.Hour), raiz)
	registroV2Exigir(t, err)
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(config, registroV2_relojVinculoPrueba{instante: ahora})
	registroV2Exigir(t, err)
	prueba, err := verificador.Verificar(context.Background(), s, decision, ds.ReferenciaMotivo, d.Contexto.Resultado, atestacion)
	registroV2Exigir(t, err)
	secreto := make([]byte, 32)
	_, err = rand.Read(secreto)
	registroV2Exigir(t, err)
	defer clear(secreto)
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:prueba", 1, secreto, "emisor:prueba:ct", audiencia,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	registroV2Exigir(t, err)
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, registroV2_relojVinculoPrueba{instante: ahora})
	registroV2Exigir(t, err)
	capacidad, err := emisor.Emitir(context.Background(), s, decision, ds.ReferenciaMotivo, d.Contexto.Resultado, atestacion, prueba)
	registroV2Exigir(t, err)
	material, err := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(s, decision, ds.ReferenciaMotivo, d.Contexto.Resultado, atestacion, prueba, capacidad, raiz)
	registroV2Exigir(t, err)
	exportacion, err := material.ExportarMaterialParaConsumidor()
	registroV2Exigir(t, err)
	return ct.AutorizacionConfirmacionIncorporacionV2{Solicitud: s, Decision: decision, Confirmacion: registro, Exportacion: exportacion}
}

func registroV2Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type registroV2GeneradorCorrelacion struct{ referencia string }

func (g *registroV2GeneradorCorrelacion) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return g.referencia, nil
}

func registroV2Ref(s string) string { return "ref:" + registroV2Hash([]byte(s)) }

// Dobles explícitos: contexto/concesión/acreditador/transacción NO prueban commit,
// autorización SQL, ni identidad corporativa. COSE/Ed25519/HMAC/exportación usan
// las APIs Go reales con claves efímeras en memoria (fixture pública copiada).
type registroV2Reloj struct {
	ahora    time.Time
	lecturas int
	despues  func(int)
}

func (r *registroV2Reloj) Ahora() time.Time {
	r.lecturas++
	if r.despues != nil {
		r.despues(r.lecturas)
	}
	return r.ahora
}

type registroV2Proveedor struct {
	t        *testing.T
	reloj    *registroV2Reloj
	llamadas int
	despues  func(*ct.AutorizacionConfirmacionIncorporacionV2)
	err      error
}

func (p *registroV2Proveedor) AutorizarConfirmacionIncorporacion(_ context.Context, m ct.MaterialConfirmacionIncorporacionV2) (ct.AutorizacionConfirmacionIncorporacionV2, error) {
	p.llamadas++
	a := registroV2Autoridad(p.t, m, p.reloj.ahora, nil, ct.AudienciaConfirmacionIncorporacionV2)
	if p.despues != nil {
		p.despues(&a)
	}
	return a, p.err
}

type registroV2Acreditador struct {
	llamadas int
	despues  func(*ct.RegistroPersonalEjercicio)
	err      error
}

func (a *registroV2Acreditador) AcreditarRegistroPersonal(_ context.Context, o ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
	a.llamadas++
	d, _ := o.Material().Datos()
	r := d.Personal
	if a.despues != nil {
		a.despues(&r)
	}
	return r, a.err
}

type registroV2TX struct {
	t                 *testing.T
	reloj             *registroV2Reloj
	llamadas, efectos int
	original          ct.OrdenConfirmacionIncorporacionV2
	persistido        ct.ResultadoRegistroIncorporacionV2
	ultimo            ct.ResultadoRegistroIncorporacionV2
	despues           func(*ct.ResultadoRegistroIncorporacionV2)
	err               error
}

func (x *registroV2TX) RegistrarORecuperarIncorporacion(_ context.Context, o ct.OrdenConfirmacionIncorporacionV2, pre ct.AcreditacionPersonalIncorporacion) (ct.ResultadoRegistroIncorporacionV2, error) {
	x.llamadas++
	_, e := pre.RegistroPara(o, x.reloj.ahora)
	registroV2Exigir(x.t, e)
	var r ct.ResultadoRegistroIncorporacionV2
	if x.efectos == 0 {
		x.original = o
		r = registroV2ResultadoOriginal(x.t, o, x.reloj.ahora)
		x.persistido = r.Copia()
		x.efectos++
	} else {
		r = x.persistido.Copia()
		r.Recuperado = true
		r.ConsumosActuales = registroV2Consumos(x.t, o, x.reloj.ahora)
	}
	x.ultimo = r.Copia()
	if x.despues != nil {
		x.despues(&r)
	}
	return r, x.err
}

func registroV2Consumos(t *testing.T, o ct.OrdenConfirmacionIncorporacionV2, ahora time.Time) ct.ConsumosRegistroIncorporacionV2 {
	h, e := o.Exportacion().HuellaConjuntoSHA256()
	registroV2Exigir(t, e)
	dec := o.Exportacion().ResumenCapacidad().DecisionRef()
	return ct.ConsumosRegistroIncorporacionV2{
		DecisionCTRef: dec, HuellaExportacionCT: h, ConsumoCTSHA256: registroV2Hash([]byte("ct" + dec)), AuditoriaAD3CTRef: "aud:ad3:ct:" + dec, ConsumidaCTEn: ahora,
		DecisionLecturaRef: "decision:lectura:" + dec, ConsumoLecturaSHA256: registroV2Hash([]byte("lectura" + dec)), AuditoriaAD3LecturaRef: "aud:ad3:lectura:" + dec,
		AuditoriaPersonalLecturaRef: "aud:personal:lectura:" + dec, LeidaPersonalEn: ahora,
	}
}

func registroV2ResultadoOriginal(t *testing.T, o ct.OrdenConfirmacionIncorporacionV2, ahora time.Time) ct.ResultadoRegistroIncorporacionV2 {
	t.Helper()
	d, e := o.Material().Datos()
	registroV2Exigir(t, e)
	b, e := os.ReadFile("../../modules/contrataciontemporal/adapters/seguimientoejercicio/testdata/definicion-ejercicio.json")
	registroV2Exigir(t, e)
	var fuente struct {
		Publicacion domain.PublicacionDefinicionSeguimiento `json:"publicacion"`
	}
	registroV2Exigir(t, json.Unmarshal(b, &fuente))
	def, e := domain.RestaurarDefinicionSeguimiento(fuente.Publicacion)
	registroV2Exigir(t, e)
	// Raíz de ejercicio suministrada por el doble de almacén, NO por el servicio.
	antes, e := domain.NuevoSeguimiento(def, domain.AltaSeguimiento{Referencia: registroV2Ref("seguimiento:ejercicio:001"), OrganizacionRef: d.Preparacion.OrganizacionRef,
		ExpedienteRef: d.Confirmacion.SolicitudPersonal.ExpedienteRef, RelacionRef: d.Confirmacion.ResultadoPersonal.RelacionRef,
		PeriodoPrevisto: d.Confirmacion.PeriodoIncorporacion, CreadoEn: ahora.Add(-time.Minute)})
	registroV2Exigir(t, e)
	periodo := d.Confirmacion.PeriodoIncorporacion
	transicion := domain.DatosTransicionSeguimiento{ActuacionRef: registroV2Ref("actuacion:ejercicio:ct:001"), TransicionClave: ct.TransicionConfirmarIncorporacion,
		MotivoClave: d.Confirmacion.MotivoClave, ActorRef: d.Preparacion.ActorRef, UnidadRef: d.Preparacion.UnidadRef, EfectivoEn: periodo.Desde, RegistradaEn: ahora,
		Documentos: d.Confirmacion.Documentos, Periodo: &periodo, ReciboRef: registroV2Ref("recibo:ejercicio:ct:001"), CorrelacionRef: d.Preparacion.CorrelacionRef}
	despues, e := antes.Aplicar(def, d.Confirmacion.VersionSeguimientoEsperada, transicion)
	registroV2Exigir(t, e)
	ca, e := domain.SerializarEstadoSeguimientoCanonico(def, antes.Estado())
	registroV2Exigir(t, e)
	cp, e := domain.SerializarEstadoSeguimientoCanonico(def, despues.Estado())
	registroV2Exigir(t, e)
	canon, e := o.Material().MaterialCanonico()
	registroV2Exigir(t, e)
	intencion, e := ct.IntencionRegistroIncorporacionV2(o.Material())
	registroV2Exigir(t, e)
	r := ct.ReciboRegistroIncorporacionV2{Esquema: ct.EsquemaReciboRegistroIncorporacionV2, MaterialOriginalCanonico: canon, MaterialOriginalSHA256: registroV2Hash(canon),
		IntencionSHA256: registroV2Hash(intencion), ContextoOriginalRef: d.Contexto.Resultado.RegistroContextoRef,
		VersionSolicitudPersonal: d.Confirmacion.SolicitudPersonal.VersionExpediente, VersionActualExpediente: d.VersionActualExpediente,
		SeguimientoRef: antes.Estado().Referencia, DefinicionSeguimiento: def.Referencia(), HuellaRaizSeguimiento: antes.Estado().HuellaRaizSHA256,
		HuellaEstadoAnterior: registroV2Hash(ca), HuellaEstadoResultante: registroV2Hash(cp), VersionSeguimientoAnterior: antes.Version(), VersionSeguimientoResultante: despues.Version(),
		Transicion: transicion, AuditoriaCTRef: "auditoria:ejercicio:ct:001", OutboxCTRef: "outbox:ejercicio:ct:001", ConsumosOriginales: registroV2Consumos(t, o, ahora), EjercicioSintetico: true}
	registroV2Exigir(t, r.ValidarPara(o))
	h, e := ct.NuevaHistoriaRegistroIncorporacionV2(o, r, def.Publicacion(), antes.Estado(), despues.Estado())
	registroV2Exigir(t, e)
	return ct.ResultadoRegistroIncorporacionV2{Recibo: r, Historia: h, ConsumosActuales: r.ConsumosOriginales}
}

type preparadorAplicacion struct {
	p                   ct.PreparacionIncorporacionAplicacionV2
	vista               ct.ProyeccionIncorporacionAplicacionV2
	llamadas, consultas int
	hook                func()
	err                 error
}

func (p *preparadorAplicacion) Preparar(ctx context.Context, i ct.IntencionIncorporacionAplicacionV2) (ct.PreparacionIncorporacionAplicacionV2, error) {
	p.llamadas++
	if p.hook != nil {
		p.hook()
	}
	return p.p, p.err
}
func (p *preparadorAplicacion) Consultar(ctx context.Context, s string) (ct.ProyeccionIncorporacionAplicacionV2, error) {
	p.consultas++
	if p.hook != nil {
		p.hook()
	}
	return p.vista, p.err
}

type altaProveedorApp struct {
	t        *testing.T
	reloj    *registroV2Reloj
	contexto ct.ContextoAutorizacionAltaV3
	org      string
	n        int
	denegar  bool
}

func (p *altaProveedorApp) ResolverAutoridad(context.Context, personal.PreparacionAlta) (personal.AutoridadAlta, error) {
	return personal.AutoridadAlta{OrganizacionRef: p.org, Contexto: p.contexto}, nil
}
func (p *altaProveedorApp) AutorizarAlta(_ context.Context, m personal.MaterialAlta) (personal.AutorizacionAlta, error) {
	p.n++
	a := autorizacionPrueba(p.t, personal.AutoridadAlta{OrganizacionRef: p.org, Contexto: p.contexto}, m, p.reloj.ahora, p.n, !p.denegar, nil)
	p.reloj.ahora = p.reloj.ahora.Add(time.Microsecond)
	return a, nil
}

type altaTXApp struct {
	reloj    *registroV2Reloj
	n        int
	original personal.ReciboAlta
	orden    personal.OrdenAlta
	mutar    func(*personal.ResultadoTransaccionAlta)
	hook     func()
	err      error
}

func (x *altaTXApp) RegistrarORecuperarAlta(ctx context.Context, o personal.OrdenAlta) (personal.ResultadoTransaccionAlta, error) {
	x.n++
	x.orden = o
	m := o.Material()
	dec := o.Exportacion().ResumenCapacidad().DecisionRef()
	replay := x.original.Resultado.ResultadoRef != ""
	if !replay {
		s := m.Preparacion.Solicitud
		h, _ := s.HuellaSHA256()
		x.original = personal.ReciboAlta{Material: m, Resultado: ct.ResultadoAltaPersonalRPT{Esquema: ct.EsquemaAltaPersonalRPT, ContratoVersion: 1,
			ResultadoRef: "resultado:app:1", ReciboRef: "recibo:app:1", SolicitudRef: s.SolicitudRef, CorrelacionRef: s.CorrelacionRef, IdempotenciaRef: s.IdempotenciaRef,
			HuellaSolicitudSHA256: h, Estado: ct.AltaPersonalRPTConfirmada, RelacionRef: registroV2Ref("relacion:app"), OcupacionRef: "ocupacion:app:1"},
			RegistradoEn: x.reloj.ahora, DecisionOriginalRef: dec, AuditoriaRef: "auditoria:app:1", OutboxRef: "outbox:app:1", EjercicioSintetico: true}
	}
	r := personal.ResultadoTransaccionAlta{Recibo: x.original, Replay: replay, DecisionConsumidaRef: dec}
	if x.mutar != nil {
		x.mutar(&r)
	}
	if x.hook != nil {
		x.hook()
	}
	return r, x.err
}
func (x *altaTXApp) registro(t *testing.T) ct.RegistroPersonalEjercicio {
	a := x.original
	b, e := json.Marshal(struct {
		Esquema  string
		Material personal.MaterialAlta
	}{personal.EsquemaMaterialAlta, a.Material})
	registroV2Exigir(t, e)
	h, e := a.Material.HuellaSHA256()
	registroV2Exigir(t, e)
	return ct.RegistroPersonalEjercicio{Solicitud: a.Material.Preparacion.Solicitud, Resultado: a.Resultado, MaterialCanonico: b, MaterialSHA256: h,
		RegistradoEn: a.RegistradoEn, DecisionOriginalRef: a.DecisionOriginalRef, AuditoriaRef: a.AuditoriaRef, OutboxRef: a.OutboxRef, EjercicioSintetico: true}
}

type permisoLecturaApp func(context.Context, lector.MaterialV2) (lector.AutorizacionV2, error)

func (f permisoLecturaApp) AutorizarLecturaIncorporacionV2(c context.Context, m lector.MaterialV2) (lector.AutorizacionV2, error) {
	return f(c, m)
}

type txLecturaApp func(context.Context, lector.Selector, lector.OrdenV2) (lector.Resultado, error)

func (f txLecturaApp) LeerRegistroPersonalV2(c context.Context, s lector.Selector, o lector.OrdenV2) (lector.Resultado, error) {
	return f(c, s, o)
}

type casoApp struct {
	s                  *Servicio
	p                  *preparadorAplicacion
	i                  ct.IntencionIncorporacionAplicacionV2
	reloj              *registroV2Reloj
	alta               *altaTXApp
	pa                 *altaProveedorApp
	ctTX               *registroV2TX
	lecturas, permisos int
	denegarLectura     bool
	cruzar             bool
	hookLectura        func()
}

func nuevoCasoApp(t *testing.T) *casoApp {
	t.Helper()
	ahora := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	d := registroV2Datos(t, ahora)
	d.VersionActualExpediente = 8
	c := &casoApp{reloj: &registroV2Reloj{ahora: ahora}}
	c.p = &preparadorAplicacion{p: ct.PreparacionIncorporacionAplicacionV2{SolicitudPersonal: d.Confirmacion.SolicitudPersonal, VersionActualExpediente: 8,
		VersionSeguimientoEsperada: 0, Periodo: d.Confirmacion.PeriodoIncorporacion, MotivoClave: d.Confirmacion.MotivoClave, Documentos: d.Confirmacion.Documentos,
		Preparacion: d.Preparacion, SolicitudContexto: d.SolicitudContexto, Contexto: d.Contexto, MotivoV3: d.MotivoV3, CorrelacionV3: d.CorrelacionV3}}
	c.i = ct.IntencionIncorporacionAplicacionV2{ExpedienteRef: d.Confirmacion.SolicitudPersonal.ExpedienteRef, SolicitudPersonalRef: d.Confirmacion.SolicitudPersonal.SolicitudRef,
		VersionActualExpedienteObservada: 8, MotivoClave: d.Confirmacion.MotivoClave, ConfirmaRevisionPersonal: true, ConfirmaEjercicioSintetico: true}
	for _, doc := range d.Confirmacion.Documentos {
		c.i.DocumentosRefs = append(c.i.DocumentosRefs, doc.Referencia)
	}
	c.p.vista = ct.ProyeccionIncorporacionAplicacionV2{Esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", ExpedienteRef: c.i.ExpedienteRef,
		VersionActualExpediente: 8, Preparacion: &ct.PreparacionVisibleIncorporacionV2{SolicitudPersonalRef: c.i.SolicitudPersonalRef, VersionSolicitudPersonal: 7, Periodo: c.p.p.Periodo,
			Motivos: []domain.ClaveCatalogo{c.i.MotivoClave}, DocumentosRefs: append([]string(nil), c.i.DocumentosRefs...), Disponible: true}}
	c.pa = &altaProveedorApp{t: t, reloj: c.reloj, contexto: d.Contexto, org: d.Preparacion.OrganizacionRef}
	c.alta = &altaTXApp{reloj: c.reloj}
	l, e := lector.NuevoV2(permisoLecturaApp(func(ctx context.Context, m lector.MaterialV2) (lector.AutorizacionV2, error) {
		c.permisos++
		if c.denegarLectura {
			return lector.AutorizacionV2{}, errors.New("privado")
		}
		contexto, e := m.Contexto()
		registroV2Exigir(t, e)
		if !mismoContexto(contexto, c.p.p.Contexto) || m.UnidadRef() != c.p.p.Preparacion.UnidadRef {
			t.Fatal("contexto/unidad")
		}
		d.Personal = c.alta.registro(t)
		d.Confirmacion.ResultadoPersonal = d.Personal.Resultado
		material, e := ct.NuevoMaterialConfirmacionIncorporacionV2(d, c.reloj.ahora)
		registroV2Exigir(t, e)
		recurso, e := m.Recurso()
		registroV2Exigir(t, e)
		a := registroV2Autoridad(t, material, c.reloj.ahora, func(x *core.DatosSolicitudAutorizacionLigadaV3) {
			x.Recurso = recurso
			x.Accion = lector.Accion
			x.Finalidad = lector.Finalidad
		}, lector.AudienciaV2)
		c.reloj.ahora = c.reloj.ahora.Add(time.Microsecond)
		return lector.AutorizacionV2{Solicitud: a.Solicitud, Decision: a.Decision, Confirmacion: a.Confirmacion, Exportacion: a.Exportacion}, nil
	}), txLecturaApp(func(ctx context.Context, sel lector.Selector, o lector.OrdenV2) (lector.Resultado, error) {
		c.lecturas++
		r := c.alta.registro(t)
		if sel.MaterialSHA256 != r.MaterialSHA256 || sel.SolicitudRef != r.Solicitud.SolicitudRef || sel.ResultadoRef != r.Resultado.ResultadoRef {
			t.Fatal("selector")
		}
		if c.cruzar {
			r.DecisionOriginalRef = "decision:cruzada"
		}
		if c.hookLectura != nil {
			c.hookLectura()
		}
		return lector.Resultado{Registro: r, DecisionLecturaRef: o.Exportacion().ResumenCapacidad().DecisionRef(), ConsumoHuellaSHA256: registroV2Hash([]byte("consumo")),
			AuditoriaLecturaRef: "aud:lectura:1", LeidaEn: c.reloj.ahora}, nil
	}), c.reloj)
	registroV2Exigir(t, e)
	b, e := puente.NuevoV2(l, c.reloj)
	registroV2Exigir(t, e)
	c.ctTX = &registroV2TX{t: t, reloj: c.reloj}
	confirmador, e := appct.NuevoServicioConfirmacionIncorporacionV2(&registroV2Proveedor{t: t, reloj: c.reloj}, b, c.ctTX, c.reloj)
	registroV2Exigir(t, e)
	fuente, e := os.ReadFile("../../modules/personal/adapters/fuenteejercicio/testdata/fuente-ejercicio.json")
	registroV2Exigir(t, e)
	fuente = bytes.ReplaceAll(fuente, []byte("expediente:ejercicio:ct:0001"), []byte(c.i.ExpedienteRef))
	terna := fuenteejercicio.TernaEsperada{Referencia: "fuente:personal:ejercicio:20260908", Version: 1, HuellaSHA256: registroV2Hash(fuente)}
	c.s, e = Nuevo(Configuracion{Preparador: c.p, FuentePersonal: fuente, TernaPersonal: terna, ProveedorAlta: c.pa, TransaccionAlta: c.alta, LectorPersonal: l, Confirmador: confirmador, Reloj: c.reloj})
	registroV2Exigir(t, e)
	fuente[0] = 'x' // constructor ha copiado fuente
	return c
}
func TestAplicacionIncorporacionNominalYReplay(t *testing.T) {
	c := nuevoCasoApp(t)
	ctx := context.Background()
	vista, e := c.s.Consultar(ctx, c.i.ExpedienteRef)
	registroV2Exigir(t, e)
	if vista.Recibo != nil || c.alta.n != 0 || c.permisos != 0 || c.ctTX.llamadas != 0 {
		t.Fatal("GET efecto")
	}
	vista.Preparacion.DocumentosRefs[0] = "mutado"
	if c.p.vista.Preparacion.DocumentosRefs[0] == "mutado" {
		t.Fatal("alias")
	}
	r, e := c.s.Confirmar(ctx, c.i)
	registroV2Exigir(t, e)
	if r.VersionSolicitudPersonal != 7 || r.VersionActualExpediente != 8 || r.VersionSeguimientoAnterior != 0 || r.VersionSeguimientoResultante != 1 ||
		r.SolicitudPersonalRef != c.i.SolicitudPersonalRef || r.RelacionRef != c.alta.original.Resultado.RelacionRef || c.lecturas != 2 || c.ctTX.efectos != 1 {
		t.Fatal("nominal")
	}
	c.reloj.ahora = c.reloj.ahora.Add(time.Microsecond)
	replay, e := c.s.Confirmar(ctx, c.i)
	registroV2Exigir(t, e)
	if !reflect.DeepEqual(r, replay) || c.ctTX.efectos != 1 || c.alta.n != 2 || c.lecturas != 4 {
		t.Fatal("replay no preservado")
	}
}
func TestAplicacionIncorporacionFronteras(t *testing.T) {
	for _, caso := range []string{"solicitud", "version", "documentos", "motivo", "periodo", "contexto", "alta_denegada", "lector_denegado", "original_cruzado", "cancel_preparar", "cancel_lector", "reloj", "error_privado"} {
		t.Run(caso, func(t *testing.T) {
			c := nuevoCasoApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch caso {
			case "solicitud":
				c.p.p.SolicitudPersonal.SolicitudRef = "solicitud:otra"
			case "version":
				c.p.p.VersionActualExpediente = 9
			case "documentos":
				c.p.p.Documentos[0].Referencia = "documento:otro"
			case "motivo":
				c.p.p.MotivoClave = "otro"
			case "periodo":
				c.p.p.Periodo.Hasta = c.p.p.Periodo.Desde
			case "contexto":
				c.pa.org = "organizacion:otra"
			case "alta_denegada":
				c.pa.denegar = true
			case "lector_denegado":
				c.denegarLectura = true
			case "original_cruzado":
				c.cruzar = true
			case "cancel_preparar":
				c.p.hook = cancel
			case "cancel_lector":
				c.hookLectura = cancel
			case "reloj":
				c.p.hook = func() { c.reloj.ahora = c.reloj.ahora.Add(-time.Hour) }
			case "error_privado":
				c.p.err = errors.New("datos privados")
			}
			r, e := c.s.Confirmar(ctx, c.i)
			if e == nil || !reflect.DeepEqual(r, ct.ReciboIncorporacionAplicacionV2{}) || c.ctTX.llamadas != 0 {
				t.Fatal("fallo no cerrado")
			}
			if e.Error() == "datos privados" {
				t.Fatal("filtración")
			}
			if caso != "lector_denegado" && caso != "original_cruzado" && caso != "cancel_lector" && c.alta.n != 0 {
				t.Fatal("efecto prematuro")
			}
		})
	}
}
func TestAplicacionIncorporacionGETYForma(t *testing.T) {
	for _, caso := range []string{"extra_recibo", "expediente", "error", "cancel", "ultimo_reloj", "intencion_vacia", "consentimiento", "duplicado"} {
		t.Run(caso, func(t *testing.T) {
			c := nuevoCasoApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch caso {
			case "intencion_vacia", "consentimiento", "duplicado":
				i := c.i.Copia()
				if caso == "intencion_vacia" {
					i = ct.IntencionIncorporacionAplicacionV2{}
				}
				if caso == "consentimiento" {
					i.ConfirmaRevisionPersonal = false
				}
				if caso == "duplicado" {
					i.DocumentosRefs = append(i.DocumentosRefs, i.DocumentosRefs[0])
				}
				if _, e := c.s.Confirmar(ctx, i); e == nil || c.p.llamadas != 0 {
					t.Fatal("forma")
				}
				return
			case "extra_recibo":
				c.p.vista.Recibo = &ct.ReciboIncorporacionAplicacionV2{}
			case "expediente":
				c.p.vista.ExpedienteRef = "exp:otro"
			case "error":
				c.p.err = errors.New("privado")
			case "cancel":
				c.p.hook = cancel
			case "ultimo_reloj":
				c.reloj.despues = func(n int) {
					if n == 2 {
						cancel()
					}
				}
			}
			p, e := c.s.Consultar(ctx, c.i.ExpedienteRef)
			if e == nil || !reflect.DeepEqual(p, ct.ProyeccionIncorporacionAplicacionV2{}) || c.alta.n != 0 || c.permisos != 0 {
				t.Fatal("GET no cerrado")
			}
		})
	}
}

func TestAplicacionIncorporacionCancelacionFinalYProyeccionHistorica(t *testing.T) {
	baseline := nuevoCasoApp(t)
	r, e := baseline.s.Confirmar(context.Background(), baseline.i)
	registroV2Exigir(t, e)
	final := baseline.reloj.lecturas
	t.Run("ultimo_reloj_confirmar", func(t *testing.T) {
		c := nuevoCasoApp(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		c.reloj.despues = func(n int) {
			if n == final {
				cancel()
			}
		}
		r, e := c.s.Confirmar(ctx, c.i)
		if !errors.Is(e, context.Canceled) || !reflect.DeepEqual(r, ct.ReciboIncorporacionAplicacionV2{}) || c.ctTX.efectos != 1 {
			t.Fatal("postcommit no ocultar cancelación")
		}
	})
	t.Run("GET_historico_sin_autorizar_ni_alta", func(t *testing.T) {
		baseline.p.vista.Preparacion = nil
		baseline.p.vista.Recibo = &r
		baseline.reloj.ahora = baseline.reloj.ahora.Add(24 * time.Hour)
		na, nl, nct := baseline.alta.n, baseline.lecturas, baseline.ctTX.llamadas
		vista, e := baseline.s.Consultar(context.Background(), baseline.i.ExpedienteRef)
		registroV2Exigir(t, e)
		if !reflect.DeepEqual(*vista.Recibo, r) || na != baseline.alta.n || nl != baseline.lecturas || nct != baseline.ctTX.llamadas {
			t.Fatal("GET efectos/refecha")
		}
		vista.Recibo.ReciboRef = "mutado"
		if baseline.p.vista.Recibo.ReciboRef == "mutado" {
			t.Fatal("alias histórico")
		}
		b, e := json.Marshal(vista)
		registroV2Exigir(t, e)
		for _, campo := range []string{"Material", "Contexto", "Capacidad", "Recuperado", "confirmacion_ref", "material_canonico", "sesion_ref"} {
			if bytes.Contains(b, []byte(campo)) {
				t.Fatal("proyección expone autoridad")
			}
		}
	})
}
func TestAplicacionIncorporacionDependenciasYCopias(t *testing.T) {
	c := nuevoCasoApp(t)
	for _, nombre := range []string{"preparador", "proveedor", "tx", "lector", "ct", "reloj", "fuente", "terna"} {
		t.Run(nombre, func(t *testing.T) {
			cfg := c.s.c
			switch nombre {
			case "preparador":
				var p *preparadorAplicacion
				cfg.Preparador = p
			case "proveedor":
				var p *altaProveedorApp
				cfg.ProveedorAlta = p
			case "tx":
				var p *altaTXApp
				cfg.TransaccionAlta = p
			case "lector":
				cfg.LectorPersonal = nil
			case "ct":
				cfg.Confirmador = nil
			case "reloj":
				var p *registroV2Reloj
				cfg.Reloj = p
			case "fuente":
				cfg.FuentePersonal = []byte("{}")
			case "terna":
				cfg.TernaPersonal.Version++
			}
			if s, e := Nuevo(cfg); e == nil || s != nil {
				t.Fatal("dependencia aceptada")
			}
		})
	}
	i := c.i.Copia()
	i.DocumentosRefs[0] = "otro"
	if c.i.DocumentosRefs[0] == "otro" {
		t.Fatal("alias intención")
	}
	for _, metodo := range []string{"GET", "POST"} {
		t.Run("nil_"+metodo, func(t *testing.T) {
			var s *Servicio
			if metodo == "GET" {
				_, e := s.Consultar(context.Background(), c.i.ExpedienteRef)
				if e == nil {
					t.Fatal("nil")
				}
			} else {
				_, e := s.Confirmar(context.Background(), c.i)
				if e == nil {
					t.Fatal("nil")
				}
			}
		})
	}
}
