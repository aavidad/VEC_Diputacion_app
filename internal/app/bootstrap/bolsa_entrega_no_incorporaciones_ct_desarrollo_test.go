package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

type lectorNoIncorporacionesPrueba struct{ lectorContratosCTPrueba }

func (l *lectorNoIncorporacionesPrueba) LeerNoIncorporacionesBolsa(ctx context.Context, desde puertosct.CursorPublicacionContratosBolsa, limite int) ([]puertosct.EventoContratoBolsaPublicado, error) {
	return l.LeerContratosBolsa(ctx, desde, limite)
}

type buzonNoIncorporacionesEntregaPrueba struct {
	huellas      map[string]string
	cursor       puertosbolsa.CursorContratosParticipacion
	consecuencia []*puertosbolsa.ConsecuenciaNoIncorporacion
}

func (b *buzonNoIncorporacionesEntregaPrueba) CursorNoIncorporaciones(context.Context) (puertosbolsa.CursorContratosParticipacion, bool, error) {
	return b.cursor, b.cursor.OrigenRef != "", nil
}

func (b *buzonNoIncorporacionesEntregaPrueba) RegistrarNoIncorporacion(_ context.Context, e puertosbolsa.EventoNoIncorporacionRecibido) (puertosbolsa.ResultadoRegistroNoIncorporacion, error) {
	if previa, ok := b.huellas[e.Evento.EventoRef]; ok {
		if previa != e.HuellaSHA256 {
			return puertosbolsa.ResultadoRegistroNoIncorporacion{}, puertosbolsa.ErrEventoContratoDivergente
		}
		return puertosbolsa.ResultadoRegistroNoIncorporacion{Reutilizado: true, Estado: "aplicada"}, nil
	}
	b.huellas[e.Evento.EventoRef] = e.HuellaSHA256
	b.consecuencia = append(b.consecuencia, e.Consecuencia)
	b.cursor = puertosbolsa.CursorContratosParticipacion{Posicion: e.OrigenPosicion, OrigenRef: e.Evento.OrigenRef}
	return puertosbolsa.ResultadoRegistroNoIncorporacion{Estado: "aplicada", ParticipacionRef: "participacion:1"}, nil
}

type catalogoBajaPrueba struct{}

func (catalogoBajaPrueba) ResolverSancion(_ context.Context, clave string, _ time.Time) (puertosbolsa.ResolucionCatalogoSancion, error) {
	return puertosbolsa.ResolucionCatalogoSancion{Consecuencia: puertosbolsa.ConsecuenciaSancion{Clave: clave, Etiqueta: "Baja", Efecto: "excluir",
		ReglaRef: "vec.bolsa.reglas:3:" + clave, Huella: strings.Repeat("c", 64)},
		Recurso: puertosbolsa.PlazoSancion{UltimoDia: "2026-10-20", ReglaRef: "vec.bolsa.reglas:3:b24.consecuencias", Huella: strings.Repeat("d", 64)}}, nil
}

func noIncorporacionPublicadaPrueba(origen string, posicion int64) puertosct.EventoContratoBolsaPublicado {
	ref := sha256.Sum256([]byte("no_incorporacion\x1f" + origen))
	eventoRef := "evento:ct:no-incorporacion-bolsa:" + hex.EncodeToString(ref[:])
	c := `{"tipo": "no_incorporacion", "esquema": "vec.contratacion-temporal.no-incorporacion-bolsa.v1", "actor_ref": "per_actor", "evento_ref": "` + eventoRef +
		`", "origen_ref": "` + origen + `", "ocurrido_en": "2026-09-21T09:00:00.000000Z", "motivo_clave": "no_presentado", "resuelta_por": "per_segunda", "expediente_ref": "expediente:ct:1", "resolucion_ref": "resolucion:rrhh:1", "llamamiento_ref": "llamamiento:1", "organizacion_ref": "organizacion:1", "resolucion_sha256": "` +
		strings.Repeat("b", 64) + `", "consecuencia_clave": "b24.sancion.baja_llamamiento_directo", "fecha_notificacion": "2026-09-20"}`
	h := sha256.Sum256([]byte(c))
	return puertosct.EventoContratoBolsaPublicado{EventoRef: eventoRef, Contenido: []byte(c), HuellaSHA256: hex.EncodeToString(h[:]), OrigenRef: origen,
		OrigenPosicion: posicion, OrigenCreadaEn: time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC)}
}

func TestEntregaNoIncorporacionesCTBolsaIdempotenteYConLaConsecuenciaDelCatalogo(t *testing.T) {
	lector := &lectorNoIncorporacionesPrueba{}
	lector.eventos = []puertosct.EventoContratoBolsaPublicado{noIncorporacionPublicadaPrueba("evento:ct124:1", 10), noIncorporacionPublicadaPrueba("evento:ct124:2", 11)}
	roto := noIncorporacionPublicadaPrueba("evento:ct124:3", 12)
	roto.HuellaSHA256 = strings.Repeat("0", 64)
	lector.eventos = append(lector.eventos, roto)
	buzon := &buzonNoIncorporacionesEntregaPrueba{huellas: map[string]string{}}
	receptor, err := aplicacionbolsa.NuevoServicioRecepcionNoIncorporaciones(buzon, catalogoBajaPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	relevo := &entregaNoIncorporacionesCTBolsa{lector: lector, receptor: receptor, lote: 2}
	r, err := relevo.entregar(context.Background())
	if err != nil || r.nuevos != 2 || r.rechazados != 1 || len(buzon.consecuencia) != 2 || buzon.consecuencia[0] == nil ||
		buzon.consecuencia[0].Efecto != "excluir" || buzon.consecuencia[0].Clave != "b24.sancion.baja_llamamiento_directo" {
		t.Fatalf("primera pasada: %+v %v", r, err)
	}
	// El cursor de la bandeja evita releer lo ya entregado.
	buzon.cursor = puertosbolsa.CursorContratosParticipacion{}
	r, err = relevo.entregar(context.Background())
	if err != nil || r.nuevos != 0 || r.reentregas != 2 {
		t.Fatalf("reentrega: %+v %v", r, err)
	}
	if _, err := iniciarEntregaNoIncorporacionesCTBolsaDesarrollo(context.Background(), config.NuevaConfiguracionEntregaContratosCTBolsa("", ""), nil, nil, nil); err == nil {
		t.Fatal("sin catálogo de Bolsa el relevo arranca")
	}
	if _, err := iniciarEntregaNoIncorporacionesCTBolsaDesarrollo(context.Background(), config.NuevaConfiguracionEntregaContratosCTBolsa("", ""), nil, nil, catalogoBajaPrueba{}); err == nil {
		t.Fatal("sin Bolsa 000042 el relevo arranca")
	}
}

// Los motivos, la consecuencia en Bolsa y la segunda persona salen de c22.
func TestNoIncorporacionLeeLaReglaC13DelCatalogo(t *testing.T) {
	f := fuenteReglasSeguimientoDesarrollo{reglas: resolutorReglasCTEjemploPrueba(t)}
	ahora := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	regla, politica, err := f.ReglaNoIncorporacion(context.Background(), ahora)
	if err != nil || !regla.Valida() || !regla.SegundaPersona || len(regla.Motivos) != 3 || !politica.ValidaEn(ahora) {
		t.Fatalf("regla c22: %+v %v", regla, err)
	}
	m, ok := regla.Motivo("no_presentado")
	if !ok || m.ConsecuenciaClave != "b24.sancion.baja_llamamiento_directo" || m.Etiqueta == "" {
		t.Fatalf("motivo: %+v", m)
	}
	if reglas.CTNoIncorporacion != "c22.no_incorporacion" {
		t.Fatal("clave de la regla")
	}
}

func TestSoloRecuperacionTrasNoIncorporacion(t *testing.T) {
	a := puertosct.AntecedenteContinuacionLlamamiento{}
	if soloRecuperacionContinuacionDesarrollo(a, 6) || !soloRecuperacionContinuacionDesarrollo(a, 7) {
		t.Fatal("renuncia o expiración: recuperación desde la versión 7")
	}
	a.Resolucion.Solicitud.Respuesta = puertosct.RespuestaLlamamientoAceptada
	a.NoIncorporacion = &puertosct.AntecedenteNoIncorporacion{VersionResultante: 8}
	if soloRecuperacionContinuacionDesarrollo(a, 8) || !soloRecuperacionContinuacionDesarrollo(a, 9) {
		t.Fatal("no incorporación: recuperación por encima de su versión")
	}
}
