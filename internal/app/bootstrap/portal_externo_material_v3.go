package bootstrap

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// ErrMaterialV3PortalExternoInvalido rechaza un inventario de autorización
// del proceso externo ausente, manipulado o que no cuadra con lo publicado.
var ErrMaterialV3PortalExternoInvalido = errors.New("bootstrap: material de autorizacion del portal externo no valido")

// Rutas fijas del inventario dentro del material del proceso externo. Todo
// cuelga de externo/, la única carpeta que el portal externo admite como
// propia (separacionportales).
const (
	directorioMaterialV3PortalExterno  = "externo/v3"
	inventarioMaterialV3PortalExterno  = "externo/v3/inventario.json"
	semillaRaizMaterialV3PortalExterno = "externo/v3/raiz.bin"
	tamanoMaximoInventarioV3Externo    = 64 << 10
)

// Consumidores cerrados del portal externo, en el mismo orden que AD3-112.
// El proceso externo solo recibe claves derivadas para estas audiencias.
var consumidoresPortalExternoV3 = []string{
	"usuarios_preferencias", "usuarios_correos", "usuarios_imagen", "mi_bolsa", "portal_candidato",
}

func audienciasConsumidorPortalExternoV3(consumidor string) []string {
	switch consumidor {
	case "usuarios_preferencias":
		return []string{usuariosports.AudienciaConsultarPreferenciasExterna, usuariosports.AudienciaActualizarPreferenciasExterna}
	case "usuarios_correos":
		return []string{
			usuariosports.AudienciaConsultarCorreosExterna, usuariosports.AudienciaAnadirCorreoExterna,
			usuariosports.AudienciaReenviarCorreoExterna, usuariosports.AudienciaVerificarCorreoExterna,
			usuariosports.AudienciaActivarCorreoExterna, usuariosports.AudienciaRetirarCorreoExterna,
		}
	case "usuarios_imagen":
		return []string{usuariosports.AudienciaConsultarImagenExterna, usuariosports.AudienciaActualizarImagenExterna}
	case "mi_bolsa":
		return []string{puertosbolsa.AudienciaMiBolsa, puertosbolsa.AudienciaHistorialMiBolsa}
	case "portal_candidato":
		return []string{
			puertosbolsa.AudienciaSolicitarPausaPropia, puertosbolsa.AudienciaSolicitarReactivacionPropia,
			puertosbolsa.AudienciaResponderLlamamientoPropio, puertosbolsa.AudienciaManifestarDisposicionPropia,
			puertosbolsa.AudienciaPresentarSolicitudDocumentalPropia,
			puertosbolsa.AudienciaConfirmarContactoPropio,
		}
	default:
		return nil
	}
}

// descriptoresMaterialPortalExternoV3 reúne, de los catálogos ya existentes,
// los descriptores de las audiencias externas: mismo dominio y prefijo que
// publica la composición combinada, de modo que la clave derivada es la misma.
func descriptoresMaterialPortalExternoV3() []descriptorMaterialConsumidorV3Desarrollo {
	var todos []descriptorMaterialConsumidorV3Desarrollo
	todos = append(todos, descriptoresMaterialPreferenciasUsuariosDesarrollo()...)
	todos = append(todos, descriptoresMaterialCorreosUsuariosDesarrollo()...)
	todos = append(todos, descriptoresMaterialImagenUsuariosDesarrollo()...)
	todos = append(todos, descriptorMaterialMiBolsaDesarrollo(), descriptorMaterialHistorialMiBolsaDesarrollo())
	todos = append(todos, descriptoresMaterialPortalCandidatoDesarrollo()...)
	todos = append(todos, descriptoresMaterialContactoPropioDesarrollo()...)
	externas := map[string]bool{}
	for _, consumidor := range consumidoresPortalExternoV3 {
		for _, audiencia := range audienciasConsumidorPortalExternoV3(consumidor) {
			externas[audiencia] = true
		}
	}
	resultado := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(externas))
	for _, d := range todos {
		if externas[d.Audiencia] {
			resultado = append(resultado, d)
		}
	}
	return resultado
}

// Inventario privado del proceso externo. Solo contiene coordenadas públicas
// y nombres de fichero; las claves viven en ficheros 0600 aparte y se cotejan
// por huella al cargarlas.
type inventarioV3PortalExterno struct {
	Version       int                                         `json:"version"`
	Raiz          raizInventarioV3PortalExterno               `json:"raiz"`
	Configuracion configuracionInventarioV3PortalExterno      `json:"configuracion"`
	Consumidores  map[string][]claveInventarioV3PortalExterno `json:"consumidores"`
}

type raizInventarioV3PortalExterno struct {
	ClaveID        string    `json:"clave_id"`
	Version        uint64    `json:"version"`
	SPKISHA256     string    `json:"spki_sha256"`
	SemillaArchivo string    `json:"semilla_archivo"`
	SemillaSHA256  string    `json:"semilla_sha256"`
	ValidaDesde    time.Time `json:"valida_desde"`
	ValidaHasta    time.Time `json:"valida_hasta"`
}

type configuracionInventarioV3PortalExterno struct {
	Revision     string    `json:"revision"`
	Secuencia    uint64    `json:"secuencia"`
	HuellaSHA256 string    `json:"huella_sha256"`
	PublicadaEn  time.Time `json:"publicada_en"`
	ExpiraEn     time.Time `json:"expira_en"`
}

type claveInventarioV3PortalExterno struct {
	Audiencia        string `json:"audiencia"`
	ClaveID          string `json:"clave_id"`
	Version          uint64 `json:"version"`
	RevisionGobierno uint64 `json:"revision_gobierno"`
	Orden            uint64 `json:"orden"`
	HuellaGobierno   string `json:"huella_gobierno_sha256"`
	HuellaSecreto    string `json:"huella_secreto_sha256"`
	EmisorID         string `json:"emisor_id"`
	Archivo          string `json:"archivo"`
}

var archivoClaveV3PortalExterno = regexp.MustCompile(`^externo/v3/[a-z_]+-[0-9]{1,2}\.bin$`)

// inventarioDesdeMateriales construye el inventario a partir del material ya
// publicado por el lado interno (una entrada por audiencia y consumidor).
func inventarioDesdeMateriales(publicados map[string][]materialAtestacionContratacionTemporalDesarrollo) (inventarioV3PortalExterno, error) {
	var inv inventarioV3PortalExterno
	inv.Version = 1
	inv.Consumidores = map[string][]claveInventarioV3PortalExterno{}
	var referencia *materialAtestacionContratacionTemporalDesarrollo
	for _, consumidor := range consumidoresPortalExternoV3 {
		materiales, ok := publicados[consumidor]
		if !ok {
			continue
		}
		audiencias := audienciasConsumidorPortalExternoV3(consumidor)
		if len(materiales) != len(audiencias) {
			return inv, ErrMaterialV3PortalExternoInvalido
		}
		for i := range materiales {
			m := &materiales[i]
			if m.audienciaConsumo != audiencias[i] || len(m.claveHMAC) < sha256.Size {
				return inv, ErrMaterialV3PortalExternoInvalido
			}
			if referencia == nil {
				referencia = m
			} else if m.claveID != referencia.claveID || m.claveVersion != referencia.claveVersion ||
				m.configuracionRef != referencia.configuracionRef || !bytes.Equal(m.spki, referencia.spki) {
				return inv, ErrMaterialV3PortalExternoInvalido
			}
			inv.Consumidores[consumidor] = append(inv.Consumidores[consumidor], claveInventarioV3PortalExterno{
				Audiencia: m.audienciaConsumo, ClaveID: m.claveHMACID, Version: m.claveHMACVersion,
				RevisionGobierno: m.claveHMACRevision, Orden: m.claveHMACOrden,
				HuellaGobierno: m.claveHMACHuella, HuellaSecreto: m.claveHMACSecreto, EmisorID: m.emisorID,
				Archivo: filepath.ToSlash(filepath.Join(directorioMaterialV3PortalExterno, consumidor+"-"+numeroDecimal(uint32(i+1))+".bin")),
			})
		}
	}
	if referencia == nil {
		return inv, ErrMaterialV3PortalExternoInvalido
	}
	inv.Raiz = raizInventarioV3PortalExterno{
		ClaveID: referencia.claveID, Version: referencia.claveVersion, SPKISHA256: referencia.spkiHuella,
		SemillaArchivo: semillaRaizMaterialV3PortalExterno, ValidaDesde: referencia.validaDesde, ValidaHasta: referencia.validaHasta,
	}
	semilla := referencia.privada.Seed()
	huellaSemilla := sha256.Sum256(semilla)
	clear(semilla)
	inv.Raiz.SemillaSHA256 = hex.EncodeToString(huellaSemilla[:])
	inv.Configuracion = configuracionInventarioV3PortalExterno{
		Revision: referencia.configuracionRef, Secuencia: referencia.configuracionOrden,
		HuellaSHA256: referencia.configuracionHuella, PublicadaEn: referencia.publicadaEn, ExpiraEn: referencia.expiraEn,
	}
	return inv, nil
}

// escribirMaterialV3PortalExterno deja en el material del proceso externo el
// inventario, la semilla de la raíz de atestación y las claves derivadas de
// sus audiencias. Nunca escribe la clave base ni la de otras audiencias.
// Los consumidores ya preparados que no se piden ahora se conservan si la raíz
// es la misma; con otra raíz hay que prepararlos todos de nuevo. Cada fichero
// se escribe aparte y se renombra, y el inventario va el último: un corte a
// medias puede dejar un material incoherente, que el proceso externo rechaza
// al arrancar (falla cerrado) hasta repetir la preparación.
func escribirMaterialV3PortalExterno(destino string, publicados map[string][]materialAtestacionContratacionTemporalDesarrollo) error {
	inv, err := inventarioDesdeMateriales(publicados)
	if err != nil {
		return err
	}
	if previo, errPrevio := leerInventarioV3PortalExterno(destino); errPrevio == nil {
		mismaRaiz := previo.Raiz.ClaveID == inv.Raiz.ClaveID && previo.Raiz.Version == inv.Raiz.Version &&
			previo.Raiz.SPKISHA256 == inv.Raiz.SPKISHA256
		for consumidor, claves := range previo.Consumidores {
			if _, nuevo := inv.Consumidores[consumidor]; nuevo {
				continue
			}
			if !mismaRaiz {
				return ErrMaterialV3PortalExternoInvalido
			}
			inv.Consumidores[consumidor] = claves
		}
	}
	if err := os.MkdirAll(filepath.Join(destino, filepath.FromSlash(directorioMaterialV3PortalExterno)), 0o700); err != nil {
		return ErrMaterialV3PortalExternoInvalido
	}
	for _, d := range []string{"externo", directorioMaterialV3PortalExterno} {
		if os.Chmod(filepath.Join(destino, filepath.FromSlash(d)), 0o700) != nil {
			return ErrMaterialV3PortalExternoInvalido
		}
	}
	var semilla []byte
	for _, consumidor := range consumidoresPortalExternoV3 {
		// Solo se escriben las claves publicadas ahora; las de consumidores
		// conservados de una preparación anterior ya están en disco.
		if _, ahora := publicados[consumidor]; !ahora {
			continue
		}
		for i, clave := range inv.Consumidores[consumidor] {
			m := publicados[consumidor][i]
			if semilla == nil {
				semilla = m.privada.Seed()
				defer clear(semilla)
			}
			if err := escribirFicheroPrivado(filepath.Join(destino, filepath.FromSlash(clave.Archivo)), m.claveHMAC); err != nil {
				return err
			}
		}
	}
	if err := escribirFicheroPrivado(filepath.Join(destino, filepath.FromSlash(semillaRaizMaterialV3PortalExterno)), semilla); err != nil {
		return err
	}
	contenido, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return ErrMaterialV3PortalExternoInvalido
	}
	if err := escribirFicheroPrivado(filepath.Join(destino, filepath.FromSlash(inventarioMaterialV3PortalExterno)), append(contenido, '\n')); err != nil {
		return err
	}
	return retirarClavesSinReferenciaV3PortalExterno(destino, inv)
}

// retirarClavesSinReferenciaV3PortalExterno borra las claves derivadas que el
// inventario ya no menciona, para que no quede material sin uso en disco.
func retirarClavesSinReferenciaV3PortalExterno(destino string, inv inventarioV3PortalExterno) error {
	referenciados := map[string]bool{semillaRaizMaterialV3PortalExterno: true}
	for _, claves := range inv.Consumidores {
		for _, clave := range claves {
			referenciados[clave.Archivo] = true
		}
	}
	directorio := filepath.Join(destino, filepath.FromSlash(directorioMaterialV3PortalExterno))
	entradas, err := os.ReadDir(directorio)
	if err != nil {
		return ErrMaterialV3PortalExternoInvalido
	}
	for _, e := range entradas {
		relativa := directorioMaterialV3PortalExterno + "/" + e.Name()
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".bin") || referenciados[relativa] {
			continue
		}
		if os.Remove(filepath.Join(directorio, e.Name())) != nil {
			return ErrMaterialV3PortalExternoInvalido
		}
	}
	return nil
}

func escribirFicheroPrivado(ruta string, contenido []byte) error {
	if info, err := os.Lstat(ruta); err == nil && !info.Mode().IsRegular() {
		return ErrMaterialV3PortalExternoInvalido
	}
	temporal, err := os.CreateTemp(filepath.Dir(ruta), ".escritura-*")
	if err != nil {
		return ErrMaterialV3PortalExternoInvalido
	}
	nombre := temporal.Name()
	defer os.Remove(nombre)
	if temporal.Chmod(0o600) != nil {
		temporal.Close()
		return ErrMaterialV3PortalExternoInvalido
	}
	if _, err := temporal.Write(contenido); err != nil {
		temporal.Close()
		return ErrMaterialV3PortalExternoInvalido
	}
	if temporal.Sync() != nil || temporal.Close() != nil || os.Rename(nombre, ruta) != nil {
		return ErrMaterialV3PortalExternoInvalido
	}
	return nil
}

// leerInventarioV3PortalExterno lee y valida el inventario del proceso
// externo y reconstruye, por consumidor, el material de cada audiencia con la
// configuración que figura en él. La configuración vigente se lee después de
// la base con el rol de preflight externo.
func leerInventarioV3PortalExterno(directorio string) (inventarioV3PortalExterno, error) {
	var inv inventarioV3PortalExterno
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(directorio, filepath.FromSlash(inventarioMaterialV3PortalExterno)), tamanoMaximoInventarioV3Externo)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return inv, ErrMaterialV3PortalExternoInvalido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var sobrante any
	if decodificador.Decode(&inv) != nil || !errors.Is(decodificador.Decode(&sobrante), io.EOF) ||
		inv.Version != 1 || inv.Raiz.SemillaArchivo != semillaRaizMaterialV3PortalExterno ||
		inv.Raiz.ClaveID == "" || inv.Raiz.Version < 1 || inv.Configuracion.Revision == "" ||
		inv.Configuracion.Secuencia < 1 || len(inv.Consumidores) == 0 {
		return inventarioV3PortalExterno{}, ErrMaterialV3PortalExternoInvalido
	}
	for consumidor, claves := range inv.Consumidores {
		audiencias := audienciasConsumidorPortalExternoV3(consumidor)
		if audiencias == nil || len(claves) != len(audiencias) {
			return inventarioV3PortalExterno{}, ErrMaterialV3PortalExternoInvalido
		}
		for i, clave := range claves {
			if clave.Audiencia != audiencias[i] || !archivoClaveV3PortalExterno.MatchString(clave.Archivo) ||
				clave.ClaveID == "" || clave.Version < 1 || clave.RevisionGobierno < 1 || clave.Orden < 1 || clave.EmisorID == "" {
				return inventarioV3PortalExterno{}, ErrMaterialV3PortalExternoInvalido
			}
		}
	}
	return inv, nil
}

// materialesConsumidorV3PortalExterno reconstruye el material de cada
// audiencia de un consumidor a partir del inventario, sus ficheros y la
// configuración indicada (la vigente, leída del gobierno).
func materialesConsumidorV3PortalExterno(directorio string, inv inventarioV3PortalExterno, consumidor string, configuracion configuracionInventarioV3PortalExterno) ([]materialAtestacionContratacionTemporalDesarrollo, error) {
	claves, ok := inv.Consumidores[consumidor]
	if !ok {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	semilla, err := leerSecretoV3PortalExterno(directorio, inv.Raiz.SemillaArchivo, ed25519.SeedSize, inv.Raiz.SemillaSHA256)
	if err != nil {
		return nil, err
	}
	defer clear(semilla)
	privada := ed25519.NewKeyFromSeed(semilla)
	defer clear(privada)
	spki, err := x509.MarshalPKIXPublicKey(privada.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	huellaSPKI := sha256.Sum256(spki)
	if !huellaHexIgual(inv.Raiz.SPKISHA256, huellaSPKI[:]) {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	resultado := make([]materialAtestacionContratacionTemporalDesarrollo, 0, len(claves))
	for _, clave := range claves {
		secreto, err := leerSecretoV3PortalExterno(directorio, clave.Archivo, sha256.Size, clave.HuellaSecreto)
		if err != nil {
			borrarMaterialesV3PortalExterno(resultado)
			return nil, err
		}
		m := materialAtestacionContratacionTemporalDesarrollo{
			claveID: inv.Raiz.ClaveID, claveVersion: inv.Raiz.Version,
			privada:          append(ed25519.PrivateKey(nil), privada...),
			configuracionRef: configuracion.Revision, configuracionOrden: configuracion.Secuencia,
			configuracionHuella: configuracion.HuellaSHA256,
			publicadaEn:         configuracion.PublicadaEn.UTC(), expiraEn: configuracion.ExpiraEn.UTC(),
			validaDesde: inv.Raiz.ValidaDesde.UTC(), validaHasta: inv.Raiz.ValidaHasta.UTC(),
			spki: append([]byte(nil), spki...), spkiHuella: inv.Raiz.SPKISHA256,
			claveHMACID: clave.ClaveID, claveHMACVersion: clave.Version, claveHMACOrden: clave.Orden,
			claveHMAC: secreto, claveHMACRevision: clave.RevisionGobierno, claveHMACHuella: clave.HuellaGobierno,
			claveHMACSecreto: clave.HuellaSecreto, emisorID: clave.EmisorID, audienciaConsumo: clave.Audiencia,
		}
		if err := reconstruirClavesGobiernoPostgreSQLContratacionTemporalDesarrollo(&m); err != nil {
			m.borrarCopiasEfimeras()
			borrarMaterialesV3PortalExterno(resultado)
			return nil, ErrMaterialV3PortalExternoInvalido
		}
		config, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
			m.configuracionRef, m.configuracionOrden, m.publicadaEn, m.expiraEn, m.raiz)
		if err != nil {
			m.borrarCopiasEfimeras()
			borrarMaterialesV3PortalExterno(resultado)
			return nil, ErrMaterialV3PortalExternoInvalido
		}
		huellaConfig, err := config.HuellaSHA256ParaGobierno()
		if err != nil || huellaConfig != m.configuracionHuella {
			m.borrarCopiasEfimeras()
			borrarMaterialesV3PortalExterno(resultado)
			return nil, ErrMaterialV3PortalExternoInvalido
		}
		m.configuracion = config
		resultado = append(resultado, m)
	}
	return resultado, nil
}

func borrarMaterialesV3PortalExterno(materiales []materialAtestacionContratacionTemporalDesarrollo) {
	for i := range materiales {
		materiales[i].borrarCopiasEfimeras()
	}
}

func leerSecretoV3PortalExterno(directorio, relativa string, tamano int, huella string) ([]byte, error) {
	ruta, valida := rutaMaterialConsultaRRHHDesarrollo(directorio, filepath.FromSlash(relativa))
	if !valida {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, int64(tamano))
	if err != nil || len(contenido) != tamano {
		clear(contenido)
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	resumen := sha256.Sum256(contenido)
	if !huellaHexIgual(huella, resumen[:]) {
		clear(contenido)
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	return contenido, nil
}

func huellaHexIgual(declarada string, calculada []byte) bool {
	valor, err := hex.DecodeString(declarada)
	return err == nil && len(valor) == len(calculada) && hex.EncodeToString(valor) == declarada &&
		subtle.ConstantTimeCompare(valor, calculada) == 1
}

// materialJSONV3PortalExterno es el argumento que esperan las funciones de
// AD3-112: claves del consumidor en su orden, configuración y raíz.
func materialJSONV3PortalExterno(inv inventarioV3PortalExterno, consumidor string, configuracion configuracionInventarioV3PortalExterno) ([]byte, error) {
	claves, ok := inv.Consumidores[consumidor]
	if !ok {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	type clave struct {
		Audiencia        string `json:"audiencia_consumo"`
		ClaveID          string `json:"clave_id"`
		Version          uint64 `json:"version"`
		RevisionGobierno uint64 `json:"revision_gobierno"`
		HuellaGobierno   string `json:"huella_gobierno_sha256"`
		HuellaSecreto    string `json:"huella_secreto_sha256"`
		EmisorID         string `json:"emisor_id"`
	}
	lista := make([]clave, 0, len(claves))
	for _, c := range claves {
		lista = append(lista, clave{c.Audiencia, c.ClaveID, c.Version, c.RevisionGobierno, c.HuellaGobierno, c.HuellaSecreto, c.EmisorID})
	}
	return json.Marshal(map[string]any{
		"claves": lista,
		"configuracion": map[string]any{
			"revision": configuracion.Revision, "secuencia": configuracion.Secuencia,
			"huella_configuracion_sha256": configuracion.HuellaSHA256,
		},
		"raiz": map[string]any{
			"clave_id": inv.Raiz.ClaveID, "version": inv.Raiz.Version, "huella_spki_sha256": inv.Raiz.SPKISHA256,
			"audiencia_despliegue": audienciaAtestacionContratacionTemporalDesarrollo,
			"suite":                confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
		},
	})
}

// consumidorPortalExternoValido comprueba que el nombre pertenece a la lista
// cerrada, antes de enviarlo a PostgreSQL.
func consumidorPortalExternoValido(consumidor string) bool {
	return slices.Contains(consumidoresPortalExternoV3, consumidor)
}
