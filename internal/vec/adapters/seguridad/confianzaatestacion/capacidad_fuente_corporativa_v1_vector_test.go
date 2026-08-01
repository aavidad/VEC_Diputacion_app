package confianzaatestacion

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

const (
	dominioMACFuenteCorporativaV1            = "VEC-CONTEXTO-ACTOR-FUENTE-CORPORATIVA-V1"
	huellaConsumoFuenteCorporativaV1Esperada = "0755995c42bdbdf7de83d6066c3b17c3f95534bc17de35a5d91f43560b3f1e85"
)

type manifiestoFuenteCorporativaV1Vector struct {
	Esquema                  string `json:"esquema"`
	Version                  uint64 `json:"version"`
	FuenteRef                string `json:"fuente_ref"`
	FuenteVersion            uint64 `json:"fuente_version"`
	EventoFuenteRef          string `json:"evento_fuente_ref"`
	HuellaEventoFuenteSHA256 string `json:"huella_evento_fuente_sha256"`
	EventoFuenteEmitidoEn    string `json:"evento_fuente_emitido_en"`
	AudienciaConsumo         string `json:"audiencia_consumo"`
	Accion                   string `json:"accion"`
	TipoEfecto               string `json:"tipo_efecto"`
	OperacionRef             string `json:"operacion_ref"`
	EfectoRef                string `json:"efecto_ref"`
	HuellaEfectoSHA256       string `json:"huella_efecto_sha256"`
}

type capacidadFuenteCorporativaV1Vector struct {
	Esquema                      string `json:"esquema"`
	Version                      uint64 `json:"version"`
	FuenteRef                    string `json:"fuente_ref"`
	FuenteVersion                uint64 `json:"fuente_version"`
	EventoFuenteRef              string `json:"evento_fuente_ref"`
	HuellaEventoFuenteSHA256     string `json:"huella_evento_fuente_sha256"`
	EventoFuenteEmitidoEn        string `json:"evento_fuente_emitido_en"`
	HuellaManifiestoFuenteSHA256 string `json:"huella_manifiesto_fuente_sha256"`
	HuellaSobreCOSESign1SHA256   string `json:"huella_sobre_cose_sign1_sha256"`
	HuellaPruebaConfianzaSHA256  string `json:"huella_prueba_confianza_sha256"`
	AudienciaConsumo             string `json:"audiencia_consumo"`
	Accion                       string `json:"accion"`
	TipoEfecto                   string `json:"tipo_efecto"`
	OperacionRef                 string `json:"operacion_ref"`
	EfectoRef                    string `json:"efecto_ref"`
	HuellaEfectoSHA256           string `json:"huella_efecto_sha256"`
	ClaveID                      string `json:"clave_id"`
	ClaveVersion                 uint64 `json:"clave_version"`
	RevisionGobierno             uint64 `json:"revision_gobierno"`
	HuellaGobiernoSHA256         string `json:"huella_gobierno_sha256"`
	EmisorID                     string `json:"emisor_id"`
	ConfiguracionRevision        string `json:"configuracion_revision"`
	ConfiguracionSecuencia       uint64 `json:"configuracion_secuencia"`
	HuellaConfiguracionSHA256    string `json:"huella_configuracion_sha256"`
	RaizClaveID                  string `json:"raiz_clave_id"`
	RaizVersion                  uint64 `json:"raiz_version"`
	HuellaRaizSPKISHA256         string `json:"huella_raiz_spki_sha256"`
	AudienciaDespliegue          string `json:"audiencia_despliegue"`
	Suite                        string `json:"suite"`
	Nonce                        string `json:"nonce"`
	EmitidaEn                    string `json:"emitida_en"`
	ExpiraEn                     string `json:"expira_en"`
	MACSHA256                    string `json:"mac_sha256"`
}

func (c capacidadFuenteCorporativaV1Vector) valoresMAC() []string {
	return []string{
		c.Esquema,
		strconv.FormatUint(c.Version, 10),
		c.FuenteRef,
		strconv.FormatUint(c.FuenteVersion, 10),
		c.EventoFuenteRef,
		c.HuellaEventoFuenteSHA256,
		c.EventoFuenteEmitidoEn,
		c.HuellaManifiestoFuenteSHA256,
		c.HuellaSobreCOSESign1SHA256,
		c.HuellaPruebaConfianzaSHA256,
		c.AudienciaConsumo,
		c.Accion,
		c.TipoEfecto,
		c.OperacionRef,
		c.EfectoRef,
		c.HuellaEfectoSHA256,
		c.ClaveID,
		strconv.FormatUint(c.ClaveVersion, 10),
		strconv.FormatUint(c.RevisionGobierno, 10),
		c.HuellaGobiernoSHA256,
		c.EmisorID,
		c.ConfiguracionRevision,
		strconv.FormatUint(c.ConfiguracionSecuencia, 10),
		c.HuellaConfiguracionSHA256,
		c.RaizClaveID,
		strconv.FormatUint(c.RaizVersion, 10),
		c.HuellaRaizSPKISHA256,
		c.AudienciaDespliegue,
		c.Suite,
		c.Nonce,
		c.EmitidaEn,
		c.ExpiraEn,
	}
}

type consumoFuenteCorporativaV1Vector struct {
	Esquema                      string `json:"esquema"`
	Version                      uint64 `json:"version"`
	CapacidadRef                 string `json:"capacidad_ref"`
	FuenteRef                    string `json:"fuente_ref"`
	FuenteVersion                uint64 `json:"fuente_version"`
	EventoFuenteRef              string `json:"evento_fuente_ref"`
	HuellaEventoFuenteSHA256     string `json:"huella_evento_fuente_sha256"`
	EventoFuenteEmitidoEn        string `json:"evento_fuente_emitido_en"`
	HuellaManifiestoFuenteSHA256 string `json:"huella_manifiesto_fuente_sha256"`
	HuellaSobreCOSESign1SHA256   string `json:"huella_sobre_cose_sign1_sha256"`
	HuellaPruebaConfianzaSHA256  string `json:"huella_prueba_confianza_sha256"`
	AudienciaConsumo             string `json:"audiencia_consumo"`
	Accion                       string `json:"accion"`
	TipoEfecto                   string `json:"tipo_efecto"`
	OperacionRef                 string `json:"operacion_ref"`
	EfectoRef                    string `json:"efecto_ref"`
	HuellaEfectoSHA256           string `json:"huella_efecto_sha256"`
	ClaveID                      string `json:"clave_id"`
	ClaveVersion                 uint64 `json:"clave_version"`
	RevisionGobierno             uint64 `json:"revision_gobierno"`
	HuellaGobiernoSHA256         string `json:"huella_gobierno_sha256"`
	EmisorID                     string `json:"emisor_id"`
	ConfiguracionRevision        string `json:"configuracion_revision"`
	ConfiguracionSecuencia       uint64 `json:"configuracion_secuencia"`
	HuellaConfiguracionSHA256    string `json:"huella_configuracion_sha256"`
	RaizClaveID                  string `json:"raiz_clave_id"`
	RaizVersion                  uint64 `json:"raiz_version"`
	HuellaRaizSPKISHA256         string `json:"huella_raiz_spki_sha256"`
	AudienciaDespliegue          string `json:"audiencia_despliegue"`
	Suite                        string `json:"suite"`
	Nonce                        string `json:"nonce"`
	EmitidaEn                    string `json:"emitida_en"`
	ExpiraEn                     string `json:"expira_en"`
	MACSHA256                    string `json:"mac_sha256"`
	ConsumidaEn                  string `json:"consumida_en"`
}

func TestVectoresFuenteCorporativaV1SonCanonicosYCoherentes(t *testing.T) {
	t.Parallel()
	manifiestoBytes, manifiesto := leerVectorCerrado[manifiestoFuenteCorporativaV1Vector](
		t,
		"manifiesto_fuente_corporativa_v1.json",
	)
	capacidadBytes, capacidad := leerVectorCerrado[capacidadFuenteCorporativaV1Vector](
		t,
		"capacidad_fuente_corporativa_v1.json",
	)
	consumoBytes, consumo := leerVectorCerrado[consumoFuenteCorporativaV1Vector](
		t,
		"consumo_fuente_corporativa_v1.json",
	)

	if manifiesto.Esquema != "vec.contexto-actor.fuente-corporativa.manifiesto.v1" ||
		manifiesto.Version != 1 ||
		capacidad.Esquema != "vec.contexto-actor.fuente-corporativa.capacidad.v1" ||
		capacidad.Version != 1 {
		t.Fatal("los esquemas o versiones de los vectores F0 no son los aprobados")
	}
	if manifiesto.FuenteRef != capacidad.FuenteRef ||
		manifiesto.FuenteVersion != capacidad.FuenteVersion ||
		manifiesto.EventoFuenteRef != capacidad.EventoFuenteRef ||
		manifiesto.HuellaEventoFuenteSHA256 != capacidad.HuellaEventoFuenteSHA256 ||
		manifiesto.EventoFuenteEmitidoEn != capacidad.EventoFuenteEmitidoEn ||
		manifiesto.AudienciaConsumo != capacidad.AudienciaConsumo ||
		manifiesto.Accion != capacidad.Accion ||
		manifiesto.TipoEfecto != capacidad.TipoEfecto ||
		manifiesto.OperacionRef != capacidad.OperacionRef ||
		manifiesto.EfectoRef != capacidad.EfectoRef ||
		manifiesto.HuellaEfectoSHA256 != capacidad.HuellaEfectoSHA256 {
		t.Fatal("el manifiesto y la capacidad no conservan la misma aserción atómica")
	}
	if obtenida := huellaHexFuenteCorporativaV1(manifiestoBytes); obtenida != capacidad.HuellaManifiestoFuenteSHA256 {
		t.Fatalf("huella del manifiesto: obtenida=%s", obtenida)
	}

	materialHMAC := sha256.Sum256([]byte("material-hmac-sintetico-f0-v1"))
	defer clear(materialHMAC[:])
	preimagen := preimagenMACFuenteCorporativaV1(capacidad)
	defer clear(preimagen)
	calculador := hmac.New(sha256.New, materialHMAC[:])
	_, _ = calculador.Write(preimagen)
	macObtenido := hex.EncodeToString(calculador.Sum(nil))
	if macObtenido != capacidad.MACSHA256 {
		t.Fatalf("MAC sintético: obtenido=%s", macObtenido)
	}

	capacidadRef := "cfc_" + huellaHexFuenteCorporativaV1(capacidadBytes)
	consumoEsperado := consumoDesdeCapacidadFuenteCorporativaV1(
		capacidad,
		capacidadRef,
		consumo.ConsumidaEn,
	)
	if consumo != consumoEsperado {
		t.Fatalf("canon de consumo no reproduce la capacidad; capacidad_ref=%s", capacidadRef)
	}
	if obtenida := huellaHexFuenteCorporativaV1(consumoBytes); obtenida != huellaConsumoFuenteCorporativaV1Esperada {
		t.Fatalf("huella del consumo: obtenida=%s", obtenida)
	}
}

func leerVectorCerrado[T any](t *testing.T, nombre string) ([]byte, T) {
	t.Helper()
	var documento T
	contenido, err := os.ReadFile(filepath.Join("testdata", nombre))
	if err != nil {
		t.Fatalf("leer vector %s: %v", nombre, err)
	}
	if bytes.HasSuffix(contenido, []byte{'\r', '\n'}) {
		t.Fatalf("el vector %s usa un final de línea no canónico", nombre)
	}
	contenido = bytes.TrimSuffix(contenido, []byte{'\n'})
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&documento); err != nil {
		t.Fatalf("decodificar vector cerrado %s: %v", nombre, err)
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("el vector %s contiene datos posteriores", nombre)
	}
	canon, err := json.Marshal(documento)
	if err != nil {
		t.Fatalf("serializar vector %s: %v", nombre, err)
	}
	if !bytes.Equal(canon, contenido) {
		t.Fatalf("el vector %s no conserva orden, tipos o bytes canónicos", nombre)
	}
	for _, prohibido := range [][]byte{
		[]byte("clave_hmac"),
		[]byte("secreto_hmac"),
		[]byte("material_hmac"),
	} {
		if bytes.Contains(contenido, prohibido) {
			t.Fatalf("el vector %s contiene material HMAC", nombre)
		}
	}
	return contenido, documento
}

func preimagenMACFuenteCorporativaV1(
	capacidad capacidadFuenteCorporativaV1Vector,
) []byte {
	valores := capacidad.valoresMAC()
	if len(valores) != 32 {
		panic("contrato F0: la preimagen MAC exige 32 valores")
	}
	var salida bytes.Buffer
	salida.WriteString(dominioMACFuenteCorporativaV1)
	for _, valor := range valores {
		salida.WriteString(strconv.Itoa(len([]byte(valor))))
		salida.WriteByte(':')
		salida.WriteString(valor)
		salida.WriteByte('\n')
	}
	return salida.Bytes()
}

func huellaHexFuenteCorporativaV1(contenido []byte) string {
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:])
}

func consumoDesdeCapacidadFuenteCorporativaV1(
	capacidad capacidadFuenteCorporativaV1Vector,
	capacidadRef string,
	consumidaEn string,
) consumoFuenteCorporativaV1Vector {
	return consumoFuenteCorporativaV1Vector{
		Esquema:                      "vec.contexto-actor.fuente-corporativa.consumo.v1",
		Version:                      1,
		CapacidadRef:                 capacidadRef,
		FuenteRef:                    capacidad.FuenteRef,
		FuenteVersion:                capacidad.FuenteVersion,
		EventoFuenteRef:              capacidad.EventoFuenteRef,
		HuellaEventoFuenteSHA256:     capacidad.HuellaEventoFuenteSHA256,
		EventoFuenteEmitidoEn:        capacidad.EventoFuenteEmitidoEn,
		HuellaManifiestoFuenteSHA256: capacidad.HuellaManifiestoFuenteSHA256,
		HuellaSobreCOSESign1SHA256:   capacidad.HuellaSobreCOSESign1SHA256,
		HuellaPruebaConfianzaSHA256:  capacidad.HuellaPruebaConfianzaSHA256,
		AudienciaConsumo:             capacidad.AudienciaConsumo,
		Accion:                       capacidad.Accion,
		TipoEfecto:                   capacidad.TipoEfecto,
		OperacionRef:                 capacidad.OperacionRef,
		EfectoRef:                    capacidad.EfectoRef,
		HuellaEfectoSHA256:           capacidad.HuellaEfectoSHA256,
		ClaveID:                      capacidad.ClaveID,
		ClaveVersion:                 capacidad.ClaveVersion,
		RevisionGobierno:             capacidad.RevisionGobierno,
		HuellaGobiernoSHA256:         capacidad.HuellaGobiernoSHA256,
		EmisorID:                     capacidad.EmisorID,
		ConfiguracionRevision:        capacidad.ConfiguracionRevision,
		ConfiguracionSecuencia:       capacidad.ConfiguracionSecuencia,
		HuellaConfiguracionSHA256:    capacidad.HuellaConfiguracionSHA256,
		RaizClaveID:                  capacidad.RaizClaveID,
		RaizVersion:                  capacidad.RaizVersion,
		HuellaRaizSPKISHA256:         capacidad.HuellaRaizSPKISHA256,
		AudienciaDespliegue:          capacidad.AudienciaDespliegue,
		Suite:                        capacidad.Suite,
		Nonce:                        capacidad.Nonce,
		EmitidaEn:                    capacidad.EmitidaEn,
		ExpiraEn:                     capacidad.ExpiraEn,
		MACSHA256:                    capacidad.MACSHA256,
		ConsumidaEn:                  consumidaEn,
	}
}
