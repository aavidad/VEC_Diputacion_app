package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRegistroFirmaV2UnaFachadaConservaReciboReplay(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(map[bool]string{false: "primera", true: "replay"}[replay], func(t *testing.T) {
			m, d := fixtureRegistroFirmaV2(t)
			if replay {
				m.ComprobadaEn = m.ComprobadaEn.Add(10 * time.Minute)
			}
			canon, err := m.Canonico()
			if err != nil {
				t.Fatal(err)
			}
			h, err := m.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			ref, version := m.DocumentoCustodiaRef, m.DocumentoCustodiaVersion
			fecha := time.Date(2026, 10, 3, 11, 59, 0, 0, time.UTC)
			w := reciboFirmaSQLV2{reciboFirmaSQL118: reciboFirmaSQL118{FirmaRef: "firma:registrada", ReciboRef: "recibo:registrado", Secuencia: m.Secuencia, Resultado: "firmado",
				ExpedienteVersion: m.VersionExpediente, ActorRef: m.FirmantePrincipalRef, PerfilRef: m.PerfilActivoOperadorRef, RegistradaEn: fecha, SolicitudHuella: h,
				YaRegistrada: replay, DocumentoCustodia: &ref, VersionCustodia: &version},
				CompetenciaEvidenciaRef:          "evidencia:competencia-firmante-ct:" + strings.Repeat("d", 64),
				CompetenciaEvidenciaHuellaSHA256: strings.Repeat("e", 64)}
			contenido, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido, sql: registrarFirmaSQL172, argumentos: 13}
			tx.inspeccionar = func(args []any) {
				if args[1] != m.ComprobadaEn {
					t.Fatal("observación del verificador sustituida")
				}
				var recibido DescriptorConstructorFirmaV2
				if json.Unmarshal(args[12].([]byte), &recibido) != nil || recibido.FechaHistorica != nil || recibido.Accion != d.Accion || recibido.Recurso.RecursoContextoSHA256 != d.Recurso.RecursoContextoSHA256 {
					t.Fatal("descriptor confunde competencia, efecto o fecha histórica")
				}
			}
			p := &poolFirmaV2Prueba{tx: tx}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: p}
			v, err := r.RegistrarFirmaVerificadaV2(context.Background(), m, capacidadRegistroFirmaV2Prueba(t, m))
			if err != nil || v.ReciboRef != w.ReciboRef || v.FirmaRef != w.FirmaRef || !v.RegistradaEn.Equal(fecha) || v.YaRegistrada != replay || tx.commits != 1 || tx.rollbacks != 0 || p.inicios != 1 || tx.consultas != 1 {
				t.Fatalf("recibo cambiado o TX no única: %v", err)
			}
		})
	}
}

func TestRegistroFirmaV2RevierteSinExponerRecibo(t *testing.T) {
	for _, caso := range []string{"recibo_incoherente", "evidencia_ausente", "evidencia_alterada", "revocado_replay", "commit_incierto"} {
		t.Run(caso, func(t *testing.T) {
			m, _ := fixtureRegistroFirmaV2(t)
			canon, err := m.Canonico()
			if err != nil {
				t.Fatal(err)
			}
			h, err := m.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			ref, version := m.DocumentoCustodiaRef, m.DocumentoCustodiaVersion
			w := reciboFirmaSQLV2{reciboFirmaSQL118: reciboFirmaSQL118{FirmaRef: "firma:prueba", ReciboRef: "recibo:prueba", Secuencia: m.Secuencia, Resultado: "firmado", ExpedienteVersion: m.VersionExpediente,
				ActorRef: m.FirmantePrincipalRef, PerfilRef: m.PerfilActivoOperadorRef, RegistradaEn: m.ComprobadaEn, SolicitudHuella: h, DocumentoCustodia: &ref, VersionCustodia: &version},
				CompetenciaEvidenciaRef:          "evidencia:competencia-firmante-ct:" + strings.Repeat("d", 64),
				CompetenciaEvidenciaHuellaSHA256: strings.Repeat("e", 64)}
			if caso == "recibo_incoherente" {
				w.Secuencia++
			}
			if caso == "evidencia_ausente" {
				w.CompetenciaEvidenciaRef = ""
			}
			if caso == "evidencia_alterada" {
				w.CompetenciaEvidenciaHuellaSHA256 = "invalida"
			}
			contenido, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido, sql: registrarFirmaSQL172, argumentos: 13}
			if caso == "revocado_replay" {
				tx.falloConsulta = &pgconn.PgError{Code: "42501"}
			}
			// Incluso un código de permiso devuelto por COMMIT es error incierto.
			if caso == "commit_incierto" {
				tx.falloCommit = &pgconn.PgError{Code: "42501"}
			}
			p := &poolFirmaV2Prueba{tx: tx}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: p}
			v, err := r.RegistrarFirmaVerificadaV2(context.Background(), m, capacidadRegistroFirmaV2Prueba(t, m))
			esperado := ports.ErrResultadoFirmaDocumentoInvalido
			if caso == "revocado_replay" {
				esperado = ports.ErrFirmaDocumentoDenegada
			}
			if caso == "commit_incierto" {
				esperado = ports.ErrRegistroFirmaDocumentoNoDisponible
			}
			if !errors.Is(err, esperado) || v.ReciboRef != "" || tx.rollbacks != 1 || p.inicios != 1 || tx.consultas != 1 {
				t.Fatalf("fallo sin cerrar/no nominal: %v", err)
			}
		})
	}
}

func TestRegistroFirmaV2DescriptorGobernadoAntesDeBegin(t *testing.T) {
	for _, caso := range []string{"fecha_cliente", "original_ajeno", "cargo_posterior", "capacidad_ajena", "legacy_sin_descriptor", "puesto_no_representado", "ambito_no_representado", "acto_no_representado"} {
		t.Run(caso, func(t *testing.T) {
			m, d := fixtureRegistroFirmaV2(t)
			c := capacidadRegistroFirmaV2Prueba(t, m)
			if caso == "fecha_cliente" {
				d.FechaHistorica = &m.ComprobadaEn
			}
			if caso == "original_ajeno" {
				d.Recurso.Original.Referencia = "original:ajeno"
			}
			if caso == "cargo_posterior" {
				d.Seleccion.CargoRef = "cargo:posterior"
			}
			if caso == "fecha_cliente" || caso == "original_ajeno" || caso == "cargo_posterior" {
				datos, err := json.Marshal(d)
				if err != nil {
					t.Fatal(err)
				}
				h := sha256.Sum256(datos)
				c = ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(c.ExportarMaterialParaConsumidor(), datos, hex.EncodeToString(h[:]))
			}
			if caso == "capacidad_ajena" {
				m.ClaveIdempotencia = "clave-prueba-firma-000002"
			}
			switch caso {
			case "puesto_no_representado":
				m.PuestoFirmanteRef = "puesto:prueba"
				c = capacidadRegistroFirmaV2Prueba(t, m)
			case "ambito_no_representado":
				m.AmbitoFirmanteRef = "ambito:prueba"
				c = capacidadRegistroFirmaV2Prueba(t, m)
			case "acto_no_representado":
				m.ActoCompetenciaRef = "acto:prueba"
				c = capacidadRegistroFirmaV2Prueba(t, m)
			}
			tx := &txFirmaV2Prueba{t: t}
			p := &poolFirmaV2Prueba{tx: tx}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: p}
			if caso == "legacy_sin_descriptor" {
				c = ports.TransportarMaterialFirmaVerificadaV2(c.ExportarMaterialParaConsumidor())
			}
			if v, err := r.RegistrarFirmaVerificadaV2(context.Background(), m, c); err == nil || v.ReciboRef != "" || p.inicios != 0 {
				t.Fatal("material sin fuente abrió SQL")
			}
		})
	}
}
