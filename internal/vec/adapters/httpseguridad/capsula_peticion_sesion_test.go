package httpseguridad

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type entornoCapsulaPeticionSesion struct {
	servicio *ServicioPeticionSesion
	canal    CanalProxyAutenticado
	cuerpo   []byte
	reloj    *relojFijo
	registro *registroPeticionFalso
}

func nuevoEntornoCapsulaPeticionSesion(t *testing.T) entornoCapsulaPeticionSesion {
	t.Helper()
	ahora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	autoridad, err := NuevoServicioIdentidad(configuracion, &verificadorFalso{}, &evaluadorFalso{}, nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := []byte("{}")
	suma := sha256.Sum256(cuerpo)
	canalRef := "tls-exportador:sha256:capsula-peticion"
	peticion := AsercionPeticionVerificada{
		Emisor: configuracion.EmisorIdentidad, Audiencia: configuracion.Audiencia,
		Superficie: configuracion.Superficie, SesionID: "sesion-capsula", Metodo: "POST",
		Destino: "/api/vec/personal", CuerpoSHA256: hex.EncodeToString(suma[:]),
		NonceSHA256: hex.EncodeToString(sha256Sum([]byte("nonce-capsula"))), EmitidaEn: ahora,
		ExpiraEn: ahora.Add(time.Minute), CanalVinculadoRef: canalRef,
	}
	reloj := &relojFijo{ahora: ahora}
	registro := &registroPeticionFalso{confirmacion: confirmacionPeticionPrueba(ahora.Add(2 * time.Minute))}
	servicio, err := NuevoServicioPeticionSesion(autoridad, &verificadorPeticionFalso{peticion: peticion}, registro, seudonimizadorPeticionFalso{}, reloj)
	if err != nil {
		t.Fatal(err)
	}
	return entornoCapsulaPeticionSesion{
		servicio: servicio, canal: canalPeticionPrueba(autoridad, canalRef), cuerpo: cuerpo,
		reloj: reloj, registro: registro,
	}
}

func (e entornoCapsulaPeticionSesion) vincular(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	vinculado, err := e.servicio.ResolverYVincular(ctx, []byte("firmada"), e.canal, "POST", "/api/vec/personal", e.cuerpo)
	if err != nil {
		t.Fatalf("resolver y vincular: %v", err)
	}
	return vinculado
}

func TestCapsulaPeticionSesionConsumeRegistroUnaVezYEntregaCopia(t *testing.T) {
	entorno := nuevoEntornoCapsulaPeticionSesion(t)
	ctx := entorno.vincular(t, context.Background())
	if entorno.registro.llamadas != 1 {
		t.Fatalf("consumos de registro = %d", entorno.registro.llamadas)
	}
	confirmacion, err := entorno.servicio.ExtraerConfirmacionVinculada(ctx)
	if err != nil || confirmacion != entorno.registro.confirmacion {
		t.Fatalf("confirmacion vinculada: %#v, %v", confirmacion, err)
	}
	confirmacion.CuentaRef = "alterada"
	if _, err := entorno.servicio.ExtraerConfirmacionVinculada(ctx); !errors.Is(err, ErrAsercionPeticionNoValida) {
		t.Fatalf("doble extraccion admitida: %v", err)
	}
}

func TestCapsulaPeticionSesionCierraCrucesCancelacionCaducidadYCarrera(t *testing.T) {
	t.Run("cruce de servicio y contexto ya vinculado", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaPeticionSesion(t)
		ctx := entorno.vincular(t, context.Background())
		otro := nuevoEntornoCapsulaPeticionSesion(t)
		if _, err := otro.servicio.ExtraerConfirmacionVinculada(ctx); !errors.Is(err, ErrAsercionPeticionNoValida) {
			t.Fatalf("servicio cruzado admitido: %v", err)
		}
		if _, err := entorno.servicio.ResolverYVincular(ctx, []byte("firmada"), entorno.canal, "POST", "/api/vec/personal", entorno.cuerpo); !errors.Is(err, ErrAsercionPeticionNoValida) {
			t.Fatalf("contexto ya vinculado admitido: %v", err)
		}
		canalCruzado := entorno.canal
		canalCruzado.evidenciaRef = "tls-exportador:sha256:canal-cruzado"
		if _, err := entorno.servicio.ResolverYVincular(context.Background(), []byte("firmada"), canalCruzado, "POST", "/api/vec/personal", entorno.cuerpo); !errors.Is(err, ErrAsercionPeticionNoValida) {
			t.Fatalf("canal cruzado admitido: %v", err)
		}
		if entorno.registro.llamadas != 1 {
			t.Fatalf("el canal cruzado alcanzo el registro: %d", entorno.registro.llamadas)
		}
	})
	t.Run("cancelacion y caducidad", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaPeticionSesion(t)
		cancelado, cancelar := context.WithCancel(context.Background())
		cancelar()
		if _, err := entorno.servicio.ResolverYVincular(cancelado, []byte("firmada"), entorno.canal, "POST", "/api/vec/personal", entorno.cuerpo); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion de vinculacion: %v", err)
		}
		ctxVivo, cancelarVivo := context.WithCancel(context.Background())
		vinculado := entorno.vincular(t, ctxVivo)
		cancelarVivo()
		if _, err := entorno.servicio.ExtraerConfirmacionVinculada(vinculado); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion de extraccion: %v", err)
		}
		entorno = nuevoEntornoCapsulaPeticionSesion(t)
		ctx := entorno.vincular(t, context.Background())
		entorno.reloj.fijar(entorno.registro.confirmacion.SesionValidaHasta)
		if _, err := entorno.servicio.ExtraerConfirmacionVinculada(ctx); !errors.Is(err, ErrAsercionPeticionNoValida) {
			t.Fatalf("confirmacion caducada admitida: %v", err)
		}
	})
	t.Run("una sola extraccion concurrente", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaPeticionSesion(t)
		ctx := entorno.vincular(t, context.Background())
		var exitos atomic.Int64
		var grupo sync.WaitGroup
		for range 16 {
			grupo.Add(1)
			go func() {
				defer grupo.Done()
				if _, err := entorno.servicio.ExtraerConfirmacionVinculada(ctx); err == nil {
					exitos.Add(1)
				}
			}()
		}
		grupo.Wait()
		if exitos.Load() != 1 {
			t.Fatalf("extracciones concurrentes exitosas = %d", exitos.Load())
		}
	})
}

func TestCapsulaPeticionSesionNoFiltraEnRepresentacionNiLog(t *testing.T) {
	entorno := nuevoEntornoCapsulaPeticionSesion(t)
	ctx := entorno.vincular(t, context.Background())
	capsula, ok := ctx.Value(claveCapsulaPeticionSesion{}).(*capsulaPeticionSesionVinculada)
	if !ok {
		t.Fatal("capsula privada ausente")
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("capsula", "valor", capsula)
	representacion := fmt.Sprintf("%s %v %#v %+v %s", capsula, capsula, capsula, capsula, registro.String())
	for _, secreto := range []string{
		entorno.registro.confirmacion.SesionRef, entorno.registro.confirmacion.AutenticacionRef,
		entorno.registro.confirmacion.AsercionRef, entorno.registro.confirmacion.CuentaRef,
		entorno.canal.ReferenciaVinculacion(),
	} {
		if bytes.Contains([]byte(representacion), []byte(secreto)) {
			t.Fatalf("capsula filtrada: %q", secreto)
		}
	}
}
