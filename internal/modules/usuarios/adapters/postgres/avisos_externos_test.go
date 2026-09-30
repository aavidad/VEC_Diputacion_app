package postgres

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

const refAuditoriaInboxPG = "auditoria_tecnica_externa:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func eventoInboxPGPrueba() ports.EventoAvisoExterno {
	return ports.EventoAvisoExterno{EventoRef: "evento:1", ProductorRef: "productor:bolsa", TipoVersionado: ports.TipoAvisoLlamamientoExternoV1, OcurridoEn: "2026-09-30T12:13:14.123456Z", CorrelacionRef: "corr:1", DestinatarioExternoRef: "can_" + strings.Repeat("a", 24), ComunicacionRef: "llamamiento:" + strings.Repeat("b", 64), PlantillaRef: "plantilla:aviso", PlantillaVersion: "1"}
}
func reciboInboxPGPrueba() ports.ReciboAvisoExterno {
	h, _ := canonico.HuellaAvisoExterno(eventoInboxPGPrueba())
	return ports.ReciboAvisoExterno{ReciboRef: "aviso_recibo:" + strings.Repeat("a", 32), Huella: h, AceptadoEn: "2026-09-30T12:00:00.123456Z"}
}
func registroInboxPGPrueba(tx *txCorreoPGPrueba) *RegistroAvisosExternosPostgreSQL {
	return &RegistroAvisosExternosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return tx, nil }}
}
func TestInboxPGCommitObligatorioAntesACK(t *testing.T) {
	rec := reciboInboxPGPrueba()
	b, _ := json.Marshal(aceptacionInboxExternoSQL{Recibo: rec, AuditoriaRef: refAuditoriaInboxPG})
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
	r := registroInboxPGPrueba(tx)
	material, _ := canonico.MaterialAvisoExterno(eventoInboxPGPrueba())
	out, err := r.AceptarAvisoExterno(context.Background(), material)
	if err != nil || out != rec || tx.commits != 1 {
		t.Fatal(out, err, tx.commits)
	}
	tx = &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
	r = &RegistroAvisosExternosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return txInboxCommitFallido{tx}, nil }}
	out, err = r.AceptarAvisoExterno(context.Background(), material)
	if err == nil || out != (ports.ReciboAvisoExterno{}) {
		t.Fatal("ACK antes commit", out, err)
	}
}
func TestInboxPGRespuestaNoPuedeCambiarHuellaNiFiltrarDireccion(t *testing.T) {
	material, _ := canonico.MaterialAvisoExterno(eventoInboxPGPrueba())
	rec := reciboInboxPGPrueba()
	b, _ := json.Marshal(aceptacionInboxExternoSQL{Recibo: rec, AuditoriaRef: refAuditoriaInboxPG})
	for _, mala := range [][]byte{[]byte(strings.Replace(string(b), rec.Huella, strings.Repeat("b", 64), 1)), []byte(strings.Replace(string(b), `"replay":false`, `"replay":false,"direccion":"a@example.test"`, 1)), []byte(strings.Replace(string(b), rec.AceptadoEn, "2026-09-30T12:00:00Z", 1))} {
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: mala}}}
		if _, err := registroInboxPGPrueba(tx).AceptarAvisoExterno(context.Background(), material); err == nil || tx.commits != 0 {
			t.Fatal("confirmó respuesta inválida", err, tx.commits)
		}
	}
}
func TestInboxPGReplayNuncaEntregaSobreNiToken(t *testing.T) {
	rec := reciboInboxPGPrueba()
	buena := map[string]any{"recibo": rec, "estado": "reservado", "replay": true, "auditoria_ref": refAuditoriaInboxPG}
	for _, campo := range []string{"", "reserva_ref", "persona_ref", "sobre", "evento"} {
		m := map[string]any{}
		for k, v := range buena {
			m[k] = v
		}
		switch campo {
		case "reserva_ref":
			m[campo] = "reserva:" + strings.Repeat("b", 32)
		case "persona_ref":
			m[campo] = "per_" + strings.Repeat("p", 24)
		case "sobre":
			m[campo] = sobrePG()
		case "evento":
			m[campo] = eventoInboxPGPrueba()
		}
		b, _ := json.Marshal(m)
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
		out, err := registroInboxPGPrueba(tx).ReservarAvisoExterno(context.Background(), rec.ReciboRef)
		if campo == "" {
			if err != nil || !out.Replay || tx.commits != 1 {
				t.Fatal(out, err)
			}
		} else if err == nil || tx.commits != 0 {
			t.Fatalf("replay filtra %s", campo)
		}
	}
}
func TestInboxPGSesionInseguraOFalloNominalNoRevelaDetalles(t *testing.T) {
	material, _ := canonico.MaterialAvisoExterno(eventoInboxPGPrueba())
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: false}}}
	if _, err := registroInboxPGPrueba(tx).AceptarAvisoExterno(context.Background(), material); !errors.Is(err, ports.ErrAvisoExternoNoDisponible) || tx.commits != 0 {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		codigo string
		err    error
	}{{"23505", ports.ErrAvisoExternoConflicto}, {"22023", ports.ErrAvisoExternoInvalido}, {"42501", ports.ErrAvisoExternoNoDisponible}, {"40001", ports.ErrAvisoExternoNoDisponible}} {
		err := errorAvisoExternoSeguro(context.Background(), &pgconn.PgError{Code: caso.codigo, Detail: "secreto"})
		if !errors.Is(err, caso.err) || strings.Contains(err.Error(), "secreto") {
			t.Fatal(err)
		}
	}
}

type txInboxCommitFallido struct{ *txCorreoPGPrueba }

func (t txInboxCommitFallido) Commit(context.Context) error { return errors.New("commit perdido") }

func TestInboxPGAuditoriaObligatoriaEnExitoYReplay(t *testing.T) {
	material, _ := canonico.MaterialAvisoExterno(eventoInboxPGPrueba())
	for _, replay := range []bool{false, true} {
		rec := reciboInboxPGPrueba()
		rec.Replay = replay
		for _, audit := range []string{"", "aviso_auditoria:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "auditoria_tecnica_externa:invalida", refAuditoriaInboxPG} {
			b, _ := json.Marshal(aceptacionInboxExternoSQL{Recibo: rec, AuditoriaRef: audit})
			tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
			out, err := registroInboxPGPrueba(tx).AceptarAvisoExterno(context.Background(), material)
			if audit == refAuditoriaInboxPG {
				if err != nil || out != rec || tx.commits != 1 {
					t.Fatal(out, err)
				}
			} else if err == nil || tx.commits != 0 {
				t.Fatal("ACK sin auditoría real", audit, err)
			}
		}
	}
}
func TestInboxPGDenegacionAuditableConfirmaSoloRegistro(t *testing.T) {
	material, _ := canonico.MaterialAvisoExterno(eventoInboxPGPrueba())
	for _, caso := range []struct {
		codigo  string
		nominal error
	}{{"invalido", ports.ErrAvisoExternoInvalido}, {"conflicto", ports.ErrAvisoExternoConflicto}, {"no_disponible", ports.ErrAvisoExternoNoDisponible}} {
		b, _ := json.Marshal(denegacionInboxExternoSQL{Error: caso.codigo, AuditoriaRef: refAuditoriaInboxPG})
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
		out, err := registroInboxPGPrueba(tx).AceptarAvisoExterno(context.Background(), material)
		if !errors.Is(err, caso.nominal) || out != (ports.ReciboAvisoExterno{}) || tx.commits != 1 {
			t.Fatal("denegación no durable", out, err, tx.commits)
		}
		tx = &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
		r := &RegistroAvisosExternosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return txInboxCommitFallido{tx}, nil }}
		if _, err := r.AceptarAvisoExterno(context.Background(), material); !errors.Is(err, ports.ErrAvisoExternoNoDisponible) {
			t.Fatal("ocultó fallo commit audit", err)
		}
	}
}
func TestInboxPGDenegacionSinAuditoriaYConfirmacionAbiertaNoConfirman(t *testing.T) {
	for _, b := range [][]byte{[]byte(`{"error":"conflicto","auditoria_ref":""}`), []byte(`{"error":"conflicto","auditoria_ref":"` + refAuditoriaInboxPG + `","direccion":"a@example.test"}`), []byte(`{"confirmado":true}`), []byte(`true`)} {
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
		err := registroInboxPGPrueba(tx).ConfirmarAvisoExterno(context.Background(), reciboInboxPGPrueba().ReciboRef, "reserva:"+strings.Repeat("b", 32), "aceptado")
		if err == nil || tx.commits != 0 {
			t.Fatal("confirmó sin auditoría", err, tx.commits)
		}
	}
	b, _ := json.Marshal(map[string]any{"confirmado": true, "auditoria_ref": refAuditoriaInboxPG})
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
	if err := registroInboxPGPrueba(tx).ConfirmarAvisoExterno(context.Background(), reciboInboxPGPrueba().ReciboRef, "reserva:"+strings.Repeat("b", 32), "aceptado"); err != nil || tx.commits != 1 {
		t.Fatal(err)
	}
}

func TestInboxPGFalloAutoridadAuditoriaRevierteSinACKNiReserva(t *testing.T) {
	material, _ := canonico.MaterialAvisoExterno(eventoInboxPGPrueba())
	for _, operacion := range []string{"aceptar", "reservar", "confirmar"} {
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {err: &pgconn.PgError{Code: "XX000", Message: "autoridad auditoría indisponible"}}}}
		r := registroInboxPGPrueba(tx)
		var err error
		switch operacion {
		case "aceptar":
			_, err = r.AceptarAvisoExterno(context.Background(), material)
		case "reservar":
			_, err = r.ReservarAvisoExterno(context.Background(), reciboInboxPGPrueba().ReciboRef)
		case "confirmar":
			err = r.ConfirmarAvisoExterno(context.Background(), reciboInboxPGPrueba().ReciboRef, "reserva:"+strings.Repeat("b", 32), "aceptado")
		}
		if !errors.Is(err, ports.ErrAvisoExternoNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
			t.Fatal("fallo auditoría confirmado", operacion, err, tx.commits, tx.rollbacks)
		}
	}
}

func TestInboxPGExigeLoginWorkerExactoYTodoCanalTLSVerificado(t *testing.T) {
	base := func() *pgx.ConnConfig {
		return &pgx.ConnConfig{Config: pgconn.Config{User: loginInboxAvisosExternos, TLSConfig: &tls.Config{ServerName: "localhost", MinVersion: tls.VersionTLS12}}}
	}
	if !configuracionInboxExternoValida(base()) {
		t.Fatal("rechazó worker exacto protegido")
	}
	for _, usuario := range []string{"", "vec_area_personal", "vec_externo_usuarios", "vec_externo_avisos_usuarios_ajeno"} {
		cfg := base()
		cfg.User = usuario
		if configuracionInboxExternoValida(cfg) {
			t.Fatal("aceptó login compartido", usuario)
		}
	}
	casos := []func(*pgx.ConnConfig){
		func(c *pgx.ConnConfig) { c.TLSConfig = nil },
		func(c *pgx.ConnConfig) { c.TLSConfig.InsecureSkipVerify = true },
		func(c *pgx.ConnConfig) { c.TLSConfig.ServerName = "" },
		func(c *pgx.ConnConfig) { c.TLSConfig.MinVersion = tls.VersionTLS11 },
		func(c *pgx.ConnConfig) { c.Fallbacks = []*pgconn.FallbackConfig{{Host: "localhost", Port: 5432}} },
		func(c *pgx.ConnConfig) {
			c.Fallbacks = []*pgconn.FallbackConfig{{TLSConfig: &tls.Config{ServerName: "localhost", InsecureSkipVerify: true}}}
		},
	}
	for i, cambiar := range casos {
		cfg := base()
		cambiar(cfg)
		if configuracionInboxExternoValida(cfg) {
			t.Fatalf("canal inseguro %d admitido", i)
		}
	}
}

func TestInboxPGResultadoInciertoSeConfirmaYReplayNoEntregaReserva(t *testing.T) {
	ctx := context.Background()
	rec := reciboInboxPGPrueba()
	b, _ := json.Marshal(map[string]any{"confirmado": true, "auditoria_ref": refAuditoriaInboxPG})
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
	token := "reserva:" + strings.Repeat("b", 32)
	if err := registroInboxPGPrueba(tx).ConfirmarAvisoExterno(ctx, rec.ReciboRef, token, "reservado_incierto"); err != nil || tx.commits != 1 {
		t.Fatal("no confirmó incertidumbre", err, tx.commits)
	}
	rec.Replay = true
	b, _ = json.Marshal(map[string]any{"recibo": rec, "estado": "reservado_incierto", "replay": true, "auditoria_ref": refAuditoriaInboxPG})
	tx = &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: b}}}
	out, err := registroInboxPGPrueba(tx).ReservarAvisoExterno(ctx, rec.ReciboRef)
	if err != nil || !out.Replay || out.Estado != "reservado_incierto" || out.Recibo != rec || out.ReservaRef != "" || out.PersonaRef != "" || !reflect.DeepEqual(out.Sobre, ports.SobreDireccionCorreo{}) || out.Evento != (ports.EventoAvisoExterno{}) || tx.commits != 1 {
		t.Fatal("replay incierto abrió reserva", out, err, tx.commits)
	}
}
