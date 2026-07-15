package ports

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestConsultaReconciliacionDocumentalV4CopiaRetoYCodificaSinAmbiguedad(t *testing.T) {
	valorReto := retoValidoReconciliacionDocumentalV4(0x11)
	reto, err := NuevoRetoConsultaReconciliacionDocumentalV4(valorReto)
	if err != nil {
		t.Fatalf("crear reto: %v", err)
	}
	consulta := consultaValidaReconciliacionDocumentalV4ConReto(t, reto, "a")
	mensajeAntes, err := consulta.MensajeCanonico()
	if err != nil {
		t.Fatalf("mensaje canonico: %v", err)
	}
	huellaAntes, _ := consulta.HuellaMensajeSHA256()

	valorReto[0] ^= 0xff
	reto.valor[1] ^= 0xff
	datosDevueltos, err := consulta.Datos()
	if err != nil {
		t.Fatalf("datos: %v", err)
	}
	datosDevueltos.Reto.valor[2] ^= 0xff
	mensajeDespues, _ := consulta.MensajeCanonico()
	huellaDespues, _ := consulta.HuellaMensajeSHA256()
	if !bytes.Equal(mensajeAntes, mensajeDespues) || huellaAntes != huellaDespues {
		t.Fatal("la consulta conserva un alias mutable del reto")
	}

	campos := leerCamposCanonicosReconciliacionDocumentalV4(t, mensajeAntes)
	if len(campos) != 12 {
		t.Fatalf("numero de campos inesperado: %d", len(campos))
	}
	if string(campos[0]) != "vec.documentos.consulta-reconciliacion" ||
		len(campos[1]) != 2 ||
		binary.BigEndian.Uint16(campos[1]) != versionProtocoloReconciliacionDocumentalV4 ||
		len(campos[3]) != minimoBytesRetoReconciliacionDocumentalV4 ||
		len(campos[8]) != 8 || len(campos[9]) != 8 {
		t.Fatal("el payload no conserva dominio, version, reto o enteros canonicos")
	}
	mensajeAntes[0] ^= 0xff
	otraCopia, _ := consulta.MensajeCanonico()
	if bytes.Equal(mensajeAntes, otraCopia) {
		t.Fatal("MensajeCanonico devolvio memoria interna")
	}
}

func TestRetoConsultaReconciliacionDocumentalV4EsNominalRedactadoYNoSerializable(
	t *testing.T,
) {
	valor := retoValidoReconciliacionDocumentalV4(0x21)
	reto, err := NuevoRetoConsultaReconciliacionDocumentalV4(valor)
	if err != nil {
		t.Fatalf("crear reto: %v", err)
	}
	copia, _ := reto.BytesParaProtocolo()
	copia[0] ^= 0xff
	segunda, _ := reto.BytesParaProtocolo()
	if bytes.Equal(copia, segunda) {
		t.Fatal("el reto expone su almacenamiento")
	}
	texto := fmt.Sprintf("%v %#v", reto, reto)
	if strings.Contains(texto, fmt.Sprintf("%x", valor)) ||
		!strings.Contains(texto, "REDACTADO") {
		t.Fatalf("representacion insegura: %q", texto)
	}
	if _, err := json.Marshal(reto); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("JSON no fue bloqueado: %v", err)
	}
	if _, err := reto.MarshalText(); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("texto no fue bloqueado: %v", err)
	}
	if _, err := reto.MarshalBinary(); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("binario generico no fue bloqueado: %v", err)
	}
}

func TestConsultaReconciliacionDocumentalV4DeniegaCamposDebiles(t *testing.T) {
	base := datosConsultaValidosReconciliacionDocumentalV4(t, "b")
	casos := map[string]func(*DatosConsultaReconciliacionDocumentalV4){
		"consulta no servidor": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.ConsultaRef = "consulta:externa:001"
		},
		"consulta comodin": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.ConsultaRef = "consulta:reconciliacion:v4:*"
		},
		"reserva con dni": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.ReservaRef = "reserva:12345678z"
		},
		"efecto con nie": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.EfectoRef = "efecto:x1234567a"
		},
		"alias consulta reserva": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.ReservaRef = d.ConsultaRef
		},
		"alias reserva efecto": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.EfectoRef = d.ReservaRef
		},
		"plan no sha256": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.HuellaPlanSHA256 = "plan:*"
		},
		"plan cero": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.HuellaPlanSHA256 = strings.Repeat("0", 64)
		},
		"estado distinto": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.EstadoEsperado = EstadoEjecucionDocumentalV3Activa
		},
		"version cero": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.VersionEsperada = 0
		},
		"cercado cero": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.SecuenciaCercadoEsperada = 0
		},
		"tiempo local": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.EmitidaEn = d.EmitidaEn.In(time.FixedZone("local", 3600))
		},
		"nanosegundo no canonico": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.EmitidaEn = d.EmitidaEn.Add(time.Nanosecond)
		},
		"ventana vacia": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.ExpiraEn = d.EmitidaEn
		},
		"ventana excesiva": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.ExpiraEn = d.EmitidaEn.Add(
				maximaVigenciaConsultaReconciliacionDocumentalV4 + time.Microsecond,
			)
		},
		"reto cero": func(d *DatosConsultaReconciliacionDocumentalV4) {
			d.Reto = RetoConsultaReconciliacionDocumentalV4{}
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			mutar(&datos)
			if _, err := NuevaConsultaReconciliacionDocumentalV4(datos); !errors.Is(
				err, ErrConsultaReconciliacionDocumentalV4Invalida,
			) {
				t.Fatalf("se acepto consulta debil: %v", err)
			}
		})
	}
}

func TestRetoConsultaReconciliacionDocumentalV4DeniegaTamanoYCero(t *testing.T) {
	casos := [][]byte{
		nil,
		make([]byte, minimoBytesRetoReconciliacionDocumentalV4-1),
		make([]byte, minimoBytesRetoReconciliacionDocumentalV4),
		make([]byte, maximoBytesRetoReconciliacionDocumentalV4+1),
	}
	for indice, valor := range casos {
		if _, err := NuevoRetoConsultaReconciliacionDocumentalV4(valor); !errors.Is(
			err, ErrRetoConsultaReconciliacionDocumentalV4Invalido,
		) {
			t.Fatalf("caso %d aceptado: %v", indice, err)
		}
	}
}

func TestRespuestaCrudaReconciliacionDocumentalV4LigaConsultaYEstadosExactos(t *testing.T) {
	consulta := consultaValidaReconciliacionDocumentalV4(t, "c")
	atestacion := atestacionCrudaValidaReconciliacionDocumentalV4(t, 0x31)
	resultados := []EstadoResultadoReconciliacionDocumentalV4{
		EstadoResultadoReconciliacionDocumentalV4Aplicado,
		EstadoResultadoReconciliacionDocumentalV4NoAplicado,
		EstadoResultadoReconciliacionDocumentalV4Desconocido,
	}
	for _, resultado := range resultados {
		t.Run(string(resultado), func(t *testing.T) {
			declaracion := declaracionValidaReconciliacionDocumentalV4(t, consulta, resultado)
			respuesta, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
				consulta, declaracion, atestacion,
			)
			if err != nil {
				t.Fatalf("crear respuesta: %v", err)
			}
			if respuesta.ValidarSintaxisContra(consulta) != nil {
				t.Fatal("respuesta valida rechazada")
			}
			mensaje, _ := respuesta.MensajeCanonico()
			campos := leerCamposCanonicosReconciliacionDocumentalV4(t, mensaje)
			if len(campos) != 16 ||
				string(campos[0]) != "vec.documentos.respuesta-reconciliacion" ||
				binary.BigEndian.Uint16(campos[1]) != versionProtocoloReconciliacionDocumentalV4 ||
				string(campos[12]) != string(resultado) {
				t.Fatal("respuesta no codifica dominio, version o resultado exactos")
			}
			mensaje[0] ^= 0xff
			segunda, _ := respuesta.MensajeCanonico()
			if bytes.Equal(mensaje, segunda) {
				t.Fatal("la respuesta expone su payload interno")
			}
		})
	}
}

func TestRespuestaCrudaReconciliacionDocumentalV4DeniegaMezclaReplayYMutacion(
	t *testing.T,
) {
	consultaA := consultaValidaReconciliacionDocumentalV4(t, "d")
	consultaB := consultaValidaReconciliacionDocumentalV4(t, "e")
	atestacion := atestacionCrudaValidaReconciliacionDocumentalV4(t, 0x41)
	base := declaracionValidaReconciliacionDocumentalV4(
		t, consultaA, EstadoResultadoReconciliacionDocumentalV4Aplicado,
	)
	casos := map[string]func(*DeclaracionRespuestaReconciliacionDocumentalV4){
		"consulta": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.ConsultaRef = consultaB.datos.ConsultaRef
		},
		"huella consulta": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.HuellaConsultaSHA256 = strings.Repeat("b", 64)
		},
		"huella reto": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.HuellaRetoSHA256 = strings.Repeat("c", 64)
		},
		"reserva": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.ReservaRef = "reserva:documental:v4:otra"
		},
		"efecto": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.EfectoRef = "efecto:documental:v4:otro"
		},
		"plan": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.HuellaPlanSHA256 = strings.Repeat("d", 64)
		},
		"estado reserva": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.EstadoReservaEsperadoEco = EstadoEjecucionDocumentalV3Confirmada
		},
		"version": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.VersionReservaEsperadaEco++
		},
		"cercado": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.SecuenciaCercadoEsperadaEco++
		},
		"resultado abierto": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.Resultado = "conflicto"
		},
		"tiempo anterior": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.RespondidaEn = consultaA.datos.EmitidaEn.Add(-time.Microsecond)
		},
		"tiempo posterior": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.RespondidaEn = consultaA.datos.ExpiraEn.Add(time.Microsecond)
		},
		"tiempo no canonico": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.RespondidaEn = d.RespondidaEn.Add(time.Nanosecond)
		},
		"aplicado sin huella": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.HuellaEfectoAplicadoSHA256 = ""
		},
		"aplicado sin tamano": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.TamanoEfectoAplicado = 0
		},
		"aplicado con huella cero": func(d *DeclaracionRespuestaReconciliacionDocumentalV4) {
			d.HuellaEfectoAplicadoSHA256 = strings.Repeat("0", 64)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			declaracion := base
			mutar(&declaracion)
			if _, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
				consultaA, declaracion, atestacion,
			); !errors.Is(err, ErrRespuestaCrudaReconciliacionDocumentalV4Invalida) {
				t.Fatalf("se acepto mezcla: %v", err)
			}
		})
	}

	if _, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
		consultaB, base, atestacion,
	); !errors.Is(err, ErrRespuestaCrudaReconciliacionDocumentalV4Invalida) {
		t.Fatalf("se reprodujo respuesta de otra consulta: %v", err)
	}

	noAplicado := declaracionValidaReconciliacionDocumentalV4(
		t, consultaA, EstadoResultadoReconciliacionDocumentalV4NoAplicado,
	)
	noAplicado.HuellaEfectoAplicadoSHA256 = strings.Repeat("e", 64)
	noAplicado.TamanoEfectoAplicado = 1
	if _, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
		consultaA, noAplicado, atestacion,
	); !errors.Is(err, ErrRespuestaCrudaReconciliacionDocumentalV4Invalida) {
		t.Fatalf("no_aplicado acepto resultado material: %v", err)
	}
}

func TestVentanaRespuestaReconciliacionDocumentalV4TieneLimiteSuperiorExclusivo(t *testing.T) {
	consulta := consultaValidaReconciliacionDocumentalV4(t, "f")
	atestacion := atestacionCrudaValidaReconciliacionDocumentalV4(t, 0x51)
	declaracion := declaracionValidaReconciliacionDocumentalV4(
		t, consulta, EstadoResultadoReconciliacionDocumentalV4Desconocido,
	)
	declaracion.RespondidaEn = consulta.datos.EmitidaEn
	if _, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
		consulta, declaracion, atestacion,
	); err != nil {
		t.Fatalf("limite inferior inclusivo rechazado: %v", err)
	}
	declaracion.RespondidaEn = consulta.datos.ExpiraEn
	if _, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
		consulta, declaracion, atestacion,
	); !errors.Is(err, ErrRespuestaCrudaReconciliacionDocumentalV4Invalida) {
		t.Fatalf("limite superior inclusivo aceptado: %v", err)
	}
}

func TestRespuestaReconciliacionDocumentalV4NoEsSerializableNiAutoridad(t *testing.T) {
	consulta := consultaValidaReconciliacionDocumentalV4(t, "1")
	respuesta := respuestaValidaReconciliacionDocumentalV4(
		t, consulta, EstadoResultadoReconciliacionDocumentalV4Desconocido,
	)
	if _, err := json.Marshal(consulta); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("consulta serializable: %v", err)
	}
	if _, err := json.Marshal(respuesta); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("respuesta serializable: %v", err)
	}
	if _, err := respuesta.MarshalText(); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("texto serializable: %v", err)
	}
	if _, err := respuesta.MarshalBinary(); !errors.Is(err, ErrSerializacionReconciliacionDocumentalV4) {
		t.Fatalf("binario generico serializable: %v", err)
	}
	texto := fmt.Sprintf("%v %#v", consulta, respuesta)
	if strings.Contains(texto, consulta.datos.ConsultaRef) || !strings.Contains(texto, "REDACTADA") {
		t.Fatalf("representacion filtra referencias: %q", texto)
	}
	// La unica validacion disponible se llama expresamente ValidarSintaxis;
	// una atestacion cruda no expone ningun constructor de resultado verificado.
	atestacion, err := respuesta.AtestacionCruda()
	if err != nil || atestacion.ValidarSintaxis() != nil {
		t.Fatalf("atestacion nominal invalida: %v", err)
	}
}

func TestProyeccionCASReconciliacionDocumentalV4EsFailClosed(t *testing.T) {
	casos := map[EstadoResultadoReconciliacionDocumentalV4]struct {
		accion AccionProyectadaReconciliacionDocumentalV4
		sufijo string
	}{
		EstadoResultadoReconciliacionDocumentalV4Aplicado: {
			accion: AccionReconciliacionDocumentalV4RegistrarSoloEvidencia,
			sufijo: "a",
		},
		EstadoResultadoReconciliacionDocumentalV4NoAplicado: {
			accion: AccionReconciliacionDocumentalV4RegistrarSoloEvidencia,
			sufijo: "b",
		},
		EstadoResultadoReconciliacionDocumentalV4Desconocido: {
			accion: AccionReconciliacionDocumentalV4RegistrarSoloEvidencia,
			sufijo: "c",
		},
	}
	for resultado, caso := range casos {
		t.Run(string(resultado), func(t *testing.T) {
			consulta := consultaValidaReconciliacionDocumentalV4(t, caso.sufijo)
			respuesta := respuestaValidaReconciliacionDocumentalV4(t, consulta, resultado)
			proyeccion, err := ProyectarIntentoCASReconciliacionDocumentalV4(
				consulta, respuesta,
			)
			if err != nil {
				t.Fatalf("proyectar: %v", err)
			}
			accion, _ := proyeccion.AccionProyectada()
			condicion, _ := proyeccion.CondicionCAS()
			clave, _ := proyeccion.ClaveConsumoUnico()
			if accion != caso.accion ||
				condicion.EstadoEsperado != EstadoEjecucionDocumentalV3Indeterminada ||
				condicion.VersionEsperada != consulta.datos.VersionEsperada ||
				condicion.SecuenciaCercadoEsperada != consulta.datos.SecuenciaCercadoEsperada ||
				clave.ValidarContra(consulta) != nil ||
				!proyeccion.RequiereVerificacionCriptograficaFresca() {
				t.Fatal("la proyeccion relajo CAS, consumo o verificacion fresca")
			}
		})
	}
	if !ResultadoIntentoCASReconciliacionDocumentalV4Conflicto.SoloPermiteRegistrarEvidencia() ||
		!ResultadoIntentoCASReconciliacionDocumentalV4Aplicado.SoloPermiteRegistrarEvidencia() ||
		ResultadoIntentoCASReconciliacionDocumentalV4("otro").Valido() {
		t.Fatal("el conflicto CAS no queda cerrado a evidencia")
	}
}

func TestClaveConsumoReconciliacionDocumentalV4ExigeIndicesUnicosIndependientes(
	t *testing.T,
) {
	reto, err := NuevoRetoConsultaReconciliacionDocumentalV4(
		retoValidoReconciliacionDocumentalV4(0x71),
	)
	if err != nil {
		t.Fatalf("reto: %v", err)
	}
	consultaA := consultaValidaReconciliacionDocumentalV4ConReto(t, reto, "7")
	consultaB := consultaValidaReconciliacionDocumentalV4ConReto(t, reto, "8")
	claveA, _ := consultaA.ClaveConsumoUnico()
	claveARepetida, _ := consultaA.ClaveConsumoUnico()
	claveB, _ := consultaB.ClaveConsumoUnico()
	consumidas := map[ClaveConsumoConsultaReconciliacionDocumentalV4]struct{}{claveA: {}}
	consultasUnicas := map[string]struct{}{claveA.ConsultaRef: {}}
	retosUnicos := map[string]struct{}{claveA.HuellaRetoSHA256: {}}
	if _, existe := consumidas[claveARepetida]; !existe {
		t.Fatal("la clave comparable no detecta replay exacto")
	}
	_, consultaDuplicada := consultasUnicas[claveB.ConsultaRef]
	_, retoDuplicado := retosUnicos[claveB.HuellaRetoSHA256]
	if claveA == claveB || consultaDuplicada || !retoDuplicado {
		t.Fatal("los indices separados no distinguen consulta nueva y reto reutilizado")
	}
}

func datosConsultaValidosReconciliacionDocumentalV4(
	t *testing.T,
	sufijo string,
) DatosConsultaReconciliacionDocumentalV4 {
	t.Helper()
	reto, err := NuevoRetoConsultaReconciliacionDocumentalV4(
		retoValidoReconciliacionDocumentalV4(sufijo[0]),
	)
	if err != nil {
		t.Fatalf("crear reto: %v", err)
	}
	instante := time.Date(2026, time.July, 15, 12, 30, 0, 123456000, time.UTC)
	return DatosConsultaReconciliacionDocumentalV4{
		ConsultaRef:              "consulta:reconciliacion:v4:" + strings.Repeat(sufijo, 32),
		Reto:                     reto,
		ReservaRef:               "reserva:documental:v4:001",
		EfectoRef:                "efecto:documental:v4:001",
		HuellaPlanSHA256:         strings.Repeat("a", 64),
		EstadoEsperado:           EstadoEjecucionDocumentalV3Indeterminada,
		VersionEsperada:          17,
		SecuenciaCercadoEsperada: 23,
		EmitidaEn:                instante,
		ExpiraEn:                 instante.Add(maximaVigenciaConsultaReconciliacionDocumentalV4),
	}
}

func consultaValidaReconciliacionDocumentalV4(
	t *testing.T,
	sufijo string,
) ConsultaReconciliacionDocumentalV4 {
	t.Helper()
	datos := datosConsultaValidosReconciliacionDocumentalV4(t, sufijo)
	consulta, err := NuevaConsultaReconciliacionDocumentalV4(datos)
	if err != nil {
		t.Fatalf("crear consulta: %v", err)
	}
	return consulta
}

func consultaValidaReconciliacionDocumentalV4ConReto(
	t *testing.T,
	reto RetoConsultaReconciliacionDocumentalV4,
	sufijo string,
) ConsultaReconciliacionDocumentalV4 {
	t.Helper()
	datos := datosConsultaValidosReconciliacionDocumentalV4(t, sufijo)
	datos.Reto = reto
	consulta, err := NuevaConsultaReconciliacionDocumentalV4(datos)
	if err != nil {
		t.Fatalf("crear consulta: %v", err)
	}
	return consulta
}

func declaracionValidaReconciliacionDocumentalV4(
	t *testing.T,
	consulta ConsultaReconciliacionDocumentalV4,
	resultado EstadoResultadoReconciliacionDocumentalV4,
) DeclaracionRespuestaReconciliacionDocumentalV4 {
	t.Helper()
	huellaReto, _ := consulta.datos.Reto.HuellaSHA256()
	declaracion := DeclaracionRespuestaReconciliacionDocumentalV4{
		ConsultaRef:                 consulta.datos.ConsultaRef,
		HuellaConsultaSHA256:        consulta.huellaMensajeSHA256,
		HuellaRetoSHA256:            huellaReto,
		ReservaRef:                  consulta.datos.ReservaRef,
		EfectoRef:                   consulta.datos.EfectoRef,
		HuellaPlanSHA256:            consulta.datos.HuellaPlanSHA256,
		EstadoReservaEsperadoEco:    consulta.datos.EstadoEsperado,
		VersionReservaEsperadaEco:   consulta.datos.VersionEsperada,
		SecuenciaCercadoEsperadaEco: consulta.datos.SecuenciaCercadoEsperada,
		Resultado:                   resultado,
		RespondidaEn:                consulta.datos.EmitidaEn.Add(time.Minute),
	}
	if resultado == EstadoResultadoReconciliacionDocumentalV4Aplicado {
		declaracion.HuellaEfectoAplicadoSHA256 = strings.Repeat("f", 64)
		declaracion.TamanoEfectoAplicado = 4096
	}
	return declaracion
}

func respuestaValidaReconciliacionDocumentalV4(
	t *testing.T,
	consulta ConsultaReconciliacionDocumentalV4,
	resultado EstadoResultadoReconciliacionDocumentalV4,
) RespuestaCrudaReconciliacionDocumentalV4 {
	t.Helper()
	declaracion := declaracionValidaReconciliacionDocumentalV4(t, consulta, resultado)
	atestacion := atestacionCrudaValidaReconciliacionDocumentalV4(t, 0x61)
	respuesta, err := NuevaRespuestaCrudaReconciliacionDocumentalV4(
		consulta, declaracion, atestacion,
	)
	if err != nil {
		t.Fatalf("crear respuesta: %v", err)
	}
	return respuesta
}

func atestacionCrudaValidaReconciliacionDocumentalV4(
	t *testing.T,
	semilla byte,
) AtestacionCrudaReconciliacionDocumentalV4 {
	t.Helper()
	contenido := make([]byte, 64)
	for indice := range contenido {
		contenido[indice] = semilla + byte(indice%17)
	}
	sobre, err := NuevoSobreCriptograficoDocumentalCrudoV4(contenido)
	if err != nil {
		t.Fatalf("crear sobre: %v", err)
	}
	atestacion, err := NuevaAtestacionCrudaReconciliacionDocumentalV4(sobre)
	if err != nil {
		t.Fatalf("crear atestacion: %v", err)
	}
	return atestacion
}

func retoValidoReconciliacionDocumentalV4(semilla byte) []byte {
	valor := make([]byte, minimoBytesRetoReconciliacionDocumentalV4)
	for indice := range valor {
		valor[indice] = semilla + byte(indice%13)
	}
	return valor
}

func leerCamposCanonicosReconciliacionDocumentalV4(t *testing.T, mensaje []byte) [][]byte {
	t.Helper()
	campos := make([][]byte, 0, 16)
	for posicion := 0; posicion < len(mensaje); {
		if len(mensaje)-posicion < 4 {
			t.Fatal("prefijo de longitud truncado")
		}
		longitud := int(binary.BigEndian.Uint32(mensaje[posicion : posicion+4]))
		posicion += 4
		if longitud < 0 || longitud > len(mensaje)-posicion {
			t.Fatal("campo canonico truncado")
		}
		campos = append(campos, append([]byte(nil), mensaje[posicion:posicion+longitud]...))
		posicion += longitud
	}
	return campos
}
