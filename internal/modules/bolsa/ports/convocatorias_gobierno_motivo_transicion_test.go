package ports

import (
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type casoMotivoTransicionConvocatoria struct {
	nombre         string
	accion         string
	version        dominiobolsa.VersionConvocatoriaGobernada
	actorEsperado  string
	motivoEsperado string
	construir      func(AtestacionSelladoMotivoConvocatoria) error
}

func TestMaterialesExigenMotivoYActorExactosDeCadaTransicion(t *testing.T) {
	casos := casosMotivoTransicionConvocatoriaPrueba(t)
	if len(casos) != 6 {
		t.Fatalf("faltan acciones finales de gobierno en la prueba: %d", len(casos))
	}
	for indice := range casos {
		caso := casos[indice]
		t.Run(caso.nombre, func(t *testing.T) {
			if caso.actorEsperado == "" || caso.motivoEsperado == "" {
				t.Fatal("el caso no declara actor y motivo esperados")
			}
			valida := atestacionMotivoConvocatoriaConDatosPrueba(
				t, caso.accion, caso.version.Referencia(), caso.actorEsperado,
				"correlacion:convocatoria:001", caso.motivoEsperado, 'a',
			)
			if err := caso.construir(valida); err != nil {
				t.Fatalf("motivo exacto rechazado: %v", err)
			}

			accionCruzada := AccionCrearBorradorConvocatoria
			if caso.accion == accionCruzada {
				accionCruzada = AccionRetirarVersionConvocatoria
			}
			ataques := map[string]AtestacionSelladoMotivoConvocatoria{
				"motivo_de_otra_transicion": atestacionMotivoConvocatoriaConDatosPrueba(
					t, caso.accion, caso.version.Referencia(), caso.actorEsperado,
					"correlacion:convocatoria:001", caso.motivoEsperado+" Alterado.", 'b',
				),
				"actor_de_otra_transicion": atestacionMotivoConvocatoriaConDatosPrueba(
					t, caso.accion, caso.version.Referencia(), "per_actor_ajeno_0123456789",
					"correlacion:convocatoria:001", caso.motivoEsperado, 'c',
				),
				"accion_cruzada": atestacionMotivoConvocatoriaConDatosPrueba(
					t, accionCruzada, caso.version.Referencia(), caso.actorEsperado,
					"correlacion:convocatoria:001", caso.motivoEsperado, 'd',
				),
				"referencia_cruzada": atestacionMotivoConvocatoriaConDatosPrueba(
					t, caso.accion, "proceso:bolsa:ajeno#1", caso.actorEsperado,
					"correlacion:convocatoria:001", caso.motivoEsperado, 'e',
				),
				"correlacion_alterada_sin_rehacer_huella": atestacionMotivoConCorrelacionAlteradaPrueba(
					t, valida, "correlacion:convocatoria:ajena",
				),
			}
			for nombre, ataque := range ataques {
				t.Run(nombre, func(t *testing.T) {
					if err := caso.construir(ataque); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
						t.Fatalf("material acepto sellado semanticamente cruzado: %v", err)
					}
				})
			}
		})
	}
}

func TestCorrelacionAjenaAutoconsistenteSeRechazaAlCruzarAutorizacionPDP(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	selladoCorrelacionAjena := atestacionMotivoConvocatoriaConDatosPrueba(
		t, AccionCrearBorradorConvocatoria, version.Referencia(), version.CreadaPor,
		"correlacion:convocatoria:ajena", version.MotivoCreacion, 'a',
	)
	material, err := MaterialAltaBorradorConvocatoria(
		version, nil, nil, selladoCorrelacionAjena,
	)
	if err != nil {
		t.Fatalf("el material debe admitir una correlacion autoconsistente antes del cruce PDP: %v", err)
	}
	autorizacion := autorizacionMutacionConvocatoriaPrueba(t, material, version)
	testimonio := testimonioIdempotenciaConvocatoriaPrueba(t, material, autorizacion)
	datosSellado, err := selladoCorrelacionAjena.DatosParaConsumo()
	if err != nil {
		t.Fatal(err)
	}
	datosAutorizacion, err := autorizacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datosSellado.CorrelacionRef == datosAutorizacion.Decision.CorrelacionRef ||
		datosSellado.PrincipalRef != datosAutorizacion.Decision.PrincipalID {
		t.Fatal("la prueba no aisla exclusivamente el cruce de correlacion")
	}
	preparacion := PreparacionTransaccionGobiernoConvocatoria{
		Material: material, Idempotencia: testimonio, Autorizacion: autorizacion,
		SelladoMotivo: selladoCorrelacionAjena,
		SolicitadaEn:  instanteGobiernoConvocatoriaPrueba,
	}
	if err := preparacion.ValidarPara(version); !errors.Is(err, ErrConfirmacionGobiernoConvocatoriaInvalida) {
		t.Fatalf("la preparacion acepto motivo y PDP de correlaciones distintas: %v", err)
	}
}

func casosMotivoTransicionConvocatoriaPrueba(
	t *testing.T,
) []casoMotivoTransicionConvocatoria {
	t.Helper()
	borrador := versionGobernadaPuertosPrueba(t)
	estadoBorrador := estadoVersionConvocatoriaPrueba(t, borrador)

	configuracionActualizada := borrador.Configuracion
	configuracionActualizada.ReglasBaremacion.Version++
	configuracionActualizada.ReglasBaremacion.HuellaContenidoSHA256 = huellaConvocatoriaPrueba('d')
	actualizada, err := borrador.ActualizarBorrador(
		borrador.Revision, borrador.Contenido, configuracionActualizada,
		"per_actualiza_0123456789", "Actualizacion exacta del baremo.",
		borrador.CreadaEn.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("actualizar borrador de prueba: %v", err)
	}

	publicada := publicarInicialConvocatoriaPuertosPrueba(
		t, borrador, instanteGobiernoConvocatoriaPrueba.Add(-20*time.Minute),
	)
	publicadaEsperada := estadoVersionConvocatoriaPrueba(t, publicada)
	retirada := retirarConvocatoriaMotivoPrueba(
		t, publicada, instanteGobiernoConvocatoriaPrueba.Add(-12*time.Minute),
	)

	borradorSucesor, resultadoSucesor := publicarSucesoraMotivoPrueba(
		t, publicada, instanteGobiernoConvocatoriaPrueba.Add(-10*time.Minute),
		instanteGobiernoConvocatoriaPrueba.Add(-2*time.Minute),
	)
	borradorSucesorEsperado := estadoVersionConvocatoriaPrueba(t, borradorSucesor)
	predecesoraPublicadaEsperada := estadoVersionConvocatoriaPrueba(t, publicada)

	borradorTrasRetirada, resultadoTrasRetirada := publicarSucesoraMotivoPrueba(
		t, retirada, instanteGobiernoConvocatoriaPrueba.Add(-10*time.Minute),
		instanteGobiernoConvocatoriaPrueba.Add(-2*time.Minute),
	)
	borradorTrasRetiradaEsperado := estadoVersionConvocatoriaPrueba(t, borradorTrasRetirada)
	predecesoraRetiradaEsperada := estadoVersionConvocatoriaPrueba(t, retirada)

	return []casoMotivoTransicionConvocatoria{
		{
			nombre: "crear_borrador", accion: AccionCrearBorradorConvocatoria, version: borrador,
			actorEsperado: borrador.CreadaPor, motivoEsperado: borrador.MotivoCreacion,
			construir: func(m AtestacionSelladoMotivoConvocatoria) error {
				_, err := MaterialAltaBorradorConvocatoria(borrador, nil, nil, m)
				return err
			},
		},
		{
			nombre: "actualizar_borrador", accion: AccionActualizarBorradorConvocatoria, version: actualizada,
			actorEsperado:  actualizada.UltimaModificacionPor,
			motivoEsperado: actualizada.MotivoModificacion,
			construir: func(m AtestacionSelladoMotivoConvocatoria) error {
				_, err := MaterialActualizacionBorradorConvocatoria(estadoBorrador, actualizada, m)
				return err
			},
		},
		{
			nombre: "publicar_inicial", accion: AccionPublicarVersionConvocatoria, version: publicada,
			actorEsperado: publicada.PublicadaPor, motivoEsperado: publicada.MotivoPublicacion,
			construir: func(m AtestacionSelladoMotivoConvocatoria) error {
				_, err := MaterialPublicacionConvocatoria(estadoBorrador, publicada, nil, nil, m)
				return err
			},
		},
		{
			nombre: "publicar_y_sustituir", accion: AccionPublicarYSustituirConvocatoria,
			version:        resultadoSucesor.Publicada,
			actorEsperado:  resultadoSucesor.Publicada.PublicadaPor,
			motivoEsperado: resultadoSucesor.Publicada.MotivoPublicacion,
			construir: func(m AtestacionSelladoMotivoConvocatoria) error {
				_, err := MaterialPublicacionConvocatoria(
					borradorSucesorEsperado, resultadoSucesor.Publicada,
					&predecesoraPublicadaEsperada, &resultadoSucesor.Predecesora, m,
				)
				return err
			},
		},
		{
			nombre: "publicar_tras_retirada", accion: AccionPublicarTrasRetiradaConvocatoria,
			version:        resultadoTrasRetirada.Publicada,
			actorEsperado:  resultadoTrasRetirada.Publicada.PublicadaPor,
			motivoEsperado: resultadoTrasRetirada.Publicada.MotivoPublicacion,
			construir: func(m AtestacionSelladoMotivoConvocatoria) error {
				_, err := MaterialPublicacionConvocatoria(
					borradorTrasRetiradaEsperado, resultadoTrasRetirada.Publicada,
					&predecesoraRetiradaEsperada, &resultadoTrasRetirada.Predecesora, m,
				)
				return err
			},
		},
		{
			nombre: "retirar", accion: AccionRetirarVersionConvocatoria, version: retirada,
			actorEsperado: retirada.RetiradaPor, motivoEsperado: retirada.MotivoRetirada,
			construir: func(m AtestacionSelladoMotivoConvocatoria) error {
				_, err := MaterialRetiradaConvocatoria(publicadaEsperada, retirada, m)
				return err
			},
		},
	}
}

func estadoVersionConvocatoriaPrueba(
	t *testing.T,
	version dominiobolsa.VersionConvocatoriaGobernada,
) ReferenciaEstadoVersionConvocatoria {
	t.Helper()
	estado, err := EstadoVersionConvocatoria(version)
	if err != nil {
		t.Fatalf("obtener estado de version de prueba: %v", err)
	}
	return estado
}

func retirarConvocatoriaMotivoPrueba(
	t *testing.T,
	publicada dominiobolsa.VersionConvocatoriaGobernada,
	instante time.Time,
) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	huellaContenido, err := publicada.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstado, err := publicada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion := dominiobolsa.EvidenciaAprobacionConvocatoria{
		Accion: "retirar", Referencia: "aprobacion:retirada:motivo:001",
		HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('1'), ConvocatoriaRef: publicada.Referencia(),
		Revision: publicada.Revision, HuellaContenidoSHA256: huellaContenido,
		HuellaEstadoSHA256: huellaEstado, AprobadaPor: "per_aprueba_retirada_012345",
		AprobadaEn: instante.Add(-time.Minute),
	}
	retirada, err := publicada.Retirar(
		"per_retira_012345678901", aprobacion, "Retirada exacta de la convocatoria.", instante,
	)
	if err != nil {
		t.Fatalf("retirar convocatoria de prueba: %v", err)
	}
	return retirada
}

func publicarSucesoraMotivoPrueba(
	t *testing.T,
	predecesora dominiobolsa.VersionConvocatoriaGobernada,
	creadaEn, publicadaEn time.Time,
) (dominiobolsa.VersionConvocatoriaGobernada, dominiobolsa.ResultadoPublicacionSucesoraConvocatoria) {
	t.Helper()
	borrador, err := predecesora.NuevaVersion(
		"v2", predecesora.Contenido, predecesora.Configuracion,
		"expediente:seleccion:2026-002", "per_crea_sucesora_0123456",
		"Creacion exacta de la version sucesora.", creadaEn,
	)
	if err != nil {
		t.Fatalf("crear sucesora de prueba: %v", err)
	}
	huellaContenido, err := borrador.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstado, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion := dominiobolsa.EvidenciaAprobacionConvocatoria{
		Accion: "publicar", Referencia: "aprobacion:publicacion:sucesora:001",
		HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('2'), ConvocatoriaRef: borrador.Referencia(),
		Revision: borrador.Revision, HuellaContenidoSHA256: huellaContenido,
		HuellaEstadoSHA256: huellaEstado, AprobadaPor: "per_aprueba_sucesora_012345",
		AprobadaEn: publicadaEn.Add(-2 * time.Minute),
	}
	dependencias := dominiobolsa.EvidenciaDependenciasConvocatoria{
		Referencia:            "dependencias:publicacion:sucesora:001",
		HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('3'), ConvocatoriaRef: borrador.Referencia(),
		Revision: borrador.Revision, HuellaContenidoSHA256: huellaContenido,
		HuellaEstadoSHA256: huellaEstado, VerificadaEn: publicadaEn.Add(-time.Minute),
	}
	resultado, err := borrador.PublicarSucesora(
		predecesora, "per_publica_sucesora_01234", aprobacion, dependencias,
		"Publicacion exacta de la version sucesora.", publicadaEn,
	)
	if err != nil {
		t.Fatalf("publicar sucesora de prueba: %v", err)
	}
	return borrador, resultado
}

func atestacionMotivoConCorrelacionAlteradaPrueba(
	t *testing.T,
	original AtestacionSelladoMotivoConvocatoria,
	correlacionRef string,
) AtestacionSelladoMotivoConvocatoria {
	t.Helper()
	datos, err := original.DatosParaConsumo()
	if err != nil {
		t.Fatal(err)
	}
	datos.CorrelacionRef = correlacionRef
	alterada := AtestacionSelladoMotivoConvocatoria{datos: &datos}
	if _, err := alterada.DatosParaConsumo(); err != nil {
		t.Fatalf("el ataque debe conservar datos estructuralmente validos: %v", err)
	}
	return alterada
}
