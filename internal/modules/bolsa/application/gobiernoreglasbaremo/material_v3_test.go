package gobiernoreglasbaremo

import (
	"context"
	"errors"
	"strings"
	"testing"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestGobiernoV3ConsultaCruceDeAmbitoYRecuperacionNoEsAlta(t *testing.T) {
	s, b, r, c, p, _ := servicioGobiernoPrueba(t)
	ctx := context.Background()
	alta, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	id := p.Conjunto.Identidad()
	otra, err := reglas.NuevaIdentidadConjuntoReglasBaremo(id.Referencia(), id.Version(), id.ConvocatoriaRef(), "exp_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	debeSinError(t, err)
	motivo, err := motivoGobiernoV3(p.Motivo)
	debeSinError(t, err)
	selector := ports.SelectorGobiernoReglasV3{Identidad: otra, Estado: alta.Recibo.Estado}
	if _, err := s.ConsultarExacta(ctx, c, PeticionConsultaExactaV3{Selector: selector, Motivo: motivo}); !errors.Is(err, ports.ErrConfirmacionReglasBaremoInvalida) {
		t.Fatalf("fila de otro ámbito aceptada: %v", err)
	}
	selector.Identidad = id
	recuperacion := PeticionRecuperarReciboV3{Selector: selector, Motivo: motivo, ClaveOperacion: strings.Repeat("b", 32), HuellaSolicitudSHA256: alta.Recibo.HuellaSolicitudSHA256}
	ausente, err := s.RecuperarRecibo(ctx, c, recuperacion)
	debeSinError(t, err)
	if ausente.Existe || r.guardados != 1 || r.recuperaciones != 1 || ausente.Acceso.DecisionRef == "" {
		t.Fatal("ausencia sin auditoría o con alta")
	}
	recuperacion.ClaveOperacion = p.ClaveOperacion
	recuperacion.HuellaSolicitudSHA256 = strings.Repeat("f", 64)
	if _, err := s.RecuperarRecibo(ctx, c, recuperacion); !errors.Is(err, ports.ErrConfirmacionReglasBaremoInvalida) {
		t.Fatalf("intención de recuperación divergente: %v", err)
	}
	b.denegar = true
	consultas, recuperaciones := r.consultas, r.recuperaciones
	if _, err := s.ConsultarExacta(ctx, c, PeticionConsultaExactaV3{Selector: selector, Motivo: motivo}); !errors.Is(err, ErrGobiernoV3Prohibido) {
		t.Fatal("lectura sin concesión")
	}
	if _, err := s.RecuperarRecibo(ctx, c, recuperacion); !errors.Is(err, ErrGobiernoV3Prohibido) {
		t.Fatal("recuperación con autorización histórica")
	}
	if r.consultas != consultas || r.recuperaciones != recuperaciones || r.guardados != 1 {
		t.Fatal("denegación llegó al repositorio")
	}
}
