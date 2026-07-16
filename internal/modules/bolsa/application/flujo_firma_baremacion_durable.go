package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	duracionArrendamientoFirmaBaremacionPredeterminada = 30 * time.Second
	duracionRecuperacionFirmaBaremacion                = 5 * time.Second
)

type OpcionesFachadaFirmaBaremacionDurable struct {
	DuracionArrendamiento time.Duration
}

// FachadaFirmaBaremacionDurable separa el ciclo HTTP de las capacidades en
// memoria del motor. El repositorio conserva una saga sellada y el protector
// cifra el estado de trabajo; cada llamada vuelve a derivar la identidad desde
// la sesion autoritativa.
//
// Estado: infraestructura interna sin cableado productivo. No acredita
// persistencia tras reinicio, KMS/HSM ni ejecutores reales; esas garantias
// dependen de los adaptadores fijados por la composicion homologada.
type FachadaFirmaBaremacionDurable struct {
	repositorio           puertosbolsa.RepositorioFlujosFirmaBaremacion
	protector             puertosbolsa.ProtectorEstadoFlujoFirmaBaremacion
	ejecutor              puertosbolsa.EjecutorPasosFirmaBaremacion
	generador             puertosbolsa.GeneradorReferenciasFlujoFirmaBaremacion
	sellador              puertosbolsa.SelladorSolicitudBaremacion
	verificador           puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion
	sesiones              FuenteSesionAutenticadaBaremacion
	reloj                 puertosbolsa.Reloj
	duracionArrendamiento time.Duration
}

func NuevaFachadaFirmaBaremacionDurable(
	repositorio puertosbolsa.RepositorioFlujosFirmaBaremacion,
	protector puertosbolsa.ProtectorEstadoFlujoFirmaBaremacion,
	ejecutor puertosbolsa.EjecutorPasosFirmaBaremacion,
	generador puertosbolsa.GeneradorReferenciasFlujoFirmaBaremacion,
	sellador puertosbolsa.SelladorSolicitudBaremacion,
	verificador puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion,
	sesiones FuenteSesionAutenticadaBaremacion,
	reloj puertosbolsa.Reloj,
	opciones OpcionesFachadaFirmaBaremacionDurable,
) (*FachadaFirmaBaremacionDurable, error) {
	if dependenciaBaremacionNula(repositorio) || dependenciaBaremacionNula(protector) ||
		dependenciaBaremacionNula(ejecutor) || dependenciaBaremacionNula(generador) ||
		dependenciaBaremacionNula(sellador) || dependenciaBaremacionNula(verificador) ||
		dependenciaBaremacionNula(sesiones) ||
		dependenciaBaremacionNula(reloj) {
		return nil, ErrDependenciaBaremacionRequerida
	}
	duracion := opciones.DuracionArrendamiento
	if duracion == 0 {
		duracion = duracionArrendamientoFirmaBaremacionPredeterminada
	}
	if duracion < time.Second || duracion > puertosbolsa.DuracionMaximaArrendamientoFlujoFirma {
		return nil, ErrDependenciaBaremacionRequerida
	}
	return &FachadaFirmaBaremacionDurable{
		repositorio: repositorio, protector: protector, ejecutor: ejecutor,
		generador: generador, sellador: sellador, verificador: verificador,
		sesiones: sesiones, reloj: reloj,
		duracionArrendamiento: duracion,
	}, nil
}

// OrdenPrepararFlujoFirmaBaremacion solo admite referencias de negocio y un
// estado de trabajo interno sin capacidades. Un adaptador HTTP no debe aceptar
// EstadoTrabajoInicial del cliente: lo construye el caso de uso tecnico que ha
// fijado la decision y sus huellas. La entrada productiva permanece cerrada
// hasta imponer esta restriccion por construccion y probarla en arquitectura.
type OrdenPrepararFlujoFirmaBaremacion struct {
	ClaveIdempotencia    string
	ProcesoRef           string
	SolicitudRef         string
	BaremacionMeritoRef  string
	DecisionRef          string
	EstadoTrabajoInicial puertosbolsa.CargaProtegida
}

type OrdenReanudarFlujoFirmaBaremacion struct {
	FlujoRef          string
	ClaveIdempotencia string
}

type ProyeccionEstadoFlujoFirmaBaremacion struct {
	FlujoRef      string
	Estado        puertosbolsa.EstadoExpedienteFlujoFirmaBaremacion
	Lanzamiento   *puertosbolsa.ProyeccionLanzamientoFirmaBaremacion
	Resultado     *puertosbolsa.ResultadoFinalFlujoFirmaBaremacion
	ActualizadoEn time.Time
}

type credencialesFlujoFirmaBaremacion struct {
	indiceIdempotenciaHMAC string
	vinculoActorHMAC       string
	perfilActorClave       string
}

func (f *FachadaFirmaBaremacionDurable) Preparar(
	ctx context.Context,
	orden OrdenPrepararFlujoFirmaBaremacion,
) (puertosbolsa.ProyeccionLanzamientoFirmaBaremacion, error) {
	if f == nil || ctx == nil || ctx.Err() != nil ||
		!referenciaAplicacionBaremacionValida(orden.ClaveIdempotencia) ||
		!referenciaAplicacionBaremacionValida(orden.ProcesoRef) ||
		!referenciaAplicacionBaremacionValida(orden.SolicitudRef) ||
		!referenciaAplicacionBaremacionValida(orden.BaremacionMeritoRef) ||
		!referenciaAplicacionBaremacionValida(orden.DecisionRef) ||
		orden.EstadoTrabajoInicial.Validar() != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	credenciales, err := f.derivarCredenciales(ctx, orden.ClaveIdempotencia)
	if err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	huellaEstado := sha256.Sum256(orden.EstadoTrabajoInicial.Revelar())
	huellaSolicitud, err := f.sellarPartes(ctx, []string{
		"solicitud_preparar_flujo_firma_baremacion_v1", credenciales.indiceIdempotenciaHMAC,
		orden.ProcesoRef, orden.SolicitudRef, orden.BaremacionMeritoRef, orden.DecisionRef,
		hex.EncodeToString(huellaEstado[:]),
	})
	if err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	estadoProtegido, err := f.protector.ProtegerEstadoFlujoFirmaBaremacion(ctx, orden.EstadoTrabajoInicial)
	if err != nil || estadoProtegido.Validar() != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, errors.Join(puertosbolsa.ErrEstadoFlujoFirmaAlterado, err)
	}
	flujoRef, err := f.generador.NuevaReferenciaFlujoFirmaBaremacion()
	if err != nil || !referenciaAplicacionBaremacionValida(flujoRef) {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	ahora, err := f.ahora()
	if err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	expediente, err := f.sellarExpediente(ctx, puertosbolsa.ExpedienteFlujoFirmaBaremacion{
		FlujoRef: flujoRef, Version: 1,
		IndiceIdempotenciaHMAC: credenciales.indiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    huellaSolicitud, VinculoActorHMAC: credenciales.vinculoActorHMAC,
		PerfilActorClave: credenciales.perfilActorClave,
		ProcesoRef:       orden.ProcesoRef, SolicitudRef: orden.SolicitudRef,
		BaremacionMeritoRef: orden.BaremacionMeritoRef, DecisionRef: orden.DecisionRef,
		Estado: puertosbolsa.EstadoExpedienteFirmaPreparando, EstadoProtegido: estadoProtegido,
		PuntosControl: make([]puertosbolsa.PuntoControlFirmaBaremacion, 0, len(puertosbolsa.PasosFlujoFirmaBaremacion())),
		CreadoEn:      ahora, ActualizadoEn: ahora,
	})
	if err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	alta, err := f.repositorio.CrearORecuperarFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{Expediente: expediente},
	)
	if err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	if err := f.verificarExpediente(ctx, alta.Expediente); err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	if alta.Expediente.IndiceIdempotenciaHMAC != credenciales.indiceIdempotenciaHMAC ||
		alta.Expediente.VinculoActorHMAC != credenciales.vinculoActorHMAC ||
		alta.Expediente.HuellaSolicitudHMAC != huellaSolicitud {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, puertosbolsa.ErrClaveFlujoFirmaBaremacionReutilizada
	}
	consulta := solicitudConsultaFlujoFirma(alta.Expediente.FlujoRef, credenciales)
	expediente, err = f.ejecutarPaso(ctx, consulta, puertosbolsa.PasoPrepararFirmaBaremacion)
	if err != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, err
	}
	if expediente.ProyeccionLanzamiento == nil || expediente.ProyeccionLanzamiento.Validar() != nil {
		return puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{}, ErrResultadoBaremacionNoConfiable
	}
	return *expediente.ProyeccionLanzamiento, nil
}

func (f *FachadaFirmaBaremacionDurable) Finalizar(
	ctx context.Context,
	orden OrdenReanudarFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoFinalFlujoFirmaBaremacion, error) {
	consulta, err := f.prepararConsulta(ctx, orden)
	if err != nil {
		return puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{}, err
	}
	expediente, err := f.repositorio.ObtenerFlujoFirmaBaremacion(ctx, consulta)
	if err != nil {
		return puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{}, err
	}
	if err := f.verificarExpediente(ctx, expediente); err != nil {
		return puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{}, err
	}
	if expediente.Estado == puertosbolsa.EstadoExpedienteFirmaCompletado {
		return resultadoFinalExacto(expediente)
	}
	for _, paso := range []puertosbolsa.PasoFlujoFirmaBaremacion{
		puertosbolsa.PasoCompletarFirmaBaremacion,
		puertosbolsa.PasoCustodiarFirmaBaremacion,
		puertosbolsa.PasoRetenerFirmaBaremacion,
		puertosbolsa.PasoReservarFirmaBaremacion,
		puertosbolsa.PasoConfirmarFirmaBaremacion,
	} {
		expediente, err = f.ejecutarPaso(ctx, consulta, paso)
		if err != nil {
			return puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{}, err
		}
	}
	return resultadoFinalExacto(expediente)
}

func (f *FachadaFirmaBaremacionDurable) Consultar(
	ctx context.Context,
	orden OrdenReanudarFlujoFirmaBaremacion,
) (ProyeccionEstadoFlujoFirmaBaremacion, error) {
	consulta, err := f.prepararConsulta(ctx, orden)
	if err != nil {
		return ProyeccionEstadoFlujoFirmaBaremacion{}, err
	}
	expediente, err := f.repositorio.ObtenerFlujoFirmaBaremacion(ctx, consulta)
	if err != nil {
		return ProyeccionEstadoFlujoFirmaBaremacion{}, err
	}
	if err := f.verificarExpediente(ctx, expediente); err != nil {
		return ProyeccionEstadoFlujoFirmaBaremacion{}, err
	}
	resultado := ProyeccionEstadoFlujoFirmaBaremacion{
		FlujoRef: expediente.FlujoRef, Estado: expediente.Estado, ActualizadoEn: expediente.ActualizadoEn,
	}
	if expediente.ProyeccionLanzamiento != nil {
		proyeccion := *expediente.ProyeccionLanzamiento
		resultado.Lanzamiento = &proyeccion
	}
	if expediente.Resultado != nil {
		final := *expediente.Resultado
		resultado.Resultado = &final
	}
	return resultado, nil
}

func (f *FachadaFirmaBaremacionDurable) prepararConsulta(
	ctx context.Context,
	orden OrdenReanudarFlujoFirmaBaremacion,
) (puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion, error) {
	if f == nil || ctx == nil || ctx.Err() != nil ||
		!referenciaAplicacionBaremacionValida(orden.FlujoRef) ||
		!referenciaAplicacionBaremacionValida(orden.ClaveIdempotencia) {
		return puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	credenciales, err := f.derivarCredenciales(ctx, orden.ClaveIdempotencia)
	if err != nil {
		return puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{}, err
	}
	return solicitudConsultaFlujoFirma(orden.FlujoRef, credenciales), nil
}

func solicitudConsultaFlujoFirma(
	flujoRef string,
	credenciales credencialesFlujoFirmaBaremacion,
) puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion {
	return puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
		FlujoRef: flujoRef, IndiceIdempotenciaHMAC: credenciales.indiceIdempotenciaHMAC,
		VinculoActorHMAC: credenciales.vinculoActorHMAC,
	}
}

func (f *FachadaFirmaBaremacionDurable) ejecutarPaso(
	ctx context.Context,
	consulta puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
	paso puertosbolsa.PasoFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if consulta.Validar() != nil || !paso.Valido() {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	for intento := 0; intento < 4; intento++ {
		expediente, err := f.repositorio.ObtenerFlujoFirmaBaremacion(ctx, consulta)
		if err != nil {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
		if err := f.verificarExpediente(ctx, expediente); err != nil {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
		if punto, existe := puntoControlPaso(expediente, paso); existe && punto.Estado == puertosbolsa.EstadoPuntoControlFirmaCompletado {
			return expediente, nil
		}
		propietarioRef, err := f.generador.NuevaReferenciaPropietarioArrendamientoFirmaBaremacion()
		if err != nil || !referenciaAplicacionBaremacionValida(propietarioRef) {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
		adquirido, err := f.repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
				Consulta: consulta, VersionEsperada: expediente.Version,
				PropietarioRef: propietarioRef, Duracion: f.duracionArrendamiento,
			},
		)
		if errors.Is(err, puertosbolsa.ErrConflictoFlujoFirmaBaremacion) {
			continue
		}
		if err != nil {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
		expediente = adquirido.Expediente
		arrendamiento := adquirido.Arrendamiento
		errorExpediente := f.verificarExpediente(ctx, expediente)
		errorArrendamiento := arrendamiento.Validar()
		if errorExpediente != nil || errorArrendamiento != nil ||
			arrendamiento.FlujoRef != expediente.FlujoRef {
			if errorArrendamiento == nil && arrendamiento.FlujoRef == consulta.FlujoRef {
				f.liberarArrendamiento(ctx, arrendamiento)
			}
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, errors.Join(
				puertosbolsa.ErrEstadoFlujoFirmaAlterado,
				errorExpediente,
				errorArrendamiento,
			)
		}
		liberar := func() {
			f.liberarArrendamiento(ctx, arrendamiento)
		}
		punto, existe := puntoControlPaso(expediente, paso)
		if existe && punto.Estado == puertosbolsa.EstadoPuntoControlFirmaCompletado {
			liberar()
			return expediente, nil
		}
		if !existe {
			expediente, err = f.declararPaso(ctx, expediente, arrendamiento, paso)
			if err != nil {
				liberar()
				if recuperado, correcto := f.recuperarPasoPersistido(ctx, consulta, paso, "", ""); correcto {
					if recuperadoPunto, encontrado := puntoControlPaso(recuperado, paso); encontrado &&
						recuperadoPunto.Estado == puertosbolsa.EstadoPuntoControlFirmaCompletado {
						return recuperado, nil
					}
					continue
				}
				return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
			}
			punto, _ = puntoControlPaso(expediente, paso)
		}
		estadoTrabajo, err := f.protector.DesprotegerEstadoFlujoFirmaBaremacion(ctx, expediente.EstadoProtegido)
		if err != nil || estadoTrabajo.Validar() != nil {
			liberar()
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, errors.Join(puertosbolsa.ErrEstadoFlujoFirmaAlterado, err)
		}
		previos := append([]puertosbolsa.PuntoControlFirmaBaremacion(nil), expediente.PuntosControl[:len(expediente.PuntosControl)-1]...)
		solicitud := puertosbolsa.SolicitudEjecutarPasoFirmaBaremacion{
			FlujoRef: expediente.FlujoRef, Paso: paso, EfectoRef: punto.EfectoRef,
			ClaveIdempotenciaHMAC: punto.ClaveIdempotenciaHMAC,
			VinculoActorHMAC:      expediente.VinculoActorHMAC, PerfilActorClave: expediente.PerfilActorClave,
			ProcesoRef: expediente.ProcesoRef, SolicitudRef: expediente.SolicitudRef,
			BaremacionMeritoRef: expediente.BaremacionMeritoRef, DecisionRef: expediente.DecisionRef,
			EstadoTrabajo: estadoTrabajo, PuntosPrevios: previos,
		}
		resultado, err := f.ejecutor.EjecutarPasoFirmaBaremacion(ctx, solicitud)
		if err != nil {
			liberar()
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
		if resultado.ValidarPara(solicitud) != nil || resultado.EjecutadoEn.Before(punto.DeclaradoEn) {
			liberar()
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, ErrResultadoBaremacionNoConfiable
		}
		completado, err := f.completarPaso(ctx, expediente, arrendamiento, resultado)
		liberar()
		if err == nil {
			return completado, nil
		}
		if recuperado, correcto := f.recuperarPasoPersistido(
			ctx, consulta, paso, resultado.ResultadoRef, resultado.HuellaResultadoSHA256,
		); correcto {
			return recuperado, nil
		}
		if errors.Is(err, puertosbolsa.ErrConflictoFlujoFirmaBaremacion) {
			continue
		}
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrConflictoFlujoFirmaBaremacion
}

func (f *FachadaFirmaBaremacionDurable) declararPaso(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
	arrendamiento puertosbolsa.ArrendamientoFlujoFirmaBaremacion,
	paso puertosbolsa.PasoFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	pasos := puertosbolsa.PasosFlujoFirmaBaremacion()
	if len(expediente.PuntosControl) >= len(pasos) || pasos[len(expediente.PuntosControl)] != paso {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrPasoFlujoFirmaNoPermitido
	}
	efectoRef, err := f.generador.NuevaReferenciaEfectoFirmaBaremacion(paso)
	if err != nil || !referenciaAplicacionBaremacionValida(efectoRef) {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	claveEfecto, err := f.sellarPartes(ctx, []string{
		"idempotencia_efecto_flujo_firma_baremacion_v1", expediente.FlujoRef, string(paso),
		efectoRef, expediente.HuellaSolicitudHMAC,
	})
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	ahora, err := f.ahora()
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	siguiente, err := expediente.Clonar()
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	siguiente.Version = expediente.Version + 1
	siguiente.ActualizadoEn = ahora
	siguiente.PuntosControl = append(siguiente.PuntosControl, puertosbolsa.PuntoControlFirmaBaremacion{
		Paso: paso, Estado: puertosbolsa.EstadoPuntoControlFirmaDeclarado,
		EfectoRef: efectoRef, ClaveIdempotenciaHMAC: claveEfecto, DeclaradoEn: ahora,
	})
	if paso != puertosbolsa.PasoPrepararFirmaBaremacion {
		siguiente.Estado = puertosbolsa.EstadoExpedienteFirmaFinalizando
	}
	siguiente, err = f.sellarExpediente(ctx, sinSelloExpediente(siguiente))
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	return f.guardarExpedienteVerificado(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: expediente.Version, Arrendamiento: arrendamiento, Siguiente: siguiente,
	})
}

func (f *FachadaFirmaBaremacionDurable) completarPaso(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
	arrendamiento puertosbolsa.ArrendamientoFlujoFirmaBaremacion,
	resultado puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if len(expediente.PuntosControl) == 0 {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrPasoFlujoFirmaNoPermitido
	}
	ultimo := expediente.PuntosControl[len(expediente.PuntosControl)-1]
	if ultimo.Estado != puertosbolsa.EstadoPuntoControlFirmaDeclarado || ultimo.Paso != resultado.Paso ||
		ultimo.EfectoRef != resultado.EfectoRef {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrPasoFlujoFirmaNoPermitido
	}
	estadoProtegido, err := f.protector.ProtegerEstadoFlujoFirmaBaremacion(ctx, resultado.EstadoTrabajo)
	if err != nil || estadoProtegido.Validar() != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, errors.Join(puertosbolsa.ErrEstadoFlujoFirmaAlterado, err)
	}
	ahora, err := f.ahora()
	if err != nil || ahora.Before(resultado.EjecutadoEn) {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	siguiente, err := expediente.Clonar()
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	siguiente.Version = expediente.Version + 1
	siguiente.ActualizadoEn = ahora
	siguiente.EstadoProtegido = estadoProtegido
	indice := len(siguiente.PuntosControl) - 1
	siguiente.PuntosControl[indice].Estado = puertosbolsa.EstadoPuntoControlFirmaCompletado
	siguiente.PuntosControl[indice].ResultadoRef = resultado.ResultadoRef
	siguiente.PuntosControl[indice].HuellaResultadoSHA256 = resultado.HuellaResultadoSHA256
	siguiente.PuntosControl[indice].CompletadoEn = resultado.EjecutadoEn.UTC()
	switch resultado.Paso {
	case puertosbolsa.PasoPrepararFirmaBaremacion:
		proyeccion := *resultado.ProyeccionLanzamiento
		siguiente.ProyeccionLanzamiento = &proyeccion
		siguiente.Estado = puertosbolsa.EstadoExpedienteFirmaPendienteInteraccion
	case puertosbolsa.PasoConfirmarFirmaBaremacion:
		final := *resultado.ResultadoFinal
		siguiente.Resultado = &final
		siguiente.Estado = puertosbolsa.EstadoExpedienteFirmaCompletado
	default:
		siguiente.Estado = puertosbolsa.EstadoExpedienteFirmaFinalizando
	}
	siguiente, err = f.sellarExpediente(ctx, sinSelloExpediente(siguiente))
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	return f.guardarExpedienteVerificado(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: expediente.Version, Arrendamiento: arrendamiento, Siguiente: siguiente,
	})
}

func (f *FachadaFirmaBaremacionDurable) recuperarPasoPersistido(
	ctx context.Context,
	consulta puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
	paso puertosbolsa.PasoFlujoFirmaBaremacion,
	resultadoRef, huellaResultado string,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, bool) {
	ctxRecuperacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), duracionRecuperacionFirmaBaremacion)
	defer cancelar()
	expediente, err := f.repositorio.ObtenerFlujoFirmaBaremacion(ctxRecuperacion, consulta)
	if err != nil || f.verificarExpediente(ctxRecuperacion, expediente) != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, false
	}
	punto, existe := puntoControlPaso(expediente, paso)
	if !existe {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, false
	}
	if resultadoRef == "" {
		return expediente, true
	}
	if punto.Estado != puertosbolsa.EstadoPuntoControlFirmaCompletado ||
		punto.ResultadoRef != resultadoRef || punto.HuellaResultadoSHA256 != huellaResultado {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, false
	}
	return expediente, true
}

func (f *FachadaFirmaBaremacionDurable) liberarArrendamiento(
	ctx context.Context,
	arrendamiento puertosbolsa.ArrendamientoFlujoFirmaBaremacion,
) {
	ctxLiberacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), duracionRecuperacionFirmaBaremacion)
	defer cancelar()
	_ = f.repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
		ctxLiberacion,
		puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{Arrendamiento: arrendamiento},
	)
}

func puntoControlPaso(
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
	paso puertosbolsa.PasoFlujoFirmaBaremacion,
) (puertosbolsa.PuntoControlFirmaBaremacion, bool) {
	for _, punto := range expediente.PuntosControl {
		if punto.Paso == paso {
			return punto, true
		}
	}
	return puertosbolsa.PuntoControlFirmaBaremacion{}, false
}

func resultadoFinalExacto(
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoFinalFlujoFirmaBaremacion, error) {
	if expediente.Validar() != nil || expediente.Estado != puertosbolsa.EstadoExpedienteFirmaCompletado ||
		expediente.Resultado == nil || expediente.Resultado.Validar() != nil {
		return puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{}, ErrResultadoBaremacionNoConfiable
	}
	return *expediente.Resultado, nil
}

func sinSelloExpediente(
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	expediente.SelloEstadoHMAC = ""
	return expediente
}

func (f *FachadaFirmaBaremacionDurable) sellarExpediente(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	preparado, carga, err := expediente.PrepararSellado()
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	sello, err := f.sellador.SellarSolicitudBaremacion(ctx, carga)
	if err != nil || !selloGeneradoBaremacionValido(sello) {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	return preparado.IncorporarSello(sello)
}

func (f *FachadaFirmaBaremacionDurable) verificarExpediente(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) error {
	if f == nil || ctx == nil || ctx.Err() != nil || expediente.Validar() != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	canonica, err := puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		return errors.Join(puertosbolsa.ErrEstadoFlujoFirmaAlterado, err)
	}
	solicitud := puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion{
		RepresentacionCanonica: canonica,
		SelloHMAC:              expediente.SelloEstadoHMAC,
	}
	if err := f.verificador.VerificarEstadoFlujoFirmaBaremacion(ctx, solicitud); err != nil {
		return errors.Join(puertosbolsa.ErrEstadoFlujoFirmaAlterado, err)
	}
	return nil
}

func (f *FachadaFirmaBaremacionDurable) guardarExpedienteVerificado(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	guardado, err := f.repositorio.GuardarFlujoFirmaBaremacion(ctx, solicitud)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	if err := f.verificarExpediente(ctx, guardado); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	return guardado, nil
}

func (f *FachadaFirmaBaremacionDurable) derivarCredenciales(
	ctx context.Context,
	claveIdempotencia string,
) (credencialesFlujoFirmaBaremacion, error) {
	if ctx == nil || ctx.Err() != nil || !referenciaAplicacionBaremacionValida(claveIdempotencia) {
		return credencialesFlujoFirmaBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida)
	}
	sesiones, err := f.sesiones.BuscarSesionesAutenticadasBaremacion(ctx)
	if err != nil || len(sesiones) != 1 {
		return credencialesFlujoFirmaBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	contextoActor, vinculo, err := sesiones[0].capacidades()
	ahora, errorReloj := f.ahora()
	if err != nil || errorReloj != nil || contextoActor.Validar() != nil ||
		!vinculo.VigenteEn(ahora, contextoActor) {
		return credencialesFlujoFirmaBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err, errorReloj)
	}
	vinculoHMAC, err := f.sellarPartes(ctx, []string{
		"vinculo_actor_flujo_firma_baremacion_v1", contextoActor.Principal.ID, contextoActor.PerfilActivoRef,
	})
	if err != nil {
		return credencialesFlujoFirmaBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	indiceHMAC, err := f.sellarPartes(ctx, []string{
		"indice_idempotencia_flujo_firma_baremacion_v1", vinculoHMAC, claveIdempotencia,
	})
	if err != nil {
		return credencialesFlujoFirmaBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	return credencialesFlujoFirmaBaremacion{
		indiceIdempotenciaHMAC: indiceHMAC, vinculoActorHMAC: vinculoHMAC,
		perfilActorClave: contextoActor.PerfilActivoRef,
	}, nil
}

func (f *FachadaFirmaBaremacionDurable) sellarPartes(ctx context.Context, partes []string) (string, error) {
	if ctx == nil || ctx.Err() != nil || len(partes) == 0 || len(partes) > 64 {
		return "", puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	var canonica bytes.Buffer
	for _, parte := range partes {
		if parte == "" {
			return "", puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
		}
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(parte)))
		_, _ = canonica.Write(longitud[:])
		_, _ = canonica.WriteString(parte)
	}
	carga, err := puertosbolsa.NuevaCargaProtegida(canonica.Bytes())
	if err != nil {
		return "", err
	}
	sello, err := f.sellador.SellarSolicitudBaremacion(ctx, carga)
	if err != nil || !selloGeneradoBaremacionValido(sello) {
		return "", errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	return sello, nil
}

func (f *FachadaFirmaBaremacionDurable) ahora() (time.Time, error) {
	if f == nil || dependenciaBaremacionNula(f.reloj) {
		return time.Time{}, ErrDependenciaBaremacionRequerida
	}
	ahora := f.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return time.Time{}, ErrResultadoBaremacionNoConfiable
	}
	return ahora, nil
}
