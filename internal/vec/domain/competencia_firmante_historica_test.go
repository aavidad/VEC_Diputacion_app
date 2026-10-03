package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func canonCompetenciaFirmantePrueba() CanonCompetenciaFirmanteHistoricaV1 {
	ref := func(s string) ReferenciaHistoricaCompetenciaV1 {
		return ReferenciaHistoricaCompetenciaV1{s, 1, strings.Repeat("a", 64)}
	}
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	en := time.Date(2026, 10, 3, 9, 0, 0, 123000000, time.UTC)
	return CanonCompetenciaFirmanteHistoricaV1{
		Esquema: EsquemaCanonCompetenciaFirmanteHistoricaV1,
		Identidad: IdentidadFirmanteHistoricaV1{
			CertificadoDERSHA256: strings.Repeat("b", 64), PersonaRef: "per_1234567890abcdef1234567890abcdef",
			Cuenta: ref("cuenta:1"), VinculoCertificado: ref("vinculo:1"),
			VinculoCuentaRef: "cuenta:1", VinculoPersonaRef: "per_1234567890abcdef1234567890abcdef",
			VinculoDERSHA256: strings.Repeat("b", 64),
		},
		Competencia: AsignacionFirmanteHistoricaV1{
			Asignacion: ref("asignacion:1"), Rol: ref("rol:1"), ControlRol: ref("control:1"),
			PersonaRef: "per_1234567890abcdef1234567890abcdef", PerfilEsperadoRef: "perfil:ct:firma",
			PerfilActivoRef: "prf_1234567890abcdef1234567890abcdef", AmbitoOrganizacionRef: "org:1",
			AmbitoUnidadRef: "unidad:1", AsignacionRolRef: "rol:1", ControlRolRef: "rol:1",
			VigenteDesde: desde, VigenteHasta: hasta,
		},
		Personal: FuentePersonalFirmanteHistoricaV1{
			Cargo: ref("cargo:1"), EnlaceOcupante: ref("ocupante:1"),
			OcupantePersonaRef: "per_1234567890abcdef1234567890abcdef", CargoRefEnlace: "cargo:1",
			CargoVigenteDesde: desde, CargoVigenteHasta: hasta, EnlaceVigenteDesde: desde, EnlaceVigenteHasta: hasta,
		},
		Recurso: RecursoFirmaHistoricaV1{
			OrganizacionRef: "org:1", UnidadRef: "unidad:1", ExpedienteRef: "exp:1", DocumentoRef: "doc:1",
			RecursoAutorizableRef: "firma:1", RecursoContextoSHA256: strings.Repeat("c", 64),
			Original: ref("original:1"), PDFRaizSHA256: strings.Repeat("a", 64),
			Firmado: ref("firmado:1"), PDFFirmadoSHA256: strings.Repeat("a", 64), NumeroFirmas: 1,
		},
		RelacionCT: RelacionCTFirmanteHistoricaV1{
			ExpedienteRef: "exp:1", UnidadRef: "unidad:1", OrigenRef: "ct050:1", OrigenVersion: 4,
			PruebaSnapshotSHA256: strings.Repeat("d", 64), EventoRef: "evento:1",
			EventoHuellaSHA256: strings.Repeat("e", 64), ConfirmadaEn: desde,
		},
		Accion: "contratacion_temporal.documento.firmar", Finalidad: "formalizacion",
		Motivo: ReferenciaEntradaCatalogo{CatalogoID: "motivo_firma", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("f", 64), EntradaClave: "competencia"},
		Circuito: ref("circuito:1"), PasoRef: "paso:1", PasoOrden: 1, FechaHistorica: en,
	}
}

func TestCanonCompetenciaFirmanteHistoricaDeterministaYRecuperable(t *testing.T) {
	c := canonCompetenciaFirmantePrueba()
	b, err := c.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecuperarCanonCompetenciaFirmanteHistoricaV1(b, h)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := r.Canonico()
	if err != nil || !bytes.Equal(b, b2) {
		t.Fatal("recuperacion no determinista", err)
	}
	if _, err := json.Marshal(c); err == nil {
		t.Fatal("exportacion JSON generica")
	}
	if bytes.Contains(b, []byte("correlacion")) || bytes.Contains(b, []byte("comprobada_en")) {
		t.Fatal("el canon incluye datos efimeros")
	}
	tamper := append([]byte(" "), b...)
	sha := sha256.Sum256(tamper)
	if _, err := RecuperarCanonCompetenciaFirmanteHistoricaV1(tamper, hex.EncodeToString(sha[:])); err == nil {
		t.Fatal("acepto bytes no canonicos con huella recalculada")
	}
}

func TestCanonCompetenciaFirmanteHistoricaCrucesYMaterialVinculante(t *testing.T) {
	base := canonCompetenciaFirmantePrueba()
	primera, err := base.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	cambios := map[string]func(*CanonCompetenciaFirmanteHistoricaV1){
		"certificado": func(c *CanonCompetenciaFirmanteHistoricaV1) {
			c.Identidad.CertificadoDERSHA256 = strings.Repeat("1", 64)
			c.Identidad.VinculoDERSHA256 = strings.Repeat("1", 64)
		},
		"cuenta":   func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Identidad.Cuenta.Version++ },
		"rol":      func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Competencia.Rol.HuellaSHA256 = strings.Repeat("2", 64) },
		"cargo":    func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Personal.Cargo.Version++ },
		"original": func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Recurso.Original.Version++ },
		"pdf firmado": func(c *CanonCompetenciaFirmanteHistoricaV1) {
			c.Recurso.Firmado.HuellaSHA256 = strings.Repeat("3", 64)
			c.Recurso.PDFFirmadoSHA256 = strings.Repeat("3", 64)
		},
		"ct origen": func(c *CanonCompetenciaFirmanteHistoricaV1) { c.RelacionCT.OrigenVersion++ },
		"ct prueba": func(c *CanonCompetenciaFirmanteHistoricaV1) {
			c.RelacionCT.PruebaSnapshotSHA256 = strings.Repeat("4", 64)
		},
		"paso": func(c *CanonCompetenciaFirmanteHistoricaV1) { c.PasoOrden++ },
	}
	for nombre, cambiar := range cambios {
		t.Run(nombre, func(t *testing.T) {
			c := base
			cambiar(&c)
			h, err := c.HuellaSHA256()
			if err != nil || h == primera {
				t.Fatalf("cambio no vinculante: %v", err)
			}
		})
	}
	invalidos := map[string]func(*CanonCompetenciaFirmanteHistoricaV1){
		"persona": func(c *CanonCompetenciaFirmanteHistoricaV1) {
			c.Competencia.PersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"vinculo cuenta":          func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Identidad.VinculoCuentaRef = "cuenta:2" },
		"vinculo certificado":     func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Identidad.VinculoDERSHA256 = strings.Repeat("1", 64) },
		"rol asignacion":          func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Competencia.AsignacionRolRef = "rol:2" },
		"unidad":                  func(c *CanonCompetenciaFirmanteHistoricaV1) { c.RelacionCT.UnidadRef = "unidad:2" },
		"pdf raiz":                func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Recurso.PDFRaizSHA256 = strings.Repeat("3", 64) },
		"vigencia":                func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Competencia.VigenteHasta = c.FechaHistorica },
		"multifirma sin revision": func(c *CanonCompetenciaFirmanteHistoricaV1) { c.Recurso.NumeroFirmas = 2 },
	}
	for nombre, cambiar := range invalidos {
		t.Run(nombre, func(t *testing.T) {
			c := base
			cambiar(&c)
			if _, err := c.Canonico(); err == nil {
				t.Fatal("cruce invalido aceptado")
			}
		})
	}
}

func TestCanonCompetenciaFirmanteHistoricaDelegacion(t *testing.T) {
	c := canonCompetenciaFirmantePrueba()
	desde, hasta := c.Personal.EnlaceVigenteDesde, c.Personal.EnlaceVigenteHasta
	c.Personal.OcupantePersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	c.Personal.Delegacion = &DelegacionFirmanteHistoricaV1{
		Acto:                ReferenciaHistoricaCompetenciaV1{"delegacion:1", 1, strings.Repeat("9", 64)},
		DelegantePersonaRef: c.Personal.OcupantePersonaRef, DelegadoPersonaRef: c.Identidad.PersonaRef,
		CargoRef: c.Personal.Cargo.Referencia, VigenteDesde: desde, VigenteHasta: hasta,
	}
	if _, err := c.Canonico(); err != nil {
		t.Fatal(err)
	}
	c.Personal.Delegacion.DelegadoPersonaRef = c.Personal.OcupantePersonaRef
	if _, err := c.Canonico(); err == nil {
		t.Fatal("delegacion propia aceptada")
	}
	c.Personal.Delegacion.DelegadoPersonaRef = c.Identidad.PersonaRef
	c.Personal.Delegacion.VigenteHasta = c.FechaHistorica
	if _, err := c.Canonico(); err == nil {
		t.Fatal("delegacion caducada aceptada")
	}
}
