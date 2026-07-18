package ports

import (
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type casoMotivoTransicionConvocatoria struct {
	nombre         string
	accion         string
	version        dominiobolsa.VersionConvocatoriaGobernada
	actorEsperado  string
	motivoEsperado string
	construir      func(CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error)
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
			valida := compromisoMotivoConvocatoriaConDatosPrueba(
				t, caso.accion, caso.version.Referencia(), caso.actorEsperado,
				"correlacion:convocatoria:001", caso.motivoEsperado, 'a',
			)
			material, err := caso.construir(valida)
			if err != nil {
				t.Fatalf("motivo exacto rechazado: %v", err)
			}
			autorizacion := autorizacionMutacionConvocatoriaPrueba(t, material, caso.version)
			testimonio := testimonioIdempotenciaConvocatoriaPrueba(t, material, autorizacion)
			atestacion := atestacionMotivoMaterializadaConvocatoriaPrueba(
				t, valida, material, autorizacion, testimonio, caso.version, byte('a'+indice),
			)
			preparacion := PreparacionTransaccionGobiernoConvocatoria{
				Material: material, Idempotencia: testimonio, Autorizacion: autorizacion,
				CompromisoMotivo: valida, SelladoMotivo: atestacion,
				SolicitadaEn: instanteGobiernoConvocatoriaPrueba,
			}
			if err := preparacion.ValidarPara(caso.version); err != nil {
				t.Fatalf("materializacion posterior al PDP rechazada: %v", err)
			}

			accionCruzada := AccionCrearBorradorConvocatoria
			if caso.accion == accionCruzada {
				accionCruzada = AccionRetirarVersionConvocatoria
			}
			ataques := map[string]CompromisoMotivoGobiernoConvocatoria{
				"motivo_de_otra_transicion": compromisoMotivoConvocatoriaConDatosPrueba(
					t, caso.accion, caso.version.Referencia(), caso.actorEsperado,
					"correlacion:convocatoria:001", caso.motivoEsperado+" Alterado.", 'b',
				),
				"actor_de_otra_transicion": compromisoMotivoConvocatoriaConDatosPrueba(
					t, caso.accion, caso.version.Referencia(), "per_actor_ajeno_0123456789",
					"correlacion:convocatoria:001", caso.motivoEsperado, 'c',
				),
				"accion_cruzada": compromisoMotivoConvocatoriaConDatosPrueba(
					t, accionCruzada, caso.version.Referencia(), caso.actorEsperado,
					"correlacion:convocatoria:001", caso.motivoEsperado, 'd',
				),
				"referencia_cruzada": compromisoMotivoConvocatoriaConDatosPrueba(
					t, caso.accion, "proceso:bolsa:ajeno#1", caso.actorEsperado,
					"correlacion:convocatoria:001", caso.motivoEsperado, 'e',
				),
				"correlacion_alterada_sin_rehacer_huella": compromisoMotivoConCorrelacionAlteradaPrueba(
					t, valida, "correlacion:convocatoria:ajena",
				),
			}
			for nombre, ataque := range ataques {
				t.Run(nombre, func(t *testing.T) {
					if _, err := caso.construir(ataque); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
						t.Fatalf("material acepto sellado semanticamente cruzado: %v", err)
					}
				})
			}
		})
	}
}

func TestCorrelacionAjenaAutoconsistenteSeRechazaAlCruzarAutorizacionPDP(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	selladoCorrelacionAjena := compromisoMotivoConvocatoriaConDatosPrueba(
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
	datosSellado, err := selladoCorrelacionAjena.DatosParaMaterial()
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
	if _, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		selladoCorrelacionAjena, material, autorizacion, testimonio, version,
		instanteGobiernoConvocatoriaPrueba,
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("la materializacion acepto motivo y PDP de correlaciones distintas: %v", err)
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
		"per_actualiza_0123456789012345", "Actualizacion exacta del baremo.",
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
			construir: func(m CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error) {
				return MaterialAltaBorradorConvocatoria(borrador, nil, nil, m)
			},
		},
		{
			nombre: "actualizar_borrador", accion: AccionActualizarBorradorConvocatoria, version: actualizada,
			actorEsperado:  actualizada.UltimaModificacionPor,
			motivoEsperado: actualizada.MotivoModificacion,
			construir: func(m CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error) {
				return MaterialActualizacionBorradorConvocatoria(estadoBorrador, actualizada, m)
			},
		},
		{
			nombre: "publicar_inicial", accion: AccionPublicarVersionConvocatoria, version: publicada,
			actorEsperado: publicada.PublicadaPor, motivoEsperado: publicada.MotivoPublicacion,
			construir: func(m CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error) {
				return MaterialPublicacionConvocatoria(estadoBorrador, publicada, nil, nil, m)
			},
		},
		{
			nombre: "publicar_y_sustituir", accion: AccionPublicarYSustituirConvocatoria,
			version:        resultadoSucesor.Publicada,
			actorEsperado:  resultadoSucesor.Publicada.PublicadaPor,
			motivoEsperado: resultadoSucesor.Publicada.MotivoPublicacion,
			construir: func(m CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error) {
				return MaterialPublicacionConvocatoria(
					borradorSucesorEsperado, resultadoSucesor.Publicada,
					&predecesoraPublicadaEsperada, &resultadoSucesor.Predecesora, m,
				)
			},
		},
		{
			nombre: "publicar_tras_retirada", accion: AccionPublicarTrasRetiradaConvocatoria,
			version:        resultadoTrasRetirada.Publicada,
			actorEsperado:  resultadoTrasRetirada.Publicada.PublicadaPor,
			motivoEsperado: resultadoTrasRetirada.Publicada.MotivoPublicacion,
			construir: func(m CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error) {
				return MaterialPublicacionConvocatoria(
					borradorTrasRetiradaEsperado, resultadoTrasRetirada.Publicada,
					&predecesoraRetiradaEsperada, &resultadoTrasRetirada.Predecesora, m,
				)
			},
		},
		{
			nombre: "retirar", accion: AccionRetirarVersionConvocatoria, version: retirada,
			actorEsperado: retirada.RetiradaPor, motivoEsperado: retirada.MotivoRetirada,
			construir: func(m CompromisoMotivoGobiernoConvocatoria) (MaterialIntencionGobiernoConvocatoria, error) {
				return MaterialRetiradaConvocatoria(publicadaEsperada, retirada, m)
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
		"per_retira_01234567890123456789", aprobacion, "Retirada exacta de la convocatoria.", instante,
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

func compromisoMotivoConCorrelacionAlteradaPrueba(
	t *testing.T,
	original CompromisoMotivoGobiernoConvocatoria,
	correlacionRef string,
) CompromisoMotivoGobiernoConvocatoria {
	t.Helper()
	datos, err := original.DatosParaMaterial()
	if err != nil {
		t.Fatal(err)
	}
	datos.CorrelacionRef = correlacionRef
	alterada := CompromisoMotivoGobiernoConvocatoria{datos: &datos}
	if _, err := alterada.DatosParaMaterial(); err != nil {
		t.Fatalf("el ataque debe conservar datos estructuralmente validos: %v", err)
	}
	return alterada
}

type escenarioMaterializacionMotivoConvocatoria struct {
	version      dominiobolsa.VersionConvocatoriaGobernada
	compromiso   CompromisoMotivoGobiernoConvocatoria
	material     MaterialIntencionGobiernoConvocatoria
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacion
	testimonio   TestimonioIdempotenciaConvocatoria
	solicitud    SolicitudMaterializarSelladoMotivoGobiernoConvocatoria
	atestacion   AtestacionSelladoMotivoConvocatoria
}

func nuevoEscenarioMaterializacionMotivoConvocatoria(
	t *testing.T,
) escenarioMaterializacionMotivoConvocatoria {
	t.Helper()
	version := versionGobernadaPuertosPrueba(t)
	compromiso := compromisoMotivoConvocatoriaPrueba(
		t, AccionCrearBorradorConvocatoria, version,
		version.CreadaPor, version.MotivoCreacion, 'a',
	)
	material, err := MaterialAltaBorradorConvocatoria(version, nil, nil, compromiso)
	if err != nil {
		t.Fatal(err)
	}
	autorizacion := autorizacionMutacionConvocatoriaPrueba(t, material, version)
	testimonio := testimonioIdempotenciaConvocatoriaPrueba(t, material, autorizacion)
	solicitud, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		compromiso, material, autorizacion, testimonio, version,
		instanteGobiernoConvocatoriaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacion := atestacionMotivoMaterializadaConvocatoriaPrueba(
		t, compromiso, material, autorizacion, testimonio, version, 'a',
	)
	return escenarioMaterializacionMotivoConvocatoria{
		version: version, compromiso: compromiso, material: material,
		autorizacion: autorizacion, testimonio: testimonio,
		solicitud: solicitud, atestacion: atestacion,
	}
}

func TestAtestacionMotivoLigaIntencionDecisionActorCorrelacionEIdempotencia(t *testing.T) {
	escenario := nuevoEscenarioMaterializacionMotivoConvocatoria(t)
	preparacion := PreparacionTransaccionGobiernoConvocatoria{
		Material: escenario.material, Idempotencia: escenario.testimonio,
		Autorizacion: escenario.autorizacion, CompromisoMotivo: escenario.compromiso,
		SelladoMotivo: escenario.atestacion, SolicitadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := preparacion.ValidarPara(escenario.version); err != nil {
		t.Fatalf("preparacion exacta rechazada: %v", err)
	}
	datosOriginales, err := escenario.atestacion.DatosParaConsumo()
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*DatosAtestacionSelladoMotivoConvocatoria){
		"rotacion_hmac": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.HMAC.GeneracionClave++
			d.HMAC.ClaveHMACRef = "motivo-gobierno-v4"
			d.HMAC.ValorHMACSHA256 = huellaConvocatoriaPrueba('f')
		},
		"accion": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.Accion = AccionRetirarVersionConvocatoria
		},
		"convocatoria": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.ConvocatoriaRef = "proceso:bolsa:ajeno#1"
		},
		"principal": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.PrincipalRef = "per_actor_ajeno_012345678901"
		},
		"correlacion": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.CorrelacionRef = "correlacion:convocatoria:ajena"
		},
		"intencion": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.HuellaIntencionSHA256 = huellaConvocatoriaPrueba('d')
		},
		"decision_ref": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.DecisionRef = "autorizacion:convocatoria:ajena"
		},
		"decision_huella": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.HuellaDecisionSHA256 = huellaConvocatoriaPrueba('d')
		},
		"indice_idempotencia": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.IndiceIdempotenciaHMACSHA256 = hmacConvocatoriaPrueba("indice-ajeno", 'd')
		},
		"atestacion_idempotencia": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.AtestacionIdempotenciaRef = "atestacion:idempotencia:ajena"
		},
		"huella_atestacion_idempotencia": func(d *DatosAtestacionSelladoMotivoConvocatoria) {
			d.HuellaAtestacionIdempotenciaSHA256 = huellaConvocatoriaPrueba('d')
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := datosOriginales
			alterar(&datos)
			if _, err := NuevaAtestacionSelladoMotivoConvocatoria(
				escenario.solicitud, datos,
			); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
				t.Fatalf("constructor acepto atestacion cruzada: %v", err)
			}
			preparacionAlterada := preparacion
			preparacionAlterada.SelladoMotivo = AtestacionSelladoMotivoConvocatoria{datos: &datos}
			if err := preparacionAlterada.ValidarPara(escenario.version); !errors.Is(
				err, ErrConfirmacionGobiernoConvocatoriaInvalida,
			) {
				t.Fatalf("preparacion acepto atestacion cruzada: %v", err)
			}
		})
	}
}

func TestMaterializacionMotivoEsReconciliableEIdempotentePorIdentidad(t *testing.T) {
	escenario := nuevoEscenarioMaterializacionMotivoConvocatoria(t)
	reintento, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		escenario.compromiso, escenario.material, escenario.autorizacion,
		escenario.testimonio, escenario.version, instanteGobiernoConvocatoriaPrueba.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaInicial, errInicial := escenario.solicitud.HuellaReconciliacionSHA256()
	huellaReintento, errReintento := reintento.HuellaReconciliacionSHA256()
	if errInicial != nil || errReintento != nil ||
		!huellaMotivoGobiernoIgualConstante(huellaInicial, huellaReintento) {
		t.Fatalf("el reintento cambio la identidad reconciliable: %v / %v", errInicial, errReintento)
	}
	datosAtestacion, _ := escenario.atestacion.DatosParaConsumo()
	if _, err := NuevaAtestacionSelladoMotivoConvocatoria(reintento, datosAtestacion); err != nil {
		t.Fatalf("el reintento no pudo recuperar la misma atestacion: %v", err)
	}

	datosTestimonio, _ := escenario.testimonio.Datos()
	datosTestimonio.IndiceOperacionHMACSHA256 = hmacConvocatoriaPrueba("indice-colision", 'e')
	datosTestimonio.AtestacionRef = "atestacion:idempotencia:colision"
	datosTestimonio.HuellaAtestacionSHA256 = huellaConvocatoriaPrueba('e')
	testimonioColision, err := NuevoTestimonioIdempotenciaConvocatoria(datosTestimonio)
	if err != nil {
		t.Fatal(err)
	}
	solicitudColision, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		escenario.compromiso, escenario.material, escenario.autorizacion,
		testimonioColision, escenario.version, instanteGobiernoConvocatoriaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaColision, err := solicitudColision.HuellaReconciliacionSHA256()
	if err != nil || huellaMotivoGobiernoIgualConstante(huellaInicial, huellaColision) {
		t.Fatalf("una idempotencia distinta compartio identidad reconciliable: %v", err)
	}
	if _, err := NuevaAtestacionSelladoMotivoConvocatoria(
		solicitudColision, datosAtestacion,
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("la colision recupero una atestacion ajena: %v", err)
	}
}

func TestMaterializacionMotivoAcotaVigenciasDeDecisionEIdempotencia(t *testing.T) {
	escenario := nuevoEscenarioMaterializacionMotivoConvocatoria(t)
	datosAutorizacion, _ := escenario.autorizacion.Datos()
	datosAtestacion, _ := escenario.atestacion.DatosParaConsumo()
	datosAtestacion.AtestacionValidaHasta = datosAutorizacion.Decision.ValidaHasta
	if _, err := NuevaAtestacionSelladoMotivoConvocatoria(
		escenario.solicitud, datosAtestacion,
	); err != nil {
		t.Fatalf("frontera exacta de la decision rechazada: %v", err)
	}
	datosAtestacion.AtestacionValidaHasta = datosAutorizacion.Decision.ValidaHasta.Add(time.Microsecond)
	if _, err := NuevaAtestacionSelladoMotivoConvocatoria(
		escenario.solicitud, datosAtestacion,
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("atestacion posterior a la decision aceptada: %v", err)
	}

	datosTestimonio, _ := escenario.testimonio.Datos()
	datosTestimonio.ValidoHasta = instanteGobiernoConvocatoriaPrueba
	testimonioCaducado, err := NuevoTestimonioIdempotenciaConvocatoria(datosTestimonio)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		escenario.compromiso, escenario.material, escenario.autorizacion,
		testimonioCaducado, escenario.version, instanteGobiernoConvocatoriaPrueba,
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("testimonio caducado materializo un token: %v", err)
	}
	datosTestimonio, _ = escenario.testimonio.Datos()
	datosTestimonio.EmitidoEn = instanteGobiernoConvocatoriaPrueba.Add(time.Microsecond)
	testimonioFuturo, err := NuevoTestimonioIdempotenciaConvocatoria(datosTestimonio)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		escenario.compromiso, escenario.material, escenario.autorizacion,
		testimonioFuturo, escenario.version, instanteGobiernoConvocatoriaPrueba,
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("testimonio aun no emitido materializo un token: %v", err)
	}

	datosTestimonio, _ = escenario.testimonio.Datos()
	datosTestimonio.ValidoHasta = instanteGobiernoConvocatoriaPrueba.Add(2 * time.Minute)
	testimonioCorto, err := NuevoTestimonioIdempotenciaConvocatoria(datosTestimonio)
	if err != nil {
		t.Fatal(err)
	}
	solicitudCorta, err := NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
		escenario.compromiso, escenario.material, escenario.autorizacion,
		testimonioCorto, escenario.version, instanteGobiernoConvocatoriaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosAtestacion, _ = escenario.atestacion.DatosParaConsumo()
	datosAtestacion.AtestacionValidaHasta = datosTestimonio.ValidoHasta
	atestacionCorta, err := NuevaAtestacionSelladoMotivoConvocatoria(solicitudCorta, datosAtestacion)
	if err != nil {
		t.Fatal(err)
	}
	preparacion := PreparacionTransaccionGobiernoConvocatoria{
		Material: escenario.material, Idempotencia: testimonioCorto,
		Autorizacion: escenario.autorizacion, CompromisoMotivo: escenario.compromiso,
		SelladoMotivo: atestacionCorta, SolicitadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := preparacion.ValidarPara(escenario.version); err != nil {
		t.Fatalf("atestacion acotada por idempotencia rechazada: %v", err)
	}
	datosAtestacion.AtestacionValidaHasta = datosTestimonio.ValidoHasta.Add(time.Microsecond)
	preparacion.SelladoMotivo = AtestacionSelladoMotivoConvocatoria{datos: &datosAtestacion}
	if err := preparacion.ValidarPara(escenario.version); !errors.Is(
		err, ErrConfirmacionGobiernoConvocatoriaInvalida,
	) {
		t.Fatalf("revalidacion durable acepto sello posterior a idempotencia: %v", err)
	}
}

func TestRevalidacionDurableRechazaAtestacionEmitidaAntesDeSusPruebas(t *testing.T) {
	escenario := nuevoEscenarioMaterializacionMotivoConvocatoria(t)
	datosAutorizacion, err := escenario.autorizacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosIdempotencia, err := escenario.testimonio.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosOriginales, err := escenario.atestacion.DatosParaConsumo()
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]time.Time{
		"antes_autorizacion": datosAutorizacion.VerificadaEn.Add(-time.Microsecond),
		"antes_idempotencia": datosIdempotencia.EmitidoEn.Add(-time.Microsecond),
	}
	for nombre, emitidaEn := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := datosOriginales
			datos.AtestacionEmitidaEn = emitidaEn
			if err := validarDatosAtestacionSelladoMotivo(datos); err != nil {
				t.Fatalf("la falsificacion no quedo estructuralmente valida: %v", err)
			}
			preparacion := PreparacionTransaccionGobiernoConvocatoria{
				Material: escenario.material, Idempotencia: escenario.testimonio,
				Autorizacion: escenario.autorizacion, CompromisoMotivo: escenario.compromiso,
				SelladoMotivo: AtestacionSelladoMotivoConvocatoria{datos: &datos},
				SolicitadaEn:  instanteGobiernoConvocatoriaPrueba,
			}
			if err := preparacion.ValidarPara(escenario.version); !errors.Is(
				err, ErrConfirmacionGobiernoConvocatoriaInvalida,
			) {
				t.Fatalf("revalidacion durable acepto una atestacion anticipada: %v", err)
			}
		})
	}
}

func TestCompromisoMotivoAdmiteRotacionSinConfundirHMAC(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	semantica := SolicitudSemanticaMotivoGobiernoConvocatoria{
		DominioCriptografico: DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		Accion:               AccionCrearBorradorConvocatoria, ConvocatoriaRef: version.Referencia(),
		PrincipalRef: version.CreadaPor, CorrelacionRef: "correlacion:convocatoria:001",
		Motivo: version.MotivoCreacion, SolicitadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	huella, err := semantica.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hmacV3 := hmacMotivoConvocatoriaPrueba('a', huella)
	hmacV4 := hmacV3
	hmacV4.GeneracionClave = 4
	hmacV4.ClaveHMACRef = "motivo-gobierno-v4"
	hmacV4.ValorHMACSHA256 = huellaConvocatoriaPrueba('b')
	compromisoV3, errV3 := NuevoCompromisoMotivoGobiernoConvocatoria(semantica, hmacV3)
	compromisoV4, errV4 := NuevoCompromisoMotivoGobiernoConvocatoria(semantica, hmacV4)
	materialV3, errMaterialV3 := MaterialAltaBorradorConvocatoria(version, nil, nil, compromisoV3)
	materialV4, errMaterialV4 := MaterialAltaBorradorConvocatoria(version, nil, nil, compromisoV4)
	huellaV3, errHuellaV3 := materialV3.HuellaSHA256()
	huellaV4, errHuellaV4 := materialV4.HuellaSHA256()
	if errV3 != nil || errV4 != nil || errMaterialV3 != nil || errMaterialV4 != nil ||
		errHuellaV3 != nil || errHuellaV4 != nil || huellaMotivoGobiernoIgualConstante(huellaV3, huellaV4) ||
		hmacV3.igualConstante(hmacV4) {
		t.Fatalf("rotacion de HMAC no quedo separada: %v %v %v %v", errV3, errV4, errMaterialV3, errMaterialV4)
	}
	hmacCruzado := hmacV3
	hmacCruzado.HuellaEntradaSHA256 = huellaConvocatoriaPrueba('c')
	if _, err := NuevoCompromisoMotivoGobiernoConvocatoria(
		semantica, hmacCruzado,
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("HMAC de otra entrada aceptado: %v", err)
	}
}
