package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type buzonContratosPrueba struct {
	recibidos []ports.EventoContratoRecibido
	err       error
}

func (b *buzonContratosPrueba) CursorContratos(context.Context) (ports.CursorContratosParticipacion, bool, error) {
	return ports.CursorContratosParticipacion{}, false, b.err
}

func (b *buzonContratosPrueba) RegistrarContrato(_ context.Context, e ports.EventoContratoRecibido) (ports.ResultadoRegistroContrato, error) {
	b.recibidos = append(b.recibidos, e)
	return ports.ResultadoRegistroContrato{ParticipacionRef: "participacion:1"}, b.err
}

func eventoContratoAplicacionPrueba() (string, string) {
	origen := "ref:outbox:1"
	ref := sha256.Sum256([]byte("incorporacion\x1f" + origen))
	c := `{"tipo": "incorporacion", "inicio": "2027-01-04T00:00:00.000000Z", "esquema": "vec.contratacion-temporal.contrato-bolsa.v1", "evento_ref": "evento:ct:contrato-bolsa:` + hex.EncodeToString(ref[:]) + `", "origen_ref": "` + origen + `", "causa_clave": null, "ocurrido_en": "2027-01-02T09:00:01.000000Z", "fin_previsto": null, "categoria_ref": null, "expediente_ref": "expediente:ct:1", "llamamiento_ref": "llamamiento:1", "modalidad_clave": null, "organizacion_ref": "organizacion:1"}`
	h := sha256.Sum256([]byte(c))
	return c, hex.EncodeToString(h[:])
}

func TestRecepcionContratosValidaAntesDelInbox(t *testing.T) {
	buzon := &buzonContratosPrueba{}
	s, err := NuevoServicioRecepcionContratos(buzon)
	if err != nil {
		t.Fatal(err)
	}
	c, h := eventoContratoAplicacionPrueba()
	contenido := []byte(c)
	res, err := s.Recibir(context.Background(), contenido, h, time.Date(2027, 1, 2, 9, 0, 1, 0, time.UTC), 7)
	if err != nil || res.ParticipacionRef != "participacion:1" || len(buzon.recibidos) != 1 || buzon.recibidos[0].OrigenPosicion != 7 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	contenido[0] = 'X'
	if buzon.recibidos[0].Contenido[0] != '{' {
		t.Fatal("el inbox debe recibir una copia defensiva del contenido")
	}
	if _, err := s.Recibir(context.Background(), []byte(c), h[:63]+"0", time.Now(), 7); !errors.Is(err, dominiobolsa.ErrEventoContratoParticipacionInvalido) {
		t.Fatalf("huella falsa: %v", err)
	}
	if _, err := s.Recibir(context.Background(), []byte(c), h, time.Time{}, 7); err == nil {
		t.Fatal("origen sin instante aceptado")
	}
	if _, err := s.Recibir(context.Background(), []byte(c), h, time.Now(), -1); err == nil {
		t.Fatal("origen con posición negativa aceptado")
	}
	if len(buzon.recibidos) != 1 {
		t.Fatal("un evento inválido no debe llegar al inbox")
	}
	if _, err := NuevoServicioRecepcionContratos(nil); err == nil {
		t.Fatal("servicio sin inbox")
	}
}

func TestListarContratosSinRepositorioB13NoDisponible(t *testing.T) {
	var s *ServicioSituacionParticipacion
	if _, err := s.ListarContratos(context.Background(), ports.SolicitudCambiarSituacionParticipacion{}); err == nil {
		t.Fatal("servicio nulo")
	}
}
