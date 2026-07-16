package ports

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type configuracionTestimonioPrueba struct {
	identidadesRef                string
	identidadesRevision           uint64
	identidades                   []topologiaClaveHMACBaremacion
	indicesRef                    string
	indicesRevision               uint64
	indices                       []topologiaClaveHMACBaremacion
	clavesIdentidades             [][]byte
	clavesIndices                 [][]byte
	claveAtestacion               []byte
	resolucionSnapshotRef         string
	resolucionRevision            uint64
	resolucionHuella              string
	resolucionFormatoAtestacion   string
	resolucionEmisorAtestacionRef string
	resolucionClaveAtestacionRef  string
	claveAtestacionResolucion     []byte
}

func nuevaConfiguracionTestimonioPrueba(cantidadIdentidades, cantidadIndices int) configuracionTestimonioPrueba {
	configuracion := configuracionTestimonioPrueba{
		identidadesRef:                "llavero-principales-ficticio",
		identidadesRevision:           41,
		indicesRef:                    "llavero-indices-ficticio",
		indicesRevision:               73,
		claveAtestacion:               []byte("clave-atestacion-ficticia-no-productiva-0001"),
		resolucionSnapshotRef:         "snapshot-identidad-ficticio-v1",
		resolucionRevision:            17,
		resolucionHuella:              huellaTextoPrueba("snapshot-identidad-ficticio-v1-revision-17"),
		resolucionFormatoAtestacion:   "hmac-sha256-v1",
		resolucionEmisorAtestacionRef: "servicio-identidad-ficticio",
		resolucionClaveAtestacionRef:  "atestacion-identidad-ficticia-v1",
		claveAtestacionResolucion:     []byte("clave-atestacion-identidad-ficticia-no-productiva"),
	}
	for posicion := 0; posicion < cantidadIdentidades; posicion++ {
		configuracion.identidades = append(configuracion.identidades, topologiaClaveHMACBaremacion{
			Version: VersionPrincipalEstableBaremacionV1, GeneracionClave: uint32(100 - posicion),
			ClaveHMACRef: fmt.Sprintf("principal-ficticio-%02d", posicion),
		})
		configuracion.clavesIdentidades = append(
			configuracion.clavesIdentidades,
			[]byte(fmt.Sprintf("clave-principal-ficticia-%02d-000000000000", posicion)),
		)
	}
	for posicion := 0; posicion < cantidadIndices; posicion++ {
		configuracion.indices = append(configuracion.indices, topologiaClaveHMACBaremacion{
			Version: VersionIndiceIdempotenciaBaremacionV1, GeneracionClave: uint32(200 - posicion),
			ClaveHMACRef: fmt.Sprintf("indice-ficticio-%02d", posicion),
		})
		configuracion.clavesIndices = append(
			configuracion.clavesIndices,
			[]byte(fmt.Sprintf("clave-indice-ficticia-%02d-000000000000000", posicion)),
		)
	}
	return configuracion
}

func huellaTextoPrueba(texto string) string {
	huella := sha256.Sum256([]byte(texto))
	return hex.EncodeToString(huella[:])
}

func hmacSHA256Prueba(clave, material []byte) string {
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(material)
	return hex.EncodeToString(mac.Sum(nil))
}

func cargasProtegidasIgualesPrueba(izquierda, derecha CargaProtegida) bool {
	materialIzquierdo := izquierda.Revelar()
	defer borrarBytesBaremacion(materialIzquierdo)
	materialDerecho := derecha.Revelar()
	defer borrarBytesBaremacion(materialDerecho)
	return bytes.Equal(materialIzquierdo, materialDerecho)
}

func copiarCargaDuranteVisitaPrueba(
	visitar func(func(MaterialCanonicoEfimeroBaremacion) error) error,
) ([]byte, error) {
	var copia []byte
	err := visitar(func(material MaterialCanonicoEfimeroBaremacion) error {
		return material.VisitarBytes(func(valor []byte) error {
			copia = append([]byte(nil), valor...)
			return nil
		})
	})
	return copia, err
}

func visitarMaterialEfimeroDirectoPrueba(
	valor []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	carga, err := NuevaCargaProtegida(valor)
	if err != nil {
		return err
	}
	return visitarCargaProtegidaEfimeraBaremacion(carga, visita)
}

func exigirAliasBorradoYMaterialRevocadoPrueba(
	t *testing.T,
	alias []byte,
	material MaterialCanonicoEfimeroBaremacion,
) {
	t.Helper()
	if !bytes.Equal(alias, make([]byte, len(alias))) {
		t.Fatalf("el alias retenido conservo %d bytes sensibles", len(alias))
	}
	if material.Validar() == nil {
		t.Fatal("una copia retenida del material siguio valida")
	}
	if err := material.VisitarBytes(func([]byte) error { return nil }); err == nil {
		t.Fatal("una copia retenida permitio una segunda visita")
	}
}

func TestMaterialCanonicoEfimeroRevocaCopiasYBorraAliasEnExitoYError(t *testing.T) {
	secreto := []byte("preimagen sensible ficticia para probar revocacion")
	t.Run("exito", func(t *testing.T) {
		var retenido MaterialCanonicoEfimeroBaremacion
		var alias []byte
		err := visitarMaterialEfimeroDirectoPrueba(
			secreto,
			func(material MaterialCanonicoEfimeroBaremacion) error {
				retenido = material
				if material.Validar() != nil {
					return errors.New("material no util al entrar")
				}
				return material.VisitarBytes(func(valor []byte) error {
					alias = valor
					if !bytes.Equal(valor, secreto) {
						return errors.New("contenido inesperado")
					}
					return nil
				})
			},
		)
		if err != nil {
			t.Fatalf("visita sincrona valida: %v", err)
		}
		exigirAliasBorradoYMaterialRevocadoPrueba(t, alias, retenido)
	})

	t.Run("error", func(t *testing.T) {
		errEsperado := errors.New("error ficticio del consumidor")
		var retenido MaterialCanonicoEfimeroBaremacion
		var alias []byte
		err := visitarMaterialEfimeroDirectoPrueba(
			secreto,
			func(material MaterialCanonicoEfimeroBaremacion) error {
				retenido = material
				return material.VisitarBytes(func(valor []byte) error {
					alias = valor
					return errEsperado
				})
			},
		)
		if !errors.Is(err, errEsperado) {
			t.Fatalf("se enmascaro el error del consumidor: %v", err)
		}
		exigirAliasBorradoYMaterialRevocadoPrueba(t, alias, retenido)
	})

	t.Run("error-exterior-tras-consumo", func(t *testing.T) {
		errEsperado := errors.New("error ficticio posterior al consumo")
		var retenido MaterialCanonicoEfimeroBaremacion
		var alias []byte
		err := visitarMaterialEfimeroDirectoPrueba(
			secreto,
			func(material MaterialCanonicoEfimeroBaremacion) error {
				retenido = material
				if err := material.VisitarBytes(func(valor []byte) error {
					alias = valor
					return nil
				}); err != nil {
					return err
				}
				return errEsperado
			},
		)
		if !errors.Is(err, errEsperado) {
			t.Fatalf("se enmascaro el error exterior: %v", err)
		}
		exigirAliasBorradoYMaterialRevocadoPrueba(t, alias, retenido)
	})
}

func TestCrecimientoCanonicoBorraBackingPropietarioAnterior(t *testing.T) {
	destino := make([]byte, 32, 32)
	for posicion := range destino {
		destino[posicion] = byte(0x80 + posicion)
	}
	aliasBackingAnterior := destino[:cap(destino)]
	valor := bytes.Repeat([]byte{0xa5}, 64)
	defer borrarBytesBaremacion(valor)
	resultado := anexarCampoCanonicoIntencion(destino, 77, valor)
	defer borrarBytesBaremacion(resultado[:cap(resultado)])
	if !bytes.Equal(aliasBackingAnterior, make([]byte, len(aliasBackingAnterior))) {
		t.Fatal("el crecimiento dejo la preimagen en el backing anterior")
	}
	if len(resultado) != 32+10+len(valor) || binary.BigEndian.Uint16(resultado[32:34]) != 77 ||
		!bytes.Equal(resultado[len(resultado)-len(valor):], valor) {
		t.Fatal("el crecimiento seguro altero el campo canonico")
	}

	destinoAlias := make([]byte, 24, 24)
	for posicion := range destinoAlias {
		destinoAlias[posicion] = byte(0xc0 + posicion)
	}
	aliasAnterior := destinoAlias[:cap(destinoAlias)]
	valorEsperado := append([]byte(nil), destinoAlias[:12]...)
	defer borrarBytesBaremacion(valorEsperado)
	resultadoAlias := anexarCampoCanonicoIntencion(destinoAlias, 88, destinoAlias[:12])
	defer borrarBytesBaremacion(resultadoAlias[:cap(resultadoAlias)])
	if !bytes.Equal(aliasAnterior, make([]byte, len(aliasAnterior))) {
		t.Fatal("el crecimiento con valor alias no borro el backing anterior")
	}
	if !bytes.Equal(resultadoAlias[len(resultadoAlias)-len(valorEsperado):], valorEsperado) {
		t.Fatal("el crecimiento borro el valor alias antes de copiarlo")
	}

	destinoTexto := bytes.Repeat([]byte{0xd1}, 20)
	destinoTexto = destinoTexto[:len(destinoTexto):len(destinoTexto)]
	aliasTextoAnterior := destinoTexto[:cap(destinoTexto)]
	resultadoTexto := anexarCampoCanonicoTextoIntencion(
		destinoTexto, 99, "referencia-sensible-ficticia",
	)
	defer borrarBytesBaremacion(resultadoTexto[:cap(resultadoTexto)])
	if !bytes.Equal(aliasTextoAnterior, make([]byte, len(aliasTextoAnterior))) {
		t.Fatal("el crecimiento de texto no borro el backing anterior")
	}
	if !bytes.HasSuffix(resultadoTexto, []byte("referencia-sensible-ficticia")) {
		t.Fatal("el crecimiento de texto altero el valor")
	}

	destinoCrudo := make([]byte, 18, 18)
	for posicion := range destinoCrudo {
		destinoCrudo[posicion] = byte(0xe0 + posicion)
	}
	aliasCrudoAnterior := destinoCrudo[:cap(destinoCrudo)]
	crudoEsperado := append([]byte(nil), destinoCrudo[:9]...)
	defer borrarBytesBaremacion(crudoEsperado)
	resultadoCrudo := anexarBytesAPropietarioCanonicoIntencion(destinoCrudo, destinoCrudo[:9])
	defer borrarBytesBaremacion(resultadoCrudo[:cap(resultadoCrudo)])
	if !bytes.Equal(aliasCrudoAnterior, make([]byte, len(aliasCrudoAnterior))) {
		t.Fatal("el append crudo no borro el backing anterior")
	}
	if !bytes.Equal(resultadoCrudo[len(resultadoCrudo)-len(crudoEsperado):], crudoEsperado) {
		t.Fatal("el append crudo borro el valor alias antes de copiarlo")
	}
}

func TestMaterialCanonicoEfimeroBorraYRevocaAntePanico(t *testing.T) {
	var retenido MaterialCanonicoEfimeroBaremacion
	var alias []byte
	panico := ejecutarYRecuperarPanicoPrueba(func() {
		_ = visitarMaterialEfimeroDirectoPrueba(
			[]byte("preimagen sensible ficticia para panico"),
			func(material MaterialCanonicoEfimeroBaremacion) error {
				retenido = material
				return material.VisitarBytes(func(valor []byte) error {
					alias = valor
					panic("panico ficticio dentro de VisitarBytes")
				})
			},
		)
	})
	if panico == nil {
		t.Fatal("el panico del consumidor no se propago")
	}
	exigirAliasBorradoYMaterialRevocadoPrueba(t, alias, retenido)
}

func TestMaterialCanonicoEfimeroFallaRapidoConVisitaAsincrona(t *testing.T) {
	iniciada := make(chan struct{})
	liberar := make(chan struct{})
	terminada := make(chan error, 1)
	var retenido MaterialCanonicoEfimeroBaremacion
	var alias []byte
	inicio := time.Now()
	err := visitarMaterialEfimeroDirectoPrueba(
		[]byte("preimagen sensible ficticia para callback asincrono"),
		func(material MaterialCanonicoEfimeroBaremacion) error {
			retenido = material
			go func() {
				terminada <- material.VisitarBytes(func(valor []byte) error {
					alias = valor
					close(iniciada)
					<-liberar
					return nil
				})
			}()
			<-iniciada
			return nil
		},
	)
	if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("actividad asincrona no fallo cerrado: %v", err)
	}
	if time.Since(inicio) > 250*time.Millisecond {
		t.Fatal("el cierre espero al callback asincrono")
	}
	if retenido.Validar() == nil {
		t.Fatal("el material siguio valido durante una visita asincrona revocada")
	}
	close(liberar)
	select {
	case err := <-terminada:
		if err != nil {
			t.Fatalf("la visita iniciada no pudo terminar limpiamente: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("la visita asincrona no termino al liberarla")
	}
	exigirAliasBorradoYMaterialRevocadoPrueba(t, alias, retenido)

	invocada := false
	if err := retenido.VisitarBytes(func([]byte) error {
		invocada = true
		return nil
	}); err == nil || invocada {
		t.Fatal("una visita iniciada tras la revocacion recibio bytes")
	}
}

func TestSolicitudMotivoComparteOwnerRevocableSinRetenerCarga(t *testing.T) {
	motivo := []byte("motivo exacto ficticio con informacion sensible")
	var retenida, copia SolicitudDerivarHMACMotivoBaremacion
	var alias []byte
	err := VisitarSolicitudDerivarHMACMotivoBaremacion(
		"merito_acreditado", motivo,
		func(solicitud SolicitudDerivarHMACMotivoBaremacion) error {
			if solicitud.Validar() != nil {
				return errors.New("solicitud de motivo no valida al entrar")
			}
			clave, err := solicitud.MotivoClave()
			if err != nil || clave != "merito_acreditado" {
				return errors.New("clave de motivo inesperada")
			}
			retenida, copia = solicitud, solicitud
			return copia.VisitarMotivo(func(valor []byte) error {
				alias = valor
				if !bytes.Equal(valor, motivo) {
					return errors.New("motivo inesperado")
				}
				return nil
			})
		},
	)
	if err != nil {
		t.Fatalf("visitar solicitud de motivo: %v", err)
	}
	if !bytes.Equal(alias, make([]byte, len(alias))) {
		t.Fatal("el alias del motivo no fue borrado")
	}
	for nombre, solicitud := range map[string]SolicitudDerivarHMACMotivoBaremacion{
		"retenida": retenida, "copia": copia,
	} {
		if solicitud.Validar() == nil || solicitud.VisitarMotivo(func([]byte) error { return nil }) == nil {
			t.Fatalf("la solicitud %s siguio util tras el callback", nombre)
		}
	}
	if _, existe := reflect.TypeOf(SolicitudDerivarHMACMotivoBaremacion{}).FieldByName("Motivo"); existe {
		t.Fatal("la solicitud de motivo retiene una CargaProtegida publica")
	}
}

func identidadInternaEstablePrueba(desplazamiento byte) []byte {
	identidad := make([]byte, longitudIdentidadInternaEstableBaremacion)
	for posicion := range identidad {
		identidad[posicion] = 0x80 + byte((posicion+int(desplazamiento))%64)
	}
	return identidad
}

func claveSeudonimoPrueba(referencia string) []byte {
	huella := sha256.Sum256([]byte("clave-seudonimo-ficticia|" + referencia))
	return huella[:]
}

type consumidorIdentidadInternaPrueba struct {
	material []byte
	llamadas int
}

func (c *consumidorIdentidadInternaPrueba) ConsumirIdentidadInternaEstableIdempotenciaBaremacion(
	ctx context.Context,
	material []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.llamadas++
	c.material = append([]byte(nil), material...)
	return nil
}

type fronteraIdentidadInternaPrueba struct {
	mu                  sync.Mutex
	configuracion       configuracionTestimonioPrueba
	identidad           []byte
	ambitoEsperado      SolicitudResolverSeudonimoSujetoBaremacion
	seudonimoEsperado   SeudonimoSujetoBaremacionHMAC
	llamadas            int
	retuvoReceptor      ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion
	identidadSegunda    []byte
	forzarCruce         bool
	alterarAtestacion   bool
	alterarHuella       bool
	cancelarAlFinal     context.CancelFunc
	errorAlFinal        error
	panicoTrasRegistrar bool
}

type fronteraIdentidadBloqueantePrueba struct {
	delegado *fronteraIdentidadInternaPrueba
	inicio   chan struct{}
	liberar  chan struct{}
	unaVez   sync.Once
	llamadas int
}

func (f *fronteraIdentidadBloqueantePrueba) ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
	ctx context.Context,
	ambito SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	receptor ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion,
) error {
	f.llamadas++
	f.unaVez.Do(func() { close(f.inicio) })
	select {
	case <-f.liberar:
		return f.delegado.ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
			ctx, ambito, seudonimo, receptor,
		)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func nuevaFronteraIdentidadInternaPrueba(
	configuracion configuracionTestimonioPrueba,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidad []byte,
) *fronteraIdentidadInternaPrueba {
	return &fronteraIdentidadInternaPrueba{
		configuracion:     configuracion,
		identidad:         append([]byte(nil), identidad...),
		ambitoEsperado:    solicitud.ambitoSujeto,
		seudonimoEsperado: solicitud.seudonimo,
	}
}

func (f *fronteraIdentidadInternaPrueba) ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
	ctx context.Context,
	ambito SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	receptor ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.llamadas++
	f.retuvoReceptor = receptor
	if f.forzarCruce || ambito != f.ambitoEsperado || seudonimo != f.seudonimoEsperado {
		return errors.New("frontera ficticia: cruce ambito/seudonimo")
	}
	identidad := f.identidad
	if f.llamadas > 1 && len(f.identidadSegunda) != 0 {
		identidad = f.identidadSegunda
	}
	huellaResolucion, err := CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
		ambito, seudonimo, f.configuracion.resolucionSnapshotRef,
		f.configuracion.resolucionRevision, identidad,
	)
	if err != nil {
		return err
	}
	if f.alterarHuella {
		huellaResolucion = huellaTextoPrueba("huella-resolucion-alterada")
	}
	preimagen, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
		return VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion(
			ambito, seudonimo, f.configuracion.resolucionSnapshotRef,
			f.configuracion.resolucionRevision, huellaResolucion, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarBytesBaremacion(preimagen)
	mac := hmac.New(sha256.New, f.configuracion.claveAtestacionResolucion)
	_, _ = mac.Write(preimagen)
	atestacion := mac.Sum(nil)
	if f.alterarAtestacion {
		atestacion[0] ^= 0xff
	}
	if err := receptor.RegistrarResolucionIdentidadInternaEstableBaremacion(
		ctx, identidad, f.configuracion.resolucionSnapshotRef,
		f.configuracion.resolucionRevision, huellaResolucion,
		f.configuracion.resolucionFormatoAtestacion,
		f.configuracion.resolucionEmisorAtestacionRef,
		f.configuracion.resolucionClaveAtestacionRef, atestacion,
	); err != nil {
		return err
	}
	if f.cancelarAlFinal != nil {
		f.cancelarAlFinal()
	}
	if f.panicoTrasRegistrar {
		panic("panico ficticio de frontera tras registrar identidad")
	}
	return f.errorAlFinal
}

func huellaInstantaneaPrueba(
	dominio DominioClaveHMACBaremacion,
	referencia string,
	revision uint64,
	topologia []topologiaClaveHMACBaremacion,
) string {
	instantanea := instantaneaLlaveroHMACBaremacion{
		Dominio: dominio, LlaveroRef: referencia, Revision: revision,
		Cantidad: uint8(len(topologia)), Topologia: append([]topologiaClaveHMACBaremacion(nil), topologia...),
	}
	material := instantanea.representacionCanonicaSinHuella()
	defer borrarBytesBaremacion(material)
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:])
}

func posicionesCompletasPrueba(cantidad int) []int {
	resultado := make([]int, cantidad)
	for posicion := range resultado {
		resultado[posicion] = posicion
	}
	return resultado
}

func posicionesSinPrueba(cantidad, omitida int) []int {
	resultado := make([]int, 0, cantidad-1)
	for posicion := 0; posicion < cantidad; posicion++ {
		if posicion != omitida {
			resultado = append(resultado, posicion)
		}
	}
	return resultado
}

func referenciaGeneracionSeparacionPrueba(
	t *testing.T,
	dominio DominioClaveHMACBaremacion,
	generacion uint32,
	referencia string,
) ReferenciaGeneracionClaveHMACNominalBaremacion {
	t.Helper()
	resultado, err := NuevaReferenciaGeneracionClaveHMACNominalBaremacion(
		dominio, generacion, referencia,
	)
	if err != nil {
		t.Fatalf("crear referencia de separacion ficticia: %v", err)
	}
	return resultado
}

func referenciasSeisDominiosSeparacionPrueba(t *testing.T) []ReferenciaGeneracionClaveHMACNominalBaremacion {
	t.Helper()
	return []ReferenciaGeneracionClaveHMACNominalBaremacion{
		referenciaGeneracionSeparacionPrueba(t, DominioClavePrincipalBaremacion, 1, "principal-ficticio-v1"),
		referenciaGeneracionSeparacionPrueba(t, DominioClaveIndiceBaremacion, 1, "indice-ficticio-v1"),
		referenciaGeneracionSeparacionPrueba(t, DominioClaveSujetoBaremacion, 1, "sujeto-ficticio-v1"),
		referenciaGeneracionSeparacionPrueba(t, DominioClaveMotivoBaremacion, 1, "motivo-ficticio-v1"),
		referenciaGeneracionSeparacionPrueba(t, DominioClaveManifiestoBaremacion, 1, "manifiesto-ficticio-v1"),
		referenciaGeneracionSeparacionPrueba(t, DominioClaveIntencionBaremacion, 1, "intencion-ficticio-v1"),
	}
}

func seleccionarTopologiaPrueba(
	topologia []topologiaClaveHMACBaremacion,
	posiciones []int,
) []topologiaClaveHMACBaremacion {
	resultado := make([]topologiaClaveHMACBaremacion, 0, len(posiciones))
	for _, posicion := range posiciones {
		resultado = append(resultado, topologia[posicion])
	}
	return resultado
}

type consumidorClaveClientePrueba struct {
	material []byte
	llamadas int
}

func (c *consumidorClaveClientePrueba) ConsumirClaveClienteLoteIdempotenciaBaremacion(
	ctx context.Context,
	material []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.llamadas++
	c.material = append([]byte(nil), material...)
	return nil
}

type productorTestimonioPrueba struct {
	mu                            sync.Mutex
	configuracion                 configuracionTestimonioPrueba
	filas                         []int
	columnas                      []int
	llamadas                      int
	retuvoFuenteIdentidad         FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion
	retuvoFuente                  FuenteEfimeraClaveClienteIdempotenciaBaremacion
	retuvoReceptor                ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion
	retuvoSolicitud               SolicitudTestimonioAtomicoIdempotenciaBaremacion
	cancelarAlFinal               context.CancelFunc
	omitirEvidencia               bool
	omitirUltimaCelda             bool
	mutarEvidencia                bool
	colisionTransversal           bool
	colisionConPrincipalPosterior bool
	principalConClaveCliente      bool
	indiceSinPrincipal            bool
	indiceEsquemaAlterado         bool
	indicePoliticaAlterada        bool
	panicoAlInicio                bool
	errorAlFinal                  error
}

func (p *productorTestimonioPrueba) ProducirTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fuenteIdentidad FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fuente FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	receptor ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.llamadas++
	p.retuvoSolicitud = solicitud
	p.retuvoFuenteIdentidad = fuenteIdentidad
	p.retuvoFuente = fuente
	p.retuvoReceptor = receptor
	if p.panicoAlInicio {
		panic("panico ficticio del productor")
	}
	consumidor := &consumidorClaveClientePrueba{}
	if err := fuente.EntregarClaveClienteLoteIdempotenciaBaremacion(ctx, consumidor); err != nil {
		return err
	}
	defer borrarBytesBaremacion(consumidor.material)
	consumidorIdentidad := &consumidorIdentidadInternaPrueba{}
	if err := fuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		ctx, consumidorIdentidad,
	); err != nil {
		return err
	}
	defer borrarBytesBaremacion(consumidorIdentidad.material)
	preimagenSeudonimo, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
		return VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
			solicitud, consumidorIdentidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarBytesBaremacion(preimagenSeudonimo)
	seudonimoEsperado := hmacSHA256Prueba(
		claveSeudonimoPrueba(solicitud.seudonimo.ClaveHMACRef), preimagenSeudonimo,
	)
	if !hmac.Equal([]byte(seudonimoEsperado), []byte(solicitud.seudonimo.ValorHMAC)) {
		return errors.New("productor: seudonimo no corresponde al ancla")
	}
	preimagenPrincipal, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
		return VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion(
			solicitud, consumidorIdentidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarBytesBaremacion(preimagenPrincipal)
	filas := append([]int(nil), p.filas...)
	if len(filas) == 0 {
		filas = posicionesCompletasPrueba(len(p.configuracion.identidades))
	}
	columnas := append([]int(nil), p.columnas...)
	if len(columnas) == 0 {
		columnas = posicionesCompletasPrueba(len(p.configuracion.indices))
	}
	topologiaIdentidades := seleccionarTopologiaPrueba(p.configuracion.identidades, filas)
	topologiaIndices := seleccionarTopologiaPrueba(p.configuracion.indices, columnas)
	huellaIdentidades := huellaInstantaneaPrueba(
		DominioClavePrincipalBaremacion, p.configuracion.identidadesRef,
		p.configuracion.identidadesRevision, topologiaIdentidades,
	)
	if err := receptor.InmovilizarLlaveroIdentidadesBaremacion(
		p.configuracion.identidadesRef, p.configuracion.identidadesRevision,
		uint8(len(topologiaIdentidades)), huellaIdentidades,
	); err != nil {
		return err
	}
	principalesGenerados := make([]principalEstableBaremacionHMAC, len(filas))
	for fila, posicionOriginal := range filas {
		entrada := p.configuracion.identidades[posicionOriginal]
		preimagen := preimagenPrincipal
		if p.principalConClaveCliente {
			preimagen = append(append([]byte(nil), preimagenPrincipal...), consumidor.material...)
		}
		valor := hmacSHA256Prueba(p.configuracion.clavesIdentidades[posicionOriginal], preimagen)
		if p.principalConClaveCliente {
			borrarBytesBaremacion(preimagen)
		}
		principalesGenerados[fila] = principalEstableBaremacionHMAC{
			Version: entrada.Version, GeneracionClave: entrada.GeneracionClave,
			ClaveHMACRef: entrada.ClaveHMACRef, ValorHMAC: valor,
		}
		if err := receptor.RegistrarPrincipalEstableBaremacion(
			fila, entrada.Version, entrada.GeneracionClave, entrada.ClaveHMACRef, valor,
		); err != nil {
			return err
		}
	}
	huellaIndices := huellaInstantaneaPrueba(
		DominioClaveIndiceBaremacion, p.configuracion.indicesRef,
		p.configuracion.indicesRevision, topologiaIndices,
	)
	if err := receptor.InmovilizarLlaveroIndicesBaremacion(
		p.configuracion.indicesRef, p.configuracion.indicesRevision,
		uint8(len(topologiaIndices)), huellaIndices,
	); err != nil {
		return err
	}
	primerIndice := ""
	for fila := range filas {
		principal := principalesGenerados[fila]
		for columna, columnaOriginal := range columnas {
			if p.omitirUltimaCelda && fila == len(filas)-1 && columna == len(columnas)-1 {
				continue
			}
			entrada := p.configuracion.indices[columnaOriginal]
			preimagenIndice, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
				return VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion(
					solicitud, principal.Version, principal.GeneracionClave,
					principal.ClaveHMACRef, principal.ValorHMAC, consumidor.material, visita,
				)
			})
			if err != nil {
				return err
			}
			if p.indiceSinPrincipal {
				preimagenIndice = append([]byte("indice-sin-principal|"), consumidor.material...)
			}
			if p.indiceEsquemaAlterado {
				preimagenIndice = bytes.ReplaceAll(
					preimagenIndice, []byte(EsquemaCanonicoIndiceIdempotenciaBaremacionV1),
					[]byte("vec.bolsa.indice-idempotencia.alterado.v1"),
				)
			}
			if p.indicePoliticaAlterada {
				preimagenIndice = bytes.ReplaceAll(
					preimagenIndice, []byte(PoliticaDerivacionIdempotenciaBaremacionDEC045V1),
					[]byte("vec.bolsa.politica-derivacion.alterada.v1"),
				)
			}
			valor := hmacSHA256Prueba(p.configuracion.clavesIndices[columnaOriginal], preimagenIndice)
			borrarBytesBaremacion(preimagenIndice)
			if p.colisionTransversal && fila == len(filas)-1 && columna == len(columnas)-1 {
				valor = primerIndice
			}
			if p.colisionConPrincipalPosterior && fila == 0 && columna == 0 && len(principalesGenerados) > 1 {
				valor = principalesGenerados[1].ValorHMAC
			}
			if fila == 0 && columna == 0 {
				primerIndice = valor
			}
			if err := receptor.RegistrarIndiceIdempotenciaBaremacion(
				fila, columna, entrada.Version, entrada.GeneracionClave, entrada.ClaveHMACRef, valor,
			); err != nil {
				return err
			}
		}
	}
	if !p.omitirEvidencia {
		material, err := copiarCargaDuranteVisitaPrueba(
			receptor.VisitarMaterialCanonicoParaAtestacionBaremacion,
		)
		if err != nil {
			return err
		}
		defer borrarBytesBaremacion(material)
		huellaContenido := sha256.Sum256(material)
		mac := hmac.New(sha256.New, p.configuracion.claveAtestacion)
		_, _ = mac.Write(huellaContenido[:])
		evidencia := mac.Sum(nil)
		if err := receptor.RegistrarEvidenciaAtestacionBaremacion(
			"hmac-sha256-v1", "hsm-ficticio", "atestacion-ficticia-v1", 9,
			hex.EncodeToString(huellaContenido[:]), evidencia,
		); err != nil {
			return err
		}
		if p.mutarEvidencia {
			for posicion := range evidencia {
				evidencia[posicion] ^= 0xff
			}
		}
	}
	if p.cancelarAlFinal != nil {
		p.cancelarAlFinal()
	}
	return p.errorAlFinal
}

type verificadorTestimonioPrueba struct {
	mu                    sync.Mutex
	configuracion         configuracionTestimonioPrueba
	llamadas              int
	retuvoVista           VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
	retuvoFuenteIdentidad FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion
	retuvoFuenteClave     FuenteEfimeraClaveClienteIdempotenciaBaremacion
	cancelarAlFinal       context.CancelFunc
	omitirResolucion      bool
	panicoAlInicio        bool
	errorAlFinal          error
}

func (v *verificadorTestimonioPrueba) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fuenteIdentidad FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fuente FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	vista VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.llamadas++
	v.retuvoVista = vista
	v.retuvoFuenteIdentidad = fuenteIdentidad
	v.retuvoFuenteClave = fuente
	if v.panicoAlInicio {
		panic("panico ficticio del verificador")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	consumidorClave := &consumidorClaveClientePrueba{}
	if err := fuente.EntregarClaveClienteLoteIdempotenciaBaremacion(ctx, consumidorClave); err != nil {
		return err
	}
	defer borrarBytesBaremacion(consumidorClave.material)
	consumidorIdentidad := &consumidorIdentidadInternaPrueba{}
	if err := fuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		ctx, consumidorIdentidad,
	); err != nil {
		return err
	}
	defer borrarBytesBaremacion(consumidorIdentidad.material)
	preimagenSeudonimo, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
		return VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
			solicitud, consumidorIdentidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarBytesBaremacion(preimagenSeudonimo)
	seudonimoEsperado := hmacSHA256Prueba(
		claveSeudonimoPrueba(solicitud.seudonimo.ClaveHMACRef), preimagenSeudonimo,
	)
	if !hmac.Equal([]byte(seudonimoEsperado), []byte(solicitud.seudonimo.ValorHMAC)) {
		return errors.New("testimonio: seudonimo no corresponde al ancla")
	}
	if !v.omitirResolucion {
		if err := vista.VisitarResolucionIdentidadInternaEstableBaremacion(
			func(snapshotRef string, revision uint64, huella, formato, emisor, clave string, atestacion []byte) error {
				if snapshotRef != v.configuracion.resolucionSnapshotRef ||
					revision != v.configuracion.resolucionRevision ||
					formato != v.configuracion.resolucionFormatoAtestacion ||
					emisor != v.configuracion.resolucionEmisorAtestacionRef ||
					clave != v.configuracion.resolucionClaveAtestacionRef {
					return errors.New("testimonio: metadatos de resolucion de identidad inesperados")
				}
				huellaEsperada, err := CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
					solicitud.ambitoSujeto, solicitud.seudonimo, snapshotRef, revision,
					consumidorIdentidad.material,
				)
				if err != nil || !hmac.Equal([]byte(huellaEsperada), []byte(huella)) {
					return errors.New("testimonio: huella de resolucion de identidad invalida")
				}
				preimagen, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
					return VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion(
						solicitud.ambitoSujeto, solicitud.seudonimo, snapshotRef, revision, huella, visita,
					)
				})
				if err != nil {
					return err
				}
				defer borrarBytesBaremacion(preimagen)
				mac := hmac.New(sha256.New, v.configuracion.claveAtestacionResolucion)
				_, _ = mac.Write(preimagen)
				if !hmac.Equal(mac.Sum(nil), atestacion) {
					return errors.New("testimonio: atestacion de resolucion de identidad invalida")
				}
				return nil
			},
		); err != nil {
			return err
		}
	}
	preimagenPrincipal, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
		return VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion(
			solicitud, consumidorIdentidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarBytesBaremacion(preimagenPrincipal)
	ref, revision, cantidad, huella, err := vista.ResumenLlaveroIdentidadesBaremacion()
	if err != nil || ref != v.configuracion.identidadesRef || revision != v.configuracion.identidadesRevision ||
		int(cantidad) != len(v.configuracion.identidades) ||
		huella != huellaInstantaneaPrueba(DominioClavePrincipalBaremacion, ref, revision, v.configuracion.identidades) {
		return errors.New("testimonio: topologia de identidades inesperada")
	}
	visitadas := 0
	if err := vista.VisitarTopologiaIdentidadesBaremacion(
		func(posicion int, version uint16, generacion uint32, clave string) error {
			if posicion >= len(v.configuracion.identidades) {
				return errors.New("testimonio: identidad adicional")
			}
			esperada := v.configuracion.identidades[posicion]
			if version != esperada.Version || generacion != esperada.GeneracionClave || clave != esperada.ClaveHMACRef {
				return errors.New("testimonio: identidad distinta")
			}
			visitadas++
			return nil
		},
	); err != nil || visitadas != len(v.configuracion.identidades) {
		return errors.New("testimonio: identidades incompletas")
	}
	ref, revision, cantidad, huella, err = vista.ResumenLlaveroIndicesBaremacion()
	if err != nil || ref != v.configuracion.indicesRef || revision != v.configuracion.indicesRevision ||
		int(cantidad) != len(v.configuracion.indices) ||
		huella != huellaInstantaneaPrueba(DominioClaveIndiceBaremacion, ref, revision, v.configuracion.indices) {
		return errors.New("testimonio: topologia de indices inesperada")
	}
	visitadas = 0
	if err := vista.VisitarTopologiaIndicesBaremacion(
		func(posicion int, version uint16, generacion uint32, clave string) error {
			if posicion >= len(v.configuracion.indices) {
				return errors.New("testimonio: indice adicional")
			}
			esperada := v.configuracion.indices[posicion]
			if version != esperada.Version || generacion != esperada.GeneracionClave || clave != esperada.ClaveHMACRef {
				return errors.New("testimonio: indice distinto")
			}
			visitadas++
			return nil
		},
	); err != nil || visitadas != len(v.configuracion.indices) {
		return errors.New("testimonio: indices incompletos")
	}
	principales := 0
	principalesRecibidos := make([]principalEstableBaremacionHMAC, len(v.configuracion.identidades))
	if err := vista.VisitarPrincipalesBaremacion(
		func(posicion int, version uint16, generacion uint32, clave, valor string) error {
			if posicion >= len(v.configuracion.identidades) || !huellaSHA256Valida(valor) {
				return errors.New("testimonio: principal invalido")
			}
			esperada := v.configuracion.identidades[posicion]
			if version != esperada.Version || generacion != esperada.GeneracionClave || clave != esperada.ClaveHMACRef {
				return errors.New("testimonio: principal fuera de topologia")
			}
			valorEsperado := hmacSHA256Prueba(v.configuracion.clavesIdentidades[posicion], preimagenPrincipal)
			if !hmac.Equal([]byte(valorEsperado), []byte(valor)) {
				return errors.New("testimonio: formula de principal estable incumplida")
			}
			principalesRecibidos[posicion] = principalEstableBaremacionHMAC{
				Version: version, GeneracionClave: generacion, ClaveHMACRef: clave, ValorHMAC: valor,
			}
			principales++
			return nil
		},
	); err != nil || principales != len(v.configuracion.identidades) {
		return errors.New("testimonio: principales incompletos")
	}
	celdas := 0
	if err := vista.VisitarMatrizIndicesBaremacion(
		func(fila, columna int, version uint16, generacion uint32, clave, valor string) error {
			if fila >= len(v.configuracion.identidades) || columna >= len(v.configuracion.indices) ||
				!huellaSHA256Valida(valor) {
				return errors.New("testimonio: celda fuera de matriz")
			}
			esperada := v.configuracion.indices[columna]
			if version != esperada.Version || generacion != esperada.GeneracionClave || clave != esperada.ClaveHMACRef {
				return errors.New("testimonio: columna fuera de topologia")
			}
			principal := principalesRecibidos[fila]
			preimagenIndice, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
				return VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion(
					solicitud, principal.Version, principal.GeneracionClave,
					principal.ClaveHMACRef, principal.ValorHMAC, consumidorClave.material, visita,
				)
			})
			if err != nil {
				return err
			}
			defer borrarBytesBaremacion(preimagenIndice)
			valorEsperado := hmacSHA256Prueba(v.configuracion.clavesIndices[columna], preimagenIndice)
			if !hmac.Equal([]byte(valorEsperado), []byte(valor)) {
				return errors.New("testimonio: formula de indice DEC-045 incumplida")
			}
			celdas++
			return nil
		},
	); err != nil || celdas != len(v.configuracion.identidades)*len(v.configuracion.indices) {
		return errors.New("testimonio: matriz incompleta")
	}
	if err := vista.VisitarMaterialCanonicoAtestadoBaremacion(
		func(material MaterialCanonicoEfimeroBaremacion) error {
			return material.VisitarBytes(func([]byte) error { return nil })
		},
	); err != nil {
		return err
	}
	if err := vista.VisitarEvidenciaAtestacionBaremacion(
		func(formato, emisor, clave string, revision uint64, huella string, evidencia []byte) error {
			if formato != "hmac-sha256-v1" || emisor != "hsm-ficticio" ||
				clave != "atestacion-ficticia-v1" || revision != 9 {
				return errors.New("testimonio: metadatos de atestacion inesperados")
			}
			contenido, err := hex.DecodeString(huella)
			if err != nil {
				return err
			}
			mac := hmac.New(sha256.New, v.configuracion.claveAtestacion)
			_, _ = mac.Write(contenido)
			if !hmac.Equal(mac.Sum(nil), evidencia) {
				return errors.New("testimonio: atestacion no valida")
			}
			return nil
		},
	); err != nil {
		return err
	}
	if v.cancelarAlFinal != nil {
		v.cancelarAlFinal()
	}
	return v.errorAlFinal
}

func recorrerVistaCompletaNominalPrueba(
	vista VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	if err := vista.VisitarMaterialCanonicoAtestadoBaremacion(
		func(material MaterialCanonicoEfimeroBaremacion) error {
			return material.VisitarBytes(func([]byte) error { return nil })
		},
	); err != nil {
		return err
	}
	if err := vista.VisitarResolucionIdentidadInternaEstableBaremacion(
		func(string, uint64, string, string, string, string, []byte) error { return nil },
	); err != nil {
		return err
	}
	if _, _, _, _, err := vista.ResumenLlaveroIdentidadesBaremacion(); err != nil {
		return err
	}
	if err := vista.VisitarTopologiaIdentidadesBaremacion(
		func(int, uint16, uint32, string) error { return nil },
	); err != nil {
		return err
	}
	if _, _, _, _, err := vista.ResumenLlaveroIndicesBaremacion(); err != nil {
		return err
	}
	if err := vista.VisitarTopologiaIndicesBaremacion(
		func(int, uint16, uint32, string) error { return nil },
	); err != nil {
		return err
	}
	if err := vista.VisitarPrincipalesBaremacion(
		func(int, uint16, uint32, string, string) error { return nil },
	); err != nil {
		return err
	}
	if err := vista.VisitarMatrizIndicesBaremacion(
		func(int, int, uint16, uint32, string, string) error { return nil },
	); err != nil {
		return err
	}
	return vista.VisitarEvidenciaAtestacionBaremacion(
		func(string, string, string, uint64, string, []byte) error { return nil },
	)
}

type consumidorNominalNoAutoritativoPrueba struct {
	llamadas            int
	retuvoVista         VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
	testimonioCapturado testimonioAtomicoIdempotenciaBaremacion
	panico              bool
	cancelarAlFinal     context.CancelFunc
	errorAlFinal        error
}

type raizVerificadorTestimonioPrueba struct {
	delegado *verificadorTestimonioPrueba
}

func (r *raizVerificadorTestimonioPrueba) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidad FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	clave FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	vista VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	return r.delegado.VerificarTestimonioAtomicoIdempotenciaBaremacion(
		ctx, solicitud, identidad, clave, vista,
	)
}

type productoNominalPrueba struct {
	testimonio testimonioAtomicoIdempotenciaBaremacion
}

func (p productoNominalPrueba) validar() error { return p.testimonio.validarEstructura() }

type consumidorClaveBloqueantePrueba struct {
	inicio  chan struct{}
	liberar chan struct{}
	unaVez  sync.Once
}

func (c *consumidorClaveBloqueantePrueba) ConsumirClaveClienteLoteIdempotenciaBaremacion(
	ctx context.Context,
	_ []byte,
) error {
	c.unaVez.Do(func() { close(c.inicio) })
	select {
	case <-c.liberar:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type productorFuenteAsincronaPrueba struct {
	inicio         chan struct{}
	liberar        chan struct{}
	hecho          chan struct{}
	retuvoFuente   FuenteEfimeraClaveClienteIdempotenciaBaremacion
	retuvoReceptor ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion
}

func (p *productorFuenteAsincronaPrueba) ProducirTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	_ SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fuenteIdentidad FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fuenteClave FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	receptor ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion,
) error {
	p.retuvoFuente = fuenteClave
	p.retuvoReceptor = receptor
	if err := fuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		ctx, &consumidorIdentidadInternaPrueba{},
	); err != nil {
		return err
	}
	consumidor := &consumidorClaveBloqueantePrueba{inicio: p.inicio, liberar: p.liberar}
	go func() {
		defer close(p.hecho)
		_ = fuenteClave.EntregarClaveClienteLoteIdempotenciaBaremacion(ctx, consumidor)
	}()
	<-p.inicio
	return nil
}

type verificadorVistaAsincronaPrueba struct {
	inicio      chan struct{}
	liberar     chan struct{}
	hecho       chan struct{}
	unaVez      sync.Once
	retuvoVista VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
}

func (v *verificadorVistaAsincronaPrueba) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	_ SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fuenteIdentidad FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fuenteClave FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	vista VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	v.retuvoVista = vista
	if err := fuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		ctx, &consumidorIdentidadInternaPrueba{},
	); err != nil {
		return err
	}
	if err := fuenteClave.EntregarClaveClienteLoteIdempotenciaBaremacion(
		ctx, &consumidorClaveClientePrueba{},
	); err != nil {
		return err
	}
	if err := vista.VisitarMaterialCanonicoAtestadoBaremacion(
		func(material MaterialCanonicoEfimeroBaremacion) error {
			return material.VisitarBytes(func([]byte) error { return nil })
		},
	); err != nil {
		return err
	}
	if err := vista.VisitarResolucionIdentidadInternaEstableBaremacion(
		func(string, uint64, string, string, string, string, []byte) error { return nil },
	); err != nil {
		return err
	}
	if _, _, _, _, err := vista.ResumenLlaveroIdentidadesBaremacion(); err != nil {
		return err
	}
	if err := vista.VisitarTopologiaIdentidadesBaremacion(func(int, uint16, uint32, string) error { return nil }); err != nil {
		return err
	}
	if _, _, _, _, err := vista.ResumenLlaveroIndicesBaremacion(); err != nil {
		return err
	}
	if err := vista.VisitarTopologiaIndicesBaremacion(func(int, uint16, uint32, string) error { return nil }); err != nil {
		return err
	}
	if err := vista.VisitarPrincipalesBaremacion(func(int, uint16, uint32, string, string) error { return nil }); err != nil {
		return err
	}
	if err := vista.VisitarEvidenciaAtestacionBaremacion(
		func(string, string, string, uint64, string, []byte) error { return nil },
	); err != nil {
		return err
	}
	go func() {
		defer close(v.hecho)
		_ = vista.VisitarMatrizIndicesBaremacion(func(
			int, int, uint16, uint32, string, string,
		) error {
			v.unaVez.Do(func() {
				close(v.inicio)
				<-v.liberar
			})
			return nil
		})
	}()
	<-v.inicio
	return nil
}

func (c *consumidorNominalNoAutoritativoPrueba) ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(
	ctx context.Context,
	_ SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	vista VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.llamadas++
	c.retuvoVista = vista
	if c.panico {
		panic("panico ficticio del consumidor nominal")
	}
	if interna, ok := vista.(*vistaEfimeraTestimonioIdempotenciaBaremacion); ok {
		if err := interna.conTestimonio(func(testimonio testimonioAtomicoIdempotenciaBaremacion) error {
			clon, err := testimonio.clonar()
			if err == nil {
				c.testimonioCapturado = clon
			}
			return err
		}); err != nil {
			return err
		}
	}
	err := recorrerVistaCompletaNominalPrueba(vista)
	if c.cancelarAlFinal != nil {
		c.cancelarAlFinal()
	}
	if err != nil {
		return err
	}
	return c.errorAlFinal
}

func claveClienteFicticiaPrueba(t *testing.T) ClaveClienteIdempotenciaBaremacion {
	return claveClienteFicticiaConDesplazamientoPrueba(t, 0)
}

func claveClienteFicticiaConDesplazamientoPrueba(
	t *testing.T,
	desplazamiento byte,
) ClaveClienteIdempotenciaBaremacion {
	t.Helper()
	material := make([]byte, 32)
	for posicion := range material {
		material[posicion] = byte(0x80 + (posicion+int(desplazamiento))%64)
	}
	clave, err := NuevaClaveClienteIdempotenciaBaremacion(base64.RawURLEncoding.EncodeToString(material))
	if err != nil {
		t.Fatalf("crear clave cliente ficticia: %v", err)
	}
	return clave
}

func copiarMaterialClaveClientePrueba(clave ClaveClienteIdempotenciaBaremacion) []byte {
	if clave.propietario == nil {
		return nil
	}
	clave.propietario.mu.Lock()
	defer clave.propietario.mu.Unlock()
	return append([]byte(nil), clave.propietario.valor...)
}

func solicitudTestimonioPrueba(t *testing.T) SolicitudTestimonioAtomicoIdempotenciaBaremacion {
	return solicitudTestimonioConSujetoPrueba(t, identidadInternaEstablePrueba(1))
}

func solicitudTestimonioConSujetoPrueba(
	t *testing.T,
	identidad []byte,
) SolicitudTestimonioAtomicoIdempotenciaBaremacion {
	return solicitudTestimonioConIdentidadSeudonimoYClavePrueba(
		t, identidad, "seudonimo-sujeto-ficticio-v1", claveClienteFicticiaPrueba(t),
	)
}

func solicitudTestimonioConIdentidadSeudonimoYClavePrueba(
	t *testing.T,
	identidad []byte,
	claveSeudonimoRef string,
	claveCliente ClaveClienteIdempotenciaBaremacion,
) SolicitudTestimonioAtomicoIdempotenciaBaremacion {
	t.Helper()
	ambito, err := NuevaSolicitudResolverSeudonimoSujetoBaremacion(
		referenciaMaterialOpacaPrueba(501), referenciaMaterialOpacaPrueba(502),
		referenciaMaterialOpacaPrueba(503),
	)
	if err != nil {
		t.Fatalf("crear ambito ficticio: %v", err)
	}
	seudonimoProvisional := SeudonimoSujetoBaremacionHMAC{
		Version:      VersionSeudonimoSujetoBaremacionV1,
		ClaveHMACRef: claveSeudonimoRef, ValorHMAC: huellaIdempotenciaPruebaID(999),
	}
	provisional, err := NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
		"despliegue-ficticio-v1", ModuloIdempotenciaBolsa,
		ClaseCambioIncorporarDecision, ambito, seudonimoProvisional, claveCliente,
	)
	if err != nil {
		t.Fatalf("crear solicitud provisional ficticia: %v", err)
	}
	preimagen, err := copiarCargaDuranteVisitaPrueba(func(visita func(MaterialCanonicoEfimeroBaremacion) error) error {
		return VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
			provisional, identidad, visita,
		)
	})
	if err != nil {
		t.Fatalf("crear material de seudonimo ficticio: %v", err)
	}
	defer borrarBytesBaremacion(preimagen)
	seudonimo := seudonimoProvisional
	seudonimo.ValorHMAC = hmacSHA256Prueba(claveSeudonimoPrueba(claveSeudonimoRef), preimagen)
	solicitud, err := NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
		"despliegue-ficticio-v1", ModuloIdempotenciaBolsa,
		ClaseCambioIncorporarDecision, ambito, seudonimo, claveCliente,
	)
	if err != nil {
		t.Fatalf("crear solicitud ficticia: %v", err)
	}
	return solicitud
}

func construirProductoNominalPrueba(
	t *testing.T,
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	productor ProductorTestimonioAtomicoIdempotenciaBaremacion,
	verificador VerificadorIndependienteTestimonioIdempotenciaBaremacion,
) (productoNominalPrueba, error) {
	t.Helper()
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	if concreto, ok := productor.(*productorTestimonioPrueba); ok && concreto != nil {
		configuracion = concreto.configuracion
	}
	if concreto, ok := verificador.(*verificadorTestimonioPrueba); ok && concreto != nil {
		configuracion = concreto.configuracion
	}
	frontera := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitud, identidadInternaEstablePrueba(1),
	)
	return ejecutarFlujoNominalConFronteraPrueba(
		ctx, solicitud, frontera, productor, verificador, configuracion,
	)
}

func ejecutarFlujoNominalConFronteraPrueba(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	frontera FronteraIdentidadInternaEstableIdempotenciaBaremacion,
	productor ProductorTestimonioAtomicoIdempotenciaBaremacion,
	verificador VerificadorIndependienteTestimonioIdempotenciaBaremacion,
	configuracion configuracionTestimonioPrueba,
) (productoNominalPrueba, error) {
	consumidor := &consumidorNominalNoAutoritativoPrueba{}
	err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		ctx, solicitud, frontera, productor, verificador,
		&raizVerificadorTestimonioPrueba{
			delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
		},
		consumidor,
	)
	return productoNominalPrueba{testimonio: consumidor.testimonioCapturado}, err
}

func TestTestimonioAtomicoOchoPorOchoEsNominalCompletoYCierraVistas(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(8, 8)
	productor := &productorTestimonioPrueba{configuracion: configuracion, mutarEvidencia: true}
	verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
	producto, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t), productor, verificador,
	)
	if err != nil {
		t.Fatalf("construir producto nominal 8x8: %v", err)
	}
	if producto.validar() != nil || productor.llamadas != 1 || verificador.llamadas != 1 {
		t.Fatalf("producto o cardinalidad de llamadas invalida: productor=%d verificador=%d", productor.llamadas, verificador.llamadas)
	}
	if err := productor.retuvoReceptor.InmovilizarLlaveroIdentidadesBaremacion("otro", 1, 1, huellaTextoPrueba("x")); err == nil {
		t.Fatal("el receptor retenido siguio util tras finalizar")
	}
	if err := productor.retuvoFuente.EntregarClaveClienteLoteIdempotenciaBaremacion(
		context.Background(), &consumidorClaveClientePrueba{},
	); err == nil {
		t.Fatal("la fuente retenida permitio una segunda entrega")
	}
	if _, _, _, _, err := verificador.retuvoVista.ResumenLlaveroIndicesBaremacion(); err == nil {
		t.Fatal("la vista retenida siguio util tras verificar")
	}
}

func TestOmisionesAutoconsistentesOchoPorOchoNoSePromueven(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(8, 8)
	posiciones := []struct {
		nombre  string
		omitida int
		esFila  bool
	}{
		{"primera-fila", 0, true}, {"fila-media", 3, true}, {"ultima-fila", 7, true},
		{"primera-columna", 0, false}, {"columna-media", 3, false}, {"ultima-columna", 7, false},
	}
	for _, caso := range posiciones {
		t.Run(caso.nombre, func(t *testing.T) {
			productor := &productorTestimonioPrueba{configuracion: configuracion}
			if caso.esFila {
				productor.filas = posicionesSinPrueba(8, caso.omitida)
			} else {
				productor.columnas = posicionesSinPrueba(8, caso.omitida)
			}
			_, err := construirProductoNominalPrueba(t,
				context.Background(), solicitudTestimonioPrueba(t), productor,
				&verificadorTestimonioPrueba{configuracion: configuracion},
			)
			if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
				t.Fatalf("omision autoconsistente aceptada: %v", err)
			}
		})
	}
}

func TestMatrizUnoPorUnoAutoconsistenteConHistoricosNoSePromueve(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(8, 8)
	productor := &productorTestimonioPrueba{
		configuracion: configuracion, filas: []int{0}, columnas: []int{0},
	}
	_, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t), productor,
		&verificadorTestimonioPrueba{configuracion: configuracion},
	)
	if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("1x1 autoconsistente aceptado pese a historicos 8x8: %v", err)
	}
}

func TestMismaClaveClienteQuedaAisladaPorSujetoYEsEstableEnElMismoSujeto(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(2, 2)
	construir := func(
		solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		identidad []byte,
	) productoNominalPrueba {
		t.Helper()
		producto, err := ejecutarFlujoNominalConFronteraPrueba(
			context.Background(), solicitud,
			nuevaFronteraIdentidadInternaPrueba(configuracion, solicitud, identidad),
			&productorTestimonioPrueba{configuracion: configuracion},
			&verificadorTestimonioPrueba{configuracion: configuracion},
			configuracion,
		)
		if err != nil {
			t.Fatalf("construir producto por sujeto: %v", err)
		}
		return producto
	}
	identidadUno := identidadInternaEstablePrueba(11)
	identidadDos := identidadInternaEstablePrueba(29)
	solicitudUno := solicitudTestimonioConSujetoPrueba(t, identidadUno)
	solicitudUnoRepetida := solicitudTestimonioConSujetoPrueba(t, identidadUno)
	solicitudDos := solicitudTestimonioConSujetoPrueba(t, identidadDos)
	productoUno := construir(solicitudUno, identidadUno)
	productoUnoRepetido := construir(solicitudUnoRepetida, identidadUno)
	productoDos := construir(solicitudDos, identidadDos)
	if productoUno.testimonio.matriz[0][0].igualConstante(productoDos.testimonio.matriz[0][0]) ||
		productoUno.testimonio.principales[0].igualConstante(productoDos.testimonio.principales[0]) {
		t.Fatal("la misma clave cliente produjo principal/indice compartido entre sujetos")
	}
	materialUno, _ := productoUno.testimonio.representacionCanonicaAtestada()
	materialDos, _ := productoDos.testimonio.representacionCanonicaAtestada()
	defer destruirCargaProtegidaBaremacion(&materialUno)
	defer destruirCargaProtegidaBaremacion(&materialDos)
	bytesUno, bytesDos := materialUno.Revelar(), materialDos.Revelar()
	defer borrarBytesBaremacion(bytesUno)
	defer borrarBytesBaremacion(bytesDos)
	if bytes.Equal(bytesUno, bytesDos) {
		t.Fatal("el material atestado no quedo ligado al sujeto")
	}
	materialUnoRepetido, _ := productoUnoRepetido.testimonio.representacionCanonicaAtestada()
	defer destruirCargaProtegidaBaremacion(&materialUnoRepetido)
	bytesUnoRepetido := materialUnoRepetido.Revelar()
	defer borrarBytesBaremacion(bytesUnoRepetido)
	if !bytes.Equal(bytesUno, bytesUnoRepetido) ||
		!productoUno.testimonio.matriz[0][0].igualConstante(productoUnoRepetido.testimonio.matriz[0][0]) {
		t.Fatal("mismo sujeto y clave no produjo material estable")
	}
}

func TestRotarClaveDeSeudonimoConservaCandidatosYClaveClienteSoloCambiaIndices(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(2, 2)
	identidad := identidadInternaEstablePrueba(17)
	solicitudV1 := solicitudTestimonioConIdentidadSeudonimoYClavePrueba(
		t, identidad, "seudonimo-sujeto-ficticio-v1", claveClienteFicticiaPrueba(t),
	)
	solicitudV2 := solicitudTestimonioConIdentidadSeudonimoYClavePrueba(
		t, identidad, "seudonimo-sujeto-ficticio-v2", claveClienteFicticiaPrueba(t),
	)
	construir := func(solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion) productoNominalPrueba {
		producto, err := ejecutarFlujoNominalConFronteraPrueba(
			context.Background(), solicitud,
			nuevaFronteraIdentidadInternaPrueba(configuracion, solicitud, identidad),
			&productorTestimonioPrueba{configuracion: configuracion},
			&verificadorTestimonioPrueba{configuracion: configuracion},
			configuracion,
		)
		if err != nil {
			t.Fatalf("construir producto tras rotacion: %v", err)
		}
		return producto
	}
	productoV1 := construir(solicitudV1)
	productoV2 := construir(solicitudV2)
	for fila := range productoV1.testimonio.principales {
		if !productoV1.testimonio.principales[fila].igualConstante(productoV2.testimonio.principales[fila]) {
			t.Fatalf("rotar K_sujeto cambio principal historico %d", fila)
		}
		for columna := range productoV1.testimonio.matriz[fila] {
			if !productoV1.testimonio.matriz[fila][columna].igualConstante(productoV2.testimonio.matriz[fila][columna]) {
				t.Fatalf("rotar K_sujeto cambio indice historico %d/%d", fila, columna)
			}
		}
	}
	if productoV1.testimonio.resolucionIdentidad.HuellaSHA256 ==
		productoV2.testimonio.resolucionIdentidad.HuellaSHA256 {
		t.Fatal("la auditoria de resolucion no distinguio la rotacion de seudonimo")
	}

	solicitudOtraClave := solicitudTestimonioConIdentidadSeudonimoYClavePrueba(
		t, identidad, "seudonimo-sujeto-ficticio-v1",
		claveClienteFicticiaConDesplazamientoPrueba(t, 19),
	)
	productoOtraClave := construir(solicitudOtraClave)
	for fila := range productoV1.testimonio.principales {
		if !productoV1.testimonio.principales[fila].igualConstante(productoOtraClave.testimonio.principales[fila]) {
			t.Fatalf("la clave cliente contamino principal %d", fila)
		}
		for columna := range productoV1.testimonio.matriz[fila] {
			if productoV1.testimonio.matriz[fila][columna].igualConstante(productoOtraClave.testimonio.matriz[fila][columna]) {
				t.Fatalf("otra clave cliente no cambio indice %d/%d", fila, columna)
			}
		}
	}
}

func TestResolucionIdentidadEsAtomicaUnicaYRechazaCrucesAnclasYAtestaciones(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	identidad := identidadInternaEstablePrueba(7)
	solicitud := solicitudTestimonioConSujetoPrueba(t, identidad)
	frontera := nuevaFronteraIdentidadInternaPrueba(configuracion, solicitud, identidad)
	frontera.identidadSegunda = identidadInternaEstablePrueba(41)
	_, err := ejecutarFlujoNominalConFronteraPrueba(
		context.Background(), solicitud, frontera,
		&productorTestimonioPrueba{configuracion: configuracion},
		&verificadorTestimonioPrueba{configuracion: configuracion},
		configuracion,
	)
	if err != nil || frontera.llamadas != 1 {
		t.Fatalf("la frontera no se resolvio exactamente una vez: llamadas=%d err=%v", frontera.llamadas, err)
	}
	if err := frontera.ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Background(), solicitud.ambitoSujeto, solicitud.seudonimo, frontera.retuvoReceptor,
	); err == nil {
		t.Fatal("el receptor de resolucion retenido siguio abierto tras la fabrica")
	}

	solicitudOtroSujeto := solicitudTestimonioConSujetoPrueba(t, identidadInternaEstablePrueba(33))
	solicitudBaseCruce := solicitudTestimonioConSujetoPrueba(t, identidad)
	solicitudCruzada, err := NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
		solicitudBaseCruce.despliegueRef, solicitudBaseCruce.modulo, solicitudBaseCruce.clase,
		solicitudBaseCruce.ambitoSujeto, solicitudOtroSujeto.seudonimo, claveClienteFicticiaPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ejecutarFlujoNominalConFronteraPrueba(
		context.Background(), solicitudCruzada,
		nuevaFronteraIdentidadInternaPrueba(configuracion, solicitudBaseCruce, identidad),
		&productorTestimonioPrueba{configuracion: configuracion},
		&verificadorTestimonioPrueba{configuracion: configuracion},
		configuracion,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("ambito A + seudonimo B fue aceptado: %v", err)
	}

	for nombre, ancla := range map[string][]byte{
		"ceros": make([]byte, 32),
		"texto": bytes.Repeat([]byte("A"), 32),
		"dni":   append([]byte("12345678Z"), bytes.Repeat([]byte("0"), 23)...),
	} {
		t.Run(nombre, func(t *testing.T) {
			solicitudInvalida := solicitudTestimonioConSujetoPrueba(t, identidad)
			fronteraInvalida := nuevaFronteraIdentidadInternaPrueba(configuracion, solicitudInvalida, ancla)
			if _, err := ejecutarFlujoNominalConFronteraPrueba(
				context.Background(), solicitudInvalida, fronteraInvalida,
				&productorTestimonioPrueba{configuracion: configuracion},
				&verificadorTestimonioPrueba{configuracion: configuracion},
				configuracion,
			); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
				t.Fatalf("ancla predecible aceptada: %v", err)
			}
		})
	}

	solicitudAtestacionFalsa := solicitudTestimonioConSujetoPrueba(t, identidad)
	fronteraAtestacionFalsa := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitudAtestacionFalsa, identidad,
	)
	fronteraAtestacionFalsa.alterarAtestacion = true
	if _, err := ejecutarFlujoNominalConFronteraPrueba(
		context.Background(), solicitudAtestacionFalsa, fronteraAtestacionFalsa,
		&productorTestimonioPrueba{configuracion: configuracion},
		&verificadorTestimonioPrueba{configuracion: configuracion},
		configuracion,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("atestacion de resolucion falsa aceptada: %v", err)
	}
	solicitudHuellaFalsa := solicitudTestimonioConSujetoPrueba(t, identidad)
	fronteraHuellaFalsa := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitudHuellaFalsa, identidad,
	)
	fronteraHuellaFalsa.alterarHuella = true
	if _, err := ejecutarFlujoNominalConFronteraPrueba(
		context.Background(), solicitudHuellaFalsa, fronteraHuellaFalsa,
		&productorTestimonioPrueba{configuracion: configuracion},
		&verificadorTestimonioPrueba{configuracion: configuracion},
		configuracion,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("huella de snapshot falsa aceptada: %v", err)
	}
}

func TestPanicoDeFronteraTrasRegistrarDestruyeReceptorYClave(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	solicitud := solicitudTestimonioPrueba(t)
	aliasClave := solicitud.claveCliente.propietario.valor
	frontera := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitud, identidadInternaEstablePrueba(1),
	)
	frontera.panicoTrasRegistrar = true
	panico := ejecutarYRecuperarPanicoPrueba(func() {
		_ = ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitud, frontera,
			&productorTestimonioPrueba{configuracion: configuracion},
			&verificadorTestimonioPrueba{configuracion: configuracion},
			&raizVerificadorTestimonioPrueba{
				delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
			},
			&consumidorNominalNoAutoritativoPrueba{},
		)
	})
	if panico == nil {
		t.Fatal("el panico de frontera no se propago")
	}
	receptor, ok := frontera.retuvoReceptor.(*receptorResolucionIdentidadInternaEstableBaremacion)
	if !ok || receptor == nil {
		t.Fatal("la frontera no retuvo el receptor concreto esperado")
	}
	receptor.mu.Lock()
	cerrado := receptor.cerrado
	sinAncla := len(receptor.lote.ancla) == 0
	sinAtestacion := len(receptor.lote.instantanea.ValorAtestacion) == 0
	receptor.mu.Unlock()
	if !cerrado || !sinAncla || !sinAtestacion {
		t.Fatalf("receptor no destruido tras panico: cerrado=%v ancla=%v atestacion=%v",
			cerrado, sinAncla, sinAtestacion)
	}
	exigirClaveClienteDestruidaPrueba(t, aliasClave, solicitud.claveCliente, solicitud)
}

func TestFormulaDEC045YLaCoberturaCompletaSonObligatorias(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(2, 2)
	for nombre, productor := range map[string]*productorTestimonioPrueba{
		"principal-con-clave-cliente": {configuracion: configuracion, principalConClaveCliente: true},
		"indice-sin-principal":        {configuracion: configuracion, indiceSinPrincipal: true},
		"esquema-alterado":            {configuracion: configuracion, indiceEsquemaAlterado: true},
		"politica-alterada":           {configuracion: configuracion, indicePoliticaAlterada: true},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := construirProductoNominalPrueba(
				t, context.Background(), solicitudTestimonioPrueba(t), productor,
				&verificadorTestimonioPrueba{configuracion: configuracion},
			); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
				t.Fatalf("formula maliciosa aceptada: %v", err)
			}
		})
	}

	solicitud := solicitudTestimonioPrueba(t)
	frontera := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitud, identidadInternaEstablePrueba(1),
	)
	verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
	raizIncompleta := &verificadorTestimonioPrueba{
		configuracion: configuracion, omitirResolucion: true,
	}
	consumidor := &consumidorNominalNoAutoritativoPrueba{}
	err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		context.Background(), solicitud, frontera,
		&productorTestimonioPrueba{configuracion: configuracion}, verificador,
		&raizVerificadorTestimonioPrueba{delegado: raizIncompleta}, consumidor,
	)
	if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("raiz con cobertura incompleta fue aceptada: %v", err)
	}
	if frontera.llamadas != 1 || verificador.llamadas != 1 || raizIncompleta.llamadas != 1 || consumidor.llamadas != 0 {
		t.Fatalf("fallo no alcanzo exactamente la raiz: frontera=%d verificador=%d raiz=%d consumidor=%d",
			frontera.llamadas, verificador.llamadas, raizIncompleta.llamadas, consumidor.llamadas)
	}
}

func TestCierresFailClosedNoEsperanCallbacksAsincronos(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	solicitudProduccion := solicitudTestimonioPrueba(t)
	productorAsincrono := &productorFuenteAsincronaPrueba{
		inicio: make(chan struct{}), liberar: make(chan struct{}), hecho: make(chan struct{}),
	}
	resultadoProduccion := make(chan error, 1)
	go func() {
		err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitudProduccion,
			nuevaFronteraIdentidadInternaPrueba(
				configuracion, solicitudProduccion, identidadInternaEstablePrueba(1),
			),
			productorAsincrono,
			&verificadorTestimonioPrueba{configuracion: configuracion},
			&raizVerificadorTestimonioPrueba{
				delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
			},
			&consumidorNominalNoAutoritativoPrueba{},
		)
		resultadoProduccion <- err
	}()
	select {
	case err := <-resultadoProduccion:
		if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
			t.Fatalf("productor asincrono no fallo cerrado: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("la fabrica espero indefinidamente la entrega asincrona")
	}
	if err := productorAsincrono.retuvoFuente.EntregarClaveClienteLoteIdempotenciaBaremacion(
		context.Background(), &consumidorClaveClientePrueba{},
	); err == nil {
		t.Fatal("la fuente retenida siguio abierta tras fallo asincrono")
	}
	if err := productorAsincrono.retuvoReceptor.InmovilizarLlaveroIdentidadesBaremacion(
		"otro-llavero", 1, 1, huellaTextoPrueba("otro"),
	); err == nil {
		t.Fatal("el receptor retenido siguio abierto tras fallo asincrono")
	}
	close(productorAsincrono.liberar)
	select {
	case <-productorAsincrono.hecho:
	case <-time.After(time.Second):
		t.Fatal("el callback asincrono no termino tras liberarlo")
	}

	solicitudRaiz := solicitudTestimonioPrueba(t)
	raizAsincrona := &verificadorVistaAsincronaPrueba{
		inicio: make(chan struct{}), liberar: make(chan struct{}), hecho: make(chan struct{}),
	}
	resultadoPuente := make(chan error, 1)
	go func() {
		resultadoPuente <- ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitudRaiz,
			nuevaFronteraIdentidadInternaPrueba(
				configuracion, solicitudRaiz, identidadInternaEstablePrueba(1),
			),
			&productorTestimonioPrueba{configuracion: configuracion},
			&verificadorTestimonioPrueba{configuracion: configuracion},
			raizAsincrona, &consumidorNominalNoAutoritativoPrueba{},
		)
	}()
	select {
	case err := <-resultadoPuente:
		if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
			t.Fatalf("vista asincrona no fallo cerrado: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("el puente espero indefinidamente una visita asincrona")
	}
	if _, _, _, _, err := raizAsincrona.retuvoVista.ResumenLlaveroIndicesBaremacion(); err == nil {
		t.Fatal("la vista retenida siguio abierta tras detectar actividad asincrona")
	}
	close(raizAsincrona.liberar)
	select {
	case <-raizAsincrona.hecho:
	case <-time.After(time.Second):
		t.Fatal("la visita asincrona no termino tras liberarla")
	}
}

func ejecutarYRecuperarPanicoPrueba(ejecutar func()) (panico any) {
	defer func() { panico = recover() }()
	ejecutar()
	return nil
}

func TestPanicosCierranFuentesReceptoresYVistasRetenidas(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	solicitudProductor := solicitudTestimonioPrueba(t)
	productorPanico := &productorTestimonioPrueba{
		configuracion: configuracion, panicoAlInicio: true,
	}
	if panico := ejecutarYRecuperarPanicoPrueba(func() {
		_, _ = construirProductoNominalPrueba(
			t, context.Background(), solicitudProductor, productorPanico,
			&verificadorTestimonioPrueba{configuracion: configuracion},
		)
	}); panico == nil {
		t.Fatal("el panico del productor no se propago")
	}
	if err := productorPanico.retuvoFuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Background(), &consumidorIdentidadInternaPrueba{},
	); err == nil {
		t.Fatal("fuente de identidad siguio abierta tras panico del productor")
	}
	if err := productorPanico.retuvoFuente.EntregarClaveClienteLoteIdempotenciaBaremacion(
		context.Background(), &consumidorClaveClientePrueba{},
	); err == nil {
		t.Fatal("fuente cliente siguio abierta tras panico del productor")
	}
	if err := productorPanico.retuvoReceptor.InmovilizarLlaveroIdentidadesBaremacion(
		"otro", 1, 1, huellaTextoPrueba("otro"),
	); err == nil {
		t.Fatal("receptor siguio abierto tras panico del productor")
	}

	verificadorPanico := &verificadorTestimonioPrueba{
		configuracion: configuracion, panicoAlInicio: true,
	}
	solicitudVerificador := solicitudTestimonioPrueba(t)
	if panico := ejecutarYRecuperarPanicoPrueba(func() {
		_, _ = construirProductoNominalPrueba(
			t, context.Background(), solicitudVerificador,
			&productorTestimonioPrueba{configuracion: configuracion}, verificadorPanico,
		)
	}); panico == nil {
		t.Fatal("el panico del verificador no se propago")
	}
	if _, _, _, _, err := verificadorPanico.retuvoVista.ResumenLlaveroIndicesBaremacion(); err == nil {
		t.Fatal("vista siguio abierta tras panico del verificador")
	}
	if err := verificadorPanico.retuvoFuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Background(), &consumidorIdentidadInternaPrueba{},
	); err == nil {
		t.Fatal("fuente identidad siguio abierta tras panico del verificador")
	}
	if err := verificadorPanico.retuvoFuenteClave.EntregarClaveClienteLoteIdempotenciaBaremacion(
		context.Background(), &consumidorClaveClientePrueba{},
	); err == nil {
		t.Fatal("fuente cliente siguio abierta tras panico del verificador")
	}

	solicitudRaiz := solicitudTestimonioPrueba(t)
	raizPanicoDelegada := &verificadorTestimonioPrueba{
		configuracion: configuracion, panicoAlInicio: true,
	}
	if panico := ejecutarYRecuperarPanicoPrueba(func() {
		_ = ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitudRaiz,
			nuevaFronteraIdentidadInternaPrueba(
				configuracion, solicitudRaiz, identidadInternaEstablePrueba(1),
			),
			&productorTestimonioPrueba{configuracion: configuracion},
			&verificadorTestimonioPrueba{configuracion: configuracion},
			&raizVerificadorTestimonioPrueba{delegado: raizPanicoDelegada},
			&consumidorNominalNoAutoritativoPrueba{},
		)
	}); panico == nil {
		t.Fatal("el panico de la raiz no se propago")
	}
	if _, _, _, _, err := raizPanicoDelegada.retuvoVista.ResumenLlaveroIdentidadesBaremacion(); err == nil {
		t.Fatal("vista siguio abierta tras panico de la raiz")
	}

	solicitudConsumidor := solicitudTestimonioPrueba(t)
	consumidorPanico := &consumidorNominalNoAutoritativoPrueba{panico: true}
	if panico := ejecutarYRecuperarPanicoPrueba(func() {
		_ = ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitudConsumidor,
			nuevaFronteraIdentidadInternaPrueba(
				configuracion, solicitudConsumidor, identidadInternaEstablePrueba(1),
			),
			&productorTestimonioPrueba{configuracion: configuracion},
			&verificadorTestimonioPrueba{configuracion: configuracion},
			&raizVerificadorTestimonioPrueba{
				delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
			},
			consumidorPanico,
		)
	}); panico == nil {
		t.Fatal("el panico del consumidor no se propago")
	}
	if _, _, _, _, err := consumidorPanico.retuvoVista.ResumenLlaveroIdentidadesBaremacion(); err == nil {
		t.Fatal("vista siguio abierta tras panico del consumidor")
	}
}

func TestTestimonioIncompletoOSinEvidenciaFallaAntesDeVerificar(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(2, 2)
	for _, caso := range []struct {
		nombre    string
		productor *productorTestimonioPrueba
	}{
		{"celda-ausente", &productorTestimonioPrueba{configuracion: configuracion, omitirUltimaCelda: true}},
		{"evidencia-ausente", &productorTestimonioPrueba{configuracion: configuracion, omitirEvidencia: true}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
			_, err := construirProductoNominalPrueba(t,
				context.Background(), solicitudTestimonioPrueba(t), caso.productor, verificador,
			)
			if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) || verificador.llamadas != 0 {
				t.Fatalf("testimonio incompleto no fallo cerrado: err=%v llamadas=%d", err, verificador.llamadas)
			}
		})
	}
}

func TestColisionTransversalEntrePrincipalesFallaAntesDeVerificar(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(3, 3)
	verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
	_, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t),
		&productorTestimonioPrueba{configuracion: configuracion, colisionTransversal: true},
		verificador,
	)
	if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) || verificador.llamadas != 0 {
		t.Fatalf("colision transversal no fallo cerrada: err=%v llamadas=%d", err, verificador.llamadas)
	}
}

func TestIndiceIgualAPrincipalPosteriorYAliasLogicosFallanAntesDeRaiz(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(2, 2)
	verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
	_, err := construirProductoNominalPrueba(
		t, context.Background(), solicitudTestimonioPrueba(t),
		&productorTestimonioPrueba{
			configuracion: configuracion, colisionConPrincipalPosterior: true,
		},
		verificador,
	)
	if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) || verificador.llamadas != 0 {
		t.Fatalf("indice igual a principal posterior aceptado: err=%v llamadas=%d", err, verificador.llamadas)
	}
	for nombre, mutar := range map[string]func(*configuracionTestimonioPrueba){
		"llavero": func(c *configuracionTestimonioPrueba) { c.indicesRef = c.identidadesRef },
		"clave":   func(c *configuracionTestimonioPrueba) { c.indices[0].ClaveHMACRef = c.identidades[0].ClaveHMACRef },
	} {
		t.Run(nombre, func(t *testing.T) {
			candidata := nuevaConfiguracionTestimonioPrueba(2, 2)
			mutar(&candidata)
			raiz := &verificadorTestimonioPrueba{configuracion: candidata}
			_, err := construirProductoNominalPrueba(
				t, context.Background(), solicitudTestimonioPrueba(t),
				&productorTestimonioPrueba{configuracion: candidata}, raiz,
			)
			if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) || raiz.llamadas != 0 {
				t.Fatalf("alias logico aceptado: err=%v llamadas=%d", err, raiz.llamadas)
			}
		})
	}
}

func TestClaveClienteSoloUUIDv4CanonicoOMaterialBinarioOpaco(t *testing.T) {
	validas := []string{"123e4567-e89b-42d3-a456-426614174000"}
	material := make([]byte, 32)
	for posicion := range material {
		material[posicion] = byte(0x80 + posicion)
	}
	validas = append(validas, base64.RawURLEncoding.EncodeToString(material))
	for _, valor := range validas {
		if _, err := NuevaClaveClienteIdempotenciaBaremacion(valor); err != nil {
			t.Errorf("clave valida rechazada %q: %v", valor, err)
		}
	}
	repetida := make([]byte, 32)
	for posicion := range repetida {
		repetida[posicion] = 0xff
	}
	humana := base64.RawURLEncoding.EncodeToString([]byte("texto humano suficientemente largo para treinta y dos bytes"))
	invalidas := []string{
		"123E4567-E89B-42D3-A456-426614174000",
		"123e4567-e89b-12d3-a456-426614174000",
		"123e4567-e89b-42d3-c456-426614174000",
		"12345678Z", "X1234567L", "persona@example.invalid", "/ruta/privada/documento",
		"clave-idempotencia-legible", humana,
		base64.RawURLEncoding.EncodeToString([]byte("material-corto")),
		base64.RawURLEncoding.EncodeToString(repetida),
		base64.RawURLEncoding.EncodeToString(material) + "=",
	}
	for _, valor := range invalidas {
		if _, err := NuevaClaveClienteIdempotenciaBaremacion(valor); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
			t.Errorf("clave humana/no canonica aceptada %q: %v", valor, err)
		}
	}
	tipo := reflect.TypeOf(ClaveClienteIdempotenciaBaremacion{})
	for _, metodo := range []string{"Revelar", "RevelarParaDerivacion", "Clonar", "Validar", "Bytes"} {
		if _, existe := tipo.MethodByName(metodo); existe {
			t.Errorf("metodo publico de fuga presente: %s", metodo)
		}
	}
}

func exigirClaveClienteDestruidaPrueba(
	t *testing.T,
	alias []byte,
	clave ClaveClienteIdempotenciaBaremacion,
	solicitudes ...SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) {
	t.Helper()
	if !bytes.Equal(alias, make([]byte, len(alias))) {
		t.Fatal("el propietario no borro el material de clave cliente")
	}
	if clave.validar() == nil || clave.propietario == nil {
		t.Fatal("una copia de la clave cliente siguio valida")
	}
	clave.propietario.mu.Lock()
	destruida := clave.propietario.destruido
	sinMaterial := len(clave.propietario.valor) == 0 && clave.propietario.formato == 0 &&
		clave.propietario.reclamacion == nil
	clave.propietario.mu.Unlock()
	if !destruida || !sinMaterial {
		t.Fatal("el propietario no quedo destruido y vacio")
	}
	for posicion, solicitud := range solicitudes {
		if solicitud.validar() == nil {
			t.Fatalf("la copia %d de la solicitud siguio valida", posicion)
		}
	}
}

func TestClaveClienteOneShotSeDestruyeEnExitoErrorYPanico(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	t.Run("exito", func(t *testing.T) {
		solicitud := solicitudTestimonioPrueba(t)
		copiaSolicitud := solicitud
		copiaClave := solicitud.claveCliente
		alias := solicitud.claveCliente.propietario.valor
		productor := &productorTestimonioPrueba{configuracion: configuracion}
		producto, err := construirProductoNominalPrueba(
			t, context.Background(), solicitud, productor,
			&verificadorTestimonioPrueba{configuracion: configuracion},
		)
		if err != nil {
			t.Fatalf("flujo valido: %v", err)
		}
		defer destruirTestimonioAtomicoIdempotenciaBaremacion(&producto.testimonio)
		exigirClaveClienteDestruidaPrueba(
			t, alias, copiaClave, solicitud, copiaSolicitud, productor.retuvoSolicitud,
		)
	})

	t.Run("error", func(t *testing.T) {
		solicitud := solicitudTestimonioPrueba(t)
		copiaSolicitud := solicitud
		copiaClave := solicitud.claveCliente
		alias := solicitud.claveCliente.propietario.valor
		var productorNulo *productorTestimonioPrueba
		_, err := construirProductoNominalPrueba(
			t, context.Background(), solicitud, productorNulo,
			&verificadorTestimonioPrueba{configuracion: configuracion},
		)
		if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
			t.Fatalf("error preflight inesperado: %v", err)
		}
		exigirClaveClienteDestruidaPrueba(t, alias, copiaClave, solicitud, copiaSolicitud)
	})

	t.Run("panico", func(t *testing.T) {
		solicitud := solicitudTestimonioPrueba(t)
		copiaSolicitud := solicitud
		copiaClave := solicitud.claveCliente
		alias := solicitud.claveCliente.propietario.valor
		productor := &productorTestimonioPrueba{
			configuracion: configuracion, panicoAlInicio: true,
		}
		panico := ejecutarYRecuperarPanicoPrueba(func() {
			_, _ = construirProductoNominalPrueba(
				t, context.Background(), solicitud, productor,
				&verificadorTestimonioPrueba{configuracion: configuracion},
			)
		})
		if panico == nil {
			t.Fatal("el panico ficticio no se propago")
		}
		exigirClaveClienteDestruidaPrueba(
			t, alias, copiaClave, solicitud, copiaSolicitud, productor.retuvoSolicitud,
		)
	})
}

func TestConstructorInvalidoNoReclamaNiDestruyeClaveCliente(t *testing.T) {
	clave := claveClienteFicticiaPrueba(t)
	propietario := clave.propietario
	antes := copiarMaterialClaveClientePrueba(clave)
	defer borrarBytesBaremacion(antes)
	ambito, err := NuevaSolicitudResolverSeudonimoSujetoBaremacion(
		referenciaMaterialOpacaPrueba(801), referenciaMaterialOpacaPrueba(802),
		referenciaMaterialOpacaPrueba(803),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
		"despliegue no canonico", ModuloIdempotenciaBolsa, ClaseCambioIncorporarDecision,
		ambito,
		SeudonimoSujetoBaremacionHMAC{
			Version:      VersionSeudonimoSujetoBaremacionV1,
			ClaveHMACRef: "seudonimo-ficticio-v1", ValorHMAC: huellaIdempotenciaPruebaID(801),
		},
		clave,
	)
	if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("constructor invalido aceptado: %v", err)
	}
	despues := copiarMaterialClaveClientePrueba(clave)
	defer borrarBytesBaremacion(despues)
	if clave.propietario != propietario || clave.validar() != nil || !bytes.Equal(antes, despues) {
		t.Fatal("el constructor fallido altero el propietario aportado")
	}
	propietario.mu.Lock()
	reclamada := propietario.reclamacion != nil || propietario.destruido
	propietario.mu.Unlock()
	if reclamada {
		t.Fatal("el constructor fallido reclamo o destruyo la clave")
	}
}

func TestTokenReclamacionTieneIdentidadNoCeroYNoSeReutiliza(t *testing.T) {
	claveUno := claveClienteFicticiaConDesplazamientoPrueba(t, 3)
	claveDos := claveClienteFicticiaConDesplazamientoPrueba(t, 9)
	tokenUno := claveUno.reclamarUsoExclusivo()
	tokenDos := claveDos.reclamarUsoExclusivo()
	if tokenUno == nil || tokenDos == nil || tokenUno == tokenDos {
		t.Fatal("las reclamaciones no recibieron identidades distintas")
	}
	if reflect.TypeOf(*tokenUno).Size() == 0 || tokenUno.marcador == 0 || tokenDos.marcador == 0 {
		t.Fatal("el token usa un tipo de tamano cero o sin marcador")
	}
	claveUno.finalizarUsoYDestruir(tokenUno)
	claveDos.finalizarUsoYDestruir(tokenDos)
}

func TestReclamacionConcurrenteClaveClienteTieneUnGanadorSinSabotaje(t *testing.T) {
	const competidores = 16
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	solicitud := solicitudTestimonioPrueba(t)
	copiaSolicitud := solicitud
	copiaClave := solicitud.claveCliente
	alias := solicitud.claveCliente.propietario.valor
	fronteraBase := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitud, identidadInternaEstablePrueba(1),
	)
	frontera := &fronteraIdentidadBloqueantePrueba{
		delegado: fronteraBase, inicio: make(chan struct{}), liberar: make(chan struct{}),
	}
	productor := &productorTestimonioPrueba{configuracion: configuracion}
	verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
	raizDelegada := &verificadorTestimonioPrueba{configuracion: configuracion}
	consumidor := &consumidorNominalNoAutoritativoPrueba{}
	resultados := make(chan error, competidores)
	iniciar := make(chan struct{})
	for posicion := 0; posicion < competidores; posicion++ {
		go func() {
			<-iniciar
			resultados <- ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
				context.Background(), solicitud, frontera, productor, verificador,
				&raizVerificadorTestimonioPrueba{delegado: raizDelegada}, consumidor,
			)
		}()
	}
	close(iniciar)
	select {
	case <-frontera.inicio:
	case <-time.After(time.Second):
		t.Fatal("ningun competidor alcanzo la frontera")
	}
	for posicion := 0; posicion < competidores-1; posicion++ {
		select {
		case err := <-resultados:
			if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
				t.Fatalf("perdedor %d devolvio error inesperado: %v", posicion, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("el perdedor %d no fallo mientras el ganador estaba activo", posicion)
		}
	}
	solicitud.claveCliente.propietario.mu.Lock()
	activa := !solicitud.claveCliente.propietario.destruido &&
		solicitud.claveCliente.propietario.reclamacion != nil &&
		solicitud.claveCliente.propietario.reclamacion.marcador == 1 &&
		len(solicitud.claveCliente.propietario.valor) != 0
	solicitud.claveCliente.propietario.mu.Unlock()
	if !activa {
		t.Fatal("un perdedor saboteo la reclamacion o el material del ganador")
	}
	if solicitud.validar() == nil || copiaSolicitud.validar() == nil || copiaClave.validar() == nil {
		t.Fatal("una copia sin token siguio util mientras el ganador estaba activo")
	}
	close(frontera.liberar)
	select {
	case err := <-resultados:
		if err != nil {
			t.Fatalf("el ganador no completo el flujo: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("el ganador no termino al liberar la frontera")
	}
	if frontera.llamadas != 1 || fronteraBase.llamadas != 1 || productor.llamadas != 1 ||
		verificador.llamadas != 1 || raizDelegada.llamadas != 1 || consumidor.llamadas != 1 {
		t.Fatalf("cardinalidad one-shot invalida: frontera=%d/base=%d productor=%d verificador=%d raiz=%d consumidor=%d",
			frontera.llamadas, fronteraBase.llamadas, productor.llamadas,
			verificador.llamadas, raizDelegada.llamadas, consumidor.llamadas)
	}
	defer destruirTestimonioAtomicoIdempotenciaBaremacion(&consumidor.testimonioCapturado)
	exigirClaveClienteDestruidaPrueba(
		t, alias, copiaClave, solicitud, copiaSolicitud, productor.retuvoSolicitud,
	)
}

type dependenciaDoblePrueba struct {
	productor   productorTestimonioPrueba
	verificador verificadorTestimonioPrueba
}

type dependenciaRaizConsumidorPrueba struct {
	verificador verificadorTestimonioPrueba
	consumidor  consumidorNominalNoAutoritativoPrueba
}

func (d *dependenciaRaizConsumidorPrueba) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fi FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fc FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	v VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	return d.verificador.VerificarTestimonioAtomicoIdempotenciaBaremacion(ctx, s, fi, fc, v)
}

func (d *dependenciaRaizConsumidorPrueba) ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(
	ctx context.Context,
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	v VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	return d.consumidor.ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(ctx, s, v)
}

func TestDiversidadNominalRechazaMismoObjetoYDosPunterosDelMismoTipo(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	ejecutar := func(
		t *testing.T,
		productor ProductorTestimonioAtomicoIdempotenciaBaremacion,
		verificador VerificadorIndependienteTestimonioIdempotenciaBaremacion,
		raiz VerificadorIndependienteTestimonioIdempotenciaBaremacion,
		consumidor ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
	) {
		t.Helper()
		solicitud := solicitudTestimonioPrueba(t)
		frontera := nuevaFronteraIdentidadInternaPrueba(
			configuracion, solicitud, identidadInternaEstablePrueba(1),
		)
		err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitud, frontera, productor, verificador, raiz, consumidor,
		)
		if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
			t.Fatalf("composicion sin diversidad aceptada: %v", err)
		}
		if frontera.llamadas != 0 {
			t.Fatalf("el fallo de composicion alcanzo la frontera %d veces", frontera.llamadas)
		}
	}

	for _, dosPunteros := range []bool{false, true} {
		sufijo := "mismo-objeto"
		if dosPunteros {
			sufijo = "dos-punteros"
		}
		t.Run("productor-raiz/"+sufijo, func(t *testing.T) {
			uno := &dependenciaDoblePrueba{
				productor:   productorTestimonioPrueba{configuracion: configuracion},
				verificador: verificadorTestimonioPrueba{configuracion: configuracion},
			}
			dos := uno
			if dosPunteros {
				dos = &dependenciaDoblePrueba{
					productor:   productorTestimonioPrueba{configuracion: configuracion},
					verificador: verificadorTestimonioPrueba{configuracion: configuracion},
				}
			}
			ejecutar(t, uno, &verificadorTestimonioPrueba{configuracion: configuracion}, dos,
				&consumidorNominalNoAutoritativoPrueba{})
		})
		t.Run("verificador-raiz/"+sufijo, func(t *testing.T) {
			uno := &dependenciaDoblePrueba{
				productor:   productorTestimonioPrueba{configuracion: configuracion},
				verificador: verificadorTestimonioPrueba{configuracion: configuracion},
			}
			dos := uno
			if dosPunteros {
				dos = &dependenciaDoblePrueba{
					productor:   productorTestimonioPrueba{configuracion: configuracion},
					verificador: verificadorTestimonioPrueba{configuracion: configuracion},
				}
			}
			ejecutar(t, &productorTestimonioPrueba{configuracion: configuracion}, uno, dos,
				&consumidorNominalNoAutoritativoPrueba{})
		})
		t.Run("raiz-consumidor/"+sufijo, func(t *testing.T) {
			uno := &dependenciaRaizConsumidorPrueba{
				verificador: verificadorTestimonioPrueba{configuracion: configuracion},
			}
			dos := uno
			if dosPunteros {
				dos = &dependenciaRaizConsumidorPrueba{
					verificador: verificadorTestimonioPrueba{configuracion: configuracion},
				}
			}
			ejecutar(t, &productorTestimonioPrueba{configuracion: configuracion},
				&verificadorTestimonioPrueba{configuracion: configuracion}, uno, dos)
		})
	}
}

func (d *dependenciaDoblePrueba) ProducirTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fi FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fc FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	r ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion,
) error {
	return d.productor.ProducirTestimonioAtomicoIdempotenciaBaremacion(ctx, s, fi, fc, r)
}

func (d *dependenciaDoblePrueba) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fi FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fc FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	v VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	return d.verificador.VerificarTestimonioAtomicoIdempotenciaBaremacion(ctx, s, fi, fc, v)
}

func TestDependenciasNulasIgualesYContextosCanceladosFallanCerrado(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	var productorNulo *productorTestimonioPrueba
	var verificadorNulo *verificadorTestimonioPrueba
	verificadorTrasProductorNulo := &verificadorTestimonioPrueba{configuracion: configuracion}
	if _, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t), productorNulo, verificadorTrasProductorNulo,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("productor typed nil aceptado: %v", err)
	}
	if verificadorTrasProductorNulo.llamadas != 0 {
		t.Fatal("el fallo typed nil alcanzo indebidamente al verificador")
	}
	productorTrasVerificadorNulo := &productorTestimonioPrueba{configuracion: configuracion}
	if _, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t), productorTrasVerificadorNulo, verificadorNulo,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("verificador typed nil aceptado: %v", err)
	}
	if productorTrasVerificadorNulo.llamadas != 0 {
		t.Fatal("el fallo typed nil alcanzo indebidamente al productor")
	}
	doble := &dependenciaDoblePrueba{
		productor:   productorTestimonioPrueba{configuracion: configuracion},
		verificador: verificadorTestimonioPrueba{configuracion: configuracion},
	}
	if _, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t), doble, doble,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("misma dependencia productora/verificadora aceptada: %v", err)
	}
	if doble.productor.llamadas != 0 || doble.verificador.llamadas != 0 {
		t.Fatal("la dependencia duplicada fue invocada antes de fallar")
	}
	otroDoble := &dependenciaDoblePrueba{
		productor:   productorTestimonioPrueba{configuracion: configuracion},
		verificador: verificadorTestimonioPrueba{configuracion: configuracion},
	}
	if _, err := construirProductoNominalPrueba(t,
		context.Background(), solicitudTestimonioPrueba(t), doble, otroDoble,
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("misma implementacion bajo dos punteros aceptada como productor/verificador: %v", err)
	}
	if doble.productor.llamadas != 0 || otroDoble.verificador.llamadas != 0 {
		t.Fatal("dos punteros del mismo tipo fueron invocados antes de fallar")
	}
	ctxPrevio, cancelarPrevio := context.WithCancel(context.Background())
	cancelarPrevio()
	productorPrevio := &productorTestimonioPrueba{configuracion: configuracion}
	if _, err := construirProductoNominalPrueba(t,
		ctxPrevio, solicitudTestimonioPrueba(t), productorPrevio,
		&verificadorTestimonioPrueba{configuracion: configuracion},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion previa no preservada: %v", err)
	}
	if productorPrevio.llamadas != 0 {
		t.Fatal("la cancelacion previa alcanzo al productor")
	}
	ctxProduccion, cancelarProduccion := context.WithCancel(context.Background())
	productorCancelador := &productorTestimonioPrueba{
		configuracion: configuracion, cancelarAlFinal: cancelarProduccion,
	}
	verificadorTrasCancelacion := &verificadorTestimonioPrueba{configuracion: configuracion}
	if _, err := construirProductoNominalPrueba(t,
		ctxProduccion, solicitudTestimonioPrueba(t), productorCancelador,
		verificadorTrasCancelacion,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion al terminar produccion no preservada: %v", err)
	}
	if productorCancelador.llamadas != 1 || verificadorTrasCancelacion.llamadas != 0 {
		t.Fatalf("cancelacion de produccion alcanzo fase incorrecta: productor=%d verificador=%d",
			productorCancelador.llamadas, verificadorTrasCancelacion.llamadas)
	}
	ctxVerificacion, cancelarVerificacion := context.WithCancel(context.Background())
	productorAntesVerificacion := &productorTestimonioPrueba{configuracion: configuracion}
	verificadorCancelador := &verificadorTestimonioPrueba{
		configuracion: configuracion, cancelarAlFinal: cancelarVerificacion,
	}
	if _, err := construirProductoNominalPrueba(t,
		ctxVerificacion, solicitudTestimonioPrueba(t), productorAntesVerificacion,
		verificadorCancelador,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion al terminar verificacion no preservada: %v", err)
	}
	if productorAntesVerificacion.llamadas != 1 || verificadorCancelador.llamadas != 1 {
		t.Fatalf("cancelacion de verificacion no alcanzo fase esperada: productor=%d verificador=%d",
			productorAntesVerificacion.llamadas, verificadorCancelador.llamadas)
	}

	ctxLimite, cancelarLimite := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelarLimite()
	productorTrasLimite := &productorTestimonioPrueba{configuracion: configuracion}
	if _, err := construirProductoNominalPrueba(
		t, ctxLimite, solicitudTestimonioPrueba(t), productorTrasLimite,
		&verificadorTestimonioPrueba{configuracion: configuracion},
	); !errors.Is(err, context.DeadlineExceeded) || productorTrasLimite.llamadas != 0 {
		t.Fatalf("deadline previo no preservado: err=%v productor=%d", err, productorTrasLimite.llamadas)
	}

	solicitudFrontera := solicitudTestimonioPrueba(t)
	ctxFrontera, cancelarFrontera := context.WithCancel(context.Background())
	fronteraCanceladora := nuevaFronteraIdentidadInternaPrueba(
		configuracion, solicitudFrontera, identidadInternaEstablePrueba(1),
	)
	fronteraCanceladora.cancelarAlFinal = cancelarFrontera
	productorTrasFrontera := &productorTestimonioPrueba{configuracion: configuracion}
	if err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		ctxFrontera, solicitudFrontera, fronteraCanceladora, productorTrasFrontera,
		&verificadorTestimonioPrueba{configuracion: configuracion},
		&raizVerificadorTestimonioPrueba{
			delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
		},
		&consumidorNominalNoAutoritativoPrueba{},
	); !errors.Is(err, context.Canceled) || fronteraCanceladora.llamadas != 1 ||
		productorTrasFrontera.llamadas != 0 {
		t.Fatalf("cancelacion de frontera no preservada/aislada: err=%v frontera=%d productor=%d",
			err, fronteraCanceladora.llamadas, productorTrasFrontera.llamadas)
	}

	solicitudRaiz := solicitudTestimonioPrueba(t)
	ctxRaiz, cancelarRaiz := context.WithCancel(context.Background())
	productorAntesRaiz := &productorTestimonioPrueba{configuracion: configuracion}
	verificadorAntesRaiz := &verificadorTestimonioPrueba{configuracion: configuracion}
	raizCanceladora := &verificadorTestimonioPrueba{
		configuracion: configuracion, cancelarAlFinal: cancelarRaiz,
	}
	consumidorTrasRaiz := &consumidorNominalNoAutoritativoPrueba{}
	if err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		ctxRaiz, solicitudRaiz,
		nuevaFronteraIdentidadInternaPrueba(
			configuracion, solicitudRaiz, identidadInternaEstablePrueba(1),
		),
		productorAntesRaiz, verificadorAntesRaiz,
		&raizVerificadorTestimonioPrueba{delegado: raizCanceladora}, consumidorTrasRaiz,
	); !errors.Is(err, context.Canceled) || productorAntesRaiz.llamadas != 1 ||
		verificadorAntesRaiz.llamadas != 1 || raizCanceladora.llamadas != 1 ||
		consumidorTrasRaiz.llamadas != 0 {
		t.Fatalf("cancelacion de raiz no preservada/aislada: err=%v productor=%d verificador=%d raiz=%d consumidor=%d",
			err, productorAntesRaiz.llamadas, verificadorAntesRaiz.llamadas,
			raizCanceladora.llamadas, consumidorTrasRaiz.llamadas)
	}

	solicitudConsumidor := solicitudTestimonioPrueba(t)
	ctxConsumidor, cancelarConsumidor := context.WithCancel(context.Background())
	productorAntesConsumidor := &productorTestimonioPrueba{configuracion: configuracion}
	verificadorAntesConsumidor := &verificadorTestimonioPrueba{configuracion: configuracion}
	raizAntesConsumidor := &verificadorTestimonioPrueba{configuracion: configuracion}
	consumidorCancelador := &consumidorNominalNoAutoritativoPrueba{
		cancelarAlFinal: cancelarConsumidor,
	}
	if err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		ctxConsumidor, solicitudConsumidor,
		nuevaFronteraIdentidadInternaPrueba(
			configuracion, solicitudConsumidor, identidadInternaEstablePrueba(1),
		),
		productorAntesConsumidor, verificadorAntesConsumidor,
		&raizVerificadorTestimonioPrueba{delegado: raizAntesConsumidor}, consumidorCancelador,
	); !errors.Is(err, context.Canceled) || productorAntesConsumidor.llamadas != 1 ||
		verificadorAntesConsumidor.llamadas != 1 || raizAntesConsumidor.llamadas != 1 ||
		consumidorCancelador.llamadas != 1 {
		t.Fatalf("cancelacion de consumidor no preservada: err=%v productor=%d verificador=%d raiz=%d consumidor=%d",
			err, productorAntesConsumidor.llamadas, verificadorAntesConsumidor.llamadas,
			raizAntesConsumidor.llamadas, consumidorCancelador.llamadas)
	}
}

func TestSentinelContextoFingidoPorAdaptadorSeEnmascaraConContextoVivo(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	t.Run("frontera", func(t *testing.T) {
		solicitud := solicitudTestimonioPrueba(t)
		frontera := nuevaFronteraIdentidadInternaPrueba(
			configuracion, solicitud, identidadInternaEstablePrueba(1),
		)
		frontera.errorAlFinal = context.Canceled
		productor := &productorTestimonioPrueba{configuracion: configuracion}
		err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitud, frontera, productor,
			&verificadorTestimonioPrueba{configuracion: configuracion},
			&raizVerificadorTestimonioPrueba{
				delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
			},
			&consumidorNominalNoAutoritativoPrueba{},
		)
		if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) ||
			errors.Is(err, context.Canceled) || frontera.llamadas != 1 || productor.llamadas != 0 {
			t.Fatalf("sentinel fingido por frontera no fallo cerrado: err=%v frontera=%d productor=%d",
				err, frontera.llamadas, productor.llamadas)
		}
	})

	t.Run("productor", func(t *testing.T) {
		solicitud := solicitudTestimonioPrueba(t)
		productor := &productorTestimonioPrueba{
			configuracion: configuracion, errorAlFinal: context.DeadlineExceeded,
		}
		verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
		err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitud,
			nuevaFronteraIdentidadInternaPrueba(
				configuracion, solicitud, identidadInternaEstablePrueba(1),
			),
			productor, verificador,
			&raizVerificadorTestimonioPrueba{
				delegado: &verificadorTestimonioPrueba{configuracion: configuracion},
			},
			&consumidorNominalNoAutoritativoPrueba{},
		)
		if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) ||
			errors.Is(err, context.DeadlineExceeded) || productor.llamadas != 1 || verificador.llamadas != 0 {
			t.Fatalf("sentinel fingido por productor no fallo cerrado: err=%v productor=%d verificador=%d",
				err, productor.llamadas, verificador.llamadas)
		}
	})

	for _, caso := range []struct {
		nombre                                   string
		enRaiz, enConsumidor                     bool
		errorFingido                             error
		productorDebe, verificadorDebe, raizDebe int
		consumidorDebe                           int
	}{
		{
			nombre: "verificador", errorFingido: context.Canceled,
			productorDebe: 1, verificadorDebe: 1,
		},
		{
			nombre: "raiz", enRaiz: true, errorFingido: context.DeadlineExceeded,
			productorDebe: 1, verificadorDebe: 1, raizDebe: 1,
		},
		{
			nombre: "consumidor", enConsumidor: true, errorFingido: context.Canceled,
			productorDebe: 1, verificadorDebe: 1, raizDebe: 1, consumidorDebe: 1,
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := solicitudTestimonioPrueba(t)
			productor := &productorTestimonioPrueba{configuracion: configuracion}
			verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
			raiz := &verificadorTestimonioPrueba{configuracion: configuracion}
			consumidor := &consumidorNominalNoAutoritativoPrueba{}
			if caso.enRaiz {
				raiz.errorAlFinal = caso.errorFingido
			} else if caso.enConsumidor {
				consumidor.errorAlFinal = caso.errorFingido
			} else {
				verificador.errorAlFinal = caso.errorFingido
			}
			err := ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
				context.Background(), solicitud,
				nuevaFronteraIdentidadInternaPrueba(
					configuracion, solicitud, identidadInternaEstablePrueba(1),
				),
				productor, verificador, &raizVerificadorTestimonioPrueba{delegado: raiz}, consumidor,
			)
			if !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) ||
				errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
				productor.llamadas != caso.productorDebe ||
				verificador.llamadas != caso.verificadorDebe || raiz.llamadas != caso.raizDebe ||
				consumidor.llamadas != caso.consumidorDebe {
				t.Fatalf("sentinel fingido por %s no fallo en su fase: err=%v productor=%d verificador=%d raiz=%d consumidor=%d",
					caso.nombre, err, productor.llamadas, verificador.llamadas,
					raiz.llamadas, consumidor.llamadas)
			}
		})
	}
}

func TestSuperficiePublicaNoExponeBypassesNiAfirmaTCB(t *testing.T) {
	for _, valor := range []any{
		SolicitudTestimonioAtomicoIdempotenciaBaremacion{},
		ClaveClienteIdempotenciaBaremacion{},
		MaterialCanonicoEfimeroBaremacion{},
		ReferenciaGeneracionClaveHMACNominalBaremacion{},
		SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion{},
	} {
		tipo := reflect.TypeOf(valor)
		for posicion := 0; posicion < tipo.NumField(); posicion++ {
			if tipo.Field(posicion).PkgPath == "" {
				t.Errorf("campo autoritativo publico en %s.%s", tipo.Name(), tipo.Field(posicion).Name)
			}
		}
	}
	contenido, err := os.ReadFile("idempotencia_semantica_baremacion.go")
	if err != nil {
		t.Fatalf("leer fuente para prueba de superficie: %v", err)
	}
	texto := string(contenido)
	for _, definicion := range []string{
		"type IndiceIdempotenciaBaremacion ",
		"type IndiceIdempotenciaBaremacionComprobadoTCB ",
		"type PrincipalEstableBaremacionHMAC ",
		"type CandidatosPrincipalEstableBaremacion ",
		"type ResultadoDerivacionMatrizIndiceIdempotenciaBaremacion ",
		"type ProductoNominalNoAutoritativoIdempotenciaBaremacion ",
		"func ConstruirProductoNominalNoAutoritativoIdempotenciaBaremacion(",
		"RevisarYConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(",
		"func NuevaIntencionCambioBaremacionComprobadaTCB(",
		"func NuevaSolicitudVerificarSeparacionDominiosClaveBaremacion(",
	} {
		if strings.Contains(texto, definicion) {
			t.Errorf("API antigua insegura sigue definida: %s", definicion)
		}
	}
}

func exigirProteccionValorPrueba(t *testing.T, valor any, secretos ...string) {
	t.Helper()
	representacion := fmt.Sprintf("%v|%#v|%s|%q", valor, valor, valor, valor)
	for _, secreto := range secretos {
		if secreto != "" && strings.Contains(representacion, secreto) {
			t.Errorf("formateo filtro secreto %q: %s", secreto, representacion)
		}
	}
	if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionIdempotenciaBaremacion) {
		t.Errorf("JSON no fue denegado para %T: %v", valor, err)
	}
	if serializador, ok := valor.(encoding.TextMarshaler); ok {
		if _, err := serializador.MarshalText(); !errors.Is(err, ErrSerializacionIdempotenciaBaremacion) {
			t.Errorf("texto no fue denegado para %T: %v", valor, err)
		}
	}
	if serializador, ok := valor.(encoding.BinaryMarshaler); ok {
		if _, err := serializador.MarshalBinary(); !errors.Is(err, ErrSerializacionIdempotenciaBaremacion) {
			t.Errorf("binario no fue denegado para %T: %v", valor, err)
		}
	}
	if registrable, ok := valor.(slog.LogValuer); ok {
		registro := registrable.LogValue().String()
		for _, secreto := range secretos {
			if secreto != "" && strings.Contains(registro, secreto) {
				t.Errorf("LogValue filtro secreto %q: %s", secreto, registro)
			}
		}
	}
}

func TestSolicitudesReferenciasSnapshotsYEvidenciasEstanProtegidas(t *testing.T) {
	solicitud := solicitudTestimonioPrueba(t)
	materialClave := copiarMaterialClaveClientePrueba(solicitud.claveCliente)
	defer borrarBytesBaremacion(materialClave)
	exigirProteccionValorPrueba(t, solicitud, "despliegue-ficticio-v1")
	exigirProteccionValorPrueba(t, solicitud.claveCliente, hex.EncodeToString(materialClave))
	resolver, err := NuevaSolicitudResolverSeudonimoSujetoBaremacion(
		"123e4567-e89b-42d3-a456-426614174000",
		"223e4567-e89b-42d3-a456-426614174000",
		"323e4567-e89b-42d3-a456-426614174000",
	)
	if err != nil {
		t.Fatalf("crear solicitud resolver ficticia: %v", err)
	}
	exigirProteccionValorPrueba(t, resolver, "123e4567-e89b-42d3-a456-426614174000")
	referenciasSeparacion := referenciasSeisDominiosSeparacionPrueba(t)
	separacion, err := ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(
		referenciasSeparacion,
	)
	if err != nil {
		t.Fatalf("crear separacion ficticia: %v", err)
	}
	exigirProteccionValorPrueba(t, separacion, "sujeto-ficticio-v1")
	exigirProteccionValorPrueba(t, referenciasSeparacion[2], "sujeto-ficticio-v1")
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	instantanea := instantaneaLlaveroHMACBaremacion{
		Dominio:    DominioClavePrincipalBaremacion,
		LlaveroRef: configuracion.identidadesRef, Revision: configuracion.identidadesRevision,
		Cantidad: 1, Topologia: configuracion.identidades,
	}
	instantanea.HuellaSHA256 = huellaInstantaneaPrueba(
		instantanea.Dominio, instantanea.LlaveroRef, instantanea.Revision, instantanea.Topologia,
	)
	exigirProteccionValorPrueba(t, instantanea, configuracion.identidadesRef)
	evidencia := evidenciaAtestacionIdempotenciaBaremacion{
		Formato: "hmac-sha256-v1", EmisorRef: "hsm-ficticio",
		ClaveAtestacionRef: "atestacion-ficticia-v1", Revision: 1,
		HuellaContenidoSHA256: huellaTextoPrueba("contenido"), Valor: []byte("evidencia-ficticia-no-secreta"),
	}
	exigirProteccionValorPrueba(t, evidencia, "evidencia-ficticia-no-secreta")
	resolucion := instantaneaResolucionIdentidadInternaEstableBaremacion{
		SnapshotRef: "snapshot-identidad-ficticio-v1", Revision: 1,
		HuellaSHA256: huellaTextoPrueba("snapshot"), FormatoAtestacion: "hmac-sha256-v1",
		EmisorAtestacionRef: "identidad-ficticia", ClaveAtestacionRef: "clave-identidad-ficticia",
		ValorAtestacion: []byte("atestacion-identidad-ficticia"),
	}
	exigirProteccionValorPrueba(t, resolucion, resolucion.SnapshotRef, string(resolucion.ValorAtestacion))
	receptorResolucion := &receptorResolucionIdentidadInternaEstableBaremacion{
		lote: loteResolucionIdentidadInternaEstableBaremacion{
			ancla: identidadInternaEstablePrueba(1), instantanea: resolucion,
		},
	}
	exigirProteccionValorPrueba(t, receptorResolucion, string(receptorResolucion.lote.ancla))
}

func TestSeparacionDominiosSoloPermiteVisitaSinGetter(t *testing.T) {
	solicitud, err := ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(
		referenciasSeisDominiosSeparacionPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	visitadas := 0
	if err := solicitud.VisitarReferencias(func(
		dominio DominioClaveHMACBaremacion, generacion uint32, referencia string,
	) error {
		if !dominio.Valido() || generacion == 0 || referencia == "" {
			return errors.New("uso invalido")
		}
		visitadas++
		return nil
	}); err != nil || visitadas != 6 {
		t.Fatalf("visita controlada fallo: visitadas=%d err=%v", visitadas, err)
	}
	if _, existe := reflect.TypeOf(solicitud).MethodByName("Referencias"); existe {
		t.Fatal("existe getter publico Referencias")
	}
}

type contextoNuloPrueba struct{}

func (*contextoNuloPrueba) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*contextoNuloPrueba) Done() <-chan struct{}       { return nil }
func (*contextoNuloPrueba) Err() error                  { return nil }
func (*contextoNuloPrueba) Value(any) any               { return nil }

func TestContextoTypedNilFallaSinPanico(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	var ctx *contextoNuloPrueba
	if _, err := construirProductoNominalPrueba(t,
		ctx, solicitudTestimonioPrueba(t),
		&productorTestimonioPrueba{configuracion: configuracion},
		&verificadorTestimonioPrueba{configuracion: configuracion},
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("contexto typed nil aceptado: %v", err)
	}
}

func intencionCambioBaremacionValidaPrueba() IntencionCambioBaremacion {
	return IntencionCambioBaremacion{
		Version: VersionIntencionCambioBaremacionV1, Clase: ClaseCambioIncorporarDecision,
		ProcesoRef: referenciaMaterialOpacaPrueba(1), SolicitudRef: referenciaMaterialOpacaPrueba(2),
		SujetoSeudonimoHMAC: SeudonimoSujetoBaremacionHMAC{
			Version: VersionSeudonimoSujetoBaremacionV1, ClaveHMACRef: "seudonimo-sujeto-v1",
			ValorHMAC: huellaIdempotenciaPruebaID(1),
		},
		BaremacionMeritoRef: referenciaMaterialOpacaPrueba(3),
		VersionBase: ReferenciaVersionBaremacion{
			BaremacionMeritoRef: referenciaMaterialOpacaPrueba(3), Numero: 1,
			HuellaEstadoSHA256: huellaIdempotenciaPruebaID(2),
		},
		VersionObjetivo: 2,
		DecisionRef:     referenciaMaterialOpacaPrueba(4), NumeroDecision: 1,
		ClaseDecision: dominiobolsa.ClaseDecisionInicial, ResultadoDecision: dominiobolsa.ResultadoAceptado,
		HuellaContenidoDecisionSHA256:        huellaIdempotenciaPruebaID(3),
		HuellaEstadoResultanteDecisionSHA256: huellaIdempotenciaPruebaID(4),
		PoliticaFirmaRef:                     "politica-firma-1", PoliticaFirmaVersion: 3,
		HuellaPoliticaFirmaSHA256:    huellaIdempotenciaPruebaID(5),
		EsquemaPlanFirmaDurable:      EsquemaPlanFirmaDurableBaremacionV2,
		VersionPlanFirmaDurable:      VersionPlanFirmaDurableBaremacionV2,
		PlanFirmaDurableRef:          referenciaMaterialOpacaPrueba(5),
		HuellaPlanFirmaDurableSHA256: huellaIdempotenciaPruebaID(6),
		EstadoPlanFirmaDurable:       EstadoPlanFirmaDurableCompletado,
		DocumentoFirmableRef:         referenciaMaterialOpacaPrueba(6), VersionDocumentoFirmable: "v1",
		HuellaDocumentoFirmableSHA256: huellaIdempotenciaPruebaID(7),
		FirmaRef:                      referenciaMaterialOpacaPrueba(7), HuellaFirmaSHA256: huellaIdempotenciaPruebaID(8),
		DocumentoFirmadoRef:              referenciaMaterialOpacaPrueba(8),
		HuellaDocumentoFirmadoSHA256:     huellaIdempotenciaPruebaID(9),
		EsquemaManifiestoProbatorio:      EsquemaManifiestoMaterialEstableBaremacionV2,
		VersionManifiestoProbatorio:      VersionManifiestoMaterialEstableBaremacionV2,
		ManifiestoProbatorioRef:          referenciaMaterialOpacaPrueba(9),
		HuellaManifiestoProbatorioSHA256: huellaIdempotenciaPruebaID(10),
		SelloManifiestoProbatorioHMAC: HMACManifiestoMaterialBaremacionV2{
			Version: VersionHMACManifiestoMaterialBaremacionV2, ClaveHMACRef: "manifiesto-v2",
			ValorHMAC: huellaIdempotenciaPruebaID(11),
		},
		ObjetoCustodiadoRef: referenciaMaterialOpacaPrueba(10), VersionObjetoCustodiado: "version-7",
		ConectorCustodiaID: "almacen-s3-local", ZonaCustodia: puertosvec.ZonaAlmacenAdmitida,
		HuellaObjetoCustodiadoSHA256: huellaIdempotenciaPruebaID(9),
		FormatoDocumento: InstantaneaCatalogoFormatoDocumentoBaremacion{
			CatalogoRef: "catalogo-formatos-firma", CatalogoVersion: 2,
			HuellaCatalogoSHA256: huellaIdempotenciaPruebaID(18),
			FormatoClave:         "pdf_pades", MIMECanonico: "application/pdf",
		},
		ClasificacionDocumento: InstantaneaCatalogoClasificacionDocumentoBaremacion{
			CatalogoRef: "catalogo-clasificacion-documental", CatalogoVersion: 3,
			HuellaCatalogoSHA256: huellaIdempotenciaPruebaID(19),
			ClasificacionClave:   "datos_personales_alta",
		},
		TamanoDocumentoFirmado:            4096,
		EstadoInmovilizacionObjeto:        EstadoInmovilizacionNoAplicada,
		EstadoDisponibilidadObjeto:        EstadoDisponibilidadObjetoActivoNoEliminado,
		EsquemaEvidenciaRecuperacion:      EsquemaReciboRecuperacionBaremacionV2,
		VersionEvidenciaRecuperacion:      VersionReciboRecuperacionBaremacionV2,
		EvidenciaRecuperacionFirmadoRef:   referenciaMaterialOpacaPrueba(11),
		HuellaEvidenciaRecuperacionSHA256: huellaIdempotenciaPruebaID(12),
		EsquemaEvidenciaCustodia:          EsquemaReciboCustodiaBaremacionV2,
		VersionEvidenciaCustodia:          VersionReciboCustodiaBaremacionV2,
		EvidenciaCustodiaFirmadoRef:       referenciaMaterialOpacaPrueba(12),
		HuellaEvidenciaCustodiaSHA256:     huellaIdempotenciaPruebaID(13),
		EsquemaEvidenciaRetencion:         EsquemaReciboRetencionBaremacionV2,
		VersionEvidenciaRetencion:         VersionReciboRetencionBaremacionV2,
		EvidenciaRetencionFirmadoRef:      referenciaMaterialOpacaPrueba(13),
		HuellaEvidenciaRetencionSHA256:    huellaIdempotenciaPruebaID(14),
		PoliticaRetencionRef:              "politica-retencion-1",
		PoliticaRetencionVersion:          4,
		HuellaPoliticaRetencionSHA256:     huellaIdempotenciaPruebaID(15),
		RetenidoHasta:                     time.Date(2040, 1, 2, 3, 4, 5, 6000, time.UTC),
		HuellaAgregadoObjetivoSHA256:      huellaIdempotenciaPruebaID(16),
		MotivoClave:                       "merito_acreditado",
		MotivoHMAC: HMACMotivoBaremacion{
			Version: VersionHMACMotivoBaremacionV1, ClaveHMACRef: "motivo-baremacion-v1",
			ValorHMAC: huellaIdempotenciaPruebaID(17),
		},
	}
}

func intencionCambioBaremacionParaSolicitudPrueba(
	t *testing.T,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) IntencionCambioBaremacion {
	t.Helper()
	intencion := intencionCambioBaremacionValidaPrueba()
	if err := solicitud.VisitarAmbitoSujetoBaremacion(func(
		procesoRef, solicitudRef, baremacionMeritoRef string,
		versionSeudonimo uint16, claveSeudonimoRef, valorSeudonimoHMAC string,
	) error {
		intencion.ProcesoRef = procesoRef
		intencion.SolicitudRef = solicitudRef
		intencion.BaremacionMeritoRef = baremacionMeritoRef
		intencion.VersionBase.BaremacionMeritoRef = baremacionMeritoRef
		intencion.SujetoSeudonimoHMAC = SeudonimoSujetoBaremacionHMAC{
			Version:      versionSeudonimo,
			ClaveHMACRef: claveSeudonimoRef,
			ValorHMAC:    valorSeudonimoHMAC,
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := intencion.Validar(); err != nil {
		t.Fatalf("intencion ligada a solicitud invalida: %v", err)
	}
	return intencion
}

func huellaIdempotenciaPrueba(caracter string) string { return strings.Repeat(caracter, 64) }

func huellaIdempotenciaPruebaID(identificador uint64) string {
	return fmt.Sprintf("%064x", identificador)
}

func referenciaMaterialOpacaPrueba(identificador uint64) string {
	return fmt.Sprintf("f47ac10b-58cc-4372-a567-%012x", identificador)
}

func indiceIdempotenciaValidoPrueba() indiceIdempotenciaBaremacion {
	return indiceIdempotenciaBaremacion{
		Version: VersionIndiceIdempotenciaBaremacionV1, GeneracionClave: 1,
		ClaveHMACRef: "indice-baremacion-v1", ValorHMAC: huellaIdempotenciaPruebaID(90),
	}
}

func compararRepresentacionesDistintas(t *testing.T, primera, segunda IntencionCambioBaremacion) {
	t.Helper()
	indice := indiceIdempotenciaValidoPrueba()
	a, err := primera.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		t.Fatalf("primera intencion invalida: %v", err)
	}
	b, err := segunda.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		t.Fatalf("segunda intencion invalida: %v", err)
	}
	defer destruirCargaProtegidaBaremacion(&a)
	defer destruirCargaProtegidaBaremacion(&b)
	if cargasProtegidasIgualesPrueba(a, b) {
		t.Fatal("dos intenciones materiales distintas colisionaron")
	}
}

func leerCamposCanonicosIntencion(t *testing.T, material []byte, cantidad uint16) [][]byte {
	t.Helper()
	campos := make([][]byte, int(cantidad)+1)
	resto := material
	for esperada := uint16(0); esperada <= cantidad; esperada++ {
		if len(resto) < 10 || binary.BigEndian.Uint16(resto[:2]) != esperada {
			t.Fatalf("cabecera/orden invalido en etiqueta %d", esperada)
		}
		longitud := binary.BigEndian.Uint64(resto[2:10])
		resto = resto[10:]
		if longitud > uint64(len(resto)) {
			t.Fatalf("longitud fuera de limites en etiqueta %d", esperada)
		}
		campos[int(esperada)] = append([]byte(nil), resto[:int(longitud)]...)
		resto = resto[int(longitud):]
	}
	if len(resto) != 0 || string(campos[0]) != esquemaCanonicoIntencionCambioBaremacionV1 {
		t.Fatalf("dominio o bytes finales invalidos: dominio=%q resto=%d", campos[0], len(resto))
	}
	return campos
}

func TestTiposPersistiblesDeIdempotenciaSonEstricosYVersionados(t *testing.T) {
	for _, estado := range []EstadoOperacionIdempotenteBaremacion{
		EstadoOperacionIdempotenteAusente, EstadoOperacionIdempotenteEnCurso,
		EstadoOperacionIdempotenteConfirmada,
	} {
		if !estado.Valido() {
			t.Fatalf("estado conocido rechazado: %q", estado)
		}
	}
	for _, estado := range []EstadoOperacionIdempotenteBaremacion{"", "nueva", "pendiente", "CONFIRMADA"} {
		if estado.Valido() {
			t.Fatalf("estado desconocido aceptado: %q", estado)
		}
	}
	seudonimo := SeudonimoSujetoBaremacionHMAC{
		Version: VersionSeudonimoSujetoBaremacionV1, ClaveHMACRef: "seudonimo-sujeto-v1",
		ValorHMAC: huellaIdempotenciaPrueba("7"),
	}
	clonSeudonimo, err := seudonimo.Clonar()
	if err != nil || !seudonimo.IgualConstante(clonSeudonimo) {
		t.Fatalf("seudonimo valido no clonable: %v", err)
	}
	sello := HMACIntencionCambioBaremacion{
		Version: VersionHMACIntencionCambioBaremacionV1, ClaveHMACRef: "intencion-v1",
		ValorHMAC: huellaIdempotenciaPrueba("4"),
	}
	clonSello, err := sello.Clonar()
	if err != nil || !sello.IgualConstante(clonSello) {
		t.Fatalf("sello valido no clonable: %v", err)
	}
	motivo := HMACMotivoBaremacion{
		Version: VersionHMACMotivoBaremacionV1, ClaveHMACRef: "motivo-baremacion-v1",
		ValorHMAC: huellaIdempotenciaPrueba("6"),
	}
	clonMotivo, err := motivo.Clonar()
	if err != nil || !motivo.IgualConstante(clonMotivo) {
		t.Fatalf("motivo valido no clonable: %v", err)
	}
	for nombre, comprobar := range map[string]func() bool{
		"seudonimo version futura": func() bool { c := seudonimo; c.Version = 2; return c.Validar() == nil },
		"seudonimo DNI":            func() bool { c := seudonimo; c.ValorHMAC = "12345678Z"; return c.Validar() == nil },
		"sello sin clave":          func() bool { c := sello; c.ClaveHMACRef = ""; return c.Validar() == nil },
		"sello no HMAC":            func() bool { c := sello; c.ValorHMAC = "no-hmac"; return c.Validar() == nil },
		"motivo libre":             func() bool { c := motivo; c.ClaveHMACRef = "motivo libre"; return c.Validar() == nil },
		"motivo predecible":        func() bool { c := motivo; c.ValorHMAC = "documentacion_incorrecta"; return c.Validar() == nil },
	} {
		t.Run(nombre, func(t *testing.T) {
			if comprobar() {
				t.Fatal("valor invalido aceptado")
			}
		})
	}
}

func TestPreimagenIntencionQuedaLigadaAlIndicePrivadoCompleto(t *testing.T) {
	intencion := intencionCambioBaremacionValidaPrueba()
	indice := indiceIdempotenciaValidoPrueba()
	base, err := intencion.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		t.Fatalf("preimagen base: %v", err)
	}
	defer destruirCargaProtegidaBaremacion(&base)
	mutaciones := []indiceIdempotenciaBaremacion{indice, indice, indice, indice}
	mutaciones[0].Version = 2
	mutaciones[1].GeneracionClave = 2
	mutaciones[2].ClaveHMACRef = "indice-baremacion-v2"
	mutaciones[3].ValorHMAC = huellaIdempotenciaPrueba("8")
	for posicion, mutada := range mutaciones {
		preimagen, err := intencion.representacionCanonicaSobreProbatorioParaHMAC(mutada)
		if posicion == 0 {
			if err == nil {
				t.Fatal("version de indice no admitida aceptada")
			}
			continue
		}
		iguales := err == nil && cargasProtegidasIgualesPrueba(base, preimagen)
		destruirCargaProtegidaBaremacion(&preimagen)
		if err != nil || iguales {
			t.Fatalf("campo %d del indice no quedo ligado: %v", posicion, err)
		}
	}
	reutilizada := indice
	reutilizada.ClaveHMACRef = intencion.SujetoSeudonimoHMAC.ClaveHMACRef
	if _, err := intencion.representacionCanonicaSobreProbatorioParaHMAC(reutilizada); err == nil {
		t.Fatal("la clave de seudonimo se reutilizo para indice")
	}
}

func TestFingerprintSemanticoNoCambiaAlRotarSobresYElSobreProbatorioSi(t *testing.T) {
	base := intencionCambioBaremacionValidaPrueba()
	indice := indiceIdempotenciaValidoPrueba()
	motivo, err := NuevaCargaProtegida([]byte("contenido exacto del motivo verificado"))
	if err != nil {
		t.Fatal(err)
	}
	defer destruirCargaProtegidaBaremacion(&motivo)
	fingerprintBase, err := base.representacionCanonicaFingerprintSemanticoParaHMAC(indice, motivo)
	if err != nil {
		t.Fatal(err)
	}
	defer destruirCargaProtegidaBaremacion(&fingerprintBase)
	sobreBase, err := base.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		t.Fatal(err)
	}
	defer destruirCargaProtegidaBaremacion(&sobreBase)
	rotada := base
	rotada.SujetoSeudonimoHMAC.ClaveHMACRef = "seudonimo-sujeto-v2"
	rotada.SujetoSeudonimoHMAC.ValorHMAC = huellaIdempotenciaPruebaID(301)
	rotada.SelloManifiestoProbatorioHMAC.ClaveHMACRef = "manifiesto-baremacion-v3"
	rotada.SelloManifiestoProbatorioHMAC.ValorHMAC = huellaIdempotenciaPruebaID(302)
	rotada.MotivoHMAC.ClaveHMACRef = "motivo-baremacion-v4"
	rotada.MotivoHMAC.ValorHMAC = huellaIdempotenciaPruebaID(303)
	if err := rotada.Validar(); err != nil {
		t.Fatal(err)
	}
	fingerprintRotado, err := rotada.representacionCanonicaFingerprintSemanticoParaHMAC(indice, motivo)
	defer destruirCargaProtegidaBaremacion(&fingerprintRotado)
	if err != nil || !cargasProtegidasIgualesPrueba(fingerprintBase, fingerprintRotado) {
		t.Fatalf("rotar sobres produjo falso conflicto semantico: %v", err)
	}
	sobreRotado, err := rotada.representacionCanonicaSobreProbatorioParaHMAC(indice)
	defer destruirCargaProtegidaBaremacion(&sobreRotado)
	if err != nil || cargasProtegidasIgualesPrueba(sobreBase, sobreRotado) {
		t.Fatalf("auditoria no conservo sobres rotados: %v", err)
	}
	motivoDistinto, _ := NuevaCargaProtegida([]byte("contenido distinto del motivo verificado"))
	defer destruirCargaProtegidaBaremacion(&motivoDistinto)
	fingerprintMotivoDistinto, err := rotada.representacionCanonicaFingerprintSemanticoParaHMAC(
		indice, motivoDistinto,
	)
	defer destruirCargaProtegidaBaremacion(&fingerprintMotivoDistinto)
	if err != nil || cargasProtegidasIgualesPrueba(fingerprintBase, fingerprintMotivoDistinto) {
		t.Fatalf("cambiar el motivo claro no cambio fingerprint: %v", err)
	}
	negocioDistinto := rotada
	negocioDistinto.HuellaContenidoDecisionSHA256 = huellaIdempotenciaPruebaID(304)
	fingerprintNegocioDistinto, err := negocioDistinto.representacionCanonicaFingerprintSemanticoParaHMAC(
		indice, motivo,
	)
	defer destruirCargaProtegidaBaremacion(&fingerprintNegocioDistinto)
	if err != nil || cargasProtegidasIgualesPrueba(fingerprintBase, fingerprintNegocioDistinto) {
		t.Fatalf("cambiar contenido de negocio no cambio fingerprint: %v", err)
	}
	indiceOtroSujeto := indice
	indiceOtroSujeto.ValorHMAC = huellaIdempotenciaPruebaID(305)
	fingerprintOtroSujeto, err := base.representacionCanonicaFingerprintSemanticoParaHMAC(
		indiceOtroSujeto, motivo,
	)
	defer destruirCargaProtegidaBaremacion(&fingerprintOtroSujeto)
	if err != nil || cargasProtegidasIgualesPrueba(fingerprintBase, fingerprintOtroSujeto) {
		t.Fatalf("otro sujeto/indice no cambio fingerprint: %v", err)
	}
}

func TestVistaConstruyeRepresentacionesDeTodaLaMatrizSinExponerCelda(t *testing.T) {
	configuracion := nuevaConfiguracionTestimonioPrueba(2, 2)
	solicitudProducto := solicitudTestimonioPrueba(t)
	producto, err := construirProductoNominalPrueba(
		t, context.Background(), solicitudProducto,
		&productorTestimonioPrueba{configuracion: configuracion},
		&verificadorTestimonioPrueba{configuracion: configuracion},
	)
	if err != nil {
		t.Fatal(err)
	}
	clon, err := producto.testimonio.clonar()
	if err != nil {
		t.Fatal(err)
	}
	vista := nuevaVistaEfimeraTestimonioIdempotenciaBaremacion(clon)
	defer vista.cerrarYComprobarSinActividad()
	// El flujo one-shot destruyo el propietario de la clave anterior. Esta
	// solicitud fresca reproduce el mismo vinculo semantico con otro propietario.
	solicitud := solicitudTestimonioPrueba(t)
	intencion := intencionCambioBaremacionParaSolicitudPrueba(t, solicitud)
	motivo := []byte("motivo exacto verificado por clave historica")
	motivoProtegido, err := NuevaCargaProtegida(motivo)
	if err != nil {
		t.Fatal(err)
	}
	defer destruirCargaProtegidaBaremacion(&motivoProtegido)
	visitadas := 0
	var fingerprintRetenido, sobreRetenido MaterialCanonicoEfimeroBaremacion
	var aliasFingerprint, aliasSobre []byte
	err = vista.VisitarRepresentacionesCanonicasIntencionBaremacion(
		solicitud, intencion, motivo,
		func(
			fila, columna int,
			fingerprint, sobre MaterialCanonicoEfimeroBaremacion,
		) error {
			esperadoFingerprint, err := intencion.representacionCanonicaFingerprintSemanticoParaHMAC(
				producto.testimonio.matriz[fila][columna], motivoProtegido,
			)
			if err != nil {
				return err
			}
			defer destruirCargaProtegidaBaremacion(&esperadoFingerprint)
			esperadoSobre, err := intencion.representacionCanonicaSobreProbatorioParaHMAC(
				producto.testimonio.matriz[fila][columna],
			)
			if err != nil {
				return err
			}
			defer destruirCargaProtegidaBaremacion(&esperadoSobre)
			bytesFingerprintEsperado := esperadoFingerprint.Revelar()
			defer borrarBytesBaremacion(bytesFingerprintEsperado)
			bytesSobreEsperado := esperadoSobre.Revelar()
			defer borrarBytesBaremacion(bytesSobreEsperado)
			var coincideFingerprint, coincideSobre bool
			if err := fingerprint.VisitarBytes(func(valor []byte) error {
				aliasFingerprint = valor
				coincideFingerprint = bytes.Equal(valor, bytesFingerprintEsperado)
				return nil
			}); err != nil {
				return err
			}
			if err := sobre.VisitarBytes(func(valor []byte) error {
				aliasSobre = valor
				coincideSobre = bytes.Equal(valor, bytesSobreEsperado)
				return nil
			}); err != nil {
				return err
			}
			if !coincideFingerprint || !coincideSobre {
				return errors.New("representaciones de la celda no corresponden")
			}
			fingerprintRetenido, sobreRetenido = fingerprint, sobre
			visitadas++
			return nil
		},
	)
	if err != nil || visitadas != 4 || vista.cobertura&coberturaMatrizBaremacion == 0 {
		t.Fatalf("visita matricial nominal incompleta: visitadas=%d cobertura=%x err=%v",
			visitadas, vista.cobertura, err)
	}
	for nombre, material := range map[string]MaterialCanonicoEfimeroBaremacion{
		"fingerprint": fingerprintRetenido,
		"sobre":       sobreRetenido,
	} {
		if material.Validar() == nil || material.VisitarBytes(func([]byte) error { return nil }) == nil {
			t.Fatalf("%s retenido siguio util tras el callback", nombre)
		}
	}
	for nombre, alias := range map[string][]byte{"fingerprint": aliasFingerprint, "sobre": aliasSobre} {
		if !bytes.Equal(alias, make([]byte, len(alias))) {
			t.Fatalf("%s retenido conservo bytes tras el callback", nombre)
		}
	}
	cruzada := intencion
	cruzada.SolicitudRef = referenciaMaterialOpacaPrueba(9991)
	if err := vista.VisitarRepresentacionesCanonicasIntencionBaremacion(
		solicitud, cruzada, motivo,
		func(int, int, MaterialCanonicoEfimeroBaremacion, MaterialCanonicoEfimeroBaremacion) error {
			return nil
		},
	); !errors.Is(err, ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("intencion de otra solicitud aceptada por la vista: %v", err)
	}
}

func TestSeparacionCompletaDeSeisDominiosEsConjuntaYSinAlias(t *testing.T) {
	referencias := make([]ReferenciaGeneracionClaveHMACNominalBaremacion, 0, 20)
	for generacion := uint32(1); generacion <= 8; generacion++ {
		referencias = append(referencias,
			referenciaGeneracionSeparacionPrueba(
				t, DominioClavePrincipalBaremacion, generacion,
				fmt.Sprintf("principal-v%d", generacion),
			),
			referenciaGeneracionSeparacionPrueba(
				t, DominioClaveIndiceBaremacion, generacion,
				fmt.Sprintf("indice-v%d", generacion),
			),
		)
	}
	for _, entrada := range []struct {
		dominio DominioClaveHMACBaremacion
		ref     string
	}{
		{DominioClaveSujetoBaremacion, "sujeto-v1"},
		{DominioClaveMotivoBaremacion, "motivo-v1"},
		{DominioClaveManifiestoBaremacion, "manifiesto-v1"},
		{DominioClaveIntencionBaremacion, "intencion-v1"},
	} {
		referencias = append(referencias,
			referenciaGeneracionSeparacionPrueba(t, entrada.dominio, 1, entrada.ref),
		)
	}
	solicitud, err := ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(referencias)
	if err != nil {
		t.Fatalf("seis dominios con historicos rechazados: %v", err)
	}
	visitados := make(map[DominioClaveHMACBaremacion]int)
	ultimaGeneracion := make(map[DominioClaveHMACBaremacion]uint32)
	if err := solicitud.VisitarReferencias(func(
		dominio DominioClaveHMACBaremacion, generacion uint32, ref string,
	) error {
		if ref == "" || (ultimaGeneracion[dominio] != 0 && generacion >= ultimaGeneracion[dominio]) {
			return errors.New("orden canonico invalido")
		}
		visitados[dominio]++
		ultimaGeneracion[dominio] = generacion
		return nil
	}); err != nil || len(visitados) != 6 || visitados[DominioClavePrincipalBaremacion] != 8 ||
		visitados[DominioClaveIndiceBaremacion] != 8 {
		t.Fatalf("verificacion conjunta incompleta: dominios=%v err=%v", visitados, err)
	}
	for nombre, alterar := range map[string]func([]ReferenciaGeneracionClaveHMACNominalBaremacion){
		"falta-dominio": func(c []ReferenciaGeneracionClaveHMACNominalBaremacion) {
			c[len(c)-1] = ReferenciaGeneracionClaveHMACNominalBaremacion{}
		},
		"alias-repetido":      func(c []ReferenciaGeneracionClaveHMACNominalBaremacion) { c[len(c)-1].claveHMACRef = c[0].claveHMACRef },
		"generacion-repetida": func(c []ReferenciaGeneracionClaveHMACNominalBaremacion) { c[2].generacion = c[0].generacion },
	} {
		candidatos := append([]ReferenciaGeneracionClaveHMACNominalBaremacion(nil), referencias...)
		alterar(candidatos)
		if nombre == "falta-dominio" {
			candidatos = candidatos[:len(candidatos)-1]
		}
		if _, err := ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(candidatos); err == nil {
			t.Fatalf("caso invalido aceptado: %s", nombre)
		}
	}
	demasiadas := append([]ReferenciaGeneracionClaveHMACNominalBaremacion(nil), referencias...)
	demasiadas = append(demasiadas,
		referenciaGeneracionSeparacionPrueba(t, DominioClavePrincipalBaremacion, 9, "principal-v9"),
	)
	if _, err := ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(demasiadas); err == nil {
		t.Fatal("nueve generaciones de un dominio fueron aceptadas")
	}
	// No existe en este puerto nominal constructor ni solicitud publica de
	// sellado: la comprobacion fisica y cualquier efecto quedan para el servicio
	// de aplicacion de composicion privada pendiente.
	contenido, _ := os.ReadFile("idempotencia_semantica_baremacion.go")
	for _, prohibida := range []string{"type SolicitudSellarIntencionCambioBaremacion ", "type CriptografiaIdempotenciaBaremacion "} {
		if strings.Contains(string(contenido), prohibida) {
			t.Fatalf("efecto publico prematuro presente: %s", prohibida)
		}
	}
}

func TestIntencionCambioBaremacionExigeCadaCampoYSubcampo(t *testing.T) {
	valida := intencionCambioBaremacionValidaPrueba()
	if err := valida.Validar(); err != nil {
		t.Fatalf("fixture valida rechazada: %v", err)
	}
	tipo := reflect.TypeOf(valida)
	for posicion := 0; posicion < tipo.NumField(); posicion++ {
		campo := tipo.Field(posicion)
		t.Run("cero/"+campo.Name, func(t *testing.T) {
			candidata := valida
			valor := reflect.ValueOf(&candidata).Elem().FieldByIndex(campo.Index)
			valor.Set(reflect.Zero(valor.Type()))
			if candidata.Validar() == nil {
				t.Fatalf("campo obligatorio %s acepto cero", campo.Name)
			}
		})
	}
	for nombre, alterar := range map[string]func(*IntencionCambioBaremacion){
		"seudonimo/version":      func(i *IntencionCambioBaremacion) { i.SujetoSeudonimoHMAC.Version = 0 },
		"seudonimo/clave":        func(i *IntencionCambioBaremacion) { i.SujetoSeudonimoHMAC.ClaveHMACRef = "" },
		"seudonimo/valor":        func(i *IntencionCambioBaremacion) { i.SujetoSeudonimoHMAC.ValorHMAC = "" },
		"version/base-ref":       func(i *IntencionCambioBaremacion) { i.VersionBase.BaremacionMeritoRef = "" },
		"version/numero":         func(i *IntencionCambioBaremacion) { i.VersionBase.Numero = 0 },
		"version/huella":         func(i *IntencionCambioBaremacion) { i.VersionBase.HuellaEstadoSHA256 = "" },
		"formato/catalogo":       func(i *IntencionCambioBaremacion) { i.FormatoDocumento.CatalogoRef = "" },
		"formato/version":        func(i *IntencionCambioBaremacion) { i.FormatoDocumento.CatalogoVersion = 0 },
		"formato/huella":         func(i *IntencionCambioBaremacion) { i.FormatoDocumento.HuellaCatalogoSHA256 = "" },
		"formato/clave":          func(i *IntencionCambioBaremacion) { i.FormatoDocumento.FormatoClave = "" },
		"formato/mime":           func(i *IntencionCambioBaremacion) { i.FormatoDocumento.MIMECanonico = "" },
		"clasificacion/catalogo": func(i *IntencionCambioBaremacion) { i.ClasificacionDocumento.CatalogoRef = "" },
		"clasificacion/version":  func(i *IntencionCambioBaremacion) { i.ClasificacionDocumento.CatalogoVersion = 0 },
		"clasificacion/huella":   func(i *IntencionCambioBaremacion) { i.ClasificacionDocumento.HuellaCatalogoSHA256 = "" },
		"clasificacion/clave":    func(i *IntencionCambioBaremacion) { i.ClasificacionDocumento.ClasificacionClave = "" },
		"motivo/version":         func(i *IntencionCambioBaremacion) { i.MotivoHMAC.Version = 0 },
		"motivo/clave":           func(i *IntencionCambioBaremacion) { i.MotivoHMAC.ClaveHMACRef = "" },
		"motivo/valor":           func(i *IntencionCambioBaremacion) { i.MotivoHMAC.ValorHMAC = "" },
	} {
		t.Run(nombre, func(t *testing.T) {
			candidata := valida
			alterar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("subcampo obligatorio ausente aceptado")
			}
		})
	}
}

func TestIntencionDeniegaRelacionesMaterialesIncoherentes(t *testing.T) {
	base := intencionCambioBaremacionValidaPrueba()
	casos := map[string]func(*IntencionCambioBaremacion){
		"version futura":   func(i *IntencionCambioBaremacion) { i.Version = 2 },
		"alta no cubierta": func(i *IntencionCambioBaremacion) { i.Clase = ClaseCambioAltaBaremacion },
		"otra baremacion base": func(i *IntencionCambioBaremacion) {
			i.VersionBase.BaremacionMeritoRef = referenciaMaterialOpacaPrueba(99)
		},
		"salto version":   func(i *IntencionCambioBaremacion) { i.VersionObjetivo++ },
		"numero decision": func(i *IntencionCambioBaremacion) { i.NumeroDecision++ },
		"clase decision":  func(i *IntencionCambioBaremacion) { i.ClaseDecision = dominiobolsa.ClaseDecisionTecnica("otra") },
		"resultado": func(i *IntencionCambioBaremacion) {
			i.ResultadoDecision = dominiobolsa.ResultadoDecisionTecnica("otro")
		},
		"manifiesto V1":           func(i *IntencionCambioBaremacion) { i.VersionManifiestoProbatorio = 1 },
		"plan V1":                 func(i *IntencionCambioBaremacion) { i.VersionPlanFirmaDurable = 1 },
		"plan no completado":      func(i *IntencionCambioBaremacion) { i.EstadoPlanFirmaDurable = "en_curso" },
		"recibo custodia V1":      func(i *IntencionCambioBaremacion) { i.VersionEvidenciaCustodia = 1 },
		"recibo esquema cambiado": func(i *IntencionCambioBaremacion) { i.EsquemaEvidenciaRetencion = EsquemaReciboCustodiaBaremacionV2 },
		"clave motivo=sujeto":     func(i *IntencionCambioBaremacion) { i.MotivoHMAC.ClaveHMACRef = i.SujetoSeudonimoHMAC.ClaveHMACRef },
		"zona cuarentena":         func(i *IntencionCambioBaremacion) { i.ZonaCustodia = puertosvec.ZonaAlmacenCuarentena },
		"MIME no canonico":        func(i *IntencionCambioBaremacion) { i.FormatoDocumento.MIMECanonico = "Application/PDF" },
		"objeto distinto":         func(i *IntencionCambioBaremacion) { i.HuellaObjetoCustodiadoSHA256 = huellaIdempotenciaPrueba("c") },
		"recibo=documento":        func(i *IntencionCambioBaremacion) { i.HuellaEvidenciaCustodiaSHA256 = i.HuellaDocumentoFirmadoSHA256 },
		"recibos iguales":         func(i *IntencionCambioBaremacion) { i.HuellaEvidenciaRetencionSHA256 = i.HuellaEvidenciaCustodiaSHA256 },
		"eliminado":               func(i *IntencionCambioBaremacion) { i.EstadoDisponibilidadObjeto = "eliminado" },
		"fecha no UTC":            func(i *IntencionCambioBaremacion) { i.RetenidoHasta = time.Date(2040, 1, 2, 3, 4, 5, 6000, time.Local) },
		"fecha submicro":          func(i *IntencionCambioBaremacion) { i.RetenidoHasta = time.Date(2040, 1, 2, 3, 4, 5, 6001, time.UTC) },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			candidata := base
			alterar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("relacion incoherente aceptada")
			}
			if _, err := candidata.representacionCanonicaSobreProbatorioParaHMAC(indiceIdempotenciaValidoPrueba()); err == nil {
				t.Fatal("relacion incoherente serializada")
			}
		})
	}
}

func TestIntencionSoloAdmiteUUIDv4CanonicoEnReferenciasMateriales(t *testing.T) {
	type mutador func(*IntencionCambioBaremacion, string)
	referencias := map[string]mutador{
		"proceso":   func(i *IntencionCambioBaremacion, v string) { i.ProcesoRef = v },
		"solicitud": func(i *IntencionCambioBaremacion, v string) { i.SolicitudRef = v },
		"baremacion": func(i *IntencionCambioBaremacion, v string) {
			i.BaremacionMeritoRef, i.VersionBase.BaremacionMeritoRef = v, v
		},
		"decision":     func(i *IntencionCambioBaremacion, v string) { i.DecisionRef = v },
		"plan":         func(i *IntencionCambioBaremacion, v string) { i.PlanFirmaDurableRef = v },
		"firmable":     func(i *IntencionCambioBaremacion, v string) { i.DocumentoFirmableRef = v },
		"firma":        func(i *IntencionCambioBaremacion, v string) { i.FirmaRef = v },
		"firmado":      func(i *IntencionCambioBaremacion, v string) { i.DocumentoFirmadoRef = v },
		"manifiesto":   func(i *IntencionCambioBaremacion, v string) { i.ManifiestoProbatorioRef = v },
		"objeto":       func(i *IntencionCambioBaremacion, v string) { i.ObjetoCustodiadoRef = v },
		"recuperacion": func(i *IntencionCambioBaremacion, v string) { i.EvidenciaRecuperacionFirmadoRef = v },
		"custodia":     func(i *IntencionCambioBaremacion, v string) { i.EvidenciaCustodiaFirmadoRef = v },
		"retencion":    func(i *IntencionCambioBaremacion, v string) { i.EvidenciaRetencionFirmadoRef = v },
	}
	prohibidas := []string{"alberto-garcia", "12345678Z", "persona@example.invalid", "../objeto", "F47AC10B-58CC-4372-A567-000000000001", "f47ac10b-58cc-1372-a567-000000000001"}
	for nombre, mutar := range referencias {
		for _, valor := range prohibidas {
			candidata := intencionCambioBaremacionValidaPrueba()
			mutar(&candidata, valor)
			if candidata.Validar() == nil {
				t.Errorf("%s acepto referencia no opaca %q", nombre, valor)
			}
		}
	}
	if !referenciaMaterialOpacaBaremacionValida(referenciaMaterialOpacaPrueba(200), 512) {
		t.Fatal("UUIDv4 canonico rechazado")
	}
}

func TestMIMECanonicoEsGenericoYCatalogosHistoricosQuedanValidados(t *testing.T) {
	for _, valor := range []MIMECanonicoDocumentoBaremacion{
		"application/pdf", "text/plain", "image/svg+xml",
		"application/vnd.oasis.opendocument.text", "application/vnd.ejemplo.documento+json",
	} {
		if !valor.Valido() {
			t.Fatalf("MIME valido rechazado: %q", valor)
		}
	}
	for _, valor := range []MIMECanonicoDocumentoBaremacion{
		"", "application", "/pdf", "Application/PDF", "application/*",
		"application/pdf;version=1", "application/pd@f", "ápplication/pdf",
		MIMECanonicoDocumentoBaremacion("application/" + strings.Repeat("a", 128)),
	} {
		if valor.Valido() {
			t.Fatalf("MIME invalido aceptado: %q", valor)
		}
	}
	intencion := intencionCambioBaremacionValidaPrueba()
	intencion.FormatoDocumento.FormatoClave = "odt_xades"
	intencion.FormatoDocumento.MIMECanonico = "application/vnd.oasis.opendocument.text"
	if err := intencion.Validar(); err != nil {
		t.Fatalf("formato administrable no PDF rechazado: %v", err)
	}
	intencion.FormatoDocumento.CatalogoVersion = maximoVersionCatalogoIntencion + 1
	if intencion.Validar() == nil {
		t.Fatal("version de catalogo de formato fuera de rango aceptada")
	}
	intencion = intencionCambioBaremacionValidaPrueba()
	intencion.ClasificacionDocumento.CatalogoVersion = maximoVersionCatalogoIntencion + 1
	if intencion.Validar() == nil {
		t.Fatal("version de catalogo de clasificacion fuera de rango aceptada")
	}
}

func TestCadaCampoMaterialConAlternativaValidaCambiaLaPreimagen(t *testing.T) {
	base := intencionCambioBaremacionValidaPrueba()
	indice := indiceIdempotenciaValidoPrueba()
	preimagenBase, err := base.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		t.Fatalf("preimagen base: %v", err)
	}
	defer destruirCargaProtegidaBaremacion(&preimagenBase)
	type mutacion struct {
		campo, nombre string
		aplicar       func(*IntencionCambioBaremacion)
	}
	huella := func(id uint64) string { return huellaIdempotenciaPruebaID(100 + id) }
	mutarVersiones := func(i *IntencionCambioBaremacion) {
		i.VersionBase.Numero, i.VersionObjetivo, i.NumeroDecision = 2, 3, 2
	}
	mutarBaremacion := func(i *IntencionCambioBaremacion) {
		i.BaremacionMeritoRef = referenciaMaterialOpacaPrueba(101)
		i.VersionBase.BaremacionMeritoRef = i.BaremacionMeritoRef
	}
	mutarDocumentoObjeto := func(i *IntencionCambioBaremacion) {
		i.HuellaDocumentoFirmadoSHA256, i.HuellaObjetoCustodiadoSHA256 = huella(9), huella(9)
	}
	casos := []mutacion{
		{"ProcesoRef", "proceso", func(i *IntencionCambioBaremacion) { i.ProcesoRef = referenciaMaterialOpacaPrueba(101) }},
		{"SolicitudRef", "solicitud", func(i *IntencionCambioBaremacion) { i.SolicitudRef = referenciaMaterialOpacaPrueba(102) }},
		{"SujetoSeudonimoHMAC", "sujeto clave", func(i *IntencionCambioBaremacion) { i.SujetoSeudonimoHMAC.ClaveHMACRef = "seudonimo-sujeto-alt" }},
		{"SujetoSeudonimoHMAC", "sujeto HMAC", func(i *IntencionCambioBaremacion) { i.SujetoSeudonimoHMAC.ValorHMAC = huella(1) }},
		{"BaremacionMeritoRef", "baremacion", mutarBaremacion},
		{"VersionBase", "base referencia", mutarBaremacion},
		{"VersionBase", "base numero", mutarVersiones},
		{"VersionBase", "base huella", func(i *IntencionCambioBaremacion) { i.VersionBase.HuellaEstadoSHA256 = huella(2) }},
		{"VersionObjetivo", "objetivo", mutarVersiones},
		{"DecisionRef", "decision", func(i *IntencionCambioBaremacion) { i.DecisionRef = referenciaMaterialOpacaPrueba(103) }},
		{"NumeroDecision", "numero decision", mutarVersiones},
		{"ClaseDecision", "clase decision", func(i *IntencionCambioBaremacion) { i.ClaseDecision = dominiobolsa.ClaseDecisionRectificacion }},
		{"ResultadoDecision", "resultado", func(i *IntencionCambioBaremacion) { i.ResultadoDecision = dominiobolsa.ResultadoDesestimado }},
		{"HuellaContenidoDecisionSHA256", "contenido decision", func(i *IntencionCambioBaremacion) { i.HuellaContenidoDecisionSHA256 = huella(3) }},
		{"HuellaEstadoResultanteDecisionSHA256", "estado decision", func(i *IntencionCambioBaremacion) { i.HuellaEstadoResultanteDecisionSHA256 = huella(4) }},
		{"PoliticaFirmaRef", "politica firma", func(i *IntencionCambioBaremacion) { i.PoliticaFirmaRef = "politica-firma-2" }},
		{"PoliticaFirmaVersion", "version politica firma", func(i *IntencionCambioBaremacion) { i.PoliticaFirmaVersion++ }},
		{"HuellaPoliticaFirmaSHA256", "huella politica firma", func(i *IntencionCambioBaremacion) { i.HuellaPoliticaFirmaSHA256 = huella(5) }},
		{"PlanFirmaDurableRef", "plan", func(i *IntencionCambioBaremacion) { i.PlanFirmaDurableRef = referenciaMaterialOpacaPrueba(104) }},
		{"HuellaPlanFirmaDurableSHA256", "huella plan", func(i *IntencionCambioBaremacion) { i.HuellaPlanFirmaDurableSHA256 = huella(6) }},
		{"DocumentoFirmableRef", "firmable", func(i *IntencionCambioBaremacion) { i.DocumentoFirmableRef = referenciaMaterialOpacaPrueba(105) }},
		{"VersionDocumentoFirmable", "version firmable", func(i *IntencionCambioBaremacion) { i.VersionDocumentoFirmable = "v2" }},
		{"HuellaDocumentoFirmableSHA256", "huella firmable", func(i *IntencionCambioBaremacion) { i.HuellaDocumentoFirmableSHA256 = huella(7) }},
		{"FirmaRef", "firma", func(i *IntencionCambioBaremacion) { i.FirmaRef = referenciaMaterialOpacaPrueba(106) }},
		{"HuellaFirmaSHA256", "huella firma", func(i *IntencionCambioBaremacion) { i.HuellaFirmaSHA256 = huella(8) }},
		{"DocumentoFirmadoRef", "firmado", func(i *IntencionCambioBaremacion) { i.DocumentoFirmadoRef = referenciaMaterialOpacaPrueba(107) }},
		{"HuellaDocumentoFirmadoSHA256", "huella firmado", mutarDocumentoObjeto},
		{"ManifiestoProbatorioRef", "manifiesto", func(i *IntencionCambioBaremacion) { i.ManifiestoProbatorioRef = referenciaMaterialOpacaPrueba(108) }},
		{"HuellaManifiestoProbatorioSHA256", "huella manifiesto", func(i *IntencionCambioBaremacion) { i.HuellaManifiestoProbatorioSHA256 = huella(10) }},
		{"SelloManifiestoProbatorioHMAC", "clave manifiesto", func(i *IntencionCambioBaremacion) { i.SelloManifiestoProbatorioHMAC.ClaveHMACRef = "manifiesto-v2-alt" }},
		{"SelloManifiestoProbatorioHMAC", "HMAC manifiesto", func(i *IntencionCambioBaremacion) { i.SelloManifiestoProbatorioHMAC.ValorHMAC = huella(11) }},
		{"ObjetoCustodiadoRef", "objeto", func(i *IntencionCambioBaremacion) { i.ObjetoCustodiadoRef = referenciaMaterialOpacaPrueba(109) }},
		{"VersionObjetoCustodiado", "version objeto", func(i *IntencionCambioBaremacion) { i.VersionObjetoCustodiado = "version-8" }},
		{"ConectorCustodiaID", "conector", func(i *IntencionCambioBaremacion) { i.ConectorCustodiaID = "almacen-logico-2" }},
		{"HuellaObjetoCustodiadoSHA256", "huella objeto", mutarDocumentoObjeto},
		{"FormatoDocumento", "catalogo formato", func(i *IntencionCambioBaremacion) { i.FormatoDocumento.CatalogoRef = "catalogo-formatos-firma-alt" }},
		{"FormatoDocumento", "version formato", func(i *IntencionCambioBaremacion) { i.FormatoDocumento.CatalogoVersion++ }},
		{"FormatoDocumento", "huella formato", func(i *IntencionCambioBaremacion) { i.FormatoDocumento.HuellaCatalogoSHA256 = huella(18) }},
		{"FormatoDocumento", "clave formato", func(i *IntencionCambioBaremacion) { i.FormatoDocumento.FormatoClave = "odt_xades" }},
		{"FormatoDocumento", "MIME formato", func(i *IntencionCambioBaremacion) {
			i.FormatoDocumento.MIMECanonico = "application/vnd.oasis.opendocument.text"
		}},
		{"ClasificacionDocumento", "catalogo clasificacion", func(i *IntencionCambioBaremacion) {
			i.ClasificacionDocumento.CatalogoRef = "catalogo-clasificacion-alt"
		}},
		{"ClasificacionDocumento", "version clasificacion", func(i *IntencionCambioBaremacion) { i.ClasificacionDocumento.CatalogoVersion++ }},
		{"ClasificacionDocumento", "huella clasificacion", func(i *IntencionCambioBaremacion) { i.ClasificacionDocumento.HuellaCatalogoSHA256 = huella(19) }},
		{"ClasificacionDocumento", "clave clasificacion", func(i *IntencionCambioBaremacion) {
			i.ClasificacionDocumento.ClasificacionClave = "datos_personales_media"
		}},
		{"TamanoDocumentoFirmado", "tamano", func(i *IntencionCambioBaremacion) { i.TamanoDocumentoFirmado++ }},
		{"EstadoInmovilizacionObjeto", "inmovilizacion", func(i *IntencionCambioBaremacion) { i.EstadoInmovilizacionObjeto = EstadoInmovilizacionAplicada }},
		{"EvidenciaRecuperacionFirmadoRef", "recuperacion", func(i *IntencionCambioBaremacion) {
			i.EvidenciaRecuperacionFirmadoRef = referenciaMaterialOpacaPrueba(110)
		}},
		{"HuellaEvidenciaRecuperacionSHA256", "huella recuperacion", func(i *IntencionCambioBaremacion) { i.HuellaEvidenciaRecuperacionSHA256 = huella(12) }},
		{"EvidenciaCustodiaFirmadoRef", "custodia", func(i *IntencionCambioBaremacion) { i.EvidenciaCustodiaFirmadoRef = referenciaMaterialOpacaPrueba(111) }},
		{"HuellaEvidenciaCustodiaSHA256", "huella custodia", func(i *IntencionCambioBaremacion) { i.HuellaEvidenciaCustodiaSHA256 = huella(13) }},
		{"EvidenciaRetencionFirmadoRef", "retencion", func(i *IntencionCambioBaremacion) {
			i.EvidenciaRetencionFirmadoRef = referenciaMaterialOpacaPrueba(112)
		}},
		{"HuellaEvidenciaRetencionSHA256", "huella retencion", func(i *IntencionCambioBaremacion) { i.HuellaEvidenciaRetencionSHA256 = huella(14) }},
		{"PoliticaRetencionRef", "politica retencion", func(i *IntencionCambioBaremacion) { i.PoliticaRetencionRef = "politica-retencion-2" }},
		{"PoliticaRetencionVersion", "version politica retencion", func(i *IntencionCambioBaremacion) { i.PoliticaRetencionVersion++ }},
		{"HuellaPoliticaRetencionSHA256", "huella politica retencion", func(i *IntencionCambioBaremacion) { i.HuellaPoliticaRetencionSHA256 = huella(15) }},
		{"RetenidoHasta", "plazo", func(i *IntencionCambioBaremacion) { i.RetenidoHasta = i.RetenidoHasta.Add(time.Microsecond) }},
		{"HuellaAgregadoObjetivoSHA256", "agregado", func(i *IntencionCambioBaremacion) { i.HuellaAgregadoObjetivoSHA256 = huella(16) }},
		{"MotivoClave", "motivo clave", func(i *IntencionCambioBaremacion) { i.MotivoClave = "merito_rectificado" }},
		{"MotivoHMAC", "motivo HMAC clave", func(i *IntencionCambioBaremacion) { i.MotivoHMAC.ClaveHMACRef = "motivo-baremacion-alt" }},
		{"MotivoHMAC", "motivo HMAC valor", func(i *IntencionCambioBaremacion) { i.MotivoHMAC.ValorHMAC = huella(17) }},
	}
	cubiertos := make(map[string]struct{})
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := base
			caso.aplicar(&candidata)
			if err := candidata.Validar(); err != nil {
				t.Fatalf("alternativa no valida: %v", err)
			}
			preimagen, err := candidata.representacionCanonicaSobreProbatorioParaHMAC(indice)
			defer destruirCargaProtegidaBaremacion(&preimagen)
			if err != nil || cargasProtegidasIgualesPrueba(preimagenBase, preimagen) {
				t.Fatalf("campo material no cambio preimagen: %v", err)
			}
		})
		cubiertos[caso.campo] = struct{}{}
	}
	for _, cerrado := range []string{
		"Version", "Clase", "EsquemaPlanFirmaDurable", "VersionPlanFirmaDurable",
		"EstadoPlanFirmaDurable", "EsquemaManifiestoProbatorio", "VersionManifiestoProbatorio",
		"ZonaCustodia", "EstadoDisponibilidadObjeto", "EsquemaEvidenciaRecuperacion",
		"VersionEvidenciaRecuperacion", "EsquemaEvidenciaCustodia", "VersionEvidenciaCustodia",
		"EsquemaEvidenciaRetencion", "VersionEvidenciaRetencion",
	} {
		cubiertos[cerrado] = struct{}{}
	}
	tipo := reflect.TypeOf(base)
	for posicion := 0; posicion < tipo.NumField(); posicion++ {
		if _, existe := cubiertos[tipo.Field(posicion).Name]; !existe {
			t.Fatalf("campo sin alternativa ni cobertura cerrada: %s", tipo.Field(posicion).Name)
		}
	}
}

func TestRepresentacionCanonicaIntencionTLVCompletaYVectorDorado(t *testing.T) {
	i := intencionCambioBaremacionValidaPrueba()
	indice := indiceIdempotenciaValidoPrueba()
	primera, err := i.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		t.Fatalf("representacion inicial: %v", err)
	}
	defer destruirCargaProtegidaBaremacion(&primera)
	segunda, err := i.representacionCanonicaSobreProbatorioParaHMAC(indice)
	defer destruirCargaProtegidaBaremacion(&segunda)
	if err != nil || !cargasProtegidasIgualesPrueba(primera, segunda) {
		t.Fatalf("representacion no determinista: %v", err)
	}
	materialPrimera := primera.Revelar()
	defer borrarBytesBaremacion(materialPrimera)
	const vectorDoradoSHA256 = "e810f4b0fc9993fa6228f65f4b57fd6175dd1d85d58234932ff3879f95f4a14f"
	resumen := sha256.Sum256(materialPrimera)
	if obtenido := hex.EncodeToString(resumen[:]); obtenido != vectorDoradoSHA256 {
		t.Fatalf("vector dorado: obtenido=%s esperado=%s", obtenido, vectorDoradoSHA256)
	}
	campos := leerCamposCanonicosIntencion(t, materialPrimera, 82)
	esperados := [][]byte{
		[]byte(esquemaCanonicoIntencionCambioBaremacionV1),
		canonicoUint16Intencion(indice.Version), canonicoUint32Intencion(indice.GeneracionClave),
		[]byte(indice.ClaveHMACRef), decodificarHexCanonicoIntencion(indice.ValorHMAC),
		canonicoUint16Intencion(i.Version), []byte(i.Clase), []byte(i.ProcesoRef), []byte(i.SolicitudRef),
		canonicoUint16Intencion(i.SujetoSeudonimoHMAC.Version), []byte(i.SujetoSeudonimoHMAC.ClaveHMACRef),
		decodificarHexCanonicoIntencion(i.SujetoSeudonimoHMAC.ValorHMAC), []byte(i.BaremacionMeritoRef),
		[]byte(i.VersionBase.BaremacionMeritoRef), canonicoUint64Intencion(i.VersionBase.Numero),
		decodificarHexCanonicoIntencion(i.VersionBase.HuellaEstadoSHA256), canonicoUint64Intencion(i.VersionObjetivo),
		[]byte(i.DecisionRef), canonicoUint64Intencion(i.NumeroDecision), []byte(i.ClaseDecision), []byte(i.ResultadoDecision),
		decodificarHexCanonicoIntencion(i.HuellaContenidoDecisionSHA256),
		decodificarHexCanonicoIntencion(i.HuellaEstadoResultanteDecisionSHA256), []byte(i.PoliticaFirmaRef),
		canonicoUint32Intencion(i.PoliticaFirmaVersion), decodificarHexCanonicoIntencion(i.HuellaPoliticaFirmaSHA256),
		[]byte(i.EsquemaPlanFirmaDurable), canonicoUint16Intencion(i.VersionPlanFirmaDurable), []byte(i.PlanFirmaDurableRef),
		decodificarHexCanonicoIntencion(i.HuellaPlanFirmaDurableSHA256), []byte(i.EstadoPlanFirmaDurable),
		[]byte(i.DocumentoFirmableRef), []byte(i.VersionDocumentoFirmable),
		decodificarHexCanonicoIntencion(i.HuellaDocumentoFirmableSHA256), []byte(i.FirmaRef),
		decodificarHexCanonicoIntencion(i.HuellaFirmaSHA256), []byte(i.DocumentoFirmadoRef),
		decodificarHexCanonicoIntencion(i.HuellaDocumentoFirmadoSHA256), []byte(i.EsquemaManifiestoProbatorio),
		canonicoUint16Intencion(i.VersionManifiestoProbatorio), []byte(i.ManifiestoProbatorioRef),
		decodificarHexCanonicoIntencion(i.HuellaManifiestoProbatorioSHA256),
		canonicoUint16Intencion(i.SelloManifiestoProbatorioHMAC.Version), []byte(i.SelloManifiestoProbatorioHMAC.ClaveHMACRef),
		decodificarHexCanonicoIntencion(i.SelloManifiestoProbatorioHMAC.ValorHMAC), []byte(i.ObjetoCustodiadoRef),
		[]byte(i.VersionObjetoCustodiado), []byte(i.ConectorCustodiaID), []byte(i.ZonaCustodia),
		decodificarHexCanonicoIntencion(i.HuellaObjetoCustodiadoSHA256), []byte(i.FormatoDocumento.CatalogoRef),
		canonicoUint32Intencion(i.FormatoDocumento.CatalogoVersion),
		decodificarHexCanonicoIntencion(i.FormatoDocumento.HuellaCatalogoSHA256), []byte(i.FormatoDocumento.FormatoClave),
		[]byte(i.FormatoDocumento.MIMECanonico), []byte(i.ClasificacionDocumento.CatalogoRef),
		canonicoUint32Intencion(i.ClasificacionDocumento.CatalogoVersion),
		decodificarHexCanonicoIntencion(i.ClasificacionDocumento.HuellaCatalogoSHA256),
		[]byte(i.ClasificacionDocumento.ClasificacionClave), canonicoUint64Intencion(i.TamanoDocumentoFirmado),
		[]byte(i.EstadoInmovilizacionObjeto), []byte(i.EstadoDisponibilidadObjeto), []byte(i.EsquemaEvidenciaRecuperacion),
		canonicoUint16Intencion(i.VersionEvidenciaRecuperacion), []byte(i.EvidenciaRecuperacionFirmadoRef),
		decodificarHexCanonicoIntencion(i.HuellaEvidenciaRecuperacionSHA256), []byte(i.EsquemaEvidenciaCustodia),
		canonicoUint16Intencion(i.VersionEvidenciaCustodia), []byte(i.EvidenciaCustodiaFirmadoRef),
		decodificarHexCanonicoIntencion(i.HuellaEvidenciaCustodiaSHA256), []byte(i.EsquemaEvidenciaRetencion),
		canonicoUint16Intencion(i.VersionEvidenciaRetencion), []byte(i.EvidenciaRetencionFirmadoRef),
		decodificarHexCanonicoIntencion(i.HuellaEvidenciaRetencionSHA256), []byte(i.PoliticaRetencionRef),
		canonicoUint32Intencion(i.PoliticaRetencionVersion), decodificarHexCanonicoIntencion(i.HuellaPoliticaRetencionSHA256),
		canonicoTiempoIntencion(i.RetenidoHasta), decodificarHexCanonicoIntencion(i.HuellaAgregadoObjetivoSHA256),
		[]byte(i.MotivoClave), canonicoUint16Intencion(i.MotivoHMAC.Version), []byte(i.MotivoHMAC.ClaveHMACRef),
		decodificarHexCanonicoIntencion(i.MotivoHMAC.ValorHMAC),
	}
	defer func() {
		for _, esperado := range esperados {
			borrarBytesBaremacion(esperado)
		}
	}()
	if len(esperados) != 83 || len(campos) != 83 {
		t.Fatalf("tabla TLV incompleta: esperados=%d campos=%d", len(esperados), len(campos))
	}
	for etiqueta, esperado := range esperados {
		if !bytes.Equal(campos[etiqueta], esperado) {
			t.Fatalf("etiqueta %d no corresponde 1:1", etiqueta)
		}
	}
	for etiqueta, longitud := range map[int]int{
		1: 2, 2: 4, 4: 32, 5: 2, 9: 2, 11: 32, 14: 8, 15: 32,
		16: 8, 18: 8, 21: 32, 22: 32, 24: 4, 27: 2, 29: 32, 33: 32,
		35: 32, 37: 32, 39: 2, 41: 32, 42: 2, 44: 32, 49: 32, 51: 4,
		52: 32, 56: 4, 57: 32, 59: 8, 63: 2, 65: 32, 67: 2, 69: 32,
		71: 2, 73: 32, 75: 4, 76: 32, 77: 8, 78: 32, 80: 2, 82: 32,
	} {
		if len(campos[etiqueta]) != longitud {
			t.Fatalf("longitud no canonica etiqueta %d: %d != %d", etiqueta, len(campos[etiqueta]), longitud)
		}
	}
	alterada := primera.Revelar()
	defer borrarBytesBaremacion(alterada)
	alterada[0] ^= 0xff
	if bytes.Equal(alterada, materialPrimera) {
		t.Fatal("CargaProtegida compartio memoria")
	}
	releida := i
	releida.RetenidoHasta = time.UnixMicro(i.RetenidoHasta.UnixMicro()).UTC()
	roundtrip, err := releida.representacionCanonicaSobreProbatorioParaHMAC(indice)
	defer destruirCargaProtegidaBaremacion(&roundtrip)
	if err != nil || !cargasProtegidasIgualesPrueba(primera, roundtrip) {
		t.Fatalf("round-trip a microsegundos cambio TLV: %v", err)
	}
}

func TestRepresentacionCanonicaIntencionNoColisionaPorCamposNiOrden(t *testing.T) {
	izquierda := intencionCambioBaremacionValidaPrueba()
	izquierda.ProcesoRef, izquierda.SolicitudRef = referenciaMaterialOpacaPrueba(201), referenciaMaterialOpacaPrueba(202)
	derecha := intencionCambioBaremacionValidaPrueba()
	derecha.ProcesoRef, derecha.SolicitudRef = referenciaMaterialOpacaPrueba(203), referenciaMaterialOpacaPrueba(204)
	compararRepresentacionesDistintas(t, izquierda, derecha)
	orden := izquierda
	orden.ProcesoRef, orden.SolicitudRef = izquierda.SolicitudRef, izquierda.ProcesoRef
	compararRepresentacionesDistintas(t, izquierda, orden)
}

func TestIntencionNoAdmiteCamposEfimerosNiMemoriaMutable(t *testing.T) {
	tipo := reflect.TypeOf(IntencionCambioBaremacion{})
	prohibidos := []string{
		"contexto", "autenticacion", "autorizacion", "sesion", "token", "auditoria",
		"outbox", "correlacion", "intento", "claveidempotencia", "clavecliente",
		"solicitadaen", "confirmadaen", "creadaen", "emitidaen",
	}
	for posicion := 0; posicion < tipo.NumField(); posicion++ {
		campo := tipo.Field(posicion)
		nombre := strings.ToLower(campo.Name)
		for _, prohibido := range prohibidos {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("campo efimero %q presente en %s", prohibido, campo.Name)
			}
		}
		switch campo.Type.Kind() {
		case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface, reflect.Func, reflect.Chan:
			t.Fatalf("campo mutable/indirecto presente: %s", campo.Name)
		}
	}
	tipoIndice := reflect.TypeOf(indiceIdempotenciaBaremacion{})
	if tipoIndice.NumField() != 4 {
		t.Fatalf("indice interno contiene campos inesperados: %d", tipoIndice.NumField())
	}
	tipoSolicitud := reflect.TypeOf(SolicitudTestimonioAtomicoIdempotenciaBaremacion{})
	for _, campo := range []string{"Contexto", "Autorizacion", "Sesion", "Token", "Correlacion", "Instante"} {
		if _, existe := tipoSolicitud.FieldByName(campo); existe {
			t.Fatalf("solicitud atomica contiene campo efimero %s", campo)
		}
	}
}

func TestValoresSensiblesNoFiltranPorFormatoSerializacionNiSlog(t *testing.T) {
	i := intencionCambioBaremacionValidaPrueba()
	indice := indiceIdempotenciaValidoPrueba()
	principal := principalEstableBaremacionHMAC{
		Version: VersionPrincipalEstableBaremacionV1, GeneracionClave: 1,
		ClaveHMACRef: "principal-ficticio-v1", ValorHMAC: huellaIdempotenciaPruebaID(91),
	}
	clave := claveClienteFicticiaPrueba(t)
	solicitud := solicitudTestimonioPrueba(t)
	materialClave := copiarMaterialClaveClientePrueba(clave)
	defer borrarBytesBaremacion(materialClave)
	materialClaveSolicitud := copiarMaterialClaveClientePrueba(solicitud.claveCliente)
	defer borrarBytesBaremacion(materialClaveSolicitud)
	configuracion := nuevaConfiguracionTestimonioPrueba(1, 1)
	productor := &productorTestimonioPrueba{configuracion: configuracion}
	verificador := &verificadorTestimonioPrueba{configuracion: configuracion}
	producto, err := construirProductoNominalPrueba(t,
		context.Background(), solicitud, productor, verificador,
	)
	if err != nil {
		t.Fatalf("crear producto protegido: %v", err)
	}
	resolver, err := NuevaSolicitudResolverSeudonimoSujetoBaremacion(
		i.ProcesoRef, i.SolicitudRef, i.BaremacionMeritoRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	var solicitudMotivo SolicitudDerivarHMACMotivoBaremacion
	var verificacionMotivo SolicitudVerificarHMACMotivoBaremacion
	if err := VisitarSolicitudVerificarHMACMotivoBaremacion(
		i.MotivoClave, []byte("motivo-ficticio-con-datos-aportados"), i.MotivoHMAC,
		func(verificacion SolicitudVerificarHMACMotivoBaremacion) error {
			solicitudMotivo = verificacion.Solicitud
			verificacionMotivo = verificacion
			return verificacion.Solicitud.VisitarMotivo(func([]byte) error { return nil })
		},
	); err != nil {
		t.Fatalf("crear solicitud efimera de motivo: %v", err)
	}
	referenciasSeparacion := referenciasSeisDominiosSeparacionPrueba(t)
	referenciasSeparacion[1] = referenciaGeneracionSeparacionPrueba(
		t, DominioClaveIndiceBaremacion, 1, indice.ClaveHMACRef,
	)
	referenciasSeparacion[5] = referenciaGeneracionSeparacionPrueba(
		t, DominioClaveIntencionBaremacion, 1, "intencion-ficticia-v1",
	)
	separacion, err := ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(
		referenciasSeparacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		valor    any
		secretos []string
	}{
		{i, []string{i.ProcesoRef, i.SolicitudRef, i.HuellaFirmaSHA256}},
		{indice, []string{indice.ClaveHMACRef, indice.ValorHMAC}},
		{principal, []string{principal.ClaveHMACRef, principal.ValorHMAC}},
		{i.SujetoSeudonimoHMAC, []string{i.SujetoSeudonimoHMAC.ClaveHMACRef, i.SujetoSeudonimoHMAC.ValorHMAC}},
		{HMACIntencionCambioBaremacion{Version: 1, ClaveHMACRef: "intencion-ficticia-v1", ValorHMAC: huellaIdempotenciaPruebaID(92)}, []string{"intencion-ficticia-v1"}},
		{i.MotivoHMAC, []string{i.MotivoHMAC.ClaveHMACRef, i.MotivoHMAC.ValorHMAC}},
		{i.SelloManifiestoProbatorioHMAC, []string{i.SelloManifiestoProbatorioHMAC.ClaveHMACRef}},
		{clave, []string{hex.EncodeToString(materialClave)}},
		{solicitud, []string{
			string(solicitud.despliegueRef), hex.EncodeToString(materialClaveSolicitud),
			solicitud.ambitoSujeto.procesoRef, solicitud.ambitoSujeto.solicitudRef,
			solicitud.seudonimo.ClaveHMACRef, solicitud.seudonimo.ValorHMAC,
		}},
		{producto.testimonio, []string{producto.testimonio.principales[0].ValorHMAC, producto.testimonio.evidencia.HuellaContenidoSHA256}},
		{producto.testimonio.identidades, []string{producto.testimonio.identidades.LlaveroRef}},
		{producto.testimonio.identidades.Topologia[0], []string{producto.testimonio.identidades.Topologia[0].ClaveHMACRef}},
		{producto.testimonio.resolucionIdentidad, []string{
			producto.testimonio.resolucionIdentidad.SnapshotRef,
			producto.testimonio.resolucionIdentidad.ClaveAtestacionRef,
			string(producto.testimonio.resolucionIdentidad.ValorAtestacion),
		}},
		{producto.testimonio.evidencia, []string{string(producto.testimonio.evidencia.Valor)}},
		{i.FormatoDocumento, []string{i.FormatoDocumento.CatalogoRef, i.FormatoDocumento.HuellaCatalogoSHA256}},
		{i.ClasificacionDocumento, []string{i.ClasificacionDocumento.CatalogoRef, i.ClasificacionDocumento.HuellaCatalogoSHA256}},
		{resolver, []string{i.ProcesoRef, i.SolicitudRef}},
		{solicitudMotivo, []string{"motivo-ficticio-con-datos-aportados"}},
		{verificacionMotivo, []string{i.MotivoHMAC.ValorHMAC}},
		{separacion, []string{indice.ClaveHMACRef, "intencion-ficticia-v1"}},
		{referenciasSeparacion[1], []string{indice.ClaveHMACRef}},
		{productor.retuvoFuenteIdentidad, []string{string(identidadInternaEstablePrueba(1))}},
		{productor.retuvoFuente, []string{hex.EncodeToString(materialClaveSolicitud)}},
		{productor.retuvoReceptor, []string{producto.testimonio.principales[0].ValorHMAC}},
		{verificador.retuvoVista, []string{producto.testimonio.evidencia.HuellaContenidoSHA256}},
	}
	for _, caso := range casos {
		exigirProteccionValorPrueba(t, caso.valor, caso.secretos...)
	}
}
