package ejecucioncopias

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cs03 "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	ejadapter "vec-diputacion-granada/internal/modules/administracion/adapters/ejecucioncopias"
	cs07 "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	cs03app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	registroport "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

type observadorPublicadoPrueba struct {
	observadorAbandonoPrueba
	verificador, ventana string
}

func (o *observadorPublicadoPrueba) ConfirmarAbandono(ctx context.Context, d registroport.Declaracion, s registroport.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
	a, err := o.observadorAbandonoPrueba.ConfirmarAbandono(ctx, d, s)
	a.EstadoVerificador, a.EstadoVentana = o.verificador, o.ventana
	return a, err
}

type fuentePublicacionPrueba map[string][]byte

func (f fuentePublicacionPrueba) Leer(_ context.Context, _ string, a copias.Artefacto) ([]byte, error) {
	return append([]byte(nil), f[a.ID]...), nil
}

type ensayadorFallidoPublicado struct {
	evidencia   copias.Evidencia
	modoFallido puertos.ModoEnsayo
	ejecutados  int
}

func (e *ensayadorFallidoPublicado) Ensayar(_ context.Context, _ puertos.Conjunto, modo puertos.ModoEnsayo) (puertos.Ensayo, error) {
	e.ejecutados++
	if modo == e.modoFallido {
		return puertos.Ensayo{}, errors.New("ensayo:fallo_controlado")
	}
	return puertos.Ensayo{Modo: modo, Evidencia: e.evidencia, VerificadorVersion: "verificador:prueba", ArranqueRef: e.evidencia.ArranqueRef}, nil
}

func TestFalloTrasPublicarConJWEYDiariosRealesConciliaSinReensayar(t *testing.T) {
	for _, modo := range []puertos.ModoEnsayo{puertos.Fisico, puertos.Logico} {
		for _, condicion := range []string{"sin_observador", "verificador_activo", "ventana_activa", "segura"} {
			t.Run(string(modo)+"/"+condicion, func(t *testing.T) {
				ctx := context.Background()
				p, lectura, _, captura := propuestaPrueba(t)
				captura.Manifiesto.ConjuntoRef = p.ConjuntoRef
				m := &captura.Manifiesto
				m.Proteccion = copias.Proteccion{Formato: "jwe-json", Algoritmo: "A256KW_A256GCM", ClaveRef: "clave:prueba", ClaveVersion: "v1", AutenticacionRef: "indice:" + m.ConjuntoRef, CifradoSHA256: strings.Repeat("0", 64)}
				material := fuentePublicacionPrueba{}
				m.TamanoBytes = 0
				for n := range m.Componentes {
					a := &m.Componentes[n]
					b := []byte{byte(n + 1)}
					a.SHA256, a.TamanoBytes = cs03app.Huella(b), int64(len(b))
					m.TamanoBytes += a.TamanoBytes
					material[a.ID] = b
				}
				ensayador := &ensayadorFallidoPublicado{evidencia: captura.Origen, modoFallido: modo}
				captura.Origen.ArranqueRef = ""
				base := t.TempDir()
				dirCS07, exterior, restaurada, almacen, catalogoDir := filepath.Join(base, "cs07"), filepath.Join(base, "exterior"), filepath.Join(base, "restaurada"), filepath.Join(base, "almacen"), filepath.Join(base, "catalogo")
				for _, dir := range []string{dirCS07, exterior, restaurada, almacen, catalogoDir} {
					if err := os.Mkdir(dir, 0700); err != nil {
						t.Fatal(err)
					}
				}
				var clave [32]byte
				for n := range clave {
					clave[n] = byte(n + 1)
				}
				protector, err := cs03.NuevoProtectorJWE(clave, "clave:prueba", "v1", 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				storage, err := cs03.NuevoFilesystem(cs03.ConfiguracionFilesystem{Raiz: almacen, MaximoClaroBytes: 1 << 20}, protector)
				if err != nil {
					t.Fatal(err)
				}
				defer storage.Close()
				catalogo, err := ejadapter.AbrirCatalogo(catalogoDir, []string{restaurada})
				if err != nil {
					t.Fatal(err)
				}
				defer catalogo.Close()
				destino := ejadapter.DestinoCS03{Destino: storage, Fuente: material, Catalogo: catalogo, ProteccionEsperada: m.Proteccion}
				cfgCS07 := cs07.Config{Directorio: dirCS07, RaicesRestauradas: []string{restaurada}}
				observador := &observadorPublicadoPrueba{observadorAbandonoPrueba: observadorAbandonoPrueba{actor: p.ActorRef, efecto: "inactivo", lease: "cancelada", plataforma: "sin_efectos_pendientes"}, verificador: "detenido", ventana: "inactiva"}
				if condicion == "verificador_activo" {
					observador.verificador = "activo"
				}
				if condicion == "ventana_activa" {
					observador.ventana = "activa"
				}
				var journal *cs07.Fichero
				if condicion == "sin_observador" {
					journal, err = cs07.Abrir(cfgCS07)
				} else {
					journal, err = cs07.AbrirConObservadorAbandono(cfgCS07, observador)
				}
				if err != nil {
					t.Fatal(err)
				}
				cfg := ejadapter.ConfigRegistroCS07{Registro: journal, Destino: destino, DirectorioExterior: exterior, RaicesRestauradas: []string{restaurada}}
				registro, err := ejadapter.AbrirRegistroCS07(cfg)
				if err != nil {
					t.Fatal(err)
				}
				var eventos []string
				s, err := NuevoCopia(Dependencias{Inventario: inventarioPrueba{lectura}, Autorizador: autorizadorPrueba{}, Registro: registro, Ventana: ventanaPrueba{captura: captura, eventos: &eventos}, Destino: destino, Ensayador: ensayador})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = s.Copiar(ctx, p.Peticion); err == nil {
					t.Fatal("declaró válida una verificación fallida")
				}
				indice, err := catalogo.Leer(ctx, p.ConjuntoRef)
				if err != nil {
					t.Fatal(err)
				}
				rutaIndice := filepath.Join(almacen, indice.ObjetoRef)
				cifrado, err := os.ReadFile(rutaIndice)
				if err != nil || bytes.Contains(cifrado, []byte(p.ActorRef)) {
					t.Fatalf("índice no cifrado: %v", err)
				}
				publicado, err := destino.Recuperar(ctx, p.ConjuntoRef)
				if err != nil || publicado.IndiceAutenticadoRef != indice.ObjetoRef || publicado.Origen != captura.Origen || publicado.Manifiesto.Verificacion.Estado != "pendiente_verificacion" {
					t.Fatalf("perdió conjunto publicado: %+v %v", publicado, err)
				}
				d := registroport.Declaracion{Actor: p.ActorRef, Correlacion: p.OperacionRef}
				otro := p.Peticion
				otro.OperacionRef, otro.ConjuntoRef = "operacion:despues-fallo", "conjunto:despues-fallo"
				if condicion != "segura" {
					op, err := registro.Leer(ctx, p.OperacionRef)
					if err != nil || op.Estado != "captura_pendiente_conciliacion" || op.FalloCapturaRef != "verificacion_fallida" {
						t.Fatalf("fallo no recuperable: %+v %v", op, err)
					}
					if _, err = registro.Reservar(ctx, otro); !errors.Is(err, registroport.ErrDestinoOcupado) {
						t.Fatalf("destino incierto liberado: %v", err)
					}
					if _, err = s.Copiar(ctx, p.Peticion); !errors.Is(err, ErrConciliacion) {
						t.Fatalf("reintento con verificador incierto: %v", err)
					}
					if err = registro.Close(); err != nil {
						t.Fatal(err)
					}
					observador.verificador, observador.ventana = "detenido", "inactiva"
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
					// Una observación positiva tampoco permite usar un índice alterado.
					if err = os.WriteFile(rutaIndice, []byte("alterado"), 0600); err != nil {
						t.Fatal(err)
					}
					if _, err = s.Copiar(ctx, p.Peticion); !errors.Is(err, ErrConciliacion) {
						t.Fatalf("aceptó índice alterado: %v", err)
					}
					if _, err = registro.Reservar(ctx, otro); !errors.Is(err, registroport.ErrDestinoOcupado) {
						t.Fatalf("índice alterado liberó destino: %v", err)
					}
					if err = os.WriteFile(rutaIndice, cifrado, 0600); err != nil {
						t.Fatal(err)
					}
					if _, err = s.Copiar(ctx, p.Peticion); err == nil || errors.Is(err, ErrConciliacion) {
						t.Fatalf("abandono seguro no confirmado: %v", err)
					}
				}
				defer registro.Close()
				declarada, err := journal.Consultar(ctx, d, p.OperacionRef)
				if err != nil || declarada.Recibo.Estado != operacionescopias.AbandonadaDeclarada || len(declarada.Historia) != 4 {
					t.Fatalf("historia de abandono: %+v %v", declarada, err)
				}
				final := declarada.Historia[3].Comando
				if final.Evidencia != nil || final.ManifiestoSHA256 != "" || final.Ejecucion != "" || final.Abandono == nil || final.Abandono.EstadoVerificador != "detenido" || final.Abandono.EstadoVentana != "inactiva" {
					t.Fatal("inventó evidencia de ensayo")
				}
				if _, err = registro.Reservar(ctx, otro); err != nil {
					t.Fatalf("abandono no liberó destino: %v", err)
				}
				actual, err := os.ReadFile(rutaIndice)
				if err != nil || !bytes.Equal(actual, cifrado) {
					t.Fatalf("reescribió índice publicado: %v", err)
				}
				esperados := 1
				if modo == puertos.Logico {
					esperados = 2
				}
				if ensayador.ejecutados != esperados || len(eventos) != 1 || eventos[0] != "capturar" {
					t.Fatalf("repitió captura/ensayo: %d %v", ensayador.ejecutados, eventos)
				}
			})
		}
	}
}
