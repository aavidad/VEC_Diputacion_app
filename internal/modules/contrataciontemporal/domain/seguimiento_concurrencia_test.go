package domain

import (
	"errors"
	"sync"
	"testing"
)

func TestSeguimientoCASIdempotenciaYColisionSemantica(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	base := seguimientoNuevoValido(t, definicion)
	datos := datosSeguimiento("acto_incorporacion_cas_01", "confirmar_incorporacion", 1)
	datos.MotivoClave = "necesidad_servicio"
	datos.Periodo = punteroIntervalo(base.Estado().PeriodoPrevisto)
	datos.EfectivoEn = datos.Periodo.Desde
	datos.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_incorporacion", Referencia: referenciaSeguimientoPrueba("documento_incorporacion_cas_01")},
	}

	if _, err := base.Aplicar(definicion, 99, datos); !errors.Is(
		err,
		ErrVersionEnConflicto,
	) {
		t.Fatalf("CAS incorrecto no produjo conflicto: %v", err)
	}
	if base.Version() != 0 || len(base.Actuaciones()) != 0 {
		t.Fatal("el conflicto CAS mutó el agregado")
	}

	aplicado, err := base.Aplicar(definicion, 0, datos)
	if err != nil {
		t.Fatalf("aplicar actuación inicial: %v", err)
	}
	repetido, err := aplicado.Aplicar(definicion, 0, datos)
	if err != nil {
		t.Fatalf("repetición exacta no fue idempotente: %v", err)
	}
	if repetido.Version() != 1 || len(repetido.Actuaciones()) != 1 ||
		repetido.Actuaciones()[0].HuellaActuacionSHA256 !=
			aplicado.Actuaciones()[0].HuellaActuacionSHA256 {
		t.Fatal("la repetición exacta creó otra actuación")
	}

	colision := datos
	colision.ReciboRef = referenciaSeguimientoPrueba("recibo_con_otro_contenido_01")
	if _, err := aplicado.Aplicar(definicion, 1, colision); !errors.Is(
		err,
		ErrActuacionSeguimientoEnConflicto,
	) {
		t.Fatalf("la referencia reutilizada no produjo conflicto semántico: %v", err)
	}
	if aplicado.Version() != 1 || len(aplicado.Actuaciones()) != 1 {
		t.Fatal("la colisión semántica mutó el agregado")
	}
}

func TestSeguimientoEntregaCopiasDefensivasCompletas(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	seguimiento := seguimientoCompletoParaRehidratar(t, definicion)
	huellaDefinicion := definicion.Referencia().HuellaSHA256
	huellaActuacion := seguimiento.Actuaciones()[1].HuellaActuacionSHA256

	publicacion := definicion.Publicacion()
	publicacion.Estados[0].Clave = "estado_adulterado"
	publicacion.Motivos[0] = "motivo_adulterado"
	publicacion.Transiciones[0].Documentos[0].TipoClave = "documento_adulterado"
	for indice := range publicacion.Transiciones {
		if publicacion.Transiciones[indice].Calendario != nil {
			publicacion.Transiciones[indice].Calendario.AmbitosPermitidos[0] =
				"ambito_adulterado"
			break
		}
	}
	if definicion.Referencia().HuellaSHA256 != huellaDefinicion ||
		definicion.Validar() != nil {
		t.Fatal("una copia de la publicación mutó la definición")
	}

	estado := seguimiento.Estado()
	estado.PeriodosResultantes[0].Intervalo.Hasta =
		estado.PeriodosResultantes[0].Intervalo.Desde
	estado.CeseEfectivo.ActuacionRef = "cese_adulterado"
	estado.Actuaciones[1].Documentos[0].Referencia = "documento_adulterado"
	estado.Actuaciones[1].Periodo.Hasta =
		estado.Actuaciones[1].Periodo.Desde
	estado.Actuaciones[1].Calendario.HuellaSHA256 = "huella_adulterada"
	if seguimiento.Actuaciones()[1].HuellaActuacionSHA256 != huellaActuacion ||
		seguimiento.Validar(definicion) != nil {
		t.Fatal("una copia del estado mutó el agregado")
	}

	actuaciones := seguimiento.Actuaciones()
	actuaciones[1].Documentos[0].Referencia = "otro_documento_adulterado"
	actuaciones[1].Periodo.Hasta = actuaciones[1].Periodo.Desde
	actuaciones[1].Calendario.Referencia = "otro_calendario_adulterado"
	periodos := seguimiento.PeriodosResultantes()
	periodos[0].ActuacionRef = "otra_actuacion_adulterada"
	cese := seguimiento.CeseEfectivo()
	cese.ActuacionRef = "otro_cese_adulterado"
	if seguimiento.Validar(definicion) != nil {
		t.Fatal("un accesor mutó el agregado")
	}
}

func TestSeguimientoPuedeConsultarseYDerivarseConcurrentemente(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	base := seguimientoIncorporado(t, definicion)
	const trabajadores = 64
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	grupo.Add(trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		indice := indice
		go func() {
			defer grupo.Done()
			_ = base.Estado()
			_ = base.Actuaciones()
			_ = base.PeriodosResultantes()
			datos := datosSeguimiento(
				"acto_incidencia_concurrente_"+cadenaDecimalSeguimiento(indice),
				"registrar_incidencia",
				2,
			)
			datos.MotivoClave = "incidencia_catalogada"
			datos.Documentos = []DocumentoSeguimiento{{
				TipoClave: "parte_incidencia",
				Referencia: referenciaSeguimientoPrueba(
					"documento_incidencia_concurrente_" +
						cadenaDecimalSeguimiento(indice),
				),
			}}
			resultado, err := base.Aplicar(definicion, 1, datos)
			if err != nil {
				errores <- err
				return
			}
			if resultado.Version() != 2 || len(resultado.Actuaciones()) != 2 {
				errores <- ErrSeguimientoInvalido
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("derivación concurrente: %v", err)
	}
	if base.Version() != 1 || len(base.Actuaciones()) != 1 {
		t.Fatal("las derivaciones concurrentes mutaron la base compartida")
	}
}
