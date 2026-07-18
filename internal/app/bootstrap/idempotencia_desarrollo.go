package bootstrap

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

const (
	esquemaMaterialIdempotenciaDesarrolloV1   = "vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1"
	versionMaterialIdempotenciaDesarrollo     = 1
	versionHMACIdempotenciaBorrador           = uint16(2)
	minimoGeneracionesIdempotenciaDesarrollo  = 2
	maximoGeneracionesIdempotenciaDesarrollo  = 4
	referenciaProveedorIdempotenciaDesarrollo = "idempotencia-hmac-fichero-local-v1"
)

var _ gobiernoconvocatorias.DerivadorIdentidadOperacion = (*derivadorIdentidadOperacionDesarrollo)(nil)

type referenciaGeneracionIdempotenciaDesarrollo struct {
	Generacion                uint32 `json:"generacion"`
	ReferenciaLocalizador     string `json:"referencia_localizador"`
	ReferenciaHuellaSolicitud string `json:"referencia_huella_solicitud"`
}

type archivoMaterialIdempotenciaDesarrollo struct {
	Version            int                                          `json:"version"`
	Esquema            string                                       `json:"esquema"`
	Autoridad          string                                       `json:"autoridad"`
	VersionEsquemaHMAC uint16                                       `json:"version_esquema_hmac"`
	Generaciones       []referenciaGeneracionIdempotenciaDesarrollo `json:"generaciones"`
}

type claveHMACIdempotenciaDesarrollo struct {
	referencia string
	material   [sha256.Size]byte
}

type generacionIdempotenciaDesarrollo struct {
	generacion      uint32
	localizador     claveHMACIdempotenciaDesarrollo
	huellaSolicitud claveHMACIdempotenciaDesarrollo
}

type materialIdempotenciaDesarrollo struct {
	generaciones []generacionIdempotenciaDesarrollo
}

// derivadorIdentidadOperacionDesarrollo es deliberadamente privado y sólo se
// compone bajo el perfil desarrollo. Conserva claves locales en el proceso
// para hacer operable T20 sin HSM; producción debe aportar otro adaptador tras
// DerivadorIdentidadOperacion y nunca puede construir este tipo.
type derivadorIdentidadOperacionDesarrollo struct {
	generaciones []generacionIdempotenciaDesarrollo
}

type resultadoHMACIdempotenciaDesarrollo struct {
	generacion                uint32
	referenciaLocalizador     string
	referenciaHuellaSolicitud string
	localizador               [sha256.Size]byte
	huellaSolicitud           [sha256.Size]byte
}

func cargarMaterialIdempotenciaDesarrollo(
	raiz, rutaConfiguracion string,
) (materialIdempotenciaDesarrollo, error) {
	contenido, err := leerFicheroMaterialSeguro(rutaConfiguracion, 64<<10)
	if err != nil {
		return materialIdempotenciaDesarrollo{}, err
	}
	defer borrarBytes(contenido)
	if validarClavesJSONUnicas(contenido) != nil {
		return materialIdempotenciaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var archivo archivoMaterialIdempotenciaDesarrollo
	if err := decodificador.Decode(&archivo); err != nil {
		return materialIdempotenciaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) ||
		archivo.Version != versionMaterialIdempotenciaDesarrollo ||
		archivo.Esquema != esquemaMaterialIdempotenciaDesarrolloV1 ||
		archivo.Autoridad != AutoridadNoAutoritativa ||
		archivo.VersionEsquemaHMAC != versionHMACIdempotenciaBorrador ||
		len(archivo.Generaciones) < minimoGeneracionesIdempotenciaDesarrollo ||
		len(archivo.Generaciones) > maximoGeneracionesIdempotenciaDesarrollo {
		return materialIdempotenciaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}

	material := materialIdempotenciaDesarrollo{
		generaciones: make([]generacionIdempotenciaDesarrollo, 0, len(archivo.Generaciones)),
	}
	completo := false
	defer func() {
		if !completo {
			material.borrar()
		}
	}()
	for indice, persistida := range archivo.Generaciones {
		if persistida.Generacion == 0 ||
			(indice > 0 && archivo.Generaciones[indice-1].Generacion <= persistida.Generacion) ||
			persistida.ReferenciaLocalizador != referenciaLocalizadorIdempotenciaDesarrollo(persistida.Generacion) ||
			persistida.ReferenciaHuellaSolicitud != referenciaHuellaIdempotenciaDesarrollo(persistida.Generacion) {
			return materialIdempotenciaDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		generacion := generacionIdempotenciaDesarrollo{generacion: persistida.Generacion}
		generacion.localizador.referencia = persistida.ReferenciaLocalizador
		generacion.huellaSolicitud.referencia = persistida.ReferenciaHuellaSolicitud
		generacion.localizador.material, err = leerSecreto32Desarrollo(
			rutaClaveIdempotenciaDesarrollo(raiz, persistida.Generacion, "localizador"),
		)
		if err != nil {
			return materialIdempotenciaDesarrollo{}, err
		}
		generacion.huellaSolicitud.material, err = leerSecreto32Desarrollo(
			rutaClaveIdempotenciaDesarrollo(raiz, persistida.Generacion, "huella-solicitud"),
		)
		if err != nil {
			borrarBytes(generacion.localizador.material[:])
			return materialIdempotenciaDesarrollo{}, err
		}
		material.generaciones = append(material.generaciones, generacion)
		borrarBytes(generacion.localizador.material[:])
		borrarBytes(generacion.huellaSolicitud.material[:])
	}
	if !material.valido() {
		return materialIdempotenciaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	completo = true
	return material, nil
}

func nuevoDerivadorIdentidadOperacionDesarrollo(
	material *materialIdempotenciaDesarrollo,
) (*derivadorIdentidadOperacionDesarrollo, error) {
	if material == nil || !material.valido() {
		return nil, ErrMaterialDesarrolloInvalido
	}
	// Transferencia de propiedad: evita conservar una segunda copia de las
	// claves entre el cargador y el proveedor de proceso.
	generaciones := material.generaciones
	material.generaciones = nil
	return &derivadorIdentidadOperacionDesarrollo{generaciones: generaciones}, nil
}

func (d *derivadorIdentidadOperacionDesarrollo) Derivar(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudDerivacionIdempotencia,
) (gobiernoconvocatorias.ConjuntoIdentidadesOperacion, error) {
	if contextoInterfazNulo(ctx) || d == nil || !d.valido() {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
	}
	if err := ctx.Err(); err != nil {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			errors.Join(gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida, err)
	}
	preimagenLocalizador, preimagenHuella, err := solicitud.MaterialParaConectorConfiable()
	if err != nil {
		borrarBytes(preimagenLocalizador)
		borrarBytes(preimagenHuella)
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
	}
	return d.derivarMaterialEfimero(ctx, preimagenLocalizador, preimagenHuella)
}

// derivarMaterialEfimero asume propiedad de las dos copias y las borra en
// todos los desenlaces. No existe ruta de persistencia ni registro de ellas.
func (d *derivadorIdentidadOperacionDesarrollo) derivarMaterialEfimero(
	ctx context.Context,
	preimagenLocalizador, preimagenHuella []byte,
) (gobiernoconvocatorias.ConjuntoIdentidadesOperacion, error) {
	defer borrarBytes(preimagenLocalizador)
	defer borrarBytes(preimagenHuella)
	if contextoInterfazNulo(ctx) || d == nil || !d.valido() ||
		len(preimagenLocalizador) == 0 || len(preimagenHuella) == 0 {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
	}
	if err := ctx.Err(); err != nil {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			errors.Join(gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida, err)
	}
	resultados, err := d.calcularHMAC(preimagenLocalizador, preimagenHuella)
	if err != nil {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{}, err
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	identidades := make([]gobiernoconvocatorias.IdentidadOperacionDerivada, 0, len(resultados))
	for _, resultado := range resultados {
		referenciaLocalizador, errLocalizador := gobiernoconvocatorias.NuevaReferenciaClaveHMACLocalizador(
			resultado.referenciaLocalizador, resultado.generacion,
		)
		referenciaHuella, errHuella := gobiernoconvocatorias.NuevaReferenciaClaveHMACHuellaSolicitud(
			resultado.referenciaHuellaSolicitud, resultado.generacion,
		)
		localizador, errL := gobiernoconvocatorias.NuevoLocalizadorOperacion(
			versionHMACIdempotenciaBorrador, referenciaLocalizador,
			hex.EncodeToString(resultado.localizador[:]),
		)
		huella, errF := gobiernoconvocatorias.NuevaHuellaSolicitud(
			versionHMACIdempotenciaBorrador, referenciaHuella,
			hex.EncodeToString(resultado.huellaSolicitud[:]),
		)
		identidad, errIdentidad := gobiernoconvocatorias.NuevaIdentidadOperacionDerivada(localizador, huella)
		if errors.Join(errLocalizador, errHuella, errL, errF, errIdentidad) != nil {
			return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
				gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
		}
		identidades = append(identidades, identidad)
	}
	conjunto, err := gobiernoconvocatorias.NuevoConjuntoIdentidadesOperacion(identidades...)
	if err != nil {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
	}
	if err := ctx.Err(); err != nil {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{},
			errors.Join(gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida, err)
	}
	return conjunto, nil
}

func (d *derivadorIdentidadOperacionDesarrollo) calcularHMAC(
	preimagenLocalizador, preimagenHuella []byte,
) ([]resultadoHMACIdempotenciaDesarrollo, error) {
	if d == nil || !d.valido() || len(preimagenLocalizador) == 0 || len(preimagenHuella) == 0 {
		return nil, gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
	}
	resultados := make([]resultadoHMACIdempotenciaDesarrollo, len(d.generaciones))
	for indice := range d.generaciones {
		generacion := &d.generaciones[indice]
		resultados[indice] = resultadoHMACIdempotenciaDesarrollo{
			generacion:                generacion.generacion,
			referenciaLocalizador:     generacion.localizador.referencia,
			referenciaHuellaSolicitud: generacion.huellaSolicitud.referencia,
			localizador: calcularHMACSHA256Desarrollo(
				&generacion.localizador.material, preimagenLocalizador,
			),
			huellaSolicitud: calcularHMACSHA256Desarrollo(
				&generacion.huellaSolicitud.material, preimagenHuella,
			),
		}
	}
	return resultados, nil
}

func (d *derivadorIdentidadOperacionDesarrollo) valido() bool {
	if d == nil {
		return false
	}
	material := materialIdempotenciaDesarrollo{generaciones: d.generaciones}
	return material.valido()
}

func (d *derivadorIdentidadOperacionDesarrollo) borrar() {
	if d == nil {
		return
	}
	material := materialIdempotenciaDesarrollo{generaciones: d.generaciones}
	material.borrar()
	d.generaciones = nil
}

func (m *materialIdempotenciaDesarrollo) valido() bool {
	if m == nil || len(m.generaciones) < minimoGeneracionesIdempotenciaDesarrollo ||
		len(m.generaciones) > maximoGeneracionesIdempotenciaDesarrollo {
		return false
	}
	claves := make([]*[sha256.Size]byte, 0, len(m.generaciones)*2)
	for indice := range m.generaciones {
		generacion := &m.generaciones[indice]
		if generacion.generacion == 0 ||
			(indice > 0 && m.generaciones[indice-1].generacion <= generacion.generacion) ||
			generacion.localizador.referencia != referenciaLocalizadorIdempotenciaDesarrollo(generacion.generacion) ||
			generacion.huellaSolicitud.referencia != referenciaHuellaIdempotenciaDesarrollo(generacion.generacion) ||
			claveHMACDesarrolloNula(&generacion.localizador.material) ||
			claveHMACDesarrolloNula(&generacion.huellaSolicitud.material) {
			return false
		}
		claves = append(claves, &generacion.localizador.material, &generacion.huellaSolicitud.material)
	}
	for indice, clave := range claves {
		for anterior := 0; anterior < indice; anterior++ {
			if subtle.ConstantTimeCompare(clave[:], claves[anterior][:]) == 1 {
				return false
			}
		}
	}
	return true
}

func (m *materialIdempotenciaDesarrollo) separadoDe(otras ...*[sha256.Size]byte) bool {
	if !m.valido() {
		return false
	}
	for indice := range m.generaciones {
		generacion := &m.generaciones[indice]
		for _, otra := range otras {
			if otra == nil || subtle.ConstantTimeCompare(generacion.localizador.material[:], otra[:]) == 1 ||
				subtle.ConstantTimeCompare(generacion.huellaSolicitud.material[:], otra[:]) == 1 {
				return false
			}
		}
	}
	return true
}

func (m *materialIdempotenciaDesarrollo) borrar() {
	if m == nil {
		return
	}
	for indice := range m.generaciones {
		borrarBytes(m.generaciones[indice].localizador.material[:])
		borrarBytes(m.generaciones[indice].huellaSolicitud.material[:])
	}
	m.generaciones = nil
}

func calcularHMACSHA256Desarrollo(
	clave *[sha256.Size]byte,
	preimagen []byte,
) [sha256.Size]byte {
	mac := hmac.New(sha256.New, clave[:])
	_, _ = mac.Write(preimagen)
	suma := mac.Sum(nil)
	defer borrarBytes(suma)
	var resultado [sha256.Size]byte
	copy(resultado[:], suma)
	return resultado
}

func borrarResultadosHMACIdempotenciaDesarrollo(resultados []resultadoHMACIdempotenciaDesarrollo) {
	for indice := range resultados {
		borrarBytes(resultados[indice].localizador[:])
		borrarBytes(resultados[indice].huellaSolicitud[:])
	}
}

func claveHMACDesarrolloNula(clave *[sha256.Size]byte) bool {
	var cero [sha256.Size]byte
	return clave == nil || subtle.ConstantTimeCompare(clave[:], cero[:]) == 1
}

func referenciaLocalizadorIdempotenciaDesarrollo(generacion uint32) string {
	return fmt.Sprintf("clave:hmac:convocatorias:localizador:desarrollo:v%d", generacion)
}

func referenciaHuellaIdempotenciaDesarrollo(generacion uint32) string {
	return fmt.Sprintf("clave:hmac:convocatorias:huella:desarrollo:v%d", generacion)
}

func rutaClaveIdempotenciaDesarrollo(raiz string, generacion uint32, dominio string) string {
	return filepath.Join(raiz, "idempotencia", fmt.Sprintf("g%d-%s.bin", generacion, dominio))
}

func contextoInterfazNulo(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	valor := reflect.ValueOf(ctx)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
