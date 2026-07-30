package httpseguridad

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestCapsulaIdentidadPeticionRevalidaCanalYServicio(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("capsula"), canal))
	if err != nil {
		t.Fatal(err)
	}
	capsula, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
	cuenta, auditoria, errDatos := capsula.Datos(servicio, canal)
	if err != nil || errDatos != nil || cuenta.Validar() != nil || auditoria.CanalVinculadoRef() != canal.ReferenciaVinculacion() {
		t.Fatal("cápsula válida rechazada")
	}
	vinculado, err := VincularCapsulaIdentidadPeticion(context.Background(), capsula, servicio, canal)
	if err != nil {
		t.Fatal("vincular cápsula")
	}
	if _, _, err = ExtraerCapsulaIdentidadPeticion(vinculado, servicio, canal); err != nil {
		t.Fatal("extraer cápsula")
	}
	canal.evidenciaRef = "tls-exportador:sha256:cruzado"
	if _, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal); !errors.Is(err, ErrSesionNoValida) {
		t.Fatal("canal cruzado admitido")
	}
}

func TestCapsulaIdentidadPeticionCierraCancelacionYConcurrencia(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	c := configuracionInternaValida()
	v := &verificadorFalso{}
	s := debeServicio(t, c, v, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, s, c)
	v.fijarAsercion(asercionInternaValida(ahora, c, canal))
	identidad, err := s.Resolver(context.Background(), debeCredencial(t, []byte("concurrente"), canal))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = s.ProyectarCapsulaIdentidadPeticion(ctx, identidad, canal); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdida")
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
