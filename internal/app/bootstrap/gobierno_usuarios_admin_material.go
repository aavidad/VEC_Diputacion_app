package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrGobiernoUsuariosAdmin = errors.New("gobierno_usuarios_admin_no_disponible")

// ConfiguracionMaterialUsuariosAdmin procede exclusivamente del mantenimiento
// privado. No publica gobierno ni acepta una solicitud HTTP.
type ConfiguracionMaterialUsuariosAdmin struct {
	DirectorioMaterial, RutaConfiguracionHMAC string
	ArchivoSemillaRaiz                        string
	Raiz                                      administracion.MaterialRaizPerfilesV3
	Gobierno                                  administracion.GobiernoConfianzaPerfilesV3
	Entradas                                  []DescriptorClaveUsuariosAdmin
	PrefijoEvidencia                          string
}

type DescriptorClaveUsuariosAdmin struct {
	Audiencia, Dominio, PrefijoClave, EmisorID string
	Version, RevisionGobierno                  uint64
	ValidaDesde, ValidaHasta                   time.Time
}

// MaterialUsuariosAdmin oculta todos los secretos en los formatos habituales.
// Cerrar invalida también el firmante; la preparación no acredita publicación.
type MaterialUsuariosAdmin struct {
	mu             sync.RWMutex
	config         administracion.ConfiguracionConfianzaUsuariosV3
	firmante       ports.FirmanteAtestacionesAutorizacionV3
	cerrarFirmante func()
	cerrado        bool
}

func (*MaterialUsuariosAdmin) String() string               { return "[MATERIAL-USUARIOS-ADMIN-PRIVADO]" }
func (m *MaterialUsuariosAdmin) GoString() string           { return m.String() }
func (m *MaterialUsuariosAdmin) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, m.String()) }
func (m *MaterialUsuariosAdmin) MarshalJSON() ([]byte, error) {
	return []byte(`{"material":"oculto"}`), nil
}

// PrepararMaterialUsuariosAdmin reutiliza el cargador/derivador y el firmante
// existentes. La raíz debe coincidir con la pública externa fijada: nunca rota
// una raíz ni genera un maestro para obtener un positivo.
func PrepararMaterialUsuariosAdmin(ctx context.Context, cfg ConfiguracionMaterialUsuariosAdmin, reloj ports.Reloj) (*MaterialUsuariosAdmin, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaBootstrapNula(reloj) || len(cfg.Entradas) != 2 || cfg.ArchivoSemillaRaiz == "" || !identificadorSesionDesarrolloValido(cfg.PrefijoEvidencia) {
		return nil, ErrGobiernoUsuariosAdmin
	}
	material, err := cargarMaterialIdempotenciaDesarrollo(cfg.DirectorioMaterial, cfg.RutaConfiguracionHMAC)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	defer material.borrar()
	d, err := nuevoDerivadorIdentidadOperacionDesarrollo(&material)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	defer d.borrar()
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(d, reloj.Ahora())
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	defer borrarBytes(base.privada)
	defer borrarBytes(base.claveHMAC)
	if cfg.Raiz.Estado != confianza.EstadoClaveAtestacionAutorizacionV3Activa || !cfg.Raiz.RevocadaEn.IsZero() || cfg.Raiz.Audiencia != audienciaAtestacionContratacionTemporalDesarrollo {
		return nil, ErrGobiernoUsuariosAdmin
	}
	cabecera := administracion.ConfiguracionConfianzaUsuariosV3{Raiz: cfg.Raiz, Gobierno: cfg.Gobierno}
	cabecera.Cabecera.FormatoVersion = domain.VersionFormatoAtestacionAutorizacionV3
	cabecera.Cabecera.Suite = confianza.SuiteAtestacionAutorizacionV3COSEEdDSA
	cabecera.Cabecera.ClaveID = cfg.Raiz.ClaveID
	cabecera.Cabecera.Audiencia = cfg.Raiz.Audiencia
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(cfg.Raiz.ClaveID, cfg.Raiz.Version, cfg.Raiz.Publica, cfg.Raiz.Audiencia, cfg.Raiz.Estado, cfg.Raiz.ValidaDesde, cfg.Raiz.ValidaHasta, cfg.Raiz.RevocadaEn)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	g, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(cfg.Gobierno.Revision, cfg.Gobierno.Secuencia, cfg.Gobierno.PublicadaEn, cfg.Gobierno.ExpiraEn, raiz)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	h, err := g.HuellaSHA256ParaGobierno()
	if err != nil || h != cfg.Gobierno.HuellaSHA256 {
		return nil, ErrGobiernoUsuariosAdmin
	}
	admitidas := map[string]bool{administracion.AudienciaUsuariosListarV3: false, administracion.AudienciaUsuariosConsultarV3: false}
	dominios := map[string]bool{}
	prefijos := map[string]bool{}
	m := &MaterialUsuariosAdmin{config: cabecera}
	correcto := false
	defer func() {
		if !correcto {
			m.Cerrar()
		}
	}()
	for _, e := range cfg.Entradas {
		if usada, ok := admitidas[e.Audiencia]; !ok || usada || dominios[e.Dominio] || prefijos[e.PrefijoClave] || !identificadorSesionDesarrolloValido(e.Dominio) || !identificadorSesionDesarrolloValido(e.PrefijoClave) || e.Version == 0 || e.RevisionGobierno == 0 || reloj.Ahora().Before(e.ValidaDesde) || !reloj.Ahora().Before(e.ValidaHasta) {
			return nil, ErrGobiernoUsuariosAdmin
		}
		admitidas[e.Audiencia] = true
		dominios[e.Dominio] = true
		prefijos[e.PrefijoClave] = true
		n, err := derivarMaterialConsumidorV3Desarrollo(base, descriptorMaterialConsumidorV3Desarrollo{Audiencia: e.Audiencia, Dominio: e.Dominio, Prefijo: e.PrefijoClave, ProveedorNominal: "gobierno_usuarios_admin"})
		if err != nil {
			return nil, ErrGobiernoUsuariosAdmin
		}
		m.config.EntradasCapacidad = append(m.config.EntradasCapacidad, administracion.MaterialCapacidadPerfilesV3{Audiencia: e.Audiencia, ClaveID: n.claveHMACID, EmisorID: e.EmisorID, HuellaGobierno: n.claveHMACHuella, Version: e.Version, RevisionGobierno: e.RevisionGobierno, Material: n.claveHMAC, ValidaDesde: e.ValidaDesde, ValidaHasta: e.ValidaHasta, Estado: confianza.EstadoClaveHMACCapacidadAtestacionV3Emision})
	}
	m.firmante, m.cerrarFirmante, err = NuevoFirmanteAtestacionV3DesdeArchivo(ConfiguracionFirmanteAtestacionV3Privado{ClaveID: cfg.Raiz.ClaveID, Audiencia: cfg.Raiz.Audiencia, PrefijoEvidencia: cfg.PrefijoEvidencia, ArchivoSemilla: cfg.ArchivoSemillaRaiz, PublicaEsperada: cfg.Raiz.Publica}, reloj)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	correcto = true
	return m, nil
}

func (m *MaterialUsuariosAdmin) Cerrar() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.config.EntradasCapacidad {
		borrarBytes(m.config.EntradasCapacidad[i].Material)
	}
	if m.cerrarFirmante != nil {
		m.cerrarFirmante()
		m.cerrarFirmante = nil
	}
	m.firmante = nil
	m.cerrado = true
}

// Configuracion conserva los bytes administrados por Cerrar. El consumidor no
// debe retenerlos después del cierre del proceso.
func (m *MaterialUsuariosAdmin) Configuracion() (administracion.ConfiguracionConfianzaUsuariosV3, ports.FirmanteAtestacionesAutorizacionV3, error) {
	if m == nil {
		return administracion.ConfiguracionConfianzaUsuariosV3{}, nil, ErrGobiernoUsuariosAdmin
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cerrado {
		return administracion.ConfiguracionConfianzaUsuariosV3{}, nil, ErrGobiernoUsuariosAdmin
	}
	return m.config, m.firmante, nil
}

// EscribirMaterialPrivado entrega exclusivamente el payload destinado a AD188.
// El llamante conserva el fichero/IO privado; nunca es un DTO de pantalla.
func (m *MaterialUsuariosAdmin) EscribirMaterialPrivado(w io.Writer) error {
	if m == nil || dependenciaBootstrapNula(w) {
		return ErrGobiernoUsuariosAdmin
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cerrado {
		return ErrGobiernoUsuariosAdmin
	}
	type entrada struct {
		Audiencia      string    `json:"audiencia"`
		ClaveID        string    `json:"clave_id"`
		Version        uint64    `json:"version"`
		Revision       uint64    `json:"revision_gobierno"`
		HuellaGobierno string    `json:"huella_gobierno_sha256"`
		Secreto        []byte    `json:"secreto_hmac"`
		HuellaSecreto  string    `json:"huella_secreto_sha256"`
		EmisorID       string    `json:"emisor_id"`
		Desde          time.Time `json:"valida_desde"`
		Hasta          time.Time `json:"valida_hasta"`
	}
	var es []entrada
	for _, e := range m.config.EntradasCapacidad {
		h := sha256.Sum256(e.Material)
		es = append(es, entrada{e.Audiencia, e.ClaveID, e.Version, e.RevisionGobierno, e.HuellaGobierno, e.Material, hex.EncodeToString(h[:]), e.EmisorID, e.ValidaDesde, e.ValidaHasta})
	}
	b, err := json.Marshal(struct {
		Claves []entrada `json:"claves"`
	}{es})
	defer borrarBytes(b)
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	if _, err = w.Write(b); err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	return nil
}
