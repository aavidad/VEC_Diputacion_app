package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

var ErrCircuitoRRHHInvalido = errors.New("contratacion temporal: circuito RRHH invalido")

// TipoHitoCircuitoRRHH identifica el efecto administrativo. El orden, la fase
// y el perfil se publican en la definicion, no se deducen de esta lista.
type TipoHitoCircuitoRRHH string

const (
	HitoPeticionFirmada       TipoHitoCircuitoRRHH = "peticion_firmada"
	HitoAutorizacionRRHH      TipoHitoCircuitoRRHH = "autorizacion_rrhh"
	HitoCreditoComprobado     TipoHitoCircuitoRRHH = "credito_comprobado"
	HitoOfertaEmitida         TipoHitoCircuitoRRHH = "oferta_emitida"
	HitoAdjudicacion          TipoHitoCircuitoRRHH = "adjudicacion"
	HitoInformeJefatura       TipoHitoCircuitoRRHH = "informe_jefatura"
	HitoIntervencionFavorable TipoHitoCircuitoRRHH = "intervencion_favorable"
	HitoIntervencionReparo    TipoHitoCircuitoRRHH = "intervencion_reparo"
	HitoResolucionFirmada     TipoHitoCircuitoRRHH = "resolucion_firmada"
	HitoGINPIXRegistrado      TipoHitoCircuitoRRHH = "ginpix_registrado"
	HitoFirmaInteresado       TipoHitoCircuitoRRHH = "firma_interesado"
	HitoDevolucion            TipoHitoCircuitoRRHH = "devolucion"
	HitoReinicio              TipoHitoCircuitoRRHH = "reinicio"
	HitoIncorporacion         TipoHitoCircuitoRRHH = "incorporacion"
)

func (t TipoHitoCircuitoRRHH) Valido() bool {
	switch t {
	case HitoPeticionFirmada, HitoAutorizacionRRHH, HitoCreditoComprobado,
		HitoOfertaEmitida, HitoAdjudicacion, HitoInformeJefatura,
		HitoIntervencionFavorable, HitoIntervencionReparo,
		HitoResolucionFirmada, HitoGINPIXRegistrado, HitoFirmaInteresado,
		HitoDevolucion, HitoReinicio, HitoIncorporacion:
		return true
	default:
		return false
	}
}

type TransicionCircuitoRRHH struct {
	Clave                 ClaveCatalogo        `json:"clave"`
	Tipo                  TipoHitoCircuitoRRHH `json:"tipo"`
	Origen                ClaveFase            `json:"origen"`
	Destino               ClaveFase            `json:"destino"`
	RequiereDocumento     bool                 `json:"requiere_documento"`
	RequiereFirma         bool                 `json:"requiere_firma"`
	PerfilClave           ClaveCatalogo        `json:"perfil_clave"`
	FirmasRequeridas      []ClaveCatalogo      `json:"firmas_requeridas,omitempty"`
	AutorizanteCargoClave ClaveCatalogo        `json:"autorizante_cargo_clave,omitempty"`
}

type DefinicionCircuitoRRHH struct {
	Flujo         ReferenciaFlujo          `json:"flujo"`
	EstadoInicial ClaveFase                `json:"estado_inicial"`
	Transiciones  []TransicionCircuitoRRHH `json:"transiciones"`
}

func NuevaDefinicionCircuitoRRHH(ref string, version uint64, inicial ClaveFase, transiciones []TransicionCircuitoRRHH) (DefinicionCircuitoRRHH, error) {
	d := DefinicionCircuitoRRHH{
		Flujo:         ReferenciaFlujo{DefinicionRef: ref, Version: version},
		EstadoInicial: inicial,
		Transiciones:  append([]TransicionCircuitoRRHH(nil), transiciones...),
	}
	huella, err := d.calcularHuella()
	if err != nil {
		return DefinicionCircuitoRRHH{}, err
	}
	d.Flujo.HuellaSHA256 = huella
	if d.Validar() != nil {
		return DefinicionCircuitoRRHH{}, ErrCircuitoRRHHInvalido
	}
	return d, nil
}

func (d DefinicionCircuitoRRHH) Validar() error {
	if d.Flujo.Validar() != nil || !d.EstadoInicial.Valida() ||
		len(d.Transiciones) == 0 || len(d.Transiciones) > 128 {
		return ErrCircuitoRRHHInvalido
	}
	vistas := make(map[ClaveCatalogo]struct{}, len(d.Transiciones))
	tieneEntrada := false
	for _, t := range d.Transiciones {
		if !t.Clave.Valida() || !t.Tipo.Valido() ||
			!t.Origen.Valida() || !t.Destino.Valida() ||
			!t.PerfilClave.Valida() ||
			(t.RequiereFirma && len(t.FirmasRequeridas) == 0) ||
			(!t.RequiereFirma && len(t.FirmasRequeridas) != 0) {
			return ErrCircuitoRRHHInvalido
		}
		if t.Tipo == HitoAutorizacionRRHH {
			if !t.AutorizanteCargoClave.Valida() {
				return ErrCircuitoRRHHInvalido
			}
		} else if t.AutorizanteCargoClave != "" {
			return ErrCircuitoRRHHInvalido
		}
		cargos := make(map[ClaveCatalogo]struct{}, len(t.FirmasRequeridas))
		for _, cargo := range t.FirmasRequeridas {
			if !cargo.Valida() {
				return ErrCircuitoRRHHInvalido
			}
			if _, repetido := cargos[cargo]; repetido {
				return ErrCircuitoRRHHInvalido
			}
			cargos[cargo] = struct{}{}
		}
		if _, existe := vistas[t.Clave]; existe {
			return ErrCircuitoRRHHInvalido
		}
		vistas[t.Clave] = struct{}{}
		tieneEntrada = tieneEntrada || t.Origen == d.EstadoInicial
	}
	if !tieneEntrada {
		return ErrCircuitoRRHHInvalido
	}
	huella, err := d.calcularHuella()
	if err != nil || huella != d.Flujo.HuellaSHA256 {
		return ErrCircuitoRRHHInvalido
	}
	return nil
}

func (d DefinicionCircuitoRRHH) calcularHuella() (string, error) {
	if d.Flujo.DefinicionRef == "" || d.Flujo.Version == 0 {
		return "", ErrCircuitoRRHHInvalido
	}
	material := struct {
		Referencia    string                   `json:"referencia"`
		Version       uint64                   `json:"version"`
		EstadoInicial ClaveFase                `json:"estado_inicial"`
		Transiciones  []TransicionCircuitoRRHH `json:"transiciones"`
	}{d.Flujo.DefinicionRef, d.Flujo.Version, d.EstadoInicial, d.Transiciones}
	b, err := json.Marshal(material)
	if err != nil {
		return "", ErrCircuitoRRHHInvalido
	}
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:]), nil
}

func (d DefinicionCircuitoRRHH) transicion(clave ClaveCatalogo) (TransicionCircuitoRRHH, bool) {
	for _, t := range d.Transiciones {
		if t.Clave == clave {
			return t, true
		}
	}
	return TransicionCircuitoRRHH{}, false
}

type HitoCircuitoRRHH struct {
	Secuencia                uint64               `json:"secuencia"`
	VersionExpedienteEntrada uint64               `json:"version_expediente_entrada"`
	Clave                    ClaveCatalogo        `json:"clave"`
	ActuacionClave           ClaveCatalogo        `json:"actuacion_clave"`
	Tipo                     TipoHitoCircuitoRRHH `json:"tipo"`
	Origen                   ClaveFase            `json:"origen"`
	Destino                  ClaveFase            `json:"destino"`
	ActorRef                 string               `json:"actor_ref"`
	PerfilClave              ClaveCatalogo        `json:"perfil_clave"`
	PerfilRef                string               `json:"perfil_ref"`
	UnidadRef                string               `json:"unidad_ref"`
	DocumentoRef             string               `json:"documento_ref,omitempty"`
	HuellaDocumentoSHA256    string               `json:"huella_documento_sha256,omitempty"`
	ActoAutorizacionRef      string               `json:"acto_autorizacion_ref,omitempty"`
	AutorizanteRef           string               `json:"autorizante_ref,omitempty"`
	CargoAutorizanteClave    ClaveCatalogo        `json:"cargo_autorizante_clave,omitempty"`
	Firmas                   []FirmaCircuitoRRHH  `json:"firmas,omitempty"`
	CreditoRef               string               `json:"credito_ref,omitempty"`
	OfertaRef                string               `json:"oferta_ref,omitempty"`
	AdjudicacionRef          string               `json:"adjudicacion_ref,omitempty"`
	RetornoRef               string               `json:"retorno_ref,omitempty"`
	ActoIncorporacionRef     string               `json:"acto_incorporacion_ref,omitempty"`
	HuellaContratoSHA256     string               `json:"huella_contrato_sha256,omitempty"`
	ReciboRef                string               `json:"recibo_ref"`
	RegistradoEn             time.Time            `json:"registrado_en"`
}

type FirmaCircuitoRRHH struct {
	CargoClave            ClaveCatalogo `json:"cargo_clave"`
	FirmaRef              string        `json:"firma_ref"`
	FirmanteRef           string        `json:"firmante_ref"`
	HuellaDocumentoSHA256 string        `json:"huella_documento_sha256"`
}

func (f FirmaCircuitoRRHH) validar() error {
	if !f.CargoClave.Valida() || !referenciaValida(f.FirmaRef) ||
		!referenciaValida(f.FirmanteRef) ||
		!huellaValida(f.HuellaDocumentoSHA256) {
		return ErrCircuitoRRHHInvalido
	}
	return nil
}

func (h HitoCircuitoRRHH) validar() error {
	if h.Secuencia == 0 || h.VersionExpedienteEntrada == 0 ||
		!h.Clave.Valida() || !h.ActuacionClave.Valida() || !h.Tipo.Valido() ||
		!h.Origen.Valida() || !h.Destino.Valida() ||
		!referenciaValida(h.ActorRef) || !h.PerfilClave.Valida() ||
		!referenciaValida(h.PerfilRef) ||
		!referenciaValida(h.UnidadRef) || !referenciaValida(h.ReciboRef) ||
		!instanteCanonico(h.RegistradoEn) {
		return ErrCircuitoRRHHInvalido
	}
	for _, ref := range []string{h.DocumentoRef, h.ActoAutorizacionRef,
		h.AutorizanteRef, h.CreditoRef, h.OfertaRef,
		h.AdjudicacionRef, h.RetornoRef, h.ActoIncorporacionRef} {
		if ref != "" && !referenciaValida(ref) {
			return ErrCircuitoRRHHInvalido
		}
	}
	if h.HuellaContratoSHA256 != "" && !huellaValida(h.HuellaContratoSHA256) {
		return ErrCircuitoRRHHInvalido
	}
	if (h.DocumentoRef == "") != (h.HuellaDocumentoSHA256 == "") ||
		h.HuellaDocumentoSHA256 != "" && !huellaValida(h.HuellaDocumentoSHA256) {
		return ErrCircuitoRRHHInvalido
	}
	cargos := make(map[ClaveCatalogo]struct{}, len(h.Firmas))
	for _, firma := range h.Firmas {
		if firma.validar() != nil ||
			firma.HuellaDocumentoSHA256 != h.HuellaDocumentoSHA256 {
			return ErrCircuitoRRHHInvalido
		}
		if _, repetida := cargos[firma.CargoClave]; repetida {
			return ErrCircuitoRRHHInvalido
		}
		cargos[firma.CargoClave] = struct{}{}
	}
	switch h.Tipo {
	case HitoAutorizacionRRHH:
		if h.ActoAutorizacionRef == "" || h.AutorizanteRef == "" ||
			!h.CargoAutorizanteClave.Valida() {
			return ErrCircuitoRRHHInvalido
		}
	case HitoCreditoComprobado:
		if h.CreditoRef == "" {
			return ErrCircuitoRRHHInvalido
		}
	case HitoOfertaEmitida:
		if h.OfertaRef == "" {
			return ErrCircuitoRRHHInvalido
		}
	case HitoAdjudicacion:
		if h.OfertaRef == "" || h.AdjudicacionRef == "" {
			return ErrCircuitoRRHHInvalido
		}
	case HitoIntervencionReparo:
		if h.RetornoRef == "" {
			return ErrCircuitoRRHHInvalido
		}
	case HitoIncorporacion:
		if h.ActoIncorporacionRef == "" {
			return ErrCircuitoRRHHInvalido
		}
	case HitoDevolucion:
		if h.HuellaContratoSHA256 == "" {
			return ErrCircuitoRRHHInvalido
		}
	}
	return nil
}

type CircuitoAdministrativo struct {
	Definicion   ReferenciaFlujo    `json:"definicion"`
	EstadoActual ClaveFase          `json:"estado_actual"`
	Hitos        []HitoCircuitoRRHH `json:"hitos"`
}

func NuevoCircuitoAdministrativo(d DefinicionCircuitoRRHH) (CircuitoAdministrativo, error) {
	if d.Validar() != nil {
		return CircuitoAdministrativo{}, ErrCircuitoRRHHInvalido
	}
	return CircuitoAdministrativo{Definicion: d.Flujo, EstadoActual: d.EstadoInicial, Hitos: []HitoCircuitoRRHH{}}, nil
}

func (c CircuitoAdministrativo) clonar() CircuitoAdministrativo {
	hitos := make([]HitoCircuitoRRHH, len(c.Hitos))
	copy(hitos, c.Hitos)
	c.Hitos = hitos
	for i := range c.Hitos {
		c.Hitos[i].Firmas = append([]FirmaCircuitoRRHH(nil), c.Hitos[i].Firmas...)
	}
	return c
}

func (e Expediente) circuitoValido() bool {
	c := e.Circuito
	if c == nil || c.Definicion != e.Flujo || c.Definicion.Validar() != nil ||
		c.Hitos == nil ||
		len(c.Hitos) > 128 {
		return false
	}
	origen := e.Actuaciones[0].FaseDestino
	versionAnterior := uint64(0)
	for i, h := range c.Hitos {
		if h.validar() != nil || h.Secuencia != uint64(i+1) ||
			h.VersionExpedienteEntrada < versionAnterior ||
			h.VersionExpedienteEntrada >= e.Version || h.Origen != origen {
			return false
		}
		a := e.Actuaciones[h.VersionExpedienteEntrada]
		if a.AccionClave != h.ActuacionClave || a.ActorRef != h.ActorRef ||
			a.UnidadRef != h.UnidadRef || a.ReciboRef != h.ReciboRef ||
			!a.RealizadaEn.Equal(h.RegistradoEn) {
			return false
		}
		origen = h.Destino
		versionAnterior = h.VersionExpedienteEntrada
	}
	return origen == c.EstadoActual
}

// HabilitaInformeJefaturaCircuitoRRHH exige que la oferta y la adjudicación
// verificadas precedan al informe en el flujo nuevo.
func (e Expediente) HabilitaInformeJefaturaCircuitoRRHH() bool {
	if e.Circuito == nil || e.Asignacion == nil || e.InformeJuridico != nil ||
		len(e.Circuito.Hitos) < 2 {
		return false
	}
	ultimo := e.Circuito.Hitos[len(e.Circuito.Hitos)-1]
	penultimo := e.Circuito.Hitos[len(e.Circuito.Hitos)-2]
	return ultimo.Tipo == HitoAdjudicacion &&
		penultimo.Tipo == HitoOfertaEmitida &&
		ultimo.OfertaRef == penultimo.OfertaRef &&
		ultimo.VersionExpedienteEntrada == penultimo.VersionExpedienteEntrada
}

// HabilitaInformeSubsanacionCircuitoRRHH liga el informe nuevo al reparo
// vigente ya subsanado. La firma del documento se comprueba al registrar el
// hito posterior desde una fuente de firmas acreditada.
func (e Expediente) HabilitaInformeSubsanacionCircuitoRRHH() bool {
	if e.Circuito == nil || !e.PuedeReemitirInformeTrasSubsanacion() ||
		len(e.Circuito.Hitos) == 0 {
		return false
	}
	ultimo := e.Circuito.Hitos[len(e.Circuito.Hitos)-1]
	return ultimo.Tipo == HitoIntervencionReparo &&
		e.Fiscalizacion != nil && e.Fiscalizacion.Retorno != nil &&
		ultimo.RetornoRef == e.Fiscalizacion.Retorno.RetornoRef
}

// AdjuntarHitosCircuito enlaza hitos acreditados a la última actuación real
// del expediente. No crea otra versión: el adaptador confirma actuación,
// hitos, auditoría y outbox en la misma transacción y CAS.
func (e Expediente) AdjuntarHitosCircuito(
	definicion DefinicionCircuitoRRHH,
	versionEsperada uint64,
	datos []HitoCircuitoRRHH,
) (Expediente, error) {
	if e.Validar() != nil || e.Circuito == nil ||
		definicion.Validar() != nil || definicion.Flujo != e.Flujo ||
		versionEsperada != e.Version || e.Version < 2 ||
		len(datos) == 0 || len(datos) > 2 {
		return Expediente{}, ErrTransicionInvalida
	}
	ultima := e.Actuaciones[len(e.Actuaciones)-1]
	siguiente := e.Clonar()
	for _, h := range datos {
		transicion, ok := definicion.transicion(h.Clave)
		if !ok || transicion.Origen != siguiente.Circuito.EstadoActual ||
			transicion.PerfilClave != h.PerfilClave ||
			transicion.AutorizanteCargoClave != h.CargoAutorizanteClave ||
			transicion.RequiereDocumento && h.DocumentoRef == "" ||
			!firmasCircuitoCoinciden(transicion.FirmasRequeridas, h.Firmas) ||
			h.ActuacionClave != ultima.AccionClave ||
			h.ActorRef != ultima.ActorRef || h.UnidadRef != ultima.UnidadRef ||
			h.ReciboRef != ultima.ReciboRef ||
			!h.RegistradoEn.Equal(ultima.RealizadaEn) ||
			h.Secuencia != 0 || h.VersionExpedienteEntrada != 0 ||
			h.Origen != "" || h.Destino != "" || h.Tipo != "" {
			return Expediente{}, ErrTransicionInvalida
		}
		h.Tipo, h.Origen, h.Destino = transicion.Tipo, transicion.Origen, transicion.Destino
		h.Secuencia = uint64(len(siguiente.Circuito.Hitos) + 1)
		h.VersionExpedienteEntrada = e.Version - 1
		if h.validar() != nil || !siguiente.permiteHitoCircuito(h) {
			return Expediente{}, ErrTransicionInvalida
		}
		h.Firmas = append([]FirmaCircuitoRRHH(nil), h.Firmas...)
		siguiente.Circuito.Hitos = append(siguiente.Circuito.Hitos, h)
		siguiente.Circuito.EstadoActual = h.Destino
	}
	if siguiente.Validar() != nil {
		return Expediente{}, ErrTransicionInvalida
	}
	return siguiente, nil
}

func firmasCircuitoCoinciden(requeridas []ClaveCatalogo, firmas []FirmaCircuitoRRHH) bool {
	if len(requeridas) != len(firmas) {
		return false
	}
	for i, firma := range firmas {
		if firma.validar() != nil || firma.CargoClave != requeridas[i] {
			return false
		}
	}
	return true
}

func (e Expediente) permiteHitoCircuito(h HitoCircuitoRRHH) bool {
	hitos := e.Circuito.Hitos
	inicio := 0
	for i := len(hitos) - 1; i >= 0; i-- {
		if hitos[i].Tipo == HitoReinicio {
			inicio = i + 1
			break
		}
	}
	ultimo := func(tipo TipoHitoCircuitoRRHH) *HitoCircuitoRRHH {
		for i := len(hitos) - 1; i >= inicio; i-- {
			if hitos[i].Tipo == tipo {
				return &hitos[i]
			}
		}
		return nil
	}
	despues := func(tipo TipoHitoCircuitoRRHH, anterior *HitoCircuitoRRHH) bool {
		v := ultimo(tipo)
		return v != nil && (anterior == nil || v.Secuencia > anterior.Secuencia)
	}
	peticion, autorizacion, credito := ultimo(HitoPeticionFirmada), ultimo(HitoAutorizacionRRHH), ultimo(HitoCreditoComprobado)
	oferta, adjudicacion := ultimo(HitoOfertaEmitida), ultimo(HitoAdjudicacion)
	informe, favorable, reparo := ultimo(HitoInformeJefatura), ultimo(HitoIntervencionFavorable), ultimo(HitoIntervencionReparo)
	resolucion, ginpix, firma := ultimo(HitoResolucionFirmada), ultimo(HitoGINPIXRegistrado), ultimo(HitoFirmaInteresado)
	devolucion, incorporacion := ultimo(HitoDevolucion), ultimo(HitoIncorporacion)
	switch h.Tipo {
	case HitoPeticionFirmada:
		return peticion == nil && e.Analisis != nil &&
			e.ViaCobertura == nil && e.Asignacion == nil &&
			(len(hitos) == 0 || hitos[len(hitos)-1].Tipo == HitoReinicio)
	case HitoAutorizacionRRHH:
		return peticion != nil && autorizacion == nil && e.Analisis != nil &&
			e.ViaCobertura == nil && e.Asignacion == nil
	case HitoCreditoComprobado:
		return autorizacion != nil && credito == nil && e.Analisis != nil &&
			e.ViaCobertura != nil && e.Asignacion == nil &&
			e.Analisis.HabilitaAvance() &&
			e.Analisis.ValidacionRC.Resultado == RCValidada &&
			h.CreditoRef == e.Analisis.ValidacionRC.ReciboRef
	case HitoOfertaEmitida:
		return credito != nil && oferta == nil && e.Asignacion != nil &&
			e.InformeJuridico == nil
	case HitoAdjudicacion:
		return oferta != nil && adjudicacion == nil && e.Asignacion != nil &&
			e.InformeJuridico == nil && h.OfertaRef == oferta.OfertaRef
	case HitoInformeJefatura:
		if h.HuellaContratoSHA256 == "" ||
			e.InformeJuridico == nil || h.DocumentoRef != e.InformeJuridico.DocumentoRef {
			return false
		}
		if reparo != nil && !despues(HitoInformeJefatura, reparo) {
			return h.RetornoRef == reparo.RetornoRef
		}
		if devolucion != nil && !despues(HitoInformeJefatura, devolucion) {
			return h.RetornoRef == devolucion.RetornoRef && h.RetornoRef != ""
		}
		return adjudicacion != nil && informe == nil && h.RetornoRef == ""
	case HitoIntervencionFavorable, HitoIntervencionReparo:
		if informe == nil || !despues(HitoInformeJefatura, favorable) ||
			!despues(HitoInformeJefatura, reparo) ||
			h.HuellaContratoSHA256 != informe.HuellaContratoSHA256 ||
			e.Fiscalizacion == nil ||
			e.Fiscalizacion.InformeJuridicoRef != e.InformeJuridico.InformeRef {
			return false
		}
		if h.Tipo == HitoIntervencionReparo {
			return e.Fiscalizacion.Resultado == FiscalizacionDesfavorable &&
				e.Fiscalizacion.Retorno != nil &&
				h.RetornoRef == e.Fiscalizacion.Retorno.RetornoRef
		}
		return h.RetornoRef == "" &&
			(e.Fiscalizacion.Resultado == FiscalizacionFavorable ||
				e.Fiscalizacion.Resultado == FiscalizacionFavorableConObservaciones)
	case HitoResolucionFirmada:
		if informe == nil ||
			h.HuellaContratoSHA256 != informe.HuellaContratoSHA256 ||
			(resolucion != nil && (devolucion == nil || devolucion.Secuencia < resolucion.Secuencia)) {
			return false
		}
		if favorable != nil && favorable.Secuencia > informe.Secuencia &&
			favorable.HuellaContratoSHA256 == h.HuellaContratoSHA256 {
			return true
		}
		return devolucion != nil && incorporacion == nil &&
			devolucion.HuellaContratoSHA256 == h.HuellaContratoSHA256 &&
			favorable != nil && favorable.Secuencia < devolucion.Secuencia &&
			informe.Secuencia > devolucion.Secuencia
	case HitoGINPIXRegistrado:
		return resolucion != nil && (ginpix == nil || ginpix.Secuencia < resolucion.Secuencia)
	case HitoFirmaInteresado:
		return ginpix != nil && (firma == nil || firma.Secuencia < ginpix.Secuencia)
	case HitoIncorporacion:
		return firma != nil && incorporacion == nil && h.ActoIncorporacionRef != ""
	case HitoDevolucion:
		return resolucion != nil && incorporacion == nil &&
			(devolucion == nil || devolucion.Secuencia < resolucion.Secuencia) &&
			h.HuellaContratoSHA256 == resolucion.HuellaContratoSHA256 &&
			h.RetornoRef != ""
	case HitoReinicio:
		return incorporacion != nil && hitos[len(hitos)-1].Tipo != HitoReinicio &&
			h.ActoIncorporacionRef == incorporacion.ActoIncorporacionRef
	default:
		return false
	}
}
