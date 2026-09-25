package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type lectorContratosCTPrueba struct {
	eventos []puertosct.EventoContratoBolsaPublicado
	desdes  []puertosct.CursorPublicacionContratosBolsa
	err     error
}

func (l *lectorContratosCTPrueba) LeerContratosBolsa(_ context.Context, desde puertosct.CursorPublicacionContratosBolsa, limite int) ([]puertosct.EventoContratoBolsaPublicado, error) {
	l.desdes = append(l.desdes, desde)
	if l.err != nil {
		return nil, l.err
	}
	var salida []puertosct.EventoContratoBolsaPublicado
	for _, e := range l.eventos {
		posterior := desde.Vacio() || e.OrigenCreadaEn.After(desde.CreadaEn) ||
			(e.OrigenCreadaEn.Equal(desde.CreadaEn) && e.OrigenRef > desde.OrigenRef)
		if posterior && len(salida) < limite {
			salida = append(salida, e)
		}
	}
	return salida, nil
}

// buzonContratosEntregaPrueba imita el inbox: idempotente por evento_ref.
type buzonContratosEntregaPrueba struct {
	huellas map[string]string
	cursor  puertosbolsa.CursorContratosParticipacion
	err     error
}

func (b *buzonContratosEntregaPrueba) CursorContratos(context.Context) (puertosbolsa.CursorContratosParticipacion, bool, error) {
	return b.cursor, !b.cursor.CreadaEn.IsZero(), b.err
}

func (b *buzonContratosEntregaPrueba) RegistrarContrato(_ context.Context, e puertosbolsa.EventoContratoRecibido) (puertosbolsa.ResultadoRegistroContrato, error) {
	if b.err != nil {
		return puertosbolsa.ResultadoRegistroContrato{}, b.err
	}
	if previa, ok := b.huellas[e.Evento.EventoRef]; ok {
		if previa != e.HuellaSHA256 {
			return puertosbolsa.ResultadoRegistroContrato{}, puertosbolsa.ErrEventoContratoDivergente
		}
		return puertosbolsa.ResultadoRegistroContrato{Reutilizado: true}, nil
	}
	b.huellas[e.Evento.EventoRef] = e.HuellaSHA256
	if e.OrigenCreadaEn.After(b.cursor.CreadaEn) {
		b.cursor = puertosbolsa.CursorContratosParticipacion{CreadaEn: e.OrigenCreadaEn, OrigenRef: e.Evento.OrigenRef}
	}
	return puertosbolsa.ResultadoRegistroContrato{ParticipacionRef: "participacion:1"}, nil
}

func eventoPublicadoPrueba(i int, base time.Time) puertosct.EventoContratoBolsaPublicado {
	origen := fmt.Sprintf("ref:outbox:%03d", i)
	ref := sha256.Sum256([]byte("incorporacion\x1f" + origen))
	eventoRef := "evento:ct:contrato-bolsa:" + hex.EncodeToString(ref[:])
	c := `{"tipo": "incorporacion", "inicio": null, "esquema": "vec.contratacion-temporal.contrato-bolsa.v1", "evento_ref": "` + eventoRef + `", "origen_ref": "` + origen + `", "causa_clave": null, "ocurrido_en": "2027-01-02T09:00:01.000000Z", "fin_previsto": null, "categoria_ref": null, "expediente_ref": "expediente:ct:1", "llamamiento_ref": "llamamiento:1", "modalidad_clave": null, "organizacion_ref": "organizacion:1"}`
	h := sha256.Sum256([]byte(c))
	return puertosct.EventoContratoBolsaPublicado{EventoRef: eventoRef, Contenido: []byte(c), HuellaSHA256: hex.EncodeToString(h[:]), OrigenRef: origen, OrigenCreadaEn: base.Add(time.Duration(i) * time.Second)}
}

func nuevaEntregaPrueba(t *testing.T, lector *lectorContratosCTPrueba, buzon *buzonContratosEntregaPrueba, lote int) *entregaContratosCTBolsa {
	t.Helper()
	receptor, err := aplicacionbolsa.NuevoServicioRecepcionContratos(buzon)
	if err != nil {
		t.Fatal(err)
	}
	return &entregaContratosCTBolsa{lector: lector, receptor: receptor, lote: lote, relectura: time.Minute}
}

func TestEntregaContratosCTPaginaYEsIdempotente(t *testing.T) {
	base := time.Date(2027, 1, 2, 9, 0, 0, 0, time.UTC)
	lector := &lectorContratosCTPrueba{}
	for i := 1; i <= 5; i++ {
		lector.eventos = append(lector.eventos, eventoPublicadoPrueba(i, base))
	}
	buzon := &buzonContratosEntregaPrueba{huellas: map[string]string{}}
	relevo := nuevaEntregaPrueba(t, lector, buzon, 2)
	r, err := relevo.entregar(context.Background())
	if err != nil || r.nuevos != 5 || r.reentregas != 0 || len(lector.desdes) != 3 || !lector.desdes[0].Vacio() {
		t.Fatalf("primera pasada r=%+v err=%v desdes=%v", r, err, lector.desdes)
	}
	// Segunda pasada: relee desde el cursor menos la ventana y no duplica.
	lector.desdes = nil
	r, err = relevo.entregar(context.Background())
	if err != nil || r.nuevos != 0 || r.reentregas != 5 || len(buzon.huellas) != 5 {
		t.Fatalf("segunda pasada r=%+v err=%v", r, err)
	}
	if want := base.Add(5*time.Second - time.Minute); !lector.desdes[0].CreadaEn.Equal(want) || lector.desdes[0].OrigenRef != "" {
		t.Fatalf("relectura desde %v, se esperaba %v", lector.desdes[0], want)
	}
}

func TestEntregaContratosCTRechazosNoBloqueanEIndisponibilidadDetiene(t *testing.T) {
	base := time.Date(2027, 1, 2, 9, 0, 0, 0, time.UTC)
	malo := eventoPublicadoPrueba(1, base)
	malo.HuellaSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	lector := &lectorContratosCTPrueba{eventos: []puertosct.EventoContratoBolsaPublicado{malo, eventoPublicadoPrueba(2, base)}}
	buzon := &buzonContratosEntregaPrueba{huellas: map[string]string{}}
	r, err := nuevaEntregaPrueba(t, lector, buzon, 10).entregar(context.Background())
	if err != nil || r.rechazados != 1 || r.nuevos != 1 {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	caido := errors.New("PostgreSQL CT caído")
	_, err = nuevaEntregaPrueba(t, &lectorContratosCTPrueba{err: caido}, &buzonContratosEntregaPrueba{huellas: map[string]string{}}, 10).entregar(context.Background())
	if !errors.Is(err, caido) {
		t.Fatalf("la indisponibilidad de CT debe detener la pasada: %v", err)
	}
	_, err = nuevaEntregaPrueba(t, &lectorContratosCTPrueba{eventos: lector.eventos[1:]}, &buzonContratosEntregaPrueba{huellas: map[string]string{}, err: puertosbolsa.ErrContratosParticipacionNoDisponible}, 10).entregar(context.Background())
	if !errors.Is(err, puertosbolsa.ErrContratosParticipacionNoDisponible) {
		t.Fatalf("la indisponibilidad de Bolsa debe detener la pasada: %v", err)
	}
}

func TestEntregaContratosCTSeDetieneYRespetaConfiguracion(t *testing.T) {
	buzon := &buzonContratosEntregaPrueba{huellas: map[string]string{}}
	relevo := nuevaEntregaPrueba(t, &lectorContratosCTPrueba{}, buzon, 10)
	esperas := make(chan time.Duration, 4)
	detener := mantenerEntregaContratosCTBolsa(relevo, 7*time.Second, func(ctx context.Context, d time.Duration) error {
		esperas <- d
		<-ctx.Done()
		return ctx.Err()
	})
	if d := <-esperas; d != 7*time.Second {
		t.Fatalf("intervalo configurado ignorado: %v", d)
	}
	detener()
	detener()
	parar, err := iniciarEntregaContratosCTBolsaDesarrollo(config.NuevaConfiguracionEntregaContratosCTBolsa("0", "", ""), nil, nil)
	if err != nil || parar == nil {
		t.Fatalf("desactivado: %v", err)
	}
	parar()
	if _, err := iniciarEntregaContratosCTBolsaDesarrollo(config.NuevaConfiguracionEntregaContratosCTBolsa("x", "", ""), nil, nil); err == nil {
		t.Fatal("configuración inválida aceptada")
	}
	if _, err := iniciarEntregaContratosCTBolsaDesarrollo(config.NuevaConfiguracionEntregaContratosCTBolsa("", "", ""), nil, nil); err == nil {
		t.Fatal("relevo sin pools aceptado")
	}
}
