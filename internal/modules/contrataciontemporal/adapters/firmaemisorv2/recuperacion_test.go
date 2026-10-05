package firmaemisorv2

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El emisor sintético de firma existente fabrica la decisión; este adaptador
// de prueba sólo declara la audiencia propia de recuperación en su transporte.
type emisorRecuperacionPrueba struct{ base *emisorPrueba }

func (e emisorRecuperacionPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context,
	s vd.SolicitudAutorizacionLigadaV3, c vd.ResultadoContextoActorRegistradoV2,
) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, conf, exportador, err := e.base.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil || exportador == nil {
		return d, conf, exportador, err
	}
	x, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return d, conf, nil, err
	}
	viejo := x.ResumenCapacidad()
	nuevo, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(viejo.DecisionRef(),
		viejo.DecisionHuellaSHA256(), viejo.MotivoHuellaSHA256(), viejo.ContextoRef(), viejo.ContextoHuellaSHA256(),
		viejo.Operacion(), viejo.EfectoRef(), viejo.EfectoHuellaSHA256(), ports.AudienciaRecuperacionFirmasR5V2,
		viejo.EmitidaEn(), viejo.ExpiraEn())
	if err != nil {
		return d, conf, nil, err
	}
	material, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(x.CapacidadCanonica(), nuevo,
		x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), x.PersonaVersion(), x.PerfilVersion(),
		x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
	if err != nil {
		return d, conf, nil, err
	}
	return d, conf, &exportadorPrueba{material: material}, nil
}

func materialRecuperacionEmisorPrueba(m ports.MaterialFirmaVerificadaV2) ports.MaterialConsultaFirmasR5V2 {
	return ports.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: m.OrganizacionRef, ExpedienteRef: m.ExpedienteRef, VersionExpediente: m.VersionExpediente,
		Documento: m.Documento, FirmantePrincipalCandidatoRef: "per_candidato_ajeno",
		ClaveIdempotencia: m.ClaveIdempotencia, PasoOrden: m.PasoOrden, CatalogoHuella: m.CatalogoHuella},
		Via: m.Via}
}

func TestEmisorRecuperacion48UsaActorConfiableYCorrelacion(t *testing.T) {
	a, f, e, m, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	e.campos = ports.CamposRecuperacionFirmasV2()
	a.emisor = emisorRecuperacionPrueba{base: e}
	consulta := materialRecuperacionEmisorPrueba(m)
	capacidad, err := a.AutorizarRecuperacionFirmasV2(ctx, consulta)
	if err != nil || e.llamadas != 1 || f.llamadas != 1 {
		t.Fatalf("emisión 48 no ligada: %v", err)
	}
	campos, obligaciones := capacidad.RestriccionesParaConsumidor()
	if len(campos) != 48 || len(obligaciones) != 0 ||
		capacidad.ExportarMaterialParaConsumidor().ResumenCapacidad().AudienciaConsumo() != ports.AudienciaRecuperacionFirmasR5V2 {
		t.Fatal("la capacidad no conserva los 48 campos exactos")
	}
	d, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	actor, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if d.Accion != ports.AccionRecuperarFirmasR5V2 || d.Finalidad != ports.FinalidadFirmaDocumento ||
		actor.PrincipalID != f.base.Resultado.Contexto.Principal.ID ||
		actor.PrincipalID == consulta.FirmantePrincipalCandidatoRef ||
		d.Recurso.Referencia != consulta.ExpedienteRef ||
		d.Recurso.Atributos["material_sha256"] == "" {
		t.Fatal("actor o recurso reconstruidos desde el candidato")
	}
}

func TestEmisorRecuperacionRechaza44YCruceAntesDeEntregar(t *testing.T) {
	for _, caso := range []string{"campos44", "candidato", "correlacion"} {
		t.Run(caso, func(t *testing.T) {
			a, _, e, m, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
			e.campos = ports.CamposRecuperacionFirmasV2()
			a.emisor = emisorRecuperacionPrueba{base: e}
			consulta := materialRecuperacionEmisorPrueba(m)
			switch caso {
			case "campos44":
				e.campos = ports.CamposConsultaFirmasR5V2()
			case "candidato":
				consulta.FirmantePrincipalCandidatoRef = "persona:ajena"
			case "correlacion":
				ctx = context.Background()
			}
			capacidad, err := a.AutorizarRecuperacionFirmasV2(ctx, consulta)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) ||
				capacidad.ExportarMaterialParaConsumidor().ValidarEstructura() == nil ||
				(caso != "campos44" && e.llamadas != 0) {
				t.Fatalf("%s produjo capacidad: %v", caso, err)
			}
		})
	}
}

func TestEmisorRecuperacionNoAceptaCampoExtraDeProyeccion(t *testing.T) {
	a, _, e, m, _, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	e.campos = append(ports.CamposRecuperacionFirmasV2(), "PerfilActivoOperadorRef")
	a.emisor = emisorRecuperacionPrueba{base: e}
	consulta := materialRecuperacionEmisorPrueba(m)
	capacidad, err := a.AutorizarRecuperacionFirmasV2(ctx, consulta)
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) ||
		capacidad.ExportarMaterialParaConsumidor().ValidarEstructura() == nil ||
		strings.Contains(err.Error(), consulta.FirmantePrincipalCandidatoRef) {
		t.Fatal("campo extra expuso capacidad o candidato")
	}
}
