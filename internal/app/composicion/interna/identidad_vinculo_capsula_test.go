package interna

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

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
