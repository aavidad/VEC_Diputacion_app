package postgres

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func expedienteAltaCivilSQLPrueba(t *testing.T) ([]byte, domain.Expediente) {
	t.Helper()
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	necesidad := evidenciaAltaConNecesidad(t)
	evidencia.Expediente.Solicitud.Necesidad = &necesidad
	if err := evidencia.Expediente.Validar(); err != nil {
		t.Fatal(err)
	}
	legado, err := json.Marshal(evidencia.Expediente)
	if err != nil {
		t.Fatal(err)
	}
	civil := bytes.ReplaceAll(legado, []byte(`"inicio":"2026-09-01T00:00:00Z"`), []byte(`"inicio":"2026-09-01"`))
	civil = bytes.ReplaceAll(civil, []byte(`"fin":"2026-09-30T00:00:00Z"`), []byte(`"fin":"2026-09-30"`))
	if bytes.Equal(civil, legado) || bytes.Count(civil, []byte(`"inicio":"2026-09-01"`)) != 2 ||
		bytes.Count(civil, []byte(`"fin":"2026-09-30"`)) != 2 {
		t.Fatal("la muestra no contiene los dos periodos civiles del alta v3")
	}
	return civil, evidencia.Expediente
}

func TestPreparacionAnalisisRehidrataAltaV3Civil(t *testing.T) {
	civil, expediente := expedienteAltaCivilSQLPrueba(t)
	solicitud := solicitudAnalisisPostgreSQLPrueba(t, expediente)
	fila := filaAnalisisPostgreSQLPrueba(t, "reservada", solicitud, expediente, nil).(filaPreparacionPrueba)
	fila.valores[1] = string(civil)
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador, err := nuevoPreparadorOperacionAnalisisPostgreSQL(&iniciadorPreparacionPrueba{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	preparacion, err := preparador.PrepararOperacionAnalisis(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := preparacion.DatosPara(solicitud)
	if err != nil || datos.Estado != ports.PreparacionOperacionAnalisisReservada ||
		datos.ExpedienteAnterior == nil || datos.ExpedienteAnterior.Validar() != nil || !tx.configurada || tx.confirmaciones != 1 {
		t.Fatalf("la reserva v3 no se pudo rehidratar: %v", err)
	}
}

func TestRehidratarExpedienteSQLConservaHistoriaYDosPeriodos(t *testing.T) {
	civil, esperado := expedienteAltaCivilSQLPrueba(t)
	original := bytes.Clone(civil)
	huella := sha256.Sum256(civil)
	sello := hmac.New(sha256.New, []byte("clave-sintetica-de-prueba"))
	_, _ = sello.Write(civil)
	hmacOriginal := sello.Sum(nil)

	var recuperado domain.Expediente
	if err := decodificarExpedienteSQL(civil, &recuperado); err != nil {
		t.Fatal(err)
	}
	if recuperado.Validar() != nil || recuperado.Solicitud.Necesidad == nil ||
		!recuperado.Solicitud.Periodo.Inicio.Equal(esperado.Solicitud.Periodo.Inicio) ||
		!recuperado.Solicitud.Periodo.Fin.Equal(esperado.Solicitud.Periodo.Fin) ||
		!recuperado.Solicitud.Necesidad.Periodo.Inicio.Equal(esperado.Solicitud.Necesidad.Periodo.Inicio) ||
		!recuperado.Solicitud.Necesidad.Periodo.Fin.Equal(esperado.Solicitud.Necesidad.Periodo.Fin) ||
		recuperado.Solicitud.Periodo.Inicio.Location() != time.UTC {
		t.Fatal("periodos civiles no restaurados a medianoche UTC")
	}
	if !bytes.Equal(civil, original) || sha256.Sum256(civil) != huella {
		t.Fatal("el contenido histórico se modificó al rehidratar")
	}
	sello = hmac.New(sha256.New, []byte("clave-sintetica-de-prueba"))
	_, _ = sello.Write(civil)
	if !hmac.Equal(sello.Sum(nil), hmacOriginal) {
		t.Fatal("el HMAC del original ha cambiado")
	}

	respuesta, err := json.Marshal(struct {
		Esquema    string          `json:"esquema"`
		Expediente json.RawMessage `json:"expediente"`
	}{"ejemplo.sintetico", civil})
	if err != nil {
		t.Fatal(err)
	}
	var anidada struct {
		Esquema    string             `json:"esquema"`
		Expediente *domain.Expediente `json:"expediente"`
	}
	if err := decodificarConExpedienteSQL(respuesta, &anidada, "expediente"); err != nil ||
		anidada.Expediente == nil || anidada.Expediente.Validar() != nil {
		t.Fatalf("respuesta de seguimiento no rehidratada: %v", err)
	}
}

func TestRehidratarExpedienteSQLLegadoYFechasInvalidas(t *testing.T) {
	civil, esperado := expedienteAltaCivilSQLPrueba(t)
	legado, err := json.Marshal(esperado)
	if err != nil {
		t.Fatal(err)
	}
	adaptado, err := rehidratarFechasExpedienteSQL(legado)
	if err != nil || !bytes.Equal(adaptado, legado) {
		t.Fatalf("RFC3339 legado alterado: %v", err)
	}
	var recuperado domain.Expediente
	if err := decodificarExpedienteSQL(legado, &recuperado); err != nil || recuperado.Validar() != nil {
		t.Fatalf("RFC3339 legado rechazado: %v", err)
	}
	for _, valor := range []string{"2026-02-30", "2026-9-01", "2026-13-01", "2026-09-01x"} {
		t.Run(valor, func(t *testing.T) {
			invalido := bytes.Replace(civil, []byte(`"inicio":"2026-09-01"`), []byte(`"inicio":"`+valor+`"`), 1)
			if err := decodificarExpedienteSQL(invalido, &domain.Expediente{}); err == nil {
				t.Fatal("fecha civil inválida aceptada")
			}
		})
	}
	conDesconocido := bytes.Replace(civil, []byte(`"solicitud":{`), []byte(`"solicitud":{"campo_ajeno":1,`), 1)
	if err := decodificarExpedienteSQL(conDesconocido, &domain.Expediente{}); err == nil {
		t.Fatal("el decoder ha permitido un campo desconocido")
	}
	conDuplicado := bytes.Replace(civil, []byte(`"solicitud":{`), []byte(`"solicitud":{"campo_ajeno":1,"campo_ajeno":2,`), 1)
	if err := decodificarExpedienteSQL(conDuplicado, &domain.Expediente{}); err == nil {
		t.Fatal("la adaptación ha ocultado claves duplicadas")
	}
	if strings.Contains(string(civil), "campo_ajeno") {
		t.Fatal("la prueba modificó el original")
	}
}
