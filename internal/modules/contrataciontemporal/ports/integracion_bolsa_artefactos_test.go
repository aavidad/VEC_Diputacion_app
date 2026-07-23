package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type bandejaCASEventoBolsaPrueba struct{}

func (bandejaCASEventoBolsaPrueba) RegistrarEventoLlamamiento(
	ctx context.Context,
	comando ComandoRegistrarEventoBolsa,
	instanteActual time.Time,
) (AcuseEventoLlamamientoBolsa, error) {
	if err := ctx.Err(); err != nil {
		return AcuseEventoLlamamientoBolsa{}, err
	}
	evento, _, err := comando.DatosParaEfectoEn(instanteActual)
	if err != nil {
		return AcuseEventoLlamamientoBolsa{}, err
	}
	return acuseEventoBolsaPrueba(evento, instanteActual), nil
}

var _ BandejaEventosLlamamientoBolsa = bandejaCASEventoBolsaPrueba{}

func TestReinicioRealRehidrataArtefactosYLlegaARegistrarEvento(t *testing.T) {
	base := instanteBolsaPrueba()
	fresco := base.Add(3 * time.Minute)
	sellador := selladorRespuestaBolsaPrueba()
	verificadorInicial := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	)

	comandoLlamamiento := comandoLlamamientoPrueba(t, base, sellador)
	reciboLlamamiento := reciboLlamamientoPrueba(t, comandoLlamamiento, base)
	firmarLlamamientoPrueba(
		t, sellador, comandoLlamamiento, &reciboLlamamiento,
	)
	comprobanteLlamamiento, evidenciaLlamamiento, err :=
		verificadorInicial.VerificarReciboLlamamiento(
			context.Background(),
			comandoLlamamiento,
			reciboLlamamiento,
			fresco,
		)
	if err != nil {
		t.Fatalf("autenticar llamamiento inicial: %v", err)
	}
	artefactoLlamamiento, err := NuevoArtefactoProbatorioLlamamientoBolsa(
		comandoLlamamiento,
		reciboLlamamiento,
		evidenciaLlamamiento,
		comprobanteLlamamiento,
	)
	if err != nil {
		t.Fatalf("crear artefacto de llamamiento: %v", err)
	}
	enlace, err := NuevoEnlaceEventoLlamamientoBolsa(
		PreparacionEnlaceEventoLlamamientoBolsa{
			Comando: comandoLlamamiento, Recibo: reciboLlamamiento,
			Comprobante: comprobanteLlamamiento,
		},
	)
	if err != nil {
		t.Fatalf("crear enlace inicial: %v", err)
	}
	evento := nuevoEventoParaEnlaceBolsaPrueba(t, base, enlace)
	comprobanteEvento, evidenciaEvento, err := verificadorInicial.VerificarEvento(
		context.Background(), evento, enlace, fresco,
	)
	if err != nil {
		t.Fatalf("autenticar evento inicial: %v", err)
	}
	artefactoEvento, err := NuevoArtefactoProbatorioEventoBolsa(
		evento, enlace, evidenciaEvento, comprobanteEvento, fresco,
	)
	if err != nil {
		t.Fatalf("crear artefacto de evento: %v", err)
	}
	if artefactoLlamamiento.Esquema == "" ||
		artefactoLlamamiento.Version == 0 ||
		!huellaSHA256Valida(artefactoLlamamiento.HuellaArtefactoSHA256) ||
		!selloHMACBolsaValido(
			artefactoLlamamiento.SelloHMAC,
			dominioSelloRespuestaBolsa,
		) {
		t.Fatal("artefacto sin esquema, versión, huella o HMAC")
	}

	bytesLlamamiento, err := json.Marshal(artefactoLlamamiento)
	if err != nil {
		t.Fatalf("serializar llamamiento: %v", err)
	}
	bytesEvento, err := json.Marshal(artefactoEvento)
	if err != nil {
		t.Fatalf("serializar evento: %v", err)
	}

	// Simula pérdida total de memoria: ninguna capacidad opaca original queda
	// disponible para la recuperación.
	comandoLlamamiento = ComandoSolicitarLlamamientoBolsa{}
	reciboLlamamiento = ReciboSolicitudLlamamientoBolsa{}
	comprobanteLlamamiento = ComprobanteEvidenciaIntegracionBolsa{}
	enlace = EnlaceEventoLlamamientoBolsa{}
	evento = EventoLlamamientoBolsa{}
	comprobanteEvento = ComprobanteEvidenciaIntegracionBolsa{}
	artefactoLlamamiento = ArtefactoProbatorioLlamamientoBolsa{}
	artefactoEvento = ArtefactoProbatorioEventoBolsa{}
	verificadorInicial = nil

	llamamientoRecuperado, err :=
		DecodificarArtefactoProbatorioLlamamientoBolsa(bytesLlamamiento)
	if err != nil {
		t.Fatalf("leer artefacto de llamamiento tras reinicio: %v", err)
	}
	eventoRecuperado, err := DecodificarArtefactoProbatorioEventoBolsa(bytesEvento)
	if err != nil {
		t.Fatalf("leer artefacto de evento tras reinicio: %v", err)
	}
	autenticadorNuevo := autenticadorContextoBolsaPrueba(t)
	verificadorNuevo := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV2Prueba,
		claveRespuestaBolsaV1Prueba,
	)
	trasReinicio := base.Add(24 * time.Hour)
	if _, err := eventoRecuperado.Rehidratar(
		context.Background(),
		LlamamientoProbatorioRehidratadoBolsa{},
		verificadorNuevo,
		trasReinicio,
	); err == nil {
		t.Fatal("evento rehidratado sin recibo de llamamiento autenticado")
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := llamamientoRecuperado.Rehidratar(
		cancelado,
		autenticadorNuevo,
		verificadorNuevo,
		trasReinicio,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("rehidratación perdió cancelación: %v", err)
	}
	llamamientoRehidratado, err := llamamientoRecuperado.Rehidratar(
		context.Background(),
		autenticadorNuevo,
		verificadorNuevo,
		trasReinicio,
	)
	if err != nil {
		t.Fatalf("rehidratar llamamiento a +24h: %v", err)
	}
	enlaceRehidratado, err := enlaceDesdeLlamamientoProbatorioBolsa(
		llamamientoRehidratado,
	)
	if err != nil {
		t.Fatalf("reconstruir enlace exacto: %v", err)
	}
	eventoRehidratado, err := eventoRecuperado.Rehidratar(
		context.Background(),
		llamamientoRehidratado,
		verificadorNuevo,
		trasReinicio,
	)
	if err != nil {
		t.Fatalf("rehidratar evento a +24h: %v", err)
	}
	if _, _, err := verificadorNuevo.VerificarEvento(
		context.Background(),
		eventoRecuperado.Evento,
		enlaceRehidratado,
		trasReinicio,
	); !errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("evento histórico se consideró transporte fresco: %v", err)
	}
	comandoRegistro, err := NuevoComandoRegistrarEventoRehidratadoBolsa(
		eventoRehidratado,
		trasReinicio,
	)
	if err != nil {
		t.Fatalf("evidencia reautenticada no llegó al registro: %v", err)
	}
	if _, _, err := comandoRegistro.DatosParaEfectoEn(trasReinicio); err != nil {
		t.Fatalf("comando durable de registro no conserva prueba: %v", err)
	}
	limiteRetencion := eventoRecuperado.Evento.Procedencia.Evidencia.RetenerHasta
	if _, err := NuevoComandoRegistrarEventoRehidratadoBolsa(
		eventoRehidratado,
		limiteRetencion,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("capacidad rehidratada conservó un instante antiguo: %v", err)
	}
	if _, err := (bandejaCASEventoBolsaPrueba{}).RegistrarEventoLlamamiento(
		context.Background(),
		comandoRegistro,
		limiteRetencion,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) &&
		!errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("frontera CAS promovió un comando tras la retención: %v", err)
	}
}

func TestOrdenProbatoriaSobreviveReinicioSinContextoFresco(t *testing.T) {
	base := instanteBolsaPrueba()
	fresco := base.Add(3 * time.Minute)
	comando := comandoOrdenPrueba(t, base)
	recibo := reciboOrdenPrueba(t, base)
	firmarOrdenPrueba(t, selladorRespuestaBolsaPrueba(), comando, &recibo)
	comprobante, evidencia, err := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	).VerificarReciboOrden(context.Background(), comando, recibo, fresco)
	if err != nil {
		t.Fatalf("autenticar orden: %v", err)
	}
	artefacto, err := NuevoArtefactoProbatorioOrdenBolsa(
		comando, recibo, evidencia, comprobante,
	)
	if err != nil {
		t.Fatalf("crear artefacto de orden: %v", err)
	}
	contenido, err := json.Marshal(artefacto)
	if err != nil {
		t.Fatalf("serializar orden: %v", err)
	}
	comando = ComandoPrepararOrdenBolsa{}
	recibo = ReciboOrdenBolsa{}
	comprobante = ComprobanteEvidenciaIntegracionBolsa{}
	artefacto = ArtefactoProbatorioOrdenBolsa{}

	recuperado, err := DecodificarArtefactoProbatorioOrdenBolsa(contenido)
	if err != nil {
		t.Fatalf("recuperar orden: %v", err)
	}
	trasReinicio := base.Add(24 * time.Hour)
	ordenRehidratada, err := recuperado.Rehidratar(
		context.Background(),
		autenticadorContextoBolsaPrueba(t),
		verificadorRespuestaBolsaPrueba(
			claveRespuestaBolsaV2Prueba,
			claveRespuestaBolsaV1Prueba,
		),
		trasReinicio,
	)
	if err != nil {
		t.Fatalf("orden histórica no reautenticada: %v", err)
	}
	datosLlamamiento := datosContextoBolsaPrueba(
		trasReinicio,
		"operacion:llamamiento:tras-reinicio",
		ordenRehidratada.recibo.Orden,
		ordenRehidratada.recibo.AccionLlamamiento,
	)
	if _, err := NuevoComandoLlamamientoDesdeOrdenProbatoriaBolsa(
		PreparacionLlamamientoDesdeOrdenProbatoriaBolsa{
			Contexto:                emitirContextoBolsaPrueba(t, datosLlamamiento),
			OrdenProbatoria:         ordenRehidratada,
			MaximaPosicionEvaluable: 50,
		},
		trasReinicio,
	); err != nil {
		t.Fatalf("orden histórica auténtica no habilitó petición fresca: %v", err)
	}
}

func TestArtefactosCerradosRechazanCamposYAtaquesAunqueRecalculenHuella(t *testing.T) {
	base := instanteBolsaPrueba()
	comando := comandoOrdenPrueba(t, base)
	recibo := reciboOrdenPrueba(t, base)
	firmarOrdenPrueba(t, selladorRespuestaBolsaPrueba(), comando, &recibo)
	comprobante, evidencia, err := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	).VerificarReciboOrden(
		context.Background(), comando, recibo, base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("autenticar orden: %v", err)
	}
	artefacto, err := NuevoArtefactoProbatorioOrdenBolsa(
		comando, recibo, evidencia, comprobante,
	)
	if err != nil {
		t.Fatalf("crear artefacto: %v", err)
	}
	contenido, _ := json.Marshal(artefacto)
	conCampoAjeno := bytes.Replace(
		contenido,
		[]byte(`"version":1,`),
		[]byte(`"version":1,"campo_ajeno":true,`),
		1,
	)
	if _, err := DecodificarArtefactoProbatorioOrdenBolsa(
		conCampoAjeno,
	); !errors.Is(
		err,
		ErrEvidenciaBolsaNoAutenticada,
	) {
		t.Fatalf("sobre con campo desconocido aceptado: %v", err)
	}

	alterado := artefacto
	alterado.Comando.MaximoPosiciones--
	alterado.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(alterado)
	if _, err := alterado.Rehidratar(
		context.Background(),
		autenticadorContextoBolsaPrueba(t),
		verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba),
		base.Add(24*time.Hour),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) &&
		!errors.Is(err, ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("comando alterado y rehuellado fue autenticado: %v", err)
	}

	selloAlterado := artefacto
	selloAlterado.Evidencia.SelloHMAC = selloNominalBolsaPrueba(
		claveRespuestaBolsaV1Prueba, "4",
	)
	selloAlterado.SelloHMAC = selloAlterado.Evidencia.SelloHMAC
	selloAlterado.HuellaArtefactoSHA256 =
		huellaArtefactoProbatorioBolsa(selloAlterado)
	if _, err := selloAlterado.Rehidratar(
		context.Background(),
		autenticadorContextoBolsaPrueba(t),
		verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba),
		base.Add(24*time.Hour),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("HMAC alterado y rehuellado fue aceptado: %v", err)
	}
}

func TestArtefactosRechazanEncuadreNoCanonicoDuplicadoOAcosado(t *testing.T) {
	artefacto := artefactoOrdenBolsaPrueba(t)
	canonico, err := json.Marshal(artefacto)
	if err != nil {
		t.Fatalf("codificar artefacto: %v", err)
	}
	duplicado := bytes.Replace(
		canonico,
		[]byte(`"version":1`),
		[]byte(`"version":1,"version":1`),
		1,
	)
	duplicadoAnidado := []byte(`{"campo":{"valor":1,"valor":1}}`)
	reordenado := bytes.Replace(
		canonico,
		[]byte(
			`{"esquema":"vec.contratacion-temporal.artefacto-bolsa","version":1,`,
		),
		[]byte(
			`{"version":1,"esquema":"vec.contratacion-temporal.artefacto-bolsa",`,
		),
		1,
	)
	profundo := `{"campo":` +
		strings.Repeat("[", MaximaProfundidadArtefactoProbatorioBolsa+1) +
		`0` +
		strings.Repeat("]", MaximaProfundidadArtefactoProbatorioBolsa+1) +
		`}`
	elementos := `{"campo":[` +
		strings.Repeat(
			`0,`,
			MaximosElementosArtefactoProbatorioBolsa,
		) +
		`0]}`
	sobredimensionado := make(
		[]byte,
		MaximoBytesArtefactoProbatorioBolsa+1,
	)
	copy(sobredimensionado, canonico)
	for i := len(canonico); i < len(sobredimensionado); i++ {
		sobredimensionado[i] = ' '
	}
	casos := map[string][]byte{
		"duplicado":          duplicado,
		"duplicado_anidado":  duplicadoAnidado,
		"contenido_sobrante": append(append([]byte{}, canonico...), []byte(`{}`)...),
		"sobredimensionado":  sobredimensionado,
		"profundidad":        []byte(profundo),
		"coleccion":          []byte(elementos),
		"campos_reordenados": reordenado,
		"espacio_no_canonico": append(
			[]byte(" "),
			canonico...,
		),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := DecodificarArtefactoProbatorioOrdenBolsa(
				contenido,
			); !errors.Is(
				err,
				ErrEvidenciaBolsaNoAutenticada,
			) {
				t.Fatalf("encuadre hostil aceptado: %v", err)
			}
		})
	}
}

func TestCabeceraEsquemaVersionYTipoSonEstrictosEnLosTresArtefactos(
	t *testing.T,
) {
	orden, llamamiento, evento := artefactosProbatoriosBolsaPrueba(t)
	for nombre, variantes := range map[string][][]byte{
		"orden":       variantesCabeceraArtefactoBolsa(orden),
		"llamamiento": variantesCabeceraArtefactoBolsa(llamamiento),
		"evento":      variantesCabeceraArtefactoBolsa(evento),
	} {
		t.Run(nombre, func(t *testing.T) {
			for indice, contenido := range variantes {
				var err error
				switch nombre {
				case "orden":
					_, err = DecodificarArtefactoProbatorioOrdenBolsa(contenido)
				case "llamamiento":
					_, err = DecodificarArtefactoProbatorioLlamamientoBolsa(
						contenido,
					)
				case "evento":
					_, err = DecodificarArtefactoProbatorioEventoBolsa(contenido)
				}
				if !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
					t.Fatalf("cabecera alterada %d aceptada: %v", indice, err)
				}
			}
		})
	}
}

func TestUnmarshalArtefactosNoPromueveEstadoParcialAlRechazar(t *testing.T) {
	orden, llamamiento, evento := artefactosProbatoriosBolsaPrueba(t)

	ordenAntes := orden
	if err := orden.UnmarshalJSON(
		variantesCabeceraArtefactoBolsa(orden)[0],
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("orden inválida no rechazada: %v", err)
	}
	if orden != ordenAntes {
		t.Fatal("orden receptora fue modificada antes de completar la validación")
	}

	llamamientoAntes := llamamiento
	if err := llamamiento.UnmarshalJSON(
		variantesCabeceraArtefactoBolsa(llamamiento)[1],
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("llamamiento inválido no rechazado: %v", err)
	}
	if llamamiento != llamamientoAntes {
		t.Fatal("llamamiento receptor fue modificado antes de validar")
	}

	eventoAntes := evento
	if err := evento.UnmarshalJSON(
		variantesCabeceraArtefactoBolsa(evento)[2],
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evento inválido no rechazado: %v", err)
	}
	if evento != eventoAntes {
		t.Fatal("evento receptor fue modificado antes de completar la validación")
	}
}

func TestRetencionCaducadaImpideRehidratarYPruebaViejaNoRegistra(t *testing.T) {
	base := instanteBolsaPrueba()
	fresco := base.Add(3 * time.Minute)
	evento, enlace := eventoYEnlaceBolsaPrueba(t, base)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	comprobante, evidencia, err := verificador.VerificarEvento(
		context.Background(), evento, enlace, fresco,
	)
	if err != nil {
		t.Fatalf("autenticar evento: %v", err)
	}
	if _, err := NuevoComandoRegistrarEventoBolsa(
		evento,
		enlace,
		comprobante,
		base.Add(24*time.Hour),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("comprobante antiguo se reutilizó como fresco: %v", err)
	}
	if _, err := verificador.reautenticarEvento(
		context.Background(),
		evento,
		enlace,
		evidencia,
		evento.Procedencia.Evidencia.RetenerHasta,
	); !errors.Is(err, ErrEventoBolsaInvalido) &&
		!errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evento sobrevivió al límite de retención: %v", err)
	}
}

func autenticadorContextoBolsaPrueba(
	t *testing.T,
) *AutenticadorContextoPeticionIntegracionBolsa {
	t.Helper()
	autenticador, err := NuevoAutenticadorContextoPeticionIntegracionBolsa(
		autoridadPeticionBolsaPrueba,
		clavePeticionBolsaV1Prueba,
		nil,
		&verificadorHMACBolsaPrueba{
			claves: map[string][]byte{
				clavePeticionBolsaV1Prueba: secretoPeticionBolsaV1Prueba,
			},
		},
	)
	if err != nil {
		t.Fatalf("crear autenticador de contexto: %v", err)
	}
	return autenticador
}

func artefactoOrdenBolsaPrueba(
	t *testing.T,
) ArtefactoProbatorioOrdenBolsa {
	t.Helper()
	base := instanteBolsaPrueba()
	comando := comandoOrdenPrueba(t, base)
	recibo := reciboOrdenPrueba(t, base)
	firmarOrdenPrueba(t, selladorRespuestaBolsaPrueba(), comando, &recibo)
	comprobante, evidencia, err := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	).VerificarReciboOrden(
		context.Background(),
		comando,
		recibo,
		base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("autenticar orden: %v", err)
	}
	artefacto, err := NuevoArtefactoProbatorioOrdenBolsa(
		comando,
		recibo,
		evidencia,
		comprobante,
	)
	if err != nil {
		t.Fatalf("crear artefacto de orden: %v", err)
	}
	return artefacto
}

func artefactosProbatoriosBolsaPrueba(
	t *testing.T,
) (
	ArtefactoProbatorioOrdenBolsa,
	ArtefactoProbatorioLlamamientoBolsa,
	ArtefactoProbatorioEventoBolsa,
) {
	t.Helper()
	base := instanteBolsaPrueba()
	fresco := base.Add(3 * time.Minute)
	sellador := selladorRespuestaBolsaPrueba()
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)

	orden := artefactoOrdenBolsaPrueba(t)
	comando := comandoLlamamientoPrueba(t, base, sellador)
	recibo := reciboLlamamientoPrueba(t, comando, base)
	firmarLlamamientoPrueba(t, sellador, comando, &recibo)
	comprobanteLlamamiento, evidenciaLlamamiento, err :=
		verificador.VerificarReciboLlamamiento(
			context.Background(),
			comando,
			recibo,
			fresco,
		)
	if err != nil {
		t.Fatalf("autenticar llamamiento: %v", err)
	}
	llamamiento, err := NuevoArtefactoProbatorioLlamamientoBolsa(
		comando,
		recibo,
		evidenciaLlamamiento,
		comprobanteLlamamiento,
	)
	if err != nil {
		t.Fatalf("crear artefacto de llamamiento: %v", err)
	}
	enlace, err := NuevoEnlaceEventoLlamamientoBolsa(
		PreparacionEnlaceEventoLlamamientoBolsa{
			Comando: comando, Recibo: recibo, Comprobante: comprobanteLlamamiento,
		},
	)
	if err != nil {
		t.Fatalf("crear enlace: %v", err)
	}
	eventoValor := nuevoEventoParaEnlaceBolsaPrueba(t, base, enlace)
	comprobanteEvento, evidenciaEvento, err := verificador.VerificarEvento(
		context.Background(),
		eventoValor,
		enlace,
		fresco,
	)
	if err != nil {
		t.Fatalf("autenticar evento: %v", err)
	}
	evento, err := NuevoArtefactoProbatorioEventoBolsa(
		eventoValor,
		enlace,
		evidenciaEvento,
		comprobanteEvento,
		fresco,
	)
	if err != nil {
		t.Fatalf("crear artefacto de evento: %v", err)
	}
	return orden, llamamiento, evento
}

func variantesCabeceraArtefactoBolsa(artefacto any) [][]byte {
	codificar := func(valor any) []byte {
		contenido, err := json.Marshal(valor)
		if err != nil {
			panic(err)
		}
		return contenido
	}
	switch valor := artefacto.(type) {
	case ArtefactoProbatorioOrdenBolsa:
		esquema := valor
		esquema.Esquema = "vec.otro"
		esquema.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(esquema)
		version := valor
		version.Version++
		version.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(version)
		tipo := valor
		tipo.Tipo = tipoArtefactoEventoBolsa
		tipo.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(tipo)
		return [][]byte{codificar(esquema), codificar(version), codificar(tipo)}
	case ArtefactoProbatorioLlamamientoBolsa:
		esquema := valor
		esquema.Esquema = "vec.otro"
		esquema.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(esquema)
		version := valor
		version.Version++
		version.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(version)
		tipo := valor
		tipo.Tipo = tipoArtefactoEventoBolsa
		tipo.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(tipo)
		return [][]byte{codificar(esquema), codificar(version), codificar(tipo)}
	case ArtefactoProbatorioEventoBolsa:
		esquema := valor
		esquema.Esquema = "vec.otro"
		esquema.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(esquema)
		version := valor
		version.Version++
		version.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(version)
		tipo := valor
		tipo.Tipo = tipoArtefactoOrdenBolsa
		tipo.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(tipo)
		return [][]byte{codificar(esquema), codificar(version), codificar(tipo)}
	default:
		panic("tipo de artefacto no soportado")
	}
}
