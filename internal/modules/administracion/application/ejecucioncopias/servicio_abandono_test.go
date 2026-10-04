package ejecucioncopias

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	ejadapter "vec-diputacion-granada/internal/modules/administracion/adapters/ejecucioncopias"
	cs07 "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	registroport "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// Proveedor controlado para la integración del servicio con los diarios reales.
// La composición de origen real debe aportar su propia observación vigente.
type observadorAbandonoPrueba struct {
	actor                     string
	efecto, lease, plataforma string
}

func (o *observadorAbandonoPrueba) ConfirmarAbandono(_ context.Context, d registroport.Declaracion, s registroport.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
	if d.Actor != o.actor || d.Correlacion != s.Operacion {
		return operacionescopias.ObservacionAbandono{}, registroport.ErrAbandonoNoAutorizado
	}
	return operacionescopias.ObservacionAbandono{Operacion: s.Operacion, Destino: s.Destino, FalloReferencia: s.FalloReferencia, FalloSHA256: s.FalloSHA256,
		Lease: "lease:prueba", EstadoEfecto: o.efecto, EstadoLease: o.lease, EstadoPlataforma: o.plataforma}, nil
}

func TestCapturaFallidaConDiariosRealesSeAbandonaSoloTrasObservacionVigente(t *testing.T) {
	for _, condicion := range []string{"sin_observador", "efecto_activo", "lease_pendiente", "plataforma_pendiente", "actor_revocado", "segura"} {
		t.Run(condicion, func(t *testing.T) {
			ctx := context.Background()
			p, lectura, _, _ := propuestaPrueba(t)
			base := t.TempDir()
			dirCS07, exterior, restaurada := filepath.Join(base, "cs07"), filepath.Join(base, "exterior"), filepath.Join(base, "restaurada")
			for _, dir := range []string{dirCS07, exterior, restaurada} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			observador := &observadorAbandonoPrueba{actor: p.ActorRef, efecto: "inactivo", lease: "cancelada", plataforma: "sin_efectos_pendientes"}
			switch condicion {
			case "efecto_activo":
				observador.efecto = "activo"
			case "lease_pendiente":
				observador.lease = "pendiente"
			case "plataforma_pendiente":
				observador.plataforma = "pendiente"
			case "actor_revocado":
				observador.actor = "actor:otro"
			}
			cfgCS07 := cs07.Config{Directorio: dirCS07, RaicesRestauradas: []string{restaurada}}
			var journal *cs07.Fichero
			var err error
			if condicion == "sin_observador" {
				journal, err = cs07.Abrir(cfgCS07)
			} else {
				journal, err = cs07.AbrirConObservadorAbandono(cfgCS07, observador)
			}
			if err != nil {
				t.Fatal(err)
			}
			var eventos []string
			destino := &destinoPrueba{conjuntos: map[string]puertos.Conjunto{}, eventos: &eventos}
			cfg := ejadapter.ConfigRegistroCS07{Registro: journal, Destino: destino, DirectorioExterior: exterior, RaicesRestauradas: []string{restaurada}}
			registro, err := ejadapter.AbrirRegistroCS07(cfg)
			if err != nil {
				t.Fatal(err)
			}
			s, err := NuevoCopia(Dependencias{Inventario: inventarioPrueba{lectura}, Autorizador: autorizadorPrueba{}, Registro: registro,
				Ventana: ventanaPrueba{eventos: &eventos}, Destino: destino, Ensayador: ensayadorPrueba{eventos: &eventos}})
			if err != nil {
				t.Fatal(err)
			}
			d := registroport.Declaracion{Actor: p.ActorRef, Correlacion: p.OperacionRef}
			otro := p.Peticion
			otro.OperacionRef, otro.ConjuntoRef = "operacion:otra-copia", "conjunto:otro"
			if _, err = s.Copiar(ctx, p.Peticion); err == nil {
				t.Fatal("un fallo de captura dio recibo de éxito")
			}
			if condicion != "segura" {
				if !errors.Is(err, ErrConciliacion) {
					t.Fatalf("faltó conciliación: %v", err)
				}
				op, err := registro.Leer(ctx, p.OperacionRef)
				if err != nil || op.Estado != "captura_pendiente_conciliacion" || op.FalloCapturaRef != "captura_fallida" {
					t.Fatalf("fallo no recuperable: %+v %v", op, err)
				}
				declarada, err := journal.Consultar(ctx, d, p.OperacionRef)
				if err != nil || declarada.Recibo.Estado != operacionescopias.Capturando || len(declarada.Historia) != 1 {
					t.Fatalf("falseó progreso CS07: %+v %v", declarada, err)
				}
				if _, err = registro.Reservar(ctx, otro); !errors.Is(err, registroport.ErrDestinoOcupado) {
					t.Fatalf("liberó destino incierto: %v", err)
				}
				if _, err = s.Copiar(ctx, p.Peticion); !errors.Is(err, ErrConciliacion) {
					t.Fatalf("reintento incierto: %v", err)
				}
				posterior, err := registro.Leer(ctx, p.OperacionRef)
				if err != nil || posterior != op {
					t.Fatalf("reintento añadió estado: %+v %v", posterior, err)
				}
				if err = registro.Close(); err != nil {
					t.Fatal(err)
				}
				// La comprobación posterior conserva la misma operación y reserva.
				observador.actor, observador.efecto, observador.lease, observador.plataforma = p.ActorRef, "inactivo", "cancelada", "sin_efectos_pendientes"
				journal, err = cs07.AbrirConObservadorAbandono(cfgCS07, observador)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Registro = journal
				registro, err = ejadapter.AbrirRegistroCS07(cfg)
				if err != nil {
					t.Fatal(err)
				}
				s.d.Registro = registro
				if _, err = s.Copiar(ctx, p.Peticion); err == nil || errors.Is(err, ErrConciliacion) {
					t.Fatalf("no terminó el abandono seguro sin declarar copia válida: %v", err)
				}
			}
			defer registro.Close()
			op, err := registro.Leer(ctx, p.OperacionRef)
			if err != nil || op.Estado != string(operacionescopias.AbandonadaDeclarada) {
				t.Fatalf("estado terminal: %+v %v", op, err)
			}
			declarada, err := journal.Consultar(ctx, d, p.OperacionRef)
			if err != nil || declarada.Recibo.Estado != operacionescopias.AbandonadaDeclarada || len(declarada.Historia) != 2 {
				t.Fatalf("progreso legítimo: %+v %v", declarada, err)
			}
			terminal := declarada.Historia[1].Comando
			if terminal.Accion != "abandonar_captura" || terminal.Abandono == nil || terminal.Evidencia != nil || terminal.ManifiestoSHA256 != "" || terminal.Ejecucion != "" {
				t.Fatal("abandono inventó ensayo o manifiesto")
			}
			if err = registro.AbandonarCaptura(ctx, p.OperacionRef, "captura_fallida"); err != nil {
				t.Fatal(err)
			}
			replay, err := registro.Leer(ctx, p.OperacionRef)
			if err != nil || replay != op {
				t.Fatalf("replay exterior duplicó abandono: %+v %v", replay, err)
			}
			if _, err = registro.Reservar(ctx, otro); err != nil {
				t.Fatalf("destino no liberado tras prueba positiva: %v", err)
			}
			capturas := 0
			for _, evento := range eventos {
				if evento == "capturar" {
					capturas++
				}
				if evento == "publicar_previa" || evento == "ensayar:fisico" || evento == "ensayar:logico" {
					t.Fatalf("efecto tras captura fallida: %s", evento)
				}
			}
			if capturas != 1 {
				t.Fatalf("repitió captura durante conciliación: %d", capturas)
			}
		})
	}
}
