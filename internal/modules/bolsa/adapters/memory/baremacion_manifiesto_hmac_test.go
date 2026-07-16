package memory

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type verificadorManifiestoConmutablePrueba struct {
	rechazar atomic.Bool
	motivo   error
}

func (v *verificadorManifiestoConmutablePrueba) VerificarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	if solicitud.Finalidad == puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3 &&
		v.rechazar.Load() {
		return v.motivo
	}
	return (verificadorHMACMemoriaPrueba{}).VerificarSelloBaremacion(ctx, solicitud)
}

type escenarioManifiestoHistoricoPrueba struct {
	repositorio *RepositorioBaremaciones
	reloj       *relojMemoriaPrueba
	base        dominiobolsa.BaremacionMerito
	alta        puertosbolsa.ResultadoConfirmarCambioBaremacion
	confirmar   puertosbolsa.SolicitudConfirmarCambioBaremacion
}

func nuevoEscenarioManifiestoHistoricoPrueba(
	t *testing.T,
	verificador puertosbolsa.VerificadorSellosBaremacion,
) escenarioManifiestoHistoricoPrueba {
	t.Helper()
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	repositorio, err := NuevoRepositorioBaremaciones(
		reloj, verificador, PerfilRepositorioBaremacionesSoloPruebas(),
	)
	if err != nil {
		t.Fatalf("crear repositorio: %v", err)
	}
	base := nuevaBaremacionMemoriaPrueba(t)
	alta := confirmarAltaMemoria(t, repositorio, base)
	reloj.fijar(instanteMemoriaPrueba.Add(15 * time.Minute))
	actualizada := incorporarDecisionMemoriaPrueba(t, base)
	reserva, err := repositorio.ReservarCambio(
		context.Background(), solicitudReservaDecisionMemoria(alta.Version.Referencia),
	)
	if err != nil {
		t.Fatalf("reservar decision: %v", err)
	}
	return escenarioManifiestoHistoricoPrueba{
		repositorio: repositorio,
		reloj:       reloj,
		base:        base,
		alta:        alta,
		confirmar: solicitudConfirmarDecisionMemoria(
			reserva.Token, alta.Version.Referencia, actualizada,
		),
	}
}

func sustituirSelloManifiestoPrueba(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	sello string,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	clon, err := solicitud.Clonar()
	if err != nil {
		t.Fatalf("clonar confirmacion: %v", err)
	}
	manifiesto := clon.Manifiesto.Clonar()
	manifiesto.SelloManifiestoHMACSHA256 = sello
	ultima := &clon.Agregado.Decisiones[len(clon.Agregado.Decisiones)-1]
	firma := ultima.Firma
	firma.SelloManifiestoProbatorioHMACSHA256 = sello
	decision, err := dominiobolsa.ConstituirDecisionFirmada(ultima.Contenido, firma)
	if err != nil {
		t.Fatalf("reconstituir decision: %v", err)
	}
	*ultima = decision
	clon.Manifiesto = &manifiesto
	return sellarConfirmacionMemoria(clon)
}

func sustituirPorManifiestoAlternativoPrueba(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	clon, err := solicitud.Clonar()
	if err != nil {
		t.Fatalf("clonar confirmacion: %v", err)
	}
	base := clon.Manifiesto.Clonar()
	base.Referencia += "-alternativo"
	base.CreadoEn = base.CreadoEn.Add(time.Second)
	base.HuellaManifiestoSHA256 = ""
	base.SelloManifiestoHMACSHA256 = ""
	preparado, representacion, err := base.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto alternativo: %v", err)
	}
	sello := calcularSelloMemoria(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		representacion.Revelar(),
	)
	manifiesto, err := preparado.IncorporarSello(sello)
	if err != nil {
		t.Fatalf("sellar manifiesto alternativo: %v", err)
	}
	ultima := &clon.Agregado.Decisiones[len(clon.Agregado.Decisiones)-1]
	firma := ultima.Firma
	firma.ManifiestoProbatorioRef = manifiesto.Referencia
	firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
	firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
	decision, err := dominiobolsa.ConstituirDecisionFirmada(ultima.Contenido, firma)
	if err != nil {
		t.Fatalf("reconstituir decision alternativa: %v", err)
	}
	*ultima = decision
	clon.Manifiesto = &manifiesto
	return sellarConfirmacionMemoria(clon)
}

func comprobarDecisionSinEfectosPrueba(
	t *testing.T,
	repositorio *RepositorioBaremaciones,
) {
	t.Helper()
	repositorio.mu.RLock()
	defer repositorio.mu.RUnlock()
	activas := 0
	for _, reserva := range repositorio.reservasPorAmbito {
		if reserva.Estado == estadoReservaActiva {
			activas++
		}
	}
	if len(repositorio.versionesPorBaremacion["baremacion-001"]) != 1 ||
		len(repositorio.auditorias) != 1 || len(repositorio.eventosOutbox) != 1 ||
		len(repositorio.manifiestosPorReferencia) != 0 || len(repositorio.manifiestoRefPorVersion) != 0 ||
		activas != 1 {
		t.Fatalf(
			"confirmacion fallida produjo efectos: versiones=%d auditorias=%d eventos=%d manifiestos=%d indices=%d reservas_activas=%d",
			len(repositorio.versionesPorBaremacion["baremacion-001"]), len(repositorio.auditorias),
			len(repositorio.eventosOutbox), len(repositorio.manifiestosPorReferencia),
			len(repositorio.manifiestoRefPorVersion), activas,
		)
	}
}

func TestRepositorioBaremacionesRechazaSelloManifiestoInventadoSinEfectos(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	inventada := sustituirSelloManifiestoPrueba(t, escenario.confirmar, hmacMemoria("f"))
	if _, err := escenario.repositorio.ConfirmarCambio(context.Background(), inventada); !errors.Is(
		err, puertosbolsa.ErrSelloBaremacionNoAutentico,
	) {
		t.Fatalf("sello inventado admitido: %v", err)
	}
	comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
	if _, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar); err != nil {
		t.Fatalf("el ataque consumio la reserva legitima: %v", err)
	}
}

func TestRepositorioBaremacionesFallaCerradoAnteClaveDesconocidaOIndisponibleSinEfectos(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		motivo error
	}{
		{"clave desconocida", errors.New("clave hmac desconocida")},
		{"servicio indisponible", errors.New("servicio hmac indisponible")},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			verificador := &verificadorManifiestoConmutablePrueba{motivo: caso.motivo}
			escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
			verificador.rechazar.Store(true)
			if _, err := escenario.repositorio.ConfirmarCambio(
				context.Background(), escenario.confirmar,
			); !errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
				t.Fatalf("fallo del verificador no cerro la confirmacion: %v", err)
			}
			comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
			verificador.rechazar.Store(false)
			if _, err := escenario.repositorio.ConfirmarCambio(
				context.Background(), escenario.confirmar,
			); err != nil {
				t.Fatalf("el fallo criptografico consumio la reserva: %v", err)
			}
		})
	}
}

func TestRepositorioBaremacionesRechazaIntercambioDeManifiestosSinEfectos(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	alternativa := sustituirPorManifiestoAlternativoPrueba(t, escenario.confirmar)
	intercambiada := alternativa
	manifiestoOriginal := escenario.confirmar.Manifiesto.Clonar()
	intercambiada.Manifiesto = &manifiestoOriginal
	if _, err := escenario.repositorio.ConfirmarCambio(context.Background(), intercambiada); !errors.Is(
		err, puertosbolsa.ErrSolicitudBaremacionInvalida,
	) {
		t.Fatalf("intercambio de manifiestos admitido: %v", err)
	}
	comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
}

func TestRepositorioBaremacionesClonaManifiestoDeEntradaYSalida(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	resultado, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatalf("confirmar decision: %v", err)
	}
	referencia := escenario.confirmar.Manifiesto.Referencia
	escenario.confirmar.Manifiesto.Autorizaciones[0].AutorizacionRef = "entrada-manipulada"
	escenario.confirmar.Manifiesto.Evidencias[0].Referencia = "entrada-manipulada"
	escenario.confirmar.Agregado.Decisiones[0].Contenido.Motivo = "entrada manipulada"
	resultado.Version.Agregado.Decisiones[0].Contenido.Motivo = "salida manipulada"

	vigente, err := escenario.repositorio.ObtenerVersionVigente(
		context.Background(),
		puertosbolsa.SolicitudObtenerBaremacionVigente{
			Contexto: contextoMemoriaPrueba(
				puertosbolsa.AccionConsultarBaremacionVigente, escenario.base.ID, escenario.reloj.Ahora(),
			),
			BaremacionMeritoRef: escenario.base.ID,
		},
	)
	if err != nil || vigente.Agregado.Decisiones[0].Contenido.Motivo == "salida manipulada" {
		t.Fatalf("entrada o salida comparte memoria con el repositorio: %v", err)
	}
	escenario.repositorio.mu.RLock()
	persistido := escenario.repositorio.manifiestosPorReferencia[referencia]
	escenario.repositorio.mu.RUnlock()
	if persistido.Manifiesto.Autorizaciones[0].AutorizacionRef == "entrada-manipulada" ||
		persistido.Manifiesto.Evidencias[0].Referencia == "entrada-manipulada" {
		t.Fatal("el manifiesto persistido comparte slices con la entrada")
	}

	solicitudEvidencia := puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
		Contexto: contextoMemoriaPrueba(
			puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
			resultado.Evidencia.AuditoriaRef, escenario.reloj.Ahora(),
		),
		BaremacionMeritoRef: escenario.base.ID,
		NumeroVersion:       resultado.Version.Referencia.Numero,
		AuditoriaRef:        resultado.Evidencia.AuditoriaRef,
		EventoOutboxRef:     resultado.Evidencia.EventoOutboxRef,
	}
	recuperada, err := escenario.repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitudEvidencia)
	if err != nil || recuperada.Manifiesto == nil || recuperada.Manifiesto.Referencia != referencia {
		t.Fatalf("no se recupero el manifiesto completo: %+v / %v", recuperada.Manifiesto, err)
	}
	recuperada.Manifiesto.Autorizaciones[0].AutorizacionRef = "salida-manifiesto-manipulada"
	recuperadaOtra, err := escenario.repositorio.ObtenerEvidenciaTransaccion(context.Background(), solicitudEvidencia)
	if err != nil || recuperadaOtra.Manifiesto == nil ||
		recuperadaOtra.Manifiesto.Autorizaciones[0].AutorizacionRef == "salida-manifiesto-manipulada" {
		t.Fatalf("el manifiesto recuperado comparte memoria: %+v / %v", recuperadaOtra.Manifiesto, err)
	}
}

func TestRepositorioBaremacionesDetectaMutacionDelMapaHistoricoEnLecturasYReintento(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	resultado, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatalf("confirmar decision: %v", err)
	}
	referencia := escenario.confirmar.Manifiesto.Referencia
	escenario.repositorio.mu.Lock()
	persistido := escenario.repositorio.manifiestosPorReferencia[referencia]
	persistido.Manifiesto.Evidencias[0].HuellaEvidenciaSHA256 = huellaMemoria("e")
	escenario.repositorio.manifiestosPorReferencia[referencia] = persistido
	escenario.repositorio.mu.Unlock()

	operaciones := operacionesLecturaManifiestoPrueba(escenario, resultado)
	for nombre, operacion := range operaciones {
		t.Run(nombre, func(t *testing.T) {
			if err := operacion(); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
				t.Fatalf("mapa historico manipulado no fue detectado: %v", err)
			}
		})
	}
	if _, err := escenario.repositorio.ConfirmarCambio(
		context.Background(), escenario.confirmar,
	); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("reintento sobre mapa manipulado admitido: %v", err)
	}
	comprobarLongitudesInternas(t, escenario.repositorio, 2, 2, 2)
}

func TestRepositorioBaremacionesReverificaHMACHistoricaEnLecturasYReintento(t *testing.T) {
	verificador := &verificadorManifiestoConmutablePrueba{motivo: errors.New("llavero no disponible")}
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
	resultado, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatalf("confirmar decision: %v", err)
	}
	verificador.rechazar.Store(true)

	// El alta V1 no tenia manifiesto y sigue siendo legible; una version que ya
	// contiene decisiones se cierra si su HMAC historica no puede verificarse.
	if _, err := escenario.repositorio.ObtenerVersion(
		context.Background(),
		puertosbolsa.SolicitudObtenerVersionBaremacion{
			Contexto: contextoMemoriaPrueba(
				puertosbolsa.AccionConsultarVersionBaremacion, escenario.base.ID, escenario.reloj.Ahora(),
			),
			BaremacionMeritoRef: escenario.base.ID,
			Numero:              1,
		},
	); err != nil {
		t.Fatalf("alta V1 sin manifiesto dejo de ser legible: %v", err)
	}
	for nombre, operacion := range operacionesLecturaManifiestoPrueba(escenario, resultado) {
		t.Run(nombre, func(t *testing.T) {
			if err := operacion(); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
				t.Fatalf("lectura sin verificacion HMAC cerrada: %v", err)
			}
		})
	}
	if _, err := escenario.repositorio.ConfirmarCambio(
		context.Background(), escenario.confirmar,
	); !errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
		t.Fatalf("reintento sin verificacion HMAC admitido: %v", err)
	}
	comprobarLongitudesInternas(t, escenario.repositorio, 2, 2, 2)
}

func operacionesLecturaManifiestoPrueba(
	escenario escenarioManifiestoHistoricoPrueba,
	resultado puertosbolsa.ResultadoConfirmarCambioBaremacion,
) map[string]func() error {
	return map[string]func() error{
		"vigente": func() error {
			_, err := escenario.repositorio.ObtenerVersionVigente(
				context.Background(),
				puertosbolsa.SolicitudObtenerBaremacionVigente{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarBaremacionVigente, escenario.base.ID, escenario.reloj.Ahora(),
					),
					BaremacionMeritoRef: escenario.base.ID,
				},
			)
			return err
		},
		"version": func() error {
			_, err := escenario.repositorio.ObtenerVersion(
				context.Background(),
				puertosbolsa.SolicitudObtenerVersionBaremacion{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarVersionBaremacion, escenario.base.ID, escenario.reloj.Ahora(),
					),
					BaremacionMeritoRef: escenario.base.ID,
					Numero:              resultado.Version.Referencia.Numero,
				},
			)
			return err
		},
		"evidencia": func() error {
			_, err := escenario.repositorio.ObtenerEvidenciaTransaccion(
				context.Background(),
				puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
						resultado.Evidencia.AuditoriaRef, escenario.reloj.Ahora(),
					),
					BaremacionMeritoRef: escenario.base.ID,
					NumeroVersion:       resultado.Version.Referencia.Numero,
					AuditoriaRef:        resultado.Evidencia.AuditoriaRef,
					EventoOutboxRef:     resultado.Evidencia.EventoOutboxRef,
				},
			)
			return err
		},
	}
}
