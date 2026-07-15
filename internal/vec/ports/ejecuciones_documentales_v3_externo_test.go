package ports_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

// consumidorPostgreSQLDocumentalV3Prueba demuestra que un adaptador situado
// fuera de package ports puede extraer y persistir todo el material nominal
// que liga el resultado KMS, sin acceder a campos privados ni recibir una
// autoridad autocreable.
type consumidorPostgreSQLDocumentalV3Prueba struct{}

func (c *consumidorPostgreSQLDocumentalV3Prueba) ReleerYConsumirOrdenDespachoDocumentalV3(
	ctx context.Context,
	solicitud ports.SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
) (ports.EstadoCrudoOrdenDespachoDocumentalV3, error) {
	if c == nil || ctx == nil {
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{},
			ports.ErrOrdenDespachoDocumentalV3Invalida
	}
	select {
	case <-ctx.Done():
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{}, ctx.Err()
	default:
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{},
			ports.ErrOrdenDespachoDocumentalV3Invalida
	}
	material, err := solicitud.MaterialCrudo()
	if err != nil {
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{}, err
	}
	huellaMaterial, err := material.HuellaSHA256()
	if err != nil {
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{}, err
	}
	datos, err := resultado.DatosCrudos()
	if err != nil {
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{}, err
	}
	if datos.HuellaSolicitudSHA256 != huellaMaterial {
		return ports.EstadoCrudoOrdenDespachoDocumentalV3{},
			ports.ErrOrdenDespachoDocumentalV3Invalida
	}
	// Referenciar cada campo mantiene comprobable el contrato de persistencia
	// externa ante futuros cambios de visibilidad o de perfil KMS.
	_ = []any{
		datos.HuellaSolicitudSHA256,
		datos.HuellaMaterialCrudoSHA256,
		datos.ComprobacionRef,
		datos.Algoritmo,
		datos.Audiencia,
		datos.Contexto,
		datos.ClaveGestionadaRef,
		datos.RevisionClaveGestionada,
		datos.EvidenciaOperacionRef,
		datos.HuellaAtestacionSHA256,
		datos.ComprobadaEn,
		datos.HuellaResultadoCrudoSHA256,
	}
	return ports.EstadoCrudoOrdenDespachoDocumentalV3{}, errors.New("doble de persistencia")
}

var _ ports.ConsumidorPrivadoOrdenDespachoDocumentalV3 = (*consumidorPostgreSQLDocumentalV3Prueba)(nil)

var errVerificacionKMSExternaPrueba = errors.New("verificacion KMS externa rechazada")

type verificadorKMSExternoPrueba struct {
	claves       map[string][]byte
	comprobadaEn time.Time
}

func (v *verificadorKMSExternoPrueba) VerificarOrdenDespachoDocumentalV3(
	ctx context.Context,
	solicitud ports.SolicitudComprobarOrdenDespachoDocumentalV3,
) (ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3, error) {
	if v == nil || ctx == nil {
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{},
			errVerificacionKMSExternaPrueba
	}
	select {
	case <-ctx.Done():
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, ctx.Err()
	default:
	}
	material, err := solicitud.MaterialCrudo()
	if err != nil {
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	cercado, inicio, reclamacion, err := material.Pruebas()
	if err != nil {
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	resolver := func(referencia string, revision uint64) ([]byte, error) {
		clave, existe := v.claves[fmt.Sprintf("%s#%d", referencia, revision)]
		if !existe {
			return nil, errVerificacionKMSExternaPrueba
		}
		return append([]byte(nil), clave...), nil
	}
	for _, prueba := range []ports.PruebaCrudaAtestacionDespachoDocumentalV3{
		cercado, inicio, reclamacion,
	} {
		if err := verificarPruebaHMACExterna(prueba, resolver); err != nil {
			return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
		}
	}
	mensaje, err := material.MensajeCanonico()
	if err != nil {
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	_, _, _, claveRef, revision, err := cercado.Perfil()
	if err != nil {
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	clave, err := resolver(claveRef, revision)
	if err != nil {
		return ports.ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, err
	}
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(mensaje)
	huellaAtestacion := sha256.Sum256(mac.Sum(nil))
	return ports.NuevoResultadoCrudoVerificacionOrdenDespachoDocumentalV3(
		solicitud, "comprobacion:kms:externa", "evidencia:kms:externa",
		fmt.Sprintf("%x", huellaAtestacion[:]), v.comprobadaEn,
	)
}

var _ ports.VerificadorOrdenDespachoDocumentalV3 = (*verificadorKMSExternoPrueba)(nil)

func verificarPruebaHMACExterna(
	prueba ports.PruebaCrudaAtestacionDespachoDocumentalV3,
	resolver func(string, uint64) ([]byte, error),
) error {
	algoritmo, _, _, claveRef, revision, errPerfil := prueba.Perfil()
	mensaje, errMensaje := prueba.MensajeCanonico()
	sobre, errSobre := prueba.SobreCriptografico()
	if errPerfil != nil || errMensaje != nil || errSobre != nil ||
		algoritmo != ports.AlgoritmoSelloEvidenciaHMACSHA256V3 {
		return errVerificacionKMSExternaPrueba
	}
	clave, err := resolver(claveRef, revision)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(mensaje)
	if !hmac.Equal(sobre, mac.Sum(nil)) {
		return errVerificacionKMSExternaPrueba
	}
	return nil
}

func TestVerificadorKMSExternoPuedeComprobarLasTresPruebasConAPIExportada(t *testing.T) {
	claveRef, revision := "clave:kms:externa", uint64(7)
	clave := []byte("clave de prueba externa con entropia suficiente")
	resolver := func(referencia string, recibida uint64) ([]byte, error) {
		if referencia != claveRef || recibida != revision {
			return nil, errVerificacionKMSExternaPrueba
		}
		return append([]byte(nil), clave...), nil
	}
	perfiles := []struct {
		audiencia string
		contexto  string
	}{
		{ports.AudienciaAtestacionTokenCercadoV3, ports.ContextoAtestacionTokenCercadoV3},
		{ports.AudienciaAtestacionInicioEfectoV3, ports.ContextoAtestacionInicioEfectoV3},
		{ports.AudienciaAtestacionReclamacionV3, ports.ContextoAtestacionReclamacionV3},
	}
	for indice, perfil := range perfiles {
		mensaje := []byte(fmt.Sprintf("mensaje canonico externo %d", indice))
		mac := hmac.New(sha256.New, clave)
		_, _ = mac.Write(mensaje)
		prueba, err := ports.NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
			ports.AlgoritmoSelloEvidenciaHMACSHA256V3, perfil.audiencia, perfil.contexto,
			claveRef, revision, fmt.Sprintf("evidencia:kms:externa:%d", indice),
			mensaje, mac.Sum(nil),
		)
		if err != nil || verificarPruebaHMACExterna(prueba, resolver) != nil {
			t.Fatalf("prueba externa %d rechazada: %v", indice, err)
		}
	}

	mensaje := []byte("mensaje con MAC incorrecta")
	pruebaAlterada, err := ports.NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
		ports.AlgoritmoSelloEvidenciaHMACSHA256V3,
		ports.AudienciaAtestacionTokenCercadoV3, ports.ContextoAtestacionTokenCercadoV3,
		claveRef, revision, "evidencia:kms:externa:alterada", mensaje,
		bytes.Repeat([]byte{0x5a}, sha256.Size),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := verificarPruebaHMACExterna(pruebaAlterada, resolver); !errors.Is(
		err, errVerificacionKMSExternaPrueba,
	) {
		t.Fatalf("MAC externa incorrecta aceptada: %v", err)
	}

	var verificadorNulo *verificadorKMSExternoPrueba
	var puerto ports.VerificadorOrdenDespachoDocumentalV3 = verificadorNulo
	if _, err := puerto.VerificarOrdenDespachoDocumentalV3(
		context.Background(), ports.SolicitudComprobarOrdenDespachoDocumentalV3{},
	); !errors.Is(err, errVerificacionKMSExternaPrueba) {
		t.Fatalf("receptor externo typed nil no fallo cerrado: %v", err)
	}
	contextoCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	verificador := &verificadorKMSExternoPrueba{}
	if _, err := verificador.VerificarOrdenDespachoDocumentalV3(
		contextoCancelado, ports.SolicitudComprobarOrdenDespachoDocumentalV3{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("verificador externo no propago cancelacion: %v", err)
	}
}

func TestDatosCrudosResultadoVerificacionDocumentalV3SeRedactanFueraDelPaquete(t *testing.T) {
	secreto := "kms:referencia:que-no-debe-aparecer"
	datos := ports.DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3{
		ComprobacionRef: secreto,
	}
	texto := fmt.Sprintf("%v|%+v|%#v", datos, datos, datos)
	if strings.Contains(texto, secreto) || !strings.Contains(texto, "REDACTADOS") {
		t.Fatalf("la representacion externa no fue redactada: %s", texto)
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Info("resultado", "datos", datos)
	if strings.Contains(bitacora.String(), secreto) {
		t.Fatalf("slog filtro material KMS: %s", bitacora.String())
	}
	if _, err := json.Marshal(datos); !errors.Is(err, ports.ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("JSON acepto datos KMS: %v", err)
	}
	if _, err := datos.MarshalText(); !errors.Is(err, ports.ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("texto acepto datos KMS: %v", err)
	}
	if _, err := datos.MarshalBinary(); !errors.Is(err, ports.ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("binario acepto datos KMS: %v", err)
	}
}
