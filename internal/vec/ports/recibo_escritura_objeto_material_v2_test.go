package ports

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

var instanteReciboMaterialV2Prueba = time.Date(2026, 7, 15, 12, 34, 56, 789000000, time.UTC)

type criptografiaMaterialV2Prueba struct {
	clave             []byte
	claveRef          string
	claveVersion      uint32
	rechazar          bool
	alterarAlFirmar   bool
	cancelarAlAtestar context.CancelFunc
}

type perfilPublicadoMaterialV2Prueba struct {
	version     uint32
	capacidades CapacidadesAlmacenObjetos
}

type planPublicadoMaterialV2Prueba struct {
	version          uint32
	huella           string
	conectorLogicoID string
	moduloID         string
	accionNegocio    string
	accionTecnica    string
	recursoRef       string
	operacionRef     string
	cargaRef         string
	efectoRef        string
	clasificacion    string
}

type registroAutoritativoMaterialV2Prueba struct {
	perfiles                 map[string]perfilPublicadoMaterialV2Prueba
	planes                   map[string]planPublicadoMaterialV2Prueba
	referencias              map[string]string
	rechazarPerfil           bool
	rechazarPlan             bool
	rechazarReferencia       bool
	devolverReferenciaFisica bool
	cancelarEnPerfil         context.CancelFunc
	cancelarEnReserva        context.CancelFunc
}

func nuevoRegistroAutoritativoMaterialV2Prueba() *registroAutoritativoMaterialV2Prueba {
	return &registroAutoritativoMaterialV2Prueba{
		perfiles:    make(map[string]perfilPublicadoMaterialV2Prueba),
		planes:      make(map[string]planPublicadoMaterialV2Prueba),
		referencias: make(map[string]string),
	}
}

func (r *registroAutoritativoMaterialV2Prueba) VerificarPerfilPublicadoMaterialV2(
	_ context.Context,
	solicitud SolicitudVerificarPerfilPublicadoMaterialV2,
) error {
	if r.cancelarEnPerfil != nil {
		r.cancelarEnPerfil()
	}
	if r.rechazarPerfil {
		return errors.New("perfil no publicado")
	}
	referencia, version, conector, huella, canonico, err := solicitud.RevelarParaHomologacion()
	publicado, existe := r.perfiles[referencia]
	if err != nil || !existe || version != publicado.version ||
		conector != publicado.capacidades.ConectorID {
		return errors.New("perfil ajeno")
	}
	esperado := perfilMaterialSinAtestacionPrueba(referencia, version, publicado.capacidades)
	bytesEsperados, err := esperado.canonicoSinAtestacion()
	huellaEsperada := sha256.Sum256(bytesEsperados)
	if err != nil || !bytes.Equal(canonico, bytesEsperados) || huella != huellaEsperada {
		return errors.New("perfil no homologado")
	}
	return nil
}

func perfilMaterialSinAtestacionPrueba(
	referencia string,
	version uint32,
	capacidades CapacidadesAlmacenObjetos,
) PerfilCapacidadesAlmacenMaterialV2 {
	return PerfilCapacidadesAlmacenMaterialV2{
		esquema:        EsquemaPerfilCapacidadesAlmacenMaterialV2,
		versionEsquema: VersionEsquemaMaterialAlmacenV2,
		referencia:     referencia, version: version,
		conectorLogicoID:  capacidades.ConectorID,
		escrituraEnFlujo:  capacidades.EscrituraEnFlujo,
		referenciasOpacas: capacidades.ReferenciasOpacas,
		integridadSHA256:  capacidades.IntegridadSHA256,
		versionado:        capacidades.Versionado, retencion: capacidades.Retencion,
		bloqueoLegal:           capacidades.BloqueoLegal,
		cifradoEnTransito:      capacidades.CifradoEnTransito,
		cifradoEnReposo:        capacidades.CifradoEnReposo,
		cifradoPorObjeto:       capacidades.CifradoPorObjeto,
		preservaObjetoOriginal: capacidades.PreservaObjetoOriginal,
		tamanoMaximoObjeto:     capacidades.TamanoMaximoObjeto,
	}
}

func (r *registroAutoritativoMaterialV2Prueba) VerificarPlanMaterialAlmacenV2(
	_ context.Context,
	solicitud SolicitudVerificarPlanMaterialAlmacenV2,
) (ResultadoVerificacionPlanMaterialAlmacenV2, error) {
	if r.rechazarPlan {
		return ResultadoVerificacionPlanMaterialAlmacenV2{}, errors.New("plan no publicado")
	}
	referencia, version, conector, modulo, accionNegocio, accionTecnica, recurso,
		operacion, carga, efecto, clasificacion, _, err :=
		solicitud.RevelarParaVerificacionPlanMaterial()
	publicado, existe := r.planes[referencia]
	if err != nil || !existe || version != publicado.version ||
		conector != publicado.conectorLogicoID || modulo != publicado.moduloID ||
		accionNegocio != publicado.accionNegocio || accionTecnica != publicado.accionTecnica ||
		recurso != publicado.recursoRef || operacion != publicado.operacionRef ||
		carga != publicado.cargaRef || efecto != publicado.efectoRef ||
		clasificacion != publicado.clasificacion {
		return ResultadoVerificacionPlanMaterialAlmacenV2{}, errors.New("plan no ligado")
	}
	return NuevoResultadoVerificacionPlanMaterialAlmacenV2(solicitud, publicado.huella)
}

func (r *registroAutoritativoMaterialV2Prueba) ReservarORecuperarReferenciaReciboMaterialV2(
	_ context.Context,
	solicitud SolicitudReservarReferenciaReciboMaterialV2,
) (ResultadoReferenciaReciboMaterialV2, error) {
	huella, err := solicitud.HuellaIdentidad()
	if err != nil {
		return ResultadoReferenciaReciboMaterialV2{}, err
	}
	indice := hex.EncodeToString(huella[:])
	referencia, existe := r.referencias[indice]
	if !existe {
		referencia = "recibo:material:durable:" + indice[:32]
		r.referencias[indice] = referencia
	}
	if r.devolverReferenciaFisica {
		referencia = "bucket:recibos"
	}
	resultado, err := NuevoResultadoReferenciaReciboMaterialV2(solicitud, referencia)
	if r.cancelarEnReserva != nil {
		r.cancelarEnReserva()
	}
	return resultado, err
}

func (r *registroAutoritativoMaterialV2Prueba) VerificarReferenciaReciboMaterialV2(
	_ context.Context,
	solicitud SolicitudReservarReferenciaReciboMaterialV2,
	resultado ResultadoReferenciaReciboMaterialV2,
) error {
	if r.rechazarReferencia || resultado.validarPara(solicitud) != nil {
		return errors.New("referencia no verificada")
	}
	huella, _ := solicitud.HuellaIdentidad()
	returnado, existe := r.referencias[hex.EncodeToString(huella[:])]
	if !existe || resultado.referencia != returnado {
		return errors.New("referencia no original")
	}
	return nil
}

func (c *criptografiaMaterialV2Prueba) AtestarMaterialAlmacenV2(
	_ context.Context,
	solicitud SolicitudAtestarMaterialAlmacenV2,
) (AtestacionCriptograficaMaterialAlmacenV2, error) {
	dominio, mensaje, _, err := solicitud.RevelarParaAtestacion()
	if err != nil {
		return AtestacionCriptograficaMaterialAlmacenV2{}, err
	}
	codigo := codigoHMACMaterialV2Prueba(c.clave, dominio, mensaje)
	if c.alterarAlFirmar {
		codigo[0] ^= 1
	}
	atestacion, err := NuevaAtestacionCriptograficaMaterialAlmacenV2(
		solicitud, AlgoritmoAtestacionMaterialHMACSHA256,
		c.claveRef, c.claveVersion, codigo,
	)
	if c.cancelarAlAtestar != nil {
		c.cancelarAlAtestar()
	}
	return atestacion, err
}

func (c *criptografiaMaterialV2Prueba) VerificarAtestacionMaterialAlmacenV2(
	_ context.Context,
	solicitud SolicitudVerificarAtestacionMaterialAlmacenV2,
) error {
	if c.rechazar {
		return errors.New("atestacion rechazada")
	}
	dominio, mensaje, algoritmo, claveRef, claveVersion, codigo, err :=
		solicitud.RevelarParaVerificacion()
	if err != nil || algoritmo != AlgoritmoAtestacionMaterialHMACSHA256 ||
		claveRef != c.claveRef || claveVersion != c.claveVersion ||
		!hmac.Equal(codigo, codigoHMACMaterialV2Prueba(c.clave, dominio, mensaje)) {
		return errors.New("atestacion falsa")
	}
	return nil
}

func codigoHMACMaterialV2Prueba(clave []byte, dominio string, mensaje []byte) []byte {
	mac := hmac.New(sha256.New, clave)
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(dominio)))
	_, _ = mac.Write(longitud[:])
	_, _ = mac.Write([]byte(dominio))
	binary.BigEndian.PutUint64(longitud[:], uint64(len(mensaje)))
	_, _ = mac.Write(longitud[:])
	_, _ = mac.Write(mensaje)
	return mac.Sum(nil)
}

type escenarioReciboMaterialV2Prueba struct {
	solicitud    SolicitudEscribirObjeto
	resultado    ResultadoOperacionObjeto
	capacidades  CapacidadesAlmacenObjetos
	perfil       PerfilCapacidadesAlmacenMaterialV2
	plan         SeleccionPlanMaterialAlmacenV2
	criptografia *criptografiaMaterialV2Prueba
	registro     *registroAutoritativoMaterialV2Prueba
}

func nuevoEscenarioReciboMaterialV2Prueba(t *testing.T) escenarioReciboMaterialV2Prueba {
	t.Helper()
	contextoOperacion := contextoAlmacenVinculadoPrueba(t, AccionAlmacenEscribir)
	solicitud := SolicitudEscribirObjeto{
		Contexto: contextoOperacion, ClaveIdempotencia: "idempotencia:material:v2:001",
		Zona: ZonaAlmacenAdmitida, MIME: "application/pdf", Tamano: 128,
		HuellaSHA256: strings.Repeat("a", 64), Contenido: strings.NewReader(strings.Repeat("x", 128)),
	}
	objetoRef := ReferenciaObjetoAlmacen{
		Referencia: "objeto:material:v2:001", Version: "version:material:v2:007",
	}
	evidencia := evidenciaAlmacenVinculadaPrueba(
		contextoOperacion, objetoRef, "evidencia:creacion:material:v2:001",
		"conector:almacen:corporativo", "", instanteReciboMaterialV2Prueba,
	)
	resultado := ResultadoOperacionObjeto{
		Objeto: ObjetoAlmacenado{
			Objeto: objetoRef, ConectorID: evidencia.ConectorID, Zona: solicitud.Zona,
			MIME: solicitud.MIME, Tamano: solicitud.Tamano, HuellaSHA256: solicitud.HuellaSHA256,
			EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: instanteReciboMaterialV2Prueba,
			RetenidoHasta: instanteReciboMaterialV2Prueba.Add(365 * 24 * time.Hour),
			Inmovilizado:  true,
		},
		Evidencia: evidencia,
	}
	capacidades := CapacidadesAlmacenObjetos{
		ConectorID: evidencia.ConectorID, EscrituraEnFlujo: true,
		ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
		Retencion: true, BloqueoLegal: true, CifradoEnTransito: true,
		CifradoEnReposo: true, CifradoPorObjeto: true,
		TamanoMaximoObjeto: 32 << 20, PreservaObjetoOriginal: true,
	}
	criptografia := &criptografiaMaterialV2Prueba{
		clave:    []byte("clave-exclusiva-material-v2-de-prueba-32-bytes"),
		claveRef: "clave:atestacion:material:v2", claveVersion: 7,
	}
	registro := nuevoRegistroAutoritativoMaterialV2Prueba()
	const perfilRef = "perfil:capacidades:material:v2:001"
	registro.perfiles[perfilRef] = perfilPublicadoMaterialV2Prueba{
		version: 3, capacidades: capacidades,
	}
	perfil, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), perfilRef, 3,
		capacidades, criptografia, criptografia, registro,
	)
	if err != nil {
		t.Fatalf("crear perfil atestado: %v", err)
	}
	plan, err := NuevaSeleccionPlanMaterialAlmacenV2("plan:material:v2:001", 5)
	if err != nil {
		t.Fatalf("seleccionar plan material: %v", err)
	}
	proyeccion, _ := solicitud.Contexto.Proyeccion()
	registro.planes[plan.referencia] = planPublicadoMaterialV2Prueba{
		version: plan.version, huella: strings.Repeat("9", 64),
		conectorLogicoID: capacidades.ConectorID, moduloID: proyeccion.ModuloID,
		accionNegocio: proyeccion.AccionNegocio, accionTecnica: proyeccion.AccionTecnica,
		recursoRef: proyeccion.RecursoRef, operacionRef: proyeccion.OperacionRef,
		cargaRef: proyeccion.CargaRef, efectoRef: proyeccion.EfectoRef,
		clasificacion: proyeccion.Clasificacion,
	}
	return escenarioReciboMaterialV2Prueba{
		solicitud: solicitud, resultado: resultado, capacidades: capacidades,
		perfil: perfil, plan: plan, criptografia: criptografia, registro: registro,
	}
}

func (e escenarioReciboMaterialV2Prueba) crearRecibo(
	t *testing.T,
	resultado ResultadoOperacionObjeto,
) ReciboEscrituraObjetoMaterialV2 {
	t.Helper()
	recibo, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), e.solicitud, resultado, e.capacidades, e.perfil, e.registro, e.plan,
		e.registro, e.registro, e.registro,
		e.criptografia, e.criptografia,
	)
	if err != nil {
		t.Fatalf("crear recibo material: %v", err)
	}
	return recibo
}

func TestReciboEscrituraObjetoMaterialV2VectorDoradoTLVYHuella(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	recibo := escenario.crearRecibo(t, escenario.resultado)
	canonico, err := recibo.BytesCanonicos()
	if err != nil {
		t.Fatalf("obtener bytes canonicos: %v", err)
	}
	huella, err := recibo.HuellaSHA256()
	if err != nil {
		t.Fatalf("obtener huella: %v", err)
	}
	const hexadecimalDorado = "000000000000000000287665632e616c6d6163656e2e72656369626f2d6573637269747572612d6d6174657269616c2e76320001000000000000000200020002000000000000003872656369626f3a6d6174657269616c3a64757261626c653a66383936663238616637343731616663653638363735343431366232396362650003000000000000001c636f6e6563746f723a616c6d6163656e3a636f72706f72617469766f0004000000000000002270657266696c3a63617061636964616465733a6d6174657269616c3a76323a30303100050000000000000004000000030006000000000000002007c9049cbfedd70e4b7b918efb7fe462eef89ecf443549c3bef5ff060fb53fbe00070000000000000005626f6c736100080000000000000018626f6c73612e6465636973696f6e2e637573746f64696172000900000000000000086573637269626972000a00000000000000137265637572736f3a616c6d6163656e3a303031000b00000000000000156f7065726163696f6e3a616c6d6163656e3a303031000c000000000000001463617267613a646f63756d656e74616c3a303031000d000000000000001265666563746f3a616c6d6163656e3a303031000e0000000000000014706c616e3a6d6174657269616c3a76323a303031000f000000000000000400000005001000000000000000209999999999999999999999999999999999999999999999999999999999999999001100000000000000207cb30b9524a2bacbe349610b88e0e108969092e89ab651def3338afc57b59c5c001200000000000000156461746f735f706572736f6e616c65735f616c7461001300000000000000166f626a65746f3a6d6174657269616c3a76323a3030310014000000000000001776657273696f6e3a6d6174657269616c3a76323a3030370015000000000000000861646d69746964610016000000000000000f6170706c69636174696f6e2f70646600170000000000000008000000000000008000180000000000000020aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa0019000000000000002265766964656e6369613a6372656163696f6e3a6d6174657269616c3a76323a303031001a0000000000000008000656a58d148608001b000000000000000101001c00000000000000080006735419286608001d000000000000000c696e6d6f76696c697a61646f001e000000000000000661637469766f"
	const huellaDorada = "8c02c0cd606a3028f7f3deed4ec0fcde1d7a303a85bc44f0af9bb8ef3a5ad07a"
	if hex.EncodeToString(canonico) != hexadecimalDorado || hex.EncodeToString(huella[:]) != huellaDorada {
		t.Fatalf("actualizar vector dorado:\nHEX=%s\nSHA=%s", hex.EncodeToString(canonico), hex.EncodeToString(huella[:]))
	}
	etiquetas := leerEtiquetasTLVMaterialV2Prueba(t, canonico)
	if len(etiquetas) != 31 || etiquetas[0] != 0 || etiquetas[len(etiquetas)-1] != 30 {
		t.Fatalf("secuencia TLV inesperada: %v", etiquetas)
	}
	if bytes.Contains(canonico, []byte(escenario.resultado.Evidencia.CorrelacionRef)) ||
		bytes.Contains(canonico, []byte(escenario.resultado.Evidencia.AutorizacionRef)) ||
		bytes.Contains(canonico, []byte(escenario.resultado.Evidencia.SujetoSeudonimoHMAC)) ||
		bytes.Contains(canonico, []byte(escenario.resultado.Evidencia.HuellaSolicitudHMAC)) ||
		bytes.Contains(canonico, []byte(escenario.resultado.Evidencia.HuellaPlanEfectoSHA256)) {
		t.Fatal("la preimagen material contiene datos del intento o el plan V1")
	}
}

func leerEtiquetasTLVMaterialV2Prueba(t *testing.T, canonico []byte) []uint16 {
	t.Helper()
	var etiquetas []uint16
	for posicion := 0; posicion < len(canonico); {
		if len(canonico)-posicion < 10 {
			t.Fatal("cabecera TLV truncada")
		}
		etiqueta := binary.BigEndian.Uint16(canonico[posicion : posicion+2])
		longitud := binary.BigEndian.Uint64(canonico[posicion+2 : posicion+10])
		posicion += 10
		if longitud > uint64(len(canonico)-posicion) {
			t.Fatal("valor TLV truncado")
		}
		posicion += int(longitud)
		etiquetas = append(etiquetas, etiqueta)
	}
	return etiquetas
}

func TestReciboMaterialV2OriginalYReintentoUsanHechosOriginales(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	original := escenario.crearRecibo(t, escenario.resultado)
	reintento := escenario.resultado
	reintento.Evidencia.Referencia = "evidencia:respuesta:reintento:v2:002"
	reintento.Evidencia.RealizadaEn = instanteReciboMaterialV2Prueba.Add(17 * time.Second)
	reintento.Evidencia.ReintentoIdempotente = true
	if err := reintento.ValidarEscritura(escenario.solicitud, escenario.capacidades); err != nil {
		t.Fatalf("precondicion: reintento cotejado: %v", err)
	}
	recuperado := escenario.crearRecibo(t, reintento)
	bytesOriginal, _ := original.BytesCanonicos()
	bytesRecuperado, _ := recuperado.BytesCanonicos()
	huellaOriginal, _ := original.HuellaSHA256()
	huellaRecuperada, _ := recuperado.HuellaSHA256()
	if !bytes.Equal(bytesOriginal, bytesRecuperado) || huellaOriginal != huellaRecuperada {
		t.Fatal("una respuesta idempotente altero el recibo material")
	}
	if len(escenario.registro.referencias) != 1 ||
		original.referenciaDurableOriginal != recuperado.referenciaDurableOriginal {
		t.Fatal("el registro no recupero la referencia durable original")
	}
	instantanea, err := recuperado.Instantanea()
	if err != nil || instantanea.evidenciaCreacionRef != escenario.resultado.Objeto.EvidenciaCreacionRef ||
		!instantanea.almacenadoEn.Equal(escenario.resultado.Objeto.AlmacenadoEn) ||
		instantanea.evidenciaCreacionRef == reintento.Evidencia.Referencia ||
		instantanea.almacenadoEn.Equal(reintento.Evidencia.RealizadaEn) {
		t.Fatal("el reintento sustituyo la referencia o el instante originales")
	}
}

func TestReciboMaterialV2ExigePlanLigadoYReferenciaDurableVerificada(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	crearConRegistro := func(registro *registroAutoritativoMaterialV2Prueba) error {
		_, err := NuevoReciboEscrituraObjetoMaterialV2(
			context.Background(), escenario.solicitud, escenario.resultado,
			escenario.capacidades, escenario.perfil, registro, escenario.plan,
			registro, registro, registro, escenario.criptografia, escenario.criptografia,
		)
		return err
	}

	for nombre, mutar := range map[string]func(*planPublicadoMaterialV2Prueba){
		"operacion": func(p *planPublicadoMaterialV2Prueba) { p.operacionRef = "operacion:material:ajena" },
		"carga":     func(p *planPublicadoMaterialV2Prueba) { p.cargaRef = "carga:material:ajena" },
		"efecto":    func(p *planPublicadoMaterialV2Prueba) { p.efectoRef = "efecto:material:ajeno" },
	} {
		t.Run("plan_no_ligado_"+nombre, func(t *testing.T) {
			planNoLigado := *escenario.registro
			planNoLigado.planes = make(map[string]planPublicadoMaterialV2Prueba, len(escenario.registro.planes))
			for referencia, plan := range escenario.registro.planes {
				mutar(&plan)
				planNoLigado.planes[referencia] = plan
			}
			if err := crearConRegistro(&planNoLigado); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
				t.Fatalf("plan no ligado aceptado: %v", err)
			}
		})
	}

	planDenegado := *escenario.registro
	planDenegado.rechazarPlan = true
	if err := crearConRegistro(&planDenegado); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("plan no verificado aceptado: %v", err)
	}

	referenciaDenegada := *escenario.registro
	referenciaDenegada.rechazarReferencia = true
	if err := crearConRegistro(&referenciaDenegada); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("referencia no confirmada por el registro aceptada: %v", err)
	}

	var registroNulo *registroAutoritativoMaterialV2Prueba
	if _, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), escenario.solicitud, escenario.resultado,
		escenario.capacidades, escenario.perfil, registroNulo, escenario.plan,
		escenario.registro, escenario.registro, escenario.registro,
		escenario.criptografia, escenario.criptografia,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("homologador de perfil tipado nulo aceptado: %v", err)
	}
	if _, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), escenario.solicitud, escenario.resultado,
		escenario.capacidades, escenario.perfil, escenario.registro, escenario.plan,
		registroNulo, escenario.registro, escenario.registro,
		escenario.criptografia, escenario.criptografia,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("verificador de plan tipado nulo aceptado: %v", err)
	}
	if _, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), escenario.solicitud, escenario.resultado,
		escenario.capacidades, escenario.perfil, escenario.registro, escenario.plan,
		escenario.registro, registroNulo, escenario.registro,
		escenario.criptografia, escenario.criptografia,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("registro durable tipado nulo aceptado: %v", err)
	}
	if _, err := NuevoReciboEscrituraObjetoMaterialV2(
		context.Background(), escenario.solicitud, escenario.resultado,
		escenario.capacidades, escenario.perfil, escenario.registro, escenario.plan,
		escenario.registro, escenario.registro, registroNulo,
		escenario.criptografia, escenario.criptografia,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("verificador durable tipado nulo aceptado: %v", err)
	}
}

func TestReciboMaterialV2RevalidaPerfilYRechazaRevocacionOCancelacion(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	crear := func(ctx context.Context, homologador VerificadorPerfilPublicadoMaterialV2) error {
		_, err := NuevoReciboEscrituraObjetoMaterialV2(
			ctx, escenario.solicitud, escenario.resultado, escenario.capacidades,
			escenario.perfil, homologador, escenario.plan,
			escenario.registro, escenario.registro, escenario.registro,
			escenario.criptografia, escenario.criptografia,
		)
		return err
	}

	revocado := *escenario.registro
	revocado.rechazarPerfil = true
	if err := crear(context.Background(), &revocado); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("perfil revocado despues de construirse produjo recibo: %v", err)
	}

	ctxDurante, cancelarDurante := context.WithCancel(context.Background())
	cancelaAlConsultar := *escenario.registro
	cancelaAlConsultar.cancelarEnPerfil = cancelarDurante
	if err := crear(ctxDurante, &cancelaAlConsultar); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) ||
		!errors.Is(ctxDurante.Err(), context.Canceled) {
		t.Fatalf("cancelacion durante homologacion no fallo cerrada: err=%v ctx=%v", err, ctxDurante.Err())
	}

	referenciasAntes := len(escenario.registro.referencias)
	ctxAntes, cancelarAntes := context.WithCancel(context.Background())
	cancelarAntes()
	if err := crear(ctxAntes, escenario.registro); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) ||
		len(escenario.registro.referencias) != referenciasAntes {
		t.Fatalf("contexto ya cancelado alcanzo una frontera con efectos: %v", err)
	}

	ctxReserva, cancelarReserva := context.WithCancel(context.Background())
	registroCancela := *escenario.registro
	registroCancela.cancelarEnReserva = cancelarReserva
	if _, err := NuevoReciboEscrituraObjetoMaterialV2(
		ctxReserva, escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.perfil, escenario.registro, escenario.plan,
		escenario.registro, &registroCancela, &registroCancela,
		escenario.criptografia, escenario.criptografia,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) ||
		!errors.Is(ctxReserva.Err(), context.Canceled) {
		t.Fatalf("cancelacion durante reserva durable produjo recibo: err=%v ctx=%v", err, ctxReserva.Err())
	}

	ctxFirma, cancelarFirma := context.WithCancel(context.Background())
	firmaCancela := *escenario.criptografia
	firmaCancela.cancelarAlAtestar = cancelarFirma
	if _, err := NuevoReciboEscrituraObjetoMaterialV2(
		ctxFirma, escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.perfil, escenario.registro, escenario.plan,
		escenario.registro, escenario.registro, escenario.registro,
		&firmaCancela, escenario.criptografia,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) ||
		!errors.Is(ctxFirma.Err(), context.Canceled) {
		t.Fatalf("cancelacion durante atestacion produjo recibo: err=%v ctx=%v", err, ctxFirma.Err())
	}
}

func TestReciboMaterialV2RechazaContextoCruzadoPlanV1AliasFisicoYSubmicrosegundo(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	crear := func(solicitud SolicitudEscribirObjeto, resultado ResultadoOperacionObjeto,
		capacidades CapacidadesAlmacenObjetos, perfil PerfilCapacidadesAlmacenMaterialV2,
		plan SeleccionPlanMaterialAlmacenV2, registro *registroAutoritativoMaterialV2Prueba,
	) error {
		_, err := NuevoReciboEscrituraObjetoMaterialV2(
			context.Background(), solicitud, resultado, capacidades, perfil, registro, plan,
			registro, registro, registro, escenario.criptografia, escenario.criptografia,
		)
		return err
	}

	cruzado := escenario.resultado
	cruzado.Evidencia.OperacionRef = "operacion:almacen:ajena"
	if err := crear(escenario.solicitud, cruzado, escenario.capacidades,
		escenario.perfil, escenario.plan, escenario.registro); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("contexto cruzado aceptado: %v", err)
	}

	proyeccion, _ := escenario.solicitud.Contexto.Proyeccion()
	publicado := escenario.registro.planes[escenario.plan.referencia]
	publicado.huella = proyeccion.HuellaPlanEfectoSHA256
	escenario.registro.planes[escenario.plan.referencia] = publicado
	if err := crear(escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.perfil, escenario.plan, escenario.registro); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("plan de autorizacion V1 aceptado como material: %v", err)
	}
	publicado.huella = strings.Repeat("9", 64)
	escenario.registro.planes[escenario.plan.referencia] = publicado
	planNoPublicado, _ := NuevaSeleccionPlanMaterialAlmacenV2("plan:material:v2:no-publicado", 1)
	if err := crear(escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.perfil, planNoPublicado, escenario.registro); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("plan autodeclarado no publicado aceptado: %v", err)
	}

	submicro := escenario.resultado
	submicro.Objeto.AlmacenadoEn = submicro.Objeto.AlmacenadoEn.Add(time.Nanosecond)
	submicro.Evidencia.RealizadaEn = submicro.Objeto.AlmacenadoEn
	if err := crear(escenario.solicitud, submicro, escenario.capacidades,
		escenario.perfil, escenario.plan, escenario.registro); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("instante submicrosegundo aceptado: %v", err)
	}

	for _, alias := range []string{
		"bucket:documentos", "ruta:privada", "arn:aws:s3:objeto", "etag:abc",
		"kms:clave", "https://almacen.local", "objeto/privado", "objeto\\privado",
		"objeto..privado", "objeto?version", "objeto#fragmento", "dni:12345678Z",
		"nombre:persona", "correo:persona", "objeto:confusáble",
	} {
		t.Run(alias, func(t *testing.T) {
			if aliasLogicoMaterialV2Valido(alias, 512) {
				t.Fatalf("alias fisico aceptado: %q", alias)
			}
		})
	}
	escenario.registro.devolverReferenciaFisica = true
	if err := crear(escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.perfil, escenario.plan, escenario.registro); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("registro devolvio referencia fisica y fue aceptada: %v", err)
	}
	for _, pii := range []string{"12345678Z", "X1234567L", "persona@example.org"} {
		if aliasLogicoMaterialV2Valido(pii, 512) {
			t.Fatalf("identificador personal aparente aceptado: %q", pii)
		}
	}
}

func TestPerfilYReciboMaterialV2ExigenAtestacionCriptograficaVerdaderaYClaveVersionada(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	falso := *escenario.criptografia
	falso.rechazar = true
	escenario.registro.perfiles["perfil:capacidades:material:v2:falso"] = perfilPublicadoMaterialV2Prueba{
		version: 4, capacidades: escenario.capacidades,
	}
	if _, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:falso", 4,
		escenario.capacidades, escenario.criptografia, &falso, escenario.registro,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("perfil con atestacion no verificada aceptado: %v", err)
	}
	alterada := *escenario.criptografia
	alterada.alterarAlFirmar = true
	escenario.registro.perfiles["perfil:capacidades:material:v2:alterado"] = perfilPublicadoMaterialV2Prueba{
		version: 5, capacidades: escenario.capacidades,
	}
	if _, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:alterado", 5,
		escenario.capacidades, &alterada, escenario.criptografia, escenario.registro,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("perfil con HMAC falso aceptado: %v", err)
	}
	homologacionDenegada := *escenario.registro
	homologacionDenegada.rechazarPerfil = true
	if _, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:001", 3,
		escenario.capacidades, escenario.criptografia, escenario.criptografia,
		&homologacionDenegada,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("perfil firmado pero no homologado aceptado: %v", err)
	}
	capacidadesNoPublicadas := escenario.capacidades
	capacidadesNoPublicadas.TamanoMaximoObjeto++
	if _, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:001", 3,
		capacidadesNoPublicadas, escenario.criptografia, escenario.criptografia,
		escenario.registro,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("capacidades firmadas pero distintas del perfil publicado aceptadas: %v", err)
	}

	var atestadorNulo *criptografiaMaterialV2Prueba
	if _, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:nulo", 1,
		escenario.capacidades, atestadorNulo, escenario.criptografia, escenario.registro,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("dependencia tipada nula aceptada: %v", err)
	}
	var homologadorNulo *registroAutoritativoMaterialV2Prueba
	if _, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:001", 3,
		escenario.capacidades, escenario.criptografia, escenario.criptografia,
		homologadorNulo,
	); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("homologador tipado nulo aceptado: %v", err)
	}

	recibo := escenario.crearRecibo(t, escenario.resultado)
	if err := recibo.VerificarAtestacion(context.Background(), escenario.criptografia); err != nil {
		t.Fatalf("atestacion autentica rechazada: %v", err)
	}
	if err := recibo.VerificarAtestacion(context.Background(), &falso); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("atestacion falsa aceptada: %v", err)
	}
	claveDistinta := *escenario.criptografia
	claveDistinta.claveVersion++
	if err := recibo.VerificarAtestacion(context.Background(), &claveDistinta); !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
		t.Fatalf("otra version de clave aceptada: %v", err)
	}
}

func TestMaterialV2CopiasDefensivasYSerializacionRedactada(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	recibo := escenario.crearRecibo(t, escenario.resultado)
	canonicoPrimero, _ := recibo.BytesCanonicos()
	canonicoPrimero[0] ^= 1
	canonicoSegundo, _ := recibo.BytesCanonicos()
	if bytes.Equal(canonicoPrimero, canonicoSegundo) || recibo.Validar() != nil {
		t.Fatal("la mutacion de una copia altero el recibo")
	}

	canonico, _ := recibo.BytesCanonicos()
	solicitud, _ := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	_, mensajePrimero, _, _ := solicitud.RevelarParaAtestacion()
	mensajePrimero[0] ^= 1
	_, mensajeSegundo, _, _ := solicitud.RevelarParaAtestacion()
	if bytes.Equal(mensajePrimero, mensajeSegundo) {
		t.Fatal("la solicitud criptografica compartio su memoria")
	}
	codigo := codigoHMACMaterialV2Prueba(
		escenario.criptografia.clave, dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	atestacion, err := NuevaAtestacionCriptograficaMaterialAlmacenV2(
		solicitud, AlgoritmoAtestacionMaterialHMACSHA256,
		escenario.criptografia.claveRef, escenario.criptografia.claveVersion, codigo,
	)
	if err != nil {
		t.Fatal(err)
	}
	codigo[0] ^= 1
	peticion, _ := nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(solicitud, atestacion)
	_, _, _, _, _, codigoPrimero, _ := peticion.RevelarParaVerificacion()
	codigoPrimero[0] ^= 1
	_, _, _, _, _, codigoSegundo, _ := peticion.RevelarParaVerificacion()
	if bytes.Equal(codigoPrimero, codigoSegundo) ||
		escenario.criptografia.VerificarAtestacionMaterialAlmacenV2(context.Background(), peticion) != nil {
		t.Fatal("la atestacion compartio su memoria")
	}
	proyeccion, _ := escenario.solicitud.Contexto.Proyeccion()
	solicitudPlan, _ := nuevaSolicitudVerificarPlanMaterialAlmacenV2(
		escenario.plan, escenario.capacidades.ConectorID, proyeccion,
	)
	resultadoPlan, _ := escenario.registro.VerificarPlanMaterialAlmacenV2(
		context.Background(), solicitudPlan,
	)
	solicitudPerfil, _ := nuevaSolicitudVerificarPerfilPublicadoMaterialV2(escenario.perfil)
	solicitudReferencia, _ := nuevaSolicitudReservarReferenciaReciboMaterialV2([]byte("identidad-durable-prueba"))
	resultadoReferencia, _ := escenario.registro.ReservarORecuperarReferenciaReciboMaterialV2(
		context.Background(), solicitudReferencia,
	)

	valores := []any{
		escenario.plan, solicitudPlan, resultadoPlan, solicitudPerfil,
		solicitudReferencia, resultadoReferencia, solicitud, atestacion, peticion,
		escenario.perfil, recibo.instantanea, recibo,
	}
	for _, valor := range valores {
		t.Run(reflect.TypeOf(valor).Name(), func(t *testing.T) {
			texto := fmt.Sprintf("%v %#v %+v", valor, valor, valor)
			if texto != strings.TrimSpace(strings.Repeat(textoRedactadoMaterialAlmacenV2+" ", 3)) {
				t.Fatalf("formato no redactado: %s", texto)
			}
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionMaterialAlmacenV2Prohibida) {
				t.Fatalf("JSON generico permitido: %v", err)
			}
			if _, err := valor.(encoding.TextMarshaler).MarshalText(); !errors.Is(err, ErrSerializacionMaterialAlmacenV2Prohibida) {
				t.Fatalf("texto generico permitido: %v", err)
			}
			if _, err := valor.(encoding.BinaryMarshaler).MarshalBinary(); !errors.Is(err, ErrSerializacionMaterialAlmacenV2Prohibida) {
				t.Fatalf("binario generico permitido: %v", err)
			}
			registro := slog.AnyValue(valor).Resolve().String()
			if registro != textoRedactadoMaterialAlmacenV2 {
				t.Fatalf("slog no redactado: %s", registro)
			}
		})
	}
}

func TestMaterialV2MutacionDeCadaCampoQuedaCubiertaYFallaCerrada(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	recibo := escenario.crearRecibo(t, escenario.resultado)

	mutacionesPerfil := map[string]func(*PerfilCapacidadesAlmacenMaterialV2){
		"esquema":                func(p *PerfilCapacidadesAlmacenMaterialV2) { p.esquema += ".otro" },
		"versionEsquema":         func(p *PerfilCapacidadesAlmacenMaterialV2) { p.versionEsquema++ },
		"referencia":             func(p *PerfilCapacidadesAlmacenMaterialV2) { p.referencia += ":otra" },
		"version":                func(p *PerfilCapacidadesAlmacenMaterialV2) { p.version++ },
		"conectorLogicoID":       func(p *PerfilCapacidadesAlmacenMaterialV2) { p.conectorLogicoID += ":otro" },
		"escrituraEnFlujo":       func(p *PerfilCapacidadesAlmacenMaterialV2) { p.escrituraEnFlujo = !p.escrituraEnFlujo },
		"referenciasOpacas":      func(p *PerfilCapacidadesAlmacenMaterialV2) { p.referenciasOpacas = !p.referenciasOpacas },
		"integridadSHA256":       func(p *PerfilCapacidadesAlmacenMaterialV2) { p.integridadSHA256 = !p.integridadSHA256 },
		"versionado":             func(p *PerfilCapacidadesAlmacenMaterialV2) { p.versionado = !p.versionado },
		"retencion":              func(p *PerfilCapacidadesAlmacenMaterialV2) { p.retencion = !p.retencion },
		"bloqueoLegal":           func(p *PerfilCapacidadesAlmacenMaterialV2) { p.bloqueoLegal = !p.bloqueoLegal },
		"cifradoEnTransito":      func(p *PerfilCapacidadesAlmacenMaterialV2) { p.cifradoEnTransito = !p.cifradoEnTransito },
		"cifradoEnReposo":        func(p *PerfilCapacidadesAlmacenMaterialV2) { p.cifradoEnReposo = !p.cifradoEnReposo },
		"cifradoPorObjeto":       func(p *PerfilCapacidadesAlmacenMaterialV2) { p.cifradoPorObjeto = !p.cifradoPorObjeto },
		"preservaObjetoOriginal": func(p *PerfilCapacidadesAlmacenMaterialV2) { p.preservaObjetoOriginal = !p.preservaObjetoOriginal },
		"tamanoMaximoObjeto":     func(p *PerfilCapacidadesAlmacenMaterialV2) { p.tamanoMaximoObjeto++ },
		"huella":                 func(p *PerfilCapacidadesAlmacenMaterialV2) { p.huella[0] ^= 1 },
		"atestacion":             func(p *PerfilCapacidadesAlmacenMaterialV2) { p.atestacion.codigo[0] ^= 1 },
	}
	comprobarCoberturaMutacionesMaterialV2(t, reflect.TypeOf(escenario.perfil), mutacionesPerfil)
	for nombre, mutar := range mutacionesPerfil {
		t.Run("perfil_"+nombre, func(t *testing.T) {
			alterado := escenario.perfil
			alterado.atestacion.codigo = append([]byte(nil), escenario.perfil.atestacion.codigo...)
			mutar(&alterado)
			err := alterado.Validar()
			if nombre == "atestacion" {
				err = alterado.VerificarAtestacion(context.Background(), escenario.criptografia)
			}
			if !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
				t.Fatal("la mutacion del perfil no fallo cerrada")
			}
		})
	}

	mutacionesPlan := map[string]func(*HuellaPlanMaterialAlmacenV2){
		"referencia":    func(p *HuellaPlanMaterialAlmacenV2) { p.referencia += ":otra" },
		"version":       func(p *HuellaPlanMaterialAlmacenV2) { p.version++ },
		"suma":          func(p *HuellaPlanMaterialAlmacenV2) { p.suma[0] ^= 1 },
		"huellaVinculo": func(p *HuellaPlanMaterialAlmacenV2) { p.huellaVinculo[0] ^= 1 },
	}
	comprobarCoberturaMutacionesMaterialV2(t, reflect.TypeOf(recibo.huellaPlanMaterial), mutacionesPlan)
	for nombre, mutar := range mutacionesPlan {
		t.Run("plan_"+nombre, func(t *testing.T) {
			alterado := recibo
			alterado.atestacion.codigo = append([]byte(nil), recibo.atestacion.codigo...)
			mutar(&alterado.huellaPlanMaterial)
			if !errors.Is(alterado.Validar(), ErrReciboEscrituraObjetoMaterialV2NoValido) {
				t.Fatal("la mutacion del plan material no fallo cerrada")
			}
		})
	}

	mutacionesInstantanea := map[string]func(*InstantaneaObjetoMaterialV2){
		"esquema":              func(i *InstantaneaObjetoMaterialV2) { i.esquema += ".otro" },
		"versionEsquema":       func(i *InstantaneaObjetoMaterialV2) { i.versionEsquema++ },
		"conectorLogicoID":     func(i *InstantaneaObjetoMaterialV2) { i.conectorLogicoID += ":otro" },
		"objetoRef":            func(i *InstantaneaObjetoMaterialV2) { i.objetoRef += ":otro" },
		"objetoVersion":        func(i *InstantaneaObjetoMaterialV2) { i.objetoVersion += ":otra" },
		"zona":                 func(i *InstantaneaObjetoMaterialV2) { i.zona = ZonaAlmacenCuarentena },
		"mime":                 func(i *InstantaneaObjetoMaterialV2) { i.mime = "application/octet-stream" },
		"tamano":               func(i *InstantaneaObjetoMaterialV2) { i.tamano++ },
		"huellaContenido":      func(i *InstantaneaObjetoMaterialV2) { i.huellaContenido[0] ^= 1 },
		"evidenciaCreacionRef": func(i *InstantaneaObjetoMaterialV2) { i.evidenciaCreacionRef += ":otra" },
		"almacenadoEn":         func(i *InstantaneaObjetoMaterialV2) { i.almacenadoEn = i.almacenadoEn.Add(time.Microsecond) },
		"tieneRetencion": func(i *InstantaneaObjetoMaterialV2) {
			i.tieneRetencion = !i.tieneRetencion
			i.retenidoHasta = time.Time{}
		},
		"retenidoHasta":        func(i *InstantaneaObjetoMaterialV2) { i.retenidoHasta = i.retenidoHasta.Add(time.Microsecond) },
		"estadoInmovilizacion": func(i *InstantaneaObjetoMaterialV2) { i.estadoInmovilizacion = EstadoInmovilizacionMaterialNoAplicada },
		"estadoObjeto":         func(i *InstantaneaObjetoMaterialV2) { i.estadoObjeto = EstadoObjetoMaterialV2("archivado") },
	}
	comprobarCoberturaMutacionesMaterialV2(t, reflect.TypeOf(recibo.instantanea), mutacionesInstantanea)
	for nombre, mutar := range mutacionesInstantanea {
		t.Run("instantanea_"+nombre, func(t *testing.T) {
			alterado := recibo
			alterado.atestacion.codigo = append([]byte(nil), recibo.atestacion.codigo...)
			mutar(&alterado.instantanea)
			err := alterado.Validar()
			if nombre == "atestacion" {
				err = alterado.VerificarAtestacion(context.Background(), escenario.criptografia)
			}
			if !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
				t.Fatal("la mutacion de la instantanea no fallo cerrada")
			}
		})
	}

	mutacionesRecibo := map[string]func(*ReciboEscrituraObjetoMaterialV2){
		"esquema":                   func(r *ReciboEscrituraObjetoMaterialV2) { r.esquema += ".otro" },
		"versionEsquema":            func(r *ReciboEscrituraObjetoMaterialV2) { r.versionEsquema++ },
		"referenciaDurableOriginal": func(r *ReciboEscrituraObjetoMaterialV2) { r.referenciaDurableOriginal += ":otra" },
		"perfilReferencia":          func(r *ReciboEscrituraObjetoMaterialV2) { r.perfilReferencia += ":otra" },
		"perfilVersion":             func(r *ReciboEscrituraObjetoMaterialV2) { r.perfilVersion++ },
		"huellaPerfil":              func(r *ReciboEscrituraObjetoMaterialV2) { r.huellaPerfil[0] ^= 1 },
		"moduloID":                  func(r *ReciboEscrituraObjetoMaterialV2) { r.moduloID += ":otro" },
		"accionNegocio":             func(r *ReciboEscrituraObjetoMaterialV2) { r.accionNegocio += ":otra" },
		"accionTecnica":             func(r *ReciboEscrituraObjetoMaterialV2) { r.accionTecnica = AccionAlmacenPromover },
		"recursoRef":                func(r *ReciboEscrituraObjetoMaterialV2) { r.recursoRef += ":otro" },
		"operacionRef":              func(r *ReciboEscrituraObjetoMaterialV2) { r.operacionRef += ":otra" },
		"cargaRef":                  func(r *ReciboEscrituraObjetoMaterialV2) { r.cargaRef += ":otra" },
		"efectoRef":                 func(r *ReciboEscrituraObjetoMaterialV2) { r.efectoRef += ":otro" },
		"huellaPlanMaterial":        func(r *ReciboEscrituraObjetoMaterialV2) { r.huellaPlanMaterial.suma[0] ^= 1 },
		"clasificacion":             func(r *ReciboEscrituraObjetoMaterialV2) { r.clasificacion += ":otra" },
		"instantanea":               func(r *ReciboEscrituraObjetoMaterialV2) { r.instantanea.objetoVersion += ":otra" },
		"huella":                    func(r *ReciboEscrituraObjetoMaterialV2) { r.huella[0] ^= 1 },
		"atestacion":                func(r *ReciboEscrituraObjetoMaterialV2) { r.atestacion.codigo[0] ^= 1 },
	}
	comprobarCoberturaMutacionesMaterialV2(t, reflect.TypeOf(recibo), mutacionesRecibo)
	for nombre, mutar := range mutacionesRecibo {
		t.Run("recibo_"+nombre, func(t *testing.T) {
			alterado := recibo
			alterado.atestacion.codigo = append([]byte(nil), recibo.atestacion.codigo...)
			mutar(&alterado)
			err := alterado.Validar()
			if nombre == "atestacion" {
				err = alterado.VerificarAtestacion(context.Background(), escenario.criptografia)
			}
			if !errors.Is(err, ErrReciboEscrituraObjetoMaterialV2NoValido) {
				t.Fatal("la mutacion del recibo no fallo cerrada")
			}
		})
	}

	if (PerfilCapacidadesAlmacenMaterialV2{}).Validar() == nil ||
		(InstantaneaObjetoMaterialV2{}).Validar() == nil ||
		(ReciboEscrituraObjetoMaterialV2{}).Validar() == nil ||
		(EstadoInmovilizacionObjetoMaterialV2("")).valida() ||
		(EstadoObjetoMaterialV2("")).valida() {
		t.Fatal("un valor cero o estado abierto fue aceptado")
	}
}

func comprobarCoberturaMutacionesMaterialV2[T any](
	t *testing.T,
	tipo reflect.Type,
	mutaciones map[string]func(*T),
) {
	t.Helper()
	if tipo.NumField() != len(mutaciones) {
		t.Fatalf("cobertura incompleta: tipo=%s campos=%d mutaciones=%d", tipo, tipo.NumField(), len(mutaciones))
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		if _, existe := mutaciones[tipo.Field(indice).Name]; !existe {
			t.Fatalf("campo sin mutacion: %s.%s", tipo, tipo.Field(indice).Name)
		}
	}
}

func TestPerfilMaterialV2MutaAnteCadaCapacidadYNoIncluyeOrigenesFisicos(t *testing.T) {
	escenario := nuevoEscenarioReciboMaterialV2Prueba(t)
	original, _ := escenario.perfil.BytesCanonicos()
	mutaciones := []func(*CapacidadesAlmacenObjetos){
		func(c *CapacidadesAlmacenObjetos) { c.Retencion = !c.Retencion },
		func(c *CapacidadesAlmacenObjetos) { c.BloqueoLegal = !c.BloqueoLegal },
		func(c *CapacidadesAlmacenObjetos) { c.CifradoPorObjeto = !c.CifradoPorObjeto },
		func(c *CapacidadesAlmacenObjetos) { c.PreservaObjetoOriginal = !c.PreservaObjetoOriginal },
		func(c *CapacidadesAlmacenObjetos) { c.TamanoMaximoObjeto++ },
	}
	for indice, mutar := range mutaciones {
		capacidades := escenario.capacidades
		mutar(&capacidades)
		referencia := fmt.Sprintf("perfil:capacidades:material:v2:%03d", indice+20)
		escenario.registro.perfiles[referencia] = perfilPublicadoMaterialV2Prueba{
			version: 3, capacidades: capacidades,
		}
		perfil, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
			context.Background(), referencia, 3, capacidades,
			escenario.criptografia, escenario.criptografia, escenario.registro,
		)
		if err != nil {
			t.Fatalf("perfil alternativo %d: %v", indice, err)
		}
		canonico, _ := perfil.BytesCanonicos()
		if bytes.Equal(original, canonico) {
			t.Fatalf("la capacidad %d no altero la proyeccion", indice)
		}
	}
	capacidades := escenario.capacidades
	capacidades.OrigenesCargaDirecta = []string{
		"https://ruta-fisica-que-no-debe-aparecer.example.org",
	}
	escenario.registro.perfiles["perfil:capacidades:material:v2:origenes-omitidos"] = perfilPublicadoMaterialV2Prueba{
		version: 3, capacidades: capacidades,
	}
	perfil, err := NuevoPerfilCapacidadesAlmacenMaterialV2(
		context.Background(), "perfil:capacidades:material:v2:origenes-omitidos", 3,
		capacidades, escenario.criptografia, escenario.criptografia, escenario.registro,
	)
	if err != nil {
		t.Fatal(err)
	}
	canonico, _ := perfil.BytesCanonicos()
	if bytes.Contains(canonico, []byte("ruta-fisica")) {
		t.Fatal("el perfil material incorporo un origen fisico")
	}
}
