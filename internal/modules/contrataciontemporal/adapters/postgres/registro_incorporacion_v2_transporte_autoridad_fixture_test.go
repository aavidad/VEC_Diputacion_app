package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	gocose "github.com/veraison/go-cose"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

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
	contenido, err := os.ReadFile("../../../personal/adapters/fuenteejercicio/testdata/fuente-ejercicio.json")
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
func registroV2Autoridad(t *testing.T, m ct.MaterialConfirmacionIncorporacionV2, ahora time.Time, mutar func(*core.DatosSolicitudAutorizacionLigadaV3), audiencia string, capturar *core.InstantaneaAutorizacion) ct.AutorizacionConfirmacionIncorporacionV2 {
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
	decision, registro, err := registroV2_concesionAutorizacionV3Prueba(t, s, d.Contexto.Resultado, ds.ReferenciaMotivo, ahora.Add(-time.Millisecond), fmt.Sprintf("dec_%032x", ahora.UnixMicro()), true, capturar)
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
