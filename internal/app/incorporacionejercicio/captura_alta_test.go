package incorporacionejercicio

// Fixtures nominales públicas copiadas; autoridades y TX son DOBLES, no PG.
import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	gocose "github.com/veraison/go-cose"
	"reflect"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registroConcesionV3Doble struct{ registradaEn time.Time }

func (r registroConcesionV3Doble) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.registradaEn, nil
}

type revalidadorVinculoPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorVinculoPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type resolutorResultadoVinculoPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorResultadoVinculoPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type relojVinculoPrueba struct{ instante time.Time }

func (d relojVinculoPrueba) Ahora() time.Time { return d.instante }

func contextoAutorizacionAltaV3Prueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoAutorizacionAltaV3 {
	return contextoAutorizacionAltaV3PruebaConMarcas(t, ahora, "a", "a")
}

func contextoAutorizacionAltaV3PruebaConMarcas(
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
			marcaActor + marcaPerfil,
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
		revalidadorVinculoPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojVinculoPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

func concesionAutorizacionV3Prueba(
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
			registroConcesionV3Doble{registradaEn: ahora},
			orden,
		)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion, nil
}

type correlacionPrueba struct{}

func (correlacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil
}

// Exportación construida por el camino nominal real: decisión, concesión,
// firma COSE Ed25519 efímera, verificación V3, capacidad HMAC y material común.
// Ninguna clave sale de la memoria del test; esto no acredita gobierno SQL.
func autorizacionPrueba(t *testing.T, actor AutoridadAlta, m MaterialAlta, ahora time.Time, numero int, conceder bool,
	mutar func(*core.DatosSolicitudAutorizacionLigadaV3),
) AutorizacionAlta {
	t.Helper()
	ctx := context.Background()
	r, err := RecursoAltaEjercicio(m)
	if err != nil {
		t.Fatal(err)
	}
	cor, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, correlacionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	d := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: actor.Contexto.Vinculo, Accion: AccionAltaEjercicio, Finalidad: FinalidadAltaEjercicio, Recurso: r, Correlacion: cor,
		ReferenciaMotivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}}
	if mutar != nil {
		mutar(&d)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	decision, registro, err := concesionAutorizacionV3Prueba(t, s, actor.Contexto.Resultado, d.ReferenciaMotivo, ahora, fmt.Sprintf("dec_%032x", numero), conceder)
	if err != nil {
		t.Fatal(err)
	}
	a := AutorizacionAlta{Solicitud: s, Decision: decision, Confirmacion: registro}
	if !conceder {
		return a
	}
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cabecera := core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:personal", Audiencia: "vec/prueba/personal"}
	sf, err := vecports.NuevaSolicitudFirmaAtestacionAutorizacionV3(cabecera, decision, d.ReferenciaMotivo, actor.Contexto.Resultado)
	if err != nil {
		t.Fatal(err)
	}
	mensaje, err := sf.Mensaje()
	if err != nil {
		t.Fatal(err)
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		t.Fatal(err)
	}
	cose := gocose.NewSign1Message()
	cose.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	cose.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(cabecera.ClaveID)
	cose.Payload = mensaje
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, privada)
	if err != nil {
		t.Fatal(err)
	}
	if err = cose.Sign(rand.Reader, aad, firmante); err != nil {
		t.Fatal(err)
	}
	cose.Payload = nil
	cose.Headers.RawProtected = nil
	cose.Headers.RawUnprotected = nil
	sobre, err := cose.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	firma, err := vecports.NuevoResultadoFirmaAtestacionAutorizacionV3(sf, sobre, "evidencia:prueba:personal", ahora)
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := vecports.NuevaAtestacionAutorizacionV3(sf, firma)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(cabecera.ClaveID, 1, publica, cabecera.Audiencia, confianza.EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:personal", 1, ahora.Add(-time.Minute), ahora.Add(time.Hour), raiz)
	if err != nil {
		t.Fatal(err)
	}
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(config, relojVinculoPrueba{instante: ahora})
	if err != nil {
		t.Fatal(err)
	}
	prueba, err := verificador.Verificar(ctx, s, decision, d.ReferenciaMotivo, actor.Contexto.Resultado, atestacion)
	if err != nil {
		t.Fatal(err)
	}
	materialClave := make([]byte, 32)
	if _, err = rand.Read(materialClave); err != nil {
		t.Fatal(err)
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:prueba", 1, materialClave, "emisor:prueba:personal", AudienciaAltaEjercicio,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	if err != nil {
		t.Fatal(err)
	}
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, relojVinculoPrueba{instante: ahora.Add(time.Microsecond)})
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(ctx, s, decision, d.ReferenciaMotivo, actor.Contexto.Resultado, atestacion, prueba)
	if err != nil {
		t.Fatal(err)
	}
	material, err := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(s, decision, d.ReferenciaMotivo, actor.Contexto.Resultado, atestacion, prueba, capacidad, raiz)
	if err != nil {
		t.Fatal(err)
	}
	a.Exportacion, err = material.ExportarMaterialParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

type AutoridadAlta = personal.AutoridadAlta
type MaterialAlta = personal.MaterialAlta
type AutorizacionAlta = personal.AutorizacionAlta

const AudienciaAltaEjercicio = personal.AudienciaAltaEjercicio

const AccionAltaEjercicio = personal.AccionAltaEjercicio
const FinalidadAltaEjercicio = personal.FinalidadAltaEjercicio

var RecursoAltaEjercicio = personal.RecursoAltaEjercicio

type delegadoCaptura func(context.Context, personal.OrdenAlta) (personal.ResultadoTransaccionAlta, error)

func (f delegadoCaptura) RegistrarORecuperarAlta(c context.Context, o personal.OrdenAlta) (personal.ResultadoTransaccionAlta, error) {
	return f(c, o)
}
func ordenParaCaptura(t *testing.T) (personal.OrdenAlta, personal.ResultadoTransaccionAlta, *registroV2Reloj) {
	c := nuevoCasoApp(t)
	_, e := c.s.Confirmar(context.Background(), c.i)
	registroV2Exigir(t, e)
	return c.alta.orden, personal.ResultadoTransaccionAlta{Recibo: c.alta.original, DecisionConsumidaRef: c.alta.orden.Exportacion().ResumenCapacidad().DecisionRef()}, c.reloj
}
func TestCapturaAltaPublicacionUnica(t *testing.T) {
	o, r, reloj := ordenParaCaptura(t)
	n := 0
	c := &capturaAlta{reloj: reloj, delegado: delegadoCaptura(func(context.Context, personal.OrdenAlta) (personal.ResultadoTransaccionAlta, error) {
		n++
		return r, nil
	})}
	recibido, e := c.RegistrarORecuperarAlta(context.Background(), o)
	registroV2Exigir(t, e)
	recibido.Recibo.AuditoriaRef = "modificado"
	oo, rr, e := c.tomar(context.Background())
	registroV2Exigir(t, e)
	if rr != r || oo.Material() != o.Material() || n != 1 {
		t.Fatal("copia nominal")
	}
	if _, _, e = c.tomar(context.Background()); e == nil {
		t.Fatal("tomar dos veces")
	}
	if _, e = c.RegistrarORecuperarAlta(context.Background(), o); e == nil || n != 1 {
		t.Fatal("delegar dos veces")
	}
}
func TestCapturaAltaCeroAnteFallo(t *testing.T) {
	o, r, relojBase := ordenParaCaptura(t)
	for _, caso := range []string{"delegado_error", "recibo_invalido", "cancel_delegado", "cancel_ultimo_reloj", "retroceso", "orden_cero", "nil_delegado"} {
		t.Run(caso, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reloj := &registroV2Reloj{ahora: relojBase.ahora}
			n := 0
			tx := delegadoCaptura(func(context.Context, personal.OrdenAlta) (personal.ResultadoTransaccionAlta, error) {
				n++
				v := r
				switch caso {
				case "delegado_error":
					return v, errors.New("privado")
				case "recibo_invalido":
					v.Recibo.AuditoriaRef = ""
				case "cancel_delegado":
					cancel()
				case "retroceso":
					reloj.ahora = reloj.ahora.Add(-time.Second)
				}
				return v, nil
			})
			c := &capturaAlta{reloj: reloj, delegado: tx}
			if caso == "nil_delegado" {
				var x delegadoCaptura
				c.delegado = x
			}
			if caso == "cancel_ultimo_reloj" {
				reloj.despues = func(n int) {
					if n == 2 {
						cancel()
					}
				}
			}
			oo := o
			if caso == "orden_cero" {
				oo = personal.OrdenAlta{}
			}
			v, e := c.RegistrarORecuperarAlta(ctx, oo)
			if e == nil || !reflect.DeepEqual(v, personal.ResultadoTransaccionAlta{}) {
				t.Fatal("resultado no cero")
			}
			_, rr, e := c.tomar(context.Background())
			if e == nil || !reflect.DeepEqual(rr, personal.ResultadoTransaccionAlta{}) {
				t.Fatal("captura publicada")
			}
			if (caso == "orden_cero" || caso == "nil_delegado") && n != 0 {
				t.Fatal("delegación prematura")
			}
			if errors.Is(e, ct.ErrComposicionIncorporacionAplicacion) == false {
				t.Fatal("error no saneado")
			}
		})
	}
}
