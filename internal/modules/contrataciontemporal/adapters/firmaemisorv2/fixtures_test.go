package firmaemisorv2

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type relojPrueba struct{ ahora time.Time }

func (r *relojPrueba) Ahora() time.Time { return r.ahora }

type fuentePrueba struct {
	base     ContextoActorFirmaV2
	fallo    error
	antes    func()
	llamadas int
}

func (f *fuentePrueba) RevalidarContextoActorFirmaV2(context.Context) (ContextoActorFirmaV2, error) {
	f.llamadas++
	if f.antes != nil {
		f.antes()
	}
	return f.base, f.fallo
}

type emisorPrueba struct {
	t                    *testing.T
	reloj                *relojPrueba
	llamadas             int
	solicitud            vd.SolicitudAutorizacionLigadaV3
	fallo                error
	antes                func()
	campos, obligaciones []string
	alterar              func(vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	exportador           vp.ExportadorMaterialConsumoAutorizacionAtestadaV3
}

type exportadorPrueba struct {
	material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	fallo    error
	antes    func()
}

func (*exportadorPrueba) String() string         { return "[EXPORTACION-PRUEBA]" }
func (e *exportadorPrueba) LogValue() slog.Value { return slog.StringValue(e.String()) }

func (e *exportadorPrueba) ExportarMaterialParaConsumidor() (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if e.antes != nil {
		e.antes()
	}
	return e.material, e.fallo
}

type registroPrueba time.Time

func (r registroPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, vp.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return time.Time(r), nil
}

// Este emisor evalúa/registrar la decisión con autoridades sintéticas y exporta
// un transporte estructural. No acredita COSE, SQL ni consumo durable.
func (e *emisorPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s vd.SolicitudAutorizacionLigadaV3, b vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	e.solicitud = s
	if e.antes != nil {
		e.antes()
	}
	if e.fallo != nil {
		return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.fallo
	}
	if ctx.Err() != nil {
		return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ctx.Err()
	}
	d, err := s.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	actor, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	ahora := e.reloj.ahora
	var ambitos []vd.AmbitoPerfil
	for _, clave := range []string{"organizacion_ref", "unidad_ref"} {
		if v, ok := d.Recurso.Ambitos[clave]; ok {
			ambitos = append(ambitos, vd.AmbitoPerfil{Clave: clave, Valores: []string{v}})
		}
	}
	i := instantaneaPrueba(e.t, actor.PrincipalID, actor.PerfilActivoRef, ambitos, ahora,
		vd.ConcesionRol{Accion: d.Accion, ModuloID: d.Recurso.ModuloID, TipoRecurso: d.Recurso.Tipo,
			Finalidades: []string{d.Finalidad}, GarantiaMinima: vd.AuthAssuranceHigh, CamposPermitidos: e.campos, Obligaciones: e.obligaciones})
	evidencia, err := vd.NuevaEvidenciaEvaluacionAutorizacionV3(s, i, "decision:prueba", ahora, ahora.Add(90*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	decision, err := vd.NuevaDecisionAutorizacionLigadaV3(s, evidencia)
	if err != nil {
		e.t.Fatal(err)
	}
	orden, err := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, decision, d.ReferenciaMotivo, b)
	if err != nil {
		e.t.Fatal(err)
	}
	confirmacion, err := vp.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroPrueba(ahora), orden)
	if err != nil {
		e.t.Fatal(err)
	}
	dc, _ := vd.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	mc, _ := vd.RepresentacionCanonicaMotivoAutorizacionV2(d.ReferenciaMotivo)
	hd, _ := vd.HuellaSHA256DecisionAutorizacionV3(decision)
	hm, _ := vd.HuellaSHA256MotivoAutorizacionV2(d.ReferenciaMotivo)
	hr, _ := d.Recurso.HuellaContextoAutorizacionSHA256()
	audiencia := ports.AudienciaFirmaVecV2
	if d.Accion == ports.AccionRegistrarFirmaExterna {
		audiencia = ports.AudienciaFirmaExternaV2
	}
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", hd, hm, b.RegistroContextoRef, b.HuellaSHA256, d.Accion,
		d.Recurso.Referencia, hr, audiencia, ahora, ahora.Add(5*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		e.t.Fatal(err)
	}
	material, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen, dc, mc, b.RepresentacionCanonica,
		b.Contexto.Instantanea.PersonaVersion, b.Contexto.Instantanea.PerfilVersion, []byte("prueba"), []byte("prueba"), []byte("prueba"), spki)
	if err != nil {
		e.t.Fatal(err)
	}
	if e.alterar != nil {
		material = e.alterar(material)
	}
	if e.exportador != nil {
		return decision, confirmacion, e.exportador, nil
	}
	return decision, confirmacion, &exportadorPrueba{material: material}, nil
}

func instantaneaPrueba(t *testing.T, principal, perfil string, ambitos []vd.AmbitoPerfil, ahora time.Time, concesion vd.ConcesionRol) vd.InstantaneaAutorizacion {
	t.Helper()
	version := vd.VersionRol{RolID: "rol_prueba", Version: 1, Nombre: "Prueba", Estado: vd.EstadoVersionRolPublicada,
		Concesiones: []vd.ConcesionRol{concesion}, PublicadaPor: "seguridad-prueba", PublicadaEn: ahora.Add(-24 * time.Hour)}
	hc, err := vd.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	return vd.InstantaneaAutorizacion{AsignacionPerfil: vd.AsignacionPerfil{AsignacionID: "asignacion-prueba", Version: 1,
		PerfilActivoRef: perfil, PrincipalID: principal, VersionRolRef: version.Referencia(), Estado: vd.EstadoAsignacionPerfilActiva,
		Ambitos: ambitos, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "identidad-prueba", EmitidaEn: ahora.Add(-2 * time.Hour)},
		VersionRol: version, ControlVigenciaVersionRol: vd.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1,
			Estado: vd.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: hc}
}

// autorizacionPrueba sustituye a la fuente común de autorización: devuelve la
// asignación del perfil pedido con los ámbitos que fija cada prueba.
type autorizacionPrueba struct {
	t        *testing.T
	ahora    time.Time
	ambitos  []vd.AmbitoPerfil
	ajena    bool
	fallo    error
	llamadas int
}

func (a *autorizacionPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (vd.InstantaneaAutorizacion, error) {
	a.llamadas++
	if a.fallo != nil {
		return vd.InstantaneaAutorizacion{}, a.fallo
	}
	if a.ajena {
		perfil = "prf_ajeno_0123456789abcdef"
	}
	return instantaneaPrueba(a.t, principal, perfil, a.ambitos, a.ahora, vd.ConcesionRol{Accion: ports.AccionRegistrarFirmaVec,
		ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaVec, Finalidades: []string{ports.FinalidadFirmaDocumento},
		GarantiaMinima: vd.AuthAssuranceHigh}), nil
}

func escenario(t *testing.T, via string) (*Emisor, *fuentePrueba, *emisorPrueba, ports.MaterialFirmaVerificadaV2, vd.RecursoAutorizable, context.Context) {
	t.Helper()
	ahora := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", vd.AuthMethodCertificate, vd.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	actor, _ := vinculo.Datos()
	h, cert, original, firmado := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
	evidencia := []byte(`[{"orden":1}]`)
	he := sha256.Sum256(evidencia)
	m := ports.MaterialFirmaVerificadaV2{MaterialFirmaExterna: ports.MaterialFirmaExterna{
		Via: via, OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba", VersionExpediente: 7,
		Documento: "informe_definitivo", CatalogoRef: "catalogo:prueba", CatalogoHuella: h, PasoRef: "paso:primero", PasoOrden: 1, Secuencia: 1,
		HistoriaHuella: h, OriginalRef: "original:prueba", OriginalVersion: 1, OriginalHuella: original, FirmadoHuella: firmado, CertificadoHuella: cert, FirmanteRef: "ref:" + cert,
		FirmantePrincipalRef: actor.PrincipalID, PerfilFirmanteRef: "perfil:firmante", CargoFirmante: "ct_cargo_prueba", UnidadFirmanteRef: "unidad:prueba", PerfilActivoFirmanteRef: actor.PerfilActivoRef,
		AsignacionFirmanteRef: "asignacion:prueba", AsignacionFirmanteVersion: 1, AsignacionFirmanteHuella: h, VersionRolFirmanteRef: "rol:ct_cargo_prueba:v1", VersionRolFirmanteHuella: h,
		ControlVigenciaFirmanteRef: "rol:ct_cargo_prueba:v1", ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: h,
		AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z", PoliticaVerificacion: "politica:vec:firma:verificacion-autonoma:v2",
		RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente", ClaveIdempotencia: "clave-prueba-firma-000001", DocumentoCustodiaRef: "custodia:prueba", DocumentoCustodiaVersion: 1},
		PerfilActivoOperadorRef: actor.PerfilActivoRef, CuentaFirmanteRef: actor.CuentaRef, VinculoCredencialFirmanteRef: "vinculo:prueba", VinculoCredencialFirmanteRevision: 1, VinculoCredencialFirmanteHuella: h,
		RolIDFirmante: "ct_cargo_prueba", CatalogoVersion: 1, EntradaDocumentoRef: "original:prueba", EntradaDocumentoVersion: 1, EntradaDocumentoLongitud: 100, EntradaDocumentoHuella: original,
		OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 120, 180, 20}, RevisionHuellaSHA256: firmado, ContenidoFirmadoHuellaSHA256: h, RevisionLongitud: 200,
		EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(he[:]), ComprobadaEn: ahora}
	if via == ports.ViaFirmaExternaPortafirmas {
		m.FirmantePrincipalRef = "per_firmante_externo"
		m.PerfilActivoFirmanteRef = "prf_firmante_externo"
		m.CuentaFirmanteRef = "cuenta:firmante-externo"
		m.ReferenciaPortafirmasDeclarada = "portafirmas:prueba"
		m.FechaPortafirmasDeclarada = "2026-10-03T11:00:00Z"
	}
	if m.Validar() != nil {
		t.Fatal("material sintético inválido")
	}
	r := recursoPrueba(t, m)
	f := &fuentePrueba{base: ContextoActorFirmaV2{Resultado: resultado, Vinculo: vinculo, CertificadoCanalSHA256: cert}}
	reloj := &relojPrueba{ahora: ahora}
	e := &emisorPrueba{t: t, reloj: reloj}
	autorizacion := &autorizacionPrueba{t: t, ahora: ahora, ambitos: []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef}}}}
	adaptador, err := NuevoEmisorConAmbitos(f, e, vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_prueba", CatalogoVersion: 1, CatalogoHuellaSHA256: h, EntradaClave: "motivo_11111111111111111111111111111111"}, reloj, autorizacion)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := vp.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return adaptador, f, e, m, r, ctx
}

func recursoPrueba(t *testing.T, m ports.MaterialFirmaVerificadaV2) vd.RecursoAutorizable {
	t.Helper()
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	tipo := ports.TipoRecursoFirmaVec
	if m.Via == ports.ViaFirmaExternaPortafirmas {
		tipo = ports.TipoRecursoFirmaExterna
	}
	return vd.RecursoAutorizable{Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: tipo,
		Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef}, Atributos: map[string]string{"material_sha256": h, "descriptor_firma_sha256": strings.Repeat("e", 64)}}
}
