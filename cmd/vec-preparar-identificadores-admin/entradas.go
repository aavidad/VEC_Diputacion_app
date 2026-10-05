package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

// Formatos privados que produce el circuito de arranque de fuentes iniciales.
// La decodificación es exacta: un campo de más o de menos es un rechazo.

type originalesArranque struct {
	Entorno         string               `json:"entorno"`
	Alcance         string               `json:"alcance"`
	OrganizacionRef string               `json:"organizacion_ref"`
	Personas        [2]originalesPersona `json:"personas"`
}
type originalesPersona struct {
	Nombre                     string `json:"nombre"`
	PersonaRef                 string `json:"persona_ref"`
	CuentaOrdinariaOriginal    string `json:"cuenta_ordinaria_original"`
	CuentaPrivilegiadaOriginal string `json:"cuenta_privilegiada_original"`
	SujetoOriginal             string `json:"sujeto_original"`
}

type certificadosArranque struct {
	Entorno      string                `json:"entorno"`
	Certificados [2]certificadoPersona `json:"certificados"`
}
type certificadoPersona struct {
	PersonaRef      string `json:"persona_ref"`
	NombreSintetico string `json:"nombre_sintetico"`
	CertDERSHA256   string `json:"cert_der_sha256"`
	CADERSHA256     string `json:"ca_der_sha256"`
}

// materialHMAC es el material oficial que el proveedor DEV calculó para la
// fuente HMAC del plan; su SHA256 es plan.fuente_hmac.huella_sha256.
type materialHMAC struct {
	Version       uint64                 `json:"version"`
	FuenteRef     string                 `json:"fuente_ref"`
	FuenteVersion uint64                 `json:"fuente_version"`
	Esquema       string                 `json:"esquema_hmac"`
	DominioRef    string                 `json:"dominio_hmac_ref"`
	ClaveID       string                 `json:"clave_hmac_id"`
	ClaveVersion  uint64                 `json:"clave_hmac_version"`
	Personas      [2]materialHMACPersona `json:"personas"`
}
type materialHMACPersona struct {
	PersonaRef         string `json:"persona_ref"`
	CuentaOrdinaria    string `json:"cuenta_ordinaria_id_hmac_hex"`
	CuentaPrivilegiada string `json:"cuenta_privilegiada_id_hmac_hex"`
	Sujeto             string `json:"sujeto_id_hmac_hex"`
}

// proveedorHMAC es la misma configuración de identidad que lee vec-admin
// (bloque "identidad" de su configuración privada). El arranque escribe
// incluir_cuenta_ordinaria=false y vec-admin exige true; ese valor solo decide
// si el proveedor admite la cuenta ordinaria en la misma llamada y no altera
// las huellas de sujeto, cuenta ni alias ordinario. El cotejo usa siempre true,
// como el runtime, y el campo no se compara con la configuración de vec-admin.
type proveedorHMAC struct {
	DirectorioMaterial     string `json:"directorio_material"`
	RutaConfiguracionHMAC  string `json:"ruta_configuracion_hmac"`
	EspacioIdentidad       string `json:"espacio_identidad"`
	DominioRef             string `json:"dominio_ref"`
	EspacioClave           string `json:"espacio_clave"`
	DominioHMAC            string `json:"dominio_hmac"`
	IncluirCuentaOrdinaria bool   `json:"incluir_cuenta_ordinaria"`
}

// Acuse confirmado de vec-aplicar-fuentes-admin (envoltura completa).
type acuseFuentes struct {
	Estado           string         `json:"estado"`
	Codigo           *string        `json:"codigo"`
	Recibo           *reciboFuentes `json:"recibo"`
	Replay           bool           `json:"replay"`
	AuditoriaIntento auditoriaAcuse `json:"auditoria_intento"`
}
type auditoriaAcuse struct {
	AuditoriaRef   string `json:"auditoria_ref"`
	Secuencia      uint64 `json:"secuencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}
type reciboFuentes struct {
	Esquema               string                 `json:"esquema"`
	CA                    reciboParcial[datosCA] `json:"ca"`
	IS                    reciboParcial[datosIS] `json:"is"`
	ReciboRef             string                 `json:"recibo_ref"`
	OperacionRef          string                 `json:"operacion_ref"`
	PlanSHA256            string                 `json:"plan_sha256"`
	PreimagenSHA256       string                 `json:"preimagen_sha256"`
	ConfiguracionSHA256   string                 `json:"configuracion_sha256"`
	OperadorLogin         string                 `json:"operador_login"`
	AprobacionRef         string                 `json:"aprobacion_ref"`
	AuditoriaRef          string                 `json:"auditoria_ref"`
	AuditoriaSecuencia    uint64                 `json:"auditoria_secuencia"`
	AuditoriaHuellaSHA256 string                 `json:"auditoria_huella_sha256"`
	RegistradaEn          string                 `json:"registrada_en"`
}

// reciboParcial agrupa los campos comunes de los recibos CA e IS.
type reciboParcial[D any] struct {
	Esquema       string `json:"esquema"`
	Version       uint64 `json:"version"`
	ReciboRef     string `json:"recibo_ref"`
	OperacionRef  string `json:"operacion_ref"`
	PlanSHA256    string `json:"plan_sha256"`
	AprobacionRef string `json:"aprobacion_ref"`
	AlcanceFuente string `json:"alcance_fuente"`
	RegistradaEn  string `json:"registrada_en"`
	Datos         D      `json:"datos"`
	HuellaSHA256  string `json:"huella_sha256"`
}

type cuentasPersona struct {
	PersonaRef, CuentaOrdinariaRef, CuentaPrivilegiadaRef string
}

// cuentasDesdeAcuse exige un recibo permitido del mismo plan y que CA e IS
// declaren exactamente las mismas cuentas para cada persona.
func cuentasDesdeAcuse(a acuseFuentes, plan domain.PlanFuentesInicialesAdminV1, huellaPlan string) (map[string]cuentasPersona, bool) {
	r := a.Recibo
	if a.Estado != "permitido" || a.Codigo != nil || r == nil || r.Esquema != "vec.admin.fuentes-confirmadas.v1" || r.PlanSHA256 != huellaPlan ||
		r.OperacionRef != plan.OperacionRef || r.CA.PlanSHA256 != huellaPlan || r.IS.PlanSHA256 != huellaPlan ||
		r.CA.Esquema != "vec.ca.fuentes-iniciales-admin.v1" || r.IS.Esquema != "vec.is.fuentes-iniciales-admin.v1" {
		return nil, false
	}
	ca, is := r.CA.Datos, r.IS.Datos
	if ca.OrganizacionRef != plan.Organizacion.OrganizacionRef || is.PoliticaRef != plan.PoliticaADMIN.PoliticaRef {
		return nil, false
	}
	cuentas := map[string]cuentasPersona{}
	vistas := map[string]bool{}
	for _, x := range ca.Personas {
		if !referenciaOpaca(x.CuentaOrdinariaRef, "cta_") || !referenciaOpaca(x.CuentaPrivilegiadaRef, "cta_") || vistas[x.CuentaOrdinariaRef] || vistas[x.CuentaPrivilegiadaRef] ||
			x.CuentaOrdinariaRef == x.CuentaPrivilegiadaRef {
			return nil, false
		}
		vistas[x.CuentaOrdinariaRef], vistas[x.CuentaPrivilegiadaRef] = true, true
		cuentas[x.PersonaRef] = cuentasPersona{x.PersonaRef, x.CuentaOrdinariaRef, x.CuentaPrivilegiadaRef}
	}
	personasIS := map[string]bool{}
	for _, x := range is.Personas {
		y, ok := cuentas[x.PersonaRef]
		if !ok || personasIS[x.PersonaRef] || y.CuentaOrdinariaRef != x.CuentaOrdinariaRef || y.CuentaPrivilegiadaRef != x.CuentaPrivilegiadaRef {
			return nil, false
		}
		personasIS[x.PersonaRef] = true
	}
	return cuentas, mismasPersonas(plan, cuentas)
}

type datosCA struct {
	OrganizacionRef     string             `json:"organizacion_ref"`
	OrganizacionVersion uint64             `json:"organizacion_version"`
	Personas            [2]cuentaPersonaCA `json:"personas"`
}
type cuentaPersonaCA struct {
	PersonaRef              string `json:"persona_ref"`
	CuentaOrdinariaRef      string `json:"cuenta_ordinaria_ref"`
	CuentaPrivilegiadaRef   string `json:"cuenta_privilegiada_ref"`
	VersionTitularidad      uint64 `json:"version_titularidad"`
	PersonaVersion          uint64 `json:"persona_version"`
	ProyeccionCuentaVersion uint64 `json:"proyeccion_cuenta_version"`
}
type datosIS struct {
	Personas    [2]cuentaPersonaIS `json:"personas"`
	PoliticaRef string             `json:"politica_ref"`
}
type cuentaPersonaIS struct {
	PersonaRef            string `json:"persona_ref"`
	CuentaOrdinariaRef    string `json:"cuenta_ordinaria_ref"`
	CuentaPrivilegiadaRef string `json:"cuenta_privilegiada_ref"`
	VersionTitularidad    uint64 `json:"version_titularidad"`
}

func mismasPersonas[V any](plan domain.PlanFuentesInicialesAdminV1, m map[string]V) bool {
	if len(m) != len(plan.Personas) {
		return false
	}
	for _, p := range plan.Personas {
		if _, ok := m[p.PersonaRef]; !ok {
			return false
		}
	}
	return true
}

func huellaSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func huellaValida(v string) bool {
	return domain.HuellaAdministracionPerfilesValida(v) && v != strings.Repeat("0", 64)
}

var errHMACInvalida = errors.New("hmac_invalida")

func hmacHex(v string) ([32]byte, error) {
	var r [32]byte
	if !huellaValida(v) {
		return r, errHMACInvalida
	}
	b, err := hex.DecodeString(v)
	if err != nil {
		return r, err
	}
	if len(b) != 32 {
		return r, errHMACInvalida
	}
	copy(r[:], b)
	return r, nil
}

func referenciaOpaca(v, prefijo string) bool {
	if !strings.HasPrefix(v, prefijo) || len(v) < len(prefijo)+22 || len(v) > len(prefijo)+128 {
		return false
	}
	for _, c := range v[len(prefijo):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// identificadorOriginal replica el contrato del cargador del runtime: ASCII
// visible, hasta 512 bytes; las cuentas, además, en minúsculas.
func identificadorOriginal(v string, cuenta bool) bool {
	if v == "" || len(v) > 512 || cuenta && strings.ToLower(v) != v {
		return false
	}
	for _, c := range v {
		if c <= 32 || c >= 127 {
			return false
		}
	}
	return true
}

// mismoProveedor compara los seis campos que determinan las huellas.
func mismoProveedor(a, b proveedorHMAC) bool {
	return a.DirectorioMaterial == b.DirectorioMaterial && a.RutaConfiguracionHMAC == b.RutaConfiguracionHMAC && a.EspacioIdentidad == b.EspacioIdentidad &&
		a.DominioRef == b.DominioRef && a.EspacioClave == b.EspacioClave && a.DominioHMAC == b.DominioHMAC
}
