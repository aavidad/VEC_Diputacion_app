package postgres

// Fixtures de ejercicio: contexto/registro dobles explícitos; evaluación,
// COSE, verificación y exportación reales. La instantánea se fija ANTES de
// cualquiera de las dos solicitudes y no cambia con la acción.
import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"

	gocose "github.com/veraison/go-cose"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func instantaneaLectorV2(t *testing.T, contexto ct.ContextoAutorizacionAltaV3, org, unidad string, ahora time.Time) core.InstantaneaAutorizacion {
	t.Helper()
	v, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	rol := core.VersionRol{RolID: "tecnico_rrhh", Version: 1, Nombre: "Ejercicio CT y Personal", Estado: core.EstadoVersionRolPublicada,
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
		Concesiones: []core.ConcesionRol{
			{Accion: ct.AccionConfirmarIncorporacion, ModuloID: ct.ModuloContratacion, TipoRecurso: ct.TipoRecursoConfirmacionIncorporacionV2, Finalidades: []string{ct.FinalidadConfirmarIncorporacion}, GarantiaMinima: core.AuthAssuranceHigh},
			{Accion: lector.Accion, ModuloID: "personal", TipoRecurso: lector.TipoRecursoV2, Finalidades: []string{lector.Finalidad}, GarantiaMinima: core.AuthAssuranceHigh},
		}}
	h, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	s := core.InstantaneaAutorizacion{
		AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-ejercicio-ct-personal", Version: 1, PerfilActivoRef: v.PerfilActivoRef, PrincipalID: v.PrincipalID, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
			Ambitos:      []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{org}}, {Clave: "unidad_ref", Valores: []string{unidad}}},
			VigenteDesde: ahora.Add(-48 * time.Hour), VigenteHasta: ahora.Add(48 * time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-49 * time.Hour)},
		VersionRol: rol, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: h}
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	return s
}

func permisoFijoV2(t *testing.T, s core.SolicitudAutorizacionLigadaV3, contexto ct.ContextoAutorizacionAltaV3, snapshot core.InstantaneaAutorizacion, ahora time.Time, referencia, audiencia string) lector.AutorizacionV2 {
	t.Helper()
	ctx := context.Background()
	d, err := s.Datos()
	if err != nil {
		t.Fatal(err)
	}
	evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, snapshot, referencia, ahora, ahora.Add(90*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(s, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	concedida, _, err := decision.Resultado()
	if err != nil {
		t.Fatal(err)
	}
	a := lector.AutorizacionV2{Solicitud: s, Decision: decision}
	if !concedida {
		return a
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, decision, d.ReferenciaMotivo, contexto.Resultado)
	if err != nil {
		t.Fatal(err)
	}
	registro, err := vecports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroV2_registroConcesionV3Doble{registradaEn: ahora}, orden)
	if err != nil {
		t.Fatal(err)
	}
	a.Confirmacion = registro
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(privada)
	cabecera := core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:personal", Audiencia: "vec/prueba/personal"}
	sf, err := vecports.NuevaSolicitudFirmaAtestacionAutorizacionV3(cabecera, decision, d.ReferenciaMotivo, contexto.Resultado)
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
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(config, registroV2_relojVinculoPrueba{instante: ahora})
	if err != nil {
		t.Fatal(err)
	}
	prueba, err := verificador.Verificar(ctx, s, decision, d.ReferenciaMotivo, contexto.Resultado, atestacion)
	if err != nil {
		t.Fatal(err)
	}
	materialClave := make([]byte, 32)
	defer clear(materialClave)
	if _, err = rand.Read(materialClave); err != nil {
		t.Fatal(err)
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:prueba", 1, materialClave, "emisor:prueba:personal", audiencia,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	if err != nil {
		t.Fatal(err)
	}
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, registroV2_relojVinculoPrueba{instante: ahora.Add(time.Microsecond)})
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(ctx, s, decision, d.ReferenciaMotivo, contexto.Resultado, atestacion, prueba)
	if err != nil {
		t.Fatal(err)
	}
	material, err := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(s, decision, d.ReferenciaMotivo, contexto.Resultado, atestacion, prueba, capacidad, raiz)
	if err != nil {
		t.Fatal(err)
	}
	a.Exportacion, err = material.ExportarMaterialParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func solicitudLectorV2(t *testing.T, m lector.MaterialV2, mutar func(*core.DatosSolicitudAutorizacionLigadaV3)) core.SolicitudAutorizacionLigadaV3 {
	t.Helper()
	r, err := m.Recurso()
	if err != nil {
		t.Fatal(err)
	}
	cor, err := core.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), &registroV2GeneradorCorrelacion{referencia: "correlacion_22222222222222222222222222222222"})
	if err != nil {
		t.Fatal(err)
	}
	cx, err := m.Contexto()
	if err != nil {
		t.Fatal(err)
	}
	d := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: cx.Vinculo, Accion: lector.Accion, Finalidad: lector.Finalidad, Recurso: r, Correlacion: cor,
		ReferenciaMotivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}}
	if mutar != nil {
		mutar(&d)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
