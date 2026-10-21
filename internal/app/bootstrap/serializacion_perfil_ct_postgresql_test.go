package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSerializacionPerfilCTConservaAsignacionHastaRevalidarPostgreSQL18(t *testing.T) {
	for _, segundo := range []bool{false, true} {
		t.Run(fmt.Sprintf("segundo_rrhh_%t", segundo), func(t *testing.T) { comprobarSerializacionPerfilCTPostgreSQL(t, segundo) })
	}
}

func comprobarSerializacionPerfilCTPostgreSQL(t *testing.T, segundo bool) {
	t.Helper()
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if _, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	base := &autoridadConsultasContratacionTemporalDesarrollo{sello: s.sello, resolvedor: &resolvedorIdentidadDesarrollo{}}
	if segundo {
		base.resolvedor.segundoTecnicoRRHH = &identidadCertificadoDesarrollo{principal: principalDeSoportePrueba(s)}
	}
	publicada := make(chan struct{})
	continuar := make(chan struct{})
	segundaEntra := make(chan struct{})
	segundaPublicada := make(chan struct{})
	resultados := make(chan error, 2)
	crear := func(expediente string, primera bool) http.Handler {
		return fronteraSerializadaPrueba(base, principalDeSoportePrueba(s), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			preparada, err := s.autoridadAsignaciones.PrepararInstantanea(r.Context(), rolDeExpedientePrueba(s, expediente))
			if err == nil {
				err = s.autoridadAsignaciones.PublicarInstantanea(r.Context(), preparada)
			}
			if primera {
				close(publicada)
				<-continuar
			} else {
				close(segundaPublicada)
			}
			if err == nil {
				// La revalidación central exige la asignación exacta que usó la
				// decisión. Una publicación intermedia la hace obsoleta.
				var exacta bool
				exacta, _, err = instantaneaCentralCTExacta(r.Context(), gobierno, preparada)
				if err == nil && !exacta {
					err = fmt.Errorf("asignación sustituida antes de revalidar %s", expediente)
				}
			}
			resultados <- err
			w.WriteHeader(http.StatusNoContent)
		}))
	}
	a, b := crear("expediente:ct150b:a", true), crear("expediente:ct150b:b", false)
	go a.ServeHTTP(httptest.NewRecorder(), peticionSerializadaPrueba(ctx))
	<-publicada
	go func() { close(segundaEntra); b.ServeHTTP(httptest.NewRecorder(), peticionSerializadaPrueba(ctx)) }()
	<-segundaEntra
	// Esperar a que la segunda esté en cola; sin el arreglo llega a publicar
	// y sustituye la asignación de la primera, reproduciendo la carrera.
	limite := time.NewTimer(5 * time.Second)
	defer limite.Stop()
esperar:
	for {
		base.serializacionPerfiles.mu.Lock()
		turno := base.serializacionPerfiles.perfiles[perfil]
		enCola := turno != nil && turno.usuarios == 2
		base.serializacionPerfiles.mu.Unlock()
		if enCola {
			break
		}
		select {
		case <-segundaPublicada:
			break esperar
		case <-time.After(time.Millisecond):
		case <-limite.C:
			close(continuar)
			t.Fatal("la segunda petición no llegó a la frontera")
		}
	}
	close(continuar)
	for range 2 {
		if err := <-resultados; err != nil {
			t.Fatal(err)
		}
	}
	// Esperar también a los defer de ambas fronteras, después de revalidar.
	_, liberar, err := base.serializacionPerfiles.adquirir(ctx, perfil)
	if err != nil {
		t.Fatal(err)
	}
	liberar()
	despues := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if despues.versiones != antes.versiones+2 {
		t.Fatalf("historia perdida o duplicada: antes=%+v después=%+v", antes, despues)
	}
	// Una versión antigua no puede volver a publicarse para saltar el orden.
	antigua := s.instantanea
	if err := s.autoridadAsignaciones.PublicarInstantanea(context.Background(), antigua); err == nil {
		t.Fatal("se relajó la guarda de preimagen")
	}
}
