package gobiernoconvocatorias

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestResolucionDiarioImpideMezclarAliasPrimariaYRecibo(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	consulta, err := solicitudConsultaEscenario(t, e)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := e.diario.ConsultarIdentidades(context.Background(), consulta)
	if err != nil || resultado.ValidarPara(consulta) != nil || len(resultado.Coincidencias) != 1 {
		t.Fatalf("proyección del diario inválida: %v", err)
	}
	original := resultado.Coincidencias[0]
	if len(original.Resolucion.IdentidadesConsultadas) != 2 {
		t.Fatal("la resolución no incluyó todos los aliases encontrados")
	}

	t.Run("consulta", func(t *testing.T) {
		alterado := resultado
		alterado.Coincidencias = append([]CoincidenciaIdentidadBorrador(nil), resultado.Coincidencias...)
		alterado.Coincidencias[0].Resolucion.IdentidadPrimaria = consulta.Identidades[1]
		if alterado.ValidarPara(consulta) == nil {
			t.Fatal("se mezcló la resolución de aliases con otra primaria")
		}
	})

	t.Run("orden aliases", func(t *testing.T) {
		alterado := original.Resolucion
		alterado.IdentidadesConsultadas = append(
			[]ProyeccionIdentidadOperacion(nil), alterado.IdentidadesConsultadas...,
		)
		alterado.IdentidadesConsultadas[0], alterado.IdentidadesConsultadas[1] =
			alterado.IdentidadesConsultadas[1], alterado.IdentidadesConsultadas[0]
		if alterado.validarPara(consulta.Identidades) {
			t.Fatal("se aceptaron aliases reordenados frente a la consulta canónica")
		}
	})

	t.Run("replay", func(t *testing.T) {
		resultadoAjeno := original.Resultado
		copia := *resultadoAjeno.Recibo
		copia.IdentidadPrimaria = consulta.Identidades[1]
		resultadoAjeno.Recibo = &copia
		if _, err := e.servicio.resolverResultadoDiario(context.Background(),
			resultadoAjeno, original.Resolucion.IdentidadPrimaria,
		); err == nil {
			t.Fatal("un recibo de otra primaria se aceptó en replay")
		}
	})

	if !reflect.DeepEqual(recibo, *original.Resultado.Recibo) {
		t.Fatal("la consulta válida no conservó el recibo original")
	}
}

func TestResolucionDiarioCierraRecuperacionYReclamacionAlteradas(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	e.diario.fallarTrasReserva = true
	if _, err := e.servicio.Crear(context.Background(), e.orden); !errors.Is(err, ErrOperacionBorradorIndeterminada) {
		t.Fatalf("no se preparó la recuperación: %v", err)
	}
	consulta, err := solicitudConsultaEscenario(t, e)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := e.diario.ConsultarIdentidades(context.Background(), consulta)
	if err != nil || len(resultado.Coincidencias) != 1 {
		t.Fatalf("reserva recuperable ausente: %v", err)
	}
	if resultado.ValidarPara(consulta) != nil {
		t.Fatal("la resolución confiable del diario no superó la validación estructural")
	}

	e.reloj.avanzar(3 * time.Minute)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	if e.diario.ultimaReclamacion == nil || e.diario.ultimaReclamacion.Validar() != nil {
		t.Fatal("no se conservó una reclamación válida de control")
	}
	reclamacion := *e.diario.ultimaReclamacion
	reclamacion.ResolucionAnterior.IdentidadPrimaria =
		reclamacion.ResolucionAnterior.IdentidadesConsultadas[len(reclamacion.ResolucionAnterior.IdentidadesConsultadas)-1]
	if reclamacion.Validar() == nil {
		t.Fatal("la reclamación aceptó una primaria ajena a la resolución anterior")
	}
}

func TestRotacionAdmitePrimariaHistoricaMasAntiguaQueAlias(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 1)
	original, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	intencion, err := nuevaIntencionAltaBorradorCanonica(
		e.catalogo.plantilla.Referencia, e.orden.CodigoVersionPublica, e.orden.ExpedienteRef,
		e.orden.Contenido, e.orden.MotivoCatalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := nuevaSolicitudDerivacionIdempotencia(e.orden.ClaveCliente, intencion, e.orden.Actor)
	if err != nil {
		t.Fatal(err)
	}
	conjunto, err := (derivadorPrueba{generaciones: []uint32{3, 2}}).Derivar(
		context.Background(), solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	identidades, err := conjunto.proyecciones()
	if err != nil {
		t.Fatal(err)
	}
	e.diario.mu.Lock()
	e.diario.aliases[claveL(identidades[1])] = aliasDiarioPrueba{
		identidad: identidades[1], clavePrimaria: claveL(original.IdentidadPrimaria),
	}
	e.diario.mu.Unlock()

	replay, err := e.reiniciar(t, 3, 2).Crear(context.Background(), e.orden)
	if err != nil || !reflect.DeepEqual(original, replay) {
		t.Fatalf("alias g2 no resolvió la primaria histórica g1: %v", err)
	}
	if replay.IdentidadPrimaria.Localizador.GeneracionClave != 1 || e.confirmador.llamadas != 1 {
		t.Fatal("el replay sustituyó la primaria histórica o repitió el efecto")
	}
}

func TestReservaAjenaSeProyectaEnCursoConVentanaCompleta(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	if e.diario.ultima == nil || len(e.diario.ultima.IdentidadesConsulta) != 2 {
		t.Fatal("solicitud de reserva de referencia ausente")
	}
	solicitud := *e.diario.ultima
	primariaHistorica := solicitud.IdentidadesConsulta[1]
	final, err := resolucionIdentidadPrueba(
		primariaHistorica, solicitud.IdentidadesConsulta, solicitud.SolicitadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	control := ResultadoOperacionDiario{
		Estado: ResultadoDiarioEnCurso, Revision: 2, Cercado: 1,
		ArrendamientoIniciaEn: solicitud.Proyeccion.ArrendamientoIniciaEn.Add(-time.Second),
		ArrendamientoVenceEn:  solicitud.Proyeccion.ArrendamientoVenceEn.Add(-time.Second),
	}
	merge := ResultadoReservaDecisionBorrador{
		Resolucion: final, Resultado: control,
	}
	if err := merge.ValidarPara(solicitud); err != nil {
		t.Fatalf("reserva ajena g3/g2 sobre primaria g2 fue rechazada: %v", err)
	}
	reservadaAjena := merge
	reservadaAjena.Resultado.Estado = ResultadoDiarioReservado
	if reservadaAjena.ValidarPara(solicitud) == nil {
		t.Fatal("una reserva ajena se presentó como creada por la solicitud perdedora")
	}

	resolucionParcial, err := resolucionIdentidadPrueba(
		primariaHistorica,
		[]ProyeccionIdentidadOperacion{solicitud.IdentidadesConsulta[1]},
		solicitud.SolicitadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	mergeParcial := merge
	mergeParcial.Resolucion = resolucionParcial
	if mergeParcial.ValidarPara(solicitud) == nil {
		t.Fatal("una reserva ajena sin acreditar la ventana completa fue aceptada")
	}
}

func TestCarreraDeReservaNoConfirmaLaReservaAutoritativaAjena(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	e.diario.forzarMergeG2 = true
	_, err := e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrOperacionBorradorEnCurso) {
		t.Fatalf("la reserva ajena no se proyectó en curso: %v", err)
	}
	if e.confirmador.ultima != nil || e.diario.ultima == nil {
		t.Fatal("la reserva ajena alcanzó el confirmador")
	}
	propuestaLocal := e.diario.ultima.Proyeccion.IdentidadPrimaria
	if propuestaLocal.Localizador.GeneracionClave != 3 {
		t.Fatalf("la propuesta local perdió su identidad: generación=%d",
			propuestaLocal.Localizador.GeneracionClave)
	}
	if e.confirmador.efectos != 0 || e.cifrador.llamadas != 0 {
		t.Fatal("la carrera ajena produjo cifrado o efectos")
	}
}

func TestAutoridadPoliticaIndependienteRechazaPerfilDegradadoAntesDeCifrar(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	e.perfiles.degradar = true
	_, err := e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrResultadoBorradorInseguro) {
		t.Fatalf("perfil autoaprobado no falló cerrado: %v", err)
	}
	if e.politicas.llamadas != 1 || e.perfiles.llamadas != 1 ||
		e.cifrador.llamadas != 0 || e.confirmador.llamadas != 0 {
		t.Fatalf("el downgrade alcanzó cifrado: política=%d resolución=%d cifrado=%d confirmación=%d",
			e.politicas.llamadas, e.perfiles.llamadas, e.cifrador.llamadas, e.confirmador.llamadas)
	}
}

func TestConfirmacionNoAceptaAcreditacionAusenteRevocadaOAjena(t *testing.T) {
	casos := []modoConfirmacion{
		confirmarSinAcreditacion,
		confirmarAcreditacionRevocada,
		confirmarAcreditacionAjena,
	}
	for _, modo := range casos {
		t.Run(string(modo), func(t *testing.T) {
			e := nuevoEscenario(t, modo, 3, 2)
			_, err := e.servicio.Crear(context.Background(), e.orden)
			if !errors.Is(err, ErrOperacionBorradorIndeterminada) {
				t.Fatalf("confirmación KMS insegura devolvió éxito: %v", err)
			}
			if e.confirmador.efectos != 1 {
				t.Fatal("la respuesta insegura debe tratarse como commit de desenlace desconocido")
			}
		})
	}
}

func TestAcreditacionKMSLigaTodosLosControlesDeConfirmacion(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil || e.confirmador.ultima == nil {
		t.Fatalf("confirmación de referencia ausente: %v", err)
	}
	confirmacion := *e.confirmador.ultima
	original := recibo.AcreditacionKMS
	resultadoOriginal := ResultadoConfirmacionAtomica{
		Estado: ResultadoDiarioConfirmado, Recibo: recibo, AcreditacionKMS: original,
	}
	if resultadoOriginal.ValidarPara(confirmacion) != nil {
		t.Fatal("la acreditación de referencia no valida")
	}
	perfilAjeno, err := NuevoPerfilCifradoBorrador(
		"perfil:cifrado:borradores:ajeno", 2, huellaHexPrueba('0'),
		"algoritmo-aead-ajeno", "algoritmo-envoltura-ajeno",
	)
	if err != nil {
		t.Fatal(err)
	}
	identidadAjena := e.diario.ultima.IdentidadesConsulta[1]
	mutaciones := map[string]func(*AcreditacionKMSConfirmacionBorrador){
		"atestacion":    func(a *AcreditacionKMSConfirmacionBorrador) { a.AtestacionRef = "atestacion:kms:ajena" },
		"perfil":        func(a *AcreditacionKMSConfirmacionBorrador) { a.Perfil = perfilAjeno },
		"clave maestra": func(a *AcreditacionKMSConfirmacionBorrador) { a.ClaveMaestraRef = "clave:kms:ajena" },
		"version clave": func(a *AcreditacionKMSConfirmacionBorrador) { a.VersionClave++ },
		"aad":           func(a *AcreditacionKMSConfirmacionBorrador) { a.HuellaAAD = huellaHexPrueba('1') },
		"envoltura":     func(a *AcreditacionKMSConfirmacionBorrador) { a.HuellaEnvolturaSHA256 = huellaHexPrueba('2') },
		"sobre":         func(a *AcreditacionKMSConfirmacionBorrador) { a.HuellaSobreSHA256 = huellaHexPrueba('3') },
		"identidad":     func(a *AcreditacionKMSConfirmacionBorrador) { a.IdentidadPrimaria = identidadAjena },
		"revision":      func(a *AcreditacionKMSConfirmacionBorrador) { a.RevisionReserva++ },
		"cercado":       func(a *AcreditacionKMSConfirmacionBorrador) { a.Cercado++ },
		"lease": func(a *AcreditacionKMSConfirmacionBorrador) {
			a.ArrendamientoVenceEn = a.ArrendamientoVenceEn.Add(-time.Microsecond)
		},
		"transaccion": func(a *AcreditacionKMSConfirmacionBorrador) { a.TransaccionRef = "transaccion:ajena" },
		"recibo":      func(a *AcreditacionKMSConfirmacionBorrador) { a.ReciboRef = "recibo:ajeno" },
		"tiempo atestacion": func(a *AcreditacionKMSConfirmacionBorrador) {
			a.AtestacionEmitidaEn = a.AtestacionEmitidaEn.Add(time.Microsecond)
		},
		"tiempo confirmacion": func(a *AcreditacionKMSConfirmacionBorrador) {
			a.ConfirmacionSolicitadaEn = a.ConfirmacionSolicitadaEn.Add(time.Microsecond)
		},
		"misma clave para emision y revalidacion": func(a *AcreditacionKMSConfirmacionBorrador) {
			a.FirmaRevalidacionKMS.VerificadorRef = a.FirmaAtestacionKMS.VerificadorRef
			a.FirmaRevalidacionKMS.HuellaClavePublicaSHA256 =
				a.FirmaAtestacionKMS.HuellaClavePublicaSHA256
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterada := original
			mutar(&alterada)
			alterada.HuellaAcreditacionSHA256 = alterada.calcularHuella()
			reciboAlterado := recibo
			reciboAlterado.AcreditacionKMS = alterada
			resultado := ResultadoConfirmacionAtomica{
				Estado: ResultadoDiarioConfirmado, Recibo: reciboAlterado,
				AcreditacionKMS: alterada,
			}
			if resultado.ValidarPara(confirmacion) == nil {
				t.Fatal("la acreditación se reutilizó fuera de su vínculo")
			}
		})
	}
}

func TestAcreditacionKMSLigaElCuerpoCompletoDelRecibo(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	original := recibo.AcreditacionKMS
	mutaciones := map[string]func(*ProyeccionReciboBorrador){
		"accion": func(r *ProyeccionReciboBorrador) {
			r.Accion = "bolsa.convocatoria.borrador.actualizar"
		},
		"estado": func(r *ProyeccionReciboBorrador) { r.EstadoPrincipal.Revision++ },
		"decision": func(r *ProyeccionReciboBorrador) {
			r.Decision.DecisionRef = "decision:pdp:otra"
		},
		"sellado": func(r *ProyeccionReciboBorrador) {
			r.SelladoMotivo.TokenConsumoRef = "consumo:sellado:otro"
		},
		"auditoria": func(r *ProyeccionReciboBorrador) { r.AuditoriaRef = "auditoria:otra" },
		"outbox":    func(r *ProyeccionReciboBorrador) { r.EventoOutboxRef = "outbox:otro" },
		"procedencia": func(r *ProyeccionReciboBorrador) {
			r.Procedencia.ProveedorRef = "proveedor-pruebas-alterado"
		},
		"instante": func(r *ProyeccionReciboBorrador) {
			r.ConfirmadaEn = r.ConfirmadaEn.Add(time.Microsecond)
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterado := recibo
			mutar(&alterado)
			if original.validaParaRecibo(alterado) {
				t.Fatal("la acreditación aceptó un cuerpo de recibo distinto")
			}
		})
	}
}

func TestVerificadorReciboRechazaFirmasKMSAlteradas(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*AcreditacionKMSConfirmacionBorrador){
		"atestacion": func(a *AcreditacionKMSConfirmacionBorrador) {
			a.FirmaAtestacionKMS = alterarFirmaEvidenciaPrueba(t, a.FirmaAtestacionKMS)
		},
		"revalidacion": func(a *AcreditacionKMSConfirmacionBorrador) {
			a.FirmaRevalidacionKMS = alterarFirmaEvidenciaPrueba(t, a.FirmaRevalidacionKMS)
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterado := recibo
			acreditacion := alterado.AcreditacionKMS
			mutar(&acreditacion)
			acreditacion.HuellaAcreditacionSHA256 = acreditacion.calcularHuella()
			alterado.AcreditacionKMS = acreditacion
			if err := e.verificador.VerificarReciboBorrador(context.Background(), alterado); err == nil {
				t.Fatal("el verificador aceptó una firma KMS alterada")
			}
		})
	}
}

func TestVerificadorReciboRechazaRecalculoDeHuellasSinClave(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	alterado := recibo
	alterado.AuditoriaRef = "auditoria:recalculada-por-intruso"
	acreditacion := alterado.AcreditacionKMS
	acreditacion.HuellaCuerpoReciboSHA256 = huellaCuerpoReciboBorrador(alterado)
	acreditacion.HuellaAcreditacionSHA256 = acreditacion.calcularHuella()
	alterado.AcreditacionKMS = acreditacion
	if !acreditacion.validaParaRecibo(alterado) {
		t.Fatal("el escenario no reconstruyó de forma coherente las huellas sin clave")
	}
	if err := e.verificador.VerificarReciboBorrador(context.Background(), alterado); err == nil {
		t.Fatal("el verificador aceptó un recibo rehecho sin la clave de revalidación")
	}
}

func TestFirmasKMSPersistidasSeRehidratanSinClavePrivada(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	rehidratado := recibo
	acreditacion := rehidratado.AcreditacionKMS
	acreditacion.FirmaAtestacionKMS = restaurarFirmaEvidenciaPrueba(
		t, acreditacion.FirmaAtestacionKMS,
	)
	acreditacion.FirmaRevalidacionKMS = restaurarFirmaEvidenciaPrueba(
		t, acreditacion.FirmaRevalidacionKMS,
	)
	rehidratado.AcreditacionKMS = acreditacion
	if err := e.verificador.VerificarReciboBorrador(context.Background(), rehidratado); err != nil {
		t.Fatalf("el verificador solo-públicas rechazó el recibo rehidratado: %v", err)
	}
}

func TestFirmaParaPersistenciaRechazaMetadatosAlterados(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	original := recibo.AcreditacionKMS.FirmaAtestacionKMS
	mutaciones := map[string]func(*FirmaEvidenciaBorrador){
		"algoritmo":     func(f *FirmaEvidenciaBorrador) { f.AlgoritmoFirma = "" },
		"verificador":   func(f *FirmaEvidenciaBorrador) { f.VerificadorRef = "" },
		"clave publica": func(f *FirmaEvidenciaBorrador) { f.HuellaClavePublicaSHA256 = "" },
		"preimagen":     func(f *FirmaEvidenciaBorrador) { f.HuellaPreimagenSHA256 = "" },
		"firma":         func(f *FirmaEvidenciaBorrador) { f.firmaBase64URLSinRelleno = "!" },
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterada := original
			mutar(&alterada)
			if persistida, err := alterada.FirmaBase64URLParaPersistencia(); !errors.Is(err, ErrRevalidacionKMSBorradorFallo) || persistida != "" {
				t.Fatalf("la firma alterada se expuso para persistencia: %q, %v", persistida, err)
			}
		})
	}
}

func restaurarFirmaEvidenciaPrueba(
	t *testing.T,
	firma FirmaEvidenciaBorrador,
) FirmaEvidenciaBorrador {
	t.Helper()
	representacion, err := firma.FirmaBase64URLParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	restaurada, err := RestaurarFirmaEvidenciaBorradorPersistida(
		firma.AlgoritmoFirma, firma.VerificadorRef,
		firma.HuellaClavePublicaSHA256, firma.HuellaPreimagenSHA256,
		representacion,
	)
	if err != nil || restaurada != firma {
		t.Fatalf("firma durable no rehidratada exactamente: %v", err)
	}
	return restaurada
}

func alterarFirmaEvidenciaPrueba(
	t *testing.T,
	firma FirmaEvidenciaBorrador,
) FirmaEvidenciaBorrador {
	t.Helper()
	representacion, err := firma.FirmaBase64URLParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	bytesFirma, err := base64.RawURLEncoding.DecodeString(representacion)
	if err != nil || len(bytesFirma) == 0 {
		t.Fatalf("firma de prueba inválida: %v", err)
	}
	bytesFirma[0] ^= 0x01
	firma.firmaBase64URLSinRelleno = base64.RawURLEncoding.EncodeToString(bytesFirma)
	return firma
}

func TestPerfilDesarrolloNuncaPuedeDeclararseAutoritativo(t *testing.T) {
	procedencia, err := NuevaProcedenciaActoBorrador(
		"desarrollo", AutoridadActoAutoritativa,
		"proveedor:seguridad:desarrollo:t21", true,
	)
	if err == nil || procedencia != (ProcedenciaActoBorrador{}) {
		t.Fatalf("el núcleo aceptó reetiquetar desarrollo como autoritativo: %+v, %v", procedencia, err)
	}
}
