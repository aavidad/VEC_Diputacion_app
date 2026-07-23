package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRehidratacionSeguimientoReproduceEstadoVersionPeriodosYHuellas(
	t *testing.T,
) {
	definicion := definicionSeguimientoValida(t, false)
	original := seguimientoCompletoParaRehidratar(t, definicion)
	estado := original.Estado()

	codificado, err := json.Marshal(estado)
	if err != nil {
		t.Fatalf("serializar estado: %v", err)
	}
	var persistido EstadoPersistidoSeguimiento
	if err := json.Unmarshal(codificado, &persistido); err != nil {
		t.Fatalf("decodificar estado: %v", err)
	}
	rehidratado, err := RehidratarSeguimiento(definicion, persistido)
	if err != nil {
		t.Fatalf("rehidratar estado completo: %v", err)
	}
	canonOriginal, err := SerializarEstadoSeguimientoCanonico(original.Estado())
	if err != nil {
		t.Fatalf("canon original: %v", err)
	}
	canonRehidratado, err := SerializarEstadoSeguimientoCanonico(rehidratado.Estado())
	if err != nil {
		t.Fatalf("canon rehidratado: %v", err)
	}
	if !bytes.Equal(canonOriginal, canonRehidratado) ||
		rehidratado.Version() != original.Version() ||
		rehidratado.EstadoActual() != original.EstadoActual() ||
		len(rehidratado.PeriodosResultantes()) != len(original.PeriodosResultantes()) {
		t.Fatal("la rehidratación no reprodujo exactamente el agregado")
	}
	for indice, actuacion := range original.Actuaciones() {
		if rehidratado.Actuaciones()[indice].HuellaActuacionSHA256 !=
			actuacion.HuellaActuacionSHA256 {
			t.Fatalf("la huella %d cambió al rehidratar", indice)
		}
	}
}

func TestRehidratacionRechazaAdulteracionDeCadaEvento(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	original := seguimientoCompletoParaRehidratar(t, definicion).Estado()
	for indice := range original.Actuaciones {
		t.Run(original.Actuaciones[indice].ActuacionRef, func(t *testing.T) {
			adulterado := original.clonar()
			adulterado.Actuaciones[indice].ReciboRef += "_alterado"
			if _, err := RehidratarSeguimiento(definicion, adulterado); !errors.Is(
				err,
				ErrSeguimientoInvalido,
			) {
				t.Fatalf("se aceptó el evento %d adulterado: %v", indice, err)
			}
		})
	}
}

func TestRehidratacionRechazaCamposAdulteradosDeActuacion(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	original := seguimientoCompletoParaRehidratar(t, definicion).Estado()
	casos := []struct {
		nombre    string
		adulterar func(*EstadoPersistidoSeguimiento)
	}{
		{"secuencia", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].Secuencia++
		}},
		{"versión", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].VersionSeguimiento++
		}},
		{"definición", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].Definicion.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"referencia", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].ActuacionRef = "acto_prorroga_alterado_01"
		}},
		{"transición", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].TransicionClave = "registrar_incidencia"
		}},
		{"clase", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].Clase = TransicionRectificacion
		}},
		{"origen", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].EstadoOrigen = "pendiente_incorporacion"
		}},
		{"destino", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].EstadoDestino = "cesada"
		}},
		{"motivo", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].MotivoClave = "incidencia_catalogada"
		}},
		{"actor", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].ActorRef = "actor_publico_opaco_02"
		}},
		{"unidad", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].UnidadRef = "unidad_gestora_opaca_02"
		}},
		{"instante efectivo", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].EfectivoEn = e.Actuaciones[1].EfectivoEn.AddDate(0, 0, 1)
		}},
		{"instante registrado", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].RegistradaEn = e.Actuaciones[1].RegistradaEn.AddDate(0, 0, 1)
		}},
		{"documento", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].Documentos[0].Referencia = "documento_prorroga_alterado_01"
		}},
		{"periodo", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].Periodo.Hasta = e.Actuaciones[1].Periodo.Hasta.AddDate(0, 1, 0)
		}},
		{"calendario", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].Calendario.Version++
		}},
		{"correlación", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].CorrelacionRef = "correlacion_alterada_01"
		}},
		{"huella de petición", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].HuellaPeticionSHA256 = strings.Repeat("b", 64)
		}},
		{"huella anterior", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].HuellaAnteriorSHA256 = strings.Repeat("b", 64)
		}},
		{"huella de actuación", func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[1].HuellaActuacionSHA256 = strings.Repeat("b", 64)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adulterado := original.clonar()
			caso.adulterar(&adulterado)
			if _, err := RehidratarSeguimiento(definicion, adulterado); !errors.Is(
				err,
				ErrSeguimientoInvalido,
			) {
				t.Fatalf("se aceptó una actuación adulterada: %v", err)
			}
		})
	}
}

func TestCanonSeguimientoEsDeterministaYCerrado(t *testing.T) {
	definicionA := definicionSeguimientoValida(t, true)
	borrador := borradorDesdeDefinicion(definicionA)
	invertirEstados(borrador.Estados)
	invertirClaves(borrador.Motivos)
	invertirTransiciones(borrador.Transiciones)
	definicionB, err := PublicarDefinicionSeguimiento(borrador)
	if err != nil {
		t.Fatalf("publicar definición reordenada: %v", err)
	}
	if !definicionA.Referencia().Coincide(definicionB.Referencia()) {
		t.Fatal("el orden de entrada alteró la huella canónica de definición")
	}

	seguimiento := seguimientoCompletoParaRehidratar(
		t,
		definicionSeguimientoValida(t, false),
	)
	primero, err := SerializarEstadoSeguimientoCanonico(seguimiento.Estado())
	if err != nil {
		t.Fatalf("primer canon: %v", err)
	}
	segundo, err := SerializarEstadoSeguimientoCanonico(seguimiento.Estado())
	if err != nil {
		t.Fatalf("segundo canon: %v", err)
	}
	if !bytes.Equal(primero, segundo) {
		t.Fatal("la serialización canónica no es determinista")
	}

	publicacion := definicionA.Publicacion()
	publicacion.Canon.Dominio = "dominio_no_admitido"
	if _, err := RestaurarDefinicionSeguimiento(publicacion); !errors.Is(
		err,
		ErrDefinicionSeguimientoInvalida,
	) {
		t.Fatalf("se aceptó un esquema canónico desconocido: %v", err)
	}
}

func TestPublicacionDefinicionDistingueReintentoExactoYColision(t *testing.T) {
	referencia := definicionSeguimientoValida(t, false).Referencia()
	if err := ValidarReintentoPublicacionDefinicionSeguimiento(
		referencia,
		referencia,
	); err != nil {
		t.Fatalf("se rechazó la repetición exacta: %v", err)
	}
	otra := referencia
	otra.HuellaSHA256 = strings.Repeat("b", 64)
	if err := ValidarReintentoPublicacionDefinicionSeguimiento(
		referencia,
		otra,
	); !errors.Is(err, ErrPublicacionDefinicionSeguimientoEnConflicto) {
		t.Fatalf("no se detectó la colisión de publicación: %v", err)
	}
}

func TestSeguimientoRechazaColeccionesYReferenciasExcesivas(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	borrador := borradorDesdeDefinicion(definicion)
	borrador.Estados = make(
		[]EstadoDefinidoSeguimiento,
		maximoEstadosSeguimiento+1,
	)
	for indice := range borrador.Estados {
		borrador.Estados[indice] = EstadoDefinidoSeguimiento{
			Clave: ClaveCatalogo("estado_" + cadenaDecimalSeguimiento(indice)),
		}
	}
	if _, err := PublicarDefinicionSeguimiento(borrador); !errors.Is(
		err,
		ErrDefinicionSeguimientoInvalida,
	) {
		t.Fatalf("se aceptaron estados excesivos: %v", err)
	}

	base := seguimientoIncorporado(t, definicion)
	datos := datosSeguimiento("acto_documentos_excesivos_01", "registrar_incidencia", 2)
	datos.MotivoClave = "incidencia_catalogada"
	datos.Documentos = make([]DocumentoSeguimiento, maximoDocumentosPorTransicion+1)
	if _, err := base.Aplicar(definicion, 1, datos); !errors.Is(
		err,
		ErrTransicionInvalida,
	) {
		t.Fatalf("se aceptaron documentos excesivos: %v", err)
	}

	estado := base.Estado()
	estado.Actuaciones = make(
		[]ActuacionSeguimiento,
		maximoActuacionesSeguimiento+1,
	)
	if _, err := RehidratarSeguimiento(definicion, estado); !errors.Is(
		err,
		ErrSeguimientoInvalido,
	) {
		t.Fatalf("se aceptaron actuaciones excesivas: %v", err)
	}

	alta := AltaSeguimiento{
		Referencia:      strings.Repeat("a", 161),
		OrganizacionRef: referenciaSeguimientoPrueba("organizacion_publica_01"),
		ExpedienteRef:   referenciaSeguimientoPrueba("expediente_temporal_01"),
		RelacionRef:     referenciaSeguimientoPrueba("relacion_laboral_opaca_01"),
		PeriodoPrevisto: base.Estado().PeriodoPrevisto,
		CreadoEn:        instanteSeguimientoBase,
	}
	if _, err := NuevoSeguimiento(definicion, alta); !errors.Is(
		err,
		ErrSeguimientoInvalido,
	) {
		t.Fatalf("se aceptó una referencia excesiva: %v", err)
	}
}

func TestSerializadorCanonicoRechazaEstadoInvalidoAntesDeRecorrerColecciones(
	t *testing.T,
) {
	invalido := EstadoPersistidoSeguimiento{
		Version: 0, EstadoActual: "vigente",
		ActualizadoEn:    instanteSeguimientoBase,
		HuellaRaizSHA256: strings.Repeat("a", 64),
	}
	if _, err := SerializarEstadoSeguimientoCanonico(invalido); !errors.Is(
		err,
		ErrSeguimientoInvalido,
	) {
		t.Fatalf("se serializó un estado estructuralmente vacío: %v", err)
	}

	definicion := definicionSeguimientoValida(t, false)
	estado := seguimientoIncorporado(t, definicion).Estado()
	estado.PeriodosResultantes = make(
		[]PeriodoResultanteSeguimiento,
		maximoActuacionesSeguimiento+1,
	)
	if _, err := SerializarEstadoSeguimientoCanonico(estado); !errors.Is(
		err,
		ErrSeguimientoInvalido,
	) {
		t.Fatalf("se recorrió una colección excesiva de periodos: %v", err)
	}
}

func seguimientoCompletoParaRehidratar(
	t *testing.T,
	definicion DefinicionSeguimiento,
) Seguimiento {
	t.Helper()
	seguimiento := seguimientoIncorporado(t, definicion)
	prorroga := prorrogaSeguimientoValida(seguimiento, 2)
	var err error
	seguimiento, err = seguimiento.Aplicar(definicion, 1, prorroga)
	if err != nil {
		t.Fatalf("prorrogar para rehidratación: %v", err)
	}
	incidencia := datosSeguimiento("acto_incidencia_rehidratacion_01", "registrar_incidencia", 3)
	incidencia.MotivoClave = "incidencia_catalogada"
	incidencia.Documentos = []DocumentoSeguimiento{
		{TipoClave: "parte_incidencia", Referencia: referenciaSeguimientoPrueba("documento_incidencia_rehidratacion_01")},
	}
	seguimiento, err = seguimiento.Aplicar(definicion, 2, incidencia)
	if err != nil {
		t.Fatalf("registrar incidencia para rehidratación: %v", err)
	}
	cese := ceseSeguimientoValido(seguimiento, 4)
	cese.ActuacionRef = referenciaSeguimientoPrueba("acto_cese_rehidratacion_01")
	cese.ReciboRef = referenciaSeguimientoPrueba("recibo_acto_cese_rehidratacion_01")
	cese.CorrelacionRef = referenciaSeguimientoPrueba("correlacion_acto_cese_rehidratacion_01")
	seguimiento, err = seguimiento.Aplicar(definicion, 3, cese)
	if err != nil {
		t.Fatalf("cesar para rehidratación: %v", err)
	}
	return seguimiento
}

func invertirEstados(valores []EstadoDefinidoSeguimiento) {
	for izquierda, derecha := 0, len(valores)-1; izquierda < derecha; izquierda, derecha =
		izquierda+1, derecha-1 {
		valores[izquierda], valores[derecha] = valores[derecha], valores[izquierda]
	}
}

func invertirClaves(valores []ClaveCatalogo) {
	for izquierda, derecha := 0, len(valores)-1; izquierda < derecha; izquierda, derecha =
		izquierda+1, derecha-1 {
		valores[izquierda], valores[derecha] = valores[derecha], valores[izquierda]
	}
}

func invertirTransiciones(valores []TransicionDefinidaSeguimiento) {
	for izquierda, derecha := 0, len(valores)-1; izquierda < derecha; izquierda, derecha =
		izquierda+1, derecha-1 {
		valores[izquierda], valores[derecha] = valores[derecha], valores[izquierda]
	}
}

func cadenaDecimalSeguimiento(valor int) string {
	const digitos = "0123456789"
	if valor == 0 {
		return "0"
	}
	var invertida [20]byte
	posicion := len(invertida)
	for valor > 0 {
		posicion--
		invertida[posicion] = digitos[valor%10]
		valor /= 10
	}
	return string(invertida[posicion:])
}
