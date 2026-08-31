package ports

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type verificadorAtestacionCanceladorAutoridadObjetoV1Prueba struct {
	delegado VerificadorAtestacionMaterialAlmacenV2
	cancelar context.CancelFunc
	mutar    func()
}

func (v verificadorAtestacionCanceladorAutoridadObjetoV1Prueba) VerificarAtestacionMaterialAlmacenV2(
	ctx context.Context,
	solicitud SolicitudVerificarAtestacionMaterialAlmacenV2,
) error {
	err := v.delegado.VerificarAtestacionMaterialAlmacenV2(ctx, solicitud)
	if v.mutar != nil {
		v.mutar()
	}
	if v.cancelar != nil {
		v.cancelar()
	}
	return err
}

type verificadorReferenciaCanceladorAutoridadObjetoV1Prueba struct {
	delegado VerificadorReferenciaReciboMaterialV2
	cancelar context.CancelFunc
	mutar    func()
}

func (v verificadorReferenciaCanceladorAutoridadObjetoV1Prueba) VerificarReferenciaReciboMaterialV2(
	ctx context.Context,
	solicitud SolicitudReservarReferenciaReciboMaterialV2,
	resultado ResultadoReferenciaReciboMaterialV2,
) error {
	err := v.delegado.VerificarReferenciaReciboMaterialV2(ctx, solicitud, resultado)
	if v.mutar != nil {
		v.mutar()
	}
	if v.cancelar != nil {
		v.cancelar()
	}
	return err
}

type verificadorCOSESign1AutoridadObjetoV1Prueba struct {
	publica      ed25519.PublicKey
	claveRef     string
	claveVersion uint32
}

func (v verificadorCOSESign1AutoridadObjetoV1Prueba) VerificarAtestacionMaterialAlmacenV2(
	_ context.Context,
	solicitud SolicitudVerificarAtestacionMaterialAlmacenV2,
) error {
	dominio, mensaje, algoritmo, claveRef, claveVersion, codigo, err :=
		solicitud.RevelarParaVerificacion()
	const cabecera = "\xd2\x84\x43\xa1\x01\x27\xa0\xf6\x58\x40"
	if err != nil || algoritmo != AlgoritmoAtestacionMaterialCOSESign1 ||
		claveRef != v.claveRef || claveVersion != v.claveVersion ||
		len(codigo) != len(cabecera)+ed25519.SignatureSize ||
		!bytes.Equal(codigo[:len(cabecera)], []byte(cabecera)) ||
		!ed25519.Verify(v.publica, estructuraFirmaCOSESign1AutoridadObjetoV1Prueba(
			dominio, mensaje,
		), codigo[len(cabecera):]) {
		return errors.New("cose sign1 no verificable")
	}
	return nil
}

func nuevoSobreCOSESign1AutoridadObjetoV1Prueba(
	privada ed25519.PrivateKey,
	dominio string,
	mensaje []byte,
) []byte {
	const cabecera = "\xd2\x84\x43\xa1\x01\x27\xa0\xf6\x58\x40"
	firma := ed25519.Sign(
		privada, estructuraFirmaCOSESign1AutoridadObjetoV1Prueba(dominio, mensaje),
	)
	return append(append([]byte(nil), []byte(cabecera)...), firma...)
}

func estructuraFirmaCOSESign1AutoridadObjetoV1Prueba(dominio string, mensaje []byte) []byte {
	estructura := []byte{0x84, 0x6a}
	estructura = append(estructura, []byte("Signature1")...)
	estructura = anexarBstrCOSEAutoridadObjetoV1Prueba(estructura, []byte{0xa1, 0x01, 0x27})
	estructura = anexarBstrCOSEAutoridadObjetoV1Prueba(estructura, []byte(dominio))
	return anexarBstrCOSEAutoridadObjetoV1Prueba(estructura, mensaje)
}

func anexarBstrCOSEAutoridadObjetoV1Prueba(destino, valor []byte) []byte {
	longitud := len(valor)
	switch {
	case longitud < 24:
		destino = append(destino, 0x40|byte(longitud))
	case longitud <= 0xff:
		destino = append(destino, 0x58, byte(longitud))
	default:
		var bytesLongitud [2]byte
		binary.BigEndian.PutUint16(bytesLongitud[:], uint16(longitud))
		destino = append(destino, 0x59)
		destino = append(destino, bytesLongitud[:]...)
	}
	return append(destino, valor...)
}

type escenarioAutoridadObjetoEsperadoDocumentalV1Prueba struct {
	materializacion escenarioMaterializacionDocumentalV4
	perfil          PerfilCapacidadesAlmacenMaterialV2
	plan            SeleccionPlanMaterialAlmacenV2
	recibo          ReciboEscrituraObjetoMaterialV2
	autoridad       AutoridadObjetoEsperadoDocumentalV1
	criptografia    *criptografiaMaterialV2Prueba
	registro        *registroAutoritativoMaterialV2Prueba
}

func nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(
	t *testing.T,
) escenarioAutoridadObjetoEsperadoDocumentalV1Prueba {
	t.Helper()
	materializacion := escenarioMaterializacionDocumentalV4Prueba(t)
	criptografia := &criptografiaMaterialV2Prueba{
		clave:        []byte("clave-exclusiva-autoridad-objeto-v1-prueba"),
		claveRef:     "clave:atestacion:objeto-esperado:v1",
		claveVersion: 11,
	}
	registro := nuevoRegistroAutoritativoMaterialV2Prueba()
	const perfilRef = "perfil:capacidades:objeto-esperado:v1"
	registro.perfiles[perfilRef] = perfilPublicadoMaterialV2Prueba{
		version: 13, capacidades: materializacion.capacidades,
	}
	perfil, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), perfilRef, 13, materializacion.capacidades,
		criptografia, criptografia, registro,
	)
	if err != nil {
		t.Fatalf("crear perfil material atestado: %v", err)
	}
	plan, err := NuevaSeleccionPlanMaterialAlmacenV2(
		"plan:material:objeto-esperado:v1", 17,
	)
	if err != nil {
		t.Fatalf("seleccionar plan material: %v", err)
	}
	proyeccion, err := materializacion.solicitud.Contexto.Proyeccion()
	if err != nil {
		t.Fatalf("proyectar contexto V4: %v", err)
	}
	registro.planes[plan.referencia] = planPublicadoMaterialV2Prueba{
		version: plan.version, huella: strings.Repeat("9", 64),
		conectorLogicoID: materializacion.capacidades.ConectorID,
		moduloID:         proyeccion.ModuloID,
		accionNegocio:    proyeccion.AccionNegocio,
		accionTecnica:    proyeccion.AccionTecnica,
		recursoRef:       proyeccion.RecursoRef,
		operacionRef:     proyeccion.OperacionRef,
		cargaRef:         proyeccion.CargaRef,
		efectoRef:        proyeccion.EfectoRef,
		clasificacion:    proyeccion.Clasificacion,
	}
	escenario := escenarioAutoridadObjetoEsperadoDocumentalV1Prueba{
		materializacion: materializacion,
		perfil:          perfil,
		plan:            plan,
		criptografia:    criptografia,
		registro:        registro,
	}
	escenario.recibo = escenario.crearRecibo(t, materializacion.resultado)
	autoridad, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), escenario.materializacion.declaracion, escenario.recibo,
		escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatalf("crear autoridad de objeto esperado: %v", err)
	}
	escenario.autoridad = autoridad
	return escenario
}

func (e escenarioAutoridadObjetoEsperadoDocumentalV1Prueba) crearRecibo(
	t *testing.T,
	resultado ResultadoOperacionObjeto,
) ReciboEscrituraObjetoMaterialV2 {
	t.Helper()
	recibo, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), e.materializacion.solicitud, resultado,
		e.materializacion.capacidades, e.perfil, e.registro, e.plan,
		e.registro, e.registro, e.registro, e.criptografia, e.criptografia,
	)
	if err != nil {
		t.Fatalf("crear recibo material V2: %v", err)
	}
	return recibo
}

func (e escenarioAutoridadObjetoEsperadoDocumentalV1Prueba) crearAutoridad(
	t *testing.T,
) AutoridadObjetoEsperadoDocumentalV1 {
	t.Helper()
	if e.autoridad.datos == nil {
		t.Fatal("fixture sin autoridad de objeto esperado verificada")
	}
	return clonarAutoridadObjetoEsperadoDocumentalV1Prueba(e.autoridad)
}

func clonarEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(
	original escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) escenarioAutoridadObjetoEsperadoDocumentalV1Prueba {
	copia := original
	copia.materializacion.declaracion = clonarDeclaracionAutoridadObjetoEsperadoDocumentalV1(
		original.materializacion.declaracion,
	)
	copia.perfil.atestacion.codigo = append([]byte(nil), original.perfil.atestacion.codigo...)
	copia.recibo = clonarReciboAutoridadObjetoEsperadoDocumentalV1(original.recibo)
	copia.autoridad = clonarAutoridadObjetoEsperadoDocumentalV1Prueba(original.autoridad)

	criptografia := *original.criptografia
	criptografia.clave = append([]byte(nil), original.criptografia.clave...)
	copia.criptografia = &criptografia

	registro := *original.registro
	registro.perfiles = make(map[string]perfilPublicadoMaterialV2Prueba, len(original.registro.perfiles))
	for referencia, perfil := range original.registro.perfiles {
		perfil.capacidades = clonarCapacidadesAlmacenDocumentalV4(perfil.capacidades)
		registro.perfiles[referencia] = perfil
	}
	registro.planes = make(map[string]planPublicadoMaterialV2Prueba, len(original.registro.planes))
	for referencia, plan := range original.registro.planes {
		registro.planes[referencia] = plan
	}
	registro.referencias = make(map[string]string, len(original.registro.referencias))
	for identidad, referencia := range original.registro.referencias {
		registro.referencias[identidad] = referencia
	}
	copia.registro = &registro
	return copia
}

func clonarAutoridadObjetoEsperadoDocumentalV1Prueba(
	autoridad AutoridadObjetoEsperadoDocumentalV1,
) AutoridadObjetoEsperadoDocumentalV1 {
	copia := autoridad
	datos := *autoridad.datos
	datos.recibo = clonarReciboAutoridadObjetoEsperadoDocumentalV1(datos.recibo)
	datos.atestacionRecibo = clonarAtestacionAutoridadObjetoEsperadoDocumentalV1(
		datos.atestacionRecibo,
	)
	datos.declaracion = clonarDeclaracionAutoridadObjetoEsperadoDocumentalV1(datos.declaracion)
	copia.datos = &datos
	return copia
}

func TestAutoridadObjetoEsperadoDocumentalV1(t *testing.T) {
	fixture := nuevoEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(t)
	casos := []struct {
		nombre string
		probar func(*testing.T, escenarioAutoridadObjetoEsperadoDocumentalV1Prueba)
	}{
		{"sella recibo material exacto", probarAutoridadObjetoEsperadoDocumentalV1SellaReciboMaterialExacto},
		{"deriva pareja solo de instantanea V2", probarAutoridadObjetoEsperadoDocumentalV1DerivaParejaSoloDeInstantaneaV2},
		{"admite recibo COSE Sign1 verificado", probarAutoridadObjetoEsperadoDocumentalV1AdmiteReciboCOSESign1Verificado},
		{"entrega copias y permanece opaca", probarAutoridadObjetoEsperadoDocumentalV1EntregaCopiasYPermaneceOpaca},
		{"rechaza evidencia sin recibo y objeto ajeno", probarAutoridadObjetoEsperadoDocumentalV1RechazaEvidenciaSinReciboYObjetoAjeno},
		{"falla cerrada en dependencias y cancelacion", probarAutoridadObjetoEsperadoDocumentalV1FallaCerradaEnDependenciasYCancelacion},
		{"reverifica antes del registro", probarAutoridadObjetoEsperadoDocumentalV1ReverificaAntesDelRegistro},
		{"cancelacion durante cada verificador devuelve cero", probarAutoridadObjetoEsperadoDocumentalV1CancelacionDuranteCadaVerificadorDevuelveCero},
		{"revalida compromisos tras verificadores vivos", probarAutoridadObjetoEsperadoDocumentalV1RevalidaCompromisosTrasVerificadoresVivos},
		{"mutacion durante cada verificador devuelve cero", probarAutoridadObjetoEsperadoDocumentalV1MutacionDuranteCadaVerificadorDevuelveCero},
		{"detecta alteracion interna", probarAutoridadObjetoEsperadoDocumentalV1DetectaAlteracionInterna},
		{"preparacion concurrente", probarAutoridadObjetoEsperadoDocumentalV1PreparacionConcurrente},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			caso.probar(t, clonarEscenarioAutoridadObjetoEsperadoDocumentalV1Prueba(fixture))
		})
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1SellaReciboMaterialExacto(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	if err := autoridad.Validar(); err != nil {
		t.Fatalf("autoridad valida rechazada: %v", err)
	}
	proyeccion, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatalf("preparar registro: %v", err)
	}
	contexto, err := escenario.materializacion.solicitud.Contexto.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	huellaRecibo, err := escenario.recibo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := escenario.recibo.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	huellaDeclaracion, err := huellaDeclaracionAutoridadObjetoEsperadoDocumentalV1(
		escenario.materializacion.declaracion,
	)
	if err != nil {
		t.Fatal(err)
	}
	if proyeccion.Esquema != EsquemaAutoridadObjetoEsperadoDocumentalV1 ||
		proyeccion.Version != VersionAutoridadObjetoEsperadoDocumentalV1 ||
		proyeccion.ReciboMaterialRef != escenario.recibo.referenciaDurableOriginal ||
		proyeccion.HuellaReciboMaterialSHA256 != hex.EncodeToString(huellaRecibo[:]) ||
		proyeccion.HuellaDeclaracionV4SHA256 != hex.EncodeToString(huellaDeclaracion[:]) ||
		proyeccion.Objeto != escenario.materializacion.resultado.Objeto.Objeto ||
		proyeccion.ConectorID != escenario.materializacion.resultado.Objeto.ConectorID ||
		proyeccion.EfectoRef != contexto.EfectoRef ||
		proyeccion.HuellaPlanEfectoSHA256 != contexto.HuellaPlanEfectoSHA256 ||
		proyeccion.HuellaManifiestoSHA256 != contexto.HuellaManifiestoSHA256 ||
		proyeccion.PasoRef != contexto.PasoRef ||
		proyeccion.HuellaPasoSHA256 != contexto.HuellaPasoSHA256 ||
		!bytes.Equal(proyeccion.ReciboMaterialCanonico, canonico) {
		t.Fatal("la proyeccion no conserva el cruce exacto V4/recibo material")
	}
	atestacion := proyeccion.AtestacionReciboMaterial
	if atestacion.Algoritmo != AlgoritmoAtestacionMaterialHMACSHA256 ||
		atestacion.ClaveRef != escenario.criptografia.claveRef ||
		atestacion.ClaveVersion != escenario.criptografia.claveVersion ||
		atestacion.Dominio != dominioAtestacionReciboEscrituraMaterialV2 ||
		len(atestacion.Codigo) != sha256.Size {
		t.Fatal("la proyeccion no entrega material verificable del recibo")
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1DerivaParejaSoloDeInstantaneaV2(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	esperada := ReferenciaObjetoAlmacen{
		Referencia: escenario.recibo.instantanea.objetoRef,
		Version:    escenario.recibo.instantanea.objetoVersion,
	}
	// La declaracion aportada queda fuera de la capacidad tras el cotejo. Estas
	// alteraciones privadas simulan un propietario que conserva su copia.
	escenario.materializacion.declaracion.datos.resultado.Objeto.Objeto.Version =
		"version:documental:v4:externa"
	escenario.materializacion.declaracion.datos.solicitud.Contexto.EfectoRef =
		"efecto:documental:v4:externo"
	escenario.materializacion.declaracion.datos.contexto.datos.efectoRef =
		"efecto:documental:v4:contexto-externo"
	escenario.recibo.atestacion.codigo[0] ^= 1
	manifiesto := escenario.materializacion.declaracion.datos.vinculoEjecucion.datos.
		vinculoActivacion.Manifiesto.datos
	manifiesto.BorradorRef += ":mutado-por-propietario"
	manifiesto.HuellaPlanSHA256 = strings.Repeat("4", 64)
	proyeccion, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil || proyeccion.Objeto != esperada {
		t.Fatalf("la pareja no procede de la instantanea V2 sellada: proyeccion=%v err=%v", proyeccion, err)
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1AdmiteReciboCOSESign1Verificado(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	semilla := sha256.Sum256([]byte("semilla-ed25519-cose-sign1-autoridad-objeto-v1"))
	privada := ed25519.NewKeyFromSeed(semilla[:])
	verificador := verificadorCOSESign1AutoridadObjetoV1Prueba{
		publica:  privada.Public().(ed25519.PublicKey),
		claveRef: "clave:cose-sign1:objeto-esperado:v1", claveVersion: 19,
	}
	recibo := escenario.recibo
	canonico, err := recibo.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	if err != nil {
		t.Fatal(err)
	}
	recibo.atestacion, err = NuevaAtestacionCriptograficaMaterialAlmacenV2(
		solicitud, AlgoritmoAtestacionMaterialCOSESign1,
		verificador.claveRef, verificador.claveVersion,
		nuevoSobreCOSESign1AutoridadObjetoV1Prueba(
			privada, dominioAtestacionReciboEscrituraMaterialV2, canonico,
		),
	)
	if err != nil || recibo.Validar() != nil {
		t.Fatalf("crear recibo COSE Sign1: %v", err)
	}
	autoridad, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), escenario.materializacion.declaracion, recibo,
		escenario.registro, verificador,
	)
	if err != nil {
		t.Fatalf("crear autoridad con COSE Sign1: %v", err)
	}
	proyeccion, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, verificador,
	)
	if err != nil ||
		proyeccion.AtestacionReciboMaterial.Algoritmo != AlgoritmoAtestacionMaterialCOSESign1 ||
		!bytes.Equal(proyeccion.AtestacionReciboMaterial.Codigo, recibo.atestacion.codigo) {
		t.Fatalf("preparar recibo COSE Sign1: proyeccion=%v err=%v", proyeccion, err)
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1EntregaCopiasYPermaneceOpaca(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	primera, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatal(err)
	}
	canonicoOriginal := append([]byte(nil), primera.ReciboMaterialCanonico...)
	codigoOriginal := append([]byte(nil), primera.AtestacionReciboMaterial.Codigo...)
	primera.ReciboMaterialCanonico[0] ^= 1
	primera.AtestacionReciboMaterial.Codigo[0] ^= 1
	segunda, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(segunda.ReciboMaterialCanonico, canonicoOriginal) ||
		!bytes.Equal(segunda.AtestacionReciboMaterial.Codigo, codigoOriginal) {
		t.Fatal("el consumidor pudo mutar el material sellado")
	}
	for nombre, valor := range map[string]any{
		"autoridad":  autoridad,
		"proyeccion": segunda,
	} {
		if _, err := json.Marshal(valor); !errors.Is(
			err, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida,
		) {
			t.Fatalf("%s admitio JSON generico: %v", nombre, err)
		}
		texto := fmt.Sprintf("%+v", valor)
		if texto != textoAutoridadObjetoEsperadoDocumentalV1Redactado ||
			strings.Contains(texto, escenario.materializacion.resultado.Objeto.Objeto.Referencia) {
			t.Fatalf("%s se filtro por formato: %q", nombre, texto)
		}
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Info("prueba", "autoridad", autoridad)
	if strings.Contains(bitacora.String(), escenario.materializacion.resultado.Objeto.Objeto.Referencia) {
		t.Fatal("la autoridad filtro la referencia en logging")
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1RechazaEvidenciaSinReciboYObjetoAjeno(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	if huellaEvidenciaOperacionAlmacenDocumental(
		escenario.materializacion.resultado.Evidencia,
	) == "" {
		t.Fatal("precondicion: la evidencia autocontenida debe tener huella")
	}
	if _, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), escenario.materializacion.declaracion,
		ReciboEscrituraObjetoMaterialV2{}, escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("la evidencia V4 sin recibo autenticado creo autoridad: %v", err)
	}

	resultadoAjeno := escenario.materializacion.resultado
	resultadoAjeno.Objeto.Objeto = ReferenciaObjetoAlmacen{
		Referencia: "objeto:documental:v4:ajeno", Version: "version:documental:v4:ajena",
	}
	resultadoAjeno.Evidencia = evidenciaAlmacenVinculadaPrueba(
		escenario.materializacion.solicitud.Contexto, resultadoAjeno.Objeto.Objeto,
		"evidencia:almacen:v4:ajena", resultadoAjeno.Objeto.ConectorID, "",
		resultadoAjeno.Objeto.AlmacenadoEn,
	)
	resultadoAjeno.Objeto.EvidenciaCreacionRef = resultadoAjeno.Evidencia.Referencia
	reciboAjeno := escenario.crearRecibo(t, resultadoAjeno)
	if _, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
		context.Background(), escenario.materializacion.declaracion, reciboAjeno,
		escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("se acepto recibo material de otra pareja objeto/version: %v", err)
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1FallaCerradaEnDependenciasYCancelacion(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	var verificadorNulo *criptografiaMaterialV2Prueba
	casos := []struct {
		nombre     string
		ctx        context.Context
		referencia VerificadorReferenciaReciboMaterialV2
		atestacion VerificadorAtestacionMaterialAlmacenV2
	}{
		{"contexto nulo", nil, escenario.registro, escenario.criptografia},
		{"contexto cancelado", ctxCancelado, escenario.registro, escenario.criptografia},
		{"referencia nula tipada", context.Background(), (*registroAutoritativoMaterialV2Prueba)(nil), escenario.criptografia},
		{"atestacion nula tipada", context.Background(), escenario.registro, verificadorNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaAutoridadObjetoEsperadoDocumentalV1(
				caso.ctx, escenario.materializacion.declaracion, escenario.recibo,
				caso.referencia, caso.atestacion,
			); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
				t.Fatalf("dependencia invalida admitida: %v", err)
			}
		})
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1ReverificaAntesDelRegistro(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	escenario.criptografia.rechazar = true
	if _, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("se tolero atestacion revocada: %v", err)
	}
	escenario.criptografia.rechazar = false
	escenario.registro.rechazarReferencia = true
	if _, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
		t.Fatalf("se tolero referencia durable revocada: %v", err)
	}
	escenario.registro.rechazarReferencia = false
	if _, err := autoridad.PrepararRegistro(
		context.Background(), escenario.registro, escenario.criptografia,
	); err != nil {
		t.Fatalf("la autoridad no se recupero con autoridades validas: %v", err)
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1CancelacionDuranteCadaVerificadorDevuelveCero(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	cero := ProyeccionAutoridadObjetoEsperadoDocumentalV1{}

	t.Run("atestacion", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		proyeccion, err := autoridad.PrepararRegistro(
			ctx, escenario.registro,
			verificadorAtestacionCanceladorAutoridadObjetoV1Prueba{
				delegado: escenario.criptografia, cancelar: cancelar,
			},
		)
		if !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) ||
			!errors.Is(ctx.Err(), context.Canceled) || !reflect.DeepEqual(proyeccion, cero) {
			t.Fatalf("cancelacion durante atestacion no fallo cerrada: proyeccion=%v err=%v", proyeccion, err)
		}
	})

	t.Run("referencia durable", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		proyeccion, err := autoridad.PrepararRegistro(
			ctx,
			verificadorReferenciaCanceladorAutoridadObjetoV1Prueba{
				delegado: escenario.registro, cancelar: cancelar,
			},
			escenario.criptografia,
		)
		if !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) ||
			!errors.Is(ctx.Err(), context.Canceled) || !reflect.DeepEqual(proyeccion, cero) {
			t.Fatalf("cancelacion durante referencia no fallo cerrada: proyeccion=%v err=%v", proyeccion, err)
		}
	})
}

func probarAutoridadObjetoEsperadoDocumentalV1RevalidaCompromisosTrasVerificadoresVivos(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	proyeccion, err := autoridad.PrepararRegistro(
		context.Background(),
		verificadorReferenciaCanceladorAutoridadObjetoV1Prueba{
			delegado: escenario.registro,
			mutar: func() {
				autoridad.datos.huellaManifiestoSHA256 = strings.Repeat("3", 64)
			},
		},
		escenario.criptografia,
	)
	if !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) ||
		!reflect.DeepEqual(proyeccion, ProyeccionAutoridadObjetoEsperadoDocumentalV1{}) {
		t.Fatalf("la mutacion durante reverificacion alcanzo registro: proyeccion=%v err=%v", proyeccion, err)
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1MutacionDuranteCadaVerificadorDevuelveCero(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridadBase := escenario.crearAutoridad(t)
	cero := ProyeccionAutoridadObjetoEsperadoDocumentalV1{}
	for _, caso := range []struct {
		nombre     string
		atestacion bool
		mutar      func(AutoridadObjetoEsperadoDocumentalV1)
	}{
		{"atestacion/recibo", true, func(a AutoridadObjetoEsperadoDocumentalV1) {
			a.datos.recibo.atestacion.codigo[0] ^= 1
		}},
		{"atestacion/vinculo-manifiesto", true, func(a AutoridadObjetoEsperadoDocumentalV1) {
			a.datos.declaracion.datos.vinculoEjecucion.datos.vinculoActivacion.
				Manifiesto.datos.BorradorRef += ":alterado"
		}},
		{"referencia/recibo", false, func(a AutoridadObjetoEsperadoDocumentalV1) {
			a.datos.recibo.atestacion.codigo[0] ^= 1
		}},
		{"referencia/vinculo-manifiesto", false, func(a AutoridadObjetoEsperadoDocumentalV1) {
			a.datos.declaracion.datos.vinculoEjecucion.datos.vinculoActivacion.
				Manifiesto.datos.BorradorRef += ":alterado"
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := clonarAutoridadObjetoEsperadoDocumentalV1Prueba(autoridadBase)
			mutar := func() { caso.mutar(autoridad) }
			var proyeccion ProyeccionAutoridadObjetoEsperadoDocumentalV1
			var err error
			if caso.atestacion {
				proyeccion, err = autoridad.PrepararRegistro(
					context.Background(), escenario.registro,
					verificadorAtestacionCanceladorAutoridadObjetoV1Prueba{
						delegado: escenario.criptografia, mutar: mutar,
					},
				)
			} else {
				proyeccion, err = autoridad.PrepararRegistro(
					context.Background(),
					verificadorReferenciaCanceladorAutoridadObjetoV1Prueba{
						delegado: escenario.registro, mutar: mutar,
					},
					escenario.criptografia,
				)
			}
			if !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) ||
				!reflect.DeepEqual(proyeccion, cero) {
				t.Fatalf("la mutacion viva alcanzo registro: proyeccion=%v err=%v", proyeccion, err)
			}
		})
	}
	if err := autoridadBase.Validar(); err != nil {
		t.Fatalf("los casos compartieron estado mutable con el fixture base: %v", err)
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1DetectaAlteracionInterna(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	mutaciones := map[string]func(*datosAutoridadObjetoEsperadoDocumentalV1){
		"objeto": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.objeto.Version = "version:documental:v4:alterada"
		},
		"recibo": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.recibo.atestacion.codigo[0] ^= 1
		},
		"atestacion conservada": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.atestacionRecibo.Codigo[0] ^= 1
		},
		"sello atestacion": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.selloAtestacionRecibo[0] ^= 1
		},
		"referencia durable": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.reciboRef += ":alterada"
		},
		"conector": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.conectorID += ":alterado"
		},
		"efecto": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.efectoRef += ":alterado"
		},
		"plan": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.huellaPlanEfectoSHA256 = strings.Repeat("1", 64)
		},
		"manifiesto": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.huellaManifiestoSHA256 = strings.Repeat("2", 64)
		},
		"paso": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.pasoRef = PasoOperacionAlmacen("99_paso_alterado")
		},
		"contexto V4": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.contextoV4.CorrelacionRef += ":alterada"
		},
		"declaracion V4 exacta": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.huellaDeclaracionV4[0] ^= 1
		},
		"vinculo manifiesto V4": func(d *datosAutoridadObjetoEsperadoDocumentalV1) {
			d.declaracion.datos.vinculoEjecucion.datos.vinculoActivacion.
				Manifiesto.datos.BorradorRef += ":alterado"
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			copia := clonarAutoridadObjetoEsperadoDocumentalV1Prueba(autoridad)
			mutar(copia.datos)
			if err := copia.Validar(); !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) {
				t.Fatal("Validar admitio un compromiso alterado")
			}
			proyeccion, err := copia.PrepararRegistro(
				context.Background(), escenario.registro, escenario.criptografia,
			)
			if !errors.Is(err, ErrAutoridadObjetoEsperadoDocumentalV1NoValida) ||
				!reflect.DeepEqual(proyeccion, ProyeccionAutoridadObjetoEsperadoDocumentalV1{}) {
				t.Fatal("la alteracion interna alcanzo la frontera de registro")
			}
		})
	}
}

func probarAutoridadObjetoEsperadoDocumentalV1PreparacionConcurrente(
	t *testing.T,
	escenario escenarioAutoridadObjetoEsperadoDocumentalV1Prueba,
) {
	autoridad := escenario.crearAutoridad(t)
	const repeticiones = 24
	errores := make(chan error, repeticiones)
	var grupo sync.WaitGroup
	grupo.Add(repeticiones)
	for i := 0; i < repeticiones; i++ {
		go func() {
			defer grupo.Done()
			proyeccion, err := autoridad.PrepararRegistro(
				context.Background(), escenario.registro, escenario.criptografia,
			)
			if err == nil && proyeccion.Objeto != escenario.materializacion.resultado.Objeto.Objeto {
				err = errors.New("pareja de objeto inestable")
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("preparacion concurrente: %v", err)
		}
	}
}
