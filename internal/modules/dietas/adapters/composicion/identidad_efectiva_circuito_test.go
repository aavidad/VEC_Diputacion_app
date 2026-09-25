package composicion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorCircuitoPrueba struct {
	llamadas  int
	solicitud vecdomain.SolicitudAutorizacionLigadaV3
}

func (e *emisorCircuitoPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, _ vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	e.solicitud = s
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("denegada")
}

type fuenteCompetenciaPrueba struct {
	unidad string
	err    error
	etapa  domain.EtapaCircuito
}

func (f *fuenteCompetenciaPrueba) EstadoCompetencias(context.Context, vecdomain.ResultadoContextoActorRegistradoV2) (dietasports.EstadoCompetenciasCircuito, error) {
	return dietasports.EstadoCompetenciasCircuito{Fuente: dietasports.FuenteCompetenciaAcreditada, Etapas: []domain.EtapaCircuito{domain.EtapaRevision}}, nil
}

func (f *fuenteCompetenciaPrueba) UnidadCompetente(_ context.Context, _ vecdomain.ResultadoContextoActorRegistradoV2, e domain.EtapaCircuito) (string, error) {
	f.etapa = e
	return f.unidad, f.err
}

func motivoCircuitoPrueba() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos-dietas", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "revisar-circuito"}
}

func decisionCircuitoPrueba() dietasports.SolicitudOperacionCircuito {
	return dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionDecidirCircuito, Decision: dietasports.SolicitudDecisionCircuito{Referencia: "dco_" + strings.Repeat("a", 22), Etapa: domain.EtapaRevision, Decision: domain.DecisionAprobar, ClaveIdempotencia: "clave_0123456789abcdef", VersionEsperada: 2}}
}

// Sin catálogo de validadores nadie queda acreditado: ni se emite V3 ni se
// deduce la unidad de la asignación D7 o del perfil.
func TestCircuitoSinCatalogoNoEmiteNiAcredita(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	emisor := &emisorCircuitoPrueba{}
	r, err := NuevoResolutorIdentidadEfectivaCircuito(&fuenteContextoR15{identidad: base}, FuenteCompetenciaCircuitoSinCatalogo{}, emisor, motivoCircuitoPrueba())
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []dietasports.SolicitudOperacionCircuito{
		decisionCircuitoPrueba(),
		{Operacion: dietasports.OperacionListarBandeja, Consulta: dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaLiquidacion, Limite: 20}},
		{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: dietasports.SolicitudDocumentoCircuito{Referencia: "dco_" + strings.Repeat("a", 22), Etapa: domain.EtapaFiscalizacion}},
	} {
		if _, err := r.ResolverIdentidadEfectivaCircuito(context.Background(), op); !errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente) {
			t.Fatalf("%s: %v", op.Operacion, err)
		}
	}
	estado, err := r.EstadoCompetenciasCircuito(context.Background())
	if err != nil || estado.Fuente != dietasports.FuenteCompetenciaSinFuente || len(estado.Etapas) != 0 {
		t.Fatalf("estado: %#v %v", estado, err)
	}
	if emisor.llamadas != 0 {
		t.Fatal("se emitió V3 sin fuente de competencia")
	}
}

// Con una fuente acreditada se consulta la etapa exacta; un vínculo caducado,
// un fallo de la autoridad común o de la fuente deniegan sin detalle.
func TestCircuitoUsaUnidadDeLaFuenteYDeniegaSiV3Falla(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	emisor := &emisorCircuitoPrueba{}
	fuente := &fuenteCompetenciaPrueba{unidad: "unidad:acreditada"}
	r, _ := NuevoResolutorIdentidadEfectivaCircuito(&fuenteContextoR15{identidad: base}, fuente, emisor, motivoCircuitoPrueba())
	if _, err := r.ResolverIdentidadEfectivaCircuito(context.Background(), decisionCircuitoPrueba()); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("error: %v", err)
	}
	if fuente.etapa != domain.EtapaRevision {
		t.Fatalf("etapa consultada: %s", fuente.etapa)
	}
	previas := emisor.llamadas
	fuente.err, fuente.unidad = errors.New("fuente caída"), ""
	if _, err := r.ResolverIdentidadEfectivaCircuito(context.Background(), decisionCircuitoPrueba()); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || emisor.llamadas != previas {
		t.Fatalf("fuente caída: %v %d", err, emisor.llamadas)
	}
}

func TestAudienciaPropiaPorAccionDelCircuito(t *testing.T) {
	vistas := map[string]bool{}
	for accion, audiencia := range audienciasCircuito {
		if vistas[audiencia] || !strings.HasPrefix(audiencia, "vec_dietas.") || !strings.HasSuffix(audiencia, ".v1") {
			t.Fatalf("audiencia de %s: %s", accion, audiencia)
		}
		vistas[audiencia] = true
	}
	if len(vistas) != 9 {
		t.Fatalf("audiencias: %d", len(vistas))
	}
}
