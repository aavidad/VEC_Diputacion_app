package httpseguridad

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

type verificadorPeticionFalso struct {
	peticion AsercionPeticionVerificada
}

func (v *verificadorPeticionFalso) VerificarPeticion(context.Context, []byte) (AsercionPeticionVerificada, error) {
	return v.peticion, nil
}

type seudonimizadorPeticionFalso struct{}

func (seudonimizadorPeticionFalso) SeudonimizarSesionPeticion(context.Context, string, string) (SeudonimoSesionPeticion, error) {
	var huella [32]byte
	huella[0] = 1
	return SeudonimoSesionPeticion{
		EsquemaHMAC: "vec.identidad.hmac-sha256.v1", DominioHMACRef: "idh_aaaaaaaaaaaaaaaaaaaaaa",
		ClaveHMACID: "clave-prueba", ClaveHMACVersion: 1, SesionIDHMAC: huella,
	}, nil
}

type registroPeticionFalso struct {
	confirmacion ConfirmacionPeticionSesion
	llamadas     int
	alConsumir   func()
}

func (r *registroPeticionFalso) ConsumirYRevalidar(context.Context, SolicitudConsumoPeticionSesion) (ConfirmacionPeticionSesion, error) {
	r.llamadas++
	if r.alConsumir != nil {
		r.alConsumir()
	}
	return r.confirmacion, nil
}

func TestPeticionCotejaAudienciaCanalYCaducidad(t *testing.T) {
	c := configuracionInternaValida()
	agora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	cuerpo := []byte("{}")
	suma := sha256.Sum256(cuerpo)
	a := AsercionPeticionVerificada{Emisor: c.EmisorIdentidad, Audiencia: c.Audiencia, Superficie: c.Superficie, SesionID: "sesion-1", Metodo: "POST", Destino: "/api/x", CuerpoSHA256: hex.EncodeToString(suma[:]), NonceSHA256: hex.EncodeToString(suma[:]), EmitidaEn: agora, ExpiraEn: agora.Add(time.Minute), CanalVinculadoRef: "tls-exportador:sha256:canal"}
	if !peticionCoincide(a, c, a.CanalVinculadoRef, "POST", "/api/x", cuerpo, agora) {
		t.Fatal("peticion valida rechazada")
	}
	for _, cambiar := range []func(*AsercionPeticionVerificada){func(x *AsercionPeticionVerificada) { x.Audiencia = "otra" }, func(x *AsercionPeticionVerificada) { x.CanalVinculadoRef = "otro" }, func(x *AsercionPeticionVerificada) { x.ExpiraEn = agora }} {
		x := a
		cambiar(&x)
		if peticionCoincide(x, c, a.CanalVinculadoRef, "POST", "/api/x", cuerpo, agora) {
			t.Fatal("cotejo hostil admitido")
		}
	}
}

func TestServicioPeticionExigeCanalDeLaMismaAutoridadYRevalidaAlRegresar(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	autoridad, err := NuevoServicioIdentidad(configuracion, &verificadorFalso{}, &evaluadorFalso{}, nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	otraAutoridad, err := NuevoServicioIdentidad(configuracion, &verificadorFalso{}, &evaluadorFalso{}, nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := []byte("{}")
	suma := sha256.Sum256(cuerpo)
	peticion := AsercionPeticionVerificada{
		Emisor: configuracion.EmisorIdentidad, Audiencia: configuracion.Audiencia,
		Superficie: configuracion.Superficie, SesionID: "sesion-externa-1", Metodo: "POST",
		Destino: "/api/vec/bolsa/mi-bolsa", CuerpoSHA256: hex.EncodeToString(suma[:]),
		NonceSHA256: hex.EncodeToString(sha256Sum([]byte("nonce-1"))), EmitidaEn: ahora,
		ExpiraEn: ahora.Add(time.Minute), CanalVinculadoRef: "tls-exportador:sha256:canal-1",
	}
	registro := &registroPeticionFalso{confirmacion: confirmacionPeticionPrueba(ahora.Add(2 * time.Minute))}
	reloj := &relojFijo{ahora: ahora}
	servicio, err := NuevoServicioPeticionSesion(autoridad, &verificadorPeticionFalso{peticion: peticion}, registro, seudonimizadorPeticionFalso{}, reloj)
	if err != nil {
		t.Fatal(err)
	}
	canalAjeno := canalPeticionPrueba(otraAutoridad, peticion.CanalVinculadoRef)
	if _, err = servicio.Resolver(context.Background(), []byte("firmada"), canalAjeno, peticion.Metodo, peticion.Destino, cuerpo); !errors.Is(err, ErrAsercionPeticionNoValida) {
		t.Fatalf("canal de otra autoridad admitido: %v", err)
	}
	if registro.llamadas != 0 {
		t.Fatal("el canal ajeno alcanzo el registro")
	}
	canal := canalPeticionPrueba(autoridad, peticion.CanalVinculadoRef)
	if _, err = servicio.Resolver(context.Background(), []byte("firmada"), canal, peticion.Metodo, peticion.Destino, cuerpo); err != nil {
		t.Fatalf("peticion valida rechazada: %v", err)
	}
	if registro.llamadas != 1 {
		t.Fatalf("consumos = %d", registro.llamadas)
	}

	registro.alConsumir = func() { reloj.fijar(peticion.ExpiraEn) }
	if _, err = servicio.Resolver(context.Background(), []byte("firmada"), canal, peticion.Metodo, peticion.Destino, cuerpo); !errors.Is(err, ErrAsercionPeticionNoValida) {
		t.Fatalf("asercion caducada durante el registro admitida: %v", err)
	}
}

func sha256Sum(valor []byte) []byte {
	suma := sha256.Sum256(valor)
	return suma[:]
}

func confirmacionPeticionPrueba(vigenteHasta time.Time) ConfirmacionPeticionSesion {
	return ConfirmacionPeticionSesion{
		SesionRef: "ses_aaaaaaaaaaaaaaaaaaaaaa", AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaa",
		AsercionRef: "ase_aaaaaaaaaaaaaaaaaaaaaa", CuentaRef: "cta_aaaaaaaaaaaaaaaaaaaaaa",
		ControlSesionRef: "cse_aaaaaaaaaaaaaaaaaaaaaa", ControlSesionRevision: 1,
		SesionValidaHasta: vigenteHasta,
	}
}

func canalPeticionPrueba(autoridad *ServicioIdentidad, referencia string) CanalProxyAutenticado {
	return CanalProxyAutenticado{
		tipo: CanalProxyTLSMutuo, identidadPar: "dns:proxy-interno.mulhacen.test",
		evidenciaRef: referencia, superficie: autoridad.configuracion.Superficie,
		instanciaRef: autoridad.instanciaRef, servicio: autoridad,
	}
}
