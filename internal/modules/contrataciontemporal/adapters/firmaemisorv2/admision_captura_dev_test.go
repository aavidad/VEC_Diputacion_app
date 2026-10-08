package firmaemisorv2

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type denegacionesCapturaPrueba struct{}

func (denegacionesCapturaPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, vp.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}

type motivoCapturaPrueba struct{}

func (motivoCapturaPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, vd.ReferenciaEntradaCatalogo, time.Time) error {
	return nil
}

type decisionRefCapturaPrueba struct{}

func (decisionRefCapturaPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return "decision:prueba", nil
}

type relojCancelaAdmisionPrueba struct {
	ahora                time.Time
	llamadas, cancelarEn int
	cancelar             context.CancelFunc
}

func (r *relojCancelaAdmisionPrueba) Ahora() time.Time {
	r.llamadas++
	if r.llamadas == r.cancelarEn && r.cancelar != nil {
		r.cancelar()
	}
	return r.ahora
}

// El PDP es el servicio V real en memoria; sólo la atestación criptográfica
// de esta prueba es estructural. No acredita consumo SQL ni firma legal.
type emisorCapturaRealEnMemoriaPrueba struct {
	t                 *testing.T
	servicio          *vecapp.ServicioAutorizacionSolicitudLigadaV3
	reloj             *relojPrueba
	llamadas          int
	ligarOtroMaterial bool
}

func (e *emisorCapturaRealEnMemoriaPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context,
	vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2,
) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Fatal("la vía DEV llamó al emisor HIGH")
	return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
}
func (e *emisorCapturaRealEnMemoriaPrueba) EmitirMaterialAutorizacionAtestadaV3ConCaptura(ctx context.Context,
	s vd.SolicitudAutorizacionLigadaV3, b vd.ResultadoContextoActorRegistradoV2,
) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, vp.CapturaEvaluacionSolicitudLigadaV3, error) {
	e.llamadas++
	decision, confirmacion, captura, err := e.servicio.ExigirSolicitudLigadaV3ConCaptura(ctx, s, b)
	if err != nil {
		return decision, confirmacion, nil, captura, err
	}
	d, err := s.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	dc, _ := vd.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	mc, _ := vd.RepresentacionCanonicaMotivoAutorizacionV2(d.ReferenciaMotivo)
	hd, _ := vd.HuellaSHA256DecisionAutorizacionV3(decision)
	hm, _ := vd.HuellaSHA256MotivoAutorizacionV2(d.ReferenciaMotivo)
	hr, _ := d.Recurso.HuellaContextoAutorizacionSHA256()
	ahora := e.reloj.Ahora()
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", hd, hm,
		b.RegistroContextoRef, b.HuellaSHA256, d.Accion, d.Recurso.Referencia, hr,
		ports.AudienciaFirmaVecV2, ahora, ahora.Add(5*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		e.t.Fatal(err)
	}
	material, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		dc, mc, b.RepresentacionCanonica, b.Contexto.Instantanea.PersonaVersion,
		b.Contexto.Instantanea.PerfilVersion, []byte("prueba"), []byte("prueba"), []byte("prueba"), spki)
	if err != nil {
		e.t.Fatal(err)
	}
	materialCaptura := material
	if e.ligarOtroMaterial {
		materialCaptura, err = vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("y", 512)), resumen,
			dc, mc, b.RepresentacionCanonica, b.Contexto.Instantanea.PersonaVersion,
			b.Contexto.Instantanea.PerfilVersion, []byte("prueba"), []byte("prueba"), []byte("prueba"), spki)
		if err != nil {
			e.t.Fatal(err)
		}
	}
	ligada, err := captura.LigarMaterial(s, b, decision, confirmacion, materialCaptura, ports.AudienciaFirmaVecV2)
	if err != nil {
		return decision, confirmacion, nil, nil, err
	}
	return decision, confirmacion, &exportadorPrueba{material: material}, ligada, nil
}

func TestFirmaVecDesarrolloConsumeCapturaPDPRealEnMemoria(t *testing.T) {
	base, fuente, _, m, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	ahora := base.reloj.Ahora().UTC().Truncate(time.Microsecond)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		fuente.base.Resultado.Contexto.PersonaRef, fuente.base.Resultado.Contexto.PerfilActivoRef,
		vd.AuthMethodCertificate, vd.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	fuente.base.Resultado, fuente.base.Vinculo = resultado, vinculo
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	m.CuentaFirmanteRef = datos.CuentaRef
	r := recursoPrueba(t, m)
	snapshot := instantaneaPrueba(t, datos.PrincipalID, datos.PerfilActivoRef,
		[]vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef}}}, ahora,
		vd.ConcesionRol{Accion: ports.AccionRegistrarFirmaVec, ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoFirmaVec, Finalidades: []string{ports.FinalidadFirmaDocumento},
			GarantiaMinima: vd.AuthAssuranceSubstantial})
	rolSHA, _ := snapshot.VersionRol.HuellaSHA256()
	controlSHA, _ := snapshot.ControlVigenciaVersionRol.HuellaSHA256()
	gate, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{
		EntornoConfiable: vecapp.EntornoAdmisionFirmaDesarrollo, Reloj: base.reloj,
		Politica: vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo{
			Referencia: "politica:ct:firma_vec_dev:v1", HuellaSHA256: strings.Repeat("a", 64), Version: 1,
			RetiradaEn: ahora.Add(10 * time.Minute), PoliticaAutenticacionRef: datos.PoliticaGarantiaRef,
			PoliticaAutenticacionSHA256: datos.PoliticaGarantiaHuellaSHA256,
			RolVersionRef:               snapshot.VersionRol.Referencia(), RolSHA256: rolSHA,
			ControlRevision: snapshot.ControlVigenciaVersionRol.Revision, ControlSHA256: controlSHA,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(autorizacionAdmisionPrueba{snapshot: snapshot},
		registroPrueba(ahora), denegacionesCapturaPrueba{}, motivoCapturaPrueba{}, base.reloj,
		decisionRefCapturaPrueba{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorCapturaRealEnMemoriaPrueba{t: t, servicio: servicio, reloj: base.reloj.(*relojPrueba)}
	dev, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		autorizacionAdmisionPrueba{snapshot: snapshot}, gate)
	if err != nil {
		t.Fatal(err)
	}
	material, err := dev.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
	if err != nil || material.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaFirmaVecV2 || emisor.llamadas != 1 {
		t.Fatalf("captura PDP real en memoria no llegó al adaptador: err=%v llamadas=%d", err, emisor.llamadas)
	}
	if _, err := gate.Admitir(ctx, vinculo, resultado, snapshot, vecapp.DescriptorAdmisionGarantiaActo{}); !errors.Is(err, vecapp.ErrAdmisionGarantiaActoDenegada) {
		t.Fatal("descriptor ajeno admitido")
	}
	emisor.ligarOtroMaterial = true
	material, err = dev.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || material.ResumenCapacidad().AudienciaConsumo() != "" || emisor.llamadas != 2 {
		t.Fatalf("captura real ligada a otro material admitida: err=%v llamadas=%d", err, emisor.llamadas)
	}
	emisor.ligarOtroMaterial = false
	gateCorto, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{
		EntornoConfiable: vecapp.EntornoAdmisionFirmaDesarrollo, Reloj: base.reloj,
		Politica: vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo{
			Referencia: "politica:ct:firma_vec_dev:v1", HuellaSHA256: strings.Repeat("a", 64), Version: 1,
			RetiradaEn: ahora.Add(2 * time.Second), PoliticaAutenticacionRef: datos.PoliticaGarantiaRef,
			PoliticaAutenticacionSHA256: datos.PoliticaGarantiaHuellaSHA256,
			RolVersionRef:               snapshot.VersionRol.Referencia(), RolSHA256: rolSHA,
			ControlRevision: snapshot.ControlVigenciaVersionRol.Revision, ControlSHA256: controlSHA,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	devCorto, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		autorizacionAdmisionPrueba{snapshot: snapshot}, gateCorto)
	if err != nil {
		t.Fatal(err)
	}
	material, err = devCorto.AutorizarMaterialFirmaVerificadaV2(ctx, m, r)
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || material.ResumenCapacidad().AudienciaConsumo() != "" || emisor.llamadas != 3 {
		t.Fatalf("decisión posterior a retirada admitida: err=%v llamadas=%d", err, emisor.llamadas)
	}
	ctxCancel, cancelar := context.WithCancel(ctx)
	defer cancelar()
	relojGate := &relojCancelaAdmisionPrueba{ahora: ahora, cancelarEn: 3, cancelar: cancelar}
	gateCancelado, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{
		EntornoConfiable: vecapp.EntornoAdmisionFirmaDesarrollo, Reloj: relojGate,
		Politica: vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo{
			Referencia: "politica:ct:firma_vec_dev:v1", HuellaSHA256: strings.Repeat("a", 64), Version: 1,
			RetiradaEn: ahora.Add(10 * time.Minute), PoliticaAutenticacionRef: datos.PoliticaGarantiaRef,
			PoliticaAutenticacionSHA256: datos.PoliticaGarantiaHuellaSHA256,
			RolVersionRef:               snapshot.VersionRol.Referencia(), RolSHA256: rolSHA,
			ControlRevision: snapshot.ControlVigenciaVersionRol.Revision, ControlSHA256: controlSHA,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	devCancelado, err := NuevoEmisorFirmaVecDesarrollo(fuente, emisor, base.motivo, base.reloj,
		autorizacionAdmisionPrueba{snapshot: snapshot}, gateCancelado)
	if err != nil {
		t.Fatal(err)
	}
	material, err = devCancelado.AutorizarMaterialFirmaVerificadaV2(ctxCancel, m, r)
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || !errors.Is(err, context.Canceled) ||
		material.ResumenCapacidad().AudienciaConsumo() != "" || emisor.llamadas != 4 {
		t.Fatalf("cancelación durante segunda admisión perdida: err=%v llamadas=%d", err, emisor.llamadas)
	}
}
