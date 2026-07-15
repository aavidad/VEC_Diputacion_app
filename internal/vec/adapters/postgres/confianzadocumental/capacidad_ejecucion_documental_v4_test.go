package confianzadocumental

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestDecisionCanonicaRemotaV4CoincideByteAByteConFormatoDelNucleo(t *testing.T) {
	escenario := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	proyeccion, err := domain.ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(
		escenario.payload,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosHistoricos, err := proyeccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := decisionCanonicaDesdeHistoricoV4(datosHistoricos)
	if err != nil {
		t.Fatal(err)
	}
	datosEvidencia, err := escenario.escenario.evidencia.Datos()
	if err != nil {
		t.Fatal(err)
	}
	esperada, err := datosEvidencia.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(obtenida, esperada) {
		t.Fatalf("la decision remota diverge del formato canonico: obtenida=%s esperada=%s", obtenida, esperada)
	}
}

func TestCapacidadV4LigaTodosLosArtefactosYCruceCaducidadYClaveFallan(t *testing.T) {
	escenario := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := escenario.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), escenario.vinculo, escenario.cabecera, escenario.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := escenario.escenario.emitidaEn.Add(time.Microsecond)
	escenario.servicio.reloj = &relojContadorAtestacionPDP{instante: instante}
	captura := &registradorAtestacionPDPCaptura{}
	escenario.servicio.repositorioEjecucionV4 = captura
	if _, err = escenario.servicio.EjecutarPlanDocumentalV4(context.Background(), autoridad); err != nil {
		t.Fatal(err)
	}
	metadatos, preimagen, efecto, err := serializarEjecucionDocumentalAtestadaV4(captura.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	datos := captura.solicitud.prueba.datos
	artefactos := artefactosEjecucionDocumentalV4{
		metadatos: metadatos, payload: append([]byte(nil), datos.PayloadVECAD1...),
		sobre:     append([]byte(nil), datos.SobreCOSESign1...),
		evidencia: append([]byte(nil), datos.EvidenciaCanonica...), preimagen: preimagen,
		decisionCanonica: append([]byte(nil), datos.DecisionCanonica...), efecto: efecto,
	}
	material := materialEmisorCapacidadDocumentalV4{
		claveID: "clave:capacidad:v4:1", version: 1,
		secreto: bytes.Repeat([]byte{0x5a}, 32), emisorID: "emisor:capacidad:v4:prueba",
		validaDesde: instante.Add(-time.Minute), validaHasta: instante.Add(time.Minute),
		estado: "activa",
	}
	capacidad, err := emitirCapacidadEjecucionDocumentalV4(
		instante, artefactos, material, autoridad.pruebaPDP,
		escenario.escenario.decision.ValidaHasta,
	)
	if err != nil {
		t.Fatal(err)
	}
	artefactos.capacidad = capacidad
	if err = artefactos.validarEn(instante); err != nil {
		t.Fatalf("capacidad valida rechazada: %v", err)
	}
	if _, err = serializarPaqueteEjecucionDocumentalV4(artefactos); err != nil {
		t.Fatalf("paquete valido rechazado: %v", err)
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*artefactosEjecucionDocumentalV4)
	}{
		{"metadatos", func(a *artefactosEjecucionDocumentalV4) { a.metadatos[0] ^= 1 }},
		{"payload", func(a *artefactosEjecucionDocumentalV4) { a.payload[0] ^= 1 }},
		{"sobre", func(a *artefactosEjecucionDocumentalV4) { a.sobre[0] ^= 1 }},
		{"evidencia", func(a *artefactosEjecucionDocumentalV4) { a.evidencia[0] ^= 1 }},
		{"preimagen", func(a *artefactosEjecucionDocumentalV4) { a.preimagen[0] ^= 1 }},
		{"decision", func(a *artefactosEjecucionDocumentalV4) { a.decisionCanonica[0] ^= 1 }},
		{"efecto", func(a *artefactosEjecucionDocumentalV4) { a.efecto[0] ^= 1 }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := clonarArtefactosPruebaV4(artefactos)
			caso.mutar(&copia)
			if copia.validarEn(instante) == nil {
				t.Fatal("se acepto un artefacto distinto del autenticado")
			}
		})
	}
	if artefactos.validarEn(instante.Add(emisionCapacidadMaximaV4)) == nil {
		t.Fatal("se acepto una capacidad caducada")
	}
	var documento capacidadEjecucionDocumentalV4JSON
	if err = decodificarJSONExactoDocumentalV4(capacidad, &documento); err != nil {
		t.Fatal(err)
	}
	materialAjeno := material
	materialAjeno.secreto = bytes.Repeat([]byte{0x6b}, 32)
	if documento.validarConMaterialEn(instante, artefactos, materialAjeno) == nil {
		t.Fatal("se acepto una capacidad con otra clave")
	}
	cruzados := clonarArtefactosPruebaV4(artefactos)
	cruzados.efecto = append([]byte(nil), artefactos.efecto...)
	cruzados.efecto[len(cruzados.efecto)-2] ^= 1
	if documento.validarConMaterialEn(instante, cruzados, material) == nil {
		t.Fatal("se acepto una capacidad cruzada con otro efecto")
	}
}

func TestEmisorHTTPV4RechazaCOSEAlteradoSinEmitirPaquete(t *testing.T) {
	escenario := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	instante := escenario.escenario.emitidaEn.Add(time.Microsecond)
	escenario.servicio.reloj = &relojContadorAtestacionPDP{instante: instante}
	material := materialEmisorCapacidadDocumentalV4{
		claveID: "clave:capacidad:v4:1", version: 1,
		secreto: bytes.Repeat([]byte{0x7c}, 32), emisorID: "emisor:capacidad:v4:prueba",
		validaDesde: instante.Add(-time.Minute), validaHasta: instante.Add(time.Minute),
		estado: "activa",
	}
	manejador := &manejadorHTTPEmisorCapacidadDocumentalV4{
		servicio: escenario.servicio, material: material,
	}
	solicitudAplicacion, err := escenario.vinculo.PrepararSolicitudAplicacionEn(instante)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := solicitudAplicacion.PreimagenRecursoParaEvidenciaDurable()
	if err != nil {
		t.Fatal(err)
	}
	preimagenBytes, _ := preimagen.SerializacionCanonicaParaPersistencia()
	sobreBytes, _ := escenario.sobre.COSESign1()
	sobreBytes[len(sobreBytes)-1] ^= 1
	cuerpo, err := jsonSolicitudRemotaPruebaV4(escenario.cabecera, sobreBytes, preimagenBytes)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodPost, rutaHTTPEmisorCapacidadDocumentalV4, bytes.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden || respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("COSE alterado no fue denegado de forma cerrada: %d %q", respuesta.Code, respuesta.Body.String())
	}
}

func TestEmisorHTTPV4VerificaCOSEYEmitePaqueteAutenticado(t *testing.T) {
	escenario := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	instante := escenario.escenario.emitidaEn.Add(time.Microsecond)
	escenario.servicio.reloj = &relojContadorAtestacionPDP{instante: instante}
	material := materialEmisorCapacidadDocumentalV4{
		claveID: "clave:capacidad:v4:1", version: 7,
		secreto: bytes.Repeat([]byte{0x7d}, 32), emisorID: "emisor:capacidad:v4:prueba",
		validaDesde: instante.Add(-time.Minute), validaHasta: instante.Add(time.Minute),
		estado: "activa",
	}
	manejador := &manejadorHTTPEmisorCapacidadDocumentalV4{
		servicio: escenario.servicio, material: material,
	}
	solicitudAplicacion, err := escenario.vinculo.PrepararSolicitudAplicacionEn(instante)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := solicitudAplicacion.PreimagenRecursoParaEvidenciaDurable()
	if err != nil {
		t.Fatal(err)
	}
	preimagenBytes, _ := preimagen.SerializacionCanonicaParaPersistencia()
	sobreBytes, _ := escenario.sobre.COSESign1()
	cuerpo, err := jsonSolicitudRemotaPruebaV4(escenario.cabecera, sobreBytes, preimagenBytes)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodPost, rutaHTTPEmisorCapacidadDocumentalV4, bytes.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK {
		t.Fatalf("emision valida rechazada: %d %q", respuesta.Code, respuesta.Body.String())
	}
	artefactos, err := interpretarPaqueteEjecucionDocumentalV4(respuesta.Body.Bytes(), instante)
	if err != nil {
		t.Fatalf("paquete emitido invalido: %v", err)
	}
	var capacidad capacidadEjecucionDocumentalV4JSON
	if decodificarJSONExactoDocumentalV4(artefactos.capacidad, &capacidad) != nil ||
		capacidad.validarConMaterialEn(instante, artefactos, material) != nil {
		t.Fatal("el paquete no conserva una capacidad HMAC valida")
	}
}

func TestPreimagenMACCapacidadV4EsLongitudUTF8YOrdenCerrado(t *testing.T) {
	obtenida := preimagenMACCapacidadDocumentalV4([]string{"á", "b"})
	esperada := []byte("2:á\n1:b\n")
	if !bytes.Equal(obtenida, esperada) {
		t.Fatalf("preimagen incompatible con PostgreSQL: %q", obtenida)
	}
	suma := sha256.Sum256(obtenida)
	if hex.EncodeToString(suma[:]) == "" {
		t.Fatal("vector SHA-256 vacio")
	}
}

func clonarArtefactosPruebaV4(a artefactosEjecucionDocumentalV4) artefactosEjecucionDocumentalV4 {
	return artefactosEjecucionDocumentalV4{
		metadatos: append([]byte(nil), a.metadatos...), payload: append([]byte(nil), a.payload...),
		sobre: append([]byte(nil), a.sobre...), evidencia: append([]byte(nil), a.evidencia...),
		preimagen:        append([]byte(nil), a.preimagen...),
		decisionCanonica: append([]byte(nil), a.decisionCanonica...),
		efecto:           append([]byte(nil), a.efecto...), capacidad: append([]byte(nil), a.capacidad...),
	}
}

func jsonSolicitudRemotaPruebaV4(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre, preimagen []byte,
) ([]byte, error) {
	return json.Marshal(solicitudRemotaCapacidadV4{
		Esquema: esquemaSolicitudRemotaCapacidadV4,
		Cabecera: cabeceraRemotaCapacidadV4{
			FormatoVersion: cabecera.FormatoVersion, Suite: cabecera.Suite,
			ClaveID: cabecera.ClaveID, Audiencia: cabecera.Audiencia,
		},
		Sobre: sobre, Preimagen: preimagen,
	})
}
