package ports

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var instantePuertoCarga = time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)

func recursoBaseCargaPuertoPrueba() domain.RecursoAutorizable {
	return domain.RecursoAutorizable{
		Referencia: "solicitud:0123456789abcdef", ModuloID: "bolsa", Tipo: "solicitud",
		Ambitos:   map[string]string{"proceso": "proceso:0123456789abcdef"},
		Atributos: map[string]string{"categoria": "auxiliar_administrativo"},
	}
}

func cargaReservadaPuertoPrueba(t *testing.T) domain.CargaDocumental {
	t.Helper()
	carga, err := domain.NuevaCargaDocumental(
		"carga:puerto:0123456789abcdef", "persona:0123456789abcdef", "solicitud:0123456789abcdef",
		"bolsa", "solicitud", "operacion:carga:0123456789abcdef", "correlacion:0123456789abcdef",
		"aportar_documentacion_bolsa", "datos_personales", "application/pdf", 2048,
		strings.Repeat("a", 64), "hmac-sha256:idempotencia_v1:"+strings.Repeat("c", 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("b", 64),
		instantePuertoCarga, instantePuertoCarga.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	return carga
}

func TestPreimagenReservaCargaDocumentalV3VinculaReservaYContenido(t *testing.T) {
	carga := cargaReservadaPuertoPrueba(t)
	token, errToken := NuevoTokenReservaCargaDocumental()
	if errToken != nil {
		t.Fatal(errToken)
	}
	solicitud := SolicitudReservarCargaDocumental{
		Token:                  token,
		IndiceIdempotenciaHMAC: carga.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    carga.HuellaSolicitudHMAC,
		Carga:                  carga,
		DecisionPreparacion: ConsumoDecisionPreparacionCargaDocumentalV1{
			DecisionRef:            "decision:preparar:0123456789abcdef",
			EfectoRef:              "efecto:carga:preparar:0123456789abcdef",
			HuellaPlanEfectoSHA256: strings.Repeat("e", 64),
			EsquemaHuellaDecision:  EsquemaHuellaDecisionAutorizacionReforzadaV1,
			HuellaDecisionSHA256:   strings.Repeat("f", 64),
		},
		SolicitadaEn: instantePuertoCarga, ReservaExpiraEn: carga.ExpiraEn,
	}
	reserva := ReservaRecursoCargaDocumentalV3{
		RecursoRef: carga.RecursoRef, PersonaRef: carga.PrincipalID,
		ClaveReserva: "reserva:0123456789abcdef", HuellaReservaSHA256: strings.Repeat("a", 64),
		VersionReserva: 1,
	}
	preimagen, err := PreimagenReservaCargaDocumentalV3(solicitud, reserva)
	if err != nil || HuellaPreimagenCargaDocumentalV3(preimagen) == "" {
		t.Fatalf("reserva válida no produjo preimagen: %v", err)
	}
	cruce := reserva
	cruce.PersonaRef = "persona:ajena:0123456789abcdef"
	if _, err := PreimagenReservaCargaDocumentalV3(solicitud, cruce); !errors.Is(err, ErrReservaCargaDocumentalInvalida) {
		t.Fatalf("persona ajena admitida: %v", err)
	}
	otraVersion := reserva
	otraVersion.VersionReserva++
	otra, err := PreimagenReservaCargaDocumentalV3(solicitud, otraVersion)
	if err != nil || HuellaPreimagenCargaDocumentalV3(preimagen) == HuellaPreimagenCargaDocumentalV3(otra) {
		t.Fatal("la versión D16 no queda ligada al efecto V3")
	}
}

func TestAutorizacionCargaDocumentalV3RechazaAusenciaYCaducidad(t *testing.T) {
	actor, ahora, _ := escenarioOrdenAcreditacionContextoActorV2Prueba(t)
	carga, err := domain.NuevaCargaDocumental(
		"carga:puerto:0123456789abcdef", actor.Contexto.PersonaRef,
		"ref:"+strings.Repeat("a", 64), "meritos", "hecho", "operacion:carga:0123456789abcdef",
		"correlacion:0123456789abcdef", "aportar_documentacion_bolsa", "datos_personales",
		"application/pdf", 2048, strings.Repeat("b", 64),
		"hmac-sha256:idempotencia_v1:"+strings.Repeat("c", 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("d", 64), ahora, ahora.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	preimagen := []byte(`{"accion":"documentos.carga.preparar_reserva","carga_id":"carga:puerto:0123456789abcdef"}`)
	base := AutorizacionCargaDocumentalV3{
		Accion: AccionCargaDocumentalPrepararReservaV3, Finalidad: carga.Finalidad,
		RecursoRef: carga.RecursoRef, PersonaRef: carga.PrincipalID,
		PerfilActivoRef: actor.Contexto.PerfilActivoRef, CorrelacionRef: carga.CorrelacionRef,
	}
	if err := base.ValidarPara(AccionCargaDocumentalPrepararReservaV3, preimagen, actor, carga, ahora); !errors.Is(err, ErrDecisionPreparacionCargaNoDisponible) {
		t.Fatalf("material V3 ausente admitido: %v", err)
	}
	resumen, err := NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:carga:0123456789abcdef", strings.Repeat("1", 64), strings.Repeat("2", 64),
		"contexto:carga:0123456789abcdef", strings.Repeat("3", 64),
		AccionCargaDocumentalPrepararReservaV3, carga.ID, HuellaPreimagenCargaDocumentalV3(preimagen),
		AudienciaCargaDocumentalPrepararReservaV3, ahora.Add(-4*time.Second), ahora.Add(-time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := hex.DecodeString(spkiMaterialPostgreSQLV3Hex)
	if err != nil {
		t.Fatal(err)
	}
	base.Material, err = NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(capacidadCanonicaPostgreSQLV3), resumen,
		[]byte("decision-canonica"), []byte("motivo-canonico"), []byte("contexto-canonico"),
		3, 4, []byte("payload-ad3"), []byte("sobre-cose"), []byte("evidencia"),
		spki,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.ValidarPara(AccionCargaDocumentalPrepararReservaV3, preimagen, actor, carga, ahora); !errors.Is(err, ErrDecisionPreparacionCargaNoDisponible) {
		t.Fatalf("material V3 caducado admitido: %v", err)
	}
	if err := base.ValidarPara(accionCargaDocumentalConfirmar, preimagen, actor, carga, ahora.Add(-2*time.Second)); !errors.Is(err, ErrDecisionPreparacionCargaNoDisponible) {
		t.Fatalf("audiencia de otro paso admitida: %v", err)
	}
	if err := base.ValidarPara(AccionCargaDocumentalPrepararConfirmacionV3, preimagen,
		actor, carga, ahora.Add(-2*time.Second)); !errors.Is(err, ErrDecisionPreparacionCargaNoDisponible) {
		t.Fatalf("decision de reserva usada en TX de confirmacion: %v", err)
	}
}

func TestPreparacionEmisionReciboV3EsOpacaYUnSoloIntento(t *testing.T) {
	actor, _, _ := escenarioOrdenAcreditacionContextoActorV2Prueba(t)
	carga, err := domain.NuevaCargaDocumental(
		"carga:puerto:0123456789abcdef", actor.Contexto.PersonaRef,
		"solicitud:0123456789abcdef", "bolsa", "solicitud", "operacion:carga:0123456789abcdef",
		"correlacion:0123456789abcdef", "aportar_documentacion_bolsa", "datos_personales",
		"application/pdf", 2048, strings.Repeat("a", 64),
		"hmac-sha256:idempotencia_v1:"+strings.Repeat("c", 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("b", 64),
		instantePuertoCarga, instantePuertoCarga.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := carga.Preparar("hmac-sha256:sesion_v1:"+strings.Repeat("d", 64),
		"decision:preparar:0123456789abcdef", instantePuertoCarga.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := manifiestoPreparacionPuertoPrueba(t, preparada)
	registro := RegistroReciboCargaDirecta{
		IndiceHMAC:             "hmac-sha256:indice_v1:" + strings.Repeat("1", 64),
		GrupoHMAC:              "hmac-sha256:grupo_v1:" + strings.Repeat("2", 64),
		VinculoHMAC:            "hmac-sha256:vinculo_v1:" + strings.Repeat("3", 64),
		EvidenciaAltaRef:       "evidencia:alta:0123456789abcdef",
		AutorizacionEmisionRef: preparada.AutorizacionPreparacionRef,
		ExpiraEn:               preparada.ExpiraEn,
	}
	secreto := []byte("recibo-efimero-de-prueba")
	preparacion, err := NuevaPreparacionEmisionReciboCargaDirectaV3(secreto, registro, preparada, actor, manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := preparacion.Preimagen()
	if err != nil || !strings.Contains(string(preimagen), `"version_carga":2`) ||
		strings.Contains(string(preimagen), string(secreto)) {
		t.Fatalf("preimagen incompleta o secreto filtrado: %v", err)
	}
	if _, err := json.Marshal(preparacion); !errors.Is(err, ErrSerializacionReciboCargaProhibida) {
		t.Fatalf("serializacion admitida: %v", err)
	}
	llamadas := 0
	_, err = preparacion.ConsumirParaRegistro(func([]byte, RegistroReciboCargaDirectaV3) (ReciboCargaDirecta, error) {
		llamadas++
		return ReciboCargaDirecta{}, nil
	}, AutorizacionCargaDocumentalV3{}, instantePuertoCarga.Add(2*time.Second))
	if !errors.Is(err, ErrReciboCargaDirectaNoValido) || llamadas != 0 {
		t.Fatalf("sin V3 se registro: %v", err)
	}
	if _, err = preparacion.Preimagen(); !errors.Is(err, ErrReciboCargaDirectaNoValido) {
		t.Fatal("material se reutilizo tras fallo")
	}
	preparacion.Descartar()
	preparacion, err = NuevaPreparacionEmisionReciboCargaDirectaV3(secreto, registro, preparada, actor, manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err = preparacion.Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	instante := instantePuertoCarga.Add(2 * time.Second)
	resumen, err := NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:respuesta:0123456789abcdef", strings.Repeat("1", 64), strings.Repeat("2", 64),
		"contexto:respuesta:0123456789abcdef", strings.Repeat("3", 64),
		AccionCargaDocumentalPrepararRespuestaV3, preparada.ID, HuellaPreimagenCargaDocumentalV3(preimagen),
		AudienciaCargaDocumentalPrepararRespuestaV3, instante.Add(-time.Second), instante.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := hex.DecodeString(spkiMaterialPostgreSQLV3Hex)
	if err != nil {
		t.Fatal(err)
	}
	material, err := NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(capacidadCanonicaPostgreSQLV3), resumen, []byte("decision"), []byte("motivo"),
		[]byte("contexto"), 3, 4, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	autorizacion := AutorizacionCargaDocumentalV3{Material: material,
		Accion: AccionCargaDocumentalPrepararRespuestaV3, Finalidad: preparada.Finalidad,
		RecursoRef: preparada.RecursoRef, PersonaRef: preparada.PrincipalID,
		PerfilActivoRef: actor.Contexto.PerfilActivoRef, CorrelacionRef: preparada.CorrelacionRef}
	cruce := autorizacion
	cruce.PersonaRef = "per_ajena0123456789abcdefghijkl"
	if _, err = preparacion.ConsumirParaRegistro(func([]byte, RegistroReciboCargaDirectaV3) (ReciboCargaDirecta, error) {
		llamadas++
		return ReciboCargaDirecta{}, nil
	}, cruce, instante); !errors.Is(err, ErrReciboCargaDirectaNoValido) || llamadas != 0 {
		t.Fatalf("otra persona registro recibo: %v", err)
	}
	preparacion, err = NuevaPreparacionEmisionReciboCargaDirectaV3(secreto, registro, preparada, actor, manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	_, err = preparacion.ConsumirParaRegistro(func(_ []byte, r RegistroReciboCargaDirectaV3) (ReciboCargaDirecta, error) {
		llamadas++
		if r.Carga.ID != preparada.ID || r.Carga.Version != preparada.Version {
			t.Fatal("carga/version distinta")
		}
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoDisponible // COMMIT posible, respuesta perdida.
	}, autorizacion, instante)
	if !errors.Is(err, ErrReciboCargaDirectaNoDisponible) || llamadas != 1 {
		t.Fatalf("respuesta ambigua: %v", err)
	}
	if _, err = preparacion.ConsumirParaRegistro(func([]byte, RegistroReciboCargaDirectaV3) (ReciboCargaDirecta, error) {
		llamadas++
		return ReciboCargaDirecta{}, nil
	}, autorizacion, instante); !errors.Is(err, ErrReciboCargaDirectaNoValido) || llamadas != 1 {
		t.Fatalf("reintento tras COMMIT pudo reemitir: %v", err)
	}
}

func manifiestoPreparacionPuertoPrueba(
	t *testing.T,
	preparada domain.CargaDocumental,
) domain.ManifiestoPreparacionCargaDirectaV1 {
	t.Helper()
	huellaRecursoBase, err := HuellaRecursoBaseCargaDocumental(recursoBaseCargaPuertoPrueba())
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := domain.NuevoManifiestoPreparacionCargaDirectaV1(
		preparada,
		domain.ContextoManifiestoPreparacionCargaDirectaV1{
			CargaRef:                preparada.IndiceIdempotenciaHMAC,
			SujetoSeudonimoHMAC:     "hmac-sha256:seudonimo_v1:" + strings.Repeat("d", 64),
			HuellaRecursoBaseSHA256: huellaRecursoBase,
			HuellaRecursoSHA256:     strings.Repeat("e", 64), ConectorAlmacenID: "almacen_s3_corporativo",
			EsquemaContexto: EsquemaContextoOperacionAlmacenV1,
			AccionNegocio:   AccionNegocioPrepararCargaDocumental,
			AccionTecnica:   AccionAlmacenPrepararCargaDirecta,
			PasoRef:         string(PasoAlmacenPrepararCargaDirecta), EfectoRef: "efecto:carga:preparar:0123456789abcdef",
			HuellaPlanEfectoSHA256: strings.Repeat("f", 64),
			EsquemaHuellaDecision:  EsquemaHuellaDecisionAutorizacionReforzadaV1,
			DecisionRef:            preparada.AutorizacionPreparacionRef,
			HuellaDecisionSHA256:   strings.Repeat("1", 64),
			ContextoVerificadoEn:   instantePuertoCarga.Add(500 * time.Millisecond),
			DecisionValidaHasta:    preparada.ExpiraEn.Add(time.Minute),
		},
	)
	if err != nil {
		t.Fatalf("crear manifiesto de puerto: %v", err)
	}
	return manifiesto
}

func TestTokenReservaCargaTieneDominioCriptograficoPropio(t *testing.T) {
	const entropiaComun = "0123456789abcdef0123456789abcdef"
	operacionCarga, err := nuevaOperacionCapacidadReservaConFuenteEntropia(
		dominioHuellaTokenReservaCargaDocumental, strings.NewReader(entropiaComun),
	)
	if err != nil {
		t.Fatal(err)
	}
	operacionDocumento, err := nuevaOperacionCapacidadReservaConFuenteEntropia(
		dominioHuellaTokenReservaGeneracionDocumento, strings.NewReader(entropiaComun),
	)
	if err != nil {
		t.Fatal(err)
	}
	tokenCarga := TokenReservaCargaDocumental{operar: operacionCarga}
	tokenDocumento := TokenReservaGeneracionDocumento{operar: operacionDocumento}
	huellaCarga, err := tokenCarga.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaDocumento, err := tokenDocumento.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if huellaCarga == huellaDocumento || tokenCarga.CoincideConHuellaSHA256(huellaDocumento) {
		t.Fatal("la misma entropia produjo autoridad cruzable entre dominios")
	}
}

func TestReservaCargaVinculaIdempotenciaYSolicitudExactas(t *testing.T) {
	carga := cargaReservadaPuertoPrueba(t)
	solicitud := SolicitudReservarCargaDocumental{
		IndiceIdempotenciaHMAC: "hmac-sha256:idempotencia_v1:" + strings.Repeat("c", 64),
		HuellaSolicitudHMAC:    carga.HuellaSolicitudHMAC,
		Carga:                  carga,
		DecisionPreparacion: ConsumoDecisionPreparacionCargaDocumentalV1{
			DecisionRef:            "decision:preparar:0123456789abcdef",
			EfectoRef:              "efecto:carga:preparar:0123456789abcdef",
			HuellaPlanEfectoSHA256: strings.Repeat("e", 64),
			EsquemaHuellaDecision:  EsquemaHuellaDecisionAutorizacionReforzadaV1,
			HuellaDecisionSHA256:   strings.Repeat("f", 64),
		},
		SolicitadaEn:    instantePuertoCarga,
		ReservaExpiraEn: instantePuertoCarga.Add(2 * time.Minute),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("reserva valida: %v", err)
	}
	alterada := solicitud
	alterada.HuellaSolicitudHMAC = "hmac-sha256:solicitud_v1:" + strings.Repeat("d", 64)
	if !errors.Is(alterada.Validar(), ErrReservaCargaDocumentalInvalida) {
		t.Fatal("se acepto una reserva cuya huella no corresponde al agregado")
	}
	alterada = solicitud
	alterada.ReservaExpiraEn = carga.ExpiraEn.Add(time.Second)
	if !errors.Is(alterada.Validar(), ErrReservaCargaDocumentalInvalida) {
		t.Fatal("se acepto una reserva mas larga que la carga")
	}
}

func TestHuellaRecursoBaseCargaIncluyeTodoElContextoYRechazaAtributosReservados(t *testing.T) {
	recurso := recursoBaseCargaPuertoPrueba()
	huella, err := HuellaRecursoBaseCargaDocumental(recurso)
	if err != nil || !esSHA256Hexadecimal(huella) {
		t.Fatalf("huella base: %q, %v", huella, err)
	}
	mismaEntrada := recursoBaseCargaPuertoPrueba()
	mismaEntrada.Ambitos = map[string]string{"proceso": "proceso:0123456789abcdef"}
	mismaEntrada.Atributos = map[string]string{"categoria": "auxiliar_administrativo"}
	huellaMisma, err := HuellaRecursoBaseCargaDocumental(mismaEntrada)
	if err != nil || huellaMisma != huella {
		t.Fatal("la misma entrada canonica produjo otra huella")
	}

	mutaciones := []struct {
		nombre string
		muta   func(*domain.RecursoAutorizable)
	}{
		{"referencia", func(r *domain.RecursoAutorizable) { r.Referencia = "solicitud:otra:0123456789abcdef" }},
		{"modulo", func(r *domain.RecursoAutorizable) { r.ModuloID = "seleccion" }},
		{"tipo", func(r *domain.RecursoAutorizable) { r.Tipo = "expediente" }},
		{"ambito", func(r *domain.RecursoAutorizable) { r.Ambitos["proceso"] = "proceso:otro:0123456789abcdef" }},
		{"atributo", func(r *domain.RecursoAutorizable) { r.Atributos["categoria"] = "administrativo" }},
	}
	for _, prueba := range mutaciones {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado := recursoBaseCargaPuertoPrueba()
			prueba.muta(&alterado)
			huellaAlterada, err := HuellaRecursoBaseCargaDocumental(alterado)
			if err != nil {
				t.Fatal(err)
			}
			if huellaAlterada == huella {
				t.Fatal("la mutacion no cambio la huella base")
			}
		})
	}

	for _, caso := range []struct {
		enAmbito bool
		clave    string
		valor    string
	}{
		{true, AtributoAlmacenEfectoRef, "efecto:inyectado"},
		{false, AtributoAlmacenHuellaManifiestoSHA256, strings.Repeat("a", 64)},
		{false, "almacen_atributo_futuro", "valor"},
	} {
		inyectado := recursoBaseCargaPuertoPrueba()
		if caso.enAmbito {
			inyectado.Ambitos[caso.clave] = caso.valor
		} else {
			inyectado.Atributos[caso.clave] = caso.valor
		}
		if _, err := HuellaRecursoBaseCargaDocumental(inyectado); !errors.Is(err, ErrRecursoBaseCargaDocumentalInvalido) {
			t.Fatalf("atributo reservado aceptado (ambito=%v clave=%s): %v", caso.enAmbito, caso.clave, err)
		}
	}
}

func TestManifiestoPreparacionVinculaElRecursoBaseCompleto(t *testing.T) {
	reservada := cargaReservadaPuertoPrueba(t)
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:0123456789abcdef", instantePuertoCarga.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := manifiestoPreparacionPuertoPrueba(t, preparada)
	base := recursoBaseCargaPuertoPrueba()
	if err := ValidarRecursoBaseManifiestoPreparacionCargaDocumental(manifiesto, preparada, base); err != nil {
		t.Fatalf("recurso base exacto: %v", err)
	}
	for _, muta := range []func(*domain.RecursoAutorizable){
		func(r *domain.RecursoAutorizable) { r.Ambitos["proceso"] = "proceso:alterado:0123456789abcdef" },
		func(r *domain.RecursoAutorizable) { r.Atributos["categoria"] = "administrativo" },
		func(r *domain.RecursoAutorizable) { r.Atributos["nuevo"] = "valor" },
	} {
		alterado := recursoBaseCargaPuertoPrueba()
		muta(&alterado)
		if !errors.Is(
			ValidarRecursoBaseManifiestoPreparacionCargaDocumental(manifiesto, preparada, alterado),
			ErrConfirmacionCargaDocumentalInvalida,
		) {
			t.Fatal("el manifiesto acepto otro contexto ABAC base")
		}
	}
}

func TestConfirmacionTransicionExigeInmutabilidadAutorizacionYAuditoriaAtomica(t *testing.T) {
	anterior := cargaReservadaPuertoPrueba(t)
	siguiente, err := anterior.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:0123456789abcdef", instantePuertoCarga.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaAnterior, _ := anterior.HuellaSHA256()
	huellaSiguiente, _ := siguiente.HuellaSHA256()
	auditoria := domain.AuditEntry{
		ActorID: "persona:0123456789abcdef", ActorProfile: "perfil:externo:0123456789abcdef",
		ActorRoles:       []string{"usuario_externo"},
		AuthorizationRef: siguiente.AutorizacionPreparacionRef, Purpose: siguiente.Finalidad,
		Action: "vec.documentos.carga.preparar", ModuleID: siguiente.ModuloID, SubjectRef: siguiente.ID,
		ObjectVersion: siguiente.Version, Result: "correcto", BeforeHash: huellaAnterior, AfterHash: huellaSiguiente,
		CorrelationRef: siguiente.CorrelacionRef, Metadata: map[string]string{"estado": string(siguiente.Estado)},
		OccurredAt: instantePuertoCarga.Add(time.Second),
	}
	evento := domain.Event{
		Type: "vec.documentos.carga.preparada", ModuleID: siguiente.ModuloID, SubjectRef: siguiente.ID,
		ActorID: auditoria.ActorID, OccurredAt: auditoria.OccurredAt, Payload: map[string]string{
			"carga_ref": siguiente.ID, "estado": string(siguiente.Estado), "version": "2",
		},
	}
	confirmacion, err := NuevaConfirmacionTransicionCargaDocumental(anterior, siguiente, auditoria, evento)
	if err != nil || confirmacion.ValidarContra(anterior) != nil {
		t.Fatalf("confirmacion valida: %#v, %v", confirmacion, err)
	}
	instantanea := InstantaneaConfirmacionTransicionCargaDocumental(confirmacion)
	confirmacion.Auditoria.ActorRoles[0] = "rol_mutado"
	confirmacion.Auditoria.Metadata["estado"] = "alterado"
	confirmacion.Evento.Payload["estado"] = "alterado"
	if instantanea.Auditoria.ActorRoles[0] != "usuario_externo" ||
		instantanea.Auditoria.Metadata["estado"] != string(siguiente.Estado) ||
		instantanea.Evento.Payload["estado"] != string(siguiente.Estado) ||
		instantanea.ValidarContra(anterior) != nil {
		t.Fatal("la instantanea de transicion compartio datos mutables con la entrada")
	}

	pruebas := []struct {
		nombre string
		muta   func(*domain.CargaDocumental, *domain.AuditEntry, *domain.Event)
	}{
		{"cambia recurso", func(c *domain.CargaDocumental, _ *domain.AuditEntry, _ *domain.Event) {
			c.RecursoRef = "solicitud:otra:0123456789abcdef"
		}},
		{"otra autorizacion", func(_ *domain.CargaDocumental, a *domain.AuditEntry, _ *domain.Event) {
			a.AuthorizationRef = "decision:otra:0123456789abcdef"
		}},
		{"otro actor en evento", func(_ *domain.CargaDocumental, _ *domain.AuditEntry, e *domain.Event) {
			e.ActorID = "servicio:otro:0123456789abcdef"
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			carga := siguiente
			auditada := auditoria
			emitido := evento
			prueba.muta(&carga, &auditada, &emitido)
			if _, err := NuevaConfirmacionTransicionCargaDocumental(anterior, carga, auditada, emitido); !errors.Is(err, ErrConfirmacionCargaDocumentalInvalida) {
				t.Fatalf("se acepto una confirmacion cruzada: %v", err)
			}
		})
	}
}

func TestConfirmarPreparacionExigeManifiestoAtomicoYLaLecturaEsCoherente(t *testing.T) {
	reservada := cargaReservadaPuertoPrueba(t)
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:0123456789abcdef", instantePuertoCarga.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	auditoria, evento := auditoriaYEventoCargaPuertoPrueba(t, reservada, preparada)
	confirmacion, err := NuevaConfirmacionTransicionCargaDocumental(reservada, preparada, auditoria, evento)
	if err != nil {
		t.Fatal(err)
	}
	token, _ := NuevoTokenReservaCargaDocumental()
	manifiesto := manifiestoPreparacionPuertoPrueba(t, preparada)
	solicitud := SolicitudConfirmarPreparacionCargaDocumental{
		Token: token, Confirmacion: confirmacion, Manifiesto: manifiesto,
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud atomica valida: %v", err)
	}
	instantanea, err := InstantaneaSolicitudConfirmarPreparacionCargaDocumental(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	solicitud.Confirmacion.Auditoria.Metadata["estado"] = "alterado-despues-de-copiar"
	solicitud.Confirmacion.Evento.Payload["estado"] = "alterado-despues-de-copiar"
	if instantanea.Confirmacion.Auditoria.Metadata["estado"] != string(preparada.Estado) ||
		instantanea.Confirmacion.Evento.Payload["estado"] != string(preparada.Estado) ||
		instantanea.Confirmacion.ValidarContra(reservada) != nil {
		t.Fatal("la instantanea validada compartio mapas con la solicitud original")
	}
	consumo, err := ConsumoDecisionDesdeManifiestoPreparacionCargaDocumental(instantanea.Manifiesto)
	if err != nil || consumo.DecisionRef != preparada.AutorizacionPreparacionRef ||
		consumo.EfectoRef == "" || consumo.HuellaPlanEfectoSHA256 == "" || consumo.HuellaDecisionSHA256 == "" {
		t.Fatalf("consumo durable de decision: %#v, %v", consumo, err)
	}
	if err := (PreparacionCargaDocumentalPersistida{
		Carga: preparada, Manifiesto: manifiesto,
	}).Validar(); err != nil {
		t.Fatalf("instantanea coherente: %v", err)
	}
	sinManifiesto := solicitud
	sinManifiesto.Manifiesto = domain.ManifiestoPreparacionCargaDirectaV1{}
	if !errors.Is(sinManifiesto.Validar(), ErrConfirmacionCargaDocumentalInvalida) {
		t.Fatal("se permitio consumir la reserva sin manifiesto")
	}
	cruzada := PreparacionCargaDocumentalPersistida{Carga: cargaReservadaPuertoPrueba(t), Manifiesto: manifiesto}
	if !errors.Is(cruzada.Validar(), ErrConfirmacionCargaDocumentalInvalida) {
		t.Fatal("se devolvio agregado y manifiesto de instantaneas distintas")
	}
}

func cadenaCargaPuertoPrueba(
	t *testing.T,
) (domain.CargaDocumental, domain.CargaDocumental, domain.CargaDocumental, domain.CargaDocumental, domain.CargaDocumental) {
	t.Helper()
	reservada := cargaReservadaPuertoPrueba(t)
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("c", 64),
		"decision:preparar:0123456789abcdef",
		instantePuertoCarga.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido := domain.ContenidoCargaDocumental{
		ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:0123456789abcdef", Version: "version:1",
		Zona: domain.ZonaContenidoCargaCuarentena, MIME: reservada.MIMEDeclarado, Tamano: reservada.TamanoDeclarado,
		HuellaSHA256: reservada.HuellaDeclaradaSHA256, EvidenciaRef: "evidencia:almacen:0123456789abcdef",
		RegistradoEn: instantePuertoCarga.Add(2 * time.Second),
	}
	cuarentena, err := preparada.RegistrarCuarentena(
		contenido,
		"decision:confirmar:0123456789abcdef",
		contenido.RegistradoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	analisis := domain.AnalisisCargaDocumental{
		ObjetoReferencia: contenido.Referencia, ObjetoVersion: contenido.Version,
		HuellaObjetoSHA256: contenido.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: domain.EstadoAnalisisCargaLimpio, CodigoResultado: "limpio",
		EvidenciaRef: "evidencia:analisis:0123456789abcdef", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: instantePuertoCarga.Add(3 * time.Second),
	}
	analizada, err := cuarentena.RegistrarAnalisis(
		analisis,
		"decision:analizar:0123456789abcdef",
		analisis.CompletadoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	admitido := contenido
	admitido.Referencia = "objeto:admitido:0123456789abcdef"
	admitido.Zona = domain.ZonaContenidoCargaAdmitida
	admitido.EvidenciaRef = "evidencia:promocion:0123456789abcdef"
	admitido.RegistradoEn = instantePuertoCarga.Add(4 * time.Second)
	admitida, err := analizada.Admitir(
		admitido,
		"decision:promover:0123456789abcdef",
		admitido.RegistradoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	return reservada, preparada, cuarentena, analizada, admitida
}

func auditoriaYEventoCargaPuertoPrueba(
	t *testing.T,
	anterior, siguiente domain.CargaDocumental,
) (domain.AuditEntry, domain.Event) {
	t.Helper()
	accion, tipoEvento, valida := correspondenciaTransicionCargaDocumental(anterior.Estado, siguiente.Estado)
	if !valida {
		t.Fatal("transicion de prueba no reconocida")
	}
	huellaAnterior, err := anterior.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaSiguiente, err := siguiente.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	auditoria := domain.AuditEntry{
		ActorID: "servicio:carga:0123456789abcdef", ActorProfile: "perfil:servicio:carga",
		ActorRoles: []string{"servicio_carga"}, AuthorizationRef: autorizacionTransicionCarga(siguiente),
		Purpose: siguiente.Finalidad, Action: accion, ModuleID: siguiente.ModuloID, SubjectRef: siguiente.ID,
		ObjectVersion: siguiente.Version, Result: "correcto", BeforeHash: huellaAnterior, AfterHash: huellaSiguiente,
		CorrelationRef: siguiente.CorrelacionRef, Metadata: map[string]string{"estado": string(siguiente.Estado)},
		OccurredAt: siguiente.ActualizadaEn,
	}
	evento := domain.Event{
		Type: tipoEvento, ModuleID: siguiente.ModuloID, SubjectRef: siguiente.ID, ActorID: auditoria.ActorID,
		OccurredAt: siguiente.ActualizadaEn, Payload: map[string]string{
			"carga_ref": siguiente.ID,
			"estado":    string(siguiente.Estado),
			"version":   fmt.Sprint(siguiente.Version),
		},
	}
	return auditoria, evento
}

func TestConfirmacionTransicionSoloPermiteCamposPositivosDeCadaPaso(t *testing.T) {
	_, preparada, cuarentena, analizada, admitida := cadenaCargaPuertoPrueba(t)
	pruebas := []struct {
		nombre    string
		anterior  domain.CargaDocumental
		siguiente domain.CargaDocumental
		muta      func(*domain.CargaDocumental)
	}{
		{
			nombre:   "reescribe autorizacion de preparacion al recibir",
			anterior: preparada, siguiente: cuarentena,
			muta: func(c *domain.CargaDocumental) {
				c.AutorizacionPreparacionRef = "decision:preparar:reescrita"
			},
		},
		{
			nombre:   "reescribe vinculo de sesion al recibir",
			anterior: preparada, siguiente: cuarentena,
			muta: func(c *domain.CargaDocumental) {
				c.VinculoSesionHMAC = "hmac-sha256:sesion_v2:" + strings.Repeat("e", 64)
			},
		},
		{
			nombre:   "reescribe evidencia de cuarentena al analizar",
			anterior: cuarentena, siguiente: analizada,
			muta: func(c *domain.CargaDocumental) {
				contenido := *c.ContenidoCuarentena
				contenido.EvidenciaRef = "evidencia:almacen:reescrita"
				c.ContenidoCuarentena = &contenido
			},
		},
		{
			nombre:   "reescribe evidencia de analisis al promover",
			anterior: analizada, siguiente: admitida,
			muta: func(c *domain.CargaDocumental) {
				analisis := *c.Analisis
				analisis.EvidenciaRef = "evidencia:analisis:reescrita"
				c.Analisis = &analisis
			},
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterada := clonarCargaDocumentalPuerto(prueba.siguiente)
			prueba.muta(&alterada)
			if err := alterada.Validar(); err != nil {
				t.Fatalf("la alteracion debe seguir siendo valida aisladamente para probar el cotejo: %v", err)
			}
			auditoria, evento := auditoriaYEventoCargaPuertoPrueba(t, prueba.anterior, alterada)
			if _, err := NuevaConfirmacionTransicionCargaDocumental(
				prueba.anterior,
				alterada,
				auditoria,
				evento,
			); !errors.Is(err, ErrConfirmacionCargaDocumentalInvalida) {
				t.Fatalf("se acepto reescribir un campo historico: %v", err)
			}
		})
	}
}

func TestConfirmacionTransicionAceptaSoloLasCorrespondenciasDeclaradas(t *testing.T) {
	reservada, preparada, cuarentena, analizada, admitida := cadenaCargaPuertoPrueba(t)
	analisisRetenido := *analizada.Analisis
	analisisRetenido.Estado = domain.EstadoAnalisisCargaNoConcluyente
	analisisRetenido.CodigoResultado = "motor_no_disponible"
	retenida, err := cuarentena.RegistrarAnalisis(
		analisisRetenido,
		"decision:analizar:retenida:0123456789abcdef",
		analisisRetenido.CompletadoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	pasos := []struct {
		nombre    string
		anterior  domain.CargaDocumental
		siguiente domain.CargaDocumental
	}{
		{"preparar", reservada, preparada},
		{"confirmar cuarentena", preparada, cuarentena},
		{"registrar analisis", cuarentena, analizada},
		{"retener por seguridad", cuarentena, retenida},
		{"promover", analizada, admitida},
	}
	for _, paso := range pasos {
		t.Run(paso.nombre, func(t *testing.T) {
			auditoria, evento := auditoriaYEventoCargaPuertoPrueba(t, paso.anterior, paso.siguiente)
			confirmacion, err := NuevaConfirmacionTransicionCargaDocumental(
				paso.anterior,
				paso.siguiente,
				auditoria,
				evento,
			)
			if err != nil || confirmacion.ValidarContra(paso.anterior) != nil {
				t.Fatalf("correspondencia declarada rechazada: %v", err)
			}
		})
	}
}

func TestConfirmacionTransicionExigeAccionEventoEInstanteExactos(t *testing.T) {
	reservada, preparada, _, _, _ := cadenaCargaPuertoPrueba(t)
	auditoria, evento := auditoriaYEventoCargaPuertoPrueba(t, reservada, preparada)

	pruebas := []struct {
		nombre string
		muta   func(*domain.AuditEntry, *domain.Event)
	}{
		{"accion de otra transicion", func(a *domain.AuditEntry, _ *domain.Event) {
			a.Action = accionCargaDocumentalConfirmar
		}},
		{"evento de otra transicion", func(_ *domain.AuditEntry, e *domain.Event) {
			e.Type = eventoCargaDocumentalRecibida
		}},
		{"auditoria un nanosegundo distinta", func(a *domain.AuditEntry, _ *domain.Event) {
			a.OccurredAt = a.OccurredAt.Add(time.Nanosecond)
		}},
		{"evento un nanosegundo distinto", func(_ *domain.AuditEntry, e *domain.Event) {
			e.OccurredAt = e.OccurredAt.Add(time.Nanosecond)
		}},
		{"payload con campo no autorizado", func(_ *domain.AuditEntry, e *domain.Event) {
			e.Payload["dato_no_declarado"] = "valor"
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			auditada := clonarAuditoriaCargaDocumental(auditoria)
			emitido := clonarEventoCargaDocumental(evento)
			prueba.muta(&auditada, &emitido)
			if _, err := NuevaConfirmacionTransicionCargaDocumental(
				reservada,
				preparada,
				auditada,
				emitido,
			); !errors.Is(err, ErrConfirmacionCargaDocumentalInvalida) {
				t.Fatalf("se acepto correspondencia no exacta: %v", err)
			}
		})
	}
}

func TestNuevaConfirmacionTomaCopiasProfundas(t *testing.T) {
	_, preparada, cuarentena, _, _ := cadenaCargaPuertoPrueba(t)
	auditoria, evento := auditoriaYEventoCargaPuertoPrueba(t, preparada, cuarentena)
	confirmacion, err := NuevaConfirmacionTransicionCargaDocumental(preparada, cuarentena, auditoria, evento)
	if err != nil {
		t.Fatal(err)
	}
	evidenciaOriginal := confirmacion.Carga.ContenidoCuarentena.EvidenciaRef
	rolOriginal := confirmacion.Auditoria.ActorRoles[0]
	metadatoOriginal := confirmacion.Auditoria.Metadata["estado"]
	estadoEventoOriginal := confirmacion.Evento.Payload["estado"]

	cuarentena.ContenidoCuarentena.EvidenciaRef = "evidencia:mutada:fuera"
	auditoria.ActorRoles[0] = "rol_mutado"
	auditoria.Metadata["estado"] = "mutado"
	evento.Payload["estado"] = "mutado"

	if confirmacion.Carga.ContenidoCuarentena.EvidenciaRef != evidenciaOriginal ||
		confirmacion.Auditoria.ActorRoles[0] != rolOriginal ||
		confirmacion.Auditoria.Metadata["estado"] != metadatoOriginal ||
		confirmacion.Evento.Payload["estado"] != estadoEventoOriginal {
		t.Fatal("la confirmacion conservo alias mutables de sus argumentos")
	}
	if err := confirmacion.ValidarContra(preparada); err != nil {
		t.Fatalf("la copia profunda dejo de ser valida tras mutar los argumentos: %v", err)
	}
}
