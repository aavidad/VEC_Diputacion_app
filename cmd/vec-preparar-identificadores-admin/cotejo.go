package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

type entradasCotejadas struct {
	plan           domain.PlanFuentesInicialesAdminV1
	originales     originalesArranque
	certificados   certificadosArranque
	material       materialHMAC
	proveedor      proveedorHMAC
	materialSHA256 string
	cuentas        map[string]cuentasPersona
}

// entradaV1 reproduce los campos del formato v1 que lee
// NuevaFuenteIdentificadoresADMINDesdeArchivo; el comando comprueba después
// que el cargador acepta el archivo escrito.
type entradaV1 struct {
	Persona     string `json:"persona_ref"`
	Cuenta      string `json:"cuenta_ref"`
	Ordinaria   string `json:"cuenta_ordinaria_ref"`
	Certificado string `json:"certificado_sha256"`
	CA          string `json:"ca_sha256"`
	Espacio     string `json:"espacio_identidad"`
	Dominio     string `json:"dominio_hmac_ref"`
	Clave       string `json:"clave_hmac_id"`
	Version     uint64 `json:"clave_hmac_version"`
	Fuente      string `json:"fuente_ref"`
	FuenteSHA   string `json:"fuente_sha256"`
	SujetoID    string `json:"sujeto_id"`
	CuentaID    string `json:"cuenta_id"`
	OrdinariaID string `json:"cuenta_ordinaria_id"`
}
type documentoV1 struct {
	Version  uint64      `json:"version"`
	Entradas []entradaV1 `json:"entradas"`
}

func (e entradasCotejadas) documento() ([]byte, error) {
	p, o, m := e.plan, e.originales, e.material
	ca := p.PoliticaADMIN.CAHuellaSHA256
	if o.Entorno != "desarrollo" || o.Alcance != "sintetico_declarado" || o.OrganizacionRef != p.Organizacion.OrganizacionRef || e.certificados.Entorno != "desarrollo" ||
		!strings.HasPrefix(e.proveedor.EspacioIdentidad, "https://") || !referenciaOpaca(e.proveedor.DominioRef, "idh_") {
		return nil, fallo("entrada_invalida", nil)
	}
	// La fuente HMAC del plan es exactamente el material confirmado.
	if e.materialSHA256 != p.FuenteHMAC.HuellaSHA256 || m.FuenteRef != p.FuenteHMAC.Referencia || m.Version != 1 || m.FuenteVersion != p.FuenteHMAC.Version ||
		m.Esquema != identidad.EsquemaHMACSHA256V1 || m.DominioRef != e.proveedor.DominioRef || m.ClaveID == "" || m.ClaveVersion == 0 || m.ClaveVersion > 1<<63-1 {
		return nil, fallo("entradas_divergentes", nil)
	}
	originales := map[string]originalesPersona{}
	for _, x := range o.Personas {
		if !identificadorOriginal(x.SujetoOriginal, false) || !identificadorOriginal(x.CuentaPrivilegiadaOriginal, true) || !identificadorOriginal(x.CuentaOrdinariaOriginal, true) ||
			x.CuentaPrivilegiadaOriginal == x.CuentaOrdinariaOriginal {
			return nil, fallo("entrada_invalida", nil)
		}
		originales[x.PersonaRef] = x
	}
	certificados := map[string]string{}
	for _, x := range e.certificados.Certificados {
		if !huellaValida(x.CertDERSHA256) || x.CADERSHA256 != ca || x.CertDERSHA256 == ca {
			return nil, fallo("entradas_divergentes", nil)
		}
		certificados[x.PersonaRef] = x.CertDERSHA256
	}
	hmacs := map[string]materialHMACPersona{}
	for _, x := range m.Personas {
		hmacs[x.PersonaRef] = x
	}
	if !mismasPersonas(p, originales) || !mismasPersonas(p, certificados) || !mismasPersonas(p, hmacs) || certificados[p.Personas[0].PersonaRef] == certificados[p.Personas[1].PersonaRef] {
		return nil, fallo("entradas_divergentes", nil)
	}

	prov := e.proveedor
	seud, cerrar, err := bootstrap.NuevoSeudonimizadorSesionDesdeArchivo(bootstrap.ConfiguracionSeudonimosSesionPrivada{
		DirectorioMaterial: prov.DirectorioMaterial, RutaConfiguracionHMAC: prov.RutaConfiguracionHMAC,
		EspacioIdentidad: prov.EspacioIdentidad, DominioRef: prov.DominioRef, EspacioClave: prov.EspacioClave,
		DominioHMAC: prov.DominioHMAC, IncluirCuentaOrdinaria: true})
	if err != nil {
		return nil, fallo("proveedor_no_disponible", err)
	}
	defer cerrar()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()

	doc := documentoV1{Version: 1}
	for _, persona := range p.Personas {
		ref := persona.PersonaRef
		x, h, c := originales[ref], hmacs[ref], e.cuentas[ref]
		if err := cotejarPersona(ctx, seud, prov, m, x, h); err != nil {
			return nil, err
		}
		doc.Entradas = append(doc.Entradas, entradaV1{Persona: ref, Cuenta: c.CuentaPrivilegiadaRef, Ordinaria: c.CuentaOrdinariaRef,
			Certificado: certificados[ref], CA: ca, Espacio: prov.EspacioIdentidad, Dominio: m.DominioRef, Clave: m.ClaveID, Version: m.ClaveVersion,
			Fuente: p.FuenteHMAC.Referencia, FuenteSHA: p.FuenteHMAC.HuellaSHA256,
			SujetoID: x.SujetoOriginal, CuentaID: x.CuentaPrivilegiadaOriginal, OrdinariaID: x.CuentaOrdinariaOriginal})
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, fallo("entrada_invalida", err)
	}
	if len(b) > 32768 {
		return nil, fallo("entrada_invalida", nil)
	}
	return b, nil
}

// cotejarPersona invoca la misma composición que el runtime
// (SeudonimizarAltaConAliasCuentaOrdinaria con referencias aleatorias de
// aserción y sesión) y exige las tres HMAC confirmadas en el material.
func cotejarPersona(ctx context.Context, seud identidad.SeudonimizadorAlta, prov proveedorHMAC, m materialHMAC, x originalesPersona, h materialHMACPersona) error {
	sujeto, err1 := hmacHex(h.Sujeto)
	cuenta, err2 := hmacHex(h.CuentaPrivilegiada)
	ordinaria, err3 := hmacHex(h.CuentaOrdinaria)
	if err := errors.Join(err1, err2, err3); err != nil {
		return fallo("entrada_invalida", err)
	}
	var aleatorio [32]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return fallo("proveedor_no_disponible", err)
	}
	d, alias, err := identidad.SeudonimizarAltaConAliasCuentaOrdinaria(ctx, seud, identidad.IdentificadoresAlta{EspacioIdentidad: prov.EspacioIdentidad,
		AsercionID: hex.EncodeToString(aleatorio[:16]), SesionID: hex.EncodeToString(aleatorio[16:]),
		SujetoID: x.SujetoOriginal, CuentaID: x.CuentaPrivilegiadaOriginal, CuentaOrdinariaID: x.CuentaOrdinariaOriginal}, prov.EspacioIdentidad, prov.DominioRef)
	defer clear(alias)
	switch {
	case err != nil:
		return fallo("proveedor_no_disponible", err)
	case d.Esquema != m.Esquema || d.EspacioIdentidad != prov.EspacioIdentidad || d.DominioRef != m.DominioRef || d.ClaveID != m.ClaveID || d.ClaveVersion != m.ClaveVersion:
		return fallo("coordenadas_divergentes", nil)
	case subtle.ConstantTimeCompare(d.SujetoIDHMAC[:], sujeto[:]) != 1:
		return fallo("sujeto_divergente", nil)
	case subtle.ConstantTimeCompare(d.CuentaIDHMAC[:], cuenta[:]) != 1:
		return fallo("cuenta_divergente", nil)
	case subtle.ConstantTimeCompare(alias, ordinaria[:]) != 1:
		return fallo("alias_ordinario_divergente", nil)
	}
	return nil
}
