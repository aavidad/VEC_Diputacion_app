package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	gocose "github.com/veraison/go-cose"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	audienciaAtestacionRutasDietasDesarrollo = "vec:desarrollo:dietas:rutas:atestacion:v3"
	audienciaConsumoRutasDietasDesarrollo    = "vec_dietas_rutas_v1.acceso.v1"
)

var errMaterialRutasDietasDesarrollo = errors.New("bootstrap: material de autorizacion de rutas dietas no disponible")

// El cargador privado de la composición debe decodificar estos datos con límites
// y campos desconocidos rechazados. Todas las versiones y ventanas proceden del
// gobierno provisionado: este proveedor no publica claves ni prolonga su vigencia.
type datosMaterialRutasDietasDesarrollo struct {
	redaccionMaterialRutasDietas
	ClaveID                 string                                                   `json:"clave_id"`
	ClaveVersion            uint64                                                   `json:"clave_version"`
	EstadoRaiz              confianzaatestacion.EstadoClaveAtestacionAutorizacionV3  `json:"estado_raiz"`
	RevocadaEn              time.Time                                                `json:"revocada_en"`
	EstadoHMAC              confianzaatestacion.EstadoClaveHMACCapacidadAtestacionV3 `json:"estado_hmac"`
	HMACRevocadaEn          time.Time                                                `json:"hmac_revocada_en"`
	PrivadaEd25519          []byte                                                   `json:"privada_ed25519"`
	PublicaEd25519          []byte                                                   `json:"publica_ed25519"`
	ValidaDesde             time.Time                                                `json:"valida_desde"`
	ValidaHasta             time.Time                                                `json:"valida_hasta"`
	ConfiguracionReferencia string                                                   `json:"configuracion_referencia"`
	ConfiguracionOrden      uint64                                                   `json:"configuracion_orden"`
	PublicadaEn             time.Time                                                `json:"publicada_en"`
	ExpiraEn                time.Time                                                `json:"expira_en"`
	ClaveHMACID             string                                                   `json:"clave_hmac_id"`
	ClaveHMACVersion        uint64                                                   `json:"clave_hmac_version"`
	MaterialHMAC            []byte                                                   `json:"material_hmac"`
	EmisorID                string                                                   `json:"emisor_id"`
	HMACValidaDesde         time.Time                                                `json:"hmac_valida_desde"`
	HMACValidaHasta         time.Time                                                `json:"hmac_valida_hasta"`
	RevisionGobierno        uint64                                                   `json:"revision_gobierno"`
	HuellaGobierno          string                                                   `json:"huella_gobierno"`
}

func (d *datosMaterialRutasDietasDesarrollo) borrarCopiasEfimeras() {
	if d != nil {
		clear(d.PrivadaEd25519)
		clear(d.MaterialHMAC)
	}
}

type materialAtestacionRutasDietasDesarrollo struct {
	redaccionMaterialRutasDietas
	claveID       string
	privada       ed25519.PrivateKey
	raiz          confianzaatestacion.RaizPublicaAtestacionAutorizacionV3
	configuracion confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV3
	capacidad     confianzaatestacion.ClaveHMACCapacidadAtestacionV3
}

func (m *materialAtestacionRutasDietasDesarrollo) borrarCopiasEfimeras() {
	if m != nil {
		clear(m.privada)
		m.capacidad = confianzaatestacion.ClaveHMACCapacidadAtestacionV3{}
	}
}

// Sólo reconstruye las representaciones nominales del material previamente
// provisionado. Las audiencias Dietas nunca se reciben del cliente HTTP.
func nuevoMaterialAtestacionRutasDietasDesarrollo(d datosMaterialRutasDietasDesarrollo) (materialAtestacionRutasDietasDesarrollo, error) {
	vacio := materialAtestacionRutasDietasDesarrollo{}
	if len(d.PrivadaEd25519) != ed25519.PrivateKeySize || len(d.PublicaEd25519) != ed25519.PublicKeySize ||
		!bytes.Equal(ed25519.PrivateKey(d.PrivadaEd25519).Public().(ed25519.PublicKey), d.PublicaEd25519) {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	raiz, err := confianzaatestacion.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		d.ClaveID, d.ClaveVersion, ed25519.PublicKey(d.PublicaEd25519), audienciaAtestacionRutasDietasDesarrollo,
		d.EstadoRaiz, d.ValidaDesde, d.ValidaHasta, d.RevocadaEn,
	)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	configuracion, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		d.ConfiguracionReferencia, d.ConfiguracionOrden, d.PublicadaEn, d.ExpiraEn, raiz,
	)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	capacidad, err := confianzaatestacion.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		d.ClaveHMACID, d.ClaveHMACVersion, d.MaterialHMAC, d.EmisorID, audienciaConsumoRutasDietasDesarrollo,
		d.EstadoHMAC, d.HMACValidaDesde, d.HMACValidaHasta,
		d.HMACRevocadaEn, d.RevisionGobierno, d.HuellaGobierno,
	)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	return materialAtestacionRutasDietasDesarrollo{claveID: d.ClaveID, privada: append(ed25519.PrivateKey(nil), d.PrivadaEd25519...), raiz: raiz, configuracion: configuracion, capacidad: capacidad}, nil
}

type proveedorMaterialAccesoRutasDietasDesarrollo struct {
	redaccionMaterialRutasDietas
	mu        sync.Mutex
	atestador *aplicacionvec.ServicioAtestacionesAutorizacionV3
	confianza *confianzaatestacion.ServicioConfianzaAtestacionAutorizacionV3
	emisor    *confianzaatestacion.EmisorCapacidadesAtestacionAutorizacionV3
	firmante  *firmanteAtestacionRutasDietasDesarrollo
	raiz      confianzaatestacion.RaizPublicaAtestacionAutorizacionV3
}

func nuevoProveedorMaterialAccesoRutasDietasDesarrollo(m materialAtestacionRutasDietasDesarrollo, reloj puertosvec.Reloj) (*proveedorMaterialAccesoRutasDietasDesarrollo, error) {
	if len(m.privada) != ed25519.PrivateKeySize {
		return nil, errMaterialRutasDietasDesarrollo
	}
	confianza, err := confianzaatestacion.NuevoServicioConfianzaAtestacionAutorizacionV3(m.configuracion, reloj)
	if err != nil {
		return nil, errMaterialRutasDietasDesarrollo
	}
	emisor, err := confianzaatestacion.NuevoEmisorCapacidadesAtestacionAutorizacionV3(m.capacidad, reloj)
	if err != nil {
		return nil, errMaterialRutasDietasDesarrollo
	}
	firmante := &firmanteAtestacionRutasDietasDesarrollo{claveID: m.claveID, privada: append(ed25519.PrivateKey(nil), m.privada...), reloj: reloj}
	atestador, err := aplicacionvec.NuevoServicioAtestacionesAutorizacionV3(dominiovec.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: dominiovec.VersionFormatoAtestacionAutorizacionV3,
		Suite:          confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        m.claveID, Audiencia: audienciaAtestacionRutasDietasDesarrollo,
	}, firmante)
	if err != nil {
		clear(firmante.privada)
		return nil, errMaterialRutasDietasDesarrollo
	}
	return &proveedorMaterialAccesoRutasDietasDesarrollo{atestador: atestador, confianza: confianza, emisor: emisor, firmante: firmante, raiz: m.raiz}, nil
}

// Cerrar impide nuevas emisiones y borra la copia Ed25519 poseída. El adaptador
// central no expone borrado de su copia HMAC: se libera su referencia, sin
// atribuir al recolector de Go una garantía de borrado físico de memoria.
func (p *proveedorMaterialAccesoRutasDietasDesarrollo) Cerrar() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.firmante != nil {
		clear(p.firmante.privada)
	}
	p.atestador, p.confianza, p.emisor, p.firmante = nil, nil, nil, nil
}

// ProveerMaterialAccesoRutas conserva la concesión ya registrada; no vuelve a
// decidir permisos ni registra otra concesión. El consumidor SQL vuelve a
// verificar gobierno, revocación y consumo único en su transacción.
func (p *proveedorMaterialAccesoRutasDietasDesarrollo) ProveerMaterialAccesoRutas(
	ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	contextoOriginal dominiovec.ResultadoContextoActorRegistradoV2,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if ctx == nil || p == nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	if err := ctx.Err(); err != nil {
		return vacio, errors.Join(errMaterialRutasDietasDesarrollo, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.atestador == nil || p.confianza == nil || p.emisor == nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	datos, err := solicitud.Datos()
	if err != nil || (datos.Accion != "dietas.ruta.catalogo.consultar" && datos.Accion != "dietas.ruta.calculo.solicitar") {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	contexto, err := contextoOriginal.Clonar()
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	motivo := datos.ReferenciaMotivo
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, contexto)
	registro, errRegistro := confirmacion.Datos()
	if err != nil || errRegistro != nil || datos.VinculoAutenticacionActor.ValidarPara(contexto) != nil ||
		confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(registro.RegistradaEn) {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	atestacion, err := p.atestador.Atestar(ctx, decision, motivo, contexto)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	prueba, err := p.confianza.Verificar(ctx, solicitud, decision, motivo, contexto, atestacion)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	capacidad, err := p.emisor.Emitir(ctx, solicitud, decision, motivo, contexto, atestacion, prueba)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	material, err := confianzaatestacion.NuevoMaterialConsumoAutorizacionAtestadaV3(solicitud, decision, motivo, contexto, atestacion, prueba, capacidad, p.raiz)
	if err != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	exportacion, err := material.ExportarMaterialParaConsumidor()
	if err != nil || exportacion.ResumenCapacidad().AudienciaConsumo() != audienciaConsumoRutasDietasDesarrollo {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	if err := ctx.Err(); err != nil {
		return vacio, errors.Join(errMaterialRutasDietasDesarrollo, err)
	}
	return exportacion, nil
}

type firmanteAtestacionRutasDietasDesarrollo struct {
	redaccionMaterialRutasDietas
	claveID string
	privada ed25519.PrivateKey
	reloj   puertosvec.Reloj
}

func (f *firmanteAtestacionRutasDietasDesarrollo) FirmarAtestacionAutorizacionV3(ctx context.Context, solicitud puertosvec.SolicitudFirmaAtestacionAutorizacionV3) (puertosvec.ResultadoFirmaAtestacionAutorizacionV3, error) {
	vacio := puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}
	if ctx == nil || f == nil || f.reloj == nil || len(f.privada) != ed25519.PrivateKeySize {
		return vacio, puertosvec.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	cabecera, err := solicitud.Cabecera()
	if err != nil || cabecera.ClaveID != f.claveID || cabecera.Audiencia != audienciaAtestacionRutasDietasDesarrollo ||
		cabecera.FormatoVersion != dominiovec.VersionFormatoAtestacionAutorizacionV3 || cabecera.Suite != confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA {
		return vacio, puertosvec.ErrFirmaAtestacionNoDisponible
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		return vacio, puertosvec.ErrFirmaAtestacionNoDisponible
	}
	defer clear(mensaje)
	aad, err := confianzaatestacion.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		return vacio, puertosvec.ErrFirmaAtestacionNoDisponible
	}
	sobre := gocose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	sobre.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(f.claveID)
	sobre.Payload = mensaje
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, f.privada)
	if err != nil || sobre.Sign(rand.Reader, aad, firmante) != nil {
		return vacio, puertosvec.ErrFirmaAtestacionNoDisponible
	}
	sobre.Payload = nil
	sobre.Headers.RawProtected, sobre.Headers.RawUnprotected = nil, nil
	firma, err := sobre.MarshalCBOR()
	if err != nil {
		return vacio, puertosvec.ErrFirmaAtestacionNoDisponible
	}
	defer clear(firma)
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	huella := sha256.Sum256(mensaje)
	return puertosvec.NuevoResultadoFirmaAtestacionAutorizacionV3(solicitud, firma, "evidencia:firma:dietas:rutas:"+hex.EncodeToString(huella[:8]), f.reloj.Ahora())
}

// Impide que fmt o slog vuelquen los campos privados al diagnosticar composición.
type redaccionMaterialRutasDietas struct{}

func (redaccionMaterialRutasDietas) String() string     { return "[material rutas dietas redactado]" }
func (r redaccionMaterialRutasDietas) GoString() string { return r.String() }
func (r redaccionMaterialRutasDietas) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r redaccionMaterialRutasDietas) LogValue() slog.Value { return slog.StringValue(r.String()) }
func (redaccionMaterialRutasDietas) MarshalJSON() ([]byte, error) {
	return nil, errMaterialRutasDietasDesarrollo
}
