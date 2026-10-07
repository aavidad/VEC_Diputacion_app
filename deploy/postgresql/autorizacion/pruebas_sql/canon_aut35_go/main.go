// Emite vectores SQL de igualdad de bytes contra el canon Go V1.
// Uso: go run ./deploy/postgresql/autorizacion/pruebas_sql/canon_aut35_go > /tmp/aut35_vectores.sql
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	d "vec-diputacion-granada/internal/vec/domain"
)

func main() {
	ref := func(s string) d.ReferenciaHistoricaCompetenciaV1 {
		return d.ReferenciaHistoricaCompetenciaV1{Referencia: s, Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	}
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	en := time.Date(2026, 10, 3, 9, 0, 0, 123456000, time.UTC)
	persona := "per_1234567890abcdef1234567890abcdef"
	c := d.CanonCompetenciaFirmanteHistoricaV1{
		Esquema: d.EsquemaCanonCompetenciaFirmanteHistoricaV1,
		Identidad: d.IdentidadFirmanteHistoricaV1{
			CertificadoDERSHA256: strings.Repeat("b", 64), PersonaRef: persona,
			Persona: ref(persona), Cuenta: ref("cuenta:1"), VinculoCuentaPersona: ref("vinculo-cuenta-persona:1"),
			CuentaPersonaCuentaRef: "cuenta:1", CuentaPersonaPersonaRef: persona,
			VinculoCertificado: ref("vinculo:1"), VinculoCuentaRef: "cuenta:1", VinculoPersonaRef: persona,
			VinculoDERSHA256: strings.Repeat("b", 64),
		},
		Competencia: d.AsignacionFirmanteHistoricaV1{
			Asignacion: ref("asignacion:1"), Rol: ref("rol:ct_direccion_rrhh:v1"), RolID: "ct_direccion_rrhh",
			ControlRol: ref("control:1"), PersonaRef: persona, PerfilEsperadoRef: "perfil:ct:firma",
			PerfilActivoRef: "prf_1234567890abcdef1234567890abcdef", ModuloID: "contratacion_temporal",
			TipoRecurso: "documento", RecursoRef: "doc:1", AmbitoOrganizacionRef: "org:1",
			AmbitoUnidadRef: "unidad:1", AsignacionRolRef: "rol:ct_direccion_rrhh:v1",
			ControlRolRef: "rol:ct_direccion_rrhh:v1", VigenteDesde: desde, VigenteHasta: hasta,
		},
		Personal: d.FuentePersonalFirmanteHistoricaV1{
			Cargo: ref("cargo:1"), EnlaceOcupante: ref("ocupante:1"), OcupantePersonaRef: persona,
			CargoRefEnlace: "cargo:1", CargoVigenteDesde: desde, CargoVigenteHasta: hasta,
			EnlaceVigenteDesde: desde, EnlaceVigenteHasta: hasta,
		},
		Recurso: d.RecursoFirmaHistoricaV1{
			OrganizacionRef: "org:1", UnidadRef: "unidad:1", ExpedienteRef: "exp:1", DocumentoRef: "doc:1",
			RecursoAutorizableRef: "doc:1", ModuloID: "contratacion_temporal", TipoRecurso: "documento",
			RecursoContextoSHA256: strings.Repeat("c", 64), Original: ref("original:1"),
			PDFRaizSHA256: strings.Repeat("a", 64), Firmado: ref("firmado:1"),
			PDFFirmadoSHA256: strings.Repeat("a", 64), NumeroFirmas: 1,
		},
		RelacionCT: d.RelacionCTFirmanteHistoricaV1{
			ExpedienteRef: "exp:1", UnidadRef: "unidad:1", OrigenRef: "ct050:1", OrigenVersion: 4,
			PruebaSnapshotSHA256: strings.Repeat("d", 64), EventoRef: "evento:1",
			EventoHuellaSHA256: strings.Repeat("e", 64), ConfirmadaEn: desde,
		},
		Accion: "contratacion_temporal.documento.firmar", Finalidad: "formalizacion",
		Motivo: d.ReferenciaEntradaCatalogo{CatalogoID: "motivo_firma", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("f", 64), EntradaClave: "competencia"},
		Circuito: ref("circuito:1"), PasoRef: "paso:1", PasoOrden: 1, FechaHistorica: en,
	}
	fmt.Println("\\set ON_ERROR_STOP on\nBEGIN;\nSET LOCAL ROLE vec_autorizacion_propietario;\nSET LOCAL timezone='UTC';")
	vector := func(nombre string, v d.CanonCompetenciaFirmanteHistoricaV1) {
		b, err := v.Canonico()
		if err != nil {
			panic(nombre + ": " + err.Error())
		}
		fmt.Printf("DO $v$ BEGIN IF convert_to(vec_autorizacion.canon_json_competencia_firmante_ct_v1(convert_from(decode('%s','hex'),'UTF8')::jsonb,'raiz'),'UTF8') IS DISTINCT FROM decode('%s','hex') THEN RAISE EXCEPTION 'AUT35 vector %s no coincide con Go V1'; END IF; END $v$;\n", hex.EncodeToString(b), hex.EncodeToString(b), nombre)
	}
	vector("titular_microsegundo", c)
	delegado := c
	delegado.Personal.OcupantePersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	delegado.Personal.Delegacion = &d.DelegacionFirmanteHistoricaV1{
		Acto: ref("delegacion:1"), DelegantePersonaRef: delegado.Personal.OcupantePersonaRef,
		DelegadoPersonaRef: persona, CargoRef: delegado.Personal.Cargo.Referencia,
		VigenteDesde: desde, VigenteHasta: hasta,
	}
	vector("delegacion", delegado)
	segunda := c
	segunda.Recurso.NumeroFirmas = 2
	r := ref("revision:1")
	segunda.Recurso.EntradaRevision = &r
	vector("dos_firmas", segunda)
	escapada := c
	escapada.PasoRef = "paso:entrecomillas\\\"<&>"
	vector("comillas_barra_html", escapada)
	for _, s := range []string{"&<>\u2028\u2029", "comilla:\" barra:\\"} {
		b, err := json.Marshal(s)
		if err != nil {
			panic(err)
		}
		fmt.Printf("DO $v$ BEGIN IF vec_autorizacion.canon_texto_json_go_ct_v1(convert_from(decode('%s','hex'),'UTF8')) IS DISTINCT FROM convert_from(decode('%s','hex'),'UTF8') THEN RAISE EXCEPTION 'AUT35 escalar Go no coincide'; END IF; END $v$;\n", hex.EncodeToString([]byte(s)), hex.EncodeToString(b))
	}
	fmt.Println("ROLLBACK;\n\\echo AUT35_GO_BYTES_OK")
}
