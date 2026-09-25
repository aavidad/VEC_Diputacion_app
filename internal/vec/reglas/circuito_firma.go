package reglas

import (
	"context"
	"errors"
	"regexp"
	"strconv"
)

// Un circuito de firma describe, por documento, los pasos ordenados de firma
// o visto bueno: quién interviene, qué condición abre el paso, qué habilita la
// firma y qué ocurre ante una devolución o una sustitución del firmante. Cada
// entrada del catálogo es un paso (unidad «ninguna»). Los valores concretos
// (documentos, cargos, orden, habilitaciones) viven en el catálogo; aquí solo
// se fija el vocabulario cerrado que el código sabe interpretar.
//
// El circuito no firma ni autoriza: el cargo es informativo y perfil_ref es
// una referencia opaca que la autoridad de permisos deberá resolver. Una
// firma sin portafirmas corporativo no tiene eficacia administrativa.

// ErrCircuitoFirmaInvalido: el catálogo no forma circuitos completos.
var ErrCircuitoFirmaInvalido = errors.New("reglas: circuito de firma no valido")

// AccionFirma distingue la firma del visto bueno.
type AccionFirma string

const (
	AccionFirmaFirma      AccionFirma = "firma"
	AccionFirmaVistoBueno AccionFirma = "visto_bueno"
)

// CondicionPasoFirma es lo que debe ocurrir para que el paso quede abierto.
type CondicionPasoFirma string

const (
	CondicionBorradorGenerado  CondicionPasoFirma = "borrador_generado"
	CondicionFirmaPasoAnterior CondicionPasoFirma = "firma_paso_anterior"
)

// HabilitacionFirma es lo que permite la firma del paso.
type HabilitacionFirma string

const (
	HabilitaSiguientePaso        HabilitacionFirma = "siguiente_paso"
	HabilitaRemisionIntervencion HabilitacionFirma = "remision_intervencion"
	HabilitaEnvioNotificacion    HabilitacionFirma = "envio_notificacion"
	HabilitaEnvioComunicacion    HabilitacionFirma = "envio_comunicacion"
	HabilitaCierreCircuito       HabilitacionFirma = "cierre_circuito"
)

// DevolucionFirma indica adónde vuelve el documento si el firmante lo devuelve
// o lo rechaza.
type DevolucionFirma string

const (
	DevuelveARedaccion   DevolucionFirma = "vuelve_a_redaccion"
	DevuelvePasoAnterior DevolucionFirma = "vuelve_paso_anterior"
)

// SustitucionFirma indica si otra persona puede firmar en lugar del cargo.
type SustitucionFirma string

const (
	SustitucionSuplenteDesignado SustitucionFirma = "suplente_designado"
	SustitucionNoAdmitida        SustitucionFirma = "no_admitida"
)

// EstadoPasoFirma es el estado visible de un paso.
type EstadoPasoFirma string

const (
	EstadoPasoPendienteFirma EstadoPasoFirma = "pendiente_firma"
	EstadoPasoEnEspera       EstadoPasoFirma = "en_espera"
	EstadoPasoFirmado        EstadoPasoFirma = "firmado"
	EstadoPasoDevuelto       EstadoPasoFirma = "devuelto"
)

const maximoPasosCircuitoFirma = 16

// perfilRefValido admite referencias opacas del tipo «perfil:ct:jefatura».
var perfilRefValido = regexp.MustCompile(`^[a-z][a-z0-9._:-]{2,127}$`)

// PasoFirma es un paso resuelto del circuito.
type PasoFirma struct {
	Orden       int
	Cargo       string
	PerfilRef   string
	Accion      AccionFirma
	Condicion   CondicionPasoFirma
	Habilita    HabilitacionFirma
	Devolucion  DevolucionFirma
	Sustitucion SustitucionFirma
	// Referencia es catalogo:version:entrada del paso.
	Referencia string
}

// CircuitoDocumento agrupa los pasos de un documento en orden.
type CircuitoDocumento struct {
	Documento string
	Etiqueta  string
	Pasos     []PasoFirma
}

// CircuitoFirma es el conjunto de circuitos vigente con su procedencia.
type CircuitoFirma struct {
	Documentos     []CircuitoDocumento
	CatalogoID     string
	Version        int
	HuellaCatalogo string
	PaqueteEjemplo bool
}

// CircuitoFirma resuelve el catálogo vigente como circuitos por documento.
func (r *Resolutor) CircuitoFirma(ctx context.Context) (CircuitoFirma, error) {
	vigentes, err := r.Reglas(ctx)
	if err != nil {
		return CircuitoFirma{}, err
	}
	return CircuitoFirmaDesdeReglas(vigentes)
}

// CircuitoFirmaDesdeReglas agrupa las reglas por documento en el orden de
// aparición. Exige pasos 1..n sin huecos, la condición coherente con la
// posición, «siguiente_paso» en todos salvo el último y un vocabulario
// conocido: un circuito incompleto invalida el catálogo entero.
func CircuitoFirmaDesdeReglas(vigentes []Regla) (CircuitoFirma, error) {
	if len(vigentes) == 0 {
		return CircuitoFirma{}, ErrCircuitoFirmaInvalido
	}
	circuito := CircuitoFirma{
		CatalogoID: vigentes[0].ReferenciaEntrada.CatalogoID, Version: vigentes[0].ReferenciaEntrada.CatalogoVersion,
		HuellaCatalogo: vigentes[0].HuellaCatalogo, PaqueteEjemplo: vigentes[0].PaqueteEjemplo,
	}
	indices := map[string]int{}
	for _, regla := range vigentes {
		paso, documento, etiqueta, err := pasoDesdeRegla(regla)
		if err != nil {
			return CircuitoFirma{}, err
		}
		indice, existe := indices[documento]
		if !existe {
			indice = len(circuito.Documentos)
			indices[documento] = indice
			circuito.Documentos = append(circuito.Documentos, CircuitoDocumento{Documento: documento, Etiqueta: etiqueta})
		}
		actual := &circuito.Documentos[indice]
		if actual.Etiqueta != etiqueta || paso.Orden != len(actual.Pasos)+1 || paso.Orden > maximoPasosCircuitoFirma {
			return CircuitoFirma{}, ErrCircuitoFirmaInvalido
		}
		actual.Pasos = append(actual.Pasos, paso)
	}
	for _, documento := range circuito.Documentos {
		for indice, paso := range documento.Pasos {
			ultimo := indice == len(documento.Pasos)-1
			if ultimo == (paso.Habilita == HabilitaSiguientePaso) {
				return CircuitoFirma{}, ErrCircuitoFirmaInvalido
			}
		}
	}
	return circuito, nil
}

func pasoDesdeRegla(regla Regla) (PasoFirma, string, string, error) {
	a := regla.Atributos
	orden, err := strconv.Atoi(a["paso"])
	paso := PasoFirma{
		Orden: orden, Cargo: a["cargo"], PerfilRef: a["perfil_ref"],
		Accion: AccionFirma(a["accion"]), Condicion: CondicionPasoFirma(a["condicion"]),
		Habilita: HabilitacionFirma(a["habilita"]), Devolucion: DevolucionFirma(a["devolucion"]),
		Sustitucion: SustitucionFirma(a["sustitucion"]), Referencia: regla.Referencia,
	}
	documento, etiqueta := a["documento"], a["documento_etiqueta"]
	if err != nil || orden < 1 || strconv.Itoa(orden) != a["paso"] || regla.Unidad != UnidadNinguna ||
		!claveCanonica(documento) || etiqueta == "" || paso.Cargo == "" || !perfilRefValido.MatchString(paso.PerfilRef) ||
		(paso.Accion != AccionFirmaFirma && paso.Accion != AccionFirmaVistoBueno) ||
		(orden == 1) != (paso.Condicion == CondicionBorradorGenerado) ||
		(orden > 1 && paso.Condicion != CondicionFirmaPasoAnterior) ||
		!habilitacionConocida(paso.Habilita) ||
		(paso.Devolucion != DevuelveARedaccion && paso.Devolucion != DevuelvePasoAnterior) ||
		(orden == 1 && paso.Devolucion == DevuelvePasoAnterior) ||
		(paso.Sustitucion != SustitucionSuplenteDesignado && paso.Sustitucion != SustitucionNoAdmitida) {
		return PasoFirma{}, "", "", ErrCircuitoFirmaInvalido
	}
	return paso, documento, etiqueta, nil
}

func habilitacionConocida(h HabilitacionFirma) bool {
	switch h {
	case HabilitaSiguientePaso, HabilitaRemisionIntervencion, HabilitaEnvioNotificacion,
		HabilitaEnvioComunicacion, HabilitaCierreCircuito:
		return true
	default:
		return false
	}
}

// EstadoPasoSinFirmas es el estado de un paso cuando el documento aún no
// tiene firmas registradas: el primero espera su firma y los demás esperan al
// anterior. Mientras no exista registro de firmas es el único estado posible.
func EstadoPasoSinFirmas(orden int) EstadoPasoFirma {
	if orden == 1 {
		return EstadoPasoPendienteFirma
	}
	return EstadoPasoEnEspera
}
