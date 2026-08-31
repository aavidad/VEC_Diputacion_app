package interna

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestFachadaIdentidadOfflineAutenticarYVincularNoExponeCanalEnFronteras(t *testing.T) {
	archivos := token.NewFileSet()
	archivo, err := parser.ParseFile(archivos, "identidad.go", nil, 0)
	if err != nil {
		t.Fatalf("analizar identidad.go: %v", err)
	}
	contieneCanal := func(nodo ast.Node) bool {
		encontrado := false
		ast.Inspect(nodo, func(actual ast.Node) bool {
			selector, ok := actual.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "CanalProxyAutenticado" {
				encontrado = true
				return false
			}
			return !encontrado
		})
		return encontrado
	}
	ast.Inspect(archivo, func(nodo ast.Node) bool {
		switch declaracion := nodo.(type) {
		case *ast.FuncType:
			if contieneCanal(declaracion) {
				t.Fatalf("firma o callback expone CanalProxyAutenticado en %s", archivos.Position(declaracion.Pos()))
			}
		case *ast.TypeSpec:
			if contieneCanal(declaracion.Type) {
				t.Fatalf("tipo o campo expone CanalProxyAutenticado en %s", archivos.Position(declaracion.Pos()))
			}
		}
		return true
	})
	for _, declaracion := range archivo.Decls {
		grupo, ok := declaracion.(*ast.GenDecl)
		if ok && grupo.Tok == token.VAR && contieneCanal(grupo) {
			t.Fatalf("variable de paquete expone CanalProxyAutenticado en %s", archivos.Position(grupo.Pos()))
		}
	}
}

func TestFachadaIdentidadOfflineAutenticarYVincularExitoYConservaContexto(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	type claveContextoPrueba struct{}
	valor := &struct{ marca string }{marca: "valor-sintetico"}
	limite := time.Now().Add(time.Minute)
	var vinculado context.Context
	var cancelar context.CancelFunc
	var err error

	codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
		ctx = context.WithValue(ctx, claveContextoPrueba{}, valor)
		ctx, cancelar = context.WithDeadline(ctx, limite)
		vinculado, err = entorno.fachada.AutenticarYVincular(
			ctx, []byte("asercion-corporativa-protegida"),
		)
	})
	if codigo != http.StatusNoContent || err != nil || vinculado == nil {
		t.Fatalf("autenticar y vincular: codigo=%d contexto_nulo=%t error=%v", codigo, vinculado == nil, err)
	}
	if vinculado.Value(claveContextoPrueba{}) != valor {
		t.Fatal("el contexto derivado no conserva los valores del contexto de peticion")
	}
	limiteObtenido, tieneLimite := vinculado.Deadline()
	if !tieneLimite || !limiteObtenido.Equal(limite) {
		t.Fatalf("deadline alterado: presente=%t", tieneLimite)
	}
	if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{1, 3, 1, 2}) {
		t.Fatalf("secuencia de autenticacion y vinculo incompleta: %+v", efectos)
	}
	cuenta, auditoria, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado)
	if err != nil || cuenta.Validar() != nil ||
		auditoria.CanalVinculadoRef() != entorno.canal.ReferenciaVinculacion() {
		t.Fatalf("extraccion por la instancia emisora rechazada: %v", err)
	}
	otro := nuevoEntornoIdentidadOfflinePrueba(t)
	cuentaCruzada, auditoriaCruzada, err := otro.servicio.ExtraerCapsulaIdentidadPeticion(vinculado)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) ||
		cuentaCruzada.Validar() == nil || auditoriaCruzada.CanalVinculadoRef() != "" {
		t.Fatalf("extraccion por servicio cruzado admitida: %v", err)
	}

	cancelar()
	<-vinculado.Done()
	if !errors.Is(vinculado.Err(), context.Canceled) {
		t.Fatalf("cancelacion del padre no conservada: %v", vinculado.Err())
	}
}

func TestFachadaIdentidadOfflineAutenticarYVincularFallaCerrado(t *testing.T) {
	t.Run("fachada nula", func(t *testing.T) {
		var fachada *FachadaIdentidadOffline
		ctx, err := fachada.AutenticarYVincular(context.Background(), []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, ErrIdentidadOfflineNoDisponible)
	})

	t.Run("contexto nulo", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		ctx, err := entorno.fachada.AutenticarYVincular(nil, []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, ErrIdentidadOfflineNoDisponible)
		var contextoNulo *contextoIdentidadOfflineNulo
		ctx, err = entorno.fachada.AutenticarYVincular(contextoNulo, []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, ErrIdentidadOfflineNoDisponible)
	})

	t.Run("canal ausente", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		ctx, err := entorno.fachada.AutenticarYVincular(context.Background(), []byte("x"))
		exigirContextoVinculadoNulo(t, ctx, err, httpseguridad.ErrCanalProxyNoAutenticado)
		if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{}) {
			t.Fatalf("el canal ausente produjo efectos: %+v", efectos)
		}
	})

	t.Run("asercion invalida", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var resultado context.Context
		var err error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			resultado, err = entorno.fachada.AutenticarYVincular(ctx, nil)
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrAsercionAusente)
	})

	t.Run("sesion revocada", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		entorno.registro.errorRevalidacion = errors.New("revocacion sintetica")
		var resultado context.Context
		var err error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			resultado, err = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-revocada"))
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrSesionNoValida)
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularRechazaContextoPrevioYCruces(t *testing.T) {
	t.Run("capsula previa de la misma instancia", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		capsula, err := entorno.autenticarEnC4(t, []byte("asercion-previa"))
		if err != nil {
			t.Fatalf("autenticar capsula previa: %v", err)
		}
		previo, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), capsula, entorno.canal,
		)
		if err != nil {
			t.Fatalf("vincular capsula previa: %v", err)
		}
		var resultado context.Context
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxConPrevio := context.WithValue(
				previo, claveContextoCanalTLSInterno{}, ctx.Value(claveContextoCanalTLSInterno{}),
			)
			resultado, err = entorno.fachada.AutenticarYVincular(
				ctxConPrevio, []byte("asercion-con-contexto-previo"),
			)
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrSesionNoValida)
	})

	t.Run("capsula previa de otra instancia", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		ajeno := nuevoEntornoIdentidadOfflinePrueba(t)
		capsulaAjena, err := ajeno.autenticarEnC4(t, []byte("asercion-ajena"))
		if err != nil {
			t.Fatalf("autenticar capsula ajena: %v", err)
		}
		ctxAjeno, err := ajeno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), capsulaAjena, ajeno.canal,
		)
		if err != nil {
			t.Fatalf("vincular capsula ajena: %v", err)
		}
		var resultado context.Context
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxCruzado := context.WithValue(
				ctxAjeno, claveContextoCanalTLSInterno{}, ctx.Value(claveContextoCanalTLSInterno{}),
			)
			resultado, err = entorno.fachada.AutenticarYVincular(
				ctxCruzado, []byte("asercion-con-capsula-cruzada"),
			)
		})
		exigirContextoVinculadoNulo(t, resultado, err, httpseguridad.ErrSesionNoValida)
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularConsumoUnico(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	const contendientes = 16
	resultados := make([]context.Context, contendientes)
	errores := make([]error, contendientes)
	var resultadoReplay context.Context
	var errorReplay error

	codigo := entorno.ejecutarEnC4(t, func(ctx context.Context) {
		inicio := make(chan struct{})
		var grupo sync.WaitGroup
		grupo.Add(contendientes)
		for indice := range contendientes {
			go func(indice int) {
				defer grupo.Done()
				<-inicio
				resultados[indice], errores[indice] = entorno.fachada.AutenticarYVincular(
					ctx, []byte("asercion-concurrente"),
				)
			}(indice)
		}
		close(inicio)
		grupo.Wait()
		resultadoReplay, errorReplay = entorno.fachada.AutenticarYVincular(
			ctx, []byte("asercion-replay"),
		)
	})
	if codigo != http.StatusNoContent {
		t.Fatalf("respuesta C4 inesperada: %d", codigo)
	}
	ganadores, rechazados := 0, 0
	var ganador context.Context
	for indice, err := range errores {
		switch {
		case err == nil && resultados[indice] != nil:
			ganadores++
			ganador = resultados[indice]
		case errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) && resultados[indice] == nil:
			rechazados++
		default:
			t.Fatalf("resultado concurrente inesperado %d: contexto_nulo=%t error=%v", indice, resultados[indice] == nil, err)
		}
	}
	if ganadores != 1 || rechazados != contendientes-1 {
		t.Fatalf("consumo concurrente: ganadores=%d rechazados=%d", ganadores, rechazados)
	}
	exigirContextoVinculadoNulo(
		t, resultadoReplay, errorReplay, httpseguridad.ErrCanalProxyNoAutenticado,
	)
	if efectos := entorno.efectos(); efectos != (efectosIdentidadOfflinePrueba{1, 3, 1, 2}) {
		t.Fatalf("los perdedores o el replay produjeron efectos: %+v", efectos)
	}
	if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(ganador); err != nil {
		t.Fatalf("el unico ganador no es extraible: %v", err)
	}
}

func TestFachadaIdentidadOfflineAutenticarYVincularCompiteConAutenticar(t *testing.T) {
	t.Run("Autenticar gana antes de AutenticarYVincular", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capsula httpseguridad.CapsulaIdentidadPeticion
		var vinculado context.Context
		var errorAutenticar, errorVincular error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			capsula, errorAutenticar = entorno.fachada.Autenticar(ctx, []byte("asercion-autenticar-primero"))
			vinculado, errorVincular = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-vincular-despues"))
		})
		if errorAutenticar != nil || vinculado != nil ||
			!errors.Is(errorVincular, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("resultado mixto inesperado: autenticar=%v contexto_nulo=%t vincular=%v",
				errorAutenticar, vinculado == nil, errorVincular)
		}
		ctxCapsula, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), capsula, entorno.canal,
		)
		if err != nil {
			t.Fatalf("la unica capsula ganadora no se pudo vincular: %v", err)
		}
		if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(ctxCapsula); err != nil {
			t.Fatalf("la unica capsula ganadora no se pudo extraer: %v", err)
		}
	})

	t.Run("AutenticarYVincular gana antes de Autenticar", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capsula httpseguridad.CapsulaIdentidadPeticion
		var vinculado context.Context
		var errorAutenticar, errorVincular error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			vinculado, errorVincular = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-vincular-primero"))
			capsula, errorAutenticar = entorno.fachada.Autenticar(ctx, []byte("asercion-autenticar-despues"))
		})
		if errorVincular != nil || vinculado == nil ||
			!errors.Is(errorAutenticar, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("resultado mixto inesperado: vincular=%v contexto_nulo=%t autenticar=%v",
				errorVincular, vinculado == nil, errorAutenticar)
		}
		if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
			t.Fatalf("el unico contexto ganador no se pudo extraer: %v", err)
		}
		exigirCapsulaIdentidadNoUtilizable(t, entorno, capsula)
	})

	t.Run("carrera mixta tiene un unico ganador", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var capsula httpseguridad.CapsulaIdentidadPeticion
		var vinculado context.Context
		var errorAutenticar, errorVincular error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			inicio := make(chan struct{})
			var grupo sync.WaitGroup
			grupo.Add(2)
			go func() {
				defer grupo.Done()
				<-inicio
				capsula, errorAutenticar = entorno.fachada.Autenticar(ctx, []byte("asercion-carrera-autenticar"))
			}()
			go func() {
				defer grupo.Done()
				<-inicio
				vinculado, errorVincular = entorno.fachada.AutenticarYVincular(ctx, []byte("asercion-carrera-vincular"))
			}()
			close(inicio)
			grupo.Wait()
		})

		ganadores := 0
		if errorAutenticar == nil {
			ganadores++
		}
		if errorVincular == nil {
			ganadores++
		}
		if ganadores != 1 {
			t.Fatalf("la carrera mixta no tuvo un unico ganador: autenticar=%v vincular=%v", errorAutenticar, errorVincular)
		}
		if errorAutenticar == nil {
			if vinculado != nil || !errors.Is(errorVincular, httpseguridad.ErrCanalProxyNoAutenticado) {
				t.Fatalf("Autenticar ganador dejo un segundo contexto: contexto_nulo=%t error=%v", vinculado == nil, errorVincular)
			}
			ctxCapsula, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
				context.Background(), capsula, entorno.canal,
			)
			if err != nil {
				t.Fatalf("capsula ganadora de carrera no vinculable: %v", err)
			}
			if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(ctxCapsula); err != nil {
				t.Fatalf("capsula ganadora de carrera no extraible: %v", err)
			}
			return
		}
		if vinculado == nil || !errors.Is(errorAutenticar, httpseguridad.ErrCanalProxyNoAutenticado) {
			t.Fatalf("AutenticarYVincular ganador dejo una segunda capsula: contexto_nulo=%t error=%v", vinculado == nil, errorAutenticar)
		}
		if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
			t.Fatalf("contexto ganador de carrera no extraible: %v", err)
		}
		exigirCapsulaIdentidadNoUtilizable(t, entorno, capsula)
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularCancelacionAntesYDurante(t *testing.T) {
	t.Run("antes no consume el canal", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		var cancelado, valido context.Context
		var errorCancelado, errorValido error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxCancelado, cancelar := context.WithCancel(ctx)
			cancelar()
			cancelado, errorCancelado = entorno.fachada.AutenticarYVincular(
				ctxCancelado, []byte("asercion-cancelada"),
			)
			valido, errorValido = entorno.fachada.AutenticarYVincular(
				ctx, []byte("asercion-posterior"),
			)
		})
		exigirContextoVinculadoNulo(t, cancelado, errorCancelado, context.Canceled)
		if errorValido != nil || valido == nil {
			t.Fatalf("la cancelacion previa consumio el canal: contexto_nulo=%t error=%v", valido == nil, errorValido)
		}
	})

	t.Run("durante la verificacion", func(t *testing.T) {
		entorno := nuevoEntornoIdentidadOfflinePrueba(t)
		verificador := &verificadorCancelacionVinculoPrueba{iniciado: make(chan struct{})}
		servicio := debeServicioIdentidadOfflinePrueba(
			t, entorno.configuracion, verificador, entorno.evaluador, entorno.registro, entorno.ahora,
		)
		canal, err := servicio.AutenticarCanalTLSMutuo(entorno.intercambio.estadoServidor)
		if err != nil {
			t.Fatalf("autenticar canal del servicio bloqueante: %v", err)
		}
		verificador.asercion = asercionIdentidadOfflinePrueba(entorno.ahora, entorno.configuracion, canal)
		fachada, err := NuevaFachadaIdentidadOffline(servicio, entorno.propietario)
		if err != nil {
			t.Fatalf("crear fachada bloqueante: %v", err)
		}
		var resultado context.Context
		var errorResultado error
		entorno.ejecutarEnC4(t, func(ctx context.Context) {
			ctxCancelado, cancelar := context.WithCancel(ctx)
			terminado := make(chan struct{})
			go func() {
				resultado, errorResultado = fachada.AutenticarYVincular(
					ctxCancelado, []byte("asercion-cancelada-durante-verificacion"),
				)
				close(terminado)
			}()
			<-verificador.iniciado
			cancelar()
			<-terminado
		})
		exigirContextoVinculadoNulo(t, resultado, errorResultado, context.Canceled)
		if verificador.llamadas.Load() != 1 || entorno.registro.altas.Load() != 0 ||
			entorno.registro.revalidaciones.Load() != 0 {
			t.Fatalf("cancelacion durante produjo efectos: verificaciones=%d altas=%d revalidaciones=%d",
				verificador.llamadas.Load(), entorno.registro.altas.Load(), entorno.registro.revalidaciones.Load())
		}
	})
}

func TestFachadaIdentidadOfflineAutenticarYVincularConservaAutenticar(t *testing.T) {
	entorno := nuevoEntornoIdentidadOfflinePrueba(t)
	capsula, err := entorno.autenticarEnC4(t, []byte("regresion-autenticar"))
	if err != nil {
		t.Fatalf("Autenticar cambio su resultado: %v", err)
	}
	vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("Autenticar dejo de emitir una capsula vinculable: %v", err)
	}
	if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
		t.Fatalf("Autenticar dejo de emitir una capsula extraible: %v", err)
	}
}

type verificadorCancelacionVinculoPrueba struct {
	asercion httpseguridad.AsercionProxyIdentidad
	iniciado chan struct{}
	unaVez   sync.Once
	llamadas atomic.Int32
}

func (v *verificadorCancelacionVinculoPrueba) Verificar(
	ctx context.Context,
	_ []byte,
) (httpseguridad.AsercionProxyIdentidad, error) {
	v.llamadas.Add(1)
	v.unaVez.Do(func() { close(v.iniciado) })
	<-ctx.Done()
	resultado := v.asercion
	resultado.Factores = append([]httpseguridad.FactorAutenticacion(nil), v.asercion.Factores...)
	return resultado, nil
}

func exigirContextoVinculadoNulo(
	t *testing.T,
	resultado context.Context,
	err error,
	esperado error,
) {
	t.Helper()
	if resultado != nil || !errors.Is(err, esperado) {
		t.Fatalf("fallo no cerrado: contexto_nulo=%t error=%v esperado=%v", resultado == nil, err, esperado)
	}
}

func exigirCapsulaIdentidadNoUtilizable(
	t *testing.T,
	entorno *entornoIdentidadOfflinePrueba,
	capsula httpseguridad.CapsulaIdentidadPeticion,
) {
	t.Helper()
	ctx, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if ctx != nil || !errors.Is(err, httpseguridad.ErrSesionNoValida) {
		t.Fatalf("el perdedor devolvio una capsula utilizable: contexto_nulo=%t error=%v", ctx == nil, err)
	}
}
