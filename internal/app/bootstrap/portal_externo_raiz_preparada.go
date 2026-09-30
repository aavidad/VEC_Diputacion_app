package bootstrap

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/config"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// Este estado solo lo usa la preparación explícita del operador. El servidor
// usa el inventario publicado y nunca crea o rota una raíz al arrancar.
const estadoRaizPreparadaPortalExterno = "externo/v3/raiz-preparada.json"

type estadoRaizPropiaPortalExterno struct {
	Version           int       `json:"version"`
	Semilla           []byte    `json:"semilla"`
	ValidaDesde       time.Time `json:"valida_desde"`
	ValidaHasta       time.Time `json:"valida_hasta"`
	PreimagenRotacion string    `json:"preimagen_rotacion_sha256,omitempty"`
}

// prepararBasePropiaPortalExterno deriva exclusivamente las capacidades desde
// la idempotencia externa. La raíz de firma se genera al azar una sola vez y
// se conserva en el almacén privado externo, independiente de esas claves.
func prepararBasePropiaPortalExterno(destino, preimagenRotacion string, ahora time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
	var vacio materialAtestacionContratacionTemporalDesarrollo
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(destino, filepath.Join(destino, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	defer idempotencia.borrar()
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	if err != nil {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	defer derivador.borrar()
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
	if err != nil {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	fallo := func() (materialAtestacionContratacionTemporalDesarrollo, error) {
		base.borrarCopiasEfimeras()
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	estado, err := leerOCrearRaizPropiaPortalExterno(destino, preimagenRotacion, base.validaDesde, base.validaHasta)
	if err != nil {
		return fallo()
	}
	defer clear(estado.Semilla)
	clear(base.privada)
	base.privada = ed25519.NewKeyFromSeed(estado.Semilla)
	base.spki, err = x509.MarshalPKIXPublicKey(base.privada.Public().(ed25519.PublicKey))
	if err != nil {
		return fallo()
	}
	huella := sha256.Sum256(base.spki)
	base.spkiHuella = hex.EncodeToString(huella[:])
	base.claveID = "clave:atestacion:externo:" + base.spkiHuella
	base.claveVersion = 1
	base.validaDesde, base.validaHasta = estado.ValidaDesde.UTC(), estado.ValidaHasta.UTC()
	base.claveHMACID = strings.Replace(base.claveHMACID, "clave:capacidad:ct:", "clave:capacidad:ct:externo:", 1)
	base.emisorID = strings.Replace(base.emisorID, "emisor:ct:desarrollo:", "emisor:externo:", 1)
	base.configuracionRef = strings.Replace(base.configuracionRef, "confianza:atestacion:ct:desarrollo:", "confianza:atestacion:externo:", 1)
	if err := reconstruirClavesGobiernoPostgreSQLContratacionTemporalDesarrollo(&base); err != nil {
		return fallo()
	}
	base.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(base.configuracionRef, base.configuracionOrden, base.publicadaEn, base.expiraEn, base.raiz)
	if err != nil {
		return fallo()
	}
	base.configuracionHuella, err = base.configuracion.HuellaSHA256ParaGobierno()
	if err != nil {
		return fallo()
	}
	return base, nil
}

// El descriptor privado se crea de forma exclusiva. Un estado incompleto,
// una semilla antigua sin descriptor o un archivo manipulado se rechazan;
// nunca se "recuperan" generando otra identidad de firma.
func leerOCrearRaizPropiaPortalExterno(destino, preimagen string, desde, hasta time.Time) (estadoRaizPropiaPortalExterno, error) {
	var vacio estadoRaizPropiaPortalExterno
	if preimagen != "" && !huellaHexIgual(preimagen, debeDecodificarHuellaPortalExterno(preimagen)) {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	for _, relativa := range []string{"externo", directorioMaterialV3PortalExterno} {
		ruta := filepath.Join(destino, filepath.FromSlash(relativa))
		if err := os.Mkdir(ruta, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return vacio, ErrPreparacionPortalExternoInvalida
		}
		resuelta, err := filepath.EvalSymlinks(ruta)
		info, errInfo := os.Lstat(ruta)
		if err != nil || errInfo != nil || resuelta != ruta || !info.IsDir() || info.Mode().Perm() != 0o700 {
			return vacio, ErrPreparacionPortalExternoInvalida
		}
	}
	raiz, err := os.OpenRoot(filepath.Join(destino, filepath.FromSlash(directorioMaterialV3PortalExterno)))
	if err != nil {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	defer raiz.Close()
	cerrojo, err := raiz.OpenFile(".preparacion.lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	defer cerrojo.Close()
	descriptor := cerrojo.Fd()
	if descriptor > math.MaxInt {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	fd := int(descriptor)
	if syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	ruta := filepath.Join(destino, filepath.FromSlash(estadoRaizPreparadaPortalExterno))
	contenido, err := leerFicheroMaterialSeguro(ruta, 4096)
	var estado estadoRaizPropiaPortalExterno
	if err == nil {
		defer clear(contenido)
		d := json.NewDecoder(bytes.NewReader(contenido))
		d.DisallowUnknownFields()
		if validarClavesJSONUnicas(contenido) != nil || d.Decode(&estado) != nil || d.Decode(new(any)) != io.EOF || estado.Version != 1 || len(estado.Semilla) != ed25519.SeedSize || !estado.ValidaDesde.Before(estado.ValidaHasta) {
			clear(estado.Semilla)
			return vacio, ErrPreparacionPortalExternoInvalida
		}
		if preimagen == "" {
			if inv, err := leerInventarioV3PortalExterno(destino); err == nil && !raizPreparadaCoincidePortalExterno(estado, inv.Raiz.SPKISHA256) {
				clear(estado.Semilla)
				return vacio, ErrPreparacionPortalExternoInvalida
			}
			if estado.PreimagenRotacion != "" {
				inv, err := leerInventarioV3PortalExterno(destino)
				if err != nil || !raizPreparadaCoincidePortalExterno(estado, inv.Raiz.SPKISHA256) {
					clear(estado.Semilla)
					return vacio, ErrPreparacionPortalExternoInvalida
				}
			}
			return estado, nil
		}
		inv, err := leerInventarioV3PortalExterno(destino)
		if err != nil || (inv.Raiz.SPKISHA256 != preimagen && !raizPreparadaCoincidePortalExterno(estado, inv.Raiz.SPKISHA256)) {
			clear(estado.Semilla)
			return vacio, ErrPreparacionPortalExternoInvalida
		}
		if estado.PreimagenRotacion == preimagen {
			return estado, nil
		}
		if inv.Raiz.SPKISHA256 != preimagen || !raizPreparadaCoincidePortalExterno(estado, preimagen) {
			clear(estado.Semilla)
			return vacio, ErrPreparacionPortalExternoInvalida
		}
		clear(estado.Semilla)
	} else {
		if _, e := os.Lstat(ruta); !errors.Is(e, os.ErrNotExist) || preimagen != "" {
			return vacio, ErrPreparacionPortalExternoInvalida
		}
		// No adoptar el antiguo material exportado desde el proceso interno.
		for _, nombre := range []string{"raiz.bin", "inventario.json"} {
			if _, e := raiz.Lstat(nombre); !errors.Is(e, os.ErrNotExist) {
				return vacio, ErrPreparacionPortalExternoInvalida
			}
		}
	}
	estado = estadoRaizPropiaPortalExterno{Version: 1, Semilla: make([]byte, ed25519.SeedSize), ValidaDesde: desde.UTC(), ValidaHasta: hasta.UTC(), PreimagenRotacion: preimagen}
	if _, err := rand.Read(estado.Semilla); err != nil {
		clear(estado.Semilla)
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	contenido, err = json.Marshal(estado)
	if err != nil {
		clear(estado.Semilla)
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	defer clear(contenido)
	if err := escribirFicheroPrivado(ruta, contenido); err != nil {
		clear(estado.Semilla)
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	directorio, err := raiz.Open(".")
	if err != nil {
		clear(estado.Semilla)
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	err = directorio.Sync()
	_ = directorio.Close()
	if err != nil {
		clear(estado.Semilla)
		return vacio, ErrPreparacionPortalExternoInvalida
	}
	return estado, nil
}

func debeDecodificarHuellaPortalExterno(s string) []byte {
	b, _ := hex.DecodeString(s)
	if len(b) != sha256.Size {
		return nil
	}
	return b
}

func raizPreparadaCoincidePortalExterno(estado estadoRaizPropiaPortalExterno, huella string) bool {
	privada := ed25519.NewKeyFromSeed(estado.Semilla)
	defer clear(privada)
	spki, err := x509.MarshalPKIXPublicKey(privada.Public().(ed25519.PublicKey))
	if err != nil {
		return false
	}
	suma := sha256.Sum256(spki)
	return huellaHexIgual(huella, suma[:])
}
