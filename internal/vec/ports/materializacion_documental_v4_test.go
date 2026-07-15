package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestEntradaNeutralDocumentalCanonizaCopiaYNoConcedeHMAC(t *testing.T) {
	parrafos := []string{"Uno", "Dos: con delimitador\ny unicode á"}
	contenido := domain.ContenidoDocumento{Titulo: "Resolución", Parrafos: parrafos}
	huellaDeclarada := "hmac-sha256:entrada-neutral-v4:" + strings.Repeat("a", 64)
	preparacion, err := NuevaPreparacionEntradaNeutralDocumentalNominal(contenido)
	if err != nil {
		t.Fatal(err)
	}
	preparacionBytes, _ := preparacion.ContenidoCanonico()
	preparacionContenido, _ := preparacion.Contenido()
	preparacionBytes[0] ^= 0xff
	preparacionContenido.Parrafos[0] = "alias"
	if preparacion.Validar() != nil {
		t.Fatal("los accesores alteraron la preparacion opaca")
	}
	entrada, err := NuevaEntradaNeutralDocumentalNominal(preparacion, huellaDeclarada)
	if err != nil {
		t.Fatal(err)
	}
	canonicoOriginal, err := entrada.ContenidoCanonico()
	if err != nil {
		t.Fatal(err)
	}
	parrafos[0] = "alterado fuera"
	contenido.Titulo = "alterado fuera"
	canonicoExpuesto, _ := entrada.ContenidoCanonico()
	canonicoExpuesto[0] ^= 0xff
	contenidoExpuesto, _ := entrada.Contenido()
	contenidoExpuesto.Parrafos[0] = "alterado desde clon"
	contenidoExpuesto.Titulo = "alterado desde clon"
	canonicoOtraLectura, _ := entrada.ContenidoCanonico()
	contenidoOtraLectura, _ := entrada.Contenido()
	if !bytes.Equal(canonicoOriginal, canonicoOtraLectura) ||
		contenidoOtraLectura.Titulo != "Resolución" || contenidoOtraLectura.Parrafos[0] != "Uno" {
		t.Fatal("la entrada neutral compartio memoria con una fuente o un accesor")
	}
	huella, _ := entrada.HuellaHMACDeclarada()
	if huella != huellaDeclarada || entrada.Validar() != nil {
		t.Fatal("la HMAC declarada o la entrada canonica cambiaron")
	}
	if !strings.Contains(entrada.String(), "OPACA") || strings.Contains(entrada.String(), "Resolución") {
		t.Fatal("la representacion de depuracion revelo contenido")
	}
	if _, err := json.Marshal(entrada); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
		t.Fatalf("serializacion JSON no bloqueada: %v", err)
	}
	if _, err := entrada.MarshalText(); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
		t.Fatalf("serializacion de texto no bloqueada: %v", err)
	}
	if _, err := json.Marshal(preparacion); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
		t.Fatalf("serializacion de preparacion no bloqueada: %v", err)
	}
	for nombre, valor := range map[string]any{"preparacion": preparacion, "entrada": entrada} {
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, huellaDeclarada) || strings.Contains(texto, "Resolución") {
			t.Fatalf("%s filtro material protegido: %s", nombre, texto)
		}
		if slog.Any("valor", valor).Value.Resolve().Kind() != slog.KindString {
			t.Fatalf("%s no se redacto en slog", nombre)
		}
		binario := valor.(interface{ MarshalBinary() ([]byte, error) })
		if _, err := binario.MarshalBinary(); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
			t.Fatalf("%s se serializo como binario: %v", nombre, err)
		}
	}

	// La longitud prefijada impide ambigüedad entre particiones de los mismos bytes.
	preparacionPrimera, _ := NuevaPreparacionEntradaNeutralDocumentalNominal(
		domain.ContenidoDocumento{Titulo: "a", Parrafos: []string{"bc"}},
	)
	preparacionSegunda, _ := NuevaPreparacionEntradaNeutralDocumentalNominal(
		domain.ContenidoDocumento{Titulo: "ab", Parrafos: []string{"c"}},
	)
	primera, _ := NuevaEntradaNeutralDocumentalNominal(preparacionPrimera, huellaDeclarada)
	segunda, _ := NuevaEntradaNeutralDocumentalNominal(preparacionSegunda, huellaDeclarada)
	bytesPrimera, _ := primera.ContenidoCanonico()
	bytesSegunda, _ := segunda.ContenidoCanonico()
	if bytes.Equal(bytesPrimera, bytesSegunda) {
		t.Fatal("el codec confundio dos estructuras neutrales distintas")
	}

	var grupo sync.WaitGroup
	erroresAcceso := make(chan error, 32)
	for indice := 0; indice < 32; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			for repeticion := 0; repeticion < 50; repeticion++ {
				copiaBytes, errBytes := entrada.ContenidoCanonico()
				copiaContenido, errContenido := entrada.Contenido()
				if errBytes != nil || errContenido != nil {
					erroresAcceso <- errors.Join(errBytes, errContenido)
					return
				}
				copiaBytes[0] ^= 0xff
				copiaContenido.Parrafos[0] = "mutación concurrente local"
			}
		}()
	}
	grupo.Wait()
	close(erroresAcceso)
	for err := range erroresAcceso {
		t.Fatalf("acceso concurrente fallo: %v", err)
	}
	if entrada.Validar() != nil {
		t.Fatal("una mutacion concurrente de copias altero la entrada")
	}
}

func TestEntradaNeutralDocumentalPreflightRechazaAntesDeCanonizar(t *testing.T) {
	huella := "hmac-sha256:entrada-neutral-v4:" + strings.Repeat("b", 64)
	casos := []domain.ContenidoDocumento{
		{},
		{Titulo: "invalido\x00"},
		{Titulo: "invalido\x01"},
		{Titulo: "invalido\x7f"},
		{Titulo: strings.Repeat("x", maximoBytesEntradaNeutralDocumental)},
		{Titulo: "titulo", Parrafos: make([]string, maximosParrafosEntradaNeutral+1)},
	}
	for indice, contenido := range casos {
		if _, err := NuevaPreparacionEntradaNeutralDocumentalNominal(contenido); !errors.Is(err, ErrEntradaNeutralDocumentalInvalida) {
			t.Fatalf("caso %d no rechazado: %v", indice, err)
		}
	}
	preparacion, _ := NuevaPreparacionEntradaNeutralDocumentalNominal(domain.ContenidoDocumento{Titulo: "valido"})
	if _, err := NuevaEntradaNeutralDocumentalNominal(
		PreparacionEntradaNeutralDocumentalNominal{}, huella,
	); !errors.Is(err, ErrEntradaNeutralDocumentalInvalida) {
		t.Fatalf("preparacion cero aceptada: %v", err)
	}
	if _, err := NuevaEntradaNeutralDocumentalNominal(
		preparacion, "sha256:"+strings.Repeat("a", 64),
	); !errors.Is(err, ErrEntradaNeutralDocumentalInvalida) {
		t.Fatalf("HMAC solo declarada con formato invalido aceptada: %v", err)
	}
}

type escritorHostilMaterialDocumentalV4 struct {
	recibido []byte
	retenido []byte
}

func (e *escritorHostilMaterialDocumentalV4) Write(p []byte) (int, error) {
	e.recibido = append(e.recibido, p...)
	e.retenido = p
	for indice := range p {
		p[indice] ^= 0xff
	}
	return len(p), nil
}

type escritorParcialMaterialDocumentalV4 struct{}

func (escritorParcialMaterialDocumentalV4) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func TestSumideroSalidaDocumentalAislaDestinoCierraYHashea(t *testing.T) {
	destino := &escritorHostilMaterialDocumentalV4{}
	sumidero, err := NuevoSumideroLimitadoSalidaDocumental(destino, 1024)
	if err != nil {
		t.Fatal(err)
	}
	original := []byte("contenido documental estable")
	esperado := append([]byte(nil), original...)
	if n, err := sumidero.Write(original); err != nil || n != len(original) {
		t.Fatalf("escritura valida: n=%d err=%v", n, err)
	}
	for indice := range original {
		original[indice] = 'x'
	}
	salida, err := sumidero.Cerrar()
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := salida.Datos()
	suma := sha256.Sum256(esperado)
	if datos.HuellaSHA256 != hex.EncodeToString(suma[:]) || datos.Tamano != uint64(len(esperado)) ||
		!bytes.Equal(destino.recibido, esperado) || bytes.Equal(destino.retenido, esperado) {
		t.Fatal("el destino hostil altero la observacion privada")
	}
	repetida, err := sumidero.Cerrar()
	datosRepetidos, _ := repetida.Datos()
	if err != nil || datosRepetidos != datos {
		t.Fatal("el cierre idempotente no devolvio la misma observacion")
	}
	if _, err := sumidero.Write([]byte("posterior")); !errors.Is(err, ErrSumideroSalidaDocumentalCerrado) {
		t.Fatalf("escritura posterior al cierre aceptada: %v", err)
	}

	otroDestino := &bytes.Buffer{}
	otro, _ := NuevoSumideroLimitadoSalidaDocumental(otroDestino, 1024)
	_, _ = otro.Write([]byte("contenido documental estable!"))
	otraSalida, _ := otro.Cerrar()
	otrosDatos, _ := otraSalida.Datos()
	if otrosDatos.HuellaSHA256 == datos.HuellaSHA256 {
		t.Fatal("dos salidas distintas compartieron SHA-256")
	}
}

func TestSumideroSalidaDocumentalFallaCerradoEnLimiteYParcial(t *testing.T) {
	destino := &bytes.Buffer{}
	limitado, _ := NuevoSumideroLimitadoSalidaDocumental(destino, 4)
	if n, err := limitado.Write([]byte("cinco")); n != 0 || !errors.Is(err, ErrLimiteSalidaDocumentalExcedido) {
		t.Fatalf("limite no aplicado antes del destino: n=%d err=%v", n, err)
	}
	if destino.Len() != 0 {
		t.Fatal("se escribio parcialmente un bloque que excedia el limite")
	}
	if _, err := limitado.Cerrar(); !errors.Is(err, ErrLimiteSalidaDocumentalExcedido) {
		t.Fatalf("el cierre olvido el fallo de limite: %v", err)
	}
	if _, err := limitado.Write([]byte("x")); !errors.Is(err, ErrSumideroSalidaDocumentalCerrado) {
		t.Fatalf("se reanudo un sumidero fallido: %v", err)
	}

	parcial, _ := NuevoSumideroLimitadoSalidaDocumental(escritorParcialMaterialDocumentalV4{}, 64)
	if _, err := parcial.Write([]byte("contenido")); !errors.Is(err, ErrEscrituraSalidaDocumentalIncompleta) {
		t.Fatalf("escritura corta aceptada: %v", err)
	}
	if _, err := parcial.Cerrar(); !errors.Is(err, ErrEscrituraSalidaDocumentalIncompleta) {
		t.Fatalf("se fabrico salida de una escritura parcial: %v", err)
	}

	vacio, _ := NuevoSumideroLimitadoSalidaDocumental(io.Discard, 64)
	if _, err := vacio.Cerrar(); !errors.Is(err, ErrSalidaDocumentalVacia) {
		t.Fatalf("se fabrico salida vacia: %v", err)
	}

	bloqueExcesivo, _ := NuevoSumideroLimitadoSalidaDocumental(
		io.Discard,
		maximoBytesSalidaDocumental,
	)
	if n, err := bloqueExcesivo.Write(make([]byte, maximoBytesBloqueSalidaDocumental+1)); n != 0 || !errors.Is(err, ErrBloqueSalidaDocumentalExcedido) {
		t.Fatalf("bloque que multiplicaba memoria aceptado: n=%d err=%v", n, err)
	}
	if _, err := bloqueExcesivo.Cerrar(); !errors.Is(err, ErrBloqueSalidaDocumentalExcedido) {
		t.Fatalf("el cierre olvido el bloque excesivo: %v", err)
	}
}

func TestSumideroSalidaDocumentalSerializaEscriturasYCierreConcurrentes(t *testing.T) {
	destino := &bytes.Buffer{}
	sumidero, _ := NuevoSumideroLimitadoSalidaDocumental(destino, 64*1024)
	if _, err := sumidero.Write([]byte("semilla|")); err != nil {
		t.Fatal(err)
	}
	inicio := make(chan struct{})
	errores := make(chan error, 64)
	var grupo sync.WaitGroup
	for indice := 0; indice < 64; indice++ {
		grupo.Add(1)
		go func(valor byte) {
			defer grupo.Done()
			<-inicio
			_, err := sumidero.Write(bytes.Repeat([]byte{valor}, 64))
			errores <- err
		}(byte(indice + 1))
	}
	tipoCierre := make(chan struct {
		salida SalidaObservadaDocumental
		err    error
	}, 1)
	go func() {
		<-inicio
		salida, err := sumidero.Cerrar()
		tipoCierre <- struct {
			salida SalidaObservadaDocumental
			err    error
		}{salida: salida, err: err}
	}()
	close(inicio)
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil && !errors.Is(err, ErrSumideroSalidaDocumentalCerrado) {
			t.Fatalf("error concurrente inesperado: %v", err)
		}
	}
	cierre := <-tipoCierre
	if cierre.err != nil {
		t.Fatal(cierre.err)
	}
	datos, _ := cierre.salida.Datos()
	suma := sha256.Sum256(destino.Bytes())
	if datos.Tamano != uint64(destino.Len()) || datos.HuellaSHA256 != hex.EncodeToString(suma[:]) {
		t.Fatal("la observacion concurrente no coincide con los bytes serializados")
	}
}

func TestSumideroSalidaDocumentalTypedNilFallaCerradoSinPanic(t *testing.T) {
	var sumidero *SumideroLimitadoSalidaDocumental
	if _, err := sumidero.Write([]byte("dato")); !errors.Is(
		err, ErrSumideroSalidaDocumentalInvalido,
	) {
		t.Fatalf("Write sobre typed nil no fallo cerrado: %v", err)
	}
	if _, err := sumidero.Cerrar(); !errors.Is(err, ErrSumideroSalidaDocumentalInvalido) {
		t.Fatalf("Cerrar sobre typed nil no fallo cerrado: %v", err)
	}
}

func TestPreparacionEscrituraAlmacenV4CotejaContratoCompleto(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	if err := (VinculoEjecucionEscrituraAlmacenDocumental{}).ValidarContra(
		escenario.ejecucion.ordenDespachoConsumida,
	); !errors.Is(err, ErrPruebaEscrituraAlmacenInvalida) {
		t.Fatalf("el vinculo cero no fallo cerrado: %v", err)
	}
	if escenario.preparacion.Validar() != nil || escenario.declaracion.Validar() != nil ||
		escenario.declaracion.ValidarContraEjecucion(
			escenario.ejecucion.ordenDespachoConsumida,
			escenario.salida,
		) != nil {
		t.Fatal("la preparacion exacta fue rechazada")
	}
	ordenCruzada := clonarOrdenDespachoDocumentalV3ConsumidaNominal(
		escenario.ejecucion.ordenDespachoConsumida,
	)
	ordenCruzada.solicitud.vinculo.ReservaRef = "reserva:documental:v3:otra"
	if escenario.declaracion.ValidarContraEjecucion(ordenCruzada, escenario.salida) == nil {
		t.Fatal("la materializacion acepto otro vinculo estable")
	}
	objeto, _ := escenario.declaracion.Objeto()
	evidencia, _ := escenario.declaracion.EvidenciaOperacion()
	if objeto.Zona != ZonaAlmacenCuarentena || objeto.Objeto != evidencia.Objeto ||
		objeto.EvidenciaCreacionRef != evidencia.Referencia ||
		evidencia.ReintentoIdempotente {
		t.Fatal("la declaracion opaca no conservo la creacion inicial exacta")
	}

	mutaciones := map[string]func(*escenarioMaterializacionDocumentalV4){
		"idempotencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.solicitud.ClaveIdempotencia = "idempotencia:materializacion:v4:otra"
		},
		"zona solicitud": func(e *escenarioMaterializacionDocumentalV4) {
			e.solicitud.Zona = ZonaAlmacenAdmitida
		},
		"mime solicitud": func(e *escenarioMaterializacionDocumentalV4) {
			e.solicitud.MIME = "application/octet-stream"
		},
		"tamano solicitud": func(e *escenarioMaterializacionDocumentalV4) { e.solicitud.Tamano++ },
		"huella solicitud": func(e *escenarioMaterializacionDocumentalV4) {
			e.solicitud.HuellaSHA256 = strings.Repeat("f", 64)
		},
		"contexto": func(e *escenarioMaterializacionDocumentalV4) {
			e.solicitud.Contexto = ContextoOperacionAlmacen{}
		},
		"objeto referencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.Objeto.Referencia = "objeto:documental:v4:otro"
		},
		"objeto version": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.Objeto.Version = "version:documental:v4:otra"
		},
		"objeto conector": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.ConectorID = "conector:documental:v4:otro"
		},
		"objeto zona": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.Zona = ZonaAlmacenAdmitida
		},
		"objeto mime":   func(e *escenarioMaterializacionDocumentalV4) { e.resultado.Objeto.MIME += "+otro" },
		"objeto tamano": func(e *escenarioMaterializacionDocumentalV4) { e.resultado.Objeto.Tamano++ },
		"objeto huella": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.HuellaSHA256 = strings.Repeat("e", 64)
		},
		"objeto evidencia creacion": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.EvidenciaCreacionRef = "evidencia:almacen:v4:otra"
		},
		"objeto tiempo": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Objeto.AlmacenadoEn = e.resultado.Objeto.AlmacenadoEn.Add(time.Second)
		},
		"evidencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Evidencia.Referencia = "evidencia:almacen:v4:otra"
		},
		"efecto evidencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Evidencia.EfectoRef = "efecto:documental:v4:otro"
		},
		"plan evidencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Evidencia.HuellaPlanEfectoSHA256 = strings.Repeat("1", 64)
		},
		"manifiesto evidencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Evidencia.HuellaManifiestoSHA256 = strings.Repeat("2", 64)
		},
		"decision evidencia": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Evidencia.HuellaDecisionSHA256 = strings.Repeat("3", 64)
		},
		"reintento autoafirmado": func(e *escenarioMaterializacionDocumentalV4) {
			e.resultado.Evidencia.ReintentoIdempotente = true
		},
		"salida huella": func(e *escenarioMaterializacionDocumentalV4) {
			datos, _ := e.salida.Datos()
			datos.HuellaSHA256 = strings.Repeat("4", 64)
			e.salida = SalidaObservadaDocumental{datos: &datos}
		},
		"salida tamano": func(e *escenarioMaterializacionDocumentalV4) {
			datos, _ := e.salida.Datos()
			datos.Tamano++
			e.salida = SalidaObservadaDocumental{datos: &datos}
		},
		"vinculo ausente": func(e *escenarioMaterializacionDocumentalV4) {
			e.vinculo = VinculoEjecucionEscrituraAlmacenDocumental{}
		},
		"vinculo estable": func(e *escenarioMaterializacionDocumentalV4) {
			e.vinculo.datos.vinculoActivacion.ReservaRef = "reserva:documental:v3:otra"
		},
		"huella vinculo estable": func(e *escenarioMaterializacionDocumentalV4) {
			e.vinculo.datos.HuellaVinculoEstableSHA256 = strings.Repeat("8", 64)
		},
		"huella orden despacho": func(e *escenarioMaterializacionDocumentalV4) {
			e.vinculo.datos.HuellaOrdenDespachoSHA256 = strings.Repeat("9", 64)
		},
		"secuencia orden despacho": func(e *escenarioMaterializacionDocumentalV4) {
			e.vinculo.datos.ordenDespachoConsumida.solicitud.orden.datos.ReciboInicio.SecuenciaCercado++
		},
		"comprobacion TCB": func(e *escenarioMaterializacionDocumentalV4) {
			e.vinculo.datos.ordenDespachoConsumida.resultado.comprobacionRef = "comprobacion:kms:v4:otra"
		},
		"politica capacidades": func(e *escenarioMaterializacionDocumentalV4) {
			e.politica.HuellaCapacidadesSHA256 = strings.Repeat("6", 64)
		},
		"capacidad adicional": func(e *escenarioMaterializacionDocumentalV4) {
			e.capacidades.PromocionAtomica = !e.capacidades.PromocionAtomica
		},
	}
	for nombre, mutar := range mutaciones {
		copia := escenario.clonarEntradas()
		mutar(&copia)
		if _, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
			copia.solicitud, copia.resultado, copia.capacidades, copia.salida,
			copia.vinculo, copia.politica,
		); !errors.Is(err, ErrPruebaEscrituraAlmacenInvalida) {
			t.Fatalf("manipulacion de %s aceptada: %v", nombre, err)
		}
	}
}

func TestPreparacionEscrituraAlmacenV4ConservaReintentoIdempotente(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	reintento := escenario.resultado
	reintento.Evidencia.Referencia = "evidencia:almacen:v4:reintento:001"
	reintento.Evidencia.RealizadaEn = escenario.resultado.Evidencia.RealizadaEn.Add(2 * time.Second)
	reintento.Evidencia.ReintentoIdempotente = true

	datosVinculo := escenario.vinculo.datos
	vinculoActivacion := datosVinculo.vinculoActivacion
	token := nuevoTokenCercadoEjecucionDocumentalV3Prueba(
		t, "token:cercado:v4:reintento", escenario.ejecucion.token.Secuencia()+1,
		vinculoActivacion, "clave:cercado:v4:reintento", 1,
		"evidencia:cercado:v4:reintento",
	)
	instanteCercado := escenario.resultado.Objeto.AlmacenadoEn.Add(time.Second)
	ordenDespachoConsumida := ordenDespachoDocumentalV3ConsumidaNominalPrueba(
		t, vinculoActivacion, token, instanteCercado,
		instanteCercado.Add(5*time.Minute), "materializacion-reintento",
	)
	vinculo, err := NuevoVinculoEjecucionEscrituraAlmacenDocumental(
		ordenDespachoConsumida,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparacion, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		escenario.solicitud, reintento, escenario.capacidades, escenario.salida,
		vinculo, escenario.politica,
	)
	if err != nil {
		t.Fatalf("reintento idempotente valido rechazado: %v", err)
	}
	declaracion, _ := NuevaDeclaracionEscrituraAlmacenDocumental(preparacion)
	objeto, _ := declaracion.Objeto()
	evidencia, _ := declaracion.EvidenciaOperacion()
	if objeto.EvidenciaCreacionRef != escenario.resultado.Evidencia.Referencia ||
		evidencia.Referencia != reintento.Evidencia.Referencia ||
		!objeto.AlmacenadoEn.Before(instanteCercado) ||
		!evidencia.RealizadaEn.After(instanteCercado) || !evidencia.ReintentoIdempotente {
		t.Fatal("el reintento confundio creacion, evidencia actual o tiempos")
	}

	antesDelCercado := reintento
	antesDelCercado.Evidencia.RealizadaEn = instanteCercado.Add(-time.Microsecond)
	if _, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		escenario.solicitud, antesDelCercado, escenario.capacidades,
		escenario.salida, vinculo, escenario.politica,
	); err == nil {
		t.Fatal("un reintento observado antes del cercado fue aceptado")
	}
}

func TestPoliticaAlmacenV4SeparaCapacidadesDeRetencionYBloqueo(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	if !escenario.politica.Requisitos.Retencion || !escenario.politica.Requisitos.BloqueoLegal {
		t.Fatal("el escenario no exige las capacidades de retencion y bloqueo")
	}
	objeto, _ := escenario.declaracion.Objeto()
	if !objeto.RetenidoHasta.IsZero() || objeto.Inmovilizado {
		t.Fatal("una capacidad del conector se confundio con estado inicial del objeto")
	}

	retenido := escenario.resultado
	retenido.Objeto.RetenidoHasta = retenido.Objeto.AlmacenadoEn.Add(time.Hour)
	retenido.Objeto.Inmovilizado = true
	politica, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		escenario.politica.PoliticaRef, escenario.politica.Version+1,
		strings.Repeat("8", 64), escenario.politica.Requisitos,
		escenario.capacidades, retenido.Objeto.RetenidoHasta, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		escenario.solicitud, retenido, escenario.capacidades, escenario.salida,
		escenario.vinculo, politica,
	); err != nil {
		t.Fatalf("estado inicial gobernado rechazado: %v", err)
	}

	sinCapacidad := escenario.capacidades
	sinCapacidad.BloqueoLegal = false
	if _, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		"politica:inmutabilidad:v4:insuficiente", 1, strings.Repeat("9", 64),
		escenario.politica.Requisitos, sinCapacidad, time.Time{}, false,
	); err == nil {
		t.Fatal("una politica acepto capacidades insuficientes")
	}
}

func TestHuellaEvidenciaOperacionAlmacenEnumeraTodosLosCampos(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	base := escenario.resultado.Evidencia
	huellaBase := huellaEvidenciaOperacionAlmacenDocumental(base)
	if !esSHA256Hexadecimal(huellaBase) {
		t.Fatal("huella base invalida")
	}
	mutaciones := map[string]func(*EvidenciaOperacionAlmacen){
		"referencia":     func(e *EvidenciaOperacionAlmacen) { e.Referencia += ":otra" },
		"conector":       func(e *EvidenciaOperacionAlmacen) { e.ConectorID += ":otro" },
		"accion negocio": func(e *EvidenciaOperacionAlmacen) { e.AccionNegocio += ":otra" },
		"accion":         func(e *EvidenciaOperacionAlmacen) { e.Accion = AccionAlmacenLeer },
		"efecto":         func(e *EvidenciaOperacionAlmacen) { e.EfectoRef += ":otro" },
		"plan":           func(e *EvidenciaOperacionAlmacen) { e.HuellaPlanEfectoSHA256 = strings.Repeat("1", 64) },
		"manifiesto":     func(e *EvidenciaOperacionAlmacen) { e.HuellaManifiestoSHA256 = strings.Repeat("2", 64) },
		"paso hash":      func(e *EvidenciaOperacionAlmacen) { e.HuellaPasoSHA256 = strings.Repeat("3", 64) },
		"paso":           func(e *EvidenciaOperacionAlmacen) { e.PasoRef = PasoOperacionAlmacen("otro_paso") },
		"decision":       func(e *EvidenciaOperacionAlmacen) { e.HuellaDecisionSHA256 = strings.Repeat("4", 64) },
		"objeto ref":     func(e *EvidenciaOperacionAlmacen) { e.Objeto.Referencia += ":otro" },
		"objeto version": func(e *EvidenciaOperacionAlmacen) { e.Objeto.Version += ":otra" },
		"operacion":      func(e *EvidenciaOperacionAlmacen) { e.OperacionRef += ":otra" },
		"correlacion":    func(e *EvidenciaOperacionAlmacen) { e.CorrelacionRef += ":otra" },
		"autorizacion":   func(e *EvidenciaOperacionAlmacen) { e.AutorizacionRef += ":otra" },
		"finalidad":      func(e *EvidenciaOperacionAlmacen) { e.Finalidad += ":otra" },
		"clasificacion":  func(e *EvidenciaOperacionAlmacen) { e.Clasificacion += ":otra" },
		"realizada":      func(e *EvidenciaOperacionAlmacen) { e.RealizadaEn = e.RealizadaEn.Add(time.Second) },
		"carga":          func(e *EvidenciaOperacionAlmacen) { e.CargaRef += ":otra" },
		"seudonimo": func(e *EvidenciaOperacionAlmacen) {
			e.SujetoSeudonimoHMAC = "hmac-sha256:seudonimo-v4:" + strings.Repeat("1", 64)
		},
		"recurso": func(e *EvidenciaOperacionAlmacen) { e.RecursoRef += ":otro" },
		"modulo":  func(e *EvidenciaOperacionAlmacen) { e.ModuloID += ":otro" },
		"solicitud": func(e *EvidenciaOperacionAlmacen) {
			e.HuellaSolicitudHMAC = "hmac-sha256:solicitud-v4:" + strings.Repeat("2", 64)
		},
		"fundamento": func(e *EvidenciaOperacionAlmacen) { e.FundamentoRef = "fundamento:documental:v4" },
		"reintento":  func(e *EvidenciaOperacionAlmacen) { e.ReintentoIdempotente = true },
	}
	esquemaInvalido := base
	esquemaInvalido.EsquemaContexto += ":otro"
	if esquemaInvalido.Validar() == nil || huellaEvidenciaOperacionAlmacenDocumental(esquemaInvalido) != "" {
		t.Fatal("un esquema fijo alterado no fallo cerrado")
	}
	for nombre, mutar := range mutaciones {
		copia := base
		mutar(&copia)
		if err := copia.Validar(); err != nil {
			t.Fatalf("la mutacion valida de %s fallo antes del codec: %v", nombre, err)
		}
		if huellaEvidenciaOperacionAlmacenDocumental(copia) == huellaBase {
			t.Fatalf("el campo %s no participa en la huella V1", nombre)
		}
	}
}

func TestPruebaCrudaEscrituraAlmacenEsNominalOpacaYCopia(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	declaracion := escenario.declaracion
	bytesSobre := []byte("cose-sign1-prueba-escritura-almacen-v4")
	sobre, err := NuevoSobreCriptograficoDocumentalCrudoV4(bytesSobre)
	if err != nil {
		t.Fatal(err)
	}
	bytesSobre[0] ^= 0xff
	prueba, err := NuevaPruebaCrudaEscrituraAlmacen(
		"prueba:escritura:almacen:v4:001", declaracion, sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	pruebaRef, _ := prueba.PruebaRef()
	if pruebaRef != "prueba:escritura:almacen:v4:001" {
		t.Fatal("la referencia nominal no quedo disponible para la capa interna")
	}
	mensaje, _ := prueba.Mensaje()
	if !bytes.Contains(mensaje, []byte(escenario.vinculo.datos.HuellaVinculoEstableSHA256)) ||
		!bytes.Contains(mensaje, []byte(escenario.vinculo.datos.HuellaOrdenDespachoSHA256)) {
		t.Fatal("la declaracion no comprometio el vinculo estable y la orden de despacho")
	}
	mensaje[0] ^= 0xff
	sobreCrudo, _ := prueba.SobreCrudo()
	cose, _ := sobreCrudo.COSESign1()
	cose[0] ^= 0xff
	coseOtraLectura, _ := sobreCrudo.COSESign1()
	if string(coseOtraLectura) != "cose-sign1-prueba-escritura-almacen-v4" ||
		prueba.ValidarSintaxis() != nil {
		t.Fatal("la prueba cruda compartio memoria con una entrada o accesor")
	}
	if _, err := json.Marshal(prueba); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
		t.Fatalf("serializacion de prueba cruda no bloqueada: %v", err)
	}

	// Otro mensaje puede formar un sobre nominal, pero nunca obtiene aqui un
	// recibo autoritativo: la comprobacion COSE queda fuera de ports.
	politicaAlterada := escenario.politica
	politicaAlterada.Version++
	preparacionAlterada, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.salida, escenario.vinculo, politicaAlterada,
	)
	if err != nil {
		t.Fatal(err)
	}
	alterada, _ := NuevaDeclaracionEscrituraAlmacenDocumental(preparacionAlterada)
	pruebaAlterada, err := NuevaPruebaCrudaEscrituraAlmacen(
		"prueba:escritura:almacen:v4:002", alterada, sobre,
	)
	if err != nil || pruebaAlterada.ValidarSintaxis() != nil {
		t.Fatalf("la prueba nominal alterada debia seguir siendo solo sintactica: %v", err)
	}
	huellaPrimera, _ := prueba.HuellaMensajeSHA256()
	huellaSegunda, _ := pruebaAlterada.HuellaMensajeSHA256()
	if huellaPrimera == huellaSegunda {
		t.Fatal("dos declaraciones distintas compartieron huella de mensaje")
	}
	if len(prueba.mensaje) > maximoBytesMensajeEscrituraAlmacenV4 {
		t.Fatal("el mensaje valido excedio el presupuesto COSE")
	}
	excesiva := prueba
	excesiva.mensaje = bytes.Repeat([]byte{'x'}, maximoBytesMensajeEscrituraAlmacenV4+1)
	excesiva.huellaMensajeSHA256 = huellaSHA256MaterialDocumental(excesiva.mensaje)
	if excesiva.ValidarSintaxis() == nil {
		t.Fatal("un mensaje un byte sobre el limite COSE fue aceptado")
	}
}

func TestPreparacionYDeclaracionAlmacenV4SonOpacasDefensivasYConcurrentes(t *testing.T) {
	origenes := []string{"https://carga.example.test"}
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	escenario.capacidades.CargaDirectaTemporal = true
	escenario.capacidades.OrigenesCargaDirecta = origenes
	escenario.politica.Requisitos.CargaDirectaTemporal = true
	politica, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		escenario.politica.PoliticaRef, escenario.politica.Version,
		escenario.politica.HuellaSHA256, escenario.politica.Requisitos,
		escenario.capacidades, time.Time{}, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparacion, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		escenario.solicitud, escenario.resultado, escenario.capacidades,
		escenario.salida, escenario.vinculo, politica,
	)
	if err != nil {
		t.Fatal(err)
	}
	origenes[0] = "https://mutado.example.test"
	escenario.capacidades.OrigenesCargaDirecta[0] = "https://otro.example.test"
	declaracion, err := NuevaDeclaracionEscrituraAlmacenDocumental(preparacion)
	if err != nil {
		t.Fatal(err)
	}
	capacidades, _ := declaracion.Capacidades()
	if capacidades.OrigenesCargaDirecta[0] != "https://carga.example.test" {
		t.Fatal("la preparacion retuvo un alias de capacidades")
	}
	capacidades.OrigenesCargaDirecta[0] = "https://accesor.example.test"
	segundaLectura, _ := declaracion.Capacidades()
	if segundaLectura.OrigenesCargaDirecta[0] != "https://carga.example.test" ||
		declaracion.Validar() != nil {
		t.Fatal("un accesor altero la declaracion opaca")
	}
	for _, valor := range []any{escenario.vinculo, preparacion, declaracion} {
		texto := fmt.Sprintf("%s|%v|%+v|%#v", valor, valor, valor, valor)
		if !strings.Contains(texto, "OPAC") ||
			strings.Contains(texto, escenario.solicitud.ClaveIdempotencia) ||
			strings.Contains(texto, escenario.resultado.Objeto.Objeto.Referencia) {
			t.Fatalf("representacion no redactada: %s", texto)
		}
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
			t.Fatalf("serializacion no bloqueada: %v", err)
		}
		if slog.Any("valor", valor).Value.Resolve().Kind() != slog.KindString {
			t.Fatal("slog no aplico redaccion opaca")
		}
		binario, ok := valor.(interface{ MarshalBinary() ([]byte, error) })
		if !ok {
			t.Fatal("valor sensible sin bloqueo binario")
		}
		if _, err := binario.MarshalBinary(); !errors.Is(err, ErrSerializacionMaterialDocumentalProhibida) {
			t.Fatalf("serializacion binaria no bloqueada: %v", err)
		}
	}
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(PreparacionEscrituraAlmacenDocumentalV4Nominal{}),
		reflect.TypeOf(DeclaracionEscrituraAlmacenDocumental{}),
	} {
		for indice := 0; indice < tipo.NumField(); indice++ {
			if tipo.Field(indice).IsExported() {
				t.Fatalf("%s expone el campo %s", tipo.Name(), tipo.Field(indice).Name)
			}
		}
	}

	var grupo sync.WaitGroup
	errores := make(chan error, 64)
	for indice := 0; indice < 64; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			if err := preparacion.Validar(); err != nil {
				errores <- err
				return
			}
			copia, err := declaracion.Capacidades()
			if err == nil {
				copia.OrigenesCargaDirecta[0] = "https://local.example.test"
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPreparacionAlmacenV4DetectaCadaCampoDeContextoCapacidadesYPolitica(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	mutacionesContexto := map[string]func(*ProyeccionContextoOperacionAlmacen){
		"esquema":        func(p *ProyeccionContextoOperacionAlmacen) { p.Esquema += ":otro" },
		"operacion":      func(p *ProyeccionContextoOperacionAlmacen) { p.OperacionRef += ":otra" },
		"correlacion":    func(p *ProyeccionContextoOperacionAlmacen) { p.CorrelacionRef += ":otra" },
		"autorizacion":   func(p *ProyeccionContextoOperacionAlmacen) { p.AutorizacionRef += ":otra" },
		"finalidad":      func(p *ProyeccionContextoOperacionAlmacen) { p.Finalidad += ":otra" },
		"clasificacion":  func(p *ProyeccionContextoOperacionAlmacen) { p.Clasificacion += ":otra" },
		"accion negocio": func(p *ProyeccionContextoOperacionAlmacen) { p.AccionNegocio += ":otra" },
		"accion tecnica": func(p *ProyeccionContextoOperacionAlmacen) { p.AccionTecnica = AccionAlmacenLeer },
		"carga":          func(p *ProyeccionContextoOperacionAlmacen) { p.CargaRef += ":otra" },
		"seudonimo": func(p *ProyeccionContextoOperacionAlmacen) {
			p.SujetoSeudonimoHMAC = "hmac-sha256:otro:" + strings.Repeat("1", 64)
		},
		"recurso":        func(p *ProyeccionContextoOperacionAlmacen) { p.RecursoRef += ":otro" },
		"modulo":         func(p *ProyeccionContextoOperacionAlmacen) { p.ModuloID += ":otro" },
		"tipo recurso":   func(p *ProyeccionContextoOperacionAlmacen) { p.TipoRecurso += ":otro" },
		"huella recurso": func(p *ProyeccionContextoOperacionAlmacen) { p.HuellaRecursoSHA256 = strings.Repeat("1", 64) },
		"huella solicitud": func(p *ProyeccionContextoOperacionAlmacen) {
			p.HuellaSolicitudHMAC = "hmac-sha256:otra:" + strings.Repeat("2", 64)
		},
		"efecto":            func(p *ProyeccionContextoOperacionAlmacen) { p.EfectoRef += ":otro" },
		"plan efecto":       func(p *ProyeccionContextoOperacionAlmacen) { p.HuellaPlanEfectoSHA256 = strings.Repeat("2", 64) },
		"manifiesto":        func(p *ProyeccionContextoOperacionAlmacen) { p.HuellaManifiestoSHA256 = strings.Repeat("3", 64) },
		"paso huella":       func(p *ProyeccionContextoOperacionAlmacen) { p.HuellaPasoSHA256 = strings.Repeat("4", 64) },
		"paso":              func(p *ProyeccionContextoOperacionAlmacen) { p.PasoRef = "otro_paso" },
		"objeto referencia": func(p *ProyeccionContextoOperacionAlmacen) { p.ObjetoVinculado.Referencia = "objeto:otro" },
		"objeto version":    func(p *ProyeccionContextoOperacionAlmacen) { p.ObjetoVinculado.Version = "version:otra" },
		"decision":          func(p *ProyeccionContextoOperacionAlmacen) { p.HuellaDecisionSHA256 = strings.Repeat("5", 64) },
		"verificada":        func(p *ProyeccionContextoOperacionAlmacen) { p.VerificadaEn = p.VerificadaEn.Add(time.Microsecond) },
		"vigencia":          func(p *ProyeccionContextoOperacionAlmacen) { p.ValidaHasta = p.ValidaHasta.Add(time.Microsecond) },
	}
	for nombre, mutar := range mutacionesContexto {
		copia := PreparacionEscrituraAlmacenDocumentalV4Nominal{
			datos: clonarDatosPreparacionEscrituraAlmacenDocumentalV4(escenario.preparacion.datos),
		}
		mutar(&copia.datos.solicitud.Contexto)
		if copia.Validar() == nil {
			t.Fatalf("mutacion de contexto %s aceptada", nombre)
		}
	}

	mutacionesCapacidades := map[string]func(*CapacidadesAlmacenObjetos){
		"conector":          func(c *CapacidadesAlmacenObjetos) { c.ConectorID += ":otro" },
		"escritura flujo":   func(c *CapacidadesAlmacenObjetos) { c.EscrituraEnFlujo = !c.EscrituraEnFlujo },
		"lectura flujo":     func(c *CapacidadesAlmacenObjetos) { c.LecturaEnFlujo = !c.LecturaEnFlujo },
		"referencias":       func(c *CapacidadesAlmacenObjetos) { c.ReferenciasOpacas = !c.ReferenciasOpacas },
		"integridad":        func(c *CapacidadesAlmacenObjetos) { c.IntegridadSHA256 = !c.IntegridadSHA256 },
		"versionado":        func(c *CapacidadesAlmacenObjetos) { c.Versionado = !c.Versionado },
		"retencion":         func(c *CapacidadesAlmacenObjetos) { c.Retencion = !c.Retencion },
		"bloqueo":           func(c *CapacidadesAlmacenObjetos) { c.BloqueoLegal = !c.BloqueoLegal },
		"promocion":         func(c *CapacidadesAlmacenObjetos) { c.PromocionAtomica = !c.PromocionAtomica },
		"carga directa":     func(c *CapacidadesAlmacenObjetos) { c.CargaDirectaTemporal = !c.CargaDirectaTemporal },
		"cifrado transito":  func(c *CapacidadesAlmacenObjetos) { c.CifradoEnTransito = !c.CifradoEnTransito },
		"cifrado reposo":    func(c *CapacidadesAlmacenObjetos) { c.CifradoEnReposo = !c.CifradoEnReposo },
		"cifrado objeto":    func(c *CapacidadesAlmacenObjetos) { c.CifradoPorObjeto = !c.CifradoPorObjeto },
		"tamano maximo":     func(c *CapacidadesAlmacenObjetos) { c.TamanoMaximoObjeto++ },
		"preserva original": func(c *CapacidadesAlmacenObjetos) { c.PreservaObjetoOriginal = !c.PreservaObjetoOriginal },
		"origenes":          func(c *CapacidadesAlmacenObjetos) { c.OrigenesCargaDirecta = []string{"https://otro.example.test"} },
	}
	for nombre, mutar := range mutacionesCapacidades {
		copia := PreparacionEscrituraAlmacenDocumentalV4Nominal{
			datos: clonarDatosPreparacionEscrituraAlmacenDocumentalV4(escenario.preparacion.datos),
		}
		mutar(&copia.datos.capacidades)
		if copia.Validar() == nil {
			t.Fatalf("mutacion de capacidades %s aceptada", nombre)
		}
	}

	mensajeBase := serializarDeclaracionEscrituraAlmacenDocumental(escenario.declaracion)
	mutacionesPolitica := map[string]func(*VinculoPoliticaInmutabilidadDocumental){
		"referencia":         func(p *VinculoPoliticaInmutabilidadDocumental) { p.PoliticaRef += ":otra" },
		"version":            func(p *VinculoPoliticaInmutabilidadDocumental) { p.Version++ },
		"huella":             func(p *VinculoPoliticaInmutabilidadDocumental) { p.HuellaSHA256 = strings.Repeat("8", 64) },
		"requisitos":         func(p *VinculoPoliticaInmutabilidadDocumental) { p.Requisitos.Retencion = false },
		"huella requisitos":  func(p *VinculoPoliticaInmutabilidadDocumental) { p.HuellaRequisitosSHA256 = strings.Repeat("9", 64) },
		"huella capacidades": func(p *VinculoPoliticaInmutabilidadDocumental) { p.HuellaCapacidadesSHA256 = strings.Repeat("a", 64) },
		"retencion gobernada": func(p *VinculoPoliticaInmutabilidadDocumental) {
			p.RetencionHasta = escenario.resultado.Objeto.AlmacenadoEn.Add(time.Hour)
		},
		"bloqueo gobernado": func(p *VinculoPoliticaInmutabilidadDocumental) { p.ExigeInmovilizacionInicial = true },
	}
	for nombre, mutar := range mutacionesPolitica {
		copia := DeclaracionEscrituraAlmacenDocumental{
			datos: clonarDatosPreparacionEscrituraAlmacenDocumentalV4(escenario.declaracion.datos),
		}
		mutar(&copia.datos.politica)
		if copia.Validar() == nil && bytes.Equal(
			mensajeBase, serializarDeclaracionEscrituraAlmacenDocumental(copia),
		) {
			t.Fatalf("mutacion valida de politica %s no cambio el mensaje", nombre)
		}
	}
}

func TestCapacidadesAlmacenV4AcotanOrigenesAntesDeCopiarYSerializar(t *testing.T) {
	escenario := escenarioMaterializacionDocumentalV4Prueba(t)
	crearOrigenes := func(cantidad, longitud int) []string {
		origenes := make([]string, 0, cantidad)
		for indice := 0; indice < cantidad; indice++ {
			sufijo := fmt.Sprintf("%02d.example", indice)
			relleno := longitud - len("https://") - len(sufijo)
			if relleno < 1 {
				t.Fatal("longitud de prueba insuficiente")
			}
			origenes = append(origenes, "https://"+strings.Repeat("a", relleno)+sufijo)
		}
		return origenes
	}
	requisitos := escenario.politica.Requisitos
	requisitos.CargaDirectaTemporal = true
	frontera := escenario.capacidades
	frontera.CargaDirectaTemporal = true
	frontera.OrigenesCargaDirecta = crearOrigenes(
		maximoBytesOrigenesCargaDirectaV4/maximoBytesOrigenCargaDirectaV4,
		maximoBytesOrigenCargaDirectaV4,
	)
	politica, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		"politica:origenes:v4:frontera", 1, strings.Repeat("b", 64),
		requisitos, frontera, time.Time{}, false,
	)
	if err != nil || politica.HuellaCapacidadesSHA256 == "" {
		t.Fatalf("frontera agregada valida rechazada: %v", err)
	}

	porOrigen := frontera
	porOrigen.OrigenesCargaDirecta = crearOrigenes(1, maximoBytesOrigenCargaDirectaV4+1)
	if _, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		"politica:origenes:v4:exceso-individual", 1, strings.Repeat("c", 64),
		requisitos, porOrigen, time.Time{}, false,
	); err == nil {
		t.Fatal("un origen excesivo alcanzo la canonizacion")
	}

	agregado := frontera
	agregado.OrigenesCargaDirecta = crearOrigenes(
		maximoBytesOrigenesCargaDirectaV4/maximoBytesOrigenCargaDirectaV4+1,
		maximoBytesOrigenCargaDirectaV4,
	)
	if _, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		"politica:origenes:v4:exceso-agregado", 1, strings.Repeat("d", 64),
		requisitos, agregado, time.Time{}, false,
	); err == nil {
		t.Fatal("el agregado excesivo alcanzo la canonizacion")
	}
	if _, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		escenario.solicitud, escenario.resultado, agregado, escenario.salida,
		escenario.vinculo, escenario.politica,
	); !errors.Is(err, ErrPruebaEscrituraAlmacenInvalida) {
		t.Fatalf("el preflight no rechazo el agregado antes de copiar: %v", err)
	}
}

type escenarioMaterializacionDocumentalV4 struct {
	ejecucion   escenarioEjecucionDocumentalV3Prueba
	solicitud   SolicitudEscribirObjeto
	resultado   ResultadoOperacionObjeto
	capacidades CapacidadesAlmacenObjetos
	salida      SalidaObservadaDocumental
	vinculo     VinculoEjecucionEscrituraAlmacenDocumental
	politica    VinculoPoliticaInmutabilidadDocumental
	preparacion PreparacionEscrituraAlmacenDocumentalV4Nominal
	declaracion DeclaracionEscrituraAlmacenDocumental
}

func (e escenarioMaterializacionDocumentalV4) clonarEntradas() escenarioMaterializacionDocumentalV4 {
	e.capacidades = clonarCapacidadesAlmacenDocumentalV4(e.capacidades)
	e.vinculo = clonarVinculoEjecucionEscrituraAlmacenDocumentalV4(e.vinculo)
	return e
}

func escenarioMaterializacionDocumentalV4Prueba(t *testing.T) escenarioMaterializacionDocumentalV4 {
	t.Helper()
	ejecucion := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	destino := &bytes.Buffer{}
	sumidero, err := NuevoSumideroLimitadoSalidaDocumental(destino, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sumidero.Write([]byte("%PDF-salida-materializada-v4")); err != nil {
		t.Fatal(err)
	}
	salida, err := sumidero.Cerrar()
	if err != nil {
		t.Fatal(err)
	}
	datosSalida, _ := salida.Datos()
	datosManifiesto, _ := ejecucion.manifiesto.Datos()
	decision, recurso, vinculosAlmacen, verificadaAutorizacion := autorizacionAlmacenPrueba(
		t, AccionNegocioCustodiarDecisionBaremacion,
		[]string{"documento_custodiado", "evidencia_custodia"}, false,
	)
	decision.DecisionRef = ejecucion.consumo.DecisionRef
	vinculosAlmacen.EfectoRef = datosManifiesto.EfectoRef
	recurso.Atributos[AtributoAlmacenEfectoRef] = vinculosAlmacen.EfectoRef
	recurso.Atributos[AtributoAlmacenHuellaManifiestoSHA256] = datosManifiesto.HuellaPlanSHA256
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision.ContextoRecursoHuellaSHA256 = huellaRecurso
	if decision.ValidarEvidenciaInstantanea() != nil {
		t.Fatal("la decision alineada con V4 quedo invalida")
	}
	pasoDocumental := &datosPasoGeneracionDocumental{
		pasoRef: "materializar_documento_v4", referenciaLogica: "documento:materializado:v4:001",
		claveIdempotencia: "idempotencia:materializacion:v4:001",
		formato:           domain.FormatoDocumentoPDF, zona: ZonaAlmacenCuarentena,
		mime: "application/pdf", tamano: int64(datosSalida.Tamano),
		huellaSHA256: datosSalida.HuellaSHA256, huellaPasoSHA256: strings.Repeat("6", 64),
	}
	especificacion := especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioCustodiarDecisionBaremacion,
		camposExactos: []string{"documento_custodiado", "evidencia_custodia"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: pasoDocumental.pasoRef, accion: AccionAlmacenEscribir,
			huellaPasoSHA256: pasoDocumental.huellaPasoSHA256, pasoDocumental: pasoDocumental,
		}},
		huellaManifiestoSHA256: datosManifiesto.HuellaPlanSHA256,
	}
	contexto, err := nuevoContextoOperacionAlmacen(
		decision, recurso, vinculosAlmacen, verificadaAutorizacion, especificacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	proyeccion, _ := contexto.Proyeccion()
	consumo := ConsumoDecisionEjecucionDocumentalV3{
		DecisionRef: proyeccion.AutorizacionRef, EfectoRef: proyeccion.EfectoRef,
		EsquemaHuellaDecision: EsquemaHuellaDecisionAutorizacionReforzadaV1,
		HuellaDecisionSHA256:  proyeccion.HuellaDecisionSHA256,
		HuellaPlanSHA256:      datosManifiesto.HuellaPlanSHA256,
	}
	vinculoActivacion := ejecucion.vinculoActivacion
	vinculoActivacion.ConsumoDecision = consumo
	if vinculoActivacion.Validar() != nil {
		t.Fatal("vinculo de materializacion invalido")
	}
	token := nuevoTokenCercadoEjecucionDocumentalV3Prueba(
		t, "token:cercado:materializacion:v4:001", 51, vinculoActivacion,
		"clave:cercado:materializacion:v4", 1,
		"evidencia:cercado:materializacion:v4:001",
	)
	solicitudCercado, _ := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		vinculoActivacion, token,
	)
	verificacionCercado, err := NuevosMetadatosComprobacionTokenCercadoDocumentalV3Nominal(
		solicitudCercado, "verificacion:cercado:materializacion:v4:001",
		verificadaAutorizacion.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenDespachoConsumida := ordenDespachoDocumentalV3ConsumidaNominalPrueba(
		t, vinculoActivacion, token, verificacionCercado.verificadaEn,
		verificacionCercado.verificadaEn.Add(9*time.Minute), "materializacion-v4",
	)
	vinculo, err := NuevoVinculoEjecucionEscrituraAlmacenDocumental(
		ordenDespachoConsumida,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := verificacionCercado.verificadaEn.Add(time.Second)
	solicitud := SolicitudEscribirObjeto{
		Contexto: contexto, ClaveIdempotencia: pasoDocumental.claveIdempotencia,
		Zona: pasoDocumental.zona, MIME: pasoDocumental.mime, Tamano: pasoDocumental.tamano,
		HuellaSHA256: pasoDocumental.huellaSHA256, Contenido: bytes.NewReader(destino.Bytes()),
	}
	referencia := ReferenciaObjetoAlmacen{
		Referencia: "objeto:documental:v4:001", Version: "version:documental:v4:001",
	}
	evidencia := evidenciaAlmacenVinculadaPrueba(
		contexto, referencia, "evidencia:almacen:v4:001", "conector:almacen:v4", "", instante,
	)
	objeto := ObjetoAlmacenado{
		Objeto: referencia, ConectorID: evidencia.ConectorID, Zona: ZonaAlmacenCuarentena,
		MIME: "application/pdf", Tamano: int64(datosSalida.Tamano),
		HuellaSHA256:         datosSalida.HuellaSHA256,
		EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: instante,
	}
	resultado := ResultadoOperacionObjeto{Objeto: objeto, Evidencia: evidencia}
	capacidades := CapacidadesAlmacenObjetos{
		ConectorID: evidencia.ConectorID, EscrituraEnFlujo: true, LecturaEnFlujo: true,
		ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
		Retencion: true, BloqueoLegal: true, CifradoEnTransito: true,
		CifradoEnReposo: true, TamanoMaximoObjeto: 1024 * 1024,
	}
	requisitos := RequisitosAlmacenObjetos{
		EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, Versionado: true, Retencion: true, BloqueoLegal: true,
		CifradoEnTransito: true, CifradoEnReposo: true,
		TamanoMinimoObjeto: int64(datosSalida.Tamano),
	}
	politica, err := NuevoVinculoPoliticaInmutabilidadDocumental(
		"politica:inmutabilidad:v4", 1, strings.Repeat("7", 64), requisitos,
		capacidades, time.Time{}, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparacion, err := NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
		solicitud, resultado, capacidades, salida, vinculo, politica,
	)
	if err != nil {
		t.Fatal(err)
	}
	declaracion, err := NuevaDeclaracionEscrituraAlmacenDocumental(preparacion)
	if err != nil {
		t.Fatal(err)
	}
	ejecucion.consumo = consumo
	ejecucion.vinculoActivacion = vinculoActivacion
	ejecucion.token = token
	ejecucion.verificacionCercado = verificacionCercado
	ejecucion.ordenDespachoConsumida = ordenDespachoConsumida
	return escenarioMaterializacionDocumentalV4{
		ejecucion: ejecucion, solicitud: solicitud, resultado: resultado,
		capacidades: capacidades, salida: salida, vinculo: vinculo, politica: politica,
		preparacion: preparacion, declaracion: declaracion,
	}
}
