package memory

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instanteMemoriaPrueba = time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)

const (
	principalBaremacionMemoriaPrueba = "per_0123456789abcdefghijkl"
	perfilBaremacionMemoriaPrueba    = "prf_0123456789abcdefghijkl"
	principalAjenoMemoriaPrueba      = "per_otra_persona_abcdefghijkl"
)

var claveHMACMemoriaPrueba = []byte("clave-de-prueba-no-productiva-32b")

type verificadorHMACMemoriaPrueba struct{}

func (verificadorHMACMemoriaPrueba) VerificarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	esperado := calcularSelloMemoria(solicitud.Finalidad, solicitud.RepresentacionCanonica.Revelar())
	if !hmac.Equal([]byte(esperado), []byte(solicitud.SelloHMAC)) {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

type relojMemoriaPrueba struct {
	mu       sync.RWMutex
	instante time.Time
}

type relojAvisadoMemoriaPrueba struct {
	base   *relojMemoriaPrueba
	avisos chan struct{}
}

func (r *relojAvisadoMemoriaPrueba) Ahora() time.Time {
	instante := r.base.Ahora()
	select {
	case r.avisos <- struct{}{}:
	default:
	}
	return instante
}

func (r *relojMemoriaPrueba) Ahora() time.Time {
	if r == nil {
		return time.Time{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.instante
}

func (r *relojMemoriaPrueba) fijar(instante time.Time) {
	r.mu.Lock()
	r.instante = instante
	r.mu.Unlock()
}

func TestRepositorioBaremacionesRechazaRelojAusenteOCero(t *testing.T) {
	perfil := PerfilRepositorioBaremacionesSoloPruebas()
	verificador := verificadorHMACMemoriaPrueba{}
	if _, err := NuevoRepositorioBaremaciones(nil, verificador, perfil); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("reloj nil admitido: %v", err)
	}
	var relojNulo *relojMemoriaPrueba
	if _, err := NuevoRepositorioBaremaciones(relojNulo, verificador, perfil); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("reloj con puntero nil admitido: %v", err)
	}
	if _, err := NuevoRepositorioBaremaciones(&relojMemoriaPrueba{}, verificador, perfil); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("reloj a cero admitido: %v", err)
	}
	if _, err := NuevoRepositorioBaremaciones(&relojMemoriaPrueba{instante: instanteMemoriaPrueba}, nil, perfil); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("verificador ausente admitido: %v", err)
	}
	if _, err := NuevoRepositorioBaremaciones(
		&relojMemoriaPrueba{instante: instanteMemoriaPrueba}, verificador, PerfilUsoRepositorioBaremacionesMemoria{},
	); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("perfil productivo/indefinido admitido: %v", err)
	}
}

func TestRepositorioBaremacionesAltaAtomicaIdempotenteYConCopiasProfundas(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	baremacion := nuevaBaremacionMemoriaPrueba(t)
	reserva := solicitudReservaAltaMemoria()

	capacidad, err := repositorio.ReservarCambio(context.Background(), reserva)
	if err != nil {
		t.Fatalf("reservar alta: %v", err)
	}
	if err := capacidad.Validar(); err != nil {
		t.Fatalf("capacidad invalida: %v", err)
	}
	decodificado, err := base64.RawURLEncoding.DecodeString(capacidad.Token.Revelar())
	if err != nil || len(decodificado) != 32 {
		t.Fatalf("token no opaco o no canonico: longitud=%d err=%v", len(decodificado), err)
	}

	confirmacion := solicitudConfirmarAltaMemoria(capacidad.Token, baremacion)
	resultado, err := repositorio.ConfirmarCambio(context.Background(), confirmacion)
	if err != nil {
		t.Fatalf("confirmar alta: %v", err)
	}
	if err := resultado.Validar(); err != nil || resultado.Version.Referencia.Numero != 1 {
		t.Fatalf("resultado invalido: %+v, err=%v", resultado, err)
	}
	if !resultado.Version.ConfirmadaEn.Equal(reloj.Ahora()) ||
		resultado.Version.ConfirmadaEn.Equal(confirmacion.ConfirmadaEn) {
		t.Fatalf("la fecha probatoria no procede del reloj fiable: solicitud=%v resultado=%v",
			confirmacion.ConfirmadaEn, resultado.Version.ConfirmadaEn)
	}
	comprobarUnicaTransaccionDerivada(t, repositorio, resultado)

	// El reintento exacto de confirmacion devuelve el mismo recibo sin crear
	// otra version, auditoria ni evento.
	repetido, err := repositorio.ConfirmarCambio(context.Background(), confirmacion)
	if err != nil {
		t.Fatalf("reintento exacto: %v", err)
	}
	if repetido.Evidencia != resultado.Evidencia ||
		repetido.Version.Referencia != resultado.Version.Referencia {
		t.Fatalf("el reintento no devolvio el mismo resultado: %+v / %+v", repetido, resultado)
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)

	// La misma reserva, incluso tras confirmarse, solo revela la version; el
	// token de capacidad no se vuelve a entregar.
	reservaRepetida, err := repositorio.ReservarCambio(context.Background(), reserva)
	if err != nil {
		t.Fatalf("reserva idempotente confirmada: %v", err)
	}
	if !reservaRepetida.Repetida || reservaRepetida.VersionConfirmada == nil ||
		reservaRepetida.Token.Revelar() != "" {
		t.Fatalf("respuesta idempotente insegura: %+v", reservaRepetida)
	}
	reloj.fijar(reserva.ExpiraEn.Add(time.Hour))
	if _, err := repositorio.ReservarCambio(context.Background(), reserva); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("una autorizacion expirada recupero el tombstone: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)

	// Ni el agregado de entrada ni los objetos devueltos comparten slices con
	// el estado custodiado.
	baremacion.EvidenciasIniciales[0].Referencia.DocumentoRef = "manipulado-entrada"
	resultado.Version.Agregado.EvidenciasIniciales[0].Referencia.DocumentoRef = "manipulado-salida"
	resultado.Version.Agregado.CalculoInicial.Evidencias[0].Referencia.DocumentoRef = "manipulado-calculo"
	contextoLectura := contextoMemoriaPrueba(
		puertosbolsa.AccionConsultarBaremacionVigente, baremacion.ID, reloj.Ahora(),
	)
	vigente := obtenerVigenteMemoria(t, repositorio, contextoLectura)
	if vigente.Agregado.EvidenciasIniciales[0].Referencia.DocumentoRef != "documento-001" ||
		vigente.Agregado.CalculoInicial.Evidencias[0].Referencia.DocumentoRef != "documento-001" {
		t.Fatalf("una copia externa altero el repositorio: %+v", vigente.Agregado)
	}

	reutilizada := reserva
	reutilizada.Contexto = contextoMemoriaPrueba(
		puertosbolsa.AccionReservarAltaBaremacion, reutilizada.BaremacionMeritoRef, reloj.Ahora(),
	)
	reutilizada.ExpiraEn = reutilizada.ExpiraEn.Add(-time.Minute)
	reutilizada = sellarReservaMemoria(reutilizada)
	if _, err := repositorio.ReservarCambio(context.Background(), reutilizada); !errors.Is(err, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada) {
		t.Fatalf("clave reutilizada con otra semantica admitida: %v", err)
	}

	contextoAjeno := contextoMemoriaPruebaIdentidad(
		puertosbolsa.AccionConsultarBaremacionVigente, baremacion.ID,
		principalAjenoMemoriaPrueba, "sujeto-ajeno", "autenticacion-ajena", "correlacion-ajena", reloj.Ahora(),
	)
	if _, err := repositorio.ObtenerVersionVigente(context.Background(), puertosbolsa.SolicitudObtenerBaremacionVigente{
		Contexto: contextoAjeno, BaremacionMeritoRef: baremacion.ID,
	}); !errors.Is(err, puertosbolsa.ErrBaremacionNoEncontrada) {
		t.Fatalf("lectura de otro sujeto no se cerro como inexistente: %v", err)
	}

	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := repositorio.ObtenerVersionVigente(ctxCancelado, puertosbolsa.SolicitudObtenerBaremacionVigente{
		Contexto: contextoLectura, BaremacionMeritoRef: baremacion.ID,
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion ignorada: %v", err)
	}
}

func TestRepositorioBaremacionesAbandonoVinculadoYConTombstone(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	reserva := solicitudReservaAltaMemoria()
	capacidad, err := repositorio.ReservarCambio(context.Background(), reserva)
	if err != nil {
		t.Fatal(err)
	}

	contextoAjeno := contextoMemoriaPruebaIdentidad(
		puertosbolsa.AccionAbandonarAltaBaremacion, reserva.BaremacionMeritoRef,
		principalAjenoMemoriaPrueba, "sujeto-001", "autenticacion-ajena", "correlacion-1", reloj.Ahora(),
	)
	if err := repositorio.AbandonarReserva(context.Background(), puertosbolsa.SolicitudAbandonarReservaBaremacion{
		Contexto: contextoAjeno, Token: capacidad.Token, Clase: reserva.Clase, BaremacionMeritoRef: reserva.BaremacionMeritoRef,
	}); !errors.Is(err, puertosbolsa.ErrReservaBaremacionNoValida) {
		t.Fatalf("otro principal abandono la reserva: %v", err)
	}

	abandono := puertosbolsa.SolicitudAbandonarReservaBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionAbandonarAltaBaremacion, reserva.BaremacionMeritoRef, reloj.Ahora(),
		),
		Token: capacidad.Token, Clase: reserva.Clase, BaremacionMeritoRef: reserva.BaremacionMeritoRef,
	}
	if err := repositorio.AbandonarReserva(context.Background(), abandono); err != nil {
		t.Fatalf("abandonar: %v", err)
	}
	if err := repositorio.AbandonarReserva(context.Background(), abandono); err != nil {
		t.Fatalf("reintento de abandono no idempotente: %v", err)
	}
	if _, err := repositorio.ConfirmarCambio(context.Background(), solicitudConfirmarAltaMemoria(
		capacidad.Token, nuevaBaremacionMemoriaPrueba(t),
	)); !errors.Is(err, puertosbolsa.ErrReservaBaremacionNoValida) {
		t.Fatalf("token abandonado confirmado: %v", err)
	}
	if _, err := repositorio.ReservarCambio(context.Background(), reserva); !errors.Is(err, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada) {
		t.Fatalf("clave abandonada reutilizada: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 0, 0, 0)
}

func TestRepositorioBaremacionesExpiraSinReactivarClaveNiToken(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	reserva := solicitudReservaAltaMemoria()
	capacidad, err := repositorio.ReservarCambio(context.Background(), reserva)
	if err != nil {
		t.Fatal(err)
	}
	reloj.fijar(reserva.ExpiraEn.Add(time.Nanosecond))
	confirmacion := solicitudConfirmarAltaMemoria(capacidad.Token, nuevaBaremacionMemoriaPrueba(t))
	confirmacion.ConfirmadaEn = reserva.ExpiraEn.Add(-time.Nanosecond)
	if _, err := repositorio.ConfirmarCambio(context.Background(), confirmacion); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("reserva expirada confirmada: %v", err)
	}
	if _, err := repositorio.ReservarCambio(context.Background(), reserva); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("autorizacion vencida reutilizada: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 0, 0, 0)
}

func TestRepositorioBaremacionesRevalidaRelojDentroDeSeccionCritica(t *testing.T) {
	t.Run("reservar", func(t *testing.T) {
		base := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
		reloj := &relojAvisadoMemoriaPrueba{base: base, avisos: make(chan struct{}, 4)}
		repositorio := nuevoRepositorioPrueba(t, reloj)
		vaciarAvisos(reloj.avisos)
		repositorio.mu.Lock()
		resultado := make(chan error, 1)
		go func() {
			_, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria())
			resultado <- err
		}()
		esperarAviso(t, reloj.avisos)
		base.fijar(solicitudReservaAltaMemoria().ExpiraEn.Add(time.Nanosecond))
		repositorio.mu.Unlock()
		if err := <-resultado; !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
			t.Fatalf("reserva vencida mientras esperaba el mutex: %v", err)
		}
		comprobarLongitudesInternas(t, repositorio, 0, 0, 0)
	})

	t.Run("confirmar", func(t *testing.T) {
		base := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
		reloj := &relojAvisadoMemoriaPrueba{base: base, avisos: make(chan struct{}, 8)}
		repositorio := nuevoRepositorioPrueba(t, reloj)
		capacidad, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria())
		if err != nil {
			t.Fatal(err)
		}
		vaciarAvisos(reloj.avisos)
		repositorio.mu.Lock()
		resultado := make(chan error, 1)
		baremacion := nuevaBaremacionMemoriaPrueba(t)
		go func() {
			_, err := repositorio.ConfirmarCambio(context.Background(), solicitudConfirmarAltaMemoria(
				capacidad.Token, baremacion,
			))
			resultado <- err
		}()
		esperarAviso(t, reloj.avisos)
		base.fijar(solicitudReservaAltaMemoria().ExpiraEn.Add(time.Nanosecond))
		repositorio.mu.Unlock()
		if err := <-resultado; !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
			t.Fatalf("confirmacion vencida mientras esperaba el mutex: %v", err)
		}
		comprobarLongitudesInternas(t, repositorio, 0, 0, 0)
	})

	t.Run("abandonar", func(t *testing.T) {
		base := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
		reloj := &relojAvisadoMemoriaPrueba{base: base, avisos: make(chan struct{}, 8)}
		repositorio := nuevoRepositorioPrueba(t, reloj)
		capacidad, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria())
		if err != nil {
			t.Fatal(err)
		}
		vaciarAvisos(reloj.avisos)
		repositorio.mu.Lock()
		resultado := make(chan error, 1)
		go func() {
			resultado <- repositorio.AbandonarReserva(context.Background(), puertosbolsa.SolicitudAbandonarReservaBaremacion{
				Contexto: contextoMemoriaPrueba(
					puertosbolsa.AccionAbandonarAltaBaremacion, "baremacion-001", instanteMemoriaPrueba,
				),
				Token: capacidad.Token, Clase: puertosbolsa.ClaseCambioAltaBaremacion, BaremacionMeritoRef: "baremacion-001",
			})
		}()
		esperarAviso(t, reloj.avisos)
		base.fijar(solicitudReservaAltaMemoria().ExpiraEn.Add(time.Nanosecond))
		repositorio.mu.Unlock()
		if err := <-resultado; !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
			t.Fatalf("abandono vencido mientras esperaba el mutex: %v", err)
		}
	})

	lecturas := []struct {
		nombre   string
		preparar func(*RepositorioBaremaciones, puertosbolsa.ResultadoConfirmarCambioBaremacion) func() error
	}{
		{
			nombre: "leer_vigente",
			preparar: func(repositorio *RepositorioBaremaciones, resultado puertosbolsa.ResultadoConfirmarCambioBaremacion) func() error {
				solicitud := puertosbolsa.SolicitudObtenerBaremacionVigente{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarBaremacionVigente, resultado.Version.Referencia.BaremacionMeritoRef, instanteMemoriaPrueba,
					),
					BaremacionMeritoRef: resultado.Version.Referencia.BaremacionMeritoRef,
				}
				return func() error {
					_, err := repositorio.ObtenerVersionVigente(context.Background(), solicitud)
					return err
				}
			},
		},
		{
			nombre: "leer_version",
			preparar: func(repositorio *RepositorioBaremaciones, resultado puertosbolsa.ResultadoConfirmarCambioBaremacion) func() error {
				solicitud := puertosbolsa.SolicitudObtenerVersionBaremacion{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarVersionBaremacion, resultado.Version.Referencia.BaremacionMeritoRef, instanteMemoriaPrueba,
					),
					BaremacionMeritoRef: resultado.Version.Referencia.BaremacionMeritoRef,
					Numero:              resultado.Version.Referencia.Numero,
				}
				return func() error {
					_, err := repositorio.ObtenerVersion(context.Background(), solicitud)
					return err
				}
			},
		},
		{
			nombre: "leer_evidencia_transaccion",
			preparar: func(repositorio *RepositorioBaremaciones, resultado puertosbolsa.ResultadoConfirmarCambioBaremacion) func() error {
				solicitud := puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion, resultado.Evidencia.AuditoriaRef, instanteMemoriaPrueba,
					),
					BaremacionMeritoRef: resultado.Version.Referencia.BaremacionMeritoRef,
					NumeroVersion:       resultado.Version.Referencia.Numero,
					AuditoriaRef:        resultado.Evidencia.AuditoriaRef,
					EventoOutboxRef:     resultado.Evidencia.EventoOutboxRef,
				}
				return func() error {
					_, err := repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitud)
					return err
				}
			},
		},
	}
	for _, caso := range lecturas {
		t.Run(caso.nombre, func(t *testing.T) {
			base := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
			reloj := &relojAvisadoMemoriaPrueba{base: base, avisos: make(chan struct{}, 16)}
			repositorio := nuevoRepositorioPrueba(t, reloj)
			resultadoAlta := confirmarAltaMemoria(t, repositorio, nuevaBaremacionMemoriaPrueba(t))
			operacion := caso.preparar(repositorio, resultadoAlta)
			vaciarAvisos(reloj.avisos)
			repositorio.mu.Lock()
			resultado := make(chan error, 1)
			go func() { resultado <- operacion() }()
			esperarAviso(t, reloj.avisos)
			base.fijar(instanteMemoriaPrueba.Add(puertosbolsa.VentanaMaximaUsoAutorizacionBaremacion + time.Nanosecond))
			repositorio.mu.Unlock()
			if err := <-resultado; !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
				t.Fatalf("lectura autorizada antes del bloqueo sobrevivio a su caducidad: %v", err)
			}
		})
	}
}

func TestRepositorioBaremacionesNoRetieneTokenEnClaroYAcotaMemoria(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	capacidad, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria())
	if err != nil {
		t.Fatal(err)
	}
	secreto := capacidad.Token.Revelar()
	repositorio.mu.RLock()
	volcado := fmt.Sprintf("%#v %#v", repositorio.reservasPorAmbito, repositorio.ambitoPorHuellaToken)
	repositorio.mu.RUnlock()
	if strings.Contains(volcado, secreto) {
		t.Fatal("el token de capacidad se conserva en claro")
	}
	if _, err := repositorio.ConfirmarCambio(context.Background(), solicitudConfirmarAltaMemoria(
		capacidad.Token, nuevaBaremacionMemoriaPrueba(t),
	)); err != nil {
		t.Fatal(err)
	}
	repositorio.mu.RLock()
	volcado = fmt.Sprintf("%#v %#v", repositorio.reservasPorAmbito, repositorio.ambitoPorHuellaToken)
	repositorio.mu.RUnlock()
	if strings.Contains(volcado, secreto) {
		t.Fatal("la confirmacion idempotente retiene el token en claro")
	}

	repositorioLleno := nuevoRepositorioPrueba(t, reloj)
	repositorioLleno.mu.Lock()
	for indice := 0; indice < maximoReservasMemoria; indice++ {
		repositorioLleno.reservasPorAmbito[fmt.Sprintf("reserva-%d", indice)] = reservaBaremacion{}
	}
	repositorioLleno.mu.Unlock()
	if _, err := repositorioLleno.ReservarCambio(context.Background(), solicitudReservaAltaMemoria()); !errors.Is(err, puertosbolsa.ErrReservaBaremacionNoValida) {
		t.Fatalf("repositorio sin capacidad no fallo cerrado: %v", err)
	}
}

func TestRepositorioBaremacionesConcurrenciaConfirmaUnaSolaVez(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	reserva := solicitudReservaAltaMemoria()

	const trabajadores = 48
	var exitos int32
	tokens := make(chan puertosbolsa.TokenReservaBaremacion, trabajadores)
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, err := repositorio.ReservarCambio(context.Background(), reserva)
			if err == nil {
				atomic.AddInt32(&exitos, 1)
				tokens <- resultado.Token
				return
			}
			if !errors.Is(err, puertosbolsa.ErrCambioBaremacionEnCurso) {
				errores <- err
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("error concurrente inesperado al reservar: %v", err)
	}
	if exitos != 1 {
		t.Fatalf("se emitieron %d capacidades para una operacion", exitos)
	}
	token := <-tokens
	confirmacion := solicitudConfirmarAltaMemoria(token, nuevaBaremacionMemoriaPrueba(t))

	resultados := make(chan puertosbolsa.ResultadoConfirmarCambioBaremacion, trabajadores)
	errores = make(chan error, trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, err := repositorio.ConfirmarCambio(context.Background(), confirmacion)
			if err != nil {
				errores <- err
				return
			}
			resultados <- resultado
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("confirmacion idempotente concurrente: %v", err)
	}
	close(resultados)
	var evidencia *puertosbolsa.EvidenciaTransaccionBaremacion
	for resultado := range resultados {
		if evidencia == nil {
			copia := resultado.Evidencia
			evidencia = &copia
			continue
		}
		if *evidencia != resultado.Evidencia {
			t.Fatalf("un reintento creo otra evidencia: %+v / %+v", *evidencia, resultado.Evidencia)
		}
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)
}

func TestRepositorioBaremacionesNoRevelaNiBloqueaReservasDeOtroSujeto(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	legitima := solicitudReservaAltaMemoria()
	capacidad, err := repositorio.ReservarCambio(context.Background(), legitima)
	if err != nil {
		t.Fatal(err)
	}
	ajena := legitima
	ajena.Contexto = contextoMemoriaPruebaIdentidad(
		puertosbolsa.AccionReservarAltaBaremacion, ajena.BaremacionMeritoRef,
		principalAjenoMemoriaPrueba, "sujeto-ajeno", "autenticacion-ajena", "correlacion-ajena", reloj.Ahora(),
	)
	ajena.ClaveIdempotencia = "alta-ajena"
	ajena = sellarReservaMemoria(ajena)
	if _, err := repositorio.ReservarCambio(context.Background(), ajena); !errors.Is(err, puertosbolsa.ErrBaremacionNoEncontrada) {
		t.Fatalf("se revelo una reserva activa de otro sujeto: %v", err)
	}
	if _, err := repositorio.ConfirmarCambio(context.Background(), solicitudConfirmarAltaMemoria(
		capacidad.Token, nuevaBaremacionMemoriaPrueba(t),
	)); err != nil {
		t.Fatalf("la tentativa ajena bloqueo la reserva legitima: %v", err)
	}
}

func TestRepositorioBaremacionesOCCAppendOnlyEHistoriaExacta(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	base := nuevaBaremacionMemoriaPrueba(t)
	resultadoAlta := confirmarAltaMemoria(t, repositorio, base)

	reloj.fijar(instanteMemoriaPrueba.Add(15 * time.Minute))
	actualizada := incorporarDecisionMemoriaPrueba(t, base)
	reservaDecision := solicitudReservaDecisionMemoria(resultadoAlta.Version.Referencia)
	capacidad, err := repositorio.ReservarCambio(context.Background(), reservaDecision)
	if err != nil {
		t.Fatalf("reservar decision: %v", err)
	}
	confirmacion := solicitudConfirmarDecisionMemoria(capacidad.Token, resultadoAlta.Version.Referencia, actualizada)

	// Una HMAC distinta, aunque formalmente valida y acompañada del token,
	// no consume ni modifica la reserva legitima.
	ataque := confirmacion
	ataque.HuellaSolicitudHMAC = hmacMemoria("7")
	if _, err := repositorio.ConfirmarCambio(context.Background(), ataque); !errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
		t.Fatalf("confirmacion no vinculada admitida: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)

	resultadoDecision, err := repositorio.ConfirmarCambio(context.Background(), confirmacion)
	if err != nil {
		t.Fatalf("confirmar decision: %v", err)
	}
	if resultadoDecision.Version.Referencia.Numero != 2 || len(resultadoDecision.Version.Agregado.Decisiones) != 1 {
		t.Fatalf("version append-only incorrecta: %+v", resultadoDecision.Version)
	}
	comprobarLongitudesInternas(t, repositorio, 2, 2, 2)
	repositorio.mu.RLock()
	if repositorio.auditorias[1].HuellaAnteriorAuditoriaSHA256 != repositorio.auditorias[0].HuellaRegistroSHA256 ||
		repositorio.eventosOutbox[1].HuellaEventoAnteriorSHA256 != repositorio.eventosOutbox[0].HuellaRegistroSHA256 ||
		repositorio.auditorias[0].Secuencia != 1 || repositorio.auditorias[1].Secuencia != 2 ||
		repositorio.eventosOutbox[0].Secuencia != 1 || repositorio.eventosOutbox[1].Secuencia != 2 ||
		repositorio.eventosOutbox[1].Estado != estadoEventoOutboxPendiente ||
		repositorio.auditorias[1].ManifiestoProbatorioRef != confirmacion.Manifiesto.Referencia ||
		repositorio.auditorias[1].HuellaManifiestoSHA256 != confirmacion.Manifiesto.HuellaManifiestoSHA256 ||
		repositorio.auditorias[1].DocumentoFirmadoCustodiadoRef == "" ||
		repositorio.auditorias[1].EvidenciaCustodiaFirmadoRef == "" ||
		repositorio.auditorias[1].EvidenciaRetencionFirmadoRef == "" ||
		repositorio.eventosOutbox[1].ManifiestoProbatorioRef != confirmacion.Manifiesto.Referencia ||
		repositorio.eventosOutbox[1].DocumentoFirmadoRef != repositorio.auditorias[1].DocumentoFirmadoCustodiadoRef ||
		len(repositorio.referenciasTransaccion) != 4 {
		repositorio.mu.RUnlock()
		t.Fatal("auditoria u outbox no forman una cadena tipada e integra")
	}
	repositorio.mu.RUnlock()

	v1, err := repositorio.ObtenerVersion(context.Background(), puertosbolsa.SolicitudObtenerVersionBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConsultarVersionBaremacion, base.ID, reloj.Ahora(),
		), BaremacionMeritoRef: base.ID, Numero: 1,
	})
	if err != nil || len(v1.Agregado.Decisiones) != 0 || v1.Referencia != resultadoAlta.Version.Referencia {
		t.Fatalf("version historica alterada: %+v, err=%v", v1, err)
	}
	v2, err := repositorio.ObtenerVersion(context.Background(), puertosbolsa.SolicitudObtenerVersionBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConsultarVersionBaremacion, base.ID, reloj.Ahora(),
		), BaremacionMeritoRef: base.ID, Numero: 2,
	})
	if err != nil || len(v2.Agregado.Decisiones) != 1 || v2.Referencia != resultadoDecision.Version.Referencia {
		t.Fatalf("version historica 2 incorrecta: %+v, err=%v", v2, err)
	}
	v2.Agregado.Decisiones[0].Contenido.Motivo = "manipulado"
	v2Otra, err := repositorio.ObtenerVersion(context.Background(), puertosbolsa.SolicitudObtenerVersionBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConsultarVersionBaremacion, base.ID, reloj.Ahora(),
		), BaremacionMeritoRef: base.ID, Numero: 2,
	})
	if err != nil || v2Otra.Agregado.Decisiones[0].Contenido.Motivo == "manipulado" {
		t.Fatalf("historial comparte memoria: %+v, err=%v", v2Otra, err)
	}

	// La decision que autorizo la primera reserva tecnica no puede autorizar
	// un segundo efecto, aun conservando exactamente actor, sesion y recurso.
	reutilizada := solicitudReservaDecisionMemoria(resultadoDecision.Version.Referencia)
	reutilizada.ClaveIdempotencia = "decision-baremacion-segunda"
	reutilizada = sellarReservaMemoria(reutilizada)
	if _, err := repositorio.ReservarCambio(context.Background(), reutilizada); !errors.Is(
		err, puertosbolsa.ErrAutorizacionBaremacionReutilizada,
	) {
		t.Fatalf("DecisionRef consumida se reutilizo para otro efecto: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 2, 2, 2)
	repositorio.mu.RLock()
	if len(repositorio.usosAutorizacion) != 4 {
		repositorio.mu.RUnlock()
		t.Fatalf("consumos de autorizacion no atomicos: %d", len(repositorio.usosAutorizacion))
	}
	repositorio.mu.RUnlock()

	obsoleta := solicitudReservaDecisionMemoria(resultadoAlta.Version.Referencia)
	obsoleta.ClaveIdempotencia = "decision-obsoleta"
	obsoleta = sellarReservaMemoria(obsoleta)
	if _, err := repositorio.ReservarCambio(context.Background(), obsoleta); !errors.Is(err, puertosbolsa.ErrVersionBaremacionConflicto) {
		t.Fatalf("OCC obsoleto admitido: %v", err)
	}
	ajena := solicitudReservaDecisionMemoria(resultadoDecision.Version.Referencia)
	ajena.Contexto = contextoMemoriaPruebaIdentidad(
		puertosbolsa.AccionReservarDecisionBaremacion, ajena.BaremacionMeritoRef,
		principalAjenoMemoriaPrueba, "sujeto-ajeno", "autenticacion-ajena", "correlacion-ajena", reloj.Ahora(),
	)
	ajena.ClaveIdempotencia = "decision-ajena"
	ajena = sellarReservaMemoria(ajena)
	if _, err := repositorio.ReservarCambio(context.Background(), ajena); !errors.Is(err, puertosbolsa.ErrBaremacionNoEncontrada) {
		t.Fatalf("OCC revelo una baremacion de otro sujeto: %v", err)
	}
}

func TestRepositorioBaremacionesNoMutaAnteConfirmacionInvalida(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	base := nuevaBaremacionMemoriaPrueba(t)
	reserva := solicitudReservaAltaMemoria()
	capacidad, err := repositorio.ReservarCambio(context.Background(), reserva)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := solicitudConfirmarAltaMemoria(capacidad.Token, base)
	invalida := confirmacion
	invalida.Trazabilidad.MotivoClave = ""
	if _, err := repositorio.ConfirmarCambio(context.Background(), invalida); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("confirmacion incompleta admitida: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 0, 0, 0)
	if _, err := repositorio.ConfirmarCambio(context.Background(), confirmacion); err != nil {
		t.Fatalf("la entrada invalida consumio la reserva legitima: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)

	modificada := confirmacion
	modificada.Trazabilidad.Motivo = "Otro motivo formalmente valido."
	modificada = sellarConfirmacionMemoria(modificada)
	if _, err := repositorio.ConfirmarCambio(context.Background(), modificada); !errors.Is(err, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada) {
		t.Fatalf("token confirmado se reutilizo con otro contenido: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)
}

func TestRepositorioBaremacionesRecuperaTransaccionTipadaConAutorizacionExacta(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	resultado := confirmarAltaMemoria(t, repositorio, nuevaBaremacionMemoriaPrueba(t))
	solicitud := puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
			resultado.Evidencia.AuditoriaRef, reloj.Ahora(),
		),
		BaremacionMeritoRef: resultado.Version.Referencia.BaremacionMeritoRef,
		NumeroVersion:       resultado.Version.Referencia.Numero,
		AuditoriaRef:        resultado.Evidencia.AuditoriaRef,
		EventoOutboxRef:     resultado.Evidencia.EventoOutboxRef,
	}
	recuperada, err := repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitud)
	if err != nil || recuperada.ValidarPara(solicitud) != nil || recuperada.Evidencia != resultado.Evidencia {
		t.Fatalf("recuperacion probatoria invalida: %+v / %v", recuperada, err)
	}
	recuperada.Auditoria.CamposPermitidos[0] = "manipulado"
	recuperadaOtra, err := repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitud)
	if err != nil || recuperadaOtra.Auditoria.CamposPermitidos[0] == "manipulado" {
		t.Fatalf("la recuperacion comparte la lista autorizada custodiada: %+v / %v", recuperadaOtra, err)
	}
	solicitud.Contexto = contextoMemoriaPrueba(
		puertosbolsa.AccionConsultarVersionBaremacion,
		resultado.Version.Referencia.BaremacionMeritoRef, reloj.Ahora(),
	)
	if _, err := repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitud); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("una lectura de version sirvio para recuperar auditoria: %v", err)
	}
}

func TestRepositorioBaremacionesCadenaManipuladaImpideAnexarYRecuperar(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	base := nuevaBaremacionMemoriaPrueba(t)
	alta := confirmarAltaMemoria(t, repositorio, base)
	reloj.fijar(instanteMemoriaPrueba.Add(15 * time.Minute))
	actualizada := incorporarDecisionMemoriaPrueba(t, base)
	reserva, err := repositorio.ReservarCambio(context.Background(), solicitudReservaDecisionMemoria(alta.Version.Referencia))
	if err != nil {
		t.Fatalf("reservar con cadena integra: %v", err)
	}
	repositorio.mu.Lock()
	repositorio.auditorias[0].Motivo = "manipulacion no sellada"
	repositorio.mu.Unlock()

	solicitudRecuperar := puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
			alta.Evidencia.AuditoriaRef, reloj.Ahora(),
		),
		BaremacionMeritoRef: base.ID, NumeroVersion: 1,
		AuditoriaRef: alta.Evidencia.AuditoriaRef, EventoOutboxRef: alta.Evidencia.EventoOutboxRef,
	}
	if _, err := repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitudRecuperar); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("cadena manipulada recuperada como confiable: %v", err)
	}

	if _, err := repositorio.ConfirmarCambio(context.Background(), solicitudConfirmarDecisionMemoria(
		reserva.Token, alta.Version.Referencia, actualizada,
	)); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("se anexo sobre una cadena manipulada: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 1, 1, 1)
}

func TestRepositorioBaremacionesSelloAutenticaRepresentacionCanonicaExacta(t *testing.T) {
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio := nuevoRepositorioPrueba(t, reloj)
	reserva := solicitudReservaAltaMemoria()
	alterada := reserva
	alterada.ExpiraEn = alterada.ExpiraEn.Add(-time.Second)
	if _, err := repositorio.ReservarCambio(context.Background(), alterada); !errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
		t.Fatalf("sello autentico de otra representacion admitido al reservar: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 0, 0, 0)

	capacidad, err := repositorio.ReservarCambio(context.Background(), reserva)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := solicitudConfirmarAltaMemoria(capacidad.Token, nuevaBaremacionMemoriaPrueba(t))
	alteradaConfirmacion := confirmacion
	alteradaConfirmacion.Trazabilidad.Motivo = "Motivo diferente pero formalmente valido."
	if _, err := repositorio.ConfirmarCambio(context.Background(), alteradaConfirmacion); !errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
		t.Fatalf("sello autentico de otra representacion admitido al confirmar: %v", err)
	}
	comprobarLongitudesInternas(t, repositorio, 0, 0, 0)
	if _, err := repositorio.ConfirmarCambio(context.Background(), confirmacion); err != nil {
		t.Fatalf("el ataque consumio la reserva legitima: %v", err)
	}
}

func vaciarAvisos(avisos chan struct{}) {
	for {
		select {
		case <-avisos:
		default:
			return
		}
	}
}

func esperarAviso(t *testing.T, avisos chan struct{}) {
	t.Helper()
	select {
	case <-avisos:
	case <-time.After(2 * time.Second):
		t.Fatal("la operacion no alcanzo la lectura previa del reloj")
	}
}

func nuevoRepositorioPrueba(t *testing.T, reloj puertosbolsa.Reloj) *RepositorioBaremaciones {
	t.Helper()
	repositorio, err := NuevoRepositorioBaremaciones(
		reloj, verificadorHMACMemoriaPrueba{}, PerfilRepositorioBaremacionesSoloPruebas(),
	)
	if err != nil {
		t.Fatalf("crear repositorio: %v", err)
	}
	return repositorio
}

func contextoMemoriaPrueba(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	return contextoMemoriaPruebaIdentidad(
		accion, recursoRef, principalBaremacionMemoriaPrueba, "sujeto-001", "autenticacion-1", "correlacion-1", instante,
	)
}

func contextoMemoriaPruebaIdentidad(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	return contextoMemoriaPruebaAutorizacion(
		accion, recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef,
		referenciaAutorizacionMemoria(accion), instante,
	)
}

func contextoMemoriaPruebaAutorizacion(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef, autorizacionRef string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	campos, existe := puertosbolsa.CamposRequeridosOperacionBaremacion(accion)
	if !existe {
		panic("accion de prueba desconocida")
	}
	clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
	if !existe {
		panic("clase de recurso de prueba desconocida")
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(clase),
		Ambitos: map[string]string{"sujeto_ref": sujetoRef},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		panic(err)
	}
	contextoActor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instante, principalRef, perfilBaremacionMemoriaPrueba,
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		panic(err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		panic(err)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: autorizacionRef, Concedida: true, Codigo: "concedida", PrincipalID: contextoActor.Principal.ID,
		PerfilActivoRef: contextoActor.PerfilActivoRef, Accion: string(accion), RecursoRef: recursoRef,
		ModuloID: "bolsa", TipoRecurso: string(clase), ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: "baremacion_proceso_selectivo", CorrelacionRef: correlacionRef,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion-tecnico-v1", AsignacionHuellaSHA256: huellaMemoria("1"),
		VersionRolRef: "rol-tecnico-v1", VersionRolHuellaSHA256: huellaMemoria("2"),
		ControlVigenciaVersionRolRef:      "rol-tecnico-v1",
		ControlVigenciaVersionRolRevision: 1, ControlVigenciaVersionRolHuellaSHA256: huellaMemoria("3"),
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  dominiovec.AuthAssuranceHigh, CamposPermitidos: campos,
		EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(4 * time.Minute),
	}
	contexto, err := puertosbolsa.NuevaAutorizacionOperacionBaremacion(
		decision,
		puertosbolsa.VinculoAutenticacionBaremacion{
			SujetoRef: sujetoRef, Metodo: datosVinculo.MetodoObservado,
			Garantia: datosVinculo.GarantiaObservada, AutenticacionRef: datosVinculo.AutenticacionRef,
			SesionRef: datosVinculo.SesionRef, SesionEmitidaEn: datosVinculo.SesionEmitidaEn,
			SesionValidaHasta: datosVinculo.SesionValidaHasta, VinculoAutenticacionActor: vinculo,
		},
		instante,
	)
	if err != nil {
		panic(err)
	}
	return contexto
}

func referenciaAutorizacionMemoria(accion puertosbolsa.AccionOperacionBaremacion) string {
	return "autorizacion-" + strings.ReplaceAll(string(accion), ".", "-")
}

func solicitudReservaAltaMemoria() puertosbolsa.SolicitudReservarCambioBaremacion {
	solicitud := puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto:          contextoMemoriaPrueba(puertosbolsa.AccionReservarAltaBaremacion, "baremacion-001", instanteMemoriaPrueba),
		Clase:             puertosbolsa.ClaseCambioAltaBaremacion,
		ClaveIdempotencia: "alta-baremacion-001", BaremacionMeritoRef: "baremacion-001",
		HuellaSolicitudHMAC: hmacMemoria("0"), SolicitadaEn: instanteMemoriaPrueba.Add(-2 * time.Minute),
		ExpiraEn: instanteMemoriaPrueba.Add(5 * time.Minute),
	}
	return sellarReservaMemoria(solicitud)
}

func solicitudConfirmarAltaMemoria(
	token puertosbolsa.TokenReservaBaremacion,
	baremacion dominiobolsa.BaremacionMerito,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	solicitud := puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoMemoriaPrueba(puertosbolsa.AccionConfirmarAltaBaremacion, baremacion.ID, instanteMemoriaPrueba),
		Token:    token, Clase: puertosbolsa.ClaseCambioAltaBaremacion,
		HuellaSolicitudHMAC: hmacMemoria("0"), Agregado: baremacion,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "alta_autobaremacion", Motivo: "Alta de la autobaremacion calculada oficialmente.",
		},
		ConfirmadaEn: instanteMemoriaPrueba.Add(-time.Minute),
	}
	return sellarConfirmacionMemoria(solicitud)
}

func solicitudReservaDecisionMemoria(
	version puertosbolsa.ReferenciaVersionBaremacion,
) puertosbolsa.SolicitudReservarCambioBaremacion {
	solicitud := puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionReservarDecisionBaremacion, "baremacion-001", instanteMemoriaPrueba.Add(15*time.Minute),
		), Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		ClaveIdempotencia: "decision-baremacion-001", BaremacionMeritoRef: "baremacion-001",
		VersionEsperada: &version, HuellaSolicitudHMAC: hmacMemoria("0"),
		SolicitadaEn: instanteMemoriaPrueba.Add(14 * time.Minute),
		ExpiraEn:     instanteMemoriaPrueba.Add(20 * time.Minute),
	}
	return sellarReservaMemoria(solicitud)
}

func solicitudConfirmarDecisionMemoria(
	token puertosbolsa.TokenReservaBaremacion,
	version puertosbolsa.ReferenciaVersionBaremacion,
	baremacion dominiobolsa.BaremacionMerito,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	clon, err := baremacion.ClonarCanonica()
	if err != nil || len(clon.Decisiones) == 0 {
		panic(puertosbolsa.ErrSolicitudBaremacionInvalida)
	}
	baremacion = clon
	ultima := &baremacion.Decisiones[len(baremacion.Decisiones)-1]
	solicitud := puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConfirmarDecisionBaremacion, baremacion.ID, instanteMemoriaPrueba.Add(15*time.Minute),
		), Token: token, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		VersionEsperada: &version, HuellaSolicitudHMAC: hmacMemoria("0"), Agregado: baremacion,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "decision_tecnica_firmada", Motivo: "Incorporacion de la decision tecnica validada y firmada.",
		},
		ConfirmadaEn: instanteMemoriaPrueba.Add(15 * time.Minute),
	}
	manifiestoBase, err := manifiestoMemoriaPrueba(
		version, ultima.Contenido, ultima.Firma,
		solicitud.Contexto.Proyeccion().AutorizacionRef, solicitud.ConfirmadaEn,
	)
	if err != nil {
		panic(err)
	}
	preparado, representacion, err := manifiestoBase.PrepararSellado()
	if err != nil {
		panic(err)
	}
	selloManifiesto := calcularSelloMemoria(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV2,
		representacion.Revelar(),
	)
	manifiesto, err := preparado.IncorporarSello(selloManifiesto)
	if err != nil {
		panic(err)
	}
	ultima.Firma.ManifiestoProbatorioRef = manifiesto.Referencia
	ultima.Firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
	ultima.Firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
	decisionActualizada, err := dominiobolsa.ConstituirDecisionFirmada(ultima.Contenido, ultima.Firma)
	if err != nil {
		panic(err)
	}
	*ultima = decisionActualizada
	solicitud.Agregado = baremacion
	solicitud.Manifiesto = &manifiesto
	if err := manifiesto.ValidarPara(version, ultima.Contenido); err != nil {
		panic(fmt.Sprintf("manifiesto invalido: %v", err))
	}
	if err := ultima.Validar(); err != nil {
		panic(fmt.Sprintf("decision con manifiesto invalida: %v", err))
	}
	if err := solicitud.Validar(); err != nil {
		panic(fmt.Sprintf("confirmacion con manifiesto invalida: %v", err))
	}
	return sellarConfirmacionMemoria(solicitud)
}

func manifiestoMemoriaPrueba(
	version puertosbolsa.ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	firma dominiobolsa.FirmaDecisionTecnica,
	autorizacionConfirmacionRef string,
	creadoEn time.Time,
) (puertosbolsa.ManifiestoProbatorioBaremacion, error) {
	autorizaciones := make([]puertosbolsa.AutorizacionProbatoriaBaremacion, 0, 24)
	agregarAutorizacion := func(
		accion puertosbolsa.AccionOperacionBaremacion,
		recurso, referencia string,
	) error {
		clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
		if !existe {
			return puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		if referencia == "" {
			referencia = fmt.Sprintf("autorizacion-manifiesto-memoria-%02d", len(autorizaciones)+1)
		}
		autorizaciones = append(autorizaciones, puertosbolsa.AutorizacionProbatoriaBaremacion{
			Secuencia: uint32(len(autorizaciones) + 1), Accion: accion,
			ClaseRecurso: clase, RecursoRef: recurso, AutorizacionRef: referencia,
		})
		return nil
	}
	agregar := func(accion puertosbolsa.AccionOperacionBaremacion, recurso, referencia string) error {
		return agregarAutorizacion(accion, recurso, referencia)
	}
	if err := agregar(puertosbolsa.AccionConsultarBaremacionVigente, contenido.BaremacionMeritoRef, ""); err != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
	}
	for _, paso := range []struct {
		accion  puertosbolsa.AccionOperacionBaremacion
		recurso string
	}{
		{puertosbolsa.AccionRecuperarCalculoBaremacion, contenido.CalculoOficial.CalculoRef},
		{puertosbolsa.AccionConsultarCriterioBaremacion, contenido.Criterio.ProcesoRef},
	} {
		if err := agregar(paso.accion, paso.recurso, ""); err != nil {
			return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
		}
	}
	evidenciasMerito, err := contenido.CalculoOficial.EvidenciasCanonicas()
	if err != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
	}
	for _, evidencia := range evidenciasMerito {
		if err := agregar(puertosbolsa.AccionConsultarEvidenciaBaremacion, evidencia.Referencia.DocumentoRef, ""); err != nil {
			return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
		}
		if err := agregar(puertosbolsa.AccionConsultarRepresentacionBaremacion, evidencia.Referencia.RepresentacionRef, ""); err != nil {
			return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
		}
	}
	accionAdopcion, existe := puertosbolsa.AccionAdopcionParaClase(contenido.Clase)
	if !existe {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	for _, paso := range []struct {
		accion     puertosbolsa.AccionOperacionBaremacion
		recurso    string
		referencia string
	}{
		{accionAdopcion, contenido.BaremacionMeritoRef, contenido.AutorizacionRef},
		{puertosbolsa.AccionConsultarPoliticaFirmaBaremacion, firma.PoliticaFirmaRef, ""},
		{puertosbolsa.AccionCodificarDecisionBaremacion, contenido.ID, ""},
		{puertosbolsa.AccionCustodiarDecisionBaremacion, contenido.ID, ""},
		{puertosbolsa.AccionPrepararFirmaDecisionBaremacion, contenido.ID, ""},
		{puertosbolsa.AccionConsultarFirmaDecisionBaremacion, firma.SesionFirmaInteractivaRef, ""},
		{puertosbolsa.AccionValidarFirmaDecisionBaremacion, firma.FirmaRef, ""},
		{puertosbolsa.AccionSellarTiempoDecisionBaremacion, firma.FirmaRef, ""},
		{puertosbolsa.AccionValidarFirmaDecisionBaremacion, firma.FirmaRef, ""},
		{puertosbolsa.AccionAumentarFirmaDecisionBaremacion, firma.FirmaRef, ""},
		{puertosbolsa.AccionValidarFirmaDecisionBaremacion, firma.FirmaRef, ""},
		{puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion, firma.DocumentoFirmadoRef, ""},
		{puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion, firma.DocumentoFirmadoRef, ""},
		{puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion, firma.DocumentoFirmadoRef, ""},
		{puertosbolsa.AccionReservarDecisionBaremacion, contenido.BaremacionMeritoRef, ""},
		{puertosbolsa.AccionConfirmarDecisionBaremacion, contenido.BaremacionMeritoRef, autorizacionConfirmacionRef},
	} {
		if err := agregar(paso.accion, paso.recurso, paso.referencia); err != nil {
			return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
		}
	}
	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
	}
	evidencias := []puertosbolsa.EvidenciaProbatoriaBaremacion{
		{Tipo: puertosbolsa.EvidenciaEstadoBaseBaremacion, Referencia: contenido.BaremacionMeritoRef, HuellaEvidenciaSHA256: version.HuellaEstadoSHA256},
		{Tipo: puertosbolsa.EvidenciaCalculoOficialBaremacion, Referencia: "evidencia-gobierno-calculo-memoria-001", HuellaEvidenciaSHA256: huellaMemoria("5")},
		{Tipo: puertosbolsa.EvidenciaCriterioPublicadoBaremacion, Referencia: contenido.Criterio.ProcesoRef, HuellaEvidenciaSHA256: contenido.Criterio.HuellaSHA256},
	}
	for _, evidencia := range evidenciasMerito {
		evidencias = append(evidencias,
			puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaDocumentoMeritoBaremacion, Referencia: evidencia.Referencia.DocumentoRef, HuellaEvidenciaSHA256: evidencia.Referencia.HuellaSHA256},
			puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaRepresentacionBaremacion, Referencia: evidencia.Referencia.RepresentacionRef, HuellaEvidenciaSHA256: evidencia.Referencia.HuellaSHA256},
		)
	}
	evidencias = append(evidencias,
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaContenidoDecisionBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: huellaContenido},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaPoliticaFirmaBaremacion, Referencia: "aprobacion-politica-firma-memoria-001", HuellaEvidenciaSHA256: firma.HuellaPoliticaFirmaSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaDocumentoCanonicoBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: firma.HuellaDocumentoFirmableSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaCustodiaFirmableBaremacion, Referencia: firma.EvidenciaCustodiaRef, HuellaEvidenciaSHA256: firma.HuellaDocumentoFirmableSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaPreparacionFirmaBaremacion, Referencia: "evidencia-preparacion-firma-memoria-001", HuellaEvidenciaSHA256: huellaMemoria("6")},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaConsultaFirmaBaremacion, Referencia: "evidencia-consulta-firma-memoria-001", HuellaEvidenciaSHA256: huellaMemoria("7")},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaValidacionInicialBaremacion, Referencia: firma.ValidacionInicialFirmaRef, HuellaEvidenciaSHA256: firma.HuellaValidacionInicialSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaSelloTiempoBaremacion, Referencia: firma.SelloTiempoRef, HuellaEvidenciaSHA256: firma.HuellaSelloTiempoSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaVinculoRevisionSelladaBaremacion, Referencia: firma.VinculoRevisionSelladaRef, HuellaEvidenciaSHA256: firma.HuellaVinculoRevisionSelladaSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaValidacionDocumentoSelladoBaremacion, Referencia: firma.ValidacionDocumentoSelladoRef, HuellaEvidenciaSHA256: firma.HuellaValidacionDocumentoSelladoSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaAumentoLongevidadBaremacion, Referencia: firma.AumentoLongevidadRef, HuellaEvidenciaSHA256: firma.HuellaAumentoLongevidadSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaVinculoRevisionLongevaBaremacion, Referencia: firma.VinculoRevisionLongevaRef, HuellaEvidenciaSHA256: firma.HuellaVinculoRevisionLongevaSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaValidacionFinalBaremacion, Referencia: firma.ValidacionFirmaRef, HuellaEvidenciaSHA256: firma.HuellaValidacionSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaRecuperacionFirmadoBaremacion, Referencia: firma.EvidenciaRecuperacionFirmadoRef, HuellaEvidenciaSHA256: firma.HuellaEvidenciaRecuperacionSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaCustodiaFirmadoBaremacion, Referencia: firma.EvidenciaCustodiaDocumentoFirmadoRef, HuellaEvidenciaSHA256: firma.HuellaDocumentoSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaRetencionFirmadoBaremacion, Referencia: firma.EvidenciaRetencionDocumentoFirmadoRef, HuellaEvidenciaSHA256: firma.HuellaDocumentoSHA256},
	)
	for indice := range evidencias {
		evidencias[indice].Secuencia = uint32(indice + 1)
	}
	return puertosbolsa.ManifiestoProbatorioBaremacion{
		Esquema:        puertosbolsa.EsquemaManifiestoProbatorioBaremacion,
		Finalidad:      puertosbolsa.FinalidadManifiestoProbatorioBaremacion,
		VersionEsquema: puertosbolsa.VersionManifiestoProbatorioBaremacion,
		Referencia:     "manifiesto-probatorio-" + contenido.ID, ProcesoRef: contenido.ProcesoRef,
		SolicitudRef: contenido.SolicitudRef, SujetoRef: contenido.SujetoRef,
		BaremacionMeritoRef: contenido.BaremacionMeritoRef, DecisionRef: contenido.ID,
		VersionBase: version.Numero, HuellaVersionBaseSHA256: version.HuellaEstadoSHA256,
		Autorizaciones: autorizaciones, Evidencias: evidencias, CreadoEn: creadoEn.UTC(),
	}, nil
}

func sellarReservaMemoria(solicitud puertosbolsa.SolicitudReservarCambioBaremacion) puertosbolsa.SolicitudReservarCambioBaremacion {
	representacion, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		panic(err)
	}
	solicitud.HuellaSolicitudHMAC = calcularSelloMemoria(puertosbolsa.FinalidadSelloReservaBaremacion, representacion.Revelar())
	return solicitud
}

func sellarConfirmacionMemoria(solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	representacion, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		panic(err)
	}
	solicitud.HuellaSolicitudHMAC = calcularSelloMemoria(puertosbolsa.FinalidadSelloConfirmacionBaremacion, representacion.Revelar())
	return solicitud
}

func calcularSelloMemoria(finalidad puertosbolsa.FinalidadSelloBaremacion, contenido []byte) string {
	representacion, err := puertosbolsa.NuevaCargaProtegida(contenido)
	if err != nil {
		panic(err)
	}
	material, err := (puertosbolsa.SolicitudSellarSelloBaremacion{
		Finalidad: finalidad, RepresentacionCanonica: representacion,
	}).MaterialCanonicoHMAC()
	if err != nil {
		panic(err)
	}
	mac := hmac.New(sha256.New, claveHMACMemoriaPrueba)
	_, _ = mac.Write(material.Revelar())
	return "hmac-sha256:memoria_1:" + hex.EncodeToString(mac.Sum(nil))
}

func confirmarAltaMemoria(
	t *testing.T,
	repositorio *RepositorioBaremaciones,
	baremacion dominiobolsa.BaremacionMerito,
) puertosbolsa.ResultadoConfirmarCambioBaremacion {
	t.Helper()
	capacidad, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria())
	if err != nil {
		t.Fatalf("reservar alta: %v", err)
	}
	resultado, err := repositorio.ConfirmarCambio(
		context.Background(), solicitudConfirmarAltaMemoria(capacidad.Token, baremacion),
	)
	if err != nil {
		t.Fatalf("confirmar alta: %v", err)
	}
	return resultado
}

func obtenerVigenteMemoria(
	t *testing.T,
	repositorio *RepositorioBaremaciones,
	contexto puertosbolsa.ContextoOperacionBaremacion,
) puertosbolsa.VersionBaremacion {
	t.Helper()
	version, err := repositorio.ObtenerVersionVigente(context.Background(), puertosbolsa.SolicitudObtenerBaremacionVigente{
		Contexto: contexto, BaremacionMeritoRef: "baremacion-001",
	})
	if err != nil {
		t.Fatalf("obtener vigente: %v", err)
	}
	return version
}

func comprobarUnicaTransaccionDerivada(
	t *testing.T,
	repositorio *RepositorioBaremaciones,
	resultado puertosbolsa.ResultadoConfirmarCambioBaremacion,
) {
	t.Helper()
	repositorio.mu.RLock()
	defer repositorio.mu.RUnlock()
	if len(repositorio.auditorias) != 1 || len(repositorio.eventosOutbox) != 1 {
		t.Fatalf("transaccion parcial: auditorias=%d eventos=%d", len(repositorio.auditorias), len(repositorio.eventosOutbox))
	}
	if len(repositorio.referenciasTransaccion) != 2 {
		t.Fatalf("referencias transaccionales no reservadas de forma unica: %d", len(repositorio.referenciasTransaccion))
	}
	auditoria := repositorio.auditorias[0]
	evento := repositorio.eventosOutbox[0]
	if auditoria.HuellaRegistroSHA256 != huellaAuditoria(auditoria) ||
		evento.HuellaRegistroSHA256 != huellaEvento(evento) ||
		evento.AuditoriaRef != auditoria.Referencia ||
		evento.HuellaAuditoriaSHA256 != auditoria.HuellaRegistroSHA256 ||
		resultado.Evidencia.AuditoriaRef != auditoria.Referencia ||
		resultado.Evidencia.EventoOutboxRef != evento.Referencia ||
		resultado.Evidencia.HuellaAuditoriaSHA256 != auditoria.HuellaRegistroSHA256 ||
		resultado.Evidencia.HuellaEventoOutboxSHA256 != evento.HuellaRegistroSHA256 {
		t.Fatalf("evidencia transaccional no enlazada: auditoria=%+v evento=%+v evidencia=%+v", auditoria, evento, resultado.Evidencia)
	}
	contexto := contextoMemoriaPrueba(
		puertosbolsa.AccionConfirmarAltaBaremacion, "baremacion-001", instanteMemoriaPrueba,
	).Proyeccion()
	if auditoria.PrincipalRef != contexto.PrincipalRef ||
		auditoria.AutorizacionRef != contexto.AutorizacionRef ||
		auditoria.FinalidadClave != contexto.FinalidadClave ||
		auditoria.Resultado != "correcto" {
		t.Fatalf("auditoria no derivada del contexto autorizado: %+v", auditoria)
	}
	if !auditoria.SolicitadaConfirmacionEn.Equal(instanteMemoriaPrueba.Add(-time.Minute)) ||
		!auditoria.RegistradaEn.Equal(instanteMemoriaPrueba) ||
		!evento.RegistradoEn.Equal(instanteMemoriaPrueba) {
		t.Fatalf("tiempos declarado y fiable no quedaron separados: auditoria=%+v evento=%+v", auditoria, evento)
	}
}

func comprobarLongitudesInternas(
	t *testing.T,
	repositorio *RepositorioBaremaciones,
	versiones, auditorias, eventos int,
) {
	t.Helper()
	repositorio.mu.RLock()
	defer repositorio.mu.RUnlock()
	if len(repositorio.versionesPorBaremacion["baremacion-001"]) != versiones ||
		len(repositorio.auditorias) != auditorias || len(repositorio.eventosOutbox) != eventos {
		t.Fatalf("estado parcial: versiones=%d auditorias=%d eventos=%d",
			len(repositorio.versionesPorBaremacion["baremacion-001"]),
			len(repositorio.auditorias), len(repositorio.eventosOutbox))
	}
}

func nuevaBaremacionMemoriaPrueba(t *testing.T) dominiobolsa.BaremacionMerito {
	t.Helper()
	criterio := criterioMemoriaPrueba()
	evidencias := evidenciasMemoriaPrueba()
	calculo := calculoMemoriaPrueba(criterio, evidencias)
	baremacion, err := dominiobolsa.NuevaBaremacionMerito(dominiobolsa.AltaMeritoBaremable{
		ID: "baremacion-001", ProcesoRef: criterio.ProcesoRef, SolicitudRef: "solicitud-001",
		SujetoRef: "sujeto-001", Criterio: criterio, EvidenciasIniciales: evidencias,
		PuntosDeclarados: 5_000_000, CalculoOficial: calculo,
		CreadaEn: instanteMemoriaPrueba.Add(-19 * time.Minute),
	})
	if err != nil {
		t.Fatalf("crear baremacion: %v", err)
	}
	return baremacion
}

func incorporarDecisionMemoriaPrueba(
	t *testing.T,
	baremacion dominiobolsa.BaremacionMerito,
) dominiobolsa.BaremacionMerito {
	t.Helper()
	contexto := contextoMemoriaPrueba(
		puertosbolsa.AccionAdoptarDecisionInicialBaremacion, baremacion.ID, instanteMemoriaPrueba.Add(time.Minute),
	).Proyeccion()
	propuesta := dominiobolsa.PropuestaDecisionTecnica{
		ID: "decision-001", CalculoOficial: baremacion.CalculoInicial,
		PuntosReconocidos: 4_000_000, Resultado: dominiobolsa.ResultadoAceptado,
		DecisorRef:         contexto.PrincipalRef,
		PerfilDecisorClave: contexto.PerfilActorClave,
		ValoracionesEvidencia: []dominiobolsa.ValoracionEvidencia{{
			Evidencia: baremacion.EvidenciasIniciales[0], Estado: dominiobolsa.EstadoEvidenciaApta,
			ResultadoSubsanacion: dominiobolsa.ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido", Motivo: "Documento autentico, integro y suficiente.",
		}},
		MotivoClave: "valoracion_inicial", Motivo: "Valoracion conforme al criterio publicado.",
		FuentesNormativasRefs: []string{"norma-baremo-v7"},
		AutorizacionRef:       contexto.AutorizacionRef,
		FinalidadClave:        contexto.FinalidadClave,
		CorrelacionRef:        contexto.CorrelacionRef,
		DecididaEn:            instanteMemoriaPrueba.Add(time.Minute),
	}
	contenido, err := baremacion.PrepararDecisionInicial(propuesta)
	if err != nil {
		t.Fatalf("preparar decision: %v", err)
	}
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella contenido: %v", err)
	}
	firmadaEn := instanteMemoriaPrueba.Add(2 * time.Minute)
	firma := firmaMemoriaPrueba(contenido, huella, firmadaEn)
	decision, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma)
	if err != nil {
		t.Fatalf("constituir decision: %v", err)
	}
	actualizada, err := baremacion.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar decision: %v", err)
	}
	return actualizada
}

func criterioMemoriaPrueba() dominiobolsa.ReferenciaCriterio {
	return dominiobolsa.ReferenciaCriterio{
		ProcesoRef: "proceso-selectivo-2026-017", Clave: "experiencia.entidad_publica.grupo_c1",
		Version: 7, HuellaSHA256: huellaMemoria("a"), PuntosMaximos: 10 * dominiobolsa.UnidadesPorPunto,
		ReglaCalculo: dominiobolsa.ReferenciaReglaCalculo{
			Clave: "experiencia_publica_dias", Version: 3, HuellaSHA256: huellaMemoria("9"),
		},
	}
}

func evidenciasMemoriaPrueba() []dominiobolsa.EvidenciaMerito {
	return []dominiobolsa.EvidenciaMerito{{Referencia: dominiobolsa.ReferenciaEvidencia{
		DocumentoRef: "documento-001", VersionDocumento: 3,
		RepresentacionRef: "representacion-001", HuellaSHA256: huellaMemoria("b"),
	}}}
}

func calculoMemoriaPrueba(
	criterio dominiobolsa.ReferenciaCriterio,
	evidencias []dominiobolsa.EvidenciaMerito,
) dominiobolsa.CalculoOficialBaremacion {
	return dominiobolsa.CalculoOficialBaremacion{
		CalculoRef: "calculo-oficial-inicial", ProcesoRef: criterio.ProcesoRef,
		SolicitudRef: "solicitud-001", SujetoRef: "sujeto-001", BaremacionMeritoRef: "baremacion-001",
		Criterio: criterio, Regla: criterio.ReglaCalculo, Evidencias: evidencias,
		EntradaRef: "entrada-calculo-inicial", HuellaEntradaSHA256: huellaMemoria("1"),
		PuntosCalculados: 4_250_000, DesgloseRef: "desglose-calculo-inicial",
		HuellaDesgloseSHA256: huellaMemoria("2"), ResultadoRef: "resultado-calculo-inicial",
		HuellaResultadoSHA256: huellaMemoria("3"), MotorCalculoRef: "motor-baremo-oficial",
		VersionMotorCalculo: "motor-v2.1.0", EvidenciaEjecucionRef: "ejecucion-calculo-inicial",
		HuellaEjecucionSHA256: huellaMemoria("4"),
		CalculadoEn:           instanteMemoriaPrueba.Add(-20 * time.Minute),
	}
}

func huellaMemoria(caracter string) string { return strings.Repeat(caracter, 64) }

func hmacMemoria(caracter string) string {
	return "hmac-sha256:memoria_1:" + huellaMemoria(caracter)
}
