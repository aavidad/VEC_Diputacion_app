package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSolicitudTransicionFuenteAutoridadV1SobreviveAReinicio(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	preparadaEn := fuente.CreadaEn.Add(time.Hour)
	registradaEn := preparadaEn.Add(time.Hour)
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, fuente, EstadoFuenteAutoridadPublicada, "per_publicador_reinicio_00000001",
		"publicacion_tras_reinicio", "a", registradaEn, preparadaEn,
	)
	bytesSolicitud, err := solicitud.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	serializada, err := json.Marshal(solicitud)
	if err != nil || !bytes.Equal(serializada, bytesSolicitud) {
		t.Fatalf("json.Marshal no conservó el compromiso: %s / %s, error=%v", serializada, bytesSolicitud, err)
	}
	rehidratada, err := RehidratarSolicitudTransicionFuenteAutoridadV1(bytesSolicitud)
	if err != nil {
		t.Fatalf("rehidratar tras reinicio: %v", err)
	}
	bytesRehidratados, err := rehidratada.BytesCanonicos()
	if err != nil || !bytes.Equal(bytesSolicitud, bytesRehidratados) {
		t.Fatalf("la solicitud cambió tras reinicio: %s / %s, error=%v", bytesSolicitud, bytesRehidratados, err)
	}
	if _, err := fuente.AplicarTransicionV1(rehidratada, evidencia, registradaEn); err != nil {
		t.Fatalf("completar tras reinicio: %v", err)
	}

	for nombre, alterada := range map[string][]byte{
		"espacio":           append([]byte(" "), bytesSolicitud...),
		"campo desconocido": bytes.Replace(bytesSolicitud, []byte(`{`), []byte(`{"desconocido":true,`), 1),
		"lista de valores":  append(append([]byte(nil), bytesSolicitud...), []byte(`[]`)...),
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := RehidratarSolicitudTransicionFuenteAutoridadV1(alterada); !errors.Is(err, ErrTransicionAutoridadInvalida) {
				t.Fatalf("solicitud no canónica aceptada: %v", err)
			}
		})
	}
}

func TestEstadoPersistibleFuenteAutoridadV1IdaVueltaConservaHuellas(t *testing.T) {
	borrador := nuevaFuenteAutoridadPrueba(t)
	estadoBorrador, err := borrador.EstadoPersistibleV1()
	if err != nil {
		t.Fatalf("EstadoPersistibleV1() error = %v", err)
	}
	if !bytes.Contains(estadoBorrador, []byte(`"ediciones_borrador":[]`)) ||
		!bytes.Contains(estadoBorrador, []byte(`"transiciones":[]`)) {
		t.Fatalf("las listas vacias no se emitieron como []: %s", estadoBorrador)
	}
	serializadoGenerico, err := json.Marshal(borrador)
	if err != nil || !bytes.Equal(serializadoGenerico, estadoBorrador) {
		t.Fatalf("json.Marshal eludió EstadoPersistibleV1: %s / %s, error=%v",
			serializadoGenerico, estadoBorrador, err)
	}
	var decodificadoGenerico FuenteAutoridadVersionada
	if err := json.Unmarshal(estadoBorrador, &decodificadoGenerico); err != nil {
		t.Fatalf("json.Unmarshal rechazó el estado canónico: %v", err)
	}
	estadoDecodificado, err := decodificadoGenerico.EstadoPersistibleV1()
	if err != nil || !bytes.Equal(estadoDecodificado, estadoBorrador) {
		t.Fatalf("json.Unmarshal alteró el estado canónico: %s / %s, error=%v",
			estadoDecodificado, estadoBorrador, err)
	}
	estadoConEspacio := bytes.Replace(
		estadoBorrador,
		[]byte(`"formato_version":1`),
		[]byte(`"formato_version": 1`),
		1,
	)
	if err := json.Unmarshal(estadoConEspacio, &decodificadoGenerico); !errors.Is(err, ErrEstadoPersistibleFuenteAutoridadInvalido) {
		t.Fatalf("json.Unmarshal eludió la barrera canónica: %v", err)
	}

	contenido := borrador.Contenido
	contenido.Nombre += " con persistencia comprobada"
	editada, err := borrador.ActualizarBorrador(
		borrador.Revision, contenido, "per_editor_persistencia_v1_000001", "revision_persistencia_v1",
		borrador.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("ActualizarBorrador() error = %v", err)
	}
	publicadaEn := borrador.CreadaEn.Add(2 * time.Hour)
	actorPublicador := "per_publicador_persistencia_v1_001"
	motivoPublicacion := CodigoMotivoFuenteAutoridad("publicacion_persistencia_v1")
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, editada, EstadoFuenteAutoridadPublicada, actorPublicador, motivoPublicacion,
		"d", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	original, err := editada.AplicarTransicionV1(solicitud, evidencia, evidencia.ComprobadaEn)
	if err != nil {
		t.Fatalf("AplicarTransicionV1() error = %v", err)
	}
	estado, err := original.EstadoPersistibleV1()
	if err != nil {
		t.Fatalf("EstadoPersistibleV1() completo error = %v", err)
	}
	if !bytes.Contains(estado, []byte(`"ediciones_borrador":[{`)) ||
		!bytes.Contains(estado, []byte(`"transiciones":[{`)) ||
		!bytes.Contains(estado, []byte(`"firmas_refs":["firma:presidencia:d","firma:secretaria:d"]`)) {
		t.Fatalf("el estado completo no contiene todos sus anidados canonicos: %s", estado)
	}

	enLimite := original
	enLimite.Transiciones = append([]TransicionFuenteAutoridad(nil), original.Transiciones...)
	enLimite.Transiciones[0].RegistradaEn = enLimite.Transiciones[0].ExpiraEn
	if !errors.Is(enLimite.Validar(), ErrFuenteAutoridadInvalida) {
		t.Fatal("el agregado aceptó RegistradaEn igual al límite exclusivo ExpiraEn")
	}
	registroOriginal := []byte(`"registrada_en":"` +
		textoInstantePersistibleAutoridadV1(original.Transiciones[0].RegistradaEn) + `"`)
	registroEnLimite := []byte(`"registrada_en":"` +
		textoInstantePersistibleAutoridadV1(original.Transiciones[0].ExpiraEn) + `"`)
	if bytes.Count(estado, registroOriginal) != 1 {
		t.Fatalf("el vector no identifica un único registro de transición: %s", estado)
	}
	estadoEnLimite := bytes.Replace(estado, registroOriginal, registroEnLimite, 1)
	if _, err := RehidratarFuenteAutoridadV1(estadoEnLimite); !errors.Is(err, ErrEstadoPersistibleFuenteAutoridadInvalido) {
		t.Fatalf("la rehidratación aceptó el límite exclusivo: %v", err)
	}

	rehidratada, err := RehidratarFuenteAutoridadV1(estado)
	if err != nil {
		t.Fatalf("RehidratarFuenteAutoridadV1() error = %v", err)
	}
	repetido, err := rehidratada.EstadoPersistibleV1()
	if err != nil {
		t.Fatalf("EstadoPersistibleV1() tras rehidratar error = %v", err)
	}
	if !bytes.Equal(estado, repetido) {
		t.Fatalf("la ida y vuelta altero el estado\nantes:   %s\ndespues: %s", estado, repetido)
	}

	huellaContenidoOriginal, err := original.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaContenidoRehidratada, err := rehidratada.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstadoOriginal, err := original.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstadoRehidratada, err := rehidratada.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if huellaContenidoOriginal != huellaContenidoRehidratada || huellaEstadoOriginal != huellaEstadoRehidratada {
		t.Fatalf("la rehidratacion cambio huellas: contenido %s/%s; estado %s/%s",
			huellaContenidoOriginal, huellaContenidoRehidratada, huellaEstadoOriginal, huellaEstadoRehidratada)
	}
	if huellaEstadoOriginal != huellaBytesFuenteAutoridad(estado) {
		t.Fatalf("HuellaEstadoSHA256 no usa exactamente EstadoPersistibleV1: %s", huellaEstadoOriginal)
	}
}

func TestRehidratarFuenteAutoridadV1RechazaJSONNoCanonico(t *testing.T) {
	estado, err := nuevaFuenteAutoridadPrueba(t).EstadoPersistibleV1()
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string][]byte{
		"espacio inicial":      append([]byte(" "), estado...),
		"salto final":          append(append([]byte(nil), estado...), '\n'),
		"segundo valor JSON":   append(append([]byte(nil), estado...), []byte(`{}`)...),
		"campo desconocido":    bytes.Replace(estado, []byte(`{`), []byte(`{"desconocido":true,`), 1),
		"campo repetido":       bytes.Replace(estado, []byte(`{"esquema":`), []byte(`{"id":"reglamento_bolsas_2026","esquema":`), 1),
		"esquema desconocido":  bytes.Replace(estado, []byte(esquemaEstadoPersistibleFuenteAutoridadV1), []byte("vec.fuente_autoridad.estado.v2"), 1),
		"version desconocida":  bytes.Replace(estado, []byte(`"formato_version":1`), []byte(`"formato_version":2`), 1),
		"lista nula":           bytes.Replace(estado, []byte(`"ediciones_borrador":[]`), []byte(`"ediciones_borrador":null`), 1),
		"instante no canonico": bytes.Replace(estado, []byte(`2026-07-17T09:30:00.123456Z`), []byte(`2026-07-17T09:30:00.123456+00:00`), 1),
		"clave con otro orden": moverPrimerCampoPersistibleV1AlFinal(t, estado),
		"entero desbordado":    bytes.Replace(estado, []byte(`"version":1`), []byte(`"version":18446744073709551616`), 1),
	}
	for nombre, alterado := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := RehidratarFuenteAutoridadV1(alterado); !errors.Is(err, ErrEstadoPersistibleFuenteAutoridadInvalido) {
				t.Fatalf("error = %v; se esperaba estado persistible invalido", err)
			}
		})
	}
}

func TestRehidratarFuenteAutoridadV1ExigeOrdenCanonicoDeListas(t *testing.T) {
	estado, err := nuevaFuenteAutoridadPrueba(t).EstadoPersistibleV1()
	if err != nil {
		t.Fatal(err)
	}
	canonico := []byte(`["personal_funcionario","personal_laboral"]`)
	noCanonico := []byte(`["personal_laboral","personal_funcionario"]`)
	alterado := bytes.Replace(estado, canonico, noCanonico, 1)
	if bytes.Equal(estado, alterado) {
		t.Fatalf("el vector de prueba no contenia la lista esperada: %s", estado)
	}
	if _, err := RehidratarFuenteAutoridadV1(alterado); !errors.Is(err, ErrEstadoPersistibleFuenteAutoridadInvalido) {
		t.Fatalf("una lista desordenada fue aceptada: %v", err)
	}
}

func TestRehidratarFuenteAutoridadV1AplicaLimiteAntesDeDecodificar(t *testing.T) {
	demasiadoGrande := make([]byte, maximoBytesEstadoAutoridad+1)
	if _, err := RehidratarFuenteAutoridadV1(demasiadoGrande); !errors.Is(err, ErrEstadoPersistibleFuenteAutoridadInvalido) {
		t.Fatalf("estado por encima del limite aceptado: %v", err)
	}
}

func TestRehidratarFuenteAutoridadV1CortaArraysPatologicosAntesDeAsignar(t *testing.T) {
	estado, err := nuevaFuenteAutoridadPrueba(t).EstadoPersistibleV1()
	if err != nil {
		t.Fatal(err)
	}
	arrayExcesivo := []byte("[" + strings.Repeat("{},", maximoEdicionesFuenteAutoridad) + "{}]")
	alterado := bytes.Replace(estado, []byte(`"ediciones_borrador":[]`),
		append([]byte(`"ediciones_borrador":`), arrayExcesivo...), 1)
	if validarEstructuraJSONAutoridadV1(alterado) == nil {
		t.Fatal("la prelectura aceptó el array excesivo")
	}
	if _, err := RehidratarFuenteAutoridadV1(alterado); !errors.Is(err, ErrEstadoPersistibleFuenteAutoridadInvalido) {
		t.Fatalf("array excesivo aceptado: %v", err)
	}
}

// moverPrimerCampoPersistibleV1AlFinal conserva un JSON valido y demuestra
// que V1 fija tambien el orden de sus campos, no solo su valor semantico.
func moverPrimerCampoPersistibleV1AlFinal(t *testing.T, estado []byte) []byte {
	t.Helper()
	prefijo := `{"esquema":"` + esquemaEstadoPersistibleFuenteAutoridadV1 + `",`
	if !strings.HasPrefix(string(estado), prefijo) || len(estado) < len(prefijo)+1 || estado[len(estado)-1] != '}' {
		t.Fatalf("estado canonico inesperado: %s", estado)
	}
	campo := strings.TrimSuffix(prefijo, ",")
	resto := string(estado[len(prefijo) : len(estado)-1])
	return []byte("{" + resto + "," + campo[1:] + "}")
}
