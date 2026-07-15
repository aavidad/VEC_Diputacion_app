package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

type verificadorAtestacionPrueba struct{}

func (verificadorAtestacionPrueba) VerificarAtestacionConsumoReciboCargaDirecta(
	context.Context,
	ports.ContextoOperacionAlmacen,
	string,
	ports.ComprobanteConsumoReciboCargaDirecta,
) error {
	return nil
}

type objetoFalso struct {
	contenido []byte
	version   string
	mime      string
	checksum  string
	metadatos map[string]string
	creadoEn  time.Time
	cifrado   types.ServerSideEncryption
	claveKMS  string
	bucketKey bool
	retencion *types.ObjectLockRetention
	hold      types.ObjectLockLegalHoldStatus
}

type clienteFalso struct {
	mu                sync.Mutex
	objetos           map[string]map[string]*objetoFalso
	actuales          map[string]string
	secuencia         uint64
	puts              int
	gets              int
	deletes           int
	deleteRechazados  int
	ignorarRetencion  bool
	ignorarHold       bool
	omitirChecksum    bool
	omitirVersion     bool
	manipularChecksum bool
	fallarDelete      bool
	ignorarBucketKey  bool
}

func nuevoClienteFalso() *clienteFalso {
	return &clienteFalso{objetos: map[string]map[string]*objetoFalso{}, actuales: map[string]string{}}
}

func (*clienteFalso) HeadBucket(context.Context, *awss3.HeadBucketInput, ...func(*awss3.Options)) (*awss3.HeadBucketOutput, error) {
	return &awss3.HeadBucketOutput{}, nil
}

func (*clienteFalso) GetBucketVersioning(context.Context, *awss3.GetBucketVersioningInput, ...func(*awss3.Options)) (*awss3.GetBucketVersioningOutput, error) {
	return &awss3.GetBucketVersioningOutput{Status: types.BucketVersioningStatusEnabled}, nil
}

func (*clienteFalso) GetObjectLockConfiguration(context.Context, *awss3.GetObjectLockConfigurationInput, ...func(*awss3.Options)) (*awss3.GetObjectLockConfigurationOutput, error) {
	return &awss3.GetObjectLockConfigurationOutput{ObjectLockConfiguration: &types.ObjectLockConfiguration{
		ObjectLockEnabled: types.ObjectLockEnabledEnabled,
	}}, nil
}

func (c *clienteFalso) PutObject(_ context.Context, entrada *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.puts++
	indice := indiceFalso(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key))
	if awsv2.ToString(entrada.IfNoneMatch) == "*" && c.actuales[indice] != "" {
		return nil, &smithy.GenericAPIError{Code: "PreconditionFailed", Message: "redactado"}
	}
	contenido, err := io.ReadAll(entrada.Body)
	if err != nil {
		return nil, err
	}
	if entrada.ContentLength == nil || int64(len(contenido)) != awsv2.ToInt64(entrada.ContentLength) {
		return nil, &smithy.GenericAPIError{Code: "IncompleteBody", Message: "redactado"}
	}
	suma := sha256.Sum256(contenido)
	checksum := base64.StdEncoding.EncodeToString(suma[:])
	if checksum != awsv2.ToString(entrada.ChecksumSHA256) {
		return nil, &smithy.GenericAPIError{Code: "BadDigest", Message: "redactado"}
	}
	c.secuencia++
	version := "v-" + stringValor(c.secuencia)
	objeto := &objetoFalso{
		contenido: append([]byte(nil), contenido...), version: version, mime: awsv2.ToString(entrada.ContentType),
		checksum: checksum, metadatos: clonarMapa(entrada.Metadata), creadoEn: time.Now().UTC(),
		cifrado: entrada.ServerSideEncryption, claveKMS: awsv2.ToString(entrada.SSEKMSKeyId),
		bucketKey: awsv2.ToBool(entrada.BucketKeyEnabled) && !c.ignorarBucketKey, hold: types.ObjectLockLegalHoldStatusOff,
	}
	if entrada.ObjectLockLegalHoldStatus != "" && !c.ignorarHold {
		objeto.hold = entrada.ObjectLockLegalHoldStatus
	}
	if entrada.ObjectLockRetainUntilDate != nil && !c.ignorarRetencion {
		objeto.retencion = &types.ObjectLockRetention{
			Mode:            types.ObjectLockRetentionMode(entrada.ObjectLockMode),
			RetainUntilDate: awsv2.Time(entrada.ObjectLockRetainUntilDate.UTC()),
		}
	}
	if c.objetos[indice] == nil {
		c.objetos[indice] = map[string]*objetoFalso{}
	}
	c.objetos[indice][version] = objeto
	c.actuales[indice] = version
	respuesta := &awss3.PutObjectOutput{ChecksumSHA256: awsv2.String(checksum), VersionId: awsv2.String(version),
		ServerSideEncryption: entrada.ServerSideEncryption, SSEKMSKeyId: entrada.SSEKMSKeyId,
		BucketKeyEnabled: awsv2.Bool(objeto.bucketKey)}
	if c.omitirChecksum {
		respuesta.ChecksumSHA256 = nil
	}
	if c.omitirVersion {
		respuesta.VersionId = nil
	}
	return respuesta, nil
}

func (c *clienteFalso) HeadObject(_ context.Context, entrada *awss3.HeadObjectInput, _ ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), awsv2.ToString(entrada.VersionId))
	if err != nil {
		return nil, err
	}
	checksum := objeto.checksum
	if c.manipularChecksum {
		checksum = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, 32))
	}
	respuesta := &awss3.HeadObjectOutput{
		ContentLength: awsv2.Int64(int64(len(objeto.contenido))), ContentType: awsv2.String(objeto.mime),
		ChecksumSHA256: awsv2.String(checksum), Metadata: clonarMapa(objeto.metadatos), LastModified: awsv2.Time(objeto.creadoEn),
		VersionId: awsv2.String(objeto.version), ServerSideEncryption: objeto.cifrado, SSEKMSKeyId: awsv2.String(objeto.claveKMS),
		BucketKeyEnabled: awsv2.Bool(objeto.bucketKey), ObjectLockLegalHoldStatus: objeto.hold,
	}
	if objeto.retencion != nil && objeto.retencion.RetainUntilDate != nil {
		respuesta.ObjectLockRetainUntilDate = awsv2.Time(*objeto.retencion.RetainUntilDate)
		respuesta.ObjectLockMode = types.ObjectLockMode(objeto.retencion.Mode)
	}
	if c.omitirChecksum {
		respuesta.ChecksumSHA256 = nil
	}
	if c.omitirVersion {
		respuesta.VersionId = nil
	}
	return respuesta, nil
}

func (c *clienteFalso) GetObject(_ context.Context, entrada *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets++
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), awsv2.ToString(entrada.VersionId))
	if err != nil {
		return nil, err
	}
	return &awss3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(append([]byte(nil), objeto.contenido...))),
		ContentLength: awsv2.Int64(int64(len(objeto.contenido))), ContentType: awsv2.String(objeto.mime),
		ChecksumSHA256: awsv2.String(objeto.checksum), VersionId: awsv2.String(objeto.version),
		Metadata: clonarMapa(objeto.metadatos), ServerSideEncryption: objeto.cifrado, SSEKMSKeyId: awsv2.String(objeto.claveKMS),
		BucketKeyEnabled: awsv2.Bool(objeto.bucketKey),
	}, nil
}

func (c *clienteFalso) PutObjectRetention(_ context.Context, entrada *awss3.PutObjectRetentionInput, _ ...func(*awss3.Options)) (*awss3.PutObjectRetentionOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), awsv2.ToString(entrada.VersionId))
	if err != nil {
		return nil, err
	}
	if !c.ignorarRetencion && entrada.Retention != nil {
		copia := *entrada.Retention
		objeto.retencion = &copia
	}
	return &awss3.PutObjectRetentionOutput{}, nil
}

func (c *clienteFalso) GetObjectRetention(_ context.Context, entrada *awss3.GetObjectRetentionInput, _ ...func(*awss3.Options)) (*awss3.GetObjectRetentionOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), awsv2.ToString(entrada.VersionId))
	if err != nil {
		return nil, err
	}
	respuesta := &awss3.GetObjectRetentionOutput{}
	if objeto.retencion != nil {
		copia := *objeto.retencion
		respuesta.Retention = &copia
	}
	return respuesta, nil
}

func (c *clienteFalso) PutObjectLegalHold(_ context.Context, entrada *awss3.PutObjectLegalHoldInput, _ ...func(*awss3.Options)) (*awss3.PutObjectLegalHoldOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), awsv2.ToString(entrada.VersionId))
	if err != nil {
		return nil, err
	}
	if !c.ignorarHold && entrada.LegalHold != nil {
		objeto.hold = entrada.LegalHold.Status
	}
	return &awss3.PutObjectLegalHoldOutput{}, nil
}

func (c *clienteFalso) GetObjectLegalHold(_ context.Context, entrada *awss3.GetObjectLegalHoldInput, _ ...func(*awss3.Options)) (*awss3.GetObjectLegalHoldOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), awsv2.ToString(entrada.VersionId))
	if err != nil {
		return nil, err
	}
	return &awss3.GetObjectLegalHoldOutput{LegalHold: &types.ObjectLockLegalHold{Status: objeto.hold}}, nil
}

func (c *clienteFalso) DeleteObject(_ context.Context, entrada *awss3.DeleteObjectInput, _ ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deletes++
	if c.fallarDelete {
		return nil, &smithy.GenericAPIError{Code: "InternalError", Message: "redactado"}
	}
	indice := indiceFalso(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key))
	version := awsv2.ToString(entrada.VersionId)
	objeto, err := c.buscar(awsv2.ToString(entrada.Bucket), awsv2.ToString(entrada.Key), version)
	if err != nil {
		return nil, err
	}
	if (!c.ignorarHold && objeto.hold == types.ObjectLockLegalHoldStatusOn) ||
		(!c.ignorarRetencion && objeto.retencion != nil && objeto.retencion.RetainUntilDate != nil && objeto.retencion.RetainUntilDate.After(time.Now().UTC())) {
		c.deleteRechazados++
		return nil, &smithy.GenericAPIError{Code: "AccessDenied", Message: "redactado"}
	}
	delete(c.objetos[indice], version)
	if c.actuales[indice] == version {
		c.actuales[indice] = ""
	}
	return &awss3.DeleteObjectOutput{VersionId: awsv2.String(version)}, nil
}

func (c *clienteFalso) buscar(bucket, clave, version string) (*objetoFalso, error) {
	indice := indiceFalso(bucket, clave)
	if version == "" {
		version = c.actuales[indice]
	}
	if version == "" || c.objetos[indice] == nil || c.objetos[indice][version] == nil {
		return nil, &smithy.GenericAPIError{Code: "NoSuchKey", Message: "redactado"}
	}
	return c.objetos[indice][version], nil
}

type presignadorFalso struct {
	mu      sync.Mutex
	entrada *awss3.PutObjectInput
}

func (p *presignadorFalso) PresignPutObject(_ context.Context, entrada *awss3.PutObjectInput, opciones ...func(*awss3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entrada = entrada
	configuracion := &awss3.PresignOptions{}
	for _, opcion := range opciones {
		opcion(configuracion)
	}
	cabeceras := http.Header{}
	cabeceras.Set("Content-Length", strconv.FormatInt(awsv2.ToInt64(entrada.ContentLength), 10))
	cabeceras.Set("Content-Type", awsv2.ToString(entrada.ContentType))
	cabeceras.Set("If-None-Match", awsv2.ToString(entrada.IfNoneMatch))
	cabeceras.Set("X-Amz-Checksum-Sha256", awsv2.ToString(entrada.ChecksumSHA256))
	cabeceras.Set("X-Amz-Sdk-Checksum-Algorithm", "SHA256")
	cabeceras.Set("X-Amz-Server-Side-Encryption", string(entrada.ServerSideEncryption))
	if entrada.SSEKMSKeyId != nil {
		cabeceras.Set("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id", awsv2.ToString(entrada.SSEKMSKeyId))
	}
	if awsv2.ToBool(entrada.BucketKeyEnabled) {
		cabeceras.Set("X-Amz-Server-Side-Encryption-Bucket-Key-Enabled", "true")
	}
	for clave, valor := range entrada.Metadata {
		cabeceras.Set("X-Amz-Meta-"+clave, valor)
	}
	nombres := make([]string, 0, len(cabeceras)+1)
	for nombre := range cabeceras {
		nombres = append(nombres, strings.ToLower(nombre))
	}
	nombres = append(nombres, "host")
	sort.Strings(nombres)
	consulta := url.Values{}
	consulta.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	consulta.Set("X-Amz-Checksum-Sha256", awsv2.ToString(entrada.ChecksumSHA256))
	consulta.Set("X-Amz-Sdk-Checksum-Algorithm", "SHA256")
	consulta.Set("X-Amz-Expires", stringValor(uint64(configuracion.Expires/time.Second)))
	consulta.Set("X-Amz-SignedHeaders", strings.Join(nombres, ";"))
	consulta.Set("X-Amz-Signature", "firma-opaca")
	return &v4.PresignedHTTPRequest{
		URL:    "https://objetos.interno.example/" + awsv2.ToString(entrada.Bucket) + "/" + awsv2.ToString(entrada.Key) + "?" + consulta.Encode(),
		Method: http.MethodPut, SignedHeader: cabeceras,
	}, nil
}

func configuracionPrueba(fuerte bool) Configuracion {
	return Configuracion{
		ConectorID: "s3_pruebas", Endpoint: "https://objetos.interno.example", Region: "eu-west-1",
		BucketCuarentena: "vec-cuarentena", BucketAdmitida: "vec-admitida", PathStyle: true,
		TamanoMaximo: 4 * 1024 * 1024, DuracionCargaDirecta: 5 * time.Minute,
		RetencionMinimaAdmitida: time.Hour,
		ClaveDerivacion:         bytes.Repeat([]byte{0x42}, 32), Cifrado: types.ServerSideEncryptionAes256,
		RedesPermitidas: []netip.Prefix{netip.MustParsePrefix("10.20.0.0/16")},
		PerfilFuerte:    fuerte, ProbarCapacidades: fuerte, ModoRetencion: types.ObjectLockRetentionModeCompliance,
	}
}

func TestSondaFuertePruebaPromocionRetencionYHoldDestructivos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatalf("crear almacen fuerte: %v", err)
	}
	capacidades, err := almacen.Capacidades(context.Background())
	if err != nil || !capacidades.Versionado || !capacidades.IntegridadSHA256 || !capacidades.Retencion ||
		!capacidades.BloqueoLegal || !capacidades.PromocionAtomica || !capacidades.PreservaObjetoOriginal ||
		!capacidades.CifradoPorObjeto || !capacidades.CargaDirectaTemporal {
		t.Fatalf("capacidades no probadas: %+v, %v", capacidades, err)
	}
	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	if cliente.deleteRechazados < 3 {
		t.Fatalf("la sonda no intento DELETE protegido por separado: %d", cliente.deleteRechazados)
	}
	for indice := range cliente.objetos {
		if strings.Contains(indice, "..") || strings.Contains(strings.Split(indice, "\x00")[1], "\\") {
			t.Fatalf("clave no canonica: %q", indice)
		}
	}
}

func TestSondaNoDeclaraWORMSiBackendIgnoraProteccion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	casos := []struct {
		nombre string
		mutar  func(*clienteFalso)
	}{
		{"retencion ignorada", func(c *clienteFalso) { c.ignorarRetencion = true }},
		{"hold ignorado", func(c *clienteFalso) { c.ignorarHold = true }},
		{"checksum ausente", func(c *clienteFalso) { c.omitirChecksum = true }},
		{"version ausente", func(c *clienteFalso) { c.omitirVersion = true }},
		{"bucket key KMS ignorada", func(c *clienteFalso) {
			c.ignorarBucketKey = true
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			cliente := nuevoClienteFalso()
			caso.mutar(cliente)
			configuracion := configuracionPrueba(true)
			if caso.nombre == "bucket key KMS ignorada" {
				configuracion.Cifrado = types.ServerSideEncryptionAwsKms
				configuracion.ClaveKMS = "kms://clave-documental-pruebas"
				configuracion.UsarBucketKeyKMS = true
			}
			if _, err := NuevoConCliente(context.Background(), configuracion, cliente, &presignadorFalso{}, relojPrueba{ahora}); !errors.Is(err, ErrSondaS3NoSuperada) {
				t.Fatalf("sonda debio cerrar: %v", err)
			}
		})
	}
}

func TestSinSondaDeniegaAntesDePut(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(false), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("documento de bolsa")
	solicitud := solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-sin-sonda")
	if _, err := almacen.Escribir(context.Background(), solicitud); !errors.Is(err, ports.ErrCapacidadAlmacenNoDisponible) {
		t.Fatalf("escritura no denegada: %v", err)
	}
	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	if cliente.puts != 0 {
		t.Fatalf("se hizo PUT antes de verificar capacidad: %d", cliente.puts)
	}
}

func TestContextoCanceladoDeniegaAntesDeCualquierEfecto(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	cliente.mu.Lock()
	putsAntes := cliente.puts
	cliente.mu.Unlock()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	contenido := []byte("no debe salir del proceso")
	if _, err := almacen.Escribir(ctx, solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-cancelada")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no conservada: %v", err)
	}
	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	if cliente.puts != putsAntes {
		t.Fatalf("se produjo un efecto remoto tras cancelar: %d -> %d", putsAntes, cliente.puts)
	}
}

func TestFalloDeAleatoriedadDeniegaAntesDeReservarIdempotencia(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	cliente.mu.Lock()
	putsAntes := cliente.puts
	cliente.mu.Unlock()
	lectorAnterior := rand.Reader
	rand.Reader = lectorQueFalla{}
	defer func() { rand.Reader = lectorAnterior }()
	contenido := []byte("no debe reservarse sin evidencia aleatoria")
	if _, err := almacen.Escribir(context.Background(), solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-sin-rng")); !errors.Is(err, ErrOperacionS3) {
		t.Fatalf("fallo RNG no cerrado: %v", err)
	}
	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	if cliente.puts != putsAntes {
		t.Fatalf("se reservo idempotencia sin evidencia: %d -> %d", putsAntes, cliente.puts)
	}
}

func TestEscrituraEsStreamingIdempotenteYNoUsaETag(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("documento de bolsa firmado")
	solicitud := solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-escritura-uno")
	primero, err := almacen.Escribir(context.Background(), solicitud)
	if err != nil || primero.ValidarEscritura(solicitud, almacen.capacidades) != nil {
		t.Fatalf("primera escritura: %+v, %v", primero, err)
	}
	solicitud.Contenido = bytes.NewReader(contenido)
	repetido, err := almacen.Escribir(context.Background(), solicitud)
	if err != nil || repetido.Objeto.Objeto != primero.Objeto.Objeto || !repetido.Evidencia.ReintentoIdempotente {
		t.Fatalf("reintento: %+v, %v", repetido, err)
	}
	alterado := append([]byte(nil), contenido...)
	alterado[0] ^= 0xff
	suma := sha256.Sum256(alterado)
	solicitud.Contenido = bytes.NewReader(alterado)
	solicitud.HuellaSHA256 = hex.EncodeToString(suma[:])
	if _, err := almacen.Escribir(context.Background(), solicitud); !errors.Is(err, ports.ErrIdempotenciaAlmacenReutilizada) {
		t.Fatalf("idempotencia cruzada aceptada: %v", err)
	}
}

func TestIdempotenciaEsGlobalEntreZonasYMarcadorEsInmutable(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("documento con identidad global")
	solicitud := solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-global-zonas")
	if _, err := almacen.Escribir(context.Background(), solicitud); err != nil {
		t.Fatalf("escritura inicial: %v", err)
	}
	referencia := almacen.referenciaIdempotente("escribir", solicitud.ClaveIdempotencia)
	if referencia != almacen.referenciaIdempotente("promover", solicitud.ClaveIdempotencia) ||
		referencia != almacen.referenciaIdempotente("carga-directa", solicitud.ClaveIdempotencia) {
		t.Fatal("la clave de idempotencia no comparte un espacio global")
	}
	solicitud.Zona = ports.ZonaAlmacenAdmitida
	solicitud.Contenido = bytes.NewReader(contenido)
	if _, err := almacen.Escribir(context.Background(), solicitud); !errors.Is(err, ports.ErrIdempotenciaAlmacenReutilizada) {
		t.Fatalf("reutilizacion cruzada aceptada: %v", err)
	}

	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	if cliente.actuales[indiceFalso(almacen.configuracion.BucketAdmitida, claveObjeto(referencia))] != "" {
		t.Fatal("la colision global produjo un objeto en la segunda zona")
	}
	marcador, errorMarcador := cliente.buscar(
		almacen.configuracion.BucketCuarentena,
		prefijoClaveIdempotencia+referencia,
		"",
	)
	if errorMarcador != nil || marcador.hold != types.ObjectLockLegalHoldStatusOn ||
		marcador.metadatos[metaEsquema] != esquemaIdempotencia ||
		bytes.Contains(marcador.contenido, []byte(solicitud.ClaveIdempotencia)) {
		t.Fatalf("marcador no durable u opaco: %#v, %v", marcador, errorMarcador)
	}
}

func TestEscriturasConcurrentesMismaPeticionProducenUnaVersionCanonica(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("contenido concurrente exacto")
	base := solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-concurrente")
	const trabajadores = 16
	inicio := make(chan struct{})
	resultados := make(chan ports.ResultadoOperacionObjeto, trabajadores)
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	grupo.Add(trabajadores)
	for i := 0; i < trabajadores; i++ {
		go func() {
			defer grupo.Done()
			<-inicio
			solicitud := base
			solicitud.Contenido = bytes.NewReader(contenido)
			resultado, err := almacen.Escribir(context.Background(), solicitud)
			resultados <- resultado
			errores <- err
		}()
	}
	close(inicio)
	grupo.Wait()
	close(resultados)
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("escritura concurrente: %v", err)
		}
	}
	var canonica ports.ReferenciaObjetoAlmacen
	for resultado := range resultados {
		if canonica.Validar() != nil {
			canonica = resultado.Objeto.Objeto
		}
		if resultado.Objeto.Objeto != canonica {
			t.Fatalf("se publicaron referencias distintas: %+v / %+v", canonica, resultado.Objeto.Objeto)
		}
	}
	referencia := almacen.referenciaIdempotente("escribir", base.ClaveIdempotencia)
	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	versiones := cliente.objetos[indiceFalso(almacen.configuracion.BucketCuarentena, claveObjeto(referencia))]
	if len(versiones) != 1 {
		t.Fatalf("versiones de negocio creadas: %d", len(versiones))
	}
	marcadores := cliente.objetos[indiceFalso(almacen.configuracion.BucketCuarentena, prefijoClaveIdempotencia+referencia)]
	if len(marcadores) != 1 {
		t.Fatalf("versiones de marcador creadas: %d", len(marcadores))
	}
}

func TestRespuestaPostPUTInvalidaSeCompensaYNoEnvenenaIdempotencia(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, &presignadorFalso{}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("objeto que debe poder reintentarse")
	solicitud := solicitudEscrituraPrueba(t, ahora, contenido, "idempotencia-compensada")
	cliente.mu.Lock()
	cliente.omitirChecksum = true
	cliente.mu.Unlock()
	if _, err := almacen.Escribir(context.Background(), solicitud); !errors.Is(err, ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("respuesta sin checksum aceptada: %v", err)
	}
	referencia := almacen.referenciaIdempotente("escribir", solicitud.ClaveIdempotencia)
	cliente.mu.Lock()
	if cliente.actuales[indiceFalso(almacen.configuracion.BucketCuarentena, claveObjeto(referencia))] != "" {
		cliente.mu.Unlock()
		t.Fatal("la version no verificable quedo envenenando la clave")
	}
	cliente.omitirChecksum = false
	cliente.mu.Unlock()
	solicitud.Contenido = bytes.NewReader(contenido)
	if _, err := almacen.Escribir(context.Background(), solicitud); err != nil {
		t.Fatalf("reintento tras compensacion: %v", err)
	}
}

func TestLectorVerificadoDetectaManipulacionEnMismaLecturaFinal(t *testing.T) {
	correcto := []byte("contenido exacto")
	suma := sha256.Sum256(correcto)
	lector := nuevoLectorVerificado(io.NopCloser(&lectorEOFConDatos{datos: correcto}), int64(len(correcto)), hex.EncodeToString(suma[:]))
	buffer := make([]byte, len(correcto))
	n, err := lector.Read(buffer)
	if n != len(correcto) || !errors.Is(err, io.EOF) || !bytes.Equal(buffer, correcto) {
		t.Fatalf("lectura valida: n=%d err=%v", n, err)
	}
	lector = nuevoLectorVerificado(io.NopCloser(&lectorEOFConDatos{datos: correcto}), int64(len(correcto)), strings.Repeat("0", 64))
	n, err = lector.Read(buffer)
	if n != len(correcto) || !errors.Is(err, ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("huella manipulada no detectada en lectura final: n=%d err=%v", n, err)
	}
}

func TestCargaDirectaFirmaClaveTamanoMIMEChecksumMetadatosYCifrado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	presignador := &presignadorFalso{}
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, presignador, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("titulo academico")
	suma := sha256.Sum256(contenido)
	solicitud := ports.SolicitudPrepararCargaDirecta{
		Contexto:          contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenPrepararCargaDirecta),
		ClaveIdempotencia: "idempotencia-carga-directa-uno", MIME: "application/pdf",
		Tamano: int64(len(contenido)), HuellaSHA256: hex.EncodeToString(suma[:]), ExpiraEn: ahora.Add(2 * time.Minute),
	}
	instrucciones, err := almacen.PrepararCargaDirecta(context.Background(), solicitud)
	if err != nil || instrucciones.ValidarPara(solicitud, almacen.capacidades) != nil {
		t.Fatalf("preparar carga: %v", err)
	}
	presignador.mu.Lock()
	defer presignador.mu.Unlock()
	entrada := presignador.entrada
	if entrada == nil || !strings.HasPrefix(awsv2.ToString(entrada.Key), prefijoClaveCargaDirecta+prefijoSesionCargaDirecta) ||
		strings.Contains(awsv2.ToString(entrada.Key), solicitud.ClaveIdempotencia) ||
		awsv2.ToInt64(entrada.ContentLength) != solicitud.Tamano || awsv2.ToString(entrada.ContentType) != solicitud.MIME ||
		awsv2.ToString(entrada.ChecksumSHA256) != sha256Base64(solicitud.HuellaSHA256) ||
		awsv2.ToString(entrada.IfNoneMatch) != "*" || entrada.ServerSideEncryption != types.ServerSideEncryptionAes256 ||
		entrada.Metadata[metaPreparacionSHA256] == "" || entrada.Metadata[metaVinculoSesion] == "" {
		t.Fatalf("PUT no ligado exactamente: %#v", entrada)
	}
}

func TestCargaDirectaKMSFirmaClaveYBucketKeyYLosVerificaEnElBackend(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	presignador := &presignadorFalso{}
	configuracion := configuracionPrueba(true)
	configuracion.Cifrado = types.ServerSideEncryptionAwsKms
	configuracion.ClaveKMS = "kms://clave-documental-pruebas"
	configuracion.UsarBucketKeyKMS = true
	almacen, err := NuevoConCliente(context.Background(), configuracion, cliente, presignador, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("documento cifrado con kms")
	suma := sha256.Sum256(contenido)
	solicitud := ports.SolicitudPrepararCargaDirecta{
		Contexto:          contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenPrepararCargaDirecta),
		ClaveIdempotencia: "idempotencia-carga-kms", MIME: "application/pdf",
		Tamano: int64(len(contenido)), HuellaSHA256: hex.EncodeToString(suma[:]), ExpiraEn: ahora.Add(2 * time.Minute),
	}
	if _, err := almacen.PrepararCargaDirecta(context.Background(), solicitud); err != nil {
		t.Fatalf("preparar carga KMS: %v", err)
	}
	presignador.mu.Lock()
	defer presignador.mu.Unlock()
	if presignador.entrada == nil || presignador.entrada.ServerSideEncryption != types.ServerSideEncryptionAwsKms ||
		awsv2.ToString(presignador.entrada.SSEKMSKeyId) != configuracion.ClaveKMS ||
		!awsv2.ToBool(presignador.entrada.BucketKeyEnabled) {
		t.Fatalf("condiciones KMS incompletas: %#v", presignador.entrada)
	}
}

func TestCargaDirectaConfirmaHEADCanonizaYReintentaSinCargaTemporal(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	presignador := &presignadorFalso{}
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, presignador, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("titulo academico firmado")
	suma := sha256.Sum256(contenido)
	preparacion := ports.SolicitudPrepararCargaDirecta{
		Contexto:          contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenPrepararCargaDirecta),
		ClaveIdempotencia: "idempotencia-carga-confirmada", MIME: "application/pdf",
		Tamano: int64(len(contenido)), HuellaSHA256: hex.EncodeToString(suma[:]), ExpiraEn: ahora.Add(2 * time.Minute),
	}
	if _, err := almacen.PrepararCargaDirecta(context.Background(), preparacion); err != nil {
		t.Fatal(err)
	}
	presignador.mu.Lock()
	entradaTemporal := *presignador.entrada
	presignador.mu.Unlock()
	sesionRef := strings.TrimPrefix(awsv2.ToString(entradaTemporal.Key), prefijoClaveCargaDirecta)
	entradaTemporal.Body = bytes.NewReader(contenido)
	salidaTemporal, err := cliente.PutObject(context.Background(), &entradaTemporal)
	if err != nil {
		t.Fatalf("simular PUT del navegador: %v", err)
	}
	contextoConfirmacion := contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenConfirmarCargaDirecta)
	confirmacion := confirmacionCargaDirectaPrueba(t, ahora, contextoConfirmacion, sesionRef)
	resultado, err := almacen.ConfirmarCargaDirecta(context.Background(), confirmacion)
	if err != nil || resultado.ValidarCargaDirecta(preparacion, confirmacion, almacen.capacidades) != nil {
		t.Fatalf("confirmacion canonica: %+v, %v", resultado, err)
	}
	if resultado.Objeto.Objeto.Referencia != almacen.referenciaIdempotente("carga-directa", preparacion.ClaveIdempotencia) ||
		resultado.Objeto.Objeto.Version == awsv2.ToString(salidaTemporal.VersionId) || resultado.Objeto.Zona != ports.ZonaAlmacenCuarentena {
		t.Fatalf("la carga temporal se uso como objeto canonico: %+v", resultado.Objeto)
	}
	repetido, err := almacen.ConfirmarCargaDirecta(context.Background(), confirmacion)
	if err != nil || repetido.Objeto.Objeto != resultado.Objeto.Objeto || !repetido.Evidencia.ReintentoIdempotente ||
		repetido.ValidarCargaDirecta(preparacion, confirmacion, almacen.capacidades) != nil {
		t.Fatalf("reintento sin staging: %+v, %v", repetido, err)
	}
}

func TestCargaDirectaNoDeclaraExitoHastaConfirmarLimpiezaTemporal(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cliente := nuevoClienteFalso()
	presignador := &presignadorFalso{}
	almacen, err := NuevoConCliente(context.Background(), configuracionPrueba(true), cliente, presignador, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("carga temporal que exige limpieza confirmada")
	suma := sha256.Sum256(contenido)
	preparacion := ports.SolicitudPrepararCargaDirecta{
		Contexto:          contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenPrepararCargaDirecta),
		ClaveIdempotencia: "idempotencia-limpieza-temporal", MIME: "application/pdf",
		Tamano: int64(len(contenido)), HuellaSHA256: hex.EncodeToString(suma[:]), ExpiraEn: ahora.Add(2 * time.Minute),
	}
	if _, err := almacen.PrepararCargaDirecta(context.Background(), preparacion); err != nil {
		t.Fatal(err)
	}
	presignador.mu.Lock()
	entradaTemporal := *presignador.entrada
	presignador.mu.Unlock()
	sesionRef := strings.TrimPrefix(awsv2.ToString(entradaTemporal.Key), prefijoClaveCargaDirecta)
	entradaTemporal.Body = bytes.NewReader(contenido)
	if _, err := cliente.PutObject(context.Background(), &entradaTemporal); err != nil {
		t.Fatalf("simular PUT temporal: %v", err)
	}
	confirmacion := confirmacionCargaDirectaPrueba(
		t, ahora, contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenConfirmarCargaDirecta), sesionRef,
	)
	cliente.mu.Lock()
	cliente.fallarDelete = true
	cliente.mu.Unlock()
	if _, err := almacen.ConfirmarCargaDirecta(context.Background(), confirmacion); err == nil {
		t.Fatal("se declaro exito pese a fallar la retirada temporal")
	}
	cliente.mu.Lock()
	cliente.fallarDelete = false
	cliente.mu.Unlock()
	resultado, err := almacen.ConfirmarCargaDirecta(context.Background(), confirmacion)
	if err != nil || !resultado.Evidencia.ReintentoIdempotente {
		t.Fatalf("reintento de limpieza: %+v, %v", resultado, err)
	}
	cliente.mu.Lock()
	defer cliente.mu.Unlock()
	if cliente.actuales[indiceFalso(almacen.configuracion.BucketCuarentena, claveCargaDirecta(sesionRef))] != "" {
		t.Fatal("la carga temporal siguio presente tras el reintento")
	}
}

func TestPresignadorAWSRealConservaTodasLasCondicionesDeCarga(t *testing.T) {
	configuracion := configuracionPrueba(false)
	configuracion.AccessKeyID = "credencial-prueba-no-real"
	configuracion.SecretAccessKey = "secreto-prueba-no-real"
	_, presignador, err := nuevoClienteReal(context.Background(), configuracion)
	if err != nil {
		t.Fatalf("crear presignador AWS: %v", err)
	}
	huella := strings.Repeat("a", 64)
	metadatos := map[string]string{
		metaEsquema: esquemaCargaDirecta, metaConector: configuracion.ConectorID,
		metaZona: string(ports.ZonaAlmacenCuarentena), metaTamano: "123", metaSHA256: huella,
		metaEvidencia: "evidencia-presign-real", metaAlmacenadoEn: time.Now().UTC().Format(time.RFC3339Nano),
		metaIdempotencia: strings.Repeat("b", 64), metaVinculoSesion: strings.Repeat("c", 64),
		metaFinalReferencia: prefijoReferencia + strings.Repeat("d", 64), metaMIME: "application/pdf",
		metaExpiraEn: time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano), metaPreparacionSHA256: strings.Repeat("b", 64),
	}
	firmada, err := presignador.PresignPutObject(context.Background(), &awss3.PutObjectInput{
		Bucket: awsv2.String(configuracion.BucketCuarentena), Key: awsv2.String(prefijoClaveCargaDirecta + prefijoSesionCargaDirecta + strings.Repeat("e", 64)),
		ContentLength: awsv2.Int64(123), ContentType: awsv2.String("application/pdf"), IfNoneMatch: awsv2.String("*"),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256, ChecksumSHA256: awsv2.String(sha256Base64(huella)),
		Metadata: metadatos, ServerSideEncryption: types.ServerSideEncryptionAes256,
	}, func(opciones *awss3.PresignOptions) { opciones.Expires = time.Minute })
	if err != nil || firmada == nil {
		t.Fatalf("presign AWS real: %v", err)
	}
	cabeceras, err := cabecerasCargaDirecta(firmada.SignedHeader)
	if err != nil || len(cabeceras) < 10 {
		t.Fatalf("condiciones firmadas no transportables al navegador: %v; %#v", err, firmada.SignedHeader)
	}
	if !strings.HasPrefix(firmada.URL, configuracion.Endpoint+"/") || strings.Contains(firmada.URL, configuracion.SecretAccessKey) {
		t.Fatalf("URL firmada insegura: %q", firmada.URL)
	}
}

func TestPresignadorAWSRealAdmiteDireccionamientoVirtualConfigurable(t *testing.T) {
	configuracion := configuracionPrueba(false)
	configuracion.PathStyle = false
	configuracion.AccessKeyID = "credencial-prueba-no-real"
	configuracion.SecretAccessKey = "secreto-prueba-no-real"
	_, presignador, err := nuevoClienteReal(context.Background(), configuracion)
	if err != nil {
		t.Fatalf("crear presignador AWS: %v", err)
	}
	huella := strings.Repeat("a", 64)
	firmada, err := presignador.PresignPutObject(context.Background(), &awss3.PutObjectInput{
		Bucket: awsv2.String(configuracion.BucketCuarentena), Key: awsv2.String("vec/cargas/v1/objeto-opaco"),
		ContentLength: awsv2.Int64(1), ContentType: awsv2.String("application/octet-stream"), IfNoneMatch: awsv2.String("*"),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256, ChecksumSHA256: awsv2.String(sha256Base64(huella)),
		Metadata: map[string]string{"vec-prueba": "opaca"}, ServerSideEncryption: types.ServerSideEncryptionAes256,
	}, func(opciones *awss3.PresignOptions) { opciones.Expires = time.Minute })
	if err != nil || firmada == nil {
		t.Fatalf("presign virtual-host: %v", err)
	}
	direccion, err := url.Parse(firmada.URL)
	if err != nil || direccion.Scheme != "https" || direccion.Hostname() != configuracion.BucketCuarentena+".objetos.interno.example" ||
		!strings.HasSuffix(direccion.EscapedPath(), "/vec/cargas/v1/objeto-opaco") ||
		configuracion.origenCargaDirecta() != "https://"+configuracion.BucketCuarentena+".objetos.interno.example" {
		t.Fatalf("direccionamiento virtual incorrecto: %q, origen=%q", firmada.URL, configuracion.origenCargaDirecta())
	}
}

func TestConfiguracionRechazaTransporteClaroAliasBucketsYNoFiltraSecretos(t *testing.T) {
	base := configuracionPrueba(false)
	casos := []Configuracion{base, base, base, base, base}
	casos[0].Endpoint = "http://objetos.interno.example"
	casos[1].BucketAdmitida = casos[1].BucketCuarentena
	casos[2].ClaveDerivacion = []byte("muy-corta")
	casos[3].SessionToken = "token-huerfano"
	casos[4].Endpoint = "https://objetos.interno.example/ruta-ambigua"
	for _, configuracion := range casos {
		err := configuracion.Validar()
		if !errors.Is(err, ErrConfiguracionInvalida) {
			t.Fatalf("configuracion insegura aceptada: %+v", configuracion)
		}
		if strings.Contains(err.Error(), configuracion.SecretAccessKey) && configuracion.SecretAccessKey != "" {
			t.Fatal("error filtro credencial")
		}
	}
}

func TestClienteHTTPSeguroNoHeredaProxyDelProceso(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "https://proxy-no-autorizado.example:8443")
	cliente, err := clienteHTTPSeguro("", []netip.Prefix{netip.MustParsePrefix("10.20.0.0/16")})
	if err != nil {
		t.Fatalf("crear cliente HTTPS: %v", err)
	}
	transporte, ok := cliente.Transport.(*http.Transport)
	if !ok || transporte.Proxy != nil || transporte.DialContext == nil || cliente.CheckRedirect == nil {
		t.Fatal("el transporte del almacen no aplica todos los limites de red")
	}
	peticion, err := http.NewRequest(http.MethodGet, "https://destino-no-autorizado.example", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := cliente.CheckRedirect(peticion, nil); !errors.Is(err, errRedireccionS3NoPermitida) {
		t.Fatalf("redireccion no cerrada: %v", err)
	}
}

type resolvedorRedPrueba struct {
	direcciones []netip.Addr
	err         error
}

func (r resolvedorRedPrueba) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return append([]netip.Addr(nil), r.direcciones...), r.err
}

func TestResolucionS3SoloConservaDireccionesDeLaListaPermitida(t *testing.T) {
	redes := []netip.Prefix{netip.MustParsePrefix("10.20.0.0/16")}
	resolvedor := resolvedorRedPrueba{direcciones: []netip.Addr{
		netip.MustParseAddr("203.0.113.9"),
		netip.MustParseAddr("10.20.4.7"),
		netip.MustParseAddr("10.20.4.7"),
	}}
	direcciones, err := resolverDireccionesPermitidas(context.Background(), "rgw.interno.example", redes, resolvedor)
	if err != nil {
		t.Fatalf("resolver destino permitido: %v", err)
	}
	if len(direcciones) != 1 || direcciones[0] != netip.MustParseAddr("10.20.4.7") {
		t.Fatalf("direcciones filtradas = %v", direcciones)
	}
	if _, err := resolverDireccionesPermitidas(
		context.Background(), "203.0.113.9", redes, resolvedorRedPrueba{},
	); !errors.Is(err, ErrOperacionS3) {
		t.Fatalf("literal fuera de red aceptado: %v", err)
	}
}

func TestPerfilFuerteExigeListaDeRedYRechazaListasAmbiguas(t *testing.T) {
	configuracion := configuracionPrueba(true)
	configuracion.RedesPermitidas = nil
	if !errors.Is(configuracion.Validar(), ErrConfiguracionInvalida) {
		t.Fatal("perfil fuerte sin lista de red aceptado")
	}
	for _, valor := range []string{
		"0.0.0.0/0",
		"10.20.0.1/16",
		"10.20.0.0/16,10.20.0.0/16",
		"direccion-invalida",
	} {
		if _, err := parsearRedesPermitidas(valor); !errors.Is(err, ErrConfiguracionInvalida) {
			t.Fatalf("lista ambigua aceptada %q: %v", valor, err)
		}
	}
}

func TestConfiguracionNoEsSerializableNiRegistrableConSecretos(t *testing.T) {
	configuracion := configuracionPrueba(false)
	configuracion.AccessKeyID = "identidad-muy-secreta"
	configuracion.SecretAccessKey = "secreto-muy-sensible"
	configuracion.SessionToken = "token-muy-sensible"
	representaciones := []string{
		fmt.Sprintf("%v", configuracion),
		fmt.Sprintf("%+v", configuracion),
		fmt.Sprintf("%#v", configuracion),
	}
	serializada, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	representaciones = append(representaciones, string(serializada))
	for _, representacion := range representaciones {
		if representacion != "configuracion_s3_redactada" && representacion != `"configuracion_s3_redactada"` {
			t.Fatalf("representacion no cerrada: %q", representacion)
		}
		for _, secreto := range []string{configuracion.AccessKeyID, configuracion.SecretAccessKey, configuracion.SessionToken} {
			if strings.Contains(representacion, secreto) {
				t.Fatal("la configuracion filtro material secreto")
			}
		}
	}
}

func TestConfiguracionDesdeMapaComponePerfilFuerteSinValoresDeProductoEnElNucleo(t *testing.T) {
	clave := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x91}, 32))
	configuracion, err := ConfiguracionDesdeMapa(map[string]string{
		"conector_id": "ceph_documental", "endpoint": "https://rgw.interno.example", "region": "eu-west-1",
		"bucket_cuarentena": "vec-cuarentena", "bucket_admitida": "vec-admitida", "path_style": "false",
		"tamano_maximo_bytes": "1048576", "duracion_carga_directa": "2m",
		"retencion_minima_admitida": "24h", "clave_derivacion_base64url": clave,
		"cifrado": "AES256", "perfil_fuerte": "true", "probar_capacidades": "true",
		"permitir_eliminacion": "false", "modo_retencion": "COMPLIANCE",
		"redes_permitidas": "10.20.0.0/16,2001:db8:20::/48",
	})
	if err != nil || configuracion.Validar() != nil || configuracion.PathStyle ||
		configuracion.ConectorID != "ceph_documental" || configuracion.TamanoMaximo != 1048576 ||
		configuracion.RetencionMinimaAdmitida != 24*time.Hour || len(configuracion.ClaveDerivacion) != 32 {
		t.Fatalf("composicion fuerte: %v", err)
	}
}

func TestErrorRemotoNoPropagaMensajeNiCredenciales(t *testing.T) {
	secreto := "credencial-que-no-debe-aparecer"
	err := errorRemoto(context.Background(), errors.New("fallo remoto con "+secreto))
	if !errors.Is(err, ErrOperacionS3) || strings.Contains(err.Error(), secreto) {
		t.Fatalf("error remoto no redaccion: %v", err)
	}
}

type lectorEOFConDatos struct {
	datos []byte
	leido bool
}

type lectorQueFalla struct{}

func (lectorQueFalla) Read([]byte) (int, error) {
	return 0, errors.New("fallo de entropia simulado")
}

func (l *lectorEOFConDatos) Read(p []byte) (int, error) {
	if l.leido {
		return 0, io.EOF
	}
	l.leido = true
	return copy(p, l.datos), io.EOF
}

func solicitudEscrituraPrueba(t *testing.T, ahora time.Time, contenido []byte, idempotencia string) ports.SolicitudEscribirObjeto {
	t.Helper()
	suma := sha256.Sum256(contenido)
	return ports.SolicitudEscribirObjeto{
		Contexto: contextoAlmacenPrueba(t, ahora, ports.AccionAlmacenEscribir), ClaveIdempotencia: idempotencia,
		Zona: ports.ZonaAlmacenCuarentena, MIME: "application/pdf", Tamano: int64(len(contenido)),
		HuellaSHA256: hex.EncodeToString(suma[:]), Contenido: bytes.NewReader(contenido),
	}
}

func contextoAlmacenPrueba(t *testing.T, ahora time.Time, accionTecnica string) ports.ContextoOperacionAlmacen {
	t.Helper()
	accionNegocio := ports.AccionNegocioCustodiarDecisionBaremacion
	campos := []string{"documento_custodiado", "evidencia_custodia"}
	if accionTecnica == ports.AccionAlmacenPrepararCargaDirecta {
		accionNegocio = ports.AccionNegocioPrepararCargaDocumental
		campos = []string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}
	} else if accionTecnica == ports.AccionAlmacenConfirmarCargaDirecta {
		accionNegocio = ports.AccionNegocioConfirmarCargaDocumental
		campos = []string{"contenido_cuarentena", "estado"}
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: "operacion:s3:prueba", CargaRef: "carga:s3:prueba", Clasificacion: "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64), EfectoRef: "efecto:s3:prueba",
	}
	atributos := map[string]string{
		ports.AtributoAlmacenOperacionRef: vinculos.OperacionRef, ports.AtributoAlmacenCargaRef: vinculos.CargaRef,
		ports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		ports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		ports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		ports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:s3:prueba", ModuloID: "bolsa", Tipo: "documento_bolsa",
		Ambitos: map[string]string{"organizacion": "diputacion_granada"}, Atributos: atributos,
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	_, vinculoActor, err := pruebasvec.NuevoContextoYVinculo(
		ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:s3:prueba", Concedida: true, Codigo: "concedida",
		PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl",
		Accion: accionNegocio, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaRecurso, Finalidad: "custodia_documental", CorrelacionRef: "correlacion:s3:prueba",
		VinculoAutenticacionActor: vinculoActor, AsignacionRef: "asignacion:s3:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:s3:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:s3:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64), RevisionCatalogoPoliticas: 1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo, PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: campos,
		EmitidaEn: ahora.Add(-time.Minute), ValidaHasta: ahora.Add(4 * time.Minute),
	}
	var contexto ports.ContextoOperacionAlmacen
	if accionTecnica == ports.AccionAlmacenPrepararCargaDirecta {
		contexto, err = ports.NuevoContextoPrepararCargaDirectaAlmacen(decision, recurso, vinculos, ahora)
	} else if accionTecnica == ports.AccionAlmacenConfirmarCargaDirecta {
		contexto, err = ports.NuevoContextoConfirmarCargaDirectaAlmacen(decision, recurso, vinculos, ahora)
	} else {
		contexto, err = ports.NuevoContextoCustodiarDecisionBaremacionAlmacen(decision, recurso, vinculos, ahora)
	}
	if err != nil {
		t.Fatalf("contexto s3: %v", err)
	}
	return contexto
}

func confirmacionCargaDirectaPrueba(
	t *testing.T,
	ahora time.Time,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
) ports.SolicitudConfirmarCargaDirecta {
	t.Helper()
	recibo, err := ports.NuevoReciboCargaDirecta("recibo:mac:s3:v1:0123456789abcdefghijkl")
	if err != nil {
		t.Fatal(err)
	}
	consumo := ports.SolicitudConsumirReciboCargaDirecta{
		Contexto: contexto, SesionRef: sesionRef, Recibo: recibo, ValidaHasta: ahora.Add(4 * time.Minute),
	}
	resultado := ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC:          "hmac-sha256:indice_v1:" + strings.Repeat("3", 64),
		GrupoHMAC:           "hmac-sha256:grupo_v1:" + strings.Repeat("4", 64),
		VinculoHMAC:         "hmac-sha256:vinculo_v1:" + strings.Repeat("5", 64),
		EvidenciaConsumoRef: "evidencia:consumo:s3:uno", IntencionConfirmacionRef: "confirmacion:intencion:s3:uno",
		HuellaIntencionHMAC: "hmac-sha256:intencion_v1:" + strings.Repeat("7", 64),
		RegistradoEn:        ahora.Add(-20 * time.Second), ConsumidoEn: ahora.Add(-10 * time.Second), ExpiraEn: ahora.Add(2 * time.Minute),
	}
	atestacion := "hmac-sha256:atestacion_v1:" + strings.Repeat("6", 64)
	comprobante, err := ports.NuevoComprobanteConsumoReciboCargaDirecta(consumo, resultado, atestacion)
	if err != nil {
		t.Fatalf("comprobante: %v", err)
	}
	confirmacion, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), contexto, sesionRef, comprobante, verificadorAtestacionPrueba{},
	)
	if err != nil {
		t.Fatalf("confirmacion: %v", err)
	}
	return confirmacion
}

func indiceFalso(bucket, clave string) string { return bucket + "\x00" + clave }

func clonarMapa(origen map[string]string) map[string]string {
	resultado := make(map[string]string, len(origen))
	for clave, valor := range origen {
		resultado[clave] = valor
	}
	return resultado
}

func stringValor(valor uint64) string {
	const digitos = "0123456789abcdef"
	if valor == 0 {
		return "0"
	}
	var buffer [16]byte
	posicion := len(buffer)
	for valor > 0 {
		posicion--
		buffer[posicion] = digitos[valor&15]
		valor >>= 4
	}
	return string(buffer[posicion:])
}

var _ clienteSDK = (*clienteFalso)(nil)
var _ presignadorSDK = (*presignadorFalso)(nil)
